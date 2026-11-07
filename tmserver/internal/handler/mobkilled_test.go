package handler

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
	"openwyd-nextgen/tmserver/internal/worldcfg"
)

// TestAddClamp protege o acúmulo de HP/MP por nível contra overflow de int32 e
// o teto da engine (o loop de subida de nível pode somar num MaxHp já grande).
func TestAddClamp(t *testing.T) {
	tests := []struct {
		name          string
		v, inc, limit int32
		want          int32
	}{
		{"normal add", 800, 3, level.MaxHPCap, 803},
		{"clamp at cap", level.MaxHPCap - 1, 5, level.MaxHPCap, level.MaxHPCap},
		{"no negative", 0, -10, level.MaxHPCap, 0},
		{"overflow guarded", 2_000_000_000, 2_000_000_000, level.MaxHPCap, level.MaxHPCap},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addClamp(tt.v, tt.inc, tt.limit); got != tt.want {
				t.Errorf("addClamp(%d,%d,%d) = %d, want %d", tt.v, tt.inc, tt.limit, got, tt.want)
			}
		})
	}
}

// expMobTemplate monta um template de monstro STRUCT_MOB de 816 bytes (Merchant=0)
// com o nível, a recompensa de morte e o clã dados.
func expMobTemplate(lvl int32, exp int64, clan uint8) []byte {
	b := make([]byte, 816)
	copy(b[0:16], "Cobaia")
	b[16] = clan
	binary.LittleEndian.PutUint64(b[32:], uint64(exp))
	const cs = 92
	binary.LittleEndian.PutUint32(b[cs+0:], uint32(lvl))
	binary.LittleEndian.PutUint32(b[cs+16:], 100) // MaxHp
	binary.LittleEndian.PutUint32(b[cs+24:], 100) // Hp
	return b
}

func mobTemplateWithDrop(slot int, item world.Item) []byte {
	b := expMobTemplate(100, 0, 0)
	o := 268 + slot*8 // STRUCT_MOB.Carry[64]
	binary.LittleEndian.PutUint16(b[o:], uint16(item.Index))
	for i, ef := range item.Effects {
		b[o+2+i*2] = ef.Effect
		b[o+3+i*2] = ef.Value
	}
	return b
}

// mobKilledWorld monta um mundo sem serving mais um matador mortal destacado (sem
// sessão — os envios de pacote do grantExp são pulados, a matemática de estado é a mesma).
func mobKilledWorld(t *testing.T) (*Dispatcher, *world.World, *world.Entity) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: 16}, log, nil, d.Handle)
	killer := &world.Entity{
		ID: 0, Mode: world.MobUser, Name: "Heroi",
		Level: 1, ClassMaster: classMasterMortal,
		BaseMaxHP: 80, BaseMaxMP: 45, HP: 40, MaxHP: 80, MP: 20, MaxMP: 45,
		// Atributos base da classe TK, nenhum ponto alocado e sem equipamento, então o
		// CurrentScore vivo (Str/…) é igual ao BaseScore sem equipamento (BaseStr/…).
		Str: 8, Int: 4, Dex: 7, Con: 6,
		BaseStr: 8, BaseInt: 4, BaseDex: 7, BaseCon: 6,
	}
	return d, w, killer
}

// TestMobKilledGrantsExp é a issue #43 ponta a ponta: um personagem (mortal)
// de nível 1 padrão matando um mob de mesmo nível precisa ganhar a recompensa
// do ramo de campo geral — não o roteamento celestial quase zero, nem dezenas de níveis.
func TestMobKilledGrantsExp(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	mobID := w.SpawnMob(expMobTemplate(1, 1000, 0), 6, 5)
	if mobID < 0 {
		t.Fatal("SpawnMob failed")
	}

	d.mobKilled(w, killer, w.Entity(mobID))
	// 450*1000/31=14516 → ÷1 → ×0.6=8709 → eMob cap 1000 → Kefra down 500 → −15%.
	if killer.Exp != 425 {
		t.Errorf("killer.Exp = %d, want 425", killer.Exp)
	}
	if killer.Level != 1 {
		t.Errorf("killer.Level = %d, want 1 (425 < nextLevel[2]=1124)", killer.Level)
	}
}

