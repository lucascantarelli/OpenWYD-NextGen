package store

import (
	"context"
	"fmt"
)

// SetKefraDefeated flips the Kefra state from the admin surface (issue 45).
//
// The revision is bumped in the same statement: SaveKefraState refuses a write
// whose revision is not greater than the stored one, so the running world's next
// save (which uses its own, lower in-memory revision) cannot silently undo this
// edit. What it cannot do is change the live fight: Kefra lives in the world
// goroutine, so the flag takes effect on the next boot. applied=false means the
// row does not exist yet (tmServer never saved a spawn).
func (s *Store) SetKefraDefeated(ctx context.Context, defeated bool) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE kefra_state SET defeated = $1, revision = revision + 1 WHERE id = 1`,
		defeated,
	)
	if err != nil {
		return false, fmt.Errorf("store: set kefra defeated: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
