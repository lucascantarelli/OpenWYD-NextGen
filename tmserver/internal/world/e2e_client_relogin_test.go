//go:build e2e

// Fecha o ciclo de troca de canal pelo lado do CLIENTE: o pacote de relogin aqui
// é montado byte a byte segundo o struct que a DLL do cliente compila
// (Basedef.h:1546-1561) e emoldurado pelo CPSock (CPSock.cpp:535-583), levando em
// Zero[52] o Enc que o servidor emitiu em MSG_DBCNFServerChange. O aceite é
// medido contra o tmServer de destino vivo.
package world

import (
	"encoding/binary"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"openwyd-nextgen/internal/crossover"
	"openwyd-nextgen/tmserver/internal/protocol"
)

// cpsockKeyWord é a tabela de ofuscação do CPSock, copiada de
// Source/Code/CPSock.cpp:29 ("7.xx keys"). 256 pares: [2i] é a chave do quadro e
// [2i+1] o byte usado na transformação do corpo.
var cpsockKeyWord = [512]byte{
	0x84, 0x87, 0x37, 0xd7, 0xea, 0x79, 0x91, 0x7d, 0x4b, 0x4b, 0x85, 0x7d, 0x87, 0x81, 0x91, 0x7c, 0x0f, 0x73, 0x91, 0x91, 0x87, 0x7d, 0x0d, 0x7d, 0x86, 0x8f, 0x73, 0x0f, 0xe1, 0xdd, 0x85, 0x7d,
	0x05, 0x7d, 0x85, 0x83, 0x87, 0x9c, 0x85, 0x33, 0x0d, 0xe2, 0x87, 0x19, 0x0f, 0x79, 0x85, 0x86, 0x37, 0x7d, 0xd7, 0xdd, 0xe9, 0x7d, 0xd7, 0x7d, 0x85, 0x79, 0x05, 0x7d, 0x0f, 0xe1, 0x87, 0x7e,
	0x23, 0x87, 0xf5, 0x79, 0x5f, 0xe3, 0x4b, 0x83, 0xa3, 0xa2, 0xae, 0x0e, 0x14, 0x7d, 0xde, 0x7e, 0x85, 0x7a, 0x85, 0xaf, 0xcd, 0x7d, 0x87, 0xa5, 0x87, 0x7d, 0xe1, 0x7d, 0x88, 0x7d, 0x15, 0x91,
	0x23, 0x7d, 0x87, 0x7c, 0x0d, 0x7a, 0x85, 0x87, 0x17, 0x7c, 0x85, 0x7d, 0xac, 0x80, 0xbb, 0x79, 0x84, 0x9b, 0x5b, 0xa5, 0xd7, 0x8f, 0x05, 0x0f, 0x85, 0x7e, 0x85, 0x80, 0x85, 0x98, 0xf5, 0x9d,
	0xa3, 0x1a, 0x0d, 0x19, 0x87, 0x7c, 0x85, 0x7d, 0x84, 0x7d, 0x85, 0x7e, 0xe7, 0x97, 0x0d, 0x0f, 0x85, 0x7b, 0xea, 0x7d, 0xad, 0x80, 0xad, 0x7d, 0xb7, 0xaf, 0x0d, 0x7d, 0xe9, 0x3d, 0x85, 0x7d,
	0x87, 0xb7, 0x23, 0x7d, 0xe7, 0xb7, 0xa3, 0x0c, 0x87, 0x7e, 0x85, 0xa5, 0x7d, 0x76, 0x35, 0xb9, 0x0d, 0x6f, 0x23, 0x7d, 0x87, 0x9b, 0x85, 0x0c, 0xe1, 0xa1, 0x0d, 0x7f, 0x87, 0x7d, 0x84, 0x7a,
	0x84, 0x7b, 0xe1, 0x86, 0xe8, 0x6f, 0xd1, 0x79, 0x85, 0x19, 0x53, 0x95, 0xc3, 0x47, 0x19, 0x7d, 0xe7, 0x0c, 0x37, 0x7c, 0x23, 0x7d, 0x85, 0x7d, 0x4b, 0x79, 0x21, 0xa5, 0x87, 0x7d, 0x19, 0x7d,
	0x0d, 0x7d, 0x15, 0x91, 0x23, 0x7d, 0x87, 0x7c, 0x85, 0x7a, 0x85, 0xaf, 0xcd, 0x7d, 0x87, 0x7d, 0xe9, 0x3d, 0x85, 0x7d, 0x15, 0x79, 0x85, 0x7d, 0xc1, 0x7b, 0xea, 0x7d, 0xb7, 0x7d, 0x85, 0x7d,
	0x85, 0x7d, 0x0d, 0x7d, 0xe9, 0x73, 0x85, 0x79, 0x05, 0x7d, 0xd7, 0x7d, 0x85, 0xe1, 0xb9, 0xe1, 0x0f, 0x65, 0x85, 0x86, 0x2d, 0x7d, 0xd7, 0xdd, 0xa3, 0x8e, 0xe6, 0x7d, 0xde, 0x7e, 0xae, 0x0e,
	0x0f, 0xe1, 0x89, 0x7e, 0x23, 0x7d, 0xf5, 0x79, 0x23, 0xe1, 0x4b, 0x83, 0x0c, 0x0f, 0x85, 0x7b, 0x85, 0x7e, 0x8f, 0x80, 0x85, 0x98, 0xf5, 0x7a, 0x85, 0x1a, 0x0d, 0xe1, 0x0f, 0x7c, 0x89, 0x0c,
	0x85, 0x0b, 0x23, 0x69, 0x87, 0x7b, 0x23, 0x0c, 0x1f, 0xb7, 0x21, 0x7a, 0x88, 0x7e, 0x8f, 0xa5, 0x7d, 0x80, 0xb7, 0xb9, 0x18, 0xbf, 0x4b, 0x19, 0x85, 0xa5, 0x91, 0x80, 0x87, 0x81, 0x87, 0x7c,
	0x0f, 0x73, 0x91, 0x91, 0x84, 0x87, 0x37, 0xd7, 0x86, 0x79, 0xe1, 0xdd, 0x85, 0x7a, 0x73, 0x9b, 0x05, 0x7d, 0x0d, 0x83, 0x87, 0x9c, 0x85, 0x33, 0x87, 0x7d, 0x85, 0x0f, 0x87, 0x7d, 0x0d, 0x7d,
	0xf6, 0x7e, 0x87, 0x7d, 0x88, 0x19, 0x89, 0xf5, 0xd1, 0xdd, 0x85, 0x7d, 0x8b, 0xc3, 0xea, 0x7a, 0xd7, 0xb0, 0x0d, 0x7d, 0x87, 0xa5, 0x87, 0x7c, 0x73, 0x7e, 0x7d, 0x86, 0x87, 0x23, 0x85, 0x10,
	0xd7, 0xdf, 0xed, 0xa5, 0xe1, 0x7a, 0x85, 0x23, 0xea, 0x7e, 0x85, 0x98, 0xad, 0x79, 0x86, 0x7d, 0x85, 0x7d, 0xd7, 0x7d, 0xe1, 0x7a, 0xf5, 0x7d, 0x85, 0xb0, 0x2b, 0x37, 0xe1, 0x7a, 0x87, 0x79,
	0x84, 0x7d, 0x73, 0x73, 0x87, 0x7d, 0x23, 0x7d, 0xe9, 0x7d, 0x85, 0x7e, 0x02, 0x7d, 0xdd, 0x2d, 0x87, 0x79, 0xe7, 0x79, 0xad, 0x7c, 0x23, 0xda, 0x87, 0x0d, 0x0d, 0x7b, 0xe7, 0x79, 0x9b, 0x7d,
	0xd7, 0x8f, 0x05, 0x7d, 0x0d, 0x34, 0x8f, 0x7d, 0xad, 0x87, 0xe9, 0x7c, 0x85, 0x80, 0x85, 0x79, 0x8a, 0xc3, 0xe7, 0xa5, 0xe8, 0x6b, 0x0d, 0x74, 0x10, 0x73, 0x33, 0x17, 0x0d, 0x37, 0x21, 0x19,
}

