package refine

import (
	"testing"

	"openwyd-nextgen/tmserver/internal/world"
)

// scriptedRoll devolve uma função de sorteio que entrega vals em ordem (repetindo o
// último se acabar) e registra o módulo de cada chamada, para um teste poder
// verificar os valores consumidos e a ORDEM em que foram sorteados.
func scriptedRoll(vals ...int) (func(int) int, *[]int) {
	var mods []int
	i := 0
	return func(n int) int {
		mods = append(mods, n)
		v := vals[len(vals)-1]
		if i < len(vals) {
			v = vals[i]
		}
		i++
		return v % n
	}, &mods
}

// noAbility substitui BASE_GetItemAbility nos caminhos que não são de boot, onde
// ClasseBonus nunca o consulta.
func noAbility(world.Item, uint8) int { return 0 }

// TestClasseBonusDrawsTableIndexFirst fixa a ordem de chamadas do RNG legado: todo
// ramo de SetItemBonus2 abre com `int _rand = rand()%N;` e só então sorteia
// o sanc (Server.cpp:2723-2726). Trocar os dois dessincronizaria em silêncio uma
// sequência de rand() capturada.
func TestClasseBonusDrawsTableIndexFirst(t *testing.T) {
	dest := world.Item{Index: 100, Effects: [3]world.Effect{{Effect: efSanc, Value: 0}}}
	roll, mods := scriptedRoll(3, 1)

	if !ClasseBonus(&dest, nPosHelm, roll, noAbility) {
		t.Fatal("ClasseBonus reported an uncovered nPos for a helm")
	}

	if len(*mods) != 2 || (*mods)[0] != len(bonusValue3) || (*mods)[1] != 2 {
		t.Fatalf("roll moduli = %v, want [%d 2] (table index drawn first)", *mods, len(bonusValue3))
	}
	want := bonusValue3[3]
	if dest.Effects[1] != (world.Effect{Effect: uint8(want[0]), Value: uint8(want[1])}) ||
		dest.Effects[2] != (world.Effect{Effect: uint8(want[2]), Value: uint8(want[3])}) {
		t.Errorf("effects = %+v, want row %v", dest.Effects, want)
	}
	if got := Level(dest); got != 1 {
		t.Errorf("sanc level = %d, want 1 (0 + roll(2)=1)", got)
	}
}

func TestClasseBonusTablePerSlot(t *testing.T) {
	cases := []struct {
		name string
		nPos int
		size int
	}{
		{"helm", nPosHelm, len(bonusValue3)},
		{"chest", nPosChest, len(bonusValue2)},
		{"legs", nPosLegs, len(bonusValue2)},
		{"glove", nPosGlove, len(bonusValue4)},
		{"boot", nPosBoot, len(bonusValue5)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dest := world.Item{Index: 100, Effects: [3]world.Effect{{Effect: efSanc, Value: 0}}}
			roll, mods := scriptedRoll(0)
			if !ClasseBonus(&dest, c.nPos, roll, noAbility) {
				t.Fatalf("nPos %d reported as uncovered", c.nPos)
			}
			if (*mods)[0] != c.size {
				t.Errorf("table modulus = %d, want %d", (*mods)[0], c.size)
			}
		})
	}
}

