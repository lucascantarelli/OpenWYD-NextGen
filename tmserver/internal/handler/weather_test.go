package handler

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/rng"
	"openwyd-nextgen/tmserver/internal/world"
)

// startServerWeather é um harness de serving que entrega o Dispatcher e o World
// de volta, então os testes podem semear o estado de eventos do mundo e acionar Tick direto.
func startServerWeather(t *testing.T, persist world.Persistence, baseMobs map[int][]byte) (string, func(), *Dispatcher, *world.World) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log, BaseMobs: baseMobs})
	w := world.New(world.Config{GridDim: 16}, log, persist, d.Handle)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	return ln.Addr().String(), func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	}, d, w
}

// weatherTemplate é um BaseMob mínimo de 816 bytes para o login de personagem pegar o
// ramo do template (o outro ramo é o fallback relacional).
func weatherTemplate() []byte {
	tmpl := make([]byte, content.BaseMobSize)
	copy(tmpl[0:16], "Template")
	binary.LittleEndian.PutUint16(tmpl[40:], 2112)
	binary.LittleEndian.PutUint16(tmpl[42:], 2112)
	return tmpl
}

// TestLoginSnapshotCarriesWeather cobre os DOIS construtores de CNFCharacterLogin: o
// legado embarca CurrentWeather dentro do blob de login em vez de num
// pacote próprio (sm.Weather = CurrentWeather, ProcessDBMessage.cpp:834), e até a
// issue #116 os dois pontos de chamada fixavam 0 no código.
func TestLoginSnapshotCarriesWeather(t *testing.T) {
	tests := []struct {
		name     string
		baseMobs map[int][]byte
	}{
		{"template path", map[int][]byte{1: weatherTemplate()}},
		{"relational fallback", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newDB()
			db.loadResult = world.CharacterState{Slot: 0, Name: "Hero", Class: 1, Level: 50, HP: 1200, MaxHP: 1200}
			addr, stop, d, w := startServerWeather(t, db, tt.baseMobs)
			defer stop()

			// Semeia o clima de dentro do loop, então ele está definido antes do login.
			setWeatherInLoop(t, w, d, 2)

			c := loginAndSelect(t, addr)
			defer c.Close()
			var body protocol.MsgCharacterLoginBody
			send(t, c, protocol.MsgCharacterLogin, body.Encode())
			ty, payload := read(t, c)
			if ty != protocol.MsgCNFCharacterLogin {
				t.Fatalf("got %#x, want CNFCharacterLogin", ty)
			}
			if got := binary.LittleEndian.Uint16(payload[1032:]); got != 2 {
				t.Fatalf("Weather @body1032 = %d, want 2", got)
			}
		})
	}
}

