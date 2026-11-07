package content

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ItemEntry é uma linha do ItemList.csv (data-formats.md §3.1).
//
// UNVERIFIED: só Index e Name estão mapeados com confiança; o mapeamento completo
// de coluna→STRUCT_ITEMLIST (Price/Grade/Extra/nPos e os pares de efeito EF_*)
// não está confirmado pelo CSV e fica cru para depois (o .bin é a forma
// compilada). Fields guarda as colunas cruas separadas por vírgula.
type ItemEntry struct {
	Index  int
	Name   string
	Fields []string
}

// ItemList é o catálogo de itens indexado pelo índice do item (g_pItemList).
type ItemList struct {
	items map[int]ItemEntry
}

// Get devolve a entrada de um índice de item.
func (l *ItemList) Get(index int) (ItemEntry, bool) { e, ok := l.items[index]; return e, ok }

// Len devolve o número de itens carregados.
func (l *ItemList) Len() int { return len(l.items) }

// Prices devolve um mapa de índice de item → Price base (STRUCT_ITEMLIST.Price). No
// CSV essa é a 6ª coluna (índice 5, base 0):
// "831,Garra,837.0,4.10.0.0.11,43,530,..." → 530 (BASE_ReadItemListFile, Basedef.cpp).
func (l *ItemList) Prices() map[int]int32 {
	out := make(map[int]int32, len(l.items))
	for idx, e := range l.items {
		if len(e.Fields) > 5 {
			if p, err := strconv.Atoi(strings.TrimSpace(e.Fields[5])); err == nil {
				out[idx] = int32(p)
			}
		}
	}
	return out
}

// BaseEffect é um efeito estático de item (STRUCT_ITEMLIST.stEffect): um id de efeito
// EF_* e seu valor — as stats inerentes do item (dano da arma, AC da armadura, bônus
// de atributo). Vêm do catálogo por índice de item, distintos dos refinos por
// instância guardados no STRUCT_ITEM.
type BaseEffect struct {
	Eff uint8
	Val int16
}

// efName mapeia os tokens EF_<name> do ItemList.csv para seus ids STRUCT_EFFECT.cEffect
// (os mesmos ids que os refinos de instância usam). Só entram efeitos que o modelo de
// score consegue representar. EF_WTYPE entra porque alguns affects de Huntress
// dependem do tipo de arma equipada, mas ele ainda não entra no CurrentScore. Os
// puramente visuais/de requisito (EF_CLASS/EF_GRID/EF_RANGE/EF_REGEN*/…) são ignorados. Os
// ids batem com ItemEffect.h. EF_SANC carrega o nível de refino do item (as joias), consumido
// como multiplicador pelo handler, não como stat fixa.
//
// EF_CRITICAL/EF_CRITICAL2 SÃO stats de score (Basedef.cpp:3209 deriva MOB.Critical
// deles) — foram arquivados como visuais aqui, e é por isso que equipamento de crit não
// dava nada (issue #102). A maior parte do crit vive no catálogo: os itens de corpo de
// classe e os conjuntos de armadura carregam EF_CRITICAL direto.
//
// EF_RESIST1..4/EF_RESISTALL (ItemEffect.h:106-111) são a mesma classe de bug (issue #211):
// eles TAMBÉM são stats de score (CMob.cpp:640-643 deriva MOB.Resist[0..3] deles via
// BASE_GetMobAbility), então deixá-los fora desta whitelist descartava em silêncio
// o efeito de todo item de imunidade/resist no parse — os itens pareciam certos no
// catálogo mas não davam nada no personagem.
//
// EF_MAGIC/EF_MAGICADD (ItemEffect.h:117,124-125) são a mesma classe de bug de novo (issue #223):
// Basedef.cpp:3194-3195 deriva o score de Magic do conjurador de BASE_GetMobAbility(EF_MAGIC)+
// BASE_GetMobAbility(EF_MAGICADD), então o bônus de um item "Ataque Mágico" era descartado
// em silêncio no parse — o item parecia certo no catálogo mas não dava dano mágico extra.
var efName = map[string]uint8{
	"EF_DAMAGE": 2, "EF_AC": 3, "EF_HP": 4, "EF_MP": 5,
	"EF_STR": 7, "EF_INT": 8, "EF_DEX": 9, "EF_CON": 10,
	"EF_SPECIAL1": 11, "EF_SPECIAL2": 12, "EF_SPECIAL3": 13, "EF_SPECIAL4": 14,
	"EF_SPECIALALL": 74,
	"EF_POS":        17, "EF_WTYPE": 21, "EF_CRITICAL": 42, "EF_SANC": 43,
	"EF_HPADD": 45, "EF_MPADD": 46,
	"EF_RESIST1": 49, "EF_RESIST2": 50, "EF_RESIST3": 51, "EF_RESIST4": 52, "EF_ACADD": 53, "EF_RESISTALL": 54,
	"EF_MAGIC":     60,
	"EF_DAMAGEADD": 67, "EF_MAGICADD": 68, "EF_HPADD2": 69, "EF_MPADD2": 70, "EF_CRITICAL2": 71,
	"EF_ITEMLEVEL": 87, "EF_MOBTYPE": 112, "EF_RUNSPEED": 29,
	// EF_ITEMTYPE também não é stat de score: ele é lido só pelos casadores de
	// combine (GetFunc.cpp:487 Agatha) como gate de receita, exatamente como EF_NOSANC
	// controla o caminho de refino. Sem ele nesta whitelist esses gates leem 0 para
	// todo item e passam/falham em silêncio do jeito errado.
	"EF_ITEMTYPE": 113,
	// Gates de refino (caminho de pó do _MSG_UseItem.cpp): EF_NOSANC marca um item que
	// nunca pode ser refinado; os dois efeitos de incubação dirigem o ramo do ovo de montaria.
	"EF_NOSANC": 126, "EF_INCUBATE": 78, "EF_INCUDELAY": 84,
}

