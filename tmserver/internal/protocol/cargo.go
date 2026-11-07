package protocol

// Auxiliares de wire do Cargo (armazém da conta). Deposit (0x0388) e Withdraw
// (0x0387) são decodificados com StandardParm (Parm = quantidade de ouro); as
// atualizações de slot de item que o cliente mostra no cofre reusam
// EncodeSendItemBody com ItemPlaceCargo. Só o eco dedicado de ouro do cargo vive aqui.

// updateCargoCoinSize é o tamanho no wire de _MSG_UpdateCargoCoin (0x0339),
// 57 bytes no total (protocol-spec.md §opcodes).
const updateCargoCoinSize = 57

// EncodeUpdateCargoCoin monta _MSG_UpdateCargoCoin (0x0339): o ouro do cargo
// compartilhado da conta, enviado após depósito/saque ou quando o cofre é aberto.
// Envie com HEADER.ID = a conn do jogador.
//
// UNVERIFIED: o layout de campos deste corpo de 45 bytes não está documentado; o
// ouro é escrito no offset 0 do corpo como marcador, a ser fixado por captura do
// cliente (SendFunc.cpp:SendCargoCoin legado). O opcode e o tamanho total estão confirmados.
func EncodeUpdateCargoCoin(coin int32) []byte {
	b := make([]byte, updateCargoCoinSize-HeaderSize) // 45
	le.PutUint32(b[0:4], uint32(coin))                // Coin @body0 (offset UNVERIFIED)
	return b
}
