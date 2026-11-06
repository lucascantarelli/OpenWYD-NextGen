package bot

import (
	"fmt"
	"math/rand"
	"sort"
)

// ProfileName identifica um perfil comportamental.
type ProfileName string

// Perfis suportados. Cada um só ajusta limiares e pesos: a execução continua
// determinística e reproduzível a partir de uma semente.
const (
	ProfileCasual      ProfileName = "casual"
	ProfileFarmer      ProfileName = "farmer"
	ProfileCombatente  ProfileName = "combatente"
	ProfileComerciante ProfileName = "comerciante"
	ProfileExplorador  ProfileName = "explorador"
	ProfileSocial      ProfileName = "social"
	ProfileQuest       ProfileName = "quest"
	ProfileStress      ProfileName = "stress"
	ProfileBugHunter   ProfileName = "bughunter"
)

// Profile é a parametrização de um bot. Frações vão de 0 a 1; pesos são
// relativos entre si.
type Profile struct {
	Name ProfileName
	// HealBelow é a fração de HP que dispara cura.
	HealBelow float64
	// FleeBelow é a fração de HP que dispara recuo para a cidade.
	FleeBelow float64
	// RestockBelow é o estoque de poções que dispara reabastecimento.
	RestockBelow int
	// InventoryFullAt é o número de espaços livres que conta como mochila cheia.
	InventoryFullAt int
	// Aggro é o peso de procurar alvo em vez de vagar.
	Aggro float64
	// ChatChance é a chance por passo de falar no chat.
	ChatChance float64
	// LootChance é a chance de recolher loot pendente no passo.
	LootChance float64
	// TradeChance é a chance de procurar negociação no passo.
	TradeChance float64
}

// DefaultProfiles devolve a tabela de perfis padrão.
func DefaultProfiles() map[ProfileName]Profile {
	return map[ProfileName]Profile{
		ProfileCasual:      {Name: ProfileCasual, HealBelow: 0.5, FleeBelow: 0.2, RestockBelow: 3, InventoryFullAt: 2, Aggro: 0.3, ChatChance: 0.02, LootChance: 0.5, TradeChance: 0.0},
		ProfileFarmer:      {Name: ProfileFarmer, HealBelow: 0.6, FleeBelow: 0.25, RestockBelow: 5, InventoryFullAt: 3, Aggro: 0.8, ChatChance: 0.01, LootChance: 0.9, TradeChance: 0.05},
		ProfileCombatente:  {Name: ProfileCombatente, HealBelow: 0.45, FleeBelow: 0.15, RestockBelow: 4, InventoryFullAt: 2, Aggro: 0.95, ChatChance: 0.02, LootChance: 0.4, TradeChance: 0.0},
		ProfileComerciante: {Name: ProfileComerciante, HealBelow: 0.7, FleeBelow: 0.35, RestockBelow: 2, InventoryFullAt: 1, Aggro: 0.1, ChatChance: 0.05, LootChance: 0.2, TradeChance: 0.6},
		ProfileExplorador:  {Name: ProfileExplorador, HealBelow: 0.55, FleeBelow: 0.25, RestockBelow: 4, InventoryFullAt: 2, Aggro: 0.4, ChatChance: 0.03, LootChance: 0.5, TradeChance: 0.0},
		ProfileSocial:      {Name: ProfileSocial, HealBelow: 0.6, FleeBelow: 0.3, RestockBelow: 3, InventoryFullAt: 2, Aggro: 0.2, ChatChance: 0.25, LootChance: 0.3, TradeChance: 0.2},
		ProfileQuest:       {Name: ProfileQuest, HealBelow: 0.55, FleeBelow: 0.25, RestockBelow: 4, InventoryFullAt: 2, Aggro: 0.6, ChatChance: 0.02, LootChance: 0.7, TradeChance: 0.05},
		ProfileStress:      {Name: ProfileStress, HealBelow: 0.3, FleeBelow: 0.1, RestockBelow: 8, InventoryFullAt: 1, Aggro: 1.0, ChatChance: 0.0, LootChance: 0.1, TradeChance: 0.0},
		ProfileBugHunter:   {Name: ProfileBugHunter, HealBelow: 0.35, FleeBelow: 0.05, RestockBelow: 6, InventoryFullAt: 1, Aggro: 0.9, ChatChance: 0.0, LootChance: 0.2, TradeChance: 0.3},
	}
}

