package handler

import (
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// partyDif é a diferença máxima de nível permitida numa party (PARTY_DIF).
//
// UNVERIFIED: o valor e os ajustes de tier por ClassMaster/MAX_CLEVEL
// (lote2-party-guilda-guerra.md) não estão documentados; placeholder + simplificado.
const partyDif = 100

// partyLevelOK aplica a regra (simplificada) de nível de party compartilhada por convite e
// aceite: mesmo tier, nível muito alto, ou dentro de partyDif (tiers UNVERIFIED).
func partyLevelOK(a, b *world.Entity) bool {
	if a.Level >= 1000 || b.Level >= 1000 || a.ClassMaster == b.ClassMaster {
		return true
	}
	diff := int(a.Level) - int(b.Level)
	if diff < 0 {
		diff = -diff
	}
	return diff < partyDif
}

// sendReqParty trata _MSG_SendReqParty (0x037F): convida um jogador para uma party.
func (d *Dispatcher) sendReqParty(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		return
	}
	var body protocol.MsgSendReqPartyBody
	if err := body.Decode(payload); err != nil {
		// Um retorno silencioso aqui é o que tornava "o convite não faz nada" impossível de
		// diagnosticar: o pacote chegava, era roteável e não produzia reação.
		d.log.Warn("party invite: bad packet", "conn", s.Conn, "len", len(payload), "err", err)
		return
	}
	if int(body.PartyID) != s.Conn { // PartyID precisa ser o convidador
		d.log.Warn("party invite: PartyID is not the inviter", "conn", s.Conn,
			"party_id", body.PartyID, "target", body.Unk, "len", len(payload))
		return
	}
	if e.Leader != 0 { // já é membro em outro lugar
		d.sendClientMessage(w, s, partyAlreadyIn)
		return
	}
	target := int(body.Unk)
	if target == 0 && body.Target != 0 {
		target = int(body.Target)
	}
	other := w.Session(target)
	te := w.Entity(target)
	if target <= 0 || target >= world.MaxUser || other == nil || other.Mode != world.UserPlay || te == nil {
		// O legado responde _NN_Not_Connected aqui; ficar em silêncio deixava o jogador
		// com um clique que não fazia nada (QA 05/11).
		d.log.Warn("party invite: target not in play", "conn", s.Conn, "target", target)
		d.sendClientMessage(w, s, partyTargetGone)
		return
	}
	// O legado distingue os dois conflitos de party pelo dono: o alvo é
	// já membro de alguém (_NN_Other_Partys_Member) contra já ser líder
	// (_NN_Have_Own_Party_Already). isInParty junta os dois, então diga qual é.
	if te.Leader != 0 {
		d.sendClientMessage(w, s, partyOtherMember)
		return
	}
	if partyMemberCount(te) != 0 {
		d.sendClientMessage(w, s, partyOwnGroup)
		return
	}
	if !partyLevelOK(e, te) {
		d.sendClientMessage(w, s, partyLevelTooFar)
		return
	}
	te.LastReqParty = s.Conn // gate anti-forja do AcceptParty
	out := d.reqPartyBody(e, s.Conn)
	w.SendTo(other, protocol.Header{Type: protocol.MsgSendReqParty, ID: protocol.IDScene}, out.Encode())
}

// acceptParty trata _MSG_AcceptParty (0x03AB): entra na party do convidador. O
// gate LastReqParty bloqueia aceites forjados (PARTYHACK).
func (d *Dispatcher) acceptParty(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || s.Mode != world.UserPlay {
		return
	}
	var body protocol.MsgAcceptPartyBody
	if err := body.Decode(payload); err != nil {
		d.log.Warn("party accept: bad packet", "conn", s.Conn, "len", len(payload), "err", err)
		return
	}
	leaderConn := int(body.LeaderID)
	if leaderConn <= 0 || leaderConn >= world.MaxUser {
		return
	}
	if e.LastReqParty != leaderConn { // aceite forjado sem um convite correspondente
		d.log.Warn("PARTYHACK: accept without invite", "conn", s.Conn, "leader", leaderConn)
		w.AddCrackError(s, 1, 0)
		return
	}
	leaderSess, le := w.Session(leaderConn), w.Entity(leaderConn)
	if leaderSess == nil || leaderSess.Mode != world.UserPlay || le == nil {
		return
	}
	if cstr(body.MobName[:]) != le.Name {
		return
	}
	if isInParty(e) || le.Leader != 0 || !partyLevelOK(le, e) {
		return
	}
	slot, ok := addMember(le, s.Conn)
	if !ok {
		d.sendClientMessage(w, s, partyIsFull)
		return
	}
	e.Leader = leaderConn
	e.LastReqParty = 0
	d.syncAcceptedParty(w, leaderConn, s.Conn, slot+1)
}

