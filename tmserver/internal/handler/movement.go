package handler

import (
	"encoding/binary"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// Janela de tick anti-speedhack (_MSG_Action.cpp): o movetime do cliente
// (HEADER.ClientTick) precisa ficar nesta faixa do tempo do servidor, senão
// AddCrackError e a ação é descartada (sem broadcast). A comparação funciona
// com qualquer base de relógio do servidor porque o cliente ressincroniza seu movetime a partir dos
// ticks carimbados pelo servidor que recebe (CPSock.cpp:541 carimba CurrentTime em todo
// frame de saída; nosso enqueue espelha isso).
const (
	moveFutureWindow = 15000  // movetime > now + 15000
	movePastWindow   = 120000 // movetime < now - 120000
)

// action trata _MSG_Action / _MSG_Action2 / _MSG_Action3 (0x036C/0366/0368),
// _MSG_Action.cpp. Aplica estado de jogo, vivacidade, a janela de tick
// anti-speedhack e limites de posição, cancela uma troca em andamento, e então move a
// entidade para o DESTINO dela (pMob.TargetX/Y — a posição que toda leitura legada
// usa) e roda a reconciliação de visão do GridMulticast (moveMulticast), que
// repassa o frame cru — preservando seu Type — às janelas de visão antiga∪nova.
//
// UNVERIFIED / adiado (não reproduzido aqui): o limite de Speed vs AttackRun&0xF
// (o legado limita e continua), a correção de salto >VIEWGRID (GetAction +
// crack(1,5)) e o redirecionamento de célula alvo ocupada (GetEmptyMobGrid/BASE_GetRoute).
func (d *Dispatcher) action(w *world.World, s *world.Session, h protocol.Header, payload []byte) {
	if s.Mode != world.UserPlay {
		return // SendHpMode no original; nenhum efeito no mundo
	}
	e := w.Entity(s.Conn)
	if e == nil || e.HP == 0 {
		w.AddCrackError(s, 5, 3) // agir morto
		return
	}
	// Mover cancela uma troca em andamento dos dois lados (_MSG_Action.cpp:41-50).
	if s.Trade.Active {
		d.removeTrade(w, s)
		return
	}
	var body protocol.MsgActionBody
	if err := body.Decode(payload); err != nil {
		return
	}
	if h.Type == protocol.MsgAction3 && !d.consumeIllusion(w, s, e, h.ClientTick) {
		return
	}
	if h.Type == protocol.MsgAction && !d.consumeActionAfterIllusion(w, s, h.ClientTick) {
		return
	}
	if !s.LoggedFirstAction {
		s.LoggedFirstAction = true
		d.log.Info("movement: first action after login",
			"conn", s.Conn,
			"type", h.Type,
			"client_tick", h.ClientTick,
			"server_x", e.X,
			"server_y", e.Y,
			"login_spawn_x", s.LoginSpawnX,
			"login_spawn_y", s.LoginSpawnY,
			"pos_x", body.PosX,
			"pos_y", body.PosY,
			"target_x", body.TargetX,
			"target_y", body.TargetY,
			"effect", body.Effect,
			"speed", body.Speed,
			"route", body.Route)
		if d.dropBootAutoWalk(w, s, e, h, body) {
			return
		}
	}

	// Anti-speedhack: o movetime precisa ficar dentro da janela do tempo do servidor.
	// Códigos de crack por legado: 102 para Action/Action2, 104 para Action3.
	mt, now := int64(h.ClientTick), int64(w.Now())
	if mt > now+moveFutureWindow || mt < now-movePastWindow {
		code := 102
		if h.Type == protocol.MsgAction3 {
			code = 104
		}
		w.AddCrackError(s, 1, code)
		return
	}

	// Limites: o destino precisa ficar dentro da grade do mundo (o legado descarta
	// TargetX/Y <= 0 ou >= 4096 em silêncio; mantemos a contabilidade de crack).
	dim := int16(w.GridDim())
	if outOfBounds(body.PosX, dim) || outOfBounds(body.PosY, dim) ||
		outOfBounds(body.TargetX, dim) || outOfBounds(body.TargetY, dim) {
		w.AddCrackError(s, 1, 100)
		return
	}
	if !d.castleMoveAllowed(s.Conn, body.TargetX, body.TargetY) {
		d.doTeleport(w, s, e.X, e.Y)
		return
	}
	if d.runeQuestAmonGuard(w, s, e, body.TargetX, body.TargetY) {
		return
	}

	// Movimento no-op: o legado só processa quando o destino difere da
	// posição (autoritativa pelo destino) atual.
	if body.TargetX == e.X && body.TargetY == e.Y {
		return
	}
	// Dentro do ramo de destino alterado, como o bloqueio de nível legado.
	if d.newbieZoneGuard(w, s, e, body.TargetX, body.TargetY) {
		return
	}

	oldX, oldY := e.X, e.Y
	// Autoritativo pelo destino, como o pMob.TargetX/Y legado (cauda do GridMulticast,
	// SendFunc.cpp:929-930): toda leitura do lado do servidor — centro do multicast, combate,
	// GetInView, NoViewMob — usa o destino, nunca o início da rota.
	w.SetEntityPos(s.Conn, body.TargetX, body.TargetY)
	e.Route = body.Route
	// Rastrear a última cidade em que o jogador está (para a regra de respawn por cidade).
	if city := world.Village(body.TargetX, body.TargetY); city >= 0 && city <= 3 {
		e.LastCity = int16(city)
	}
	// GridMulticast: deltas de criação/remoção de visão + frame cru (Type original) para
	// todos na janela de visão antiga ou nova.
	d.moveMulticast(w, s.Conn, oldX, oldY, h.Type, payload)
	if h.Type == protocol.MsgAction3 {
		w.SendTo(s, protocol.Header{Type: protocol.MsgAction3, ID: uint16(s.Conn)}, payload)
		d.sendSetHpMp(w, s, e)
	}
}

func outOfBounds(v, dim int16) bool { return v < 0 || v >= dim }

func (d *Dispatcher) consumeIllusion(w *world.World, s *world.Session, e *world.Entity, tick uint32) bool {
	if e.Class != 3 || e.LearnedSkill&2 == 0 {
		w.AddCrackError(s, 10, 28)
		return false
	}
	mana := 45
	if d.spells != nil {
		if sp, ok := d.spells.Get(73); ok {
			mana = sp.ManaSpent
		}
	}
	if e.MP < int32(mana) {
		d.sendSetHpMp(w, s, e)
		return false
	}
	e.MP -= int32(mana)
	s.ReqMp -= int32(mana)
	setReqMp(s, e)
	if tick != protocol.SkipCheckTick && s.LastIllusionTick != protocol.SkipCheckTick && tick < s.LastIllusionTick+900 {
		w.AddCrackError(s, 1, 105)
		return false
	}
	s.LastIllusionTick = tick
	return true
}

func (d *Dispatcher) consumeActionAfterIllusion(w *world.World, s *world.Session, tick uint32) bool {
	if tick != protocol.SkipCheckTick && s.LastIllusionTick != 0 && s.LastIllusionTick != protocol.SkipCheckTick && tick < s.LastIllusionTick+900 {
		w.AddCrackError(s, 1, 103)
		return false
	}
	return true
}

func (d *Dispatcher) dropBootAutoWalk(w *world.World, s *world.Session, e *world.Entity, h protocol.Header, body protocol.MsgActionBody) bool {
	if h.Type != protocol.MsgAction || body.Effect != 0 || body.Speed != int32(attackRunOf(e)&0x0F) {
		return false
	}
	if body.PosX != s.LoginSpawnX || body.PosY != s.LoginSpawnY || e.X != s.LoginSpawnX || e.Y != s.LoginSpawnY {
		return false
	}
	if body.TargetX == e.X && body.TargetY == e.Y {
		return false
	}
	if w.Now()-s.LoginTick > 3000 {
		return false
	}
	d.log.Warn("movement: dropping boot auto-walk",
		"conn", s.Conn,
		"from_x", body.PosX,
		"from_y", body.PosY,
		"target_x", body.TargetX,
		"target_y", body.TargetY)
	correction := protocol.MsgActionBody{PosX: e.X, PosY: e.Y, Effect: 1, Speed: 6, TargetX: e.X, TargetY: e.Y}
	w.SendTo(s, protocol.Header{Type: protocol.MsgAction3, ID: uint16(s.Conn)}, correction.Encode())
	return true
}

// motion trata _MSG_Motion (0x036A): emotes/animações, transmitidos a quem está em visão.
func (d *Dispatcher) motion(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if s.Mode != world.UserPlay || e == nil || e.HP == 0 {
		w.AddCrackError(s, 4, 6)
		return
	}
	w.BroadcastInView(s.Conn, protocol.MsgMotion, payload)
}

// changeCity trata _MSG_ChangeCity (0x0291): define a cidade de spawn, empacotada em
// bits nos bits 6-7 de MOB.Merchant (lote2-movimento.md). O layout de bits documentado é
// preservado; villageAt (BASE_GetVillage) é um placeholder.
func (d *Dispatcher) changeCity(w *world.World, s *world.Session, _ protocol.Header, _ []byte) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	if city := villageAt(e.X, e.Y); city >= 0 && city <= 4 {
		e.Merchant = (e.Merchant & 0x3F) | uint8(city<<6)
	}
}

