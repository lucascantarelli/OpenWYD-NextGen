//go:build integration

// Testes de integração do store de edição de NPC (npc-editing-plan.md). Eles exigem
// um banco real e ficam fora do build padrão. Rode com:
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

// TestNPCConfigCRUD percorre o caminho de escrita completo do moderador: upsert de
// definição, definir sua loja, alternar visibilidade, sobrepor preço e apagar — verificando que a
// versão de config sobe a cada escrita e que as leituras refletem as escritas.
func TestNPCConfigCRUD(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	// Estado limpo para as tabelas de NPC (deixe o esquema de conta em paz).
	_, _ = pool.Exec(ctx, `DELETE FROM npc_audit; DELETE FROM npc_shop_item; DELETE FROM npc_definition; DELETE FROM item_price; UPDATE npc_config_meta SET version = 0 WHERE id = TRUE`)

	s := New(pool)

	v0, err := s.NPCConfigVersion(ctx)
	if err != nil {
		t.Fatalf("version: %v", err)
	}

	// Uma conta de moderador para satisfazer a FK de updated_by / account_id da auditoria.
	var modID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO account (name, pass_hash, role) VALUES ('mod_npc_test','x','moderator') RETURNING id`).
		Scan(&modID); err != nil {
		t.Fatalf("seed moderator: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM account WHERE id = $1`, modID) })

	id, err := s.UpsertNPCDefinition(ctx, domain.NPCDefinition{
		Slug: "shop-int-1", TemplateName: "Keeper", DisplayName: "Keeper",
		Enabled: true, PosX: 100, PosY: 200, Merchant: 1,
	}, modID)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := s.SetNPCShop(ctx, id, []domain.NPCShopItem{
		{Slot: 0, ItemIndex: 1100, Quantity: 120}, {Slot: 5, ItemIndex: 1101, Eff1: 1, EffV1: 9},
	}, modID); err != nil {
		t.Fatalf("set shop: %v", err)
	}
	if err := s.SetNPCVisibility(ctx, id, false, modID); err != nil {
		t.Fatalf("set visibility: %v", err)
	}
	if err := s.SetItemPrice(ctx, 1100, 777, modID); err != nil {
		t.Fatalf("set item price: %v", err)
	}

	// Relê o snapshot completo.
	defs, err := s.ListNPCDefinitions(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("definitions = %d, want 1", len(defs))
	}
	got := defs[0]
	if got.Slug != "shop-int-1" || got.Enabled || got.PosX != 100 || got.Merchant != 1 {
		t.Errorf("definition mismatch: %+v", got)
	}
	if len(got.Shop) != 2 || got.Shop[0].Quantity != 120 || got.Shop[1].ItemIndex != 1101 || got.Shop[1].EffV1 != 9 {
		t.Errorf("shop mismatch: %+v", got.Shop)
	}

	prices, err := s.ItemPriceOverrides(ctx)
	if err != nil {
		t.Fatalf("prices: %v", err)
	}
	if len(prices) != 1 || prices[0].ItemIndex != 1100 || prices[0].Price != 777 {
		t.Errorf("price override mismatch: %+v", prices)
	}

	// A versão precisa ter avançado com as quatro escritas.
	vN, err := s.NPCConfigVersion(ctx)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if vN != v0+4 {
		t.Errorf("version = %d, want %d (4 writes)", vN, v0+4)
	}

	// O delete cascateia as linhas de loja.
	if err := s.DeleteNPCDefinition(ctx, id, modID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	defs, _ = s.ListNPCDefinitions(ctx)
	if len(defs) != 0 {
		t.Errorf("definitions after delete = %d, want 0", len(defs))
	}
}

// TestSeedNPCDefinitionsIdempotent confere que a importação em massa insere uma vez e pula
// slugs existentes numa re-execução.
func TestSeedNPCDefinitionsIdempotent(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM npc_shop_item; DELETE FROM npc_definition WHERE slug LIKE 'seed-int-%'`)

	s := New(pool)
	defs := []domain.NPCDefinition{
		{Slug: "seed-int-1", TemplateName: "A", Enabled: true, Merchant: 1, GeneratorIndex: 1,
			Shop: []domain.NPCShopItem{{Slot: 0, ItemIndex: 10, Quantity: 2}}},
		{Slug: "seed-int-2", TemplateName: "B", Enabled: true, Merchant: 1, GeneratorIndex: 2},
	}
	n, err := s.SeedNPCDefinitions(ctx, defs)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n != 2 {
		t.Fatalf("first seed inserted %d, want 2", n)
	}
	n, err = s.SeedNPCDefinitions(ctx, defs)
	if err != nil {
		t.Fatalf("seed (2nd): %v", err)
	}
	if n != 0 {
		t.Errorf("second seed inserted %d, want 0 (idempotent)", n)
	}
	// Os itens de loja são reenfileirados a cada passada e deduplicados por
	// ON CONFLICT (npc_id, slot) DO NOTHING — verifique que a segunda passada não os
	// duplicou, o que o seed enfileirado poderia esconder.
	var shopRows int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM npc_shop_item si
		JOIN npc_definition d ON d.id = si.npc_id
		WHERE d.slug = 'seed-int-1'`).Scan(&shopRows); err != nil {
		t.Fatalf("count shop items: %v", err)
	}
	if shopRows != 1 {
		t.Errorf("shop items for seed-int-1 = %d, want 1", shopRows)
	}
}

