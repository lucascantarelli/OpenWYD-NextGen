// Package dropadmin is the moderator drop-editing surface (issue 34), the write
// side of the overlays tmServer reads at boot behind its editing flags.
//
// Two tables are edited here:
//
//   - DropRate[64], the divisor per Carry slot. The slot decides an item's
//     rarity, not the item itself; the base value shown in the UI is read from
//     the same table the world uses (internal/droprate).
//   - Carry[64] of a mob template, the drop table itself. Only the item index
//     comes from the browser: effects are preserved server-side for a slot whose
//     item did not change, and a new item starts with none. Ao contrário do
//     editor de Equip, o carry não exige override de stats — é uma tabela
//     independente, aplicada no carregamento do template.
package dropadmin

import (
	"context"
	"errors"
	"fmt"

	"github.com/jeanluca/w2pp-openwyd/internal/domain"
	"github.com/jeanluca/w2pp-openwyd/internal/droprate"
	"github.com/jeanluca/w2pp-openwyd/internal/savefmt"
	"github.com/jeanluca/w2pp-openwyd/internal/store"
	"github.com/jeanluca/w2pp-openwyd/webserver/internal/itemcatalog"
	"github.com/jeanluca/w2pp-openwyd/webserver/internal/npctemplates"
)

// Store is the persistence surface this service needs (satisfied by *store.Store).
type Store interface {
	AccountRole(ctx context.Context, id int64) (string, error)
	ListDropRateOverrides(ctx context.Context) ([]store.DropRateOverride, error)
	UpsertDropRateOverride(ctx context.Context, slot int, value int64, moderatorID int64, reason string) error
	DeleteDropRateOverride(ctx context.Context, slot int, moderatorID int64, reason string) (bool, error)
	GetMobCarryOverride(ctx context.Context, templateName string) (domain.MobCarryOverride, error)
	SetMobCarry(ctx context.Context, templateName string, slots []domain.MobTemplateEquipItem, moderatorID int64, reason string) error
	DeleteMobCarry(ctx context.Context, templateName string, moderatorID int64, reason string) (bool, error)
}

// Result is the business outcome of an admin operation. Only infra failures are
// returned as errors; these ride in the response body.
type Result int

const (
	// OK means the operation succeeded.
	OK Result = iota
	// Forbidden means the caller is not a moderator/admin.
	Forbidden
	// Invalid means the request failed validation.
	Invalid
	// NotFound means there was no overlay row to reset, or no template to read.
	NotFound
)

// MaxRate mirrors the CHECK on drop_rate_override.value; tmServer uses the same
// constant (internal/droprate.MaxRate), so both sides reject the same range.
const MaxRate = droprate.MaxRate

// maxCarryIndex bounds STRUCT_ITEM.Index (int16) — the same ceiling the carry
// overlay column accepts.
const maxCarryIndex = 32767

// Slot is one Carry slot as the rate editor sees it: the code value, the
// effective value and whether an overlay row exists.
type Slot struct {
	Slot       int
	Base       int
	Current    int
	Overridden bool
}

// CarrySlot is one Carry[] position of a template, with the item resolved to its
// catalog name so the screen never shows a bare index.
type CarrySlot struct {
	Slot         int
	ItemIndex    int
	ItemName     string
	Eff          [3][2]uint8
	FromOverride bool
}

// Carry is one template's drop table as the editor reads it.
type Carry struct {
	TemplateName string
	DisplayName  string
	Level        int
	Overridden   bool
	Slots        []CarrySlot
}

// CarryInput is one slot of a write: the position and the item only.
type CarryInput struct {
	Slot      int
	ItemIndex int
}

// Service implements the moderator drop operations.
type Service struct {
	store     Store
	readRaw   func(templateName string) ([]byte, error)
	itemNames map[int32]string
}

// New builds the service over the given store.
func New(s Store) *Service { return &Service{store: s} }

// SetTemplateReader installs the read-through template reader ListCarry and the
// write validation use. Left unset when -content wasn't configured, in which
// case the carry editor reports NotFound instead of inventing a table.
func (s *Service) SetTemplateReader(r func(templateName string) ([]byte, error)) { s.readRaw = r }

// SetItemCatalog installs the catalog used to validate item indices and to label
// the slots with names. Without it (no -content) the editor cannot validate, so
// writes are rejected as Invalid — never written blind.
func (s *Service) SetItemCatalog(catalog itemcatalog.Catalog) {
	s.itemNames = make(map[int32]string, len(catalog.Items))
	for _, it := range catalog.Items {
		s.itemNames[it.Index] = it.DisplayName
	}
}

