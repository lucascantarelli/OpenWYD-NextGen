// Package droptool monta o equivalente somente leitura, no web-api, do relatório
// do antigo DropTool.exe. A ferramenta legada varria os templates STRUCT_MOB em
// TMsrv/run/npc e invertia a tabela Carry[] de cada mob em relatórios item→mob e
// mob→item; ela não era consumida pelo servidor de jogo em runtime.
package droptool

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"openwyd-nextgen/internal/npctemplate"
	"openwyd-nextgen/internal/savefmt"
	"openwyd-nextgen/webserver/internal/itemcatalog"
)

// ItemEntry é um item conhecido pelo relatório de drop.
type ItemEntry struct {
	Index int32
	Name  string
}

// MobEntry é um template de mob decodificado de tmserver/content/TMsrv/run/npc.
type MobEntry struct {
	TemplateName string
	DisplayName  string
	Level        int32
}

// DropEntry é um slot STRUCT_MOB.Carry não vazio.
type DropEntry struct {
	TemplateName string
	MobName      string
	MobLevel     int32
	Slot         int32
	ItemIndex    int32
	ItemName     string
	RateDivisor  int32
}

// ItemDropEntry é a projeção por item servida ao portal.
type ItemDropEntry struct {
	ItemIndex int32
	ItemName  string
	Mobs      []ItemDropMob
}

// ItemDropMob é um mob/slot onde o item pode cair.
type ItemDropMob struct {
	TemplateName string
	MobName      string
	MobLevel     int32
	Slot         int32
	RateDivisor  int32
}

// MobDropEntry é a projeção por mob servida ao portal.
type MobDropEntry struct {
	TemplateName string
	MobName      string
	MobLevel     int32
	Items        []MobDropItem
}

// MobDropItem é um slot de drop num template de mob.
type MobDropItem struct {
	Slot        int32
	ItemIndex   int32
	ItemName    string
	RateDivisor int32
}

// ExclusionRange espelha as faixas "de,até" opcionais do ItensExceptions.txt que
// o DropTool legado usava para suprimir faixas conhecidas do catálogo nos
// cruzamentos item→mob.
type ExclusionRange struct {
	From int32
	To   int32
}

// Options controla uma varredura da árvore de conteúdo.
type Options struct {
	Exclusions []ExclusionRange
}

// Filter controla as projeções em memória devolvidas por Catalog.
type Filter struct {
	ItemIndex            int32
	ItemQuery            string
	MobQuery             string
	IncludeZeroDropItems bool
}

// Catalog é uma varredura da árvore de conteúdo. É imutável depois de construído
// e seguro para compartilhar entre requisições.
type Catalog struct {
	Items []ItemEntry
	Mobs  []MobEntry
	Drops []DropEntry
}

// LoadContentExclusions lê o ItensExceptions.txt legado opcional ao lado dos
// arquivos de execução do TMsrv. Arquivo ausente é normal e significa nenhuma
// exclusão.
func LoadContentExclusions(contentDir string) ([]ExclusionRange, error) {
	return LoadExclusions(filepath.Join(contentDir, "TMsrv", "run", "ItensExceptions.txt"))
}

// LoadExclusions interpreta um arquivo no formato ItensExceptions.txt. Linhas
// começadas por '#' e linhas vazias são ignoradas; cada linha de dados é "de,até".
func LoadExclusions(path string) ([]ExclusionRange, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("droptool: open exclusions %s: %w", path, err)
	}
	defer f.Close()

	var out []ExclusionRange
	sc := bufio.NewScanner(f)
	for lineNo := 1; sc.Scan(); lineNo++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fromRaw, toRaw, ok := strings.Cut(line, ",")
		if !ok {
			return nil, fmt.Errorf("droptool: exclusions %s:%d: want from,to", path, lineNo)
		}
		from, err := strconv.Atoi(strings.TrimSpace(fromRaw))
		if err != nil {
			return nil, fmt.Errorf("droptool: exclusions %s:%d: parse from: %w", path, lineNo, err)
		}
		to, err := strconv.Atoi(strings.TrimSpace(toRaw))
		if err != nil {
			return nil, fmt.Errorf("droptool: exclusions %s:%d: parse to: %w", path, lineNo, err)
		}
		out = append(out, ExclusionRange{From: int32(from), To: int32(to)})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("droptool: scan exclusions %s: %w", path, err)
	}
	return out, nil
}

