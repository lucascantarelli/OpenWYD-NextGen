package handler

import (
	"fmt"
	"time"

	"openwyd-nextgen/tmserver/internal/combat"
	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/rng"
	"openwyd-nextgen/tmserver/internal/route"
	"openwyd-nextgen/tmserver/internal/world"
)

// Ajuste da IA de criaturas (CMob.cpp; o loop de orquestração que dirigia isto por tick não está
// na fonte disponível, então cadências/alcances são fiéis em espírito, UNVERIFIED).
const (
	leashRadius      = 16   // HALFGRIDX: alvo fora da caixa natal da criatura perde o aggro
	meleeRange       = 1    // piso de alcance quando o template não tem equipamento EF_RANGE
	kefraBossRange   = 25   // CMob.cpp BattleProcessor: KEFRA_BOSS ignora EF_RANGE
	mobAttackCadence = 1000 // ms entre os ataques de uma criatura (≈ a guarda de 800ms do jogador)
	// wakeRadius condiciona a varredura de aggro por tick: uma criatura ociosa sem jogador dentro
	// desta distância Chebyshev pula FindEnemyFromView por completo. É ViewRange
	// em vez da janela de aggro do jogador, para encontros visíveis guarda-vs-criatura ainda
	// começarem com o observador perto da borda da cena.
	wakeRadius = world.ViewRange
	// roamRadius condiciona o perambular do mesmo modo, mas precisa passar de ViewRange (16) para um
	// jogador nunca ver uma patrulha congelada: as criaturas começam a andar logo além do que
	// o cliente consegue ver. Mesma classe de divergência que wakeRadius.
	roamRadius = 20
)

// Tick é o gancho de IA de criaturas por tick do mundo (registrado via World.SetTickHandler).
// Roda dentro da goroutine do loop, então muta entidades direto. Cada
// criatura viva pega um jogador ou criatura hostil próximo como alvo, depois persegue e
// o ataca corpo a corpo. NPCs de não-combate (lojas/quest givers/serviços da cidade) e criaturas mortas são
// pulados.
//
// UNVERIFIED / adiado (captura): pathfinding real (damos um passo Chebyshev
// de tile), perambular/segmentos/invocações, e o fluxo de morte/ressurreição do jogador (um
// jogador levado a 0 HP simplesmente deixa de ser um alvo válido).
func (d *Dispatcher) Tick(w *world.World) {
	// Um incremento por tick, logo no início: tanto regenPlayers (fase de regen natural) quanto
	// sweepAffects (escalonamento por jogador) leem este contador, e regenPlayers roda
	// primeiro, então ele não pode viver dentro de nenhuma das varreduras.
	d.tickCount++
	d.tickKefra(w)
	d.tickNightmare(w)
	d.tickRuneQuest(w)
	d.tickRuneQuestRooms(w)
	d.ensureGuildStateLoaded(w)

	// Gate de dormência: tirar um snapshot das (poucas) posições de jogadores em jogo uma vez, para as
	// ~10k criaturas ociosas longe de qualquer jogador pularem a varredura de aggro de 81 células. As
	// fatias temporárias vivem no Dispatcher (só no loop) para evitar alocação por tick.
	d.playersX, d.playersY = d.playersX[:0], d.playersY[:0]
	w.ForEachPlayer(func(_ *world.Session, e *world.Entity) {
		d.playersX = append(d.playersX, e.X)
		d.playersY = append(d.playersY, e.Y)
	})

	w.ForEachMob(func(id int, e *world.Entity) {
		if !runsMobAI(e) || e.HP <= 0 {
			return // NPCs de cidade/serviço não lutam; criaturas mortas não fazem nada
		}
		if e.Summoner != 0 {
			d.summonTick(w, id, e) // mascotes invocados seguem a própria IA (summon.go)
			return
		}
		if e.Target == 0 && hasEnemyList(e) {
			e.Mode = world.MobCombat
			d.mobBattle(w, id, e)
			return
		}
		if e.Target == 0 {
			if !d.playerNear(e.X, e.Y, roamRadius) {
				return // dormente: ninguém perto o bastante para ver ou receber aggro
			}
			// Aggro só enquanto a criatura está dentro da própria caixa de segmento — uma criatura
			// voltando ou perambulando longe da âncora não entra em combate (StandingBy
			// CMob.cpp:162 compara a posição do MOB com SegmentX±HALFGRIDX).
			// Seguidores nunca se auto-aggroam (Leader != 0): StandingBy só chama
			// GetEnemyFromView para líderes (CMob.cpp:158); eles entram em combates via
			// o arrasto de grupo (setGroupBattle).
			if e.Leader == 0 && d.playerNear(e.X, e.Y, wakeRadius) &&
				chebyshev(e.X, e.Y, e.SegmentX, e.SegmentY) <= leashRadius {
				if p := w.FindEnemyFromView(e.X, e.Y, e.Clan); p != 0 &&
					(!world.IsPlayer(p) || !inSafeCity(w, p)) {
					setGroupBattle(w, id, e, w.Entity(p))
				}
			}
			if e.Target == 0 {
				d.mobRoam(w, id, e)
				return
			}
		}
		if e.Target != 0 {
			d.mobBattle(w, id, e)
		}
	})
	d.tickQuestAreaReset(w)
	d.guardQuest256Areas(w)
	d.regenPlayers(w)
	d.sweepAffects(w)
	d.sweepGuilty(w)
	d.sweepDuelInvites(w)
	d.sweepDuelArena(w)
	d.tickKingdomRvR(w)
	d.tickTowerWar(w)
	d.tickCastle(w)
	d.tickWater(w)
	d.respawnMobs(w)
	d.generateMobs(w)
	// Eventos de mundo (issue #116). tickWeather fica depois de generateMobs para isto
	// ler na ordem legada de ProcessMinTimer: geradores primeiro
	// (ProcessSecMinTimer.cpp:2720-2789), depois o sorteio de clima (:2791).
	d.tickWeather(w)
	d.pollNPCConfig(w) // recarga a quente de edições de NPC do moderador (npc-editing-plan.md)
	d.pollWorldEventConfig(w)
	d.applyNoviceChannel(w)
}

