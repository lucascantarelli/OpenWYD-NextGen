package store

import (
	"context"
	"fmt"
)

// GuildMemberRow é uma linha de guild_member como a superfície de admin a lê: a
// referência do personagem mais o nível/slot que o mundo carregará no boot.
type GuildMemberRow struct {
	GuildID     int32
	CharacterID int64
	AccountID   int64
	Slot        int16
	Name        string
	GuildLevel  int16
}

// GuildRoster devolve toda associação de guild numa consulta, ordenada para o
// líder (nível 9) vir primeiro dentro de cada guild. A tabela é pequena (uma
// guild tem no máximo umas 20 linhas), então a superfície de admin a agrupa em
// Go em vez de emitir uma
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
