package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// Persistência de Carry[] (issue 34, fatia 3). O slot é a chave de raridade de drop
// e a tabela é autorotativa por template: SetMobCarry substitui todo slot do
// template, DeleteMobCarry o devolve ao arquivo. O tmServer lê isto
// com um snapshot uma vez no boot; nada aqui toca estado de jogo vivo.

// ListMobCarryOverrides devolve a sobreposição de carry de todo template, ordenada
// por template_name e slot — o snapshot que o dbServer serve ao tmServer.
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

// GetMobCarryOverride carrega a sobreposição de um template. ErrNotFound quando ausente.
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

// SetMobCarry substitui a sobreposição Carry[] inteira do template e escreve uma
// linha de auditoria, tudo numa transação. Diferente do escritor de Equip[] isto não
// exige sobreposição de stats: o carry é independente da superfície de stats.
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

// DeleteMobCarry remove a sobreposição do template, restaurando o Carry[] do arquivo.
// applied é falso quando o template não tinha sobreposição.
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

// carryJSON fotografa as linhas do template para a trilha de auditoria; nil
// significa que não
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
