package world

// NightmareCycle é o estado transitório de evento de um mapa compartilhado. OpeningUnix identifica
// uma ocorrência para ticks atrasados/repetidos não invocarem nem admitirem duas vezes.
type NightmareCycle struct {
	OpeningUnix int64
	Available   bool
	Started     bool
	Finished    bool
	Admissions  int
}

// NightmareState contém os três ciclos de posse do loop (Normal, Mystic, Arcane).
type NightmareState struct {
	Initialized bool
	Cycles      [3]NightmareCycle
}

// NightmareState devolve um snapshot por valor. Só no loop.
func (w *World) NightmareState() NightmareState { return w.nightmare }

// SetNightmareState substitui o estado transitório dos ciclos. Só no loop.
func (w *World) SetNightmareState(st NightmareState) { w.nightmare = st }

// NightmareGenerator devolve o tipo de evento de um gerador legado, ou -1.
func NightmareGenerator(idx int) int {
	switch {
	case idx >= 2368 && idx <= 2375:
		return 0
	case idx >= 2377 && idx <= 2384:
		return 1
	case idx >= 2385 && idx <= 2394:
		return 2
	default:
		return -1
	}
}

// NightmareMap devolve o tipo de evento de um mapa legado de 128 tiles, ou -1.
func NightmareMap(x, y int16) int {
	switch {
	case x/128 == 10 && y/128 == 2:
		return 0
	case x/128 == 8 && y/128 == 2:
		return 1
	case x/128 == 9 && y/128 == 1:
		return 2
	default:
		return -1
	}
}
