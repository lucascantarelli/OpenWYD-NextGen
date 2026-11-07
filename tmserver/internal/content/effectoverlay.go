package content

// ItemEffectOverride é um efeito base editado por moderador (issue 44) como o world
// o consome: Value substitui Effect em ItemIndex. A chave é o código de
// ItemEffect.h — a mesma chave que BaseEffects usa — não uma posição nos 3
// slots do STRUCT_ITEM, porque uma linha do catálogo pode carregar até uma dúzia de pares (o item 1188 tem seis).
type ItemEffectOverride struct {
	ItemIndex int
	Effect    uint8
	Value     int16
}

// EffectOverrideStats conta o que a sobreposição de boot realmente fez, para o
// log do servidor mostrar isso em vez de uma contagem crua de linhas.
type EffectOverrideStats struct {
	Replaced int // o item já declarava este efeito: valor trocado
	Added    int // o item não o declarava: par acrescentado
	Ignored  int // código que o modelo de score não representa: omitido de propósito
}

// scoreEffectCodes é o conjunto de códigos que BaseEffects consegue emitir (os
// valores de efName). Uma sobreposição para qualquer outro é recusada: os handlers
// somam equipamento no score percorrendo exatamente esses códigos, e um código
// injetado aqui ficaria morto em silêncio (pior, faria o catálogo alegar um bônus
// que o jogo nunca aplica). EF_RANGE, EF_VOLATILE e companhia têm leitores próprios.
var scoreEffectCodes = func() map[uint8]struct{} {
	out := make(map[uint8]struct{}, len(efName))
	for _, code := range efName {
		out[code] = struct{}{}
	}
	return out
}()

// ApplyItemEffectOverrides aplica a sobreposição da web nos efeitos do catálogo.
//
// A precedência continua instância > sobreposição > catálogo: isto só reescreve a
// camada do catálogo, então os refinos do próprio item (stEffect/stValue) ainda vencem
// na hora do score. O mapa de entrada nunca é mutado — uma sobreposição vazia devolve o
// catálogo como está, então a flag desligada e ligada sem linhas são indistinguíveis.
func ApplyItemEffectOverrides(base map[int][]BaseEffect, rows []ItemEffectOverride) (map[int][]BaseEffect, EffectOverrideStats) {
	var stats EffectOverrideStats
	if len(rows) == 0 {
		return base, stats
	}

	out := make(map[int][]BaseEffect, len(base))
	for idx, effs := range base {
		out[idx] = effs
	}

	for _, row := range rows {
		if _, ok := scoreEffectCodes[row.Effect]; !ok {
			stats.Ignored++
			continue
		}
		// Copy-on-write por item: os slices do catálogo são compartilhados com quem chama.
		effs, seen := out[row.ItemIndex], false
		next := make([]BaseEffect, len(effs))
		copy(next, effs)
		for i := range next {
			if next[i].Eff == row.Effect {
				next[i].Val = row.Value
				seen = true
				break
			}
		}
		if seen {
			stats.Replaced++
		} else {
			next = append(next, BaseEffect{Eff: row.Effect, Val: row.Value})
			stats.Added++
		}
		out[row.ItemIndex] = next
	}
	return out, stats
}
