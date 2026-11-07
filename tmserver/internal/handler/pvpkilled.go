package handler

import (
	"fmt"

	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/world"
)

// Limites de sequência de mortes (GetFunc.cpp SetCurKill/SetTotKill).
const (
	maxCurKill = 200
	maxTotKill = 32767
)

// pkPointMin/pkPointMax são os limites legados do SetPKPoint (GetFunc.cpp:
// valor<1→1, valor>150→150); 0 é reservado para significar "nunca persistido" (ver
// completeCharacterLogin).
const (
	pkPointMin uint8 = 1
	pkPointMax uint8 = 150
)

// pvpKilled roda as penalidades de morte para um jogador morto por outro jogador
// (MobKilled.cpp "#pragma region PvP", issue #210). É um port em linha reta
// só das fórmulas confiavelmente extraíveis — o ramo PvP real do MobKilled.cpp
// está cheio do que parece artefato de descompilador (um `if` órfão, sem chaves,
// numa checagem de coordenada que parece engolir todo o bloco arena/exp/
// CurKill), então isto espelha o estilo linear limpo do mobKilled em vez de
// replicar aquele fluxo de controle.
//
// Geografia (issue #421): uma morte dentro de uma cidade ou de uma área de guerra de guilda (posição
// da vítima, BASE_GetArena/BASE_GetVillage) pega o ramo de arena legado — a
// vítima mantém a EXP e recebe _NN_In_Arena_No_Exp_Loss, e o matador paga
// a taxa de CP de arena (pvpLostPkZone). Em todo o resto vale o ramo de campo.
// O `if (killed_x == 1 && killed_y == 31)` órfão define ZoneUnk (seu corpo se
// perdeu na descompilação), então uma morte no bloco de mapa (1,31) — a arena de duelo —
// também pega o ramo de arena (mapPKZone, issue #427).
//
// Adiado/UNVERIFIED (documentado, não portado — ver o plano da issue #210):
// - Bypasses de Castle-siege/RvR-Tower AtWar — esses sistemas são eles próprios
// UNVERIFIED (tmserver/internal/war/tower.go); só o bypass de guerra de guilda é
// implementado (guildsAtWar).
// - SameClan (legado: clã 7 contra clã 8, um par de facções NPC de guerra de reino
// não documentado, NÃO "mesma guilda") — sem modelo em lugar nenhum deste repo, tratado como
// sempre falso: uma morte normal sempre incrementa a sequência do matador e reseta
// a da vítima.
// - A regra dos itens 548/549 de "triplicar a penalidade de PKPoint" — equipamento
// não identificado, não perseguido.
// - Perda de item na morte sob #ifdef PKDrop — desativada no build de referência
// (Basedef.h:89, `//#define PKDrop`); intencionalmente não portada.
// - Indireção de perda de EXP bancada por extra.Hold/DEADPOINT — sem análogo em
// lugar nenhum deste código; a perda de EXP da vítima abaixo é uma dedução
// IMEDIATA em vez disso (divergência documentada entre intenção e mecanismo).
func (d *Dispatcher) pvpKilled(w *world.World, killer, victim *world.Entity) {
	ks := w.Session(killer.ID)
	vs := w.Session(victim.ID)
	if world.PvPZone(victim.X, victim.Y) || mapPKZone(victim.X, victim.Y) {
		d.pvpKilledInZone(w, ks, vs, killer, victim)
	} else {
		d.pvpKilledInField(w, ks, vs, killer, victim)
	}

	// Contabilidade de sequência de mortes (SameClan tratado como sempre falso — ver o doc acima).
	if killer.CurKill < maxCurKill {
		killer.CurKill++
	}
	if killer.TotKill < maxTotKill {
		killer.TotKill++
	}
	victim.CurKill = 0

	if ks != nil {
		d.broadcastPKState(w, ks, killer)
	}
	if vs != nil {
		d.broadcastPKState(w, vs, victim)
	}
}

// mapPKZone informa se (x,y) está no bloco de mapa (1,31), o ZoneUnk legado
// (MobKilled.cpp:3095 `killed_x = TargetX / 128`, :3131). Lido ao pé da letra, o
// `if` sem corpo aninharia todo o bloco de arena dentro de (1,31) e silenciaria
// também a arena de Armia; a leitura adotada é que (1,31) define ZoneUnk, dentro do qual
// a arena de duelo (x 142–195, y 4007–4082) cai inteira.
func mapPKZone(x, y int16) bool {
	return x/128 == 1 && y/128 == 31
}

// pvpKilledInZone é o ramo de arena legado (MobKilled.cpp:3131-3228): sem perda
// de EXP, sem recuperação de PKPoint da vítima, e — a menos que a vítima já estivesse Guilty —
// o matador perde CP na taxa de arena, anunciado com _DD_PKPointMinus mesmo
// quando a perda arredonda para 0 (o legado envia isso dentro de `if (!killed_guilty)`).
// O ramo de arena não consulta AtWar.
func (d *Dispatcher) pvpKilledInZone(w *world.World, ks, vs *world.Session, killer, victim *world.Entity) {
	if vs != nil {
		d.sendClientMessage(w, vs, inArenaNoExpLoss)
	}
	if pkGuilty(victim) {
		return
	}
	lost := pvpLostPkZone(victim.PKPoint)
	killer.PKPoint = clampPKPoint(int(killer.PKPoint) + lost)
	if ks != nil {
		// O legado imprime o Lostpk negativo como está ("Menos_-9").
		d.sendClientMessage(w, ks, fmt.Sprintf(pkPointMinus, int(killer.PKPoint)-75, lost))
	}
}