// villageAt mapeia uma posição para um id de vila 0..4, ou -1.
//
// UNVERIFIED: BASE_GetVillage não está na fonte (lote2-movimento.md). Isto é
// um placeholder que devolve -1 (sem vila) até a tabela de regiões ser capturada.
func villageAt(int16, int16) int { return -1 }

// reqTeleport trata _MSG_ReqTeleport (0x0290): o cliente pisou num tile de teleporte
// (o pacote é só header). O destino + custo vêm da posição do
// jogador (world.TeleportDest / GetTeleportPosition). Com ouro suficiente ele debita
// e teleporta (_MSG_ReqTeleport.cpp).
func (d *Dispatcher) reqTeleport(w *world.World, s *world.Session, _ protocol.Header, _ []byte) {
	if s.Mode != world.UserPlay {
		return
	}
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 {
		return
	}
	// GetFunc.cpp:1007: a cidade só abre depois que o chefe é derrotado.
	if e.X&^3 == 2364 && e.Y&^3 == 3924 {
		st, loaded := w.KefraState()
		if !loaded || !st.Defeated {
			return
		}
		x := 3250 + int16(w.Rand().Intn(3))
		y := 1703 + int16(w.Rand().Intn(3))

		x, y, ok := d.freeCellAtOrNear(w, x, y)
		if !ok {
			return
		}

		d.doTeleport(w, s, x, y)
		return
	}
	// GetFunc.cpp:994: só a entrada do Salão gasta um ticket de Kefra. Manter
	// pedidos rejeitados fora do fluxo MSVC; entradas bem-sucedidas sorteiam X e depois Y.
	if e.X&^3 == 2364 && e.Y&^3 == 3892 {
		if e.KefraTicket <= 0 {
			return
		}
		x := 2364 + int16(w.Rand().Intn(3))
		y := 3906 + int16(w.Rand().Intn(3))

		x, y, ok := d.freeCellAtOrNear(w, x, y)
		if !ok {
			return
		}

		e.KefraTicket--
		d.sendKefraBalance(w, s, e.KefraTicket)
		d.doTeleport(w, s, x, y)
		return
	}
	// GetTeleportPosition só abre o portal Azran→Vale para a
	// Fada do Vale equipada (item 3916 em Equip[13]); preservar aquele gate legado.
	access := world.TeleportAccess{
		HasFairy:      e.Equip[fairyEquipSlot].Index == 3916,
		KefraDefeated: d.expEvents.KefraLive, // nome legado: true significa derrotado
	}
	destX, destY, cost, ok := world.TeleportDestWithAccess(e.X, e.Y, access, w.Rand())
	if !ok {
		return // nenhum tile de teleporte aqui
	}
	if cost > 0 && cost > e.Coin {
		d.sendClientMessage(w, s, notEnoughMoney) // _MSG_ReqTeleport.cpp:39-64
		return
	}

	// Manter o destino original quando ele é utilizável; senão achar uma célula
	// próxima desocupada e caminhável antes de cobrar pelo teleporte.
	// Noatum -> Nippleheim: manter o fallback dentro da área de pouso original.
	if e.X&^3 == 1052 && e.Y&^3 == 1708 {
		destX, destY, ok = d.freeTeleportCellInArea(
			w, destX, destY, 3650, 3110, 3652, 3112,
		)
	} else {
		destX, destY, ok = d.freeCellAtOrNear(w, destX, destY)
	}
	if !ok {
		return
	}

	if cost > 0 {
		e.Coin -= cost
		d.sendEtc(w, s, e)
	}
	d.doTeleport(w, s, destX, destY)
}

