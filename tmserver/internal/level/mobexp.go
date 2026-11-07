package level

// MobExpForLevel é a recompensa de template balanceada (STRUCT_MOB.Exp) de um
// monstro do nível informado — a curva de projeto por trás do cmd/exptool, que
// regrava os templates de NPC do Release (issue #43: os dados entregues misturavam
// Exp=0 com valores absurdos, então abates não davam nada ou davam dezenas de níveis).
//
// O alvo é o ritmo clássico em abates-por-nível para um MORTAL solo
// matando mobs do próprio nível (killsPerLevel abaixo), invertido pelo
// pipeline de recompensa de campo geral (SoloExpReward) sob as flags padrão de evento:
// KefraLive=false (÷2) e sem evento de newbie (−15%), ou seja, ganho = 0.425·E·min(A,1)
// com A = 0.6·450/((30+L)·div). Rodar com -kefra-live dobra o ganho e
// portanto reduz pela metade os abates-por-nível — desvio aceitável de evento, como a
// legacy globals.
func MobExpForLevel(lvl int32) int64 {
	if lvl < 1 {
		lvl = 1
	}
	if lvl > MaxLevel {
		lvl = MaxLevel
	}
	// EXP total deste nível até o próximo (nextLevel é cumulativo).
	delta := float64(NextLevelExp(lvl) - NextLevelExp(lvl-1))
	a := 0.6 * 450 / (float64(30+lvl) * mortalDivisor(lvl))
	if a > 1 {
		a = 1 // the eMob cap (SoloExpReward) binds below ~level 240
	}
	e := delta / (killsPerLevel(lvl) * 0.425 * a)
	if e < 1 {
		return 1
	}
	return int64(e)
}

// killsPerLevel é o alvo de ritmo: abates de mesmo nível necessários para ganhar
// um nível. Sensação clássica do WYD — rápido no Campo de Treino (≤35), desacelerando
// no meio do jogo e com grind no teto. Linear dentro de cada faixa e
// contínuo nas emendas. O ritmo real do Campo de Treino também carrega
// o multiplicador do campo (TrainingFieldExpReward, issue #383), que esta curva
// de propósito não embute, para os mesmos templates continuarem sãos em outros lugares.
func killsPerLevel(lvl int32) float64 {
	lerp := func(from, to float64, at, span int32) float64 {
		return from + (to-from)*float64(at)/float64(span)
	}
	switch {
	case lvl <= 35:
		return lerp(6, 10, lvl-1, 34)
	case lvl <= 99:
		return lerp(10, 30, lvl-35, 64)
	case lvl <= 199:
		return lerp(30, 80, lvl-99, 100)
	case lvl <= 299:
		return lerp(80, 200, lvl-199, 100)
	default:
		return lerp(200, 500, lvl-299, 100)
	}
}

// mortalDivisor espelha as faixas MORTAL de applyTierDivisors como floats, para a
// inversão analítica acima (a curva de projeto precisa da razão, não dos
// int-truncation semantics).
func mortalDivisor(lvl int32) float64 {
	switch {
	case lvl <= 200:
		return 1
	case lvl <= 300:
		return 1.07
	case lvl <= 356:
		return 1.25
	case lvl <= 370:
		return 1.70
	case lvl <= 380:
		return 2.10
	case lvl <= 390:
		return 2.60
	default:
		return 4
	}
}
