package bot

import (
	"os"
	"path/filepath"
	"testing"
)

func writeScenario(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cenario.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("escrever cenário: %v", err)
	}
	return path
}

const validScenario = `{
  "id": "smoke",
  "objective": "provar login de um bot",
  "seed": 11,
  "duration_seconds": 5,
  "tick_millis": 100,
  "address": "127.0.0.1:8281",
  "bots": [{"account": "bot1", "password": "bot123", "slot": 0, "profile": "farmer", "region": "armia"}],
  "success": [{"metric": "bots_connected", "op": "==", "value": 1}],
  "failure": [{"metric": "bots_failed", "op": ">", "value": 0}]
}`

func TestLoadScenarioAcceptsValidFile(t *testing.T) {
	s, err := LoadScenario(writeScenario(t, validScenario))
	if err != nil {
		t.Fatalf("LoadScenario: %v", err)
	}
	if s.ID != "smoke" || len(s.Bots) != 1 || s.Bots[0].Profile != ProfileFarmer {
		t.Fatalf("cenário lido errado: %+v", s)
	}
}

func TestScenarioRejectsUnknownField(t *testing.T) {
	bad := `{"id":"x","objective":"y","seed":1,"duration_seconds":5,"tick_millis":100,
	         "address":"127.0.0.1:1","bots":[{"account":"a","password":"b","slot":0,"profile":"farmer"}],
	         "extra":true}`
	if _, err := LoadScenario(writeScenario(t, bad)); err == nil {
		t.Fatal("campo desconhecido foi aceito")
	}
}

func TestScenarioValidation(t *testing.T) {
	base := func() *Scenario {
		s, err := LoadScenario(writeScenario(t, validScenario))
		if err != nil {
			t.Fatalf("cenário base: %v", err)
		}
		return s
	}

	cases := []struct {
		name string
		mut  func(*Scenario)
	}{
		{"sem id", func(s *Scenario) { s.ID = "" }},
		{"sem objetivo", func(s *Scenario) { s.Objective = "" }},
		{"sem endereço", func(s *Scenario) { s.Address = "" }},
		{"duração zero", func(s *Scenario) { s.DurationSeconds = 0 }},
		{"duração acima do teto", func(s *Scenario) { s.DurationSeconds = MaxDurationSeconds + 1 }},
		{"tick zero", func(s *Scenario) { s.TickMillis = 0 }},
		{"sem bots", func(s *Scenario) { s.Bots = nil }},
		{"perfil desconhecido", func(s *Scenario) { s.Bots[0].Profile = "ninja" }},
		{"conta repetida", func(s *Scenario) { s.Bots = append(s.Bots, s.Bots[0]) }},
		{"slot negativo", func(s *Scenario) { s.Bots[0].Slot = -1 }},
		{"conta longa", func(s *Scenario) { s.Bots[0].Account = "0123456789abcdef" }},
		{"métrica desconhecida", func(s *Scenario) { s.Success[0].Metric = "lucro" }},
		{"operador desconhecido", func(s *Scenario) { s.Success[0].Op = "~=" }},
	}
	for _, c := range cases {
		s := base()
		c.mut(s)
		if err := s.Validate(); err == nil {
			t.Fatalf("%s: cenário inválido foi aceito", c.name)
		}
	}
}

func TestNewRunnerRefusesRemoteTarget(t *testing.T) {
	s, err := LoadScenario(writeScenario(t, validScenario))
	if err != nil {
		t.Fatalf("cenário: %v", err)
	}
	s.Address = "200.1.2.3:8281"
	if _, err := NewRunner(s, Options{}); err == nil {
		t.Fatal("alvo público foi aceito sem -allow-remote")
	}
	if _, err := NewRunner(s, Options{AllowRemote: true}); err != nil {
		t.Fatalf("alvo remoto liberado recusado: %v", err)
	}
}

func TestNewRunnerAcceptsLoopback(t *testing.T) {
	s, err := LoadScenario(writeScenario(t, validScenario))
	if err != nil {
		t.Fatalf("cenário: %v", err)
	}
	for _, addr := range []string{"127.0.0.1:8281", "[::1]:8281", "localhost:8281"} {
		s.Address = addr
		if _, err := NewRunner(s, Options{}); err != nil {
			t.Fatalf("%s recusado: %v", addr, err)
		}
	}
}

func TestEvaluateReportsObservedValues(t *testing.T) {
	rep := Report{Bots: 2, Connected: 1, Failed: 1}
	ok, why := Evaluate([]Condition{{Metric: "bots_connected", Op: "==", Value: 2}}, rep)
	if ok || len(why) != 1 {
		t.Fatalf("critério deveria falhar: ok=%v why=%v", ok, why)
	}
	ok, _ = Evaluate([]Condition{
		{Metric: "bots_total", Op: "==", Value: 2},
		{Metric: "bots_failed", Op: ">=", Value: 1},
	}, rep)
	if !ok {
		t.Fatal("critérios satisfeitos foram reprovados")
	}
	if _, why := Evaluate([]Condition{{Metric: "inexistente", Op: ">", Value: 0}}, rep); len(why) != 1 {
		t.Fatal("métrica inexistente não foi reportada")
	}
}
