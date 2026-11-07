package handler

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

func TestSummonCount(t *testing.T) {
	tests := []struct {
		iv, evo, want int
	}{
		{1, 60, 2},  // Condor: Evocação/30
		{1, 29, 0},  // abaixo do primeiro limiar → nada é gerado (a conjuração devolve a mana)
		{2, 90, 3},  // Javali: /30
		{3, 120, 3}, // Lobo: /40
		{5, 79, 1},  // Tigre: /40
		{6, 160, 2}, // Gorila: /80
		{7, 79, 0},  // Dragão Negro: /80
		{8, 0, 1},   // Succubus: sempre exatamente um
		{9, 0, 1},   // valor 9 de Invocação Final: um template sem bônus
		{10, 400, 0},
	}
	for _, tt := range tests {
		if got := summonCount(tt.iv, tt.evo); got != tt.want {
			t.Errorf("summonCount(%d, %d) = %d, want %d", tt.iv, tt.evo, got, tt.want)
		}
	}
}

// summonTemplate monta um STRUCT_MOB de 816 bytes no formato do BaseSummon (os mesmos offsets
// do aggressiveMob): atributos fixos conhecidos para as asserções de escala pelo dono serem exatas.
func summonTemplate(name string) []byte {
	b := make([]byte, 816)
	copy(b[0:16], name)
	const cs = 92
	binary.LittleEndian.PutUint32(b[cs+0:], 10)   // Level
	binary.LittleEndian.PutUint32(b[cs+8:], 20)   // Damage
	binary.LittleEndian.PutUint32(b[cs+16:], 100) // MaxHp
	binary.LittleEndian.PutUint32(b[cs+24:], 100) // Hp
	binary.LittleEndian.PutUint16(b[cs+34:], 99)  // Int: nunca hesita em batalha
	return b
}

// summonDB é um BM pronto para evocar: skill 56 aprendida (bit 56%24=8), Evocação 60
// (BaseSpecial[2] → 2 Condores), Int 100 para a escala de atributos.
func summonDB(evocacao int16) *fakeDB {
	db := newDB()
	db.loadResult = world.CharacterState{
		Slot: 0, Name: "Beast", Class: 2, X: 5, Y: 5,
		HP: 1000, MaxHP: 1000, MP: 500, MaxMP: 500, Damage: 200, AC: 40,
		Level: 50, Int: 100, LearnedSkill: 1 << 8,
		BaseSpecial: [4]int16{0, 0, evocacao, 0},
	}
	return db
}

// evokeSpell é a linha de catálogo do Evocar Condor (skill 56: InstanceType 11,
// InstanceValue 1 → id de summon 0).
func evokeSpell() *content.SkillData {
	return content.NewSkillData([]content.Spell{
		{Index: 56, ManaSpent: 10, InstanceType: 11, InstanceValue: 1, MaxTarget: 1, Name: "Evocar Condor"},
	})
}

// startServerSummon monta um servidor com invocação: spells + templates de summon +
// o tick de IA (10ms), opcionalmente uma criatura extra gerada antes do loop.
func startServerSummon(t *testing.T, db world.Persistence, mob []byte, mobX, mobY int16) (string, func(), *atomic.Uint32) {
	mobs := [][]byte{summonTemplate("Condor")}
	return startServerSummonWith(t, db, evokeSpell(), mobs, mob, mobX, mobY)
}

func startServerSummonWith(t *testing.T, db world.Persistence, spells *content.SkillData, summonMobs [][]byte, mob []byte, mobX, mobY int16) (string, func(), *atomic.Uint32) {
	return startServerSummonTick(t, db, spells, summonMobs, mob, mobX, mobY, 10*time.Millisecond)
}

