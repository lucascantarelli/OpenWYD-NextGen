package bot

import (
	"errors"
	"testing"
)

func TestMachineRejectsIncompleteTransition(t *testing.T) {
	_, err := NewMachine(StateIdle, []Transition{{From: StateIdle, Event: EventStop}})
	if err == nil {
		t.Fatal("transição sem destino foi aceita")
	}
}

func TestMachineFiresAndRecordsHistory(t *testing.T) {
	m, err := NewMachine(StateConnecting, []Transition{
		{From: StateConnecting, Event: EventConnected, To: StateAuthenticating, Name: "connect"},
		{From: StateAuthenticating, Event: EventAuthenticated, To: StateSelectingChar, Name: "auth"},
	})
	if err != nil {
		t.Fatalf("NewMachine: %v", err)
	}
	ctx := &Context{}
	if tr, err := m.Fire(EventConnected, ctx); err != nil || tr.To != StateAuthenticating {
		t.Fatalf("Fire(EventConnected) = %v, %v", tr.To, err)
	}
	if _, err := m.Fire(EventAuthenticated, ctx); err != nil {
		t.Fatalf("Fire(EventAuthenticated): %v", err)
	}
	if m.State() != StateSelectingChar {
		t.Fatalf("estado = %s, quer %s", m.State(), StateSelectingChar)
	}
	h := m.History()
	if len(h) != 2 || h[0].From != StateConnecting || h[1].To != StateSelectingChar {
		t.Fatalf("histórico = %+v", h)
	}
	if h[0].Step != 1 || h[1].Step != 2 {
		t.Fatalf("passos do histórico fora de ordem: %+v", h)
	}
}

func TestMachineNoTransitionKeepsState(t *testing.T) {
	m, err := NewMachine(StateIdle, []Transition{
		{From: StateIdle, Event: EventStop, To: StateDisconnecting},
	})
	if err != nil {
		t.Fatalf("NewMachine: %v", err)
	}
	_, err = m.Fire(EventDied, &Context{})
	var nt ErrNoTransition
	if !errors.As(err, &nt) {
		t.Fatalf("erro = %v, quer ErrNoTransition", err)
	}
	if m.State() != StateIdle {
		t.Fatalf("estado mudou sem transição: %s", m.State())
	}
}

func TestMachineGuardOrderSelectsFirstApplicable(t *testing.T) {
	m, err := NewMachine(StateAttacking, []Transition{
		{From: StateAttacking, Event: EventTargetDown, To: StateLooting, Name: "com-loot",
			Guard: func(c *Context) bool { return c.Loot > 0 }},
		{From: StateAttacking, Event: EventTargetDown, To: StateHunting, Name: "sem-loot"},
	})
	if err != nil {
		t.Fatalf("NewMachine: %v", err)
	}
	if tr, _ := m.Fire(EventTargetDown, &Context{Loot: 2}); tr.Name != "com-loot" {
		t.Fatalf("com loot escolheu %q", tr.Name)
	}
	m.Reset()
	if tr, _ := m.Fire(EventTargetDown, &Context{}); tr.Name != "sem-loot" {
		t.Fatalf("sem loot escolheu %q", tr.Name)
	}
}

func TestMachineResetClearsHistory(t *testing.T) {
	m, _ := NewMachine(StateIdle, []Transition{{From: StateIdle, Event: EventStop, To: StateDisconnecting}})
	if _, err := m.Fire(EventStop, &Context{}); err != nil {
		t.Fatalf("Fire: %v", err)
	}
	m.Reset()
	if m.State() != StateIdle || len(m.History()) != 0 {
		t.Fatalf("Reset não voltou ao inicial: estado=%s histórico=%d", m.State(), len(m.History()))
	}
}

func TestDefaultTransitionsCoverLifecycle(t *testing.T) {
	m, err := NewMachine(StateConnecting, DefaultTransitions(NewMetrics()))
	if err != nil {
		t.Fatalf("tabela padrão inválida: %v", err)
	}
	states := m.States()
	want := []State{StateConnecting, StateAuthenticating, StateSelectingChar, StateEnteringWorld, StateMovingToHunt}
	for _, w := range want {
		found := false
		for _, s := range states {
			if s == w {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("estado %s ausente da tabela padrão (tem %v)", w, states)
		}
	}
}

func TestContextFractionsSaturate(t *testing.T) {
	c := &Context{HP: 150, HPmax: 100, MP: -10, MPmax: 50}
	if c.HPFraction() != 1 {
		t.Fatalf("HPFraction = %v", c.HPFraction())
	}
	if c.MPFraction() != 0 {
		t.Fatalf("MPFraction = %v", c.MPFraction())
	}
	if (&Context{}).HPFraction() != 0 {
		t.Fatal("divisão por zero não saturou em 0")
	}
}
