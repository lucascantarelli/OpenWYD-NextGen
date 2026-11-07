package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"openwyd-nextgen/internal/domain"
)

// Modelos de leitura da receita de donate para o painel de faturamento
// (web.v1.DonateRevenueAdminService). Tudo aqui é SOMENTE LEITURA e abrange
// donate_topup_order (dinheiro real), donate_shop_audit (movimentos de carteira),
// account (identidade do comprador) e donate_payer_profile. O ciclo de escrita
// mora em donate_topup.go e donate.go; este arquivo de propósito não acrescenta nada a ele.
//
// Reconhecimento de receita: o dinheiro é reconhecido em confirmed_at, nunca em created_at —
// um pedido só é receita depois que o gateway liquidou e o ConfirmTopupOrder
// o virou PAID. created_at é usado SÓ para os contadores de funil
// (criados/pendentes), que medem volume, não receita.
//
// Os índices de que estas consultas dependem estão em
// internal/migrations/0017_donate_revenue_indexes.up.sql.

// RevenueWindow é o intervalo semiaberto [From, To) a que toda leitura de receita
// se limita. Quem chama sempre o resolve para um intervalo concreto e limitado
// antes de entrar, então nenhuma consulta aqui precisa de predicado de data com
// guarda de NULL e toda varredura de faixa continua sargable numa coluna indexada.
type RevenueWindow struct {
	From time.Time
	To   time.Time
}

// TopupOrderFilter restringe ListTopupOrders. Zero significa "qualquer" em cada dimensão.
type TopupOrderFilter struct {
	Status        int16 // 0 = qualquer; 1 = PENDING; 2 = PAID
	PaymentMethod int16 // 0 = qualquer
	AccountID     int64 // 0 = qualquer
}

// Palavras de balde da série. Passadas a date_trunc como PARÂMETRO DE BIND (nunca
// concatenação de string) e sempre uma destas constantes do servidor — o enum de
// quem chama é mapeado para elas, nunca repassado como texto livre.
const (
	BucketDay   = "day"
	BucketWeek  = "week"
	BucketMonth = "month"
)

// Ações do livro de donate que movem carteira. As ações de CRUD de catálogo escritas
// por UpsertDonateShopItem/SetDonateShopItemEnabled/DeleteDonateShopItem são
// excluídas de propósito — elas mudam config, não saldo.
const (
	LedgerActionPurchase = "purchase"
	LedgerActionCredit   = "credit_balance"
)

// Os discriminadores de status abaixo são escritos como literais SQL (1 = PENDING,
// 2 = PAID, iguais a TopupStatusPending/TopupStatusPaid) em vez de parâmetros
// de bind, de propósito: donate_topup_order_confirmed_idx (migration 0017) é um
// índice PARCIAL em `status = 2`, e o Postgres só consegue provar que uma consulta
// implica o predicado de índice parcial quando o valor é literal. Um `$n` derrotaria
// o índice sob plano genérico.

// revenueZone é o fuso em que os baldes de dia/semana/mês fecham. O painel é
// lido contra extratos bancários brasileiros, então "hoje" e "este mês" precisam
// terminar à meia-noite de Brasília — NÃO à meia-noite UTC, o que deslocaria toda
// borda em três horas e faria o fechamento do mês discordar do extrato. É uma
// divergência consciente do resto do servidor (a regra de dia da recompensa
// diária é UTC); não "corrija" isto para UTC.
//
// Ele é sempre passado ao date_trunc de três parâmetros, que fixa a borda
// independente do TimeZone da sessão. A forma de dois parâmetros seguiria o
// fuso da sessão em silêncio e tornaria os resultados dependentes do ambiente.
//
// O Brasil não tem horário de verão desde 2019, então os passos de generate_series
// abaixo hoje não cruzam descontinuidade de horário de verão.
const revenueZone = "America/Sao_Paulo"

// ledgerSubjectExpr é a expressão canônica de "de quem a carteira se moveu" sobre
// donate_shop_audit. Existe porque essa tabela é assimétrica: para
// action='purchase' a coluna account_id É o comprador, mas para
// action='credit_balance' a coluna é o MODERADOR que concedeu o crédito e
// a conta creditada aparece só no campo account_id do snapshot JSON (donate.go:207). Um
// relatório de gasto ingênuo com "GROUP BY account_id" atribuiria, portanto, todo
// crédito de cortesia ao moderador que o concedeu.
//
// O índice de expressão donate_shop_audit_subject_idx (migration 0017) casa com
// este texto. Se editar um, edite o outro, ou o planejador para de usar o
// índice em silêncio.
const ledgerSubjectExpr = `COALESCE(NULLIF(au.after->>'account_id', '')::bigint, au.account_id)`

