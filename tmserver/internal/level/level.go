// Package level guarda a curva de experiência e a matemática de level-up do WYD
// como funções puras (captura-wyd-levelup.md, a partir de CMob::CheckGetLevel /
// GetExpApply / BASE_GetHpMp / BASE_GetBonusScorePoint no fonte original). Como o
// package combat, é isolado de I/O para poder ter teste golden exato.
//
// Escopo: o caminho solo MORTAL (ClassMaster 2, Basedef.h:238) — o caso dominante. As tiers
// ARCH/CELESTIAL (outra curva, g_pNextLevel_2, gates de quest, meia EXP) e a
// distribuição de party ainda NÃO são modeladas aqui; o mundo não guarda estado de tier para elas.
package level

// MaxLevel é MAX_LEVEL (Basedef.h:177): um MORTAL (ou ARCH) nunca passa disso.
const MaxLevel int32 = 399

// MaxCLevel é MAX_CLEVEL (Basedef.h:178): o teto de nível na curva CELESTIAL
// (CELESTIAL/CELESTIALCS/SCELESTIAL). CheckGetLevel escolhe MaxCLevel em vez de MaxLevel
// para essas tiers (CMob.cpp:1085-1086).
const MaxCLevel int32 = 199

// MaxHPCap / MaxMPCap são MAX_HP / MAX_MP (Basedef.h:263-264).
const (
	MaxHPCap int32 = 1_000_000_000
	MaxMPCap int32 = 1_000_000_000
)

// MaxExp é o teto de experiência: g_pNextLevel[MAX_LEVEL+1] = g_pNextLevel[400]
// (o MobKilled.cpp limita a Exp acumulada a ele).
const MaxExp int64 = 4_100_000_000

// incHP / incMP são g_pIncrementHp / g_pIncrementMp (Basedef.cpp:63-64): os
// ganhos por nível, indexados por classe (0 TK, 1 FM, 2 BM, 3 HT).
var (
	incHP = [4]int32{3, 1, 1, 2}
	incMP = [4]int32{1, 3, 2, 1}
)

// baseSIDCHM é BaseSIDCHM[classe][Str,Int,Dex,Con,HP,MP] (Basedef.cpp:44): os
// atributos iniciais por classe e o HP/MP base, usados pelas fórmulas de score.
var baseSIDCHM = [4][6]int32{
	{8, 4, 7, 6, 80, 45},  // TK
	{5, 8, 5, 5, 60, 65},  // FM
	{6, 6, 9, 5, 70, 55},  // BM
	{8, 9, 13, 6, 75, 60}, // HT
}

// validClass protege as tabelas por classe contra classe fora da faixa (mobs ou
// dados corrompidos); quem chama recebe a semântica da classe 0 em vez de panic.
func validClass(cls uint8) int {
	if int(cls) >= len(baseSIDCHM) {
		return 0
	}
	return int(cls)
}

// BaseStats devolve BaseSIDCHM[cls]: Str, Int, Dex, Con, HP e MP de um
// personagem novo daquela classe.
func BaseStats(cls uint8) [6]int32 { return baseSIDCHM[validClass(cls)] }

// IncHP / IncMP são os incrementos de HP/MP por nível de uma classe.
func IncHP(cls uint8) int32 { return incHP[validClass(cls)] }
func IncMP(cls uint8) int32 { return incMP[validClass(cls)] }

// NextLevelExp devolve g_pNextLevel[curLevel+1]: a experiência total necessária
// para o próximo nível. Níveis no MaxLevel ou acima devolvem MaxExp (sem outro
// level-up). curLevel é limitado para a consulta nunca sair da faixa.
func NextLevelExp(curLevel int32) int64 {
	next := max(curLevel+1, 1)
	if int(next) >= len(nextLevel) {
		return MaxExp
	}
	return nextLevel[next]
}

// MaxLevelForTier é o teto de nível de uma tier: MaxCLevel para as tiers
// celestiais (que andam em g_pNextLevel_2), MaxLevel nos demais casos (CMob.cpp:1082-1086).
func MaxLevelForTier(classMaster uint8) int32 {
	if isCelestialTier(classMaster) {
		return MaxCLevel
	}
	return MaxLevel
}

// NextLevelExpTier devolve a experiência total necessária para chegar a curLevel+1 na
// curva da tier informada: g_pNextLevel_2 para tiers celestiais, g_pNextLevel
// (a curva Mortal/Arch) nos demais casos (CMob.cpp:1092-1093). No teto da tier
// ou acima dele devolve o topo da curva para nenhum outro level-up disparar.
func NextLevelExpTier(curLevel int32, classMaster uint8) int64 {
	if !isCelestialTier(classMaster) {
		return NextLevelExp(curLevel)
	}
	next := max(curLevel+1, 1)
	if next >= MaxCLevel+1 || int(next) >= len(nextLevel2) {
		return nextLevel2[len(nextLevel2)-1]
	}
	return nextLevel2[next]
}

