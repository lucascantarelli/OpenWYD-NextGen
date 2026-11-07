package level

const (
	classArch        uint8 = 1
	classMortal      uint8 = 2
	classCelestial   uint8 = 3
	classCelestialCS uint8 = 4
	classSCelestial  uint8 = 5

	// soloExpGate é a janela de concessão (MobKilled.cpp:1284): recompensa escalada
	// fora de (0, 10M] pula a concessão inteira — o gate legado NÃO limita.
	soloExpGate int64 = 10_000_000
)

// ExpEvents são as flags globais de evento de EXP (globais de MobKilled.cpp:1372-1384
// NewbieEventServer / DOUBLEMODE / KefraLive).
type ExpEvents struct {
	DoubleMode  bool
	NewbieEvent bool
	KefraLive   bool // Nome legado: true significa derrotado (cidade aberta); false corta o EXP pela metade.
}

// SoloExpReward calcula a experiência PvE solo de um matador de nível killerLevel
// abatendo um mob de nível mobLevel cuja recompensa de template é mobExp. É o
// ramo de campo geral da distribuição legada (MobKilled.cpp:1272-1425)
// reduzido a uma party de um — o ramo que rege os mapas normais, inclusive o
// o Campo de Treino. (game-rules.md §1 antes transcrevia o
// ramo de mapa Nightmare em :443-590, cujas tabelas de divisor diferem.)
//
// Pipeline, na ordem legada: escala de razão de nível do GetExpApply → +MAX_LEVEL+1
// deslocamento de nível das tiers celestiais → ×450/(30+myLevel) → o gate (0,10M] →
// divisores de nível por tier → ×0.6 → limitado ao valor de GetExpApply sem divisão
// (eMob, :1360) → bônus de exp do item → newbie +25% → double ×2 → Kefra-vivo ÷2 →
// ±15% newbie swing.
//
// Não modelado (adiado junto com os sistemas a que pertence): a divisão de party e
// seu fator g_EmptyMob/PARTYBONUS, o bônus da fada de estimação, os +5% da guerra
// RvR e o acúmulo em DayLog/Hold (:1386-1408).
func SoloExpReward(mobExp int64, killerLevel, mobLevel int32, classMaster uint8, expBonus int32, ev ExpEvents) int64 {
	exp := soloBaseExp(mobExp, killerLevel, mobLevel, classMaster, expBonus)
	if exp <= 0 {
		return 0
	}
	if ev.NewbieEvent && killerLevel < 100 && !isCelestialTier(classMaster) {
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
	if exp <= 0 {
		return 0
	}
	return exp
}

// TrainingFieldExpReward é a recompensa solo por um abate no Campo de Treino
// (geradores que nascem em células AttributeMap 0x80): a recompensa base sem eventos
// de SoloExpReward vezes mult. Divergência consciente do legado (issue
// #383): o campo precisa levar um personagem novo ao nível 35 exibido em cerca
// de 15 minutos, então seu ritmo não pode depender dos eventos globais — o ÷2 do
// Kefra e o ±15% de newbie (×0.425 num servidor padrão) são pulados, e o mesmo
// vale para o DOUBLEMODE. O produto é limitado pelo gate de concessão (0,10M].
func TrainingFieldExpReward(mobExp int64, killerLevel, mobLevel int32, classMaster uint8, expBonus int32, mult int64) int64 {
	exp := soloBaseExp(mobExp, killerLevel, mobLevel, classMaster, expBonus)
	if exp <= 0 || mult <= 0 {
		return 0
	}
	if exp > soloExpGate/mult {
		return soloExpGate
	}
	return exp * mult
}

// soloBaseExp é o início independente de eventos do pipeline de campo geral:
// tudo até o bônus de EXP de item, inclusive.
func soloBaseExp(mobExp int64, killerLevel, mobLevel int32, classMaster uint8, expBonus int32) int64 {
	isExp := ExpApply(mobExp, killerLevel, mobLevel)
	if isExp <= 0 {
		return 0
	}
	// Solo: o matador é o único membro da party, então o teto por membro, eMob, é
	// o próprio valor de GetExpApply dele (:1276 e :1360 calculam a mesma expressão).
	eMob := isExp
	myLevel := int64(killerLevel)
	if classMaster != classMortal && classMaster != classArch {
		myLevel += int64(MaxLevel) + 1
	}
	exp := 450 * isExp / (30 + myLevel)
	if exp <= 0 || exp > soloExpGate {
		return 0
	}
	exp = applyTierDivisors(exp, myLevel, classMaster)
	exp = 6 * exp / 10
	if exp > eMob {
		exp = eMob
	}
	if expBonus > 0 && expBonus < 500 {
		exp += exp * int64(expBonus) / 100
	}
	return exp
}

func isCelestialTier(classMaster uint8) bool {
	switch classMaster {
	case classCelestial, classCelestialCS, classSCelestial:
		return true
	default:
		return false
	}
}

// applyTierDivisors é a tabela de divisores por faixa de nível da tier do
// ramo de campo geral (MobKilled.cpp:1286-1356). O código C divide um int por
// literais mistos float/double (`exp /= 1.07f` vs `exp /= 1.70`), truncando
// de volta para int; a largura float32/float64 é preservada por literal para as
// bordas de truncagem casarem.
func applyTierDivisors(exp, myLevel int64, classMaster uint8) int64 {
	div32 := func(d float32) int64 { return int64(float32(exp) / d) }
	div64 := func(d float64) int64 { return int64(float64(exp) / d) }
	switch classMaster {
	case classMortal:
		switch {
		case myLevel <= 200: // ÷1
		case myLevel <= 300:
			exp = div32(1.07)
		case myLevel <= 356:
			exp = div32(1.25)
		case myLevel <= 370:
			exp = div64(1.70)
		case myLevel <= 380:
			exp = div32(2.10)
		case myLevel <= 390:
			exp = div64(2.60)
		case myLevel <= 399:
			exp /= 4
		}
	case classArch:
		switch {
		case myLevel <= 200: // ÷1
		case myLevel <= 300:
			exp = div32(0.85)
		case myLevel <= 356:
			exp = div32(0.90)
		case myLevel <= 360:
			exp = div32(4.50)
		case myLevel <= 370:
			exp = div32(5.90)
		case myLevel <= 380:
			exp /= 11
		case myLevel <= 390:
			exp /= 17
		case myLevel <= 400:
			exp /= 35
		}
	default:
		switch {
		case myLevel < 120:
			exp /= 10
		case myLevel < 150:
			exp /= 20
		case myLevel < 170:
			exp /= 40
		case myLevel < 180:
			exp /= 80
		case myLevel < 190:
			exp /= 160
		default:
			exp /= 320
		}
	}
	return exp
}
