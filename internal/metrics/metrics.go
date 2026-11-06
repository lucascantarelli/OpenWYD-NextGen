// Package metrics expõe /metrics no formato texto do Prometheus sem puxar uma
// dependência nova: contadores, gauges, histogramas e gauges calculados no
// scrape. O laço do mundo só toca em memória (mutex curto); nenhuma coleta faz
// I/O a partir dele.
package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// Labels são os labels de uma amostra. O serviço que expõe a métrica deve ir em
// "service" para que os quatro processos sejam distinguíveis no scrape.
type Labels map[string]string

type kind uint8

const (
	kindCounter kind = iota
	kindGauge
	kindHistogram
)

func (k kind) String() string {
	switch k {
	case kindCounter:
		return "counter"
	case kindGauge:
		return "gauge"
	default:
		return "histogram"
	}
}

// buckets são os limites do histograma, em segundos. Cobrem do tick do mundo
// (sub-milissegundo) ao timeout de RPC (segundos).
var buckets = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type sample struct {
	labels  Labels
	value   float64
	count   uint64
	sum     float64
	buckets []uint64 // cumulative, um por limite
}

type family struct {
	mu      sync.Mutex
	name    string
	help    string
	kind    kind
	samples map[string]*sample
}

type gaugeFunc struct {
	name   string
	help   string
	labels Labels
	f      func() float64
}

// Registry acumula as métricas de um processo. É seguro para uso concorrente.
type Registry struct {
	mu       sync.RWMutex
	families map[string]*family
	funcs    []gaugeFunc
}

// New cria o registry e registra as métricas de identidade do processo.
func New(service string) *Registry {
	r := &Registry{families: make(map[string]*family)}
	r.Set("up", "1 quando o processo está no ar.", 1, Labels{"service": service})
	return r
}

func (r *Registry) family(name, help string, k kind) *family {
	r.mu.RLock()
	f := r.families[name]
	r.mu.RUnlock()
	if f != nil {
		return f
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if f = r.families[name]; f != nil {
		return f
	}
	f = &family{name: name, help: help, kind: k, samples: make(map[string]*sample)}
	r.families[name] = f
	return f
}

func newSample(labels Labels) *sample {
	cp := make(Labels, len(labels))
	for k, v := range labels {
		cp[k] = v
	}
	return &sample{labels: cp, buckets: make([]uint64, len(buckets))}
}

func (f *family) sample(labels Labels) *sample {
	key := labelsKey(labels)
	s := f.samples[key]
	if s == nil {
		s = newSample(labels)
		f.samples[key] = s
	}
	return s
}

// Inc soma 1 a um contador.
func (r *Registry) Inc(name, help string, labels Labels) {
	r.Add(name, help, 1, labels)
}

// Add soma v a um contador ou gauge (o tipo é fixado na primeira chamada).
func (r *Registry) Add(name, help string, v float64, labels Labels) {
	f := r.family(name, help, kindCounter)
	f.mu.Lock()
	f.sample(labels).value += v
	f.mu.Unlock()
}

// Set define o valor de um gauge.
func (r *Registry) Set(name, help string, v float64, labels Labels) {
	f := r.family(name, help, kindGauge)
	f.mu.Lock()
	f.sample(labels).value = v
	f.mu.Unlock()
}

// Observe registra uma observação no histograma (em segundos).
func (r *Registry) Observe(name, help string, v float64, labels Labels) {
	f := r.family(name, help, kindHistogram)
	f.mu.Lock()
	s := f.sample(labels)
	s.count++
	s.sum += v
	for i, b := range buckets {
		if v <= b {
			s.buckets[i]++
		}
	}
	f.mu.Unlock()
}

// GaugeFunc registra um gauge avaliado apenas no scrape. É o lugar certo para
// estatísticas de pool/fila: o laço do mundo nunca paga a consulta.
func (r *Registry) GaugeFunc(name, help string, labels Labels, f func() float64) {
	cp := make(Labels, len(labels))
	for k, v := range labels {
		cp[k] = v
	}
	r.mu.Lock()
	r.funcs = append(r.funcs, gaugeFunc{name: name, help: help, labels: cp, f: f})
	r.mu.Unlock()
}

// Write serializa o registry no formato texto do Prometheus (version 0.0.4).
func (r *Registry) Write(w io.Writer) error {
	r.mu.RLock()
	fams := make([]*family, 0, len(r.families))
	for _, f := range r.families {
		fams = append(fams, f)
	}
	funcs := append([]gaugeFunc(nil), r.funcs...)
	r.mu.RUnlock()

	sort.Slice(fams, func(i, j int) bool { return fams[i].name < fams[j].name })
	for _, f := range fams {
		if err := f.write(w); err != nil {
			return err
		}
	}
	for _, g := range funcs {
		if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n", g.name, g.help, g.name); err != nil {
			return err
		}
		v := math.NaN()
		if g.f != nil {
			v = g.f()
		}
		if _, err := fmt.Fprintf(w, "%s%s %s\n", g.name, labelsText(g.labels), formatFloat(v)); err != nil {
			return err
		}
	}
	return nil
}