// TestMobKilledLevelUp cruza um limiar e confere as concessões de subida de nível.
func TestMobKilledLevelUp(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	killer.Exp = 1100 // nextLevel[2]=1124: a próxima morte precisa cruzar exatamente um nível
	mobID := w.SpawnMob(expMobTemplate(1, 1000, 0), 6, 5)

	d.mobKilled(w, killer, w.Entity(mobID))
	if killer.Exp != 1525 {
		t.Fatalf("killer.Exp = %d, want 1525", killer.Exp)
	}
	if killer.Level != 2 {
		t.Fatalf("killer.Level = %d, want 2 (1525 < nextLevel[3]=1826)", killer.Level)
	}
	if killer.SkillBonus != 3 || killer.SpecialBonus != 2 {
		t.Errorf("SkillBonus/SpecialBonus = %d/%d, want 3/2", killer.SkillBonus, killer.SpecialBonus)
	}
	// Um TK na base da classe sem nada alocado recebe nível*5 pontos de atributo
	// livres (B10): 2 níveis → 10. Zero aqui é o bug do "sem pontos ao subir de nível".
	if killer.ScoreBonus != 10 {
		t.Errorf("ScoreBonus = %d, want 10 (level 2 × 5, nothing spent)", killer.ScoreBonus)
	}
	if killer.BaseMaxHP != 83 { // TK IncHP = 3
		t.Errorf("BaseMaxHP = %d, want 83", killer.BaseMaxHP)
	}
	if killer.HP != killer.MaxHP || killer.MP != killer.MaxMP || killer.MaxHP <= 0 {
		t.Errorf("HP/MP = %d/%d of %d/%d, want full heal", killer.HP, killer.MP, killer.MaxHP, killer.MaxMP)
	}
}

// TestMobKilledLevelUpPointsIgnoreEquipAttributes é a guarda de regressão do
// bug do "nenhum ponto de atributo ao subir de nível usando equipamento": BASE_GetBonusScorePoint
// precisa ler o BaseScore sem equipamento. Aqui os atributos vivos do CurrentScore estão
// inflados muito acima da base da classe pelo equipamento (Str 8→108), enquanto o
// BaseScore alocado está intacto. A concessão de pontos livres precisa ficar nível*5 (nada alocado);
// o código pré-correção alimentava o e.Str inflado, contava 100 pontos como "gastos", e
// entregava ZERO ao jogador.
func TestMobKilledLevelUpPointsIgnoreEquipAttributes(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	// +100 Str do equipamento: o CurrentScore diverge do BaseScore alocado.
	killer.Str = killer.BaseStr + 100
	killer.Exp = 1100 // cruza exatamente um nível (nextLevel[2]=1124)
	mobID := w.SpawnMob(expMobTemplate(1, 1000, 0), 6, 5)

	d.mobKilled(w, killer, w.Entity(mobID))
	if killer.Level != 2 {
		t.Fatalf("killer.Level = %d, want 2", killer.Level)
	}
	if killer.ScoreBonus != 10 {
		t.Errorf("ScoreBonus = %d, want 10 — equipment attribute bonus must not consume free points", killer.ScoreBonus)
	}
}

// TestMobKilledLevelUpPointsAfterAllocation: pontos já gastos no
// BaseScore reduzem a concessão (a identidade idempotente do BASE_GetBonusScorePoint),
// enquanto equipamento por cima deles continua não reduzindo. Nível 2 concede 10; 4 alocados
// em Str deixam 6 livres mesmo com equipamento inflando ainda mais o Str vivo.
func TestMobKilledLevelUpPointsAfterAllocation(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	killer.BaseStr = 8 + 4           // 4 pontos alocados acima da base do TK
	killer.Str = killer.BaseStr + 30 // mais equipamento por cima
	killer.Exp = 1100
	mobID := w.SpawnMob(expMobTemplate(1, 1000, 0), 6, 5)

	d.mobKilled(w, killer, w.Entity(mobID))
	if killer.ScoreBonus != 6 {
		t.Errorf("ScoreBonus = %d, want 6 (level 2 × 5 − 4 allocated)", killer.ScoreBonus)
	}
}

