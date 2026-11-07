package handler

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/rng"
	"openwyd-nextgen/tmserver/internal/route"
	"openwyd-nextgen/tmserver/internal/world"
)

func TestChebyshevAndStep(t *testing.T) {
	if d := chebyshev(5, 5, 8, 6); d != 3 { // distância de movimento de rei = max(|dx|,|dy|)
		t.Errorf("chebyshev = %d, want 3", d)
	}
	if d := chebyshev(5, 5, 5, 5); d != 0 {
		t.Errorf("chebyshev same cell = %d, want 0", d)
	}
	if step(7) != 1 || step(-3) != -1 || step(0) != 0 {
		t.Errorf("step direction wrong: %d %d %d", step(7), step(-3), step(0))
	}
}

// aggressiveMob monta um template STRUCT_MOB de 816 bytes para uma criatura que sempre
// age (Int alto → sem hesitação) e bate forte (Damage alto). Merchant=0 para a
// IA tratá-la como combatente, não como lojista; Clan=5 para g_pClanTable marcá-la
// hostil a jogadores (clã 0) — um template de clã 0 seria AMISTOSO e nunca
// daria aggro (clanTable[0][0]==1).
func aggressiveMob() []byte {
	b := make([]byte, 816)
	copy(b[0:16], "Biter")
	b[16] = 5                                     // Clan @16: hostil ao clã de jogador 0
	const cs = 92                                 // CurrentScore
	binary.LittleEndian.PutUint32(b[cs+0:], 50)   // Level
	binary.LittleEndian.PutUint32(b[cs+8:], 500)  // Damage
	binary.LittleEndian.PutUint32(b[cs+16:], 800) // MaxHp
	binary.LittleEndian.PutUint32(b[cs+24:], 800) // Hp
	binary.LittleEndian.PutUint16(b[cs+34:], 99)  // Int (nunca hesita: 99 < rand%100 nunca é verdade)
	return b
}

// startServerMobAI é o harness de combate mais um tick de IA de criaturas: nasce uma
// criatura agressiva em (mobX,mobY) antes do loop começar e pulsa a IA a cada
// 10ms com relógio congelado. gridDim dimensiona o mundo (use uma grade do tamanho da cidade para testar
// zonas seguras de cidade).
func startServerMobAI(t *testing.T, persist world.Persistence, gridDim int, mobX, mobY int16) (string, func()) {
	t.Helper()
	return startServerMobAIWith(t, persist, gridDim, aggressiveMob(), mobX, mobY)
}

// startServerMobAIWith é startServerMobAI com um template de criatura montado pelo chamador, para
// casos que precisam de um clã / Dex / equipamento EF_RANGE específico.
func startServerMobAIWith(t *testing.T, persist world.Persistence, gridDim int, tmpl []byte, mobX, mobY int16) (string, func()) {
	t.Helper()
	return startServerMobAIRouted(t, persist, gridDim, tmpl, mobX, mobY, nil)
}

// startServerMobAIRouted é startServerMobAIWith mais uma grade de caminhabilidade
// (nil = passo cego), espelhando a ligação de Config.Heights de PROD.
func startServerMobAIRouted(t *testing.T, persist world.Persistence, gridDim int, tmpl []byte, mobX, mobY int16, heights *content.Grid) (string, func()) {
	t.Helper()
	return startServerMobAISpawn(t, persist, gridDim, world.MobSpawn{Template: tmpl, X: mobX, Y: mobY, GenIndex: -1}, heights)
}

// startServerMobAISpawn é o harness de forma completa: a criatura vem de um
// MobSpawn completo (rota/waypoints incluídos) exatamente como o spawnNPCs de PROD o monta.
func startServerMobAISpawn(t *testing.T, persist world.Persistence, gridDim int, sp world.MobSpawn, heights *content.Grid) (string, func()) {
	t.Helper()
	return startServerMobAISpawns(t, persist, gridDim, []world.MobSpawn{sp}, heights)
}

func startServerMobAISpawns(t *testing.T, persist world.Persistence, gridDim int, spawns []world.MobSpawn, heights *content.Grid) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	clock := &atomic.Uint32{}
	clock.Store(serverTime)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log, Heights: heights})
	w := world.New(world.Config{GridDim: gridDim, Now: clock.Load}, log, persist, d.Handle)
	for _, sp := range spawns {
		w.SpawnMobAt(sp) // spawn em tempo de init (seguro no loop)
	}
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

func startServerMobAIGenerator(t *testing.T, persist world.Persistence, gridDim int, gen *world.Generator) (string, func(), []int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	clock := &atomic.Uint32{}
	clock.Store(serverTime)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: gridDim, Now: clock.Load}, log, persist, d.Handle)
	w.RegisterGenerators([]*world.Generator{gen})
	ids := w.GenerateMob(0)
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
	}, ids
}

func waitMobChat(t *testing.T, c net.Conn, mobID int, want string) {
	t.Helper()
	for i := 0; i < 12; i++ {
		h, payload, ok := readMaybeHeader(t, c)
		if !ok {
			continue
		}
		if h.Type != protocol.MsgMessageChat {
			continue
		}
		if int(h.ID) != mobID {
			t.Fatalf("chat HEADER.ID = %d, want mob %d", h.ID, mobID)
		}
		if len(payload) != protocol.MessageLength {
			t.Fatalf("chat payload length = %d, want %d", len(payload), protocol.MessageLength)
		}
		if got := cstr(payload); got != want {
			t.Fatalf("chat text = %q, want %q", got, want)
		}
		return
	}
	t.Fatalf("did not receive mob chat %q from %d", want, mobID)
}

func TestMobFightActionChat(t *testing.T) {
	tmpl := aggressiveMob()
	tmpl[16] = 2                                     // amistosa: sem aggro ambiente
	binary.LittleEndian.PutUint32(tmpl[92+16:], 800) // sobrevive ao primeiro golpe do jogador
	binary.LittleEndian.PutUint32(tmpl[92+24:], 800)
	gen := &world.Generator{
		MinuteGenerate: -1, MinGroup: 0, MaxGroup: 0, MaxNumMob: 1,
		SegX: [5]int16{6}, SegY: [5]int16{5},
		LeaderTmpl:  tmpl,
		FightAction: [4]string{"Lute!", "Lute!", "Lute!", "Lute!"},
	}
	addr, stop, ids := startServerMobAIGenerator(t, combatDB(), 16, gen)
	defer stop()
	if len(ids) != 1 {
		t.Fatalf("generated %d mobs, want 1", len(ids))
	}

	c := enterWorld(t, addr)
	defer c.Close()
	attackFrame(t, c, serverTime, ids[0], 0)

	waitMobChat(t, c, ids[0], "Lute!")
}

func TestMobDieActionChat(t *testing.T) {
	tmpl := aggressiveMob()
	tmpl[16] = 2                                   // amistosa: sem aggro ambiente
	binary.LittleEndian.PutUint32(tmpl[92+16:], 1) // primeiro golpe do jogador mata
	binary.LittleEndian.PutUint32(tmpl[92+24:], 1)
	gen := &world.Generator{
		MinuteGenerate: -1, MinGroup: 0, MaxGroup: 0, MaxNumMob: 1,
		SegX: [5]int16{6}, SegY: [5]int16{5},
		LeaderTmpl: tmpl,
		DieAction:  [4]string{"Adeus!", "Adeus!", "Adeus!", "Adeus!"},
	}
	addr, stop, ids := startServerMobAIGenerator(t, combatDB(), 16, gen)
	defer stop()
	if len(ids) != 1 {
		t.Fatalf("generated %d mobs, want 1", len(ids))
	}

	c := enterWorld(t, addr)
	defer c.Close()
	attackFrame(t, c, serverTime, ids[0], 0)

	waitMobChat(t, c, ids[0], "Adeus!")
}

