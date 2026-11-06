package domain

import "time"

// AccountRow é uma conta como a UI de moderação a vê. Nunca carrega pass_hash,
// pin_hash nem block_pass_hash: o caminho de leitura é moldado de propósito para
// um segredo não ser serializado por acidente.
type AccountRow struct {
	ID            int64
	Name          string
	Email         string
	Role          string // 'player' | 'moderator' | 'admin'
	IsBlocked     bool
	BlockedReason string
	DonateBalance int32
	CreatedAt     time.Time
	// RoleChangedAt/RoleChangedBy são nil em conta que a web nunca tocou.
	RoleChangedAt *time.Time
	RoleChangedBy *int64
	Characters    int32
}

// AccountFilter é a consulta de /contas. Query é prefixo de login (o nome
// canônico é minúsculo), Role é casamento exato ("" = qualquer) e Blocked usa as
// constantes Blocked*. AfterID é o cursor de keyset (0 = desde o início).
type AccountFilter struct {
	Query   string
	Role    string
	Blocked int32
	AfterID int64
	Limit   int
}

// Blocked* são os valores tri-estado de AccountFilter.Blocked.
const (
	BlockedAny int32 = -1
	BlockedNo  int32 = 0
	BlockedYes int32 = 1
)

// AccountAudit é uma linha de account_admin_audit.
type AccountAudit struct {
	ID        int64
	ActorID   int64
	TargetID  int64
	Action    string
	OldValue  string
	NewValue  string
	Reason    string
	CreatedAt time.Time
}
