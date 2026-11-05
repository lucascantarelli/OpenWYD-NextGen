package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/tmserver/internal/content"
)

// ContentCatalog is the boot-time snapshot tmServer builds the world from
// (issue 58, fase 2). Each field is exactly the shape the file loaders used to
// return, so the world code does not change — only where the bytes come from.
type ContentCatalog struct {
	Items      []content.ItemEntry
	Skills     []content.Spell
	Templates  map[string][]byte // template name → canonical 816-byte STRUCT_MOB
	WorldItems []content.InitItem
}

// ContentCatalogClient fetches the static catalogs from dbServer. It is separate
// from Client because it wraps a different gRPC service (read-only content).
type ContentCatalogClient struct {
	api dbv1.ContentCatalogServiceClient
}

// NewContentCatalog wraps a gRPC connection as a content-catalog source.
func NewContentCatalog(conn grpc.ClientConnInterface) *ContentCatalogClient {
	return &ContentCatalogClient{api: dbv1.NewContentCatalogServiceClient(conn)}
}

// Fetch pulls all four catalogs. The tmServer calls it once, before the world
// loop starts; a failure is reported to the caller so it can fall back to the
// content files instead of booting with an empty world.
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
