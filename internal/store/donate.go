package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// Persistência da loja de donate web (issue #34). O Postgres é dono do catálogo
// (donate_shop_item) e da carteira de donate (account.donate_balance); uma compra
// debita a carteira e enfileira o item em delivery_queue numa transação.
// O tmServer nunca lê o catálogo — só drena delivery_queue para o
// cargo da conta dentro do seu loop de dono único. Creditar/debitar donate_balance
// é atualização PARCIAL de conta, então nenhum save do tmServer a sobrescreve (SaveCharacter
// nunca toca account; SaveCargo escreve só cargo_coin).

// ErrInsufficientDonate é devolvido por BuyDonateItem quando o saldo de donate
// da conta está abaixo do preço do item.
var ErrInsufficientDonate = errors.New("store: insufficient donate balance")

// ErrShopItemDisabled é devolvido por BuyDonateItem quando a oferta existe
// mas não está habilitada para venda.
var ErrShopItemDisabled = errors.New("store: donate shop item disabled")

// itemPayload é o formato de delivery_queue.payload para kind='item': tudo que a
// drenagem do tmServer precisa para materializar um STRUCT_ITEM no cargo da conta.
// ExpiresAt é Unix-segundos absoluto (0 = permanente).
type itemPayload struct {
	ItemIndex int32 `json:"item_index"`
	Eff1      uint8 `json:"eff1"`
	EffV1     uint8 `json:"effv1"`
	Eff2      uint8 `json:"eff2"`
	EffV2     uint8 `json:"effv2"`
	Eff3      uint8 `json:"eff3"`
	EffV3     uint8 `json:"effv3"`
	ExpiresAt int64 `json:"expires_at"`
}

// donateShopItemCols é a lista de colunas compartilhada por toda leitura de
// donate_shop_item, casada posicionalmente por scanDonateShopItem.
const donateShopItemCols = `id, item_index, eff1, effv1, eff2, effv2, eff3, effv3, price, title, description, enabled, expires_days`

// ListDonateShopItems devolve toda oferta da loja (habilitada ou não), por id —
// a tabela de moderação. ListEnabledDonateShopItems é a vitrine do jogador.
func (s *Store) ListDonateShopItems(ctx context.Context) ([]domain.DonateShopItem, error) {
	return s.queryDonateShopItems(ctx, `SELECT `+donateShopItemCols+` FROM donate_shop_item ORDER BY id`)
}

// ListEnabledDonateShopItems devolve só as ofertas à venda (a vitrine web).
func (s *Store) ListEnabledDonateShopItems(ctx context.Context) ([]domain.DonateShopItem, error) {
	return s.queryDonateShopItems(ctx, `SELECT `+donateShopItemCols+` FROM donate_shop_item WHERE enabled = TRUE ORDER BY id`)
}

func (s *Store) queryDonateShopItems(ctx context.Context, sql string) ([]domain.DonateShopItem, error) {
	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("store: list donate shop items: %w", err)
	}
	defer rows.Close()
	var out []domain.DonateShopItem
	for rows.Next() {
		d, err := scanDonateShopItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDonateShopItem carrega uma oferta por id. Devolve ErrNotFound quando ausente.
func (s *Store) GetDonateShopItem(ctx context.Context, id int64) (domain.DonateShopItem, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+donateShopItemCols+` FROM donate_shop_item WHERE id = $1`, id)
	d, err := scanDonateShopItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DonateShopItem{}, ErrNotFound
	}
	if err != nil {
		return domain.DonateShopItem{}, fmt.Errorf("store: get donate shop item %d: %w", id, err)
	}
	return d, nil
}

// scanRow é a superfície de leitura compartilhada por pgx.Row e pgx.Rows (ambos têm Scan).
type scanRow interface {
	Scan(dest ...any) error
}

func scanDonateShopItem(row scanRow) (domain.DonateShopItem, error) {
	var d domain.DonateShopItem
	err := row.Scan(&d.ID, &d.ItemIndex, &d.Eff1, &d.EffV1, &d.Eff2, &d.EffV2, &d.Eff3, &d.EffV3,
		&d.Price, &d.Title, &d.Description, &d.Enabled, &d.ExpiresDays)
	return d, err
}

