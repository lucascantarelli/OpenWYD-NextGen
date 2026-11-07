package handler

import (
	"encoding/binary"
	"fmt"

	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/loot"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// coinCap é o guarda de estouro do ouro total (game-rules.md §7, MobKilled.cpp:2715).
const coinCap = 2_000_000_000

// Efeito de level-up (captura-wyd-levelup.md §7): MSG_Motion com estes valores é a
// animação/som de subir de nível no cliente.
const (
	motionLevelUp     = 14
	motionLevelUpParm = 3
)

// babyMountLevelCap é o teto de crescimento da montaria (MobKilled.cpp:296).
const (
	babyMountGrowthHi = 2390
	babyMountLevelCap = 100
)

// mobKilled executa as recompensas de morte de um mob abatido por um jogador ou
// por uma invocação dele (game-rules.md §1-2, MobKilled.cpp). Esta leva implementa
// as rolagens de drop exatas (ouro §2.1 e item por slot §2.2 com a tabela real
// g_pDropRate). O Carry do mob é a tabela de loot dele.
//
// UNVERIFIED / adiado: distribuição de EXP em party no campo genérico e a
// confirmação de morte _MSG_CNFMobKill. Regiões de Pergaminho d'Água usam a
// distribuição legada própria logo abaixo.
func (d *Dispatcher) mobKilled(w *world.World, killer, mob *world.Entity) {
	// Registra o evento mesmo se o dono da invocação desconectou antes do abate.
	// Recompõe os auxiliares depois de DespawnMob liberar o slot de população.
	defer d.kefraMobKilled(w, mob)
	d.kingdomKingKilled(w, mob)
	reward := killer
	if killer.Summoner != 0 {
		d.growBabyMountOnKill(w, killer, mob)
		owner := w.Entity(killer.Summoner)
		mode, ok := w.SessionMode(killer.Summoner)
		if owner == nil || !world.IsPlayer(owner.ID) || !ok || mode != world.UserPlay {
			sendDieAction(w, mob)
			w.DespawnMob(mob.ID, 1)
			d.nightmareMobKilled(w, mob)
			return
		}
		reward = owner
	}
	if d.towerKilled(w, reward, mob) {
		return
	}
	d.castleBossKilled(w, reward, mob)
	d.waterMobKilled(w, reward, mob)
	d.runeQuestMobKilled(w, reward, mob)
	// O alvo da recompensa é jogador, então o id de entidade é o slot de conexão;
	// a sessão é necessária para os pacotes de ouro/level-up (nil se desconectou).
	ks := w.Session(reward.ID)
	// O script de morte legado do Nightmare sorteia o DieSay e regenera ANTES de
	// rolar o loot (MobKilled.cpp:1430,1882,2693). O mob que morre continua contado
	// até o despawn final: o limite de população do GenerateMob depende disso.
	nightmareDeath := world.NightmareGenerator(int(mob.GenIndex)) >= 0
	if nightmareDeath {
		sendDieAction(w, mob)
		d.nightmareMobKilled(w, mob)
	}

	// Drop de ouro → moeda do alvo (limitada ao teto). O novo total vai para o
	// cliente do alvo (MSG_UpdateEtc); senão o ganho só apareceria no relog.
	if gold := loot.GoldDrop(w.Rand(), int(mob.Level), int(mob.Coin)); gold > 0 {
		reward.Coin += int32(gold)
		if reward.Coin > coinCap {
			reward.Coin = coinCap
		}
		if ks != nil {
			d.sendEtc(w, ks, reward)
		}
	}

	// Experiência → premiado (solo). O total bruto chega ao cliente pelo eco
	// MSG_Attack do handler de ataque (CurrentExp); grantExp também aplica os
	// level-ups. Mob de clan 4 nunca dá EXP: o legado embrulha toda a distribuição
	// em `MOB.Clan != 4` (MobKilled.cpp:402); ouro e drops ficam fora desse gate.
	if mob.Clan != 4 && !d.grantNightmareExp(w, reward, mob) && !d.grantWaterExp(w, reward, mob) {
		d.grantExp(w, ks, reward, mob)
	}

	d.tryWorldEventDrop(w, reward)

	// Drop de item: cada slot de loot ocupado rola contra a chance de g_pDropRate.
	for slot := range mob.Carry {
		it := mob.Carry[slot]
		// MobKilled.cpp só admite aqui entradas reais do catálogo de drop. Índices
		// baixos são objetos internos/legados e o 454 é suprimido de propósito.
		if it.Index <= 390 || int(it.Index) >= maxItemList || it.Index == 454 {
			continue
		}
		// UNVERIFIED: killer.DropBonus (item/event bonus) → 0 placeholder.
		rate := loot.EffectiveDropRate(slot, 0, int(mob.Level))
		if loot.Drops(w.Rand(), rate) {
			if d.castleKeyDrop(w, reward, it) {
				continue
			}
			// O loot comum de mob legado passa por PutItem: vai direto para o Carry
			// de quem matou e é sincronizado com SendItem. Não é objeto de chão e
			// portanto não pode usar CreateGroundItem.
			d.putMobDrop(w, reward, it)
		}
	}
	d.terraMisticaKill(w, ks, reward, mob)

	gen := int(mob.GenIndex)
	if w.Entity(mob.ID) != mob {
		// Um hook de trilha de runa já o apagou (limpeza do Lich), e o slot pode
		// ter um spawn novo: mantém só o sorteio do DieSay.
		w.Rand().Intn(4)
	} else {
		if !nightmareDeath {
			sendDieAction(w, mob)
		}

		// Despawn: avisa os clientes na visão que o mob morreu (RemoveMob, tipo 1 =
		// morte) e libera a célula da grade + o slot de entidade, para o corpo sumir
		// e não poder ser re-alvejado. Sem isso o cliente segue desenhando o morto.
		w.DespawnMob(mob.ID, 1)
	}
	d.runeQuestMobRemoved(w, reward, gen)
}

// putMobDrop espelha o PutItem legado para loot comum de mob. Um Carry acessível
// cheio perde o item rolado e recebe o mesmo aviso de falta de espaço; não há
// fallback para o chão.
func (d *Dispatcher) putMobDrop(w *world.World, reward *world.Entity, it world.Item) bool {
	if reward == nil {
		return false
	}
	slot := firstEmptyAccessibleCarry(reward)
	if slot < 0 {
		if s := w.Session(reward.ID); s != nil {
			d.notify(w, s, NoticeNoSpaceToTrade)
		}
		return false
	}
	reward.Carry[slot] = it
	if s := w.Session(reward.ID); s != nil {
		d.sendSlot(w, s, world.ItemPlaceCarry, slot, it)
	}
	return true
}

func (d *Dispatcher) growBabyMountOnKill(w *world.World, killer, mob *world.Entity) {
	if !babyMountKillEligible(killer, mob) {
		return
	}
	owner := w.Entity(killer.Summoner)
	if owner == nil {
		return
	}
	if m, ok := w.SessionMode(owner.ID); !ok || m != world.UserPlay {
		return
	}
	mount := &owner.Equip[mountEquipSlot]
	changed, leveled := applyBabyMountKill(mount, int(mob.Level))
	if !changed {
		return
	}
	if s := w.Session(owner.ID); s != nil {
		if leveled {
			d.notify(w, s, NoticeMountLevel)
		}
		d.sendSlot(w, s, world.ItemPlaceEquip, mountEquipSlot, *mount)
	}
}

func babyMountKillEligible(killer, mob *world.Entity) bool {
	return killer != nil &&
		mob != nil &&
		killer.ID >= world.MaxUser &&
		killer.Clan == summonClan &&
		killer.Summoner > 0 &&
		killer.Summoner < world.MaxUser &&
		killer.EquipVisual[0] >= babyMountFaceLo &&
		killer.EquipVisual[0] <= babyMountFaceHi &&
		!world.IsPlayer(mob.ID) &&
		mob.Clan != summonClan
}

func applyBabyMountKill(mount *world.Item, mobLevel int) (changed, leveled bool) {
	if mount == nil || mount.Index < babyMountLo || mount.Index >= babyMountGrowthHi {
		return false, false
	}
	xp := mount.Effects[1].Effect
	if int(xp) >= mobLevel || xp >= babyMountLevelCap {
		return false, false
	}
	growth := mount.Effects[2].Value + 1
	if growth < babyMountGrowthThreshold(*mount) {
		mount.Effects[2].Value = growth
		return true, false
	}
	mount.Effects[2].Value = 1
	mount.Effects[1].Effect = xp + 1
	return true, true
}

func babyMountGrowthThreshold(mount world.Item) uint8 {
	xp := mount.Effects[1].Effect
	switch mount.Index {
	case 2330:
		return xp + 25
	case 2331:
		return xp + 35
	case 2332:
		return xp + 45
	case 2333:
		return xp + 55
	case 2334:
		return xp + 65
	case 2335:
		return xp + 75
	default:
		return xp + 100
	}
}

func sendDieAction(w *world.World, mob *world.Entity) {
	if mob == nil {
		return
	}
	say := w.Rand().Intn(4) // MobKilled.cpp sorteia o DieSay depois das recompensas, antes do DeleteMob.
	gen := w.GeneratorAt(int(mob.GenIndex))
	if gen == nil || mob.Leader != 0 || gen.DieAction[say] == "" {
		return
	}
	sendMobChat(w, mob.ID, gen.DieAction[say])
}

// grantExp dá experiência de PvE solo a quem matou e aplica os level-ups
// resultantes (captura-wyd-levelup.md, CMob::CheckGetLevel). O ganho é escalado
// pelo GetExpApply na razão de nível atacante↔alvo; o total é limitado ao teto da
// curva. Cada nível sobe MaxHp/MaxMp pelo incremento da classe, recompõe HP/MP e
// recalcula os pontos livres de atributo (BASE_GetBonusScorePoint — idempotente a
// partir de nível+stats, então não precisa ser persistido). Ao subir de nível, o
// cliente de quem matou recebe um score novo e o efeito de level-up, que também
// aparece para os jogadores na visão.
//
// UNVERIFIED / adiado: distribuição em party e os itens de recompensa por nível
// (DoItemLevel).
func (d *Dispatcher) grantExp(w *world.World, ks *world.Session, killer, mob *world.Entity) {
	if archExpLocked(killer) {
		return
	}
	var gain int64
	if d.trainingFieldBonus(w, killer, mob) {
		gain = level.TrainingFieldExpReward(mob.Exp, killer.Level, mob.Level, killer.ClassMaster, d.expBonus(killer), d.fieldExp)
	} else {
		gain = level.SoloExpReward(mob.Exp, killer.Level, mob.Level, killer.ClassMaster, d.expBonus(killer), d.expEvents)
	}
	d.applyMobExp(w, ks, killer, gain)
}

// trainingFieldTopLevel é o primeiro nível armazenado que o campo de treino
// recusa (_MSG_Action.cpp:215 bloqueia Level >= 35, ou seja 36+ na tela); o bônus
// do campo para na mesma borda.
const trainingFieldTopLevel = 35

// trainingFieldBonus diz se um abate ganha o multiplicador de EXP do campo de
// treino (issue #383): o mob veio de um gerador numa célula 0x80 do AttributeMap
// e quem matou é um mortal que o campo ainda aceita.
func (d *Dispatcher) trainingFieldBonus(w *world.World, killer, mob *world.Entity) bool {
	if d.fieldExp <= 0 || killer.ClassMaster != classMasterMortal || killer.Level >= trainingFieldTopLevel {
		return false
	}
	gen := w.GeneratorAt(int(mob.GenIndex))
	return gen != nil && gen.TrainingField
}

// applyMobExp aplica uma recompensa de PvE já calculada, sem recalcular bônus.
func (d *Dispatcher) applyMobExp(w *world.World, ks *world.Session, killer *world.Entity, gain int64) {
	if archExpLocked(killer) {
		return
	}
	if gain <= 0 {
		return
	}
	previousExp := killer.Exp
	killer.Exp += gain
	if killer.Exp > level.MaxExp {
		killer.Exp = level.MaxExp
	}
	if applied := killer.Exp - previousExp; applied > 0 && ks != nil {
		// SendMsgExp legado (MobKilled.cpp:2606, TNColor::Default):
		// MSG_Exp_Msg_Panel_ (0x5000) pinta o ganho sobre o personagem. Antes saía
		// como MsgMessageChat, que o cliente desenha como o jogador falando a linha
		// (QA 06/11).
		d.sendMsgExp(w, ks, fmt.Sprintf(expGainFormat, applied), expPanelColor)
	}

	// Um ataque de jogador carrega a EXP pós-abate no eco autoritativo do ataque.
	// Abates por invocação não geram esse eco do dono, então o estado etc completo
	// é reenviado aqui quando a recompensa não cruzou fronteira de nível. O
	// tratamento de level-up já manda MSG_UpdateEtc com os campos novos do score.
	if applied := killer.Exp - previousExp; applied > 0 {
		if !d.applyLevelUps(w, ks, killer) && ks != nil {
			d.sendEtc(w, ks, killer)
		}
	}
}

// grantDirectExp concede uma quantia fixa, já calculada. Recompensa de quest não
// pode passar pelo escalonamento de nível do monstro, bônus de equipamento nem
// eventos de EXP.
func (d *Dispatcher) grantDirectExp(w *world.World, s *world.Session, e *world.Entity, gain int64) int64 {
	if archExpLocked(e) {
		return 0
	}
	if gain <= 0 {
		return 0
	}
	previous := e.Exp
	e.Exp += gain
	if e.Exp > level.MaxExp {
		e.Exp = level.MaxExp
	}
	applied := e.Exp - previous
	if applied <= 0 || s == nil {
		return applied
	}
	// SendClientMessage legado (_MSG_UseItem.cpp:2386,2416,2464), ou seja, painel de
	// mensagem com ID 0 — NÃO MsgMessageChat, que o cliente desenha como o
	// personagem falando (QA 06/11).
	d.sendClientMessage(w, s, fmt.Sprintf(expGainFormat, applied))
	if !d.applyLevelUps(w, s, e) {
		d.sendEtc(w, s, e)
	}
	motion := protocol.EncodeMotion(motionLevelUp, motionLevelUpParm)
	w.Send(s, protocol.MsgMotion, motion)
	w.BroadcastInView(e.ID, protocol.MsgMotion, motion)
	return applied
}

// archExpLocked usa >= para que personagens que passaram por cima de um lock antes
// da issue #327 não continuem progredindo até o Lindy completar os destravamentos
// que faltam.
func archExpLocked(e *world.Entity) bool {
	return e.ClassMaster == classMasterArch && ((e.Level >= 354 && e.ArchLv355 == 0) || (e.Level >= 369 && e.ArchLv370 == 0))
}

// isCelestialTier diz se o tier anda na curva Celestial (g_pNextLevel_2) e no teto
// (MAX_CLEVEL): CELESTIAL/CELESTIALCS/SCELESTIAL (CMob.cpp:1085).
func isCelestialTier(classMaster uint8) bool {
	return classMaster == classMasterCelestial ||
		classMaster == classMasterCelestialCS ||
		classMaster == classMasterSCelestial
}

// applyLevelUps porta o CMob::CheckGetLevel depois de a Exp já ter sido elevada ao
// total desejado. É ciente do tier: Mortal/Arch andam em g_pNextLevel até MAX_LEVEL
// e ganham pontos de skill/especial por nível; os tiers Celestial andam em
// g_pNextLevel_2 até MAX_CLEVEL, ganham só AC + pontos de atributo e ficam travados
// nos níveis 40/90 até /destravar40 e /destravar90 marcarem as flags
// (CMob.cpp:1107, 1121-1151). É compartilhado por EXP de abate, EXP de quest, Ehre
// e o reset Celestial; cada nível alcançado também roda SetCircletSubGod, como o
// legado faz depois de CheckGetLevel devolver 4 (SendFunc.cpp:1039,
// _MSG_UseItem.cpp:2393/2431/2471/3599, _MSG_CombineItemEhre.cpp:165).
func (d *Dispatcher) applyLevelUps(w *world.World, s *world.Session, e *world.Entity) bool {
	return d.levelUps(w, s, e, true)
}

// applyLevelUpsWithoutCirclet é o applyLevelUps para as checagens de nível legadas
// que nunca chamam SetCircletSubGod: Poeira de Fada (_MSG_UseItem.cpp:1019/1036),
// a checagem do próprio atacante depois de uma cura (_MSG_Attack.cpp:1764) e o
// comando de EXP de GM (imple.cpp:159). Um Sub-Celestial que chega a 120/150/180/199
// por ali mantém a flag AddNNN em 0 e, como o add só dispara naquele nível exato,
// nunca recebe aquele add de diadema (issue #409).
func (d *Dispatcher) applyLevelUpsWithoutCirclet(w *world.World, s *world.Session, e *world.Entity) bool {
	return d.levelUps(w, s, e, false)
}

func (d *Dispatcher) levelUps(w *world.World, s *world.Session, e *world.Entity, circlet bool) bool {
	gained := int32(0) // níveis realmente cruzados — a concessão de Ponto de Caos abaixo é por nível
	celestial := isCelestialTier(e.ClassMaster)
	levelCap := level.MaxLevelForTier(e.ClassMaster)
	for e.Level < levelCap && e.Exp >= level.NextLevelExpTier(e.Level, e.ClassMaster) {
		if archExpLocked(e) {
			break
		}
		// Gates de quest Celestial: os tetos 40/90 continuam travados até
		// /destravar40 e /destravar90 marcarem as flags. No gate o CheckGetLevel
		// devolve 0 sem subir de nível (CMob.cpp:1107), então o laço para aqui.
		if celestial && ((e.Level == 39 && e.CelLv40 == 0) || (e.Level == 89 && e.CelLv90 == 0)) {
			break
		}
		e.Level++
		// Os incrementos de HP/MP pertencem ao BaseScore (CMob.cpp:1116) — escrever
		// só o MaxHP vivo seria desfeito no próximo refreshScore (= base + equip).
		e.BaseMaxHP = addClamp(e.BaseMaxHP, level.IncHP(e.Class), level.MaxHPCap)
		e.BaseMaxMP = addClamp(e.BaseMaxMP, level.IncMP(e.Class), level.MaxMPCap)
		// +1 de AC por nível para todo tier (CMob.cpp:1133,1145,1150). Isso mantém o
		// valor vivo certo pelo resto da sessão; playerBaseAC reconstrói o mesmo
		// total a partir do nível (agora maior) no próximo login.
		e.BaseAC++
		// SkillBonus +3/nível (+4 a partir de 200) e SpecialBonus +2/nível são
		// concessões só de Mortal/Arch; os tiers Celestial dão AC + pontos de
		// atributo e nada mais (CMob.cpp:1121-1151).
		if !celestial {
			if e.Level >= 200 {
				e.SkillBonus += 4
			} else {
				e.SkillBonus += 3
			}
			e.SpecialBonus += 2
		}
		// O legado sobe um nível por CheckGetLevel e roda SetCircletSubGod depois de
		// cada um, então cada nível cruzado é checado, não só o último.
		if circlet {
			d.setCircletSubGod(w, s, e)
		}
		gained++
	}
	if gained == 0 {
		return false
	}
	// BASE_GetBonusScorePoint lê os atributos do BaseScore, sem equipamento: ele
	// deriva os "pontos já gastos" como (atributo − base da classe). Os e.Str/e.Int/…
	// vivos são CurrentScore (base + equipamento), então alimentá-los aqui conta a
	// mais o que o equipamento soma e leva a concessão a 0 — o bug de "nenhum ponto
	// ao subir de nível" para qualquer personagem com equipamento de atributo. Usa o
	// BaseScore alocado.
	e.ScoreBonus = scoreBonusFor(e)
	d.refreshScore(e)             // funde os ganhos de HP/MP da base no score vivo
	e.HP, e.MP = e.MaxHP, e.MaxMP // cura cheia ao subir de nível

	// Level-up visível: uma janela de score nova (atributos próprios) + o pacote etc
	// que carrega o novo ScoreBonus (pontos livres de atributo) — o UpdateScore NÃO
	// carrega isso, então sem SendEtc o cliente nunca mostra os pontos ganhos. Some
	// o brilho de level-up para quem matou e para todos que conseguem ver.
	motion := protocol.EncodeMotion(motionLevelUp, motionLevelUpParm)
	if s != nil {
		d.sendScore(w, s, e)
		d.sendEtc(w, s, e)
		// _NN_Level_Up (SendFunc.cpp:1040): só o brilho não era legível.
		d.sendClientMessage(w, s, levelUpText)
		w.Send(s, protocol.MsgMotion, motion)
	}
	w.BroadcastInView(e.ID, protocol.MsgMotion, motion)

	// Pontos de Caos devolvidos pelos níveis recém-ganhos (issue #279). De propósito
	// por último: pode emitir um CreateMob que recolore o nick, o que precisa cair
	// depois da atualização de score/etc acima, não no meio dela.
	d.grantLevelUpPKPoint(w, s, e, gained)
	return true
}

// addClamp devolve v+inc limitado a [0, limit], evitando estouro de int32.
func addClamp(v, inc, limit int32) int32 {
	sum := int64(v) + int64(inc)
	if sum > int64(limit) {
		return limit
	}
	if sum < 0 {
		return 0
	}
	return int32(sum)
}

// Terra Mística passo 2 (MobKilled.cpp:2909-2921): depois das rolagens de loot, um
// abate de mob com face 239/241 tem 1 chance em 20 de mover a flag do alvo de 1
// para 2. O legado curto-circuita o teste de face antes do rand(), então outros
// mobs não consomem número aleatório; a rolagem acontece qualquer que seja a flag.
const (
	terraMisticaMissionFlag = 1
	terraMisticaFaceA       = 239
	terraMisticaFaceB       = 241
	terraMisticaKillOdds    = 20

	// mobFaceOffset é STRUCT_MOB.Equip[0].sIndex no template de 816 bytes.
	mobFaceOffset = 140
)

// msgWatchingTownSuccess é o _NN_Watching_Town_Success (4) do Language.txt, em Windows-1252.
const msgWatchingTownSuccess = "Quest: Voc\xea aprova a prote\xe7\xe3o da cidade."

// mobFace devolve o Equip[0].sIndex do mob a partir do template. O EquipVisual
// carrega o overlay de refino, então o template cru é a fonte fiel.
func mobFace(mob *world.Entity) int {
	if len(mob.Template) < mobFaceOffset+2 {
		return 0
	}
	return int(binary.LittleEndian.Uint16(mob.Template[mobFaceOffset:]))
}

func (d *Dispatcher) terraMisticaKill(w *world.World, ks *world.Session, reward, mob *world.Entity) {
	if !world.IsPlayer(reward.ID) {
		return
	}
	if face := mobFace(mob); face != terraMisticaFaceA && face != terraMisticaFaceB {
		return
	}
	if w.Rand().Intn(terraMisticaKillOdds) != 0 || reward.TerraMistica != terraMisticaMissionFlag {
		return
	}
	reward.TerraMistica = terraMisticaRewardFlag
	if ks == nil {
		return
	}
	d.sendClientMessage(w, ks, msgWatchingTownSuccess)
	// Persiste fora do loop, como amuMistico e exploitLeader.
	w.SaveCharacterThen(ks, func(*world.World, *world.Session) {})
	d.log.Info("terra mistica mission", "conn", ks.Conn, "name", reward.Name)
}
