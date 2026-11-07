package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// WorldEventConfigVersion devolve a versão monotônica da config de moderador
// de eventos. Atualizações de progresso do evento não sobem este valor.
func (s *Store) WorldEventConfigVersion(ctx context.Context) (int64, error) {
	var v int64
	err := s.pool.QueryRow(ctx, `SELECT version FROM world_event_meta WHERE id = TRUE`).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("store: world event config version: %w", err)
	}
	return v, nil
}

// WorldEventConfig devolve a única linha global de configuração de evento de mundo.
func (s *Store) WorldEventConfig(ctx context.Context) (domain.WorldEventConfig, error) {
	cfg, err := scanWorldEventConfig(s.pool.QueryRow(ctx, worldEventConfigSelect))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.WorldEventConfig{NoticeEnabled: true}, nil
	}
	if err != nil {
		return domain.WorldEventConfig{}, fmt.Errorf("store: world event config: %w", err)
	}
	return cfg, nil
}

// UpsertWorldEventConfig substitui a config global de evento gerida pelo moderador,
// gravando auditoria e subindo a versão da config na mesma transação.
func (s *Store) UpsertWorldEventConfig(ctx context.Context, cfg domain.WorldEventConfig, moderatorID int64) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		if err := lockWorldEventMeta(ctx, tx); err != nil {
			return err
		}
		before, _ := fetchWorldEventConfigJSON(ctx, tx)
		if _, err := tx.Exec(ctx, `
			INSERT INTO world_event_config (
				id, enabled, item_index, rate, start_index, current_index, end_index,
				indexed, notice_enabled, double_exp_enabled, newbie_event_enabled,
				updated_by, updated_at
			)
			VALUES (TRUE,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now())
			ON CONFLICT (id) DO UPDATE SET
				enabled              = EXCLUDED.enabled,
				item_index           = EXCLUDED.item_index,
				rate                 = EXCLUDED.rate,
				start_index          = EXCLUDED.start_index,
				current_index        = EXCLUDED.current_index,
				end_index            = EXCLUDED.end_index,
				indexed              = EXCLUDED.indexed,
				notice_enabled       = EXCLUDED.notice_enabled,
				double_exp_enabled   = EXCLUDED.double_exp_enabled,
				newbie_event_enabled = EXCLUDED.newbie_event_enabled,
				updated_by           = EXCLUDED.updated_by,
				updated_at           = now()`,
			cfg.Enabled, cfg.ItemIndex, cfg.Rate, cfg.StartIndex, cfg.CurrentIndex,
			cfg.EndIndex, cfg.Indexed, cfg.NoticeEnabled, cfg.DoubleExpEnabled,
			cfg.NewbieEventEnabled, nullableID(moderatorID)); err != nil {
			return fmt.Errorf("store: upsert world event config: %w", err)
		}
		after, _ := fetchWorldEventConfigJSON(ctx, tx)
		return auditWorldEventAndBump(ctx, tx, moderatorID, "set_config", before, after)
	})
}

// UpdateWorldEventProgress persiste o contador de evento do tmServer sem subir a
// versão de config do moderador. expectedVersion impede que um snapshot velho do tmServer
// sobrescreva uma edição mais nova do portal; applied é falso quando a versão mudou.
func (s *Store) UpdateWorldEventProgress(ctx context.Context, expectedVersion int64, currentIndex int32) (applied bool, err error) {
	err = s.inTx(ctx, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `
			WITH matching_meta AS MATERIALIZED (
				SELECT 1
				  FROM world_event_meta
				 WHERE id = TRUE AND version = $1
				   FOR UPDATE
			)
			UPDATE world_event_config
			   SET current_index = GREATEST(current_index, $2)
			 WHERE id = TRUE
			   AND EXISTS (SELECT 1 FROM matching_meta)`, expectedVersion, currentIndex)
		if err != nil {
			return fmt.Errorf("store: update world event progress: %w", err)
		}
		applied = ct.RowsAffected() == 1
		return nil
	})
	return applied, err
}

const worldEventConfigSelect = `
	SELECT enabled, item_index, rate, start_index, current_index, end_index,
	       indexed, notice_enabled, double_exp_enabled, newbie_event_enabled
	FROM world_event_config WHERE id = TRUE`

type worldEventScanRow interface {
	Scan(dest ...any) error
}

func scanWorldEventConfig(row worldEventScanRow) (domain.WorldEventConfig, error) {
	var cfg domain.WorldEventConfig
	err := row.Scan(&cfg.Enabled, &cfg.ItemIndex, &cfg.Rate, &cfg.StartIndex,
		&cfg.CurrentIndex, &cfg.EndIndex, &cfg.Indexed, &cfg.NoticeEnabled,
		&cfg.DoubleExpEnabled, &cfg.NewbieEventEnabled)
	return cfg, err
}

func fetchWorldEventConfigJSON(ctx context.Context, tx pgx.Tx) ([]byte, error) {
	var js []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(c) FROM world_event_config c WHERE id = TRUE`).Scan(&js)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return js, err
}

func lockWorldEventMeta(ctx context.Context, tx pgx.Tx) error {
	// Edições do portal e progresso do tmServer tomam este lock antes de tocar na
	// linha de config, então expectedVersion não pode passar enquanto uma edição de
	// moderador está pendente.
	var version int64
	if err := tx.QueryRow(ctx, `SELECT version FROM world_event_meta WHERE id = TRUE FOR UPDATE`).Scan(&version); err != nil {
		return fmt.Errorf("store: lock world event meta: %w", err)
	}
	return nil
}

func auditWorldEventAndBump(ctx context.Context, tx pgx.Tx, moderatorID int64, action string, before, after []byte) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO world_event_audit (account_id, action, before, after)
		VALUES ($1,$2,$3,$4)`,
		moderatorID, action, nullableJSON(before), nullableJSON(after)); err != nil {
		return fmt.Errorf("store: write world event audit: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE world_event_meta SET version = version + 1 WHERE id = TRUE`); err != nil {
		return fmt.Errorf("store: bump world event config version: %w", err)
	}
	return nil
}
