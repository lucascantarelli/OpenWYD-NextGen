package handler

import (
	"encoding/binary"
	"fmt"

	"openwyd-nextgen/tmserver/internal/combat"
	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// attackCadence é o mínimo de ms entre ataques (handlers/_MSG_Attack.md §4):
// ClientTick < LastAttackTick + 800 ⇒ AddCrackError(1,107).
const attackCadence = 800

// Sentinelas de Dam por alvo (_MSG_Attack.cpp:355): o cliente marca cada
// entrada como corpo a corpo ou skill; qualquer outro dano alegado não-zero é crack.
const (
	damMelee = -2
	damSkill = -1
)

// affectSamaritano é o affect 24 NUM JOGADOR (SkillData.csv:14, a única linha de skill
// que o usa). O mesmo id num MOB é o contador de vida da invocação —
// affectSummonLife, summon.go:25 — por isso toda leitura dele é condicionada por índice.
const affectSamaritano = 24

// removeSamaritano porta DoRemoveSamaritano (Server.cpp:9129), que o legado
// chama em TODO pacote de ataque — corpo a corpo ou conjuração, antes de qualquer dano ser resolvido
// (_MSG_Attack.cpp:274). Samaritano é um buff puramente defensivo: ele deve
// durar só enquanto o dono não está revidando.
//
// Isto roda no TOPO do attack, enquanto a conjuração que instala o affect só
// chega bem depois em applyCastAffect — então conjurar Samaritano nunca cancela o
// buff que está sendo colocado.
func (d *Dispatcher) removeSamaritano(w *world.World, e *world.Entity) {
	if !e.ClearFirstAffect(affectSamaritano) {
		return
	}
	// Perder o buff encolhe MaxHP, então o refresh pode baixar o HP atual —
	// a mesma coisa que o GetCurrentScore do legado faz antes do SendScore.
	d.refreshScore(e)
	if s := w.Session(e.ID); s != nil {
		d.sendScore(w, s, e)
		d.sendAffect(w, s, e)
	}
}

// attack trata _MSG_Attack / _MSG_AttackOne / _MSG_AttackTwo (0x0367/039D/039E),
// handlers/_MSG_Attack.md. O dano é AUTORITATIVO DO SERVIDOR: o Dam[] do cliente
// é recalculado pelas fórmulas de combate (game-rules.md §4) e sobrescrito
// antes do broadcast. Conjurações são validadas (máscara de aprendidas, classe, Passive) e
// têm a mana cobrada aqui, espelhando _MSG_Attack.cpp.
func (d *Dispatcher) attack(w *world.World, s *world.Session, h protocol.Header, payload []byte) {
	if s.TradeMode != 0 {
		return // não pode atacar durante troca automática
	}
	if s.Mode != world.UserPlay {
		return // SendHpMode no original
	}
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}

	var body protocol.MsgAttackBody
	if err := body.Decode(payload); err != nil {
		return
	}

	// Vivacidade: os mortos só podem agir com a skill de ressurreição (99). Usar <= 0 (não
	// == 0) para uma borda de HP negativo nunca deixar passar uma ação.
	if e.HP <= 0 && int(body.SkillIndex) != combat.ResurrectSkill {
		w.AddCrackError(s, 1, 8)
		return
	}

	// Cadência anti-velocidade + sanidade de tick (aritmética int64 evita underflow de uint32 no
	// primeiro ataque, quando LastAttackTick == 0). SkipCheckTick ignora as checagens.
	tick := h.ClientTick
	if tick != protocol.SkipCheckTick {
		last := int64(s.LastAttackTick)
		if int64(tick) < last+attackCadence {
			w.AddCrackError(s, 1, 107) // rápido demais
			return
		}
		if int64(tick) < last-100 {
			w.AddCrackError(s, 4, 7) // tick no passado demais
			return
		}
	}
	s.LastAttackTick = tick
	s.LastAttack = int(body.SkillIndex)

	skillnum := int(body.SkillIndex)
	// GUARDA DE REGRESSÃO (B12, TestMeleeAlwaysDamagesMob): o caminho de skill (máscara de
	// aprendidas, gate de classe, mana) só engata quando o pacote de fato MARCA um
	// acerto de skill — uma entrada de Dam com a sentinela -1. A codificação de corpo a corpo
	// de SkillIndex/Dam do cliente 12000 é UNVERIFIED (agent-prompt-skills.md), e
	// condicionar o ataque inteiro a esses campos uma vez matou silenciosamente todo o dano
	// de jogador→criatura. Corpo a corpo (sem entradas -1) NUNCA pode ser descartado ou
	// ter mana cobrada, diga o SkillIndex o que disser — o dano é autoritativo do servidor.
	skillIntent := false
	for i := range body.Dam {
		if body.Dam[i].Damage == damSkill {
			skillIntent = true
			break
		}
	}
	var cast castInfo
	if skillIntent {
		var ok bool
		cast, ok = d.validateCast(w, s, e, skillnum, tick)
		if !ok {
			return
		}
		if skillnum == 97 && !validateGuardianCannon(w, &body, payload) {
			return
		}
	}
	if cast.isSkill {
		// Mana (BASE_GetManaSpent → abortar porque quebrou): MP insuficiente cancela a conjuração;
		// senão gastar e ecoar o MP autoritativo no corpo do ataque
		// (_MSG_Attack.cpp:240-256). ReqMp é o alvo CUser.ReqMp do servidor,
		// inicializado no login e limitado por SetReqMp.
		spent := combat.ManaSpent(cast.spell.ManaSpent, int(e.SaveMana), cast.special)
		if int(e.MP)-spent < 0 {
			d.sendSetHpMp(w, s, e)
			return
		}
		reqMpBefore := s.ReqMp
		e.MP -= int32(spent)
		s.ReqMp -= int32(spent)
		setReqMp(s, e)
		body.ReqMp = int16(s.ReqMp)

		// Evocação do BM (InstanceType 11, _MSG_Attack.cpp:809-837): gerar as
		// invocações uma vez por conjuração — as entradas de Dam não carregam dano para isso. Uma conjuração
		// que não gera nada devolve sua mana (o legado restaura Mp/ReqMp).
		if cast.spell.InstanceType == 11 {
			count := summonCount(cast.spell.InstanceValue, effectiveSpecial(e, 2))
			if !d.generateSummon(w, s, e, cast.spell.InstanceValue-1, count) {
				e.MP += int32(spent)
				s.ReqMp = reqMpBefore
				body.ReqMp = int16(s.ReqMp)
			}
		}
	}

	if cast.isSkill && skillnum == combat.ResurrectSkill && e.HP <= 0 {
		d.applyBookResurrection(w, s, e)
		body.ReqMp = int16(s.ReqMp)
	}

	// Atacar derruba Samaritano (_MSG_Attack.cpp:274), antes do laço de dano.
	d.removeSamaritano(w, e)

	// Poder de ataque autoritativo do servidor = CurrentScore.Damage + o dano da arma
	// equipada, com os +20% do buff Divino embutidos (effectiveDamage).
	atkDamage := int(d.effectiveDamage(e))
	doubleCritical := uint8(0)
	doubleCriticalReady := false
	var healExp int64
	var hpSyncTargets []int
	// O legado lê o HP do conjurador uma vez, antes do laço de alvos
	// (_MSG_Attack.cpp:281), então Julgamento Divino soma o mesmo HP pré-acerto a
	// todo alvo mesmo que cada acerto o corte para HP/6+1 (issue #392).
	cast.casterHP = e.HP
	for i := range body.Dam {
		tid := int(body.Dam[i].TargetID)
		target := w.Entity(tid)
		if target == nil || target.Mode == world.MobEmpty {
			writeDamage(payload, i, 0)
			continue
		}
		// Alegado por entrada: -1 marca um acerto de skill; todo o resto resolve como
		// corpo a corpo. O legado quebra alegados diferentes de -2/-1/0 (_MSG_Attack.cpp:355),
		// mas a codificação de corpo a corpo do cliente 12000 é UNVERIFIED e o valor
		// alegado é IGNORADO de qualquer modo (autoritativo do servidor) — então tolerar + logar
		// em vez de zerar, senão codificações desconhecidas desativam silenciosamente todo o dano
		// de jogador de novo (B12).
		claim := body.Dam[i].Damage
		if claim != damMelee && claim != damSkill && claim != 0 {
			d.log.Debug("attack: unexpected Dam claim (resolving as melee)",
				"conn", s.Conn, "claim", claim, "skill", skillnum)
		}
		// NPCs intocáveis (lojas/bancos/quest givers/serviços da cidade) nunca levam dano.
		if !world.IsPlayer(tid) && target.NonCombatNPC {
			writeDamage(payload, i, 0)
			continue
		}
		// O ramo isFrag legado trata clãs de reino iguais (7/7 ou 8/8) como
		// aliados e sai desta entrada de alvo antes de SetBattle poder rodar. Aplicar o
		// gate antes da resolução de acerto/efeito para uma skill agressiva não deixar
		// dano, affects, entradas de EnemyList ou estado de batalha de grupo para trás.
		skillHit := cast.isSkill && claim == damSkill
		combatHit := !skillHit || cast.spell.Aggressive != 0
		if !world.IsPlayer(tid) && combatHit && sameKingdom(e.Clan, target.Clan) {
			writeDamage(payload, i, 0)
			continue
		}
		if !d.towerAttackAllowed(e, target) {
			writeDamage(payload, i, 0)
			continue
		}
		// Alvos mortos só podem ser atingidos pelas skills de ressurreição (31/99)
		// (_MSG_Attack.cpp:333).
		if target.HP <= 0 && skillnum != 31 && skillnum != combat.ResurrectSkill {
			writeDamage(payload, i, 0)
			continue
		}
		// skillHit: esta entrada resolve pelo pipeline de skill; qualquer
		// outra coisa (sentinela -2, 0, ou um alegado desconhecido) é corpo a corpo.
		pvpHit := world.IsPlayer(tid) && tid != s.Conn
		// Canal verde (novato): sem PvP. O legado bloqueia o ramo inteiro de PvP
		// quando NewbieEventServer está ligado (_MSG_Attack.cpp:405), e o duelo
		// incluído — o ponto do canal é ninguém morrer para outro jogador.
		if pvpHit && combatHit && d.isNoviceChannel() {
			writeDamage(payload, i, 0)
			continue
		}
		// Gate de PvP: dano de combate (corpo a corpo ou skill agressiva) exige o
		// atacante optar pelo modo PK (tecla K, _MSG_PKMode). O servidor legado
		// deixa isso para o cliente; mantemos como guarda anti-cheat, que o
		// cliente não modificado nunca percebe porque ele já recusa
		// atacar sem K. Não usar world.Village como aproximação de zona segura
		// de cidade aqui: os retângulos grosseiros de cidade causaram a issue #67 ao
		// zerar o dano PvP perto de todo ponto de spawn.
		if pvpHit && combatHit && !e.PKMode && !d.dueling(s.Conn, tid) && !d.towerPvP(e, target) {
			writeDamage(payload, i, 0)
			continue
		}
		// Gate legado do lado do servidor (_MSG_Attack.cpp:405-419, issue #420): num tile
		// de arena PvP do AttributeMap (bit 0x40, sob o ATACANTE), um alvo
		// que não tem K ligado nem Guilty não leva dano. Sem exceção de duelo —
		// o tile da arena de duelo é 0x44 e o legado não tem nenhuma. towerPvP substitui
		// a guarda de estado de guerra legada (RvR/cerco de castelo não são modelados).
		if pvpHit && combatHit && !d.towerPvP(e, target) && d.pvpArenaTile(e.X, e.Y) && !target.PKMode && !pkGuilty(target) {
			writeDamage(payload, i, 0)
			continue
		}
		// Um atacante maximamente caótico não pode ferir um alvo comparativamente limpo
		// (_MSG_Attack.cpp "PK - War - Miss": pointPK<=10 && SummonerPointPK>10 ⇒
		// dam=0 + _DN_CantKillUser). Compara o byte cru de PKPoint, não o valor
		// de exibição -75. Como o carimbo Guilty abaixo, só roda fora de cidades
		// e áreas de guerra de guilda (arena == 5 && village == 5, posição do atacante;
		// issue #421).
		if pvpHit && combatHit && !d.dueling(s.Conn, tid) && !world.PvPZone(e.X, e.Y) && int(e.PKPoint) <= 10 && int(target.PKPoint) > 10 {
			d.sendClientMessage(w, s, fmt.Sprintf(cantKillUser, int(e.PKPoint)))
			writeDamage(payload, i, 0)
			continue
		}

		var dmg int
		if skillHit {
			if !d.validateSkillTarget(w, s, e, target, i, cast, tick) {
				writeDamage(payload, i, 0)
				continue
			}
			dmg = d.resolveSkillHit(w, e, target, tid, skillnum, cast)
			skipGenericAffect := d.applySkillSpecial(w, s, e, target, tid, skillnum, cast, &body, &dmg)
			// O cliente pode auto-mirar linhas de skill agressivas pela hotbar.
			// Não transformar essas linhas em dano contra o conjurador; custos explícitos
			// de HP, como Julgamento Divino, são tratados em applySkillSpecial.
			if dmg > 0 && tid == s.Conn && cast.spell.Aggressive != 0 && skillnum != 30 {
				dmg = 0
			}
			if dmg > 0 && tid != s.Conn {
				if miss := combat.ResolveParry(w.Rand(), skillnum, d.parryRate(e, target), target.Rsv&world.RsvBlock != 0); miss != 0 {
					dmg = miss
				}
			}
			if !skipGenericAffect {
				d.applyCastAffect(w, e, target, tid, cast)
			}
		} else {
			if !doubleCriticalReady {
				progress := body.Progress
				doubleCritical, _ = combat.DoubleCritical(w.Rand(), attackRunOf(e), int(effectiveCritical(e)), &s.CriticalProgress, &progress)
				body.Progress = progress
				body.DoubleCritical = doubleCritical
				writeAttackProgress(payload, progress)
				writeDoubleCritical(payload, doubleCritical)
				doubleCriticalReady = true
			}
			dmg = combat.ResolveHit(w.Rand(), combat.HitInput{
				AttackerDamage: atkDamage,
				TargetAC:       int(effectiveAC(target)),
				TargetIsPlayer: world.IsPlayer(tid),
				DoubleCritical: doubleCritical,
				Master:         e.Master,
				UseSkill:       false,
				SkillIndex:     skillnum,
				ParryRate:      d.parryRate(e, target),
				TargetRsvBlock: target.Rsv&world.RsvBlock != 0,
			})
			dmg = d.applyAirBladeProc(w, e, target, h.Type, &body, payload, dmg)
		}
		if dmg > 0 {
			dmg = applyHuntressForceDamage(e, target, tid, dmg)
			dmg = d.applyManaControl(w, e, target, tid, dmg)
			hpBefore := target.HP
			target.HP -= int32(dmg)
			if target.HP < 0 {
				target.HP = 0
			}
			ts := w.Session(tid)
			// Baixar o alvo de cura da vítima pelo dano, senão o tick de regen o cura
			// de volta na hora (_MSG_Attack.cpp:1638-1642).
			damageReqHp(ts, target, int32(dmg))
			if ts != nil && target.HP != hpBefore {
				seen := false
				for _, syncID := range hpSyncTargets {
					if syncID == tid {
						seen = true
						break
					}
				}
				if !seen {
					hpSyncTargets = append(hpSyncTargets, tid)
				}
			}
			d.applyOnHitAffects(w, e, target, tid)
			d.applyHpAbs(w, s, e, dmg)
			// Acertar um golpe PvP num alvo comparativamente limpo (PKPoint>10)
			// marca OS DOIS lados como Guilty (_MSG_Attack.cpp: SetGuilty(conn,8);
			// SetGuilty(idx,8)) — retransmitindo o nick de qualquer lado que ainda não
			// estivesse vermelho. Um golpe de duelo nunca pode marcar PK (critérios de aceite
			// da issue #118), e nem um golpe dado de dentro de uma cidade ou de uma
			// área de guerra de guilda (posição do atacante, issue #421).
			if pvpHit && combatHit && !d.dueling(s.Conn, tid) && !world.PvPZone(e.X, e.Y) && int(target.PKPoint) > 10 {
				d.markGuilty(w, s, e)
				d.markGuilty(w, w.Session(tid), target)
			}
		} else if dmg < 0 && cast.isSkill && cast.spell.InstanceType == 6 {
			// Cura: um Dam negativo é a quantidade curada; limitar ao máximo do alvo.
			before := target.HP
			target.HP += d.foemaHealAmount(target, -int32(dmg))
			if m := effectiveMaxHP(target); target.HP > m {
				target.HP = m
			}
			if ts := w.Session(tid); ts != nil {
				ts.ReqHp = target.HP
				setReqHp(ts, target)
			}
			healExp += healExpGain(e, target, before)
		}
		// Uma criatura atingida ou morre (recompensas) ou registra o atacante no
		// próprio EnemyList — e arrasta seu grupo de spawn para a luta (SetBattle +
		// propagação por PartyList, setGroupBattle). A provocação acontece mesmo num
		// acerto bloqueado, igual ao AddEnemyList-no-ataque original.
		if !world.IsPlayer(tid) {
			if target.HP == 0 {
				d.mobKilled(w, e, target)
			} else {
				setGroupBattle(w, tid, target, e)
				// As invocações do atacante entram na luta do dono (a propagação de
				// EnemyList legada; summon.go).
				d.commandSummons(w, s.Conn, target)
			}
		} else if pvpHit && target.HP == 0 {
			if d.dueling(s.Conn, tid) {
				// Eliminações em duelo são tratadas pela varredura da arena de duelo (duel.go), não
				// como uma morte PvP relevante para o caos — uma morte em duelo não pode conceder/custar
				// PKPoint ou EXP (mesma exclusão do gate de Guilty acima). A morte
				// legada ainda cai no ramo de arena do MobKilled via ZoneUnk
				// (map 1,31; issue #427), então a vítima lê o texto de sem perda de EXP.
				if ts := w.Session(tid); ts != nil {
					d.sendClientMessage(w, ts, inArenaNoExpLoss)
				}
			} else {
				d.pvpKilled(w, e, target)
			}
		}
		writeDamage(payload, i, int32(dmg))
	}

	if healExp > 0 && !archExpLocked(e) {
		if healExp > 200 {
			healExp = 200
		}
		e.Exp += healExp
		if e.Exp > level.MaxExp {
			e.Exp = level.MaxExp
		}
		d.applyLevelUpsWithoutCirclet(w, s, e)
	}

	// Sobrescrever o status do atacante com os valores autoritativos do servidor para todo
	// destinatário (e o próprio cliente do atacante) ver o HP/MP real e a
	// experiência pós-morte. CurrentExp é como o cliente atualiza sua barra de exp — não há
	// pacote de exp separado (MSG_UpdateScore não carrega exp).
	writeAttackerStatus(payload, e.HP, e.MP, e.Exp, body.ReqMp)

	// Transmitir o resultado autoritativo do servidor com HEADER.ID = ESCENE_FIELD, exatamente
	// como o original (_MSG_Attack.cpp:25 `m->ID = ESCENE_FIELD`). Isto importa para a
	// barra de exp: o cliente aplica Dam[] aos alvos nomeados independentemente do header (então um
	// ataque de criatura com HEADER.ID = a criatura ainda machuca o jogador), mas só aplica
	// o CurrentExp/CurrentHp/CurrentMp do PRÓPRIO ATACANTE quando o ataque chega como
	// evento de campo/cena. Com HEADER.ID = o conn do atacante a barra de exp nunca mexia.
	// O GridMulticast original (em volta do alvo) inclui o atacante, então nós dois
	// ecoamos para o atacante e enviamos aos jogadores em visão.
	hdr := protocol.Header{Type: protocol.MsgAttack, ID: protocol.IDScene}
	w.SendTo(s, hdr, payload)
	w.ForEachInView(s.Conn, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, hdr, payload)
	})
	if cast.isSkill && skillnum == 30 {
		d.sendSetHpMp(w, s, e) // um sync após cada acerto de Julgamento (_MSG_Attack.cpp:1749)
	}
	for _, tid := range hpSyncTargets {
		ts := w.Session(tid)
		target := w.Entity(tid)
		if ts != nil && target != nil {
			d.sendSetHpMp(w, ts, target)
		}
	}
}

