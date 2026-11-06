package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Guild admin writes (issue 45, slice 2). Every one of them runs in a single
// transaction with its guild_admin_audit row: a leadership transfer that commits
// without the audit trail is precisely what a moderation log cannot allow, and a
// half-applied transfer (new master promoted, old one still level 9) would make
// the guild unusable in the world.
//
// applied=false means "nothing to do" (character is not a member, already master,
// relation absent); it is not an error.

// SetGuildMasterByCharacter promotes one member to level 9 and demotes the
// previous leader to level 0, mirroring both in `character` so the next boot and
// the save path agree with guild_member.
func (s *Store) SetGuildMasterByCharacter(ctx context.Context, guildID int32, characterID int64, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var level int
		err := tx.QueryRow(ctx,
			`SELECT guild_level FROM guild_member WHERE guild_id = $1 AND character_id = $2 FOR UPDATE`,
			guildID, characterID).Scan(&level)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("store: lock guild member %d/%d: %w", guildID, characterID, err)
		}
		if level == 9 {
			return nil
		}

		before, err := guildMemberSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx,
			`UPDATE guild_member SET guild_level = 0, updated_at = now() WHERE guild_id = $1 AND guild_level = 9`,
			guildID); err != nil {
			return fmt.Errorf("store: demote guild %d master: %w", guildID, err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE guild_member SET guild_level = 9, updated_at = now() WHERE guild_id = $1 AND character_id = $2`,
			guildID, characterID); err != nil {
			return fmt.Errorf("store: promote guild %d member %d: %w", guildID, characterID, err)
		}
		// Mirror in `character`: the login/save path reads guild_level from there.
		if _, err := tx.Exec(ctx,
			`UPDATE characters SET guild_level = 0 WHERE guild_id = $1 AND guild_level = 9`, guildID); err != nil {
			return fmt.Errorf("store: demote character guild level for %d: %w", guildID, err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE characters SET guild_level = 9 WHERE id = $1 AND guild_id = $2`,
			characterID, guildID); err != nil {
			return fmt.Errorf("store: promote character %d guild level: %w", characterID, err)
		}

		after, err := guildMemberSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}
		applied = true
		return auditGuild(ctx, tx, moderatorID, guildID, "set_master",
			map[string]any{"character_id": characterID}, before, after, reason)
	})
	return applied, err
}

// RemoveGuildMemberByCharacter drops one membership and clears the character's
// guild fields. The leader is refused one layer up: removing the level-9 row
// would leave a guild with no one able to hand it over.
func (s *Store) RemoveGuildMemberByCharacter(ctx context.Context, guildID int32, characterID int64, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var level int
		err := tx.QueryRow(ctx,
			`SELECT guild_level FROM guild_member WHERE guild_id = $1 AND character_id = $2 FOR UPDATE`,
			guildID, characterID).Scan(&level)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("store: lock guild member %d/%d: %w", guildID, characterID, err)
		}

		before, err := guildMemberSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM guild_member WHERE guild_id = $1 AND character_id = $2`,
			guildID, characterID); err != nil {
			return fmt.Errorf("store: delete guild member %d/%d: %w", guildID, characterID, err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE characters SET guild_id = 0, guild_level = 0 WHERE id = $1 AND guild_id = $2`,
			characterID, guildID); err != nil {
			return fmt.Errorf("store: clear character %d guild: %w", characterID, err)
		}

		after, err := guildMemberSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}
		applied = true
		return auditGuild(ctx, tx, moderatorID, guildID, "remove_member",
			map[string]any{"character_id": characterID, "guild_level": level}, before, after, reason)
	})
	return applied, err
}

// SetGuildRelationAudited upserts one directed ally/war relation or removes it
// (kind 0). The pair is validated one layer up; the table's own CHECKs (kind 1/2,
// guild_id <> target_guild_id) remain the last line of defence.
func (s *Store) SetGuildRelationAudited(ctx context.Context, guildID, targetGuildID int32, kind int, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := guildRelationSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}
		var errExec error
		if kind == 0 {
			tag, err := tx.Exec(ctx,
				`DELETE FROM guild_relation WHERE guild_id = $1 AND target_guild_id = $2`,
				guildID, targetGuildID)
			if err != nil {
				return fmt.Errorf("store: delete guild relation %d->%d: %w", guildID, targetGuildID, err)
			}
			applied = tag.RowsAffected() > 0
			errExec = nil
		} else {
			if _, err := tx.Exec(ctx, `
				INSERT INTO guild_relation(guild_id, target_guild_id, kind, updated_at)
				VALUES ($1, $2, $3, now())
				ON CONFLICT (guild_id, kind)
				DO UPDATE SET target_guild_id = EXCLUDED.target_guild_id, updated_at = now()`,
				guildID, targetGuildID, kind); err != nil {
				return fmt.Errorf("store: set guild relation %d->%d: %w", guildID, targetGuildID, err)
			}
			applied = true
		}
		if errExec != nil {
			return errExec
		}
		after, err := guildRelationSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}
		if !applied {
			return nil
		}
		return auditGuild(ctx, tx, moderatorID, guildID, "relation",
			map[string]any{"target_guild_id": targetGuildID, "kind": kind}, before, after, reason)
	})
	return applied, err
}

// guildMemberSnapshot is the audit's before/after: the guild's membership as the
// table has it, read back inside the transaction.
func guildMemberSnapshot(ctx context.Context, tx pgx.Tx, guildID int32) ([]byte, error) {
	var js []byte
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(jsonb_agg(m ORDER BY m.guild_level DESC, m.name), '[]'::jsonb)
		  FROM guild_member m WHERE m.guild_id = $1`, guildID).Scan(&js); err != nil {
		return nil, fmt.Errorf("store: snapshot guild %d members: %w", guildID, err)
	}
	return js, nil
}

// guildRelationSnapshot is the same idea for the directed relations of a guild.
func guildRelationSnapshot(ctx context.Context, tx pgx.Tx, guildID int32) ([]byte, error) {
	var js []byte
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(jsonb_agg(r ORDER BY r.kind), '[]'::jsonb)
		  FROM guild_relation r WHERE r.guild_id = $1`, guildID).Scan(&js); err != nil {
		return nil, fmt.Errorf("store: snapshot guild %d relations: %w", guildID, err)
	}
	return js, nil
}

// auditGuild appends one row to guild_admin_audit. target/before/after are JSONB;
// before/after come from the table, so the trail shows the state that was really
// stored rather than the request that was sent.
func auditGuild(ctx context.Context, tx pgx.Tx, moderatorID int64, guildID int32, action string, target map[string]any, before, after []byte, reason string) error {
	rawTarget, err := json.Marshal(target)
	if err != nil {
		return fmt.Errorf("store: marshal guild audit target: %w", err)
	}
	targetJSON := string(rawTarget)
	if _, err := tx.Exec(ctx, `
		INSERT INTO guild_admin_audit (moderator_id, guild_id, action, target, before, after, reason)
		VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6::jsonb,$7)`,
		moderatorID, guildID, action, targetJSON, jsonText(before), jsonText(after), reason); err != nil {
		return fmt.Errorf("store: audit guild %d %s: %w", guildID, action, err)
	}
	return nil
}
