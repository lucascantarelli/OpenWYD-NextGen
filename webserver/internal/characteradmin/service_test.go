package characteradmin

import (
	"context"
	"testing"

	"github.com/jeanluca/w2pp-openwyd/internal/domain"
	"github.com/jeanluca/w2pp-openwyd/internal/store"
)

type fakeStore struct {
	role     string
	hits     []store.CharacterSearchHit
	char     domain.Character
	detail   Detail
	notFound bool
	grants   []domain.Item
	quant    int
	reason   string
}

func (f *fakeStore) AccountRole(context.Context, int64) (string, error) {
	if f.role == "" {
		return "", store.ErrNotFound
	}
	return f.role, nil
}

func (f *fakeStore) SearchCharacters(context.Context, string, int) ([]store.CharacterSearchHit, error) {
	return f.hits, nil
}

func (f *fakeStore) LoadCharacterByID(context.Context, int64) (domain.Character, int64, string, error) {
	if f.notFound {
		return domain.Character{}, 0, "", store.ErrNotFound
	}
	return f.detail.Character, f.detail.AccountID, f.detail.AccountName, nil
}

func (f *fakeStore) EnqueueItemGrant(_ context.Context, _, _, _ int64, item domain.Item, quantity int, _, reason string) ([]int64, error) {
	f.grants = append(f.grants, item)
	f.quant = quantity
	f.reason = reason
	return []int64{1, 2}, nil
}

func (f *fakeStore) PendingItemGrants(context.Context, int64, int) ([]store.CharacterItemGrant, error) {
	return nil, nil
}

// TestGrantValidationAndQueue covers the issue-43 grant rules: staff only, item
// index/quantity bounds and a required reason, with the units landing in the
// mailbox rather than in the character row.
func TestGrantValidationAndQueue(t *testing.T) {
	st := &fakeStore{role: "moderator", detail: Detail{AccountID: 7}}
	svc := New(st)
	item := domain.Item{Index: 1234, Eff1: 2, EffV1: 10}

	if res, _, err := svc.Grant(context.Background(), 1, 1, item, 0, "teste"); res != Invalid || err != nil {
		t.Fatalf("Grant(quantity=0) = %v, %v; want Invalid", res, err)
	}
	if res, _, err := svc.Grant(context.Background(), 1, 1, item, 1, "  "); res != Invalid || err != nil {
		t.Fatalf("Grant(no reason) = %v, %v; want Invalid", res, err)
	}
	if res, _, err := svc.Grant(context.Background(), 1, 1, domain.Item{}, 1, "teste"); res != Invalid || err != nil {
		t.Fatalf("Grant(no item) = %v, %v; want Invalid", res, err)
	}

	st.notFound = true
	if res, _, err := svc.Grant(context.Background(), 1, 99, item, 1, "teste"); res != NotFound || err != nil {
		t.Fatalf("Grant(missing character) = %v, %v; want NotFound", res, err)
	}
	st.notFound = false

	res, ids, err := svc.Grant(context.Background(), 1, 1, item, 2, "recompensa de evento")
	if err != nil || res != OK {
		t.Fatalf("Grant = %v, %v", res, err)
	}
	if len(ids) != 2 || st.quant != 2 || st.reason != "recompensa de evento" {
		t.Fatalf("queued ids=%v quantity=%d reason=%q", ids, st.quant, st.reason)
	}
}

func TestListGrantsRequiresStaff(t *testing.T) {
	st := &fakeStore{role: "player"}
	if res, _, err := New(st).ListGrants(context.Background(), 3, 1); res != Forbidden || err != nil {
		t.Fatalf("ListGrants(player) = %v, %v; want Forbidden", res, err)
	}
}

func TestSearchRequiresStaffAndRealQuery(t *testing.T) {
	st := &fakeStore{role: "player", hits: []store.CharacterSearchHit{{Name: "Cantarelli"}}}
	svc := New(st)

	if res, _, err := svc.Search(context.Background(), 3, "canta"); res != Forbidden || err != nil {
		t.Fatalf("Search(player) = %v, %v; want Forbidden", res, err)
	}

	st.role = "moderator"
	// One letter would scan the whole table.
	if res, _, err := svc.Search(context.Background(), 1, "c"); res != Invalid || err != nil {
		t.Fatalf("Search(short query) = %v, %v; want Invalid", res, err)
	}
	res, hits, err := svc.Search(context.Background(), 1, "  cantarelli ")
	if err != nil || res != OK || len(hits) != 1 {
		t.Fatalf("Search = %v, %d hits, %v", res, len(hits), err)
	}
}

func TestGetReturnsDetailOrNotFound(t *testing.T) {
	st := &fakeStore{
		role: "admin",
		detail: Detail{
			Character:   domain.Character{Name: "Cantarelli", Level: 12, Coin: 5000},
			AccountID:   7,
			AccountName: "test",
		},
	}
	svc := New(st)

	res, detail, err := svc.Get(context.Background(), 1, 42)
	if err != nil || res != OK {
		t.Fatalf("Get = %v, %v", res, err)
	}
	if detail.Character.Name != "Cantarelli" || detail.AccountName != "test" || detail.AccountID != 7 {
		t.Fatalf("detail = %+v", detail)
	}

	st.notFound = true
	if res, _, err := svc.Get(context.Background(), 1, 42); res != NotFound || err != nil {
		t.Fatalf("Get(missing) = %v, %v; want NotFound", res, err)
	}
	if res, _, err := svc.Get(context.Background(), 1, 0); res != Invalid || err != nil {
		t.Fatalf("Get(id=0) = %v, %v; want Invalid", res, err)
	}
}
