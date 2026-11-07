package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Limites de execução. São deliberadamente baixos: o servidor de jogo é um
// único laço, então carga alta precisa de ambiente separado, não de um
// "só mais 500 bots" numa stack em uso.
const (
	// MaxBotsPerScenario é o teto de bots num cenário.
	MaxBotsPerScenario = 2000
	// MaxDurationSeconds é o teto de duração de um cenário.
	MaxDurationSeconds = 24 * 3600
)

// Spec é um bot declarado no cenário.
type Spec struct {
	// Account é a conta de teste; precisa existir no banco.
	Account string `json:"account"`
	// Password é a senha da conta.
	Password string `json:"password"`
	// Slot é o personagem a selecionar.
	Slot int `json:"slot"`
	// Profile é o nome do perfil comportamental.
	Profile ProfileName `json:"profile"`
	// Region é a região de caça pretendida ("armia", "noatun", …). Obrigatória
	// para perfis que caçam e vazia para os que não caçam.
	Region string `json:"region,omitempty"`
}

// Condition é um critério de sucesso ou de falha sobre uma métrica do relatório.
type Condition struct {
	Metric string  `json:"metric"`
	Op     string  `json:"op"`
	Value  float64 `json:"value"`
}

// Scenario é um cenário versionado: o que roda, por quanto tempo, com qual
// semente e como se sabe que passou.
type Scenario struct {
	// ID identifica o cenário e nomeia a execução nos logs.
	ID string `json:"id"`
	// Objective explica em uma frase o que o cenário prova.
	Objective string `json:"objective"`
	// Seed fixa a sequência de decisões dos bots.
	Seed int64 `json:"seed"`
	// DurationSeconds é o tempo de execução do cenário.
	DurationSeconds int `json:"duration_seconds"`
	// TickMillis é o intervalo entre passos de decisão de cada bot.
	TickMillis int `json:"tick_millis"`
	// Address é o endereço do tmServer; precisa ser local ou estar liberado.
	Address string `json:"address"`
	// Bots é a população.
	Bots []Spec `json:"bots"`
	// Success e Failure são avaliados sobre o relatório final.
	Success []Condition `json:"success,omitempty"`
	Failure []Condition `json:"failure,omitempty"`
}

// métricas que os cenários podem citar em Success/Failure.
var knownMetrics = map[string]struct{}{
	"bots_connected": {},
	"bots_failed":    {},
	"bots_total":     {},
	"actions":        {},
	"errors":         {},
	"frames":         {},
	"states_visited": {},
	"p99_ms":         {},
	"ticks":          {},
}

// operadores aceitos nos critérios.
var knownOps = map[string]struct{}{
	">": {}, ">=": {}, "<": {}, "<=": {}, "==": {}, "!=": {},
}

// LoadScenario lê e valida um cenário em JSON.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("bot: ler cenário %s: %w", path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	var s Scenario
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("bot: cenário %s inválido: %w", path, err)
	}
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("bot: cenário %s: %w", path, err)
	}
	return &s, nil
}

// Validate recusa o cenário que não pode rodar ou que não pode ser julgado.
func (s *Scenario) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("id é obrigatório")
	}
	if strings.TrimSpace(s.Objective) == "" {
		return fmt.Errorf("objective é obrigatório")
	}
	if strings.TrimSpace(s.Address) == "" {
		return fmt.Errorf("address é obrigatório")
	}
	if s.DurationSeconds <= 0 || s.DurationSeconds > MaxDurationSeconds {
		return fmt.Errorf("duration_seconds deve ficar em 1..%d (veio %d)", MaxDurationSeconds, s.DurationSeconds)
	}
	if s.TickMillis <= 0 {
		return fmt.Errorf("tick_millis deve ser positivo (veio %d)", s.TickMillis)
	}
	if len(s.Bots) == 0 || len(s.Bots) > MaxBotsPerScenario {
		return fmt.Errorf("bots deve ficar em 1..%d (veio %d)", MaxBotsPerScenario, len(s.Bots))
	}

	accounts := make(map[string]struct{}, len(s.Bots))
	for i, b := range s.Bots {
		if strings.TrimSpace(b.Account) == "" {
			return fmt.Errorf("bots[%d]: account é obrigatório", i)
		}
		if len(b.Account) > maxAccountLen {
			return fmt.Errorf("bots[%d]: account %q excede %d bytes", i, b.Account, maxAccountLen)
		}
		if len(b.Password) > maxPasswordLen {
			return fmt.Errorf("bots[%d]: senha excede %d bytes", i, maxPasswordLen)
		}
		if b.Slot < 0 {
			return fmt.Errorf("bots[%d]: slot negativo (%d)", i, b.Slot)
		}
		if _, err := LookupProfile(b.Profile); err != nil {
			return fmt.Errorf("bots[%d]: %w", i, err)
		}
		if _, dup := accounts[b.Account]; dup {
			return fmt.Errorf("bots[%d]: conta %q repetida no cenário", i, b.Account)
		}
		accounts[b.Account] = struct{}{}
	}

	for _, group := range []struct {
		name  string
		items []Condition
	}{{"success", s.Success}, {"failure", s.Failure}} {
		for i, c := range group.items {
			if _, ok := knownMetrics[c.Metric]; !ok {
				return fmt.Errorf("%s[%d]: métrica %q desconhecida", group.name, i, c.Metric)
			}
			if _, ok := knownOps[c.Op]; !ok {
				return fmt.Errorf("%s[%d]: operador %q desconhecido", group.name, i, c.Op)
			}
		}
	}
	return nil
}

// KnownMetrics devolve os nomes de métrica que os cenários podem usar, ordenados.
func KnownMetrics() []string {
	out := make([]string, 0, len(knownMetrics))
	for name := range knownMetrics {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Evaluate aplica as condições sobre o relatório. Devolve true quando todas
// passam; devolve, para cada condição falsa, a explicação do valor observado.
func Evaluate(conds []Condition, r Report) (bool, []string) {
	var failures []string
	for _, c := range conds {
		got, ok := r.metric(c.Metric)
		if !ok {
			failures = append(failures, fmt.Sprintf("%s: métrica indisponível", c.Metric))
			continue
		}
		if !compare(got, c.Op, c.Value) {
			failures = append(failures, fmt.Sprintf("%s=%g não satisfaz %s %g", c.Metric, got, c.Op, c.Value))
		}
	}
	return len(failures) == 0, failures
}

// compare aplica o operador com tolerância zero: são contadores inteiros ou
// percentis calculados, não medidas de ponto flutuante acumuladas.
func compare(got float64, op string, want float64) bool {
	switch op {
	case ">":
		return got > want
	case ">=":
		return got >= want
	case "<":
		return got < want
	case "<=":
		return got <= want
	case "==":
		return got == want
	case "!=":
		return got != want
	}
	return false
}
