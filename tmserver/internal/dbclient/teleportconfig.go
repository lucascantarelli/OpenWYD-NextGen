package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
)

// RemoteRoute is one teleport_route row as tmServer consumes it (issue 42).
type RemoteRoute struct {
	SrcX, SrcY  int16
	DstX, DstY  int16
	Cost        int32
	Requirement string
	Enabled     bool
}

// TeleportConfig adapts the dbServer TeleportConfigService: tmServer polls the
// overlay rows and publishes them into the world's read-only snapshot. Writes go
// through the web-api only — this client never mutates config.
type TeleportConfig struct {
	api dbv1.TeleportConfigServiceClient
}

// NewTeleportConfig wraps a gRPC connection as an overlay source.
func NewTeleportConfig(conn grpc.ClientConnInterface) *TeleportConfig {
	return &TeleportConfig{api: dbv1.NewTeleportConfigServiceClient(conn)}
}

// Routes fetches every overlay row (compiled-only routes are not listed here —
// tmServer already has them compiled in).
func (c *TeleportConfig) Routes(ctx context.Context) ([]RemoteRoute, error) {
	resp, err := c.api.ListTeleportRoutes(ctx, &dbv1.ListTeleportRoutesRequest{})
	if err != nil {
		return nil, fmt.Errorf("dbclient: list teleport routes: %w", err)
	}
	out := make([]RemoteRoute, 0, len(resp.GetRoutes()))
	for _, r := range resp.GetRoutes() {
		out = append(out, RemoteRoute{
			SrcX: int16(r.GetSrcX()), SrcY: int16(r.GetSrcY()),
			DstX: int16(r.GetDstX()), DstY: int16(r.GetDstY()),
			Cost: r.GetCost(), Requirement: r.GetRequirement(), Enabled: r.GetEnabled(),
		})
	}
	return out, nil
}
