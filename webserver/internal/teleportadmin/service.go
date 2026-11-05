// Package teleportadmin is the web-api side of the moderator teleport editor
// (issue 42): it merges the compiled route table (internal/teleportdata) with the
// teleport_route overlay, validates writes, and never touches live game state.
// tmServer polls the table and publishes its own read-only snapshot.
package teleportadmin

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/store"
	"openwyd-nextgen/internal/teleportdata"
)

// WorldSize bounds every coordinate: the world grid is 4096×4096 cells and the
// AttributeMap covers it 4:1.
const WorldSize = 4096

// Store is the persistence surface the service needs (satisfied by *store.Store).
type Store interface {
	AccountRole(ctx context.Context, id int64) (string, error)
	ListTeleportRoutes(ctx context.Context) ([]domain.TeleportRoute, error)
	UpsertTeleportRoute(ctx context.Context, r domain.TeleportRoute, moderatorID int64) error
	SetTeleportRouteEnabled(ctx context.Context, srcX, srcY int, enabled bool, moderatorID int64) (bool, error)
	DeleteTeleportRoute(ctx context.Context, srcX, srcY int, moderatorID int64) (bool, error)
}

// Result is the business outcome of an admin operation. Only infra failures are
// returned as errors; these ride in the response body (web.proto AdminResult).
type Result int

const (
	// OK means the operation succeeded.
	OK Result = iota
	// Forbidden means the caller is not a moderator/admin.
	Forbidden
	// Invalid means the request failed validation.
	Invalid
	// NotFound means there is no overlay row (and, for writes that need one, no
	// compiled route either).
	NotFound
)

// RouteView is one row of the merged list: the effective route plus where it
// came from, so the UI can say "padrão do servidor" vs "editado na web".
type RouteView struct {
	Route domain.TeleportRoute
	// FromContent is true when the origin exists in the compiled table.
	FromContent bool
	// HasOverride is true when a teleport_route row exists for the origin.
	HasOverride bool
}

// Simulation is an origin resolved without consuming the world's RNG.
type Simulation struct {
	OK          bool
	SrcX, SrcY  int
	DstX, DstY  int
	Cost        int
	Requirement string
	Warnings    []string
}

// Service implements the moderator teleport operations.
type Service struct {
	store Store
}

// New builds the service over the given store.
func New(s Store) *Service { return &Service{store: s} }

// List returns the merged view (compiled + overlay), ordered by origin.
func (s *Service) List(ctx context.Context, moderatorID int64) (Result, []RouteView, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, nil, err
	}
	overlay, err := s.store.ListTeleportRoutes(ctx)
	if err != nil {
		return Invalid, nil, fmt.Errorf("teleportadmin: list: %w", err)
	}
	return OK, merge(overlay), nil
}

// Upsert validates and stores the route at the (normalized) origin tile. The
// returned route carries the tile actually stored; warnings explain any
// normalization the caller should show the moderator.
func (s *Service) Upsert(ctx context.Context, moderatorID int64, in domain.TeleportRoute) (Result, []string, domain.TeleportRoute, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, nil, domain.TeleportRoute{}, err
	}
	route, warnings, ok := normalize(in)
	if !ok {
		return Invalid, warnings, domain.TeleportRoute{}, nil
	}
	if err := s.store.UpsertTeleportRoute(ctx, route, moderatorID); err != nil {
		return Invalid, warnings, domain.TeleportRoute{}, fmt.Errorf("teleportadmin: upsert: %w", err)
	}
	return OK, warnings, route, nil
}

// SetEnabled toggles a route. A compiled route with no overlay row gets one
// created (copying the compiled values) so "desativar" works on the 23 default
// routes straight from the list.
func (s *Service) SetEnabled(ctx context.Context, moderatorID int64, srcX, srcY int, enabled bool) (Result, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, err
	}
	nx, ny := srcX&^3, srcY&^3
	if !validCoord(nx) || !validCoord(ny) {
		return Invalid, nil
	}
	applied, err := s.store.SetTeleportRouteEnabled(ctx, nx, ny, enabled, moderatorID)
	if err != nil {
		return Invalid, fmt.Errorf("teleportadmin: set enabled: %w", err)
	}
	if applied {
		return OK, nil
	}
	// No row yet: only a compiled route can be toggled this way, and it becomes
	// an overlay row so the world sees the change on its next poll.
	compiled, ok := teleportdata.Lookup(int16(nx), int16(ny))
	if !ok {
		return NotFound, nil
	}
	route := domain.TeleportRoute{
		SrcX: nx, SrcY: ny, DstX: int(compiled.DstX), DstY: int(compiled.DstY),
		Cost: int(compiled.Cost), Requirement: compiled.Requirement,
		Enabled: enabled, Note: "desativada pela web (rota do conteúdo)",
	}
	if err := s.store.UpsertTeleportRoute(ctx, route, moderatorID); err != nil {
		return Invalid, fmt.Errorf("teleportadmin: upsert on toggle: %w", err)
	}
	return OK, nil
}

