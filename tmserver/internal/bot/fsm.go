package bot

import (
	"fmt"
	"sort"
)

// State é um estado do comportamento do bot. Os nomes são os do documento de
// arquitetura (IDLE, CONNECTING, HUNTING, …) e servem de valor de métrica.
type State string

// Estados canônicos do comportamento.
const (
	StateIdle            State = "IDLE"
	StateConnecting      State = "CONNECTING"
	StateAuthenticating  State = "AUTHENTICATING"
	StateSelectingChar   State = "SELECTING_CHARACTER"
	StateEnteringWorld   State = "ENTERING_WORLD"
	StateMovingToHunt    State = "MOVING_TO_HUNT"
	StateHunting         State = "HUNTING"
	StateAttacking       State = "ATTACKING"
	StateLooting         State = "LOOTING"
	StateHealing         State = "HEALING"
	StateReturningToTown State = "RETURNING_TO_TOWN"
	StateSellingItems    State = "SELLING_ITEMS"
	StateRestocking      State = "RESTOCKING"
	StateTrading         State = "TRADING"
	StatePartyForming    State = "PARTY_FORMING"
	StateQuesting        State = "QUESTING"
	StateGuildActivity   State = "GUILD_ACTIVITY"
	StateDead            State = "DEAD"
	StateRespawning      State = "RESPAWNING"
	StateDisconnecting   State = "DISCONNECTING"
)

// Event é um estímulo que pode provocar transição.
type Event string

// Eventos canônicos.
const (
	EventConnected      Event = "CONNECTED"
	EventAuthenticated  Event = "AUTHENTICATED"
	EventCharacterReady Event = "CHARACTER_READY"
	EventWorldEntered   Event = "WORLD_ENTERED"
	EventReachedArea    Event = "REACHED_AREA"
	EventTargetFound    Event = "TARGET_FOUND"
	EventTargetDown     Event = "TARGET_DOWN"
	EventNoTarget       Event = "NO_TARGET"
	EventHpLow          Event = "HP_LOW"
	EventMpLow          Event = "MP_LOW"
	EventInventoryFull  Event = "INVENTORY_FULL"
	EventPotionMissing  Event = "POTION_MISSING"
	EventLootPending    Event = "LOOT_PENDING"
	EventHealed         Event = "HEALED"
	EventInTown         Event = "IN_TOWN"
	EventSold           Event = "SOLD"
	EventRestocked      Event = "RESTOCKED"
	EventDied           Event = "DIED"
	EventRespawned      Event = "RESPAWNED"
	EventFinished       Event = "FINISHED"
	EventStop           Event = "STOP"
)

// Context é o estado observável do bot num instante. É o que guardas e ações
// leem e escrevem; campos numéricos são do jogo, não do executor.
type Context struct {
	// Slot é o personagem selecionado.
	Slot int
	// HP e MP são os valores atuais; HPMax e MPMax, os máximos.
	HP, HPmax int32
	MP, MPmax int32
	// InventoryFree é quantos espaços livres restam.
	InventoryFree int
	// Potions é o estoque de consumíveis.
	Potions int
	// Gold é o ouro carregado.
	Gold int64
	// Target é o id do alvo atual; zero quando não há alvo.
	Target int32
	// Loot é quantos itens aguardam coleta no chão.
	Loot int
	// Ticks é quantos passos o bot já executou.
	Ticks int
	// LastError é o último erro não fatal observado.
	LastError string
}

// HPFraction devolve a fração de HP entre 0 e 1, saturando no máximo.
func (c *Context) HPFraction() float64 {
	if c.HPmax <= 0 {
		return 0
	}
	f := float64(c.HP) / float64(c.HPmax)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// MPFraction devolve a fração de MP entre 0 e 1, saturando no máximo.
func (c *Context) MPFraction() float64 {
	if c.MPmax <= 0 {
		return 0
	}
	f := float64(c.MP) / float64(c.MPmax)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// Guard decide se uma transição pode acontecer.
type Guard func(*Context) bool

// Action é o efeito da transição. Devolver erro mantém o bot vivo: o erro vai
// para o relatório e para o contador de erros, sem derrubar o cenário.
type Action func(*Context) error

// Transition é uma aresta da máquina de estados. A ordem declarada é a ordem
// de avaliação quando mais de uma aresta casa o mesmo par (estado, evento).
type Transition struct {
	From   State
	Event  Event
	To     State
	Guard  Guard
	Action Action
	// Name é o rótulo curto que aparece no relatório e nas métricas.
	Name string
}

// Record é uma transição efetivamente executada, para reconstruir a execução.
type Record struct {
	Step  int
	From  State
	To    State
	Event Event
	Name  string
}

// ErrNoTransition indica que nenhuma aresta casou o par (estado, evento).
type ErrNoTransition struct {
	From  State
	Event Event
}

// Error implementa error.
func (e ErrNoTransition) Error() string {
	return fmt.Sprintf("bot: sem transição de %s com %s", e.From, e.Event)
}

// Machine é a FSM de um bot: estado atual, tabela de transições e histórico.
type Machine struct {
	initial State
	state   State
	table   map[[2]string][]Transition
	history []Record
}

// NewMachine valida a tabela e cria a máquina no estado inicial.
func NewMachine(initial State, transitions []Transition) (*Machine, error) {
	if initial == "" {
		return nil, fmt.Errorf("bot: FSM sem estado inicial")
	}
	table := make(map[[2]string][]Transition, len(transitions))
	for i, t := range transitions {
		if t.From == "" || t.Event == "" || t.To == "" {
			return nil, fmt.Errorf("bot: transição %d incompleta (%s -%s-> %s)", i, t.From, t.Event, t.To)
		}
		key := [2]string{string(t.From), string(t.Event)}
		table[key] = append(table[key], t)
	}
	return &Machine{initial: initial, state: initial, table: table}, nil
}

// State devolve o estado atual.
func (m *Machine) State() State { return m.state }

// History devolve o histórico de transições executadas.
func (m *Machine) History() []Record {
	out := make([]Record, len(m.history))
	copy(out, m.history)
	return out
}

// Reset volta ao estado inicial e limpa o histórico.
func (m *Machine) Reset() {
	m.state = m.initial
	m.history = m.history[:0]
}

// Fire aplica o primeiro evento cuja guarda passe. Sem aresta aplicável devolve
// ErrNoTransition sem mexer no estado — o chamador decide se é fim de roteiro.
func (m *Machine) Fire(ev Event, ctx *Context) (Transition, error) {
	candidates := m.table[[2]string{string(m.state), string(ev)}]
	for _, t := range candidates {
		if t.Guard != nil && !t.Guard(ctx) {
			continue
		}
		from := m.state
		m.state = t.To
		m.history = append(m.history, Record{
			Step:  len(m.history) + 1,
			From:  from,
			To:    t.To,
			Event: ev,
			Name:  t.Name,
		})
		return t, nil
	}
	return Transition{}, ErrNoTransition{From: m.state, Event: ev}
}

// States devolve os estados alcançáveis, ordenados, para o relatório.
func (m *Machine) States() []State {
	seen := map[State]struct{}{m.initial: {}}
	for _, list := range m.table {
		for _, t := range list {
			seen[t.From] = struct{}{}
			seen[t.To] = struct{}{}
		}
	}
	out := make([]State, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
