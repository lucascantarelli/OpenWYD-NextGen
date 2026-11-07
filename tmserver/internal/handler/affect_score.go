package handler

import (
	"openwyd-nextgen/tmserver/internal/world"
)

const maxLegacyDamage int32 = 1_000_000_000

// applyAffectScore recalcula as contribuições de affect no score vivo (o
// "Buff Loop" de BASE_GetCurrentScore — Source/Buff Loop.txt). Os resultados são
// CACHEADOS na entidade (campos Aff* + Rsv) e aplicados em TEMPO DE LEITURA pelos
// getters efetivos, nunca embutidos no CurrentScore cru — isso mantém o
// score persistido livre de buff para a derivação da base no login (base = carregado −
// equip) não contar um buff duas vezes.
//
// Tipos implementados: 1 (lentidão/velocidade de ataque/debuff de INT), 2 (haste/velocidade de corrida),
// 3 (debuff de resistência), 4 (pergaminho), 5/6 (DEX percentual), 7 (lâmina congelada
// velocidade de ataque/debuff de INT), 8 (Jóias PvP — resistência/conjuração/lifesteal/%HP/%AC/
// %Damage/%Magic/MP↔HP, um por bit de Level), 9/10 (buff/debuff de dano),
// 11/12/21/31 (AC), 13
// (Assalto: +15% dano, DAMAGEMULTI, −10% MaxHP), 14/24 (CON/HP —
// Possuído e Samaritano), 15 (Special, teto 400 na leitura), 16 (transformação BM → transform.go),
// 19/26/28/36 (flags de Rsv), 25 (resistências a fogo/gelo/trovão), 27 (Frost condicionado à arma),
// 29 (multiplicador de atributo Soul), 30/37 (ForceMobDamage/ForceDamage), 38/42
// (troca HP↔MP), 39 (Baú de XP → AffExpBonus). Adiados (precisam de sistemas que ainda não
// modelamos): 17/20/22 (periódicos — tratados pelo motor de tick), 34/35
// (Divino/Vigor — já em tempo de leitura via affectMul).
func applyAffectScore(e *world.Entity) {
	applyAffectScoreWithItemAbility(e, nil)
}

func (d *Dispatcher) applyAffectScore(e *world.Entity) {
	applyAffectScoreWithItemAbility(e, d.itemAbility)
}

