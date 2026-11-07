package handler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// gmCommandTimeout limita a ida e volta ao dbServer fora do loop para ban/unban.
const gmCommandTimeout = 10 * time.Second

// runGMCommand é o bus de comandos de GM/moderação in-game (issue #122). É o
// substituto moderno do ProcessImple legado do imple.cpp: o cliente envia
// "/gm <sub> <args>" como sussurro para o alvo "gm", então args é a String do sussurro
// (o resto da linha digitada). A autoridade é o AccessLevel da sessão — derivado
// do account.role no login — NÃO o frágil "character Level >= 1000" legado.
//
// Todo comando é autorizado contra o tier de moderador e auditado (slog)
// antes do dispatch, espelhando o Log("adm ...") legado que precedia toda
// ação admin. Um comando negado é silencioso (como no original).
func (d *Dispatcher) runGMCommand(w *world.World, s *world.Session, args []byte) {
	if s.Mode != world.UserPlay {
		return
	}
	if s.AccessLevel < world.AccessModerator {
		// Negação silenciosa + auditoria da tentativa (um não-GM sondando o bus).
		d.log.Warn("gm command denied: not a moderator",
			"conn", s.Conn, "account", s.AccountName, "role", s.AccessLevel)
		return
	}
	line := strings.TrimSpace(cstr(args))
	if line == "" {
		return
	}
	sub := strings.ToLower(firstToken(line))
	rest := strings.TrimSpace(strings.TrimPrefix(line, firstToken(line)))

	// Audita ANTES do dispatch: conta, alvo/args e a autoridade que o executou.
	d.log.Info("gm command",
		"account", s.AccountName, "id", s.AccountID, "role", s.AccessLevel,
		"cmd", sub, "args", rest)

	switch sub {
	case "kick":
		d.gmKick(w, s, rest)
	case "notice", "aviso":
		d.gmNotice(w, s, rest)
	case "goto", "ir":
		d.gmGoto(w, s, rest)
	case "summon", "puxar":
		d.gmSummon(w, s, rest)
	case "spawn":
		d.gmSpawn(w, s, rest)
	case "item":
		d.gmItem(w, s, rest)
	case "setlevel":
		d.gmSetLevel(w, s, rest)
	case "setgold":
		d.gmSetGold(w, s, rest)
	case "ban":
		d.gmBan(w, s, rest, true)
	case "unban":
		d.gmBan(w, s, rest, false)
	case "guildname":
		d.gmSetGuildName(w, s, rest)
	case "guildfame":
		d.gmSetGuildFame(w, s, rest)
	case "weather", "clima":
		d.gmWeather(w, s, rest)
	default:
		d.log.Warn("gm command: unknown subcommand", "account", s.AccountName, "cmd", sub)
	}
}

// gmKick desconecta um jogador online pelo nome do personagem. Espelha a guarda legada
// (imple.cpp:1824): um moderador não pode expulsar outro moderador/admin de tier
// igual ou superior.
func (d *Dispatcher) gmKick(w *world.World, s *world.Session, rest string) {
	name := firstToken(rest)
	if name == "" {
		return
	}
	target, _ := w.SessionByName(name)
	if target == nil {
		d.notify(w, s, NoticeNotConnected)
		return
	}
	if target.AccessLevel >= s.AccessLevel {
		d.log.Warn("gm kick denied: target outranks caller",
			"account", s.AccountName, "target", name, "targetRole", target.AccessLevel)
		return
	}
	d.log.Info("gm kick", "account", s.AccountName, "target", name, "targetConn", target.Conn)
	w.Close(target) // teardown completo + save do personagem (removeSession)
}

// gmNotice transmite um anúncio para o servidor inteiro a todo jogador em jogo.
//
// UNVERIFIED: o wire dourado dedicado do anúncio (o aviso grande centralizado) não está
// capturado (notice.go §UNVERIFIED). Como primeira versão sai como uma linha de chat
// prefixada com "[GM]", entregue a todos os jogadores; o formato especial do aviso é
// fixado quando existir uma captura.
func (d *Dispatcher) gmNotice(w *world.World, s *world.Session, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	msg := "[GM] " + text
	payload := protocol.EncodeMessageChatBody(msg)
	// HEADER.ID = a conn do GM: o cliente renderiza como uma linha de chat (a origem
	// do anúncio). Sem filtro de distância — ForEachPlaying(-1) alcança toda sessão.
	w.ForEachPlaying(-1, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgMessageChat, ID: uint16(s.Conn)}, payload)
	})
	d.log.Info("gm notice", "account", s.AccountName, "text", text)
}

