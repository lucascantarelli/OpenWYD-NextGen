package store

import (
	"context"
	"fmt"

	"openwyd-nextgen/internal/domain"
)

// ListExpRanking devolve a projeção persistida de Top EXP de personagem em ordem
// de ranking. Entradas com level >= 1000 são ignoradas, igual à carga do CRanking
// legado.
func (s *Store) ListExpRanking(ctx context.Context, limit, offset int) ([]domain.RankingEntry, int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT name, class, clan, guild_id, level, exp, class_master, count(*) OVER()
		  FROM characters
		 WHERE level < 1000
		 ORDER BY
		       CASE class_master WHEN 2 THEN 1 WHEN 1 THEN 2 ELSE class_master END DESC,
		       exp DESC,
		       level DESC,
		       name ASC
		 LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list exp ranking: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.RankingEntry, 0, limit)
	total := 0
	for rows.Next() {
		var e domain.RankingEntry
		if err := rows.Scan(&e.Name, &e.Class, &e.Clan, &e.GuildID, &e.Level, &e.Exp, &e.ClassMaster, &total); err != nil {
			return nil, 0, fmt.Errorf("store: scan exp ranking: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: list exp ranking: %w", err)
	}
	total, err = fallbackTotal(ctx, len(entries), offset, total, s.countExpRanking)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func (s *Store) countExpRanking(ctx context.Context) (int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM characters WHERE level < 1000`).Scan(&total); err != nil {
		return 0, fmt.Errorf("store: count exp ranking: %w", err)
	}
	return total, nil
}

// fallbackTotal cobre o ponto cego de uma window function `count(*) OVER()`: ela
// só aparece em linhas devolvidas, então uma página além da última linha (resultado
// vazio, offset não zero) informaria total=0. Só nesse caso, rode um
// COUNT(*) simples em vez disso.
func fallbackTotal(ctx context.Context, rowsReturned, offset, windowTotal int, count func(context.Context) (int, error)) (int, error) {
	if rowsReturned != 0 || offset == 0 {
		return windowTotal, nil
	}
	return count(ctx)
}
