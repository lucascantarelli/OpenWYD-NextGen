//go:build integration

package store

import (
	"sync"
	"testing"

	"openwyd-nextgen/internal/domain"
)

func TestPurchaseKingdomCapeRevisionIsSingleWinner(t *testing.T) {
	s, ctx := freshStore(t)
	accountID, err := s.SaveAccount(ctx, domain.Account{Name: "cape", PassHash: "hash", Characters: []domain.Character{{Slot: 0, Name: "Hero", Carry: []domain.Item{{Slot: 1, Index: 697}}}}})
	if err != nil {
		t.Fatal(err)
	}
	quote, err := s.QuoteKingdomCape(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if quote.Revision != defaultCapeRevision || quote.HekalotiaCost != defaultCapeCost || quote.AkeloniaCost != defaultCapeCost {
		t.Fatalf("initial quote = %+v, want revision=%d costs=%d/%d", quote, defaultCapeRevision, defaultCapeCost, defaultCapeCost)
	}

	var wg sync.WaitGroup
	results := make(chan bool, 2)
	for _, kingdom := range []uint8{kingdomHekalotia, kingdomAkelonia} {
		wg.Add(1)
		go func(kingdom uint8) {
			defer wg.Done()
			cape := int16(545)
			if kingdom == kingdomAkelonia {
				cape = 546
			}
			_, ok, purchaseErr := s.PurchaseKingdomCape(ctx, accountID, quote.Revision, kingdom, domain.Character{Slot: 0, Clan: kingdom, Equip: []domain.Item{{Slot: 15, Index: cape}}})
			if purchaseErr != nil {
				t.Errorf("purchase: %v", purchaseErr)
			}
			results <- ok
		}(kingdom)
	}
	wg.Wait()
	close(results)
	winners := 0
	for ok := range results {
		if ok {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("committed purchases = %d, want 1", winners)
	}
	updated, err := s.QuoteKingdomCape(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != quote.Revision+1 {
		t.Fatalf("revision = %d, want %d", updated.Revision, quote.Revision+1)
	}
	if updated.HekalotiaCost < minimumCapeCost || updated.HekalotiaCost > maximumCapeCost || updated.AkeloniaCost < minimumCapeCost || updated.AkeloniaCost > maximumCapeCost {
		t.Fatalf("out-of-range quote: %+v", updated)
	}
}

func TestQuoteKingdomCapeRepairsMissingSingleton(t *testing.T) {
	s, ctx := freshStore(t)
	accountID, err := s.SaveAccount(ctx, domain.Account{Name: "cape-missing", PassHash: "hash", Characters: []domain.Character{{Slot: 0, Name: "Hero"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM sapphire_balance`); err != nil {
		t.Fatalf("delete singleton: %v", err)
	}

	quote, err := s.QuoteKingdomCape(ctx)
	if err != nil {
		t.Fatalf("quote without singleton: %v", err)
	}
	if quote.Revision != defaultCapeRevision || quote.HekalotiaCost != defaultCapeCost || quote.AkeloniaCost != defaultCapeCost {
		t.Fatalf("fallback quote = %+v, want defaults", quote)
	}

	updated, ok, err := s.PurchaseKingdomCape(ctx, accountID, quote.Revision, kingdomAkelonia, domain.Character{Slot: 0, Clan: kingdomAkelonia, Equip: []domain.Item{{Slot: 15, Index: 546}}})
	if err != nil {
		t.Fatalf("purchase without singleton: %v", err)
	}
	if !ok {
		t.Fatal("purchase without singleton was rejected")
	}
	if updated.Revision != quote.Revision+1 || updated.HekalotiaCost != defaultCapeCost-1 || updated.AkeloniaCost != defaultCapeCost+1 {
		t.Fatalf("repaired quote = %+v, want revision=%d costs=%d/%d", updated, quote.Revision+1, defaultCapeCost-1, defaultCapeCost+1)
	}
	var rows int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM sapphire_balance`).Scan(&rows); err != nil {
		t.Fatalf("count singleton: %v", err)
	}
	if rows != 1 {
		t.Fatalf("sapphire_balance rows = %d, want 1", rows)
	}
}

func TestClampCapeCost(t *testing.T) {
	for _, tc := range []struct{ in, want int32 }{{-1, 4}, {4, 4}, {18, 18}, {32, 32}, {99, 32}} {
		if got := clampCapeCost(tc.in); got != tc.want {
			t.Errorf("clampCapeCost(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
