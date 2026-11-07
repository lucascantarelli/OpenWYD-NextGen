package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/store"
)

// DropConfigStore é a superfície de leitura de que o overlay de drop precisa
// (satisfeita por *store.Store).
type DropConfigStore interface {
	ListDropRateOverrides(ctx context.Context) ([]store.DropRateOverride, error)
	ListMobCarryOverrides(ctx context.Context) ([]domain.MobCarryOverride, error)
}

// DropConfigServer implementa dbv1.DropConfigServiceServer: o tmServer busca os
// overlays de drop do moderador uma vez no boot, sob flags próprias (issue 34).
// Somente leitura do lado do tmServer.
type DropConfigServer struct {
	dbv1.UnimplementedDropConfigServiceServer
	store DropConfigStore
}

// NewDropConfig monta o serviço de configuração de drop sobre o store recebido.
func NewDropConfig(s DropConfigStore) *DropConfigServer {
	return &DropConfigServer{store: s}
}

// ListDropRateOverrides devolve todo slot de Carry editado.
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

// ListMobCarryOverrides devolve o overlay de Carry[] de cada template editado.
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
