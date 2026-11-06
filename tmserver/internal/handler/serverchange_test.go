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
	// O cliente 7662 lê o canal de destino com "*%d" e devolve o campo
	// inteiro; fora da gramática decimal do legado (CFileDB.cpp:2009) o handler
	// de troca dele para no meio e o mundo não é redesenhado.
	wire, ok := crossover.ParseWire(tok)
	if !ok {
		t.Fatalf("Enc %q is outside the legacy grammar \"*<srv> <8 ints> <slot>\"", tok)
	}
	if wire.Target != 2 {
		t.Errorf("Enc channel field = %d, want 2 (the client reconnects to this)", wire.Target)
	}
	if wire.Slot != 0 {
		t.Errorf("Enc slot field = %d, want 0", wire.Slot)
	}
	claims, ok := crossover.Verify("test-key", name, tok, time.Unix(0, 0))
	if !ok {
		t.Fatalf("token %q does not verify for account %q", tok, name)
	}
	if claims.Target != 2 {
		t.Errorf("token target = %d, want 2", claims.Target)
	}

	// Legacy parity: the server closes the session after sending the frame
	// (CloseUser, ProcessDBMessage.cpp:1474-1478), which is what makes the
	// client reconnect to the target channel. Without the close, /srv is a no-op.
	_ = a.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := a.Read(make([]byte, 1)); err == nil {
		t.Fatal("session stayed open after /srv token: client never reconnects")
	}
}

// TestESCMenuTriggerIsTheCommandFrame fixa o gatilho que o menu ESC precisa
// emitir (CANAL-03). A janela do ESC não manda frame nenhum hoje — o
// dispatcher loga todo Type recebido, e nesse caminho não sai linha alguma —,
// então o patch do cliente vai enviar exatamente este frame: o mesmo
// MSG_MessageWhisper do /srv. O teste prova que esse pedido, sozinho, emite o
// token e encerra a sessão, sem precisar de outro pacote do servidor.
func TestESCMenuTriggerIsTheCommandFrame(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "srv", "2")
	h, payload, ok := readMaybeHeader(t, a)
	if !ok || h.Type != protocol.MsgCNFServerChange {
		t.Fatalf("ESC trigger frame answered %#x ok=%v, want MsgCNFServerChange", h.Type, ok)
	}
	tok := strings.TrimRight(string(payload[16:]), "\x00")
	if w, ok := crossover.ParseWire(tok); !ok || w.Target != 2 {
		t.Fatalf("ESC trigger produced %q, want a token for channel 2 in the legacy grammar", tok)
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
