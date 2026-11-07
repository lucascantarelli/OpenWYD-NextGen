//go:build e2e

// Troca de canal ponta a ponta contra a stack VIVA: login + seleção de personagem
// no canal 1, "/srv 2", o token MSG_CNFServerChange, o servidor fechando a
// sessão, e um segundo login no canal 2 carregando esse token em Zero[52].
// Espelha o fluxo legado (canais.md §8.1.2, CFileDB.cpp:662-812).
//
//	W2PP_E2E_ADDR=tmserver:8281 W2PP_E2E_ADDR2=tmserver2:8282 \
//	W2PP_E2E_ACCOUNT=test W2PP_E2E_PASSWORD=test123 W2PP_E2E_SLOT=0 \
//	  go test -tags=e2e -run TestE2EChannelChange ./tmserver/internal/world/ -v
package world

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"openwyd-nextgen/internal/crossover"
	"openwyd-nextgen/tmserver/internal/protocol"
)

// liveReadMaybe é o liveRead sem Fatal: reporta false em timeout/EOF para um
// teste poder drenar o snapshot de login até o frame que quer (ou o fechamento).
func liveReadMaybe(c net.Conn, timeout time.Duration) (protocol.Header, []byte, bool) {
	_ = c.SetReadDeadline(time.Now().Add(timeout))
	var sz [2]byte
	if _, err := io.ReadFull(c, sz[:]); err != nil {
		return protocol.Header{}, nil, false
	}
	size := int(binary.LittleEndian.Uint16(sz[:]))
	buf := make([]byte, size)
	copy(buf, sz[:])
	if _, err := io.ReadFull(c, buf[2:]); err != nil {
		return protocol.Header{}, nil, false
	}
	h, payload, _, err := protocol.Decode(buf)
	if err != nil {
		return protocol.Header{}, nil, false
	}
	return h, payload, true
}

func e2eSlot(t *testing.T) int {
	t.Helper()
	v := os.Getenv("W2PP_E2E_SLOT")
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		t.Fatalf("W2PP_E2E_SLOT=%q: %v", v, err)
	}
	return n
}

// liveLogin executa um MSG_AccountLogin, pondo token em Zero[52] quando ele é
// um token de troca de canal. Um save em andamento depois que a sessão anterior fecha
// responde AlreadyPlaying/StillPlaying; isso é retentado numa conexão nova,
// exatamente como o cliente faz.
func liveLogin(t *testing.T, addr, account, password, token string) net.Conn {
	t.Helper()
	for attempt := 0; attempt < 20; attempt++ {
		c := liveDial(t, addr)
		if _, err := c.Write(loginFrame(account, password, token)); err != nil {
			_ = c.Close()
			t.Fatalf("write login: %v", err)
		}
		h, payload := liveRead(t, c)
		switch h.Type {
		case protocol.MsgCNFAccountLogin:
			if len(payload) < 1900 {
				_ = c.Close()
				t.Fatalf("CNFAccountLogin body too short: %d", len(payload))
			}
			return c
		case protocol.MsgAlreadyPlaying, protocol.MsgStillPlaying:
			_ = c.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		default:
			_ = c.Close()
			t.Fatalf("login on %s: type %#x, want CNFAccountLogin", addr, h.Type)
		}
	}
	t.Fatalf("login on %s: account still saving after retries", addr)
	return nil
}

// liveSelectCharacter envia MSG_CharacterLogin e espera o snapshot
// em jogo (MSG_CNFCharacterLogin), pulando os frames de visão que o precedem.
func liveSelectCharacter(t *testing.T, c net.Conn, slot int) {
	t.Helper()
	body := protocol.MsgCharacterLoginBody{Slot: int32(slot)}
	wire, err := protocol.Encode(protocol.Header{Type: protocol.MsgCharacterLogin}, body.Encode(), 9)
	if err != nil {
		t.Fatalf("encode character login: %v", err)
	}
	if _, err := c.Write(wire); err != nil {
		t.Fatalf("write character login: %v", err)
	}
	for i := 0; i < 20; i++ {
		h, _ := liveRead(t, c)
		switch h.Type {
		case protocol.MsgCNFCharacterLogin:
			return
		case protocol.MsgCharacterLoginFail:
			t.Fatalf("slot %d refused with MsgCharacterLoginFail", slot)
		}
	}
	t.Fatalf("slot %d: no MsgCNFCharacterLogin", slot)
}