// startServerSummonTick é startServerSummonWith com o período do tick de IA exposto:
// a vida útil do summon é de 20 ticks de affectTickPeriod, então um tick lento mantém o pet
// vivo pelo teste inteiro e isola o que os handlers fazem do que a expiração faz.
func startServerSummonTick(t *testing.T, db world.Persistence, spells *content.SkillData, summonMobs [][]byte, mob []byte, mobX, mobY int16, tick time.Duration) (string, func(), *atomic.Uint32) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	clock := &atomic.Uint32{}
	clock.Store(serverTime)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log, Spells: spells, SummonMobs: summonMobs})
	w := world.New(world.Config{GridDim: 16, Now: clock.Load}, log, db, d.Handle)
	if mob != nil {
		w.SpawnMobAt(world.MobSpawn{Template: mob, X: mobX, Y: mobY, GenIndex: -1})
	}
	w.SetTickHandler(tick, d.Tick)
	w.SetSessionEndHandler(d.SessionEnd)
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
	}, clock
}

// petFromCreateMob extrai (mobID, nome, nível, dano, maxHP) de um
// corpo de MSG_CreateMob (offsets: createmob.go).
func petFromCreateMob(payload []byte) (id int, name string, level, damage, maxHP int32) {
	id = int(binary.LittleEndian.Uint16(payload[4:6]))
	name = cstr(payload[6:22])
	level = int32(binary.LittleEndian.Uint32(payload[124:128]))
	damage = int32(binary.LittleEndian.Uint32(payload[132:136]))
	maxHP = int32(binary.LittleEndian.Uint32(payload[140:144]))
	return
}

// collectPets lê frames por até timeout, devolvendo os CreateMobs de pet
// (nomes terminando no marcador legado '^').
func collectPets(t *testing.T, c net.Conn, timeout time.Duration) map[int][]byte {
	t.Helper()
	pets := make(map[int][]byte)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ty, payload, ok := readMaybeRaw(t, c)
		if !ok || ty != protocol.MsgCreateMob {
			continue
		}
		if _, name, _, _, _ := petFromCreateMob(payload); strings.HasSuffix(name, "^") {
			id, _, _, _, _ := petFromCreateMob(payload)
			pets[id] = payload
		}
	}
	return pets
}

func babyMountItem(index int16, hp uint16) world.Item {
	it := world.Item{Index: index}
	putShort(&it.Effects[0], hp)
	it.Effects[1] = world.Effect{Effect: 5, Value: 10} // bytes de sanc/life do mount
	it.Effects[2] = world.Effect{Effect: 30, Value: 1} // bytes de feed/kill
	return it
}

func babyMountDB(carry, equip world.Item) *fakeDB {
	db := newDB()
	st := world.CharacterState{
		Slot: 0, Name: "Hero", X: 5, Y: 5,
		HP: 1000, MaxHP: 1000, Level: 50,
	}
	st.Carry[0] = carry
	st.Equip[mountEquipSlot] = equip
	db.loadResult = st
	return db
}

func babyMountSummons() [][]byte {
	mobs := make([][]byte, 40)
	mob := summonTemplate("Porco")
	binary.LittleEndian.PutUint16(mob[140:], 315) // Equip[0] face; 315+(2330-2330)
	mobs[10] = mob                                // MountProcess: 2330 - 2320
	return mobs
}

func startServerBabyMount(t *testing.T, db world.Persistence) (string, func(), *atomic.Uint32) {
	return startServerSummonWith(t, db, nil, babyMountSummons(), nil, 0, 0)
}

func expectPetRemove(t *testing.T, c net.Conn, petID int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		h, _, ok := readMaybeHeaderRaw(t, c)
		if !ok {
			continue
		}
		if h.Type == protocol.MsgRemoveMob && int(h.ID) == petID {
			return
		}
	}
	t.Fatalf("no RemoveMob for pet %d", petID)
}

func TestBabyMountSpawnsOnEquip(t *testing.T) {
	addr, stop, _ := startServerBabyMount(t, babyMountDB(babyMountItem(2330, 80), world.Item{}))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceEquip, mountEquipSlot, 0)
	expect(t, c, protocol.MsgTradingItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgSendItem)
	ue := expect(t, c, protocol.MsgUpdateEquip)
	if got := le16(ue[mountEquipSlot*2:]); got != 2330 {
		t.Fatalf("mount visual = %d, want baby mount item 2330", got)
	}

	pets := collectPets(t, c, time.Second)
	if len(pets) != 1 {
		t.Fatalf("baby mount pets = %d, want 1", len(pets))
	}
	for _, payload := range pets {
		if name := cstr(payload[6:22]); name != "Porco^" {
			t.Errorf("baby mount name = %q, want Porco^", name)
		}
		if face := le16(payload[22:24]); face != 315 {
			t.Errorf("baby mount face = %d, want 315", face)
		}
		if createType := le16(payload[172:174]); createType&3 != 3 {
			t.Errorf("baby mount CreateType = %#x, want summon appear bits", createType)
		}
		if hp := int32(le(payload[148:152])); hp != 80 {
			t.Errorf("baby mount HP = %d, want item HP 80", hp)
		}
	}
}