// pvpKilledInField é o ramo de campo aberto: perda de EXP para a vítima, a
// taxa de CP de campo para o matador, e a recuperação de PKPoint da vítima.
func (d *Dispatcher) pvpKilledInField(w *world.World, ks, vs *world.Session, killer, victim *world.Entity) {
	atWar := d.guildsAtWar(killer.Guild, victim.Guild)

	if loss := pvpExpLoss(victim.Level, victim.ClassMaster, victim.PKPoint); loss > 0 {
		victim.Exp -= loss
		if victim.Exp < 0 {
			victim.Exp = 0
		}
		if vs != nil {
			d.sendChatText(w, vs, fmt.Sprintf("Voce perdeu %d pontos de experiencia", loss))
		}
	}

	if lost := pvpLostPk(victim.PKPoint, pkGuilty(victim), atWar); lost != 0 {
		killer.PKPoint = clampPKPoint(int(killer.PKPoint) + lost)
		if ks != nil {
			d.sendChatText(w, ks, notifyPKPointDelta(killer.PKPoint, lost))
		}
	}
	if victim.PKPoint < pkPointNeutral && !atWar {
		victim.PKPoint++
		if vs != nil {
			d.sendChatText(w, vs, notifyPKPointDelta(victim.PKPoint, 1))
		}
	}
}

// pvpExpLoss é a perda de EXP da vítima numa morte PvP (região "Lose EXP" do
// MobKilled.cpp): um divisor por nível do intervalo de exp entre o nível
// atual e o próximo da vítima, escalado por quão caótica a vítima já estava, e depois um
// /6 fixo.
func pvpExpLoss(victimLevel int32, victimClassMaster uint8, victimPKPoint uint8) int64 {
	nextExp := level.NextLevelExpTier(victimLevel, victimClassMaster)
	curExp := level.NextLevelExpTier(victimLevel-1, victimClassMaster)
	alpha := nextExp - curExp
	delta := alpha / 20
	switch {
	case victimLevel >= 250:
		delta = alpha / 100
	case victimLevel >= 200:
		delta = alpha / 85
	case victimLevel >= 150:
		delta = alpha / 70
	case victimLevel >= 100:
		delta = alpha / 55
	case victimLevel >= 90:
		delta = alpha / 50
	case victimLevel >= 80:
		delta = alpha / 45
	case victimLevel >= 70:
		delta = alpha / 40
	case victimLevel >= 60:
		delta = alpha / 35
	case victimLevel >= 50:
		delta = alpha / 30
	case victimLevel >= 40:
		delta = alpha / 25
	case victimLevel >= 30:
		delta = alpha / 22
	}
	if delta < 0 {
		delta = 0
	}
	if delta > 150000 {
		delta = 150000
	}
	if victimPKPoint > 10 && victimPKPoint <= 25 {
		delta *= 3
	} else {
		delta *= 5
	}
	return delta / 6
}

// pvpLostPk é a mudança de PKPoint do matador por aplicar uma morte PvP
// (MobKilled.cpp:3316-3334, "PK Drop - CP"): escalada por quão caótica a vítima
// já estava, limitada a [-9,0], e zerada por completo contra uma vítima
// já culpada ou durante uma guerra de guilda.
func pvpLostPk(victimPKPoint uint8, victimGuilty, atWar bool) int {
	if victimGuilty || atWar {
		return 0
	}
	return clampLostPk(3 * int(victimPKPoint) / -25)
}

// pvpLostPkZone é a mudança de PKPoint do matador por uma morte dentro de uma cidade ou de
// uma área de guerra de guilda (ramo de arena do MobKilled.cpp): a taxa /-20 em vez da
// /-25 do campo, limitada a [-9,0]. A checagem de Guilty é de quem chama.
func pvpLostPkZone(victimPKPoint uint8) int {
	return clampLostPk(3 * int(victimPKPoint) / -20)
}

func clampLostPk(lost int) int {
	if lost < -9 {
		lost = -9
	}
	if lost > 0 {
		lost = 0
	}
	return lost
}

// clampPKPoint impõe os limites de escrita do SetPKPoint legado (GetFunc.cpp:
// [1,150]) — 0 é reservado para significar "nunca persistido" em outro lugar deste código, então
// a aritmética nunca pode produzi-lo.
func clampPKPoint(v int) uint8 {
	if v < int(pkPointMin) {
		return pkPointMin
	}
	if v > int(pkPointMax) {
		return pkPointMax
	}
	return uint8(v)
}

// guildsAtWar reporta guerra de guilda mútua (o único bypass AtWar implementado;
// a checagem de guerra de guilda do MobKilled.cpp:3113-3125), espelhando a checagem dupla
// legada contra d.guildWars (guild_state.go).
func (d *Dispatcher) guildsAtWar(a, b uint16) bool {
	if a == 0 || b == 0 {
		return false
	}
	return d.guildWars[a] == b && d.guildWars[b] == a
}