// UpsertDonateShopItem insere oferta nova (d.ID == 0) ou atualiza a existente
// por id, gravando auditoria na mesma transação. Devolve o id da oferta;
// atualizar um id inexistente devolve ErrNotFound.
func (s *Store) UpsertDonateShopItem(ctx context.Context, d domain.DonateShopItem, moderatorID int64) (int64, error) {
	var id int64
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		if d.ID == 0 {
			if err := tx.QueryRow(ctx, `
				INSERT INTO donate_shop_item
					(item_index, eff1, effv1, eff2, effv2, eff3, effv3, price, title, description, enabled, expires_days, updated_by, updated_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13, now())
				RETURNING id`,
				d.ItemIndex, d.Eff1, d.EffV1, d.Eff2, d.EffV2, d.Eff3, d.EffV3,
				d.Price, d.Title, d.Description, d.Enabled, d.ExpiresDays, nullableID(moderatorID),
			).Scan(&id); err != nil {
				return fmt.Errorf("store: insert donate shop item: %w", err)
			}
			after, _ := fetchDonateItemJSON(ctx, tx, id)
			return donateAudit(ctx, tx, &id, moderatorID, "create", nil, after)
		}

		before, _ := fetchDonateItemJSON(ctx, tx, d.ID)
		if before == nil {
			return ErrNotFound
		}
		id = d.ID
		if _, err := tx.Exec(ctx, `
			UPDATE donate_shop_item SET
				item_index=$2, eff1=$3, effv1=$4, eff2=$5, effv2=$6, eff3=$7, effv3=$8,
				price=$9, title=$10, description=$11, enabled=$12, expires_days=$13,
				updated_by=$14, updated_at=now()
			WHERE id=$1`,
			id, d.ItemIndex, d.Eff1, d.EffV1, d.Eff2, d.EffV2, d.Eff3, d.EffV3,
			d.Price, d.Title, d.Description, d.Enabled, d.ExpiresDays, nullableID(moderatorID),
		); err != nil {
			return fmt.Errorf("store: update donate shop item %d: %w", id, err)
		}
		after, _ := fetchDonateItemJSON(ctx, tx, id)
		return donateAudit(ctx, tx, &id, moderatorID, "update", before, after)
	})
	return id, err
}

// SetDonateShopItemEnabled alterna se uma oferta está à venda. Devolve
// ErrNotFound se a oferta não existir.
func (s *Store) SetDonateShopItemEnabled(ctx context.Context, id int64, enabled bool, moderatorID int64) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		before, _ := fetchDonateItemJSON(ctx, tx, id)
		if before == nil {
			return ErrNotFound
		}
		if _, err := tx.Exec(ctx,
			`UPDATE donate_shop_item SET enabled=$2, updated_by=$3, updated_at=now() WHERE id=$1`,
			id, enabled, nullableID(moderatorID)); err != nil {
			return fmt.Errorf("store: set donate shop item %d enabled: %w", id, err)
		}
		after, _ := fetchDonateItemJSON(ctx, tx, id)
		return donateAudit(ctx, tx, &id, moderatorID, "set_enabled", before, after)
	})
}

// DeleteDonateShopItem remove uma oferta. Devolve ErrNotFound se ausente.
func (s *Store) DeleteDonateShopItem(ctx context.Context, id int64, moderatorID int64) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		before, _ := fetchDonateItemJSON(ctx, tx, id)
		if before == nil {
			return ErrNotFound
		}
		if _, err := tx.Exec(ctx, `DELETE FROM donate_shop_item WHERE id = $1`, id); err != nil {
			return fmt.Errorf("store: delete donate shop item %d: %w", id, err)
		}
		return donateAudit(ctx, tx, &id, moderatorID, "delete", before, nil)
	})
}

