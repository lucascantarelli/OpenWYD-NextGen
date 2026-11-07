package world

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
)

// Diagnóstico de envio S→C (docs/migration/investigacao-freeze-cliente.md): o
// incidente de travamento do cliente de 18-19/jul mostrou o servidor saudável enquanto o cliente
// parava de processar em silêncio o que recebia — e nada logava o lado
// S→C. Todo frame enfileirado é contado por tipo e lembrado num pequeno anel,
// e o quadro completo é despejado quando a conexão fecha, então todo travamento
// deixa um post-mortem do que foi enviado ao cliente por último. Tudo isso é estado de sessão
// de posse do loop (enqueue e removeSession rodam no loop), então sem locks.

// lastSentRing é quantos frames S→C finais cada sessão lembra. 64 cobre
// os segundos finais mesmo sob uma rajada de CreateMob, custando ~1KB por sessão.
const lastSentRing = 64

// sentRecord é um frame S→C lembrado: tipo, HEADER.ID (a entidade assunto —
// sobre qual criatura/jogador era a mensagem), tamanho do payload e quando foi enfileirado.
type sentRecord struct {
	t   protocol.Type
	id  uint16
	len uint16
	at  int64 // milissegundos unix
}

// noteSend registra um frame S→C enfileirado no diagnóstico da sessão.
func (s *Session) noteSend(h protocol.Header, payloadLen int) {
	s.sentFrames++
	s.sentBytes += uint64(payloadLen + protocol.HeaderSize)
	if s.sentByType == nil {
		s.sentByType = make(map[protocol.Type]uint32, 32)
	}
	s.sentByType[h.Type]++
	s.lastSent[s.lastSentIdx%lastSentRing] = sentRecord{
		t: h.Type, id: h.ID, len: uint16(payloadLen), at: time.Now().UnixMilli(),
	}
	s.lastSentIdx++
}

// SentOfType devolve quantos frames S→C do tipo t foram enfileirados para s nesta
// sessão. Só no loop; usado pelos handlers para logar deltas por ação (ex.: quantos
// CreateMob/RemoveMob um teleporte produziu).
func (w *World) SentOfType(s *Session, t protocol.Type) uint32 { return s.sentByType[t] }

// logSendStats emite o post-mortem S→C da sessão: totais, a quebra por
// tipo e os frames finais com sua idade relativa ao fechamento. Chamado
// de removeSession, então toda desconexão (inclusive um usuário matando um cliente
// travado) deixa a evidência nos logs.
func (w *World) logSendStats(s *Session) {
	if s.sentFrames == 0 {
		return
	}
	w.log.Info("session send stats",
		"conn", s.Conn,
		"frames", s.sentFrames,
		"bytes", s.sentBytes,
		"out_high_water", s.outHighWater,
		"out_high_water_bytes", s.outHighWaterBytes,
		"by_type", formatByType(s.sentByType),
	)
	w.log.Info("session last sends", "conn", s.Conn, "tail", formatLastSends(&s.lastSent, s.lastSentIdx, time.Now().UnixMilli()))
}

// formatByType renderiza um mapa de contagem por tipo como "0x0366:120 0x0181:80 …",
// em ordem decrescente de contagem para o tráfego dominante aparecer primeiro.
func formatByType(m map[protocol.Type]uint32) string {
	type kv struct {
		t protocol.Type
		n uint32
	}
	rows := make([]kv, 0, len(m))
	for t, n := range m {
		rows = append(rows, kv{t, n})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].n != rows[j].n {
			return rows[i].n > rows[j].n
		}
		return rows[i].t < rows[j].t
	})
	var b strings.Builder
	for i, r := range rows {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(formatSendType(r.t))
		b.WriteByte(':')
		b.WriteString(strconv.FormatUint(uint64(r.n), 10))
	}
	return b.String()
}

// formatLastSends renderiza o anel do mais antigo→mais novo como
// "0x0165(30000)/504/-830ms …" — tipo(HEADER.ID)/tamanho-do-payload/idade-antes-do-fechamento.
func formatLastSends(ring *[lastSentRing]sentRecord, next int, nowMs int64) string {
	n := next
	if n > lastSentRing {
		n = lastSentRing
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		r := ring[(next-n+i)%lastSentRing]
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(formatSendType(r.t))
		b.WriteByte('(')
		b.WriteString(strconv.FormatUint(uint64(r.id), 10))
		b.WriteString(")/")
		b.WriteString(strconv.FormatUint(uint64(r.len), 10))
		b.WriteString("/-")
		b.WriteString(strconv.FormatInt(nowMs-r.at, 10))
		b.WriteString("ms")
	}
	return b.String()
}

// formatSendType renderiza um tipo de mensagem como 0xNNNN (mesmo formato do
// formatType de pacote recebido do pacote handler; duplicado porque world não pode importar
// handler).
func formatSendType(t protocol.Type) string {
	const hexdigits = "0123456789abcdef"
	v := uint16(t)
	return "0x" + string([]byte{hexdigits[v>>12&0xf], hexdigits[v>>8&0xf], hexdigits[v>>4&0xf], hexdigits[v&0xf]})
}