// BaseEffects devolve índice de item → seus efeitos estáticos relevantes ao score, lidos dos
// pares "EF_<name>,value" finais de cada linha do ItemList (o array stEffect da
// linha). Procura tokens EF_ em qualquer ponto da linha (robusto à contagem exata de
// colunas) e lê a coluna seguinte como valor.
func (l *ItemList) BaseEffects() map[int][]BaseEffect {
	out := make(map[int][]BaseEffect, len(l.items))
	for idx, e := range l.items {
		var effs []BaseEffect
		for i := 0; i+1 < len(e.Fields); i++ {
			id, ok := efName[strings.TrimSpace(e.Fields[i])]
			if !ok {
				continue
			}
			v, err := strconv.Atoi(strings.TrimSpace(e.Fields[i+1]))
			if err != nil {
				continue
			}
			effs = append(effs, BaseEffect{Eff: id, Val: int16(v)})
		}
		if len(effs) > 0 {
			out[idx] = effs
		}
	}
	return out
}

// Volatiles devolve índice de item → seu valor EF_VOLATILE (o cValue do par
// EF_VOLATILE). No _MSG_UseItem o servidor classifica a ação por esse valor: 0 = equipável,
// 64/65/66 = Divina 7/15/30d, 58 = Vigor, 1 = poção de HP/MP, etc.
// (captura-wyd-affect-divina.md §B; note que o *id* de EF_VOLATILE é 38).
func (l *ItemList) Volatiles() map[int]int {
	out := make(map[int]int)
	for idx, e := range l.items {
		for i := 0; i+1 < len(e.Fields); i++ {
			if strings.TrimSpace(e.Fields[i]) == "EF_VOLATILE" {
				if v, err := strconv.Atoi(strings.TrimSpace(e.Fields[i+1])); err == nil {
					out[idx] = v
				}
				break
			}
		}
	}
	return out
}

// Ranges devolve índice de item → seu valor EF_RANGE (o alcance de ataque que um
// item equipado concede). O alcance de um mob é o máximo de EF_RANGE sobre os 16 equips
// do template dele (BASE_GetMobAbility → BASE_GetMaxAbility, Basedef.cpp:2415/2523);
// EF_RANGE fica de propósito FORA de efName/BaseEffects — não é stat de score e não
// pode entrar no CurrentScore. Note que EF_RANGE é isento do multiplicador de refino
// (Basedef.cpp:1854).
func (l *ItemList) Ranges() map[int]int16 {
	out := make(map[int]int16)
	for idx, e := range l.items {
		for i := 0; i+1 < len(e.Fields); i++ {
			if strings.TrimSpace(e.Fields[i]) == "EF_RANGE" {
				if v, err := strconv.Atoi(strings.TrimSpace(e.Fields[i+1])); err == nil {
					out[idx] = int16(v)
				}
				break
			}
		}
	}
	return out
}

// Grades devolve índice de item → Grade (STRUCT_ITEMLIST.Grade — coluna 8 do CSV
// pelo sscanf de BASE_ReadItemListFile em Basedef.cpp:5718). UNVERIFIED contra toda linha
// até o content_test fixar amostras; usado para o ExpBonus de grade 7 (+2 por peça).
func (l *ItemList) Grades() map[int]int {
	out := make(map[int]int)
	for idx, e := range l.items {
		if len(e.Fields) > 8 {
			if v, err := strconv.Atoi(strings.TrimSpace(e.Fields[8])); err == nil {
				out[idx] = v
			}
		}
	}
	return out
}

