package handler

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/npccfg"
	"openwyd-nextgen/tmserver/internal/world"
)

// merchantTemplate monta um STRUCT_MOB mínimo de 816 bytes com
// CurrentScore.Merchant marcado (a flag de loja que o tmServer respeita, offset
// 104) e HP diferente de zero para o SpawnMobAt mantê-lo vivo.
func merchantTemplate(name string) []byte {
	tmpl := make([]byte, 816)
	copy(tmpl[0:16], name)
	tmpl[92+12] = 1                                  // CurrentScore.Merchant = normal shop
	binary.LittleEndian.PutUint32(tmpl[92+16:], 100) // MaxHp
	binary.LittleEndian.PutUint32(tmpl[92+24:], 100) // Hp
	return tmpl
}

// staticSource é um npccfg.Source fixo para acionar applyNPCConfig direto.
type staticSource struct{ snap npccfg.Snapshot }

func (s staticSource) Version(context.Context) (int64, error)            { return s.snap.Version, nil }
func (s staticSource) Snapshot(context.Context) (npccfg.Snapshot, error) { return s.snap, nil }

func newNPCDispatcher(base map[int]int32) (*Dispatcher, *world.World) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log, ItemPrices: base, NpcConfig: staticSource{}})
	w := world.New(world.Config{GridDim: 32}, log, world.NopPersistence{}, d.Handle)
	return d, w
}

// TestApplyNPCConfigSpawnsMerchant confere que aplicar no boot materializa uma
// definição de mercador habilitada: a entidade existe na posição, é mercador,
// carrega o estoque definido pelo moderador e o preço global sobreposto aplica.
func TestApplyNPCConfigSpawnsMerchant(t *testing.T) {
	d, w := newNPCDispatcher(map[int]int32{1100: 999})
	snap := npccfg.Snapshot{
		Version: 1,
		Defs: []npccfg.Definition{{
			Slug: "shop-1", Template: merchantTemplate("Keeper"), DisplayName: "Ferreiro", Enabled: true,
			X: 8, Y: 8, Merchant: 1,
			// Slots 0/2 são da aba 1 (identidade); slot 9 é da aba 2 → Carry[27]; slot
			// 18 é da aba 3 → Carry[54]. Prova que o mapeamento de ShopSlot aplica.
			Shop: []npccfg.ShopItem{
				{Slot: 0, Index: 1100, Quantity: 120}, {Slot: 2, Index: 1101},
				{Slot: 9, Index: 1102}, {Slot: 18, Index: 1103},
			},
		}},
		PriceOverrides: map[int]int32{1100: 500},
	}
	d.applyNPCConfig(w, snap, false)

	m, ok := d.managedNPCs["shop-1"]
	if !ok {
		t.Fatal("managed NPC not registered")
	}
	id := m.id
	e := w.Entity(id)
	if e == nil {
		t.Fatal("managed NPC entity missing")
	}
	if e.Merchant == 0 {
		t.Errorf("entity Merchant = 0, want merchant")
	}
	if e.X != 8 || e.Y != 8 {
		t.Errorf("entity at (%d,%d), want (8,8)", e.X, e.Y)
	}
	if e.Name != "Ferreiro" {
		t.Errorf("entity Name = %q, want Ferreiro", e.Name)
	}
	if e.Carry[0].Index != 1100 || e.Carry[2].Index != 1101 {
		t.Errorf("shop stock = [0]=%d [2]=%d, want 1100/1101", e.Carry[0].Index, e.Carry[2].Index)
	}
	if e.Carry[0].Effects[0].Effect != 61 || e.Carry[0].Effects[0].Value != 120 {
		t.Errorf("stack effect = %+v, want EF_AMOUNT 120", e.Carry[0].Effects[0])
	}
	// Abas 2/3 passam por ShopSlot: slot 9 → Carry[27], slot 18 → Carry[54].
	if e.Carry[27].Index != 1102 {
		t.Errorf("tab-2 slot 9 landed at Carry[27]=%d, want 1102", e.Carry[27].Index)
	}
	if e.Carry[54].Index != 1103 {
		t.Errorf("tab-3 slot 18 landed at Carry[54]=%d, want 1103", e.Carry[54].Index)
	}
	if e.Carry[9].Index != 0 || e.Carry[18].Index != 0 {
		t.Errorf("gap slots written: Carry[9]=%d Carry[18]=%d, want empty", e.Carry[9].Index, e.Carry[18].Index)
	}
	if e.Carry[1].Index != 0 {
		t.Errorf("slot 1 = %d, want empty", e.Carry[1].Index)
	}
	if d.itemPrices[1100] != 500 {
		t.Errorf("price[1100] = %d, want overridden 500", d.itemPrices[1100])
	}
	if d.npcVersion != 1 {
		t.Errorf("npcVersion = %d, want 1", d.npcVersion)
	}
}