func applyAffectScoreWithItemAbility(e *world.Entity, itemAbility func(world.Item, uint8) int) {
	e.Rsv = 0
	e.AffDamage, e.AffAC, e.AffMaxHP, e.AffMaxMP, e.AffRunSpeed, e.AffAttackSpeed, e.AffExpBonus = 0, 0, 0, 0, 0, 0, 0
	e.AffStr, e.AffInt, e.AffDex, e.AffCon, e.AffCritical = 0, 0, 0, 0, 0
	e.AffForceDamage, e.AffForceMobDamage = 0, 0
	e.AffHpAbs, e.AffMagic = 0, 0
	e.AffSpecial = [4]int16{}
	e.AffResist = [4]int16{}
	e.AffDamageMultiPct = 100
	if !e.HasAnyAffect() {
		return
	}
	for i := range e.Affect {
		af := e.Affect[i]
		if af.Type == 0 {
			continue
		}
		value := int32(af.Value)
		level := int32(af.Level)
		switch af.Type {
		case 1: // Toque Sagrado/Lentidão: Run -= Value; usuários de robe também perdem INT.
			e.AffRunSpeed -= value
			e.AffAttackSpeed -= 30
			if e.Equip[0].Index > 50 {
				e.AffInt -= 40
			}
		case 2: // Velocidade: Run += Value e RSV_HASTE.
			e.AffRunSpeed += value
			e.Rsv |= world.RsvHaste
		case 3: // debuff sagrado: as quatro resistências caem (pela metade sem robe)
			tval := value
			if e.Equip[0].Index < 50 {
				tval /= 2
			} else {
				tval -= 10
			}
			for k := range e.AffResist {
				e.AffResist[k] -= int16(tval)
			}
		case 4: // pergaminho de combate: +30 de dano (o multi ×4 e o +5 de magia adiados)
			e.AffDamage += 30
			e.AffMagic += 5
		case 5: // Fanatismo/Incapacitador: Dex *= (100-Value)%.
			base := int32(e.Dex + e.AffDex)
			e.AffDex += clampInt16(base*(100-value)/100 - base)
		case 6: // Linha Proteção Divina/Absoluta: Dex *= (100+Value)%.
			base := int32(e.Dex + e.AffDex)
			e.AffDex += clampInt16(base*(100+value)/100 - base)
		case 7: // Lâmina Congelada: Att -= Level/10+10; usuários de robe perdem INT.
			e.AffAttackSpeed -= level/10 + 10
			if e.Equip[0].Index > 50 {
				e.AffInt -= int16(level/10 + 20)
			}
		case 8: // Jóias PvP (Vol 242): cada bit de Level é o bônus de uma joia,
			// portado de BASE_GetCurrentScore (Basedef.cpp:4478-4548). Bits 0 e
			// 6 não têm efeito de score (bit 0 não é lido no original; o
			// Accuracy += 50 do bit 6 cai em CMob.Accuracy, campo que o combate legado
			// nunca lê — a paridade é só o ícone do buff). Os bônus percentuais usam
			// a forma quantizada legada (x/100)*pct e leem a base crua, então
			// joias empilhadas somam em vez de compor — alguns pontos a menos quando duas
			// joias percentuais estão ativas juntas, dentro da tolerância de buff.
			if level&(1<<1) != 0 { // Resistência: +25 nas quatro resistências
				for k := range e.AffResist {
					e.AffResist[k] += 25
				}
			}
			if level&(1<<2) != 0 { // Revelação: conjurar enquanto é atingido
				e.Rsv |= world.RsvCast
			}
			if level&(1<<3) != 0 { // Absorção: lifesteal no acerto (consumido no combate)
				e.AffHpAbs += 20
			}
			if level&(1<<4) != 0 { // Proteção: +10% MaxHp, +10% AC
				e.AffMaxHP += (e.MaxHP / 100) * 10
				e.AffAC += (e.AC / 100) * 10
			}
			if level&(1<<5) != 0 { // Poder: +10% MaxHp, +10% Damage, +20% Magic
				e.AffMaxHP += (e.MaxHP / 100) * 10
				e.AffDamage += (e.Damage / 100) * 10
				e.AffMagic += (int32(e.Magic) / 100) * 20
			}
			if level&(1<<7) != 0 { // Magia: metade do pool máximo de MP vira HP máximo
				mana := (e.MaxMP + 1) / 2
				e.AffMaxHP += mana
				e.AffMaxMP -= mana
			}
		case 9: // buff de dano; um Foema com o bit 19 aprendido o triplica
			add := (level*5/20 + value) * 3 / 2
			if e.Class == 1 && e.LearnedSkill&0x80000 != 0 {
				add *= 3
			}
			e.AffDamage += add
			e.AffMagic += 5
		case 10: // debuff de dano
			e.AffDamage -= level/5 + value
		case 11: // buff de AC
			e.AffAC += level/3 + value
		case 12: // debuff percentual de AC: AC *= (100-Value)%.
			base := e.AC + e.AffAC
			e.AffAC += base*(100-value)/100 - base
		case 13: // Assalto: +15% dano, +DAMAGEMULTI, −10% MaxHP.
			dmg := e.Damage + e.AffDamage
			boosted := dmg + (dmg/100)*15
			if boosted >= maxLegacyDamage {
				boosted = maxLegacyDamage
			}
			e.AffDamage += boosted - dmg
			e.AffDamageMultiPct += level/10 + value
			e.AffMaxHP += e.MaxHP*9/10 - e.MaxHP // MaxHp*9/10 do legado; M - M/10 é 1 acima disso (#394)
		case 14: // Possuído (skill 3): +CON e o dobro disso em MaxHP
			applyConHpBuff(e, level, value)
		case 15: // as quatro árvores Special (+teto 400 aplicado na leitura)
			v := int16(level/10 + value)
			for k := range e.AffSpecial {
				e.AffSpecial[k] += v
			}
		case affectTransform: // transformação em fera do BM (malha tratada por equipVisual)
			applyTransformScore(e, af)
		case 19:
			e.Rsv |= world.RsvBlock
		case 21: // maldição/Meditação: −AC e +DAMAGEMULTI.
			e.AffAC -= level/3 + 10
			e.AffDamageMultiPct += level/10 + value
		case affectSamaritano: // Samaritano: +CON e +2×valor de MaxHP
			// DIVERGÊNCIA DELIBERADA DO LEGADO (issue #267). Basedef.cpp:4225
			// ("else if(Type == 24) // Samaritano") e Buff Loop.txt:295 ambos fazem
			// disto um buff de AC, `Ac += Ac/4 + Value`, e SkillData.csv:14 dá à
			// skill Value 0 — então o efeito legado é um +25% de AC fixo. No jogo a
			// skill deve subir HP e Constituição, que é a forma do
			// affect-14, então aplicamos essa forma aqui com o Value próprio do Samaritano
			// (0) e maestria. Mesmo espírito do remapeamento de elemento do case 25 da
			// issue #233 e da política de duração das issues #92/#236.
			//
			// A guarda IsPlayer é estrutural, não defensiva: num MOB este mesmo
			// id de affect é o contador de vida da invocação guardado em Affect[0]
			// (summon.go:25). O legado desambigua os dois por índice
			// (Server.cpp:5843, `Type == 24 && idx >= MAX_USER`), e sem a
			// guarda uma invocação que algum dia chegue ao refreshScore pegaria um buff de jogador
			// do próprio timer de morte.
			if world.IsPlayer(e.ID) {
				applyConHpBuff(e, level, value)
			}
		case 25: // Proteção Elemental: os três elementos, nunca sagrado.
			add := int16((value + level/4) / 10)
			if level >= 255 {
				add += 20
			}
			// DIVERGÊNCIA DELIBERADA DO LEGADO (issue #233). Basedef.cpp:4239
			// soma em locais chamados Fogo/Trovao/Gelo, mas esses locais estão mal nomeados
			// (Basedef.cpp:3919 liga Sagrado←Resist[0], Trovao←[1], Fogo←[2],
			// Gelo←[3]), então os índices que de fato toca são 1/2/3 — deixando a
			// resistência real a fogo (Resist[0]) intocada e buffando o sagrado. A
			// ordem real dos elementos vem do conteúdo: Orb_de_Fogo carrega
			// EF_RESIST1 e Orb_Sagrada EF_RESIST3 (ItemList.csv:1013,1019), e
			// EF_RESIST1..4 mapeiam para Resist[0..3] (CMob.cpp:640-643) — então [0]=fogo,
			// [1]=gelo, [2]=sagrado, [3]=trovão. Buffamos os três elementos reais e
			// pulamos o sagrado, que não é elemental. Mesmo espírito da divergência do ÷8 affectTime
			// da issue #92 (content/skilldata.go).
			for _, k := range [3]int{0, 1, 3} {
				e.AffResist[k] += add
			}
		case 26:
			e.Rsv |= world.RsvParry
		case 27:
			if itemAbility != nil && itemAbility(e.Equip[weaponSlotR], efWType) == 101 {
				e.Rsv |= world.RsvFrost
			}
		case 28:
			e.Rsv |= world.RsvHide
		case 29:
			applySoulScore(e)
		case world.AffectForceMobDamage:
			e.AffForceMobDamage += level
		case 31: // Escudo Dourado: AC + Level/2 + Value
			e.AffAC += level/2 + value
		case 36:
			if itemAbility != nil && itemAbility(e.Equip[weaponSlotR], efWType) == 41 {
				e.Rsv |= world.RsvDrain
			}
		case 37:
			e.AffForceDamage += int32(e.Special[2])
		case 38: // Troca de Espíritos: metade do MP máximo vira HP máximo
			mana := e.MaxMP / 2
			e.AffMaxHP += mana
			e.AffMaxMP -= mana
		case world.AffectExpChest:
			e.AffExpBonus += 100
		case 42: // vínculo de alma: move um quinto do pool de mana para HP
			mana := e.MaxMP/5 + (e.Level+level)/2
			e.AffMaxHP += mana
			e.AffMaxMP -= mana
		}
	}
}

