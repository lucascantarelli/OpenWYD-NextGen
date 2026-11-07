package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/tmserver/internal/worldcfg"
)

// WorldEventConfig adapta o WorldEventConfigService do dbServer à porta
// worldcfg.Source do tmServer.
type WorldEventConfig struct {
	api dbv1.WorldEventConfigServiceClient
}

// NewWorldEventConfig embrulha uma conexão gRPC como fonte de config de evento de mundo.
func NewWorldEventConfig(conn grpc.ClientConnInterface) *WorldEventConfig {
	return &WorldEventConfig{api: dbv1.NewWorldEventConfigServiceClient(conn)}
}

var _ worldcfg.Source = (*WorldEventConfig)(nil)

// Version devolve a versão atual da config de evento de mundo gerida pelo portal.
func (c *WorldEventConfig) Version(ctx context.Context) (int64, error) {
	resp, err := c.api.WorldEventConfigVersion(ctx, &dbv1.WorldEventConfigVersionRequest{})
	if err != nil {
		return 0, fmt.Errorf("dbclient: world event config version: %w", err)
	}
	return resp.GetVersion(), nil
}

// Snapshot busca o snapshot completo da config de evento.
func (c *WorldEventConfig) Snapshot(ctx context.Context) (worldcfg.Snapshot, error) {
	resp, err := c.api.GetWorldEventConfig(ctx, &dbv1.GetWorldEventConfigRequest{})
	if err != nil {
		return worldcfg.Snapshot{}, fmt.Errorf("dbclient: get world event config: %w", err)
	}
	return worldcfg.Snapshot{Version: resp.GetVersion(), Event: dbWorldEventToConfig(resp.GetConfig())}, nil
}

// UpdateProgress persiste o contador atual do evento sem subir a versão da config.
func (c *WorldEventConfig) UpdateProgress(ctx context.Context, expectedVersion int64, currentIndex int32) (bool, error) {
	resp, err := c.api.UpdateWorldEventProgress(ctx, &dbv1.UpdateWorldEventProgressRequest{
		ExpectedVersion: expectedVersion,
		CurrentIndex:    currentIndex,
	})
	if err != nil {
		return false, fmt.Errorf("dbclient: update world event progress: %w", err)
	}
	return resp.GetApplied(), nil
}

func dbWorldEventToConfig(cfg *dbv1.WorldEventConfig) worldcfg.EventConfig {
	if cfg == nil {
		return worldcfg.EventConfig{NoticeEnabled: true}
	}
	return worldcfg.EventConfig{
		Enabled: cfg.GetEnabled(), ItemIndex: cfg.GetItemIndex(), Rate: cfg.GetRate(),
		StartIndex: cfg.GetStartIndex(), CurrentIndex: cfg.GetCurrentIndex(), EndIndex: cfg.GetEndIndex(),
		Indexed: cfg.GetIndexed(), NoticeEnabled: cfg.GetNoticeEnabled(),
		DoubleExpEnabled: cfg.GetDoubleExpEnabled(), NewbieEventEnabled: cfg.GetNewbieEventEnabled(),
	}
}
