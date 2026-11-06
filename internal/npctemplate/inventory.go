package npctemplate

import (
	"fmt"
	"os"
	"sort"

	"openwyd-nextgen/internal/savefmt"
)

// Inventory é o censo reproduzível de um diretório de templates: cada arquivo
// classificado pelo layout em disco. É uma lista repetível dos válidos, dos
// legados e dos que não são template, para auditar uma árvore de conteúdo sem
// ler o log de um deploy.
type Inventory struct {
	Dir   string
	Stats ScanStats
	// Names lista, ordenados, os nomes de arquivo de cada layout. Arquivos que
	// não puderam ser lidos ficam em savefmt.MobVersionUnknown junto dos que não
	// são template, porque nenhum dos dois rende um template utilizável.
	Names map[savefmt.MobVersion][]string
}

// TakeInventory percorre dir (um arquivo por template, sem recursão) e
// classifica cada entrada por savefmt.DetectMobVersion. Só faz stat nos
// arquivos — nada é decodificado nem escrito —, então é seguro rodar contra uma
// árvore de conteúdo somente leitura.
func TakeInventory(dir string) (Inventory, error) {
	inv := Inventory{Dir: dir, Names: make(map[savefmt.MobVersion][]string)}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return inv, fmt.Errorf("npctemplate: read %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		info, err := entry.Info()
		if err != nil {
			inv.Stats.CountUnreadable()
			inv.Names[savefmt.MobVersionUnknown] = append(inv.Names[savefmt.MobVersionUnknown], name)
			continue
		}
		version := savefmt.DetectMobVersion(int(info.Size()))
		inv.Stats.Count(version)
		inv.Names[version] = append(inv.Names[version], name)
	}
	for _, names := range inv.Names {
		sort.Strings(names)
	}
	return inv, nil
}

// Report rende o inventário em linhas legíveis: um resumo por layout e depois
// os nomes de cada arquivo não canônico (o conjunto canônico é grande e
// desinteressante demais para listar).
func (inv Inventory) Report(sample int) []string {
	// Só os layouts não canônicos têm os nomes listados; o conjunto canônico é
	// resumido. Manter essa lista separada (em vez de fatiar a ordem do resumo)
	// evita que reordenar o resumo passe a despejar 1769 nomes canônicos.
	nonCanonical := []savefmt.MobVersion{
		savefmt.MobVersionLegacy756,
		savefmt.MobVersionLegacy756Padded,
		savefmt.MobVersionUnknown,
	}
	order := append([]savefmt.MobVersion{savefmt.MobVersionCurrent}, nonCanonical...)

	lines := []string{fmt.Sprintf("%s: %d files", inv.Dir, inv.Stats.Total())}
	for _, v := range order {
		lines = append(lines, fmt.Sprintf("  %-18s %d", v.String(), len(inv.Names[v])))
	}
	for _, v := range nonCanonical {
		names := inv.Names[v]
		if len(names) == 0 {
			continue
		}
		shown := names
		if sample > 0 && len(shown) > sample {
			shown = shown[:sample]
		}
		lines = append(lines, "", fmt.Sprintf("%s (%d):", v.String(), len(names)))
		for _, n := range shown {
			lines = append(lines, "  "+n)
		}
		if len(shown) < len(names) {
			lines = append(lines, fmt.Sprintf("  … and %d more", len(names)-len(shown)))
		}
	}
	return lines
}
