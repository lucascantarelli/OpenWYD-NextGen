package world

import "openwyd-nextgen/internal/metrics"

// Os helpers abaixo só tocam em memória (registry com mutex curto) e são
// nil-safe: testes e mundos sem -metrics-addr não pagam nada.

func (w *World) metricLabels(extra ...string) metrics.Labels {
	l := metrics.Labels{"service": "tmserver"}
	for i := 0; i+1 < len(extra); i += 2 {
		l[extra[i]] = extra[i+1]
	}
	return l
}

func (w *World) metricInc(name, help string, extra ...string) {
	if w.metrics == nil {
		return
	}
	w.metrics.Inc(name, help, w.metricLabels(extra...))
}

func (w *World) metricAdd(name, help string, v float64) {
	if w.metrics == nil {
		return
	}
	w.metrics.Add(name, help, v, w.metricLabels())
}

func (w *World) metricSet(name, help string, v float64) {
	if w.metrics == nil {
		return
	}
	w.metrics.Set(name, help, v, w.metricLabels())
}

func (w *World) metricObserve(name, help string, v float64, extra ...string) {
	if w.metrics == nil {
		return
	}
	w.metrics.Observe(name, help, v, w.metricLabels(extra...))
}
