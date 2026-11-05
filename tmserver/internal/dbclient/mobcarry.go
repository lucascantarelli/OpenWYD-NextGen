package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/tmserver/internal/mobcarry"
)

// MobCarrySource fetches the moderator-edited Carry[] overlays (issue 34, fatia
// 3) from the dbServer's DropConfigService. Fetched once at boot — no version
// poll, no hot-reload: the table is baked into the template bytes the world
// spawns from.
type MobCarrySource struct {
	api dbv1.DropConfigServiceClient
}

// NewMobCarrySource wraps a gRPC connection as a carry overlay source.
func NewMobCarrySource(conn grpc.ClientConnInterface) *MobCarrySource {
	return &MobCarrySource{api: dbv1.NewDropConfigServiceClient(conn)}
}

// Fetch returns every template's carry override, keyed by template_name.
func (c *MobCarrySource) Fetch(ctx context.Context) (map[string]mobcarry.Override, error) {
	resp, err := c.api.ListMobCarryOverrides(ctx, &dbv1.ListMobCarryOverridesRequest{})
	if err != nil {
		return nil, fmt.Errorf("dbclient: list mob carry overrides: %w", err)
	}
	out := make(map[string]mobcarry.Override, len(resp.GetOverrides()))
	for _, ov := range resp.GetOverrides() {
		slots := make([]mobcarry.Slot, 0, len(ov.GetSlots()))
		for _, s := range ov.GetSlots() {
			slots = append(slots, mobcarry.Slot{
				Slot:      int(s.GetSlot()),
				ItemIndex: uint16(s.GetItemIndex()),
				Eff: [3][2]uint8{
					{uint8(s.GetEff1()), uint8(s.GetEffv1())},
					{uint8(s.GetEff2()), uint8(s.GetEffv2())},
					{uint8(s.GetEff3()), uint8(s.GetEffv3())},
				},
			})
		}
		out[ov.GetTemplateName()] = mobcarry.Override{Slots: slots}
	}
	return out, nil
}
