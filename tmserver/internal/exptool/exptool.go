// Package exptool restampa o campo Exp de recompensa de kill dos templates de npc
// do Release com a curva de nível balanceada (level.MobExpForLevel) — o herdeiro Go
// da ferramenta legada ExpTool (Source/Code/_legacy/ExpTool/main.cpp:128), feita para
// a issue #43 (templates distribuídos misturavam Exp=0 com valores lixo).
package exptool

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"openwyd-nextgen/internal/savefmt"
	"openwyd-nextgen/tmserver/internal/level"
)

// Entry reports one restamped template.
type Entry struct {
	File   string
	Name   string
	Level  int32
	OldExp int64
	NewExp int64
}

// Result resume uma execução do Stamp.
type Result struct {
	Stamped []Entry
	Skipped int // diretórios, NPCs/mercadores e templates de nível 0
	// StampedVariant conta quantos dos Stamped estavam em um dos layouts
	// 756/920 bytes (data-formats.md §1.4.1). Eles guardam Exp como um
	// campo em offset diferente, então savefmt.PutMobExp roteia a escrita; o
	// registro é corrigido no lugar e o arquivo mantém seu tamanho.
	StampedVariant int
	// SkippedNonTemplate conta arquivos que não são STRUCT_MOB em nenhum layout.
	SkippedNonTemplate int
}

// Stamp percorre dir (um STRUCT_MOB cru por arquivo) e reescreve o campo Exp de
// todo monstro real (Merchant==0, Level>=1) com level.MobExpForLevel,
// deixando todo outro byte intocado. Templates de NPC/mercador e de nível 0 são
// pulados. Os três layouts de template são restampados — a árvore de conteúdo mista
// continuaria com 222 criaturas no Exp distribuído enquanto o log de boot
// dizia aos operadores para rodar esta ferramenta (issue #244).
// Com dryRun o relatório é produzido mas nada é escrito.
func Stamp(dir string, dryRun bool) (Result, error) {
	var res Result
	entries, err := os.ReadDir(dir)
	if err != nil {
		return res, fmt.Errorf("reading template dir: %w", err)
	}
	for _, de := range entries {
		if de.IsDir() {
			res.Skipped++
			continue
		}
		path := filepath.Join(dir, de.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return res, fmt.Errorf("reading %s: %w", de.Name(), err)
		}
		// DecodeMobAny (não protocol.ParseMobBasics) porque aquele indexa os
		// offsets canônicos e leria lixo de um registro legado.
		mob, version, derr := savefmt.DecodeMobAny(raw)
		if derr != nil {
			res.SkippedNonTemplate++
			continue
		}
		// CurrentScore.Merchant é a flag que o tmServer respeita, não o
		// Mob.Merchant de topo — o mesmo campo que o importador de NPC lê.
		lvl := mob.CurrentScore.Level
		if mob.CurrentScore.Merchant != 0 || lvl < 1 {
			res.Skipped++
			continue
		}
		newExp := level.MobExpForLevel(lvl)
		res.Stamped = append(res.Stamped, Entry{
			File: de.Name(), Name: cstr(mob.Name[:]), Level: lvl, OldExp: mob.Exp, NewExp: newExp,
		})
		if version != savefmt.MobVersionCurrent {
			res.StampedVariant++
		}
		if dryRun || newExp == mob.Exp {
			continue
		}
		if err := savefmt.PutMobExp(raw, newExp); err != nil {
			return res, fmt.Errorf("stamping %s: %w", de.Name(), err)
		}
		info, err := de.Info()
		if err != nil {
			return res, fmt.Errorf("stat %s: %w", de.Name(), err)
		}
		if err := os.WriteFile(path, raw, info.Mode().Perm()); err != nil {
			return res, fmt.Errorf("writing %s: %w", de.Name(), err)
		}
	}
	sort.Slice(res.Stamped, func(i, j int) bool {
		a, b := res.Stamped[i], res.Stamped[j]
		if a.Level != b.Level {
			return a.Level < b.Level
		}
		return a.File < b.File
	})
	return res, nil
}

// cstr corta um campo de nome de largura fixa no primeiro NUL.
func cstr(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}
