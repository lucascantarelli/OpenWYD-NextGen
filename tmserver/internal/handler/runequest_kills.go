package handler

import (
	"slices"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// Sala da Pista de Runas em jogo: os hooks de morte por sala (MobKilled.cpp:2158-2500,
// 2958-2986), os prêmios de saída Torre/Sulrang (ProcessSecMinTimer.cpp:1186-1416),
// os portais do Balrog (:254-514), o contador do Coelho (:536, 1856, 2004, 2049),
// as armadilhas do Labirinto (:1715-1804), a guarda da sala Amon (_MSG_Action.cpp:177-189)
// e a passagem de liderança (Server.cpp:8205-8220).
//
// Bugs de origem corrigidos (o stream do rand() não muda com cada correção): o prêmio
// do grupo 3 da Torre dava a runa do líder ao líder do grupo 1
// (ProcessSecMinTimer.cpp:1275); o ticket do grupo 2 do Sulrang ia para o
// Party[3] fora dos limites (:1373); a varredura de líder das armadilhas testava Party[1] duas vezes e
// nunca Party[2] (SendFunc.cpp:588, Server.cpp:6111).
//
// O timer original roda a cada 500ms (Server.cpp:4086) e o tick do mundo é
// 1s, então as checagens por tick abaixo rodam na metade da taxa da origem: o portal
// do Balrog sorteia metade dos rand() por segundo, SecCounter%12 é 6 ticks e
// SecCounter%5 (2,5s) é arredondado para 3 ticks.

// Blocos do gerador da pista de runas (Basedef.h:431-463).
const (
	runeQuestLich1          = 5653
	runeQuestLich2          = 5654
	runeQuestTower1         = 5706
	runeQuestTowerMobFirst  = 5709
	runeQuestTowerMobLast   = 5764
	runeQuestAmonBoss       = 5789
	runeQuestAmonMobFirst   = 5790
	runeQuestAmonMobLast    = 5848
	runeQuestKalintzFirst   = 5948
	runeQuestKalintzLast    = 5955
	runeQuestSulrangFirst   = 5972
	runeQuestSulrangLast    = 5975
	runeQuestLabyrinthFirst = 5854
	runeQuestLabyrinthLast  = 5898
	runeQuestLabyrinthBoss  = 5849
	runeQuestBalrogBoss     = 5899
	runeQuestCoelhoFirst    = 5775
	runeQuestCoelhoLast     = 5785
	runeQuestCoelhoFirst2   = 5768
	runeQuestCoelhoLast2    = 5774
	runeQuestCoelhoBoss     = 5767
)

// Índices de sala em RuneQuestState.Rooms.
const (
	runeRoomLich = iota
	runeRoomTorre
	runeRoomAmon
	runeRoomSulrang
	runeRoomLabyrinth
	runeRoomBalrog
	runeRoomCoelho
)

const (
	// runeQuestKalintzRoll é o sorteio rand()%7 do chefe; 5955 nunca é sorteado.
	runeQuestKalintzRoll = 7
	// runeQuestCoelhoStart é o contador que a entrada define para a sala do Coelho.
	runeQuestCoelhoStart = 100
	// itemSementeDeCristal é a tarifa do portal do Balrog, uma por salto.
	itemSementeDeCristal = 4032

	runeQuestCoelhoPeriod = 6 // ticks: SecCounter%12 do timer de 500ms
	runeQuestTrapPeriod   = 3 // ticks: SecCounter%5 do timer de 500ms, arredondado para cima
	runeQuestTrapDamage   = 2000
)

// Pouso do Labirinto depois que um grupo limpa seu último bloco (MobKilled.cpp:2323).
// Os membros do grupo 1 pousam uma casa a leste do líder no original.
const (
	runeQuestLabyrinthBossX = 3351
	runeQuestLabyrinthBossY = 1334
)

var (
	// runeQuestCoelhoBox é o retângulo SendSignalParmArea do contador
	// (3330,1475)-(3448,1525), cujas bordas superiores não são inclusivas.
	runeQuestCoelhoBox = areaBox{3330, 1475, 3448, 1525}
	// runeQuestCoelhoGate é onde os coelhos escapam (ProcessSecMinTimer.cpp:1856).
	runeQuestCoelhoGate = areaBox{3423, 1492, 3442, 1511}
	// runeQuestAmonBox é a câmara do chefe onde só líderes registrados podem entrar.
	runeQuestAmonBox = areaBox{3378, 1427, 3398, 1448}
)

// runeQuestPartyLeader é o `partyleader = Leader; if (<= 0) conn` da origem.
func runeQuestPartyLeader(e *world.Entity) int {
	if e.Leader > 0 {
		return e.Leader
	}
	return e.ID
}

func runeQuestPlaying(w *world.World, conn int) bool {
	if conn <= 0 || conn >= world.MaxUser {
		return false
	}
	mode, ok := w.SessionMode(conn)
	return ok && mode == world.UserPlay
}

// runeQuestPutItem é a guarda do PutItem (Server.cpp:1059-1065) na frente da
// entrega compartilhada de carry.
func (d *Dispatcher) runeQuestPutItem(w *world.World, conn int, it world.Item) {
	if !runeQuestPlaying(w, conn) {
		return
	}
	d.putMobDrop(w, w.Entity(conn), it)
}

func runeQuestTicket(sanc int) world.Item {
	return world.Item{Index: itemPistaDaRunas, Effects: [3]world.Effect{{Effect: efSanc, Value: uint8(sanc)}}}
}

// runeQuestReward paga uma sala limpa: uma runa para o líder e uma por
// membro jogando, depois o ticket da próxima sala para o líder (ticketSanc < 0
// para nenhum). A runa do líder é sorteada mesmo quando o líder não pode recebê-la,
// como na origem.
func (d *Dispatcher) runeQuestReward(w *world.World, leader, sala, ticketSanc int) {
	runes := world.RuneQuestRunes[sala]
	roll := func() world.Item { return world.Item{Index: runes[w.Rand().Intn(len(runes))]} }
	d.runeQuestPutItem(w, leader, roll())
	if le := w.Entity(leader); le != nil && world.IsPlayer(leader) {
		for _, member := range le.PartyList {
			if !runeQuestPlaying(w, member) {
				continue
			}
			d.runeQuestPutItem(w, member, roll())
		}
	}
	if ticketSanc >= 0 {
		d.runeQuestPutItem(w, leader, runeQuestTicket(ticketSanc))
	}
	name := ""
	if le := w.Entity(leader); le != nil {
		name = le.Name
	}
	d.log.Info("rune quest complete", "room", sala, "leader", name)
}

// teleportRuneQuestParty move um líder e seus membros jogando.
func (d *Dispatcher) teleportRuneQuestParty(w *world.World, leader int, x, y, memberX int16) {
	le := w.Entity(leader)
	if le == nil {
		return
	}
	if s := w.Session(leader); s != nil {
		d.doTeleport(w, s, x, y)
	}
	for _, member := range le.PartyList {
		if member == leader || !runeQuestPlaying(w, member) {
			continue
		}
		d.doTeleport(w, w.Session(member), memberX, y)
	}
}

func (d *Dispatcher) runeQuestGenerate(w *world.World, gen int) {
	d.revealSpawned(w, w.GenerateMob(gen))
}

// runeQuestMobKilled roda os hooks de sala de uma criatura morta por recompensa (um jogador)
// enquanto a criatura ainda conta no seu gerador (MobKilled.cpp:2158-2500).
func (d *Dispatcher) runeQuestMobKilled(w *world.World, reward, mob *world.Entity) {
	gen := int(mob.GenIndex)
	if gen < runeQuestLich1 || gen > runeQuestSulrangLast {
		return
	}
	st := w.RuneQuestState()
	leader := runeQuestPartyLeader(reward)
	switch {
	case gen == runeQuestLich1 || gen == runeQuestLich2:
		roll := w.Rand().Intn(100)
		// Os dois desfechos apagam todo Lich, inclusive o que está morrendo.
		var lichs []int
		w.ForEachMob(func(id int, e *world.Entity) {
			if e.GenIndex == runeQuestLich1 || e.GenIndex == runeQuestLich2 {
				lichs = append(lichs, id)
			}
		})
		for _, id := range lichs {
			w.RemoveGeneratorMob(id, 3)
		}
		if roll < 20 {
			d.runeQuestReward(w, leader, runeRoomLich, 1)
		} else {
			d.runeQuestGenerate(w, runeQuestLich1)
			d.runeQuestGenerate(w, runeQuestLich1)
			d.runeQuestGenerate(w, runeQuestLich2)
			d.runeQuestGenerate(w, runeQuestLich2)
		}

	case gen >= runeQuestTower1 && gen <= runeQuestTower1+2:
		st.Rooms[runeRoomTorre].Party[gen-runeQuestTower1].MobCount = 0

	case gen >= runeQuestTowerMobFirst && gen <= runeQuestTowerMobLast:
		// Uma morte só pontua enquanto a torre do próprio grupo está de pé.
		room := &st.Rooms[runeRoomTorre]
		for t := range room.Party {
			g := w.GeneratorAt(runeQuestTower1 + t)
			if g != nil && g.CurrentNumMob >= 1 && room.Party[t].LeaderID == leader {
				room.Party[t].MobCount++
				break
			}
		}

	case gen == runeQuestAmonBoss:
		d.runeQuestReward(w, leader, runeRoomAmon, 3)

	case gen >= runeQuestSulrangFirst && gen <= runeQuestSulrangLast:
		p := &st.Rooms[runeRoomSulrang].Party[0]
		if p.LeaderID != 0 && p.MobCount == 0 {
			p.MobCount = 1
			d.runeQuestGenerate(w, runeQuestKalintzFirst+w.Rand().Intn(runeQuestKalintzRoll))
		}

	case gen >= runeQuestLabyrinthFirst && gen <= runeQuestLabyrinthLast:
		// Só a última criatura do bloco conta; a que morre ainda está nele.
		if g := w.GeneratorAt(gen); g == nil || g.CurrentNumMob != 1 {
			break
		}
		room := &st.Rooms[runeRoomLabyrinth]
		for t := range room.Party {
			p := &room.Party[t]
			if leader != p.LeaderID || room.Party[(t+1)%3].MobCount < 1 || room.Party[(t+2)%3].MobCount < 1 {
				continue
			}
			p.MobCount--
			if p.MobCount == 0 {
				memberX := int16(runeQuestLabyrinthBossX)
				if t == 0 {
					memberX++
				}
				d.teleportRuneQuestParty(w, leader, runeQuestLabyrinthBossX, runeQuestLabyrinthBossY, memberX)
				d.runeQuestGenerate(w, runeQuestLabyrinthBoss)
			}
			break
		}

	case gen == runeQuestLabyrinthBoss:
		d.runeQuestReward(w, leader, runeRoomLabyrinth, 5)

	case gen == runeQuestBalrogBoss:
		st.Rooms[runeRoomBalrog].Party[0].MobCount = 1
		d.runeQuestReward(w, leader, runeRoomBalrog, 6)

	case gen >= runeQuestCoelhoFirst && gen <= runeQuestCoelhoLast:
		p := &st.Rooms[runeRoomCoelho].Party[0]
		if p.MobCount == 1 {
			p.MobCount--
			d.runeQuestGenerate(w, runeQuestCoelhoBoss)
		} else if p.MobCount > 0 {
			p.MobCount--
		}

	case gen == runeQuestCoelhoBoss:
		d.runeQuestReward(w, leader, runeRoomCoelho, -1)
	}
	w.SetRuneQuestState(st)
}

// runeQuestMobRemoved roda os hooks que seguem o DeleteMob da morte
// (MobKilled.cpp:2958-2986): as salas Torre e Amon repõem uma criatura morta na
// hora, e cada morte de Kalintz gera a próxima e pontua para o grupo do
// matador. Blocos que a fila de respawn por morte já repõe (MinuteGenerate <= 0,
// não pertencentes a evento) são pulados para não serem gerados duas vezes.
func (d *Dispatcher) runeQuestMobRemoved(w *world.World, reward *world.Entity, gen int) {
	switch {
	case gen >= runeQuestTowerMobFirst && gen <= runeQuestTowerMobLast,
		gen >= runeQuestAmonMobFirst && gen <= runeQuestAmonMobLast:
		if g := w.GeneratorAt(gen); g != nil && (world.IsRuneQuestGenerator(gen) || g.MinuteGenerate > 0) {
			d.runeQuestGenerate(w, gen)
		}

	case gen >= runeQuestKalintzFirst && gen <= runeQuestKalintzLast:
		st := w.RuneQuestState()
		room := &st.Rooms[runeRoomSulrang]
		if room.Party[0].LeaderID == 0 || room.Party[0].MobCount == 0 {
			return
		}
		d.runeQuestGenerate(w, runeQuestKalintzFirst+w.Rand().Intn(runeQuestKalintzRoll))
		leader := runeQuestPartyLeader(reward)
		for t := range room.Party {
			if room.Party[t].LeaderID == leader {
				room.Party[t].MobCount++
				break
			}
		}
		w.SetRuneQuestState(st)
	}
}

// runeQuestPrizes paga os vencedores de Torre e Sulrang na saída, antes de
// os registros serem limpos (ProcessSecMinTimer.cpp:1186-1416).
func (d *Dispatcher) runeQuestPrizes(w *world.World, st *world.RuneQuestState) {
	d.runeQuestPrize(w, st, runeRoomTorre, 2)
	d.runeQuestPrize(w, st, runeRoomSulrang, 4)
}

// runeQuestPrize paga o grupo cujo score está estritamente acima dos outros dois, se
// seu líder ainda detém o nome registrado.
func (d *Dispatcher) runeQuestPrize(w *world.World, st *world.RuneQuestState, sala, ticketSanc int) {
	p := &st.Rooms[sala].Party
	for t := range p {
		if p[t].MobCount <= p[(t+1)%3].MobCount || p[t].MobCount <= p[(t+2)%3].MobCount {
			continue
		}
		if le := w.Entity(p[t].LeaderID); p[t].LeaderID != 0 && le != nil && le.Name == p[t].LeaderName {
			d.runeQuestReward(w, p[t].LeaderID, sala, ticketSanc)
		}
		return
	}
}

// tickRuneQuestRooms roda as checagens por segundo de sala da pista de runas.
func (d *Dispatcher) tickRuneQuestRooms(w *world.World) {
	d.tickRuneQuestBalrog(w)
	d.sweepRuneQuestCoelhoGate(w)
	if d.tickCount%runeQuestCoelhoPeriod == 0 {
		d.sendRuneQuestCoelhoCount(w)
	}
	if d.tickCount%runeQuestTrapPeriod == 0 {
		d.pulseRuneQuestTraps(w)
	}
}

// tickRuneQuestBalrog move um grupo do Balrog cujo líder está sobre um portal
// (ProcessSecMinTimer.cpp:254-514). Os líderes são visitados na ordem de conexão,
// como faz o laço de jogadores da origem.
func (d *Dispatcher) tickRuneQuestBalrog(w *world.World) {
	st := w.RuneQuestState()
	room := st.Rooms[runeRoomBalrog]
	order := []int{0, 1, 2}
	slices.SortStableFunc(order, func(a, b int) int { return room.Party[a].LeaderID - room.Party[b].LeaderID })
	for _, x := range order {
		p := room.Party[x]
		e := w.Entity(p.LeaderID)
		if p.LeaderID == 0 || !runeQuestPlaying(w, p.LeaderID) || e == nil || e.HP <= 0 || e.Name != p.LeaderName {
			continue
		}
		if e.Leader != 0 && e.Leader != -1 {
			continue
		}
		// Sorteado antes do teste da casa, então cada segundo na pista queima um.
		rd := w.Rand().Intn(3)
		px, py := e.X&^3, e.Y&^3
		i := slices.IndexFunc(world.RuneQuestBalrogPortals, func(pt world.RuneQuestBalrogPortal) bool {
			return pt.X == px && pt.Y == py
		})
		if i < 0 {
			continue
		}
		dest := world.RuneQuestBalrogPortals[i].Dest[rd]
		// O chefe nasce antes de a tarifa ser verificada.
		if dest == world.RuneQuestBalrogBossRoom && w.RuneQuestState().Rooms[runeRoomBalrog].Party[0].MobCount == 0 {
			d.runeQuestGenerate(w, runeQuestBalrogBoss)
		}
		slot := -1
		for inv := 0; inv < activeCarryLimit(e); inv++ {
			if e.Carry[inv].Index == itemSementeDeCristal {
				slot = inv
				break
			}
		}
		if slot < 0 {
			continue
		}
		e.Carry[slot] = world.Item{}
		d.sendSlot(w, w.Session(p.LeaderID), world.ItemPlaceCarry, slot, e.Carry[slot])
		pos := world.RuneQuestBalrogPos[dest]
		d.teleportRuneQuestParty(w, p.LeaderID, pos[0], pos[1], pos[0])
	}
}

// sendRuneQuestCoelhoCount mostra os coelhos restantes da sala do Coelho
// (SendSignalParmArea, ProcessSecMinTimer.cpp:536).
func (d *Dispatcher) sendRuneQuestCoelhoCount(w *world.World) {
	body := protocol.EncodeStandardParm(int32(w.RuneQuestState().Rooms[runeRoomCoelho].Party[0].MobCount))
	w.ForEachPlaying(-1, func(s *world.Session, e *world.Entity) {
		if runeQuestCoelhoBox.containsExclusive(e.X, e.Y) {
			w.SendTo(s, protocol.Header{Type: protocol.MsgMobLeft, ID: protocol.IDScene}, body)
		}
	})
}

// runeQuestCoelhoEscaped devolve um coelho ao contador quando ele sai
// da sala (ProcessSecMinTimer.cpp:1859-1862,2006-2010).
func runeQuestCoelhoEscaped(w *world.World, gen, first, last int) {
	if gen < first || gen > last {
		return
	}
	st := w.RuneQuestState()
	if p := &st.Rooms[runeRoomCoelho].Party[0]; p.MobCount < runeQuestCoelhoStart && p.MobCount != 0 {
		p.MobCount++
		w.SetRuneQuestState(st)
	}
}

// runeQuestRouteEnd é a metade do fim-de-rota (Processor 0x10000) da fuga:
// só os blocos do coelho principal contam ali.
func runeQuestRouteEnd(w *world.World, mob *world.Entity) {
	runeQuestCoelhoEscaped(w, int(mob.GenIndex), runeQuestCoelhoFirst, runeQuestCoelhoLast)
}

// sweepRuneQuestCoelhoGate apaga qualquer criatura que não seja chefe parada no
// portão de fuga; os segundos blocos de coelho devolvem um coelho ao contador antes
// (ProcessSecMinTimer.cpp:1856-1865,2049-2059).
func (d *Dispatcher) sweepRuneQuestCoelhoGate(w *world.World) {
	var gone []int
	w.ForEachMob(func(id int, e *world.Entity) {
		if e.GenIndex != runeQuestCoelhoBoss && runeQuestCoelhoGate.contains(e.X, e.Y) && e.Merchant == 0 && e.Summoner == 0 {
			gone = append(gone, id)
		}
	})
	for _, id := range gone {
		e := w.Entity(id)
		runeQuestCoelhoEscaped(w, int(e.GenIndex), runeQuestCoelhoFirst2, runeQuestCoelhoLast2)
		w.RemoveGeneratorMob(id, 3)
	}
}

// pulseRuneQuestTraps dispara as armadilhas do Labirinto: uma caixa com um líder
// do Labirinto registrado dentro é segura para todos; senão ela pisca e acerta
// todo jogador vivo ali dentro (SendEnvEffectLeader / SendDamageLeader).
func (d *Dispatcher) pulseRuneQuestTraps(w *world.World) {
	room := w.RuneQuestState().Rooms[runeRoomLabyrinth]
	isLeader := func(conn int) bool {
		for _, p := range room.Party {
			if p.LeaderID != 0 && p.LeaderID == conn {
				return true
			}
		}
		return false
	}
	for _, t := range world.RuneQuestTraps {
		box := areaBox{t[0], t[1], t[2], t[3]}
		occupied, guarded := false, false
		w.ForEachPlayer(func(s *world.Session, e *world.Entity) {
			if !box.containsExclusive(e.X, e.Y) {
				return
			}
			if isLeader(s.Conn) {
				guarded = true
			} else {
				occupied = true
			}
		})
		if guarded {
			continue
		}
		if occupied {
			sendEnvEffectBox(w, box)
		}
		w.ForEachPlaying(-1, func(s *world.Session, e *world.Entity) {
			if e.HP <= 0 || !box.contains(e.X, e.Y) {
				return
			}
			d.applyEnvDamage(w, s, e, min(runeQuestTrapDamage, e.HP-1))
		})
	}
}

// runeQuestAmonGuard recusa um movimento para dentro da câmara do chefe Amon a menos que quem se move
// seja um líder Amon registrado (_MSG_Action.cpp:177-189).
func (d *Dispatcher) runeQuestAmonGuard(w *world.World, s *world.Session, e *world.Entity, x, y int16) bool {
	if !runeQuestAmonBox.contains(x, y) {
		return false
	}
	if runeQuestPartyLeader(e) == s.Conn {
		for _, p := range w.RuneQuestState().Rooms[runeRoomAmon].Party {
			if p.LeaderID == s.Conn {
				return false
			}
		}
	}
	pos := world.RuneQuestEntryPos[runeRoomAmon][w.Rand().Intn(3)]
	d.doTeleport(w, s, pos[0], pos[1])
	return true
}

// transferRuneQuestLeader entrega todo registro detido pelo líder que sai
// ao novo líder (Server.cpp:8205-8220).
func transferRuneQuestLeader(w *world.World, oldConn int, oldName string, newConn int, newName string) {
	st := w.RuneQuestState()
	changed := false
	for sala := range st.Rooms {
		for t := range st.Rooms[sala].Party {
			p := &st.Rooms[sala].Party[t]
			if p.LeaderID == oldConn && p.LeaderName == oldName {
				p.LeaderID, p.LeaderName = newConn, newName
				changed = true
			}
		}
	}
	if changed {
		w.SetRuneQuestState(st)
	}
}