func TestMobKillSendsDieActionChat(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	clock := &atomic.Uint32{}
	clock.Store(serverTime)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: 64, Now: clock.Load}, log, combatDB(), d.Handle)

	target := aggressiveMob()
	copy(target[0:16], "Target")
	target[16] = 5
	binary.LittleEndian.PutUint32(target[92+8:], 1)
	binary.LittleEndian.PutUint32(target[92+16:], 30)
	binary.LittleEndian.PutUint32(target[92+24:], 30)
	gen := &world.Generator{
		MinuteGenerate: -1, MinGroup: 0, MaxGroup: 0, MaxNumMob: 1,
		SegX: [5]int16{10}, SegY: [5]int16{10},
		LeaderTmpl: target,
		DieAction:  [4]string{"MobDown", "MobDown", "MobDown", "MobDown"},
	}
	w.RegisterGenerators([]*world.Generator{gen})
	ids := w.GenerateMob(0)
	if len(ids) != 1 {
		t.Fatalf("generated %d mobs, want 1", len(ids))
	}
	targetID := ids[0]

	guard := aggressiveMob()
	copy(guard[0:16], "Guard")
	guard[16] = 7
	binary.LittleEndian.PutUint32(guard[92+8:], 5000)
	guardID := w.SpawnMobAt(world.MobSpawn{Template: guard, X: 9, Y: 10, GenIndex: -1})
	if guardID < 0 {
		t.Fatal("SpawnMobAt guard failed")
	}

	w.SetTickHandler(10*time.Millisecond, d.Tick)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	defer func() {
		cancel()
		<-done
	}()

	c := enterWorld(t, ln.Addr().String())
	defer c.Close()
	actionFrameAt(t, c, serverTime, 8, 10)

	waitMobChat(t, c, targetID, "MobDown")
}