// ForExpTier mapeia experiência total de volta para um nível na curva da tier.
// A receita de refino purificado do Ehre subtrai experiência e então faz esta
// inverse lookup exactly like CMob's legacy g_pNextLevel_2 scan.
func ForExpTier(exp int64, classMaster uint8) int32 {
	levelCap := MaxLevelForTier(classMaster)
	var result int32
	for current := int32(0); current < levelCap; current++ {
		if exp < NextLevelExpTier(current, classMaster) {
			break
		}
		result = current + 1
	}
	return result
}

// ExpApply é GetExpApply do caminho MORTAL (GetFunc.cpp:1028): escala a
// recompensa base do mob pela razão de nível atacante↔alvo. attacker é o nível do
// matador, target o do mob. Alvos de nível maior dão bônus (limitado a 200%);
// matador muito acima do mob (razão < 80% e nível ≥ 49) é penalizado.
func ExpApply(exp int64, attacker, target int32) int64 {
	if exp <= 0 {
		return 0
	}
	if target > MaxLevel+1 || attacker < 0 || target < 0 {
		return exp
	}
	a := int64(attacker) + 1
	t := int64(target) + 1
	mult := t * 100 / a
	switch {
	case mult < 80 && a >= 50:
		mult = mult*2 - 100
	case mult > 200:
		mult = 200
	}
	if mult < 0 {
		mult = 0
	}
	return (exp*mult + 1) / 100
}

// ScoreBonus é BASE_GetBonusScorePoint do MORTAL (Basedef.cpp:898): os pontos
// livres de atributo que o personagem deveria ter = pontos dados pelo nível menos
// os já gastos acima da base da classe. É idempotente (função do nível e dos
// atributos atuais), então pode ser recalculado a cada level-up sem
// ser persistido.
func ScoreBonus(cls uint8, level int32, str, intel, dex, con int16) int32 {
	c := validClass(cls)
	used := (int32(str) - baseSIDCHM[c][0]) +
		(int32(intel) - baseSIDCHM[c][1]) +
		(int32(dex) - baseSIDCHM[c][2]) +
		(int32(con) - baseSIDCHM[c][3])

	leveluse := level * 5
	if level >= 254 {
		leveluse += (level - 254) * 5
	}
	if level >= 299 {
		leveluse += (level - 299) * 10
	}
	if level >= 354 {
		leveluse += (level - 354) * -8
	}
	if bonus := leveluse - used; bonus > 0 {
		return bonus
	}
	return 0
}

// ScoreBonusCelestial é BASE_GetBonusScorePoint das tiers celestiais
// (Basedef.cpp:980-1172): 6 pontos por nível mais as concessões de quest (100 por
// estágio de cristal Arch, 200 por reset da Pedra da Furia, os 1001 fixos e o
// bônus de tier do ArchLevel), com pontos extras a partir do nível 120. Como ScoreBonus,
// trava um gasto a mais em 0 em vez de retomar o excesso do BaseScore.
// CELESTIALCS/SCELESTIAL também creditam a tier estacionada (Basedef.cpp:1084-1101 e
// :1149-1166): otherLvl é QuestInfo.Celestial.SubCelestialLevel para um
// CELESTIALCS e CelestialLevel para um SCELESTIAL, valendo 3 pontos por nível mais
// todos os bônus de 120+; passe 0 para um CELESTIAL simples (issue #410).
func ScoreBonusCelestial(cls uint8, lvl int32, str, intel, dex, con int16, cristal, reset, archLevel uint8, otherLvl int32) int32 {
	c := validClass(cls)
	used := (int32(str) - baseSIDCHM[c][0]) +
		(int32(intel) - baseSIDCHM[c][1]) +
		(int32(dex) - baseSIDCHM[c][2]) +
		(int32(con) - baseSIDCHM[c][3])

	leveluse := lvl*6 + int32(cristal)*100 + int32(reset)*200 + 1001
	switch archLevel {
	case 1:
		leveluse += 100
	case 2:
		leveluse += 300
	case 3:
		leveluse += 600
	case 4:
		leveluse += 900
	case 5:
		leveluse += 1200
	}
	if lvl >= 120 {
		leveluse += (lvl - 119) * 6
	}
	for _, from := range [...]int32{150, 170, 180, 190} {
		if lvl >= from {
			leveluse += (lvl - from + 1) * 2
		}
	}
	leveluse += otherLvl * 6 / 2
	if otherLvl >= 120 {
		leveluse += (otherLvl - 119) * 6
	}
	for _, from := range [...]int32{150, 170, 180, 190} {
		if otherLvl >= from {
			leveluse += (otherLvl - from + 1) * 2
		}
	}
	if bonus := leveluse - used; bonus > 0 {
		return bonus
	}
	return 0
}
