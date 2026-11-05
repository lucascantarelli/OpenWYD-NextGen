package handler

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/jeanluca/w2pp-openwyd/tmserver/internal/npccfg"
	"github.com/jeanluca/w2pp-openwyd/tmserver/internal/protocol"
	"github.com/jeanluca/w2pp-openwyd/tmserver/internal/world"
)

// Moderator-edited NPC overlay (npc-editing-plan.md). The web writes cold config
// to Postgres; the tmServer reads it (via dbServer) and MATERIALIZES the live
// entities inside the single-owner loop — the web never touches live state.
//
// The database is the single source of spawn blocks (issue 57): the tmServer no
// longer parses NPCGener.txt. At boot it registers one generator slot per content
// definition and materializes the entities from the same snapshot, so the file and
// the table can no longer disagree. Event-owned blocks (water rooms, Kefra,
// Nightmare, rune quest) are registered but never materialized here — their event
// code spawns them.

const (
	// npcPollPeriod is how many mob-AI ticks (1s each) between config version
	// polls. Moderator edits are rare, so ~15s keeps the DB load negligible while
	// still feeling live.
	npcPollPeriod = 15
	// npcFetchTimeout bounds the off-loop version/snapshot gRPC calls.
	npcFetchTimeout = 5 * time.Second
)

// Boot retry budget. Vars, not consts, so tests can shrink them.
var (
	// npcBootRetryBudget bounds how long boot waits for the dbServer to become
	// reachable and finish reconciling the content catalog. Both services are
	// deployed together and the dbServer seeds the catalog BEFORE it listens, so
	// the tmServer legitimately wins the race; without this wait the deploy
	// crash-loops (production incident, 2026-08-13).
	npcBootRetryBudget = 90 * time.Second
	// npcBootRetryInitial and npcBootRetryMax bracket the exponential backoff.
	npcBootRetryInitial = 1 * time.Second
	npcBootRetryMax     = 5 * time.Second
)

// FetchNPCConfig loads the spawn catalogue before the loop starts. Transient
// failures are retried until npcBootRetryBudget expires — the dbServer may still be
// unreachable, or still mid-seed and answering with an empty catalogue (it seeds
// the whole catalogue BEFORE it listens: production incident, 2026-08-13). Corrupt
// data (duplicate generator index) never heals on its own, so it fails immediately
// instead of burning the budget.
func (d *Dispatcher) FetchNPCConfig(ctx context.Context) (npccfg.Snapshot, error) {
	if d.npcSource == nil {
		return npccfg.Snapshot{}, nil
	}
	deadline := time.Now().Add(npcBootRetryBudget)
	delay := npcBootRetryInitial
	for attempt := 1; ; attempt++ {
		snap, retriable, err := d.fetchNPCConfigAttempt(ctx)
		if err == nil {
			return snap, nil
		}
		if !retriable {
			return npccfg.Snapshot{}, err
		}
		if time.Now().After(deadline) {
			return npccfg.Snapshot{}, fmt.Errorf("gave up after %d attempts in %s: %w", attempt, npcBootRetryBudget, err)
		}
		d.log.Warn("npc config boot load failed, retrying", "attempt", attempt, "retry_in", delay, "err", err)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return npccfg.Snapshot{}, fmt.Errorf("npc config boot load: %w", ctx.Err())
		case <-timer.C:
		}
		delay = min(delay*2, npcBootRetryMax)
	}
}

// fetchNPCConfigAttempt runs one boot-load attempt. retriable reports whether the
// failure is the kind that clears once the dbServer finishes coming up.
func (d *Dispatcher) fetchNPCConfigAttempt(ctx context.Context) (snap npccfg.Snapshot, retriable bool, err error) {
	fetchCtx, cancel := context.WithTimeout(ctx, npcFetchTimeout)
	defer cancel()
	snap, err = d.npcSource.Snapshot(fetchCtx)
	if err != nil {
		return npccfg.Snapshot{}, true, fmt.Errorf("npc config boot load: %w", err)
	}
	seen := make(map[int]struct{}, len(snap.Defs))
	for _, def := range snap.Defs {
		if def.Origin != "content" || def.GeneratorIndex < 0 || world.EventOwnedGenerator(def.GeneratorIndex) {
			continue
		}
		if _, exists := seen[def.GeneratorIndex]; exists {
			return npccfg.Snapshot{}, false, fmt.Errorf("npc config duplicate generator index %d", def.GeneratorIndex)
		}
		seen[def.GeneratorIndex] = struct{}{}
	}
	if len(seen) == 0 {
		// An empty catalogue means we read it before the seed committed — not that
		// this shard is supposed to have no monsters.
		return npccfg.Snapshot{}, true, errors.New("npc config catalogue is empty")
	}
	return snap, false, nil
}

