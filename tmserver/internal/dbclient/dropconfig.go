package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
)

// RemoteDropRate é uma linha de drop_rate_override como o tmServer a consome
// (issue 34): o divisor substitui o valor do código para aquele slot de Carry.
type RemoteDropRate struct {
	Slot  int
	Value int64
}

// DropConfig adapta o DropConfigService do dbServer. As escritas vão pelo
// web-api — este cliente nunca altera config.
type DropConfig struct {
	api dbv1.DropConfigServiceClient
}

// NewDropConfig embrulha uma conexão gRPC como fonte de overlay de taxa de drop.
func NewDropConfig(conn grpc.ClientConnInterface) *DropConfig {
	return &DropConfig{api: dbv1.NewDropConfigServiceClient(conn)}
}

// Overrides busca toda linha do overlay (slots sem linha mantêm o valor do código).
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
