package domain

import "time"

// AccountRow is one account as the moderation UI sees it. It never carries
// pass_hash, pin_hash or block_pass_hash: the read path is deliberately shaped
// so a secret cannot be serialized by accident (issue 32).
type AccountRow struct {
	ID            int64
	Name          string
	Email         string
	Role          string // 'player' | 'moderator' | 'admin'
	IsBlocked     bool
	BlockedReason string
	DonateBalance int32
	CreatedAt     time.Time
	// RoleChangedAt/RoleChangedBy are nil for accounts never touched by the web.
	RoleChangedAt *time.Time
	RoleChangedBy *int64
	Characters    int32
}

// AccountFilter is the /contas query. Query is a login prefix (the canonical name
// is lowercase), Role is an exact match ("" = any) and Blocked uses the
// Blocked* constants. AfterID is the keyset cursor (0 = from the start).
type AccountFilter struct {
	Query   string
	Role    string
	Blocked int32
	AfterID int64
	Limit   int
}

// Blocked* are the tri-state values of AccountFilter.Blocked.
const (
	BlockedAny int32 = -1
	BlockedNo  int32 = 0
	BlockedYes int32 = 1
)

// AccountAudit is one row of account_admin_audit.
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