// castInfo é o contexto de skill resolvido para um pacote de ataque.
type castInfo struct {
	isSkill bool
	spell   content.Spell
	special int // CurrentScore.Special[kind] (ou Level para skills Sephira)
	master  int // maestria de mitigação de skill (regra bit-14 do TK; 0 caso contrário)
	// casterHP é o HP do conjurador antes do laço de alvos (o `hp` legado).
	casterHP int32
}

// validateCast roda os gates de skill de _MSG_Attack.cpp:98-215 para SkillIndex ∈
// [0,MaxSkillIndex): rejeitar Passive, gate de classe, checagem de máscara de aprendidas (crack), e
// resolve o Special/maestria que as fórmulas de dano precisam. Corpo a corpo (SkillIndex fora do
// intervalo, tipicamente -1) passa com isSkill=false. Devolve ok=false quando
// o ataque inteiro precisa ser descartado.
func (d *Dispatcher) validateCast(w *world.World, s *world.Session, e *world.Entity, skillnum int, tick uint32) (castInfo, bool) {
	if skillnum < 0 || skillnum >= content.MaxSkillIndex || d.spells == nil {
		return castInfo{}, true // corpo a corpo
	}
	spell, ok := d.spells.Get(skillnum)
	if !ok {
		return castInfo{}, false
	}
	if spell.Passive == 1 && tick != protocol.SkipCheckTick {
		return castInfo{}, false
	}
	cast := castInfo{isSkill: true, spell: spell}
	if tick != protocol.SkipCheckTick {
		// O gate de classe só se aplica aos quatro blocos de classe base. Linhas Shared/Sephira
		// (96+) o pulam, mas ainda usam o mesmo gate vivo de aprendidas:
		// LearnedSkill & (1 << (skillnum % 24)). O antigo ramo SecLearnedSkill em
		// _MSG_Attack é inalcançável porque fica sob skillnum>=MAX_SKILLINDEX.
		if content.SkillClass(skillnum) <= 3 && content.SkillClass(skillnum) != int(e.Class) {
			return castInfo{}, false
		}
		if e.LearnedSkill&learnedSkillBit(skillnum) == 0 {
			w.AddCrackError(s, 8, 10)
			return castInfo{}, false
		}
	}
	cast.special = effectiveSpecial(e, content.SkillKind(skillnum))
	// Escudo Dourado (85) cobra 100×Special de ouro ao conjurar (_MSG_Attack.cpp:222).
	if skillnum == 85 {
		coin := int32(100 * cast.special)
		if e.Coin < coin {
			return castInfo{}, false
		}
		e.Coin -= coin
		d.sendEtc(w, s, e)
	}
	// Maestria de mitigação de skill: só um TK com o bit 14 aprendido ganha Special[2]/20
	// (limitado 0..15); todos os outros conjuram com 0 (_MSG_Attack.cpp:258-268).
	if e.Class == 0 && e.LearnedSkill&(1<<14) != 0 {
		m := int(e.Special[2]) / 20
		if m < 0 {
			m = 0
		}
		if m > 15 {
			m = 15
		}
		cast.master = m
	}
	return cast, true
}