// bucketStep mapeia uma palavra de balde validada para seu passo de generate_series.
func bucketStep(bucket string) (string, error) {
	switch bucket {
	case BucketDay:
		return "1 day", nil
	case BucketWeek:
		return "1 week", nil
	case BucketMonth:
		return "1 month", nil
	default:
		return "", fmt.Errorf("store: revenue series: unknown bucket %q", bucket)
	}
}

// escapeLike neutraliza os metacaracteres de LIKE num prefixo digitado pelo usuário
// para um '%' literal não transformar a busca de prefixo em varredura curinga da
// tabela inteira. O ESCAPE '\' padrão se aplica.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// RevenueTotals é o cabeçalho de KPI sobre donate_topup_order. Os números PAID são
// limitados por confirmed_at (reconhecimento de receita); os de criados/pendentes
// por created_at (volume do funil). accountID 0 significa todas as contas.
func (s *Store) RevenueTotals(ctx context.Context, w RevenueWindow, accountID int64) (domain.RevenueTotals, error) {
	var t domain.RevenueTotals
	// O WHERE externo faz OR das duas faixas de data para o planejador poder BitmapOr
	// dos índices de confirmed_at e created_at; os FILTERs então separam o conjunto.
	//
	// `status = 2` precisa aparecer no WHERE EXTERNO, não só dentro dos FILTERs:
	// cláusulas FILTER são aplicadas depois da varredura e não entram como quals de
	// índice; sem ele o planejador não prova que a consulta implica o
	// predicado parcial de donate_topup_order_confirmed_idx e a consulta inteira
	// degrada para seq scan. Não custa nada semanticamente — confirmed_at só é
	// definido quando um pedido vira PAID, então uma linha não PAID nunca pode
	// satisfazer o primeiro ramo de qualquer forma.
	err := s.pool.QueryRow(ctx, `
		SELECT
			count(*)                   FILTER (WHERE status = 2 AND confirmed_at >= $1 AND confirmed_at < $2),
			COALESCE(sum(amount_cents) FILTER (WHERE status = 2 AND confirmed_at >= $1 AND confirmed_at < $2), 0),
			COALESCE(sum(credits)      FILTER (WHERE status = 2 AND confirmed_at >= $1 AND confirmed_at < $2), 0),
			count(DISTINCT account_id) FILTER (WHERE status = 2 AND confirmed_at >= $1 AND confirmed_at < $2),
			count(*)                   FILTER (WHERE created_at >= $1 AND created_at < $2),
			count(*)                   FILTER (WHERE status = 1 AND created_at >= $1 AND created_at < $2),
			COALESCE(sum(amount_cents) FILTER (WHERE status = 1 AND created_at >= $1 AND created_at < $2), 0)
		FROM donate_topup_order
		WHERE ((status = 2 AND confirmed_at >= $1 AND confirmed_at < $2)
		    OR (created_at >= $1 AND created_at < $2))
		  AND ($3 = 0 OR account_id = $3)`,
		w.From, w.To, accountID,
	).Scan(&t.PaidOrders, &t.GrossCents, &t.CreditsSold, &t.DistinctBuyers,
		&t.CreatedOrders, &t.PendingOrders, &t.PendingCents)
	if err != nil {
		return domain.RevenueTotals{}, fmt.Errorf("store: revenue totals: %w", err)
	}
	return t, nil
}

