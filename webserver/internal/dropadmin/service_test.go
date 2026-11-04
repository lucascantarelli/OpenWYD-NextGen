package dropadmin

import (
	"context"
	"testing"

	"github.com/jeanluca/w2pp-openwyd/internal/domain"
	"github.com/jeanluca/w2pp-openwyd/internal/savefmt"
	"github.com/jeanluca/w2pp-openwyd/internal/store"
	"github.com/jeanluca/w2pp-openwyd/webserver/internal/itemcatalog"
)

type fakeStore struct {
	roles   map[int64]string
	rows    map[int]int64
	lastSlot  int
	lastValue int64
	lastReason string

	carry       map[string][]domain.MobTemplateEquipItem
	carryReason string
}

func newFake() *fakeStore {
	return &fakeStore{
		roles: map[int64]string{1: "moderator", 2: "admin", 3: "player"},
		rows:  map[int]int64{},
		carry: map[string][]domain.MobTemplateEquipItem{},
	}
}

func (f *fakeStore) AccountRole(_ context.Context, id int64) (string, error) {
	role, ok := f.roles[id]
	if !ok {
		return "", store.ErrNotFound
	}
	return role, nil
}

func (f *fakeStore) ListDropRateOverrides(context.Context) ([]store.DropRateOverride, error) {
	out := make([]store.DropRateOverride, 0, len(f.rows))
	for slot, value := range f.rows {
		out = append(out, store.DropRateOverride{Slot: slot, Value: value})
	}
	return out, nil
}

func (f *fakeStore) UpsertDropRateOverride(_ context.Context, slot int, value int64, _ int64, reason string) error {
	f.rows[slot] = value
	f.lastSlot, f.lastValue, f.lastReason = slot, value, reason
	return nil
}

func (f *fakeStore) DeleteDropRateOverride(_ context.Context, slot int, _ int64, _ string) (bool, error) {
	if _, ok := f.rows[slot]; !ok {
		return false, nil
	}
	delete(f.rows, slot)
	return true, nil
}

func (f *fakeStore) GetMobCarryOverride(_ context.Context, templateName string) (domain.MobCarryOverride, error) {
	slots, ok := f.carry[templateName]
	if !ok || len(slots) == 0 {
		return domain.MobCarryOverride{}, store.ErrNotFound
	}
	return domain.MobCarryOverride{TemplateName: templateName, Slots: slots}, nil
}

func (f *fakeStore) SetMobCarry(_ context.Context, templateName string, slots []domain.MobTemplateEquipItem, _ int64, reason string) error {
	f.carry[templateName] = slots
	f.carryReason = reason
	return nil
}

func (f *fakeStore) DeleteMobCarry(_ context.Context, templateName string, _ int64, _ string) (bool, error) {
	if len(f.carry[templateName]) == 0 {
		return false, nil
	}
	delete(f.carry, templateName)
	return true, nil
}

// carryTemplate builds a raw 816-byte template with one populated Carry slot.
func carryTemplate(t *testing.T, item int16, eff, value uint8) []byte {
	t.Helper()
	mob := savefmt.Mob{}
	copy(mob.Name[:], "Chefe de Teste")
	mob.CurrentScore.Level = 120
	mob.Carry[56] = savefmt.Item{Index: item, Effects: [3]savefmt.Effect{{Effect: eff, Value: value}}}
	return savefmt.EncodeMob(mob)
}

func carryService(t *testing.T) (*Service, *fakeStore) {
	t.Helper()
	raw := carryTemplate(t, 100, 9, 2)
	f := newFake()
	s := New(f)
	s.SetTemplateReader(func(name string) ([]byte, error) {
		if name != "Chefe" {
			return nil, store.ErrNotFound
		}
		return raw, nil
	})
	s.SetItemCatalog(itemcatalog.Catalog{Items: []itemcatalog.Entry{
		{Index: 100, DisplayName: "Poção de Vida"},
		{Index: 412, DisplayName: "Espada Longa"},
	}})
	return s, f
}

// O efeito do item que continua no lugar não pode ser apagado por um save que
// só mandou o índice; um item novo começa sem efeitos.
func TestSetCarryPreservesEffectsOnlyForTheSameItem(t *testing.T) {
	s, f := carryService(t)
	ctx := context.Background()

	res, err := s.SetCarry(ctx, 1, "Chefe", []CarryInput{
		{Slot: 56, ItemIndex: 100},
		{Slot: 0, ItemIndex: 412},
	}, "ajuste de drop")
	if err != nil || res != OK {
		t.Fatalf("SetCarry = %v, %v", res, err)
	}

	bySlot := map[int]domain.MobTemplateEquipItem{}
	for _, it := range f.carry["Chefe"] {
		bySlot[int(it.Slot)] = it
	}
	if got := bySlot[56]; got.Eff1 != 9 || got.EffV1 != 2 {
		t.Fatalf("slot 56 perdeu os efeitos: %+v", got)
	}
	if got := bySlot[0]; got.ItemIndex != 412 || got.Eff1 != 0 {
		t.Fatalf("slot 0 = %+v, want item novo sem efeitos", got)
	}
	if f.carryReason != "ajuste de drop" {
		t.Fatalf("motivo auditado = %q", f.carryReason)
	}

	res, carry, err := s.ListCarry(ctx, 1, "Chefe")
	if err != nil || res != OK || !carry.Overridden || carry.DisplayName != "Chefe de Teste" || carry.Level != 120 {
		t.Fatalf("ListCarry = %v, %+v, %v", res, carry, err)
	}
	if carry.Slots[56].ItemName != "Poção de Vida" || carry.Slots[56].ItemIndex != 100 {
		t.Fatalf("slot 56 = %+v", carry.Slots[56])
	}
}