func TestApplyNPCConfigDisplayNameFallbackAndSharedTemplate(t *testing.T) {
	d, w := newNPCDispatcher(nil)
	tmpl := merchantTemplate("Default")
	d.applyNPCConfig(w, npccfg.Snapshot{
		Version: 1,
		Defs: []npccfg.Definition{
			{Slug: "named-a", Template: tmpl, DisplayName: "Armeiro", Enabled: true, X: 7, Y: 7, Merchant: 1},
			{Slug: "named-b", Template: tmpl, DisplayName: "Alquimista", Enabled: true, X: 8, Y: 8, Merchant: 1},
			{Slug: "fallback", Template: tmpl, DisplayName: " ", Enabled: true, X: 9, Y: 9, Merchant: 1},
		},
	}, false)

	for slug, want := range map[string]string{
		"named-a":  "Armeiro",
		"named-b":  "Alquimista",
		"fallback": "Default",
	} {
		m, ok := d.managedNPCs[slug]
		if !ok {
			t.Fatalf("%s not spawned", slug)
		}
		if got := w.Entity(m.id).Name; got != want {
			t.Errorf("%s name = %q, want %q", slug, got, want)
		}
	}
	if got := string(tmpl[:7]); got != "Default" {
		t.Errorf("shared template mutated to %q, want Default", got)
	}
}

// TestApplyNPCConfigReconcile confere que um segundo snapshot que desabilita o NPC e
// limpa a sobreposição de preço despawna a entidade e restaura o preço base.
func TestApplyNPCConfigReconcile(t *testing.T) {
	d, w := newNPCDispatcher(map[int]int32{1100: 999})
	tmpl := merchantTemplate("Keeper")
	d.applyNPCConfig(w, npccfg.Snapshot{
		Version: 1,
		Defs: []npccfg.Definition{{
			Slug: "shop-1", Template: tmpl, Enabled: true, X: 8, Y: 8, Merchant: 1,
			Shop: []npccfg.ShopItem{{Slot: 0, Index: 1100}},
		}},
		PriceOverrides: map[int]int32{1100: 500},
	}, false)
	id := d.managedNPCs["shop-1"].id

	// Reload: NPC disabled, override gone.
	d.applyNPCConfig(w, npccfg.Snapshot{
		Version: 2,
		Defs: []npccfg.Definition{{
			Slug: "shop-1", Template: tmpl, Enabled: false, X: 8, Y: 8, Merchant: 1,
		}},
	}, false)

	if _, ok := d.managedNPCs["shop-1"]; ok {
		t.Error("disabled NPC still managed")
	}
	if w.Entity(id) != nil {
		t.Error("disabled NPC entity not despawned")
	}
	if d.itemPrices[1100] != 999 {
		t.Errorf("price[1100] = %d, want base 999 after override cleared", d.itemPrices[1100])
	}
	if d.npcVersion != 2 {
		t.Errorf("npcVersion = %d, want 2", d.npcVersion)
	}
}

// TestApplyNPCConfigKeepsUnchangedNPCs: um reload que edita UMA definição tem de
// deixar toda outra entidade viva exatamente como estava. A versão que apagava e
// recriava tudo a cada apply é o que duplicava
// cada mercador no cliente vivo (report in-game, Armia, 04/11/2026).
func TestApplyNPCConfigKeepsUnchangedNPCs(t *testing.T) {
	d, w := newNPCDispatcher(nil)
	tmpl := merchantTemplate("Keeper")
	base := npccfg.Snapshot{Version: 1, Defs: []npccfg.Definition{
		{Slug: "a", Template: tmpl, DisplayName: "A", Enabled: true, X: 6, Y: 6, Merchant: 1},
		{Slug: "b", Template: tmpl, DisplayName: "B", Enabled: true, X: 7, Y: 7, Merchant: 1},
	}}
	d.applyNPCConfig(w, base, false)
	kept := d.managedNPCs["a"].id

	edited := npccfg.Snapshot{Version: 2, Defs: []npccfg.Definition{
		{Slug: "a", Template: tmpl, DisplayName: "A", Enabled: true, X: 6, Y: 6, Merchant: 1},
		{Slug: "b", Template: tmpl, DisplayName: "B editado", Enabled: true, X: 7, Y: 7, Merchant: 1},
	}}
	d.applyNPCConfig(w, edited, true)

	if got := d.managedNPCs["a"].id; got != kept {
		t.Errorf("unchanged NPC despawned/respawned: id %d → %d", kept, got)
	}
	if e := w.Entity(kept); e == nil || e.Name != "A" {
		t.Errorf("unchanged NPC entity gone or renamed: %+v", e)
	}
	if e := w.Entity(d.managedNPCs["b"].id); e == nil || e.Name != "B editado" {
		t.Errorf("edited NPC not re-materialized with the new definition")
	}
	if len(d.managedNPCs) != 2 {
		t.Errorf("managed = %d, want 2", len(d.managedNPCs))
	}
}