// actionFrameAt fixa o jogador em (x,y): _MSG_Action define a posição da entidade
// para o DESTINO do corpo (TargetX/TargetY, semântica legada de pMob.TargetX).
func actionFrameAt(t *testing.T, c net.Conn, tick uint32, x, y int16) {
	t.Helper()
	body := protocol.MsgActionBody{PosX: x, PosY: y, Speed: 30, TargetX: x, TargetY: y}
	wire, err := protocol.Encode(protocol.Header{Type: protocol.MsgAction, ClientTick: tick}, body.Encode(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write(wire); err != nil {
		t.Fatal(err)
	}
}

// TestMobAttacksAdjacentPlayer é o caso e2e da IA: uma criatura nascida ao lado de
// onde o jogador está o pega como alvo (FindPlayerNear), e como está adjacente
// o tick de IA a faz atacar corpo a corpo — o jogador recebe um
// MSG_Attack autoritativo do servidor nomeando a si mesmo como alvo danificado.
func TestMobAttacksAdjacentPlayer(t *testing.T) {
	addr, stop := startServerMobAI(t, combatDB(), 16, 6, 5) // criatura um tile a leste de (5,5)
	defer stop()

	c := enterWorld(t, addr) // conn do jogador 1
	defer c.Close()
	actionFrame(t, c, serverTime, 5) // fixar o jogador em (5,5), adjacente à criatura

	// Em poucos ticks a criatura deve atacar o jogador.
	for i := 0; i < 20; i++ {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			continue
		}
		if ty != protocol.MsgAttack {
			continue // pular o movimento/ruído da criatura
		}
		var body protocol.MsgAttackBody
		if err := body.Decode(payload); err != nil {
			t.Fatal(err)
		}
		if len(body.Dam) == 0 || body.Dam[0].TargetID != 1 {
			t.Fatalf("mob attack Dam = %+v, want target player conn 1", body.Dam)
		}
		if body.Dam[0].Damage <= 0 {
			t.Fatalf("mob attack damage = %d, want > 0", body.Dam[0].Damage)
		}
		return // sucesso
	}
	t.Fatal("player never received a mob MSG_Attack")
}

// TestMobAttackHeaderIsSceneField: o ataque da criatura precisa chegar ao jogador com
// HEADER.ID = ESCENE_FIELD (GetFunc.cpp GetAttack define sm->ID = ESCENE_FIELD). O
// cliente só põe a VÍTIMA no estado de morte a partir de um ataque de campo/cena; com
// HEADER.ID = a criatura, o jogador morto continuava agindo (atacando, auto-poção).
func TestMobAttackHeaderIsSceneField(t *testing.T) {
	addr, stop := startServerMobAI(t, combatDB(), 16, 6, 5) // criatura ao lado de (5,5)
	defer stop()

	c := enterWorld(t, addr) // conn do jogador 1
	defer c.Close()
	actionFrame(t, c, serverTime, 5) // fixar em (5,5), adjacente à criatura

	for i := 0; i < 20; i++ {
		h, _, ok := readMaybeHeader(t, c)
		if !ok || h.Type != protocol.MsgAttack {
			continue
		}
		if h.ID != protocol.IDScene {
			t.Fatalf("mob attack HEADER.ID = %d, want ESCENE_FIELD %d", h.ID, protocol.IDScene)
		}
		return // sucesso
	}
	t.Fatal("player never received a mob MSG_Attack")
}

// TestFriendlyClanMobDoesNotAttack: mesma montagem adjacente de
// TestMobAttacksAdjacentPlayer, mas o template da criatura carrega Clan 2 — amistoso
// a jogadores (clã 0) em g_pClanTable — então a IA nunca pode adquirir o jogador.
func TestFriendlyClanMobDoesNotAttack(t *testing.T) {
	tmpl := aggressiveMob()
	tmpl[16] = 2 // Clan @16: clanTable[2][0]==1 → amistoso a jogadores
	addr, stop := startServerMobAIWith(t, combatDB(), 16, tmpl, 6, 5)
	defer stop()

	c := enterWorld(t, addr)
	defer c.Close()
	actionFrame(t, c, serverTime, 5) // fixar em (5,5), adjacente à criatura

	// Com tick de 10ms uma criatura hostil ataca em ~50ms; algumas leituras com timeout
	// são um negativo confiável (mesma técnica de TestMobIgnoresPlayerInCity).
	for i := 0; i < 3; i++ {
		ty, _, ok := readMaybe(t, c)
		if ok && ty == protocol.MsgAttack {
			t.Fatal("friendly-clan mob attacked a player")
		}
	}
}

func TestGuardAttacksHostileMob(t *testing.T) {
	guard := aggressiveMob()
	copy(guard[0:16], "Guard")
	guard[16] = 7 // clanTable[7][5]==0, mas clanTable[7][0]==1 (amistoso a jogadores)
	const cs = 92
	binary.LittleEndian.PutUint32(guard[cs+8:], 5000) // um golpe limpo mata a criatura fraca

	hostile := aggressiveMob()
	copy(hostile[0:16], "Intruder")
	hostile[16] = 5
	binary.LittleEndian.PutUint32(hostile[cs+8:], 1)   // inofensivo se algum dia rodar um tick
	binary.LittleEndian.PutUint32(hostile[cs+16:], 30) // MaxHp
	binary.LittleEndian.PutUint32(hostile[cs+24:], 30) // Hp

	spawns := []world.MobSpawn{
		{Template: guard, X: 6, Y: 5, GenIndex: -1},
		{Template: hostile, X: 7, Y: 5, GenIndex: -1},
	}
	addr, stop := startServerMobAISpawns(t, combatDB(), 16, spawns, nil)
	defer stop()

	c := enterWorld(t, addr)
	defer c.Close()
	actionFrameAt(t, c, serverTime, 5, 5) // observador mantém as duas criaturas acordadas e em visão

	guardID := world.MaxUser
	hostileID := world.MaxUser + 1
	sawAttack := false
	for i := 0; i < 30; i++ {
		h, payload, ok := readMaybeHeaderRaw(t, c)
		if !ok {
			continue
		}
		switch h.Type {
		case protocol.MsgAttack:
			var body protocol.MsgAttackBody
			if err := body.Decode(payload); err != nil {
				t.Fatal(err)
			}
			if int(body.AttackerID) != guardID {
				continue
			}
			if len(body.Dam) == 0 || int(body.Dam[0].TargetID) != hostileID {
				t.Fatalf("guard attack Dam = %+v, want target mob %d", body.Dam, hostileID)
			}
			if body.Dam[0].Damage <= 0 {
				t.Fatalf("guard attack damage = %d, want > 0", body.Dam[0].Damage)
			}
			sawAttack = true
		case protocol.MsgRemoveMob:
			if int(h.ID) == hostileID && sawAttack {
				return
			}
		}
	}
	if !sawAttack {
		t.Fatal("guard never attacked the hostile mob")
	}
	t.Fatal("hostile mob was attacked but not removed")
}

func TestGuardAggrosHostileMobInCityAtViewEdge(t *testing.T) {
	guard := aggressiveMob()
	copy(guard[0:16], "Guard")
	guard[16] = 7 // guarda amistoso a jogadores, hostil ao clã 5
	const cs = 92
	binary.LittleEndian.PutUint32(guard[cs+8:], 5000)

	hostile := aggressiveMob()
	copy(hostile[0:16], "Intruder")
	hostile[16] = 5
	binary.LittleEndian.PutUint32(hostile[cs+8:], 1)
	binary.LittleEndian.PutUint32(hostile[cs+16:], 30)
	binary.LittleEndian.PutUint32(hostile[cs+24:], 30)

	spawns := []world.MobSpawn{
		{Template: guard, X: 2090, Y: 2095, GenIndex: -1},
		{Template: hostile, X: 2091, Y: 2095, GenIndex: -1},
	}
	addr, stop := startServerMobAISpawns(t, combatDB(), 2200, spawns, nil)
	defer stop()

	c := enterWorld(t, addr)
	defer c.Close()
	actionFrameAt(t, c, serverTime, 2075, 2095) // dentro de Armia, 15 tiles do guarda

	guardID := world.MaxUser
	hostileID := world.MaxUser + 1
	for i := 0; i < 30; i++ {
		h, payload, ok := readMaybeHeaderRaw(t, c)
		if !ok || h.Type != protocol.MsgAttack {
			continue
		}
		var body protocol.MsgAttackBody
		if err := body.Decode(payload); err != nil {
			t.Fatal(err)
		}
		if int(body.AttackerID) == guardID && len(body.Dam) > 0 && int(body.Dam[0].TargetID) == hostileID {
			return
		}
	}
	t.Fatal("guard did not acquire the hostile mob inside the visible city area")
}

// TestMobDistance fixa BASE_GetDistance (Basedef.cpp:6484): a
// g_pDistanceTable euclidiana arredondada para deltas ≤6, max+1 além disso — NÃO Chebyshev.
func TestMobDistance(t *testing.T) {
	tests := []struct {
		dx, dy int16
		want   int
	}{
		{0, 0, 0},
		{1, 0, 1},
		{1, 1, 1}, // adjacência diagonal ainda é 1
		{2, 2, 3}, // euclidiana arredondada, não Chebyshev 2
		{3, 3, 4}, // diagonal custa mais que Chebyshev (3) — linha 3 da tabela
		{4, 0, 4}, // tiro reto em 4
		{6, 6, 6}, // canto da tabela
		{7, 0, 8}, // além da tabela: max+1
		{3, 10, 11},
	}
	for _, tt := range tests {
		if got := mobDistance(0, 0, tt.dx, tt.dy); got != tt.want {
			t.Errorf("mobDistance(0,0,%d,%d) = %d, want %d", tt.dx, tt.dy, got, tt.want)
		}
		// A métrica é simétrica em sinal e ordem de eixos.
		if got := mobDistance(tt.dx, tt.dy, 0, 0); got != tt.want {
			t.Errorf("mobDistance(%d,%d,0,0) = %d, want %d", tt.dx, tt.dy, got, tt.want)
		}
	}
}

// TestBattleCode fixa a decisão de alcance do BattleProcessor (CMob.cpp:308-327),
// incluindo o detalhe de paridade de que o sorteio %100 é tirado só dentro do
// ramo de alcance. rng.NewSeeded(1) produz 41 como seu primeiro Intn(100).
func TestBattleCode(t *testing.T) {
	tests := []struct {
		name             string
		dis, reach, dex  int
		want             int
		wantRandConsumed bool
	}{
		{"melee adjacent attacks", 1, 1, 0, battleAttack, true},
		{"ranged in reach attacks (high dex)", 3, 4, 99, battleAttack, true}, // sorteio 41 ≤ 99
		{"ranged crowded backs away (low dex)", 3, 4, 0, battleRetreat, true},
		{"ranged target beyond 4 attacks despite low dex", 5, 6, 0, battleAttack, true},
		{"short reach never retreats", 2, 3, 0, battleAttack, true}, // reach<4 → sem ramo de recuo
		{"out of reach chases", 5, 4, 0, battleChase, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := rng.NewSeeded(1) // primeiro Intn(100) = 41
			if got := battleCode(r, tt.dis, tt.reach, tt.dex); got != tt.want {
				t.Errorf("battleCode(dis=%d,reach=%d,dex=%d) = %#x, want %#x", tt.dis, tt.reach, tt.dex, got, tt.want)
			}
			// Verificar que o sorteio de rand aconteceu exatamente quando o original o tira, comparando
			// com a posição do fluxo de um gerador novo.
			next, fresh := r.Rand(), rng.NewSeeded(1).Rand()
			consumed := next != fresh
			if consumed != tt.wantRandConsumed {
				t.Errorf("rand consumed = %v, want %v", consumed, tt.wantRandConsumed)
			}
		})
	}
}

// rangedMob é aggressiveMob com um efeito de instância EF_RANGE=4 em Equip[0] e
// Dex fixo — a forma de um template de arqueiro/mago (o alcance vem do equipamento,
// BASE_GetMobAbility).
func rangedMob(dex uint16) []byte {
	b := aggressiveMob()
	const cs = 92
	binary.LittleEndian.PutUint16(b[cs+36:], dex) // Dex (sorteio de recuo)
	binary.LittleEndian.PutUint16(b[140:], 1234)  // Equip[0].sIndex (qualquer item não-zero)
	b[142] = 27                                   // Eff[0] = EF_RANGE
	b[143] = 4                                    // valor 4
	return b
}