func learnedSkillBit(skillnum int) int32 {
	return int32(1) << uint(skillnum%content.MaxSkill)
}

func (d *Dispatcher) validateSkillTarget(w *world.World, s *world.Session, caster, target *world.Entity, targetSlot int, cast castInfo, tick uint32) bool {
	sp := cast.spell
	// MaxTarget é um índice máximo legado de Dam[], não uma contagem: _MSG_Attack.cpp rejeita
	// só i > MaxTarget. Manter esse formato off-by-one para paridade de pacote.
	if tick != protocol.SkipCheckTick && sp.MaxTarget >= 0 && targetSlot > sp.MaxTarget {
		w.AddCrackError(s, 10, 28)
		return false
	}
	if (sp.Index == 41 || sp.Index == 44) && targetSlot >= foemaMultiBuffTargetCap(cast.special) {
		return false
	}
	if sp.BParty != 0 && !skillSameLeaderOrGuild(w, caster, target) {
		w.AddCrackError(s, 10, 27)
		return false
	}
	if sp.Range > 0 && mobDistance(caster.X, caster.Y, target.X, target.Y) > sp.Range {
		return false
	}
	// TargetType é, fora isso, orientação de cliente/UI no _MSG_Attack
	// legado local. O único gate seguro no servidor é o tipo de alvo 0 para efeitos só-em-si;
	// skills de grupo e invocações são isentas porque seu espalhamento de alvos é
	// controlado por bParty/InstanceType.
	if sp.TargetType == 0 && sp.BParty == 0 && sp.InstanceType != 11 && sp.Aggressive == 0 &&
		(sp.AffectType > 0 || sp.TickType > 0) && target.ID != caster.ID {
		return false
	}
	return true
}