type questArea struct {
	x1, y1 int16
	x2, y2 int16
}

func (a questArea) contains(x, y int16) bool {
	return x > a.x1 && y > a.y1 && x < a.x2 && y < a.y2
}

// guardQuest256Areas porta a guarda QuestFlag de ProcessSecMinTimer para as cinco
// arenas da Quest 256. Entrar sem a flag volátil correspondente faz o
// jogador ser retornado, que é o comportamento que o Mestre Grifo precisa cumprir antes de teleportar.
func (d *Dispatcher) guardQuest256Areas(w *world.World) {
	w.ForEachPlayer(func(s *world.Session, e *world.Entity) {
		if e.Level >= 1000 {
			return
		}
		for _, step := range quest256Steps {
			if step.area.contains(e.X, e.Y) && e.QuestFlag != step.flag {
				d.recall(w, s, e)
				return
			}
		}
	})
}

// battleDragBox é a caixa de engajamento de SetBattle: um membro do grupo entra na luta só
// quando o alvo está a ±23 dele (Server.cpp:8029).
const battleDragBox = 23

// setGroupBattle coloca uma criatura em combate com o alvo e arrasta o grupo dela para a
// mesma luta — SetBattle (Server.cpp:8013-8047) mais a propagação por PartyList
// do tick de IA (ProcessSecMinTimer.cpp:1882-1929): resolver o líder da criatura
// (ou ela mesma), então cada membro vivo dentro da caixa de arrasto registra o alvo
// no próprio EnemyList. O próprio líder também é arrastado (o bloco original só
// percorre PartyList — líderes entram pelos próprios caminhos; incluir isto aqui é uma
// divergência pequena e coerente).
func setGroupBattle(w *world.World, id int, e, target *world.Entity) {
	if target == nil {
		return
	}
	setBattle(w, id, e, target)
	lid := e.Leader
	if lid == 0 {
		lid = id
	}
	le := w.Entity(lid)
	if le == nil {
		return
	}
	drag := func(mid int, me *world.Entity) {
		if mid == id || me == nil || me.HP <= 0 || !runsMobAI(me) {
			return
		}
		if abs16(me.X-target.X) <= battleDragBox && abs16(me.Y-target.Y) <= battleDragBox {
			setBattle(w, mid, me, target)
		}
	}
	if lid != id {
		drag(lid, le)
	}
	for _, m := range le.PartyList {
		if m < world.MaxUser {
			continue // slot vazio (0) ou um id de jogador — só grupos de criaturas
		}
		drag(m, w.Entity(m))
	}
}

func runsMobAI(e *world.Entity) bool {
	if e == nil {
		return false
	}
	// Guardas da cidade são protegidos do dano de jogadores como outros NPCs da cidade, mas eles
	// ainda policiam criaturas hostis na cidade (paridade guarda-vs-criatura).
	return !e.NonCombatNPC || e.Clan == 7 || e.Clan == 8
}

func setBattle(w *world.World, id int, e, target *world.Entity) {
	if e == nil || target == nil || id <= 0 || target.ID <= 0 || id >= world.MaxMob || target.ID >= world.MaxMob || id == target.ID {
		return
	}
	targetInWorld := w.Entity(target.ID) != nil
	if e.Mode == world.MobEmpty || (targetInWorld && target.Mode == world.MobEmpty) {
		return
	}
	if world.IsPlayer(target.ID) {
		if sameKingdom(e.Clan, target.Clan) {
			return
		}
		if targetInWorld {
			if m, ok := w.SessionMode(target.ID); !ok || m != world.UserPlay {
				return
			}
		}
	}
	if abs16(e.X-target.X) > battleDragBox || abs16(e.Y-target.Y) > battleDragBox {
		return
	}
	e.Mode = world.MobCombat
	addEnemyList(e, target)
	sendFightAction(w, id, e)
	if targetInWorld {
		selectTargetFromEnemyList(w, e)
	} else if e.Target == 0 {
		// Vários testes de baixo nível passam uma entidade atacante sintética em vez de um
		// jogador do mundo. Alvos de produção estão sempre em World e usam a
		// seleção por EnemyList acima.
		e.Target = target.ID
	}
}

func sendFightAction(w *world.World, id int, e *world.Entity) {
	if e == nil {
		return
	}
	say := w.Rand().Intn(4) // SetBattle de Server.cpp consome rand()%4 ao engajar.
	gen := w.GeneratorAt(int(e.GenIndex))
	if gen == nil || e.Leader != 0 || gen.FightAction[say] == "" {
		return
	}
	sendMobChat(w, id, gen.FightAction[say])
}

func sendMobChat(w *world.World, id int, text string) {
	if text == "" {
		return
	}
	payload := protocol.EncodeMessageChatBody(text)
	w.ForEachInView(id, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgMessageChat, ID: uint16(id)}, payload)
	})
}

