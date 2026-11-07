package refine

import (
	"testing"

	"openwyd-nextgen/tmserver/internal/world"
)

// fakeTable é a tabela SancRate.txt efetiva depois da correção do Âmago da issue #103:
// as linhas PO/PL do arquivo aplicadas sobre os padrões de Basedef.cpp.
type fakeTable [3][12]int

func (f fakeTable) Rate(anvil, idx int) int {
	if anvil < 0 || anvil >= len(f) || idx < 0 || idx >= len(f[anvil]) {
		return 0
	}
	return f[anvil][idx]
}

var liveTable = fakeTable{
	{100, 100, 100, 85, 70, 40, 100, 0, 0, 0, 0, 0},      // PO: 0..5 do arquivo, o resto são padrões compilados
	{100, 100, 100, 100, 100, 100, 80, 70, 70, 10, 0, 0}, // PL: 0..9 do arquivo
	{100, 80, 60, 40, 20, 10, 10, 10, 5, 5, 5, 5},        // Âmago, agora em linha própria
}

func TestSuccessRate(t *testing.T) {
	cases := []struct {
		name  string
		it    world.Item
		anvil int
		want  int
	}{
		// A tabela é indexada em nível+1, então um refino A PARTIR de +N lê o slot N+1.
		{"Ori +0→+1", sanc(0), Ori, 100},
		{"Ori +2→+3", sanc(2), Ori, 85},
		{"Ori +3→+4", sanc(3), Ori, 70},
		{"Ori +4→+5", sanc(4), Ori, 40},
		// O arquivo não tem linha PO 6, então isto cai no 100 compilado. Apontado
		// na issue #103: um operador controla isso adicionando uma.
		{"Ori +5→+6 is free on the current table", sanc(5), Ori, 100},

		{"Lac +0→+1", sanc(0), Lac, 100},
		{"Lac +5→+6", sanc(5), Lac, 80},
		{"Lac +6→+7", sanc(6), Lac, 70},
		{"Lac +8→+9", sanc(8), Lac, 10},

		// O pity soma o peso de nível+1 por ponto acumulado (g_pSuccessRate).
		{"Lac +8 with pity 3 adds 3*1", sanc(encode(8, 3)), Lac, 13},
		{"Ori +4 with pity 2 adds 2*4", sanc(encode(4, 2)), Ori, 48},
		{"Ori +0 with pity 1 adds 5 over a full rate", sanc(encode(0, 1)), Ori, 105},

		// +10 é fixo a montante da tabela.
		{"+10 with Ori", sanc(230), Ori, 15},
		{"+10 with Lac", sanc(230), Lac, 15},
		{"+10 with Amago", sanc(230), Amago, 100},

		// Lactolerium_100 força a linha Amago para uma garantia fixa.
		{"Amago is always 100 regardless of level", sanc(4), Amago, 100},
		{"Amago is always 100 even at +8", sanc(8), Amago, 100},

		// Guardas.
		{"unknown anvil", sanc(0), 3, 0},
		{"negative anvil", sanc(0), -1, 0},
		{"growth items never refine", sancGrowth(2350, 0), Ori, 0},
		// Bloqueado antes do sorteio; 0 evita que a entrada inalcançável indexe fora (OOB).
		{"+11 is out of the table's reach", sanc(234), Ori, 0},
		{"+15 is out of the table's reach", sanc(250), Lac, 0},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := SuccessRate(liveTable, tt.it, tt.anvil); got != tt.want {
				t.Errorf("SuccessRate = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStep(t *testing.T) {
	// g_pSancGrade é todo 1 e SancRate.txt não tem linhas PO_A..PL_E para sobrescrever
	// isso, então todo refino bem-sucedido é exatamente +1.
	for _, anvil := range []int{Ori, Lac} {
		for grade := 1; grade <= 5; grade++ {
			if got := Step(anvil, grade); got != 1 {
				t.Errorf("Step(%d,%d) = %d, want 1", anvil, grade, got)
			}
		}
	}
	// Grades fora da faixa caem num +1 simples (_MSG_UseItem.cpp:874).
	for _, tt := range []struct{ anvil, grade int }{{Ori, 0}, {Ori, 6}, {Ori, -1}, {Amago, 1}, {-1, 1}} {
		if got := Step(tt.anvil, tt.grade); got != 1 {
			t.Errorf("Step(%d,%d) = %d, want 1", tt.anvil, tt.grade, got)
		}
	}
}
