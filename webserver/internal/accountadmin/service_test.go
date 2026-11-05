package accountadmin

import (
	"context"
	"errors"
	"testing"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/store"
)

// fakeStore implements just enough of the store to exercise the service rules.
type fakeStore struct {
	actorRole string
	rows      []domain.AccountRow
	row       domain.AccountRow
	audit     []domain.AccountAudit
	roleErr   error
	writeErr  error
	createdID int64

	lastRole    string
	lastReason  string
	lastBlocked bool
	lastFilter  domain.AccountFilter
	created     bool
}

func (f *fakeStore) AccountRole(context.Context, int64) (string, error) {
	if f.roleErr != nil {
		return "", f.roleErr
	}
	return f.actorRole, nil
}

func (f *fakeStore) ListAccounts(_ context.Context, filter domain.AccountFilter) ([]domain.AccountRow, error) {
	f.lastFilter = filter
	return f.rows, nil
}

func (f *fakeStore) CountAccounts(context.Context, domain.AccountFilter) (int, error) {
	return len(f.rows), nil
}

func (f *fakeStore) AccountAdminRow(context.Context, int64) (domain.AccountRow, error) {
	return f.row, nil
}

func (f *fakeStore) ListAccountAudit(context.Context, int64, int) ([]domain.AccountAudit, error) {
	return f.audit, nil
}

func (f *fakeStore) SetAccountRole(_ context.Context, _ int64, _ int64, role, reason string) (domain.AccountRow, error) {
	f.lastRole, f.lastReason = role, reason
	if f.writeErr != nil {
		return domain.AccountRow{}, f.writeErr
	}
	return f.row, nil
}

func (f *fakeStore) SetAccountBlocked(_ context.Context, _ int64, _ int64, blocked bool, reason string) (domain.AccountRow, error) {
	f.lastBlocked, f.lastReason = blocked, reason
	if f.writeErr != nil {
		return domain.AccountRow{}, f.writeErr
	}
	return f.row, nil
}

func (f *fakeStore) CreateAccountAdmin(_ context.Context, _, _, _, _ string, _ int64, _ string) (int64, error) {
	f.created = true
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.createdID, nil
}

func TestListRequiresStaff(t *testing.T) {
	svc := New(&fakeStore{actorRole: "player"})
	res, _, _, err := svc.List(context.Background(), 2, domain.AccountFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != Forbidden {
		t.Fatalf("player list: got %v, want Forbidden", res)
	}

	svc = New(&fakeStore{actorRole: "moderator", rows: []domain.AccountRow{{ID: 1}}})
	res, rows, total, err := svc.List(context.Background(), 2, domain.AccountFilter{Query: "te"})
	if err != nil || res != OK {
		t.Fatalf("moderator list: res=%v err=%v", res, err)
	}
	if len(rows) != 1 || total != 1 {
		t.Fatalf("rows=%d total=%d", len(rows), total)
	}
}

func TestSetRoleIsAdminOnly(t *testing.T) {
	for _, role := range []string{"player", "moderator"} {
		svc := New(&fakeStore{actorRole: role})
		res, _, _, err := svc.SetRole(context.Background(), 2, 3, "admin", "promoção combinada")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", role, err)
		}
		if res != Forbidden {
			t.Fatalf("%s set role: got %v, want Forbidden", role, res)
		}
	}
}

func TestSetRoleValidatesRoleAndReason(t *testing.T) {
	svc := New(&fakeStore{actorRole: "admin"})
	if res, _, msg, _ := svc.SetRole(context.Background(), 1, 2, "root", "motivo suficientemente longo"); res != Invalid || msg != "papel inválido" {
		t.Fatalf("invalid role: res=%v msg=%q", res, msg)
	}
	if res, _, msg, _ := svc.SetRole(context.Background(), 1, 2, "moderator", "curto"); res != Invalid || msg == "" {
		t.Fatalf("short reason: res=%v msg=%q", res, msg)
	}
}

