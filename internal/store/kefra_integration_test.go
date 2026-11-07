//go:build integration

package store

import (
	"testing"

	"openwyd-nextgen/internal/domain"
)

func TestKefraBalancePersistence(t *testing.T) {
	s, ctx := freshStore(t)
	accountID, err := s.SaveAccount(ctx, domain.Account{
		Name: "kefra", PassHash: "$argon2id$test",
		Characters: []domain.Character{
			{Slot: 0, Name: "Survivor", KefraTicket: 37, Carry: []domain.Item{{Slot: 0, Index: 4127}}},
			{Slot: 1, Name: "Other", KefraTicket: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ch, err := s.LoadCharacter(ctx, accountID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ch.KefraTicket != 37 {
		t.Fatalf("imported balance=%d, want 37", ch.KefraTicket)
	}

	// Uma falha depois do UPDATE do personagem precisa reverter saldo e
	// inventário juntos. Slot é SMALLINT no PostgreSQL; este não pode ser salvo.
	ch.KefraTicket = 137
	ch.Carry = []domain.Item{{Slot: 40000, Index: 1100}}
	if err := s.SaveCharacter(ctx, accountID, ch); err == nil {
		t.Fatal("expected invalid item save to fail")
	}
	reloaded, err := s.LoadCharacter(ctx, accountID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.KefraTicket != 37 || len(reloaded.Carry) != 1 || reloaded.Carry[0].Index != 4127 {
		t.Fatalf("failed save was not atomic: balance=%d carry=%v", reloaded.KefraTicket, reloaded.Carry)
	}

	ch.Carry = nil
	if err := s.SaveCharacter(ctx, accountID, ch); err != nil {
		t.Fatal(err)
	}
	reloaded, err = s.LoadCharacter(ctx, accountID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.KefraTicket != 137 || len(reloaded.Carry) != 0 {
		t.Fatalf("conversion not persisted: balance=%d carry=%v", reloaded.KefraTicket, reloaded.Carry)
	}
	reloaded.KefraTicket--
	if err := s.SaveCharacter(ctx, accountID, reloaded); err != nil {
		t.Fatal(err)
	}
	reloaded, err = s.LoadCharacter(ctx, accountID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.KefraTicket != 136 {
		t.Fatalf("entry not persisted: balance=%d", reloaded.KefraTicket)
	}
	other, err := s.LoadCharacter(ctx, accountID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if other.KefraTicket != 8 {
		t.Fatalf("other character balance changed: %d", other.KefraTicket)
	}
}

// O squash do baseline absorveu a 0022: o replay de 0022_kefra_ticket.down.sql
// deixou de existir (`_archived/` fica fora do //go:embed *.sql). O que ainda
// precisa ser verdade é o que a migration garantia: a coluna existe no schema
// final, é NOT NULL DEFAULT 0 e um personagem gravado sem ela continua legível.
func TestKefraTicketColumnDefaultsToZero(t *testing.T) {
	s, ctx := freshStore(t)
	accountID, err := s.SaveAccount(ctx, domain.Account{
		Name: "premigration", PassHash: "$argon2id$test",
		Characters: []domain.Character{{Slot: 0, Name: "Existing", Coin: 777}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var def string
	var notNull bool
	if err := s.pool.QueryRow(ctx, `SELECT column_default, is_nullable = 'NO'
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'characters' AND column_name = 'kefra_ticket'`).Scan(&def, &notNull); err != nil {
		t.Fatalf("coluna kefra_ticket ausente do baseline: %v", err)
	}
	if def != "0" || !notNull {
		t.Fatalf("kefra_ticket default=%q notNull=%v, want 0/true", def, notNull)
	}
	ch, err := s.LoadCharacter(ctx, accountID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ch.KefraTicket != 0 || ch.Coin != 777 {
		t.Fatalf("migration lost existing state: tickets=%d coin=%d", ch.KefraTicket, ch.Coin)
	}
}
