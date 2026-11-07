package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
)

// RemoteRoute é uma linha de teleport_route como o tmServer a consome (issue 42).
type RemoteRoute struct {
	SrcX, SrcY  int16
	DstX, DstY  int16
	Cost        int32
	Requirement string
	Enabled     bool
}

// TeleportConfig adapta o TeleportConfigService do dbServer: o tmServer consulta as
// linhas do overlay e as publica no snapshot somente leitura do mundo. As escritas
// só pelo web-api — este cliente nunca altera config.
type TeleportConfig struct {
	api dbv1.TeleportConfigServiceClient
}

// NewTeleportConfig embrulha uma conexão gRPC como fonte de overlay.
func NewTeleportConfig(conn grpc.ClientConnInterface) *TeleportConfig {
	return &TeleportConfig{api: dbv1.NewTeleportConfigServiceClient(conn)}
}

// Routes busca toda linha do overlay (rotas só compiladas não aparecem aqui —
// o tmServer já as tem compiladas).
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