// SlotDropRate é g_pDropRate[64], duplicado do pacote de runtime porque o
// webserver não pode importar tmserver/internal/loot. Os valores vêm de
// Basedef.cpp e docs/migration/game-rules.md §2.2; divisor maior = mais raro.
var SlotDropRate [savefmt.MaxCarry]int

func init() {
	fill := func(from, to, v int) {
		for i := from; i <= to; i++ {
			SlotDropRate[i] = v
		}
	}
	fill(0, 7, 900)
	fill(8, 11, 4)
	fill(12, 15, 900)
	fill(16, 23, 20000)
	fill(24, 47, 2000)
	fill(48, 55, 3000)
	SlotDropRate[56] = 1
	for i, v := range []int{35, 500, 2500, 5000, 5000, 10000, 20000} {
		SlotDropRate[57+i] = v
	}
}

// Scan lê <contentDir>/Common/ItemList.csv e <contentDir>/TMsrv/run/npc/,
// aceitando todo layout STRUCT_MOB que o savefmt conhece (816 canônico mais as
// formas legadas 756/920 — data-formats.md §1.4.1). Arquivos que não são template
// nenhum entram nas estatísticas devolvidas e são ignorados, igual aos scanners de
// conteúdo já usados pela UI de moderação.
func Scan(contentDir string, logger *slog.Logger, opts Options) (Catalog, npctemplate.ScanStats, error) {
	catalog, err := itemcatalog.Scan(contentDir)
	if err != nil {
		return Catalog{}, npctemplate.ScanStats{}, fmt.Errorf("droptool: item catalog: %w", err)
	}
	out, stats, err := build(catalog.Items, opts, func(visit func(string, savefmt.Mob)) (npctemplate.ScanStats, error) {
		return npctemplate.ScanDir(contentDir, logger, visit)
	})
	if err != nil {
		return Catalog{}, stats, fmt.Errorf("droptool: %w", err)
	}
	return out, stats, nil
}

// FromBlobs monta o mesmo catálogo a partir do que o banco guarda — as linhas
// do catálogo de item mais os blobs STRUCT_MOB canônicos de npc_template
// (issues 58/62). ItemList.csv e npc/ saíram de tmserver/content/ na fase 5, então a
// varredura de diretório só pode vir vazia.
func FromBlobs(items []itemcatalog.Entry, blobs map[string][]byte, opts Options) (Catalog, npctemplate.ScanStats) {
	out, stats, _ := build(items, opts, func(visit func(string, savefmt.Mob)) (npctemplate.ScanStats, error) {
		return npctemplate.ScanBlobs(blobs, visit), nil
	})
	return out, stats
}

// build projeta a lista de itens e os templates decodificados no catálogo do
// DropTool. walk fornece os templates, então o diretório e o banco
// compartilham uma única projeção em vez de divergirem.
func build(items []itemcatalog.Entry, opts Options, walk func(func(string, savefmt.Mob)) (npctemplate.ScanStats, error)) (Catalog, npctemplate.ScanStats, error) {
	itemByIndex := make(map[int32]string, len(items))
	out := Catalog{Items: make([]ItemEntry, 0, len(items))}
	for _, item := range items {
		itemByIndex[item.Index] = item.Name
		out.Items = append(out.Items, ItemEntry{Index: item.Index, Name: item.Name})
	}

	knownItems := make(map[int32]bool, len(out.Items))
	for _, item := range out.Items {
		knownItems[item.Index] = true
	}
	stats, err := walk(func(templateName string, mob savefmt.Mob) {
		mobName := cString(mob.Name[:])
		mobEntry := MobEntry{
			TemplateName: templateName,
			DisplayName:  mobName,
			Level:        mob.CurrentScore.Level,
		}
		out.Mobs = append(out.Mobs, mobEntry)

		for slot, item := range mob.Carry {
			itemIndex := int32(item.Index)
			if itemIndex <= 0 || excluded(itemIndex, opts.Exclusions) {
				continue
			}
			itemName := itemByIndex[itemIndex]
			out.Drops = append(out.Drops, DropEntry{
				TemplateName: templateName,
				MobName:      mobName,
				MobLevel:     mob.CurrentScore.Level,
				Slot:         int32(slot),
				ItemIndex:    itemIndex,
				ItemName:     itemName,
				RateDivisor:  int32(SlotDropRate[slot]),
			})
			if !knownItems[itemIndex] {
				knownItems[itemIndex] = true
				out.Items = append(out.Items, ItemEntry{Index: itemIndex, Name: itemName})
			}
		}
	})
	if err != nil {
		return Catalog{}, stats, err
	}

	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].Index < out.Items[j].Index })
	sort.Slice(out.Mobs, func(i, j int) bool {
		return out.Mobs[i].TemplateName < out.Mobs[j].TemplateName
	})
	sort.Slice(out.Drops, func(i, j int) bool {
		a, b := out.Drops[i], out.Drops[j]
		if a.ItemIndex != b.ItemIndex {
			return a.ItemIndex < b.ItemIndex
		}
		if a.TemplateName != b.TemplateName {
			return a.TemplateName < b.TemplateName
		}
		return a.Slot < b.Slot
	})
	return out, stats, nil
}

