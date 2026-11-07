package refine

import "openwyd-nextgen/tmserver/internal/world"

// bonusValue3/2/4/5 são g_pBonusValue3/2/4/5 (Basedef.cpp:432/467/396/353):
// os bolsões de re-sorteio do SetItemBonus2, um por classe de slot de equip. Cada linha é
// {efeito1, valor1, efeito2, valor2}.
var (
	bonusValue3 = [25][4]int{ // Elmo (nPos 2)
		{4, 60, 26, 18}, {4, 60, 26, 15}, {4, 60, 26, 12},
		{4, 50, 26, 18}, {4, 50, 26, 15}, {4, 50, 26, 12},
		{4, 40, 26, 18}, {4, 40, 26, 15}, {4, 40, 26, 12},
		{4, 30, 26, 18}, {4, 30, 26, 15}, {4, 30, 26, 12},
		{4, 60, 60, 12}, {4, 60, 60, 10}, {4, 60, 60, 8}, {4, 60, 60, 6},
		{4, 50, 60, 12}, {4, 50, 60, 10}, {4, 50, 60, 8}, {4, 50, 60, 6},
		{4, 40, 60, 12}, {4, 40, 60, 10}, {4, 40, 60, 8}, {4, 40, 60, 6}, {4, 40, 60, 4},
	}

	bonusValue2 = [48][4]int{ // Peito/Calça (nPos 4|8)
		{2, 30, 3, 30}, {2, 30, 3, 25}, {2, 30, 3, 20}, {2, 30, 3, 10},
		{2, 24, 3, 30}, {2, 24, 3, 25}, {2, 24, 3, 20}, {2, 24, 3, 15},
		{2, 18, 3, 30}, {2, 18, 3, 25}, {2, 18, 3, 20}, {2, 18, 3, 15},
		{2, 30, 71, 50}, {2, 30, 71, 60}, {2, 30, 71, 70},
		{2, 24, 71, 50}, {2, 24, 71, 60}, {2, 24, 71, 70},
		{2, 18, 71, 50}, {2, 18, 71, 60}, {2, 18, 71, 70},
		{60, 10, 3, 30}, {60, 10, 3, 25}, {60, 10, 3, 20}, {60, 10, 3, 15}, {60, 10, 3, 10},
		{60, 8, 3, 30}, {60, 8, 3, 25}, {60, 8, 3, 20}, {60, 8, 3, 15}, {60, 8, 3, 10},
		{60, 6, 3, 30}, {60, 6, 3, 25}, {60, 6, 3, 20}, {60, 6, 3, 15}, {60, 6, 3, 10},
		{60, 10, 71, 50}, {60, 10, 71, 60}, {60, 10, 71, 70},
		{60, 8, 71, 50}, {60, 8, 71, 60}, {60, 8, 71, 70},
		{60, 6, 71, 50}, {60, 6, 71, 60}, {60, 6, 71, 70},
		{60, 4, 71, 50}, {60, 4, 71, 60}, {60, 4, 71, 70},
	}

	bonusValue4 = [30][4]int{ // Luva (nPos 16)
		{2, 30, 72, 30}, {2, 30, 72, 25}, {2, 30, 72, 20}, {2, 30, 72, 15}, {2, 30, 72, 10},
		{2, 24, 72, 30}, {2, 24, 72, 25}, {2, 24, 72, 20}, {2, 24, 72, 15}, {2, 24, 72, 10},
		{2, 18, 72, 30}, {2, 18, 72, 25}, {2, 18, 72, 20}, {2, 18, 72, 15}, {2, 18, 72, 10},
		{60, 10, 72, 30}, {60, 10, 72, 25}, {60, 10, 72, 20}, {60, 10, 72, 15},
		{60, 8, 72, 30}, {60, 8, 72, 25}, {60, 8, 72, 20}, {60, 8, 72, 15},
		{60, 6, 72, 30}, {60, 6, 72, 25}, {60, 6, 72, 20}, {60, 6, 72, 15},
	}

	bonusValue5 = [30][4]int{ // Bota (nPos 32)
		{2, 30, 74, 18}, {2, 30, 74, 15}, {2, 30, 74, 12},
		{2, 24, 74, 18}, {2, 24, 74, 15}, {2, 24, 74, 12},
		{2, 18, 74, 18}, {2, 18, 74, 15}, {2, 18, 74, 12},
		{2, 12, 74, 18}, {2, 12, 74, 15}, {2, 12, 74, 12},
		{2, 6, 74, 18}, {2, 6, 74, 15}, {2, 6, 74, 12},
		{2, 30, 60, 10}, {2, 30, 60, 8}, {2, 30, 60, 6},
		{2, 24, 60, 10}, {2, 24, 60, 8}, {2, 24, 60, 6},
		{2, 18, 60, 10}, {2, 18, 60, 8}, {2, 18, 60, 6},
		{2, 12, 60, 10}, {2, 12, 60, 8}, {2, 12, 60, 6},
		{2, 6, 60, 10}, {2, 6, 60, 8}, {2, 6, 60, 6},
	}
)

