package worldevents

import "time"

// TowerPhase espelha GTorreState.
type TowerPhase uint8

const (
	TowerIdle TowerPhase = iota
	TowerAnnounced
	TowerOpen
)

// TowerAction é uma transição disparada por timer, emitida por Tower.Step.
type TowerAction uint8

const (
	TowerNone TowerAction = iota
	TowerAnnounce
	TowerStart
	TowerEnd
)

// Tower é a máquina de estados legada de calendário de captura da torre por dia da semana.
type Tower struct {
	Hour  int
	phase TowerPhase
}

// NewTower devolve um evento de torre agendado para a hora informada.
func NewTower(hour int) Tower { return Tower{Hour: hour} }

// Phase reporta a fase atual compatível com GTorreState.
func (t *Tower) Phase() TowerPhase { return t.phase }

// Step avança no máximo uma transição por tick de minuto. A cadeia ordenada é
// leitura consciente de CWarTower.cpp:203/211/224; comportamento de ifs
// independentes no minuto 30 a partir de Announced é UNVERIFIED.
func (t *Tower) Step(now time.Time, enabled bool) TowerAction {
	if !enabled || now.Weekday() == time.Saturday || now.Weekday() == time.Sunday || now.Hour() != t.Hour {
		return TowerNone
	}
	switch {
	case now.Minute() <= 5 && t.phase == TowerIdle:
		t.phase = TowerAnnounced
		return TowerAnnounce
	case now.Minute() >= 6 && t.phase == TowerAnnounced:
		t.phase = TowerOpen
		return TowerStart
	case now.Minute() == 30 && t.phase == TowerOpen:
		t.phase = TowerIdle
		return TowerEnd
	default:
		return TowerNone
	}
}
