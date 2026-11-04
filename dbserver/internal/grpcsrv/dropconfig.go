package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "github.com/jeanluca/w2pp-openwyd/api/db/v1"
	"github.com/jeanluca/w2pp-openwyd/internal/domain"
	"github.com/jeanluca/w2pp-openwyd/internal/store"
)

// DropConfigStore is the read surface the drop overlay needs (satisfied by
// *store.Store).
type DropConfigStore interface {
	ListDropRateOverrides(ctx context.Context) ([]store.DropRateOverride, error)
	ListMobCarryOverrides(ctx context.Context) ([]domain.MobCarryOverride, error)
}

// DropConfigServer implements dbv1.DropConfigServiceServer: tmServer fetches the
// moderator drop overlays once at boot, behind their own flags (issue 34).
// Read-only from tmServer's side.
type DropConfigServer struct {
	dbv1.UnimplementedDropConfigServiceServer
	store DropConfigStore
}

// NewDropConfig builds the drop config service over the given store.
func NewDropConfig(s DropConfigStore) *DropConfigServer {
	return &DropConfigServer{store: s}
}

// ListDropRateOverrides returns every edited Carry slot.
func (s *DropConfigServer) ListDropRateOverrides(ctx context.Context, _ *dbv1.ListDropRateOverridesRequest) (*dbv1.ListDropRateOverridesResponse, error) {
	rows, err := s.store.ListDropRateOverrides(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list drop rate overrides: %v", err)
	}
	out := make([]*dbv1.DropRateOverride, 0, len(rows))
	for _, r := range rows {
		out = append(out, &dbv1.DropRateOverride{Slot: int32(r.Slot), Value: r.Value})
	}
	return &dbv1.ListDropRateOverridesResponse{Rows: out}, nil
}

// ListMobCarryOverrides returns the Carry[] overlay of every edited template.
func (s *DropConfigServer) ListMobCarryOverrides(ctx context.Context, _ *dbv1.ListMobCarryOverridesRequest) (*dbv1.ListMobCarryOverridesResponse, error) {
	rows, err := s.store.ListMobCarryOverrides(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list mob carry overrides: %v", err)
	}
	out := make([]*dbv1.MobCarryOverride, 0, len(rows))
	for _, ov := range rows {
		slots := make([]*dbv1.MobCarrySlot, 0, len(ov.Slots))
		for _, it := range ov.Slots {
			slots = append(slots, &dbv1.MobCarrySlot{
				Slot:      int32(it.Slot),
				ItemIndex: it.ItemIndex,
				Eff1:      int32(it.Eff1), Effv1: int32(it.EffV1),
				Eff2: int32(it.Eff2), Effv2: int32(it.EffV2),
				Eff3: int32(it.Eff3), Effv3: int32(it.EffV3),
			})
		}
		out = append(out, &dbv1.MobCarryOverride{TemplateName: ov.TemplateName, Slots: slots})
	}
	return &dbv1.ListMobCarryOverridesResponse{Overrides: out}, nil
}
