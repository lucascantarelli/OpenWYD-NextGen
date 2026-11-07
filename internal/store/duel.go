package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// RecordDuelResult incrementa as vitórias do vencedor e as derrotas do perdedor em
// character_pvp_stats, fazendo upsert de uma linha se o personagem nunca duelou
// antes (issue #118). Com chave pelo nome do personagem (já UNIQUE), então o
// tmServer nunca precisa carregar um character_id do banco por Session/Entity.
// Os dois incrementos acontecem numa transação; se qualquer nome de personagem
// desconhecido, o resultado inteiro é descartado (ErrNotFound) em vez de
// registrar um duelo desequilibrado.
func (s *Store) RecordDuelResult(ctx context.Context, winnerName, loserName string) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		if err := bumpDuelWins(ctx, tx, winnerName); err != nil {
			return err
		}
		return bumpDuelLosses(ctx, tx, loserName)
	})
}

func bumpDuelWins(ctx context.Context, tx pgx.Tx, name string) error {
	tag, err := tx.Exec(ctx, `
		INSERT INTO character_pvp_stats (character_id, wins, updated_at)
		SELECT id, 1, now() FROM characters WHERE name = $1
		ON CONFLICT (character_id) DO UPDATE
		SET wins = character_pvp_stats.wins + 1, updated_at = now()`,
		name)
	if err != nil {
		return fmt.Errorf("store: record duel result %s (wins): %w", name, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store: record duel result: %w: character %q", ErrNotFound, name)
	}
	return nil
}

func bumpDuelLosses(ctx context.Context, tx pgx.Tx, name string) error {
	tag, err := tx.Exec(ctx, `
		INSERT INTO character_pvp_stats (character_id, losses, updated_at)
		SELECT id, 1, now() FROM characters WHERE name = $1
		ON CONFLICT (character_id) DO UPDATE
		SET losses = character_pvp_stats.losses + 1, updated_at = now()`,
		name)
	if err != nil {
		return fmt.Errorf("store: record duel result %s (losses): %w", name, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store: record duel result: %w: character %q", ErrNotFound, name)
	}
	return nil
}

// ListDuelRanking devolve o placar persistido de vitórias/derrotas de duelo em
// ordem de ranking
func (s *Store) ListDuelRanking(ctx context.Context, limit, offset int) ([]domain.DuelRankingEntry, int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.name, c.class, c.clan, c.guild_id, p.wins, p.losses, count(*) OVER()
		  FROM character_pvp_stats p
		  JOIN characters c ON c.id = p.character_id
		 ORDER BY p.wins DESC, p.losses ASC, c.name ASC
		 LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list duel ranking: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.DuelRankingEntry, 0, limit)
	total := 0
	for rows.Next() {
		var e domain.DuelRankingEntry
		if err := rows.Scan(&e.Name, &e.Class, &e.Clan, &e.GuildID, &e.Wins, &e.Losses, &total); err != nil {
			return nil, 0, fmt.Errorf("store: scan duel ranking: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: list duel ranking: %w", err)
	}
	total, err = fallbackTotal(ctx, len(entries), offset, total, s.countDuelRanking)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func (s *Store) countDuelRanking(ctx context.Context) (int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM character_pvp_stats`).Scan(&total); err != nil {
		return 0, fmt.Errorf("store: count duel ranking: %w", err)
	}
	return total, nil
}
