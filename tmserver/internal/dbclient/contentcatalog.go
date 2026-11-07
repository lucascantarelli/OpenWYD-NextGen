package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/tmserver/internal/content"
)

// ContentCatalog é o snapshot de boot com que o tmServer monta o mundo
// (issue 58, fase 2). Cada campo tem exatamente o formato que os carregadores de
// arquivo devolviam, então o código do mundo não muda — só de onde vêm os bytes.
type ContentCatalog struct {
	Items      []content.ItemEntry
	Skills     []content.Spell
	Templates  map[string][]byte // nome do template → STRUCT_MOB canônico de 816 bytes
	WorldItems []content.InitItem
}

// ContentCatalogClient busca os catálogos estáticos do dbServer. É separado do
// Client porque embrulha outro serviço gRPC (conteúdo somente leitura).
type ContentCatalogClient struct {
	api dbv1.ContentCatalogServiceClient
}

// NewContentCatalog embrulha uma conexão gRPC como fonte de catálogo de conteúdo.
func NewContentCatalog(conn grpc.ClientConnInterface) *ContentCatalogClient {
	return &ContentCatalogClient{api: dbv1.NewContentCatalogServiceClient(conn)}
}

// Fetch puxa os quatro catálogos. O tmServer o chama uma vez, antes de o loop do
// mundo começar; uma falha é reportada a quem chama para ele cair nos arquivos
// de conteúdo em vez de subir com um mundo vazio.
func (c *ContentCatalogClient) Fetch(ctx context.Context) (ContentCatalog, error) {
	out := ContentCatalog{Templates: make(map[string][]byte)}

	items, err := c.api.FetchItemCatalog(ctx, &dbv1.FetchItemCatalogRequest{})
	if err != nil {
		return ContentCatalog{}, fmt.Errorf("dbclient: fetch item catalog: %w", err)
	}
	for _, it := range items.GetItems() {
		out.Items = append(out.Items, content.ItemEntry{
			Index:  int(it.GetItemIndex()),
			Name:   it.GetName(),
			Fields: it.GetFields(),
		})
	}

	skills, err := c.api.FetchSkillCatalog(ctx, &dbv1.FetchSkillCatalogRequest{})
	if err != nil {
		return ContentCatalog{}, fmt.Errorf("dbclient: fetch skill catalog: %w", err)
	}
	for _, sp := range skills.GetSkills() {
		out.Skills = append(out.Skills, content.Spell{
			Index:             int(sp.GetSkillIndex()),
			SkillPoint:        int(sp.GetSkillPoint()),
			TargetType:        int(sp.GetTargetType()),
			ManaSpent:         int(sp.GetManaSpent()),
			Delay:             int(sp.GetDelayTicks()),
			Range:             int(sp.GetRangeTiles()),
			InstanceType:      int(sp.GetInstanceType()),
			InstanceValue:     int(sp.GetInstanceValue()),
			TickType:          int(sp.GetTickType()),
			TickValue:         int(sp.GetTickValue()),
			AffectType:        int(sp.GetAffectType()),
			AffectValue:       int(sp.GetAffectValue()),
			AffectTime:        int(sp.GetAffectTime()),
			InstanceAttribute: int(sp.GetInstanceAttribute()),
			TickAttribute:     int(sp.GetTickAttribute()),
			Aggressive:        int(sp.GetAggressive()),
			MaxTarget:         int(sp.GetMaxTarget()),
			BParty:            int(sp.GetParty()),
			AffectResist:      int(sp.GetAffectResist()),
			Passive:           int(sp.GetPassive()),
			Name:              sp.GetName(),
		})
	}

	tpls, err := c.api.FetchNPCTemplates(ctx, &dbv1.FetchNPCTemplatesRequest{})
	if err != nil {
		return ContentCatalog{}, fmt.Errorf("dbclient: fetch npc templates: %w", err)
	}
	for _, t := range tpls.GetTemplates() {
		out.Templates[t.GetTemplateName()] = t.GetBlob()
	}

	worldItems, err := c.api.FetchWorldItems(ctx, &dbv1.FetchWorldItemsRequest{})
	if err != nil {
		return ContentCatalog{}, fmt.Errorf("dbclient: fetch world items: %w", err)
	}
	for _, it := range worldItems.GetItems() {
		out.WorldItems = append(out.WorldItems, content.InitItem{
			Index:  int16(it.GetItemIndex()),
			PosX:   int16(it.GetPosX()),
			PosY:   int16(it.GetPosY()),
			Rotate: int16(it.GetRotate()),
		})
	}
	return out, nil
}
