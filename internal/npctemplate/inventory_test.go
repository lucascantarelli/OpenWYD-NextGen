package npctemplate

import (
	"log/slog"
	"strings"
	"testing"

	"openwyd-nextgen/internal/savefmt"
)

func TestTakeInventory(t *testing.T) {
	dir := makeNPCDir(t)
	npcDir := Dir(dir)
	writeTemplate(t, dir, "Canonical", Size)
	writeTemplate(t, dir, "AlsoCanonical", Size)
	writeTemplate(t, dir, "Legacy", savefmt.MobSizeLegacy756)
	writeTemplate(t, dir, "LegacyPadded", savefmt.MobSizeLegacy756Padded)
	writeTemplate(t, dir, "Nota.txt", 12)

	inv, err := TakeInventory(npcDir)
	if err != nil {
		t.Fatalf("TakeInventory: %v", err)
	}
	if got, want := inv.Stats.Total(), 5; got != want {
		t.Errorf("Total = %d, want %d", got, want)
	}
	if got, want := inv.Stats.Current816, 2; got != want {
		t.Errorf("Current816 = %d, want %d", got, want)
	}
	if got, want := inv.Stats.Legacy756, 1; got != want {
		t.Errorf("Legacy756 = %d, want %d", got, want)
	}
	if got, want := inv.Stats.Legacy756Padded, 1; got != want {
		t.Errorf("Legacy756Padded = %d, want %d", got, want)
	}
	if got, want := inv.Stats.Rejected, 1; got != want {
		t.Errorf("Rejected = %d, want %d", got, want)
	}
	if got, want := inv.Stats.Loaded(), 4; got != want {
		t.Errorf("Loaded = %d, want %d", got, want)
	}
	// Os nomes são ordenados para o relatório ser estável entre execuções.
	canonical := inv.Names[savefmt.MobVersionCurrent]
	if len(canonical) != 2 || canonical[0] != "AlsoCanonical" || canonical[1] != "Canonical" {
		t.Errorf("canonical names = %v, want sorted [AlsoCanonical Canonical]", canonical)
	}

	report := strings.Join(inv.Report(0), "\n")
	for _, want := range []string{"current-816", "legacy-756", "legacy-756+pad920", "Nota.txt"} {
		if !strings.Contains(report, want) {
			t.Errorf("report is missing %q:\n%s", want, report)
		}
	}
	// O conjunto canônico é resumido, nunca enumerado.
	if strings.Contains(report, "Canonical\n") {
		t.Errorf("report enumerates canonical templates:\n%s", report)
	}
}

func TestTakeInventoryMissingDir(t *testing.T) {
	if _, err := TakeInventory(t.TempDir() + "/nope"); err == nil {
		t.Error("TakeInventory on a missing dir succeeded, want error")
	}
}

func TestScanStatsLogValue(t *testing.T) {
	var s ScanStats
	s.Count(savefmt.MobVersionCurrent)
	s.Count(savefmt.MobVersionLegacy756)
	s.Count(savefmt.MobVersionUnknown)
	s.CountUnreadable()

	if s.Total() != 4 || s.Loaded() != 2 {
		t.Fatalf("Total/Loaded = %d/%d, want 4/2", s.Total(), s.Loaded())
	}

	v := s.LogValue()
	if v.Kind() != slog.KindGroup {
		t.Fatalf("LogValue kind = %v, want a group", v.Kind())
	}
	found := make(map[string]int64)
	for _, attr := range v.Group() {
		found[attr.Key] = attr.Value.Int64()
	}
	for key, want := range map[string]int64{
		"total": 4, "loaded": 2, "current816": 1,
		"legacy756": 1, "legacy756_padded920": 0, "unreadable": 1, "rejected": 1,
	} {
		if got, ok := found[key]; !ok || got != want {
			t.Errorf("LogValue[%q] = %v (present=%v), want %d", key, got, ok, want)
		}
	}
}