func foemaMultiBuffTargetCap(special int) int {
	n := special/25 + 2
	if n >= protocol.MaxTarget {
		return protocol.MaxTarget
	}
	if n <= 1 {
		return 2
	}
	return n
}

func skillSameLeaderOrGuild(w *world.World, a, b *world.Entity) bool {
	leader := a.Leader
	if leader == 0 {
		leader = a.ID
	}
	targetLeader := b.Leader
	if targetLeader == 0 {
		targetLeader = b.ID
	}
	if leader == targetLeader {
		return true
	}
	guild := int(a.Guild)
	if s := w.Session(a.ID); s != nil && s.GuildDisable {
		guild = 0
	}
	targetGuild := int(b.Guild)
	if s := w.Session(b.ID); s != nil && s.GuildDisable {
		targetGuild = 0
	}
	return guild != 0 && guild == targetGuild
}

func validateGuardianCannon(w *world.World, body *protocol.MsgAttackBody, payload []byte) bool {
	if body.PosX == 0 || body.PosY == 0 || int(body.PosX) >= w.GridDim() || int(body.PosY) >= w.GridDim() {
		return false
	}
	gi := w.GroundItemAt(int16(body.PosX), int16(body.PosY))
	if gi == nil || gi.Item.Index != 746 {
		return false
	}
	body.Motion = 1
	if len(payload) >= protocol.MsgAttackDamOffset {
		payload[34] = 1
	}
	return true
}

// applyCastAffect aplica o affect/tick de uma conjuração a um alvo (_MSG_Attack.cpp
// bloco de buff ~1172-1240): conjurações agressivas pulam aliados (mesmo líder/guilda) e
// sorteiam o gate de resistência; depois SetAffect (só jogadores) + SetTick entram e o
// score/ícones do alvo atualizam. Delay = 100 + Special, Level = Special.
func (d *Dispatcher) applyCastAffect(w *world.World, e, target *world.Entity, tid int, cast castInfo) {
	sp := cast.spell
	if sp.AffectType <= 0 && sp.TickType <= 0 {
		return
	}
	if sp.Aggressive != 0 {
		if !world.IsPlayer(tid) && target.NonCombatNPC {
			return
		}
		leader := e.Leader
		if leader == 0 {
			leader = e.ID
		}
		tleader := target.Leader
		if tleader == 0 {
			tleader = tid
		}
		guild, tguild := int(e.Guild), int(target.Guild)
		if guild == 0 && tguild == 0 {
			guild = -1
		}
		if leader == tleader || guild == tguild {
			return // aliados nunca recebem affects hostis
		}
		// Sorteio de resistência: rand()%100 vs RegenMP + AffectResist + vantagem de nível.
		// UNVERIFIED: RegenMP ainda não é modelado na Entity (0); somadores de nível
		// de tier (celestial +MAX_LEVEL) esperam o sistema de tiers.
		if sp.AffectResist >= 1 && sp.AffectResist <= 4 {
			difLevel := -(int(target.Level) - int(e.Level)) / 2
			if w.Rand().Intn(100) > sp.AffectResist+difLevel {
				return
			}
		}
		if target.Rsv&world.RsvBlock != 0 {
			return
		}
		if world.IsPlayer(e.ID) && target.Clan == 6 {
			return // clã 6 é imune a affects conjurados por jogadores
		}
	}
	delay := 100 + cast.special
	applied := target.SetAffect(sp.AffectType, sp.AffectValue, sp.AffectTime, sp.Aggressive, delay, cast.special, d.affectDur)
	if target.SetTick(sp.TickType, sp.TickValue, sp.AffectTime, sp.Aggressive, delay, cast.special, d.affectDur) {
		applied = true
	}
	if !applied {
		return
	}
	// Uma transformação que entra (skills 64/66/68/70/71) também troca a malha do corpo, que
	// todo mundo em visão precisa renderizar — o legado segue o SetAffect com
	// GetCurrentScore + SendScore + SendEquip(conn,0) (_MSG_Attack.cpp:1242-1248).
	// refreshEquip recalcula equipamento/brilho visual (onde fica a sobreposição de fera),
	// transmite UpdateEquip para si+em visão e reenvia o score.
	if sp.AffectType == affectTransform {
		if ts := w.Session(tid); ts != nil {
			d.refreshEquip(w, ts, target)
			d.sendAffect(w, ts, target)
			return
		}
	}
	d.refreshScore(target)
	if ts := w.Session(tid); ts != nil {
		d.sendScore(w, ts, target) // UpdateScore carrega o array de ícones Affect[32]
		d.sendAffect(w, ts, target)
	}
}

