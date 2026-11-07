//go:build integration

// Testes de integração do store de recompensa diária (issue #35). Eles exigem um
// banco real e ficam fora do build padrão. Rode com:
//
//	W2PP_TEST_DSN=postgres://postgres:dev@localhost:5432/postgres go test -tags=integration ./internal/store/
package store

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"openwyd-nextgen/internal/domain"
)

// TestDailyRewardCRUD percorre o caminho de escrita do catálogo de moderação: criar,
// atualizar, alternar o enabled, listar, obter e apagar.
func TestDailyRewardCRUD(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM daily_reward_audit; DELETE FROM daily_reward_claim; DELETE FROM daily_reward_item`)
	s := New(pool)

	var modID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO account (name, pass_hash, role) VALUES ('mod_dailyreward_test','x','moderator') RETURNING id`).
		Scan(&modID); err != nil {
		t.Fatalf("seed moderator: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM account WHERE id = $1`, modID) })

	id, err := s.UpsertDailyRewardItem(ctx, domain.DailyRewardItem{
		ItemIndex: 3540, Eff1: 1, EffV1: 9, Title: "Set Celestial",
		Description: "shiny", Enabled: true, ExpiresDays: 30,
	}, modID)
	if err != nil {
		t.Fatalf("upsert create: %v", err)
	}

	got, err := s.GetDailyRewardItem(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ItemIndex != 3540 || !got.Enabled || got.ExpiresDays != 30 || got.Eff1 != 1 || got.EffV1 != 9 {
		t.Errorf("offer mismatch: %+v", got)
	}

	// Atualiza no lugar (mesmo id).
	got.Title = "Set Celestial +11"
	if _, err := s.UpsertDailyRewardItem(ctx, got, modID); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	if updated, _ := s.GetDailyRewardItem(ctx, id); updated.Title != "Set Celestial +11" {
		t.Errorf("update not applied: %+v", updated)
	}

	// Atualizar um id inexistente é NotFound.
	if _, err := s.UpsertDailyRewardItem(ctx, domain.DailyRewardItem{ID: 999999, ItemIndex: 1}, modID); !errors.Is(err, ErrNotFound) {
		t.Errorf("upsert missing id err = %v, want ErrNotFound", err)
	}

	if err := s.SetDailyRewardItemEnabled(ctx, id, false, modID); err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	all, err := s.ListDailyRewardItems(ctx)
	if err != nil || len(all) != 1 || all[0].Enabled {
		t.Fatalf("list = %+v (err %v), want 1 disabled offer", all, err)
	}
	if enabled, _ := s.ListEnabledDailyRewardItems(ctx); len(enabled) != 0 {
		t.Errorf("enabled list = %d, want 0 (offer disabled)", len(enabled))
	}

	if err := s.DeleteDailyRewardItem(ctx, id, modID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetDailyRewardItem(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete err = %v, want ErrNotFound", err)
	}
}

// TestDailyRewardClaimFlow percorre o gate de resgate uma vez por conta por dia UTC
// e o enfileiramento em delivery_queue que ele dispara.
func TestDailyRewardClaimFlow(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM daily_reward_audit; DELETE FROM daily_reward_claim; DELETE FROM daily_reward_item; DELETE FROM delivery_queue`)
	s := New(pool)

	item1, err := s.UpsertDailyRewardItem(ctx, domain.DailyRewardItem{
		ItemIndex: 1234, Eff1: 2, EffV1: 5, Title: "Reward One", Enabled: true, ExpiresDays: 7,
	}, 0)
	if err != nil {
		t.Fatalf("upsert item1: %v", err)
	}
	item2, err := s.UpsertDailyRewardItem(ctx, domain.DailyRewardItem{
		ItemIndex: 5678, Title: "Reward Two", Enabled: true,
	}, 0)
	if err != nil {
		t.Fatalf("upsert item2: %v", err)
	}
	disabledItem, err := s.UpsertDailyRewardItem(ctx, domain.DailyRewardItem{
		ItemIndex: 9999, Title: "Disabled Reward", Enabled: false,
	}, 0)
	if err != nil {
		t.Fatalf("upsert disabled item: %v", err)
	}

	var accID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO account (name, pass_hash) VALUES ('claimer_dailyreward_test','x') RETURNING id`).
		Scan(&accID); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM account WHERE id = $1`, accID) })

	// No claim yet today.
	if claimed, _, _, err := s.DailyRewardClaimedToday(ctx, accID); err != nil || claimed {
		t.Fatalf("DailyRewardClaimedToday(before) = (%v, err %v), want (false, nil)", claimed, err)
	}

	// O primeiro resgate funciona e enfileira exatamente uma entrega de item pendente com
	// a proveniência de origem esperada.
	if err := s.ClaimDailyReward(ctx, accID, item1); err != nil {
		t.Fatalf("claim item1: %v", err)
	}
	pending, err := s.PendingItemDeliveries(ctx, accID)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending = %+v (err %v), want 1", pending, err)
	}
	if pending[0].Item.Index != 1234 || pending[0].Item.Eff1 != 2 || pending[0].Item.EffV1 != 5 || pending[0].Item.ExpiresAt == 0 {
		t.Errorf("delivery payload mismatch: %+v", pending[0].Item)
	}
	var source string
	if err := pool.QueryRow(ctx, `SELECT source FROM delivery_queue WHERE account_id = $1`, accID).Scan(&source); err != nil {
		t.Fatalf("read delivery source: %v", err)
	}
	if want := fmt.Sprintf("daily_reward:%d", item1); source != want {
		t.Errorf("delivery source = %q, want %q", source, want)
	}

	// Um segundo resgate no mesmo dia UTC, mesmo para um item diferente, é bloqueado.
	if err := s.ClaimDailyReward(ctx, accID, item2); !errors.Is(err, ErrAlreadyClaimedToday) {
		t.Errorf("claim item2 same day err = %v, want ErrAlreadyClaimedToday", err)
	}

	// A leitura de status do resgate agora reflete a oferta resgatada.
	claimed, claimedItemID, title, err := s.DailyRewardClaimedToday(ctx, accID)
	if err != nil || !claimed || claimedItemID != item1 || title != "Reward One" {
		t.Fatalf("DailyRewardClaimedToday(after) = (%v, %d, %q, err %v), want (true, %d, \"Reward One\", nil)", claimed, claimedItemID, title, err, item1)
	}

	// Uma oferta desabilitada rejeita o resgate, para uma conta nova (sem resgate anterior).
	var acc2 int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO account (name, pass_hash) VALUES ('claimer2_dailyreward_test','x') RETURNING id`).
		Scan(&acc2); err != nil {
		t.Fatalf("seed account 2: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM account WHERE id = $1`, acc2) })
	if err := s.ClaimDailyReward(ctx, acc2, disabledItem); !errors.Is(err, ErrShopItemDisabled) {
		t.Errorf("claim disabled err = %v, want ErrShopItemDisabled", err)
	}

	// Uma oferta desconhecida é NotFound.
	if err := s.ClaimDailyReward(ctx, acc2, 987654); !errors.Is(err, ErrNotFound) {
		t.Errorf("claim unknown err = %v, want ErrNotFound", err)
	}

	// Um resgate registrado para ontem não bloqueia o resgate de hoje (a restrição
	// UNIQUE é por dia de calendário UTC, não permanente).
	var acc3 int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO account (name, pass_hash) VALUES ('claimer3_dailyreward_test','x') RETURNING id`).
		Scan(&acc3); err != nil {
		t.Fatalf("seed account 3: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM account WHERE id = $1`, acc3) })
	if _, err := pool.Exec(ctx, `
		INSERT INTO daily_reward_claim (account_id, reward_item_id, claim_date)
		VALUES ($1, $2, (CURRENT_DATE - INTERVAL '1 day')::date)`, acc3, item1); err != nil {
		t.Fatalf("seed yesterday claim: %v", err)
	}
	if err := s.ClaimDailyReward(ctx, acc3, item1); err != nil {
		t.Errorf("claim today (after yesterday claim) err = %v, want nil", err)
	}
}