// gmGoto teletransporta quem chamou para um jogador online de nome dado.
func (d *Dispatcher) gmGoto(w *world.World, s *world.Session, rest string) {
	name := firstToken(rest)
	if name == "" {
		return
	}
	_, te := w.SessionByName(name)
	if te == nil {
		d.notify(w, s, NoticeNotConnected)
		return
	}
	d.doTeleport(w, s, te.X, te.Y)
}

// gmSummon puxa um jogador online de nome dado para a posição de quem chamou.
func (d *Dispatcher) gmSummon(w *world.World, s *world.Session, rest string) {
	name := firstToken(rest)
	if name == "" {
		return
	}
	target, _ := w.SessionByName(name)
	if target == nil {
		d.notify(w, s, NoticeNotConnected)
		return
	}
	ce := w.Entity(s.Conn)
	if ce == nil {
		return
	}
	d.doTeleport(w, target, ce.X, ce.Y) // move a sessão do ALVO para quem chamou
}

// gmSpawn gera uma criatura de teste na posição de quem chamou a partir da lista de
// invocação do BM (o único catálogo de template de mob indexado por ID mantido em memória; SummonMobs).
// O id é o índice da lista — por exemplo, os slots de Sleipnir/Svaldfire.
func (d *Dispatcher) gmSpawn(w *world.World, s *world.Session, rest string) {
	id, err := strconv.Atoi(firstToken(rest))
	if err != nil || id < 0 || id >= len(d.summonMobs) || d.summonMobs[id] == nil {
		d.log.Warn("gm spawn: no such template", "account", s.AccountName, "id", firstToken(rest))
		return
	}
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	newID := w.SpawnMobAt(world.MobSpawn{Template: d.summonMobs[id], X: e.X, Y: e.Y, GenIndex: -1})
	if newID < 0 {
		d.log.Warn("gm spawn: world full", "account", s.AccountName)
		return
	}
	d.revealSpawned(w, []int{newID})
	d.log.Info("gm spawn", "account", s.AccountName, "template", id, "mobConn", newID)
}

// gmItem concede um item de teste ao inventário de quem chamou: "/gm item <index> [qty]".
// A quantidade opcional grava EF_AMOUNT (limitada a [1, maxStackAmount]) para os caminhos
// de pilha — dividir, juntar, consumir catalisador — serem alcançáveis sem configurar um
// pacote de loja de NPC no portal web.
func (d *Dispatcher) gmItem(w *world.World, s *world.Session, rest string) {
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return
	}
	id, err := strconv.Atoi(fields[0])
	if err != nil || id <= 0 || id >= world.MaxItem {
		return
	}
	qty := 1
	if len(fields) > 1 {
		qty, err = strconv.Atoi(fields[1])
		if err != nil {
			return
		}
		if qty < 1 {
			qty = 1
		}
		if qty > maxStackAmount {
			qty = maxStackAmount
		}
	}
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	slot := firstEmptyAccessibleCarry(e)
	if slot < 0 {
		d.notify(w, s, NoticeNoEmptySlot)
		return
	}
	it := world.Item{Index: int16(id)}
	if qty > 1 {
		setItemAmount(&it, qty)
	}
	e.Carry[slot] = it
	w.Send(s, protocol.MsgCNFGetItem, slotPayload(slot))
	d.log.Info("gm item", "account", s.AccountName, "item", id, "amount", qty, "slot", slot)
}

// gmSetLevel sobe quem chamou ao nível dado para teste. Ele reusa o
// caminho de subida de nível (applyLevelUps), então só sobe — definir um nível igual ou abaixo
// do atual é no-op (documentado; um rebaixamento precisaria desenrolar o
// score derivado e está fora do escopo).
func (d *Dispatcher) gmSetLevel(w *world.World, s *world.Session, rest string) {
	n, err := strconv.Atoi(firstToken(rest))
	if err != nil {
		return
	}
	if n < 1 {
		n = 1
	}
	if int32(n) > level.MaxLevel {
		n = int(level.MaxLevel)
	}
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	if e.Exp < level.NextLevelExp(int32(n)-1) {
		e.Exp = level.NextLevelExp(int32(n) - 1)
	}
	d.applyLevelUpsWithoutCirclet(w, s, e)
	d.log.Info("gm setlevel", "account", s.AccountName, "target", n, "level", e.Level)
}