// resolveSkillHit roda o pipeline de skill por alvo (_MSG_Attack.cpp:552-607):
// poder bruto (SkillBaseDamage) → mitigação (SkillDamage com def ×2 vs jogadores,
// ×1,5 vs Foema) → escala de resistência elemental. InstanceType 6 devolve um valor
// NEGATIVO (a quantidade de cura, como o pacote a codifica). InstanceTypes que não são de dano
// (buffs/especiais) devolvem 0 aqui — o motor de affect (M4) cuida deles.
func (d *Dispatcher) resolveSkillHit(w *world.World, e, target *world.Entity, tid, skillnum int, cast castInfo) int {
	sp := combat.SkillSpell{
		InstanceType:  cast.spell.InstanceType,
		InstanceValue: cast.spell.InstanceValue,
		AffectValue:   cast.spell.AffectValue,
	}
	caster := combat.SkillCaster{
		Class:   int(e.Class),
		Level:   int(e.Level),
		Str:     int(effectiveStr(e)),
		Int:     int(effectiveInt(e)),
		Damage:  int(d.effectiveDamage(e)),
		Magic:   int(effectiveMagic(e)),
		Special: cast.special,
	}
	// CurrentWeather escala a saída de InstanceType 2/3/5 (_MSG_Attack.cpp:520,594,972
	// → BASE_GetSkillDamage). Clima 0 é neutro, então isto é no-op até um
	// sorteio ou um override de GM movê-lo (weather.go).
	raw := combat.SkillBaseDamage(skillnum, sp, caster, int(d.currentWeather()), int(d.weaponDamage(e)))

	switch {
	case sp.InstanceType >= 1 && sp.InstanceType <= 5:
		if skillnum == 79 {
			def := int(effectiveAC(target))
			if world.IsPlayer(tid) {
				def *= 3
			}
			dmg := combat.Damage(w.Rand(), raw, def, cast.master)
			if dmg > 0 {
				dmg /= 2
			}
			return dmg
		}
		def := int(effectiveAC(target))
		if world.IsPlayer(tid) {
			def *= 2
		}
		if target.Class == 1 { // Foema resiste a skills ×1,5
			def = def * 3 / 2
		}
		dmg := combat.SkillDamage(w.Rand(), raw, def, cast.master)
		var resist [4]int16
		for k := range resist {
			resist[k] = effectiveResist(target, k)
		}
		return combat.SkillResistScale(dmg, sp.InstanceType, resist, world.IsPlayer(tid))
	case sp.InstanceType == 6:
		if target.Clan == 4 {
			return 0
		}
		heal := 3*cast.special/2 + cast.spell.InstanceValue
		if skillnum == 27 {
			heal = 2*cast.special + cast.spell.InstanceValue
		}
		healCap := 1100
		if e.ClassMaster != classMasterMortal && e.ClassMaster != classMasterArch {
			heal *= 2
			healCap = 2200
		}
		if heal >= healCap {
			heal = healCap
		}
		if heal > 0 && heal < 6 {
			heal = 6
		}
		return -heal // cura vai como dano negativo
	default:
		return 0 // buffs/especiais: sem dano direto (affects entram em M4)
	}
}

func (d *Dispatcher) applySkillSpecial(w *world.World, s *world.Session, e, target *world.Entity, tid, skillnum int, cast castInfo, body *protocol.MsgAttackBody, dmg *int) bool {
	switch {
	case skillnum == 6: // Furia Divina: só deslocamento.
		*dmg = 0
		d.applyDivineFury(w, e, target, tid, cast.special)
		return true

	case cast.spell.InstanceType == 7: // Flash: limpar estado de combate.
		target.Target = 0
		clearEnemyList(target)
		if !world.IsPlayer(tid) && target.Mode == world.MobCombat {
			target.Mode = world.MobPeace
		}
		return true

	case cast.spell.InstanceType == 8:
		d.clearDetoxAffects(w, e, target, tid)
		if skillnum == 31 {
			d.applyFoemaResurrection(w, s, e, target, tid, body)
		}
		return true

	case cast.spell.InstanceType == 9:
		d.applyFoemaSummon(w, s, e, target, tid, cast)
		return true

	case cast.spell.InstanceType == 12: // Chamas Etereas.
		d.applyEtherealFlame(w, e, target, tid)
		return true

	case skillnum == 30: // Julgamento Divino.
		*dmg += int(cast.casterHP)
		e.HP = e.HP/6 + 1
		s.ReqHp = e.HP
		setReqHp(s, e)
		return false // attack envia o único SetHpMp depois do laço

	case skillnum == 22: // Exterminar gasta todo o MP restante no acerto.
		currentMP := e.MP
		e.MP = 0
		s.ReqMp = 0
		body.ReqMp = 0
		*dmg += int(currentMP) + int(effectiveInt(e))/2
		d.applyExterminarMotion(w, e, target, tid)
		return false

	case skillnum == 98: // Muro de Espinhos cria uma criatura Vinha/Vine.
		*dmg = 0
		d.createVine(w, body)
		return true

	case skillnum == 47: // Cancelamento remove o bloqueio primeiro.
		if target.ClearFirstAffect(19) {
			d.refreshScore(target)
			if ts := w.Session(tid); ts != nil {
				d.sendScore(w, ts, target)
				d.sendAffect(w, ts, target)
			}
			return true
		}
		return false
	}
	return false
}

