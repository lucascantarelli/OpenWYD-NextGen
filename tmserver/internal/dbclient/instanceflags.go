package dbclient

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "github.com/jeanluca/w2pp-openwyd/api/db/v1"
)

// InstanceFlag is one process flag as the portal shows it (issue 36). The JSON
// shape is the contract with the portal, so the tags are part of the API.
type InstanceFlag struct {
	Name            string `json:"name"`
	Value           string `json:"value"`
	RestartRequired bool   `json:"restart_required"`
	Hint            string `json:"hint"`
}

// InstanceFlags records the boot snapshot of tmServer's process flags.
type InstanceFlags struct {
	api dbv1.InstanceFlagsServiceClient
}

// NewInstanceFlags wraps a gRPC connection as a flags recorder.
func NewInstanceFlags(conn grpc.ClientConnInterface) *InstanceFlags {
	return &InstanceFlags{api: dbv1.NewInstanceFlagsServiceClient(conn)}
}

// Record stores one snapshot. Marshalling lives here so callers pass typed flags
// and cannot send malformed JSON.
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
