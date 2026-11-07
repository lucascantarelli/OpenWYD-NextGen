package refine

import "openwyd-nextgen/tmserver/internal/world"

// efAmount é EF_AMOUNT (ItemEffect.h): um tamanho de pilha, quando o item carrega
// um. Compartilhado pelos pacotes handler e combine, que já dependem
// de refine para a codificação de sanc.
const efAmount = 61

// Amount devolve o tamanho da pilha de um item (BASE_GetItemAmount), ou 1 para item
// sem par EF_AMOUNT.
func Amount(it world.Item) int {
	for _, ef := range it.Effects {
		if ef.Effect == efAmount {
			return int(ef.Value)
		}
	}
	return 1
}
