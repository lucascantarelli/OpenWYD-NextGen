package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/internal/domain"
)

// TeleportConfigStore is the read surface the teleport-overlay service needs
// (satisfied by *store.Store). tmServer only reads through here; every write
// goes through the web-api.
type TeleportConfigStore interface {
	ListTeleportRoutes(ctx context.Context) ([]domain.TeleportRoute, error)
}

// TeleportConfigServer implements dbv1.TeleportConfigServiceServer: the
// read-only bridge tmServer polls to apply moderator-edited teleport routes
// (issue 42).
type TeleportConfigServer struct {
	dbv1.UnimplementedTeleportConfigServiceServer
	store TeleportConfigStore
}

// NewTeleportConfig builds the teleport-config service over the given store.
func NewTeleportConfig(s TeleportConfigStore) *TeleportConfigServer {
	return &TeleportConfigServer{store: s}
}

// ListTeleportRoutes returns the full overlay snapshot. There is no version
// field on purpose: the payload is a few dozen rows, so the 15s poll just
// re-reads it and the world swaps the snapshot atomically.
func (s *TeleportConfigServer) ListTeleportRoutes(ctx context.Context, _ *dbv1.ListTeleportRoutesRequest) (*dbv1.ListTeleportRoutesResponse, error) {
	routes, err := s.store.ListTeleportRoutes(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list teleport routes: %v", err)
	}
	out := make([]*dbv1.TeleportRoute, 0, len(routes))
	for _, r := range routes {
		out = append(out, &dbv1.TeleportRoute{
			SrcX: int32(r.SrcX), SrcY: int32(r.SrcY),
			DstX: int32(r.DstX), DstY: int32(r.DstY),
			Cost: int32(r.Cost), Requirement: r.Requirement,
			Enabled: r.Enabled, Note: r.Note,
		})
	}
	return &dbv1.ListTeleportRoutesResponse{Routes: out}, nil
}