// setWeatherInLoop muta o estado de clima pertencente ao loop a partir da goroutine do loop
// (o estado do mundo é lock-free justamente porque só o loop o toca).
func setWeatherInLoop(t *testing.T, w *world.World, d *Dispatcher, weather int32) {
	t.Helper()
	done := make(chan struct{})
	w.GoDetached(func() func(*world.World) {
		return func(*world.World) {
			d.events.weather = weather
			close(done)
		}
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop callback did not run")
	}
}

// TestTickWeatherBroadcastsChange aciona o gate de minuto com um stream semeado
// e afirma que o MSG_UpdateWeather global alcança um cliente em jogo com
// HEADER.ID = ESCENE_FIELD (SendWeather, SendFunc.cpp:1669-1696).
func TestTickWeatherBroadcastsChange(t *testing.T) {
	db := newDB()
	db.loadResult = world.CharacterState{Slot: 0, Name: "Hero", Class: 1, Level: 50, X: 5, Y: 5, HP: 1200, MaxHP: 1200}
	addr, stop, d, w := startServerWeather(t, db, nil)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	// Fixa o sorteio: um eventRNG cujo primeiro draw cai na faixa do clima 1,
	// e um tickCount que põe o próximo Tick exatamente no gate de minuto.
	forceWeatherRoll(t, w, d, 35)

	h, payload, ok := readMaybeHeader(t, c)
	if !ok {
		t.Fatal("no weather broadcast after the minute tick")
	}
	if h.Type != protocol.MsgUpdateWeather {
		t.Fatalf("got %#x, want MsgUpdateWeather", h.Type)
	}
	if h.ID != protocol.IDScene {
		t.Fatalf("HEADER.ID = %d, want IDScene (%d)", h.ID, protocol.IDScene)
	}
	if got := int32(binary.LittleEndian.Uint32(payload)); got != 1 {
		t.Fatalf("CurrentWeather = %d, want 1", got)
	}
}

// TestTickWeatherHonoursForceOverride fixa a semântica do override de GM: o sorteio
// continua consumido a cada minuto (o legado sorteia antes de ler
// ForceWeather), mas o valor forçado é o que chega ao cliente, e só uma vez.
func TestTickWeatherHonoursForceOverride(t *testing.T) {
	db := newDB()
	db.loadResult = world.CharacterState{Slot: 0, Name: "Hero", Class: 1, Level: 50, X: 5, Y: 5, HP: 1200, MaxHP: 1200}
	addr, stop, d, w := startServerWeather(t, db, nil)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	// Semeia um stream cujo primeiro draw escolheria clima 1 de outro modo, então a
	// asserção abaixo prova que o override venceu o sorteio em vez de concordar
	// com ele por sorte.
	runInLoop(t, w, func() {
		d.events.forceWeather = 2
		d.eventRNG = seededForRoll(35) // faixa → clima 1
		d.tickCount = weatherTickPeriod - 1
	})
	runInLoop(t, w, func() { d.Tick(w) })

	payload, ok := readUntilType(t, c, protocol.MsgUpdateWeather)
	if !ok {
		t.Fatal("forced weather did not broadcast")
	}
	if got := int32(binary.LittleEndian.Uint32(payload)); got != 2 {
		t.Fatalf("CurrentWeather = %d, want the forced 2 (the roll wanted 1)", got)
	}

	// Um segundo minuto ainda precisa consumir um draw — o legado sorteia antes de
	// olhar ForceWeather, então o stream avança no mesmo ritmo de qualquer jeito
	// — mas com o valor já igual ao override nada vai para o wire.
	counter := &countingRand{inner: seededForRoll(35)}
	runInLoop(t, w, func() {
		d.eventRNG = counter
		d.tickCount = weatherTickPeriod - 1
	})
	runInLoop(t, w, func() { d.Tick(w) })

	var draws int
	runInLoop(t, w, func() { draws = counter.draws })
	if draws != 1 {
		t.Errorf("draws while an override was active = %d, want 1 (the stream must still advance)", draws)
	}
	if h, _, ok := readMaybeHeader(t, c); ok && h.Type == protocol.MsgUpdateWeather {
		t.Error("weather rebroadcast while the value was unchanged")
	}
}

// countingRand conta draws para os testes afirmarem que o stream avançou (ou não).
type countingRand struct {
	inner *rng.MSVC
	draws int
}

func (c *countingRand) Intn(n int) int {
	c.draws++
	return c.inner.Intn(n)
}

// forceWeatherRoll semeia o eventRNG para o próximo draw de RollWeather devolver `roll`,
// deixa o tickCount a um passo do gate de minuto, e roda um Tick.
func forceWeatherRoll(t *testing.T, w *world.World, d *Dispatcher, roll int) {
	t.Helper()
	runInLoop(t, w, func() {
		d.eventRNG = seededForRoll(roll)
		d.tickCount = weatherTickPeriod - 1
	})
	runInLoop(t, w, func() { d.Tick(w) })
}

// seededForRoll acha uma semente MSVC cujo primeiro Intn(1200) é igual a roll. O LCG
// é barato e sobrejetivo sobre o módulo, então uma varredura curta sempre acha.
func seededForRoll(roll int) *rng.MSVC {
	for seed := uint32(1); seed < 100000; seed++ {
		r := rng.NewSeeded(seed)
		if r.Intn(1200) == roll {
			return rng.NewSeeded(seed)
		}
	}
	panic("no seed produces the requested roll")
}

// runInLoop executa fn na goroutine do loop do mundo e espera por ela.
func runInLoop(t *testing.T, w *world.World, fn func()) {
	t.Helper()
	done := make(chan struct{})
	w.GoDetached(func() func(*world.World) {
		return func(*world.World) {
			fn()
			close(done)
		}
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop callback did not run")
	}
}

// TestGMWeatherRequiresModerator: o bus já barra por AccessLevel, então o
// /gm weather de um jogador comum precisa ser descartado em silêncio e não mudar nada.
func TestGMWeatherRequiresModerator(t *testing.T) {
	addr, stop, d, w := startServerWeather(t, gmDB(), nil)
	defer stop()
	c := enterWorldAs(t, addr, "player")
	defer c.Close()

	gmFrame(t, c, "weather 2")
	if ty, _, ok := readMaybe(t, c); ok {
		t.Errorf("player got %#x; a non-GM must be silently denied", ty)
	}
	var weather int32
	runInLoop(t, w, func() { weather = d.events.weather })
	if weather != 0 {
		t.Errorf("weather = %d after a denied command, want 0", weather)
	}
}

// TestGMWeatherSetsAndRestoresAuto cobre as três melhorias sobre o /weather
// legado (imple.cpp:1122-1129): ele valida o valor, transmite a
// mudança, e "auto" restaura ForceWeather = -1 (o legado não tinha volta).
func TestGMWeatherSetsAndRestoresAuto(t *testing.T) {
	addr, stop, d, w := startServerWeather(t, gmDB(), nil)
	defer stop()
	c := enterWorldAs(t, addr, "mod")
	defer c.Close()

	// Todo /gm weather responde com uma linha de chat; drenar até ela é o que torna
	// a leitura de estado seguinte livre de corrida (o comando viaja pelo socket
	// e é aplicado na goroutine do loop).
	gmWeatherCmd(t, c, "weather 2")
	var weather, force int32
	runInLoop(t, w, func() { weather, force = d.events.weather, d.events.forceWeather })
	if weather != 2 || force != 2 {
		t.Fatalf("weather/force = %d/%d, want 2/2", weather, force)
	}

	gmWeatherCmd(t, c, "weather auto")
	runInLoop(t, w, func() { force = d.events.forceWeather })
	if force != weatherAuto {
		t.Fatalf("forceWeather = %d after /gm weather auto, want %d", force, weatherAuto)
	}

	// Valores inválidos são recusados sem tocar no estado.
	gmWeatherCmd(t, c, "weather 9")
	runInLoop(t, w, func() { weather, force = d.events.weather, d.events.forceWeather })
	if weather != 2 || force != weatherAuto {
		t.Fatalf("invalid /gm weather 9 changed state: weather=%d force=%d", weather, force)
	}
}

// gmWeatherCmd envia uma linha /gm weather e espera pelo seu reconhecimento no chat,
// para quem chamou então ler o estado pertencente ao loop sem correr contra o handler.
func gmWeatherCmd(t *testing.T, c net.Conn, line string) {
	t.Helper()
	gmFrame(t, c, line)
	if _, ok := readUntilType(t, c, protocol.MsgMessageChat); !ok {
		t.Fatalf("no chat acknowledgement for /gm %s", line)
	}
}

// readUntilType drena frames até `want` aparecer ou a conexão ficar muda.
func readUntilType(t *testing.T, c net.Conn, want protocol.Type) ([]byte, bool) {
	t.Helper()
	for {
		h, payload, ok := readMaybeHeader(t, c)
		if !ok {
			return nil, false
		}
		if h.Type == want {
			return payload, true
		}
	}
}