// freeTeleportCellInArea mantém o fallback de teleporte dentro da sua área de pouso.
func (d *Dispatcher) freeTeleportCellInArea(
	w *world.World,
	x, y, minX, minY, maxX, maxY int16,
) (int16, int16, bool) {
	if x >= minX && x <= maxX &&
		y >= minY && y <= maxY &&
		d.cellAvailable(w, x, y) {
		return x, y, true
	}

	for cy := minY; cy <= maxY; cy++ {
		for cx := minX; cx <= maxX; cx++ {
			if cx == x && cy == y {
				continue
			}
			if d.cellAvailable(w, cx, cy) {
				return cx, cy, true
			}
		}
	}

	return 0, 0, false
}

// doTeleport move o jogador para (x,y) e reconcilia a visibilidade (DoTeleport,
// Server.cpp): RemoveMob da visão antiga, um MSG_Action com Effect=1 para saltar
// o próprio avatar do jogador, e depois CreateMob para o entorno novo.
func (d *Dispatcher) doTeleport(w *world.World, s *world.Session, x, y int16) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	oldX, oldY := e.X, e.Y
	cmBefore := w.SentOfType(s, protocol.MsgCreateMob)
	rmBefore := w.SentOfType(s, protocol.MsgRemoveMob)
	w.SetEntityPos(s.Conn, x, y)
	if c := world.Village(x, y); c >= 0 && c <= 3 {
		e.LastCity = int16(c)
	}
	// Saltar o próprio avatar do jogador: MSG_Action, Effect=1 (PosX/Y antiga → TargetX/Y destino).
	body := protocol.MsgActionBody{PosX: oldX, PosY: oldY, Effect: 1, Speed: 2, TargetX: x, TargetY: y}
	payload := body.Encode()
	w.SendTo(s, protocol.Header{Type: protocol.MsgAction, ID: uint16(s.Conn)}, payload)
	// O DoTeleport passa pelo GridMulticast no legado: remove criaturas/NPCs da janela
	// antiga, cria entidades da janela nova e repassa o mesmo frame de salto aos
	// observadores, mas não reenvia o CreateMob/UpdateScore do próprio jogador.
	d.moveMulticast(w, s.Conn, oldX, oldY, protocol.MsgAction, payload)
	// Investigação de freeze (investigacao-freeze-cliente.md): os casos
	// se aglomeram logo após teleportes de cidade, e teleportes costumavam ser invisíveis nos
	// logs. Os deltas de CreateMob/RemoveMob dimensionam a rajada de revelação que o cliente
	// precisava absorver no destino.
	d.log.Info("teleport", "conn", s.Conn, "name", e.Name,
		"from_x", oldX, "from_y", oldY, "to_x", x, "to_y", y,
		"create_mobs", w.SentOfType(s, protocol.MsgCreateMob)-cmBefore,
		"remove_mobs", w.SentOfType(s, protocol.MsgRemoveMob)-rmBefore)
}

