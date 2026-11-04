package skillcatalog

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScanMirrorsContentParser pins the tolerance rules shared with
// tmserver/internal/content.parseSkillData: index from column 0 (not the row
// number), rows under 22 columns skipped, out-of-range indexes skipped, name
// from the trailing column and decoded from ISO-8859-1.
func TestScanMirrorsContentParser(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "Common"), 0o755); err != nil {
		t.Fatal(err)
	}
	rows := []string{
		// "Giro_da_Fúria" with Fúria spelled in Latin-1 (0xFA), like the real file.
		"0,24,3,15,3,5,4,5,0,0,0,0,0,10.0.0.10.0.0.0.0,10.0.0.10.0.0.0.0,4,0,1,13,0,3,0,1,Giro_da_F\xfaria",
		// Sparse index past 149 must be kept, not renumbered.
		"150,10,0,5,0,0,0,0,0,0,0,0,0,0.0.0.0.0.0.0.0,0.0.0.0.0.0.0.0,0,0,0,0,0,0,0,0,Escudo_Final",
		// Short row: dropped like the legacy loader drops it.
		"1,2,3",
		// Index outside [0,248): dropped.
		"300,24,3,15,3,5,4,5,0,0,0,0,0,0.0.0.0.0.0.0.0,0.0.0.0.0.0.0.0,4,0,1,13,0,3,0,1,Fora",
	}
	body := ""
	for _, row := range rows {
		body += row + "\n"
	}
	path := filepath.Join(dir, "Common", "SkillData.csv")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cat, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(cat.Skills) != 2 {
		t.Fatalf("len(Skills) = %d, want 2 (short and out-of-range rows dropped)", len(cat.Skills))
	}
	if cat.Version == "" {
		t.Fatal("Version is empty")
	}
	byIndex := cat.ByIndex()
	furia := byIndex[0]
	if furia.DisplayName != "Giro da Fúria" {
		t.Errorf("DisplayName = %q, want %q", furia.DisplayName, "Giro da Fúria")
	}
	if furia.Class != 0 {
		t.Errorf("Class = %d, want 0 (TK)", furia.Class)
	}
	if got := byIndex[150]; got.Name != "Escudo_Final" || got.Class != 6 {
		t.Errorf("index 150 = %+v, want Escudo_Final in class 6", got)
	}
}
