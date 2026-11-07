//go:build integration

package store

import (
	"testing"

	"openwyd-nextgen/internal/domain"
)

func TestNightmareBalancePersistence(t *testing.T) {
	s, ctx := freshStore(t)
	accountID, err := s.SaveAccount(ctx, domain.Account{
		Name: "nightmare", PassHash: "$argon2id$test",
		Characters: []domain.Character{
			{Slot: 0, Name: "Survivor", NightmareEntries: 37, LastNightmareUse: 5000000000, Carry: []domain.Item{{Slot: 0, Index: 5137}}},
			{Slot: 1, Name: "Other", NightmareEntries: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ch, err := s.LoadCharacter(ctx, accountID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ch.NightmareEntries != 37 || ch.LastNightmareUse != 5000000000 {
		t.Fatalf("imported balance=%d, want 37", ch.NightmareEntries)
	}

	// Uma falha depois do UPDATE do personagem precisa reverter saldo e
	// inventário juntos. Slot é SMALLINT no PostgreSQL; este não pode ser salvo.
	ch.NightmareEntries = 137
	ch.LastNightmareUse = 5000000001
	ch.Carry = []domain.Item{{Slot: 40000, Index: 1100}}
	if err := s.SaveCharacter(ctx, accountID, ch); err == nil {
		t.Fatal("expected invalid item save to fail")
	}
	reloaded, err := s.LoadCharacter(ctx, accountID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.NightmareEntries != 37 || reloaded.LastNightmareUse != 5000000000 || len(reloaded.Carry) != 1 || reloaded.Carry[0].Index != 5137 {
		t.Fatalf("failed save was not atomic: balance=%d carry=%v", reloaded.NightmareEntries, reloaded.Carry)
	}

	ch.Carry = nil
	if err := s.SaveCharacter(ctx, accountID, ch); err != nil {
		t.Fatal(err)
	}
	reloaded, err = s.LoadCharacter(ctx, accountID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.NightmareEntries != 137 || reloaded.LastNightmareUse != 5000000001 || len(reloaded.Carry) != 0 {
		t.Fatalf("conversion not persisted: balance=%d carry=%v", reloaded.NightmareEntries, reloaded.Carry)
	}
	reloaded.NightmareEntries--
	if err := s.SaveCharacter(ctx, accountID, reloaded); err != nil {
		t.Fatal(err)
	}
	reloaded, err = s.LoadCharacter(ctx, accountID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.NightmareEntries != 136 {
		t.Fatalf("entry not persisted: balance=%d", reloaded.NightmareEntries)
	}
	other, err := s.LoadCharacter(ctx, accountID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if other.NightmareEntries != 8 {
		t.Fatalf("other character balance changed: %d", other.NightmareEntries)
	}
}

// Mesmo caso do Kefra: a 0024 foi absorvida pelo baseline, então o teste mede
// o contrato final — colunas presentes, NOT NULL DEFAULT 0 e estado existente
// intacto — em vez de reexecutar um `down` que vive só em `_archived/`.
func TestNightmareColumnsDefaultToZero(t *testing.T) {
	s, ctx := freshStore(t)
	accountID, err := s.SaveAccount(ctx, domain.Account{
		Name: "premigration", PassHash: "$argon2id$test",
		Characters: []domain.Character{{Slot: 0, Name: "Existing", Coin: 777}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range []string{"nightmare_entries", "last_nightmare_use"} {
		var def string
		var notNull bool
		if err := s.pool.QueryRow(ctx, `SELECT column_default, is_nullable = 'NO'
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'characters' AND column_name = $1`, col).Scan(&def, &notNull); err != nil {
			t.Fatalf("coluna %s ausente do baseline: %v", col, err)
		}
		if def != "0" || !notNull {
			t.Fatalf("%s default=%q notNull=%v, want 0/true", col, def, notNull)
		}
	}
	ch, err := s.LoadCharacter(ctx, accountID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ch.NightmareEntries != 0 || ch.LastNightmareUse != 0 || ch.Coin != 777 {
		t.Fatalf("migration lost existing state: entries=%d coin=%d", ch.NightmareEntries, ch.Coin)
	}
}
