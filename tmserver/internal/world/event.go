package world

import (
	"errors"
	"math/rand/v2"
	"net"
	"time"

	"openwyd-nextgen/internal/metrics"
	"openwyd-nextgen/tmserver/internal/protocol"
)

// Motivos de desconexão da Fase 7 (migration-plan.md §5).
var (
	errRateLimited      = errors.New("world: connection exceeded message rate limit")
	errChecksumMismatch = errors.New("world: rejected frame on checksum mismatch")
)

// event é algo que o loop aplica ao estado do mundo. Todo evento é aplicado na
// goroutine do loop, então apply pode mutar o World livremente.
type event interface{ apply(*World) }

// emit envia um evento para o loop, ou devolve false se o mundo está desligando
// (para as goroutines de conexão pararem em vez de bloquear para sempre).
func (w *World) emit(ev event) bool {
	select {
	case w.events <- ev:
		return true
	case <-w.done:
		return false
	}
}

// --- conexão ---

type connectEvent struct {
	conn net.Conn
	ip   string
}

func (e connectEvent) apply(w *World) {
	conn := w.allocConn()
	if conn < 0 {
		w.log.Warn("server full; rejecting connection", "ip", e.ip)
		_ = e.conn.Close()
		return
	}
	// Trace_id do aceite (OBS-02): a conexão inteira — login, chamadas ao
	// dbServer e todo log do handler — carrega o mesmo id, então uma falha de
	// login é rastreável de ponta a ponta sem procurar por conta/horário.
	s := &Session{
		Conn:    conn,
		TraceID: metrics.NewTraceID(),
		IP:      e.ip,
		Mode:    UserAccept,
		conn:    e.conn,
		out:     newOutQueue(),
		closeCh: make(chan struct{}),
	}
	w.sessions[conn] = s
	w.entities[conn] = &Entity{ID: conn, Mode: MobUserDock}
	w.metricAdd("cpsock_connections_active", "Conexões CPSock ativas.", 1)
	w.metricInc("cpsock_connections_total", "Conexões CPSock aceitas desde o boot.")
	go w.readLoop(s)
	go w.writeLoop(s)
	w.log.Info("connection accepted", "conn", conn, "ip", e.ip, "trace_id", s.TraceID)
}

// allocConn acha um slot livre de jogador. Conn 0 é reservado (o original exige
// conn > 0 para login — handlers/_MSG_AccountLogin.md; ids de trade/ataque também usam
// (0, MAX_USER)). Os ADMIN_RESERV slots do topo também são reservados (valor UNVERIFIED).
func (w *World) allocConn() int {
	for i := 1; i < MaxUser; i++ {
		if w.sessions[i] == nil {
			return i
		}
	}
	return -1
}

// --- frame ---

type frameEvent struct {
	s       *Session
	header  protocol.Header
	payload []byte
}

func (e frameEvent) apply(w *World) {
	// Ignora frames de uma sessão cujo slot já foi liberado/reutilizado.
	if w.sessions[e.s.Conn] != e.s {
		return
	}
	// Guardas globais do despachante (protocol-spec.md §2): Ping não faz nada; o
	// SkipCheckTick interno nunca pode vir de um cliente.
	if e.header.Type == protocol.MsgPing {
		return
	}
	if e.header.ClientTick == protocol.SkipCheckTick {
		return
	}
	w.handler(w, e.s, e.header, e.payload)
}

// --- callback assíncrono (resultados do banco voltando ao loop) ---

// callbackEvent traz o resultado de trabalho fora do loop (World.Go) de volta ao
// loop. Ele é descartado se o slot da sessão foi liberado ou reutilizado nesse meio tempo.
type callbackEvent struct {
	conn int
	sess *Session
	cb   func(*World, *Session)
}

func (e callbackEvent) apply(w *World) {
	if w.sessions[e.conn] != e.sess {
		return
	}
	e.cb(w, e.sess)
}

// worldCallbackEvent traz o resultado de trabalho fora do loop que NÃO está ligado a uma
// sessão (World.GoDetached) — ex.: um reload de config de NPC. Ele sempre aplica (sem
// guarda de vivacidade da sessão), então seu callback roda dentro da goroutine do loop e
// pode mutar o estado do mundo direto.
type worldCallbackEvent struct {
	cb func(*World)
}

func (e worldCallbackEvent) apply(w *World) { e.cb(w) }

// --- desconexão ---

type disconnectEvent struct {
	s   *Session
	err error
}

func (e disconnectEvent) apply(w *World) {
	// Expor POR QUE uma conexão caiu (INITCODE inválido, EOF, tamanho excessivo, taxa, checksum)
	// em vez de engolir o motivo — fundamental para diagnosticar o handshake com o cliente.
	if e.err != nil && w.sessions[e.s.Conn] == e.s {
		w.log.Info("connection drop reason", "conn", e.s.Conn, "ip", e.s.IP, "err", e.err.Error())
	}
	w.removeSession(e.s)
}

// SetSessionEndHandler registra o gancho de desmonte de sessão (desvincular grupo). Como
// o handler de tick, ele roda DENTRO da goroutine do loop e pode mutar o estado do mundo
// direto; ele vê a sessão e sua entidade ainda vivas. Chame antes de Run.
func (w *World) SetSessionEndHandler(fn func(*World, *Session)) { w.onSessionEnd = fn }

