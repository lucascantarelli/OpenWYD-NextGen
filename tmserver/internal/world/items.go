package world

// Effect é um dos três pares efeito/valor de um item (STRUCT_ITEM, data-formats
// §1.5). Espelho local do mundo, separado de internal/savefmt de propósito: este
// tipo sustenta o caminho quente de wire/mundo, enquanto internal/savefmt modela o
// layout de arquivo de save em disco (preocupações diferentes, mesmos bytes).
type Effect struct {
	Effect uint8
	Value  uint8
}

// Item é um STRUCT_ITEM em memória. Index==0 significa slot vazio.
type Item struct {
	Index   int16
	Effects [3]Effect
	// ExpiresAt é a expiração em segundos Unix para itens temporizados (0 = permanente). Ele é
	// levado pela persistência; o mundo descarta o item quando ela passa.
	ExpiresAt int64
}

// Empty informa se o slot não contém item.
func (it Item) Empty() bool { return it.Index == 0 }

// Tipos de contêiner de item (ITEM_PLACE_*), confirmados em Basedef.h:146-148.
const (
	ItemPlaceEquip = 0
	ItemPlaceCarry = 1
	ItemPlaceCargo = 2
)

// Estados de portão de um objeto estático do mundo (GroundItem.State), Basedef.h:2209/2211. Um
// portão semeado começa StateOpen (paridade com CreateItem, Server.cpp:8075). Sem tipo para
// atribuir tanto ao GroundItem.State int16 quanto ao campo State int32 do wire.
const (
	StateOpen   = 1
	StateLocked = 3
)

// GroundItem é um item no chão (CItem / pItem[], domain-model.md
// §2.3): conteúdo, posição e um flag de presença. Mode==0 significa slot livre.
//
// O mesmo espaço de ids também contém objetos estáticos do mundo (portões/portas de
// InitItem.csv): State carrega o estado do portão (veja gate.go) e Static os marca
// para que os caminhos de coleta/decaimento os deixem em paz. Para um item largado State é 0
// e Static é false.
type GroundItem struct {
	ID     int
	Item   Item
	X      int16
	Y      int16
	Mode   int
	State  int16 // estado de portão para objetos estáticos do mundo (STATE_OPEN/STATE_LOCKED); 0 para itens largados
	Static bool  // true para portões/portas semeados no boot: nunca coletados nem decaídos
}

// CreateGroundItem coloca item no chão em (x,y) e o indexa na
// grade espacial. Devolve o novo id de chão (∈ [1, MaxItem)) ou -1 se o chão
// está cheio. Só no loop.
func (w *World) CreateGroundItem(item Item, x, y int16) int {
	for id := 1; id < MaxItem; id++ {
		if w.ground[id] == nil {
			w.ground[id] = &GroundItem{ID: id, Item: item, X: x, Y: y, Mode: 1}
			w.grid.SetItem(int(x), int(y), uint16(id))
			return id
		}
	}
	return -1
}

// SeedWorldItem coloca um objeto estático do mundo (um portão/porta de InitItem.csv) no
// espaço de ids de chão em (x,y) com um estado inicial de portão, devolvendo seu id ou -1
// quando a tabela está cheia. Diferente de CreateGroundItem, ele não indexa a
// grade — portões são buscados por id, não por posição, e não podem sombrear uma
// célula de drop. Precisa rodar no boot (antes de Serve, single-threaded) para os ids serem atribuídos em
// ordem de arquivo e baterem com a sequência legada de CreateItem que o cliente espera.
func (w *World) SeedWorldItem(item Item, x, y, state int16) int {
	for id := 1; id < MaxItem; id++ {
		if w.ground[id] == nil {
			w.ground[id] = &GroundItem{ID: id, Item: item, X: x, Y: y, Mode: 1, State: state, Static: true}
			return id
		}
	}
	return -1
}

// GroundItem devolve o item de chão com o id dado, ou nil.
func (w *World) GroundItem(id int) *GroundItem {
	if id <= 0 || id >= MaxItem {
		return nil
	}
	return w.ground[id]
}

// GroundItemAt devolve o item de chão indexado em (x,y), ou nil quando a célula está
// vazia/fora dos limites.
func (w *World) GroundItemAt(x, y int16) *GroundItem {
	if int(x) < 0 || int(y) < 0 || int(x) >= w.grid.dim || int(y) >= w.grid.dim {
		return nil
	}
	id, ok := w.grid.ItemAt(int(x), int(y))
	if !ok {
		return nil
	}
	return w.GroundItem(int(id))
}

// RemoveGroundItem limpa um item de chão e sua célula de grade. Este é o ponto
// de reivindicação atômico: como roda na única goroutine do loop, dois eventos GetItem
// para o mesmo id são serializados e só o primeiro tem sucesso (sem duplicação). Só no loop.
func (w *World) RemoveGroundItem(id int) {
	gi := w.GroundItem(id)
	if gi == nil {
		return
	}
	if cur, ok := w.grid.ItemAt(int(gi.X), int(gi.Y)); ok && int(cur) == id {
		w.grid.ClearItem(int(gi.X), int(gi.Y))
	}
	w.ground[id] = nil
}

// AddToCarry coloca item no primeiro slot vazio do inventário da entidade, devolvendo o
// índice do slot, ou -1 se o inventário está cheio. Só no loop.
func (w *World) AddToCarry(e *Entity, item Item) int {
	return addFirstEmpty(e.Carry[:], item)
}

// AddToCargo coloca item no primeiro slot vazio do armazém da conta, devolvendo o
// índice do slot, ou -1 se o cargo está cheio. É o destino de entrega da
// loja de doação web (issue #34): uma compra cai no próximo slot livre do cargo, e
// um cargo cheio (-1) significa que o item é perdido. Só no loop.
func (w *World) AddToCargo(st *CargoState, item Item) int {
	if st == nil {
		return -1
	}
	return addFirstEmpty(st.Items[:], item)
}

// addFirstEmpty coloca item no primeiro slot vazio de items, devolvendo seu
// índice, ou -1 se todos os slots estão ocupados.
func addFirstEmpty(items []Item, item Item) int {
	for i := range items {
		if items[i].Empty() {
			items[i] = item
			return i
		}
	}
	return -1
}
