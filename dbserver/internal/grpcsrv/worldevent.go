package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/internal/domain"
)

// WorldEventConfigStore é a superfície de leitura/progresso de que o tmServer
// precisa (satisfeita por *store.Store). Escritas do moderador vão pelo web-api.
type WorldEventConfigStore interface {
	WorldEventConfigVersion(ctx context.Context) (int64, error)
	WorldEventConfig(ctx context.Context) (domain.WorldEventConfig, error)
	UpdateWorldEventProgress(ctx context.Context, expectedVersion int64, currentIndex int32) (bool, error)
}

// WorldEventConfigServer implementa dbv1.WorldEventConfigServiceServer.
type WorldEventConfigServer struct {
	dbv1.UnimplementedWorldEventConfigServiceServer
	store WorldEventConfigStore
}

// NewWorldEventConfig monta o serviço sobre o store recebido.
func NewWorldEventConfig(s WorldEventConfigStore) *WorldEventConfigServer {
	return &WorldEventConfigServer{store: s}
}

// WorldEventConfigVersion devolve a versão monotônica da config para o tmServer.
func (s *WorldEventConfigServer) WorldEventConfigVersion(ctx context.Context, _ *dbv1.WorldEventConfigVersionRequest) (*dbv1.WorldEventConfigVersionResponse, error) {
	v, err := s.store.WorldEventConfigVersion(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "world event config version: %v", err)
	}
	return &dbv1.WorldEventConfigVersionResponse{Version: v}, nil
}

// GetWorldEventConfig devolve o snapshot completo da config e sua versão.
func (s *WorldEventConfigServer) GetWorldEventConfig(ctx context.Context, _ *dbv1.GetWorldEventConfigRequest) (*dbv1.GetWorldEventConfigResponse, error) {
	version, err := s.store.WorldEventConfigVersion(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "world event config version: %v", err)
	}
	cfg, err := s.store.WorldEventConfig(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "world event config: %v", err)
	}
	return &dbv1.GetWorldEventConfigResponse{Version: version, Config: worldEventConfigToDBProto(cfg)}, nil
}

// UpdateWorldEventProgress persiste o contador vivo de evento do tmServer.
func (s *WorldEventConfigServer) UpdateWorldEventProgress(ctx context.Context, req *dbv1.UpdateWorldEventProgressRequest) (*dbv1.UpdateWorldEventProgressResponse, error) {
	applied, err := s.store.UpdateWorldEventProgress(ctx, req.GetExpectedVersion(), req.GetCurrentIndex())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update world event progress: %v", err)
	}
	return &dbv1.UpdateWorldEventProgressResponse{Applied: applied}, nil
}

func worldEventConfigToDBProto(cfg domain.WorldEventConfig) *dbv1.WorldEventConfig {
	return &dbv1.WorldEventConfig{
		Enabled: cfg.Enabled, ItemIndex: cfg.ItemIndex, Rate: cfg.Rate,
		StartIndex: cfg.StartIndex, CurrentIndex: cfg.CurrentIndex, EndIndex: cfg.EndIndex,
		Indexed: cfg.Indexed, NoticeEnabled: cfg.NoticeEnabled,
		DoubleExpEnabled: cfg.DoubleExpEnabled, NewbieEventEnabled: cfg.NewbieEventEnabled,
	}
}