func TestSetCarryRejectsUnknownItemTemplateAndDuplicateSlot(t *testing.T) {
	s, _ := carryService(t)
	ctx := context.Background()
	cases := []struct {
		name string
		tpl  string
		in   []CarryInput
	}{
		{"template inexistente", "NaoExiste", []CarryInput{{Slot: 0, ItemIndex: 100}}},
		{"item fora do catálogo", "Chefe", []CarryInput{{Slot: 0, ItemIndex: 777}}},
		{"slot repetido", "Chefe", []CarryInput{{Slot: 0, ItemIndex: 100}, {Slot: 0, ItemIndex: 412}}},
		{"slot 64", "Chefe", []CarryInput{{Slot: 64, ItemIndex: 100}}},
		{"item zero", "Chefe", []CarryInput{{Slot: 0, ItemIndex: 0}}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if res, err := s.SetCarry(ctx, 1, tt.tpl, tt.in, ""); err != nil || res != Invalid {
				t.Fatalf("SetCarry = %v, %v; want Invalid", res, err)
			}
		})
	}
}

func TestResetCarryReturnsNotFoundWithoutOverride(t *testing.T) {
	s, f := carryService(t)
	ctx := context.Background()
	if res, err := s.ResetCarry(ctx, 1, "Chefe", ""); err != nil || res != NotFound {
		t.Fatalf("ResetCarry(vazio) = %v, %v; want NotFound", res, err)
	}
	f.carry["Chefe"] = []domain.MobTemplateEquipItem{{Slot: 0, ItemIndex: 412}}
	if res, err := s.ResetCarry(ctx, 1, "Chefe", ""); err != nil || res != OK {
		t.Fatalf("ResetCarry(override) = %v, %v; want OK", res, err)
	}
	if len(f.carry["Chefe"]) != 0 {
		t.Fatal("ResetCarry deixou linhas")
	}
}

func TestListRequiresModerator(t *testing.T) {
	s := New(newFake())
	if res, _, err := s.List(context.Background(), 3); err != nil || res != Forbidden {
		t.Fatalf("List(player) = %v, %v; want Forbidden", res, err)
	}
	if res, _, err := s.List(context.Background(), 99); err != nil || res != Forbidden {
		t.Fatalf("List(unknown) = %v, %v; want Forbidden", res, err)
	}
}

func TestListMarksOverridesOverTheCodeDefault(t *testing.T) {
	f := newFake()
	f.rows[12] = 450
	s := New(f)

	res, slots, err := s.List(context.Background(), 1)
	if err != nil || res != OK {
		t.Fatalf("List = %v, %v", res, err)
	}
	if len(slots) != 64 {
		t.Fatalf("len(slots) = %d, want 64", len(slots))
	}
	edited := slots[12]
	if !edited.Overridden || edited.Current != 450 || edited.Base == 450 {
		t.Fatalf("slot 12 = %+v, want overridden 450 with a different base", edited)
	}
	untouched := slots[13]
	if untouched.Overridden || untouched.Current != untouched.Base {
		t.Fatalf("slot 13 = %+v, want untouched base value", untouched)
	}
}

func TestUpsertValidatesRange(t *testing.T) {
	f := newFake()
	s := New(f)
	cases := []struct {
		name  string
		slot  int
		value int64
		want  Result
	}{
		{"ok", 12, 450, OK},
		{"slot negativo", -1, 450, Invalid},
		{"slot 64", 64, 450, Invalid},
		{"valor zero", 12, 0, Invalid},
		{"valor acima do teto", 12, MaxRate + 1, Invalid},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := s.Upsert(context.Background(), 1, tt.slot, tt.value, "teste")
			if err != nil {
				t.Fatalf("Upsert: %v", err)
			}
			if res != tt.want {
				t.Fatalf("Upsert = %v, want %v", res, tt.want)
			}
		})
	}
	if f.rows[12] != 450 || f.lastReason != "teste" {
		t.Fatalf("store rows = %v, last reason = %q", f.rows, f.lastReason)
	}
}

func TestResetReturnsNotFoundWithoutRow(t *testing.T) {
	f := newFake()
	s := New(f)
	if res, err := s.Reset(context.Background(), 1, 12, ""); err != nil || res != NotFound {
		t.Fatalf("Reset(empty) = %v, %v; want NotFound", res, err)
	}
	f.rows[12] = 450
	if res, err := s.Reset(context.Background(), 1, 12, ""); err != nil || res != OK {
		t.Fatalf("Reset(row) = %v, %v; want OK", res, err)
	}
	if _, ok := f.rows[12]; ok {
		t.Fatal("Reset left the row behind")
	}
}