func TestE2EChannelChange(t *testing.T) {
	addr1 := env("W2PP_E2E_ADDR", "127.0.0.1:8281")
	addr2 := env("W2PP_E2E_ADDR2", "127.0.0.1:8282")
	account := env("W2PP_E2E_ACCOUNT", "smoke")
	password := env("W2PP_E2E_PASSWORD", "smoke123")
	slot := e2eSlot(t)

	// 1. Canal 1: login real + selecionar personagem.
	c1 := liveLogin(t, addr1, account, password, "")
	defer c1.Close()
	liveSelectCharacter(t, c1, slot)
	t.Logf("channel 1: logged in and selected slot %d", slot)

	// 2. /srv 2, enviado como o cliente envia um comando de GM/jogador.
	var wb protocol.MsgWhisperBody
	copy(wb.MobName[:], "srv")
	wb.String = []byte("2")
	wire, err := protocol.Encode(protocol.Header{Type: protocol.MsgMessageWhisper}, wb.Encode(), 11)
	if err != nil {
		t.Fatalf("encode /srv: %v", err)
	}
	if _, err := c1.Write(wire); err != nil {
		t.Fatalf("write /srv 2: %v", err)
	}

	// O cliente drena o que o snapshot de login deixou na fila (score, PK e
	// frames de visão) até o frame do token chegar.
	var h protocol.Header
	var payload []byte
	found := false
	for i := 0; i < 100; i++ {
		var ok bool
		h, payload, ok = liveReadMaybe(c1, 5*time.Second)
		if !ok {
			break
		}
		if h.Type == protocol.MsgCNFServerChange {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("/srv 2: no MsgCNFServerChange (%#x) in the drained frames", protocol.MsgCNFServerChange)
	}
	if len(payload) != protocol.MsgCNFServerChangeBodySize {
		t.Fatalf("MsgCNFServerChange body = %d, want %d", len(payload), protocol.MsgCNFServerChangeBodySize)
	}
	gotAccount := strings.TrimRight(string(payload[:16]), "\x00")
	token := strings.TrimRight(string(payload[16:]), "\x00")
	if gotAccount != account {
		t.Fatalf("frame account = %q, want %q", gotAccount, account)
	}
	// O campo que o cliente parseia: dez inteiros decimais, o primeiro é o
	// canal de destino (CFileDB.cpp:2009, formato "*%d" no WYD.exe).
	encWire, ok := crossover.ParseWire(token)
	if !ok {
		t.Fatalf("token %q is outside the client grammar \"*<srv> <8 ints> <slot>\"", token)
	}
	if encWire.Target != 2 {
		t.Fatalf("token channel field = %d, want 2", encWire.Target)
	}
	if encWire.Slot != slot {
		t.Fatalf("token slot field = %d, want %d", encWire.Slot, slot)
	}
	t.Logf("channel 1: /srv 2 issued token %q (%d bytes)", token, len(token))

	// 3. Paridade legada: o servidor precisa fechar a sessão para o cliente
	// reconectar; sem isso o comando é no-op no cliente gráfico.
	// Drenar tudo que ainda está na fila; a próxima leitura precisa falhar (EOF/timeout).
	closed := false
	for i := 0; i < 100; i++ {
		if _, _, ok := liveReadMaybe(c1, 3*time.Second); !ok {
			closed = true
			break
		}
	}
	if !closed {
		t.Fatal("session stayed open after /srv: client never reconnects")
	}

	// 4. Canal 2: login carregando o token em Zero[52], depois selecionar o mesmo
	// personagem. Esta é a prova de que o token é aceito entre instâncias.
	c2 := liveLogin(t, addr2, account, "", token)
	defer c2.Close()
	liveSelectCharacter(t, c2, slot)
	t.Logf("channel 2: token login OK, selected slot %d", slot)
}

// TestE2EChannelChangeBack é o caminho inverso do TestE2EChannelChange: um
// personagem no canal 2 roda "/srv 1", recebe um token cujo prefixo legado
// de wire nomeia o canal 1, e loga de volta no canal 1 com esse token.
func TestE2EChannelChangeBack(t *testing.T) {
	addr1 := env("W2PP_E2E_ADDR", "127.0.0.1:8281")
	addr2 := env("W2PP_E2E_ADDR2", "127.0.0.1:8282")
	account := env("W2PP_E2E_ACCOUNT", "smoke")
	password := env("W2PP_E2E_PASSWORD", "smoke123")
	slot := e2eSlot(t)

	c2 := liveLogin(t, addr2, account, password, "")
	defer c2.Close()
	liveSelectCharacter(t, c2, slot)
	t.Logf("channel 2: logged in and selected slot %d", slot)

	var wb protocol.MsgWhisperBody
	copy(wb.MobName[:], "srv")
	wb.String = []byte("1")
	wire, err := protocol.Encode(protocol.Header{Type: protocol.MsgMessageWhisper}, wb.Encode(), 12)
	if err != nil {
		t.Fatalf("encode /srv: %v", err)
	}
	if _, err := c2.Write(wire); err != nil {
		t.Fatalf("write /srv 1: %v", err)
	}

	var token string
	for i := 0; i < 100; i++ {
		h, payload, ok := liveReadMaybe(c2, 5*time.Second)
		if !ok {
			break
		}
		if h.Type == protocol.MsgCNFServerChange {
			if len(payload) != protocol.MsgCNFServerChangeBodySize {
				t.Fatalf("MsgCNFServerChange body = %d, want %d", len(payload), protocol.MsgCNFServerChangeBodySize)
			}
			token = strings.TrimRight(string(payload[16:]), "\x00")
			break
		}
	}
	if token == "" {
		t.Fatal("/srv 1 from channel 2: no MsgCNFServerChange token")
	}
	encWire, ok := crossover.ParseWire(token)
	if !ok {
		t.Fatalf("token %q is outside the client grammar \"*<srv> <8 ints> <slot>\"", token)
	}
	if encWire.Target != 1 {
		t.Fatalf("token channel field = %d, want 1", encWire.Target)
	}
	t.Logf("channel 2: /srv 1 issued token %q (%d bytes)", token, len(token))

	c1 := liveLogin(t, addr1, account, "", token)
	defer c1.Close()
	liveSelectCharacter(t, c1, slot)
	t.Logf("channel 1: token login OK, selected slot %d", slot)
}

// TestE2EServerChangeSameChannel confirma que a guarda responde um aviso (não um
// token) quando o personagem já está no canal pedido.
func TestE2EServerChangeSameChannel(t *testing.T) {
	addr := env("W2PP_E2E_ADDR", "127.0.0.1:8281")
	account := env("W2PP_E2E_ACCOUNT", "smoke")
	password := env("W2PP_E2E_PASSWORD", "smoke123")
	slot := e2eSlot(t)

	c := liveLogin(t, addr, account, password, "")
	defer c.Close()
	liveSelectCharacter(t, c, slot)

	var wb protocol.MsgWhisperBody
	copy(wb.MobName[:], "srv")
	wb.String = []byte("1")
	wire, err := protocol.Encode(protocol.Header{Type: protocol.MsgMessageWhisper}, wb.Encode(), 12)
	if err != nil {
		t.Fatalf("encode /srv: %v", err)
	}
	if _, err := c.Write(wire); err != nil {
		t.Fatalf("write /srv 1: %v", err)
	}
	// Drenar o resto do snapshot de login; um /srv no mesmo canal responde um
	// aviso, nunca um token.
	last := protocol.Type(0)
	for i := 0; i < 100; i++ {
		h, _, ok := liveReadMaybe(c, 3*time.Second)
		if !ok {
			break
		}
		last = h.Type
		if h.Type == protocol.MsgCNFServerChange {
			t.Fatal("same-channel /srv issued a crossover token")
		}
	}
	t.Logf("same-channel /srv: no token (last frame type %#x)", last)
}
