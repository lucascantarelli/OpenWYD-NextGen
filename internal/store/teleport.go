package store

import (
	"context"
	"fmt"

	"github.com/jeanluca/w2pp-openwyd/internal/domain"
)

// ListTeleportRoutes returns the moderator-editable teleport overlay, ordered by
// origin tile (issue 42). The overlay is additive: rows here override the
// compiled table in internal/teleportdata, and an origin without a row keeps the
// compiled route.
func (s *Store) ListTeleportRoutes(ctx context.Context) ([]domain.TeleportRoute, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT src_x, src_y, dst_x, dst_y, cost, requirement, enabled, note,
		       COALESCE(updated_by, 0), updated_at
		FROM teleport_route
		ORDER BY src_x, src_y`)
	if err != nil {
		return nil, fmt.Errorf("store: list teleport routes: %w", err)
	}
	defer rows.Close()

	out := make([]domain.TeleportRoute, 0, 24)
	for rows.Next() {
		var r domain.TeleportRoute
		if err := rows.Scan(&r.SrcX, &r.SrcY, &r.DstX, &r.DstY, &r.Cost,
			&r.Requirement, &r.Enabled, &r.Note, &r.UpdatedBy, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan teleport route: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: read teleport routes: %w", err)
	}
	return out, nil
}

// UpsertTeleportRoute stores (or replaces) the route at r.SrcX,r.SrcY. The
// origin tile is the key, so editing a route never duplicates it.
func (s *Store) UpsertTeleportRoute(ctx context.Context, r domain.TeleportRoute, moderatorID int64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO teleport_route
			(src_x, src_y, dst_x, dst_y, cost, requirement, enabled, note, updated_by, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, now())
		ON CONFLICT (src_x, src_y) DO UPDATE SET
			dst_x = EXCLUDED.dst_x,
			dst_y = EXCLUDED.dst_y,
			cost = EXCLUDED.cost,
			requirement = EXCLUDED.requirement,
			enabled = EXCLUDED.enabled,
			note = EXCLUDED.note,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()`,
		r.SrcX, r.SrcY, r.DstX, r.DstY, r.Cost, r.Requirement, r.Enabled, r.Note, moderatorID)
	if err != nil {
		return fmt.Errorf("store: upsert teleport route %d,%d: %w", r.SrcX, r.SrcY, err)
	}
	return nil
}

// SetTeleportRouteEnabled toggles a route row. applied is false when there is no
// row at that origin (a compiled-only route has to be created before it can be
// disabled).
func (s *Store) SetTeleportRouteEnabled(ctx context.Context, srcX, srcY int, enabled bool, moderatorID int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE teleport_route
		SET enabled = $3, updated_by = $4, updated_at = now()
		WHERE src_x = $1 AND src_y = $2`, srcX, srcY, enabled, moderatorID)
	if err != nil {
		return false, fmt.Errorf("store: set teleport route %d,%d enabled: %w", srcX, srcY, err)
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteTeleportRoute drops the overlay row, restoring the compiled route (if
// any) at that origin. applied is false when there was no row.
func (s *Store) DeleteTeleportRoute(ctx context.Context, srcX, srcY int, moderatorID int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM teleport_route WHERE src_x = $1 AND src_y = $2`, srcX, srcY)
	if err != nil {
		return false, fmt.Errorf("store: delete teleport route %d,%d: %w", srcX, srcY, err)
	}
	return tag.RowsAffected() > 0, nil
}