// TestMobKilledClan4NoExp: mobs de clã 4 ficam fora de toda a distribuição de EXP
// (MobKilled.cpp:402) mas ainda dropam ouro/itens.
func TestMobKilledClan4NoExp(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	mobID := w.SpawnMob(expMobTemplate(1, 1000, 4), 6, 5)

	d.mobKilled(w, killer, w.Entity(mobID))
	if killer.Exp != 0 {
		t.Errorf("killer.Exp = %d, want 0 for a clan-4 mob", killer.Exp)
	}
}

// TestMobKilledDropsTemplateGold: um mob gerado de um template com Coin pode
// pagar ouro (GoldDrop, MobKilled.cpp), sempre dentro da faixa da fórmula
// legada; um template sem Coin nunca paga.
func TestMobKilledDropsTemplateGold(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	gremlin := expMobTemplate(4, 0, 0)
	binary.LittleEndian.PutUint32(gremlin[28:], 50) // o Coin do Gremlin de fábrica
	paid := 0
	for i := 0; i < 60; i++ {
		before := killer.Coin
		d.mobKilled(w, killer, w.Entity(w.SpawnMob(gremlin, 6, 5)))
		if gain := killer.Coin - before; gain != 0 {
			// 4 × (rand(q+1) + q + Coin), q = (Coin+1)/4 = 12 → [248, 296].
			if gain < 248 || gain > 296 {
				t.Fatalf("gold gain = %d, want within [248, 296]", gain)
			}
			paid++
		}
	}
	if paid == 0 {
		t.Fatal("60 kills of a Coin=50 mob paid no gold")
	}

	d, w, killer = mobKilledWorld(t)
	for i := 0; i < 60; i++ {
		d.mobKilled(w, killer, w.Entity(w.SpawnMob(expMobTemplate(4, 0, 0), 6, 5)))
	}
	if killer.Coin != 0 {
		t.Fatalf("Coin=0 template paid %d gold", killer.Coin)
	}
}

func TestMobKilledDeliversCommonDropToCarry(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	want := world.Item{Index: 777, Effects: [3]world.Effect{
		{Effect: 43, Value: 7},
		{Effect: 2, Value: 20},
		{Effect: 3, Value: 5},
	}}
	mobID := w.SpawnMob(mobTemplateWithDrop(11, want), 6, 5) // slot 11 sempre dropa

	d.mobKilled(w, killer, w.Entity(mobID))

	if got := killer.Carry[0]; got != want {
		t.Fatalf("killer.Carry[0] = %+v, want %+v", got, want)
	}
	if gi := w.GroundItemAt(6, 5); gi != nil {
		t.Fatalf("common mob drop was incorrectly placed on the ground: %+v", gi)
	}
}

func TestMobKilledFullCarryLosesCommonDrop(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	for i := 0; i < baseCarrySlots; i++ {
		killer.Carry[i] = world.Item{Index: int16(1000 + i)}
	}
	mobID := w.SpawnMob(mobTemplateWithDrop(11, world.Item{Index: 777}), 6, 5)

	d.mobKilled(w, killer, w.Entity(mobID))

	for i := 0; i < baseCarrySlots; i++ {
		if killer.Carry[i].Index != int16(1000+i) {
			t.Fatalf("full Carry slot %d changed to %+v", i, killer.Carry[i])
		}
	}
	if gi := w.GroundItemAt(6, 5); gi != nil {
		t.Fatalf("full Carry fell back to ground: %+v", gi)
	}
}

func TestMobKilledRejectsNonDroppableCatalogEntries(t *testing.T) {
	for _, index := range []int16{390, 454, 30000} {
		t.Run(fmt.Sprintf("item_%d", index), func(t *testing.T) {
			d, w, killer := mobKilledWorld(t)
			mobID := w.SpawnMob(mobTemplateWithDrop(11, world.Item{Index: index}), 6, 5)

			d.mobKilled(w, killer, w.Entity(mobID))

			if !killer.Carry[0].Empty() {
				t.Fatalf("non-droppable item %d delivered as %+v", index, killer.Carry[0])
			}
		})
	}
}