// cpsockFrame monta um pacote C→S como CPSock::AddMessage (CPSock.cpp:535-583):
// o cabeçalho de 4 bytes (Size, KeyWord, CheckSum) segue em claro e o restante
// (a partir do offset 4, Type incluído) é transformado byte a byte com a tabela;
// CheckSum = soma(transformado) − soma(original), em aritmética de 8 bits.
// O campo KeyWord no wire é o ÍNDICE sorteado, não a chave.
func cpsockFrame(keyIndex byte, typ protocol.Type, body []byte) []byte {
	size := 12 + len(body)
	pkt := make([]byte, size)
	binary.LittleEndian.PutUint16(pkt[0:2], uint16(size))
	pkt[2] = keyIndex
	binary.LittleEndian.PutUint16(pkt[4:6], uint16(typ))
	copy(pkt[12:], body)

	var sumOriginal, sumTransformed byte
	pos := int(cpsockKeyWord[keyIndex*2])
	for i := 4; i < size; i++ {
		sumOriginal += pkt[i]
		trans := int(cpsockKeyWord[(pos%256)*2+1])
		var out byte
		switch i & 3 {
		case 0:
			out = byte(int(pkt[i]) + (trans << 1))
		case 1:
			out = byte(int(pkt[i]) - (trans >> 3))
		case 2:
			out = byte(int(pkt[i]) + (trans << 2))
		case 3:
			out = byte(int(pkt[i]) - (trans >> 5))
		}
		pkt[i] = out
		sumTransformed += out
		pos++
	}
	pkt[3] = sumTransformed - sumOriginal
	return pkt
}