func TestBabyMountDeadDoesNotSpawn(t *testing.T) {
	addr, stop, _ := startServerBabyMount(t, babyMountDB(babyMountItem(2330, 0), world.Item{}))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceEquip, mountEquipSlot, 0)
	expect(t, c, protocol.MsgTradingItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgUpdateEquip)

	if pets := collectPets(t, c, 500*time.Millisecond); len(pets) != 0 {
		t.Fatalf("dead baby mount spawned %d pet(s), want none", len(pets))
	}
}

func TestBabyMountUnequipDespawnsPet(t *testing.T) {
	addr, stop, _ := startServerBabyMount(t, babyMountDB(babyMountItem(2330, 80), world.Item{}))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceEquip, mountEquipSlot, 0)
	expect(t, c, protocol.MsgTradingItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgUpdateEquip)
	pets := collectPets(t, c, time.Second)
	if len(pets) != 1 {
		t.Fatalf("baby mount pets = %d, want 1", len(pets))
	}
	petID := 0
	for id := range pets {
		petID = id
	}

	tradeItemFrame(t, c, world.ItemPlaceEquip, mountEquipSlot, world.ItemPlaceCarry, 1, 0)
	expect(t, c, protocol.MsgTradingItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgUpdateEquip)
	expectPetRemove(t, c, petID)
}

func TestBabyMountDoesNotExpireLikeSkillSummon(t *testing.T) {
	addr, stop, _ := startServerBabyMount(t, babyMountDB(babyMountItem(2330, 80), world.Item{}))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceEquip, mountEquipSlot, 0)
	expect(t, c, protocol.MsgTradingItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgUpdateEquip)
	pets := collectPets(t, c, time.Second)
	if len(pets) != 1 {
		t.Fatalf("baby mount pets = %d, want 1", len(pets))
	}
	petID := 0
	for id := range pets {
		petID = id
	}

	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		h, _, ok := readMaybeHeaderRaw(t, c)
		if !ok {
			continue
		}
		if h.Type == protocol.MsgRemoveMob && int(h.ID) == petID {
			t.Fatalf("baby mount pet %d expired; item-backed summons must not use Type-24 lifespan", petID)
		}
	}
}

func TestBabyMountSpawnsOnLogin(t *testing.T) {
	addr, stop, _ := startServerBabyMount(t, babyMountDB(world.Item{}, babyMountItem(2330, 80)))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	pets := collectPets(t, c, time.Second)
	if len(pets) != 1 {
		t.Fatalf("login baby mount pets = %d, want 1", len(pets))
	}
}