// TestRangedMobAttacksFromDistance: uma criatura cujo equipamento concede EF_RANGE=4 precisa
// atacar um jogador a 3 tiles SEM se aproximar (o original dispara a mesma
// mensagem de ataque do alcance — sem projétil, sem perseguição). Dex alto para o
// sorteio de manter distância nunca disparar.
func TestRangedMobAttacksFromDistance(t *testing.T) {
	addr, stop := startServerMobAIWith(t, combatDB(), 16, rangedMob(99), 8, 5)
	defer stop()

	c := enterWorld(t, addr) // conn do jogador 1
	defer c.Close()
	actionFrame(t, c, serverTime, 5) // fixar em (5,5): distância 3 ≤ alcance 4

	for i := 0; i < 20; i++ {
		h, payload, ok := readMaybeHeader(t, c)
		if !ok {
			continue
		}
		if h.Type == protocol.MsgAction {
			t.Fatal("ranged mob moved instead of attacking from reach")
		}
		if h.Type != protocol.MsgAttack {
			continue
		}
		var body protocol.MsgAttackBody
		if err := body.Decode(payload); err != nil {
			t.Fatal(err)
		}
		if body.PosX != 8 || body.PosY != 5 {
			t.Fatalf("attack came from (%d,%d), want the spawn (8,5) — mob should not chase", body.PosX, body.PosY)
		}
		if len(body.Dam) == 0 || body.Dam[0].TargetID != 1 || body.Dam[0].Damage <= 0 {
			t.Fatalf("ranged attack Dam = %+v, want damage on player conn 1", body.Dam)
		}
		return // sucesso
	}
	t.Fatal("player never received the ranged mob's MSG_Attack")
}

// BenchmarkTickIdle10k mede o custo por tick do mundo dormente: 10k
// criaturas ociosas e nenhum jogador em jogo — o estado estacionário comum. O gate de
// dormência (playerNear) precisa manter isto em microssegundos; sem ele toda criatura
// rodaria a varredura FindEnemyFromView de 81 células a cada segundo.
func BenchmarkTickIdle10k(b *testing.B) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	clock := &atomic.Uint32{}
	w := world.New(world.Config{GridDim: 1024, Now: clock.Load}, log, nil, d.Handle)
	tmpl := aggressiveMob()
	n := 0
	for y := int16(1); y < 1023 && n < 10000; y += 8 {
		for x := int16(1); x < 1023 && n < 10000; x += 8 {
			if w.SpawnMob(tmpl, x, y) >= 0 {
				n++
			}
		}
	}
	if n != 10000 {
		b.Fatalf("spawned %d mobs, want 10000", n)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clock.Add(1000)
		d.Tick(w)
	}
}

// heightsWith monta uma grade de caminhabilidade plana dim×dim com as células
// informadas bloqueadas (route.Blocked), espelhando um HeightMap embutido para testes de pathfinding.
func heightsWith(dim int, blocked ...[2]int16) *content.Grid {
	g := &content.Grid{Dim: dim, Data: make([]byte, dim*dim)}
	for _, c := range blocked {
		g.Data[int(c[1])*dim+int(c[0])] = route.Blocked
	}
	return g
}

// TestMobRoutesAroundObstacle: com a grade de altura ligada (forma de PROD), uma criatura
// perseguindo além de uma única célula bloqueada precisa desviar dela (os ramos de
// desvio do BASE_GetRoute) — nunca pisando na parede — e ainda alcançar e
// atacar o jogador.
func TestMobRoutesAroundObstacle(t *testing.T) {
	h := heightsWith(16, [2]int16{7, 5}) // parede entre a criatura (8,5) e o jogador (5,5)
	addr, stop := startServerMobAIRouted(t, combatDB(), 16, aggressiveMob(), 8, 5, h)
	defer stop()

	c := enterWorld(t, addr)
	defer c.Close()
	actionFrame(t, c, serverTime, 5) // fixar em (5,5)

	for i := 0; i < 40; i++ {
		h2, payload, ok := readMaybeHeader(t, c)
		if !ok {
			continue
		}
		switch h2.Type {
		case protocol.MsgAction:
			var mv protocol.MsgActionBody
			if err := mv.Decode(payload); err != nil {
				t.Fatal(err)
			}
			if mv.TargetX == 7 && mv.TargetY == 5 {
				t.Fatal("mob stepped onto the blocked cell (7,5)")
			}
		case protocol.MsgAttack:
			var atk protocol.MsgAttackBody
			if err := atk.Decode(payload); err != nil {
				t.Fatal(err)
			}
			if len(atk.Dam) > 0 && atk.Dam[0].TargetID == 1 {
				return // sucesso: desviou e alcançou o jogador
			}
		}
	}
	t.Fatal("mob never reached the player around the obstacle")
}

// TestMobHeldByWall: a caminhada gulosa em linha NÃO é A* — uma parede inteira entre a criatura e o
// jogador precisa segurar a criatura do seu lado (presa, conforme Route[0]==0), nunca atravessando
// a parede, nunca atacando. (Compare com TestMobRoutesAroundObstacle, onde um
// obstáculo de uma célula É evitável.)
func TestMobHeldByWall(t *testing.T) {
	var wall [][2]int16
	for y := int16(0); y < 16; y++ {
		wall = append(wall, [2]int16{7, y})
	}
	addr, stop := startServerMobAIRouted(t, combatDB(), 16, aggressiveMob(), 9, 5, heightsWith(16, wall...))
	defer stop()

	c := enterWorld(t, addr)
	defer c.Close()
	actionFrame(t, c, serverTime, 5) // fixar em (5,5), a oeste da parede

	for i := 0; i < 6; i++ {
		h2, payload, ok := readMaybeHeader(t, c)
		if !ok {
			continue
		}
		if h2.Type == protocol.MsgAttack {
			t.Fatal("mob attacked through a wall")
		}
		if h2.Type != protocol.MsgAction {
			continue
		}
		var mv protocol.MsgActionBody
		if err := mv.Decode(payload); err != nil {
			t.Fatal(err)
		}
		if mv.TargetX <= 7 {
			t.Fatalf("mob crossed the wall to x=%d", mv.TargetX)
		}
	}
}

// TestSetSegment fixa o port de SetSegment (CMob.cpp:494-620) por RouteType, sobre
// a lista comum só Start/Dest (waypoints [0] e [4], meios pulados por zero).
func TestSetSegment(t *testing.T) {
	base := func() *world.Entity {
		return &world.Entity{
			SegListX: [5]int16{100, 0, 0, 0, 200},
			SegListY: [5]int16{10, 0, 0, 0, 20},
		}
	}
	tests := []struct {
		name      string
		routeType uint8
		progress  int8
		dir       uint8
		wantRet   int
		wantProg  int8
		wantDir   uint8
		wantSegX  int16
	}{
		{"RT2 forward from home skips zeros to Dest", 2, 0, 0, 0, 4, 0, 200},
		{"RT2 at Dest turns around back to home", 2, 4, 0, 0, 0, 1, 100},
		{"RT2 at home going back restarts forward to Dest", 2, 0, 1, 0, 4, 0, 200},
		{"RT4 loops circularly Dest→home", 4, 4, 0, 0, 0, 0, 100},
		{"RT0 stops at the last waypoint", 0, 4, 0, 1, 4, 0, 200},
		{"RT3 finishes at home after the round trip", 3, 0, 1, 2, 0, 1, 100},
		{"RT6 always resets to home", 6, 4, 1, 0, 0, 0, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := base()
			e.RouteType, e.SegProgress, e.SegDir = tt.routeType, tt.progress, tt.dir
			e.WaitTicks = 9 // precisa ser limpo em toda escolha
			ret := setSegment(e)
			if ret != tt.wantRet || e.SegProgress != tt.wantProg || e.SegDir != tt.wantDir ||
				e.SegmentX != tt.wantSegX || e.WaitTicks != 0 {
				t.Errorf("setSegment: ret=%d prog=%d dir=%d segX=%d wait=%d, want ret=%d prog=%d dir=%d segX=%d wait=0",
					ret, e.SegProgress, e.SegDir, e.SegmentX, e.WaitTicks,
					tt.wantRet, tt.wantProg, tt.wantDir, tt.wantSegX)
			}
		})
	}
}

