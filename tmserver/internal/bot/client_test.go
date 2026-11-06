package bot

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
)

// fakeServer sobe um CPSock de mentira que exige INITCODE e responde login e
// seleção de personagem. Aceita várias conexões simultâneas, como o servidor
// real, e devolve o endereço e um canal com os tipos recebidos.
func fakeServer(t *testing.T) (addr string, received chan protocol.Type, closeFn func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ch := make(chan protocol.Type, 128)
	var wg sync.WaitGroup

	serve := func(conn net.Conn) {
		defer wg.Done()
		defer conn.Close()

		var ic [4]byte
		if _, err := io.ReadFull(conn, ic[:]); err != nil {
			return
		}
		if binary.LittleEndian.Uint32(ic[:]) != protocol.InitCode {
			t.Errorf("servidor fake recebeu INITCODE %#x", binary.LittleEndian.Uint32(ic[:]))
			return
		}

		reply := func(typ protocol.Type, payload []byte, id uint16) {
			wire, err := protocol.Encode(protocol.Header{Type: typ, ID: id}, payload, 7)
			if err != nil {
				return
			}
			_, _ = conn.Write(wire)
		}

		for {
			var sz [2]byte
			if _, err := io.ReadFull(conn, sz[:]); err != nil {
				return
			}
			size := int(binary.LittleEndian.Uint16(sz[:]))
			if size < protocol.HeaderSize {
				return
			}
			buf := make([]byte, size)
			copy(buf, sz[:])
			if _, err := io.ReadFull(conn, buf[2:]); err != nil {
				return
			}
			h, payload, _, err := protocol.Decode(buf)
			if err != nil {
				return
			}
			select {
			case ch <- h.Type:
			default:
			}

			switch h.Type {
			case protocol.MsgAccountLogin:
				var body protocol.MsgAccountLoginBody
				if err := body.Decode(payload); err != nil {
					return
				}
				switch trimNUL(body.AccountName[:]) {
				case "blocked":
					reply(protocol.MsgMessagePanel, make([]byte, 100), 0)
				case "busy":
					reply(protocol.MsgAlreadyPlaying, nil, 0)
				default:
					reply(protocol.MsgCNFAccountLogin, make([]byte, minAccountListBody), 0)
				}
			case protocol.MsgCharacterLogin:
				var body protocol.MsgCharacterLoginBody
				if err := body.Decode(payload); err != nil {
					return
				}
				if body.Slot == 9 {
					reply(protocol.MsgCharacterLoginFail, nil, 0)
					continue
				}
				reply(protocol.MsgCNFCharacterLogin, []byte{1, 2, 3}, 4242)
			}
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go serve(conn)
		}
	}()

	closeFn = func() {
		_ = ln.Close()
		wg.Wait()
	}
	return ln.Addr().String(), ch, closeFn
}