func (d *Dispatcher) applyDivineFury(w *world.World, caster, target *world.Entity, tid, special int) bool {
	if target == nil {
		return false
	}
	if !world.IsPlayer(tid) && target.NonCombatNPC {
		return false
	}
	switch target.Equip[0].Index {
	case 219, 220, 362:
		return false
	}
	if target.GenIndex == 8 || target.GenIndex == 9 || target.Clan == 6 {
		return false
	}

	nx, ny := caster.X, caster.Y
	if nx < target.X {
		nx++
	} else if nx > target.X {
		nx--
	}
	if ny < target.Y {
		ny++
	} else if ny > target.Y {
		ny--
	}
	x, y, ok := d.freeCellAtOrNear(w, nx, ny)
	if !ok {
		return false
	}

	kindValue := special/10 + 20
	if !world.IsPlayer(tid) {
		kindValue = special/5 + 40
	}
	chance := kindValue + divineFuryLevelDiff(caster, target)/4
	if chance > 50 {
		chance = 50
	}
	if w.Rand().Intn(100) >= chance {
		return false
	}
	target.Route[0] = 0
	d.moveEntityWithAction(w, tid, x, y, 2, 6)
	if !world.IsPlayer(tid) {
		setGroupBattle(w, tid, target, caster)
	}
	return true
}

func divineFuryLevelDiff(caster, target *world.Entity) int {
	casterLevel := int(caster.Level)
	if caster.ClassMaster != classMasterMortal && caster.ClassMaster != classMasterArch {
		casterLevel += int(level.MaxLevel)
	}
	targetLevel := int(target.Level)
	if target.ClassMaster != classMasterMortal && target.ClassMaster != classMasterArch {
		targetLevel += int(level.MaxLevel)
	}
	return casterLevel - targetLevel
}

func (d *Dispatcher) applyExterminarMotion(w *world.World, caster, target *world.Entity, tid int) bool {
	if target == nil {
		return false
	}
	if target.Equip[0].Index == 219 || target.Equip[0].Index == 220 {
		return false
	}
	if !world.IsPlayer(tid) && target.NonCombatNPC {
		return false
	}
	x, y, ok := d.freeCellNear(w, target.X, target.Y)
	if !ok {
		return false
	}
	d.moveEntityWithAction(w, tid, x, y, 2, 2)
	if !world.IsPlayer(tid) {
		setGroupBattle(w, tid, target, caster)
	}
	return true
}

func (d *Dispatcher) moveEntityWithAction(w *world.World, id int, x, y int16, effect, speed int32) bool {
	e := w.Entity(id)
	if e == nil {
		return false
	}
	oldX, oldY := e.X, e.Y
	w.SetEntityPos(id, x, y)
	body := protocol.MsgActionBody{PosX: oldX, PosY: oldY, Effect: effect, Speed: speed, TargetX: x, TargetY: y}
	payload := body.Encode()
	d.moveMulticast(w, id, oldX, oldY, protocol.MsgAction, payload)
	if s := w.Session(id); s != nil && s.Mode == world.UserPlay {
		w.SendTo(s, protocol.Header{Type: protocol.MsgAction, ID: uint16(id)}, payload)
	}
	return true
}

func (d *Dispatcher) createVine(w *world.World, body *protocol.MsgAttackBody) bool {
	if d.vineMob == nil || body == nil {
		return false
	}
	x, y := int16(body.TargetX), int16(body.TargetY)
	if !d.cellAvailable(w, x, y) {
		return false
	}
	id := w.SpawnMobAt(world.MobSpawn{Template: d.vineMob, X: x, Y: y, RouteType: 3, GenIndex: -1})
	if id < 0 {
		return false
	}
	mob := w.Entity(id)
	if mob == nil {
		return false
	}
	mob.Mode = world.MobPeace
	mob.WaitTicks = 40
	payload := protocol.EncodeCreateMobBody(d.createMobFrom(mob, 2))
	w.ForEachInView(id, func(vs *world.Session, _ *world.Entity) {
		if w.MarkSeen(vs, id) {
			w.SendTo(vs, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene}, payload)
		}
	})
	return true
}

func (d *Dispatcher) clearDetoxAffects(w *world.World, caster, target *world.Entity, tid int) {
	changed := false
	for i := range target.Affect {
		t := target.Affect[i].Type
		if t == 1 || t == 3 || t == 5 || t == 7 || t == 10 || t == 12 || t == 20 ||
			(t == 32 && caster.LearnedSkill&(1<<7) != 0) {
			target.Affect[i] = world.Affect{}
			changed = true
		}
	}
	if !changed {
		return
	}
	d.refreshScore(target)
	if ts := w.Session(tid); ts != nil {
		d.sendScore(w, ts, target)
		d.sendAffect(w, ts, target)
	}
}

func (d *Dispatcher) applyFoemaResurrection(w *world.World, s *world.Session, caster, target *world.Entity, tid int, body *protocol.MsgAttackBody) {
	hp := int32((w.Rand().Intn(10) + 10) * int((effectiveMaxHP(caster)+1)/100))
	caster.MP = 0
	s.ReqMp = 0
	body.ReqMp = 0
	d.sendSetHpMp(w, s, caster)
	if w.Rand().Intn(100) >= 70 {
		return
	}
	target.HP = hp
	if ts := w.Session(tid); ts != nil {
		ts.CrackError = 0
		ts.ReqHp = target.HP
		setReqHp(ts, target)
		d.sendScore(w, ts, target)
		d.sendSetHpMp(w, ts, target)
		d.sendEtc(w, ts, target)
	}
	bodyPayload := protocol.EncodeCreateMobBody(d.createMobFrom(target, 0))
	w.ForEachInView(tid, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene}, bodyPayload)
	})
}

// attrPvPArena é o bit do AttributeMap das arenas PvP (arena de Armia 0x40,
// arena de duelo 0x44), testado pelo gate de alvo legado (_MSG_Attack.cpp:409).
const attrPvPArena = 0x40

// pvpArenaTile informa se (x,y) fica num tile de arena PvP do AttributeMap.
func (d *Dispatcher) pvpArenaTile(x, y int16) bool {
	return d.attrs != nil && x >= 0 && y >= 0 && d.attrs.At(int(x)/4, int(y)/4)&attrPvPArena != 0
}

// Recusas do Teleporte (tmserver/content/TMsrv/run/Language.txt, cp1252).
const (
	msgCantSummonDeadPerson = "Personagem derrotado n\xe3o pode ser invocado ou transferido." // _NN_Cant_Summon_Dead_Person (48)
	msgSummonNotAllowedHere = "Voc\xea n\xe3o pode invocar ou transferir nesta zona."         // _NN_Summon_Not_Allowed_Here (49)
	msgTooHighLevelToSummon = "O n\xedvel do oponente \xe9 muito alto para isto."             // _NN_Too_High_Level_To_Summon (50)
	msgSummonedBy           = "Convocado por [%s]."                                           // _SN_Summoned_By_S (52)
	attrNoSummon            = 0x04                                                            // Bit do AttributeMap testado pelo Teleporte
	foemaSummonMaxCasterLvl = 1000                                                            // nível guardado a partir do qual o bit de zona é ignorado (GM)
	foemaSummonHPAllowance  = 30
)

