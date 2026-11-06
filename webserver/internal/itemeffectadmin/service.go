// Package itemeffectadmin is the web-api side of the item base-effect editor
// (issue 44, metade de escrita): it merges the adds read from
// tmserver/content/Common/ItemList.csv with the item_effect_override overlay, validates
// every write against ItemEffect.h, and never touches tmserver/content/ or live game
// state.
//
// A chave da edição é o código do efeito, não uma posição: a quantidade de pares
// `EF_*` varia por item (a 1188 tem seis) e o servidor os indexa por código
// (internal/content.BaseEffects). tmserver/content/ é conteúdo: reescrever o CSV pela web
// faria o próximo reimport sobrescrever a edição em silêncio.
package itemeffectadmin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"openwyd-nextgen/internal/itemeffects"
	"openwyd-nextgen/internal/store"
	"openwyd-nextgen/webserver/internal/itemcatalog"
)

// MaxValue bounds one value: STRUCT_EFFECT stores it as a signed short.
const MaxValue = 32767

// Store is the persistence surface the service needs (satisfied by *store.Store).
type Store interface {
	AccountRole(ctx context.Context, id int64) (string, error)
	ListItemEffectOverrides(ctx context.Context, itemIndex int32) ([]store.ItemEffectOverride, error)
	UpsertItemEffectOverride(ctx context.Context, o store.ItemEffectOverride, moderatorID int64, reason string) error
	DeleteItemEffectOverride(ctx context.Context, itemIndex int32, effect int, moderatorID int64, reason string) (bool, error)
}

// Result is the business outcome of an admin operation. Only infra failures are
// returned as errors; these ride in the response body (web.proto AdminResult).
type Result int

const (
	// OK means the operation succeeded.
	OK Result = iota
	// Forbidden means the caller is not a moderator/admin.
	Forbidden
	// Invalid means the request failed validation.
	Invalid
	// NotFound means the item is not in the catalog, or the effect has no overlay
	// row to remove.
	NotFound
)

// Effect is one base add of the item after the overlay is applied. FromContent
// says the CSV declares this effect; HasOverride says the value came from the web.
type Effect struct {
	Effect      int32
	Value       int32
	FromContent bool
	HasOverride bool
	UpdatedBy   int64
	UpdatedAt   time.Time
}

// Service implements the moderator item-effect operations.
type Service struct {
	store Store
	// base resolves an item index to its catalog adds. Nil means no content tree
	// was configured, so every edit is rejected as NotFound instead of writing
	// effects onto an item that may not exist.
	base func(index int32) ([]itemcatalog.Effect, bool)
}

// New builds the service over the given store.
func New(s Store) *Service { return &Service{store: s} }

// SetCatalog wires the scanned catalog in. Kept as a setter because the catalog
// is optional: without -content the service still starts and answers NotFound.
func (s *Service) SetCatalog(catalog itemcatalog.Catalog) {
	byIndex := make(map[int32][]itemcatalog.Effect, len(catalog.Items))
	for _, item := range catalog.Items {
		byIndex[item.Index] = item.Effects
	}
	s.base = func(index int32) ([]itemcatalog.Effect, bool) {
		effects, ok := byIndex[index]
		return effects, ok
	}
}