// applyConHpBuff é a forma de buff CON/MaxHP compartilhada pelo affect 14 (Possuído,
// Basedef.cpp:4053 / Buff Loop.txt:127) e pelo affect 24 (Samaritano, pela divergência
// da issue #267). MaxHP é somado EXPLICITAMENTE porque o refreshScore converte só
// CON de equipamento em HP antes do buff loop (Basedef.cpp:3152). CON concedida por
// affect precisa fornecer seu próprio delta de HP sem entrar de novo naquela conversão.
//
// Não portado de Basedef.cpp:4056: `if (ClassMaster != MORTAL && != ARCH) value
// *= 3`, junto com os multiplicadores *22 / *125% daquele arquivo — são tuning
// de servidor privado (os próprios comentários deles dizem "(106k)" e "(5004)") ausentes do Buff Loop.txt,
// que é a variante que este port segue. Veja docs/migration/skills/audit-affects.md.
func applyConHpBuff(e *world.Entity, level, value int32) {
	v := level*3/4 + value
	e.AffMaxHP += 2 * v
	e.AffCon += clampInt16(v)
}

const (
	soulF  = 2
	soulI  = 3
	soulD  = 4
	soulC  = 5
	soulFI = 6
	soulFD = 7
	soulFC = 8
	soulIF = 9
	soulID = 10
	soulIC = 11
	soulDF = 12
	soulDI = 13
	soulDC = 14
	soulCF = 15
	soulCI = 16
	soulCD = 17
)