// List returns all 64 rate slots, ordered by slot, with base/current/overridden.
func (s *Service) List(ctx context.Context, moderatorID int64) (Result, []Slot, error) {
	if r, err := s.authorize(ctx, moderatorID); r != OK || err != nil {
		return r, nil, err
	}
	rows, err := s.store.ListDropRateOverrides(ctx)
	if err != nil {
		return Invalid, nil, fmt.Errorf("dropadmin: list overrides: %w", err)
	}
	out := make([]Slot, 0, len(droprate.Rate))
	for slot, base := range droprate.Rate {
		out = append(out, Slot{Slot: slot, Base: base, Current: base})
	}
	for _, r := range rows {
		if r.Slot < 0 || r.Slot >= len(out) {
			continue // row out of band: the tmServer ignores it too
		}
		out[r.Slot].Current = int(r.Value)
		out[r.Slot].Overridden = true
	}
	return OK, out, nil
}

// Upsert stores one rate slot divisor. reason is audit-only and may be empty.
func (s *Service) Upsert(ctx context.Context, moderatorID int64, slot int, value int64, reason string) (Result, error) {
	if r, err := s.authorize(ctx, moderatorID); r != OK || err != nil {
		return r, err
	}
	if !validSlot(slot) || value < 1 || value > MaxRate {
		return Invalid, nil
	}
	if err := s.store.UpsertDropRateOverride(ctx, slot, value, moderatorID, reason); err != nil {
		return Invalid, fmt.Errorf("dropadmin: upsert slot %d: %w", slot, err)
	}
	return OK, nil
}

// Reset removes the rate overlay row, restoring the code value. NotFound when the
// slot had no override.
func (s *Service) Reset(ctx context.Context, moderatorID int64, slot int, reason string) (Result, error) {
	if r, err := s.authorize(ctx, moderatorID); r != OK || err != nil {
		return r, err
	}
	if !validSlot(slot) {
		return Invalid, nil
	}
	applied, err := s.store.DeleteDropRateOverride(ctx, slot, moderatorID, reason)
	if err != nil {
		return Invalid, fmt.Errorf("dropadmin: reset slot %d: %w", slot, err)
	}
	if !applied {
		return NotFound, nil
	}
	return OK, nil
}

// ListCarry returns one template's Carry[] merged with its overlay. A template
// with no overlay still lists the file's own slots (Overridden=false), so the
// editor opens on real values instead of zeros.
func (s *Service) ListCarry(ctx context.Context, moderatorID int64, templateName string) (Result, Carry, error) {
	if r, err := s.authorize(ctx, moderatorID); r != OK || err != nil {
		return r, Carry{}, err
	}
	if templateName == "" {
		return Invalid, Carry{}, nil
	}
	carry, found, err := s.currentCarry(ctx, templateName)
	if err != nil {
		return Invalid, Carry{}, err
	}
	if !found {
		return NotFound, Carry{}, nil
	}
	return OK, carry, nil
}

// SetCarry replaces the template's whole Carry[] table. The item is the only
// value taken from the request: for a slot whose item did not change the current
// effects are kept, so opening and saving without touching a slot cannot erase
// the adds of a rare drop; a newly placed item starts with none.
func (s *Service) SetCarry(ctx context.Context, moderatorID int64, templateName string, items []CarryInput, reason string) (Result, error) {
	if r, err := s.authorize(ctx, moderatorID); r != OK || err != nil {
		return r, err
	}
	if templateName == "" || len(s.itemNames) == 0 || s.readRaw == nil {
		return Invalid, nil
	}
	current, found, err := s.currentCarry(ctx, templateName)
	if err != nil {
		return Invalid, err
	}
	if !found {
		return Invalid, nil // template inexistente: nunca criar override órfão
	}
	bySlot := make(map[int]CarrySlot, len(current.Slots))
	for _, slot := range current.Slots {
		bySlot[slot.Slot] = slot
	}

	seen := make(map[int]bool, len(items))
	out := make([]domain.MobTemplateEquipItem, 0, len(items))
	for _, in := range items {
		if in.Slot < 0 || in.Slot >= savefmt.MaxCarry || seen[in.Slot] {
			return Invalid, nil
		}
		if in.ItemIndex < 1 || in.ItemIndex > maxCarryIndex {
			return Invalid, nil
		}
		if _, ok := s.itemNames[int32(in.ItemIndex)]; !ok {
			return Invalid, nil // item fora do catálogo
		}
		seen[in.Slot] = true

		var eff [3][2]uint8
		if prev, ok := bySlot[in.Slot]; ok && prev.ItemIndex == in.ItemIndex {
			eff = prev.Eff
		}
		out = append(out, domain.MobTemplateEquipItem{
			Slot:      int16(in.Slot),
			ItemIndex: int32(in.ItemIndex),
			Eff1:      eff[0][0], EffV1: eff[0][1],
			Eff2: eff[1][0], EffV2: eff[1][1],
			Eff3: eff[2][0], EffV3: eff[2][1],
		})
	}
	if err := s.store.SetMobCarry(ctx, templateName, out, moderatorID, reason); err != nil {
		return Invalid, fmt.Errorf("dropadmin: set carry %q: %w", templateName, err)
	}
	return OK, nil
}

