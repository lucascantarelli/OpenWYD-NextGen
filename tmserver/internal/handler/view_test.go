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

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// startServerView é startServerClock com uma grade do tamanho do chamador, para testes de
// janela de view que precisam de jogadores mais distantes que o mundo padrão de 16 células.
func startServerView(t *testing.T, persist world.Persistence, gridDim int) (string, func()) {
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

// actionFrameXY envia um frame de movimento com origem e destino explícitos.
func actionFrameXY(t *testing.T, c net.Conn, ty protocol.Type, tick uint32, fromX, fromY, toX, toY int16) {
	t.Helper()
	body := protocol.MsgActionBody{PosX: fromX, PosY: fromY, Speed: 30, TargetX: toX, TargetY: toY}
	copy(body.Route[:], []byte{1, 2, 3})
	wire, err := protocol.Encode(protocol.Header{Type: ty, ClientTick: tick}, body.Encode(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write(wire); err != nil {
		t.Fatal(err)
	}
}

// drainRaw esvazia os frames pendentes de uma conexão (ruído de visibilidade do login).
func drainRaw(t *testing.T, c net.Conn) {
	t.Helper()
	for {
		if _, _, ok := readMaybeRaw(t, c); !ok {
			return
		}
	}
}

// createMobFields extrai (PosX, PosY, MobID) de um corpo MSG_CreateMob.
func createMobFields(t *testing.T, payload []byte) (x, y int16, mobID int) {
	t.Helper()
	if len(payload) != 220 {
		t.Fatalf("CreateMob body = %d bytes, want 220", len(payload))
	}
	x = int16(binary.LittleEndian.Uint16(payload[0:2]))
	y = int16(binary.LittleEndian.Uint16(payload[2:4]))
	mobID = int(binary.LittleEndian.Uint16(payload[4:6]))
	return x, y, mobID
}

// TestActionDestinationAndType: a entidade de quem se move precisa cair no DESTINO
// (semântica pMob.TargetX/Y legada) e o frame repassado precisa manter seu
// Type original (Action2 continua 0x0366 no wire).
func TestActionDestinationAndType(t *testing.T) {
	addr, stop, _ := startServerClock(t, movementDB())
	defer stop()

	mover := enterWorld(t, addr) // conn 1, em (5,5)
	defer mover.Close()
	watcher := enterWorldAs(t, addr, "tradeb") // conn 2, em (5,5)
	defer watcher.Close()
	drainRaw(t, mover)
	drainRaw(t, watcher)

	actionFrameXY(t, mover, protocol.MsgAction2, serverTime, 5, 5, 8, 8)

	// O observador vê o frame de parada com seu type original.
	ty, payload, ok := readMaybeRaw(t, watcher)
	if !ok || ty != protocol.MsgAction2 {
		t.Fatalf("watcher got %#x ok=%v, want forwarded MsgAction2", ty, ok)
	}
	var got protocol.MsgActionBody
	if err := got.Decode(payload); err != nil {
		t.Fatal(err)
	}
	if got.TargetX != 8 || got.TargetY != 8 {
		t.Errorf("forwarded body target = (%d,%d), want (8,8)", got.TargetX, got.TargetY)
	}

	// A posição no servidor é o destino: um re-sync NoViewMob para quem
	// se move precisa voltar como CreateMob completo em (8,8).
	send(t, watcher, protocol.MsgNoViewMob, protocol.EncodeStandardParm(1))
	ty, payload, ok = readMaybeRaw(t, watcher)
	if !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("noViewMob reply = %#x ok=%v, want CreateMob", ty, ok)
	}
	x, y, mobID := createMobFields(t, payload)
	if x != 8 || y != 8 || mobID != 1 {
		t.Errorf("re-synced mover = id %d at (%d,%d), want id 1 at (8,8)", mobID, x, y)
	}
	// Jogadores também recebem seu estado de PK depois do snapshot.
	ty, payload, ok = readMaybeRaw(t, watcher)
	if !ok || ty != protocol.MsgPKInfo {
		t.Fatalf("frame after CreateMob = %#x ok=%v, want PKInfo", ty, ok)
	}
	if parm, _ := protocol.StandardParm(payload); parm != 0 {
		t.Errorf("PKInfo parm = %d, want 0", parm)
	}
}

// viewDeltaDB põe tester (conn 1) em (5,5) e tradeb (conn 2) em (5,45) —
// fora da janela de view de 16 células um do outro no login.
func viewDeltaDB() *fakeDB {
	db := newDB()
	mk := func(name string, x, y int16) world.CharacterState {
		return world.CharacterState{Slot: 0, Name: name, X: x, Y: y, HP: 1000, MaxHP: 1000}
	}
	db.loads = map[int64]world.CharacterState{7: mk("Hero", 5, 5), 11: mk("HeroB", 5, 45)}
	return db
}

// TestMoveViewDelta é a reconciliação do GridMulticast ponta a ponta: andar para
// dentro da view envia CreateMob+PKInfo (nos dois sentidos) antes do frame de movimento;
// andar para fora envia RemoveMob (nos dois sentidos); voltar para dentro recria —
// o que prova que o conjunto de vistos é limpo na remoção.
func TestMoveViewDelta(t *testing.T) {
	addr, stop := startServerView(t, viewDeltaDB(), 64)
	defer stop()

	a := enterWorld(t, addr) // conn 1 em (5,5)
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb") // conn 2 em (5,45)
	defer b.Close()
	drainRaw(t, a)
	drainRaw(t, b)

	// B anda até (5,21): a distância de Chebyshev até A vira 16 — entra em view.
	actionFrameXY(t, b, protocol.MsgAction, serverTime, 5, 45, 5, 21)

	// A: CreateMob(B) primeiro, depois PKInfo, depois o frame de movimento.
	ty, payload, ok := readMaybeRaw(t, a)
	if !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("A frame 1 = %#x ok=%v, want CreateMob", ty, ok)
	}
	if x, y, id := createMobFields(t, payload); id != 2 || x != 5 || y != 21 {
		t.Errorf("A sees B = id %d at (%d,%d), want id 2 at (5,21)", id, x, y)
	}
	// O snapshot precisa carregar o byte de velocidade do jogador (Score.AttackRun @body137):
	// um 0 aqui faz o cliente remoto animar o avatar em velocidade de rastejo e
	// dar borrachada para alcançar (o bug do "anda devagar + teleporta"). Os personagens
	// de teste do viewDeltaDB não têm montaria (starterEquip nunca concede o slot 14).
	if payload[124+13] != baseAttackRun {
		t.Errorf("A sees B with AttackRun %d, want %d (base player speed)", payload[124+13], baseAttackRun)
	}
	if ty, _, ok = readMaybeRaw(t, a); !ok || ty != protocol.MsgPKInfo {
		t.Fatalf("A frame 2 = %#x ok=%v, want PKInfo", ty, ok)
	}
	if ty, _, ok = readMaybeRaw(t, a); !ok || ty != protocol.MsgAction {
		t.Fatalf("A frame 3 = %#x ok=%v, want the forwarded Action", ty, ok)
	}

	// B (quem se move) fica sabendo de A simetricamente.
	ty, payload, ok = readMaybeRaw(t, b)
	if !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("B frame 1 = %#x ok=%v, want CreateMob", ty, ok)
	}
	if x, y, id := createMobFields(t, payload); id != 1 || x != 5 || y != 5 {
		t.Errorf("B sees A = id %d at (%d,%d), want id 1 at (5,5)", id, x, y)
	}
	if ty, _, ok = readMaybeRaw(t, b); !ok || ty != protocol.MsgPKInfo {
		t.Fatalf("B frame 2 = %#x ok=%v, want PKInfo", ty, ok)
	}

	// B anda de volta para fora, até (5,40): A recebe o frame (janela antiga) e depois RemoveMob;
	// B recebe RemoveMob(A).
	actionFrameXY(t, b, protocol.MsgAction, serverTime, 5, 21, 5, 40)
	if ty, _, ok = readMaybeRaw(t, a); !ok || ty != protocol.MsgAction {
		t.Fatalf("A frame = %#x ok=%v, want Action before RemoveMob", ty, ok)
	}
	h, payload, ok := readMaybeHeaderRaw(t, a)
	if !ok || h.Type != protocol.MsgRemoveMob || h.ID != 2 {
		t.Fatalf("A frame = %#x id=%d ok=%v, want RemoveMob for conn 2", h.Type, h.ID, ok)
	}
	if len(payload) != 4 || binary.LittleEndian.Uint32(payload) != 0 {
		t.Errorf("RemoveMob body = %v, want RemoveType 0", payload)
	}
	if h, _, ok = readMaybeHeaderRaw(t, b); !ok || h.Type != protocol.MsgRemoveMob || h.ID != 1 {
		t.Fatalf("B frame = %#x id=%d ok=%v, want RemoveMob for conn 1", h.Type, h.ID, ok)
	}

	// B volta a (5,21): um CreateMob novo precisa chegar (o conjunto de vistos foi limpo).
	actionFrameXY(t, b, protocol.MsgAction, serverTime, 5, 40, 5, 21)
	if ty, _, ok = readMaybeRaw(t, a); !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("A frame = %#x ok=%v, want a fresh CreateMob after re-entry", ty, ok)
	}
}