// TestMobRoamPatrolAndWait conduz mobRoam tick a tick num mundo não iniciado:
// a criatura pausa SegWait ticks no waypoint natal, depois caminha até Dest, e
// faz ping-pong de volta (RouteType 2) — a sequência de chegada/espera/avanço do
// StandingByProcessor (CMob.cpp:170-227).
func TestMobRoamPatrolAndWait(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	clock := &atomic.Uint32{}
	w := world.New(world.Config{GridDim: 32, Now: clock.Load}, log, nil, d.Handle)

	id := w.SpawnMobAt(world.MobSpawn{
		Template: aggressiveMob(),
		X:        5, Y: 5,
		RouteType: 2,
		SegX:      [5]int16{5, 0, 0, 0, 9},
		SegY:      [5]int16{5, 0, 0, 0, 5},
		SegWait:   [5]int16{2, 0, 0, 0, 0},
		GenIndex:  -1,
	})
	e := w.Entity(id)

	// O spawn pré-arma a pausa natal (WaitTicks = SegWait[0] = 2, o legado
	// WaitSec = SegmentWait[0] no generate, Server.cpp:3665).
	if e.WaitTicks != 2 {
		t.Fatalf("spawn WaitTicks = %d, want 2 (pre-armed home pause)", e.WaitTicks)
	}
	// Tick 1: na base, contando a pausa — sem movimento.
	d.mobRoam(w, id, e)
	if e.X != 5 || e.WaitTicks != 1 {
		t.Fatalf("tick1: pos/wait = (%d,%d)/%d, want (5,5)/1", e.X, e.Y, e.WaitTicks)
	}
	// Tick 2: pausa acabou → avançar até Dest e dar o primeiro passo a leste.
	d.mobRoam(w, id, e)
	if e.SegmentX != 9 || e.X != 6 {
		t.Fatalf("tick2: segX/pos = %d/(%d,%d), want 9/(6,5) (walking to Dest)", e.SegmentX, e.X, e.Y)
	}
	// Caminhar o resto até Dest.
	for i := 0; i < 3; i++ {
		d.mobRoam(w, id, e)
	}
	if e.X != 9 || e.Y != 5 {
		t.Fatalf("pos after walk = (%d,%d), want Dest (9,5)", e.X, e.Y)
	}
	// Chegou, sem espera em Dest: ping-pong de volta para a base no tick seguinte.
	d.mobRoam(w, id, e)
	if e.SegmentX != 5 || e.X != 8 || e.SegDir != 1 {
		t.Fatalf("turnaround: segX/pos/dir = %d/(%d,%d)/%d, want 5/(8,5)/1", e.SegmentX, e.X, e.Y, e.SegDir)
	}
}

// TestRouteType3DespawnsAtFinalWaypoint fixa o ramo RT3 do StandingByProcessor:
// quando uma criatura de rota 3 chega ao waypoint 4, ela devolve 0x10000 e o timer chama
// DeleteMob(index, 3). Ela não faz ping-pong de volta para a base e não fica ociosa.
func TestRouteType3DespawnsAtFinalWaypoint(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: 32}, log, nil, d.Handle)

	id := w.SpawnMobAt(world.MobSpawn{
		Template: aggressiveMob(),
		X:        5, Y: 5,
		RouteType: 3,
		SegX:      [5]int16{5, 0, 0, 0, 7},
		SegY:      [5]int16{5, 0, 0, 0, 5},
		GenIndex:  -1,
	})
	if id < 0 {
		t.Fatal("SpawnMobAt failed")
	}

	for i := 0; i < 4; i++ {
		e := w.Entity(id)
		if e == nil {
			break
		}
		d.mobRoam(w, id, e)
	}
	if e := w.Entity(id); e != nil {
		t.Fatalf("RT3 mob still alive at (%d,%d), want despawn at waypoint 4", e.X, e.Y)
	}
}

// TestMobReturnsHomeAfterCombat: uma criatura sem rota deslocada da âncora (ex.:
// seu alvo morreu em outro lugar) caminha de volta — o MOB_RETURN emergente (o
// código 16 do BattleProcessor original → GetNextPos(1)).
func TestMobReturnsHomeAfterCombat(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	clock := &atomic.Uint32{}
	w := world.New(world.Config{GridDim: 32, Now: clock.Load}, log, nil, d.Handle)

	id := w.SpawnMob(aggressiveMob(), 5, 5)
	e := w.Entity(id)
	w.SetEntityPos(id, 9, 8) // o combate a arrastou para longe

	for i := 0; i < 10 && (e.X != 5 || e.Y != 5); i++ {
		d.mobRoam(w, id, e)
	}
	if e.X != 5 || e.Y != 5 {
		t.Fatalf("mob did not return home, at (%d,%d)", e.X, e.Y)
	}
}

// TestPatrolVisibleToPlayer é a checagem de ligação e2e: uma patrulha de clã amistoso perto
// de um jogador precisa ser vista SE MOVENDO (MSG_Action transmitindo chegada a Dest e volta)
// — ou seja, o Tick roteia criaturas ociosas pelo mobRoam e o gate roamRadius
// acorda patrulhas perto de jogadores.
func TestPatrolVisibleToPlayer(t *testing.T) {
	tmpl := aggressiveMob()
	tmpl[16] = 2 // amistoso a jogadores: patrulha em vez de atacar
	addr, stop := startServerMobAISpawn(t, combatDB(), 16, world.MobSpawn{
		Template: tmpl,
		X:        8, Y: 5,
		RouteType: 2,
		SegX:      [5]int16{8, 0, 0, 0, 11},
		SegY:      [5]int16{5, 0, 0, 0, 5},
		GenIndex:  -1,
	}, nil)
	defer stop()

	c := enterWorld(t, addr)
	defer c.Close()
	actionFrame(t, c, serverTime, 5) // fixar em (5,5): dentro do roamRadius

	reachedDest, cameBack := false, false
	for i := 0; i < 60 && !cameBack; i++ {
		ty, payload, ok := readMaybe(t, c)
		if !ok || ty != protocol.MsgAction {
			continue
		}
		var mv protocol.MsgActionBody
		if err := mv.Decode(payload); err != nil {
			t.Fatal(err)
		}
		if mv.TargetX == 11 {
			reachedDest = true
		}
		if reachedDest && mv.TargetX < mv.PosX {
			cameBack = true // ping-pong: agora caminhando a oeste de novo
		}
	}
	if !reachedDest || !cameBack {
		t.Fatalf("patrol not observed: reachedDest=%v cameBack=%v", reachedDest, cameBack)
	}
}

// groupWorld monta um mundo não iniciado com um gerador registrado (líder +
// 2 seguidores, clã hostil 5) e gera seu grupo, devolvendo os ids.
func groupWorld(t *testing.T) (*Dispatcher, *world.World, []int) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	clock := &atomic.Uint32{}
	w := world.New(world.Config{GridDim: 64, Now: clock.Load}, log, nil, d.Handle)
	g := &world.Generator{
		MinuteGenerate: 1, MinGroup: 2, MaxGroup: 2, MaxNumMob: 9,
		SegX: [5]int16{20}, SegY: [5]int16{20},
		LeaderTmpl: aggressiveMob(), FollowerTmpl: aggressiveMob(),
	}
	w.RegisterGenerators([]*world.Generator{g})
	ids := w.GenerateMob(0)
	if len(ids) != 3 {
		t.Fatalf("generated %d mobs, want 3", len(ids))
	}
	return d, w, ids
}

