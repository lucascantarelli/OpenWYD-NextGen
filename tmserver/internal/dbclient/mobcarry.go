package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/tmserver/internal/mobcarry"
)

// MobCarrySource busca os overlays de Carry[] editados pelo moderador (issue 34,
// fatia 3) do DropConfigService do dbServer. Buscado uma vez no boot — sem poll
// de versão, sem hot-reload: a tabela é embutida nos bytes de template de que o
// mundo nasce.
type MobCarrySource struct {
	api dbv1.DropConfigServiceClient
}

// NewMobCarrySource embrulha uma conexão gRPC como fonte de overlay de carry.
func NewMobCarrySource(conn grpc.ClientConnInterface) *MobCarrySource {
	return &MobCarrySource{api: dbv1.NewDropConfigServiceClient(conn)}
}

// Fetch devolve o override de carry de cada template, com chave template_name.
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