func TestApplyWorldEventConfigUpdatesExpAndDropState(t *testing.T) {
	d, w, _ := mobKilledWorld(t)
	snap := worldcfg.Snapshot{
		Version: 9,
		Event: worldcfg.EventConfig{
			Enabled: true, ItemIndex: 777, Rate: 10,
			StartIndex: 100, CurrentIndex: 101, EndIndex: 200,
			Indexed: true, NoticeEnabled: true,
			DoubleExpEnabled: true, NewbieEventEnabled: true,
		},
	}

	d.applyWorldEventConfig(w, snap)
	if !d.expEvents.DoubleMode || !d.expEvents.NewbieEvent {
		t.Fatalf("exp events = %+v, want double+newbie enabled", d.expEvents)
	}
	if !w.NewbieEvent() {
		t.Fatal("world newbie spawn handicap was not enabled")
	}
	got := w.WorldEventConfig()
	if got.Version != 9 || !got.Enabled || got.ItemIndex != 777 || got.CurrentIndex != 101 || !got.Indexed {
		t.Errorf("world event config = %+v, want snapshot applied", got)
	}
}

func TestMobKilledWorldEventDropIndexed(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	w.SetWorldEventConfig(world.EventConfig{
		Version: 3, Enabled: true, ItemIndex: 777, Rate: 1,
		StartIndex: 100, CurrentIndex: 100, EndIndex: 101,
		Indexed: true,
	})
	mobID := w.SpawnMob(expMobTemplate(1, 1000, 0), 6, 5)

	d.mobKilled(w, killer, w.Entity(mobID))

	if got := w.WorldEventConfig().CurrentIndex; got != 101 {
		t.Fatalf("CurrentIndex = %d, want 101", got)
	}
	if gi := w.GroundItemAt(6, 5); gi != nil {
		t.Fatalf("event drop was placed on the ground: %+v", gi)
	}
	it := killer.Carry[0]
	if it.Index != 777 {
		t.Fatalf("event item index = %d, want 777 in carry slot 0", it.Index)
	}
	want := [2]world.Effect{{Effect: eventSerialHi, Value: 0}, {Effect: eventSerialLo, Value: 100}}
	if it.Effects[0] != want[0] || it.Effects[1] != want[1] || it.Effects[2].Effect != eventSerialRand {
		t.Errorf("event effects = %+v, want serial hi/lo and rand effect", it.Effects)
	}
}

func TestMobKilledWorldEventDropFullCarryDoesNotAdvance(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	for i := 0; i < baseCarrySlots; i++ {
		killer.Carry[i] = world.Item{Index: int16(1000 + i)}
	}
	w.SetWorldEventConfig(world.EventConfig{
		Version: 3, Enabled: true, ItemIndex: 777, Rate: 1,
		StartIndex: 100, CurrentIndex: 100, EndIndex: 101,
		Indexed: true,
	})
	mobID := w.SpawnMob(expMobTemplate(1, 1000, 0), 6, 5)

	d.mobKilled(w, killer, w.Entity(mobID))

	if got := w.WorldEventConfig().CurrentIndex; got != 100 {
		t.Fatalf("CurrentIndex = %d, want unchanged 100", got)
	}
	if gi := w.GroundItemAt(6, 5); gi != nil {
		t.Fatalf("full carry event placed ground item %+v", gi)
	}
}

func TestMobKilledWorldEventDropExhaustedDoesNothing(t *testing.T) {
	d, w, killer := mobKilledWorld(t)
	w.SetWorldEventConfig(world.EventConfig{
		Version: 3, Enabled: true, ItemIndex: 777, Rate: 1,
		StartIndex: 100, CurrentIndex: 101, EndIndex: 101,
		Indexed: true,
	})
	mobID := w.SpawnMob(expMobTemplate(1, 1000, 0), 6, 5)

	d.mobKilled(w, killer, w.Entity(mobID))

	if gi := w.GroundItemAt(6, 5); gi != nil {
		t.Fatalf("exhausted event placed ground item %+v", gi)
	}
	if got := w.WorldEventConfig().CurrentIndex; got != 101 {
		t.Errorf("CurrentIndex = %d, want unchanged 101", got)
	}
}