// ListDropItems devolve o relatório por item. Por padrão inclui só
// itens com pelo menos um drop correspondente; IncludeZeroDropItems espelha
// os cabeçalhos de item do relatório legado com zero mobs.
func (c Catalog) ListDropItems(filter Filter) []ItemDropEntry {
	itemQuery := normalize(filter.ItemQuery)
	mobQuery := normalize(filter.MobQuery)

	dropsByItem := make(map[int32][]DropEntry)
	for _, drop := range c.Drops {
		if !matchesMob(drop.TemplateName, drop.MobName, mobQuery) {
			continue
		}
		dropsByItem[drop.ItemIndex] = append(dropsByItem[drop.ItemIndex], drop)
	}

	var out []ItemDropEntry
	for _, item := range c.Items {
		if !matchesItem(item.Index, item.Name, filter.ItemIndex, itemQuery) {
			continue
		}
		drops := dropsByItem[item.Index]
		if len(drops) == 0 && (!filter.IncludeZeroDropItems || mobQuery != "") {
			continue
		}
		entry := ItemDropEntry{ItemIndex: item.Index, ItemName: item.Name}
		for _, drop := range drops {
			entry.Mobs = append(entry.Mobs, ItemDropMob{
				TemplateName: drop.TemplateName,
				MobName:      drop.MobName,
				MobLevel:     drop.MobLevel,
				Slot:         drop.Slot,
				RateDivisor:  drop.RateDivisor,
			})
		}
		out = append(out, entry)
	}
	return out
}

// ListMobDrops devolve o relatório por mob. Sem filtro de item inclui
// todos os templates de mob decodificados, inclusive os sem slot de drop.
func (c Catalog) ListMobDrops(filter Filter) []MobDropEntry {
	itemQuery := normalize(filter.ItemQuery)
	mobQuery := normalize(filter.MobQuery)

	dropsByMob := make(map[string][]DropEntry)
	for _, drop := range c.Drops {
		if !matchesItem(drop.ItemIndex, drop.ItemName, filter.ItemIndex, itemQuery) {
			continue
		}
		dropsByMob[drop.TemplateName] = append(dropsByMob[drop.TemplateName], drop)
	}

	var out []MobDropEntry
	for _, mob := range c.Mobs {
		if !matchesMob(mob.TemplateName, mob.DisplayName, mobQuery) {
			continue
		}
		drops := dropsByMob[mob.TemplateName]
		if len(drops) == 0 && (filter.ItemIndex > 0 || itemQuery != "") {
			continue
		}
		entry := MobDropEntry{
			TemplateName: mob.TemplateName,
			MobName:      mob.DisplayName,
			MobLevel:     mob.Level,
		}
		for _, drop := range drops {
			entry.Items = append(entry.Items, MobDropItem{
				Slot:        drop.Slot,
				ItemIndex:   drop.ItemIndex,
				ItemName:    drop.ItemName,
				RateDivisor: drop.RateDivisor,
			})
		}
		out = append(out, entry)
	}
	return out
}

func excluded(itemIndex int32, ranges []ExclusionRange) bool {
	for _, r := range ranges {
		from, to := r.From, r.To
		if from > to {
			from, to = to, from
		}
		if itemIndex >= from && itemIndex <= to {
			return true
		}
	}
	return false
}

func matchesItem(index int32, name string, exactIndex int32, query string) bool {
	if exactIndex > 0 && index != exactIndex {
		return false
	}
	if query == "" {
		return true
	}
	return strings.Contains(normalize(name), query) ||
		strings.Contains(strconv.FormatInt(int64(index), 10), query)
}

func matchesMob(templateName, mobName, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(normalize(templateName), query) ||
		strings.Contains(normalize(mobName), query)
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			b = b[:i]
			break
		}
	}
	runes := make([]rune, len(b))
	for i, c := range b {
		runes[i] = rune(c)
	}
	return string(runes)
}