func addEnemyList(e, target *world.Entity) {
	if e == nil || target == nil || target.ID <= 0 || target.ID >= world.MaxMob {
		return
	}
	if world.IsPlayer(target.ID) {
		if target.Rsv&world.RsvHide != 0 || target.Merchant&1 != 0 {
			return
		}
	}
	for _, id := range e.EnemyList {
		if id == target.ID {
			return
		}
	}
	for i, id := range e.EnemyList {
		if id == 0 {
			e.EnemyList[i] = target.ID
			return
		}
	}
}

func removeEnemyList(e *world.Entity, targetID int) {
	if e == nil || targetID <= 0 {
		return
	}
	for i, id := range e.EnemyList {
		if id == targetID {
			e.EnemyList[i] = 0
			return
		}
	}
}

func clearEnemyList(e *world.Entity) {
	if e != nil {
		e.EnemyList = [protocol.MaxTarget]int{}
	}
}

func hasEnemyList(e *world.Entity) bool {
	if e == nil {
		return false
	}
	for _, id := range e.EnemyList {
		if id != 0 {
			return true
		}
	}
	return false
}

func selectTargetFromEnemyList(w *world.World, e *world.Entity) {
	if e == nil {
		return
	}
	e.Target = 0
	enemyDist := [protocol.MaxTarget]int{}
	for i := range enemyDist {
		enemyDist[i] = world.MaxUser
	}
	dis := enemySelectRange(e)
	for i, enemyID := range e.EnemyList {
		if enemyID <= 0 || enemyID >= world.MaxMob {
			continue
		}
		enemy := w.Entity(enemyID)
		if enemy == nil || enemy.Mode == world.MobEmpty || enemy.HP <= 0 {
			e.EnemyList[i] = 0
			continue
		}
		if world.IsPlayer(enemyID) {
			if sameKingdom(e.Clan, enemy.Clan) {
				e.EnemyList[i] = 0
				continue
			}
			if enemy.Rsv&world.RsvHide != 0 {
				e.EnemyList[i] = 0
				continue
			}
			if enemy.Level > level.MaxLevel && enemy.Merchant&1 == 0 {
				e.EnemyList[i] = 0
				continue
			}
		}
		if enemy.X < e.X-int16(dis) || enemy.X > e.X+int16(dis) ||
			enemy.Y < e.Y-int16(dis) || enemy.Y > e.Y+int16(dis) {
			e.EnemyList[i] = 0
			continue
		}
		enemyDist[i] = mobDistance(e.X, e.Y, enemy.X, enemy.Y)
		if enemyID > world.MaxUser {
			enemyDist[i] += 2 // CMob.cpp usa > MAX_USER, então id==MAX_USER não é penalizado.
		}
	}
	bestDist := world.MaxUser
	best := 0
	for i, enemyID := range e.EnemyList {
		if enemyID != 0 && bestDist >= enemyDist[i] {
			best = enemyID
			bestDist = enemyDist[i]
		}
	}
	if bestDist != world.MaxUser {
		e.Target = best
	}
}

func enemySelectRange(e *world.Entity) int {
	dis := 6
	if e.Clan == summonClan || e.Clan == 7 || e.Clan == 8 {
		dis = 12
	}
	if int(e.X)/128 < 12 && int(e.Y)/128 > 25 {
		dis = 8
	}
	if isKefraBoss(e) {
		dis = leashRadius
	}
	return dis
}

// generateMobs é o timer de regeneração por gerador (o bloco GenerateMobs
// de ProcessSecMinTimer.cpp:2720-2760): uma vez por minuto (60 ticks na cadência
// de 1s), todo bloco cujo MinuteGenerate é positivo dispara na própria fase —
// `minute % MinuteGenerate == idx % MinuteGenerate` — completando a população
// em grupos inteiros. Não portado: o pulo fixo dos blocos
// {0,1,2,5,6,7} (um truque de evento de Coliseu) e o reaproveitamento de
// relocação de evento com MinuteGenerate>=500.
func (d *Dispatcher) generateMobs(w *world.World) {
	if d.tickCount%60 != 0 {
		return
	}
	minute := d.tickCount / 60
	for idx := 0; idx < w.GeneratorCount(); idx++ {
		g := w.GeneratorAt(idx)
		if g == nil || g.MinuteGenerate <= 0 || world.IsWaterGenerator(idx) || world.IsKefraGenerator(idx) || world.NightmareGenerator(idx) >= 0 {
			continue
		}
		if minute%g.MinuteGenerate != idx%g.MinuteGenerate {
			continue
		}
		d.revealSpawned(w, w.GenerateMob(idx))
	}
}

// playerNear informa se algum jogador do snapshot está dentro do raio (Chebyshev)
// de (x,y) — a pré-checagem barata antes da varredura de aggro / trabalho de perambular de verdade.
func (d *Dispatcher) playerNear(x, y int16, radius int) bool {
	for i := range d.playersX {
		if abs16(x-d.playersX[i]) <= radius && abs16(y-d.playersY[i]) <= radius {
			return true
		}
	}
	return false
}

// respawnMobs renasce criaturas cujo atraso pós-morte expirou e revela
// cada uma aos jogadores já em visão.
func (d *Dispatcher) respawnMobs(w *world.World) {
	d.revealSpawned(w, w.SpawnDueRespawns(w.Now()))
}

// revealSpawned anuncia criaturas recém-nascidas aos jogadores já em visão,
// para um respawn/regeneração ser visto na hora em vez de só quando um jogador
// voltar a andar (revealMobsInView). MarkSeen evita um CreateMob duplicado
// se aquele reveal de caminhada também disparar neste tick.
func (d *Dispatcher) revealSpawned(w *world.World, ids []int) {
	for _, id := range ids {
		mob := w.Entity(id)
		if mob == nil {
			continue
		}
		body := protocol.EncodeCreateMobBody(d.createMobFrom(mob, 0))
		w.ForEachInView(id, func(vs *world.Session, _ *world.Entity) {
			if w.MarkSeen(vs, id) {
				w.SendTo(vs, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene}, body)
			}
		})
	}
}

