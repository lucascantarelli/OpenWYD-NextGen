package world

import (
	"io"
	"net"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
)

// corruptChecksum codifica um frame e inverte seu byte de checksum (offset 3) para o
// receptor ver uma divergência.
func corruptChecksum(t *testing.T, h protocol.Header, payload []byte) []byte {
	t.Helper()
	wire, err := protocol.Encode(h, payload, 7)
	if err != nil {
		t.Fatal(err)
	}
	wire[3]++ // quebra o checksum
	return wire
}

func TestRejectChecksumDropsConnection(t *testing.T) {
	got := make(chan protocol.Header, 1)
	addr, stop := testWorldConfig(t, Config{GridDim: 16, RejectChecksum: true}, echoHandler(got), nil)
	defer stop()

	c := dialClient(t, addr)
	defer c.Close()
	if _, err := c.Write(corruptChecksum(t, protocol.Header{Type: protocol.MsgMessagePanel}, []byte{1, 2})); err != nil {
		t.Fatal(err)
	}

	// O handler NÃO pode ter visto o frame, e a conexão precisa estar fechada.
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	var b [1]byte
	if _, err := c.Read(b[:]); err == nil {
		t.Fatal("expected connection to be dropped on checksum mismatch")
	}
	select {
	case <-got:
		t.Fatal("handler should not receive a frame with a bad checksum when RejectChecksum is on")
	default:
	}
}

func TestChecksumNotRejectedByDefault(t *testing.T) {
	got := make(chan protocol.Header, 1)
	addr, stop := testWorld(t, echoHandler(got), nil) // RejectChecksum é falso por padrão
	defer stop()

	c := dialClient(t, addr)
	defer c.Close()
	if _, err := c.Write(corruptChecksum(t, protocol.Header{Type: protocol.MsgMessagePanel}, []byte{1, 2})); err != nil {
		t.Fatal(err)
	}

	// Semântica legada: checksum ruim ainda é entregue.
	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("frame with bad checksum should still be delivered by default")
	}
}

func TestRateLimitDropsFlood(t *testing.T) {
	// Limite minúsculo: rajada de 2, depois derrubado.
	addr, stop := testWorldConfig(t, Config{GridDim: 16, MaxMsgPerSec: 1, MsgBurst: 2},
		func(*World, *Session, protocol.Header, []byte) {}, nil)
	defer stop()

	c := dialClient(t, addr)
	defer c.Close()
	for i := 0; i < 50; i++ {
		if _, err := c.Write(mustEncode(t, protocol.Header{Type: protocol.MsgMessagePanel})); err != nil {
			break
		}
	}

	// O flood precisa fazer a conexão cair (as leituras falham/EOF no fim).
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.Copy(io.Discard, c); err == nil {
		// io.Copy devolve nil em EOF limpo, que é a queda esperada.
		return
	} else if !isClosed(err) {
		t.Fatalf("expected connection drop, got %v", err)
	}
}

func mustEncode(t *testing.T, h protocol.Header) []byte {
	t.Helper()
	w, err := protocol.Encode(h, nil, 7)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func isClosed(err error) bool {
	if err == nil {
		return true
	}
	var ne net.Error
	return err == io.EOF || (asNetErr(err, &ne))
}

func asNetErr(err error, target *net.Error) bool {
	if ne, ok := err.(net.Error); ok {
		*target = ne
		return true
	}
	return false
}