func TestCreateMobCarriesAffectIconsOnReveal(t *testing.T) {
	db := viewDeltaDB()
	heroB := db.loads[11]
	heroB.Affects = []world.Affect{{Type: 31, Time: 7}}
	db.loads[11] = heroB
	addr, stop := startServerView(t, db, 64)
	defer stop()

	a := enterWorld(t, addr) // conn 1 em (5,5)
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb") // conn 2 em (5,45), já carregando affect 31
	defer b.Close()
	drainRaw(t, a)
	drainRaw(t, b)

	actionFrameXY(t, b, protocol.MsgAction, serverTime, 5, 45, 5, 21)

	ty, payload, ok := readMaybeRaw(t, a)
	if !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("A frame = %#x ok=%v, want CreateMob(B)", ty, ok)
	}
	if _, _, id := createMobFields(t, payload); id != 2 {
		t.Fatalf("A sees mob id %d, want 2", id)
	}
	if got, want := binary.LittleEndian.Uint16(payload[54:]), protocol.PackAffect(protocol.AffectData{Type: 31, Time: 7}); got != want {
		t.Fatalf("CreateMob Affect[0] = %#x, want %#x", got, want)
	}
}

func TestPlayersCrossingViewOnMovement(t *testing.T) {
	addr, stop := startServerView(t, viewDeltaDB(), 64)
	defer stop()

	a := enterWorld(t, addr) // conn 1 em (5,5)
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb") // conn 2 em (5,45)
	defer b.Close()
	drainRaw(t, a)
	drainRaw(t, b)

	// A anda até a borda da view de B. Os dois clientes precisam conhecer o outro jogador
	// antes de qualquer frame de movimento ser animado.
	actionFrameXY(t, a, protocol.MsgAction, serverTime, 5, 5, 5, 29)

	ty, payload, ok := readMaybeRaw(t, b)
	if !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("B frame 1 = %#x ok=%v, want CreateMob(A)", ty, ok)
	}
	if x, y, id := createMobFields(t, payload); id != 1 || x != 5 || y != 29 {
		t.Fatalf("B sees A = id %d at (%d,%d), want id 1 at (5,29)", id, x, y)
	}
	if ty, _, ok = readMaybeRaw(t, b); !ok || ty != protocol.MsgPKInfo {
		t.Fatalf("B frame 2 = %#x ok=%v, want PKInfo(A)", ty, ok)
	}
	if ty, _, ok = readMaybeRaw(t, b); !ok || ty != protocol.MsgAction {
		t.Fatalf("B frame 3 = %#x ok=%v, want A movement", ty, ok)
	}

	ty, payload, ok = readMaybeRaw(t, a)
	if !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("A frame 1 = %#x ok=%v, want CreateMob(B)", ty, ok)
	}
	if x, y, id := createMobFields(t, payload); id != 2 || x != 5 || y != 45 {
		t.Fatalf("A sees B = id %d at (%d,%d), want id 2 at (5,45)", id, x, y)
	}
	if ty, _, ok = readMaybeRaw(t, a); !ok || ty != protocol.MsgPKInfo {
		t.Fatalf("A frame 2 = %#x ok=%v, want PKInfo(B)", ty, ok)
	}

	// B então se move pela view já estabelecida. A deve receber só o
	// frame de movimento, não um CreateMob duplicado.
	actionFrameXY(t, b, protocol.MsgAction, serverTime+1000, 5, 45, 5, 21)
	if ty, _, ok = readMaybeRaw(t, a); !ok || ty != protocol.MsgAction {
		t.Fatalf("A crossing frame = %#x ok=%v, want B movement only", ty, ok)
	}
	if ty, _, ok = readMaybeRaw(t, a); ok {
		t.Fatalf("A got extra frame %#x after in-view crossing, want no duplicate", ty)
	}
}