// DonateBalance devolve o saldo da carteira de donate de uma conta. Devolve ErrNotFound
// quando a conta não existe.
func (s *Store) DonateBalance(ctx context.Context, accountID int64) (int32, error) {
	var bal int32
	err := s.pool.QueryRow(ctx, `SELECT donate_balance FROM account WHERE id = $1`, accountID).Scan(&bal)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("store: donate balance a=%d: %w", accountID, err)
	}
	return bal, nil
}

// CreditDonateBalance soma amount à carteira de donate de uma conta (o caminho de
// crédito manual/admin que o webhook de pagamento futuro reusa) e devolve o novo saldo. É
// um UPDATE parcial — nunca escrita da linha inteira — então é seguro contra saves do
// tmServer. Devolve ErrNotFound se a conta está ausente.
func (s *Store) CreditDonateBalance(ctx context.Context, accountID int64, amount int32, moderatorID int64, reason string) (int32, error) {
	var newBal int32
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`UPDATE account SET donate_balance = donate_balance + $2 WHERE id = $1 RETURNING donate_balance`,
			accountID, amount).Scan(&newBal)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("store: credit donate a=%d: %w", accountID, err)
		}
		after, _ := json.Marshal(map[string]any{
			"account_id": accountID, "amount": amount, "balance": newBal, "reason": reason,
		})
		return donateAudit(ctx, tx, nil, moderatorID, "credit_balance", nil, after)
	})
	return newBal, err
}

// BuyDonateItem debita o preço da oferta do saldo de donate da conta e
// enfileira o item em delivery_queue, transacionalmente, devolvendo o novo
// saldo. O tmServer entrega o item enfileirado no cargo da conta no
// próximo login (web-platform-plan.md §mailbox). Devolve ErrNotFound (oferta ou
// conta desconhecida), ErrShopItemDisabled (oferta fora de venda) ou ErrInsufficientDonate.
func (s *Store) BuyDonateItem(ctx context.Context, accountID, shopItemID int64) (int32, error) {
	var newBal int32
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		it, err := scanDonateShopItem(tx.QueryRow(ctx,
			`SELECT `+donateShopItemCols+` FROM donate_shop_item WHERE id = $1`, shopItemID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("store: buy: load offer %d: %w", shopItemID, err)
		}
		if !it.Enabled {
			return ErrShopItemDisabled
		}

		// Trava a linha da carteira e confere o saldo antes de debitar, para o resultado
		// ser inequívoco (saldo insuficiente vs conta inexistente).
		var bal int32
		err = tx.QueryRow(ctx, `SELECT donate_balance FROM account WHERE id = $1 FOR UPDATE`, accountID).Scan(&bal)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("store: buy: load balance a=%d: %w", accountID, err)
		}
		if bal < it.Price {
			return ErrInsufficientDonate
		}
		if err := tx.QueryRow(ctx,
			`UPDATE account SET donate_balance = donate_balance - $2 WHERE id = $1 RETURNING donate_balance`,
			accountID, it.Price).Scan(&newBal); err != nil {
			return fmt.Errorf("store: buy: debit a=%d: %w", accountID, err)
		}

		payload, err := json.Marshal(itemPayload{
			ItemIndex: it.ItemIndex,
			Eff1:      it.Eff1, EffV1: it.EffV1, Eff2: it.Eff2, EffV2: it.EffV2, Eff3: it.Eff3, EffV3: it.EffV3,
			ExpiresAt: expiresDaysToUnix(it.ExpiresDays),
		})
		if err != nil {
			return fmt.Errorf("store: buy: marshal payload: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO delivery_queue (account_id, kind, payload, source)
			VALUES ($1, 'item', $2, $3)`,
			accountID, payload, fmt.Sprintf("donate_shop:%d", it.ID)); err != nil {
			return fmt.Errorf("store: buy: enqueue delivery a=%d: %w", accountID, err)
		}
		after, _ := json.Marshal(map[string]any{
			"account_id": accountID, "shop_item_id": it.ID, "price": it.Price, "balance": newBal,
		})
		return donateAudit(ctx, tx, &it.ID, accountID, "purchase", nil, after)
	})
	return newBal, err
}

// PendingItemDeliveries devolve as concessões de item pendentes da conta na
// caixa, das mais antigas para as mais novas. O tmServer a chama no loop no login,
// aplica cada uma ao cargo da conta e confirma via SaveCargoWithDeliveries.
func (s *Store) PendingItemDeliveries(ctx context.Context, accountID int64) ([]domain.Delivery, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, payload FROM delivery_queue
		WHERE account_id = $1 AND status = 'pending' AND kind = 'item'
		ORDER BY id`, accountID)
	if err != nil {
		return nil, fmt.Errorf("store: pending deliveries a=%d: %w", accountID, err)
	}
	defer rows.Close()
	var out []domain.Delivery
	for rows.Next() {
		var id int64
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, fmt.Errorf("store: scan delivery: %w", err)
		}
		var p itemPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("store: decode delivery %d payload: %w", id, err)
		}
		out = append(out, domain.Delivery{
			ID: id,
			Item: domain.Item{
				Index: int16(p.ItemIndex),
				Eff1:  p.Eff1, EffV1: p.EffV1, Eff2: p.Eff2, EffV2: p.EffV2, Eff3: p.Eff3, EffV3: p.EffV3,
				ExpiresAt: p.ExpiresAt,
			},
		})
	}
	return out, rows.Err()
}

