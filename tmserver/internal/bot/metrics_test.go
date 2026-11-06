package bot

import (
	"strings"
	"testing"
	"time"
)

func TestMetricsCountAndSnapshot(t *testing.T) {
	m := NewMetrics()
	m.Inc("actions")
	m.Add("actions", 4)
	m.Inc("errors")
	if got := m.Count("actions"); got != 5 {
		t.Fatalf("Count(actions) = %d", got)
	}
	snap := m.Snapshot()
	if len(snap) != 2 || snap[0].Name != "actions" || snap[1].Name != "errors" {
		t.Fatalf("snapshot fora de ordem: %+v", snap)
	}
}

func TestPercentileIsSane(t *testing.T) {
	m := NewMetrics()
	for i := 1; i <= 100; i++ {
		m.Observe("lat", time.Duration(i)*time.Millisecond)
	}
	if got := m.Percentile("lat", 50); got < 45 || got > 55 {
		t.Fatalf("p50 = %g", got)
	}
	if got := m.Percentile("lat", 99); got < 95 || got > 100 {
		t.Fatalf("p99 = %g", got)
	}
	if got := m.Percentile("sem-amostra", 99); got != 0 {
		t.Fatalf("p99 de métrica vazia = %g", got)
	}
}

func TestObserveIsBounded(t *testing.T) {
	m := NewMetrics()
	for i := 0; i < latencySamples+500; i++ {
		m.Observe("lat", time.Millisecond)
	}
	m.mu.Lock()
	n := len(m.latency["lat"])
	m.mu.Unlock()
	if n > latencySamples {
		t.Fatalf("amostras cresceram sem limite: %d", n)
	}
}

func TestPrometheusExposition(t *testing.T) {
	m := NewMetrics()
	m.Inc("sim_attack")
	m.Observe("actions_latency", 3*time.Millisecond)
	out := m.Prometheus()
	for _, want := range []string{
		"openwyd_bot_up 1",
		"# TYPE openwyd_bot_sim_attack_total counter",
		"openwyd_bot_sim_attack_total 1",
		"openwyd_bot_actions_latency_p99_ms",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("exposição sem %q:\n%s", want, out)
		}
	}
}

func TestSanitizeMetric(t *testing.T) {
	if got := sanitizeMetric("Hp-Low/step"); got != "hp_low_step" {
		t.Fatalf("sanitizeMetric = %q", got)
	}
}
