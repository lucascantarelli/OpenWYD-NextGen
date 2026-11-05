package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// EnqueueItemGrant puts `quantity` item grants for one character into the
// delivery_queue mailbox and records the action in character_admin_audit, all in
// one transaction (issue 43).
//
// It is the ONLY write path for a character item from the web: the tmServer
// drains the mailbox into the account cargo inside its single-owner loop at
// login, so an online character never gets its in-memory STRUCT_MOB clobbered.
//
// One row per unit: delivery_queue.payload (donate.go:itemPayload) has no
// quantity field, and encoding the stack as EF_AMOUNT would silently overwrite
// one of the moderator's chosen effects.
func (s *Store) EnqueueItemGrant(
	ctx context.Context,
	moderatorID, accountID, characterID int64,
	item domain.Item,
	quantity int,
	source, reason string,
) ([]int64, error) {
	var ids []int64
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		for i := 0; i < quantity; i++ {
			payload, err := json.Marshal(itemPayload{
				ItemIndex: int32(item.Index),
				Eff1:      item.Eff1, EffV1: item.EffV1,
				Eff2: item.Eff2, EffV2: item.EffV2,
				Eff3: item.Eff3, EffV3: item.EffV3,
				ExpiresAt: item.ExpiresAt,
			})
			if err != nil {
				return fmt.Errorf("store: marshalling grant payload: %w", err)
			}
			var id int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO delivery_queue (account_id, character_id, kind, payload, source)
				VALUES ($1, $2, 'item', $3, $4)
				RETURNING id`,
				accountID, characterID, payload, source).Scan(&id); err != nil {
				return fmt.Errorf("store: enqueue item grant: %w", err)
			}
			ids = append(ids, id)
		}

		after, err := json.Marshal(map[string]any{
			"item_index":    item.Index,
			"quantity":      quantity,
			"effects":       [][2]uint8{{item.Eff1, item.EffV1}, {item.Eff2, item.EffV2}, {item.Eff3, item.EffV3}},
			"expires_at":    item.ExpiresAt,
			"delivery_ids":  ids,
			"account_id":    accountID,
			"source":        source,
		})
		if err != nil {
			return fmt.Errorf("store: marshalling audit: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO character_admin_audit (moderator_id, character_id, action, before, after, reason)
			VALUES ($1, $2, 'grant_item', NULL, $3, $4)`,
			moderatorID, characterID, after, reason); err != nil {
			return fmt.Errorf("store: audit grant_item: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// CharacterItemGrant is one pending grant shown in the character history.
type CharacterItemGrant struct {
	DeliveryID  int64
	CharacterID int64
	ItemIndex   int32
	Status      string
	Source      string
	CreatedAt   string
}

// PendingItemGrants lists the character's mailbox rows (pending and delivered),
// newest first, so the portal can show "entrou na fila / já entregue".
func (s *Store) PendingItemGrants(ctx context.Context, characterID int64, limit int) ([]CharacterItemGrant, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, character_id, (payload->>'item_index')::int, status, COALESCE(source, ''),
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SSOF')
		  FROM delivery_queue
		 WHERE character_id = $1
		 ORDER BY id DESC
		 LIMIT $2`, characterID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list item grants for character %d: %w", characterID, err)
	}
	defer rows.Close()

	out := make([]CharacterItemGrant, 0, limit)
	for rows.Next() {
		var g CharacterItemGrant
		if err := rows.Scan(&g.DeliveryID, &g.CharacterID, &g.ItemIndex, &g.Status, &g.Source, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan item grant: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