// inSafeCity informa se o conn do jogador está dentro de um retângulo de cidade —
// uma zona segura onde criaturas não dão aggro nem atacam (para um jogador renascido poder
// se recuperar). Espelha o comportamento original de não-combate na cidade (BASE_GetVillage).
func inSafeCity(w *world.World, conn int) bool {
	e := w.Entity(conn)
	return e != nil && world.Village(e.X, e.Y) >= 0
}

// naturalRegenTicks é a cadência de regen passiva: o fio legado roda em
// SecCounter%20 de um timer de 500ms (ProcessSecMinTimer.cpp:780) = a cada 10s. Nosso
// tick é de 1s, então a cada 10 ticks.
const naturalRegenTicks = 10

// regenPlayers é a passagem de manutenção de HP/MP (ProcessSecMinTimer.cpp:760-799). Roda
// no tick de mundo de 1s, o que casa exatamente com a cadência de drenagem legada
// (SecCounter%2 de um timer de 500ms).
//
// O modelo é baseado em pedido, e é o MESMO modelo que as poções usam: nada escreve
// a barra viva direto. Os chamadores elevam o alvo ReqHp/ReqMp; applyHp/applyMp
// fecham a diferença em no máximo applyCasting por tick. Um jogador sem dívida de cura
// (ReqHp <= HP) não custa nada — applyHp informa que não houve movimento e não enviamos pacote.
//
// Os mortos (HP==0) não regeneram — precisam de _MSG_Restart antes.
//
// Adiado: o ramo Celestial (ProcessSecMinTimer.cpp:676-680) soma MOB.RegenHP
// /RegenMP em cima do fio para ClassMaster >= CELESTIAL; esses campos ainda não
// estão modelados em world.Entity (cf. combat.go:475).
func (d *Dispatcher) regenPlayers(w *world.World) {
	now := time.Now().Unix()
	w.ForEachPlayer(func(s *world.Session, e *world.Entity) {
		if e.HP <= 0 {
			return
		}
		d.expireWandererBags(w, s, e, now)
		// Expirar o buff Divino quando seu prazo de relógio de parede passa (captura §B): remover
		// o affect e empurrar o score (agora menor) + snapshot de buffs.
		if e.DivineEnd > 0 && now >= e.DivineEnd && e.HasAffect(world.AffectDivine) {
			e.ClearAffect(world.AffectDivine)
			e.DivineEnd = 0
			d.refreshScore(e)
			d.sendScore(w, s, e)
			d.sendAffect(w, s, e)
		}
		d.tickIncubation(w, s, e)
		if d.tickCount%naturalRegenTicks == 0 {
			s.ReqHp += e.Level + 30
			s.ReqMp += e.Level + 30
		}
		// As duas barras sempre drenam; só o ENVIO é ou-um-ou-outro. Mantenha isto como
		// instruções separadas — juntá-las em `if applyHp(...) else if applyMp(...)` causaria
		// curto-circuito e pararia a regeneração de MP sempre que o HP regenerasse.
		movedHP := applyHp(s, e)
		movedMP := applyMp(s, e)
		// SendScore (multicast) quando o HP mudou, senão SendSetHpMp (só para si) para uma
		// mudança só de mana — o else-if exato do original, para um tick de MP nunca custar
		// um broadcast de 152B. O UpdateScore já carrega Mp, então o ramo de HP cobre os dois.
		if movedHP {
			d.sendScore(w, s, e)
		} else if movedMP {
			d.sendSetHpMp(w, s, e)
		}
	})
}

// tickIncubation decrementa em um o cooldown de incubação de um ovo equipado
// (RegenMob, Server.cpp:4884-4894). Só um ovo EQUIPADO conta — essa é a
// mecânica: um refino falho marca o cooldown e o jogador precisa vestir o
// ovo e esperar até o fim antes de tentar de novo. Sem isto o gate
// _NN_Incu_Wait_More do caminho de refino inutilizaria um ovo para sempre.
func (d *Dispatcher) tickIncubation(w *world.World, s *world.Session, e *world.Entity) {
	egg := &e.Equip[mountEquipSlot]
	if !isEgg(*egg) {
		return
	}
	delay := itemInstanceAbility(*egg, efIncuDelay)
	if delay <= 0 {
		return
	}
	egg.Effects[2].Value = uint8(delay - 1)
	d.sendSlot(w, s, world.ItemPlaceEquip, mountEquipSlot, *egg)
	// SendClientMessage(_NN_Incu_Proceed) — ainda sem código de aviso para ele; o
	// push de item é o que o cliente renderiza.
}

// Códigos de decisão de batalha — os valores de retorno do BattleProcessor (CMob.cpp:233-328),
// consumidos pelo tick de IA original em ProcessSecMinTimer.cpp:2089-2477. Só os
// três resultados de alcance são modelados; 0x2 (teleporte), 0x10 (voltar para casa) e 0x20
// (apagar) pertencem a sistemas adiados.
const (
	battleChase   = 0x1    // fora de alcance → mover em direção ao alvo
	battleRetreat = 0x100  // criatura à distância com o alvo no alcance curto → recuar
	battleAttack  = 0x1000 // no alcance → atacar
)

