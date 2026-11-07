package protocol

// O transform de keyword ofusca o payload (bytes [4:Size)); os bytes 0..3 do
// cabeçalho sempre viajam em claro (protocol-spec.md §1.4,
// Source/Code/CPSock.cpp:428-453 decodifica / :556-581 codifica).
//
// Notas de paridade (precisa casar byte a byte):
//   - Toda aritmética é uint8 com wrap mod 256. Em C o Trans deslocado é
//     promovido para int e truncado para char ao guardar; computar a
//     expressão inteira em uint8 reproduz essa truncagem exatamente.
//   - pos nasce de keyWord[iKeyWord*2] e incrementa 1 por byte. Usar
//     um pos uint8 torna "pos % 256" implícito (rst == pos para um uint8).
//   - O caso de deslocamento é escolhido pelo índice absoluto i & 0x3 do byte,
//     então o laço precisa rodar sobre offsets absolutos do pacote a partir de

// decodePayload desofusca buf[transformStart:] no lugar, onde iKeyWord é
// HEADER.KeyWord. Devolve os dois acumuladores de checksum: sum1 sobre os
// bytes planos (depois do transform) e sum2 sobre os cifrados (antes do transform).
// Ver checksum.go para como eles se combinam.
func decodePayload(buf []byte, iKeyWord uint8) (sum1, sum2 uint8) {
	pos := keyWord[uint16(iKeyWord)*2]
	for i := transformStart; i < len(buf); i++ {
		rst := pos
		cipher := buf[i]
		sum2 += cipher
		trans := keyWord[uint16(rst)*2+1]
		switch i & 0x3 {
		case 0:
			buf[i] = cipher - (trans << 1)
		case 1:
			buf[i] = cipher + (trans >> 3)
		case 2:
			buf[i] = cipher - (trans << 2)
		case 3:
			buf[i] = cipher + (trans >> 5)
		}
		sum1 += buf[i]
		pos++
	}
	return sum1, sum2
}

// encodePayload ofusca buf[transformStart:] no lugar — o inverso exato de
// decodePayload — usando iKeyWord como semente. Devolve sum1 sobre os bytes
// planos (antes do transform) e sum2 sobre os cifrados (depois do transform).
func encodePayload(buf []byte, iKeyWord uint8) (sum1, sum2 uint8) {
	pos := keyWord[uint16(iKeyWord)*2]
	for i := transformStart; i < len(buf); i++ {
		plain := buf[i]
		sum1 += plain
		rst := pos
		trans := keyWord[uint16(rst)*2+1]
		switch i & 0x3 {
		case 0:
			buf[i] = plain + (trans << 1)
		case 1:
			buf[i] = plain - (trans >> 3)
		case 2:
			buf[i] = plain + (trans << 2)
		case 3:
			buf[i] = plain - (trans >> 5)
		}
		sum2 += buf[i]
		pos++
	}
	return sum1, sum2
}
