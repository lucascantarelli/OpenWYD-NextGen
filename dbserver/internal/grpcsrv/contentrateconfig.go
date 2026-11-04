package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "github.com/jeanluca/w2pp-openwyd/api/db/v1"
	"github.com/jeanluca/w2pp-openwyd/internal/store"
)

// ContentRateConfigStore is the read surface the rate-overlay service needs
// (satisfied by *store.Store). tmServer only reads through here; writes go through
// the web-api, which owns the validation and the audit trail.
type ContentRateConfigStore interface {
	ListContentRateOverrides(ctx context.Context) ([]store.ContentRateOverride, error)
}

// ContentRateConfigServer implements dbv1.ContentRateConfigServiceServer.
type ContentRateConfigServer struct {
	dbv1.UnimplementedContentRateConfigServiceServer
	store ContentRateConfigStore
}

// NewContentRateConfig builds the rate config service over the given store.
func NewContentRateConfig(s ContentRateConfigStore) *ContentRateConfigServer {
	return &ContentRateConfigServer{store: s}
}

// ListContentRateOverrides returns the full overlay snapshot: one row per edited
// cell, which is a handful even on a tuned server.
func (s *ContentRateConfigServer) ListContentRateOverrides(ctx context.Context, _ *dbv1.ListContentRateOverridesRequest) (*dbv1.ListContentRateOverridesResponse, error) {
	rows, err := s.store.ListContentRateOverrides(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list content rate overrides: %v", err)
	}
	out := make([]*dbv1.ContentRateOverride, 0, len(rows))
	for _, r := range rows {
		out = append(out, &dbv1.ContentRateOverride{
			File:   r.File,
			Family: r.Family,
			Name:   r.Name,
			Value:  r.Value,
		})
	}
	return &dbv1.ListContentRateOverridesResponse{Rows: out}, nil
}