// TestSetGroupBattleDragsGroup: acertar um seguidor põe todo o grupo de spawn
// no atacante — SetBattle (Server.cpp:8013) mais a propagação por PartyList
// (ProcessSecMinTimer.cpp:1882-1929). Um membro já lutando
// com outro alguém mantém seu alvo selecionado mas registra o novo inimigo.
func TestSetGroupBattleDragsGroup(t *testing.T) {
	_, w, ids := groupWorld(t)
	leader, f1, f2 := w.Entity(ids[0]), w.Entity(ids[1]), w.Entity(ids[2])
	f2.Target = 7 // já engajado em outro lugar — não pode ser arrastado de novo

	attacker := &world.Entity{ID: 3, X: 22, Y: 20} // um jogador perto do grupo
	setGroupBattle(w, ids[1], f1, attacker)

	if f1.Target != 3 || f1.Mode != world.MobCombat {
		t.Fatalf("struck follower Target/Mode = %d/%v, want 3/combat", f1.Target, f1.Mode)
	}
	if leader.Target != 3 {
		t.Fatalf("leader Target = %d, want 3 (dragged)", leader.Target)
	}
	if f2.Target != 7 {
		t.Fatalf("engaged follower Target = %d, want 7 (kept)", f2.Target)
	}
	if !enemyListContains(f2, 3) {
		t.Fatalf("engaged follower EnemyList = %v, want attacker 3 recorded", f2.EnemyList)
	}

	// Fora da caixa de arrasto de ±23 ninguém entra.
	leader.Target = 0
	far := &world.Entity{ID: 4, X: 60, Y: 60}
	setGroupBattle(w, ids[1], f1, far)
	if leader.Target != 0 {
		t.Fatalf("leader dragged from %d tiles away, want no drag", 40)
	}
}

func TestEnemyListSelectsNearestAndDropsInvalid(t *testing.T) {
	_, w, _ := groupWorld(t)
	seekerID := w.SpawnMobAt(world.MobSpawn{Template: aggressiveMob(), X: 10, Y: 10, GenIndex: -1})
	farID := w.SpawnMobAt(world.MobSpawn{Template: aggressiveMob(), X: 14, Y: 10, GenIndex: -1})
	nearID := w.SpawnMobAt(world.MobSpawn{Template: aggressiveMob(), X: 11, Y: 10, GenIndex: -1})
	seeker := w.Entity(seekerID)
	far := w.Entity(farID)
	near := w.Entity(nearID)

	addEnemyList(seeker, far)
	addEnemyList(seeker, near)
	selectTargetFromEnemyList(w, seeker)
	if seeker.Target != nearID {
		t.Fatalf("selected target = %d, want nearer mob %d", seeker.Target, nearID)
	}

	near.HP = 0
	selectTargetFromEnemyList(w, seeker)
	if seeker.Target != farID {
		t.Fatalf("selected after death = %d, want remaining mob %d", seeker.Target, farID)
	}
	if enemyListContains(seeker, nearID) {
		t.Fatalf("dead target %d still in EnemyList %v", nearID, seeker.EnemyList)
	}

	w.SetEntityPos(farID, 40, 40)
	selectTargetFromEnemyList(w, seeker)
	if seeker.Target != 0 {
		t.Fatalf("selected out-of-range target = %d, want 0", seeker.Target)
	}
	if enemyListContains(seeker, farID) {
		t.Fatalf("out-of-range target %d still in EnemyList %v", farID, seeker.EnemyList)
	}
}

func TestEnemyListTieSelectsLaterSlot(t *testing.T) {
	_, w, _ := groupWorld(t)
	seekerID := w.SpawnMobAt(world.MobSpawn{Template: aggressiveMob(), X: 10, Y: 10, GenIndex: -1})
	firstID := w.SpawnMobAt(world.MobSpawn{Template: aggressiveMob(), X: 12, Y: 10, GenIndex: -1})
	secondID := w.SpawnMobAt(world.MobSpawn{Template: aggressiveMob(), X: 10, Y: 12, GenIndex: -1})
	seeker := w.Entity(seekerID)

	addEnemyList(seeker, w.Entity(firstID))
	addEnemyList(seeker, w.Entity(secondID))
	selectTargetFromEnemyList(w, seeker)
	if seeker.Target != secondID {
		t.Fatalf("tie selected target = %d, want later slot %d", seeker.Target, secondID)
	}
}

func TestTickPromotesEnemyListWithoutCurrentTarget(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: 64}, log, nil, d.Handle)

	guard := aggressiveMob()
	guard[16] = 7
	guardID := w.SpawnMobAt(world.MobSpawn{Template: guard, X: 10, Y: 10, GenIndex: -1})
	target := aggressiveMob()
	target[16] = 5
	targetID := w.SpawnMobAt(world.MobSpawn{Template: target, X: 11, Y: 10, GenIndex: -1})
	guardEntity := w.Entity(guardID)
	addEnemyList(guardEntity, w.Entity(targetID))
	guardEntity.Target = 0
	guardEntity.Mode = world.MobIdle

	d.Tick(w)

	if guardEntity.Target != targetID {
		t.Fatalf("promoted target = %d, want %d from EnemyList %v", guardEntity.Target, targetID, guardEntity.EnemyList)
	}
	if guardEntity.Mode != world.MobCombat {
		t.Fatalf("guard mode = %v, want MobCombat", guardEntity.Mode)
	}
}

func kefraBossMob(dex uint16) []byte {
	b := aggressiveMob()
	copy(b[0:16], "Kefra")
	b[16] = 7 // hostil a criaturas do clã 5, amistoso a jogadores como os guardas
	binary.LittleEndian.PutUint16(b[92+36:], dex)
	return b
}

func TestKefraBossEnemySelectionUsesHalfGrid(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: 64}, log, nil, d.Handle)
	kefraID := w.SpawnMobAt(world.MobSpawn{Template: kefraBossMob(99), X: 10, Y: 10, GenIndex: world.KefraBossGenIndex})
	nearID := w.SpawnMobAt(world.MobSpawn{Template: aggressiveMob(), X: 25, Y: 10, GenIndex: -1}) // distância 15
	farID := w.SpawnMobAt(world.MobSpawn{Template: aggressiveMob(), X: 27, Y: 10, GenIndex: -1})  // distância 17
	kefra := w.Entity(kefraID)

	addEnemyList(kefra, w.Entity(nearID))
	selectTargetFromEnemyList(w, kefra)
	if kefra.Target != nearID {
		t.Fatalf("Kefra selected %d, want target inside HALFGRIDX %d", kefra.Target, nearID)
	}

	addEnemyList(kefra, w.Entity(farID))
	w.Entity(nearID).HP = 0
	selectTargetFromEnemyList(w, kefra)
	if kefra.Target != 0 {
		t.Fatalf("Kefra selected out-of-halfgrid target %d, want none", kefra.Target)
	}
	if enemyListContains(kefra, farID) {
		t.Fatalf("out-of-halfgrid target %d still in EnemyList %v", farID, kefra.EnemyList)
	}
}