// RevenueByMethod separa a receita reconhecida por gateway de pagamento.
func (s *Store) RevenueByMethod(ctx context.Context, w RevenueWindow, accountID int64) ([]domain.RevenueByMethod, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT payment_method, count(*), COALESCE(sum(amount_cents), 0)
		FROM donate_topup_order
		WHERE status = 2 AND confirmed_at >= $1 AND confirmed_at < $2
		  AND ($3 = 0 OR account_id = $3)
		GROUP BY payment_method
		ORDER BY payment_method`,
		w.From, w.To, accountID)
	if err != nil {
		return nil, fmt.Errorf("store: revenue by method: %w", err)
	}
	defer rows.Close()

	out := make([]domain.RevenueByMethod, 0, 4)
	for rows.Next() {
		var m domain.RevenueByMethod
		if err := rows.Scan(&m.PaymentMethod, &m.PaidOrders, &m.GrossCents); err != nil {
			return nil, fmt.Errorf("store: scan revenue by method: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: revenue by method: %w", err)
	}
	return out, nil
}

// RevenueSeries agrupa a receita reconhecida por dia/semana/mês em revenueZone.
// bucket precisa ser BucketDay/BucketWeek/BucketMonth. Baldes sem pedidos
// voltam com contadores zerados (generate_series LEFT JOIN agregado), então
// quem chama nunca preenche lacunas. Note que date_trunc('week') começa na SEGUNDA.
func (s *Store) RevenueSeries(ctx context.Context, w RevenueWindow, bucket string, accountID int64) ([]domain.RevenuePoint, error) {
	step, err := bucketStep(bucket)
	if err != nil {
		return nil, err
	}
	// O primeiro balde é alinhado à própria borda, então pode abrir um pouco
	// antes de w.From — é um balde parcial de verdade, não um erro de um.
	rows, err := s.pool.Query(ctx, `
		WITH buckets AS (
			SELECT gs AS bucket_start
			FROM generate_series(
				date_trunc($3, $1::timestamptz, $6),
				$2::timestamptz - interval '1 microsecond',
				$4::interval) AS gs
		), agg AS (
			SELECT date_trunc($3, o.confirmed_at, $6) AS bucket_start,
			       count(*)                         AS paid_orders,
			       COALESCE(sum(o.amount_cents), 0) AS gross_cents,
			       COALESCE(sum(o.credits), 0)      AS credits_sold,
			       count(DISTINCT o.account_id)     AS distinct_buyers
			FROM donate_topup_order o
			WHERE o.status = 2 AND o.confirmed_at >= $1 AND o.confirmed_at < $2
			  AND ($5 = 0 OR o.account_id = $5)
			GROUP BY 1
		)
		SELECT b.bucket_start,
		       COALESCE(a.paid_orders, 0),
		       COALESCE(a.gross_cents, 0),
		       COALESCE(a.credits_sold, 0),
		       COALESCE(a.distinct_buyers, 0)
		FROM buckets b
		LEFT JOIN agg a ON a.bucket_start = b.bucket_start
		ORDER BY b.bucket_start`,
		w.From, w.To, bucket, step, accountID, revenueZone)
	if err != nil {
		return nil, fmt.Errorf("store: revenue series: %w", err)
	}
	defer rows.Close()

	out := make([]domain.RevenuePoint, 0, 32)
	for rows.Next() {
		var p domain.RevenuePoint
		if err := rows.Scan(&p.BucketStart, &p.PaidOrders, &p.GrossCents, &p.CreditsSold, &p.DistinctBuyers); err != nil {
			return nil, fmt.Errorf("store: scan revenue series: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: revenue series: %w", err)
	}
	return out, nil
}

// topupOrderRowCols é a lista de colunas compartilhada pelas duas variantes de
// ListTopupOrders, casada posicionalmente pelo scan abaixo. count(*) OVER() vai
// junto como última coluna para o paginador não precisar de segunda ida e volta.
const topupOrderRowCols = `
	o.id, o.external_reference, o.account_id, a.name, a.email,
	COALESCE(p.name, ''), COALESCE(p.cpf, ''),
	o.credits, o.amount_cents, o.payment_method, COALESCE(o.provider, ''),
	o.status, o.created_at, o.confirmed_at, count(*) OVER()`

// topupOrderFrom junta a identidade do comprador numa consulta. Os dois joins são
// em chave primária, e é por isso que isto é um JOIN e não até `limit` buscas
// extras por página. donate_payer_profile é LEFT — um comprador que nunca preencheu
// o formulário de pagador PIX ainda tem pedidos.
const topupOrderFrom = `
	FROM donate_topup_order o
	JOIN account a ON a.id = o.account_id
	LEFT JOIN donate_payer_profile p ON p.account_id = o.account_id`

// topupOrderPaidWhere limita a janela a confirmed_at — a base da receita. O
// literal `status = 2` é o que permite o índice parcial aplicar; ver a nota de
// status no topo deste arquivo. Compartilhado literalmente entre a consulta da
// página e a de contagem, para as duas nunca divergirem.
const topupOrderPaidWhere = `
	WHERE o.status = 2 AND o.confirmed_at >= $1 AND o.confirmed_at < $2
	  AND ($3 = 0 OR o.payment_method = $3)
	  AND ($4 = 0 OR o.account_id = $4)`

// topupOrderAnyWhere é a base do funil: created_at, qualquer status. Um pedido
// PENDING não tem confirmed_at, então é invisível ao predicado acima.
const topupOrderAnyWhere = `
	WHERE o.created_at >= $1 AND o.created_at < $2
	  AND ($3 = 0 OR o.status = $3)
	  AND ($4 = 0 OR o.payment_method = $4)
	  AND ($5 = 0 OR o.account_id = $5)`

// ListTopupOrders devolve uma página da tabela de pedidos mais a contagem total.
//
// O filtro de status também escolhe a coluna de data da janela: PAID filtra e
// ordena por confirmed_at (a base da receita, casando com o índice parcial),
// qualquer outro por created_at. As duas variantes são predicados constantes
// separados em vez de uma consulta com coluna de data variável, para cada uma
// ter um predicado sargable limpo.
func (s *Store) ListTopupOrders(ctx context.Context, w RevenueWindow, f TopupOrderFilter, limit, offset int) ([]domain.TopupOrderRow, int, error) {
	var (
		where string
		order string
		args  []any
	)
	if f.Status == TopupStatusPaid {
		where = topupOrderPaidWhere
		order = `
			ORDER BY o.confirmed_at DESC, o.id DESC
			LIMIT $5 OFFSET $6`
		args = []any{w.From, w.To, f.PaymentMethod, f.AccountID, limit, offset}
	} else {
		where = topupOrderAnyWhere
		order = `
			ORDER BY o.created_at DESC, o.id DESC
			LIMIT $6 OFFSET $7`
		args = []any{w.From, w.To, f.Status, f.PaymentMethod, f.AccountID, limit, offset}
	}
	// A contagem reusa o mesmo predicado menos os dois binds de paginação, e
	// descarta os joins de identidade — conta pedidos, que os joins não mudam.
	countQ := `SELECT count(*) FROM donate_topup_order o` + where
	cArgs := args[:len(args)-2]

	rows, err := s.pool.Query(ctx, `SELECT `+topupOrderRowCols+topupOrderFrom+where+order, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list topup orders: %w", err)
	}
	defer rows.Close()

	out := make([]domain.TopupOrderRow, 0, limit)
	total := 0
	for rows.Next() {
		var r domain.TopupOrderRow
		var confirmed *time.Time
		if err := rows.Scan(&r.ID, &r.ExternalReference, &r.AccountID, &r.AccountName, &r.AccountEmail,
			&r.PayerName, &r.PayerCPF,
			&r.Credits, &r.AmountCents, &r.PaymentMethod, &r.Provider,
			&r.Status, &r.CreatedAt, &confirmed, &total); err != nil {
			return nil, 0, fmt.Errorf("store: scan topup order: %w", err)
		}
		r.ConfirmedAt = confirmed
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: list topup orders: %w", err)
	}

	total, err = fallbackTotal(ctx, len(out), offset, total, func(ctx context.Context) (int, error) {
		var n int
		if err := s.pool.QueryRow(ctx, countQ, cArgs...).Scan(&n); err != nil {
			return 0, fmt.Errorf("store: count topup orders: %w", err)
		}
		return n, nil
	})
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// listTopBuyersBody é compartilhado entre a consulta da página e sua closure de
// contagem, para as duas nunca divergirem. O WHERE externo é só status, SEM
// data — é isso que torna os agregados lifetime_* um LTV verdadeiro; a janela é
// aplicada dentro dos FILTERs e do HAVING.
const listTopBuyersBody = `
	FROM donate_topup_order o
	JOIN account a ON a.id = o.account_id
	WHERE o.status = 2
	GROUP BY o.account_id, a.name, a.email, a.donate_balance
	HAVING count(*) FILTER (WHERE o.confirmed_at >= $1 AND o.confirmed_at < $2) > 0`

// ListTopBuyers ordena contas por receita dentro da janela e carrega o agregado
// de todos os tempos junto. Só contas que pagaram dentro da janela são
// devolvidas (o HAVING); total é o número dessas contas.
func (s *Store) ListTopBuyers(ctx context.Context, w RevenueWindow, limit, offset int) ([]domain.TopBuyer, int, error) {
	// count(*) OVER() roda DEPOIS do agrupamento, então conta GRUPOS — exatamente o
	// total de compradores distintos que o paginador precisa, não a contagem de pedidos.
	rows, err := s.pool.Query(ctx, `
		SELECT o.account_id, a.name, a.email,
		       count(*)                     FILTER (WHERE o.confirmed_at >= $1 AND o.confirmed_at < $2) AS window_orders,
		       COALESCE(sum(o.amount_cents) FILTER (WHERE o.confirmed_at >= $1 AND o.confirmed_at < $2), 0) AS window_cents,
		       count(*)                         AS lifetime_orders,
		       COALESCE(sum(o.amount_cents), 0) AS lifetime_cents,
		       COALESCE(sum(o.credits), 0)      AS lifetime_credits,
		       min(o.confirmed_at)              AS first_paid_at,
		       max(o.confirmed_at)              AS last_paid_at,
		       a.donate_balance,
		       count(*) OVER()                  AS total`+
		listTopBuyersBody+`
		ORDER BY window_cents DESC, o.account_id
		LIMIT $3 OFFSET $4`,
		w.From, w.To, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list top buyers: %w", err)
	}
	defer rows.Close()

	out := make([]domain.TopBuyer, 0, limit)
	total := 0
	for rows.Next() {
		var b domain.TopBuyer
		var first, last *time.Time
		if err := rows.Scan(&b.AccountID, &b.AccountName, &b.AccountEmail,
			&b.WindowPaidOrders, &b.WindowGrossCents,
			&b.LifetimePaidOrders, &b.LifetimeGrossCents, &b.LifetimeCredits,
			&first, &last, &b.DonateBalance, &total); err != nil {
			return nil, 0, fmt.Errorf("store: scan top buyer: %w", err)
		}
		if first != nil {
			b.FirstPaidAt = *first
		}
		if last != nil {
			b.LastPaidAt = *last
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: list top buyers: %w", err)
	}

	total, err = fallbackTotal(ctx, len(out), offset, total, func(ctx context.Context) (int, error) {
		// Precisa contar GRUPOS, espelhando o count(*) OVER() acima — um count(*) simples
		// sobre a tabela reportaria pedidos em vez de compradores.
		var n int
		if err := s.pool.QueryRow(ctx,
			`SELECT count(*) FROM (SELECT 1`+listTopBuyersBody+`) t`, w.From, w.To).Scan(&n); err != nil {
			return 0, fmt.Errorf("store: count top buyers: %w", err)
		}
		return n, nil
	})
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// ListDonateLedger lê donate_shop_audit como livro de carteira com sinal. actions
// precisa ser subconjunto de {LedgerActionPurchase, LedgerActionCredit}; vazio
// significa os dois. accountID 0 significa todas as contas e é comparado ao SUJEITO
// (ver ledgerSubjectExpr), nunca à coluna crua de auditoria.
func (s *Store) ListDonateLedger(ctx context.Context, w RevenueWindow, actions []string, accountID int64, limit, offset int) ([]domain.DonateLedgerEntry, int, error) {
	if len(actions) == 0 {
		actions = []string{LedgerActionPurchase, LedgerActionCredit}
	}
	// Os joins de account e donate_shop_item são LEFT de propósito:
	// donate_shop_audit.shop_item_id de propósito não é FK, para o histórico
	// sobreviver à exclusão da oferta (0008_donate_shop), e uma linha de auditoria pode
	// sobreviver à conta. Joins internos apagariam esse histórico em silêncio.
	const body = `
		FROM donate_shop_audit au
		WHERE au.action = ANY($3::text[])
		  AND au.created_at >= $1 AND au.created_at < $2
		  AND ($4 = 0 OR ` + ledgerSubjectExpr + ` = $4)`

	rows, err := s.pool.Query(ctx, `
		WITH led AS (
			SELECT au.id, au.action, au.created_at, au.shop_item_id, au.after,
			       au.account_id AS actor_account_id,
			       `+ledgerSubjectExpr+` AS subject_account_id`+body+`
		)
		SELECT l.id, l.action, l.created_at,
		       l.subject_account_id, COALESCE(subj.name, ''),
		       l.actor_account_id,   COALESCE(actor.name, ''),
		       CASE l.action
		           WHEN '`+LedgerActionPurchase+`' THEN -COALESCE((l.after->>'price')::bigint, 0)
		           WHEN '`+LedgerActionCredit+`'   THEN  COALESCE((l.after->>'amount')::bigint, 0)
		           ELSE 0
		       END,
		       COALESCE((l.after->>'balance')::bigint, 0),
		       COALESCE(l.shop_item_id, 0),
		       COALESCE(si.title, ''),
		       COALESCE(l.after->>'reason', ''),
		       count(*) OVER()
		FROM led l
		LEFT JOIN account          subj  ON subj.id  = l.subject_account_id
		LEFT JOIN account          actor ON actor.id = l.actor_account_id
		LEFT JOIN donate_shop_item si    ON si.id    = l.shop_item_id
		ORDER BY l.created_at DESC, l.id DESC
		LIMIT $5 OFFSET $6`,
		w.From, w.To, actions, accountID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list donate ledger: %w", err)
	}
	defer rows.Close()

	out := make([]domain.DonateLedgerEntry, 0, limit)
	total := 0
	for rows.Next() {
		var e domain.DonateLedgerEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.CreatedAt,
			&e.SubjectAccountID, &e.SubjectAccountName,
			&e.ActorAccountID, &e.ActorAccountName,
			&e.CreditsDelta, &e.BalanceAfter,
			&e.ShopItemID, &e.ShopItemTitle, &e.Reason, &total); err != nil {
			return nil, 0, fmt.Errorf("store: scan donate ledger: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: list donate ledger: %w", err)
	}

	total, err = fallbackTotal(ctx, len(out), offset, total, func(ctx context.Context) (int, error) {
		var n int
		if err := s.pool.QueryRow(ctx, `SELECT count(*)`+body,
			w.From, w.To, actions, accountID).Scan(&n); err != nil {
			return 0, fmt.Errorf("store: count donate ledger: %w", err)
		}
		return n, nil
	})
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// DonateLedgerTotals agrega os movimentos de carteira na janela, em créditos.
// accountID 0 significa todas as contas; senão compara com o SUJEITO.
func (s *Store) DonateLedgerTotals(ctx context.Context, w RevenueWindow, accountID int64) (domain.DonateLedgerTotals, error) {
	var t domain.DonateLedgerTotals
	err := s.pool.QueryRow(ctx, `
		SELECT
			count(*)                                    FILTER (WHERE au.action = '`+LedgerActionPurchase+`'),
			COALESCE(sum((au.after->>'price')::bigint)  FILTER (WHERE au.action = '`+LedgerActionPurchase+`'), 0),
			count(*)                                    FILTER (WHERE au.action = '`+LedgerActionCredit+`'),
			COALESCE(sum((au.after->>'amount')::bigint) FILTER (WHERE au.action = '`+LedgerActionCredit+`'), 0)
		FROM donate_shop_audit au
		WHERE au.action IN ('`+LedgerActionPurchase+`', '`+LedgerActionCredit+`')
		  AND au.created_at >= $1 AND au.created_at < $2
		  AND ($3 = 0 OR `+ledgerSubjectExpr+` = $3)`,
		w.From, w.To, accountID,
	).Scan(&t.ShopPurchases, &t.CreditsSpent, &t.ManualCredits, &t.CreditsGranted)
	if err != nil {
		return domain.DonateLedgerTotals{}, fmt.Errorf("store: donate ledger totals: %w", err)
	}
	return t, nil
}

// SearchAccountsByNamePrefix resolve um prefixo canônico (minúsculo) de login para o
// painel transformar um nome digitado em filtro de account_id. Quem chama passa
// para minúsculas e confere o tamanho do prefixo; aqui escapamos os metacaracteres de LIKE.
func (s *Store) SearchAccountsByNamePrefix(ctx context.Context, prefix string, limit int) ([]domain.AccountSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, email, donate_balance, role, is_blocked
		FROM account
		WHERE name LIKE $1
		ORDER BY name
		LIMIT $2`, escapeLike(prefix)+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("store: search accounts: %w", err)
	}
	defer rows.Close()

	out := make([]domain.AccountSummary, 0, limit)
	for rows.Next() {
		var a domain.AccountSummary
		if err := rows.Scan(&a.ID, &a.Name, &a.Email, &a.DonateBalance, &a.Role, &a.IsBlocked); err != nil {
			return nil, fmt.Errorf("store: scan account summary: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: search accounts: %w", err)
	}
	return out, nil
}
