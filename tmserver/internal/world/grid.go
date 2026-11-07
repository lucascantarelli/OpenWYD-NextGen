package world

// gridEmpty é o sentinela de célula vazia. O pMobGrid/pItemGrid original
// guarda um índice ushort por célula (domain-model.md §4); como o índice 0 é uma
// conexão de jogador válida, uma célula vazia precisa de sentinela não zero. Aqui se usa 0xFFFF.
//
// UNVERIFIED: o sentinela exato de vazio do original não está documentado; confirme
// contra uma captura/build antes de confiar nele para paridade de visão/colisão.
const gridEmpty uint16 = 0xFFFF

// Grid é o índice espacial denso de criaturas e itens de chão por célula
// (pMobGrid/pItemGrid, domain-model.md §4). É estado derivado mantido em sincronia
// com as posições das entidades pelo loop; não é seguro para uso concorrente.
type Grid struct {
	dim  int
	mob  []uint16
	item []uint16
}

func newGrid(dim int) *Grid {
	g := &Grid{dim: dim, mob: make([]uint16, dim*dim), item: make([]uint16, dim*dim)}
	for i := range g.mob {
		g.mob[i] = gridEmpty
		g.item[i] = gridEmpty
	}
	return g
}

// Dim devolve o lado da grade.
func (g *Grid) Dim() int { return g.dim }

func (g *Grid) inBounds(x, y int) bool { return x >= 0 && y >= 0 && x < g.dim && y < g.dim }

// MobAt devolve o índice de criatura em (x,y) e se a célula está ocupada.
func (g *Grid) MobAt(x, y int) (uint16, bool) {
	if !g.inBounds(x, y) {
		return 0, false
	}
	v := g.mob[y*g.dim+x]
	return v, v != gridEmpty
}

// SetMob registra o id da criatura em (x,y). Escritas fora dos limites são ignoradas.
func (g *Grid) SetMob(x, y int, id uint16) {
	if g.inBounds(x, y) {
		g.mob[y*g.dim+x] = id
	}
}

// ClearMob esvazia a célula em (x,y).
func (g *Grid) ClearMob(x, y int) {
	if g.inBounds(x, y) {
		g.mob[y*g.dim+x] = gridEmpty
	}
}

// ItemAt devolve o índice do item de chão em (x,y) e se a célula contém um.
func (g *Grid) ItemAt(x, y int) (uint16, bool) {
	if !g.inBounds(x, y) {
		return 0, false
	}
	v := g.item[y*g.dim+x]
	return v, v != gridEmpty
}

// SetItem registra o id do item de chão em (x,y). Escritas fora dos limites são ignoradas.
func (g *Grid) SetItem(x, y int, id uint16) {
	if g.inBounds(x, y) {
		g.item[y*g.dim+x] = id
	}
}

// ClearItem esvazia a célula de item de chão em (x,y).
func (g *Grid) ClearItem(x, y int) {
	if g.inBounds(x, y) {
		g.item[y*g.dim+x] = gridEmpty
	}
}
