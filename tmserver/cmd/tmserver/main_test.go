package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/world"
)

func TestSpawnNPCsLeavesWaterGeneratorsForRoomLifecycle(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "TMsrv", "run")
	npcDir := filepath.Join(runDir, "npc")
	if err := os.MkdirAll(npcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for index := 0; index <= 10; index++ {
		fmt.Fprintf(&text, "# [%d]\nLeader: Grunt\nMinGroup: 0\nMaxGroup: 0\nMaxNumMob: 1\nStartX: %d\nStartY: 20\n\n", index, 20+index*5)
	}
	if err := os.WriteFile(filepath.Join(runDir, "NPCGener.txt"), []byte(text.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(npcDir, "Grunt"), testMobTemplate("Grunt"), 0o644); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	w := world.New(world.Config{GridDim: 128}, log, nil, nil)
	spawnNPCsForTest(w, dir, nil, log)
	if w.GeneratorAt(10) == nil || w.GeneratorAt(10).CurrentNumMob != 0 {
		t.Fatal("water generator spawned during world startup")
	}
	if w.GeneratorAt(9) == nil || w.GeneratorAt(9).CurrentNumMob != 1 {
		t.Fatal("ordinary generator failed to spawn")
	}
	for index := 0; index <= 10; index++ {
		w.ClearGenerator(index)
	}
}

// TestSpawnNPCsSkipsMissingLeaderTemplate: bloco cujo template de líder não pode
// ser decodificado nunca vira slot de gerador (o importador o descarta, então o
// catálogo lido pelo tmServer não tem definição para ele), e um seguidor ausente
// degrada o bloco para um grupo só com o líder.
func TestSpawnNPCsSkipsMissingLeaderTemplate(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "TMsrv", "run")
	npcDir := filepath.Join(runDir, "npc")
	if err := os.MkdirAll(npcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const npcGener = `# [0]
	Leader: MissingLeader
	MinGroup: 0
	MaxGroup: 0
	MaxNumMob: 1
	StartX: 10
	StartY: 10

# [1]
	Leader: ExistingLeader
	Follower: MissingFollower
	MinGroup: 1
	MaxGroup: 1
	MaxNumMob: 2
	StartX: 20
	StartY: 20
`
	if err := os.WriteFile(filepath.Join(runDir, "NPCGener.txt"), []byte(npcGener), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(npcDir, "ExistingLeader"), testMobTemplate("ExistingLeader"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	w := world.New(world.Config{GridDim: 64}, logger, nil, nil)
	spawnNPCsForTest(w, dir, nil, logger)

	if g := w.GeneratorAt(0); g != nil {
		t.Fatalf("missing leader generator = %#v, want skipped nil slot", g)
	}
	g := w.GeneratorAt(1)
	if g == nil {
		t.Fatal("existing leader generator was not registered")
	}
	if g.FollowerTmpl != nil {
		t.Fatal("missing follower template should degrade to leader-only group")
	}
	if g.CurrentNumMob != 1 {
		t.Fatalf("CurrentNumMob = %d, want one spawned leader", g.CurrentNumMob)
	}
}

func TestSpawnNPCsResolvesLegacyTemplateNames(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "TMsrv", "run")
	npcDir := filepath.Join(runDir, "npc")
	if err := os.MkdirAll(npcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const npcGener = `# [0]
	Leader: Chefe_Treina.
	MinGroup: 0
	MaxGroup: 0
	MaxNumMob: 1
	StartX: 10
	StartY: 10

# [1]
	Leader: Reiners
	MinGroup: 0
	MaxGroup: 0
	MaxNumMob: 1
	StartX: 12
	StartY: 12
`
	if err := os.WriteFile(filepath.Join(runDir, "NPCGener.txt"), []byte(npcGener), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(npcDir, "Chefe_Treina"), testMobTemplate("Chefe_Treina."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(npcDir, "reiners"), testMobTemplate("Reiners"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	w := world.New(world.Config{GridDim: 64}, logger, nil, nil)
	spawnNPCsForTest(w, dir, nil, logger)

	for i := 0; i < 2; i++ {
		g := w.GeneratorAt(i)
		if g == nil {
			t.Fatalf("generator %d was not registered", i)
		}
		if g.CurrentNumMob != 1 {
			t.Fatalf("generator %d CurrentNumMob = %d, want one spawned leader", i, g.CurrentNumMob)
		}
	}
}

func testMobTemplate(name string) []byte {
	b := make([]byte, content.BaseMobSize)
	copy(b[0:16], name)
	b[16] = 2
	const cs = 92
	binary.LittleEndian.PutUint32(b[cs+0:], 1)
	binary.LittleEndian.PutUint32(b[cs+16:], 100)
	binary.LittleEndian.PutUint32(b[cs+24:], 100)
	return b
}

func TestSpawnNPCsMarksTrainingFieldGenerators(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "TMsrv", "run")
	npcDir := filepath.Join(runDir, "npc")
	if err := os.MkdirAll(npcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const npcGener = "# [0]\nLeader: Grunt\nMinGroup: 0\nMaxGroup: 0\nMaxNumMob: 1\nStartX: 13\nStartY: 9\n\n" +
		"# [1]\nLeader: Grunt\nMinGroup: 0\nMaxGroup: 0\nMaxNumMob: 1\nStartX: 40\nStartY: 40\n\n"
	if err := os.WriteFile(filepath.Join(runDir, "NPCGener.txt"), []byte(npcGener), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(npcDir, "Grunt"), testMobTemplate("Grunt"), 0o644); err != nil {
		t.Fatal(err)
	}
	attrs := &content.Grid{Dim: 32, Data: make([]byte, 32*32)}
	attrs.Data[2*32+3] = 0x80
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	w := world.New(world.Config{GridDim: 128}, log, nil, nil)
	spawnNPCsForTest(w, dir, attrs, log)
	if g := w.GeneratorAt(0); g == nil || !g.TrainingField {
		t.Error("generator on a 0x80 cell not marked as training field")
	}
	if g := w.GeneratorAt(1); g == nil || g.TrainingField {
		t.Error("generator off the field marked as training field")
	}
	for index := 0; index <= 1; index++ {
		w.ClearGenerator(index)
	}
}
