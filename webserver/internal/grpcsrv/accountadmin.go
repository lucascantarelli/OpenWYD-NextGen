package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webv1 "openwyd-nextgen/api/web/v1"
	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/webserver/internal/accountadmin"
)

// AccountAdmin is the account moderation surface (satisfied by
// *accountadmin.Service). Kept as an interface so the server is unit-testable.
type AccountAdmin interface {
	List(ctx context.Context, actorID int64, f domain.AccountFilter) (accountadmin.Result, []domain.AccountRow, int, error)
	Get(ctx context.Context, actorID, accountID int64) (accountadmin.Result, domain.AccountRow, []domain.AccountAudit, error)
	SetRole(ctx context.Context, actorID, accountID int64, role, reason string) (accountadmin.Result, domain.AccountRow, string, error)
	SetBlocked(ctx context.Context, actorID, accountID int64, blocked bool, reason string) (accountadmin.Result, domain.AccountRow, string, error)
	Create(ctx context.Context, actorID int64, name, password, email, role, reason string) (accountadmin.Result, int64, string, error)
}

// AccountAdminServer implements webv1.AccountAdminServiceServer.
type AccountAdminServer struct {
	webv1.UnimplementedAccountAdminServiceServer
	accounts AccountAdmin
}

// NewAccountAdmin builds the AccountAdminService over the given logic.
func NewAccountAdmin(a AccountAdmin) *AccountAdminServer {
	return &AccountAdminServer{accounts: a}
}

// ListAccounts returns one keyset page of accounts plus the filtered total.
func (s *AccountAdminServer) ListAccounts(ctx context.Context, req *webv1.ListAccountsRequest) (*webv1.ListAccountsResponse, error) {
	res, rows, total, err := s.accounts.List(ctx, req.GetModeratorId(), domain.AccountFilter{
		Query:   req.GetQuery(),
		Role:    req.GetRole(),
		Blocked: req.GetBlocked(),
		AfterID: req.GetAfterId(),
		Limit:   int(req.GetLimit()),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list accounts: %v", err)
	}
	out := &webv1.ListAccountsResponse{
		Result:     accountResultToProto(res),
		Accounts:   make([]*webv1.AccountRow, 0, len(rows)),
		TotalCount: int32(total),
	}
	for _, row := range rows {
		out.Accounts = append(out.Accounts, accountRowToProto(row))
	}
	if len(rows) > 0 {
		out.NextId = rows[len(rows)-1].ID
	}
	return out, nil
}

// GetAccount returns one account and its audit trail.
func (s *AccountAdminServer) GetAccount(ctx context.Context, req *webv1.GetAccountRequest) (*webv1.GetAccountResponse, error) {
	res, row, audit, err := s.accounts.Get(ctx, req.GetModeratorId(), req.GetAccountId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get account: %v", err)
	}
	out := &webv1.GetAccountResponse{Result: accountResultToProto(res)}
	if res != accountadmin.OK {
		return out, nil
	}
	out.Account = accountRowToProto(row)
	out.Audit = make([]*webv1.AccountAuditRow, 0, len(audit))
	for _, entry := range audit {
		out.Audit = append(out.Audit, &webv1.AccountAuditRow{
			Id:            entry.ID,
			ActorId:       entry.ActorID,
			Action:        entry.Action,
			OldValue:      entry.OldValue,
			NewValue:      entry.NewValue,
			Reason:        entry.Reason,
			CreatedAtUnix: unixOrZero(entry.CreatedAt),
		})
	}
	return out, nil
}

// SetRole promotes or demotes an account.
func (s *AccountAdminServer) SetRole(ctx context.Context, req *webv1.SetRoleRequest) (*webv1.AccountWriteResponse, error) {
	res, row, message, err := s.accounts.SetRole(ctx, req.GetModeratorId(), req.GetAccountId(), req.GetRole(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set role: %v", err)
	}
	return accountWrite(res, row, message), nil
}

// SetBlocked blocks or unblocks an account.
func (s *AccountAdminServer) SetBlocked(ctx context.Context, req *webv1.SetBlockedRequest) (*webv1.AccountWriteResponse, error) {
	res, row, message, err := s.accounts.SetBlocked(ctx, req.GetModeratorId(), req.GetAccountId(), req.GetBlocked(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set blocked: %v", err)
	}
	return accountWrite(res, row, message), nil
}

// CreateAccount creates an account with an explicit role (admin only).
func (s *AccountAdminServer) CreateAccount(ctx context.Context, req *webv1.AdminCreateAccountRequest) (*webv1.AdminCreateAccountResponse, error) {
	res, id, message, err := s.accounts.Create(ctx, req.GetModeratorId(), req.GetName(), req.GetPassword(), req.GetEmail(), req.GetRole(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create account: %v", err)
	}
	return &webv1.AdminCreateAccountResponse{
		Result:    accountResultToProto(res),
		AccountId: id,
		Message:   message,
	}, nil
}

// --- mappers ---

func accountWrite(res accountadmin.Result, row domain.AccountRow, message string) *webv1.AccountWriteResponse {
	out := &webv1.AccountWriteResponse{
		Result:  accountResultToProto(res),
		Message: message,
	}
	if res == accountadmin.OK {
		out.Account = accountRowToProto(row)
	}
	return out
}

func accountRowToProto(row domain.AccountRow) *webv1.AccountRow {
	out := &webv1.AccountRow{
		Id:            row.ID,
		Name:          row.Name,
		Email:         row.Email,
		Role:          row.Role,
		IsBlocked:     row.IsBlocked,
		BlockedReason: row.BlockedReason,
		DonateBalance: row.DonateBalance,
		CreatedAtUnix: unixOrZero(row.CreatedAt),
		Characters:    row.Characters,
	}
	if row.RoleChangedAt != nil {
		out.RoleChangedAtUnix = unixOrZero(*row.RoleChangedAt)
	}
	if row.RoleChangedBy != nil {
		out.RoleChangedBy = *row.RoleChangedBy
	}
	return out
}

// accountResultToProto maps the service outcome to the shared AdminResult.
func accountResultToProto(r accountadmin.Result) webv1.AdminResult {
	switch r {
	case accountadmin.OK:
		return webv1.AdminResult_ADMIN_RESULT_OK
	case accountadmin.Forbidden:
		return webv1.AdminResult_ADMIN_RESULT_FORBIDDEN
	case accountadmin.NotFound:
		return webv1.AdminResult_ADMIN_RESULT_NOT_FOUND
	default:
		return webv1.AdminResult_ADMIN_RESULT_INVALID
	}
}
