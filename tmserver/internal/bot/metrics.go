package bot

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// latencySamples é quantas amostras de latência ficam retidas por métrica. O
// suficiente para p95/p99 de um cenário sem crescer com a duração.
const latencySamples = 4096

// Metrics são os contadores de uma execução. É seguro para uso concorrente:
// cada bot escreve de sua goroutine.
type Metrics struct {
	mu       sync.Mutex
	counters map[string]int64
	latency  map[string][]time.Duration
	created  time.Time
}

// NewMetrics cria o conjunto de contadores.
func NewMetrics() *Metrics {
	return &Metrics{
		counters: make(map[string]int64),
		latency:  make(map[string][]time.Duration),
		created:  time.Now(),
	}
}

// Inc soma um ao contador.
func (m *Metrics) Inc(name string) {
	m.Add(name, 1)
}

// Add soma n ao contador.
func (m *Metrics) Add(name string, n int64) {
	m.mu.Lock()
	m.counters[name] += n
	m.mu.Unlock()
}

// Count devolve o valor do contador.
func (m *Metrics) Count(name string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[name]
}

// Observe registra uma latência amostrada.
func (m *Metrics) Observe(name string, d time.Duration) {
	m.mu.Lock()
	s := m.latency[name]
	if len(s) >= latencySamples {
		copy(s, s[1:])
		s[len(s)-1] = d
	} else {
		s = append(s, d)
	}
	m.latency[name] = s
	m.mu.Unlock()
}

// Percentile devolve o percentil p (0..100) das amostras, em milissegundos.
// Sem amostras devolve 0.
func (m *Metrics) Percentile(name string, p float64) float64 {
	m.mu.Lock()
	s := append([]time.Duration(nil), m.latency[name]...)
	m.mu.Unlock()
	if len(s) == 0 {
		return 0
	}
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	if p <= 0 {
		return float64(s[0].Microseconds()) / 1000
	}
	if p >= 100 {
		return float64(s[len(s)-1].Microseconds()) / 1000
	}
	idx := int(p/100*float64(len(s)-1) + 0.5)
	if idx >= len(s) {
		idx = len(s) - 1
	}
	return float64(s[idx].Microseconds()) / 1000
}

// Snapshot devolve uma cópia dos contadores, ordenada por nome.
func (m *Metrics) Snapshot() []Counter {
	m.mu.Lock()
	out := make([]Counter, 0, len(m.counters))
	for k, v := range m.counters {
		out = append(out, Counter{Name: k, Value: v})
	}
	m.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Counter é um contador nomeado.
type Counter struct {
	Name  string
	Value int64
}

// Prometheus devolve o texto no formato de exposição do Prometheus: um
// `# TYPE` por métrica e o nome prefixado com `openwyd_bot_`, para o painel
// distinguir carga de bot de métrica do servidor.
func (m *Metrics) Prometheus() string {
	var b strings.Builder
	b.WriteString("# HELP openwyd_bot_up 1 enquanto o laboratório de bots está em execução.\n")
	b.WriteString("# TYPE openwyd_bot_up gauge\n")
	b.WriteString("openwyd_bot_up 1\n")
	for _, c := range m.Snapshot() {
		name := sanitizeMetric(c.Name)
		fmt.Fprintf(&b, "# TYPE openwyd_bot_%s_total counter\n", name)
		fmt.Fprintf(&b, "openwyd_bot_%s_total %d\n", name, c.Value)
	}
	for _, name := range []string{"actions", "frames"} {
		p95 := m.Percentile(name+"_latency", 95)
		p99 := m.Percentile(name+"_latency", 99)
		base := sanitizeMetric(name + "_latency")
		fmt.Fprintf(&b, "# TYPE openwyd_bot_%s_p95_ms gauge\nopenwyd_bot_%s_p95_ms %g\n", base, base, p95)
		fmt.Fprintf(&b, "# TYPE openwyd_bot_%s_p99_ms gauge\nopenwyd_bot_%s_p99_ms %g\n", base, base, p99)
	}
	return b.String()
}

// sanitizeMetric troca tudo que não é [a-z0-9_] por `_`, como o Prometheus exige.
func sanitizeMetric(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
