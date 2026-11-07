package dbclient

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/tmserver/internal/npccfg"
)

// TemplateLoader resolve um nome de template de NPC para seu STRUCT_MOB cru de 816 bytes.
// O dbServer guarda só o NOME do template (não os bytes), então o tmServer
// o resolve contra sua árvore de conteúdo local no momento do snapshot.
type TemplateLoader func(name string) ([]byte, error)

// NpcConfig adapta o NpcConfigService do dbServer à porta npccfg.Source. Ele
// busca as definições editadas pelo moderador por gRPC e resolve cada nome de
// template para bytes, para o loop materializar o NPC direto.
type NpcConfig struct {
	api      dbv1.NpcConfigServiceClient
	template TemplateLoader
	log      *slog.Logger
}

// NewNpcConfig embrulha uma conexão gRPC + carregador de template como npccfg.Source. log
// recebe um aviso sempre que um template não resolve (nil cai no log padrão do
// slog), para um arquivo de NPC ausente/corrompido nunca sumir em silêncio.
func NewNpcConfig(conn grpc.ClientConnInterface, loader TemplateLoader, log *slog.Logger) *NpcConfig {
	if log == nil {
		log = slog.Default()
	}
	return &NpcConfig{api: dbv1.NewNpcConfigServiceClient(conn), template: loader, log: log}
}

var _ npccfg.Source = (*NpcConfig)(nil)

// Version devolve a versão atual da config (poll barato).
func (c *NpcConfig) Version(ctx context.Context) (int64, error) {
	resp, err := c.api.NpcConfigVersion(ctx, &dbv1.NpcConfigVersionRequest{})
	if err != nil {
		return 0, fmt.Errorf("dbclient: npc config version: %w", err)
	}
	return resp.GetVersion(), nil
}

// Snapshot busca o conjunto completo de definições + sobreposições de preço e resolve
// os bytes de template de cada definição. Template que falha ao carregar fica nil
// (o aplicador o pula) em vez de derrubar o reload inteiro.
func (c *NpcConfig) Snapshot(ctx context.Context) (npccfg.Snapshot, error) {
	resp, err := c.api.ListNpcDefinitions(ctx, &dbv1.ListNpcDefinitionsRequest{})
	if err != nil {
		return npccfg.Snapshot{}, fmt.Errorf("dbclient: list npc definitions: %w", err)
	}
	snap := npccfg.Snapshot{Version: resp.GetVersion()}
	cache := make(map[string][]byte)
	for _, d := range resp.GetDefinitions() {
		tmpl, ok := cache[d.GetTemplateName()]
		if !ok {
			b, lerr := c.template(d.GetTemplateName())
			if lerr != nil {
				// Não derruba o reload inteiro por um template ruim; deixa nil (o
				// aplicador pula e avisa), mas mostra o motivo aqui.
				c.log.Warn("npc template resolve failed", "template", d.GetTemplateName(), "slug", d.GetSlug(), "err", lerr)
			} else {
				tmpl = b
			}
			cache[d.GetTemplateName()] = tmpl
		}
		def := npccfg.Definition{
			Slug:        d.GetSlug(),
			Template:    tmpl,
			DisplayName: d.GetDisplayName(),
			Enabled:     d.GetEnabled(),
			X:           int16(d.GetPosX()),
			Y:           int16(d.GetPosY()),
			RouteType:   uint8(d.GetRouteType()),
			Merchant:    uint8(d.GetMerchant()),
			Origin:      d.GetOrigin(), GeneratorIndex: int(d.GetGeneratorIndex()),
			MinuteGenerate: int(d.GetMinuteGenerate()), MinGroup: int(d.GetMinGroup()),
			MaxGroup: int(d.GetMaxGroup()), MaxNumMob: int(d.GetMaxNumMob()), Formation: int(d.GetFormation()),
		}
		if name := d.GetFollowerTemplate(); name != "" {
			if b, lerr := c.template(name); lerr != nil {
				c.log.Warn("npc follower template resolve failed", "template", name, "slug", d.GetSlug(), "err", lerr)
			} else {
				def.FollowerTemplate = b
			}
		}
		for i := 0; i < 5; i++ {
			if i < len(d.GetSegX()) {
				def.SegX[i] = int16(d.GetSegX()[i])
			}
			if i < len(d.GetSegY()) {
				def.SegY[i] = int16(d.GetSegY()[i])
			}
			if i < len(d.GetSegRange()) {
				def.SegRange[i] = int16(d.GetSegRange()[i])
			}
			if i < len(d.GetSegWait()) {
				def.SegWait[i] = int16(d.GetSegWait()[i])
			}
		}
		copy(def.FightAction[:], d.GetFightAction())
		copy(def.DieAction[:], d.GetDieAction())
		for _, it := range d.GetShop() {
			def.Shop = append(def.Shop, npccfg.ShopItem{
				Slot:     int(it.GetSlot()),
				Index:    uint16(it.GetItemIndex()),
				Quantity: uint8(it.GetQuantity()),
				Eff: [3][2]uint8{
					{uint8(it.GetEff1()), uint8(it.GetEffv1())},
					{uint8(it.GetEff2()), uint8(it.GetEffv2())},
					{uint8(it.GetEff3()), uint8(it.GetEffv3())},
				},
			})
		}
		snap.Defs = append(snap.Defs, def)
	}
	if prices := resp.GetPriceOverrides(); len(prices) > 0 {
		snap.PriceOverrides = make(map[int]int32, len(prices))
		for _, p := range prices {
			snap.PriceOverrides[int(p.GetItemIndex())] = int32(p.GetPrice())
		}
	}
	return snap, nil
}
