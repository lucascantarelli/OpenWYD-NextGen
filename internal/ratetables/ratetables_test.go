package ratetables

import (
	"path/filepath"
	"testing"
)

// contentDir walks up from the package to the repository's Release tree, so the
// tests parse the real files the server mounts.
func contentDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "Release")
}

func TestRowsParseRealFiles(t *testing.T) {
	rows, err := Rows(contentDir(t))
	if err != nil {
		t.Fatalf("Rows: %v", err)
	}
	cells := index(rows)

	// SancRate.txt declares PO 3 85 → the refine table must show 85 there.
	if got := cells[FileSancRate+"/PO/3"]; got != 85 {
		t.Errorf("PO índice 3 = %d; quer 85 do arquivo", got)
	}
	// Âmago is listed after PO and, unlike the legacy bug, lands on its own row.
	if got := cells[FileSancRate+"/AMAGO/11"]; got != 5 {
		t.Errorf("AMAGO índice 11 = %d; quer 5", got)
	}
	// Indices the file omits keep the compiled default.
	if got := cells[FileSancRate+"/PO/7"]; got != 0 {
		t.Errorf("PO índice 7 = %d; quer 0 compilado", got)
	}
	// CompRate keys are the identity: two keys of the same family cannot collide.
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

	// Every cell carries a usable range, or the UI has nothing to validate with.
	for _, row := range rows {
		if row.Max <= row.Min {
			t.Fatalf("linha sem faixa válida: %+v", row)
		}
		if row.Label == "" || row.Name == "" {
			t.Fatalf("linha sem rótulo/nome: %+v", row)
		}
	}
}

// index folds rows into "file/family/name" → value, the key the editor uses.
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
	// Sibling CompRate keys must stay untouched by an edit to another key.
	out2, _, _ := Apply(rows, []Override{{File: FileCompRate, Family: "Compositor", Name: "Item_+7", Value: 1}})
	if got := index(out2)[FileCompRate+"/Compositor/Item_+8"]; got != 4 {
		t.Errorf("Item_+8 virou %d ao editar Item_+7", got)
	}
	// Original slice untouched: the server keeps the file values as the fallback.
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
