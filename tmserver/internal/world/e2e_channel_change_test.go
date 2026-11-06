//go:build e2e

// End-to-end channel change against the LIVE stack: login + character selection
// on channel 1, "/srv 2", the MSG_CNFServerChange token, the server closing the
// session, and a second login on channel 2 carrying that token in Zero[52].
// Mirrors the legacy flow (canais.md §8.1.2, CFileDB.cpp:662-812).
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

	"openwyd-nextgen/tmserver/internal/protocol"
)

// liveReadMaybe is liveRead without Fatal: it reports false on timeout/EOF so a
// test can drain the login snapshot until the frame it wants (or the close).
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

// liveLogin performs one MSG_AccountLogin, putting token in Zero[52] when it is
// a channel-change token. A save in flight after the previous session closes
// answers AlreadyPlaying/StillPlaying; that is retried on a fresh connection,
// exactly as the client does.
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

// liveSelectCharacter sends MSG_CharacterLogin and waits for the in-world
// snapshot (MSG_CNFCharacterLogin), skipping the view frames that precede it.
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

	// 1. Channel 1: real login + select character.
	c1 := liveLogin(t, addr1, account, password, "")
	defer c1.Close()
	liveSelectCharacter(t, c1, slot)
	t.Logf("channel 1: logged in and selected slot %d", slot)

	// 2. /srv 2, sent the way the client sends a GM/player command.
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

	// The client drains whatever the login snapshot left queued (score, PK and
	// view frames) until the token frame arrives.
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
	if !strings.HasPrefix(token, "~x1:") {
		t.Fatalf("token %q does not carry the crossover prefix", token)
	}
	t.Logf("channel 1: /srv 2 issued token (%d bytes)", len(token))

	// 3. Legacy parity: the server must close the session so the client
	// reconnects; without this the command is a no-op in the graphic client.
	// Drain anything still queued; the next read must fail (EOF/timeout).
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

	// 4. Channel 2: login carrying the token in Zero[52], then select the same
	// character. This is the proof the token is accepted across instances.
	c2 := liveLogin(t, addr2, account, "", token)
	defer c2.Close()
	liveSelectCharacter(t, c2, slot)
	t.Logf("channel 2: token login OK, selected slot %d", slot)
}

// TestE2EServerChangeSameChannel confirms the guard answers a notice (not a
// token) when the character is already on the requested channel.
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
	// Drain the rest of the login snapshot; a same-channel /srv answers a
	// notice, never a token.
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
