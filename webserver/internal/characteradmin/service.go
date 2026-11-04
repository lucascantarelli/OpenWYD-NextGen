// Package characteradmin is the moderator-facing character inspector (issue 43).
//
// It is READ-ONLY on purpose: the web never writes live character state, and the
// mutations this issue eventually needs (grant item/coin, field edits) have to go
// through delivery_queue or an offline-only field editor. Until that lands, this
// service only answers "what does this character look like in cold storage?".
package characteradmin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jeanluca/w2pp-openwyd/internal/domain"
	"github.com/jeanluca/w2pp-openwyd/internal/itemeffects"
	"github.com/jeanluca/w2pp-openwyd/internal/store"
	"github.com/jeanluca/w2pp-openwyd/webserver/internal/itemcatalog"
	"github.com/jeanluca/w2pp-openwyd/webserver/internal/skillcatalog"
)

// MinQueryLength avoids scanning the whole character table for a single letter.
const MinQueryLength = 2

// MaxResults caps one search response; the UI paginates client-side.
const MaxResults = 50

// Grant limits. Quantity is capped at 255 because one mailbox row is written per
// unit (delivery_queue.payload has no quantity field); the reason is required so
// every grant is explainable from the audit trail alone.
const (
	MaxGrantQuantity = 255
	MinReasonLength  = 3
)

// Store is the persistence surface the service needs (satisfied by *store.Store).
type Store interface {
	AccountRole(ctx context.Context, id int64) (string, error)
	SearchCharacters(ctx context.Context, query string, limit int) ([]store.CharacterSearchHit, error)
	LoadCharacterByID(ctx context.Context, id int64) (domain.Character, int64, string, error)
	EnqueueItemGrant(ctx context.Context, moderatorID, accountID, characterID int64, item domain.Item, quantity int, source, reason string) ([]int64, error)
	PendingItemGrants(ctx context.Context, characterID int64, limit int) ([]store.CharacterItemGrant, error)
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
	// NotFound means the character does not exist.
	NotFound
)

// Detail is one character's cold state plus its identity.
//
// CharacterID is carried explicitly because domain.Character is keyed by
// (account_id, slot) — it has no id of its own, and the UI needs the character
// id to address grants.
type Detail struct {
	CharacterID int64
	Character   domain.Character
	AccountID   int64
	AccountName string
	// Items resolves an item index to its catalog name/icon, so the UI never
	// shows a bare "#1234". Missing entries mean the index is not in ItemList.csv.
	Items map[int16]ItemInfo
	// SkillNames maps a skill index to its SkillData.csv name, for the learned
	// mask and the hotbar only (the full catalog belongs to issue 46).
	SkillNames map[int32]string
	// Adds maps an effect code (STRUCT_ITEM.Eff1..3) to its label, so the UI
	// renders "Dano +70" instead of "2:70". Codes outside ItemEffect.h are
	// absent and the UI falls back to the number.
	Adds map[int32]string
}

// ItemInfo is the catalog face of an item index.
type ItemInfo struct {
	Name    string
	IconURL string
}

// Service implements the moderator character queries.
type Service struct {
	store  Store
	items  itemcatalog.Catalog
	skills skillcatalog.Catalog
}

// New builds the service over the given store.
func New(s Store) *Service { return &Service{store: s} }

// SetItemCatalog installs the item catalog scanned at boot (itemcatalog.Scan).
// Left unset when the web-api runs without -content, in which case the UI falls
// back to the raw index.
func (s *Service) SetItemCatalog(catalog itemcatalog.Catalog) { s.items = catalog }

// SetSkillCatalog installs the skill catalog scanned at boot (skillcatalog.Scan).
// Left unset when the web-api runs without -content; the UI then keeps the raw
// skill index.
func (s *Service) SetSkillCatalog(catalog skillcatalog.Catalog) { s.skills = catalog }

// Search finds characters by name or account name.
func (s *Service) Search(ctx context.Context, moderatorID int64, query string) (Result, []store.CharacterSearchHit, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, nil, err
	}
	q := strings.TrimSpace(query)
	if len([]rune(q)) < MinQueryLength {
		return Invalid, nil, nil
	}
	hits, err := s.store.SearchCharacters(ctx, q, MaxResults)
	if err != nil {
		return Invalid, nil, fmt.Errorf("characteradmin: search: %w", err)
	}
	return OK, hits, nil
}

// Get loads one character's status and inventory.
func (s *Service) Get(ctx context.Context, moderatorID, characterID int64) (Result, Detail, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, Detail{}, err
	}
	if characterID <= 0 {
		return Invalid, Detail{}, nil
	}
	ch, accountID, accountName, err := s.store.LoadCharacterByID(ctx, characterID)
	if errors.Is(err, store.ErrNotFound) {
		return NotFound, Detail{}, nil
	}
	if err != nil {
		return Invalid, Detail{}, fmt.Errorf("characteradmin: get %d: %w", characterID, err)
	}
	return OK, Detail{
		CharacterID: characterID, Character: ch,
		AccountID: accountID, AccountName: accountName,
		Items:      s.resolveItems(ch),
		SkillNames: s.resolveSkills(ch),
		Adds:       resolveAdds(ch),
	}, nil
}