// ApplyNPCConfigBoot materializes the fetched snapshot before the loop starts (no
// reveal — no players are connected yet). Generator slots were registered from this
// same snapshot, so a definition without a slot is a wiring bug: fail closed rather
// than booting a world with holes.
func (d *Dispatcher) ApplyNPCConfigBoot(w *world.World, snap npccfg.Snapshot) error {
	for _, def := range snap.Defs {
		if !def.Enabled || def.Origin != "content" || def.GeneratorIndex < 0 {
			continue
		}
		if w.GeneratorAt(def.GeneratorIndex) == nil {
			return fmt.Errorf("npc config content generator index %d (%s) has no registered slot", def.GeneratorIndex, def.Slug)
		}
	}
	d.applyNPCConfig(w, snap, false)
	d.log.Info("npc config applied at boot", "version", snap.Version, "npcs", len(d.managedNPCs))
	return nil
}

// pollNPCConfig checks the config version off the loop each npcPollPeriod ticks
// and, when it changed, reloads the snapshot and applies it inside the loop. Only
// one poll is in flight at a time (npcPolling). Called from the mob-AI Tick.
func (d *Dispatcher) pollNPCConfig(w *world.World) {
	if d.npcSource == nil || d.npcPolling {
		return
	}
	d.npcPollTick++
	if d.npcPollTick%npcPollPeriod != 0 {
		return
	}
	known := d.npcVersion // captured in-loop; the off-loop goroutine never reads loop state
	d.npcPolling = true
	w.GoDetached(func() func(*world.World) {
		ctx, cancel := context.WithTimeout(context.Background(), npcFetchTimeout)
		defer cancel()
		version, err := d.npcSource.Version(ctx)
		if err != nil {
			return func(*world.World) { d.npcPolling = false; d.log.Warn("npc config version poll failed", "err", err) }
		}
		if version == known {
			return func(*world.World) { d.npcPolling = false }
		}
		snap, err := d.npcSource.Snapshot(ctx)
		if err != nil {
			return func(*world.World) { d.npcPolling = false; d.log.Warn("npc config reload failed", "err", err) }
		}
		return func(w *world.World) {
			d.applyNPCConfig(w, snap, true)
			d.npcPolling = false
			d.log.Info("npc config reloaded", "version", snap.Version, "npcs", len(d.managedNPCs))
		}
	})
}

