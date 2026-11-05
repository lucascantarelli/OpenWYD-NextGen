package grpcsrv

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webv1 "openwyd-nextgen/api/web/v1"
	"openwyd-nextgen/webserver/internal/ratesadmin"
)

// RatesAdmin is the rate surface the server depends on (satisfied by
// *ratesadmin.Service).
type RatesAdmin interface {
	List(ctx context.Context, moderatorID int64) (ratesadmin.Result, []ratesadmin.RowView, string, error)
	Upsert(ctx context.Context, moderatorID int64, file, family, name string, value int64, reason string) (ratesadmin.Result, string, error)
	Reset(ctx context.Context, moderatorID int64, file, family, name, reason string) (ratesadmin.Result, string, error)
	Flags(ctx context.Context, moderatorID int64, server string) (ratesadmin.Result, []ratesadmin.FlagView, time.Time, bool, error)
}

// ContentRatesServer implements webv1.ContentRatesServiceServer.
type ContentRatesServer struct {
	webv1.UnimplementedContentRatesServiceServer
	admin RatesAdmin
}

// NewContentRates builds the ContentRatesService over the given logic.
func NewContentRates(a RatesAdmin) *ContentRatesServer {
	return &ContentRatesServer{admin: a}
}

// GetRates returns every cell of the three tables (file value + overlay).
func (s *ContentRatesServer) GetRates(ctx context.Context, req *webv1.GetRatesRequest) (*webv1.GetRatesResponse, error) {
	res, views, warning, err := s.admin.List(ctx, req.GetModeratorId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get rates: %v", err)
	}
	out := make([]*webv1.AdminRate, 0, len(views))
	for _, view := range views {
		out = append(out, &webv1.AdminRate{
			File:        view.Row.File,
			Family:      view.Row.Family,
			Name:        view.Row.Name,
			Value:       view.Row.Value,
			Original:    view.Original,
			MinValue:    view.Row.Min,
			MaxValue:    view.Row.Max,
			Label:       view.Row.Label,
			HasOverride: view.HasOverride,
		})
	}
	return &webv1.GetRatesResponse{
		Result:    ratesResultToProto(res),
		Rates:     out,
		ContentOk: len(views) > 0,
		Warning:   warning,
	}, nil
}

// UpsertRate stores one edited cell.
func (s *ContentRatesServer) UpsertRate(ctx context.Context, req *webv1.UpsertRateRequest) (*webv1.SetRateResponse, error) {
	res, message, err := s.admin.Upsert(ctx, req.GetModeratorId(), req.GetFile(), req.GetFamily(),
		req.GetName(), req.GetValue(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert rate: %v", err)
	}
	return &webv1.SetRateResponse{Result: ratesResultToProto(res), Message: message}, nil
}

// ResetRate drops the overlay row for one cell.
func (s *ContentRatesServer) ResetRate(ctx context.Context, req *webv1.ResetRateRequest) (*webv1.SetRateResponse, error) {
	res, message, err := s.admin.Reset(ctx, req.GetModeratorId(), req.GetFile(), req.GetFamily(),
		req.GetName(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reset rate: %v", err)
	}
	return &webv1.SetRateResponse{Result: ratesResultToProto(res), Message: message}, nil
}

// GetServerFlags returns the flags snapshot the service recorded on its last boot.
func (s *ContentRatesServer) GetServerFlags(ctx context.Context, req *webv1.GetServerFlagsRequest) (*webv1.GetServerFlagsResponse, error) {
	res, views, captured, available, err := s.admin.Flags(ctx, req.GetModeratorId(), "tmserver")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get server flags: %v", err)
	}
	flags := make([]*webv1.AdminFlag, 0, len(views))
	for _, v := range views {
		flags = append(flags, &webv1.AdminFlag{
			Name: v.Name, Value: v.Value, RestartRequired: v.RestartRequired, Hint: v.Hint,
		})
	}
	out := &webv1.GetServerFlagsResponse{
		Result:    ratesResultToProto(res),
		Flags:     flags,
		Available: available,
	}
	if available && !captured.IsZero() {
		out.CapturedAt = captured.UTC().Format(time.RFC3339)
	}
	if !available {
		out.Message = "o tmServer ainda não gravou um snapshot de flags (precisa de -dbserver no boot)"
	}
	return out, nil
}

func ratesResultToProto(r ratesadmin.Result) webv1.AdminResult {
	switch r {
	case ratesadmin.OK:
		return webv1.AdminResult_ADMIN_RESULT_OK
	case ratesadmin.Forbidden:
		return webv1.AdminResult_ADMIN_RESULT_FORBIDDEN
	case ratesadmin.NotFound:
		return webv1.AdminResult_ADMIN_RESULT_NOT_FOUND
	default:
		return webv1.AdminResult_ADMIN_RESULT_INVALID
	}
}