// noViewMob trata _MSG_NoViewMob (0x0369): o cliente pede para ressincronizar a
// visibilidade de uma entidade (Parm = id alvo). Port de _MSG_NoViewMob.cpp: um alvo
// vivo dentro do alcance de GetInView (±NoViewRange, mais frouxo que a janela
// do multicast) recebe um snapshot completo de CreateMob/CreateMobTrade + PKInfo; qualquer outra coisa
// — slot vazio, jogador fora de jogo, ou fora de alcance — recebe um RemoveMob. O
// CreateMob de payload nil que isto enviava fazia o cliente reconstruir o avatar
// a partir de um corpo truncado: o bug de "snap-back" no multiplayer.
func (d *Dispatcher) noViewMob(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	if s.Mode != world.UserPlay {
		return
	}
	id := standardParm(payload)
	if id <= 0 || id >= world.MaxMob {
		return
	}
	self := w.Entity(s.Conn)
	target := w.Entity(id)
	inPlay := target != nil && target.Mode != world.MobEmpty
	if inPlay && id < world.MaxUser {
		// Um jogador só existe para os outros enquanto em jogo (checagem pUser[MobID].Mode).
		ts := w.Session(id)
		inPlay = ts != nil && ts.Mode == world.UserPlay
	}
	if inPlay && self != nil && chebyshev(self.X, self.Y, target.X, target.Y) <= world.NoViewRange {
		w.MarkSeen(s, id)
		ty, body := d.createMobViewPacket(w, target, 1)
		w.SendTo(s, protocol.Header{Type: ty, ID: protocol.IDScene}, body)
		if id < world.MaxUser {
			// PKInfo só trafega sobre jogadores (SendPKInfo, SendFunc.cpp:1869).
			w.SendTo(s, protocol.Header{Type: protocol.MsgPKInfo, ID: uint16(id)}, protocol.EncodeStandardParm(pkInfoParm(target)))
		}
	} else {
		w.SendTo(s, protocol.Header{Type: protocol.MsgRemoveMob, ID: uint16(id)}, protocol.EncodeRemoveMobBody(0))
		w.UnmarkSeen(s, id)
	}
}

// standardParm lê o Parm int32 inicial de um corpo MSG_STANDARDPARM.
func standardParm(payload []byte) int {
	if len(payload) < 4 {
		return -1
	}
	return int(int32(binary.LittleEndian.Uint32(payload)))
}
