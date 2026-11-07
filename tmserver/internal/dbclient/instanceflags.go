package dbclient

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
)

// InstanceFlag é uma flag de processo como o portal a mostra (issue 36). O formato
// JSON é o contrato com o portal, então as tags fazem parte da API.
type InstanceFlag struct {
	Name            string `json:"name"`
	Value           string `json:"value"`
	RestartRequired bool   `json:"restart_required"`
	Hint            string `json:"hint"`
}

// InstanceFlags registra o snapshot de boot das flags de processo do tmServer.
type InstanceFlags struct {
	api dbv1.InstanceFlagsServiceClient
}

// NewInstanceFlags embrulha uma conexão gRPC como gravador de flags.
func NewInstanceFlags(conn grpc.ClientConnInterface) *InstanceFlags {
	return &InstanceFlags{api: dbv1.NewInstanceFlagsServiceClient(conn)}
}

// Record guarda um snapshot. A serialização mora aqui para quem chama passar flags
// tipadas e não poder enviar JSON malformado.
func (c *InstanceFlags) Record(ctx context.Context, server string, flags []InstanceFlag) error {
	raw, err := json.Marshal(flags)
	if err != nil {
		return fmt.Errorf("dbclient: marshal instance flags: %w", err)
	}
	if _, err := c.api.RecordInstanceFlags(ctx, &dbv1.RecordInstanceFlagsRequest{
		Server:    server,
		FlagsJson: string(raw),
	}); err != nil {
		return fmt.Errorf("dbclient: record instance flags: %w", err)
	}
	return nil
}
