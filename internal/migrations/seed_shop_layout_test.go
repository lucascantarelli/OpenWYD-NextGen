package migrations

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// A faixa 5000..5095 do catálogo é o livro de skills: 4 classes de 24 skills,
// cada classe em 3 famílias de 8 (branca/negra/especial). O NPC de skill vende a
// família inteira nos slots 0-7, 9-16 e 18-25 — o mesmo layout dos NPCs íntegros.
const (
	skillBase       = 5000
	skillsPerClass  = 24
	skillsPerFamily = 8
	skillLastIndex  = skillBase + 4*skillsPerClass - 1
)

// seedRow é o subconjunto de npc_shop_item que a verificação precisa; o seed é
// JSON por linha (json_populate_record), então o parse não depende do formato do
// INSERT.
type seedRow struct {
	NpcID     int64 `json:"npc_id"`
	Slot      int   `json:"slot"`
	ItemIndex int   `json:"item_index"`
}

// seedObject captura o objeto JSON de cada INSERT do seed.
var seedObject = regexp.MustCompile(`\$json\$\s*(\{.*?\})\s*\$json\$`)

func parseShopSeed(t *testing.T, file string) []seedRow {
	t.Helper()
	data, err := FS.ReadFile(file)
	if err != nil {
		t.Fatalf("ler %s: %v", file, err)
	}
	var rows []seedRow
	for _, m := range seedObject.FindAllSubmatch(data, -1) {
		var row seedRow
		if err := json.Unmarshal(m[1], &row); err != nil {
			t.Fatalf("json inválido em %s: %v", file, err)
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		t.Fatalf("%s não tem nenhuma linha reconhecível", file)
	}
	return rows
}

func isSkillItem(item int) bool { return item >= skillBase && item <= skillLastIndex }

// familySlots devolve os slots da família no layout de venda: 8 skills, um slot
// vazio, 8 skills, um slot vazio, 8 skills. A ordem DENTRO da família é a do
// catálogo do NPC (5024, 5027, 5025, 5026...), não a sequência numérica: por isso
// a checagem valida o conjunto de slots, não o índice de cada skill.
func familySlots(fam int) (int, int) {
	first := fam*skillsPerFamily + fam
	return first, first + skillsPerFamily - 1
}

// shopLayoutViolations devolve uma linha por inconsistência entre o que a loja
// vende e o catálogo de skills:
//
//  1. uma loja que vende skill não mistura classes (loja da Foema não vende skill
//     de TK);
//  2. a família que a loja vende está completa nos 8 slots dela — um item no lugar
//     de uma skill aparece como slot vazio ou como skill ausente.
func shopLayoutViolations(rows []seedRow) []string {
	byNPC := map[int64]map[int]int{}
	for _, r := range rows {
		if byNPC[r.NpcID] == nil {
			byNPC[r.NpcID] = map[int]int{}
		}
		byNPC[r.NpcID][r.Slot] = r.ItemIndex
	}

	var out []string
	for npc, slots := range byNPC {
		byClass := map[int]bool{}
		for _, item := range slots {
			if isSkillItem(item) {
				byClass[(item-skillBase)/skillsPerClass] = true
			}
		}
		if len(byClass) == 0 {
			continue
		}
		if len(byClass) > 1 {
			out = append(out, fmt.Sprintf("mistura classes de skill: npc %d", npc))
			continue
		}
		class := 0
		for c := range byClass {
			class = c
		}
		for fam := 0; fam < skillsPerClass/skillsPerFamily; fam++ {
			base := skillBase + class*skillsPerClass + fam*skillsPerFamily
			sold := false
			for _, item := range slots {
				if item >= base && item < base+skillsPerFamily {
					sold = true
					break
				}
			}
			if !sold {
				continue
			}
			first, last := familySlots(fam)
			seen := 0
			for slot := first; slot <= last; slot++ {
				got, ok := slots[slot]
				switch {
				case !ok:
					out = append(out, fmt.Sprintf("slot de skill vazio: npc %d família %d slot %d", npc, fam, slot))
				case got < base || got >= base+skillsPerFamily:
					out = append(out, fmt.Sprintf("slot de skill ocupado por item: npc %d família %d slot %d tem %d", npc, fam, slot, got))
				default:
					seen++
				}
			}
			if seen > 0 && seen < skillsPerFamily {
				out = append(out, fmt.Sprintf("família incompleta: npc %d família %d tem %d de %d skills", npc, fam, seen, skillsPerFamily))
			}
		}
	}
	sort.Strings(out)
	return out
}

func TestSeedSkillShopLayout(t *testing.T) {
	rows := parseShopSeed(t, "seed_npc_shop_item.up.sql")
	if got := shopLayoutViolations(rows); len(got) > 0 {
		t.Fatalf("loja de skill inconsistente com o catálogo:\n%s", strings.Join(got, "\n"))
	}
}

// TestShopLayoutCatchesItemInsteadOfSkill prova que a checagem detecta o defeito
// histórico (Garnet no slot da 8ª magia branca) em vez de passar por vacuidade.
func TestShopLayoutCatchesItemInsteadOfSkill(t *testing.T) {
	bad := []seedRow{
		{NpcID: 152, Slot: 0, ItemIndex: 5024},
		{NpcID: 152, Slot: 1, ItemIndex: 5027},
		{NpcID: 152, Slot: 2, ItemIndex: 5025},
		{NpcID: 152, Slot: 3, ItemIndex: 5026},
		{NpcID: 152, Slot: 4, ItemIndex: 5028},
		{NpcID: 152, Slot: 5, ItemIndex: 5029},
		{NpcID: 152, Slot: 6, ItemIndex: 5030},
		{NpcID: 152, Slot: 7, ItemIndex: 2444}, // Garnet no lugar de Renascimento
	}
	got := shopLayoutViolations(bad)
	if len(got) == 0 {
		t.Fatal("a checagem aceitou item no slot de skill")
	}
	if !strings.Contains(strings.Join(got, "\n"), "npc 152") {
		t.Fatalf("violação não aponta o NPC: %v", got)
	}
}
