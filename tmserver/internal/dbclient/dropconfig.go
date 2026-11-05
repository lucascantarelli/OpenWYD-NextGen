package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
)

// RemoteDropRate is one drop_rate_override row as tmServer consumes it
// (issue 34): the divisor replaces the code value for that Carry slot.
type RemoteDropRate struct {
	Slot  int
	Value int64
}

// DropConfig adapts the dbServer DropConfigService. Writes go through the
// web-api only — this client never mutates config.
type DropConfig struct {
	api dbv1.DropConfigServiceClient
}

// NewDropConfig wraps a gRPC connection as a drop-rate overlay source.
func NewDropConfig(conn grpc.ClientConnInterface) *DropConfig {
	return &DropConfig{api: dbv1.NewDropConfigServiceClient(conn)}
}

// Overrides fetches every overlay row (slots without a row keep the code value).
func (c *DropConfig) Overrides(ctx context.Context) ([]RemoteDropRate, error) {
	resp, err := c.api.ListDropRateOverrides(ctx, &dbv1.ListDropRateOverridesRequest{})
	if err != nil {
		return nil, fmt.Errorf("dbclient: list drop rate overrides: %w", err)
	}
	out := make([]RemoteDropRate, 0, len(resp.GetRows()))
	for _, r := range resp.GetRows() {
		out = append(out, RemoteDropRate{Slot: int(r.GetSlot()), Value: r.GetValue()})
	}
	return out, nil
}
