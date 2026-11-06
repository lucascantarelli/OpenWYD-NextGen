// Package itemeffects dá nome aos códigos de STRUCT_EFFECT.cEffect carregados
// em Eff1..Eff3 de um STRUCT_ITEM, para que uma tela de administração diga
// "Dano +70" em vez de "2:70".
//
// O vocabulário é o mesmo cabeçalho que o servidor legado e o cliente compilam;
// names[] espelha as constantes `#define EF_*` uma a uma. labels[] é o rótulo
// curto em português que o portal mostra, limitado de propósito aos códigos que
// aparecem como adds em saves reais — o resto cai no nome da constante (Name) e,
// quando o código não tem constante nenhuma, em "" para o chamador mostrar o
// número cru.
//
// É um pacote folha: nenhum import além da biblioteca padrão, então tanto o site
// (webserver) quanto os editores da web podem compartilhá-lo.
package itemeffects

import (
	"sort"
	"sync"
)

// names mapeia um código para a sua constante, sem o prefixo EF_.
var names = map[int32]string{
	0: "NONE", 1: "LEVEL", 2: "DAMAGE", 3: "AC", 4: "HP", 5: "MP", 6: "EXP",
	7: "STR", 8: "INT", 9: "DEX", 10: "CON",
	11: "SPECIAL1", 12: "SPECIAL2", 13: "SPECIAL3", 14: "SPECIAL4",
	15: "SCORE14", 16: "SCORE15", 17: "POS", 18: "CLASS", 19: "R1SIDC", 20: "R2SIDC",
	21: "WTYPE", 22: "REQ_STR", 23: "REQ_INT", 24: "REQ_DEX", 25: "REQ_CON",
	26: "ATTSPEED", 27: "RANGE", 28: "PRICE", 29: "RUNSPEED", 30: "SPELL",
	31: "DURATION", 32: "PARM2", 33: "GRID", 34: "GROUND", 35: "CLAN",
	36: "HWORDCOIN", 37: "LWORDCOIN", 38: "VOLATILE", 39: "KEYID",
	40: "PARRY", 41: "HITRATE", 42: "CRITICAL", 43: "SANC", 44: "SAVEMANA",
	45: "HPADD", 46: "MPADD", 47: "REGENHP", 48: "REGENMP",
	49: "RESIST1", 50: "RESIST2", 51: "RESIST3", 52: "RESIST4",
	53: "ACADD", 54: "RESISTALL", 55: "BONUS", 56: "HWORDGUILD", 57: "LWORDGUILD",
	58: "QUEST", 59: "UNIQUE", 60: "MAGIC", 61: "AMOUNT", 62: "HWORDINDEX", 63: "LWORDINDEX",
	64: "INIT1", 65: "INIT2", 66: "INIT3", 67: "DAMAGEADD", 68: "MAGICADD",
	69: "HPADD2", 70: "MPADD2", 71: "CRITICAL2", 72: "ACADD2", 73: "DAMAGE2",
	74: "SPECIALALL", 75: "CURKILL", 76: "LTOTKILL", 77: "HTOTKILL", 78: "INCUBATE",
	79: "MOUNTLIFE", 80: "MOUNTHP", 81: "MOUNTSANC", 82: "MOUNTFEED", 83: "MOUNTKILL",
	84: "INCUDELAY", 85: "SUBGUILD", 87: "ITEMLEVEL", 91: "DONATE",
	100: "GRADE0", 101: "GRADE1", 102: "GRADE2", 103: "GRADE3", 104: "GRADE4", 105: "GRADE5",
	106: "WDAY", 107: "HOUR", 108: "MIN", 109: "YEAR", 110: "WMONTH",
	112: "MOBTYPE", 113: "ITEMTYPE", 126: "NOSANC", 127: "NOTRADE",
}

// labels são os nomes curtos do portal para os códigos que de fato aparecem
// como adds. A ordem das resistências segue o conteúdo: EF_RESIST1 é Fogo, 2
// Gelo, 3 Sagrado, 4 Relâmpago — a mesma ordem de que as orbes dependem.
var labels = map[int32]string{
	2: "Dano", 3: "Defesa", 4: "HP", 5: "MP", 6: "EXP",
	7: "Força", 8: "Inteligência", 9: "Destreza", 10: "Constituição",
	11: "Especial 1", 12: "Especial 2", 13: "Especial 3", 14: "Especial 4",
	17: "Posição", 18: "Classe", 21: "Tipo de arma",
	26: "Vel. de ataque", 27: "Alcance", 29: "Vel. de movimento", 30: "Skill",
	33: "Espaço no cargo", 38: "Consumível", 40: "Aparo", 41: "Precisão",
	42: "Crítico", 43: "Refino", 44: "Economia de mana",
	45: "HP %", 46: "MP %", 47: "Regen. HP", 48: "Regen. MP",
	49: "Resist. fogo", 50: "Resist. gelo", 51: "Resist. sagrado", 52: "Resist. trovão",
	53: "Defesa add", 54: "Resist. todos", 58: "Quest", 59: "Único",
	60: "Ataque mágico", 61: "Quantidade", 67: "Dano add", 68: "Ataque mágico add",
	69: "HP % (2)", 70: "MP % (2)", 71: "Crítico (2)", 72: "Defesa add (2)", 73: "Dano (2)",
	74: "Todos os especiais", 78: "Choco", 87: "Nível do item", 91: "Donate",
	100: "Grade A", 101: "Grade B", 102: "Grade C", 103: "Grade D", 104: "Grade E",
	105: "Grade Arch", 112: "Tipo de mob", 113: "Tipo de item",
	126: "Sem refino", 127: "Intransferível",
}

// Type é uma linha do vocabulário, para UIs que precisam da tabela inteira.
type Type struct {
	Code  int32
	Name  string
	Label string
}

var (
	codesOnce sync.Once
	codes     map[string]int32
)

func codeIndex() map[string]int32 {
	codesOnce.Do(func() {
		codes = make(map[string]int32, len(names))
		for code, name := range names {
			codes["EF_"+name] = code
		}
	})
	return codes
}

// Code resolve um token de catálogo como "EF_CRITICAL" para o código numérico.
func Code(token string) (int32, bool) {
	code, ok := codeIndex()[token]
	return code, ok
}

// Known diz se o código existe no vocabulário. Uma concessão ou edição nunca
// pode guardar um código que o cliente não sabe interpretar.
func Known(code int32) bool {
	_, ok := names[code]
	return ok
}

// All devolve o vocabulário ordenado por código, para os seletores de efeito do
// portal.
func All() []Type {
	out := make([]Type, 0, len(names))
	for code, name := range names {
		out = append(out, Type{Code: code, Name: name, Label: Label(code)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// Name devolve a constante do código (sem o prefixo EF_). Vazio significa que o
// código não está no vocabulário.
func Name(code int32) string { return names[code] }

// Label devolve o rótulo do portal para um código: o nome curado em português
// quando existe, senão a constante crua. Vazio significa que o código é
// desconhecido e o chamador deve mostrar o número.
func Label(code int32) string {
	if label, ok := labels[code]; ok {
		return label
	}
	return names[code]
}