// removeParty trata _MSG_RemoveParty (0x037E): membros saem, líderes podem
// expulsar um membro, e um líder saindo promove o próximo membro quando possível.
// Parm é o alvo; fora da faixa ou ele mesmo significa "eu".
func (d *Dispatcher) removeParty(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	target := s.Conn
	if parm, ok := protocol.StandardParm(payload); ok && parm > 0 && int(parm) < world.MaxUser {
		target = int(parm)
	}
	if target != s.Conn && !hasMember(e, target) {
		target = s.Conn
	}
	if e.Leader != 0 {
		d.leaveParty(w, s.Conn)
		return
	}
	if target == s.Conn {
		d.leaderLeaveParty(w, s.Conn)
		return
	}
	d.kickPartyMember(w, s.Conn, target)
}

// SessionEnd desliga uma conexão que sai da sua party. Ele porta o
// RemoveParty(conn) que o legado roda no logout e no teardown do socket
// (_MSG_CharacterLogout.cpp:23, CloseUser → Server.cpp:7654), que o port
// estava faltando: quem saía ficava um fantasma no PartyList do líder, então aquele
// líder lia como "já em party" para sempre (isInParty) e um slot de conn reutilizado
// herdava as linhas. Registrado com world.SetSessionEndHandler e também chamado
// de characterLogout. Idempotente — a segunda execução encontra uma party vazia.
func (d *Dispatcher) SessionEnd(w *world.World, s *world.Session) {
	// Uma troca P2P morre com qualquer dos lados (CloseUser → RemoveTrade no legado):
	// o oponente recebe _MSG_QuitTrade em vez de uma janela ligada a um fantasma.
	if s.Trade.Active {
		d.removeTrade(w, s)
	}
	e := w.Entity(s.Conn)
	if e == nil || !isInParty(e) {
		return // nada a desligar; não empurrar pacote de party a um cliente sem party
	}
	if e.Leader != 0 {
		d.leaveParty(w, s.Conn)
		return
	}
	// Líderes (e donos de pet sozinhos) dissolvem; leaderLeaveParty promove o próximo
	// membro e ressincroniza, como o AcceptParty sintético em Server.cpp:8246-8260.
	d.leaderLeaveParty(w, s.Conn)
}

func (d *Dispatcher) reqPartyBody(e *world.Entity, partyID int) protocol.MsgSendReqPartyBody {
	body := protocol.MsgSendReqPartyBody{
		Class:    e.Class,
		PartyPos: 0,
		Level:    uint16(clampPartyWire(e.Level)),
		MaxHP:    uint16(partyHPWire(effectiveMaxHP(e))),
		HP:       uint16(partyHPWire(e.HP)),
		PartyID:  int16(partyID),
	}
	copy(body.MobName[:], e.Name)
	return body
}

func (d *Dispatcher) syncAcceptedParty(w *world.World, leaderConn, newMember, newMemberPartyPos int) {
	le := w.Entity(leaderConn)
	if le == nil {
		return
	}
	if partyMemberCount(le) == 1 {
		d.sendAddParty(w, leaderConn, leaderConn, 0)
		d.sendAddParty(w, newMember, newMember, newMemberPartyPos)
	}
	d.sendAddParty(w, newMember, leaderConn, 0)
	d.sendAddParty(w, leaderConn, newMember, newMemberPartyPos)
	for j, memberID := range le.PartyList {
		if memberID == 0 {
			continue
		}
		if memberID != newMember {
			d.sendAddParty(w, newMember, memberID, newMemberPartyPos)
		}
		d.sendAddParty(w, memberID, newMember, j+1)
	}
}

func (d *Dispatcher) leaveParty(w *world.World, conn int) {
	e := w.Entity(conn)
	if e == nil {
		return
	}
	leaderConn := e.Leader
	// As invocações de um membro ocupam slots no PartyList do LÍDER (summon.go:205),
	// então cortar o vínculo sem colhê-las as deixa vivas e rastreadas por ninguém —
	// a próxima evocação então conta zero pets e gera um conjunto todo novo (#234).
	d.despawnSummonsOf(w, leaderConn, conn)
	d.despawnSummonsOf(w, conn, 0)
	e.Leader = 0
	e.PartyList = [world.MaxParty]int{}
	d.sendRemoveParty(w, conn, 0)
	if le := w.Entity(leaderConn); le != nil {
		removeMember(le, conn)
		for _, m := range le.PartyList {
			if m == 0 {
				continue
			}
			d.sendRemoveParty(w, m, conn)
		}
		d.sendRemoveParty(w, leaderConn, conn)
	}
}

func (d *Dispatcher) kickPartyMember(w *world.World, leaderConn, conn int) {
	e := w.Entity(conn)
	le := w.Entity(leaderConn)
	if e == nil || le == nil || e.Leader != leaderConn {
		return
	}
	d.despawnSummonsOf(w, leaderConn, conn) // mesma colheita do leaveParty (#234)
	d.despawnSummonsOf(w, conn, 0)
	e.Leader = 0
	e.PartyList = [world.MaxParty]int{}
	removeMember(le, conn)
	d.sendRemoveParty(w, conn, 0)
	d.sendRemoveParty(w, leaderConn, conn)
	for _, m := range le.PartyList {
		if m != 0 {
			d.sendRemoveParty(w, m, conn)
		}
	}
}