// TestApplyNPCConfigSkipsInvalid confere que definições sem template ou sem
// posição são ignoradas em vez de spawnadas.
func TestApplyNPCConfigSkipsInvalid(t *testing.T) {
	d, w := newNPCDispatcher(nil)
	d.applyNPCConfig(w, npccfg.Snapshot{
		Version: 1,
		Defs: []npccfg.Definition{
			{Slug: "no-template", Enabled: true, X: 5, Y: 5, Merchant: 1},
			{Slug: "no-pos", Template: merchantTemplate("X"), Enabled: true, X: 0, Y: 0, Merchant: 1},
		},
	}, false)
	if len(d.managedNPCs) != 0 {
		t.Errorf("managed = %d, want 0 (both invalid)", len(d.managedNPCs))
	}
	_ = w
}

// --- nova tentativa de carga no boot ---
//
// Incidente de produção de 13/08/2026: tmServer e dbServer sobem juntos, e o
// dbServer reconcilia o catálogo de conteúdo ANTES de escutar, então o tmServer
// perde a corrida com frequência. O boot tem de esperar em vez de sair e queimar
// o orçamento de restart da plataforma — e ainda assim falhar rápido com dado corrompido.

// bootStep é uma resposta roteirizada do fakeBootSource.
type bootStep struct {
	snap npccfg.Snapshot
	err  error
}

// fakeBootSource reproduz respostas roteirizadas de Snapshot, uma por chamada,
// repetindo o último passo quando o roteiro acaba. Diferente do staticSource ele
// conta chamadas, que é como os testes distinguem nova tentativa de falha rápida.
type fakeBootSource struct {
	steps []bootStep
	calls int
}

func (f *fakeBootSource) Version(context.Context) (int64, error) { return 0, nil }

func (f *fakeBootSource) Snapshot(ctx context.Context) (npccfg.Snapshot, error) {
	f.calls++
	// O dbclient real falha assim com contexto de boot cancelado/expirado.
	if err := ctx.Err(); err != nil {
		return npccfg.Snapshot{}, err
	}
	step := f.steps[min(f.calls-1, len(f.steps)-1)]
	return step.snap, step.err
}

// shrinkBootRetry encolhe o orçamento de tentativas para os testes terminarem em
// milissegundos. Variáveis de pacote, restauradas no cleanup; sem t.Parallel aqui.
func shrinkBootRetry(t *testing.T) {
	t.Helper()
	budget, lo, hi := npcBootRetryBudget, npcBootRetryInitial, npcBootRetryMax
	npcBootRetryBudget, npcBootRetryInitial, npcBootRetryMax = 150*time.Millisecond, time.Millisecond, 2*time.Millisecond
	t.Cleanup(func() {
		npcBootRetryBudget, npcBootRetryInitial, npcBootRetryMax = budget, lo, hi
	})
}

// bootSnapshot cobre os n primeiros slots de gerador de conteúdo.
func bootSnapshot(version int64, n int) npccfg.Snapshot {
	snap := npccfg.Snapshot{Version: version}
	for i := 0; i < n; i++ {
		snap.Defs = append(snap.Defs, npccfg.Definition{
			Slug:           fmt.Sprintf("merchant-%d", i),
			Origin:         "content",
			GeneratorIndex: i,
		})
	}
	return snap
}