func TestKefraBossAttacksAtFixedRange(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	clock := &atomic.Uint32{}
	clock.Store(serverTime)
	w := world.New(world.Config{GridDim: 64, Now: clock.Load}, log, nil, d.Handle)
	kefraID := w.SpawnMobAt(world.MobSpawn{Template: kefraBossMob(99), X: 10, Y: 10, GenIndex: world.KefraBossGenIndex})
	targetID := w.SpawnMobAt(world.MobSpawn{Template: aggressiveMob(), X: 25, Y: 10, GenIndex: -1})
	kefra, target := w.Entity(kefraID), w.Entity(targetID)
	beforeHP := target.HP
	addEnemyList(kefra, target)

	d.mobBattle(w, kefraID, kefra)

	if target.HP >= beforeHP {
		t.Fatalf("Kefra target HP = %d, want damage at fixed range 25 from %d", target.HP, beforeHP)
	}
	if kefra.X != 10 || kefra.Y != 10 {
		t.Fatalf("Kefra moved to (%d,%d), want fixed position (10,10)", kefra.X, kefra.Y)
	}
}

func TestKefraBossDoesNotRetreat(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	clock := &atomic.Uint32{}
	clock.Store(serverTime)
	w := world.New(world.Config{GridDim: 64, Now: clock.Load}, log, nil, d.Handle)
	kefraID := w.SpawnMobAt(world.MobSpawn{Template: kefraBossMob(0), X: 10, Y: 10, GenIndex: world.KefraBossGenIndex})
	targetID := w.SpawnMobAt(world.MobSpawn{Template: aggressiveMob(), X: 11, Y: 10, GenIndex: -1})
	kefra, target := w.Entity(kefraID), w.Entity(targetID)
	beforeHP := target.HP
	addEnemyList(kefra, target)

	d.mobBattle(w, kefraID, kefra)

	if kefra.X != 10 || kefra.Y != 10 {
		t.Fatalf("Kefra retreated to (%d,%d), want fixed position (10,10)", kefra.X, kefra.Y)
	}
	if target.HP != beforeHP {
		t.Fatalf("Kefra attacked while retreat roll should only hold position: HP %d, want %d", target.HP, beforeHP)
	}
}

func enemyListContains(e *world.Entity, target int) bool {
	for _, id := range e.EnemyList {
		if id == target {
			return true
		}
	}
	return false
}

// TestFollowerDoesNotSelfAggro (e2e): um SEGUIDOR hostil adjacente ao jogador
// precisa ficar passivo quando seu líder está fora de alcance — seguidores nunca rodam a
// varredura de aggro eles mesmos (StandingBy só varre para líderes, CMob.cpp:158); eles
// só entram pelo arrasto de grupo. Compare com TestMobAttacksAdjacentPlayer: a mesma
// criatura adjacente SEM vínculo de líder ataca.
func TestFollowerDoesNotSelfAggro(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	clock := &atomic.Uint32{}
	clock.Store(serverTime)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: 64, Now: clock.Load}, log, combatDB(), d.Handle)
	leaderID := w.SpawnMob(aggressiveMob(), 40, 40) // bem fora da janela de aggro do jogador
	followerID := w.SpawnMob(aggressiveMob(), 6, 5) // adjacente ao jogador
	w.Entity(followerID).Leader = leaderID          // o vínculo de grupo do GenerateMob
	w.SetTickHandler(10*time.Millisecond, d.Tick)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	defer func() {
		cancel()
		<-done
	}()

	c := enterWorld(t, ln.Addr().String())
	defer c.Close()
	actionFrame(t, c, serverTime, 5) // fixar em (5,5), adjacente ao seguidor

	for i := 0; i < 3; i++ {
		ty, _, ok := readMaybe(t, c)
		if ok && ty == protocol.MsgAttack {
			t.Fatal("follower self-aggroed an adjacent player (leader-only scan violated)")
		}
	}
}

// TestGroupFocusesAttacker (e2e, ligação de PROD): o mundo é populado por um
// gerador registrado (RegisterGenerators + GenerateMob, exatamente como o
// spawnNPCs), o jogador acerta UM seguidor, e todo o grupo de spawn entra
// na luta — o jogador recebe MSG_Attack de pelo menos duas criaturas distintas
// (o seguidor atingido mais um companheiro arrastado).
func TestGroupFocusesAttacker(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	clock := &atomic.Uint32{}
	clock.Store(serverTime)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: 64, Now: clock.Load}, log, combatDB(), d.Handle)
	tmpl := aggressiveMob()
	tmpl[16] = 2 // clã amistoso: ninguém dá aggro primeiro — só o arrasto de retaliação
	gen := &world.Generator{
		MinuteGenerate: -1, MinGroup: 2, MaxGroup: 2, MaxNumMob: 9,
		SegX: [5]int16{20}, SegY: [5]int16{20},
		LeaderTmpl: tmpl, FollowerTmpl: tmpl,
	}
	w.RegisterGenerators([]*world.Generator{gen})
	ids := w.GenerateMob(0)
	if len(ids) != 3 {
		t.Fatalf("generated %d mobs, want 3", len(ids))
	}
	// Acertar o LÍDER: os seguidores nascem em células adjacentes (anéis do
	// emptyCellNear), então os arrastados conseguem acertar sem traçar rota em volta da vítima (o
	// passo de fallback cego não tem desvio e travaria atrás dela).
	victim := w.Entity(ids[0])
	w.SetTickHandler(10*time.Millisecond, d.Tick)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	defer func() {
		cancel()
		<-done
	}()

	c := enterWorld(t, ln.Addr().String())
	defer c.Close()
	actionFrameAt(t, c, serverTime, victim.X-1, victim.Y) // adjacente ao líder
	attackFrame(t, c, serverTime, ids[0], 0)              // acertá-lo

	attackers := map[uint16]bool{}
	for i := 0; i < 40 && len(attackers) < 2; i++ {
		ty, payload, ok := readMaybe(t, c)
		if !ok || ty != protocol.MsgAttack {
			continue
		}
		var body protocol.MsgAttackBody
		if err := body.Decode(payload); err != nil {
			t.Fatal(err)
		}
		// Contar criaturas atacando o jogador (pular o eco do nosso próprio golpe).
		if int(body.AttackerID) >= world.MaxUser &&
			len(body.Dam) > 0 && body.Dam[0].TargetID == 1 {
			attackers[body.AttackerID] = true
		}
	}
	if len(attackers) < 2 {
		t.Fatalf("group drag: %d distinct mob attackers, want >= 2 (leader + follower)", len(attackers))
	}
}

// TestGenerateMobsTimer: o timer de minuto dispara um bloco na sua fase
// `minute % MinuteGenerate == idx % MinuteGenerate`
// (ProcessSecMinTimer.cpp:2727-2735) e respeita a saturação de MaxNumMob.
func TestGenerateMobsTimer(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	clock := &atomic.Uint32{}
	w := world.New(world.Config{GridDim: 64, Now: clock.Load}, log, nil, d.Handle)
	g := &world.Generator{
		MinuteGenerate: 2, MinGroup: 0, MaxGroup: 0, MaxNumMob: 2,
		SegX: [5]int16{20}, SegY: [5]int16{20},
		LeaderTmpl: aggressiveMob(),
	}
	w.RegisterGenerators([]*world.Generator{g})

	d.tickCount = 60 // minuto 1: 1%2 != 0%2 → sem disparo
	d.generateMobs(w)
	if g.CurrentNumMob != 0 {
		t.Fatalf("fired off-phase: CurrentNumMob = %d, want 0", g.CurrentNumMob)
	}
	d.tickCount = 120 // minuto 2: 2%2 == 0%2 → dispara (grupo só com líder)
	d.generateMobs(w)
	if g.CurrentNumMob != 1 {
		t.Fatalf("minute-2 fire: CurrentNumMob = %d, want 1", g.CurrentNumMob)
	}
	d.tickCount = 121 // não é fronteira de minuto → sem disparo
	d.generateMobs(w)
	d.tickCount = 240
	d.generateMobs(w)
	if g.CurrentNumMob != 2 {
		t.Fatalf("minute-4 fire: CurrentNumMob = %d, want 2", g.CurrentNumMob)
	}
	d.tickCount = 360 // saturado (2 >= MaxNumMob)
	d.generateMobs(w)
	if g.CurrentNumMob != 2 {
		t.Fatalf("saturated fire: CurrentNumMob = %d, want 2", g.CurrentNumMob)
	}
}

