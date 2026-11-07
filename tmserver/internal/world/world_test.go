package world

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
)

func slogDiscard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// testWorld inicia um World servindo num listener de loopback e devolve seu
// endereço e uma função de cancelamento. GridDim é minúsculo para evitar a grade densa inteira.
func testWorld(t *testing.T, h Handler, persist Persistence) (addr string, stop func()) {
	return testWorldConfig(t, Config{GridDim: 16}, h, persist)
}

// testWorldConfig é o testWorld com um Config explícito (para testes de endurecimento).
func testWorldConfig(t *testing.T, cfg Config, h Handler, persist Persistence) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if persist == nil {
		persist = NopPersistence{}
	}
	if cfg.GridDim == 0 {
		cfg.GridDim = 16
	}
	w := New(cfg, slogDiscard(), persist, h)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	return ln.Addr().String(), func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("Serve did not stop")
		}
	}
}

// dialClient conecta e envia o handshake INITCODE obrigatório.
func dialClient(t *testing.T, addr string) net.Conn {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	var ic [4]byte
	binary.LittleEndian.PutUint32(ic[:], protocol.InitCode)
	if _, err := c.Write(ic[:]); err != nil {
		t.Fatal(err)
	}
	return c
}

func sendFrame(t *testing.T, c net.Conn, h protocol.Header, payload []byte) {
	t.Helper()
	wire, err := protocol.Encode(h, payload, 7)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write(wire); err != nil {
		t.Fatal(err)
	}
}

// readFrame lê um frame S→C (sem INITCODE nesta direção) e o decodifica.
func readFrame(t *testing.T, c net.Conn) (protocol.Header, []byte) {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	var sz [2]byte
	if _, err := io.ReadFull(c, sz[:]); err != nil {
		t.Fatalf("read size: %v", err)
	}
	size := int(binary.LittleEndian.Uint16(sz[:]))
	buf := make([]byte, size)
	copy(buf, sz[:])
	if _, err := io.ReadFull(c, buf[2:]); err != nil {
		t.Fatalf("read body: %v", err)
	}
	h, payload, _, err := protocol.Decode(buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return h, payload
}

// echoHandler responde a todo frame do cliente com um MessagePanel ecoando o
// payload, e registra o que recebeu.
func echoHandler(got chan<- protocol.Header) Handler {
	return func(w *World, s *Session, h protocol.Header, payload []byte) {
		got <- h
		w.Send(s, protocol.MsgMessagePanel, payload)
	}
}

func TestHeadlessConnectAndExchange(t *testing.T) {
	got := make(chan protocol.Header, 4)
	addr, stop := testWorld(t, echoHandler(got), nil)
	defer stop()

	c := dialClient(t, addr)
	defer c.Close()

	sendFrame(t, c, protocol.Header{Type: protocol.MsgAccountLogin, ID: 0}, []byte("hello-world"))

	select {
	case h := <-got:
		if h.Type != protocol.MsgAccountLogin {
			t.Errorf("handler saw Type %#x, want AccountLogin", h.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler never received the frame")
	}

	rh, rp := readFrame(t, c)
	if rh.Type != protocol.MsgMessagePanel {
		t.Errorf("response Type = %#x, want MessagePanel", rh.Type)
	}
	if string(rp) != "hello-world" {
		t.Errorf("response payload = %q, want hello-world", rp)
	}
	// HEADER.ID no S→C precisa ser o conn da sessão (1 para o primeiro cliente; conn 0
	// é reservado).
	if rh.ID != 1 {
		t.Errorf("response ID = %d, want 1", rh.ID)
	}
}

func TestPingAndSkipTickIgnored(t *testing.T) {
	got := make(chan protocol.Header, 4)
	addr, stop := testWorld(t, echoHandler(got), nil)
	defer stop()
	c := dialClient(t, addr)
	defer c.Close()

	// Ping não faz nada; frames SkipCheckTick são descartados (protocol-spec.md §2).
	sendFrame(t, c, protocol.Header{Type: protocol.MsgPing}, nil)
	sendFrame(t, c, protocol.Header{Type: protocol.MsgAction, ClientTick: protocol.SkipCheckTick}, nil)
	// Um frame normal depois deles ainda precisa ser entregue.
	sendFrame(t, c, protocol.Header{Type: protocol.MsgAction}, nil)

	select {
	case h := <-got:
		if h.Type != protocol.MsgAction || h.ClientTick == protocol.SkipCheckTick {
			t.Errorf("first delivered frame = %+v; ping/skip-tick should have been dropped", h)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no frame delivered")
	}
}

// TestConcurrentConnections é a verificação de concorrência com -race: muitos clientes martelam
// o loop ao mesmo tempo; a goroutine dona única é a única mutadora de estado, então
// não pode haver data race e todo cliente precisa ser atendido.
func TestConcurrentConnections(t *testing.T) {
	const n = 64
	var served atomic.Int64
	h := func(w *World, s *Session, _ protocol.Header, payload []byte) {
		served.Add(1)
		w.Send(s, protocol.MsgMessagePanel, payload)
	}
	addr, stop := testWorld(t, h, nil)
	defer stop()

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := dialClient(t, addr)
			defer c.Close()
			sendFrame(t, c, protocol.Header{Type: protocol.MsgAction}, []byte{byte(i)})
			readFrame(t, c) // espera o eco
		}(i)
	}
	wg.Wait()
	if served.Load() != n {
		t.Errorf("served %d connections, want %d", served.Load(), n)
	}
}

func TestGracefulShutdownSavesPlayers(t *testing.T) {
	fake := &fakePersistence{}
	ready := make(chan struct{}, 1)
	h := func(_ *World, s *Session, _ protocol.Header, _ []byte) {
		s.Mode = UserPlay
		s.AccountID = 42
		select {
		case ready <- struct{}{}:
		default:
		}
	}
	addr, stop := testWorld(t, h, fake)

	c := dialClient(t, addr)
	defer c.Close()
	sendFrame(t, c, protocol.Header{Type: protocol.MsgAction}, nil)
	<-ready // sessão agora está em UserPlay

	stop() // cancela ctx → drenagem graciosa
	if n := fake.saved.Load(); n != 1 {
		t.Errorf("SaveOnShutdown called %d times, want 1", n)
	}
}

type fakePersistence struct {
	NopPersistence
	saved atomic.Int64
}

func (f *fakePersistence) SaveOnShutdown(context.Context, CharacterSave) error {
	f.saved.Add(1)
	return nil
}

// cargoCapture registra o último SaveCargo que recebeu (ReleaseCargo salva fora do
// loop numa goroutine, então o acesso é protegido por mutex).
type cargoCapture struct {
	NopPersistence
	mu    sync.Mutex
	saves []CargoSave
}

func (c *cargoCapture) SaveCargo(_ context.Context, save CargoSave) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.saves = append(c.saves, save)
	return nil
}

func (c *cargoCapture) last() (CargoSave, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.saves) == 0 {
		return CargoSave{}, 0
	}
	return c.saves[len(c.saves)-1], len(c.saves)
}

