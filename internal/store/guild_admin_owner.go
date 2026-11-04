package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Ownership writes (issue 45, closing slice). "Dono do castelo" in this server is
// two columns, not a table of its own:
//
//   - guild_zone.charge_guild — the guild in charge of a kingdom zone (the world
//     reads it for city ownership and the siege state);
//   - guild_tower_state.owner_guild — the GTorre owner, a single row.
//
// Both are written here with the same transactional audit as the guild writes.
// The city tax is deliberately NOT part of SetZoneOwner: a siege result that also
// moved the shop tax would be a silent economy change.

// SetZoneOwner sets (or clears, guildID 0) the guild in charge of one kingdom
// zone. applied=false when the zone row does not exist.
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

// SetTowerOwner sets (or clears, guildID 0) the single GTorre owner row.
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

// guildOwnerTargetJSON is kept for symmetry with the other audit helpers when the
// target is a plain map; it guarantees we never insert invalid JSONB.
func guildOwnerTargetJSON(target map[string]any) (string, error) {
	raw, err := json.Marshal(target)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
