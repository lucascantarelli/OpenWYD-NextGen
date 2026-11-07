package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
)

// RemoteItemEffect é uma linha de item_effect_override como o tmServer a consome
// (issue 44): um valor que substitui aquele código de efeito naquele índice de item.
type RemoteItemEffect struct {
	ItemIndex int
	Effect    uint8
	Value     int16
}

// ItemEffectConfig adapta o ItemEffectConfigService do dbServer. As escritas vão
// só pelo web-api — este cliente nunca altera config.
type ItemEffectConfig struct {
	api dbv1.ItemEffectConfigServiceClient
}

// NewItemEffectConfig embrulha uma conexão gRPC como fonte de overlay.
func NewItemEffectConfig(conn grpc.ClientConnInterface) *ItemEffectConfig {
	return &ItemEffectConfig{api: dbv1.NewItemEffectConfigServiceClient(conn)}
}

// Effects busca toda linha do overlay (efeitos que o conteúdo já declara não
// aparecem aqui — o tmServer já os leu de ItemList.csv).
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