// applyNPCConfig reconciles the live world with a config snapshot, inside the
// loop. It reconciles by DIFF: a definition whose materialized key is unchanged
// keeps its live entity, so a moderator edit only touches the NPC that changed.
//
// The first implementation wiped and rebuilt the whole managed set on every
// apply. That is semantically simpler but wrong for a live shard: the client of
// a player standing in town then holds the OLD entity next to the NEW one, so
// editing a single NPC made every merchant duplicate on screen (in-game report,
// Armia, 2026-11-04 — three config bumps while the player was online). Reveal still
// announces the few spawns a real diff produces (true on hot-reload, false at
// boot), and the global item-price overlay is refreshed on every apply.
func (d *Dispatcher) applyNPCConfig(w *world.World, snap npccfg.Snapshot, reveal bool) {
	d.rebuildItemPrices(snap.PriceOverrides)

	// A disabled content definition must stop spawning from its NPCGener block.
	for _, def := range snap.Defs {
		if def.Enabled || def.Origin != "content" || def.GeneratorIndex < 0 || world.EventOwnedGenerator(def.GeneratorIndex) {
			continue
		}
		if g := w.GeneratorAt(def.GeneratorIndex); g != nil {
			w.ClearGenerator(def.GeneratorIndex)
			g.LeaderTmpl, g.FollowerTmpl = nil, nil
		}
	}

	live := make(map[string]struct{}, len(snap.Defs))
	for _, def := range snap.Defs {
		if !def.Enabled {
			continue
		}
		live[def.Slug] = struct{}{}
		key := npcDefKey(def)
		if m, ok := d.managedNPCs[def.Slug]; ok {
			if m.key == key && w.Entity(m.id) != nil {
				continue // unchanged: leave the entity and its recipe alone
			}
			d.teardownNPC(w, def.Slug, m)
		}
		if def.Origin == "content" && def.GeneratorIndex >= 0 {
			if world.EventOwnedGenerator(def.GeneratorIndex) {
				continue // the event system spawns this block; the overlay must not
			}
			g := w.GeneratorAt(def.GeneratorIndex)
			if g == nil {
				d.log.Warn("npc content generator index has no registered slot", "slug", def.Slug, "index", def.GeneratorIndex)
				continue
			}
			w.ClearGenerator(def.GeneratorIndex)
			if !def.Enabled {
				g.LeaderTmpl = nil
				g.FollowerTmpl = nil
				continue
			}
			if def.Template == nil {
				d.log.Warn("npc definition has no template - skipped", "slug", def.Slug)
				continue
			}
			g.MinuteGenerate, g.MinGroup, g.MaxGroup, g.MaxNumMob = def.MinuteGenerate, def.MinGroup, def.MaxGroup, def.MaxNumMob
			g.RouteType, g.Formation = def.RouteType, def.Formation
			g.SegX, g.SegY, g.SegRange, g.SegWait = def.SegX, def.SegY, def.SegRange, def.SegWait
			g.FightAction, g.DieAction = def.FightAction, def.DieAction
			g.LeaderTmpl = npcTemplateWithDisplayName(def.Template, def.DisplayName)
			g.FollowerTmpl = def.FollowerTemplate
			if world.IsWaterGenerator(def.GeneratorIndex) {
				continue // water rooms own these blocks; don't spawn them through NPC hot reload
			}
			ids := w.GenerateMob(def.GeneratorIndex)
			if len(ids) == 0 {
				d.log.Warn("npc content generator produced no entities", "slug", def.Slug, "index", def.GeneratorIndex)
				continue
			}
			if def.Merchant != 0 {
				applyShop(w.Entity(ids[0]), def.Shop)
			}
			d.managedNPCs[def.Slug] = managedNPC{id: ids[0], key: key, gen: def.GeneratorIndex}
			if reveal {
				d.revealSpawned(w, ids)
			}
			continue
		}
		if !def.Enabled {
			continue
		}
		if def.Template == nil {
			// Template failed to resolve (missing/corrupt file) — logged in the
			// dbclient source too, but warn here so the skip is visible alongside
			// the other skip branches (bad position, no free slot).
			d.log.Warn("npc definition has no template - skipped", "slug", def.Slug)
			continue
		}
		if def.X <= 0 || def.Y <= 0 {
			d.log.Warn("npc definition has no valid position - skipped", "slug", def.Slug, "x", def.X, "y", def.Y)
			continue
		}
		template := npcTemplateWithDisplayName(def.Template, def.DisplayName)
		id := w.SpawnMobAt(world.MobSpawn{
			Template:  template,
			X:         def.X,
			Y:         def.Y,
			RouteType: def.RouteType,
			GenIndex:  -1, // not owned by an NPCGener block (no CurrentNumMob accounting)
		})
		if id < 0 {
			d.log.Warn("npc spawn failed (no free slot)", "slug", def.Slug)
			continue
		}
		if def.Merchant != 0 {
			applyShop(w.Entity(id), def.Shop)
		}
		d.managedNPCs[def.Slug] = managedNPC{id: id, key: key, gen: -1}
		if reveal {
			d.revealSpawned(w, []int{id})
		}
	}
	for slug, m := range d.managedNPCs {
		if _, ok := live[slug]; !ok {
			d.teardownNPC(w, slug, m) // definition deleted from the config
		}
	}
	d.npcVersion = snap.Version
}

// managedNPC is one materialized definition: its live leader entity, the key that
// summarizes what was applied, and the NPCGener block it owns (-1 for a definition
// spawned on its own position). The key is what turns a reload into a diff.
type managedNPC struct {
	id  int
	key string
	gen int
}

// teardownNPC removes one managed definition. A content block is cleared whole
// (leader, followers and any queued respawn); a standalone NPC is despawned.
func (d *Dispatcher) teardownNPC(w *world.World, slug string, m managedNPC) {
	if m.gen >= 0 {
		w.ClearGenerator(m.gen)
	} else {
		w.DespawnMob(m.id, 0) // removeType 0 = out-of-view removal, never queues a respawn
	}
	delete(d.managedNPCs, slug)
}