// mobBattle avança uma criatura engajada: validar o alvo, sortear a hesitação
// por Int, depois atacar / recuar / perseguir conforme a decisão do BattleProcessor.
func (d *Dispatcher) mobBattle(w *world.World, id int, e *world.Entity) {
	selectTargetFromEnemyList(w, e)
	target := w.Entity(e.Target)
	if !validTarget(w, e, target) {
		removeEnemyList(e, e.Target)
		e.Target = 0
		e.Mode = world.MobIdle
		return
	}

	// Hesitação por Int (BattleProcessor CMob.cpp:279-284): uma criatura de Int baixo pula
	// este tick com mais frequência, o que também limita naturalmente seus ataques. O
	// original sorteia contra BaseScore.Int; nosso Int de criatura é o
	// CurrentScore do template (a distinção para criaturas é UNVERIFIED — os templates trazem os dois
	// iguais).
	if int(e.Int) < w.Rand().Intn(100) {
		return
	}

	reach := mobReach(e)
	dis := mobDistance(e.X, e.Y, target.X, target.Y)
	switch battleCode(w.Rand(), dis, reach, int(e.Dex)) {
	case battleAttack:
		d.mobAttack(w, id, e, target)
	case battleRetreat:
		if isKefraBoss(e) {
			return
		}
		d.mobRetreat(w, id, e, target)
	default:
		if isKefraBoss(e) {
			return
		}
		d.mobStep(w, id, e, target)
	}
}

func mobReach(e *world.Entity) int {
	if isKefraBoss(e) {
		return kefraBossRange
	}
	reach := int(e.Range)
	if reach < meleeRange {
		return meleeRange // sem EF_RANGE no equipamento do template → adjacência corpo a corpo
	}
	return reach
}

func isKefraBoss(e *world.Entity) bool {
	return e != nil && int(e.GenIndex) == world.KefraBossGenIndex
}

// battleCode porta a decisão de alcance do BattleProcessor (CMob.cpp:308-327): dentro
// do alcance ataca, EXCETO uma criatura à distância (Range>=4) cujo alvo chegou a <=4
// sorteia contra seu Dex para recuar e manter distância. O sorteio é consumido
// incondicionalmente dentro do ramo de alcance (CMob.cpp:310) — a ordem de chamadas de rand é
// relevante para paridade. Fora de alcance persegue. (O ajuste BASE_GetHitPosition está
// comentado na fonte, então no alcance ele sempre ataca no resto.)
func battleCode(r *rng.MSVC, dis, reach, dex int) int {
	if dis <= reach {
		roll := r.Intn(100)
		if reach >= 4 && dis <= 4 && roll > dex {
			return battleRetreat
		}
		return battleAttack
	}
	return battleChase
}

// validTarget informa se o alvo de uma criatura ainda é atacável: um
// jogador em jogo ou uma criatura hostil dentro da caixa de contenção — centrada na âncora
// de waypoint atual (o BattleProcessor contém em SegmentX±HALFGRIDX, CMob.cpp:292;
// para uma criatura sem rota a âncora é o ponto de spawn).
//
// As invocações mantêm regras próprias: mascotes atacam criaturas mas nunca jogadores ou
// outros mascotes, enquanto criaturas comuns ainda podem revidar contra mascotes. O lado
// das invocações pula a contenção de waypoint — sua contenção é a distância até o DONO,
// imposta por summonTick antes de mobBattle rodar.
func validTarget(w *world.World, e, target *world.Entity) bool {
	if target == nil || target.HP <= 0 {
		return false
	}
	if !world.IsPlayer(target.ID) {
		if target.NonCombatNPC || target.Mode == world.MobEmpty {
			return false
		}
		if e.Summoner != 0 {
			return target.Clan != summonClan // mascotes lutam com criaturas, nunca com outros mascotes
		}
		if chebyshev(e.SegmentX, e.SegmentY, target.X, target.Y) > leashRadius {
			return false
		}
		if target.Summoner != 0 {
			return true // criaturas comuns podem revidar contra mascotes
		}
		return world.ClanHostile(e.Clan, target.Clan)
	}
	if e.Summoner != 0 {
		return false // mascotes nunca atacam jogadores (clã 4 é amistoso, clan.go)
	}
	if sameKingdom(e.Clan, target.Clan) {
		return false
	}
	if m, ok := w.SessionMode(target.ID); !ok || m != world.UserPlay {
		return false
	}
	if world.Village(target.X, target.Y) >= 0 {
		return false // o alvo entrou numa cidade segura — desistir (nada de perseguir para dentro da cidade)
	}
	return chebyshev(e.SegmentX, e.SegmentY, target.X, target.Y) <= leashRadius
}

