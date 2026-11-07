package content

import (
	"strconv"
	"strings"

	"openwyd-nextgen/internal/ratetables"
)

// ApplyRateOverrides aplica a sobreposição do moderador (issue 36) nas tabelas
// carregadas de tmserver/content/Common/Settings.
//
// A identidade de uma célula é (arquivo, família, nome) — a mesma chave que o editor
// web e content_rate_override usam — nunca uma posição numérica: CompRate tem várias
// chaves por família, e uma chave posicional as faria colidir. Linhas que não
// cabem nas tabelas tipadas são contadas como ignoradas em vez de inventadas: o
// mundo só roda formato de tabela que o cliente entende.
func ApplyRateOverrides(sanc *SancRate, comp *CompRate, quests *QuestRates, overrides []ratetables.Override) (applied, ignored int) {
	for _, o := range overrides {
		switch o.File {
		case ratetables.FileSancRate:
			anvil, idx, ok := sancCell(o.Family, o.Name, sanc)
			if !ok {
				ignored++
				continue
			}
			sanc.rates[anvil][idx] = int(o.Value)
			applied++

		case ratetables.FileCompRate:
			if comp == nil || o.Family == "" || o.Name == "" {
				ignored++
				continue
			}
			if comp.rates == nil {
				comp.rates = make(map[string]map[string]int)
			}
			if comp.rates[o.Family] == nil {
				comp.rates[o.Family] = make(map[string]int)
			}
			comp.rates[o.Family][o.Name] = int(o.Value)
			applied++

		case ratetables.FileQuestsRate:
			if quests == nil {
				ignored++
				continue
			}
			if !applyQuestOverride(quests, o.Family, o.Name, int32(o.Value)) {
				ignored++
				continue
			}
			applied++

		default:
			ignored++
		}
	}
	return applied, ignored
}

// sancCell resolve "<anvil>/<index>" contra a tabela que o arquivo produziu. Os
// nomes de família são os canônicos que ratetables emite (PO, PL, AMAGO).
func sancCell(family, name string, sanc *SancRate) (int, int, bool) {
	if sanc == nil {
		return 0, 0, false
	}
	var anvil int
	switch strings.ToUpper(family) {
	case "PO":
		anvil = sancAnvilOri
	case "PL":
		anvil = sancAnvilLac
	case "AMAGO":
		anvil = sancAnvilAma
	default:
		return 0, 0, false
	}
	idx, err := strconv.Atoi(name)
	if err != nil || idx < 0 || idx >= len(sanc.rates[anvil]) {
		return 0, 0, false
	}
	return anvil, idx, true
}

// applyQuestOverride escreve uma célula de QuestsRate: EXP "<tier>.0|1" (mortal/arch),
// COIN "<tier>.0" e LEVEL "<tier>.<0..3>" (mortal mín/máx, arch mín/máx).
func applyQuestOverride(quests *QuestRates, family, name string, value int32) bool {
	if value < 0 {
		return false
	}
	tierPart, columnPart, ok := strings.Cut(name, ".")
	if !ok {
		return false
	}
	tier, err := strconv.Atoi(tierPart)
	if err != nil || tier < 0 || tier >= len(quests.tiers) {
		return false
	}
	column, err := strconv.Atoi(columnPart)
	if err != nil {
		return false
	}
	r := &quests.tiers[tier]

	switch strings.ToUpper(family) {
	case "EXP":
		switch column {
		case 0:
			r.MortalExp = int64(value)
		case 1:
			r.ArchExp = int64(value)
		default:
			return false
		}
		return true
	case "COIN":
		if column != 0 {
			return false
		}
		r.Coin = value
		return true
	case "LEVEL":
		switch column {
		case 0:
			r.MortalMin = value
		case 1:
			r.MortalMax = value
		case 2:
			r.ArchMin = value
		case 3:
			r.ArchMax = value
		default:
			return false
		}
		return true
	}
	return false
}
