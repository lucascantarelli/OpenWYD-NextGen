package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
)

// RemoteRateOverride is one content_rate_override row as tmServer consumes it
// (issue 36): the value replaces the file's at (file, family, name).
type RemoteRateOverride struct {
	File   string
	Family string
	Name   string
	Value  int64
}

// ContentRateConfig adapts the dbServer ContentRateConfigService. Writes go
// through the web-api only — this client never mutates config.
type ContentRateConfig struct {
	api dbv1.ContentRateConfigServiceClient
}

// NewContentRateConfig wraps a gRPC connection as a rate overlay source.
func NewContentRateConfig(conn grpc.ClientConnInterface) *ContentRateConfig {
	return &ContentRateConfig{api: dbv1.NewContentRateConfigServiceClient(conn)}
}

// Overrides fetches every overlay row (cells without a row keep the file value).
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