// LookupProfile devolve o perfil pelo nome, com erro explícito quando não existe.
func LookupProfile(name ProfileName) (Profile, error) {
	p, ok := DefaultProfiles()[name]
	if !ok {
		return Profile{}, fmt.Errorf("bot: perfil %q desconhecido", name)
	}
	return p, nil
}

// Validate recusa limiares fora da faixa em vez de deixar o bot se comportar
// de um jeito impossível de reproduzir.
func (p Profile) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("bot: perfil sem nome")
	}
	for field, v := range map[string]float64{
		"HealBelow":   p.HealBelow,
		"FleeBelow":   p.FleeBelow,
		"Aggro":       p.Aggro,
		"ChatChance":  p.ChatChance,
		"LootChance":  p.LootChance,
		"TradeChance": p.TradeChance,
	} {
		if v < 0 || v > 1 {
			return fmt.Errorf("bot: perfil %s com %s=%v fora de 0..1", p.Name, field, v)
		}
	}
	if p.FleeBelow > p.HealBelow {
		return fmt.Errorf("bot: perfil %s recua (%.2f) depois de curar (%.2f)", p.Name, p.FleeBelow, p.HealBelow)
	}
	if p.RestockBelow < 0 || p.InventoryFullAt < 0 {
		return fmt.Errorf("bot: perfil %s com limiar negativo", p.Name)
	}
	return nil
}

// RNG é a fonte de aleatoriedade controlada do bot: mesma semente, mesma
// sequência de decisões, independentemente da ordem de execução.
type RNG struct {
	r *rand.Rand
}

// NewRNG cria a fonte determinística a partir da semente.
func NewRNG(seed int64) *RNG { return &RNG{r: rand.New(rand.NewSource(seed))} }

// Float64 devolve um valor em [0,1).
func (g *RNG) Float64() float64 { return g.r.Float64() }

// Intn devolve um inteiro em [0,n).
func (g *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return g.r.Intn(n)
}

// Chance devolve true com probabilidade p.
func (g *RNG) Chance(p float64) bool {
	if p <= 0 {
		return false
	}
	if p >= 1 {
		return true
	}
	return g.r.Float64() < p
}

// MixPopulation distribui n bots entre os perfis pedidos, proporcionalmente aos
// pesos, sempre com soma exata em n. Perfis com peso zero nunca aparecem e a
// sobra vai para o perfil de maior peso, de forma determinística.
func MixPopulation(n int, weights map[ProfileName]float64) ([]ProfileName, error) {
	if n <= 0 {
		return nil, fmt.Errorf("bot: população pedida com %d bots", n)
	}
	if len(weights) == 0 {
		return nil, fmt.Errorf("bot: população sem perfis")
	}
	names := make([]string, 0, len(weights))
	total := 0.0
	for name, w := range weights {
		if w < 0 {
			return nil, fmt.Errorf("bot: peso negativo no perfil %q", name)
		}
		if _, err := LookupProfile(name); err != nil {
			return nil, err
		}
		if w == 0 {
			continue
		}
		names = append(names, string(name))
		total += w
	}
	if total == 0 {
		return nil, fmt.Errorf("bot: todos os pesos da população são zero")
	}
	sort.Strings(names)

	counts := make(map[string]int, len(names))
	assigned := 0
	for _, name := range names {
		c := int(float64(n) * weights[ProfileName(name)] / total)
		counts[name] = c
		assigned += c
	}
	// A sobra do truncamento vai para o maior peso (primeiro em ordem
	// alfabética em caso de empate), mantendo o resultado independente de
	// ordem de iteração de mapa.
	order := append([]string(nil), names...)
	sort.SliceStable(order, func(i, j int) bool {
		wi, wj := weights[ProfileName(order[i])], weights[ProfileName(order[j])]
		if wi != wj {
			return wi > wj
		}
		return order[i] < order[j]
	})
	for i := 0; assigned < n; i++ {
		counts[order[i%len(order)]]++
		assigned++
	}

	out := make([]ProfileName, 0, n)
	for _, name := range names {
		for i := 0; i < counts[name]; i++ {
			out = append(out, ProfileName(name))
		}
	}
	return out, nil
}
