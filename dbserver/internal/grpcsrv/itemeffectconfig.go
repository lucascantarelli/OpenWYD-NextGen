package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "github.com/jeanluca/w2pp-openwyd/api/db/v1"
	"github.com/jeanluca/w2pp-openwyd/internal/store"
)

// ItemEffectConfigStore is the read surface the item-effect overlay service needs
// (satisfied by *store.Store). tmServer only reads through here; every write goes
// through the web-api, which also owns the audit trail.
type ItemEffectConfigStore interface {
	ListItemEffectOverrides(ctx context.Context, itemIndex int32) ([]store.ItemEffectOverride, error)
}

// ItemEffectConfigServer implements dbv1.ItemEffectConfigServiceServer: the
// read-only bridge tmServer fetches once at boot to apply moderator-edited item
// base effects (issue 44).
type ItemEffectConfigServer struct {
	dbv1.UnimplementedItemEffectConfigServiceServer
	store ItemEffectConfigStore
}

// NewItemEffectConfig builds the item-effect config service over the given store.
func NewItemEffectConfig(s ItemEffectConfigStore) *ItemEffectConfigServer {
	return &ItemEffectConfigServer{store: s}
}

// ListItemEffectOverrides returns the full overlay snapshot. Like the teleport
// bridge there is no version field: the payload is one row per edited effect (a
// handful at most), so a single fetch at boot is enough.
func (s *ItemEffectConfigServer) ListItemEffectOverrides(ctx context.Context, _ *dbv1.ListItemEffectOverridesRequest) (*dbv1.ListItemEffectOverridesResponse, error) {
	// itemIndex 0 means "every row" in the store query.
	rows, err := s.store.ListItemEffectOverrides(ctx, 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list item effect overrides: %v", err)
	}
	out := make([]*dbv1.ItemEffectOverride, 0, len(rows))
	for _, r := range rows {
		out = append(out, &dbv1.ItemEffectOverride{
			ItemIndex: r.ItemIndex,
			Effect:    int32(r.Effect),
			Value:     int32(r.Value),
		})
	}
	return &dbv1.ListItemEffectOverridesResponse{Rows: out}, nil
}