// npcDefKey summarizes everything a definition applies to the live world, so an
// apply can tell “nothing changed” from “respawn this one”. The template bytes
// are part of it: a re-stamped STRUCT_MOB must re-materialize even when only a
// stat moved.
func npcDefKey(def npccfg.Definition) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s|%d|%s|%d,%d|%d|%d|%d,%d,%d,%d,%d|",
		def.Origin, def.GeneratorIndex, def.DisplayName, def.X, def.Y, def.RouteType,
		def.Merchant, def.MinuteGenerate, def.MinGroup, def.MaxGroup, def.MaxNumMob, def.Formation)
	for i := range def.SegX {
		fmt.Fprintf(h, "%d,%d,%d,%d;", def.SegX[i], def.SegY[i], def.SegRange[i], def.SegWait[i])
	}
	for i := range def.FightAction {
		fmt.Fprintf(h, "%s;%s;", def.FightAction[i], def.DieAction[i])
	}
	for _, it := range def.Shop {
		fmt.Fprintf(h, "%d:%d:%d:%v;", it.Slot, it.Index, it.Quantity, it.Eff)
	}
	h.Write(def.Template)
	h.Write(def.FollowerTemplate)
	return strconv.FormatUint(h.Sum64(), 16)
}

// npcTemplateWithDisplayName overlays the moderator display name onto a copy of
// the raw STRUCT_MOB. Templates are shared by the dbclient cache, so mutating the
// input would leak one NPC's translated name into every NPC using that template.
func npcTemplateWithDisplayName(template []byte, displayName string) []byte {
	if strings.TrimSpace(displayName) == "" {
		return template
	}
	out := append([]byte(nil), template...)
	for i := 0; i < 16 && i < len(out); i++ {
		out[i] = 0
	}
	copy(out[0:min(16, len(out))], displayName)
	return out
}

// applyShop overwrites a merchant entity's shop stock (its Carry[]) with the
// moderator-defined slots. Slot is the MSG_ShopList index (0..26); it is mapped
// to the real Carry index via protocol.ShopSlot (3 tabs of 9: Carry[0..8],
// [27..35], [54..62]) — the same mapping reqShopList reads back, so an item in
// tab 2/3 (slot >= 9) actually reaches the client. The shop is authoritative:
// slots not listed are cleared, so emptying the shop in the web sells nothing.
func applyShop(e *world.Entity, shop []npccfg.ShopItem) {
	if e == nil {
		return
	}
	for i := range e.Carry {
		e.Carry[i] = world.Item{}
	}
	for _, it := range shop {
		if it.Slot < 0 || it.Slot >= maxShopSlots {
			continue
		}
		e.Carry[protocol.ShopSlot(it.Slot)] = world.Item{
			Index:   int16(it.Index),
			Effects: shopEffects(it),
		}
	}
}

const amountEffect = 61

func shopEffects(it npccfg.ShopItem) [3]world.Effect {
	eff := [3]world.Effect{
		{Effect: it.Eff[0][0], Value: it.Eff[0][1]},
		{Effect: it.Eff[1][0], Value: it.Eff[1][1]},
		{Effect: it.Eff[2][0], Value: it.Eff[2][1]},
	}
	if it.Quantity <= 1 {
		return eff
	}
	for i := range eff {
		if eff[i].Effect == 0 {
			eff[i] = world.Effect{Effect: amountEffect, Value: it.Quantity}
			return eff
		}
	}
	return eff
}

// maxShopSlots is the number of MSG_ShopList slots (3 tabs of 9); a shop item's
// Slot is a display index in [0, maxShopSlots).
const maxShopSlots = 27

// rebuildItemPrices resets the effective price map to the content catalog, then
// applies the moderator's global overrides on top. Loop-only, so handlers reading
// d.itemPrices never race it.
func (d *Dispatcher) rebuildItemPrices(overrides map[int]int32) {
	if len(overrides) == 0 && d.baseItemPrices == nil {
		return
	}
	d.itemPrices = cloneInt32Map(d.baseItemPrices)
	if d.itemPrices == nil {
		d.itemPrices = make(map[int]int32, len(overrides))
	}
	for idx, price := range overrides {
		d.itemPrices[idx] = price
	}
}

// cloneInt32Map returns a shallow copy of m (nil for a nil input).
func cloneInt32Map(m map[int]int32) map[int]int32 {
	if m == nil {
		return nil
	}
	out := make(map[int]int32, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
