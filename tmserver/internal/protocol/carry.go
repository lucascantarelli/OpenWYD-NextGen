package protocol

const updateCarrySize = HeaderSize + 64*ItemSize + 4

// EncodeUpdateCarryBody monta MSG_UpdateCarry: o snapshot completo de Carry[64]
// mais o ouro carregado. O legado envia isto após mudanças na Bolsa do Andarilho
// para o cliente recalcular quais páginas de inventário estão disponíveis.
func EncodeUpdateCarryBody(carry [64]SelItem, coin int32) []byte {
	b := make([]byte, updateCarrySize-HeaderSize)
	for i := 0; i < 64; i++ {
		writeSelItem(b[i*ItemSize:], carry[i])
	}
	le.PutUint32(b[64*ItemSize:], uint32(coin))
	return b
}
