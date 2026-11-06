package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// CharacterSearchHit is one row of the moderator character search (issue 43):
// cold storage only, no live state.
type CharacterSearchHit struct {
	ID          int64
	Name        string
	Slot        int
	Class       uint8
	Level       int32
	Exp         int64
	Coin        int32
	AccountID   int64
	AccountName string
}

// SearchCharacters finds characters by character name or account name
// (case-insensitive substring), highest level first. An empty query returns
// nothing instead of the whole table.
func (s *Store) SearchCharacters(ctx context.Context, query string, limit int) ([]CharacterSearchHit, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.name, c.slot, c.class, c.level, c.exp, c.coin, c.account_id, a.name
		  FROM characters c
		  JOIN account a ON a.id = c.account_id
		 WHERE c.name ILIKE $1 OR a.name ILIKE $1
		 ORDER BY c.level DESC, c.name
		 LIMIT $2`, "%"+q+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("store: search characters %q: %w", q, err)
	}
	defer rows.Close()

	out := make([]CharacterSearchHit, 0, limit)
	for rows.Next() {
		var h CharacterSearchHit
		if err := rows.Scan(&h.ID, &h.Name, &h.Slot, &h.Class, &h.Level, &h.Exp, &h.Coin,
			&h.AccountID, &h.AccountName); err != nil {
			return nil, fmt.Errorf("store: scan character hit: %w", err)
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: search characters: %w", err)
	}
	return out, nil
}

// LoadCharacterByID loads one character's full cold state (stats, equip, cargo,
// affects) by character id, together with its account id and name.
//
// It resolves (account_id, slot) first and reuses LoadCharacter: the 60-column
// load already exists and duplicating it here would be the classic way to drift
// from the save format.
func (s *Store) LoadCharacterByID(ctx context.Context, id int64) (domain.Character, int64, string, error) {
	var accountID int64
	var slot int
	err := s.pool.QueryRow(ctx,
		`SELECT account_id, slot FROM characters WHERE id = $1`, id).Scan(&accountID, &slot)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Character{}, 0, "", ErrNotFound
	}
	if err != nil {
		return domain.Character{}, 0, "", fmt.Errorf("store: locate character %d: %w", id, err)
	}

	var accountName string
	if err := s.pool.QueryRow(ctx,
		`SELECT name FROM account WHERE id = $1`, accountID).Scan(&accountName); err != nil {
		return domain.Character{}, 0, "", fmt.Errorf("store: account name for character %d: %w", id, err)
	}

	ch, err := s.LoadCharacter(ctx, accountID, slot)
	if err != nil {
		return domain.Character{}, 0, "", err
	}
	return ch, accountID, accountName, nil
}
