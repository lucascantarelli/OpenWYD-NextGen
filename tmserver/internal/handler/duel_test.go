package handler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// duelDB dá a tester(7)/tradeb(11) personagens com nomes distintos na mesma
// posição inicial. Hero tem Dano suficiente para matar HeroB de HP baixo num
// único golpe corpo a corpo, então o teste de vitória por eliminação não depende da variância
// do RNG ao longo de muitos ataques.
func duelDB() *fakeDB {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7:  {Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, ClassMaster: classMasterMortal, Str: 1000},
		11: {Slot: 0, Name: "HeroB", X: 5, Y: 5, HP: 50, MaxHP: 50, Damage: 0, AC: 0},
	}
	return db
}

// startServerDuel é startServerClock com uma grade de tamanho cheio (as coordenadas de spawn
// da arena, por exemplo Y=4044, ficam bem fora da grade 16x16 que os testes de movimento
// usam) e um handler de tick rápido, para as varreduras de timeout de convite/arena resolverem em
// milissegundos em vez de segundos/ticks reais de relógio.
func startServerDuel(t *testing.T, persist world.Persistence) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	clock := &atomic.Uint32{}
	clock.Store(serverTime)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: world.DefaultGridDim, Now: clock.Load}, log, persist, d.Handle)
	w.SetTickHandler(10*time.Millisecond, d.Tick)
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
	}
}

func reqRankingFrame(t *testing.T, c net.Conn, target int, duelParm int32) {
	t.Helper()
	send(t, c, protocol.MsgReqRanking, protocol.EncodeStandardParm2(int32(target), duelParm))
}

// drainDuelStart consome os frames de entrada na arena (RemoveMob/CreateMob filtrados
// por readMaybe; o salto de teleporte do MSG_Action e a contagem de StartTime
// não são), então asserções de jogabilidade depois de startDuel veem só os próprios frames.
func drainDuelStart(t *testing.T, c net.Conn) {
	t.Helper()
	for i := 0; i < 10; i++ {
		if _, _, ok := readMaybe(t, c); !ok {
			return
		}
	}
}

