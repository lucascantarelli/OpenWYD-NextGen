package world

import (
	"encoding/binary"
	"testing"
)

// TestRespawnMob cobre o loop de respawn em execução: matar uma criatura a enfileira, ela
// fica ausente até o atraso passar, então SpawnDueRespawns a recria em seu
// ponto de origem. Usa um relógio injetável para o atraso ser determinístico.
func TestRespawnMob(t *testing.T) {
	now := uint32(1000)
	w := New(Config{GridDim: 16, Now: func() uint32 { return now }}, slogDiscard(), nil, nil)

	id := w.SpawnMob(make([]byte, structMobTemplateSize), 5, 6)
	if id < MaxUser {
		t.Fatalf("SpawnMob = %d, want a mob id >= %d", id, MaxUser)
	}

	w.DespawnMob(id, 1) // morte → deve enfileirar um respawn
	if len(w.respawnQueue) != 1 {
		t.Fatalf("respawnQueue len = %d, want 1 after kill", len(w.respawnQueue))
	}

	// Antes do atraso passar: nada respawna e a entrada é retida.
	if ids := w.SpawnDueRespawns(now); len(ids) != 0 {
		t.Fatalf("SpawnDueRespawns before due = %v, want none", ids)
	}
	if len(w.respawnQueue) != 1 {
		t.Fatalf("respawnQueue len = %d, want entry retained before due", len(w.respawnQueue))
	}

	// No/após o atraso: a criatura respawna na origem de spawn e a fila drena.
	now += DefaultRespawnDelay
	ids := w.SpawnDueRespawns(now)
	if len(ids) != 1 {
		t.Fatalf("SpawnDueRespawns at due = %v, want 1 respawn", ids)
	}
	if len(w.respawnQueue) != 0 {
		t.Fatalf("respawnQueue len = %d, want drained after respawn", len(w.respawnQueue))
	}
	e := w.Entity(ids[0])
	if e == nil {
		t.Fatal("respawned entity missing")
	}
	if e.X != 5 || e.Y != 6 {
		t.Errorf("respawn at (%d,%d), want leash origin (5,6)", e.X, e.Y)
	}
	if got, ok := w.grid.MobAt(5, 6); !ok || int(got) != ids[0] {
		t.Errorf("grid cell after respawn = (%d,%v), want (%d,true)", got, ok, ids[0])
	}
}

// TestSpawnMobCopiesTemplateCoin: o STRUCT_MOB.Coin (@28) do template chega à
// criatura viva, no primeiro spawn e em todo respawn, para loot.GoldDrop ter uma
// base para sortear (o GenerateMob legado copia o template inteiro).
func TestSpawnMobCopiesTemplateCoin(t *testing.T) {
	now := uint32(1000)
	w := New(Config{GridDim: 16, Now: func() uint32 { return now }}, slogDiscard(), nil, nil)
	tmpl := make([]byte, structMobTemplateSize)
	binary.LittleEndian.PutUint32(tmpl[28:], 50)
	coin := func(id int) int32 {
		if e := w.Entity(id); e != nil {
			return e.Coin
		}
		return -1
	}
	id := w.SpawnMob(tmpl, 5, 6)
	if got := coin(id); got != 50 {
		t.Fatalf("spawned mob Coin = %d, want 50", got)
	}
	w.DespawnMob(id, 1)
	now += DefaultRespawnDelay
	ids := w.SpawnDueRespawns(now)
	if len(ids) != 1 {
		t.Fatalf("SpawnDueRespawns = %v, want 1 respawn", ids)
	}
	if got := coin(ids[0]); got != 50 {
		t.Fatalf("respawned mob Coin = %d, want 50", got)
	}
}

// TestRespawnSkipsNPC verifica que um NPC não-criatura (Merchant != 0) nunca é
// enfileirado para respawn — lojas/quest givers não morrem.
func TestRespawnSkipsNPC(t *testing.T) {
	w := New(Config{GridDim: 16}, slogDiscard(), nil, nil)

	id := w.SpawnMob(make([]byte, structMobTemplateSize), 5, 6)
	w.Entity(id).Merchant = 1 // marca como NPC mercador

	w.DespawnMob(id, 1)
	if len(w.respawnQueue) != 0 {
		t.Fatalf("respawnQueue len = %d, want 0 for an NPC", len(w.respawnQueue))
	}
}

// TestClearSeenAllOnDespawn verifica que o id morto é retirado do conjunto de visão
// de toda sessão, para um slot reutilizado disparar um CreateMob novo.
func TestClearSeenAllOnDespawn(t *testing.T) {
	w := New(Config{GridDim: 16}, slogDiscard(), nil, nil)
	s := &Session{Conn: 0}
	w.sessions[0] = s

	id := w.SpawnMob(make([]byte, structMobTemplateSize), 5, 6)
	if !w.MarkSeen(s, id) {
		t.Fatal("MarkSeen should return true the first time")
	}

	w.DespawnMob(id, 1)
	if _, ok := s.seen[id]; ok {
		t.Errorf("seen still contains %d after despawn, want cleared", id)
	}
}