// ResetCarry removes the template's carry overlay, returning it to the file.
func (s *Service) ResetCarry(ctx context.Context, moderatorID int64, templateName, reason string) (Result, error) {
	if r, err := s.authorize(ctx, moderatorID); r != OK || err != nil {
		return r, err
	}
	if templateName == "" {
		return Invalid, nil
	}
	applied, err := s.store.DeleteMobCarry(ctx, templateName, moderatorID, reason)
	if err != nil {
		return Invalid, fmt.Errorf("dropadmin: reset carry %q: %w", templateName, err)
	}
	if !applied {
		return NotFound, nil
	}
	return OK, nil
}

// currentCarry reads the file's Carry[] and overlays the stored rows. found is
// false only when the template itself could not be read AND no overlay exists
// (no content tree, missing file or corrupt record) — in that case there is
// nothing honest to show.
func (s *Service) currentCarry(ctx context.Context, templateName string) (Carry, bool, error) {
	out := Carry{TemplateName: templateName, Slots: make([]CarrySlot, savefmt.MaxCarry)}
	for i := range out.Slots {
		out.Slots[i].Slot = i
	}

	readable := false
	if s.readRaw != nil {
		if raw, err := s.readRaw(templateName); err == nil {
			if mob, _, derr := savefmt.DecodeMobAny(raw); derr == nil {
				readable = true
				out.DisplayName = npctemplates.CString(mob.Name[:])
				out.Level = int(mob.CurrentScore.Level)
				for i := 0; i < savefmt.MaxCarry; i++ {
					it := mob.Carry[i]
					out.Slots[i].ItemIndex = int(it.Index)
					out.Slots[i].Eff = [3][2]uint8{
						{it.Effects[0].Effect, it.Effects[0].Value},
						{it.Effects[1].Effect, it.Effects[1].Value},
						{it.Effects[2].Effect, it.Effects[2].Value},
					}
				}
			}
		}
	}

	ov, err := s.store.GetMobCarryOverride(ctx, templateName)
	hasOverride := err == nil
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return Carry{}, false, fmt.Errorf("dropadmin: get carry %q: %w", templateName, err)
	}
	if hasOverride {
		out.Overridden = true
		out.Slots = make([]CarrySlot, savefmt.MaxCarry)
		for i := range out.Slots {
			out.Slots[i] = CarrySlot{Slot: i, FromOverride: true}
		}
		for _, it := range ov.Slots {
			if it.Slot < 0 || it.Slot >= savefmt.MaxCarry {
				continue
			}
			out.Slots[it.Slot] = CarrySlot{
				Slot:      int(it.Slot),
				ItemIndex: int(it.ItemIndex),
				Eff: [3][2]uint8{
					{it.Eff1, it.EffV1},
					{it.Eff2, it.EffV2},
					{it.Eff3, it.EffV3},
				},
				FromOverride: true,
			}
		}
	}

	if !readable && !hasOverride {
		return Carry{}, false, nil
	}

	for i := range out.Slots {
		out.Slots[i].ItemName = s.itemName(out.Slots[i].ItemIndex)
	}
	return out, true, nil
}

func (s *Service) itemName(index int) string {
	if index == 0 {
		return ""
	}
	if name, ok := s.itemNames[int32(index)]; ok {
		return name
	}
	return fmt.Sprintf("item #%d", index)
}

func validSlot(slot int) bool { return slot >= 0 && slot < len(droprate.Rate) }

// authorize checks the caller has a moderator/admin role. A missing account or a
// plain-player role yields Forbidden (never leaks whether the account exists).
func (s *Service) authorize(ctx context.Context, moderatorID int64) (Result, error) {
	if moderatorID <= 0 {
		return Forbidden, nil
	}
	role, err := s.store.AccountRole(ctx, moderatorID)
	if errors.Is(err, store.ErrNotFound) {
		return Forbidden, nil
	}
	if err != nil {
		return Invalid, fmt.Errorf("dropadmin: role lookup %d: %w", moderatorID, err)
	}
	if role != "moderator" && role != "admin" {
		return Forbidden, nil
	}
	return OK, nil
}
