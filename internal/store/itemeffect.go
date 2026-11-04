package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ItemEffectOverride is one moderator-edited base effect of an item (issue 44,
// metade de escrita). The key is the ItemEffect.h code, which is how the server
// indexes an item's adds (internal/content.BaseEffects) — a row replaces that
// effect's value and leaves every other one from ItemList.csv untouched.
type ItemEffectOverride struct {
	ItemIndex int32
	Effect    int
	Value     int
	UpdatedBy int64
	UpdatedAt time.Time
}

// ListItemEffectOverrides returns the overlay rows, optionally restricted to one
// item (itemIndex <= 0 lists every row), ordered by item and effect.
func (s *Store) ListItemEffectOverrides(ctx context.Context, itemIndex int32) ([]ItemEffectOverride, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT item_index, effect, value, COALESCE(updated_by, 0), updated_at
		FROM item_effect_override
		WHERE ($1 <= 0 OR item_index = $1)
		ORDER BY item_index, effect`, itemIndex)
	if err != nil {
		return nil, fmt.Errorf("store: list item effect overrides: %w", err)
	}
	defer rows.Close()

	out := make([]ItemEffectOverride, 0, 4)
	for rows.Next() {
		var o ItemEffectOverride
		if err := rows.Scan(&o.ItemIndex, &o.Effect, &o.Value, &o.UpdatedBy, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan item effect override: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: read item effect overrides: %w", err)
	}
	return out, nil
}

// UpsertItemEffectOverride replaces the (item, effect) row and records
// before/after in item_effect_audit, both in one transaction: an edit applied
// without its audit row is exactly what the moderation log cannot tolerate. The
// `after` snapshot is read back from the table instead of marshalled from the
// input, so the audit shows what was actually stored.
func (s *Store) UpsertItemEffectOverride(ctx context.Context, o ItemEffectOverride, moderatorID int64, reason string) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := itemEffectJSON(ctx, tx, o.ItemIndex, o.Effect)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO item_effect_override (item_index, effect, value, updated_by, updated_at)
			VALUES ($1,$2,$3,$4, now())
			ON CONFLICT (item_index, effect) DO UPDATE SET
				value = EXCLUDED.value,
				updated_by = EXCLUDED.updated_by,
				updated_at = now()`,
			o.ItemIndex, o.Effect, o.Value, moderatorID); err != nil {
			return fmt.Errorf("store: upsert item effect %d/%d: %w", o.ItemIndex, o.Effect, err)
		}
		after, err := itemEffectJSON(ctx, tx, o.ItemIndex, o.Effect)
		if err != nil {
			return err
		}
		return auditItemEffect(ctx, tx, moderatorID, o.ItemIndex, o.Effect, "upsert", before, after, reason)
	})
}

// DeleteItemEffectOverride drops the overlay row, so the effect falls back to the
// content value. applied is false when there was no row (nothing to revert).
func (s *Store) DeleteItemEffectOverride(ctx context.Context, itemIndex int32, effect int, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := itemEffectJSON(ctx, tx, itemIndex, effect)
		if err != nil {
			return err
		}
		if before == nil {
			return nil
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM item_effect_override WHERE item_index = $1 AND effect = $2`, itemIndex, effect); err != nil {
			return fmt.Errorf("store: delete item effect %d/%d: %w", itemIndex, effect, err)
		}
		applied = true
		return auditItemEffect(ctx, tx, moderatorID, itemIndex, effect, "delete", before, nil, reason)
	})
	return applied, err
}

// itemEffectJSON renders the stored row as JSON for the audit columns; nil means
// the effect has no override. Reading it back through to_jsonb keeps the audit
// faithful to the table (including who touched it and when) instead of to
// whatever the caller believed it was writing.
func itemEffectJSON(ctx context.Context, tx pgx.Tx, itemIndex int32, effect int) ([]byte, error) {
	var js []byte
	err := tx.QueryRow(ctx, `
		SELECT to_jsonb(o) FROM item_effect_override o
		WHERE item_index = $1 AND effect = $2`, itemIndex, effect).Scan(&js)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: read item effect %d/%d: %w", itemIndex, effect, err)
	}
	return js, nil
}

// auditItemEffect appends one edit to item_effect_audit. before/after are nil when
// the row did not exist / no longer exists, which is how a moderator tells "mudei
// o efeito" from "removi o efeito".
func auditItemEffect(ctx context.Context, tx pgx.Tx, moderatorID int64, itemIndex int32, effect int, action string, before, after []byte, reason string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO item_effect_audit (moderator_id, item_index, effect, action, before, after, reason)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7)`,
		moderatorID, itemIndex, effect, action, jsonText(before), jsonText(after), reason); err != nil {
		return fmt.Errorf("store: audit item effect %d/%d: %w", itemIndex, effect, err)
	}
	return nil
}

// jsonText passes jsonb through the text protocol: pgx encodes []byte as bytea,
// which Postgres will not cast to jsonb implicitly.
func jsonText(raw []byte) any {
	if raw == nil {
		return nil
	}
	return string(raw)
}