// mobAttack resolve um golpe (corpo a corpo ou à distância no alcance — o original usa a
// mesma mensagem de ataque para os dois, sem projétil) contra a entidade alvo e
// o transmite para jogadores próximos verem o dano. O dano é autoritativo do servidor
// pela fórmula de combate compartilhada. As barras de HP dos jogadores atualizam pela entrada Dam.
func (d *Dispatcher) mobAttack(w *world.World, id int, e, target *world.Entity) {
	// Rechecar na fronteira de mutação de HP, caso um alvo tenha sido retido ou
	// inserido por outro caminho de batalha. Fazer isto antes da cadência e do RNG para um
	// golpe aliado recusado não mudar nem o AtkTick nem o fluxo de paridade.
	if target == nil || world.IsPlayer(target.ID) && sameKingdom(e.Clan, target.Clan) {
		if target != nil {
			dropCurrentTarget(e, target.ID)
		}
		return
	}
	now := w.Now()
	if now < e.AtkTick+mobAttackCadence {
		return
	}
	e.AtkTick = now

	dmg := combat.ResolveHit(w.Rand(), combat.HitInput{
		AttackerDamage: int(e.Damage) + int(d.weaponDamage(e)),
		TargetAC:       int(target.AC),
		TargetIsPlayer: world.IsPlayer(target.ID),
		Master:         e.Master,
	})
	if dmg > 0 {
		target.HP -= int32(dmg)
		if target.HP < 0 {
			target.HP = 0
		}
		// Baixar o alvo de cura da vítima pelo dano, senão regenPlayers cura
		// de volta no tick seguinte (ProcessSecMinTimer.cpp:2389-2397).
		damageReqHp(w.Session(target.ID), target, int32(dmg))
	}

	body := protocol.MsgAttackBody{
		CurrentHp:  e.HP,
		PosX:       uint16(e.X),
		PosY:       uint16(e.Y),
		TargetX:    uint16(target.X),
		TargetY:    uint16(target.Y),
		AttackerID: uint16(id),
		Dam:        []protocol.DamEntry{{TargetID: int32(target.ID), Damage: int32(dmg)}},
	}
	// HEADER.ID = ESCENE_FIELD, como o ataque de criatura original (GetFunc.cpp GetAttack define
	// sm->ID = ESCENE_FIELD). O cliente aplica Dam[] aos alvos independentemente do header,
	// mas só registra o estado de HP→0 / morte da própria VÍTIMA a partir de um evento de campo/cena;
	// com HEADER.ID = a criatura, o jogador morto não entrava no estado de morte e continuava
	// agindo (atacando, auto-poção). O alvo está em visão, então recebe isto.
	payload := body.Encode()
	w.ForEachInView(id, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgAttack, ID: protocol.IDScene}, payload)
	})

	// Alvos criatura: uma morte feita por mascote recompensa o DONO (MobKilled.cpp:181-190 credita
	// o Summoner); uma criatura que derruba um mascote o remove de vez (removeType
	// 1; a guarda Summoner em DespawnMob o mantém fora da fila de respawn). Qualquer
	// criatura atingida mas viva registra o atacante no próprio EnemyList.
	if !world.IsPlayer(target.ID) {
		if target.HP == 0 {
			if e.Summoner != 0 {
				d.mobKilled(w, e, target)
			} else {
				sendDieAction(w, target)
				w.DespawnMob(target.ID, 1)
				d.kefraMobKilled(w, target)
				d.nightmareMobKilled(w, target)
			}
			dropCurrentTarget(e, target.ID)
		} else {
			setGroupBattle(w, target.ID, target, e)
		}
		return
	}

	// Os próprios mascotes da vítima a defendem (summon.go).
	d.commandSummons(w, target.ID, e)

	// Jogador caído: parar de mirá-lo (o fluxo de morte/ressurreição está adiado).
	if target.HP == 0 {
		if victimSession := w.Session(target.ID); victimSession != nil {
			loss := mobDeathExpLoss(target.Level, target.ClassMaster, target.PKPoint)
			if loss > target.Exp {
				loss = target.Exp
			}
			if loss > 0 {
				target.Exp -= loss
				d.sendEtc(w, victimSession, target)
				d.sendChatText(w, victimSession, fmt.Sprintf("Voce perdeu %d pontos de experiencia", loss))
			}
		}
		dropCurrentTarget(e, target.ID)
	}
}

func dropCurrentTarget(e *world.Entity, targetID int) {
	removeEnemyList(e, targetID)
	e.Target = 0
	if hasEnemyList(e) {
		e.Mode = world.MobCombat
	} else {
		e.Mode = world.MobIdle
	}
}

// mobRoam é o passo fora de combate — o ramo não-invocação de
// StandingByProcessor (CMob.cpp:156-229). Uma criatura com rota percorre seus
// waypoints (pausando SegWait em cada um, depois SetSegment escolhe o próximo); uma criatura
// sem rota volta para a âncora se o combate a deslocou (o
// MOB_RETURN do original, código 16 do BattleProcessor). Invocações (RouteType 5) nunca chegam
// aqui — o Tick as roteia para summonTick (summon.go).
func (d *Dispatcher) mobRoam(w *world.World, id int, e *world.Entity) {
	if isKefraBoss(e) {
		return
	}
	if e.RouteType == 0 || e.RouteType == 5 || !hasWaypoints(e) {
		if e.X != e.SegmentX || e.Y != e.SegmentY {
			d.stepToward(w, id, e, e.SegmentX, e.SegmentY) // voltar para casa
		}
		return
	}
	if e.X == e.SegmentX && e.Y == e.SegmentY { // chegou ao waypoint atual
		if e.RouteType == 3 && e.SegProgress == 4 {
			if e.WaitTicks > 0 {
				e.WaitTicks--
				return
			}
			runeQuestRouteEnd(w, e)
			w.RemoveGeneratorMob(id, 3) // StandingByProcessor 0x10000 → DeleteMob(index, 3)
			return
		}
		if e.RouteType == 6 {
			return // posto de guarda fixo: SetSegment sempre reinicia para 0 (CMob.cpp:167)
		}
		if e.WaitTicks > 0 {
			// Contagem regressiva da pausa (legado WaitSec -= 6 por ciclo de IA, CMob.cpp:192; nosso
			// tick é de 1s então 1 tick ≈ 1s de SegWait — cadência legada exata UNVERIFIED).
			e.WaitTicks--
			if e.WaitTicks > 0 {
				return
			}
		} else if wait := e.SegWait[e.SegProgress]; wait > 0 {
			e.WaitTicks = wait // armar a pausa (CMob.cpp:183-188), avançar quando ela acaba
			return
		}
		if setSegment(e) != 0 {
			return // rota terminou (RouteType 0/3): a criatura ficou parada
		}
	}
	if !d.stepToward(w, id, e, e.SegmentX, e.SegmentY) {
		// Não conseguiu mover em direção ao waypoint (bloqueado/preso): pular para o próximo,
		// como o original faz quando GetNextPos não consegue avançar (CMob.cpp:221-227).
		setSegment(e)
	}
}

