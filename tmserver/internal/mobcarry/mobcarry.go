// Package mobcarry aplica tabelas Carry[] editadas por moderador sobre os bytes
// crus do template STRUCT_MOB (issue 34, fatia 3) — a metade de drop da mesma
// superfície de edição que mobstat cobre para as stats.
//
// A tabela Carry[] é a tabela de drop de um mob: o SLOT decide a raridade do item
// (DropRate[slot] é o divisor) e o item nela é o que cai. Editá-la pela web é uma
// linha de sobreposição em mob_carry_override; o arquivo em tmserver/content/ nunca
// é escrito. A sobreposição é aplicada no carregamento do template, uma vez no boot,
// atrás de -mob-carry-editing (OFF = arquivos exatamente como estão).
//
// A semântica é autoritativa por template, como na sobreposição de Equip[]: um slot
// ausente na sobreposição fica vazio, não "mantém o item do arquivo". A web-api
// sempre envia a tabela inteira, então "ausente" só acontece quando um moderador
// limpou o slot.
//
// Paridade de RNG: este pacote só muda valores de item. Ele não adiciona, remove
// nem reordena chamada de rand(), então a sequência de drop continua igual à do
// servidor legado para a mesma tabela.
package mobcarry

import (
	"openwyd-nextgen/internal/savefmt"
)

// Slot é a sobreposição de uma posição de Carry[] (0..63, savefmt.MaxCarry).
type Slot struct {
	Slot      int
	ItemIndex uint16
	Eff       [3][2]uint8
}

// Override é a tabela Carry[] inteira de um template.
type Override struct {
	Slots []Slot
}

// Apply devolve template com seu Carry[] substituído por ov, codificado no
// layout canônico de 816 bytes. Um template corrompido volta inalterado; um slot
// fora da faixa é ignorado em vez de dar panic (a web valida antes, então um
// slot ignorado significa que a tabela foi editada fora de banda).
func Apply(template []byte, ov Override) []byte {
	mob, _, err := savefmt.DecodeMobAny(template)
	if err != nil {
		return template
	}

	mob.Carry = [savefmt.MaxCarry]savefmt.Item{}
	for _, s := range ov.Slots {
		if s.Slot < 0 || s.Slot >= savefmt.MaxCarry || s.ItemIndex == 0 {
			continue
		}
		mob.Carry[s.Slot] = savefmt.Item{
			Index: int16(s.ItemIndex),
			Effects: [3]savefmt.Effect{
				{Effect: s.Eff[0][0], Value: s.Eff[0][1]},
				{Effect: s.Eff[1][0], Value: s.Eff[1][1]},
				{Effect: s.Eff[2][0], Value: s.Eff[2][1]},
			},
		}
	}
	return savefmt.EncodeMob(mob)
}

// ApplyOverride aplica overrides[name] a template se presente, devolvendo o
// template inalterado caso contrário. Chamado dos dois pontos de resolução de
// template do tmServer (spawn do NPCGener e o loader de config de NPC gerenciado
// pelo banco), igual a mobstat.ApplyOverride.
func ApplyOverride(template []byte, name string, overrides map[string]Override) []byte {
	if ov, ok := overrides[name]; ok {
		return Apply(template, ov)
	}
	return template
}
