package world

import (
	"strings"
	"testing"

	"openwyd-nextgen/tmserver/internal/protocol"
)

// TestNoteSendTracking: todo frame enfileirado precisa cair nos contadores e no
// anel, e o anel precisa manter só os últimos lastSentRing frames em ordem —
// essa cauda é o post-mortem do travamento, então sua ordem é fundamental.
func TestNoteSendTracking(t *testing.T) {
	s := &Session{Conn: 7}
	for i := 0; i < lastSentRing+3; i++ {
		s.noteSend(protocol.Header{Type: protocol.MsgAction, ID: uint16(i)}, 28)
	}
	s.noteSend(protocol.Header{Type: protocol.MsgCreateMob, ID: 30000}, 504)

	if want := uint64(lastSentRing + 4); s.sentFrames != want {
		t.Fatalf("sentFrames = %d, want %d", s.sentFrames, want)
	}
	wantBytes := uint64((lastSentRing+3)*(28+protocol.HeaderSize) + 504 + protocol.HeaderSize)
	if s.sentBytes != wantBytes {
		t.Fatalf("sentBytes = %d, want %d", s.sentBytes, wantBytes)
	}
	if n := s.sentByType[protocol.MsgAction]; n != uint32(lastSentRing+3) {
		t.Fatalf("sentByType[MsgAction] = %d, want %d", n, lastSentRing+3)
	}
	if n := s.sentByType[protocol.MsgCreateMob]; n != 1 {
		t.Fatalf("sentByType[MsgCreateMob] = %d, want 1", n)
	}

	tail := formatLastSends(&s.lastSent, s.lastSentIdx, s.lastSent[(s.lastSentIdx-1)%lastSentRing].at)
	records := strings.Fields(tail)
	if len(records) != lastSentRing {
		t.Fatalf("tail records = %d, want %d (full ring)", len(records), lastSentRing)
	}
	// Entrada mais nova por último: o CreateMob com seu HEADER.ID e tamanho de payload.
	if last := records[len(records)-1]; last != "0x0364(30000)/504/-0ms" {
		t.Fatalf("newest tail record = %q, want %q", last, "0x0364(30000)/504/-0ms")
	}
	// Entrada mais antiga sobrevivente é o frame #4 (três MsgAction foram sobrescritos pelo
	// giro e um pelo CreateMob).
	if first := records[0]; !strings.HasPrefix(first, "0x036c(4)/28/") {
		t.Fatalf("oldest tail record = %q, want prefix %q", first, "0x036c(4)/28/")
	}
}

// TestFormatByTypeOrdering: a quebra por tipo precisa ler do dominante primeiro para
// o post-mortem de desconexão abrir com o tráfego que importava.
func TestFormatByTypeOrdering(t *testing.T) {
	got := formatByType(map[protocol.Type]uint32{
		protocol.MsgAction:    3,
		protocol.MsgCreateMob: 12,
		protocol.MsgRemoveMob: 3,
	})
	want := "0x0364:12 0x0165:3 0x036c:3"
	if got != want {
		t.Fatalf("formatByType = %q, want %q", got, want)
	}
}

// TestEnqueueHighWater: enqueue precisa acompanhar a marca d'água da fila de saída
// (evidência de contrapressão do escritor) sem nunca bloquear o loop.
func TestEnqueueHighWater(t *testing.T) {
	w := New(Config{}, nil, nil, nil)
	s := &Session{Conn: 1, out: newOutQueue(), closeCh: make(chan struct{})}
	w.sessions[1] = s
	for i := 0; i < 5; i++ {
		w.enqueue(s, protocol.Header{Type: protocol.MsgAction}, nil)
	}
	if s.outHighWater != 5 {
		t.Fatalf("outHighWater = %d, want 5", s.outHighWater)
	}
	if s.sentFrames != 5 {
		t.Fatalf("sentFrames = %d, want 5", s.sentFrames)
	}
}