// Grant enqueues `quantity` units of an item for one character and records the
// action in character_admin_audit. It never touches the character row: the
// tmServer drains the mailbox at login, so this is safe for online characters.
func (s *Service) Grant(ctx context.Context, moderatorID, characterID int64, item domain.Item, quantity int, reason string) (Result, []int64, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, nil, err
	}
	note := strings.TrimSpace(reason)
	if characterID <= 0 || item.Index <= 0 {
		return Invalid, nil, nil
	}
	if quantity < 1 || quantity > MaxGrantQuantity {
		return Invalid, nil, nil
	}
	if len([]rune(note)) < MinReasonLength {
		return Invalid, nil, nil
	}
	if item.Eff1 > 0 && item.EffV1 == 0 {
		return Invalid, nil, nil
	}

	_, accountID, _, err := s.store.LoadCharacterByID(ctx, characterID)
	if errors.Is(err, store.ErrNotFound) {
		return NotFound, nil, nil
	}
	if err != nil {
		return Invalid, nil, fmt.Errorf("characteradmin: grant target %d: %w", characterID, err)
	}

	ids, err := s.store.EnqueueItemGrant(ctx, moderatorID, accountID, characterID, item, quantity,
		fmt.Sprintf("admin:%d", moderatorID), note)
	if err != nil {
		return Invalid, nil, fmt.Errorf("characteradmin: grant: %w", err)
	}
	return OK, ids, nil
}

// ListGrants returns the character's mailbox rows, newest first.
func (s *Service) ListGrants(ctx context.Context, moderatorID, characterID int64) (Result, []store.CharacterItemGrant, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, nil, err
	}
	if characterID <= 0 {
		return Invalid, nil, nil
	}
	grants, err := s.store.PendingItemGrants(ctx, characterID, 20)
	if err != nil {
		return Invalid, nil, fmt.Errorf("characteradmin: list grants: %w", err)
	}
	return OK, grants, nil
}

// resolveItems maps the character's equipped/carried indexes to catalog names.
// It only answers for indexes actually on the character, so a 64-slot cargo does
// not drag the whole 3.2k catalog over gRPC.
func (s *Service) resolveItems(ch domain.Character) map[int16]ItemInfo {
	if len(s.items.Items) == 0 {
		return nil
	}
	byIndex := make(map[int32]itemcatalog.Entry, len(s.items.Items))
	for _, entry := range s.items.Items {
		byIndex[entry.Index] = entry
	}
	out := make(map[int16]ItemInfo, len(ch.Equip)+len(ch.Carry))
	for _, it := range append(append([]domain.Item{}, ch.Equip...), ch.Carry...) {
		entry, ok := byIndex[int32(it.Index)]
		if !ok {
			continue
		}
		name := entry.DisplayName
		if name == "" {
			name = entry.Name
		}
		out[it.Index] = ItemInfo{Name: name, IconURL: entry.IconURL}
	}
	return out
}

// ValidateAdds rejects an effect code outside ItemEffect.h, so a grant never
// enqueues an item whose add the client cannot interpret. Zero means "no add"
// and is always fine.
func ValidateAdds(item domain.Item) bool {
	for _, code := range [3]uint8{item.Eff1, item.Eff2, item.Eff3} {
		if code != 0 && !itemeffects.Known(int32(code)) {
			return false
		}
	}
	return true
}

// resolveAdds labels the effect codes carried by the character's items. It only
// answers for codes present on the save, so the response carries a handful of
// entries instead of the whole ItemEffect.h vocabulary.
func resolveAdds(ch domain.Character) map[int32]string {
	out := make(map[int32]string, 8)
	add := func(it domain.Item) {
		for _, code := range [3]uint8{it.Eff1, it.Eff2, it.Eff3} {
			if code == 0 {
				continue
			}
			if label := itemeffects.Label(int32(code)); label != "" {
				out[int32(code)] = label
			}
		}
	}
	for _, it := range ch.Equip {
		add(it)
	}
	for _, it := range ch.Carry {
		add(it)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// resolveSkills names the skills the character actually has: the learned mask
// and the four hotbar entries. Everything else stays out of the response.
func (s *Service) resolveSkills(ch domain.Character) map[int32]string {
	if len(s.skills.Skills) == 0 {
		return nil
	}
	byIndex := s.skills.ByIndex()
	out := make(map[int32]string, 32)
	mask := uint32(ch.LearnedSkill)
	for bit := 0; bit < 32; bit++ {
		if mask&(1<<uint(bit)) == 0 {
			continue
		}
		if entry, ok := byIndex[int32(bit)]; ok {
			out[int32(bit)] = entry.DisplayName
		}
	}
	for _, idx := range ch.SkillBar {
		if entry, ok := byIndex[int32(idx)]; ok {
			out[int32(idx)] = entry.DisplayName
		}
	}
	return out
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
		return Invalid, fmt.Errorf("characteradmin: role lookup %d: %w", moderatorID, err)
	}
	if role != "moderator" && role != "admin" {
		return Forbidden, nil
	}
	return OK, nil
}