// clientAccountLoginBody monta o corpo de MSG_AccountLogin com o layout pack(1)
// de Basedef.h:1546-1561: AccountPassword[12], AccountName[16], Zero[52],
// ClientVersion, DBNeedSave, AdapterName[4] = 104 bytes. O Enc da troca de canal
// vai em Zero, que é o campo comparado com o TempKey no dbServer
// (CFileDB.cpp:664-666).
func clientAccountLoginBody(account, password, enc string, version int32) []byte {
	const (
		passLen = 12
		nameLen = 16
		zeroLen = 52
	)
	b := make([]byte, 104)
	copy(b[0:passLen], password)
	copy(b[passLen:passLen+nameLen], account)
	copy(b[passLen+nameLen:passLen+nameLen+zeroLen], enc)
	binary.LittleEndian.PutUint32(b[80:84], uint32(version))
	return b
}

// clientEncChannel porta o consumo do Enc no cliente. O campo é `char Enc[52]`
// em MSG_DBCNFServerChange (Basedef.h:1369-1377), preenchido pelo sprintf de
// CFileDB.cpp:2009, e o cliente extrai o canal com o formato literal "*%d" — o
// literal está dentro do WYD.exe (extração em docs/migration/canais.md §8.1.2).
// Como sscanf("%d"), para no primeiro caractere que não é dígito.
func clientEncChannel(enc []byte) (int, bool) {
	s := string(enc)
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	if !strings.HasPrefix(s, "*") {
		return 0, false
	}
	n, digits := 0, 0
	for _, r := range s[1:] {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
		digits++
	}
	if digits == 0 {
		return 0, false
	}
	return n, true
}

// liveWhisper sends MSG_MessageWhisper the way the client does for both a
// command ("day", "srv") and a private message.
func liveWhisper(t *testing.T, c net.Conn, name, text string) {
	t.Helper()
	var wb protocol.MsgWhisperBody
	copy(wb.MobName[:], name)
	wb.String = []byte(text)
	wire, err := protocol.Encode(protocol.Header{Type: protocol.MsgMessageWhisper}, wb.Encode(), 11)
	if err != nil {
		t.Fatalf("encode whisper %q: %v", name, err)
	}
	if _, err := c.Write(wire); err != nil {
		t.Fatalf("write whisper %q: %v", name, err)
	}
}

