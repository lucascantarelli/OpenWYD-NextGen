package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// DropRateOverride is one moderator-edited Carry slot drop divisor (issue 34).
// The slot is the key: it is the Carry position that decides rarity, not the
// item, and the value is a divisor (1 = always drops, 20000 = ultra rare).
type DropRateOverride struct {
	Slot      int
	Value     int64
	UpdatedBy int64
	UpdatedAt time.Time
}

// ListDropRateOverrides returns every overlay row, ordered by slot.
func (s *Store) ListDropRateOverrides(ctx context.Context) ([]DropRateOverride, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT slot, value, COALESCE(updated_by, 0), updated_at
		  FROM drop_rate_override
		 ORDER BY slot`)
	if err != nil {
		return nil, fmt.Errorf("store: list drop rate overrides: %w", err)
	}
	defer rows.Close()

	var out []DropRateOverride
	for rows.Next() {
		var o DropRateOverride
		if err := rows.Scan(&o.Slot, &o.Value, &o.UpdatedBy, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan drop rate override: %w", err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpsertDropRateOverride stores one slot and its audit row in the same
// transaction. The after snapshot is read back from the table, so the trail
// shows what was stored rather than what the caller believed it was writing.
func (s *Store) UpsertDropRateOverride(ctx context.Context, slot int, value int64, moderatorID int64, reason string) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := dropRateJSON(ctx, tx, slot)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO drop_rate_override (slot, value, updated_by, updated_at)
			VALUES ($1,$2,$3, now())
			ON CONFLICT (slot) DO UPDATE SET
				value = EXCLUDED.value,
				updated_by = EXCLUDED.updated_by,
				updated_at = now()`,
			slot, value, moderatorID); err != nil {
			return fmt.Errorf("store: upsert drop rate slot %d: %w", slot, err)
		}
		after, err := dropRateJSON(ctx, tx, slot)
		if err != nil {
			return err
		}
		return auditDropRate(ctx, tx, moderatorID, slot, "upsert", before, after, reason)
	})
}

// DeleteDropRateOverride drops the slot, restoring the code value. applied is
// false when there was no row.
func (s *Store) DeleteDropRateOverride(ctx context.Context, slot int, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := dropRateJSON(ctx, tx, slot)
		if err != nil {
			return err
		}
		if before == nil {
			return nil
		}
		if _, err := tx.Exec(ctx, `DELETE FROM drop_rate_override WHERE slot = $1`, slot); err != nil {
			return fmt.Errorf("store: delete drop rate slot %d: %w", slot, err)
		}
		applied = true
		return auditDropRate(ctx, tx, moderatorID, slot, "reset", before, nil, reason)
	})
	return applied, err
}

func dropRateJSON(ctx context.Context, tx pgx.Tx, slot int) ([]byte, error) {
	var js []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(o) FROM drop_rate_override o WHERE slot = $1`, slot).Scan(&js)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: read drop rate slot %d: %w", slot, err)
	}
	return js, nil
}

func auditDropRate(ctx context.Context, tx pgx.Tx, moderatorID int64, slot int, action string, before, after []byte, reason string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO drop_rate_audit (moderator_id, slot, action, before, after, reason)
		VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6)`,
		moderatorID, slot, action, jsonText(before), jsonText(after), reason); err != nil {
		return fmt.Errorf("store: audit drop rate slot %d: %w", slot, err)
	}
	return nil
}