func (d *Dispatcher) leaderLeaveParty(w *world.World, leaderConn int) {
	le := w.Entity(leaderConn)
	if le == nil {
		return
	}
	// Toda invocação da lista vai, quem quer que a tenha evocado. O legado só apaga
	// as do próprio saído e apenas zera Summoner nas outras (Server.cpp:8237-8243),
	// confiando na sua varredura global de Type-24 para colhê-las; aqui o tempo de vida tiquetaqueia
	// dentro do summonTick, que é barrado por Summoner != 0 (mobai.go:61), então copiar
	// isso vazaria mobs permanentes sem dono. Este ramo é também o caminho do BM
	// sozinho — um dono de pet tem Leader == 0, então o RemoveParty cai aqui (#234).
	d.despawnSummonsOf(w, leaderConn, 0)
	members := partyMembers(le)
	le.PartyList = [world.MaxParty]int{}
	for _, memberID := range members {
		if me := w.Entity(memberID); me != nil && me.Leader == leaderConn {
			me.Leader = 0
			me.PartyList = [world.MaxParty]int{}
		}
		d.sendRemoveParty(w, memberID, 0)
	}
	d.sendRemoveParty(w, leaderConn, 0)
	if len(members) == 0 {
		return
	}
	newLeader := members[0]
	nle := w.Entity(newLeader)
	if nle == nil {
		return
	}
	transferRuneQuestLeader(w, leaderConn, le.Name, newLeader, nle.Name)
	nle.Leader = 0
	nle.PartyList = [world.MaxParty]int{}
	for _, memberID := range members[1:] {
		me := w.Entity(memberID)
		if me == nil {
			continue
		}
		slot, ok := addMember(nle, memberID)
		if !ok {
			return
		}
		me.Leader = newLeader
		d.syncAcceptedParty(w, newLeader, memberID, slot+1)
	}
}

func (d *Dispatcher) sendRemoveParty(w *world.World, conn, connExit int) {
	if conn <= 0 || conn >= world.MaxUser {
		return
	}
	if ms := w.Session(conn); ms != nil && ms.Mode == world.UserPlay {
		body := protocol.MsgRemovePartyBody{LeaderConn: int16(connExit)}
		w.SendTo(ms, protocol.Header{Type: protocol.MsgRemoveParty, ID: protocol.IDScene}, body.Encode())
	}
}

// addMember põe conn no primeiro slot livre de party do líder (idempotente).
func addMember(leader *world.Entity, conn int) (int, bool) {
	for i := range leader.PartyList {
		if leader.PartyList[i] == conn {
			return i, true
		}
		if leader.PartyList[i] == 0 {
			leader.PartyList[i] = conn
			return i, true
		}
	}
	return -1, false
}

func removeMember(leader *world.Entity, conn int) {
	for i := range leader.PartyList {
		if leader.PartyList[i] == conn {
			leader.PartyList[i] = 0
		}
	}
}

func hasMember(leader *world.Entity, conn int) bool {
	for _, memberID := range leader.PartyList {
		if memberID == conn {
			return true
		}
	}
	return false
}

func partyMemberCount(leader *world.Entity) int {
	n := 0
	for _, memberID := range leader.PartyList {
		if memberID != 0 {
			n++
		}
	}
	return n
}

func partyMembers(leader *world.Entity) []int {
	members := make([]int, 0, world.MaxParty)
	for _, memberID := range leader.PartyList {
		if memberID > 0 && memberID < world.MaxUser {
			members = append(members, memberID)
		}
	}
	return members
}

func isInParty(e *world.Entity) bool {
	return e.Leader != 0 || partyMemberCount(e) != 0
}

func (d *Dispatcher) sendAddParty(w *world.World, recipientID, entryID, partyPos int) {
	if recipientID <= 0 || recipientID >= world.MaxUser {
		return
	}
	rs := w.Session(recipientID)
	entry := w.Entity(entryID)
	if rs == nil || rs.Mode != world.UserPlay || entry == nil {
		return
	}
	leaderConn := uint16(30000)
	if partyPos == 0 {
		leaderConn = uint16(entryID)
	}
	body := protocol.MsgCNFAddPartyBody{
		LeaderConn: leaderConn,
		Level:      uint16(clampPartyWire(entry.Level)),
		MaxHP:      uint16(partyHPWire(effectiveMaxHP(entry))),
		HP:         uint16(partyHPWire(entry.HP)),
		PartyID:    uint16(entryID),
		Target:     52428,
	}
	copy(body.MobName[:], entry.Name)
	w.SendTo(rs, protocol.Header{Type: protocol.MsgCNFAddParty, ID: protocol.IDScene}, body.Encode())
}

func partyHPWire(v int32) int32 {
	if v < 0 {
		return 0
	}
	if v > 32000 {
		v = (v + 1) / 100
	}
	return clampPartyWire(v)
}

func clampPartyWire(v int32) int32 {
	if v < 0 {
		return 0
	}
	if v > 0xFFFF {
		return 0xFFFF
	}
	return v
}
