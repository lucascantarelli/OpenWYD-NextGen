package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"openwyd-nextgen/internal/domain"
)

// Persistência de recarga de donate (agnóstica de meio de pagamento). O Postgres é
// dono do perfil do pagador (donate_payer_profile) e dos pedidos de recarga
// (donate_topup_order); o portal não tem banco próprio. O crédito de um pedido pago
// acontece dentro de ConfirmTopupOrder, na mesma transação que vira o pedido para
// PAID; o UNIQUE(external_reference) mais o FOR UPDATE e a checagem de status
// tornam o crédito idempotente: um webhook de gateway reenviado credita exatamente
// uma vez. Creditar account.donate_balance é um UPDATE parcial, então nenhum save
// do tmServer o sobrescreve (igual a CreditDonateBalance, donate.go).
// Top-up order status values, matching webv1.TopupStatus.
const (
	TopupStatusPending int16 = 1
	TopupStatusPaid    int16 = 2
)

// ErrDuplicateExternalRef é devolvido por CreateTopupOrder quando external_reference
// já existe — o portal reusou um UUID.
var ErrDuplicateExternalRef = errors.New("store: duplicate topup external_reference")

// TopupConfirmOutcome é o resultado de ConfirmTopupOrder, espelhando
// webv1.TopupResult sem a dependência de wire.
type TopupConfirmOutcome int

const (
	// TopupConfirmed significa que o pedido era PENDING e foi creditado agora.
	TopupConfirmed TopupConfirmOutcome = iota
	// TopupAlreadyConfirmed significa que o pedido já era PAID — não foi recreditado.
	TopupAlreadyConfirmed
	// TopupNotFound significa que nenhum pedido tem essa external_reference.
	TopupNotFound
)

// SavePayerProfile faz upsert do nome + CPF do pagador por id de conta. Espera-se que
// o cpf já venha normalizado para 11 dígitos por quem chama. Uma conta ausente (violação de FK)
// devolve ErrNotFound.
func (s *Store) SavePayerProfile(ctx context.Context, accountID int64, name, cpf string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO donate_payer_profile (account_id, name, cpf, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (account_id) DO UPDATE
			SET name = EXCLUDED.name, cpf = EXCLUDED.cpf, updated_at = now()`,
		accountID, name, cpf)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: save payer profile a=%d: %w", accountID, err)
	}
	return nil
}

// GetPayerProfile devolve nome + CPF do pagador. found é falso (sem erro) quando
// a conta ainda não tem perfil.
func (s *Store) GetPayerProfile(ctx context.Context, accountID int64) (name, cpf string, found bool, err error) {
	err = s.pool.QueryRow(ctx,
		`SELECT name, cpf FROM donate_payer_profile WHERE account_id = $1`, accountID).Scan(&name, &cpf)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("store: get payer profile a=%d: %w", accountID, err)
	}
	return name, cpf, true, nil
}

// CreateTopupOrder insere um pedido PENDING e devolve seu id. Uma
// external_reference (unique_violation) devolve ErrDuplicateExternalRef; uma
// conta ausente (violação de FK) devolve ErrNotFound.
func (s *Store) CreateTopupOrder(ctx context.Context, o domain.TopupOrder) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO donate_topup_order
			(external_reference, account_id, credits, amount_cents, payment_method, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		o.ExternalReference, o.AccountID, o.Credits, o.AmountCents, o.PaymentMethod, TopupStatusPending,
	).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation on external_reference
			return 0, ErrDuplicateExternalRef
		case "23503": // foreign_key_violation on account_id
			return 0, ErrNotFound
		}
	}
	if err != nil {
		return 0, fmt.Errorf("store: create topup order ref=%q: %w", o.ExternalReference, err)
	}
	return id, nil
}

// ConfirmTopupOrder é o núcleo da idempotência: trava o pedido por
// external_reference, e só se ele ainda estiver PENDING credita o
// saldo de donate da conta e vira o pedido para PAID — tudo numa transação. Um
// pedido inexistente devolve (TopupNotFound, 0); um pedido já PAID devolve
// (TopupAlreadyConfirmed, currentBalance) SEM creditar de novo.
func (s *Store) ConfirmTopupOrder(ctx context.Context, externalRef string) (TopupConfirmOutcome, int32, error) {
	var outcome TopupConfirmOutcome
	var balance int32
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var orderID, accountID int64
		var credits int32
		var status int16
		err := tx.QueryRow(ctx, `
			SELECT id, account_id, credits, status
			FROM donate_topup_order WHERE external_reference = $1 FOR UPDATE`,
			externalRef).Scan(&orderID, &accountID, &credits, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			outcome = TopupNotFound
			return nil
		}
		if err != nil {
			return fmt.Errorf("store: confirm topup: load ref=%q: %w", externalRef, err)
		}

		if status == TopupStatusPaid {
			// Já liquidado: informa o saldo corrente, nunca recredita.
			if err := tx.QueryRow(ctx,
				`SELECT donate_balance FROM account WHERE id = $1`, accountID).Scan(&balance); err != nil {
				return fmt.Errorf("store: confirm topup: read balance a=%d: %w", accountID, err)
			}
			outcome = TopupAlreadyConfirmed
			return nil
		}

		// PENDING: credit + mark PAID together (partial UPDATE, safe vs tmServer saves).
		if err := tx.QueryRow(ctx,
			`UPDATE account SET donate_balance = donate_balance + $2 WHERE id = $1 RETURNING donate_balance`,
			accountID, credits).Scan(&balance); err != nil {
			return fmt.Errorf("store: confirm topup: credit a=%d: %w", accountID, err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE donate_topup_order SET status = $2, confirmed_at = now() WHERE id = $1`,
			orderID, TopupStatusPaid); err != nil {
			return fmt.Errorf("store: confirm topup: mark paid id=%d: %w", orderID, err)
		}
		outcome = TopupConfirmed
		return nil
	})
	if err != nil {
		return TopupNotFound, 0, err
	}
	return outcome, balance, nil
}

// GetTopupOrder devolve o status e os créditos de um pedido, limitado ao dono: um
// pedido de outra conta (ou ausente) devolve ErrNotFound para nenhum
// pedido não vaze entre contas. Quando o pedido é PAID, newBalance carrega o
// saldo de donate corrente da conta; caso contrário é 0.
func (s *Store) GetTopupOrder(ctx context.Context, externalRef string, accountID int64) (status int16, credits int32, newBalance int32, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT status, credits FROM donate_topup_order
		WHERE external_reference = $1 AND account_id = $2`,
		externalRef, accountID).Scan(&status, &credits)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, 0, ErrNotFound
	}
	if err != nil {
		return 0, 0, 0, fmt.Errorf("store: get topup order ref=%q: %w", externalRef, err)
	}
	if status == TopupStatusPaid {
		if err := s.pool.QueryRow(ctx,
			`SELECT donate_balance FROM account WHERE id = $1`, accountID).Scan(&newBalance); err != nil {
			return 0, 0, 0, fmt.Errorf("store: get topup order: read balance a=%d: %w", accountID, err)
		}
	}
	return status, credits, newBalance, nil
}
