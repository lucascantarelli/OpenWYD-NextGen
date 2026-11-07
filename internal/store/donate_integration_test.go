//go:build integration

// Testes de integração do store da loja de donate (issue #34). Eles exigem um
// banco real e ficam fora do build padrão. Rode com:
//
//	W2PP_TEST_DSN=postgres://postgres:dev@localhost:5432/postgres go test -tags=integration ./internal/store/
package store

import (
	"context"
	"errors"
	"testing"

	"openwyd-nextgen/internal/domain"
)

// TestDonateShopCRUD percorre o caminho de escrita do catálogo de moderação: criar,
// atualizar,
func TestDonateShopCRUD(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM donate_shop_audit; DELETE FROM donate_shop_item`)
	s := New(pool)

	var modID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO account (name, pass_hash, role) VALUES ('mod_donate_test','x','moderator') RETURNING id`).
		Scan(&modID); err != nil {
		t.Fatalf("seed moderator: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM account WHERE id = $1`, modID) })

	id, err := s.UpsertDonateShopItem(ctx, domain.DonateShopItem{
		ItemIndex: 3540, Eff1: 1, EffV1: 9, Price: 100, Title: "Set Celestial",
		Description: "shiny", Enabled: true, ExpiresDays: 30,
	}, modID)
	if err != nil {
		t.Fatalf("upsert create: %v", err)
	}

	got, err := s.GetDonateShopItem(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ItemIndex != 3540 || got.Price != 100 || !got.Enabled || got.ExpiresDays != 30 || got.Eff1 != 1 || got.EffV1 != 9 {
		t.Errorf("offer mismatch: %+v", got)
	}

	// Atualiza no lugar (mesmo id).
	got.Price = 250
	got.Title = "Set Celestial +11"
	if _, err := s.UpsertDonateShopItem(ctx, got, modID); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	if updated, _ := s.GetDonateShopItem(ctx, id); updated.Price != 250 || updated.Title != "Set Celestial +11" {
		t.Errorf("update not applied: %+v", updated)
	}

	// Atualizar um id inexistente é NotFound.
	if _, err := s.UpsertDonateShopItem(ctx, domain.DonateShopItem{ID: 999999, ItemIndex: 1, Price: 1}, modID); !errors.Is(err, ErrNotFound) {
		t.Errorf("upsert missing id err = %v, want ErrNotFound", err)
	}

	if err := s.SetDonateShopItemEnabled(ctx, id, false, modID); err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	all, err := s.ListDonateShopItems(ctx)
	if err != nil || len(all) != 1 || all[0].Enabled {
		t.Fatalf("list = %+v (err %v), want 1 disabled offer", all, err)
	}
	if enabled, _ := s.ListEnabledDonateShopItems(ctx); len(enabled) != 0 {
		t.Errorf("enabled list = %d, want 0 (offer disabled)", len(enabled))
	}

	if err := s.DeleteDonateShopItem(ctx, id, modID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetDonateShopItem(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete err = %v, want ErrNotFound", err)
	}
}

// TestDonateCreditAndBuy percorre o caminho de crédito da carteira + compra + drenagem.
func TestDonateCreditAndBuy(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM donate_shop_audit; DELETE FROM donate_shop_item; DELETE FROM delivery_queue`)
	s := New(pool)

	var accID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO account (name, pass_hash, donate_balance) VALUES ('buyer_donate_test','x',0) RETURNING id`).
		Scan(&accID); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM account WHERE id = $1`, accID) })

	// Credita a carteira.
	if bal, err := s.CreditDonateBalance(ctx, accID, 300, 0, "manual grant"); err != nil || bal != 300 {
		t.Fatalf("credit = (%d, %v), want (300, nil)", bal, err)
	}

	// Uma oferta habilitada a preço 100 (com um efeito + validade de 7 dias).
	itemID, err := s.UpsertDonateShopItem(ctx, domain.DonateShopItem{
		ItemIndex: 1234, Eff1: 2, EffV1: 5, Price: 100, Enabled: true, ExpiresDays: 7,
	}, 0)
	if err != nil {
		t.Fatalf("upsert offer: %v", err)
	}

	// Compra: o saldo cai para 200 e uma entrega de item pendente aparece.
	bal, err := s.BuyDonateItem(ctx, accID, itemID)
	if err != nil || bal != 200 {
		t.Fatalf("buy = (%d, %v), want (200, nil)", bal, err)
	}
	pending, err := s.PendingItemDeliveries(ctx, accID)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending = %+v (err %v), want 1", pending, err)
	}
	d := pending[0]
	if d.Item.Index != 1234 || d.Item.Eff1 != 2 || d.Item.EffV1 != 5 || d.Item.ExpiresAt == 0 {
		t.Errorf("delivery payload mismatch: %+v", d.Item)
	}

	// Comprar de novo com só 200 restantes funciona mais duas vezes, depois fica insuficiente.
	if _, err := s.BuyDonateItem(ctx, accID, itemID); err != nil {
		t.Fatalf("buy 2: %v", err)
	}
	if _, err := s.BuyDonateItem(ctx, accID, itemID); err != nil {
		t.Fatalf("buy 3: %v", err)
	}
	if _, err := s.BuyDonateItem(ctx, accID, itemID); !errors.Is(err, ErrInsufficientDonate) {
		t.Errorf("buy 4 err = %v, want ErrInsufficientDonate", err)
	}

	// Uma oferta desabilitada rejeita a compra.
	disabledID, _ := s.UpsertDonateShopItem(ctx, domain.DonateShopItem{ItemIndex: 5, Price: 1, Enabled: false}, 0)
	if _, err := s.CreditDonateBalance(ctx, accID, 10, 0, "topup"); err != nil {
		t.Fatalf("topup: %v", err)
	}
	if _, err := s.BuyDonateItem(ctx, accID, disabledID); !errors.Is(err, ErrShopItemDisabled) {
		t.Errorf("buy disabled err = %v, want ErrShopItemDisabled", err)
	}

	// Uma oferta desconhecida é NotFound.
	if _, err := s.BuyDonateItem(ctx, accID, 987654); !errors.Is(err, ErrNotFound) {
		t.Errorf("buy unknown err = %v, want ErrNotFound", err)
	}

	// Drenagem: marca a primeira entrega pendente como entregue, uma falsa como perdida, e
	// grava o cargo — tudo numa transação. A linha entregue sai do conjunto pendente.
	all, _ := s.PendingItemDeliveries(ctx, accID)
	if len(all) != 3 {
		t.Fatalf("pending before drain = %d, want 3", len(all))
	}
	deliveredIDs := []int64{all[0].ID, all[1].ID}
	lostIDs := []int64{all[2].ID}
	cargoItems := []domain.Item{{Slot: 0, Index: 1234, Eff1: 2, EffV1: 5, ExpiresAt: all[0].Item.ExpiresAt}}
	if err := s.SaveCargoWithDeliveries(ctx, accID, 0, cargoItems, deliveredIDs, lostIDs); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if remaining, _ := s.PendingItemDeliveries(ctx, accID); len(remaining) != 0 {
		t.Errorf("pending after drain = %d, want 0", len(remaining))
	}
	// O cargo agora contém o item entregue.
	_, items, err := s.LoadCargo(ctx, accID)
	if err != nil || len(items) != 1 || items[0].Index != 1234 {
		t.Errorf("cargo after drain = %+v (err %v), want 1 item index 1234", items, err)
	}
}