func (f *family) write(w io.Writer) error {
	f.mu.Lock()
	keys := make([]string, 0, len(f.samples))
	for k := range f.samples {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	snapshot := make([]*sample, 0, len(keys))
	for _, k := range keys {
		s := *f.samples[k]
		s.buckets = append([]uint64(nil), f.samples[k].buckets...)
		snapshot = append(snapshot, &s)
	}
	f.mu.Unlock()

	if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n", f.name, f.help, f.name, f.kind); err != nil {
		return err
	}
	for _, s := range snapshot {
		switch f.kind {
		case kindHistogram:
			for i, b := range buckets {
				labels := withLabel(s.labels, "le", formatFloat(b))
				if _, err := fmt.Fprintf(w, "%s_bucket%s %d\n", f.name, labelsText(labels), s.buckets[i]); err != nil {
					return err
				}
			}
			inf := withLabel(s.labels, "le", "+Inf")
			if _, err := fmt.Fprintf(w, "%s_bucket%s %d\n%s_sum%s %s\n%s_count%s %d\n",
				f.name, labelsText(inf), s.count,
				f.name, labelsText(s.labels), formatFloat(s.sum),
				f.name, labelsText(s.labels), s.count); err != nil {
				return err
			}
		default:
			if _, err := fmt.Fprintf(w, "%s%s %s\n", f.name, labelsText(s.labels), formatFloat(s.value)); err != nil {
				return err
			}
		}
	}
	return nil
}

func withLabel(l Labels, k, v string) Labels {
	cp := make(Labels, len(l)+1)
	for key, val := range l {
		cp[key] = val
	}
	cp[k] = v
	return cp
}

func labelsKey(l Labels) string {
	if len(l) == 0 {
		return ""
	}
	keys := make([]string, 0, len(l))
	for k := range l {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(l[k])
		b.WriteByte(0)
	}
	return b.String()
}

func labelsText(l Labels) string {
	if len(l) == 0 {
		return ""
	}
	keys := make([]string, 0, len(l))
	for k := range l {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteString(`="`)
		b.WriteString(escapeLabel(l[k]))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}

func escapeLabel(s string) string {
	return strings.NewReplacer(`\`, `\\`, "\n", `\n`, `"`, `\"`).Replace(s)
}

func formatFloat(v float64) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	default:
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
}

// Handler serve /metrics.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := r.Write(w); err != nil {
			http.Error(w, "metrics: "+err.Error(), http.StatusInternalServerError)
		}
	})
}

// Serve sobe o endpoint de debug (metrics/healthz/readyz) até o ctx terminar.
// Bloqueia; os chamadores usam `go metrics.Serve(...)`. Addr vazio desliga.
func Serve(ctx context.Context, addr string, r *Registry, log *slog.Logger) {
	if addr == "" {
		log.Info("metrics endpoint disabled")
		return
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", r.Handler())
	ok := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "ok\n")
	}
	mux.HandleFunc("/healthz", ok)
	mux.HandleFunc("/readyz", ok)

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	log.Info("metrics server listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Warn("metrics server stopped", "err", err)
	}
}

// UnaryServerInterceptor mede cada RPC por método e código. Não registra o
// trace_id como label (cardinalidade); ele pertence ao log/contexto.
func UnaryServerInterceptor(service string, r *Registry) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err).String()
		r.Inc("grpc_requests_total", "Total de RPCs gRPC por método e código.",
			Labels{"service": service, "method": info.FullMethod, "code": code})
		r.Observe("grpc_request_duration_seconds", "Duração dos RPCs gRPC em segundos.",
			time.Since(start).Seconds(), Labels{"service": service, "method": info.FullMethod})
		return resp, err
	}
}
