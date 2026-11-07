// Package protocol implementa o codec de wire CPSock usado entre o
// cliente WYD.exe 7662 sem modificação e o servidor de jogo (tmServer).
//
// Tudo aqui segue protocol-spec.md (Fase 1). Todo layout de wire é lido
// e escrito por offset explícito, little-endian, via encoding/binary — nunca por
// depender do layout de struct do Go (NF2/NF8, migration-plan.md §5). O
// pipeline é: gate INITCODE → framing por Size → transform de keyword → checksum.
package protocol

import (
	"encoding/binary"
	"fmt"
)

// Constantes de transporte (protocol-spec.md §4.5; Source/Code/CPSock.h, Basedef.h).
const (
	// HeaderSize é sizeof(HEADER) — um prefixo fixo de 12 bytes em todo pacote
	// exceto o handshake e o link de billing (protocol-spec.md §1.1).
	HeaderSize = 12

	// InitCode é o handshake de 4 bytes que toda conexão TCP nova precisa enviar
	// primeiro (CPSock.h:40, §1.2). Little-endian no wire: 11 F3 11 1F.
	InitCode uint32 = 0x1F11F311

	// MaxMessageSize é o Size máximo de pacote aceito (CPSock.h:38, §1.3).
	MaxMessageSize = 8192

	// AppVersion é a versão de protocolo que o cliente trafega em
	// MSG_AccountLogin.ClientVersion (Basedef.h:102, §4.1). É 7640, NÃO 7662
	// (7662 é o nome do build do executável/patch).
	AppVersion = 7640

	// SkipCheckTick é um valor de tick interno que o cliente nunca deve enviar; o
	// dispatcher descarta pacotes de cliente que o carreguem (Basedef.h:232, §2).
	SkipCheckTick uint32 = 235543242

	// MaxUser é a contagem de slots de conexão/jogador por canal; HEADER.ID precisa
	// estar em [0, MaxUser) (Basedef.h:116, §2).
	MaxUser = 1000

	// transformStart é o offset onde a obfuscação e o checksum começam; o
	// primeiros quatro bytes (Size, KeyWord, CheckSum) viajam em claro (§1.1, §1.4).
	transformStart = 4
)

// Header é o cabeçalho de pacote CPSock de 12 bytes (protocol-spec.md §1.1,
// CPSock.h:42-50). Little-endian.
//
//	Offset  Field        Type     Bytes
//	0       Size         uint16   2   tamanho total do pacote, incluindo o cabeçalho
//	2       KeyWord      uint8    1   índice aleatório (0..255) nas sementes keyWord
//	3       CheckSum     uint8    1   checksum aditivo do payload (§1.5)
//	4       Type         uint16   2   id da mensagem, incluindo as flags de direção (§2)
//	6       ID           uint16   2   slot de conexão/jogador em pUser[]
//	8       ClientTick   uint32   4   timestamp/tick
type Header struct {
	Size       uint16
	KeyWord    uint8
	CheckSum   uint8
	Type       Type
	ID         uint16
	ClientTick uint32
}

// EncodeHeader escreve h nos primeiros HeaderSize bytes de dst por
// offset explícito. dst precisa ter ao menos HeaderSize bytes.
func EncodeHeader(dst []byte, h Header) error {
	if len(dst) < HeaderSize {
		return fmt.Errorf("protocol: EncodeHeader: dst too small: have %d, need %d", len(dst), HeaderSize)
	}
	binary.LittleEndian.PutUint16(dst[0:2], h.Size)
	dst[2] = h.KeyWord
	dst[3] = h.CheckSum
	binary.LittleEndian.PutUint16(dst[4:6], uint16(h.Type))
	binary.LittleEndian.PutUint16(dst[6:8], h.ID)
	binary.LittleEndian.PutUint32(dst[8:12], h.ClientTick)
	return nil
}

// DecodeHeader lê um Header dos primeiros HeaderSize bytes de src por
// offset explícito. src precisa ter ao menos HeaderSize bytes.
func DecodeHeader(src []byte) (Header, error) {
	if len(src) < HeaderSize {
		return Header{}, fmt.Errorf("protocol: DecodeHeader: src too small: have %d, need %d", len(src), HeaderSize)
	}
	return Header{
		Size:       binary.LittleEndian.Uint16(src[0:2]),
		KeyWord:    src[2],
		CheckSum:   src[3],
		Type:       Type(binary.LittleEndian.Uint16(src[4:6])),
		ID:         binary.LittleEndian.Uint16(src[6:8]),
		ClientTick: binary.LittleEndian.Uint32(src[8:12]),
	}, nil
}
