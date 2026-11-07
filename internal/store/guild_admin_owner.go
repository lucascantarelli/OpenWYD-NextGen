package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Escritas de posse (issue 45, fatia de fechamento). "Dono do castelo" neste
// servidor são duas colunas, não uma tabela própria:
//
//   - guild_zone.charge_guild — a guild responsável por uma zona do reino (o
//     mundo lê essa coluna para dono de cidade e estado de cerco);
//   - guild_tower_state.owner_guild — o dono da GTorre, uma linha só.
//
// As duas são gravadas aqui com a mesma auditoria transacional das escritas de
// guild. O imposto da cidade de propósito NÃO faz parte de SetZoneOwner: um
// resultado de cerco que também mexesse no imposto da loja seria uma mudança
// silenciosa de economia.

// SetZoneOwner define (ou limpa, com guildID 0) a guild responsável por uma zona
// do reino. applied=falso quando a linha da zona não existe.
func (s *Store) SetZoneOwner(ctx context.Context, zone, guildID int32, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var previous int32
		var before []byte
		err := tx.QueryRow(ctx,
			`SELECT charge_guild, to_jsonb(z) FROM guild_zone z WHERE zone = $1 FOR UPDATE`,
			zone).Scan(&previous, &before)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("store: lock guild zone %d: %w", zone, err)
		}
		if previous == guildID {
			return nil
		}

		if _, err := tx.Exec(ctx,
			`UPDATE guild_zone SET charge_guild = $2, updated_at = now() WHERE zone = $1`,
			zone, guildID); err != nil {
			return fmt.Errorf("store: set zone %d owner %d: %w", zone, guildID, err)
		}

		var after []byte
		if err := tx.QueryRow(ctx,
			`SELECT to_jsonb(z) FROM guild_zone z WHERE zone = $1`, zone).Scan(&after); err != nil {
			return fmt.Errorf("store: re-read guild zone %d: %w", zone, err)
		}

		applied = true
		return auditGuild(ctx, tx, moderatorID, guildID, "zone_owner",
			map[string]any{"zone": zone, "previous_guild": previous}, before, after, reason)
	})
	return applied, err
}

// SetTowerOwner define (ou limpa, com guildID 0) a única linha de dono da GTorre.
func (s *Store) SetTowerOwner(ctx context.Context, guildID int32, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var previous int32
		var before []byte
		err := tx.QueryRow(ctx,
			`SELECT owner_guild, to_jsonb(t) FROM guild_tower_state t WHERE id = 1 FOR UPDATE`).
			Scan(&previous, &before)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("store: lock guild tower state: %w", err)
		}
		if previous == guildID {
			return nil
		}

		if _, err := tx.Exec(ctx,
			`UPDATE guild_tower_state SET owner_guild = $1, updated_at_unix = EXTRACT(EPOCH FROM now())::bigint WHERE id = 1`,
			guildID); err != nil {
			return fmt.Errorf("store: set tower owner %d: %w", guildID, err)
		}

		var after []byte
		if err := tx.QueryRow(ctx,
			`SELECT to_jsonb(t) FROM guild_tower_state t WHERE id = 1`).Scan(&after); err != nil {
			return fmt.Errorf("store: re-read guild tower state: %w", err)
		}

		applied = true
		return auditGuild(ctx, tx, moderatorID, guildID, "tower_owner",
			map[string]any{"previous_guild": previous}, before, after, reason)
	})
	return applied, err
}