// hasWaypoints informa se a rota da instância da criatura tem algum waypoint utilizável
// (0 marca um slot não usado — o caminhante os pula, CMob.cpp:608).
func hasWaypoints(e *world.Entity) bool {
	for i := range e.SegListX {
		if e.SegListX[i] != 0 {
			return true
		}
	}
	return false
}

// setSegment porta CMob::SetSegment (CMob.cpp:494-620): avançar SegProgress na
// direção atual, tratar os fins conforme RouteType, pular waypoints zero, e
// ancorar SegmentX/Y no escolhido. Devolve 0 quando um novo waypoint foi escolhido;
// não-zero quando a rota terminou e a criatura ficou parada.
//
// Comportamento de fim por RouteType (ambos os fins: além de 4 indo para frente, além de 0 indo
// para trás): 1 = reinicia em 0; 2 = ping-pong (também reinicia para frente na ponta
// natal); 3 faria ping-pong aqui, mas StandingByProcessor o apaga no waypoint 4
// antes de chamar SetSegment de novo; 4 = laço circular; 6 = sempre reinicia para 0.
// RouteType 0 percorre a lista uma vez e vira um NPC fixo — o original
// também troca Mode=4 e empacota a direção de frente em MOB.Merchant
// (CMob.cpp:558-579); nós só paramos a caminhada (nosso Merchant carrega os bits de
// cidade de spawn e um valor não-zero desativaria a IA da criatura) e reancoramos no
// waypoint final. Os caminhos de fim RouteType 1/3 do original leem além do array
// (SegmentListX[5]/[-1], um artefato de decompilação) — limitados aqui.
func setSegment(e *world.Entity) int {
	if e.RouteType == 6 {
		e.SegProgress, e.SegDir = 0, 0
		e.SegmentX, e.SegmentY = e.SegListX[0], e.SegListY[0]
		e.WaitTicks = 0
		return 0
	}
	ret := 0
	// Limitado ao dobro do tamanho da lista: todo ramo ou cai num
	// waypoint válido ou reinicia/inverte numa ponta; 12 iterações esgotam todos os estados
	// (evita que uma lista toda zero itere para sempre — hasWaypoints impede
	// isso a montante, isto é redundância).
	for iter := 0; iter < 12; iter++ {
		if e.SegDir == 0 {
			e.SegProgress++
		} else {
			e.SegProgress--
		}
		if e.SegProgress < 0 { // andou para trás além da base
			switch e.RouteType {
			case 1, 2: // reiniciar para frente a partir da base (CMob.cpp:532-541)
				e.SegProgress, e.SegDir = 0, 0
				continue
			case 3: // uma ida e volta concluída: parar na base (CMob.cpp:542-546, limitado)
				e.SegProgress = 0
				ret = 2
			default: // RouteType 0 não deveria chegar aqui (CMob.cpp:523-530)
				e.SegProgress, e.SegDir = 0, 0
				ret = 2
			}
			break
		}
		if e.SegProgress > 4 { // andou para frente além do fim
			switch e.RouteType {
			case 0: // via única: virar um NPC parado no último waypoint
				e.SegProgress = 4
				ret = 1
			case 1: // reiniciar na base (o legado lê fora dos limites aqui — veja o comentário do doc)
				e.SegProgress, e.SegDir = 0, 0
				continue
			case 2, 3: // ping-pong: virar (CMob.cpp:593-598)
				e.SegProgress, e.SegDir = 4, 1
				continue
			default: // 4: circular (CMob.cpp:600-604: -1 e depois o ++ cai em 0)
				e.SegProgress = -1
				continue
			}
			break
		}
		if e.SegListX[e.SegProgress] == 0 {
			continue // slot não usado (CMob.cpp:608)
		}
		break // achou o próximo waypoint
	}
	e.SegmentX, e.SegmentY = e.SegListX[e.SegProgress], e.SegListY[e.SegProgress]
	e.WaitTicks = 0
	return ret
}

// mobRetreat afasta uma criatura à distância de um alvo que chegou ao alcance corpo a corpo
// (GetTargetPosDistance, CMob.cpp:892-954): 1+rand%2 tiles por eixo em que o
// alvo está — X sorteado antes de Y, então no máximo duas chamadas de rand nessa ordem
// (paridade). Caminhar a rota do recuo (BASE_GetRoute) está adiado junto do resto
// do pathfinding; se a célula escolhida está ocupada a criatura mantém a posição neste
// tick. O gate AttackRun&0xF (uma criatura marcada para nunca se reposicionar) não é
// modelado — UNVERIFIED quais templates o carregam.
func (d *Dispatcher) mobRetreat(w *world.World, id int, e, target *world.Entity) {
	nx, ny := e.X, e.Y
	if target.X > e.X {
		nx -= int16(1 + w.Rand().Intn(2))
	} else if target.X < e.X {
		nx += int16(1 + w.Rand().Intn(2))
	}
	if target.Y > e.Y {
		ny -= int16(1 + w.Rand().Intn(2))
	} else if target.Y < e.Y {
		ny += int16(1 + w.Rand().Intn(2))
	}
	if nx == e.X && ny == e.Y {
		return
	}
	// Com a grade de altura carregada, caminhar o recuo como uma rota até a célula
	// escolhida (o original a passa por BASE_GetRoute, CMob.cpp:931) e pousar
	// onde o terreno permitir — nunca atravessando uma parede.
	if d.heights != nil {
		rx, ry, moved := route.Next(d.heights, int(e.X), int(e.Y), int(nx), int(ny), 2)
		if !moved {
			return
		}
		nx, ny = int16(rx), int16(ry)
	}
	if occ, ok := w.EntityAt(nx, ny); ok && occ != id {
		return
	}
	oldX, oldY := e.X, e.Y
	w.SetEntityPos(id, nx, ny)

	body := protocol.MsgActionBody{PosX: oldX, PosY: oldY, Speed: 2, TargetX: nx, TargetY: ny}
	d.moveMulticast(w, id, oldX, oldY, protocol.MsgAction, body.Encode())
}