// TestCargoLifecycle cobre o armazém com escopo de conta: instalar via
// SetCargo, ler via Cargo, snapshot via cargoSave, e persistir+despejar via
// ReleaseCargo. Rodam num World não iniciado (auxiliares só-no-loop tocam apenas o
// mapa e a porta de persistência).
func TestCargoLifecycle(t *testing.T) {
	capture := &cargoCapture{}
	w := New(Config{GridDim: 16}, slogDiscard(), capture, nil)

	// Conta desconhecida → nil; a conta 0 nunca é guardada.
	if w.Cargo(42) != nil || w.Cargo(0) != nil {
		t.Fatalf("expected nil cargo before load")
	}

	st := &CargoState{Coin: 1000}
	st.Items[3] = Item{Index: 1234, Effects: [3]Effect{{Effect: 9, Value: 1}}}
	w.SetCargo(42, st)

	got := w.Cargo(42)
	if got == nil || got.AccountID != 42 || got.Coin != 1000 || got.Items[3].Index != 1234 {
		t.Fatalf("Cargo after SetCargo: %+v", got)
	}

	// Uma mutação do handler (depósito) se reflete porque Cargo devolve o ponteiro.
	got.Coin += 500

	snap := w.cargoSave(42)
	if snap.AccountID != 42 || snap.Coin != 1500 || len(snap.Items) != 1 || snap.Items[0].Slot != 3 {
		t.Fatalf("cargoSave snapshot: %+v", snap)
	}

	// ReleaseCargo persiste e então despeja; o save roda fora do loop, drenado por saveWG.
	w.ReleaseCargo(42)
	w.saveWG.Wait()
	if w.Cargo(42) != nil {
		t.Fatalf("cargo not evicted after ReleaseCargo")
	}
	last, n := capture.last()
	if n != 1 || last.Coin != 1500 || last.AccountID != 42 {
		t.Fatalf("ReleaseCargo did not persist latest state: %+v (n=%d)", last, n)
	}

	// Liberar uma conta desconhecida não faz nada (sem save extra).
	w.ReleaseCargo(42)
	w.saveWG.Wait()
	if _, n := capture.last(); n != 1 {
		t.Fatalf("ReleaseCargo of evicted account saved again: n=%d", n)
	}
}
