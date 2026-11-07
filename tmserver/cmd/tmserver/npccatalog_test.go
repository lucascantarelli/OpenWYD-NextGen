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

// spawnNPCsForTest reproduz o boot anterior à issue 57 para os testes que verificam a
// ligação de spawn: ele lê NPCGener.txt para as definições que o importador
// construiria, registra os slots e dispara um GenerateMob por bloco spawnável no boot.
// Produção não lê mais o arquivo (o banco é o catálogo), então este
// helper mantém esses testes exercitando registerNPCGenerators + GenerateMob sem
// arrastar o loader legado de volta para cmd/tmserver.
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
			continue // mesma regra do importador: um bloco sem o seu template não pode spawnar
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
			continue // o código do evento é dono do spawn; o boot não pode materializá-lo
		}
		w.GenerateMob(idx)
	}
}
