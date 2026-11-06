package handler

import (
	"strings"
	"testing"
	"time"

	"openwyd-nextgen/internal/crossover"
	"openwyd-nextgen/tmserver/internal/protocol"
)

// TestCommandServerChangeIssuesToken covers "/srv 2" end to end (canais.md
// §8.1.2): the command must answer MSG_DBCNFServerChange carrying this account's
// name and a token that verifies for the requested channel — the frame the client
// uses to reconnect to the other server.
func TestCommandServerChangeIssuesToken(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "srv", "2")
	h, payload, ok := readMaybeHeader(t, a)
	if !ok {
		t.Fatal("no frame answered /srv")
	}
	if h.Type != protocol.MsgCNFServerChange {
		t.Fatalf("type = %#x, want MsgCNFServerChange (%#x)", h.Type, protocol.MsgCNFServerChange)
	}
	if len(payload) != protocol.MsgCNFServerChangeBodySize {
		t.Fatalf("body = %d bytes, want %d", len(payload), protocol.MsgCNFServerChangeBodySize)
	}
	name := strings.TrimRight(string(payload[:16]), "\x00")
	if name != "tester" {
		t.Errorf("account = %q, want \"tester\"", name)
	}
	tok := strings.TrimRight(string(payload[16:]), "\x00")
	claims, ok := crossover.Verify("test-key", name, tok, time.Unix(0, 0))
	if !ok {
		t.Fatalf("token %q does not verify for account %q", tok, name)
	}
	if claims.Target != 2 {
		t.Errorf("token target = %d, want 2", claims.Target)
	}
	if claims.Slot != 0 {
		t.Errorf("token slot = %d, want 0", claims.Slot)
	}
}

// TestCommandServerChangeSameChannel keeps the legacy guard: asking for the
// channel you are already on must answer a notice, never a crossover token.
func TestCommandServerChangeSameChannel(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "srv", "1") // serverIndex 0 == channel 1
	ty, _, ok := readMaybeHeader(t, a)
	if !ok {
		t.Fatal("no frame answered /srv 1")
	}
	if ty.Type == protocol.MsgCNFServerChange {
		t.Fatal("same-channel /srv issued a crossover token")
	}
}
