package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "github.com/jeanluca/w2pp-openwyd/api/db/v1"
)

// RemoteItemEffect is one item_effect_override row as tmServer consumes it
// (issue 44): a value that replaces that effect code on that item index.
type RemoteItemEffect struct {
	ItemIndex int
	Effect    uint8
	Value     int16
}

// ItemEffectConfig adapts the dbServer ItemEffectConfigService. Writes go through
// the web-api only — this client never mutates config.
type ItemEffectConfig struct {
	api dbv1.ItemEffectConfigServiceClient
}

// NewItemEffectConfig wraps a gRPC connection as an overlay source.
func NewItemEffectConfig(conn grpc.ClientConnInterface) *ItemEffectConfig {
	return &ItemEffectConfig{api: dbv1.NewItemEffectConfigServiceClient(conn)}
}

// Effects fetches every overlay row (effects the content already declares are not
// listed here — tmServer already parsed them from ItemList.csv).
func (c *ItemEffectConfig) Effects(ctx context.Context) ([]RemoteItemEffect, error) {
	resp, err := c.api.ListItemEffectOverrides(ctx, &dbv1.ListItemEffectOverridesRequest{})
	if err != nil {
		return nil, fmt.Errorf("dbclient: list item effect overrides: %w", err)
	}
	out := make([]RemoteItemEffect, 0, len(resp.GetRows()))
	for _, r := range resp.GetRows() {
		out = append(out, RemoteItemEffect{
			ItemIndex: int(r.GetItemIndex()),
			Effect:    uint8(r.GetEffect()),
			Value:     int16(r.GetValue()),
		})
	}
	return out, nil
}