// readMaybeHeaderRaw é readMaybeRaw com o header completo, para asserções de RemoveMob
// (HEADER.ID é a entidade removida).
func readMaybeHeaderRaw(t *testing.T, c net.Conn) (protocol.Header, []byte, bool) {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var sz [2]byte
	if _, err := io.ReadFull(c, sz[:]); err != nil {
		return protocol.Header{}, nil, false
	}
	buf := make([]byte, int(sz[0])|int(sz[1])<<8)
	copy(buf, sz[:])
	if _, err := io.ReadFull(c, buf[2:]); err != nil {
		return protocol.Header{}, nil, false
	}
	h, payload, _, err := protocol.Decode(buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return h, payload, true
}

// passiveMob é o template de teste de IA com um clã amigável aos jogadores, então ele
// nunca engaja e fica parado — um alvo puro de visibilidade.
func passiveMob() []byte {
	b := aggressiveMob()
	b[16] = 2 // clã 2: amigável ao clã de jogador 0 em g_pClanTable
	return b
}

func TestTeleportReconcilesMobViewLikeGridMulticast(t *testing.T) {
	db := gmDB()
	db.loads[23] = world.CharacterState{Slot: 0, Name: "Victim", Class: 0, Level: 10, X: 40, Y: 40, HP: 1000, MaxHP: 1000}
	addr, stop := startServerMobAISpawns(t, db, 64, []world.MobSpawn{
		{Template: passiveMob(), X: 6, Y: 5, GenIndex: -1},
		{Template: passiveMob(), X: 41, Y: 40, GenIndex: -1},
	}, nil)
	defer stop()
	mod := enterWorldAs(t, addr, "mod")
	defer mod.Close()
	victim := enterWorldAs(t, addr, "victim")
	defer victim.Close()
	drainRaw(t, mod)
	drainRaw(t, victim)

	gmFrame(t, mod, "goto Victim")

	oldMobID, newMobID := world.MaxUser, world.MaxUser+1
	var sawJump, removedOldMob, createdNewMob bool
	var selfCreateMob, extraScore bool
	for {
		h, payload, ok := readMaybeHeaderRaw(t, mod)
		if !ok {
			break
		}
		switch h.Type {
		case protocol.MsgAction:
			var body protocol.MsgActionBody
			if err := body.Decode(payload); err != nil {
				t.Fatalf("decode teleport action: %v", err)
			}
			if h.ID == 1 && body.Effect == 1 && body.TargetX == 40 && body.TargetY == 40 {
				sawJump = true
			}
		case protocol.MsgRemoveMob:
			if int(h.ID) == oldMobID {
				removedOldMob = true
			}
		case protocol.MsgCreateMob:
			_, _, id := createMobFields(t, payload)
			if id == newMobID {
				createdNewMob = true
			}
			if id == 1 {
				selfCreateMob = true
			}
		case protocol.MsgUpdateScore:
			extraScore = true
		}
	}
	if !sawJump {
		t.Fatal("teleport did not send self jump action")
	}
	if !removedOldMob {
		t.Fatalf("teleport did not remove old-window mob %d from the client", oldMobID)
	}
	if !createdNewMob {
		t.Fatalf("teleport did not reveal new-window mob %d", newMobID)
	}
	if selfCreateMob {
		t.Fatal("teleport resent CreateMob for the player itself; legacy DoTeleport does not re-enter the world")
	}
	if extraScore {
		t.Fatal("teleport resent UpdateScore; legacy DoTeleport only reconciles movement/view")
	}
}

// TestNoViewMobRanges: um mob fora da janela de multicast de 16 células mas dentro do
// alcance GetInView de 33 células precisa voltar como CreateMob completo; além de 33 a
// resposta é RemoveMob.
func TestNoViewMobRanges(t *testing.T) {
	// Dentro do NoViewRange (distância 25).
	addr, stop := startServerMobAIWith(t, movementDB(), 64, passiveMob(), 5, 30)
	c := enterWorld(t, addr) // conn 1 em (5,5)
	drainRaw(t, c)
	send(t, c, protocol.MsgNoViewMob, protocol.EncodeStandardParm(int32(world.MaxUser)))
	ty, payload, ok := readMaybeRaw(t, c)
	if !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("in-range reply = %#x ok=%v, want CreateMob", ty, ok)
	}
	if x, y, id := createMobFields(t, payload); id != world.MaxUser || x != 5 || y != 30 {
		t.Errorf("re-synced mob = id %d at (%d,%d), want id %d at (5,30)", id, x, y, world.MaxUser)
	}
	// Nenhum PKInfo para mobs.
	if ty, _, ok := readMaybeRaw(t, c); ok && ty == protocol.MsgPKInfo {
		t.Error("mob re-sync must not send PKInfo")
	}
	c.Close()
	stop()

	// Além do NoViewRange (distância 40).
	addr, stop = startServerMobAIWith(t, movementDB(), 64, passiveMob(), 5, 45)
	defer stop()
	c = enterWorld(t, addr)
	defer c.Close()
	drainRaw(t, c)
	send(t, c, protocol.MsgNoViewMob, protocol.EncodeStandardParm(int32(world.MaxUser)))
	h, payload, ok := readMaybeHeaderRaw(t, c)
	if !ok || h.Type != protocol.MsgRemoveMob || int(h.ID) != world.MaxUser {
		t.Fatalf("out-of-range reply = %#x id=%d ok=%v, want RemoveMob", h.Type, h.ID, ok)
	}
	if len(payload) != 4 {
		t.Errorf("RemoveMob body = %d bytes, want 4", len(payload))
	}
}

// TestMoveCancelsTrade: uma troca em andamento é derrubada para as duas partes quando
// qualquer uma delas se move (_MSG_Action.cpp:41-50), e o próprio movimento é descartado.
func TestMoveCancelsTrade(t *testing.T) {
	addr, stop, _ := startServerClock(t, tradeDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()
	openTrade(t, a, b, world.Item{}, world.Item{}, 0, 0)

	actionFrameXY(t, a, protocol.MsgAction, serverTime, 5, 5, 7, 7)

	if ty, _, ok := readMaybe(t, a); !ok || ty != protocol.MsgQuitTrade {
		t.Errorf("A got %#x ok=%v, want QuitTrade", ty, ok)
	}
	if ty, _, ok := readMaybe(t, b); !ok || ty != protocol.MsgQuitTrade {
		t.Errorf("B got %#x ok=%v, want QuitTrade", ty, ok)
	}
	// O movimento foi engolido — nenhum Action alcança B.
	if ty, _, ok := readMaybe(t, b); ok {
		t.Errorf("B got %#x, want no forwarded Action after trade cancel", ty)
	}
}