func TestBabyMountGrowsWhenPetKillsMob(t *testing.T) {
	mount := babyMountItem(2330, 80)
	mount.Effects[1].Effect = 5
	mount.Effects[2].Value = 1
	db := babyMountDB(world.Item{}, mount)

	mob := make([]byte, 816)
	copy(mob[0:16], "Rato")
	mob[16] = 5 // clã hostil
	const cs = 92
	binary.LittleEndian.PutUint32(mob[cs+0:], 6) // acima do nível 5 de montaria
	binary.LittleEndian.PutUint32(mob[cs+8:], 1)
	binary.LittleEndian.PutUint32(mob[cs+16:], 1)
	binary.LittleEndian.PutUint32(mob[cs+24:], 1)
	binary.LittleEndian.PutUint16(mob[cs+34:], 99)

	addr, stop, clock := startServerSummonWith(t, db, nil, babyMountSummons(), mob, 4, 3)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()
	actionFrameAt(t, c, serverTime, 5, 5)

	mobID := world.MaxUser
	petID := -1
	petStruck := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		clock.Add(300)
		h, payload, ok := readMaybeHeaderRaw(t, c)
		if !ok {
			continue
		}
		switch h.Type {
		case protocol.MsgCreateMob:
			if id, name, _, _, _ := petFromCreateMob(payload); strings.HasSuffix(name, "^") {
				petID = id
			}
		case protocol.MsgAttack:
			var b protocol.MsgAttackBody
			if err := b.Decode(payload); err != nil {
				continue
			}
			if petID >= 0 && int(b.AttackerID) == petID {
				for _, dam := range b.Dam {
					if int(dam.TargetID) == mobID {
						petStruck = true
					}
				}
			}
		case protocol.MsgRemoveMob:
			if int(h.ID) == mobID && petStruck {
				send(t, c, protocol.MsgCharacterLogout, nil)
				expect(t, c, protocol.MsgCNFCharacterLogout)
				save, n := db.lastSavedChar()
				if n == 0 {
					t.Fatal("character was not saved on logout")
				}
				savedMount, ok := savedItemAt(save.Equip, mountEquipSlot)
				if !ok {
					t.Fatal("saved equip slot 14 is empty; want baby mount")
				}
				if savedMount.Eff2 != 5 {
					t.Fatalf("saved mount level = %d, want 5", savedMount.Eff2)
				}
				if savedMount.EffV3 != 2 {
					t.Fatalf("saved mount growth = %d, want 2 after pet kill", savedMount.EffV3)
				}
				return
			}
		}
	}
	switch {
	case petID < 0:
		t.Fatal("no baby mount pet was summoned")
	case !petStruck:
		t.Fatal("baby mount pet never attacked the monster")
	default:
		t.Fatal("monster never died to the baby mount pet")
	}
}

// TestEvocationSpawnsScaledSummons é o caminho feliz: Evocação 60 evoca dois
// Condores, cada um escalado pelo dono conforme pSummonBonus[0] — Dano 20 + Int·80% +
// Evo·300% = 280, MaxHp 100 + Int·100% + Evo·400% = 440 — no nível do dono,
// nomeados com o marcador de pet.
func TestEvocationSpawnsScaledSummons(t *testing.T) {
	addr, stop, _ := startServerSummon(t, summonDB(60), nil, 0, 0)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	skillAttackFrame(t, c, serverTime, 1, 56, damSkill)

	pets := collectPets(t, c, time.Second)
	if len(pets) != 2 {
		t.Fatalf("pets spawned = %d, want 2 (Evocação 60 / 30)", len(pets))
	}
	for id, payload := range pets {
		if id < world.MaxUser {
			t.Errorf("pet id %d in the player range", id)
		}
		_, name, lvl, dmg, hp := petFromCreateMob(payload)
		if name != "Condor^" {
			t.Errorf("pet name = %q, want Condor^", name)
		}
		if lvl != 50 {
			t.Errorf("pet level = %d, want the owner's 50", lvl)
		}
		if dmg != 280 {
			t.Errorf("pet damage = %d, want 280 (20 + 100·80%% + 60·300%%)", dmg)
		}
		if hp != 440 {
			t.Errorf("pet maxHP = %d, want 440 (100 + 100·100%% + 60·400%%)", hp)
		}
	}
}

func TestEvocationSendsAddPartyForSummon(t *testing.T) {
	addr, stop, _ := startServerSummon(t, summonDB(30), nil, 0, 0)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	skillAttackFrame(t, c, serverTime, 1, 56, damSkill)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		ty, payload, ok := readMaybeRaw(t, c)
		if !ok || ty != protocol.MsgCNFAddParty {
			continue
		}
		if len(payload) != protocol.MsgCNFAddPartyBodySize {
			t.Fatalf("CNFAddParty payload = %d, want %d", len(payload), protocol.MsgCNFAddPartyBodySize)
		}
		leaderConn := binary.LittleEndian.Uint16(payload[0:2])
		partyID := binary.LittleEndian.Uint16(payload[8:10])
		name := strings.TrimRight(string(payload[10:26]), "\x00")
		if int(partyID) >= world.MaxUser {
			if leaderConn != 30000 {
				t.Fatalf("summon LeaderConn = %d, want 30000 for non-leader slot", leaderConn)
			}
			if name != "Condor^" {
				t.Fatalf("summon party name = %q, want Condor^", name)
			}
			return
		}
	}
	t.Fatal("no CNFAddParty frame for summoned pet")
}

