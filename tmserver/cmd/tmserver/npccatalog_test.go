package main

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"openwyd-nextgen/internal/npctemplate"
	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/npccfg"
	"openwyd-nextgen/tmserver/internal/world"
)

// spawnNPCsForTest reproduces the pre-issue-57 boot for the tests that assert the
// spawn wiring: it parses NPCGener.txt into the definitions the importer would
// build, registers the slots and fires one GenerateMob per boot-spawnable block.
// Production no longer reads the file (the database is the catalogue), so this
// helper keeps those tests exercising registerNPCGenerators + GenerateMob without
// dragging the legacy loader back into cmd/tmserver.
func spawnNPCsForTest(w *world.World, dir string, attrs *content.Grid, log *slog.Logger) {
	blocks, err := content.LoadNPCGenerators(filepath.Join(dir, "TMsrv", "run", "NPCGener.txt"))
	if err != nil {
		panic(fmt.Sprintf("spawnNPCsForTest: %v", err))
	}
	defs := make([]npccfg.Definition, 0, len(blocks))
	slots := make([]int, 0, len(blocks))
	for i, b := range blocks {
		leader, _, lerr := npctemplate.Load(dir, b.Leader)
		if lerr != nil {
			continue // same rule as the importer: a block without its template cannot spawn
		}
		def := npccfg.Definition{
			Slug: fmt.Sprintf("%s-%d", b.Leader, i), Origin: "content", GeneratorIndex: i,
			Template: leader, Enabled: true,
			MinuteGenerate: b.MinuteGenerate, MinGroup: b.MinGroup, MaxGroup: b.MaxGroup, MaxNumMob: b.MaxNumMob,
			RouteType: uint8(b.RouteType), Formation: b.Formation,
			X: b.SegX[0], Y: b.SegY[0],
		}
		if b.Follower != "" {
			if follower, _, ferr := npctemplate.Load(dir, b.Follower); ferr == nil {
				def.FollowerTemplate = follower
			}
		}
		for j := 0; j < 5; j++ {
			def.SegX[j], def.SegY[j] = b.SegX[j], b.SegY[j]
			def.SegRange[j], def.SegWait[j] = int16(b.SegRange[j]), int16(b.SegWait[j])
		}
		copy(def.FightAction[:], b.FightAction[:])
		copy(def.DieAction[:], b.DieAction[:])
		defs = append(defs, def)
		slots = append(slots, i)
	}
	registerNPCGenerators(w, defs, attrs, log)
	for _, idx := range slots {
		if world.EventOwnedGenerator(idx) {
			continue // the event code owns the spawn; boot must not materialize it
		}
		w.GenerateMob(idx)
	}
}