// Delete drops the overlay row, restoring the compiled route (if any). Deleting
// a compiled route is not possible: disable it instead.
func (s *Service) Delete(ctx context.Context, moderatorID int64, srcX, srcY int) (Result, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, err
	}
	nx, ny := srcX&^3, srcY&^3
	if !validCoord(nx) || !validCoord(ny) {
		return Invalid, nil
	}
	applied, err := s.store.DeleteTeleportRoute(ctx, nx, ny, moderatorID)
	if err != nil {
		return Invalid, fmt.Errorf("teleportadmin: delete: %w", err)
	}
	if !applied {
		return NotFound, nil
	}
	return OK, nil
}

// Simulate resolves an origin against the merged routes. It returns ok=false
// when the tile teleports nowhere (no route, or the route is disabled); the
// +0..2 destination spread stays in tmServer, so the UI shows the base tile.
func (s *Service) Simulate(ctx context.Context, moderatorID int64, srcX, srcY int) (Result, Simulation, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, Simulation{}, err
	}
	if srcX < 0 || srcX >= WorldSize || srcY < 0 || srcY >= WorldSize {
		return Invalid, Simulation{}, nil
	}
	nx, ny := srcX&^3, srcY&^3
	sim := Simulation{SrcX: nx, SrcY: ny}
	if nx != srcX || ny != srcY {
		sim.Warnings = append(sim.Warnings, fmt.Sprintf(
			"origem %d,%d não é múltipla de 4: o servidor resolve por %d,%d", srcX, srcY, nx, ny))
	}
	for _, view := range merge(mustOverlay(ctx, s.store)) {
		if view.Route.SrcX != nx || view.Route.SrcY != ny {
			continue
		}
		if !view.Route.Enabled {
			sim.Warnings = append(sim.Warnings, "esta rota está desativada: o tile não teleporta")
			return OK, sim, nil
		}
		sim.OK = true
		sim.DstX, sim.DstY = view.Route.DstX, view.Route.DstY
		sim.Cost, sim.Requirement = view.Route.Cost, view.Route.Requirement
		return OK, sim, nil
	}
	return OK, sim, nil
}

// mustOverlay reads the overlay rows for Simulate; a read failure yields an empty
// overlay so the compiled table still answers (the field list is what matters).
func mustOverlay(ctx context.Context, st Store) []domain.TeleportRoute {
	rows, err := st.ListTeleportRoutes(ctx)
	if err != nil {
		return nil
	}
	return rows
}

func (s *Service) authorize(ctx context.Context, moderatorID int64) (Result, error) {
	if moderatorID <= 0 {
		return Forbidden, nil
	}
	role, err := s.store.AccountRole(ctx, moderatorID)
	if errors.Is(err, store.ErrNotFound) {
		return Forbidden, nil
	}
	if err != nil {
		return Invalid, fmt.Errorf("teleportadmin: role lookup %d: %w", moderatorID, err)
	}
	if role != "moderator" && role != "admin" {
		return Forbidden, nil
	}
	return OK, nil
}

// merge applies the overlay on top of the compiled table and sorts by origin.
func merge(overlay []domain.TeleportRoute) []RouteView {
	views := make(map[[2]int]RouteView, len(overlay)+len(teleportdata.Routes()))
	for _, r := range teleportdata.Routes() {
		key := [2]int{int(r.SrcX), int(r.SrcY)}
		views[key] = RouteView{
			Route: domain.TeleportRoute{
				SrcX: int(r.SrcX), SrcY: int(r.SrcY), DstX: int(r.DstX), DstY: int(r.DstY),
				Cost: int(r.Cost), Requirement: r.Requirement, Enabled: true,
			},
			FromContent: true,
		}
	}
	for _, row := range overlay {
		key := [2]int{row.SrcX, row.SrcY}
		view := views[key]
		view.Route = row
		view.HasOverride = true
		views[key] = view
	}
	out := make([]RouteView, 0, len(views))
	for _, v := range views {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Route.SrcX != out[j].Route.SrcX {
			return out[i].Route.SrcX < out[j].Route.SrcX
		}
		return out[i].Route.SrcY < out[j].Route.SrcY
	})
	return out
}

func validCoord(v int) bool { return v >= 0 && v < WorldSize }

// normalize validates a route and rounds its origin to the tile the server
// actually resolves (x &^ 3), reporting the rounding instead of hiding it.
func normalize(in domain.TeleportRoute) (domain.TeleportRoute, []string, bool) {
	var warnings []string
	if !validCoord(in.SrcX) || !validCoord(in.SrcY) || !validCoord(in.DstX) || !validCoord(in.DstY) {
		return domain.TeleportRoute{}, nil, false
	}
	if in.Cost < 0 {
		return domain.TeleportRoute{}, nil, false
	}
	req := in.Requirement
	if req == "" {
		req = teleportdata.RequirementNone
	}
	switch req {
	case teleportdata.RequirementNone, teleportdata.RequirementFairy, teleportdata.RequirementKefra:
	default:
		return domain.TeleportRoute{}, nil, false
	}
	nx, ny := in.SrcX&^3, in.SrcY&^3
	if nx != in.SrcX || ny != in.SrcY {
		warnings = append(warnings, fmt.Sprintf(
			"origem %d,%d não é múltipla de 4: normalizada para %d,%d (é o tile que o servidor resolve)",
			in.SrcX, in.SrcY, nx, ny))
	}
	out := in
	out.SrcX, out.SrcY = nx, ny
	out.Requirement = req
	if out.Note == "" {
		out.Note = ""
	}
	return out, warnings, true
}
