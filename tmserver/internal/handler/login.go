package handler

import (
	"context"
	"strings"
	"time"

	"openwyd-nextgen/internal/crossover"
	"openwyd-nextgen/internal/metrics"
	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// accountLogin trata _MSG_AccountLogin (0x020D), handlers/_MSG_AccountLogin.md.
// Ele valida a versão do cliente, o modo de sessão e o gate de força bruta, então
// repassa as credenciais ao dbServer de forma assíncrona (o original encaminha
// _MSG_DBAccountLogin e espera em USER_LOGIN).
func (d *Dispatcher) accountLogin(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	if len(payload) < protocol.MsgAccountLoginBodySize {
		d.log.Warn("account login: short packet", "conn", s.Conn)
		w.Close(s)
		return
	}
	var body protocol.MsgAccountLoginBody
	if err := body.Decode(payload); err != nil {
		w.Close(s)
		return
	}

	// Checagem de versão server-authoritative (anti-cheat: bloqueia clientes forjados/antigos).
	// Este build 7662 "Cavaleiros de Kersef" envia ClientVersion=12000 (definido via
	// -client-version); divergências recebem o aviso de reexecutar e são descartadas.
	if body.ClientVersion != d.cfg.ClientVersion {
		d.log.Warn("account login: version mismatch",
			"conn", s.Conn, "got", body.ClientVersion, "want", d.cfg.ClientVersion)
		d.sendClientMessage(w, s, versionMismatchText)
		w.Close(s)
		return
	}
	if s.Mode != world.UserAccept {
		// Nenhum texto do Language.txt confirmado para este (o
		// _MSG_AccountLogin.cpp:56-63 legado registra CrackLog e responde ao cliente); mantenha
		// o placeholder até uma captura fixá-lo.
		d.notify(w, s, NoticeLoginNow)
		return
	}

	name := strings.ToLower(cstr(body.AccountName[:]))
	if name == "" {
		w.Close(s)
		return
	}
	if d.fails[name] >= d.cfg.MaxFailLogin {
		d.log.Warn("account login: locked out after wrong passwords", "conn", s.Conn, "account", name, "fails", d.fails[name])
		d.sendClientMessage(w, s, threeWrongPassText)
		return
	}

	pass := cstr(body.AccountPassword[:])
	if tok := cstr(body.Zero[:]); crossover.IsToken(tok) {
		// Troca de canal (canais.md §8.1.2): um cliente que veio de outro
		// canal envia o token /srv em Zero[52] em vez da senha — o
		// mesmo campo que o legado compara contra TempKey. Ele viaja como a
		// senha para o dbServer validá-lo; nenhum campo do proto é adicionado.
		pass = tok
		d.log.Info("account login: server-change token", "conn", s.Conn, "account", name)
	}
	s.AccountName = name
	s.Mode = world.UserLogin
	d.log.Info("account login: relaying to dbServer", "conn", s.Conn, "trace_id", s.TraceID, "account", name)

	p := w.Persistence()
	w.Go(s, func() func(*world.World, *world.Session) {
		// O trace_id da sessão viaja no metadata gRPC (OBS-02): a linha do
		// dbServer sai com o MESMO id da linha "connection accepted" aqui.
		ctx, cancel := context.WithTimeout(metrics.WithTraceID(context.Background(), s.TraceID), 10*time.Second)
		defer cancel()
		out, err := p.AccountLogin(ctx, name, pass)
		return func(w *world.World, s *world.Session) { d.completeAccountLogin(w, s, out, err) }
	})
}

// completeAccountLogin aplica o resultado de login do dbServer de volta no loop.
func (d *Dispatcher) completeAccountLogin(w *world.World, s *world.Session, out world.LoginOutcome, err error) {
	if err != nil {
		d.log.Error("account login backend error", "conn", s.Conn, "trace_id", s.TraceID, "account", s.AccountName, "err", err)
		s.Mode = world.UserAccept
		d.notify(w, s, NoticeDBError)
		w.Close(s)
		return
	}
	switch out.Result {
	case world.LoginOK:
		delete(d.fails, s.AccountName)
		if d.refuseDuplicateLogin(w, s, out.AccountID) {
			return
		}
		s.AccountID = out.AccountID
		s.AccessLevel = world.ParseAccess(out.Role) // privilégio de GM/moderação (issue #122)
		d.log.Info("account login: OK", "conn", s.Conn, "trace_id", s.TraceID, "account", s.AccountName, "id", out.AccountID, "role", s.AccessLevel, "chars", len(out.Characters))
		// Instala o cargo compartilhado da conta, carregado na mesma ida e volta ao backend.
		// Ele vive por toda a sessão da conta e é liberado na desconexão.
		cargo := out.Cargo
		w.SetCargo(out.AccountID, &cargo)
		// Drena quaisquer grants pendentes da loja web de doações (buscados na mesma
		// ida e volta do login) para o cargo recém-carregado (issue #34); os itens caem no
		// próximo slot livre, ou se perdem quando cheio. Codifique DEPOIS da drenagem para o
		// cache do vault do cliente incluir itens recém-entregues.
		w.ApplyDeliveries(s, out.PendingDeliveries)
		s.Mode = world.UserSelChar
		coin, cargoItems := d.cargoWire(w.Cargo(out.AccountID))
		body := protocol.EncodeCNFAccountLoginBody(s.AccountName, d.selCharsFrom(out.Characters), coin, cargoItems)
		w.SendTo(s, protocol.Header{Type: protocol.MsgCNFAccountLogin, ID: protocol.IDSelChar}, body)
	case world.LoginBadPassword:
		d.fails[s.AccountName]++
		d.log.Warn("account login: bad password", "conn", s.Conn, "account", s.AccountName, "fails", d.fails[s.AccountName])
		s.Mode = world.UserAccept // permite tentar de novo
		d.sendClientMessage(w, s, wrongPasswordText)
	case world.LoginNoAccount:
		s.Mode = world.UserAccept
		d.sendClientMessage(w, s, noAccountText)
	case world.LoginBlocked:
		d.sendClientMessage(w, s, blockedAccountText)
		w.Close(s)
	case world.LoginAlreadyPlaying:
		s.Mode = world.UserAccept // permite tentar de novo, como a recusa de login duplicado abaixo
		w.Send(s, protocol.MsgAlreadyPlaying, nil)
	}
}

// accountFromOthers é o Language.txt _NN_Your_Account_From_Others (134), nos
// bytes Windows-1252 da tabela, como o resto da tabela de strings legada.
const accountFromOthers = "Conta desconectada por conex\xe3o simult\xe2nea."

// refuseDuplicateLogin aplica a regra legada para um segundo login de uma
// conta que já está no jogo (DBSrv CFileDB.cpp:1001-1017,
// TMSrv ProcessDBMessage.cpp _MSG_DBSavingQuit):
//   - a sessão que já está na seleção de personagem ou em jogo recebe
//     _NN_Your_Account_From_Others e é fechada, o que a salva;
//   - o login novo é recusado com _MSG_StillPlaying (0x011D) e pode tentar
//     de novo; o cliente 7662 mostra "Conexão anterior finalizada. Tente
//     novamente." (TMSelectServerScene.cpp:1571);
//   - enquanto esse save está em voo uma nova tentativa recebe _MSG_AlreadyPlaying (0x011C),
//     então ele nunca carrega as linhas de personagem/cargo antes de serem escritas.
//
// O legado fecha o socket recusado; aqui ele volta para UserAccept para o
// cliente poder tentar de novo na mesma conexão. A checagem é por tmServer: o
// dbServer não rastreia contas online, então uma segunda instância de tmServer não
// veria esta sessão. Só no loop.
func (d *Dispatcher) refuseDuplicateLogin(w *world.World, s *world.Session, accountID int64) bool {
	refusal := protocol.Type(0)
	switch old := w.SessionByAccount(accountID, s); {
	case old != nil:
		d.log.Info("account login: duplicate, closing the previous session", "conn", s.Conn, "account", s.AccountName, "previous", old.Conn)
		d.sendClientMessage(w, old, accountFromOthers)
		w.Close(old)
		refusal = protocol.MsgStillPlaying
	case w.AccountSaving(accountID):
		d.log.Info("account login: previous session still saving", "conn", s.Conn, "account", s.AccountName)
		refusal = protocol.MsgAlreadyPlaying
	default:
		return false
	}
	s.Mode = world.UserAccept
	w.Send(s, refusal, nil)
	return true
}

func (d *Dispatcher) cargoWire(st *world.CargoState) (int32, [128]protocol.SelItem) {
	var items [128]protocol.SelItem
	if st == nil {
		return 0, items
	}
	for i := range st.Items {
		if i >= len(items) {
			break
		}
		items[i] = itemToSel(st.Items[i])
	}
	return st.Coin, items
}

// selCharsFrom mapeia os resumos de personagem do dbServer para linhas protocol.SelChar da
// STRUCT_SELCHAR exata em bytes (MSG_CNFAccountLogin / MSG_CNFNewCharacter). O
// resumo carrega o score real (ouro, HP/MP, atributos) para a tela de
// seleção pré-visualizar o personagem real de cada slot, não placeholders. O nível vai como
// armazenado, como o snapshot no mundo: as duas telas mostram Nível+1
// (TMSelectCharScene.cpp:1018, TMFieldScene.cpp:4230), então um personagem novo —
// nível 0, como os templates BaseMob legados — lê "Nv 1" nas duas.
func (d *Dispatcher) selCharsFrom(chars []world.CharSummary) []protocol.SelChar {
	out := make([]protocol.SelChar, 0, len(chars))
	for _, c := range chars {
		sc := protocol.SelChar{
			Slot:  c.Slot,
			Name:  c.Name,
			Level: int32(c.Level),
			Exp:   c.Exp,
			Guild: c.GuildID,
			Coin:  c.Coin,
			MaxHp: c.MaxHp, Hp: c.Hp, MaxMp: c.MaxMp, Mp: c.Mp,
			Str: c.Str, Int: c.Int, Dex: c.Dex, Con: c.Con,
		}
		// Pré-visualiza o equipamento salvo. Equipamento vazio cai para o BaseMob da classe para
		// personagens novos ainda renderizarem com o modelo de classe certo na tela de seleção.
		if !equipEmpty(c.Equip) {
			for i := range c.Equip {
				sc.Equip[i] = itemToSel(c.Equip[i])
			}
		} else if tmpl, ok := d.baseMobs[c.Class]; ok && len(tmpl) == content.BaseMobSize {
			sc.Equip = protocol.MobEquip(tmpl)
		}
		out = append(out, sc)
	}
	return out
}