func TestApplyBabyMountKillGrowth(t *testing.T) {
	mount := babyMountItem(2330, 80)
	mount.Effects[1].Effect = 5
	mount.Effects[2].Value = 1

	changed, leveled := applyBabyMountKill(&mount, 6)
	if !changed || leveled {
		t.Fatalf("applyBabyMountKill changed/leveled = %v/%v, want true/false", changed, leveled)
	}
	if got := mount.Effects[2].Value; got != 2 {
		t.Errorf("growth = %d, want 2", got)
	}
	if got := mount.Effects[1].Effect; got != 5 {
		t.Errorf("level = %d, want unchanged 5", got)
	}
}

func TestApplyBabyMountKillLevelUp(t *testing.T) {
	mount := babyMountItem(2330, 80)
	mount.Effects[1].Effect = 5
	mount.Effects[2].Value = 29 // limiar para 2330 no nível 5 é 30.

	changed, leveled := applyBabyMountKill(&mount, 6)
	if !changed || !leveled {
		t.Fatalf("applyBabyMountKill changed/leveled = %v/%v, want true/true", changed, leveled)
	}
	if got := mount.Effects[1].Effect; got != 6 {
		t.Errorf("level = %d, want 6", got)
	}
	if got := mount.Effects[2].Value; got != 1 {
		t.Errorf("growth after level-up = %d, want 1", got)
	}
}

func TestApplyBabyMountKillDefaultThreshold(t *testing.T) {
	mount := babyMountItem(2336, 80)
	mount.Effects[1].Effect = 5
	mount.Effects[2].Value = 104 // limiar padrão é XP+100 = 105.

	changed, leveled := applyBabyMountKill(&mount, 6)
	if !changed || !leveled {
		t.Fatalf("applyBabyMountKill changed/leveled = %v/%v, want true/true", changed, leveled)
	}
	if got := mount.Effects[1].Effect; got != 6 {
		t.Errorf("level = %d, want 6", got)
	}
}

func TestApplyBabyMountKillRejectsIneligible(t *testing.T) {
	tests := []struct {
		name     string
		mount    world.Item
		mobLevel int
	}{
		{name: "not mount", mount: babyMountItem(2329, 80), mobLevel: 6},
		{name: "mob level too low", mount: babyMountItem(2330, 80), mobLevel: 5},
		{name: "level cap", mount: func() world.Item {
			it := babyMountItem(2330, 80)
			it.Effects[1].Effect = 100
			return it
		}(), mobLevel: 101},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := tt.mount
			changed, leveled := applyBabyMountKill(&tt.mount, tt.mobLevel)
			if changed || leveled {
				t.Fatalf("applyBabyMountKill changed/leveled = %v/%v, want false/false", changed, leveled)
			}
			if tt.mount != before {
				t.Errorf("mount changed from %+v to %+v", before, tt.mount)
			}
		})
	}
}

func TestBabyMountKillEligible(t *testing.T) {
	killer := &world.Entity{
		ID:          world.MaxUser,
		Clan:        summonClan,
		Summoner:    1,
		EquipVisual: [world.MaxEquip]uint16{babyMountFaceLo},
	}
	mob := &world.Entity{ID: world.MaxUser + 1, Clan: 0}
	if !babyMountKillEligible(killer, mob) {
		t.Fatal("babyMountKillEligible = false, want true")
	}

	for _, tc := range []struct {
		name string
		k    *world.Entity
		m    *world.Entity
	}{
		{"player killer", &world.Entity{ID: 1, Clan: summonClan, Summoner: 1, EquipVisual: killer.EquipVisual}, mob},
		{"non summon clan", &world.Entity{ID: world.MaxUser, Clan: 0, Summoner: 1, EquipVisual: killer.EquipVisual}, mob},
		{"non mount face", &world.Entity{ID: world.MaxUser, Clan: summonClan, Summoner: 1}, mob},
		{"player victim", killer, &world.Entity{ID: 2, Clan: 0}},
		{"summon victim", killer, &world.Entity{ID: world.MaxUser + 1, Clan: summonClan}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if babyMountKillEligible(tc.k, tc.m) {
				t.Fatal("babyMountKillEligible = true, want false")
			}
		})
	}
}

