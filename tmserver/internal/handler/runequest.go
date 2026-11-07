package handler

import (
	"fmt"
	"time"

	"openwyd-nextgen/tmserver/internal/world"
)

// Pista de Runas (rune track): o registro no Uxmal (_MSG_Quest.cpp:1313-1395)
// e os timers de minuto de entrada/saída (ProcessSecMinTimer.cpp:1082-1429). O
// jogo da sala (mortes, bosses, prêmios, portais, contador, armadilhas) está em runequest_kills.go.

// itemPistaDaRunas é o ticket da pista de runas (ItemList.csv:6223 "Pista_da_Runas").
// Seu sanc seleciona a sala.
const itemPistaDaRunas = 5134

// uxmalMerchant é o MOB.Merchant de topo do Uxmal (byte 17 do template), o valor
// que _MSG_Quest.cpp:91 usa para rotear. CurrentScore.Merchant é 104, compartilhado com os
// treinadores, então não identifica o Uxmal.
const uxmalMerchant = 72

// Textos da pista de runas (tmserver/content/TMsrv/run/Language.txt).
const (
	msgRuneQuestBusy     = "Quest em progresso."                // _NN_Night_Already
	msgRuneQuestLeader   = "Uso restrito ao l\xedder do grupo." // _NN_Party_Leader_Only
	msgRuneQuestBring    = "Voc\xea deve trazer o item %s."     // _SN_BRINGITEM
	msgRuneQuestFull     = "H\xe1 muitos jogadores."            // _NN_Night_Limited
	msgRuneQuestRegister = "Entrada registrada."                // _NN_TicketUsed
)

// Geometria da pista de runas: o tile de lobby em volta do Uxmal em que tanto o líder quanto
// os membros precisam ficar na entrada, e o ponto de pouso da saída. O original
// sorteia rand()%1 entre y 1701 e 1686, que sempre escolhe 1701.
const (
	runeQuestLobbyTileX = 25
	runeQuestLobbyTileY = 13
	runeQuestExitX      = 3294
	runeQuestExitY      = 1701
)

// runeQuestRound é o período do evento; a entrada é no começo, a saída 15 minutos depois,
// e o Uxmal só aceita registros nos últimos 4 minutos.
const (
	runeQuestRound     = 20 * time.Minute
	runeQuestExitAfter = 15 * time.Minute
	runeQuestOpenFrom  = 16 // minuto dentro da rodada
)

func isUxmalNPC(npc *world.Entity) bool {
	return npc != nil && len(npc.Template) > 17 && npc.Template[17] == uxmalMerchant
}

// runeQuestPeriod devolve o início da rodada local de 20 minutos que contém now.
func runeQuestPeriod(now time.Time) time.Time {
	now = now.In(time.Local)
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()/20*20, 0, 0, time.Local)
}

// uxmal registra o líder que clicou na sala escolhida pelo sanc do ticket
// (_MSG_Quest.cpp:1313-1395). As checagens rodam na ordem original.
func (d *Dispatcher) uxmal(w *world.World, s *world.Session, e *world.Entity) {
	d.tickRuneQuest(w)
	if d.now().In(time.Local).Minute()%20 < runeQuestOpenFrom {
		d.sendClientMessage(w, s, msgRuneQuestBusy)
		return
	}
	if e.Leader != 0 && e.Leader != -1 {
		d.sendClientMessage(w, s, msgRuneQuestLeader)
		return
	}
	slot := -1
	for i := 0; i < activeCarryLimit(e); i++ {
		if e.Carry[i].Index == itemPistaDaRunas {
			slot = i
			break
		}
	}
	if slot < 0 {
		d.sendClientMessage(w, s, fmt.Sprintf(msgRuneQuestBring, "Pista_da_Runas"))
		return
	}
	sala := min(itemSanc(e.Carry[slot]), world.RuneQuestRooms-1)

	st := w.RuneQuestState()
	room := &st.Rooms[sala]
	groups := 3
	if sala == 0 {
		groups = 2
	}
	free := -1
	for j := 0; j < groups; j++ {
		if room.Party[j].LeaderID == 0 {
			free = j
			break
		}
	}
	if free < 0 {
		d.sendClientMessage(w, s, msgRuneQuestFull)
		return
	}
	for _, p := range room.Party {
		if p.LeaderID == s.Conn {
			d.sendClientMessage(w, s, msgRuneQuestBusy)
			return
		}
	}

	d.log.Info("register RuneQuest", "type", sala, "conn", s.Conn, "name", e.Name, "level", e.Level)
	room.Party[free] = world.RuneQuestParty{LeaderID: s.Conn, LeaderName: e.Name, Sala: sala}
	w.SetRuneQuestState(st)

	e.Carry[slot] = world.Item{}
	d.sendSlot(w, s, world.ItemPlaceCarry, slot, e.Carry[slot])
	d.sendClientMessage(w, s, msgRuneQuestRegister)
}

