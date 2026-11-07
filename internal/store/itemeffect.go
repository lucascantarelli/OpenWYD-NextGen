package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ItemEffectOverride é um efeito base de item editado por moderador (issue 44,
// metade de escrita). A chave é o código de ItemEffect.h, que é como o servidor
// indexa os adds de um item (internal/content.BaseEffects) — uma linha substitui o
// valor daquele efeito e deixa todos os outros do ItemList.csv intactos.
type ItemEffectOverride struct {
	ItemIndex int32
	Effect    int
	Value     int
	UpdatedBy int64
	UpdatedAt time.Time
}

// ListItemEffectOverrides devolve as linhas de sobreposição, opcionalmente
// restritas a um item (itemIndex <= 0 lista todas as linhas), ordenadas por item e efeito.
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

// UpsertItemEffectOverride substitui a linha (item, effect) e registra
// antes/depois em item_effect_audit, ambos numa transação: uma edição aplicada
// sem sua linha de auditoria é exatamente o que o log de moderação não tolera. O
// snapshot posterior é relido da tabela em vez de montado a partir da
// entrada, então a auditoria mostra o que foi realmente guardado.
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

// DeleteItemEffectOverride descarta a linha de sobreposição, então o efeito volta ao
// valor de conteúdo. applied é falso quando não havia linha (nada a reverter).
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

// itemEffectJSON renderiza a linha guardada como JSON para as colunas de auditoria;
// nil significa que o efeito não tem sobreposição. Relê-la via to_jsonb mantém a
// auditoria fiel à tabela (incluindo quem a tocou e quando) em vez de ao
// que quem chamou acreditava estar escrevendo.
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

// auditItemEffect anexa uma edição a item_effect_audit. Os snapshots de antes e de depois são nil quando
// a linha não existia / não existe mais, que é como um moderador distingue "mudei
// o efeito" de "removi o efeito".
func auditItemEffect(ctx context.Context, tx pgx.Tx, moderatorID int64, itemIndex int32, effect int, action string, before, after []byte, reason string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO item_effect_audit (moderator_id, item_index, effect, action, before, after, reason)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7)`,
		moderatorID, itemIndex, effect, action, jsonText(before), jsonText(after), reason); err != nil {
		return fmt.Errorf("store: audit item effect %d/%d: %w", itemIndex, effect, err)
	}
	return nil
}

// jsonText passa jsonb pelo protocolo de texto: o pgx codifica []byte como bytea,
// que o Postgres não converte para jsonb implicitamente.
func jsonText(raw []byte) any {
	if raw == nil {
		return nil
	}
	return string(raw)
}
