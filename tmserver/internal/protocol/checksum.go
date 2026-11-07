package protocol

// checksum devolve o checksum aditivo de 1 byte do CPSock, CheckSum = Sum2 - Sum1
// (wrap uint8), onde sum1 é a soma dos bytes planos do payload e sum2 a
// soma dos bytes cifrados do payload (protocol-spec.md §1.5,
// Source/Code/CPSock.cpp:583-584).
func checksum(sum1, sum2 uint8) uint8 {
	return sum2 - sum1
}

// O servidor e o cliente originais NÃO rejeitam divergência de checksum: o
// servidor devolve o pacote com ErrorCode=1 de qualquer forma (CPSock.cpp:457-466), e
// o ClientPatch transforma as checagens do cliente em NOP (Hook.cpp:211-214). Para
// compatibilidade de wire PRECISAMOS emitir checksum correto no envio (feito em
// codec.go), e na recepção calculamos e informamos divergência mas, nesta camada do
// codec, não rejeitamos — endurecer para rejeitar em divergência é decisão da Fase 7
// (protocol-spec.md §1.5, migration-plan.md §5).