func TestEvocationDoesNotDuplicateExistingSummon(t *testing.T) {
	addr, stop, _ := startServerSummon(t, summonDB(30), nil, 0, 0)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	skillAttackFrame(t, c, serverTime, 1, 56, damSkill)
	if pets := collectPets(t, c, 500*time.Millisecond); len(pets) != 1 {
		t.Fatalf("first cast pets = %d, want 1", len(pets))
	}

	skillAttackFrame(t, c, serverTime+1000, 1, 56, damSkill)
	if pets := collectPets(t, c, 500*time.Millisecond); len(pets) != 0 {
		t.Fatalf("second cast spawned %d more pets, want 0 because existing summon counts", len(pets))
	}
}

// TestEvocationRefundsWhenNothingSpawns: Evocação abaixo do limiar não gera
// nada, então a mana da conjuração volta (_MSG_Attack.cpp:830-834) — o eco do
// ataque carrega o MP intacto @40.
func TestEvocationRefundsWhenNothingSpawns(t *testing.T) {
	addr, stop, _ := startServerSummon(t, summonDB(20), nil, 0, 0)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	skillAttackFrame(t, c, serverTime, 1, 56, damSkill)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		ty, payload, ok := readMaybeRaw(t, c)
		if !ok {
			continue
		}
		if ty == protocol.MsgCreateMob {
			if _, name, _, _, _ := petFromCreateMob(payload); strings.HasSuffix(name, "^") {
				t.Fatalf("pet %q spawned with Evocação 20, want none", name)
			}
			continue
		}
		if ty != protocol.MsgAttack {
			continue
		}
		if mp := int32(binary.LittleEndian.Uint32(payload[40:44])); mp != 500 {
			t.Fatalf("post-cast MP = %d, want 500 (failed evocation must refund)", mp)
		}
		return
	}
	t.Fatal("no attack echo received")
}

// TestSummonExpires: a vida útil do Tipo-24 (20 ticks de affect) se esgota e o
// pet é removido (DeleteMob(idx,3), Server.cpp:5843).
func TestSummonExpires(t *testing.T) {
	// Evocação 30 → exatamente um pet, mantendo o wire silencioso.
	addr, stop, _ := startServerSummon(t, summonDB(30), nil, 0, 0)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	skillAttackFrame(t, c, serverTime, 1, 56, damSkill)
	pets := collectPets(t, c, 500*time.Millisecond)
	if len(pets) != 1 {
		t.Fatalf("pets = %d, want 1", len(pets))
	}
	var petID int
	for id := range pets {
		petID = id
	}

	// 20 decrementos de vida útil, um a cada 8 ticks de mundo (10ms cada) ≈ 1,6s.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		h, _, ok := readMaybeHeaderRaw(t, c)
		if !ok {
			continue
		}
		if h.Type == protocol.MsgRemoveMob && int(h.ID) == petID {
			return
		}
	}
	t.Fatal("summon never expired (no RemoveMob for the pet)")
}

// TestSummonGoneAfterRelogin: sair para a tela de personagem deixa o pet órfão;
// o próximo tick o remove, então reentrar no mundo não encontra pet (a limpeza
// preguiçosa de dono ausente no summonTick).
func TestSummonGoneAfterRelogin(t *testing.T) {
	addr, stop, _ := startServerSummon(t, summonDB(30), nil, 0, 0)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	skillAttackFrame(t, c, serverTime, 1, 56, damSkill)
	if pets := collectPets(t, c, 500*time.Millisecond); len(pets) != 1 {
		t.Fatalf("pets = %d, want 1", len(pets))
	}

	send(t, c, protocol.MsgCharacterLogout, nil)
	for { // drenar até a confirmação de logout
		ty, _, ok := readMaybeRaw(t, c)
		if !ok {
			t.Fatal("no logout confirmation")
		}
		if ty == protocol.MsgCNFCharacterLogout {
			break
		}
	}
	time.Sleep(100 * time.Millisecond) // alguns ticks de IA: o pet órfão é removido

	var body protocol.MsgCharacterLoginBody
	send(t, c, protocol.MsgCharacterLogin, body.Encode())
	if pets := collectPets(t, c, 500*time.Millisecond); len(pets) != 0 {
		t.Fatalf("pets after relogin = %d, want 0 (owner-gone cleanup)", len(pets))
	}
}

