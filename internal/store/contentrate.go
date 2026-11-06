package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ContentRateOverride is one moderator-edited rate cell (issue 36): the value
// replaces the one in tmserver/content/Common/Settings for that (file, family, name).
// Name is textual because CompRate has several keys per family.
type ContentRateOverride struct {
	File      string
	Family    string
	Name      string
	Value     int64
	UpdatedBy int64
	UpdatedAt time.Time
}

// ListContentRateOverrides returns every overlay row, ordered by file/family/name.
func (s *Store) ListContentRateOverrides(ctx context.Context) ([]ContentRateOverride, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT file, family, name, value, COALESCE(updated_by, 0), updated_at
		  FROM content_rate_override
		 ORDER BY file, family, name`)
	if err != nil {
		return nil, fmt.Errorf("store: list content rate overrides: %w", err)
	}
	defer rows.Close()

	var out []ContentRateOverride
	for rows.Next() {
		var o ContentRateOverride
		if err := rows.Scan(&o.File, &o.Family, &o.Name, &o.Value, &o.UpdatedBy, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan content rate override: %w", err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpsertContentRateOverride stores one edited cell and its audit row in the same
// transaction. The after snapshot is read back from the table so the trail shows
// what was stored, not what the caller believed it was writing.
func (s *Store) UpsertContentRateOverride(ctx context.Context, o ContentRateOverride, moderatorID int64, reason string) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := contentRateJSON(ctx, tx, o.File, o.Family, o.Name)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO content_rate_override (file, family, name, value, updated_by, updated_at)
			VALUES ($1,$2,$3,$4,$5, now())
			ON CONFLICT (file, family, name) DO UPDATE SET
				value = EXCLUDED.value,
				updated_by = EXCLUDED.updated_by,
				updated_at = now()`,
			o.File, o.Family, o.Name, o.Value, moderatorID); err != nil {
			return fmt.Errorf("store: upsert content rate %s/%s/%s: %w", o.File, o.Family, o.Name, err)
		}
		after, err := contentRateJSON(ctx, tx, o.File, o.Family, o.Name)
		if err != nil {
			return err
		}
		return auditContentRate(ctx, tx, moderatorID, o.File, o.Family, o.Name, "upsert", before, after, reason)
	})
}

// DeleteContentRateOverride drops the cell, restoring the file value. applied is
// false when there was no row.
func (s *Store) DeleteContentRateOverride(ctx context.Context, file, family, name string, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := contentRateJSON(ctx, tx, file, family, name)
		if err != nil {
			return err
		}
		if before == nil {
			return nil
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM content_rate_override WHERE file = $1 AND family = $2 AND name = $3`,
			file, family, name); err != nil {
			return fmt.Errorf("store: delete content rate %s/%s/%s: %w", file, family, name, err)
		}
		applied = true
		return auditContentRate(ctx, tx, moderatorID, file, family, name, "reset", before, nil, reason)
	})
	return applied, err
}

func contentRateJSON(ctx context.Context, tx pgx.Tx, file, family, name string) ([]byte, error) {
	var js []byte
	err := tx.QueryRow(ctx, `
		SELECT to_jsonb(o) FROM content_rate_override o
		WHERE file = $1 AND family = $2 AND name = $3`, file, family, name).Scan(&js)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: read content rate %s/%s/%s: %w", file, family, name, err)
	}
	return js, nil
}

func auditContentRate(ctx context.Context, tx pgx.Tx, moderatorID int64, file, family, name, action string, before, after []byte, reason string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO content_rate_audit (moderator_id, file, family, name, action, before, after, reason)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8)`,
		moderatorID, file, family, name, action, jsonText(before), jsonText(after), reason); err != nil {
		return fmt.Errorf("store: audit content rate %s/%s/%s: %w", file, family, name, err)
	}
	return nil
}