// TestE2EClientReloginPacket is the end-to-end proof of the channel switch from
// the client side: the frame the server emits is parsed with the client's own
// rule, and the relogin packet built from the shared Basedef/CPSock sources is
// accepted by the destination tmServer, which puts the character back in the map
// and keeps answering world requests.
func TestE2EClientReloginPacket(t *testing.T) {
	addr1 := env("W2PP_E2E_ADDR", "127.0.0.1:8281")
	addr2 := env("W2PP_E2E_ADDR2", "127.0.0.1:8282")
	account := env("W2PP_E2E_ACCOUNT", "smoke")
	password := env("W2PP_E2E_PASSWORD", "smoke123")
	slot := e2eSlot(t)
	version := int32(protocol.AppVersion)
	if v := os.Getenv("W2PP_E2E_VERSION"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("W2PP_E2E_VERSION=%q: %v", v, err)
		}
		version = int32(n)
	}

	// 1. Canal 1: login e seleção (caminho já provado).
	c1 := liveLogin(t, addr1, account, password, "")
	defer c1.Close()
	liveSelectCharacter(t, c1, slot)

	// 2. /srv 2 e captura do Enc exatamente como o servidor o emitiu.
	liveWhisper(t, c1, "srv", "2")
	var enc []byte
	for i := 0; i < 100 && enc == nil; i++ {
		h, payload, ok := liveReadMaybe(c1, 5*time.Second)
		if !ok {
			break
		}
		if h.Type == protocol.MsgCNFServerChange {
			if len(payload) != protocol.MsgCNFServerChangeBodySize {
				t.Fatalf("MsgCNFServerChange body = %d, want %d", len(payload), protocol.MsgCNFServerChangeBodySize)
			}
			enc = append([]byte(nil), payload[16:]...)
		}
	}
	if enc == nil {
		t.Fatal("/srv 2: no MsgCNFServerChange frame")
	}

	// 3. (a) O parser do CLIENTE sobre o frame do servidor: canal = 2.
	channel, ok := clientEncChannel(enc)
	if !ok {
		t.Fatalf("client parse rejected Enc %q", strings.TrimRight(string(enc), "\x00"))
	}
	if channel != 2 {
		t.Fatalf("client parse read channel %d from %q, want 2", channel, strings.TrimRight(string(enc), "\x00"))
	}
	// (b) Os campos do sprintf legado, incluindo o slot, que é o que o
	// MSG_DBServerChange leva ao dbServer (CFileDB.cpp:1956-2009).
	w, ok := crossover.ParseWire(strings.TrimRight(string(enc), "\x00"))
	if !ok {
		t.Fatalf("Enc %q is outside the legacy grammar", strings.TrimRight(string(enc), "\x00"))
	}
	if w.Slot != slot {
		t.Fatalf("Enc slot = %d, want %d", w.Slot, slot)
	}

	// 4. O servidor de origem fecha a sessão (CloseUser, ProcessDBMessage.cpp:1476).
	closed := false
	for i := 0; i < 100; i++ {
		if _, _, ok := liveReadMaybe(c1, 3*time.Second); !ok {
			closed = true
			break
		}
	}
	if !closed {
		t.Fatal("origin session stayed open after /srv")
	}

	// 5. Relogin montado como o cliente monta: pacote CPSock com Enc em Zero[52].
	// A conta ainda pode estar salvando (AlreadyPlaying/StillPlaying) — o cliente
	// repete em conexão nova, então o teste repete igual.
	c2 := liveDial(t, addr2)
	login := cpsockFrame(0x2A, protocol.MsgAccountLogin, clientAccountLoginBody(account, "", string(enc), version))
	loggedIn := false
	for attempt := 0; attempt < 20; attempt++ {
		if _, err := c2.Write(login); err != nil {
			t.Fatalf("write client-framed login: %v", err)
		}
		h, payload, ok := liveReadMaybe(c2, 5*time.Second)
		if !ok {
			break
		}
		switch h.Type {
		case protocol.MsgCNFAccountLogin:
			if len(payload) < 1900 {
				t.Fatalf("CNFAccountLogin body too short: %d", len(payload))
			}
			loggedIn = true
		case protocol.MsgAlreadyPlaying, protocol.MsgStillPlaying:
			_ = c2.Close()
			time.Sleep(500 * time.Millisecond)
			c2 = liveDial(t, addr2)
			continue
		default:
			_ = c2.Close()
			t.Fatalf("client-framed login answered %#x, want CNFAccountLogin", h.Type)
		}
		break
	}
	if !loggedIn {
		t.Fatalf("client-framed login with the crossover Enc was not accepted by %s", addr2)
	}
	defer c2.Close()
	t.Logf("channel 2: client-framed login accepted (Enc channel=%d slot=%d)", channel, w.Slot)

	// 6. Seleção do personagem: o personagem volta ao mapa no canal 2.
	liveSelectCharacter(t, c2, slot)

	// 7. Resposta do mundo depois do relogin: um comando responde no painel.
	liveWhisper(t, c2, "day", "")
	gotDay := false
	for i := 0; i < 60; i++ {
		h, payload, ok := liveReadMaybe(c2, 5*time.Second)
		if !ok {
			break
		}
		if h.Type == protocol.MsgMessagePanel && h.ID == 0 && strings.TrimRight(string(payload), "\x00") == "!#11  2" {
			gotDay = true
			break
		}
	}
	if !gotDay {
		t.Fatal("channel 2: no world answer to /day after relogin")
	}

	// 8. Movimento depois do relogin: usa o tick que o servidor carimbou nos
	// frames recebidos, como o cliente faz (CPSock.cpp:541), e a sessão segue
	// respondendo — é o "funcional no canal de destino".
	move := protocol.MsgActionBody{PosX: 1000, PosY: 1000, TargetX: 1001, TargetY: 1000}
	moveWire, err := protocol.Encode(protocol.Header{Type: protocol.MsgAction}, move.Encode(), 7)
	if err != nil {
		t.Fatalf("encode movement: %v", err)
	}
	if _, err := c2.Write(moveWire); err != nil {
		t.Fatalf("write movement: %v", err)
	}
	liveWhisper(t, c2, "day", "")
	answered := false
	for i := 0; i < 60; i++ {
		h, payload, ok := liveReadMaybe(c2, 5*time.Second)
		if !ok {
			break
		}
		if h.Type == protocol.MsgMessagePanel && h.ID == 0 && strings.TrimRight(string(payload), "\x00") == "!#11  2" {
			answered = true
			break
		}
	}
	if !answered {
		t.Fatal("channel 2: session stopped answering after a movement frame")
	}
	t.Log("channel 2: character in the map, /day and movement both served")
}