func TestDuelRequestAndAccept(t *testing.T) {
	addr, stop := startServerDuel(t, duelDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester") // conn 1 (solicitante)
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb") // conn 2 (alvo)
	defer b.Close()

	reqRankingFrame(t, a, 2, 0) // A convida B para um 1v1
	ty, payload, ok := readMaybe(t, b)
	if !ok || ty != protocol.MsgReqRanking {
		t.Fatalf("B got %#x ok=%v, want the forwarded invite", ty, ok)
	}
	parm1, parm2, decOK := protocol.StandardParm2(payload)
	if !decOK || parm1 != 1 || parm2 != 0 {
		t.Fatalf("invite payload = (%d,%d) ok=%v, want (1,0)", parm1, parm2, decOK)
	}

	reqRankingFrame(t, b, 1, 4) // B aceita (Parm1 ecoa o conn do solicitante)

	for name, c := range map[string]net.Conn{"requester": a, "accepter": b} {
		sawAction, sawStart := false, false
		for i := 0; i < 8 && (!sawAction || !sawStart); i++ {
			ty, _, ok := readMaybe(t, c)
			if !ok {
				break
			}
			switch ty {
			case protocol.MsgAction:
				sawAction = true
			case protocol.MsgStartTime:
				sawStart = true
			}
		}
		if !sawAction || !sawStart {
			t.Errorf("%s: action=%v start=%v, want both", name, sawAction, sawStart)
		}
	}
}

// TestDuelAcceptWithoutInvite: o gate de reciprocidade DuelTarget bloqueia um aceite
// forjado (espelha a checagem PARTYHACK do LastReqParty).
func TestDuelAcceptWithoutInvite(t *testing.T) {
	addr, stop := startServerDuel(t, duelDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	reqRankingFrame(t, b, 1, 4) // B "aceita" um duelo que A nunca pediu
	if ty, _, ok := readMaybe(t, a); ok {
		t.Errorf("A received %#x for a forged accept; want none", ty)
	}
	if ty, _, ok := readMaybe(t, b); ok {
		t.Errorf("B received %#x for its own forged accept; want none", ty)
	}
}

// TestDuelInviteTimeout: um convite sem resposta é limpado automaticamente (o critério
// de aceite da issue "recusado/timeout limpa o estado") — um aceite tardio
// depois do prazo é ignorado em silêncio, e nenhum dos personagens fica travado (o
// solicitante podia mandar um convite novo tranquilamente depois).
func TestDuelInviteTimeout(t *testing.T) {
	orig := duelInviteTimeout
	duelInviteTimeout = 100 * time.Millisecond
	defer func() { duelInviteTimeout = orig }()

	addr, stop := startServerDuel(t, duelDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	reqRankingFrame(t, a, 2, 0)
	if ty, _, ok := readMaybe(t, b); !ok || ty != protocol.MsgReqRanking {
		t.Fatalf("B got %#x ok=%v, want the invite", ty, ok)
	}

	time.Sleep(300 * time.Millisecond) // passado o timeout encurtado, mais a margem de tick

	reqRankingFrame(t, b, 1, 4) // aceite tardio: o convite já expirou
	if ty, _, ok := readMaybe(t, a); ok {
		t.Errorf("A got %#x after a timed-out invite; want nothing (no duel started)", ty)
	}
	if ty, _, ok := readMaybe(t, b); ok {
		t.Errorf("B got %#x after its own late accept; want nothing", ty)
	}
}

// startedDuel sobe um servidor com db, conecta "tester" (Hero, conn 1) e
// "tradeb" (HeroB, conn 2), e os conduz por convite/aceite para ambos
// estarem na arena com os frames de entrada drenados. Quem chama cuida de fechar a/b/stop.
func startedDuel(t *testing.T, db *fakeDB) (a, b net.Conn, stop func()) {
	t.Helper()
	addr, stop := startServerDuel(t, db)
	a = enterWorldAs(t, addr, "tester")
	b = enterWorldAs(t, addr, "tradeb")

	reqRankingFrame(t, a, 2, 0)
	readMaybe(t, b) // drenar o convite repassado
	reqRankingFrame(t, b, 1, 4)
	drainDuelStart(t, a)
	drainDuelStart(t, b)
	return a, b, stop
}

// TestDuelWinByElimination cobre o critério de aceite central da issue: A
// desafia B, B aceita, eles lutam, o resultado é registrado. Também
// verifica as duas mudanças de gate de combate que esta feature faz: um golpe de duelo entra
// sem PKMode ligado, e NÃO inicia o temporizador de nick PK caótico.
func TestDuelWinByElimination(t *testing.T) {
	db := duelDB()
	a, b, stop := startedDuel(t, db)
	defer stop()
	defer a.Close()
	defer b.Close()

	// A acerta B corpo a corpo sem ligar PKMode: o bypass de duelo (combat.go) precisa deixá-lo
	// entrar mesmo assim.
	attackFrame(t, a, serverTime, 2, 0)
	ty, payload, ok := readMaybe(t, a)
	if !ok || ty != protocol.MsgAttack {
		t.Fatalf("attacker got %#x ok=%v, want the MsgAttack echo", ty, ok)
	}
	var body protocol.MsgAttackBody
	if err := body.Decode(payload); err != nil {
		t.Fatal(err)
	}
	if len(body.Dam) != 1 || body.Dam[0].Damage <= 0 {
		t.Fatalf("Dam = %+v, want a landed hit (PKMode-less duel damage must not be gated)", body.Dam)
	}

	// A morte em si avisa HeroB de que não há perda de EXP (ZoneUnk, issue #427)
	// antes de a varredura da arena anunciar o resultado.
	wantPanelWithin(t, b, inArenaNoExpLoss, 5*time.Second)

	// A próxima varredura de vivacidade detecta HP<=0 de HeroB e conclui o duelo: os dois
	// lados recebem o texto legado _SS_S_WinBy_S com os dois nomes.
	result := fmt.Sprintf(duelWinBy, "Hero", "HeroB")
	wantPanelWithin(t, a, result, 5*time.Second)
	wantPanelWithin(t, b, result, 5*time.Second)

	dr, ok := db.lastDuelResult(t)
	if !ok {
		t.Fatal("RecordDuelResult was never called")
	}
	if dr.winner != "Hero" || dr.loser != "HeroB" {
		t.Errorf("duel result = %+v, want winner=Hero loser=HeroB", dr)
	}
}

// TestDuelDisconnectEndsAsWin: uma desconexão no meio do duelo resolve a luta em
// favor do lado ainda conectado, em vez de deixar a arena travada.
func TestDuelDisconnectEndsAsWin(t *testing.T) {
	db := duelDB()
	a, b, stop := startedDuel(t, db)
	defer stop()
	defer a.Close()

	b.Close() // simula uma desconexão no meio do duelo

	wantPanelWithin(t, a, fmt.Sprintf(duelWinBy, "Hero", "HeroB"), 5*time.Second)

	dr, ok := db.lastDuelResult(t)
	if !ok {
		t.Fatal("RecordDuelResult was never called")
	}
	if dr.winner != "Hero" || dr.loser != "HeroB" {
		t.Errorf("duel result = %+v, want winner=Hero loser=HeroB", dr)
	}
}

// TestDuelingHelper é um teste unitário focado na checagem de par de arena que o combat.go
// usa para furar o gate de PKMode e pular a marcação de nick PK.
func TestDuelingHelper(t *testing.T) {
	d := New(Config{Log: slog.New(slog.NewTextHandler(io.Discard, nil))})

	d.duel = duelArena{active: true, player1: 1, player2: 2}
	if !d.dueling(1, 2) || !d.dueling(2, 1) {
		t.Error("dueling should be true for both orderings of the active pair")
	}
	if d.dueling(1, 3) || d.dueling(3, 2) {
		t.Error("dueling should be false for a non-participant")
	}

	d.duel = duelArena{}
	if d.dueling(1, 2) {
		t.Error("dueling should be false when no duel is active")
	}
}

// duelCityDB é duelDB com os dois personagens dentro de Armia (2096,2096, o mesmo
// ponto que o TestVillage em world/city_test.go afirma resolver para a cidade 0).
func duelCityDB() *fakeDB {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7:  {Slot: 0, Name: "Hero", X: 2096, Y: 2096, HP: 1000, MaxHP: 1000, Damage: 500, AC: 0},
		11: {Slot: 0, Name: "HeroB", X: 2096, Y: 2096, HP: 50, MaxHP: 50, Damage: 0, AC: 0},
	}
	return db
}

// TestDuelRequestAllowedInCity: o pedido legado não tem checagem de posição, então um
// pedido de dentro de uma cidade é repassado como qualquer outro (issue #424).
func TestDuelRequestAllowedInCity(t *testing.T) {
	addr, stop := startServerDuel(t, duelCityDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester") // conn 1, dentro de Armia
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb") // conn 2, dentro de Armia
	defer b.Close()

	reqRankingFrame(t, a, 2, 0)
	if ty, _, ok := readMaybe(t, b); !ok || ty != protocol.MsgReqRanking {
		t.Fatalf("B got %#x ok=%v, want the forwarded invite", ty, ok)
	}
}

// TestDuelRequestDeniedByWhisperBlock: um alvo que bloqueia sussurros não recebe
// convite, e o solicitante lê o texto legado _NN_Deny_Whisper.
func TestDuelRequestDeniedByWhisperBlock(t *testing.T) {
	addr, stop := startServerDuel(t, duelDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	chatFrame(t, b, "whisper") // liga o bloqueio de sussurros de B
	drain(t, b)
	reqRankingFrame(t, a, 2, 0)
	wantPanelWithin(t, a, denyWhisper, 2*time.Second)
	if ty, _, ok := readMaybe(t, b); ok && ty == protocol.MsgReqRanking {
		t.Error("B got the invite despite blocking whispers")
	}
}

// TestDuelStartAnnouncesBattle: DoRanking envia _NN_Battle_Started nomeando
// o aceitante primeiro, e StartTime 2*121-2 = 240.
func TestDuelStartAnnouncesBattle(t *testing.T) {
	addr, stop := startServerDuel(t, duelDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	reqRankingFrame(t, a, 2, 0)
	readMaybe(t, b)
	reqRankingFrame(t, b, 1, 4)

	want := fmt.Sprintf(battleStarted, "HeroB", "Hero")
	for name, c := range map[string]net.Conn{"requester": a, "accepter": b} {
		sawText, startParm := false, int32(-1)
		for i := 0; i < 10 && (!sawText || startParm < 0); i++ {
			ty, p, ok := readMaybe(t, c)
			if !ok {
				break
			}
			switch ty {
			case protocol.MsgMessagePanel:
				sawText = sawText || cstr(p) == want
			case protocol.MsgStartTime:
				startParm, _ = protocol.StandardParm(p)
			}
		}
		if !sawText || startParm != 240 {
			t.Errorf("%s: battle text=%v StartTime=%d, want true/240", name, sawText, startParm)
		}
	}
}

// wantPanelWithin consulta c por um MsgMessagePanel carregando text, limitado por
// tempo de relógio em vez de uma contagem fixa de leituras: as transmissões da parede
// fechando da arena (até uma dúzia de frames MsgEnvEffect por passo) estourariam um
// orçamento fixo de leituras bem antes de um texto de resolução de duelo chegar.
func wantPanelWithin(t *testing.T, c net.Conn, text string, budget time.Duration) {
	t.Helper()
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		ty, payload, ok := readMaybe(t, c)
		if ok && ty == protocol.MsgMessagePanel && cstr(payload) == text {
			return
		}
	}
	t.Fatalf("did not observe panel %q within %v", text, budget)
}

// TestDuelDrawOnTimeout: quando a contagem da arena acaba com os dois
// duelistas ainda vivos e dentro dos limites, o duelo resolve como empate — os dois
// lados são avisados e, ao contrário de uma vitória, nada é persistido.
func TestDuelDrawOnTimeout(t *testing.T) {
	orig := duelTicks
	// Precisa resolver depois que o drainDuelStart de startedDuel retorna: aquele helper
	// drena frames até uma leitura de ~300ms dar timeout, então uma contagem menor
	// que isso colocaria o aviso de empate a ser engolido em silêncio pelo
	// dreno em vez de observado abaixo.
	duelTicks = 100
	defer func() { duelTicks = orig }()

	db := duelDB()
	a, b, stop := startedDuel(t, db)
	defer stop()
	defer a.Close()
	defer b.Close()

	draw := fmt.Sprintf(duelDraw, "Hero", "HeroB")
	wantPanelWithin(t, a, draw, 5*time.Second)
	wantPanelWithin(t, b, draw, 5*time.Second)

	if _, ok := db.lastDuelResult(t); ok {
		t.Error("RecordDuelResult was called for a draw; draws must not be persisted")
	}
}

// TestDuelEndClearsArena: ClearAreaTeleport roda tanto numa vitória quanto num empate —
// todos dentro da arena vão para (2572,1752), os mortos voltam com HP 1,
// e quem já saiu da caixa fica onde está (issue #424).
func TestDuelEndClearsArena(t *testing.T) {
	d, w, s, e := startOnePlayer(t)
	for _, tc := range []struct {
		name     string
		x, y     int16
		hp       int32
		draw     bool
		wantExit bool
		wantHP   int32
	}{
		{"dead loser revived at the exit", 160, 4044, 0, false, true, 1},
		{"living winner keeps HP", 160, 4044, 300, false, true, 300},
		{"draw also teleports", 160, 4044, 300, true, true, 300},
		{"outside the box stays put", 300, 300, 300, false, false, 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runInLoop(t, w, func() {
				w.SetEntityPos(s.Conn, tc.x, tc.y)
				e.HP = tc.hp
				d.duel = duelArena{active: true, player1: s.Conn, ticksLeft: 1, name1: "Hero", name2: "Gone"}
				if tc.draw {
					d.concludeDuelDraw(w)
				} else {
					d.concludeDuelWin(w, s.Conn, 0)
				}
				atExit := e.X == duelExitX && e.Y == duelExitY
				if atExit != tc.wantExit {
					t.Errorf("at (%d,%d), want exit=%v", e.X, e.Y, tc.wantExit)
				}
				if e.HP != tc.wantHP {
					t.Errorf("HP = %d, want %d", e.HP, tc.wantHP)
				}
				if d.duel.active {
					t.Error("arena still occupied")
				}
			})
		})
	}
}

// TestDuelSweepCadence: um passo de ProcessRanking a cada duelStepTicks ticks, e
// um duelista morto só encerra o duelo numa varredura de vivacidade (ticksLeft%5 == 0).
func TestDuelSweepCadence(t *testing.T) {
	d, w, s, e := startOnePlayer(t)
	runInLoop(t, w, func() {
		w.SetEntityPos(s.Conn, 160, 4044)
		e.HP = 300
		// player2 (conn 0) está ausente, então a primeira varredura encerra o duelo.
		d.duel = duelArena{active: true, player1: s.Conn, ticksLeft: 13, name1: "Hero", name2: "Gone"}

		d.sweepDuelArena(w) // tick 1 de 2: ainda sem passo
		if d.duel.ticksLeft != 13 {
			t.Fatalf("ticksLeft = %d after one tick, want 13", d.duel.ticksLeft)
		}
		for range 2 * 2 { // dois passos: 12, 11 — sem varredura
			d.sweepDuelArena(w)
		}
		if !d.duel.active || d.duel.ticksLeft != 11 {
			t.Fatalf("active=%v ticksLeft=%d, want a running duel at 11", d.duel.active, d.duel.ticksLeft)
		}
		for range 2 * 1 { // passo para 10: a varredura vê o oponente ausente
			d.sweepDuelArena(w)
		}
		if d.duel.active {
			t.Fatal("duel still active after the ticksLeft%5 scan")
		}
		if e.X != duelExitX || e.Y != duelExitY {
			t.Errorf("winner at (%d,%d), want the exit", e.X, e.Y)
		}
	})
}

// TestDuelStageThresholds é um teste unitário com tabela dos limites de balde do
// duelStage (os três estágios de parede fechando do Server.cpp:8834-8869).
func TestDuelStageThresholds(t *testing.T) {
	cases := []struct {
		ticksLeft int
		want      int
	}{
		{181, 0}, {180, 0},
		{179, 1}, {120, 1},
		{119, 2}, {60, 2},
		{59, 3}, {0, 3}, {-1, 3},
	}
	for _, c := range cases {
		if got := duelStage(c.ticksLeft); got != c.want {
			t.Errorf("duelStage(%d) = %d, want %d", c.ticksLeft, got, c.want)
		}
	}
}

// TestDuelWallDamageFloorsHP é um teste unitário puro de duelWallDamage: ele baixa
// o HP em 2000, ou até 1 se o duelista tiver menos que isso, e nunca
// leva o HP a 0 ou abaixo (o efeito punitivo do SendDamage, Server.cpp:5970-6009).
func TestDuelWallDamageFloorsHP(t *testing.T) {
	cases := []struct {
		name   string
		hp     int32
		wantHP int32
	}{
		{"high HP loses exactly 2000", 5000, 3000},
		{"HP just above the floor", 2001, 1},
		{"low HP floors to 1, not 0 or negative", 50, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &world.Session{}
			// MaxHP precisa ser alto o suficiente para o clamp de effectiveMaxHP do setReqHp
			// (item.go) nunca entrar e mascarar o dano que este teste verifica.
			e := &world.Entity{HP: c.hp, MaxHP: 10000}
			duelWallDamage(s, e)
			if e.HP != c.wantHP {
				t.Errorf("HP = %d, want %d", e.HP, c.wantHP)
			}
			if e.HP <= 0 {
				t.Error("duelWallDamage must never drive HP to 0 or below")
			}
		})
	}
}
