package level

// WaterTier seleciona o ramo de EXP específico da região em MobKilled.cpp:851-1270.
type WaterTier uint8

const (
	// WaterNormal é a região N (8,27) em coordenadas de mapa de 128 células.
	WaterNormal WaterTier = iota
	// WaterMystic é a região M (9,28).
	WaterMystic
	// WaterArcane é a região A (10,27).
	WaterArcane
)

// WaterExpReward  calcula a recompensa de EXP de um abate de mob na dungeon da água. É o mesmo que
// SoloExpReward, mas com divisores de tier e bônus de evento. O legado aplica a razão.
// isExp para cada membro, mas retém o teto e o bônus do matador. Nenhum divisor
// de tamanho de party sobrevive nestes três ramos. Hold/DayLog são sistemas separados.
func WaterExpReward(tier WaterTier, mobExp int64, memberLevel, mobLevel int32, classMaster uint8, killerCap int64, killerBonus int32, ev ExpEvents) int64 {
	if tier > WaterArcane || killerCap <= 0 {
		return 0
	}
	// GetExpApply tem ajustes por classe antes da razão comum de nível.
	// Arch ganha metade do EXP base; as tiers celestiais comparam em MAX_LEVEL.
	baseExp := mobExp
	applyLevel := memberLevel
	if classMaster == classArch {
		baseExp = int64(float64(baseExp) * 0.50)
	} else if isCelestialTier(classMaster) {
		applyLevel = MaxLevel
	}
	isExp := ExpApply(baseExp, applyLevel, mobLevel)
	if isExp <= 0 {
		return 0
	}
	myLevel := int64(memberLevel)
	if classMaster != classMortal && classMaster != classArch {
		myLevel += int64(MaxLevel) + 1
	}
	exp := 450 * isExp / (30 + myLevel)
	if exp <= 0 || exp > soloExpGate {
		return 0
	}
	exp = waterTierDivisors(tier, exp, myLevel, classMaster)
	exp = min(6*exp/10, killerCap)
	if killerBonus > 0 && killerBonus < 500 {
		exp += exp * int64(killerBonus) / 100
	}
	if ev.NewbieEvent && memberLevel < 100 && !isCelestialTier(classMaster) {
		exp += exp / 4
	}
	if ev.DoubleMode {
		exp *= 2
	}
	if !ev.KefraLive {
		exp /= 2
	}
	if ev.NewbieEvent {
		exp += exp * 15 / 100
	} else {
		exp -= exp * 15 / 100
	}
	return max(exp, 0)
}

// WaterExpKillerCap devolve o GetExpApply do matador antes do multiplicador
// da dungeon. É o teto eMob que o legado leva para cada ramo de party.
func WaterExpKillerCap(mobExp int64, killerLevel, mobLevel int32, classMaster uint8) int64 {
	if classMaster == classArch {
		mobExp = int64(float64(mobExp) * 0.50)
	} else if isCelestialTier(classMaster) {
		killerLevel = MaxLevel
	}
	return ExpApply(mobExp, killerLevel, mobLevel)
}

func waterTierDivisors(tier WaterTier, exp, lv int64, class uint8) int64 {
	if class != classMortal && class != classArch {
		return applyTierDivisors(exp, lv, class)
	}
	if class == classMortal {
		// O primeiro divisor do N é um literal double; todos os outros valores
		// fracionários abaixo são literais float. Preserve a conversão antes do truncamento.
		if lv <= 200 {
			if tier == WaterNormal {
				return int64(float64(exp) / 1.28)
			}
			return exp
		}
		bounds := [...]int64{300, 356, 370, 380, 390, 399}
		divisors := [3][6]float32{
			{1.03, 1.45, 1.80, 2.45, 3.70, 6.80},
			{1.08, 1.30, 1.80, 2.20, 2.60, 4.70},
			{0.10, 1, 1.55, 2.20, 2.60, 3.85},
		}
		for i, upper := range bounds {
			if lv <= upper {
				if divisors[tier][i] == 1 {
					return exp
				}
				return int64(float32(exp) / divisors[tier][i])
			}
		}
		return exp
	}
	// O ramo N de propósito não tem tabela de divisores ARCH.
	if tier == WaterNormal || lv <= 200 {
		return exp
	}
	for i, upper := range [...]int64{300, 356, 360, 370} {
		if lv <= upper {
			divisors := [2][4]float32{{0.90, 0.95, 4.50, 6.60}, {0.05, 0.85, 4, 6.60}}
			return int64(float32(exp) / divisors[tier-WaterMystic][i])
		}
	}
	for i, upper := range [...]int64{380, 390, 400} {
		if lv <= upper {
			divisors := [2][3]int64{{12, 19, 32}, {9, 15, 22}}
			return exp / divisors[tier-WaterMystic][i]
		}
	}
	return exp
}