// contentGenIndex mantém o generator index de TestDeleteNPCDefinitionContentOwned
// longe dos que os outros testes deste pacote usam (generator_index é
// globally unique).
const contentGenIndex = 200000

// TestDeleteNPCDefinitionContentOwned confere que uma definição importada de
// NPCGener.txt (origin = 'content', escrita por SeedNPCDefinitions) é recusada
// em vez de apagada, e que a recusa não deixa nada para trás: nenhuma linha de
// auditoria e nenhum incremento de versão, já que retorna antes de auditAndBump.
func TestDeleteNPCDefinitionContentOwned(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM npc_shop_item; DELETE FROM npc_definition WHERE slug LIKE 'content-int-%'`)

	s := New(pool)

	var modID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO account (name, pass_hash, role) VALUES ('mod_npc_content_test','x','moderator') RETURNING id`).
		Scan(&modID); err != nil {
		t.Fatalf("seed moderator: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM account WHERE id = $1`, modID) })

	if _, err := s.SeedNPCDefinitions(ctx, []domain.NPCDefinition{
		{Slug: "content-int-1", TemplateName: "Set_BM_2", Enabled: true, GeneratorIndex: contentGenIndex},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM npc_definition WHERE slug = 'content-int-1'`) })

	var npcID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM npc_definition WHERE slug = 'content-int-1'`).Scan(&npcID); err != nil {
		t.Fatalf("read seeded id: %v", err)
	}

	v0, err := s.NPCConfigVersion(ctx)
	if err != nil {
		t.Fatalf("version: %v", err)
	}

	if err := s.DeleteNPCDefinition(ctx, npcID, modID); !errors.Is(err, ErrContentOwned) {
		t.Fatalf("delete = %v, want ErrContentOwned", err)
	}

	var still int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM npc_definition WHERE id = $1`, npcID).Scan(&still); err != nil {
		t.Fatalf("count definition: %v", err)
	}
	if still != 1 {
		t.Errorf("definition rows after refused delete = %d, want 1 (untouched)", still)
	}

	var audits int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM npc_audit WHERE npc_id = $1`, npcID).Scan(&audits); err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if audits != 0 {
		t.Errorf("audit rows = %d, want 0 (refusal returns before auditAndBump)", audits)
	}

	vN, err := s.NPCConfigVersion(ctx)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if vN != v0 {
		t.Errorf("version = %d, want %d (refused write must not bump)", vN, v0)
	}

	// Escondê-la é a alternativa suportada e precisa continuar funcionando.
	if err := s.SetNPCVisibility(ctx, npcID, false, modID); err != nil {
		t.Errorf("set visibility on content NPC: %v", err)
	}
}

// chunkGenBase mantém os generator indices de TestSeedNPCDefinitionsSpansBatchChunks
// longe dos que os outros testes deste pacote usam.
const chunkGenBase = 100000

// TestSeedNPCDefinitionsSpansBatchChunks cobre a fronteira de fatia do
// seed enfileirado. Os ids de definição voltam posicionalmente por fatia e os itens
// de loja têm chave neles, então um off-by-one numa quebra de fatia anexaria
// silenciosamente o estoque de um mercador ao NPC errado.
func TestSeedNPCDefinitionsSpansBatchChunks(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM npc_shop_item; DELETE FROM npc_definition WHERE slug LIKE 'seed-chunk-%'`)

	s := New(pool)
	total := seedBatchChunk + 5
	defs := make([]domain.NPCDefinition, total)
	for i := range defs {
		defs[i] = domain.NPCDefinition{
			Slug: fmt.Sprintf("seed-chunk-%d", i), TemplateName: "A", Enabled: true,
			// generator_index é globalmente único, então fique bem longe de qualquer
			// linha que os outros testes deste pacote deixam para trás.
			Merchant: 1, GeneratorIndex: int32(chunkGenBase + i),
			// item_index codifica o ordinal do slug para o pareamento ser verificável.
			Shop: []domain.NPCShopItem{{Slot: 0, ItemIndex: int32(1000 + i), Quantity: 1}},
		}
	}

	n, err := s.SeedNPCDefinitions(ctx, defs)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n != total {
		t.Fatalf("seed inserted %d, want %d", n, total)
	}

	var mismatched int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM npc_shop_item si
		JOIN npc_definition d ON d.id = si.npc_id
		WHERE d.slug LIKE 'seed-chunk-%'
		  AND si.item_index <> 1000 + split_part(d.slug, '-', 3)::int`).Scan(&mismatched); err != nil {
		t.Fatalf("check shop pairing: %v", err)
	}
	if mismatched != 0 {
		t.Errorf("%d shop items landed on the wrong definition across the chunk split", mismatched)
	}
}