// removeSession desmonta uma sessão e libera seu slot. Só age se o
// slot ainda contém exatamente esta sessão (protege contra slot reutilizado). Só no loop.
func (w *World) removeSession(s *Session) {
	if s == nil || w.sessions[s.Conn] != s {
		return
	}
	w.metricAdd("cpsock_connections_active", "Conexões CPSock ativas.", -1)
	// Desmonte de estado compartilhado (o vínculo de grupo e suas invocações) enquanto a sessão e
	// sua entidade ainda são endereçáveis — o legado chama RemoveParty de CloseUser
	// pelo mesmo motivo (Server.cpp:7654).
	if w.onSessionEnd != nil {
		w.onSessionEnd(w, s)
	}
	// Primeiro o post-mortem S→C, enquanto o slot ainda identifica a sessão: o que
	// foi enviado ao cliente por último é a evidência-chave de um travamento no cliente
	// (docs/migration/investigacao-freeze-cliente.md).
	w.logSendStats(s)
	// Persistir o personagem vivo (compras/ouro/atributos) e, como a sessão de
	// conta termina com a conexão, o cargo compartilhado da conta, que é
	// despejado (ele sobrevive a cada personagem em separado, mas não à conexão).
	// Rastreado, para um novo login da conta esperar o save.
	w.saveAccountOnClose(s)
	// Avisar os jogadores em visão que esta entidade saiu (logout), para seus clientes a removerem.
	if e := w.entities[s.Conn]; e != nil && e.Mode == MobUser {
		body := protocol.EncodeRemoveMobBody(2) // 2 = logout
		w.ForEachInView(s.Conn, func(vs *Session, _ *Entity) {
			w.enqueue(vs, protocol.Header{Type: protocol.MsgRemoveMob, ID: uint16(s.Conn)}, body)
			w.UnmarkSeen(vs, s.Conn) // um jogador futuro neste id de conexão precisa recriar
		})
		// Liberar a célula de grade do jogador (ocupada desde o login/último movimento), senão a
		// perseguição das criaturas e as varreduras de visão seguem batendo num fantasma.
		if cur, ok := w.grid.MobAt(int(e.X), int(e.Y)); ok && int(cur) == s.Conn {
			w.grid.ClearMob(int(e.X), int(e.Y))
		}
	}
	s.close()
	w.sessions[s.Conn] = nil
	w.entities[s.Conn] = nil
	w.log.Info("connection closed", "conn", s.Conn)
}

// dropSession é removeSession por ponteiro, usado quando o loop precisa dispensar um
// cliente lento (fila de saída acima do limite de bytes).
func (w *World) dropSession(s *Session) { w.removeSession(s) }

// --- goroutines de I/O por conexão (NÃO o loop) ---

// readLoop desenquadra e decodifica mensagens CPSock de uma conexão e as alimenta
// no loop como frameEvents. Roda na própria goroutine e nunca
// toca o estado do mundo diretamente.
func (w *World) readLoop(s *Session) {
	fr := protocol.NewFramer(s.conn)
	bucket := newTokenBucket(w.cfg.MaxMsgPerSec, w.cfg.MsgBurst, time.Now())
	for {
		frame, err := fr.ReadFrame()
		if err != nil {
			w.emit(disconnectEvent{s: s, err: err})
			return
		}
		// Fase 7: proteção contra flood — uma conexão que excede sua taxa é derrubada.
		if !bucket.allow(time.Now()) {
			w.emit(disconnectEvent{s: s, err: errRateLimited})
			return
		}
		h, payload, mismatch, err := protocol.Decode(frame)
		if err != nil {
			w.emit(disconnectEvent{s: s, err: err})
			return
		}
		// Fase 7: checksum rejeitador opcional (desligado por padrão; o legado não rejeita).
		if mismatch && w.cfg.RejectChecksum {
			w.emit(disconnectEvent{s: s, err: errChecksumMismatch})
			return
		}
		if !w.emit(frameEvent{s: s, header: h, payload: payload}) {
			return // mundo desligando
		}
	}
}

// writeLoop codifica mensagens S→C enfileiradas e as escreve no socket. O
// escritor é dono do fechamento do socket: ao fechar ele esvazia o que ainda está na fila para
// um aviso final enviado logo antes do fechamento ser entregue. O índice de palavra-chave
// por pacote é aleatório (CPSock.cpp:535) e irrelevante para a paridade (parity-tests.md
// §4.0), então o RNG global serve aqui.
func (w *World) writeLoop(s *Session) {
	defer func() { _ = s.conn.Close() }()
	for {
		select {
		case <-s.out.ready:
			for {
				of, ok := s.out.pop()
				if !ok {
					break
				}
				if !w.writeFrame(s, of) {
					return
				}
			}
		case <-s.closeCh:
			// Fechamento gracioso: esvaziar e escrever o que ainda está na fila, depois sair
			// (o Close adiado desbloqueia o leitor).
			for {
				of, ok := s.out.pop()
				if !ok {
					return
				}
				if !w.writeFrame(s, of) {
					return
				}
			}
		}
	}
}

// writeFrame codifica e escreve um frame S→C. Devolve false (parando o
// escritor) só em erro de escrita no socket; um erro de codificação é logado e pulado.
func (w *World) writeFrame(s *Session, of outFrame) bool {
	wire, err := protocol.Encode(of.header, of.payload, uint8(rand.IntN(256)))
	if err != nil {
		w.log.Warn("encode failed", "conn", s.Conn, "err", err)
		return true
	}
	if _, err := s.conn.Write(wire); err != nil {
		w.emit(disconnectEvent{s: s, err: err})
		return false
	}
	return true
}
