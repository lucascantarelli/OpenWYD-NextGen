package store

import (
	"context"
	"fmt"
)

// GuildMemberRow is one guild_member row as the admin surface reads it: the
// character reference plus the level/slot the world will load on boot.
type GuildMemberRow struct {
	GuildID     int32
	CharacterID int64
	AccountID   int64
	Slot        int16
	Name        string
	GuildLevel  int16
}

// GuildRoster returns every guild membership in one query, ordered so the leader
// (level 9) comes first inside each guild. The table is small (a guild is 20ish
// rows at most), so the admin surface groups it in Go instead of issuing one
// query per guild.
func (s *Store) GuildRoster(ctx context.Context) ([]GuildMemberRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT guild_id, character_id, account_id, slot, name, guild_level
		  FROM guild_member
		 ORDER BY guild_id, guild_level DESC, name`)
	if err != nil {
		return nil, fmt.Errorf("store: guild roster: %w", err)
	}
	defer rows.Close()

	var out []GuildMemberRow
	for rows.Next() {
		var m GuildMemberRow
		if err := rows.Scan(&m.GuildID, &m.CharacterID, &m.AccountID, &m.Slot, &m.Name, &m.GuildLevel); err != nil {
			return nil, fmt.Errorf("store: scan guild roster: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