func trimNUL(b []byte) string {
	for i, v := range b {
		if v == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func TestClientLoginSelectAndSend(t *testing.T) {
	addr, received, closeFn := fakeServer(t)
	defer closeFn()

	c, err := Dial(addr, 2*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	if err := c.Login("bot1", "bot123", "", 12000); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if err := c.SelectCharacter(0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	if c.ID() != 4242 {
		t.Fatalf("ID do mundo = %d, quer 4242 (vem do header do CNFCharacterLogin)", c.ID())
	}
	if err := c.Send(protocol.MsgMessageChat, make([]byte, 32)); err != nil {
		t.Fatalf("Send: %v", err)
	}

	want := []protocol.Type{protocol.MsgAccountLogin, protocol.MsgCharacterLogin, protocol.MsgMessageChat}
	for i, w := range want {
		select {
		case got := <-received:
			if got != w {
				t.Fatalf("frame %d = %#x, quer %#x", i, uint16(got), uint16(w))
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("frame %d (%#x) não chegou ao servidor", i, uint16(w))
		}
	}
}

func TestClientLoginRefusedIsTyped(t *testing.T) {
	addr, _, closeFn := fakeServer(t)
	defer closeFn()

	c, err := Dial(addr, 2*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	if err := c.Login("blocked", "x", "", 12000); !errors.Is(err, ErrLoginRefused) {
		t.Fatalf("erro = %v, quer ErrLoginRefused", err)
	}
}

func TestClientAlreadyPlayingIsTyped(t *testing.T) {
	addr, _, closeFn := fakeServer(t)
	defer closeFn()

	c, err := Dial(addr, 2*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	if err := c.Login("busy", "x", "", 12000); !errors.Is(err, ErrAlreadyPlaying) {
		t.Fatalf("erro = %v, quer ErrAlreadyPlaying", err)
	}
}

func TestClientCharacterRefusedIsTyped(t *testing.T) {
	addr, _, closeFn := fakeServer(t)
	defer closeFn()

	c, err := Dial(addr, 2*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	if err := c.Login("bot1", "bot123", "", 12000); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if err := c.SelectCharacter(9); !errors.Is(err, ErrCharacterRefused) {
		t.Fatalf("erro = %v, quer ErrCharacterRefused", err)
	}
}

func TestClientRejectsLongAccount(t *testing.T) {
	addr, _, closeFn := fakeServer(t)
	defer closeFn()

	c, err := Dial(addr, 2*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	if err := c.Login("0123456789abcdef", "x", "", 12000); err == nil {
		t.Fatal("conta de 16 bytes foi aceita (campo tem 16 com NUL)")
	}
}

// TestBehaviorSequenceIsDeterministic prova a reprodução sem relógio: mesma
// semente e mesmo perfil produzem exatamente a mesma sequência de transições.
func TestBehaviorSequenceIsDeterministic(t *testing.T) {
	transitions := DefaultTransitions(NewMetrics())
	run := func() []Record {
		m, err := NewMachine(StateConnecting, transitions)
		if err != nil {
			t.Fatalf("NewMachine: %v", err)
		}
		profile, err := LookupProfile(ProfileFarmer)
		if err != nil {
			t.Fatalf("perfil: %v", err)
		}
		g := NewRNG(99)
		c := &Context{HP: 100, HPmax: 100, MP: 50, MPmax: 50, InventoryFree: 8, Potions: 5}
		for _, ev := range []Event{EventConnected, EventAuthenticated, EventCharacterReady, EventWorldEntered} {
			if _, err := m.Fire(ev, c); err != nil {
				t.Fatalf("ciclo de vida em %s: %v", ev, err)
			}
		}
		for i := 0; i < 300; i++ {
			ev := nextEvent(m.State(), c, profile, g)
			if ev == "" {
				continue
			}
			if _, err := m.Fire(ev, c); err != nil {
				continue
			}
		}
		return m.History()
	}

	a, b := run(), run()
	if len(a) == 0 {
		t.Fatal("nenhuma transição executada")
	}
	if len(a) != len(b) {
		t.Fatalf("mesma semente gerou %d e %d transições", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("transição %d divergiu: %+v != %+v", i, a[i], b[i])
		}
	}
}

func TestRunnerExecutesScenario(t *testing.T) {
	addr, _, closeFn := fakeServer(t)
	defer closeFn()

	s := &Scenario{
		ID:              "unit",
		Objective:       "provar o executor com servidor de mentira",
		Seed:            5,
		DurationSeconds: 1,
		TickMillis:      20,
		Address:         addr,
		Bots: []BotSpec{
			{Account: "bot1", Password: "x", Slot: 0, Profile: ProfileFarmer, Region: "armia"},
			{Account: "bot2", Password: "x", Slot: 0, Profile: ProfileCombatente, Region: "armia"},
		},
		Success: []Condition{
			{Metric: "bots_connected", Op: "==", Value: 2},
			{Metric: "actions", Op: ">", Value: 0},
		},
		Failure: []Condition{{Metric: "bots_failed", Op: ">", Value: 0}},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("cenário: %v", err)
	}

	r, err := NewRunner(s, Options{})
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	rep, err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Connected != 2 || rep.Failed != 0 {
		t.Fatalf("conexões = %d/%d (amostras %v)", rep.Connected, rep.Failed, rep.ErrorSamples)
	}
	if rep.Actions <= 0 {
		t.Fatal("nenhuma ação executada")
	}
	if rep.Failure {
		t.Fatalf("critério de falha satisfeito: %v", rep.FailureWhy)
	}
	if !rep.Success {
		t.Fatalf("critério de sucesso não satisfeito: %v", rep.SuccessWhy)
	}
	if len(rep.StatesVisited) < 5 {
		t.Fatalf("estados visitados = %v", rep.StatesVisited)
	}
}

func TestRunnerCountsFailedBots(t *testing.T) {
	addr, _, closeFn := fakeServer(t)
	defer closeFn()

	s := &Scenario{
		ID:              "falha",
		Objective:       "provar que conta recusada vira falha contabilizada",
		Seed:            1,
		DurationSeconds: 1,
		TickMillis:      50,
		Address:         addr,
		Bots: []BotSpec{
			{Account: "blocked", Password: "x", Slot: 0, Profile: ProfileCasual},
		},
		Failure: []Condition{{Metric: "bots_failed", Op: ">", Value: 0}},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("cenário: %v", err)
	}
	r, err := NewRunner(s, Options{})
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	rep, err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Failed != 1 || rep.Connected != 0 {
		t.Fatalf("conexões = %d, falhas = %d", rep.Connected, rep.Failed)
	}
	if !rep.Failure {
		t.Fatal("critério de falha não foi satisfeito com bot recusado")
	}
	if len(rep.ErrorSamples) == 0 {
		t.Fatal("erro do bot não foi registrado")
	}
}
