package convert

import (
	"os"
	"path/filepath"
	"testing"

	"openwyd-nextgen/internal/savefmt"
)

func TestWalkAccounts(t *testing.T) {
	root := t.TempDir()

	// Duas contas no formato atual em subdiretórios (sharding pela primeira letra).
	mkCurrent := func(first, name string) {
		var af savefmt.AccountFile
		copy(af.Info.AccountName[:], name)
		copy(af.Info.AccountPass[:], "pw")
		dir := filepath.Join(root, first)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), savefmt.Encode(af), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mkCurrent("A", "alice")
	mkCurrent("B", "bob")

	// Um arquivo legado (4294) → tem de virar Skipped, não Failed.
	if err := os.WriteFile(filepath.Join(root, "legacy"), make([]byte, 4294), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := WalkAccounts(root)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Total != 3 || rep.Converted != 2 || rep.Skipped != 1 || rep.Failed != 0 {
		t.Fatalf("report = %+v", *rep)
	}

	var names []string
	for _, r := range rep.Results {
		if r.Account != nil {
			names = append(names, r.Account.Name)
		}
	}
	if len(names) != 2 || names[0] != "alice" || names[1] != "bob" {
		t.Errorf("converted names = %v, want [alice bob]", names)
	}
}
