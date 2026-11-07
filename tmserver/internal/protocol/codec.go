package protocol

import "fmt"

// Frame é um pacote CPSock totalmente decodificado: seu cabeçalho plano e a
// corpo (bytes [HeaderSize:Size)).
type Frame struct {
	Header  Header
	Payload []byte
}

// Encode monta os bytes de wire ofuscados completos de um pacote cujo corpo plano
// é payload. h fornece Type, ID e ClientTick; Size, KeyWord e CheckSum são
// calculados aqui. iKeyWord semeia o transform de keyword — é injetável para
// os testes serem determinísticos; em produção é rand()%256 (CPSock.cpp:535).
//
// The obfuscation/checksum cover bytes [4:Size), i.e. Type, ID, ClientTick AND
// o corpo; só Size, KeyWord e CheckSum (bytes 0..3) ficam em claro
// (protocol-spec.md §1.1, §1.4).
func Encode(h Header, payload []byte, iKeyWord uint8) ([]byte, error) {
	size := HeaderSize + len(payload)
	if size < HeaderSize || size > MaxMessageSize {
		return nil, fmt.Errorf("protocol: Encode: invalid size %d (must be %d..%d)", size, HeaderSize, MaxMessageSize)
	}

	buf := make([]byte, size)
	// Escreve o cabeçalho plano completo (CheckSum preenchido depois do transform).
	h.Size = uint16(size)
	h.KeyWord = iKeyWord
	h.CheckSum = 0
	if err := EncodeHeader(buf, h); err != nil {
		return nil, fmt.Errorf("protocol: Encode: %w", err)
	}
	copy(buf[HeaderSize:], payload)

	sum1, sum2 := encodePayload(buf, iKeyWord)
	buf[3] = checksum(sum1, sum2)
	return buf, nil
}

// Decode lê um frame de wire completo (exatamente Size bytes, como produzido pelo
// framer) em seu cabeçalho plano e corpo. mismatch informa divergência de checksum;
// pela semântica legada esta camada NÃO rejeita em divergência (§1.5) — quem
// chama decide. Decode não modifica wire.
func Decode(wire []byte) (h Header, payload []byte, mismatch bool, err error) {
	if len(wire) < HeaderSize {
		return Header{}, nil, false, fmt.Errorf("protocol: Decode: frame too small: %d < %d", len(wire), HeaderSize)
	}
	size := int(uint16(wire[0]) | uint16(wire[1])<<8)
	if size != len(wire) {
		return Header{}, nil, false, fmt.Errorf("protocol: Decode: frame length %d != header Size %d", len(wire), size)
	}

	buf := make([]byte, len(wire))
	copy(buf, wire)

	iKeyWord := buf[2]
	wantSum := buf[3]
	sum1, sum2 := decodePayload(buf, iKeyWord)
	mismatch = checksum(sum1, sum2) != wantSum

	h, err = DecodeHeader(buf)
	if err != nil {
		return Header{}, nil, false, fmt.Errorf("protocol: Decode: %w", err)
	}
	payload = buf[HeaderSize:]
	return h, payload, mismatch, nil
}