// --- issue #234: summons não podem sobreviver ao laço de grupo que os mantém ---

// trackPets dobra os frames do ciclo de vida do pet que chegam dentro do timeout em live: um
// CreateMob de pet adiciona um id, um RemoveMob (HEADER.ID) o retira. O que sobra em live é
// o que o cliente ainda tem na tela.
func trackPets(t *testing.T, c net.Conn, timeout time.Duration, live map[int]bool) map[int]bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		h, payload, ok := readMaybeHeaderRaw(t, c)
		if !ok {
			continue
		}
		switch h.Type {
		case protocol.MsgCreateMob:
			if id, name, _, _, _ := petFromCreateMob(payload); strings.HasSuffix(name, "^") {
				live[id] = true
			}
		case protocol.MsgRemoveMob:
			delete(live, int(h.ID))
		}
	}
	return live
}

// summonPartySrv é um servidor com invocação cujo tick de IA é mais lento que qualquer
// janela de asserção, então um pet só pode sumir porque um handler de grupo o removeu
// de forma síncrona — nunca pela contagem de vida útil ou pelo
// ceifador de órfãos do summonTick, que de outro modo mascararia uma varredura ausente.
func summonPartySrv(t *testing.T, db world.Persistence) (string, func()) {
	t.Helper()
	addr, stop, _ := startServerSummonTick(t, db, evokeSpell(),
		[][]byte{summonTemplate("Condor")}, nil, 0, 0, 5*time.Second)
	return addr, stop
}

// summonPartyDB dá ao líder (conta 7) Evocação 30 → 1 pet e ao membro
// (conta 11) 60 → 2. O orçamento de pets é compartilhado no grupo — generateSummon conta
// TODO pet na PartyList do líder, quem quer que o tenha evocado (Server.cpp:2991-2997) —
// então o membro precisa da permissão maior para caber um pet seu ao lado
// do pet do líder.
func summonPartyDB() *fakeDB {
	db := summonDB(30)
	member := db.loadResult
	member.BaseSpecial[2] = 60
	db.loads = map[int64]world.CharacterState{7: db.loadResult, 11: member}
	return db
}

// evokeOne conjura Evocar Condor pela conn e devolve o id do único pet.
func evokeOne(t *testing.T, c net.Conn, conn int) int {
	t.Helper()
	skillAttackFrame(t, c, serverTime, conn, 56, damSkill)
	pets := collectPets(t, c, 400*time.Millisecond)
	if len(pets) != 1 {
		t.Fatalf("conn %d evoked %d pets, want 1", conn, len(pets))
	}
	for id := range pets {
		return id
	}
	return 0
}

// TestSoloSummonDespawnsOnRemoveParty é o repro da issue-234. Um BM com pets
// tem Leader == 0 com os pets na PRÓPRIA PartyList, então o botão de sair vai
// para leaderLeaveParty — que antes zerava a lista e os deixava órfãos.
func TestSoloSummonDespawnsOnRemoveParty(t *testing.T) {
	addr, stop := summonPartySrv(t, summonDB(30))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	pet := evokeOne(t, c, 1)
	removePartyFrame(t, c, 1)
	// O tick de 5s põe a expiração (20 decrementos, um a cada 8 ticks) ~13min à frente, então este
	// RemoveMob só pode ser o da saída.
	expectPetRemove(t, c, pet)
}

