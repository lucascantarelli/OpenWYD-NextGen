package handler

import (
	"context"
	"fmt"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// Geometria da arena legada (Server.cpp:720-745): os dois pontos de spawn, a
// caixa delimitadora que o ProcessRanking consulta para vivacidade e o dano da parede que fecha,
// e o ponto de saída pós-duelo (alvo da varredura do ClearAreaTeleport,
// Server.cpp:8988). O DoTeleport não recebe argumento de mapa e nossos tipos
// Session/Entity também não — este é um espaço de mapa único global, então não há
// ambiguidade de mapa a resolver para estas coordenadas.
const (
	duelSpawn1X, duelSpawn1Y int16 = 147, 4044 // requisitante (legado Ranking1X/Y)
	duelSpawn2X, duelSpawn2Y int16 = 189, 4044 // aceitante (legado Ranking2X/Y)
	duelBoxX1, duelBoxY1     int16 = 142, 4007 // legado kRanking1X/Y
	duelBoxX2, duelBoxY2     int16 = 195, 4082 // legado kRanking2X/Y
	duelExitX, duelExitY     int16 = 2572, 1752
)

// duelWatchBox é a área de espectadores ao lado do ponto de saída (legado cRanking1X/Y..
// cRanking2X/Y, Server.cpp:737-740): o ProcessRanking anuncia o resultado ali
// assim como dentro da arena.
var duelWatchBox = duelBox{2564, 1688, 2579, 1714}

// duelInviteTimeout é quanto tempo um convite de duelo pendente vive antes de se
// limpar sozinho. O _MSG_ReqRanking.cpp legado não tem opcode explícito de recusa, então um
// timeout é como "recusado" é modelado (mesma classe de placeholder que
// pkGuiltyDuration). Valor exato UNVERIFIED. Uma var (não const) para testes poderem
// encolhê-lo em vez de dormir a duração real de produção.
var duelInviteTimeout = 30 * time.Second

// duelTicks é a contagem regressiva da arena (legado DoRanking duelTime / RankingTime =
// 121). O ProcessRanking a decrementa uma vez por chamada, e ele é chamado em
// SecCounter%4 do timer de 500ms (ProcessSecMinTimer.cpp:1573) — a cada 2s, então um
// duelo dura cerca de 242s e o relógio do cliente recebe StartTime 2*121-2 = 240. Uma var
// (não const, como duelInviteTimeout) para testes poderem encolhê-la em vez de
// esperar a contagem real.
var duelTicks = 121

// duelStepTicks é quantos ticks de mundo de 1s fazem uma chamada de ProcessRanking.
const duelStepTicks = 2

// duelCheckEvery é o passo da varredura de vivacidade: o ProcessRanking só conta os
// combatentes quando RankingTime%5 == 0, então uma morte ou saída encerra o duelo no
// próximo múltiplo de 5 passos (até 10s depois), não na hora.
const duelCheckEvery = 5

// duelArena é o duelo ativo do servidor inteiro (legado RankingProgress/Ranking1/
// Ranking2/RankingTime, Server.cpp — globais de processo, não por par: só
// um duelo ocupa a arena por vez, casando com existir exatamente uma
// arena). Escopo de MVP: só 1v1; batalhas de guilda (DuelParm 1/2/3) estão adiadas.
type duelArena struct {
	active    bool
	player1   int // conn do requisitante (arg "enemy" do DoRanking, Server.cpp:9006-9007)
	player2   int // conn do aceitante (arg "conn" do DoRanking)
	ticksLeft int
	// subTicks conta ticks de mundo de 1s até o próximo passo de ProcessRanking.
	subTicks int
	// name1/name2 são fotografados em startDuel, não relidos da entidade no
	// momento da conclusão: uma desconexão (removeSession) remove a entidade do
	// registro antes de a varredura da arena notar, o que de outro modo descartaria
	// silenciosamente o RecordDuelResult exatamente no caso "oponente saiu no meio do duelo".
	name1, name2 string
}

// dueling informa se a e b são os dois lados vivos do duelo
// ativo. Usado por combat.go para furar o gate de PKMode e pular o carimbo
// caótico de nick PK para golpes trocados entre eles.
func (d *Dispatcher) dueling(a, b int) bool {
	if !d.duel.active {
		return false
	}
	return (d.duel.player1 == a && d.duel.player2 == b) || (d.duel.player1 == b && d.duel.player2 == a)
}

// reqRanking trata _MSG_ReqRanking (0x039F): pedido/aceite de duelo
// (Exec_MSG_ReqRanking, Source/Code/TMSrv/_MSG_ReqRanking.cpp, arquivo inteiro). Parm1
// é o conn alvo (pedido) ou o conn do requisitante original (aceite); Parm2
// (DuelParm) é 0 para um pedido 1v1, 4 para um aceite. DuelParm 1/2/3 (5v5/10v10/
// 100v100 batalhas de guilda via SummonGuild) estão adiados — recusados aqui.
func (d *Dispatcher) reqRanking(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	if s.Mode != world.UserPlay {
		return
	}
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 {
		return
	}
	parm1, duelParm, ok := protocol.StandardParm2(payload)
	if !ok || duelParm < 0 || duelParm > 4 || parm1 <= 0 || int(parm1) >= world.MaxUser {
		return
	}
	target := int(parm1)
	// O bloco de sussurro é checado antes da divisão pedido/aceite, então ele
	// cobre os dois (_MSG_ReqRanking.cpp:31-35).
	if other := w.Session(target); other != nil && other.Whisper {
		d.sendClientMessage(w, s, denyWhisper)
		return
	}

	if duelParm != 4 {
		d.reqRankingRequest(w, s, target, duelParm)
		return
	}
	d.reqRankingAccept(w, s, target)
}

// reqRankingRequest porta o ramo DuelParm != 4: registrar o convite
// pendente e repassá-lo ao alvo.
func (d *Dispatcher) reqRankingRequest(w *world.World, s *world.Session, target int, duelParm int32) {
	if duelParm != 0 {
		return // batalhas de guilda 5v5/10v10/100v100: adiadas, ainda não suportadas
	}
	other := w.Session(target)
	te := w.Entity(target)
	if other == nil || other.Mode != world.UserPlay || te == nil || te.HP <= 0 {
		return
	}
	// Sem checagem de posição: o pedido legado funciona em qualquer lugar, cidades incluídas
	// (issue #424).
	s.DuelTarget = target
	s.DuelTargetExpiry = time.Now().Add(duelInviteTimeout).Unix()
	// Repassar o convite: Parm1 = o conn do próprio requisitante (para o aceite do alvo
	// poder ecoá-lo de volta como tDuel, casando com m->Parm1 = conn no
	// original), Parm2 = o tipo de pedido.
	w.SendTo(other, protocol.Header{Type: protocol.MsgReqRanking, ID: uint16(target)}, protocol.EncodeStandardParm2(int32(s.Conn), duelParm))
}

// reqRankingAccept porta o ramo DuelParm == 4: validar reciprocidade
// (pUser[tDuel].RankingTarget == conn) e iniciar o duelo.
func (d *Dispatcher) reqRankingAccept(w *world.World, s *world.Session, requester int) {
	reqSess := w.Session(requester)
	reqEnt := w.Entity(requester)
	if reqSess == nil || reqSess.Mode != world.UserPlay || reqEnt == nil || reqEnt.HP <= 0 {
		return
	}
	if reqSess.DuelTarget != s.Conn {
		d.log.Debug("ReqRanking accept without a matching invite", "conn", s.Conn, "requester", requester)
		return
	}
	if d.duel.active {
		d.sendClientMessage(w, s, battleInProgress)
		d.sendClientMessage(w, reqSess, battleInProgress)
		return
	}
	reqSess.DuelTarget = 0
	d.startDuel(w, requester, s.Conn)
}

// startDuel porta o ramo 1v1 do DoRanking (Server.cpp:8997-9019): teleportar os dois
// combatentes para o spawn da arena, anunciar a batalha, iniciar a contagem regressiva,
// e ocupar a arena.
func (d *Dispatcher) startDuel(w *world.World, requester, accepter int) {
	d.duel = duelArena{active: true, player1: requester, player2: accepter, ticksLeft: duelTicks}
	if re := w.Entity(requester); re != nil {
		d.duel.name1 = re.Name
	}
	if ae := w.Entity(accepter); ae != nil {
		d.duel.name2 = ae.Name
	}

	if rs := w.Session(requester); rs != nil {
		d.doTeleport(w, rs, duelSpawn1X, duelSpawn1Y)
	}
	if as := w.Session(accepter); as != nil {
		d.doTeleport(w, as, duelSpawn2X, duelSpawn2Y)
	}

	// o conn do DoRanking é o aceitante, então o aceitante é nomeado primeiro.
	started := fmt.Sprintf(battleStarted, d.duel.name2, d.duel.name1)
	parm := protocol.EncodeStandardParm(int32(duelStepTicks*duelTicks - 2))
	for _, conn := range [2]int{accepter, requester} {
		if cs := w.Session(conn); cs != nil {
			d.sendClientMessage(w, cs, started)
			w.Send(cs, protocol.MsgStartTime, parm)
		}
	}
}

// sweepDuelInvites expira convites de duelo pendentes quando seu prazo de relógio
// de parede passa (espelha a forma de sweepGuilty, pkmode.go:97-105) — a limpeza
// de "recusado/timeout" que os critérios de aceite da issue pedem.
func (d *Dispatcher) sweepDuelInvites(w *world.World) {
	now := time.Now().Unix()
	w.ForEachPlayer(func(s *world.Session, _ *world.Entity) {
		if s.DuelTarget != 0 && now >= s.DuelTargetExpiry {
			s.DuelTarget = 0
		}
	})
}

// sweepDuelArena resolve o duelo ativo: contagem regressiva, checagens de vivacidade/limites,
// e (para o visual da parede que fecha) o autodano da caixa que encolhe. Porta o
// ProcessRanking (Server.cpp:8786-8989) como um poll dirigido por tick em vez de um
// caso especial por golpe, casando com o desenho legado: um passo a cada
// duelStepTicks ticks de mundo, uma varredura de vivacidade a cada duelCheckEvery passos.
func (d *Dispatcher) sweepDuelArena(w *world.World) {
	if !d.duel.active {
		return
	}
	d.duel.subTicks++
	if d.duel.subTicks < duelStepTicks {
		return
	}
	d.duel.subTicks = 0

	d.duel.ticksLeft--
	if d.duel.ticksLeft%duelCheckEvery == 0 &&
		(!d.duelantAlive(w, d.duel.player1) || !d.duelantAlive(w, d.duel.player2)) {
		d.duel.ticksLeft = 0
	}
	d.applyDuelWall(w, d.duel.ticksLeft)
	if d.duel.ticksLeft > 0 {
		return // ainda lutando
	}

	alive1 := d.duelantAlive(w, d.duel.player1)
	alive2 := d.duelantAlive(w, d.duel.player2)
	switch {
	case alive1 && !alive2:
		d.concludeDuelWin(w, d.duel.player1, d.duel.player2)
	case alive2 && !alive1:
		d.concludeDuelWin(w, d.duel.player2, d.duel.player1)
	default:
		d.concludeDuelDraw(w) // timeout, ou os dois fora ao mesmo tempo
	}
}

// duelantAlive informa se conn ainda é um lado vivo do duelo e
// dentro dos limites: conectado, HP > 0, e dentro da caixa da arena. Uma desconexão
// (removeSession já zera o registro) ou sair dos limites encerra
// o lado dele, espelhando a varredura de vivacidade por tick do ProcessRanking.
func (d *Dispatcher) duelantAlive(w *world.World, conn int) bool {
	s := w.Session(conn)
	e := w.Entity(conn)
	if s == nil || s.Mode != world.UserPlay || e == nil || e.HP <= 0 {
		return false
	}
	return e.X >= duelBoxX1 && e.X <= duelBoxX2 && e.Y >= duelBoxY1 && e.Y <= duelBoxY2
}

// duelantName busca o nome de conn no snapshot da arena (ainda ativo) —
// não na entidade viva, que já pode ter sumido (um perdedor desconectado).
func (d *Dispatcher) duelantName(conn int) string {
	switch conn {
	case d.duel.player1:
		return d.duel.name1
	case d.duel.player2:
		return d.duel.name2
	default:
		return ""
	}
}

// concludeDuelWin encerra o duelo ativo com o vencedor batendo o perdedor: anuncia
// o resultado, limpa a arena, registra o resultado, e libera a arena.
func (d *Dispatcher) concludeDuelWin(w *world.World, winner, loser int) {
	winnerName, loserName := d.duelantName(winner), d.duelantName(loser)
	d.duel = duelArena{}

	d.duelNoticeArea(w, fmt.Sprintf(duelWinBy, winnerName, loserName))
	d.duelClearArea(w)
	if winnerName != "" && loserName != "" {
		d.recordDuelResult(w, winnerName, loserName)
	}
}

// concludeDuelDraw encerra o duelo ativo sem vencedor (timeout, ou os dois
// lados eliminados ao mesmo tempo): anuncia o empate e limpa a arena, exatamente
// como uma vitória (o ClearAreaTeleport legado roda de qualquer modo). Empates não são
// persistidos — o RecordDuelResult só rastreia vitórias/derrotas.
func (d *Dispatcher) concludeDuelDraw(w *world.World) {
	name1, name2 := d.duel.name1, d.duel.name2
	d.duel = duelArena{}

	d.duelNoticeArea(w, fmt.Sprintf(duelDraw, name1, name2))
	d.duelClearArea(w)
}

// duelNoticeArea porta as duas chamadas SendNoticeArea do ProcessRanking
// (SendFunc.cpp:247): o resultado vai a todos na caixa de espectadores e na
// arena — os duelistas, enquanto ainda dentro, e qualquer observador.
func (d *Dispatcher) duelNoticeArea(w *world.World, text string) {
	arena := duelBox{duelBoxX1, duelBoxY1, duelBoxX2, duelBoxY2}
	w.ForEachPlaying(-1, func(s *world.Session, e *world.Entity) {
		if duelWatchBox.contains(e.X, e.Y) || arena.contains(e.X, e.Y) {
			d.sendClientMessage(w, s, text)
		}
	})
}

// duelClearArea porta o ClearAreaTeleport(kRanking..., 2572, 1752) (Server.cpp:
// 6388): todos ainda dentro da arena vão ao ponto de saída, e quem
// morreu ali volta com HP 1. Um duelista que já saiu da caixa não é
// movido, como no legado.
func (d *Dispatcher) duelClearArea(w *world.World) {
	arena := duelBox{duelBoxX1, duelBoxY1, duelBoxX2, duelBoxY2}
	w.ForEachPlaying(-1, func(s *world.Session, e *world.Entity) {
		if !arena.contains(e.X, e.Y) {
			return
		}
		if e.HP <= 0 {
			e.HP = 1
			s.ReqHp = 1
			d.sendScore(w, s, e)
			d.sendSetHpMp(w, s, e)
		}
		d.doTeleport(w, s, duelExitX, duelExitY)
	})
}

// duelWallEffect é o id de EnvEffect legado usado para todo visual da parede que fecha
// no ProcessRanking (Server.cpp: `SendEnvEffect(..., 32, 0)`, repetido para
// cada estágio/quadrante). EffectParm é sempre 0.
const duelWallEffect = 32

// duelStage agrupa ticksLeft nos três estágios da parede que fecha do ProcessRanking
// (Server.cpp:8834-8869): 0 = sem parede ainda, 1/2/3 = progressivamente mais estreita.
// Compartilhado por duelWallDamageStages e duelWallEffectStages para os limiares
// existirem uma vez só.
func duelStage(ticksLeft int) int {
	switch {
	case ticksLeft >= 180:
		return 0 // legado: sem parede antes de RankingTime chegar a 179 (duelTicks=121 nunca chega lá)
	case ticksLeft >= 120:
		return 1
	case ticksLeft >= 60:
		return 2
	default:
		return 3
	}
}

// duelWallDamageStages e duelWallEffectStages são ports 1:1 diretos dos
// três estágios da parede que fecha do ProcessRanking (Server.cpp:8834-8869), indexados
// por duelStage. As caixas de dano são as próprias faixas da "parede" (quem
// está ali leva duelWallDamage); as caixas de efeito são a mesma
// geometria dividida nos quadrantes visuais que o cliente renderiza. Todas as coordenadas de caixa
// são relativas aos limites da arena (duelBoxX1/Y1/X2/Y2).
var duelWallDamageStages = [4][]duelBox{
	0: nil,
	1: {{duelBoxX1, duelBoxY1, duelBoxX2, 4019}, {duelBoxX1, 4070, duelBoxX2, duelBoxY2}},
	2: {{duelBoxX1, duelBoxY1, duelBoxX2, 4034}, {duelBoxX1, 4055, duelBoxX2, duelBoxY2}},
	3: {{duelBoxX1, duelBoxY1, duelBoxX2, 4042}, {duelBoxX1, 4046, duelBoxX2, duelBoxY2}},
}

var duelWallEffectStages = [4][]duelBox{
	0: nil,
	1: {
		{duelBoxX1, duelBoxY1, 168, 4018}, {duelBoxX1, 4071, 168, duelBoxY2},
		{169, duelBoxY1, duelBoxX2, 4018}, {169, 4071, duelBoxX2, duelBoxY2},
	},
	2: {
		{duelBoxX1, duelBoxY1, 168, 4018}, {duelBoxX1, 4071, 168, duelBoxY2},
		{duelBoxX1, 4019, 168, 4030}, {duelBoxX1, 4059, 168, 4070},
		{168, duelBoxY1, duelBoxX2, 4018}, {168, 4071, duelBoxX2, duelBoxY2},
		{168, 4019, duelBoxX2, 4030}, {168, 4059, duelBoxX2, 4070},
	},
	3: {
		{duelBoxX1, duelBoxY1, 168, 4018}, {duelBoxX1, 4071, 168, duelBoxY2},
		{duelBoxX1, 4019, 168, 4030}, {duelBoxX1, 4059, 168, 4070},
		{duelBoxX1, 4031, 168, 4042}, {duelBoxX1, 4047, 168, 4058},
		{168, duelBoxY1, duelBoxX2, 4018}, {168, 4071, duelBoxX2, duelBoxY2},
		{168, 4019, duelBoxX2, 4030}, {168, 4059, duelBoxX2, 4070},
		{168, 4031, duelBoxX2, 4042}, {168, 4047, duelBoxX2, 4058},
	},
}

// applyDuelWall porta SendDamage/SendEnvEffect para o estágio atual da parede que
// fecha: qualquer um (dos dois duelistas — este servidor não modela espectadores
// entrando na arena) pego numa faixa da parede tem o HP baixado em 2000
// (ou para 1), e o contorno da caixa que encolhe é transmitido aos observadores em visão.
// Como o legado, roda uma vez por passo de ProcessRanking (a cada duelStepTicks).
func (d *Dispatcher) applyDuelWall(w *world.World, ticksLeft int) {
	stage := duelStage(ticksLeft)
	for _, box := range duelWallDamageStages[stage] {
		for _, conn := range [2]int{d.duel.player1, d.duel.player2} {
			s := w.Session(conn)
			e := w.Entity(conn)
			if s == nil || e == nil || e.HP <= 0 {
				continue
			}
			if e.X < box.x1 || e.X > box.x2 || e.Y < box.y1 || e.Y > box.y2 {
				continue
			}
			duelWallDamage(s, e)
			d.sendSetHpMp(w, s, e)
		}
	}
	for _, box := range duelWallEffectStages[stage] {
		body := protocol.EncodeEnvEffect(box.x1, box.y1, box.x2, box.y2, duelWallEffect, 0)
		cx, cy := box.x1+(box.x2-box.x1)/2, box.y1+(box.y2-box.y1)/2
		w.ForEachInViewAt(cx, cy, 0, func(vs *world.Session, _ *world.Entity) {
			w.SendTo(vs, protocol.Header{Type: protocol.MsgEnvEffect, ID: protocol.IDScene}, body)
		})
	}
}

// duelWallDamage baixa o HP de e em 2000, ou para 1 se já estiver abaixo (o
// port do efeito punitivo do SendDamage, Server.cpp:5970-6009), passando por
// damageReqHp para o tick de regen não curar de volta na hora (issue #99).
func duelWallDamage(s *world.Session, e *world.Entity) {
	dmg := int32(2000)
	if e.HP < dmg {
		dmg = e.HP - 1
	}
	if dmg <= 0 {
		return
	}
	e.HP -= dmg
	damageReqHp(s, e, dmg)
}

// recordDuelResult persiste a vitória/derrota fora do loop (World.GoDetached — não
// preso a nenhuma sessão, já que um lado pode já ter sumido quando isto
// roda), espelhando o padrão assíncrono de chamada ao dbServer usado em outros lugares (ex.:
// createCharacter, character.go:51-75).
func (d *Dispatcher) recordDuelResult(w *world.World, winnerName, loserName string) {
	p := w.Persistence()
	w.GoDetached(func() func(*world.World) {
		if err := p.RecordDuelResult(context.Background(), winnerName, loserName); err != nil {
			return func(*world.World) {
				d.log.Warn("duel result: persistence failed", "winner", winnerName, "loser", loserName, "err", err)
			}
		}
		return nil
	})
}
