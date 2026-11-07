// Package route porta o pathfinding de tiles original: o line-walk guloso de
// 8 direções do BASE_GetRoute sobre o grid de altura (Basedef.cpp:6194-6482) e o
// bake de tiles bloqueados do BASE_ApplyAttribute (Basedef.cpp:2624-2647). É puro
// (sem estado de mundo) para o passeio poder ser testado golden contra mapas sintéticos.
package route

import "openwyd-nextgen/tmserver/internal/content"

const (
	// MaxRoute é MAX_ROUTE (Basedef.h:184): a rota mais longa que o original
	// preenche por chamada, então um passeio avança no máximo MaxRoute-1 passos.
	MaxRoute = 24
	// MH é o maior degrau de altura que um andarilho sobe/desce entre tiles
	// adjacentes (Basedef.h:194): um vizinho é passável sse sua altura estiver
	// (cur-MH, cur+MH) exclusive.
	MH = 8
	// Blocked é a altura que o Bake escreve em células cujo atributo tem o bit 2
	// ligado — fora de toda janela (cur±MH), então nada pode entrar ou sair.
	Blocked = 127
)

// Bake grava o AttributeMap no grid de altura, exatamente como o original faz
// once at boot (Server.cpp:7954 → BASE_ApplyAttribute, Basedef.cpp:2624-2647):
// cada byte de atributo cobre um bloco 4×4 de células de altura ((x>>2)&0x3FF), e
// att&2 marca o bloco como intransponível (altura ← 127). Muta h.Data; chame
// uma vez antes de ligar o grid na IA.
func Bake(h, attr *content.Grid) {
	for y := 0; y < h.Dim; y++ {
		row := h.Data[y*h.Dim : (y+1)*h.Dim]
		arow := attr.Data[((y>>2)&0x3FF)*attr.Dim:]
		for x := range row {
			if arow[(x>>2)&0x3FF]&2 != 0 {
				row[x] = Blocked
			}
		}
	}
}

// Next porta BASE_GetRoute (Basedef.cpp:6194-6482): de (x,y) ele anda até
// min(distância, MaxRoute-1) passos gulosos em direção a (tx,ty) e devolve onde
// parou. Cada passo tenta a ordem exata de ramos do original — a direção
// primária pelo sinal de (tx-x, ty-y), depois suas alternativas laterais —
// pegando o primeiro vizinho passável (altura dentro de ±MH da célula atual,
// bytes com sinal); se nenhum passa, ou o andarilho chega ao alvo ou à margem
// do mapa, ele para antes. moved=false significa que não conseguiu dar nem um
// passo (o sinal de "travado" do chamador, Route[0]==0 no original).
//
// Peculiaridade mantida como está: depois que os ramos primários falham, o original
// desiste quando o alvo está a 1 em QUALQUER eixo (Basedef.cpp:6379) — então as
// alternativas laterais nunca disparam bem ao lado do alvo.
func Next(h *content.Grid, x, y, tx, ty, distance int) (nx, ny int, moved bool) {
	sx, sy := x, y
	dim := h.Dim
	// ht lê uma altura com sinal: o pHeight do original é char (com sinal no MSVC
	// x86) e células reais do mapa ficam negativas; Blocked=127 segue o int8 máximo.
	ht := func(x, y int) int { return int(int8(h.Data[y*dim+x])) }

	for i := 0; i < distance && i < MaxRoute-1; i++ {
		if x < 1 || y < 1 || x > dim-2 || y > dim-2 {
			break // border margin (Basedef.cpp:6205): neighbours must exist
		}
		cul := ht(x, y)
		pass := func(dx, dy int) bool {
			v := ht(x+dx, y+dy)
			return v < cul+MH && v > cul-MH
		}

		if tx == x && ty == y {
			break // arrived
		}
		// Direção primária (ordem de ramos = Basedef.cpp:6232-6305). Note a
		// convenção de dígitos: y-- é '2' e y++ é '8' (numpad em espaço de tela).
		if tx == x && ty < y && pass(0, -1) {
			y--
			continue
		}
		if tx > x && ty < y && pass(1, -1) {
			x++
			y--
			continue
		}
		if tx > x && ty == y && pass(1, 0) {
			x++
			continue
		}
		if tx > x && ty > y && pass(1, 1) {
			x++
			y++
			continue
		}
		if tx == x && ty > y && pass(0, 1) {
			y++
			continue
		}
		if tx < x && ty > y && pass(-1, 1) {
			x--
			y++
			continue
		}
		if tx < x && ty == y && pass(-1, 0) {
			x--
			continue
		}
		if tx < x && ty < y && pass(-1, -1) {
			x--
			y--
			continue
		}
		// Alternativas diagonais: tenta cada eixo da diagonal bloqueada separadamente
		// (Basedef.cpp:6307-6377).
		if tx > x && ty < y && pass(1, 0) {
			x++
			continue
		}
		if tx > x && ty < y && pass(0, -1) {
			y--
			continue
		}
		if tx > x && ty > y && pass(1, 0) {
			x++
			continue
		}
		if tx > x && ty > y && pass(0, 1) {
			y++
			continue
		}
		if tx < x && ty > y && pass(-1, 0) {
			x--
			continue
		}
		if tx < x && ty > y && pass(0, 1) {
			y++
			continue
		}
		if tx < x && ty < y && pass(-1, 0) {
			x--
			continue
		}
		if tx < x && ty < y && pass(0, -1) {
			y--
			continue
		}
		// Saída por adjacência ao alvo (Basedef.cpp:6379) — antes das
		// line deviations below.
		if tx == x+1 || ty == y+1 || tx == x-1 || ty == y-1 {
			break
		}
		// Desvios em linha reta: a cardinal direta está bloqueada, desvia
		// diagonally (Basedef.cpp:6386-6464).
		if tx == x && ty > y && pass(1, 1) {
			x++
			y++
			continue
		}
		if tx == x && ty > y && pass(-1, 1) {
			x--
			y++
			continue
		}
		if tx == x && ty < y && pass(1, -1) {
			x++
			y--
			continue
		}
		if tx == x && ty < y && pass(-1, -1) {
			x--
			y--
			continue
		}
		if tx < x && ty == y && pass(-1, 1) {
			x--
			y++
			continue
		}
		if tx < x && ty == y && pass(-1, -1) {
			x--
			y--
			continue
		}
		if tx > x && ty == y && pass(1, 1) {
			x++
			y++
			continue
		}
		if tx > x && ty == y && pass(1, -1) {
			x++
			y--
			continue
		}
		break // fully stuck (Basedef.cpp:6466)
	}
	return x, y, x != sx || y != sy
}