// List returns the item's adds merged with the overlay, content order first and
// any extra overlay-only effect appended by code. Two overrides of the same code
// cannot happen (the table is keyed by it), so the result has one row per effect.
func (s *Service) List(ctx context.Context, moderatorID int64, itemIndex int32) (Result, []Effect, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, nil, "", err
	}
	base, ok := s.baseFor(itemIndex)
	if !ok {
		return NotFound, nil, "item fora do catálogo: confira o índice", nil
	}
	overlay, err := s.store.ListItemEffectOverrides(ctx, itemIndex)
	if err != nil {
		return Invalid, nil, "", fmt.Errorf("itemeffectadmin: list: %w", err)
	}

	out := make([]Effect, 0, len(base)+len(overlay))
	seen := make(map[int32]bool, len(base)+len(overlay))
	for _, eff := range base {
		row := Effect{Effect: eff.Code, Value: eff.Value, FromContent: true}
		for _, o := range overlay {
			if o.Effect == int(eff.Code) {
				row.Value = int32(o.Value)
				row.HasOverride = true
				row.UpdatedBy = o.UpdatedBy
				row.UpdatedAt = o.UpdatedAt
				break
			}
		}
		seen[eff.Code] = true
		out = append(out, row)
	}
	extras := make([]Effect, 0, len(overlay))
	for _, o := range overlay {
		if seen[int32(o.Effect)] {
			continue
		}
		extras = append(extras, Effect{
			Effect: int32(o.Effect), Value: int32(o.Value),
			HasOverride: true, UpdatedBy: o.UpdatedBy, UpdatedAt: o.UpdatedAt,
		})
	}
	sort.Slice(extras, func(i, j int) bool { return extras[i].Effect < extras[j].Effect })
	return OK, append(out, extras...), "", nil
}

// Upsert validates and stores one effect value. The code must exist in
// ItemEffect.h: the client cannot interpret an unknown code, and a save with one
// would surface only as a broken item in game. Adding a code the content does not
// declare is allowed — it is how a moderator gives an item a new add.
func (s *Service) Upsert(ctx context.Context, moderatorID int64, itemIndex int32, effect, value int32, reason string) (Result, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, "", err
	}
	if _, ok := s.baseFor(itemIndex); !ok {
		return NotFound, "item fora do catálogo: confira o índice", nil
	}
	if msg := validate(effect, value); msg != "" {
		return Invalid, msg, nil
	}
	row := store.ItemEffectOverride{ItemIndex: itemIndex, Effect: int(effect), Value: int(value)}
	if err := s.store.UpsertItemEffectOverride(ctx, row, moderatorID, reason); err != nil {
		return Invalid, "", fmt.Errorf("itemeffectadmin: upsert: %w", err)
	}
	return OK, "", nil
}

// Delete drops the overlay row, restoring the catalog value for that effect.
func (s *Service) Delete(ctx context.Context, moderatorID int64, itemIndex int32, effect int32, reason string) (Result, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, "", err
	}
	if !itemeffects.Known(effect) {
		return Invalid, fmt.Sprintf("efeito %d não existe em ItemEffect.h", effect), nil
	}
	applied, err := s.store.DeleteItemEffectOverride(ctx, itemIndex, int(effect), moderatorID, reason)
	if err != nil {
		return Invalid, "", fmt.Errorf("itemeffectadmin: delete: %w", err)
	}
	if !applied {
		return NotFound, "este efeito não tem edição na web: já vale o conteúdo", nil
	}
	return OK, "", nil
}

// baseFor resolves the item's catalog adds; ok=false means "not in the catalog".
func (s *Service) baseFor(index int32) ([]itemcatalog.Effect, bool) {
	if s.base == nil {
		return nil, false
	}
	return s.base(index)
}

// validate returns a moderator-facing message when the write is invalid.
func validate(effect, value int32) string {
	if !itemeffects.Known(effect) {
		return fmt.Sprintf("efeito %d não existe em ItemEffect.h", effect)
	}
	if value < -MaxValue || value > MaxValue {
		return fmt.Sprintf("valor fora de -%d..%d (o campo é um short)", MaxValue, MaxValue)
	}
	return ""
}

func (s *Service) authorize(ctx context.Context, moderatorID int64) (Result, error) {
	if moderatorID <= 0 {
		return Forbidden, nil
	}
	role, err := s.store.AccountRole(ctx, moderatorID)
	if errors.Is(err, store.ErrNotFound) {
		return Forbidden, nil
	}
	if err != nil {
		return Invalid, fmt.Errorf("itemeffectadmin: role lookup %d: %w", moderatorID, err)
	}
	if role != "moderator" && role != "admin" {
		return Forbidden, nil
	}
	return OK, nil
}