// applyFoemaSummon porta o Teleporte (skill 42, InstanceType 9,
// _MSG_Attack.cpp:751-786): o alvo é puxado para perto da posição
// do Foema no servidor. O TargetX/Y do pacote nunca é destino, então um
// self-cast deixa o Foema onde ele está (issue #393).
func (d *Dispatcher) applyFoemaSummon(w *world.World, s *world.Session, caster, target *world.Entity, tid int, cast castInfo) bool {
	if caster == nil || target == nil {
		return false
	}
	if target.HP <= 0 {
		d.sendClientMessage(w, s, msgCantSummonDeadPerson)
		return false
	}
	if d.attrs != nil && caster.X >= 0 && caster.Y >= 0 &&
		d.attrs.At(int(caster.X)/4, int(caster.Y)/4)&attrNoSummon != 0 && caster.Level < foemaSummonMaxCasterLvl {
		d.sendClientMessage(w, s, msgSummonNotAllowedHere)
		return false
	}
	if target.HP > caster.HP+int32(cast.special)+foemaSummonHPAllowance {
		d.sendClientMessage(w, s, msgTooHighLevelToSummon)
		return false
	}
	if target.X&^0xFF == 0 && target.Y&^0xFF == 0 {
		d.notify(w, s, NoticeCantUseHere) // _NN_Cant_Use_That_Here: alvo no canto de mapa não usado
		return false
	}
	if !world.IsPlayer(tid) {
		return false
	}
	ts := w.Session(tid)
	if ts == nil || ts.Mode != world.UserPlay {
		return false
	}
	// GetEmptyMobGrid conta a célula do próprio movedor como livre, então um self-cast
	// teleporta no lugar em vez de pular para a próxima célula.
	x, y := caster.X, caster.Y
	if id, occupied := w.EntityAt(x, y); !occupied || id != tid {
		var ok bool
		if x, y, ok = d.freeCellAtOrNear(w, x, y); !ok {
			return false
		}
	}
	d.moveEntityWithAction(w, tid, x, y, 1, 2)
	d.sendClientMessage(w, ts, fmt.Sprintf(msgSummonedBy, caster.Name))
	if c := world.Village(x, y); c >= 0 && c <= 3 {
		target.LastCity = int16(c)
	}
	return true
}

func (d *Dispatcher) applyManaControl(w *world.World, caster, target *world.Entity, tid, dmg int) int {
	if dmg <= 0 || target == nil || !target.HasAffect(18) {
		return dmg
	}
	ts := w.Session(tid)
	if ts == nil {
		return dmg
	}
	reduced, spent, ok := manaControlDamage(target, dmg, caster != nil && caster.LearnedSkill&(1<<23) != 0)
	if !ok {
		return dmg
	}
	ts.ReqMp -= spent
	setReqMp(ts, target)
	d.sendSetHpMp(w, ts, target)
	return reduced
}

// applyHpAbs é o lifesteal da Jóia da Absorção (_MSG_Attack.cpp:1651): num
// acerto que entra, um sorteio de 50% cura o atacante em AffHpAbs% do dano causado, limitado
// a 350/acerto. O original tem um bug no ramo else (ReqHp = RecHP sobrescreve
// em vez de somar); nós somamos e limitamos para a cura ficar coerente. No-op a menos que o
// atacante carregue o buff.
func (d *Dispatcher) applyHpAbs(w *world.World, s *world.Session, e *world.Entity, dmg int) {
	if e.AffHpAbs == 0 || dmg < 1 || w.Rand().Intn(2) != 0 {
		return
	}
	rec := hpAbsHeal(dmg, e.AffHpAbs)
	if rec <= 0 {
		return
	}
	e.HP += rec
	s.ReqHp = e.HP
	setReqHp(s, e) // limita HP/ReqHp a effectiveMaxHP
	d.sendSetHpMp(w, s, e)
}

// hpAbsHeal é a quantidade de cura da Jóia da Absorção: AffHpAbs% do dano causado,
// limitado a 350 (_MSG_Attack.cpp:1653).
func hpAbsHeal(dmg int, absPct int32) int32 {
	rec := (int32(dmg)*absPct + 1) / 100
	if rec > 350 {
		rec = 350
	}
	if rec < 0 {
		rec = 0
	}
	return rec
}

func manaControlDamage(target *world.Entity, dmg int, enhanced bool) (int, int32, bool) {
	if dmg <= 0 || target == nil || !target.HasAffect(18) || target.MP <= effectiveMaxMP(target)/10 {
		return dmg, 0, false
	}
	spent := int32(dmg)
	target.MP -= spent
	if target.MP < 0 {
		target.MP = 0
	}
	divisor := int32(55)
	if enhanced {
		divisor = 50
	}
	reduced := ((spent >> 1) + (spent << 4)) / divisor
	if reduced < 0 {
		return 0, spent, true
	}
	return int(reduced), spent, true
}

// itemRawSanc devolve o cValue PACKED de sanc de um item, não seu nível de refino.
//
// O divisor de redução de cura da fada abaixo é o único lugar que quer o número
// cru: o legado lê Equip[13].stEffect[0].cValue direto da struct
// (Server.cpp:10068, :10077) em vez de passar por BASE_GetItemSanc, então o
// contador de piedade alimenta o divisor de propósito. Usar refine.Level em qualquer lugar onde
// um nível de refino real é pretendido.
func itemRawSanc(it world.Item) int {
	for _, ef := range it.Effects {
		if ef.Effect >= 116 && ef.Effect <= 125 {
			return int(ef.Value)
		}
		if ef.Effect == efSanc {
			return int(ef.Value)
		}
	}
	return 0
}

func (d *Dispatcher) foemaHealAmount(target *world.Entity, heal int32) int32 {
	switch target.Equip[fairyEquipSlot].Index {
	case 786:
		sanc := itemRawSanc(target.Equip[fairyEquipSlot])
		if sanc < 2 {
			sanc = 2
		}
		return heal / int32(sanc)
	case 1936:
		sanc := itemRawSanc(target.Equip[fairyEquipSlot])
		if sanc < 2 {
			sanc = 2
		}
		return heal / int32(sanc*100)
	case 1937:
		sanc := itemRawSanc(target.Equip[fairyEquipSlot])
		if sanc < 2 {
			sanc = 2
		}
		return heal / int32(sanc*20000)
	default:
		return heal
	}
}

func healExpGain(caster, target *world.Entity, before int32) int64 {
	if caster == nil || target == nil || caster.ID == target.ID || !world.IsPlayer(caster.ID) {
		return 0
	}
	if world.Village(target.X, target.Y) >= 0 {
		return 0
	}
	gain := (target.HP - before) >> 3
	if gain < 0 {
		return 0
	}
	if gain > 120 {
		gain = 120
	}
	return int64(gain)
}

func (d *Dispatcher) applyBookResurrection(w *world.World, s *world.Session, e *world.Entity) {
	if e.HP != 0 {
		return
	}
	rev := w.Rand().Intn(115)
	if rev > 100 {
		rev -= 15
	}
	if rev >= 40 {
		e.HP = 2
		s.CrackError = 0
		s.ReqHp = e.HP
		setReqHp(s, e)
		d.sendScore(w, s, e)
		d.sendSetHpMp(w, s, e)
		d.recall(w, s, e)
		d.sendEtc(w, s, e)
	}
	hp := int32((w.Rand().Intn(50) + 1) * int((effectiveMaxHP(e)+1)/100))
	mp := int32((w.Rand().Intn(50) + 1) * int((effectiveMaxMP(e)+1)/100))
	e.HP, e.MP = hp, mp
	s.CrackError = 0
	s.ReqHp, s.ReqMp = e.HP, e.MP
	d.sendScore(w, s, e)
	d.sendSetHpMp(w, s, e)
	d.sendEtc(w, s, e)
	body := protocol.EncodeCreateMobBody(d.createMobFrom(e, 0))
	w.ForEachInView(s.Conn, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene}, body)
	})
}