// Positions devolve índice de item → nPos (STRUCT_ITEMLIST.nPos, a classe de slot de
// equipamento — coluna 6 do CSV). nPos dirige os bônus de limiar do refino (+9): armas
// 64/192 somam +40 de dano de arma, peças de defesa 4/8/128 somam +25 de AC (captura §E).
// Confirmado por Garra (arma) nPos=64 e poções nPos=0.
func (l *ItemList) Positions() map[int]int {
	out := make(map[int]int)
	for idx, e := range l.items {
		if len(e.Fields) > 6 {
			if v, err := strconv.Atoi(strings.TrimSpace(e.Fields[6])); err == nil {
				out[idx] = v
			}
		}
	}
	return out
}

// Uniques devolve índice de item → nUnique (STRUCT_ITEMLIST.nUnique — coluna 4 do CSV,
// campo `unique` do sscanf em BASE_ReadItemListFile). nUnique em [41,50] marca os
// itens de joia de dano cujo EF_DAMAGEADD conta no score (captura §B/E) e
// controla a receita base do combine Anct (GetMatchCombine, GetFunc.cpp:94).
func (l *ItemList) Uniques() map[int]int {
	out := make(map[int]int)
	for idx, e := range l.items {
		if len(e.Fields) > 4 {
			if v, err := strconv.Atoi(strings.TrimSpace(e.Fields[4])); err == nil {
				out[idx] = v
			}
		}
	}
	return out
}

// Extras devolve índice de item → Extra (STRUCT_ITEMLIST.Extra — coluna 7 do CSV).
// Extra é o índice base do item resultante usado pelo combine Anct (joia + Extra).
func (l *ItemList) Extras() map[int]int {
	out := make(map[int]int)
	for idx, e := range l.items {
		if len(e.Fields) > 7 {
			if v, err := strconv.Atoi(strings.TrimSpace(e.Fields[7])); err == nil {
				out[idx] = v
			}
		}
	}
	return out
}

// ItemReq é o requisito de equipar de um item (STRUCT_ITEMLIST ReqLvl/Str/Int/Dex/Con).
// Um valor zero significa sem requisito.
type ItemReq struct {
	Lvl, Str, Int, Dex, Con int16
}

// Requirements devolve índice de item → seu requisito de equipar, lido da
// 4ª coluna do CSV separada por pontos "ReqLvl.ReqStr.ReqInt.ReqDex.ReqCon" (a ordem
// das colunas casa com STRUCT_ITEMLIST, confirmado contra armas de guerreiro:
// machados/espadas põem o requisito de STR no 2º valor). Itens sem requisito são
// omitidos.
func (l *ItemList) Requirements() map[int]ItemReq {
	out := make(map[int]ItemReq, len(l.items))
	for idx, e := range l.items {
		if len(e.Fields) < 4 {
			continue
		}
		parts := strings.Split(strings.TrimSpace(e.Fields[3]), ".")
		if len(parts) != 5 {
			continue
		}
		var v [5]int16
		ok := true
		for i, p := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				ok = false
				break
			}
			v[i] = int16(n)
		}
		req := ItemReq{Lvl: v[0], Str: v[1], Int: v[2], Dex: v[3], Con: v[4]}
		if ok && req != (ItemReq{}) {
			out[idx] = req
		}
	}
	return out
}

// NewItemList monta um catálogo a partir de entradas explícitas. É o ponto de entrada do
// catálogo vindo do banco (issue 58, fase 2) e dos testes, que não podem
// depender de tmserver/content/ existir.
func NewItemList(entries []ItemEntry) *ItemList {
	l := &ItemList{items: make(map[int]ItemEntry, len(entries))}
	for _, e := range entries {
		l.items[e.Index] = e
	}
	return l
}

// Entries devolve toda entrada ordenada por índice de item.
func (l *ItemList) Entries() []ItemEntry {
	out := make([]ItemEntry, 0, len(l.items))
	for _, e := range l.items {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

// LoadItemList lê ItemList.csv (índice,Name,...).
func LoadItemList(path string) (*ItemList, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("content: open ItemList: %w", err)
	}
	defer f.Close()
	return parseItemList(f)
}

func parseItemList(r io.Reader) (*ItemList, error) {
	l := &ItemList{items: make(map[int]ItemEntry)}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024) // as linhas podem ser longas
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 2 {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil {
			continue
		}
		l.items[idx] = ItemEntry{Index: idx, Name: strings.TrimSpace(fields[1]), Fields: fields}
	}
	return l, sc.Err()
}