// tickRuneQuest dispara a entrada no início de cada rodada de 20 minutos e a
// saída 15 minutos depois, uma vez por rodada. O original dispara no segundo exato;
// as flags por rodada impedem um tick atrasado de perder ou repetir qualquer passo.
func (d *Dispatcher) tickRuneQuest(w *world.World) {
	now := d.now()
	period := runeQuestPeriod(now)
	elapsed := now.Sub(period)
	st := w.RuneQuestState()
	switch {
	case !st.Initialized:
		// Um restart não pode saber quem foi admitido numa rodada em andamento, então ele
		// só entra na linha do tempo; a saída ainda roda se ela ainda estiver à frente.
		st = world.RuneQuestState{Initialized: true, PeriodUnix: period.Unix(), Entered: true, Exited: elapsed >= runeQuestExitAfter}
	case period.Unix() < st.PeriodUnix:
		// Uma correção de relógio para trás não pode reexecutar uma rodada concluída.
		return
	case period.Unix() != st.PeriodUnix:
		st.PeriodUnix, st.Entered, st.Exited = period.Unix(), false, false
	}
	if !st.Entered {
		st.Entered = true
		// Uma parada além do horário de saída pula a entrada em vez de mandar
		// jogadores para dentro só para puxá-los de volta na hora.
		if elapsed < runeQuestExitAfter {
			d.runeQuestEnter(w, &st)
		}
	}
	if !st.Exited && elapsed >= runeQuestExitAfter {
		st.Exited = true
		d.runeQuestExit(w, &st)
	}
	w.SetRuneQuestState(st)
}

// runeQuestEnter teletransporta todo grupo registrado ainda válido para sua sala
// e gera os mobs de evento da sala (ProcessSecMinTimer.cpp:1082-1148).
func (d *Dispatcher) runeQuestEnter(w *world.World, st *world.RuneQuestState) {
	for t := range st.Rooms[4].Party {
		st.Rooms[4].Party[t].MobCount = 1
	}
	st.Rooms[runeRoomCoelho].Party[0].MobCount = runeQuestCoelhoStart

	for sala := range st.Rooms {
		for t := range st.Rooms[sala].Party {
			p := &st.Rooms[sala].Party[t]
			if p.LeaderID == 0 {
				continue
			}
			leader, ls := w.Entity(p.LeaderID), w.Session(p.LeaderID)
			if leader == nil || ls == nil || leader.Name != p.LeaderName {
				continue
			}
			if !inRuneQuestLobby(leader) || (leader.Leader != 0 && leader.Leader != -1) {
				continue
			}
			pos := world.RuneQuestEntryPos[p.Sala][t]
			d.doTeleport(w, ls, pos[0], pos[1])
			for _, id := range leader.PartyList {
				if id <= 0 || id >= world.MaxUser || id == p.LeaderID {
					continue
				}
				member, ms := w.Entity(id), w.Session(id)
				if member == nil || ms == nil || ms.Mode != world.UserPlay || !inRuneQuestLobby(member) {
					continue
				}
				d.doTeleport(w, ms, pos[0], pos[1])
			}
			d.runeQuestSpawn(w, p, sala, t)
		}
	}
	d.log.Info("rune quest entry")
}

// runeQuestSpawn gera os mobs de um grupo admitido, na ordem do original.
func (d *Dispatcher) runeQuestSpawn(w *world.World, p *world.RuneQuestParty, sala, t int) {
	spawn := func(first, last int) {
		for idx := first; idx <= last; idx++ {
			d.revealSpawned(w, w.GenerateMob(idx))
		}
	}
	switch sala {
	case 0: // Lich
		switch t {
		case 0:
			spawn(runeQuestLich2, runeQuestLich2)
		case 1:
			spawn(runeQuestLich1, runeQuestLich1)
		}
	case 1: // Torre: as três torres, depois os mobs da sala
		spawn(runeQuestTower1, runeQuestTower1+2)
		spawn(runeQuestTowerMobFirst, runeQuestTowerMobLast)
	case 2: // boss Amon
		spawn(runeQuestAmonBoss, runeQuestAmonBoss)
	case 4: // Labirinto
		p.MobCount = 8 + w.Rand().Intn(8)
		if t == 0 {
			spawn(runeQuestLabyrinthFirst, runeQuestLabyrinthLast)
		}
	}
}

// runeQuestExit esvazia as salas, revive e devolve todo jogador que está dentro,
// paga os vencedores de Torre/Sulrang e libera todos os registros
// (ProcessSecMinTimer.cpp:1150-1429).
func (d *Dispatcher) runeQuestExit(w *world.World, st *world.RuneQuestState) {
	world.ForEachRuneQuestGenerator(w.ClearGenerator)
	// O original apaga todo mob da caixa, incluindo a população permanente
	// de MinuteGenerate, que o timer de minuto então repõe.
	w.ForEachMob(func(id int, e *world.Entity) {
		if world.RuneQuestArea(e.X, e.Y) {
			w.DespawnMob(id, 0)
		}
	})
	w.ForEachPlayer(func(s *world.Session, e *world.Entity) {
		if !world.RuneQuestArea(e.X, e.Y) {
			return
		}
		if e.HP <= 0 {
			e.HP = 1
			s.ReqHp = 1
			d.sendScore(w, s, e)
			d.sendSetHpMp(w, s, e)
		}
		d.doTeleport(w, s, runeQuestExitX, runeQuestExitY)
	})
	d.runeQuestPrizes(w, st)
	st.Rooms = [world.RuneQuestRooms]world.RuneQuestRoom{}
	d.log.Info("rune quest exit")
}

func inRuneQuestLobby(e *world.Entity) bool {
	return e.X/128 == runeQuestLobbyTileX && e.Y/128 == runeQuestLobbyTileY
}