func TestClasseBonusSanc(t *testing.T) {
	cases := []struct {
		name      string
		effect0   world.Effect
		roll      int // valor entregue ao sorteio de sanc(2)
		wantLevel int
		wantRolls int
	}{
		{
			name:      "unrefined item steps to +1",
			effect0:   world.Effect{Effect: efSanc, Value: encode(0, 0)},
			roll:      1,
			wantLevel: 1,
			wantRolls: 2,
		},
		{
			name:      "roll of 0 leaves the level alone",
			effect0:   world.Effect{Effect: efSanc, Value: encode(3, 0)},
			roll:      0,
			wantLevel: 3,
			wantRolls: 2,
		},
		{
			name:      "+5 clamps at the +6 ceiling",
			effect0:   world.Effect{Effect: efSanc, Value: encode(5, 0)},
			roll:      1,
			wantLevel: classeSancCap,
			wantRolls: 2,
		},
		{
			name:      "at the cap no sanc roll is drawn at all",
			effect0:   world.Effect{Effect: efSanc, Value: encode(classeSancCap, 0)},
			roll:      1,
			wantLevel: classeSancCap,
			wantRolls: 1,
		},
		{
			name:      "a non-sanc slot 0 is overwritten with a fresh EF_SANC",
			effect0:   world.Effect{Effect: 71, Value: 50},
			roll:      1,
			wantLevel: 1,
			wantRolls: 2,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dest := world.Item{Index: 100, Effects: [3]world.Effect{c.effect0}}
			roll, mods := scriptedRoll(0, c.roll)

			ClasseBonus(&dest, nPosChest, roll, noAbility)

			if dest.Effects[0].Effect != efSanc {
				t.Errorf("Effects[0] = %+v, want EF_SANC", dest.Effects[0])
			}
			if got := Level(dest); got != c.wantLevel {
				t.Errorf("level = %d, want %d", got, c.wantLevel)
			}
			if len(*mods) != c.wantRolls {
				t.Errorf("roll calls = %d (%v), want %d", len(*mods), *mods, c.wantRolls)
			}
		})
	}
}

// TestClasseBonusUncoveredSlot: SetItemBonus2 não tem ramo para armas, capas
// ou acessórios, então o item precisa voltar intocado — o chamador
// (useClasseItem) conta com o retorno falso para recusar em vez de consumir o
// item Classe (Lista de bugs.txt:6).
func TestClasseBonusUncoveredSlot(t *testing.T) {
	for _, nPos := range []int{1, 64, 128, 192, 512, 2048, -32768} {
		before := world.Item{Index: 100, Effects: [3]world.Effect{{Effect: efSanc, Value: 3}}}
		dest := before
		roll, mods := scriptedRoll(0)

		if ClasseBonus(&dest, nPos, roll, noAbility) {
			t.Errorf("nPos %d reported as covered", nPos)
		}
		if dest != before {
			t.Errorf("nPos %d mutated the item: %+v -> %+v", nPos, before, dest)
		}
		if len(*mods) != 0 {
			t.Errorf("nPos %d drew %d rolls, want 0", nPos, len(*mods))
		}
	}
}

// TestClasseBonusBootDamageClamp: um bônus EF_DAMAGE sorteado é aparado para o
// dano total do item ficar em 30 (Server.cpp:2845-2861). A linha 0 de
// bonusValue5 sorteia {EF_DAMAGE 30}, que sobre uma base 20 chega a 50.
func TestClasseBonusBootDamageClamp(t *testing.T) {
	const catalogDamage = 20
	ability := func(it world.Item, id uint8) int {
		if id != efDamageBonus {
			return 0
		}
		total := catalogDamage
		for _, ef := range it.Effects {
			if ef.Effect == id {
				total += int(ef.Value)
			}
		}
		return total
	}

	dest := world.Item{Index: 100, Effects: [3]world.Effect{{Effect: efSanc, Value: 0}}}
	roll, _ := scriptedRoll(0)

	if !ClasseBonus(&dest, nPosBoot, roll, ability) {
		t.Fatal("boot reported as uncovered")
	}

	if row := bonusValue5[0]; row[0] != efDamageBonus {
		t.Fatalf("fixture assumption broken: bonusValue5[0] = %v, want EF_DAMAGE first", row)
	}
	if got := ability(dest, efDamageBonus); got != bootDamageCap {
		t.Errorf("total EF_DAMAGE ability = %d, want %d", got, bootDamageCap)
	}
	if got := dest.Effects[1].Value; got != bootDamageCap-catalogDamage {
		t.Errorf("rolled EF_DAMAGE value = %d, want %d", got, bootDamageCap-catalogDamage)
	}
}