func applySoulScore(e *world.Entity) {
	if e.Soul == 0 {
		return
	}
	single, primary, secondary := int32(180), int32(160), int32(120)
	if e.ClassMaster != classMasterMortal {
		single, primary, secondary = 220, 180, 140
	}
	addScaled := func(dst *int16, base int16, pct int32) {
		v := int32(base) * pct / 100
		*dst += clampInt16(v - int32(base))
	}
	switch e.Soul {
	case soulF:
		addScaled(&e.AffStr, e.Str+e.AffStr, single)
	case soulI:
		addScaled(&e.AffInt, e.Int+e.AffInt, single)
	case soulD:
		addScaled(&e.AffDex, e.Dex+e.AffDex, single)
	case soulC:
		addScaled(&e.AffCon, e.Con+e.AffCon, single)
	case soulFI:
		addScaled(&e.AffStr, e.Str+e.AffStr, primary)
		addScaled(&e.AffInt, e.Int+e.AffInt, secondary)
	case soulFD:
		addScaled(&e.AffStr, e.Str+e.AffStr, primary)
		addScaled(&e.AffDex, e.Dex+e.AffDex, secondary)
	case soulFC:
		addScaled(&e.AffStr, e.Str+e.AffStr, primary)
		addScaled(&e.AffCon, e.Con+e.AffCon, secondary)
	case soulIF:
		addScaled(&e.AffInt, e.Int+e.AffInt, primary)
		addScaled(&e.AffStr, e.Str+e.AffStr, secondary)
	case soulID:
		addScaled(&e.AffInt, e.Int+e.AffInt, primary)
		addScaled(&e.AffDex, e.Dex+e.AffDex, secondary)
	case soulIC:
		addScaled(&e.AffInt, e.Int+e.AffInt, primary)
		addScaled(&e.AffCon, e.Con+e.AffCon, secondary)
	case soulDF:
		addScaled(&e.AffDex, e.Dex+e.AffDex, primary)
		addScaled(&e.AffStr, e.Str+e.AffStr, secondary)
	case soulDI:
		addScaled(&e.AffDex, e.Dex+e.AffDex, primary)
		addScaled(&e.AffInt, e.Int+e.AffInt, secondary)
	case soulDC:
		addScaled(&e.AffDex, e.Dex+e.AffDex, primary)
		addScaled(&e.AffCon, e.Con+e.AffCon, secondary)
	case soulCF:
		addScaled(&e.AffCon, e.Con+e.AffCon, primary)
		addScaled(&e.AffStr, e.Str+e.AffStr, secondary)
	case soulCI:
		addScaled(&e.AffCon, e.Con+e.AffCon, primary)
		addScaled(&e.AffInt, e.Int+e.AffInt, secondary)
	case soulCD:
		addScaled(&e.AffCon, e.Con+e.AffCon, primary)
		addScaled(&e.AffDex, e.Dex+e.AffDex, secondary)
	}
}