func TestSetRoleBusinessRulesMapToMessages(t *testing.T) {
	cases := []struct {
		err     error
		want    Result
		message string
	}{
		{store.ErrSelfChange, Invalid, "não é possível alterar o próprio papel ou bloqueio"},
		{store.ErrLastAdmin, Invalid, "esta é a última conta admin ativa — promova outra antes"},
		{store.ErrNotFound, NotFound, "conta não encontrada"},
		{store.ErrNotAdminActor, Forbidden, "só um admin pode alterar papéis"},
	}
	for _, tc := range cases {
		svc := New(&fakeStore{actorRole: "admin", writeErr: tc.err})
		res, _, msg, err := svc.SetRole(context.Background(), 1, 2, "moderator", "motivo suficientemente longo")
		if err != nil {
			t.Fatalf("%v: unexpected error: %v", tc.err, err)
		}
		if res != tc.want || msg != tc.message {
			t.Fatalf("%v: got res=%v msg=%q, want res=%v msg=%q", tc.err, res, msg, tc.want, tc.message)
		}
	}
}

func TestSetBlockedScopeAndReason(t *testing.T) {
	svc := New(&fakeStore{actorRole: "moderator", writeErr: store.ErrModeratorScope})
	res, _, msg, err := svc.SetBlocked(context.Background(), 2, 9, true, "tentativa de bloquear admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != Forbidden || msg == "" {
		t.Fatalf("moderator blocking admin: res=%v msg=%q", res, msg)
	}

	svc = New(&fakeStore{actorRole: "admin"})
	if res, _, _, _ := svc.SetBlocked(context.Background(), 1, 2, true, "curto"); res != Invalid {
		t.Fatalf("short reason accepted: %v", res)
	}
}

func TestCreateValidatesInput(t *testing.T) {
	svc := New(&fakeStore{actorRole: "admin", createdID: 42})
	ctx := context.Background()

	if res, _, _, _ := svc.Create(ctx, 1, "ab", "senha", "", "player", "motivo suficientemente longo"); res != Invalid {
		t.Fatalf("short name accepted")
	}
	if res, _, _, _ := svc.Create(ctx, 1, "novo", "123", "", "player", "motivo suficientemente longo"); res != Invalid {
		t.Fatalf("short password accepted")
	}
	if res, _, _, _ := svc.Create(ctx, 1, "novo", "senha", "nao-email", "player", "motivo suficientemente longo"); res != Invalid {
		t.Fatalf("invalid email accepted")
	}
	if res, _, _, _ := svc.Create(ctx, 1, "novo", "senha", "", "root", "motivo suficientemente longo"); res != Invalid {
		t.Fatalf("invalid role accepted")
	}

	res, id, _, err := svc.Create(ctx, 1, " Novo ", "senha", "", "moderator", "motivo suficientemente longo")
	if err != nil || res != OK || id != 42 {
		t.Fatalf("create: res=%v id=%d err=%v", res, id, err)
	}
}

func TestCreateRequiresAdmin(t *testing.T) {
	svc := New(&fakeStore{actorRole: "moderator"})
	res, _, _, err := svc.Create(context.Background(), 2, "novo", "senha", "", "player", "motivo suficientemente longo")
	if err != nil || res != Forbidden {
		t.Fatalf("moderator create: res=%v err=%v", res, err)
	}
}

func TestGetNotFound(t *testing.T) {
	svc := New(&fakeStore{actorRole: "moderator", writeErr: nil})
	svc.store = &notFoundStore{fakeStore: &fakeStore{actorRole: "moderator"}}
	res, _, _, err := svc.Get(context.Background(), 2, 99)
	if err != nil || res != NotFound {
		t.Fatalf("get missing: res=%v err=%v", res, err)
	}
}

// notFoundStore makes AccountAdminRow report ErrNotFound.
type notFoundStore struct{ *fakeStore }

func (n *notFoundStore) AccountAdminRow(context.Context, int64) (domain.AccountRow, error) {
	return domain.AccountRow{}, store.ErrNotFound
}

func TestExplainUnknownErrorIsInfra(t *testing.T) {
	svc := New(&fakeStore{actorRole: "admin", writeErr: errors.New("connection reset")})
	res, _, _, err := svc.SetRole(context.Background(), 1, 2, "player", "motivo suficientemente longo")
	if err == nil {
		t.Fatalf("unknown store error must surface as an error")
	}
	if res != Invalid {
		t.Fatalf("res=%v, want Invalid", res)
	}
}
