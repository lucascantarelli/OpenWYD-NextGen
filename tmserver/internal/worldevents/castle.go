package worldevents

// Castle é o estado puro de contagem regressiva e limpeza atrasada de Castelo/Zakum.
type Castle struct {
	active    bool
	level     int
	timeLeft  int32
	clear     uint8
	secondOdd bool
}

// Open inicia uma quest. questTime é contado em unidades legadas de dois segundos.
func (c *Castle) Open(level int, questTime int32) {
	c.active, c.level, c.timeLeft, c.clear, c.secondOdd = true, level, questTime-1, 0, false
}

// Restore reidrata um snapshot persistido de quest ativa.
func (c *Castle) Restore(level int, timeLeft int32, cleared bool) {
	c.active, c.level, c.timeLeft, c.secondOdd = true, level, timeLeft, false
	if cleared {
		c.clear = 1
	} else {
		c.clear = 0
	}
}

// TickSecond avança a contagem de dois segundos e reporta a expiração.
func (c *Castle) TickSecond() bool {
	if !c.active || c.timeLeft < 0 {
		return false
	}
	c.secondOdd = !c.secondOdd
	if c.secondOdd {
		return false
	}
	if c.timeLeft == 0 {
		c.active, c.timeLeft, c.level = false, -1, -1
		return true
	}
	c.timeLeft--
	return false
}

// MarkClear inicia a limpeza legada com atraso de 1→2→0 minuto.
func (c *Castle) MarkClear() { c.clear = 1 }

// TickMinute avança o estado de limpeza e reporta quando a limpeza venceu.
func (c *Castle) TickMinute() bool {
	switch c.clear {
	case 1:
		c.clear = 2
	case 2:
		c.active, c.clear, c.level, c.timeLeft = false, 0, -1, -1
		return true
	}
	return false
}

// State devolve o nível ativo, as unidades de dois segundos restantes e a fase de limpeza.
func (c *Castle) State() (int, int32, uint8) {
	if !c.active {
		return -1, -1, c.clear
	}
	return c.level, c.timeLeft, c.clear
}