func clampInt16(v int32) int16 {
	if v < -32768 {
		return -32768
	}
	if v > 32767 {
		return 32767
	}
	return int16(v)
}

// effectiveAC é a mitigação que o defensor de fato apresenta: o
// CurrentScore.Ac cru mais as contribuições de affect.
func effectiveAC(e *world.Entity) int32 { return e.AC + e.AffAC }

func effectiveStr(e *world.Entity) int16 { return e.Str + e.AffStr }

func effectiveInt(e *world.Entity) int16 { return e.Int + e.AffInt }

func effectiveDex(e *world.Entity) int16 { return e.Dex + e.AffDex }

// effectiveMagic é o poder de conjuração que o cliente/dano de skill veem: o Magic cru
// mais o buff de +20% da Jóia do Poder (affect 8 bit 5, cacheado em AffMagic).
func effectiveMagic(e *world.Entity) int32 { return int32(e.Magic) + e.AffMagic }

// effectiveResist é a resistência viva do elemento i (0 fogo, 1 gelo, 2 sagrado,
// 3 trovão). O legado limita cada resistência a 0..100 no FIM do
// BASE_GetCurrentScore, depois que o Buff Loop as moveu (Basedef.cpp:4737-4767)
// — clampResist sozinho só limita a parte do equipamento, então um buff somado sobre
// equipamento no teto escaparia do limite (e estouraria o int8 que o pacote
// carrega) enquanto um debuff de resistência levaria a mitigação abaixo do piso legado.
func effectiveResist(e *world.Entity, i int) int16 {
	return min(max(e.Resist[i]+e.AffResist[i], 0), resistCap)
}

func effectiveCritical(e *world.Entity) uint8 {
	v := int(e.Critical) + int(e.AffCritical) + int(skillCriticalBonus(e))
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// skillCriticalBonus é o bônus de crítico de skill de classe. O legado dá a MESMA fórmula a
// duas classes de dois bits de skill diferentes — TK "Confiança" (Basedef.cpp:3252, bônus em
// :3366-3371) e Huntress "Visão do Caçador" (:3856, bônus em :3858-3863) — então o gate
// é a única coisa que difere entre elas.
func skillCriticalBonus(e *world.Entity) int16 {
	tk := e.Class == 0 && e.LearnedSkill&(1<<7) != 0
	ht := e.Class == 3 && e.LearnedSkill&(1<<18) != 0
	if !tk && !ht {
		return 0
	}
	add := (int(e.Special[3])+1)/10 + int(effectiveDex(e))/75
	if add < 4 {
		add = 4
	}
	return int16(add)
}

// effectiveSpecial é a maestria viva de uma árvore: alocada + equipamento (o
// e.Special cru) + affects, limitada a 400 como o buff loop faz.
func effectiveSpecial(e *world.Entity, kind int) int {
	v := int(e.Special[kind]) + int(e.AffSpecial[kind])
	if v > 400 {
		v = 400
	}
	return v
}
