package world

// EventConfig é o estado global de evento de posse do loop, aplicado a partir da
// config do portal. Version é a versão de config do dbServer que produziu o snapshot e
// é usada ao persistir progresso para evitar escritas obsoletas após edições de moderador.
type EventConfig struct {
	Version       int64
	Enabled       bool
	ItemIndex     int32
	Rate          int32
	StartIndex    int32
	CurrentIndex  int32
	EndIndex      int32
	Indexed       bool
	NoticeEnabled bool
}

// SetWorldEventConfig substitui o estado global de evento. Só no loop.
func (w *World) SetWorldEventConfig(cfg EventConfig) {
	w.worldEvent = cfg
}

// WorldEventConfig devolve uma cópia do estado global de evento. Só no loop.
func (w *World) WorldEventConfig() EventConfig {
	return w.worldEvent
}

// newbieHPLevelCap é o nível abaixo do qual o evento de novato reduz o
// HP de spawn de uma criatura (Server.cpp:3326, 3616, 3755).
const newbieHPLevelCap = 120

// SetNewbieEvent alterna o handicap do NewbieEventServer. Só no loop; seguro
// para chamar no boot (antes de Serve) ou do tick quando a config do portal muda.
func (w *World) SetNewbieEvent(on bool) { w.newbieEvent = on }

// NewbieEvent informa se o handicap do evento de novato está ativo. Só no loop.
func (w *World) NewbieEvent() bool { return w.newbieEvent }

// applyNewbieHandicap invoca criaturas abaixo do nível 120 com 3/4 do HP durante o evento de novato
// (Server.cpp:3326-3327 e os dois caminhos de spawn irmãos).
//
// Note que ele escala só o HP ATUAL — MaxHp não é tocado, então uma criatura com handicap
// regenera até o total. Essa assimetria é do legado, não um descuido.
func (w *World) applyNewbieHandicap(e *Entity) {
	if !w.newbieEvent || e.Level >= newbieHPLevelCap {
		return
	}
	e.HP = 3 * e.HP / 4
}
