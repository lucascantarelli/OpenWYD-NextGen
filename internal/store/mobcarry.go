package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// Carry[] persistence (issue 34, fatia 3). The slot is the drop-rarity key and
// the table is authoritative per template: SetMobCarry replaces every slot of
// the template, DeleteMobCarry returns it to the file. The tmServer reads this
// snapshot once at boot; nothing here touches live game state.

// ListMobCarryOverrides returns every template's carry override, ordered by
// template_name and slot — the snapshot the dbServer serves to the tmServer.
func (s *Store) ListMobCarryOverrides(ctx context.Context) ([]domain.MobCarryOverride, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT template_name, slot, item_index, eff1, effv1, eff2, effv2, eff3, effv3
		  FROM mob_carry_override
		 ORDER BY template_name, slot`)
	if err != nil {
		return nil, fmt.Errorf("store: list mob carry overrides: %w", err)
	}
	defer rows.Close()

	byName := make(map[string]int, 16)
	var out []domain.MobCarryOverride
	for rows.Next() {
		var name string
		var it domain.MobTemplateEquipItem
		if err := rows.Scan(&name, &it.Slot, &it.ItemIndex,
			&it.Eff1, &it.EffV1, &it.Eff2, &it.EffV2, &it.Eff3, &it.EffV3); err != nil {
			return nil, fmt.Errorf("store: scan mob carry override: %w", err)
		}
		idx, ok := byName[name]
		if !ok {
			idx = len(out)
			byName[name] = idx
			out = append(out, domain.MobCarryOverride{TemplateName: name})
		}
		out[idx].Slots = append(out[idx].Slots, it)
	}
	return out, rows.Err()
}

// GetMobCarryOverride loads one template's override. ErrNotFound when absent.
func (s *Store) GetMobCarryOverride(ctx context.Context, templateName string) (domain.MobCarryOverride, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT slot, item_index, eff1, effv1, eff2, effv2, eff3, effv3
		  FROM mob_carry_override WHERE template_name = $1 ORDER BY slot`, templateName)
	if err != nil {
		return domain.MobCarryOverride{}, fmt.Errorf("store: get mob carry %q: %w", templateName, err)
	}
	defer rows.Close()

	out := domain.MobCarryOverride{TemplateName: templateName}
	for rows.Next() {
		var it domain.MobTemplateEquipItem
		if err := rows.Scan(&it.Slot, &it.ItemIndex,
			&it.Eff1, &it.EffV1, &it.Eff2, &it.EffV2, &it.Eff3, &it.EffV3); err != nil {
			return domain.MobCarryOverride{}, fmt.Errorf("store: scan mob carry %q: %w", templateName, err)
		}
		out.Slots = append(out.Slots, it)
	}
	if err := rows.Err(); err != nil {
		return domain.MobCarryOverride{}, err
	}
	if len(out.Slots) == 0 {
		return domain.MobCarryOverride{}, ErrNotFound
	}
	return out, nil
}

// SetMobCarry replaces the template's whole Carry[] override and writes one
// audit row, all in one transaction. Unlike the Equip[] writer this does not
// require a stat override: carry is independent of the stat surface.
func (s *Store) SetMobCarry(ctx context.Context, templateName string, slots []domain.MobTemplateEquipItem, moderatorID int64, reason string) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := carryJSON(ctx, tx, templateName)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM mob_carry_override WHERE template_name = $1`, templateName); err != nil {
			return fmt.Errorf("store: clear mob carry %q: %w", templateName, err)
		}
		for _, it := range slots {
			if _, err := tx.Exec(ctx, `
				INSERT INTO mob_carry_override
					(template_name, slot, item_index, eff1, effv1, eff2, effv2, eff3, effv3, updated_by, updated_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now())`,
				templateName, it.Slot, it.ItemIndex, it.Eff1, it.EffV1, it.Eff2, it.EffV2, it.Eff3, it.EffV3,
				nullableID(moderatorID),
			); err != nil {
				return fmt.Errorf("store: insert mob carry (%q slot %d): %w", templateName, it.Slot, err)
			}
		}
		after, err := carryJSON(ctx, tx, templateName)
		if err != nil {
			return err
		}
		return auditMobCarry(ctx, tx, moderatorID, templateName, "set", before, after, reason)
	})
}

// DeleteMobCarry removes the template's override, restoring the file's Carry[].
// applied is false when the template had no override.
func (s *Store) DeleteMobCarry(ctx context.Context, templateName string, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := carryJSON(ctx, tx, templateName)
		if err != nil {
			return err
		}
		if before == nil {
			return nil
		}
		if _, err := tx.Exec(ctx, `DELETE FROM mob_carry_override WHERE template_name = $1`, templateName); err != nil {
			return fmt.Errorf("store: delete mob carry %q: %w", templateName, err)
		}
		applied = true
		return auditMobCarry(ctx, tx, moderatorID, templateName, "reset", before, nil, reason)
	})
	return applied, err
}

// carryJSON snapshots the template's rows for the audit trail; nil means no
// override existed.
func carryJSON(ctx context.Context, tx pgx.Tx, templateName string) ([]byte, error) {
	var js []byte
	err := tx.QueryRow(ctx,
		`SELECT jsonb_agg(to_jsonb(c) ORDER BY c.slot) FROM mob_carry_override c WHERE c.template_name = $1`,
		templateName).Scan(&js)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: read mob carry %q: %w", templateName, err)
	}
	return js, nil
}

func auditMobCarry(ctx context.Context, tx pgx.Tx, moderatorID int64, templateName, action string, before, after []byte, reason string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO mob_carry_audit (moderator_id, template_name, action, before, after, reason)
		VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6)`,
		moderatorID, templateName, action, jsonText(before), jsonText(after), reason); err != nil {
		return fmt.Errorf("store: audit mob carry %q: %w", templateName, err)
	}
	return nil
}
