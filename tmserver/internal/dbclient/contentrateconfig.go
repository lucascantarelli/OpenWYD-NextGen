package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
)

// RemoteRateOverride é uma linha de content_rate_override como o tmServer a consome
// (issue 36): o valor substitui o do arquivo em (file, family, name).
type RemoteRateOverride struct {
	File   string
	Family string
	Name   string
	Value  int64
}

// ContentRateConfig adapta o ContentRateConfigService do dbServer. As escritas vão
// vão só pelo web-api — este cliente nunca altera config.
type ContentRateConfig struct {
	api dbv1.ContentRateConfigServiceClient
}

// NewContentRateConfig embrulha uma conexão gRPC como fonte de overlay de rates.
func NewContentRateConfig(conn grpc.ClientConnInterface) *ContentRateConfig {
	return &ContentRateConfig{api: dbv1.NewContentRateConfigServiceClient(conn)}
}

// Overrides busca toda linha do overlay (células sem linha mantêm o valor do arquivo).
func (c *ContentRateConfig) Overrides(ctx context.Context) ([]RemoteRateOverride, error) {
	resp, err := c.api.ListContentRateOverrides(ctx, &dbv1.ListContentRateOverridesRequest{})
	if err != nil {
		return nil, fmt.Errorf("dbclient: list content rate overrides: %w", err)
	}
	out := make([]RemoteRateOverride, 0, len(resp.GetRows()))
	for _, r := range resp.GetRows() {
		out = append(out, RemoteRateOverride{
			File: r.GetFile(), Family: r.GetFamily(), Name: r.GetName(), Value: r.GetValue(),
		})
	}
	return out, nil
}