// classeSancCap é o teto +6 que SetItemBonus2 impõe ao sanc que ele aumenta
// (Server.cpp:2727-2739 e seus três irmãos) — diferente do teto próprio do
// caminho de pó, e nunca reduzido, só elevado até ele.
const classeSancCap = 6

// efDamageBonus é EF_DAMAGE (ItemEffect.h:4), reusado aqui para o limite
// só de boot abaixo.
const efDamageBonus = 2

// classeSancEffect é EF_SANC (ItemEffect.h:100). Mantido local (em vez de
// exportado do pacote) porque SetItemBonus2 confere o slot 0
// especificamente, não a varredura geral readSlot/writeSlot do caminho de pó.
const classeSancEffect = efSanc

// Classes de slot de equip que SetItemBonus2 re-sorteia (Basedef.h:1162+). Qualquer outro
// nPos (armas, acessórios, …) fica totalmente intocado — a função
// legada simplesmente não tem ramo `if` para eles.
const (
	nPosHelm      = 2
	nPosChest     = 4
	nPosLegs      = 8
	nPosGlove     = 16
	nPosBoot      = 32
	bootDamageCap = 30
)

// ClasseBonus re-sorteia o sanc de dest (limitado a +6) e seus dois efeitos de bônus,
// escolhidos pelo nPos do slot de equip de dest (SetItemBonus2, Server.cpp:2719-2861).
// roll(n) precisa se comportar como rand()%n; itemAbility precisa somar catálogo +
// efeitos da instância de um id (BASE_GetItemAbility), necessário só para o
// limite de dano no boot. Reporta se nPos era um dos quatro slots cobertos —
// quem chama não precisa ramificar nisso (um item não coberto fica
// inalterado, igual ao legado), mas os testes acham útil.
func ClasseBonus(dest *world.Item, nPos int, roll func(int) int, itemAbility func(world.Item, uint8) int) bool {
	var table [][4]int
	switch nPos {
	case nPosHelm:
		table = bonusValue3[:]
	case nPosChest, nPosLegs:
		table = bonusValue2[:]
	case nPosGlove:
		table = bonusValue4[:]
	case nPosBoot:
		table = bonusValue5[:]
	default:
		return false
	}

	// O índice da tabela é sorteado ANTES de qualquer sorteio de sanc: todo ramo de
	// SetItemBonus2 abre com `int _rand = rand()%N;` e só então toca
	// stEffect[0] (Server.cpp:2723-2726 e seus três irmãos). Manter essa
	// ordem é o que faz uma sequência de rand() capturada reproduzir byte a byte.
	row := table[roll(len(table))]

	if dest.Effects[0].Effect == classeSancEffect {
		lvl := Level(*dest)
		if lvl < classeSancCap {
			lvl += roll(2)
			if lvl >= classeSancCap {
				lvl = classeSancCap
			}
			Set(dest, lvl, 0)
		}
	} else {
		dest.Effects[0] = world.Effect{Effect: classeSancEffect, Value: uint8(roll(2))}
	}

	dest.Effects[1] = world.Effect{Effect: uint8(row[0]), Value: uint8(row[1])}
	dest.Effects[2] = world.Effect{Effect: uint8(row[2]), Value: uint8(row[3])}

	if nPos == nPosBoot {
		clampBootDamage(dest, itemAbility)
	}
	return true
}

// clampBootDamage porta o teto de EF_DAMAGE só de boot (Server.cpp:2845-2861):
// um bônus EF_DAMAGE recém-sorteado é aparado para o dano total do item
// (catálogo + instância, lido DEPOIS do sorteio acima) nunca passar de 30.
func clampBootDamage(dest *world.Item, itemAbility func(world.Item, uint8) int) {
	for i := 1; i <= 2; i++ {
		if dest.Effects[i].Effect != efDamageBonus {
			continue
		}
		ability := itemAbility(*dest, efDamageBonus)
		if ability <= bootDamageCap {
			continue
		}
		over := ability - bootDamageCap
		v := int(dest.Effects[i].Value)
		if v < over {
			over = v
		}
		v -= over
		if v < 0 {
			v = 0
		}
		dest.Effects[i].Value = uint8(v)
	}
}
