package teleportadmin

import (
	"context"
	"testing"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/store"
	"openwyd-nextgen/internal/teleportdata"
)

type fakeStore struct {
	role    string
	rows    []domain.TeleportRoute
	upserts []domain.TeleportRoute
	deleted [][2]int
}

func (f *fakeStore) AccountRole(context.Context, int64) (string, error) {
	if f.role == "" {
		return "", store.ErrNotFound
	}
	return f.role, nil
}

func (f *fakeStore) ListTeleportRoutes(context.Context) ([]domain.TeleportRoute, error) {
	return f.rows, nil
}

func (f *fakeStore) UpsertTeleportRoute(_ context.Context, r domain.TeleportRoute, _ int64) error {
	f.upserts = append(f.upserts, r)
	f.rows = append(f.rows, r)
	return nil
}

func (f *fakeStore) SetTeleportRouteEnabled(_ context.Context, srcX, srcY int, enabled bool, _ int64) (bool, error) {
	for i, r := range f.rows {
		if r.SrcX == srcX && r.SrcY == srcY {
			f.rows[i].Enabled = enabled
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeStore) DeleteTeleportRoute(_ context.Context, srcX, srcY int, _ int64) (bool, error) {
	for i, r := range f.rows {
		if r.SrcX == srcX && r.SrcY == srcY {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			f.deleted = append(f.deleted, [2]int{srcX, srcY})
			return true, nil
		}
	}
	return false, nil
}

func TestListMergesCompiledAndOverlay(t *testing.T) {
	st := &fakeStore{role: "moderator", rows: []domain.TeleportRoute{{
		SrcX: 2116, SrcY: 2100, DstX: 1200, DstY: 1300, Cost: 1000,
		Requirement: teleportdata.RequirementNone, Enabled: true, Note: "teste",
	}}}
	res, views, err := New(st).List(context.Background(), 1)
	if err != nil || res != OK {
		t.Fatalf("List = %v, %v", res, err)
	}
	if len(views) != len(teleportdata.Routes()) {
		t.Fatalf("merged %d routes, want %d (compiled rows are listed too)", len(views), len(teleportdata.Routes()))
	}
	var overridden *RouteView
	for i := range views {
		if views[i].Route.SrcX == 2116 && views[i].Route.SrcY == 2100 {
			overridden = &views[i]
		}
	}
	if overridden == nil || !overridden.HasOverride || !overridden.FromContent || overridden.Route.Cost != 1000 {
		t.Fatalf("override not applied: %+v", overridden)
	}
}

func TestUpsertNormalizesOriginAndRejectsInvalid(t *testing.T) {
	st := &fakeStore{role: "admin"}
	svc := New(st)

	res, warnings, route, err := svc.Upsert(context.Background(), 1, domain.TeleportRoute{
		SrcX: 2118, SrcY: 2102, DstX: 1200, DstY: 1300, Cost: 0,
		Requirement: teleportdata.RequirementNone, Enabled: true,
	})
	if err != nil || res != OK {
		t.Fatalf("Upsert = %v, %v", res, err)
	}
	if route.SrcX != 2116 || route.SrcY != 2100 {
		t.Fatalf("origin not normalized: %d,%d", route.SrcX, route.SrcY)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want one normalization notice", warnings)
	}

	for _, bad := range []domain.TeleportRoute{
		{SrcX: 2116, SrcY: 2100, DstX: 5000, DstY: 0, Requirement: teleportdata.RequirementNone},
		{SrcX: 2116, SrcY: 2100, DstX: 0, DstY: 0, Cost: -1, Requirement: teleportdata.RequirementNone},
		{SrcX: 2116, SrcY: 2100, DstX: 0, DstY: 0, Requirement: "assert"},
	} {
		if res, _, _, err := svc.Upsert(context.Background(), 1, bad); res != Invalid || err != nil {
			t.Fatalf("Upsert(%+v) = %v, %v; want Invalid", bad, res, err)
		}
	}
}

// TestSetEnabledCreatesRowForCompiledRoute is the "desativar direto da lista"
// path: the 23 default routes have no row until someone disables one.
func TestSetEnabledCreatesRowForCompiledRoute(t *testing.T) {
	st := &fakeStore{role: "moderator"}
	res, err := New(st).SetEnabled(context.Background(), 1, 2116, 2100, false)
	if err != nil || res != OK {
		t.Fatalf("SetEnabled = %v, %v", res, err)
	}
	if len(st.upserts) != 1 || st.upserts[0].Enabled || st.upserts[0].Cost != 700 {
		t.Fatalf("row not created from the compiled route: %+v", st.upserts)
	}

	// A tile that is not a route at all cannot be toggled.
	if res, err := New(st).SetEnabled(context.Background(), 1, 4, 4, false); res != NotFound || err != nil {
		t.Fatalf("SetEnabled(non-route) = %v, %v; want NotFound", res, err)
	}
}

func TestDeleteAndSimulate(t *testing.T) {
	st := &fakeStore{role: "moderator", rows: []domain.TeleportRoute{{
		SrcX: 2480, SrcY: 1716, DstX: 1044, DstY: 1716, Cost: 700,
		Requirement: teleportdata.RequirementNone, Enabled: false,
	}}}
	svc := New(st)

	res, sim, err := svc.Simulate(context.Background(), 1, 2481, 1717)
	if err != nil || res != OK {
		t.Fatalf("Simulate = %v, %v", res, err)
	}
	if sim.SrcX != 2480 || sim.SrcY != 1716 {
		t.Fatalf("simulated origin = %d,%d, want the rounded tile", sim.SrcX, sim.SrcY)
	}
	if sim.OK {
		t.Fatal("disabled route simulated as reachable")
	}
	if len(sim.Warnings) != 2 { // rounding + disabled notice
		t.Fatalf("warnings = %v, want rounding + disabled", sim.Warnings)
	}

	if res, err := svc.Delete(context.Background(), 1, 2480, 1716); err != nil || res != OK {
		t.Fatalf("Delete = %v, %v", res, err)
	}
	if res, err := svc.Delete(context.Background(), 1, 2480, 1716); res != NotFound || err != nil {
		t.Fatalf("second Delete = %v, %v; want NotFound", res, err)
	}
}

func TestForbiddenWithoutStaffRole(t *testing.T) {
	svc := New(&fakeStore{role: "player"})
	if res, _, err := svc.List(context.Background(), 1); res != Forbidden || err != nil {
		t.Fatalf("List(player) = %v, %v; want Forbidden", res, err)
	}
	if res, _, err := svc.List(context.Background(), 0); res != Forbidden || err != nil {
		t.Fatalf("List(anonymous) = %v, %v; want Forbidden", res, err)
	}
}