// SaveCargoWithDeliveries persiste o cargo da conta (ouro + itens, substituição
// total como SaveCargo) E marca as linhas drenadas da caixa como entregues/perdidas,
// tudo numa transação. Confirmar as duas juntas é a fronteira anti-duplicação: uma
// queda antes do commit deixa as linhas 'pending' e o cargo intacto, então o próximo
// login drena de novo limpo, sem duplicação. Devolve ErrNotFound se a conta estiver
// ausente.
func (s *Store) SaveCargoWithDeliveries(ctx context.Context, accountID int64, coin int32, items []domain.Item, deliveredIDs, lostIDs []int64) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE account SET cargo_coin = $2 WHERE id = $1`, accountID, coin)
		if err != nil {
			return fmt.Errorf("store: drain: update cargo coin a=%d: %w", accountID, err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM item WHERE account_id = $1 AND owner_kind = 'account_cargo'`, accountID); err != nil {
			return fmt.Errorf("store: drain: clear cargo items a=%d: %w", accountID, err)
		}
		for _, it := range items {
			if err := insertItem(ctx, tx, "account_cargo", &accountID, nil, it); err != nil {
				return err
			}
		}
		if err := markDeliveries(ctx, tx, "delivered", deliveredIDs); err != nil {
			return err
		}
		return markDeliveries(ctx, tx, "lost", lostIDs)
	})
}

func markDeliveries(ctx context.Context, tx pgx.Tx, status string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx,
		`UPDATE delivery_queue SET status = $1 WHERE id = ANY($2)`, status, ids); err != nil {
		return fmt.Errorf("store: mark deliveries %s: %w", status, err)
	}
	return nil
}

// --- helpers ---

// expiresDaysToUnix converte expires_days de uma oferta em timestamp absoluto de
// expiração (Unix segundos); 0 dias = permanente (0).
func expiresDaysToUnix(days int32) int64 {
	if days <= 0 {
		return 0
	}
	return time.Now().Add(time.Duration(days) * 24 * time.Hour).Unix()
}

// donateAudit grava uma linha em donate_shop_audit. Diferente de auditAndBump, não há
// versão de config a subir — o tmServer nunca lê o catálogo de donate.
func donateAudit(ctx context.Context, tx pgx.Tx, shopItemID *int64, accountID int64, action string, before, after []byte) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO donate_shop_audit (shop_item_id, account_id, action, before, after)
		VALUES ($1,$2,$3,$4,$5)`,
		shopItemID, accountID, action, nullableJSON(before), nullableJSON(after)); err != nil {
		return fmt.Errorf("store: write donate audit: %w", err)
	}
	return nil
}

func fetchDonateItemJSON(ctx context.Context, tx pgx.Tx, id int64) ([]byte, error) {
	var js []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(d) FROM donate_shop_item d WHERE id = $1`, id).Scan(&js)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return js, err
}