// gmSetGold define o ouro carregado de quem chamou para teste.
func (d *Dispatcher) gmSetGold(w *world.World, s *world.Session, rest string) {
	n, err := strconv.Atoi(firstToken(rest))
	if err != nil || n < 0 {
		return
	}
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	e.Coin = int32(n)
	d.sendEtc(w, s, e) // UpdateEtc carrega Coin
	d.log.Info("gm setgold", "account", s.AccountName, "gold", n)
}

// gmBan bane (blocked=true) ou desbane (blocked=false) uma conta. O argumento é
// um nome de personagem quando o jogador está online (o ban cai na conta daquele
// personagem e o jogador é expulso), senão é tomado como nome de conta.
// A flag blocked é persistida via dbServer fora do loop; o login já recusa
// contas bloqueadas (LOGIN_RESULT_BLOCKED), então um ban nega o relogin na hora.
func (d *Dispatcher) gmBan(w *world.World, s *world.Session, rest string, blocked bool) {
	name := firstToken(rest)
	if name == "" {
		return
	}
	// Resolve a conta: prefere a conta de um jogador online (minúsculas canônicas,
	// definidas no login); cai para tratar o argumento como nome de conta.
	accountName := strings.ToLower(name)
	if ts, _ := w.SessionByName(name); ts != nil {
		accountName = ts.AccountName
	}
	p := w.Persistence()
	action := "unban"
	if blocked {
		action = "ban"
	}
	d.log.Info("gm "+action, "account", s.AccountName, "target", name, "targetAccount", accountName)
	w.Go(s, func() func(*world.World, *world.Session) {
		ctx, cancel := context.WithTimeout(context.Background(), gmCommandTimeout)
		defer cancel()
		err := p.SetAccountBlocked(ctx, accountName, blocked)
		return func(w *world.World, s *world.Session) {
			if err != nil {
				d.log.Error("gm "+action+" failed", "account", s.AccountName, "target", accountName, "err", err)
				return
			}
			// Num ban bem-sucedido, expulsa o jogador se ainda estiver online (re-resolve — a
			// sessão pode ter mudado enquanto o RPC estava em voo).
			if blocked {
				if ts, _ := w.SessionByName(name); ts != nil {
					w.Close(ts)
				}
			}
		}
	})
}

// gmSetGuildName registra um nome de exibição para um id de guilda (issue #131): não
// existe fluxo in-game de criação de guilda ainda, então esta é a única forma de preencher
// o nome que /nick mostra, espelhando a ferramenta legada de fama de guilda só para GM
// (Source/Comandos GM.txt).
func (d *Dispatcher) gmSetGuildName(w *world.World, s *world.Session, rest string) {
	id, err := strconv.Atoi(firstToken(rest))
	if err != nil || id <= 0 || id > 0xFFFF {
		return
	}
	name := strings.TrimSpace(strings.TrimPrefix(rest, firstToken(rest)))
	if name == "" {
		return
	}
	w.SetGuildName(uint16(id), name)
	d.log.Info("gm guildname", "account", s.AccountName, "guild", id, "name", name)
}

// gmSetGuildFame define a fama de uma guilda (issue #131), espelhando a
// ferramenta "+guildfame set" legada só para GM (Source/Comandos GM.txt).
func (d *Dispatcher) gmSetGuildFame(w *world.World, s *world.Session, rest string) {
	fields := strings.Fields(rest)
	if len(fields) != 2 {
		return
	}
	id, err1 := strconv.Atoi(fields[0])
	fame, err2 := strconv.Atoi(fields[1])
	if err1 != nil || err2 != nil || id <= 0 || id > 0xFFFF {
		return
	}
	w.SetGuildFame(uint16(id), int32(fame))
	d.log.Info("gm guildfame", "account", s.AccountName, "guild", id, "fame", fame)
}
