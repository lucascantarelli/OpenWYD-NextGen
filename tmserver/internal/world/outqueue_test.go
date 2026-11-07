package world

import (
	"testing"

	"openwyd-nextgen/tmserver/internal/protocol"
)

// createMobBody é um payload do tamanho de CreateMob (220 B no wire com o cabeçalho,
// o frame que inunda a fila quando o Coliseu entra na visão).
var createMobBody = make([]byte, 220-protocol.HeaderSize)

// TestEnqueueCreateMobBurstFits: um delta de visão de algumas centenas de CreateMob num
// tick (entrando no Coliseu, issue #419) precisa ser bufferizado, não descartado —
// o legado só falha no AddMessage acima de 128 KB (CPSock.cpp:513-520).
func TestEnqueueCreateMobBurstFits(t *testing.T) {
	w := New(Config{GridDim: 16}, slogDiscard(), nil, nil)
	s := &Session{Conn: 1, Mode: UserPlay, out: newOutQueue(), closeCh: make(chan struct{})}
	w.sessions[1] = s
	for i := 0; i < 400; i++ { // 400 × 220 B ≈ 86 KB
		w.enqueue(s, protocol.Header{Type: protocol.MsgCreateMob, ID: uint16(1000 + i)}, createMobBody)
	}
	if s.closed || w.sessions[1] != s {
		t.Fatal("a 400-CreateMob burst dropped the session; want it buffered")
	}
	if s.outHighWater != 400 || s.outHighWaterBytes != 400*220 {
		t.Fatalf("high water = %d frames / %d bytes, want 400 / %d", s.outHighWater, s.outHighWaterBytes, 400*220)
	}
}

// TestEnqueueDropsPastByteLimitOnce: um cliente que para de drenar é derrubado
// quando seus bytes não enviados passam do limite, e envios posteriores no mesmo tick são
// no-op na sessão morta em vez de derrubá-la de novo.
func TestEnqueueDropsPastByteLimitOnce(t *testing.T) {
	w := New(Config{GridDim: 16, OutBufferBytes: 10 * 220}, slogDiscard(), nil, nil)
	s := &Session{Conn: 1, Mode: UserPlay, out: newOutQueue(), closeCh: make(chan struct{})}
	w.sessions[1] = s
	for i := 0; i < 10; i++ {
		w.enqueue(s, protocol.Header{Type: protocol.MsgCreateMob}, createMobBody)
	}
	if s.closed {
		t.Fatal("dropped at exactly the limit; want the limit itself to fit")
	}
	w.enqueue(s, protocol.Header{Type: protocol.MsgCreateMob}, createMobBody)
	if !s.closed || w.sessions[1] != nil {
		t.Fatal("the frame past the byte limit did not drop the session")
	}
	sent := s.sentFrames
	for i := 0; i < 5; i++ {
		w.enqueue(s, protocol.Header{Type: protocol.MsgRemoveMob}, nil)
	}
	if s.sentFrames != sent {
		t.Fatalf("enqueue on a dropped session still counted sends (%d → %d)", sent, s.sentFrames)
	}
}

// TestOutQueuePopReleasesBytes: o escritor retirando um frame libera seus bytes, então um
// cliente que acompanha nunca atinge o limite por mais que receba ao longo do tempo.
func TestOutQueuePopReleasesBytes(t *testing.T) {
	q := newOutQueue()
	for i := 0; i < 1000; i++ {
		if _, _, ok := q.push(outFrame{payload: createMobBody}, 2*220); !ok {
			t.Fatalf("push %d refused with the writer keeping up", i)
		}
		if _, ok := q.pop(); !ok {
			t.Fatalf("pop %d found the queue empty", i)
		}
	}
	if q.bytes != 0 || len(q.frames) != 0 {
		t.Fatalf("drained queue holds %d frames / %d bytes, want 0 / 0", len(q.frames), q.bytes)
	}
}
