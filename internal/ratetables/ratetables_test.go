package ratetables

import (
	"path/filepath"
	"testing"
)

// contentDir sobe do pacote até a árvore de conteúdo do repositório, para os
// testes lerem os arquivos reais que o servidor monta.
func contentDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "tmserver", "content")
}

func TestRowsParseRealFiles(t *testing.T) {
	rows, err := Rows(contentDir(t))
	if err != nil {
		t.Fatalf("Rows: %v", err)
	}
	cells := index(rows)

	// SancRate.txt declara PO 3 85 → a tabela de refino precisa mostrar 85 ali.
	if got := cells[FileSancRate+"/PO/3"]; got != 85 {
		t.Errorf("PO índice 3 = %d; quer 85 do arquivo", got)
	}
	// Âmago vem depois de PO e, diferente do erro do legado, cai na própria linha.
	if got := cells[FileSancRate+"/AMAGO/11"]; got != 5 {
		t.Errorf("AMAGO índice 11 = %d; quer 5", got)
	}
	// Índices que o arquivo omite mantêm o default compilado.
	if got := cells[FileSancRate+"/PO/7"]; got != 0 {
		t.Errorf("PO índice 7 = %d; quer 0 compilado", got)
	}
	// As chaves de CompRate são a identidade: duas chaves da mesma família não
	// podem colidir.
	if got := cells[FileCompRate+"/Compositor/Item_+7"]; got != 2 {
		t.Errorf("Compositor Item_+7 = %d; quer 2", got)
	}
	if got := cells[FileCompRate+"/Compositor/Item_+9"]; got != 10 {
		t.Errorf("Compositor Item_+9 = %d; quer 10 (chave vizinha colidiu?)", got)
	}
	if got := cells[FileQuestsRate+"/EXP/0.0"]; got != 30000 {
		t.Errorf("tier 0 EXP mortal = %d; quer 30000", got)
	}
	if got := cells[FileQuestsRate+"/LEVEL/0.2"]; got != 39 {
		t.Errorf("tier 0 arch min = %d; quer 39", got)
	}

	// Toda célula carrega uma faixa utilizável, senão a UI não tem com que validar.
	for _, row := range rows {
		if row.Max <= row.Min {
			t.Fatalf("linha sem faixa válida: %+v", row)
		}
		if row.Label == "" || row.Name == "" {
			t.Fatalf("linha sem rótulo/nome: %+v", row)
		}
	}
}

// index dobra as linhas em "file/family/name" → valor, a chave que o editor usa.
func index(rows []Row) map[string]int64 {
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.File+"/"+row.Family+"/"+row.Name] = row.Value
	}
	return out
}

func TestApplyReplacesOnlyKnownCells(t *testing.T) {
	rows, err := Rows(contentDir(t))
	if err != nil {
		t.Fatalf("Rows: %v", err)
	}

	out, applied, ignored := Apply(rows, []Override{
		{File: FileSancRate, Family: "PO", Name: "3", Value: 50},
		{File: FileSancRate, Family: "PO", Name: "99", Value: 1}, // fora da tabela
		{File: FileQuestsRate, Family: "COIN", Name: "4.0", Value: 7},
	})
	if applied != 2 || ignored != 1 {
		t.Fatalf("applied=%d ignored=%d; quer 2/1", applied, ignored)
	}
	got := index(out)
	if got[FileSancRate+"/PO/3"] != 50 {
		t.Errorf("PO 3 = %d; quer 50 do override", got[FileSancRate+"/PO/3"])
	}
	if got[FileQuestsRate+"/COIN/4.0"] != 7 {
		t.Errorf("COIN tier 4 = %d; quer 7", got[FileQuestsRate+"/COIN/4.0"])
	}
	// Chaves vizinhas de CompRate não podem ser tocadas por uma edição de outra
	// chave.
	out2, _, _ := Apply(rows, []Override{{File: FileCompRate, Family: "Compositor", Name: "Item_+7", Value: 1}})
	if got := index(out2)[FileCompRate+"/Compositor/Item_+8"]; got != 4 {
		t.Errorf("Item_+8 virou %d ao editar Item_+7", got)
	}
	// A fatia original fica intacta: o servidor usa os valores do arquivo como
	// fallback.
	if index(rows)[FileSancRate+"/PO/3"] != 85 {
		t.Error("Apply mutou as linhas de origem")
	}
}

func TestValidateRanges(t *testing.T) {
	row := Row{File: FileSancRate, Family: "PO", Name: "1", Min: 0, Max: MaxPercent}
	if err := Validate(row, 100); err != nil {
		t.Errorf("100 dentro da faixa recusado: %v", err)
	}
	if err := Validate(row, 101); err == nil {
		t.Error("101 deveria ser recusado")
	}
	if err := Validate(row, -1); err == nil {
		t.Error("-1 deveria ser recusado")
	}
}
