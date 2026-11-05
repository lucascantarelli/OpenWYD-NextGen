package ratesadmin

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"openwyd-nextgen/internal/ratetables"
	"openwyd-nextgen/internal/store"
)

type fakeStore struct {
	role      string
	overrides []store.ContentRateOverride

	upserts []store.ContentRateOverride
	deletes []string
	applied bool
	flags   []byte
}

func (f *fakeStore) AccountRole(context.Context, int64) (string, error) {
	if f.role == "" {
		return "", store.ErrNotFound
	}
	return f.role, nil
}
func (f *fakeStore) ListContentRateOverrides(context.Context) ([]store.ContentRateOverride, error) {
	return f.overrides, nil
}
func (f *fakeStore) UpsertContentRateOverride(_ context.Context, o store.ContentRateOverride, moderatorID int64, reason string) error {
	f.upserts = append(f.upserts, o)
	return nil
}
func (f *fakeStore) DeleteContentRateOverride(_ context.Context, file, family, name string, moderatorID int64, reason string) (bool, error) {
	f.deletes = append(f.deletes, file+"/"+family+"/"+name)
	return f.applied, nil
}
func (f *fakeStore) LatestInstanceFlags(context.Context, string) ([]byte, time.Time, bool, error) {
	if f.flags == nil {
		return nil, time.Time{}, false, nil
	}
	return f.flags, time.Date(2026, 11, 3, 17, 0, 0, 0, time.UTC), true, nil
}

func releaseDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "Release")
}

func TestListMergesOverlay(t *testing.T) {
	f := &fakeStore{
		role:      "admin",
		overrides: []store.ContentRateOverride{{File: ratetables.FileSancRate, Family: "PO", Name: "3", Value: 55}},
	}
	res, views, warning, err := New(f, releaseDir(t)).List(context.Background(), 1)
	if err != nil || res != OK {
		t.Fatalf("List = %v/%v; quer OK", res, err)
	}
	if warning != "" {
		t.Errorf("aviso inesperado: %q", warning)
	}

	var cell *RowView
	for i := range views {
		if views[i].Row.File == ratetables.FileSancRate && views[i].Row.Family == "PO" && views[i].Row.Name == "3" {
			cell = &views[i]
		}
	}
	if cell == nil {
		t.Fatal("célula PO/3 não apareceu na lista")
	}
	if cell.Row.Value != 55 || cell.Original != 85 || !cell.HasOverride {
		t.Errorf("célula = %+v; quer atual 55, original 85, com override", *cell)
	}
}

func TestListWithoutContentWarns(t *testing.T) {
	res, views, warning, err := New(&fakeStore{role: "admin"}, "").List(context.Background(), 1)
	if res != OK || err != nil || len(views) != 0 || warning == "" {
		t.Fatalf("sem content = %v/%d/%q/%v; quer OK, 0 linhas e aviso", res, len(views), warning, err)
	}
}

func TestUpsertValidatesCellAndRange(t *testing.T) {
	f := &fakeStore{role: "admin"}
	svc := New(f, releaseDir(t))
	ctx := context.Background()

	if res, _, _ := svc.Upsert(ctx, 1, ratetables.FileSancRate, "PO", "3", 55, ""); res != Invalid {
		t.Errorf("sem motivo = %v; quer Invalid", res)
	}
	if res, _, _ := svc.Upsert(ctx, 1, ratetables.FileSancRate, "PO", "99", 55, "teste"); res != NotFound {
		t.Errorf("célula inexistente = %v; quer NotFound", res)
	}
	if res, msg, _ := svc.Upsert(ctx, 1, ratetables.FileSancRate, "PO", "3", 101, "teste"); res != Invalid {
		t.Errorf("taxa 101 = %v (%q); quer Invalid", res, msg)
	}
	if res, _, err := svc.Upsert(ctx, 1, ratetables.FileSancRate, "PO", "3", 55, "balanceamento"); res != OK || err != nil {
		t.Fatalf("taxa válida = %v/%v; quer OK", res, err)
	}
	if len(f.upserts) != 1 || f.upserts[0].Value != 55 || f.upserts[0].Name != "3" {
		t.Errorf("upserts = %+v", f.upserts)
	}
}

