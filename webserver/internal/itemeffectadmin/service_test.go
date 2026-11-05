package itemeffectadmin

import (
	"context"
	"testing"

	"openwyd-nextgen/internal/itemeffects"
	"openwyd-nextgen/internal/store"
	"openwyd-nextgen/webserver/internal/itemcatalog"
)

// fakeStore is the smallest persistence surface that lets the service exercise
// its merge/validation rules without a database.
type fakeStore struct {
	role     string
	overlay  []store.ItemEffectOverride
	upserted []store.ItemEffectOverride
	deleted  []int
	applyDel bool
}

func (f *fakeStore) AccountRole(_ context.Context, _ int64) (string, error) {
	if f.role == "" {
		return "", store.ErrNotFound
	}
	return f.role, nil
}

func (f *fakeStore) ListItemEffectOverrides(_ context.Context, itemIndex int32) ([]store.ItemEffectOverride, error) {
	out := make([]store.ItemEffectOverride, 0, len(f.overlay))
	for _, row := range f.overlay {
		if row.ItemIndex == itemIndex {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeStore) UpsertItemEffectOverride(_ context.Context, o store.ItemEffectOverride, _ int64, _ string) error {
	f.upserted = append(f.upserted, o)
	return nil
}

func (f *fakeStore) DeleteItemEffectOverride(_ context.Context, _ int32, effect int, _ int64, _ string) (bool, error) {
	f.deleted = append(f.deleted, effect)
	return f.applyDel, nil
}

// catalogWith mirrors a real row: several EF_* pairs, more than the three
// STRUCT_ITEM slots — the overlay therefore keys off the effect code.
func catalogWith() itemcatalog.Catalog {
	return itemcatalog.Catalog{Items: []itemcatalog.Entry{
		{Index: 165, Effects: []itemcatalog.Effect{{Code: 18, Value: 1}, {Code: 3, Value: 138}, {Code: 42, Value: 50}}},
		{Index: 1234},
	}}
}

func newService(f *fakeStore) *Service {
	s := New(f)
	s.SetCatalog(catalogWith())
	return s
}

func TestListMergesContentWithOverlay(t *testing.T) {
	f := &fakeStore{
		role: "admin",
		overlay: []store.ItemEffectOverride{
			{ItemIndex: 165, Effect: 42, Value: 70},
			{ItemIndex: 165, Effect: 29, Value: 2}, // efeito que o CSV não declara
		},
	}
	res, effects, _, err := newService(f).List(context.Background(), 1, 165)
	if err != nil || res != OK {
		t.Fatalf("List = %v, %v; quer OK", res, err)
	}
	if len(effects) != 4 {
		t.Fatalf("efeitos = %d (%+v); quer 3 do conteúdo + 1 extra", len(effects), effects)
	}
	if effects[0].Effect != 18 || !effects[0].FromContent || effects[0].HasOverride {
		t.Errorf("efeito 0 = %+v; quer classe 1 do conteúdo", effects[0])
	}
	if effects[2].Effect != 42 || effects[2].Value != 70 || !effects[2].HasOverride {
		t.Errorf("crítico = %+v; quer 70 do overlay", effects[2])
	}
	if last := effects[3]; last.Effect != 29 || last.FromContent || !last.HasOverride {
		t.Errorf("extra = %+v; quer velocidade 2 só da web", last)
	}
}

func TestListRejectsItemOutsideCatalog(t *testing.T) {
	res, _, reason, err := newService(&fakeStore{role: "admin"}).List(context.Background(), 1, 9999)
	if err != nil {
		t.Fatalf("List erro de infra: %v", err)
	}
	if res != NotFound || reason == "" {
		t.Fatalf("List = %v/%q; quer NotFound com motivo", res, reason)
	}
}

func TestUpsertValidatesEffectAndValue(t *testing.T) {
	f := &fakeStore{role: "moderator"}
	svc := newService(f)

	if res, msg, _ := svc.Upsert(context.Background(), 1, 165, 999, 1, ""); res != Invalid || msg == "" {
		t.Errorf("código fora do ItemEffect.h = %v/%q; quer Invalid", res, msg)
	}
	if res, _, _ := svc.Upsert(context.Background(), 1, 165, 42, MaxValue+1, ""); res != Invalid {
		t.Errorf("valor acima do short = %v; quer Invalid", res)
	}
	if res, msg, _ := svc.Upsert(context.Background(), 1, 9999, 42, 1, ""); res != NotFound {
		t.Errorf("item fora do catálogo = %v/%q; quer NotFound", res, msg)
	}

	res, _, err := svc.Upsert(context.Background(), 1, 165, 42, 70, "ajuste de balanceamento")
	if err != nil || res != OK {
		t.Fatalf("Upsert válido = %v, %v; quer OK", res, err)
	}
	if len(f.upserted) != 1 || f.upserted[0].Effect != 42 || f.upserted[0].Value != 70 {
		t.Fatalf("gravado = %+v; quer crítico 70", f.upserted)
	}
}

func TestDeleteMissingOverrideIsNotFound(t *testing.T) {
	svc := newService(&fakeStore{role: "admin"})
	if res, msg, _ := svc.Delete(context.Background(), 1, 165, 42, ""); res != NotFound || msg == "" {
		t.Fatalf("Delete sem linha = %v/%q; quer NotFound com motivo", res, msg)
	}

	f := &fakeStore{role: "admin", applyDel: true}
	svc = newService(f)
	if res, _, err := svc.Delete(context.Background(), 1, 165, 42, ""); err != nil || res != OK {
		t.Fatalf("Delete com linha = %v, %v; quer OK", res, err)
	}
	if len(f.deleted) != 1 || f.deleted[0] != 42 {
		t.Fatalf("deleted = %v; quer [42]", f.deleted)
	}
}

func TestWritesRequireModerator(t *testing.T) {
	svc := newService(&fakeStore{role: "player"})
	if res, _, _, _ := svc.List(context.Background(), 1, 165); res != Forbidden {
		t.Errorf("List como player = %v; quer Forbidden", res)
	}
	if res, _, _ := svc.Upsert(context.Background(), 1, 165, 42, 1, ""); res != Forbidden {
		t.Errorf("Upsert como player = %v; quer Forbidden", res)
	}
	if res, _, _ := svc.Delete(context.Background(), 1, 165, 42, ""); res != Forbidden {
		t.Errorf("Delete como player = %v; quer Forbidden", res)
	}
	if res, _, _, _ := svc.List(context.Background(), 0, 165); res != Forbidden {
		t.Errorf("List sem moderador = %v; quer Forbidden", res)
	}
}

// The picker offers codes from itemeffects.All(); every one of them has to pass
// validation, otherwise the editor would list an option the server refuses.
func TestEveryVocabularyCodeIsAccepted(t *testing.T) {
	svc := newService(&fakeStore{role: "admin"})
	for _, typ := range itemeffects.All() {
		if res, msg, _ := svc.Upsert(context.Background(), 1, 165, typ.Code, 1, ""); res != OK {
			t.Errorf("código %d (%s) recusado: %v/%q", typ.Code, typ.Name, res, msg)
		}
	}
}