// TestEvocationAfterLeavingPartyStaysCapped é a segunda metade do relato
// ("...e fazer novas evocações"): os órfãos antes eram invisíveis para a
// contagem de generateSummon, então uma nova conjuração empilhava um conjunto inteiro por cima.
func TestEvocationAfterLeavingPartyStaysCapped(t *testing.T) {
	addr, stop := summonPartySrv(t, summonDB(30))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	live := map[int]bool{evokeOne(t, c, 1): true}
	removePartyFrame(t, c, 1)
	trackPets(t, c, 400*time.Millisecond, live)
	if len(live) != 0 {
		t.Fatalf("pets alive after leaving = %v, want none", live)
	}
	skillAttackFrame(t, c, serverTime+1000, 1, 56, damSkill)
	trackPets(t, c, 400*time.Millisecond, live)
	if len(live) != 1 {
		t.Fatalf("pets alive after re-evoking = %d (%v), want 1 — evocations stacked", len(live), live)
	}
}

// partyWithPets: A (conn 1) lidera, B (conn 2) entra, cada um evoca um pet. Os
// ids devolvidos são visíveis para A, cujo stream as asserções leem.
func partyWithPets(t *testing.T, addr string) (a, b net.Conn, petA, petB int) {
	t.Helper()
	a = enterWorldAs(t, addr, "tester")
	b = enterWorldAs(t, addr, "tradeb")
	reqPartyFrame(t, a, 1, 2)
	expectPartyFrame(t, b, protocol.MsgSendReqParty)
	acceptPartyFrame(t, b, 1, "Beast")
	drainRaw(t, a)
	drainRaw(t, b)

	petA = evokeOne(t, a, 1)
	drainRaw(t, b)
	skillAttackFrame(t, b, serverTime, 2, 56, damSkill)
	pets := collectPets(t, a, 400*time.Millisecond) // A vê o pet do membro aparecer
	if len(pets) != 1 {
		t.Fatalf("member evoked %d pets as seen by the leader, want 1", len(pets))
	}
	for id := range pets {
		petB = id
	}
	drainRaw(t, b)
	return a, b, petA, petB
}

// TestSummonDespawnsWhenOwnerLeavesParty: os pets de um membro ficam na PartyList do
// LÍDER, então sair precisa levá-los — e só eles (Server.cpp:8185).
func TestSummonDespawnsWhenOwnerLeavesParty(t *testing.T) {
	addr, stop := summonPartySrv(t, summonPartyDB())
	defer stop()
	a, b, petA, petB := partyWithPets(t, addr)
	defer a.Close()
	defer b.Close()

	removePartyFrame(t, b, 2)
	live := trackPets(t, a, 600*time.Millisecond, map[int]bool{petA: true, petB: true})
	if live[petB] {
		t.Errorf("leaver's pet %d survived the party leave", petB)
	}
	if !live[petA] {
		t.Errorf("leader's own pet %d was removed; only the leaver's should go", petA)
	}
}

// TestSummonDespawnsWhenKicked é o lado do kick da mesma varredura.
func TestSummonDespawnsWhenKicked(t *testing.T) {
	addr, stop := summonPartySrv(t, summonPartyDB())
	defer stop()
	a, b, petA, petB := partyWithPets(t, addr)
	defer a.Close()
	defer b.Close()

	removePartyFrame(t, a, 2) // o líder expulsa o membro
	live := trackPets(t, a, 600*time.Millisecond, map[int]bool{petA: true, petB: true})
	if live[petB] {
		t.Errorf("kicked member's pet %d survived", petB)
	}
	if !live[petA] {
		t.Errorf("leader's own pet %d was removed by the kick", petA)
	}
}

// TestSummonDespawnsOnPartyDisband: dissolver leva todo pet da lista, não
// só os do líder — a divergência deliberada do Server.cpp:8242, que só
// zera Summoner e vazaria mobs sem dono aqui (party.go leaderLeaveParty).
func TestSummonDespawnsOnPartyDisband(t *testing.T) {
	addr, stop := summonPartySrv(t, summonPartyDB())
	defer stop()
	a, b, petA, petB := partyWithPets(t, addr)
	defer a.Close()
	defer b.Close()

	removePartyFrame(t, a, 1) // o líder sai → o grupo se dissolve
	live := trackPets(t, a, 600*time.Millisecond, map[int]bool{petA: true, petB: true})
	if len(live) != 0 {
		t.Fatalf("pets alive after the party dissolved = %v, want none", live)
	}
}