// CompRate is the reason the key is textual: two keys of one family must be two
// independent cells.
func TestCompositorKeysAreIndependentCells(t *testing.T) {
	f := &fakeStore{role: "admin"}
	svc := New(f, releaseDir(t))
	ctx := context.Background()

	if res, _, err := svc.Upsert(ctx, 1, ratetables.FileCompRate, "Compositor", "Item_+7", 3, "taxa de anct"); res != OK || err != nil {
		t.Fatalf("Item_+7 = %v/%v; quer OK", res, err)
	}
	if res, _, err := svc.Upsert(ctx, 1, ratetables.FileCompRate, "Compositor", "Item_+9", 12, "taxa de anct"); res != OK || err != nil {
		t.Fatalf("Item_+9 = %v/%v; quer OK", res, err)
	}
	if len(f.upserts) != 2 || f.upserts[0].Name == f.upserts[1].Name {
		t.Fatalf("upserts = %+v; quer duas células distintas", f.upserts)
	}
}

func TestResetRequiresExistingRow(t *testing.T) {
	f := &fakeStore{role: "admin"}
	svc := New(f, releaseDir(t))

	if res, _, _ := svc.Reset(context.Background(), 1, ratetables.FileSancRate, "PO", "3", ""); res != Invalid {
		t.Errorf("sem motivo = %v; quer Invalid", res)
	}
	if res, _, _ := svc.Reset(context.Background(), 1, ratetables.FileSancRate, "PO", "3", "voltar"); res != NotFound {
		t.Errorf("sem linha = %v; quer NotFound", res)
	}
	f.applied = true
	f.deletes = nil
	if res, _, err := svc.Reset(context.Background(), 1, ratetables.FileSancRate, "PO", "3", "voltar"); res != OK || err != nil {
		t.Fatalf("reset = %v/%v; quer OK", res, err)
	}
	if len(f.deletes) != 1 || f.deletes[0] != "SancRate/PO/3" {
		t.Errorf("deletes = %v", f.deletes)
	}
}

func TestFlagsReadsLatestSnapshot(t *testing.T) {
	f := &fakeStore{role: "admin"}
	svc := New(f, releaseDir(t))

	res, flags, _, available, err := svc.Flags(context.Background(), 1, "tmserver")
	if res != OK || err != nil || available || len(flags) != 0 {
		t.Fatalf("sem snapshot = %v/%d/%v/%v; quer OK e available=false", res, len(flags), available, err)
	}

	f.flags = []byte(`[{"name":"rates-editing","value":"true","restart_required":true,"hint":"overlay de taxas"}]`)
	res, flags, captured, available, err := svc.Flags(context.Background(), 1, "tmserver")
	if res != OK || err != nil || !available || len(flags) != 1 {
		t.Fatalf("com snapshot = %v/%d/%v/%v; quer OK/1/true", res, len(flags), available, err)
	}
	if flags[0].Name != "rates-editing" || flags[0].Value != "true" || !flags[0].RestartRequired {
		t.Errorf("flag = %+v", flags[0])
	}
	if captured.IsZero() {
		t.Error("captured_at vazio com snapshot disponível")
	}
}

func TestPlayerIsForbidden(t *testing.T) {
	f := &fakeStore{role: "player"}
	if res, _, _, _ := New(f, releaseDir(t)).List(context.Background(), 9); res != Forbidden {
		t.Errorf("player = %v; quer Forbidden", res)
	}
}

func TestQuestCellRange(t *testing.T) {
	f := &fakeStore{role: "admin"}
	svc := New(f, releaseDir(t))
	// EXP tier 0 mortal: teto de 2 bilhões, não 100.
	if res, msg, _ := svc.Upsert(context.Background(), 1, ratetables.FileQuestsRate, "EXP", "0.0", 35000, "temporada"); res != OK {
		t.Errorf("EXP 35000 = %v (%q); o teto de recompensa não é 100", res, msg)
	}
	if res, _, _ := svc.Upsert(context.Background(), 1, ratetables.FileQuestsRate, "EXP", "0.0", ratetables.MaxReward+1, "temporada"); res != Invalid {
		t.Errorf("EXP acima do teto = %v; quer Invalid", res)
	}
}