func TestFetchNPCConfigRetries(t *testing.T) {
	const content = 3
	unavailable := errors.New("connection refused")

	tests := []struct {
		name  string
		steps []bootStep
		// wantCalls é exato; 0 significa "só exigir que tenha tentado de novo".
		wantCalls int
		wantErr   bool
	}{
		{
			// As nove falhas de transporte do crash-loop de produção.
			name: "transport failure clears once dbServer listens",
			steps: []bootStep{
				{err: unavailable},
				{err: unavailable},
				{snap: bootSnapshot(7, content)},
			},
			wantCalls: 3,
		},
		{
			// A outra assinatura de produção: o dbServer respondeu no meio do seed, então o
			// catálogo voltou vazio em vez de recusar a conexão.
			name: "empty catalogue clears once the seed commits",
			steps: []bootStep{
				{snap: npccfg.Snapshot{Version: 7}},
				{snap: bootSnapshot(7, content)},
			},
			wantCalls: 2,
		},
		{
			name:      "succeeds on the first attempt",
			steps:     []bootStep{{snap: bootSnapshot(7, content)}},
			wantCalls: 1,
		},
		{
			// Índice de gerador duplicado é dado corrompido — esperar não corrige,
			// então o boot não pode gastar o orçamento antes de reportar.
			name: "duplicate generator index fails without retrying",
			steps: []bootStep{{snap: npccfg.Snapshot{Defs: []npccfg.Definition{
				{Slug: "a", Origin: "content", GeneratorIndex: 0},
				{Slug: "b", Origin: "content", GeneratorIndex: 0},
			}}}},
			wantCalls: 1,
			wantErr:   true,
		},
		{
			// O fail-closed é preservado: um dbServer que nunca volta ainda
			// para o boot em vez de subir com catálogo vazio.
			name:    "permanent failure still fails closed",
			steps:   []bootStep{{err: unavailable}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shrinkBootRetry(t)
			src := &fakeBootSource{steps: tt.steps}
			d := New(Config{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), NpcConfig: src})

			snap, err := d.FetchNPCConfig(context.Background())

			if tt.wantErr && err == nil {
				t.Fatal("FetchNPCConfig succeeded, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("FetchNPCConfig: %v", err)
			}
			switch {
			case tt.wantCalls > 0 && src.calls != tt.wantCalls:
				t.Errorf("Snapshot called %d times, want %d", src.calls, tt.wantCalls)
			case tt.wantCalls == 0 && src.calls < 2:
				t.Errorf("Snapshot called %d times, want more than one attempt", src.calls)
			}
			if !tt.wantErr && snap.Version != 7 {
				t.Errorf("snapshot version = %d, want 7", snap.Version)
			}
		})
	}
}

// TestFetchNPCConfigHonoursContext prova que SIGTERM durante o boot aborta a espera
// em vez de segurar o processo pelo orçamento inteiro. O orçamento fica
// no tamanho cheio de propósito: o cancelamento, não o orçamento, deve encerrar a espera.
func TestFetchNPCConfigHonoursContext(t *testing.T) {
	src := &fakeBootSource{steps: []bootStep{{err: errors.New("connection refused")}}}
	d := New(Config{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), NpcConfig: src})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if _, err := d.FetchNPCConfig(ctx); err == nil {
		t.Fatal("FetchNPCConfig succeeded on a cancelled context, want error")
	} else if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want it to wrap context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("returned after %s, want it to abort promptly", elapsed)
	}
}

// TestApplyNPCConfigBootRequiresRegisteredSlot: os slots vêm do mesmo snapshot
// que o boot aplica, então uma definição de conteúdo sem slot significa que os
// dois saíram de sincronia — o boot tem de falhar em vez de subir um mundo com buracos.
func TestApplyNPCConfigBootRequiresRegisteredSlot(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log, NpcConfig: staticSource{}})
	w := world.New(world.Config{GridDim: 32}, log, world.NopPersistence{}, d.Handle)
	snap := npccfg.Snapshot{Version: 3, Defs: []npccfg.Definition{{
		Slug: "a", Origin: "content", GeneratorIndex: 0, Enabled: true,
	}}}
	if err := d.ApplyNPCConfigBoot(w, snap); err == nil {
		t.Fatal("boot accepted a definition whose generator slot was never registered")
	}
	w.RegisterGenerators([]*world.Generator{{MinuteGenerate: -1, MaxNumMob: 1}})
	if err := d.ApplyNPCConfigBoot(w, snap); err != nil {
		t.Fatalf("ApplyNPCConfigBoot: %v", err)
	}
	if d.npcVersion != 3 {
		t.Errorf("npcVersion = %d, want the applied snapshot version 3", d.npcVersion)
	}
}