// startServerExpMob é um harness de serving com um monstro de EXP matável (1 HP)
// ao lado do spawn do jogador.
func startServerExpMob(t *testing.T, persist world.Persistence, mob []byte, configure ...func(*Dispatcher, *world.World)) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	clock := &atomic.Uint32{}
	clock.Store(serverTime)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: 16, Now: clock.Load}, log, persist, d.Handle)
	w.SpawnMob(mob, 6, 5) // adjacente ao spawn do jogador (5,5)
	for _, fn := range configure {
		fn(d, w)
	}
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

// TestKillGrantsExpOverWire é a regressão completa da issue #43 pelo wire:
// um personagem recém-carregado (o contrato do banco não carrega ClassMaster) mata um
// mob de mesmo nível e o eco de MSG_Attack precisa carregar a recompensa do caminho MORTAL. Se
// o carregamento de personagem deixar de assumir ClassMaster mortal por padrão, os
// divisores celestiais transformam exatamente este ganho em 0.
func TestKillGrantsExpOverWire(t *testing.T) {
	mob := expMobTemplate(10, 1000, 0)            // mesmo nível do char de skillCombatDB
	binary.LittleEndian.PutUint32(mob[92+16:], 1) // 1 HP: o primeiro corpo a corpo mata
	binary.LittleEndian.PutUint32(mob[92+24:], 1)
	addr, stop := startServerExpMob(t, skillCombatDB(0), mob)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	skillAttackFrame(t, c, serverTime, world.MaxUser, -1, -2) // corpo a corpo simples

	for i := 0; i < 10; i++ {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			t.Fatal("no MsgAttack echo after the killing blow")
		}
		if ty != protocol.MsgAttack {
			continue // pula o ruído de despawn etc
		}
		var got protocol.MsgAttackBody
		if err := got.Decode(payload); err != nil {
			t.Fatal(err)
		}
		// 450*1000/40=11250 → ÷1 → ×0.6 → eMob cap 1000 → Kefra down 500 → −15%.
		if got.CurrentExp != 425 {
			t.Fatalf("CurrentExp = %d, want 425 (mortal general-field reward)", got.CurrentExp)
		}
		return
	}
	t.Fatal("never received the MsgAttack echo")
}

func TestKillWorldEventDropSendsCarrySlot(t *testing.T) {
	mob := expMobTemplate(10, 1000, 0)
	binary.LittleEndian.PutUint32(mob[92+16:], 1)
	binary.LittleEndian.PutUint32(mob[92+24:], 1)
	addr, stop := startServerExpMob(t, skillCombatDB(0), mob, func(_ *Dispatcher, w *world.World) {
		w.SetWorldEventConfig(world.EventConfig{
			Version: 3, Enabled: true, ItemIndex: 777, Rate: 1,
			StartIndex: 100, CurrentIndex: 100, EndIndex: 101,
			Indexed: true,
		})
	})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	skillAttackFrame(t, c, serverTime, world.MaxUser, -1, -2)

	for i := 0; i < 12; i++ {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			t.Fatal("no MsgSendItem after world event drop")
		}
		if ty != protocol.MsgSendItem {
			continue
		}
		if place, slot, idx := le16(payload[0:2]), le16(payload[2:4]), le16(payload[4:6]); place != world.ItemPlaceCarry || slot != 0 || idx != 777 {
			t.Fatalf("event SendItem = place %d slot %d index %d, want carry/0/777", place, slot, idx)
		}
		if payload[6] != eventSerialHi || payload[7] != 0 ||
			payload[8] != eventSerialLo || payload[9] != 100 ||
			payload[10] != eventSerialRand {
			t.Fatalf("event SendItem effects = [%d:%d %d:%d %d:%d], want serial hi/lo/rand",
				payload[6], payload[7], payload[8], payload[9], payload[10], payload[11])
		}
		return
	}
	t.Fatal("never received world event SendItem")
}