func (d *Dispatcher) applyEtherealFlame(w *world.World, caster, target *world.Entity, tid int) {
	ts := w.Session(tid)
	if ts == nil {
		return
	}
	chance := (int(caster.BaseSpecial[1]) + 1) / 7
	if w.Rand().Intn(100) > chance {
		burn := ((target.MP + 1) / 100) * int32(10+w.Rand().Intn(10))
		target.MP -= burn
		if target.MP < 0 {
			target.MP = 0
		}
		ts.ReqMp = target.MP
		d.sendSetHpMp(w, ts, target)
		d.sendScore(w, ts, target)
		return
	}
	changed := false
	for i := range target.Affect {
		switch target.Affect[i].Type {
		case 14, 16, 18, 19:
			target.Affect[i] = world.Affect{}
			changed = true
		}
	}
	if changed {
		d.refreshEquip(w, ts, target)
		d.sendScore(w, ts, target)
		d.sendAffect(w, ts, target)
	}
}

func (d *Dispatcher) parryRate(attacker, target *world.Entity) int {
	attackDex := int(effectiveDex(attacker)) / 5
	if attacker.LearnedSkill&0x1000000 != 0 {
		attackDex += 100
	}
	if attacker.Rsv&world.RsvCast != 0 {
		attackDex += 500
	}
	return combat.ParryRate(int(effectiveDex(target)), target.Parry, attackDex, int(attacker.Rsv))
}

func applyHuntressForceDamage(attacker, target *world.Entity, tid, dmg int) int {
	if attacker == nil || target == nil || dmg <= 0 {
		return dmg
	}
	if !world.IsPlayer(tid) && attacker.AffForceMobDamage != 0 {
		dmg += int(attacker.AffForceMobDamage)
	}
	if attacker.AffForceDamage == 0 {
		return dmg
	}
	if world.IsPlayer(tid) || target.Clan == 4 {
		dmg >>= 2
	}
	if dmg <= 1 {
		return int(attacker.AffForceDamage)
	}
	return dmg + int(attacker.AffForceDamage)
}

func (d *Dispatcher) applyAirBladeProc(w *world.World, attacker, target *world.Entity, msgType protocol.Type, body *protocol.MsgAttackBody, payload []byte, dmg int) int {
	if dmg <= 0 || attacker == nil || target == nil || msgType != protocol.MsgAttackTwo ||
		attacker.Class != 3 || attacker.LearnedSkill&(1<<21) == 0 || w.Rand().Intn(4) != 0 {
		return dmg
	}
	skillDam := effectiveSpecial(attacker, 3) + int(effectiveStr(attacker))
	skillDam = combat.Damage(w.Rand(), skillDam, int(effectiveAC(target)), attacker.Master)
	if skillDam > 0 {
		skillDam /= 2
	}
	if skillDam < 60 {
		skillDam = 60
	}
	body.DoubleCritical |= 4
	writeDoubleCritical(payload, body.DoubleCritical)
	return dmg + skillDam
}

func (d *Dispatcher) applyOnHitAffects(w *world.World, attacker, target *world.Entity, tid int) {
	if attacker == nil || target == nil {
		return
	}
	if attacker.Rsv&world.RsvFrost != 0 && w.Rand().Intn(2) == 0 {
		d.applyOnHitSpell(w, target, tid, 36, effectiveSpecial(attacker, 1)+150, effectiveSpecial(attacker, 1))
	}
	if attacker.Rsv&world.RsvDrain != 0 && w.Rand().Intn(2) == 0 {
		d.applyOnHitSpell(w, target, tid, 40, effectiveSpecial(attacker, 1)+150, effectiveSpecial(attacker, 1))
	}
}

func (d *Dispatcher) applyOnHitSpell(w *world.World, target *world.Entity, tid, skillnum, delay, level int) {
	sp, ok := onHitSpell(d.spells, skillnum)
	if !ok {
		return
	}
	applied := target.SetAffect(sp.AffectType, sp.AffectValue, sp.AffectTime, sp.Aggressive, delay, level, d.affectDur)
	if target.SetTick(sp.TickType, sp.TickValue, sp.AffectTime, sp.Aggressive, delay, level, d.affectDur) {
		applied = true
	}
	if !applied {
		return
	}
	d.refreshScore(target)
	if ts := w.Session(tid); ts != nil {
		d.sendScore(w, ts, target)
		d.sendAffect(w, ts, target)
	}
}

func onHitSpell(spells *content.SkillData, skillnum int) (content.Spell, bool) {
	if spells != nil {
		if sp, ok := spells.Get(skillnum); ok {
			return sp, true
		}
	}
	switch skillnum {
	case 36:
		return content.Spell{Index: 36, AffectType: 1, AffectValue: 2, AffectTime: 1, Aggressive: 1}, true
	case 40:
		return content.Spell{Index: 40, TickType: 20, TickValue: 10, AffectTime: 1, Aggressive: 1}, true
	default:
		return content.Spell{}, false
	}
}

// writeDamage sobrescreve o dano autoritativo do servidor de Dam[i] no
// payload do pacote (o valor do cliente é ignorado).
func writeDamage(payload []byte, i int, dmg int32) {
	off := protocol.MsgAttackDamOffset + i*protocol.MsgAttackDamStride + 4
	if off+4 <= len(payload) {
		binary.LittleEndian.PutUint32(payload[off:off+4], uint32(dmg))
	}
}

func writeAttackProgress(payload []byte, progress uint16) {
	if len(payload) >= protocol.MsgAttackDamOffset {
		binary.LittleEndian.PutUint16(payload[32:34], progress)
	}
}

func writeDoubleCritical(payload []byte, doubleCritical uint8) {
	if len(payload) >= protocol.MsgAttackDamOffset {
		payload[36] = doubleCritical
	}
}

// writeAttackerStatus sobrescreve CurrentHp@4, CurrentExp@12,
// CurrentMp@40 e ReqMp@46 do atacante no corpo de MSG_Attack com os valores
// autoritativos do servidor (layout MsgAttackBody, messages.go). Esses campos fixos
// ficam abaixo da região Dam[] (offset 48), então nunca colidem com o dano
// por alvo.
func writeAttackerStatus(payload []byte, hp, mp int32, exp int64, reqMp int16) {
	if len(payload) < protocol.MsgAttackDamOffset {
		return
	}
	binary.LittleEndian.PutUint32(payload[4:8], uint32(hp))
	binary.LittleEndian.PutUint64(payload[12:20], uint64(exp))
	binary.LittleEndian.PutUint32(payload[40:44], uint32(mp))
	binary.LittleEndian.PutUint16(payload[46:48], uint16(reqMp))
}
