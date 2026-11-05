package grpcsrv

import (
	"context"
	"testing"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/internal/domain"
)

// fakeCatalogStore is the read surface NewContentCatalog needs, with one row per
// catalog. It exists to pin the proto mapping (issue 58, fase 2) without a
// database: the fields that would silently degrade a boot are the raw CSV
// columns, the nullable sentinels and the 816-byte blob.
type fakeCatalogStore struct{}

func (fakeCatalogStore) ListItemCatalog(context.Context) ([]domain.ItemCatalogEntry, error) {
	vol := int16(64)
	return []domain.ItemCatalogEntry{{
		Index:       25,
		Name:        "Titã",
		Price:       530,
		Grade:       4,
		UniqueID:    43,
		Extra:       7,
		Pos:         64,
		ReqLevel:    40,
		ReqStr:      30,
		ReqInt:      1,
		ReqDex:      2,
		ReqCon:      3,
		Volatile:    &vol,
		AttackRange: nil, // deve virar -1 no proto
		Effects:     []domain.ItemCatalogEffect{{Eff: 2, Val: 40}},
		Fields:      []string{"25", "Titã", "0.0"},
		SHA256:      "abc",
	}}, nil
}

func (fakeCatalogStore) ListSkillCatalog(context.Context) ([]domain.SkillCatalogEntry, error) {
	return []domain.SkillCatalogEntry{{
		Index: 3, Name: "Fogo", SkillPoint: 2, ManaSpent: 12, AffectTime: 150,
		MaxTarget: 3, BParty: 1, Aggressive: 1,
	}}, nil
}

func (fakeCatalogStore) ListNPCTemplates(context.Context) ([]domain.NPCTemplateEntry, error) {
	return []domain.NPCTemplateEntry{{
		Name: "Vinha", FileName: "Vine", SourceSize: 756, SHA256: "def",
		Blob: make([]byte, 816),
	}}, nil
}

func (fakeCatalogStore) ListWorldItemSeeds(context.Context) ([]domain.WorldItemSeedEntry, error) {
	return []domain.WorldItemSeedEntry{{Index: 115, X: 2100, Y: 2032, Rotate: 3}}, nil
}

func TestFetchItemCatalogMapping(t *testing.T) {
	srv := NewContentCatalog(fakeCatalogStore{})
	resp, err := srv.FetchItemCatalog(context.Background(), &dbv1.FetchItemCatalogRequest{})
	if err != nil {
		t.Fatalf("FetchItemCatalog: %v", err)
	}
	if len(resp.GetItems()) != 1 {
		t.Fatalf("items = %d", len(resp.GetItems()))
	}
	it := resp.GetItems()[0]
	if it.GetItemIndex() != 25 || it.GetName() != "Titã" || it.GetPrice() != 530 {
		t.Errorf("identidade = %+v", it)
	}
	if it.GetVolatile() != 64 {
		t.Errorf("volatile = %d, want 64", it.GetVolatile())
	}
	// nil no domínio tem de chegar como -1: o tmServer distingue "sem alcance"
	// de "alcance 0" (EF_RANGE ausente vs presente com 0).
	if it.GetAttackRange() != -1 {
		t.Errorf("attack_range = %d, want -1", it.GetAttackRange())
	}
	if len(it.GetEffects()) != 1 || it.GetEffects()[0].GetEff() != 2 || it.GetEffects()[0].GetVal() != 40 {
		t.Errorf("effects = %+v", it.GetEffects())
	}
	// fields é a linha crua: sem ela o tmServer não reconstrói preço/requisito
	// exatamente como o CSV dava.
	if len(it.GetFields()) != 3 || it.GetFields()[1] != "Titã" {
		t.Errorf("fields = %q", it.GetFields())
	}
}

func TestFetchSkillCatalogMapping(t *testing.T) {
	srv := NewContentCatalog(fakeCatalogStore{})
	resp, err := srv.FetchSkillCatalog(context.Background(), &dbv1.FetchSkillCatalogRequest{})
	if err != nil {
		t.Fatalf("FetchSkillCatalog: %v", err)
	}
	if len(resp.GetSkills()) != 1 {
		t.Fatalf("skills = %d", len(resp.GetSkills()))
	}
	sp := resp.GetSkills()[0]
	if sp.GetSkillIndex() != 3 || sp.GetName() != "Fogo" || sp.GetManaSpent() != 12 ||
		sp.GetAffectTime() != 150 || sp.GetMaxTarget() != 3 || sp.GetParty() != 1 {
		t.Errorf("skill = %+v", sp)
	}
}

func TestFetchNPCTemplatesKeepsCanonicalBlob(t *testing.T) {
	srv := NewContentCatalog(fakeCatalogStore{})
	resp, err := srv.FetchNPCTemplates(context.Background(), &dbv1.FetchNPCTemplatesRequest{})
	if err != nil {
		t.Fatalf("FetchNPCTemplates: %v", err)
	}
	if len(resp.GetTemplates()) != 1 {
		t.Fatalf("templates = %d", len(resp.GetTemplates()))
	}
	tpl := resp.GetTemplates()[0]
	if tpl.GetTemplateName() != "Vinha" || tpl.GetFileName() != "Vine" || tpl.GetSourceSize() != 756 {
		t.Errorf("template = %+v", tpl)
	}
	if len(tpl.GetBlob()) != 816 {
		t.Errorf("blob = %d bytes, want 816 canônico", len(tpl.GetBlob()))
	}
}

func TestFetchWorldItemsMapping(t *testing.T) {
	srv := NewContentCatalog(fakeCatalogStore{})
	resp, err := srv.FetchWorldItems(context.Background(), &dbv1.FetchWorldItemsRequest{})
	if err != nil {
		t.Fatalf("FetchWorldItems: %v", err)
	}
	if len(resp.GetItems()) != 1 {
		t.Fatalf("items = %d", len(resp.GetItems()))
	}
	it := resp.GetItems()[0]
	if it.GetItemIndex() != 115 || it.GetPosX() != 2100 || it.GetPosY() != 2032 || it.GetRotate() != 3 {
		t.Errorf("world item = %+v", it)
	}
}