func TestKillCommonMobDropSendsCarrySlot(t *testing.T) {
	want := world.Item{Index: 777, Effects: [3]world.Effect{
		{Effect: 43, Value: 7},
		{Effect: 2, Value: 20},
		{Effect: 3, Value: 5},
	}}
	mob := mobTemplateWithDrop(11, want) // slot 11 é o slot garantido do legado
	binary.LittleEndian.PutUint32(mob[92+16:], 1)
	binary.LittleEndian.PutUint32(mob[92+24:], 1)
	addr, stop := startServerExpMob(t, skillCombatDB(0), mob)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	skillAttackFrame(t, c, serverTime, world.MaxUser, -1, -2)

	for i := 0; i < 12; i++ {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			t.Fatal("no MsgSendItem after common mob drop")
		}
		if ty != protocol.MsgSendItem {
			continue
		}
		if place, slot, idx := le16(payload[0:2]), le16(payload[2:4]), le16(payload[4:6]); place != world.ItemPlaceCarry || slot != 0 || idx != uint16(want.Index) {
			t.Fatalf("common drop SendItem = place %d slot %d index %d, want carry/0/%d", place, slot, idx, want.Index)
		}
		for i, effect := range want.Effects {
			o := 6 + i*2
			if payload[o] != effect.Effect || payload[o+1] != effect.Value {
				t.Fatalf("common drop effect %d = %d:%d, want %d:%d", i, payload[o], payload[o+1], effect.Effect, effect.Value)
			}
		}
		return
	}
	t.Fatal("never received MsgSendItem for common mob drop")
}

// TestGrantExpTrainingField cobre a issue #383: mortes de um gerador
// do campo de treino pagam a recompensa sem evento vezes o multiplicador do campo, mas só para
// mortais que o campo ainda admite e só enquanto o multiplicador está ligado.
func TestGrantExpTrainingField(t *testing.T) {
	const mobExp, mult = 1000, 6
	field := level.TrainingFieldExpReward(mobExp, 10, 10, classMasterMortal, 0, mult)
	plain := level.SoloExpReward(mobExp, 10, 10, classMasterMortal, 0, level.ExpEvents{})
	if field <= plain {
		t.Fatalf("field reward %d not above plain %d", field, plain)
	}
	tests := []struct {
		name        string
		genIndex    int16
		killerLevel int32
		classMaster uint8
		mult        int64
		want        int64
	}{
		{name: "field generator mortal", genIndex: 0, killerLevel: 10, classMaster: classMasterMortal, mult: mult, want: field},
		{name: "ordinary generator", genIndex: 1, killerLevel: 10, classMaster: classMasterMortal, mult: mult, want: plain},
		{name: "no generator", genIndex: -1, killerLevel: 10, classMaster: classMasterMortal, mult: mult, want: plain},
		{name: "multiplier off", genIndex: 0, killerLevel: 10, classMaster: classMasterMortal, mult: 0, want: plain},
		{name: "stored level 35 is past the field", genIndex: 0, killerLevel: 35, classMaster: classMasterMortal, mult: mult,
			want: level.SoloExpReward(mobExp, 35, 10, classMasterMortal, 0, level.ExpEvents{})},
		{name: "non-mortal", genIndex: 0, killerLevel: 10, classMaster: classMasterArch, mult: mult,
			want: level.SoloExpReward(mobExp, 10, 10, classMasterArch, 0, level.ExpEvents{})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, w, killer := mobKilledWorld(t)
			w.RegisterGenerators([]*world.Generator{{TrainingField: true}, {}})
			d.fieldExp = tt.mult
			killer.Level, killer.ClassMaster = tt.killerLevel, tt.classMaster
			killer.Exp = level.NextLevelExp(tt.killerLevel - 1)
			before := killer.Exp
			d.grantExp(w, nil, killer, &world.Entity{Level: 10, Exp: mobExp, GenIndex: tt.genIndex})
			if got := killer.Exp - before; got != tt.want {
				t.Errorf("gain = %d, want %d", got, tt.want)
			}
		})
	}
}