// TestSummonPartySlotClearedOnDespawn: DespawnMob só envia RemoveMob, então a
// varredura precisa dropar também a linha de grupo do pet, senão o cliente mantém o slot.
func TestSummonPartySlotClearedOnDespawn(t *testing.T) {
	addr, stop := summonPartySrv(t, summonDB(30))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	pet := evokeOne(t, c, 1)
	removePartyFrame(t, c, 1)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		h, payload, ok := readMaybeHeaderRaw(t, c)
		if !ok {
			continue
		}
		if h.Type == protocol.MsgRemoveParty && int(protocolLe16(payload[0:2])) == pet {
			return
		}
	}
	t.Fatalf("no RemoveParty row for pet %d; the client keeps a ghost party slot", pet)
}

// TestSummonAssistsAgainstMob: uma luta de criatura perto do dono puxa o pet
// (commandSummons) — o pet ataca a criatura (combate criatura-vs-criatura) e a
// criatura acaba morrendo por ele, sumindo com o RemoveMob de morte.
func TestSummonAssistsAgainstMob(t *testing.T) {
	// Uma criatura hostil fraca bem ao lado do jogador: ela engaja o jogador,
	// o pet do jogador defende.
	mob := make([]byte, 816)
	copy(mob[0:16], "Rato")
	mob[16] = 5 // clã hostil
	const cs = 92
	binary.LittleEndian.PutUint32(mob[cs+0:], 1)   // Level
	binary.LittleEndian.PutUint32(mob[cs+8:], 1)   // Damage (inofensivo)
	binary.LittleEndian.PutUint32(mob[cs+16:], 50) // MaxHp
	binary.LittleEndian.PutUint32(mob[cs+24:], 50) // Hp
	binary.LittleEndian.PutUint16(mob[cs+34:], 99) // Int

	// (4,3): adjacente a onde o pet nasce (freeCellNear escolhe (4,4)) e a duas
	// casas do jogador — o pet chega sem desviar de ninguém
	// (o passo cego sem alturas nunca contorna uma célula ocupada).
	addr, stop, clock := startServerSummon(t, summonDB(30), mob, 4, 3)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()
	actionFrameAt(t, c, serverTime, 5, 5) // registrar na grade ao lado da criatura

	skillAttackFrame(t, c, serverTime, 1, 56, damSkill)

	// Uma única vigilância desde a conjuração — a cadeia inteira (criatura atinge o dono →
	// pet comandado → pet atinge a criatura → a criatura morre) pode acontecer em poucos
	// ticks de IA de 10ms, então nenhum frame pode ser descartado enquanto "espera o pet".
	mobID := world.MaxUser // primeiro slot de NPC: a criatura pré-gerada
	petID := -1
	petStruck := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		// Avança o relógio congelado para as cadências de ataque (1000ms) continuarem disparando.
		clock.Add(300)
		h, payload, ok := readMaybeHeaderRaw(t, c)
		if !ok {
			continue
		}
		switch h.Type {
		case protocol.MsgCreateMob:
			if id, name, _, _, _ := petFromCreateMob(payload); strings.HasSuffix(name, "^") {
				petID = id
			}
		case protocol.MsgAttack:
			var b protocol.MsgAttackBody
			if err := b.Decode(payload); err != nil {
				continue
			}
			if petID >= 0 && int(b.AttackerID) == petID {
				for _, dam := range b.Dam {
					if int(dam.TargetID) == mobID {
						petStruck = true
					}
				}
			}
		case protocol.MsgRemoveMob:
			if int(h.ID) == mobID && petStruck {
				return // o pet derrubou a criatura
			}
		}
	}
	switch {
	case petID < 0:
		t.Fatal("no pet was summoned")
	case !petStruck:
		t.Fatal("pet never attacked the monster (assist/mob-vs-mob broken)")
	default:
		t.Fatal("monster never died to the pet")
	}
}