// mobDistance é BASE_GetDistance (Basedef.cpp:6484): uma consulta euclidiana arredondada
// para deltas até 6 (g_pDistanceTable, Basedef.cpp:180-189), max+1 além disso. Esta
// é a métrica de alcance do BattleProcessor — NÃO Chebyshev (ex.: uma diagonal 3,3
// é distância 4, então não cabe num ataque de alcance 4 como um tiro reto 4,0
// cabe).
func mobDistance(x1, y1, x2, y2 int16) int {
	dx, dy := abs16(x1-x2), abs16(y1-y2)
	if dx <= 6 && dy <= 6 {
		return int(distanceTable[dy][dx])
	}
	if dx > dy {
		return dx + 1
	}
	return dy + 1
}

// distanceTable é g_pDistanceTable[7][7] (Basedef.cpp:180-189), indexado [dy][dx].
var distanceTable = [7][7]uint8{
	{0, 1, 2, 3, 4, 5, 6},
	{1, 1, 2, 3, 4, 5, 6},
	{2, 2, 3, 4, 4, 5, 6},
	{3, 3, 4, 4, 5, 5, 6},
	{4, 4, 4, 5, 5, 5, 6},
	{5, 5, 5, 5, 5, 6, 6},
	{6, 6, 6, 6, 6, 6, 6},
}

// mobStep move a criatura um tile na direção do alvo de combate. Com a grade de altura
// carregada a caminhada mira a célula a NO do alvo — GetTargetPos sempre
// persegue (tx-1, ty-1), CMob.cpp:1034.
func (d *Dispatcher) mobStep(w *world.World, id int, e, target *world.Entity) {
	tx, ty := target.X, target.Y
	if d.heights != nil {
		tx, ty = tx-1, ty-1
	}
	d.stepToward(w, id, e, tx, ty)
}

// stepToward avança a criatura um tile até (tx,ty) e transmite o movimento para
// os clientes animarem. Com a grade de altura carregada o passo vem da
// caminhada de BASE_GetRoute (route.Next), então criaturas deslizam em volta de paredes/água em vez de
// empurrar contra elas; sem a grade (testes, mapas não montados) ela cai no
// passo Chebyshev cego. Devolve false quando a criatura não conseguiu se mover.
//
// Divergência de cadência (UNVERIFIED): o ciclo de IA original roda a cada SECBATTLE=8s
// e salta speed*8/4 tiles de rota por ciclo (≈1-2 tiles/s, o cliente anima); nós
// avançamos 1 tile de rota por tick de 1s — mesmo caminho, mesma ordem de células visitadas
// (cada passo de rota aplica a mesma regra gulosa), ritmo mais suave.
//
// Se a célula escolhida está ocupada a criatura mantém a posição neste tick em vez de
// se empilhar em outra entidade (a checagem pMobGrid em GetTargetPos).
func (d *Dispatcher) stepToward(w *world.World, id int, e *world.Entity, tx, ty int16) bool {
	var nx, ny int16
	if d.heights != nil {
		rx, ry, moved := route.Next(d.heights, int(e.X), int(e.Y), int(tx), int(ty), 1)
		if !moved {
			return false // preso contra o terreno (Route[0]==0): segurar, nunca atravessar
		}
		nx, ny = int16(rx), int16(ry)
	} else {
		nx = e.X + step(tx-e.X)
		ny = e.Y + step(ty-e.Y)
		if nx == e.X && ny == e.Y {
			return false
		}
	}
	if occ, ok := w.EntityAt(nx, ny); ok && occ != id {
		return false
	}
	oldX, oldY := e.X, e.Y
	w.SetEntityPos(id, nx, ny)

	body := protocol.MsgActionBody{PosX: oldX, PosY: oldY, Speed: 2, TargetX: nx, TargetY: ny}
	d.moveMulticast(w, id, oldX, oldY, protocol.MsgAction, body.Encode())
	return true
}

// step devolve o movimento unitário (-1/0/+1) em direção a um delta, para um passo em 8 direções.
func step(delta int16) int16 {
	switch {
	case delta > 0:
		return 1
	case delta < 0:
		return -1
	default:
		return 0
	}
}

// chebyshev é a distância de movimento de rei entre duas células da grade (casa com a
// métrica de visão do mundo).
func chebyshev(x1, y1, x2, y2 int16) int {
	dx := abs16(x1 - x2)
	dy := abs16(y1 - y2)
	if dx > dy {
		return dx
	}
	return dy
}

func abs16(v int16) int {
	if v < 0 {
		return int(-v)
	}
	return int(v)
}
