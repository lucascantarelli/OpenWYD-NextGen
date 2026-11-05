package grpcsrv

import (
	"context"

	webv1 "openwyd-nextgen/api/web/v1"
	"openwyd-nextgen/internal/itemeffects"
	"openwyd-nextgen/webserver/internal/itemcatalog"
)

// ItemCatalogServer implements webv1.ItemCatalogServiceServer.
//
// It holds the catalog directly rather than going through a service layer:
// unlike the admin surfaces there is nothing to authorize and no store to
// query, just an immutable slice scanned from the read-only content mount at
// boot (webserver/cmd/webserver/main.go).
type ItemCatalogServer struct {
	webv1.UnimplementedItemCatalogServiceServer
	version         string
	iconPackVersion string
	// entries is built once in NewItemCatalog and never mutated afterwards: the
	// catalog is immutable after boot, so rebuilding ~3200 messages (~400 KB) per
	// call would be pure waste on an endpoint that takes no arguments and no
	// authorization. Every response shares this slice — gRPC only marshals it.
	entries []*webv1.ItemCatalogEntry
	// effectTypes is the ItemEffect.h vocabulary, built once for the same reason.
	effectTypes []*webv1.ItemEffectType
}

// NewItemCatalog builds the ItemCatalogService over an already-scanned catalog.
// A zero-valued catalog (no -content configured) is valid and serves an empty
// list.
func NewItemCatalog(catalog itemcatalog.Catalog) *ItemCatalogServer {
	entries := make([]*webv1.ItemCatalogEntry, 0, len(catalog.Items))
	for _, it := range catalog.Items {
		entries = append(entries, itemCatalogEntryToProto(it))
	}
	types := make([]*webv1.ItemEffectType, 0, 128)
	for _, t := range itemeffects.All() {
		types = append(types, &webv1.ItemEffectType{Code: t.Code, Name: t.Name, Label: t.Label})
	}
	return &ItemCatalogServer{
		version:         catalog.Version,
		iconPackVersion: catalog.IconPackVersion,
		entries:         entries,
		effectTypes:     types,
	}
}

// ListItems returns the whole catalog. It cannot fail: the content was read at
// boot, and its absence is a legitimate degraded mode rather than an error.
func (s *ItemCatalogServer) ListItems(_ context.Context, _ *webv1.ListItemsRequest) (*webv1.ListItemsResponse, error) {
	return &webv1.ListItemsResponse{
		Items:           s.entries,
		CatalogVersion:  s.version,
		IconPackVersion: s.iconPackVersion,
		EffectTypes:     s.effectTypes,
	}, nil
}

// itemCatalogEntryToProto is shared with NpcAdminService.ListItemCatalog so the
// moderator picker and the player-facing list can never drift apart.
func itemCatalogEntryToProto(e itemcatalog.Entry) *webv1.ItemCatalogEntry {
	effects := make([]*webv1.AdminItemEffect, 0, len(e.Effects))
	for _, effect := range e.Effects {
		label := itemeffects.Label(effect.Code)
		if label == "" {
			label = "—"
		}
		effects = append(effects, &webv1.AdminItemEffect{Code: effect.Code, Label: label, Value: effect.Value})
	}
	return &webv1.ItemCatalogEntry{
		Effects:     effects,
		ItemIndex:   e.Index,
		Name:        e.Name,
		IconKey:     e.IconKey,
		DisplayName: e.DisplayName,
		SlotMask:    e.SlotMask,
		Slots:       e.Slots,
		Grade:       e.Grade,
		Mesh:        e.Mesh,
		Texture:     e.Texture,
		IconUrl:     e.IconURL,
	}
}