// TestApplyHp cobre o port de ApplyHp (Server.cpp:9530): a barra viva fecha no
// alvo ReqHp em passos de no máximo applyCasting, ReqHp é um alvo (nunca gasto),
// e um pedido acima do máximo é limitado ao máximo efetivo vivo.
func TestApplyHp(t *testing.T) {
	tests := []struct {
		name           string
		hp, maxHP, req int32
		wantHP, wantRq int32
		wantMoved      bool
	}{
		{"no debt", 1000, 1000, 1000, 1000, 1000, false},
		{"req below hp is a no-op", 800, 1000, 500, 800, 500, false},
		{"closes a small gap in one call", 900, 5000, 1000, 1000, 1000, true},
		{"caps at applyCasting per call", 0, 10000, 5000, 2000, 5000, true},
		{"req clamped to effective max", 900, 1000, 9999, 1000, 1000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &world.Entity{HP: tt.hp, MaxHP: tt.maxHP}
			s := &world.Session{Conn: 1, ReqHp: tt.req}
			if got := applyHp(s, e); got != tt.wantMoved {
				t.Errorf("applyHp moved = %v, want %v", got, tt.wantMoved)
			}
			if e.HP != tt.wantHP {
				t.Errorf("HP = %d, want %d", e.HP, tt.wantHP)
			}
			if s.ReqHp != tt.wantRq {
				t.Errorf("ReqHp = %d, want %d (a target, not a pool)", s.ReqHp, tt.wantRq)
			}
		})
	}
}

// TestApplyHpRampsOverTicks: uma dívida de 5000 HP entra em chamadas sucessivas em vez de
// de uma vez — é isso que faz uma poção subir visivelmente em vez de saltar.
func TestApplyHpRampsOverTicks(t *testing.T) {
	e := &world.Entity{HP: 0, MaxHP: 10000}
	s := &world.Session{Conn: 1, ReqHp: 5000}
	for i, want := range []int32{2000, 4000, 5000} {
		if !applyHp(s, e) {
			t.Fatalf("call %d: applyHp reported no movement", i+1)
		}
		if e.HP != want {
			t.Fatalf("call %d: HP = %d, want %d", i+1, e.HP, want)
		}
	}
	if applyHp(s, e) {
		t.Error("applyHp moved after reaching the target, want no movement")
	}
}

// TestDamageReqHp: um golpe que entra precisa baixar o alvo de cura, senão o próximo applyHp
// desfaria o dano e deixaria a vítima quase imune. Uma poção ainda em curso
// mantém a dívida que sobrevive ao golpe (_MSG_Attack.cpp:1638-1642).
func TestDamageReqHp(t *testing.T) {
	tests := []struct {
		name           string
		hp, maxHP, req int32
		dmg            int32
		wantReq        int32
	}{
		{"no pending heal: target tracks hp", 800, 1000, 1000, 200, 800},
		{"pending potion survives, minus the hit", 500, 5000, 1500, 200, 1300},
		{"damage beyond the target floors at hp", 300, 1000, 400, 9999, 300},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// O HP já foi decrementado pelo chamador antes de damageReqHp rodar.
			e := &world.Entity{HP: tt.hp, MaxHP: tt.maxHP}
			s := &world.Session{Conn: 1, ReqHp: tt.req}
			damageReqHp(s, e, tt.dmg)
			if s.ReqHp != tt.wantReq {
				t.Errorf("ReqHp = %d, want %d", s.ReqHp, tt.wantReq)
			}
			// A invariante que importa: o tick não pode ressuscitar o dano.
			if s.ReqHp < e.HP {
				t.Errorf("ReqHp %d fell below HP %d — setReqHp must floor it", s.ReqHp, e.HP)
			}
		})
	}
}

// TestDamageThenTickDoesNotHealBack é a invariante num só lugar: depois de um golpe, o
// tick de regen não pode restaurar o HP perdido.
func TestDamageThenTickDoesNotHealBack(t *testing.T) {
	e := &world.Entity{HP: 1000, MaxHP: 1000}
	s := &world.Session{Conn: 1, ReqHp: 1000}
	e.HP -= 300 // um golpe entra
	damageReqHp(s, e, 300)
	if applyHp(s, e) {
		t.Error("applyHp healed right after a hit — the victim would be near-immune")
	}
	if e.HP != 700 {
		t.Errorf("HP = %d, want the damage to stick at 700", e.HP)
	}
}

// TestApplyMpCapsAndClamps espelha TestApplyHp para a barra de mana (Server.cpp:9564).
func TestApplyMpCapsAndClamps(t *testing.T) {
	e := &world.Entity{MP: 0, MaxMP: 10000}
	s := &world.Session{Conn: 1, ReqMp: 9999}
	if !applyMp(s, e) {
		t.Fatal("applyMp reported no movement on a full-bar debt")
	}
	if e.MP != applyCasting {
		t.Errorf("MP = %d, want %d (applyCasting cap)", e.MP, applyCasting)
	}
	e = &world.Entity{MP: 90, MaxMP: 100}
	s = &world.Session{Conn: 1, ReqMp: 500}
	if !applyMp(s, e) {
		t.Fatal("applyMp reported no movement")
	}
	if e.MP != 100 || s.ReqMp != 100 {
		t.Errorf("MP/ReqMp = %d/%d, want 100/100 (clamped to effective max)", e.MP, s.ReqMp)
	}
}

// TestMobIgnoresPlayerInCity é a correção de segurança do respawn: uma criatura nascida logo
// ao lado de um jogador que está dentro de uma cidade NÃO pode atacar — a cidade é uma zona
// segura onde um jogador revivido se recupera. (Compare com TestMobAttacksAdjacentPlayer,
// onde a mesma montagem fora de uma cidade produz ataque.)
func TestMobIgnoresPlayerInCity(t *testing.T) {
	// O retângulo de Armia é (2052,2052)-(2171,2163); use uma grade que caiba nele.
	addr, stop := startServerMobAI(t, combatDB(), 2200, 2090, 2095)
	defer stop()

	c := enterWorld(t, addr)
	defer c.Close()
	actionFrameAt(t, c, serverTime, 2089, 2095) // adjacente à criatura, dentro de Armia

	// Com tick de 10ms uma criatura com bug atacaria em ~50ms; algumas leituras (≈0,9s de
	// timeout) são um negativo confiável.
	for i := 0; i < 3; i++ {
		ty, _, ok := readMaybe(t, c)
		if ok && ty == protocol.MsgAttack {
			t.Fatal("mob attacked a player standing in a safe city")
		}
	}
}
