package bot

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
)

// Conn é o que o executor precisa de uma conexão: só o que o cliente headless
// oferece. É interface para o teste rodar sem rede.
type Conn interface {
	Login(account, password, token string, version int32) error
	SelectCharacter(slot int) error
	ReadFrame(timeout time.Duration) (protocol.Header, []byte, error)
	Send(t protocol.Type, payload []byte) error
	Close() error
}

// DialFunc abre uma conexão headless com o endereço dado.
type DialFunc func(addr string, timeout time.Duration) (Conn, error)

// Options parametriza o executor.
type Options struct {
	// Dial é o construtor de conexão; nil usa Dial.
	Dial DialFunc
	// AllowRemote libera alvo fora de loopback. Por padrão o executor recusa:
	// uma carga apontada para fora do ambiente local é acidente, não teste.
	AllowRemote bool
	// Version é a versão de protocolo enviada no login; zero usa a do projeto.
	Version int32
	// Logf recebe o andamento; nil descarta.
	Logf func(format string, args ...any)
}

// Report é o resultado de um cenário.
type Report struct {
	ScenarioID    string         `json:"scenario_id"`
	Objective     string         `json:"objective"`
	Address       string         `json:"address"`
	Seed          int64          `json:"seed"`
	StartedAt     time.Time      `json:"started_at"`
	Duration      string         `json:"duration"`
	Bots          int            `json:"bots"`
	Connected     int            `json:"bots_connected"`
	Failed        int            `json:"bots_failed"`
	Ticks         int64          `json:"ticks"`
	Actions       int64          `json:"actions"`
	Errors        int64          `json:"errors"`
	Frames        int64          `json:"frames"`
	StatesVisited []string       `json:"states_visited"`
	FinalStates   map[string]int `json:"final_states"`
	Counters      []Counter      `json:"counters"`
	P99Millis     float64        `json:"p99_ms"`
	ErrorSamples  []string       `json:"error_samples,omitempty"`
	Success       bool           `json:"success"`
	SuccessWhy    []string       `json:"success_detail,omitempty"`
	Failure       bool           `json:"failure"`
	FailureWhy    []string       `json:"failure_detail,omitempty"`
}

// metric resolve uma métrica citada em critério de cenário.
func (r Report) metric(name string) (float64, bool) {
	switch name {
	case "bots_connected":
		return float64(r.Connected), true
	case "bots_failed":
		return float64(r.Failed), true
	case "bots_total":
		return float64(r.Bots), true
	case "actions":
		return float64(r.Actions), true
	case "errors":
		return float64(r.Errors), true
	case "frames":
		return float64(r.Frames), true
	case "states_visited":
		return float64(len(r.StatesVisited)), true
	case "ticks":
		return float64(r.Ticks), true
	case "p99_ms":
		return r.P99Millis, true
	}
	return 0, false
}

// Runner executa um cenário: N bots, cada um com sua semente derivada, por um
// tempo fixo, coletando métricas e veredito.
type Runner struct {
	scen    *Scenario
	opts    Options
	metrics *Metrics

	mu      sync.Mutex
	errors  []string
	actions int64
	ticks   int64
	frames  int64
}

// NewRunner valida o cenário e o alvo e prepara o executor.
func NewRunner(s *Scenario, opts Options) (*Runner, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if opts.Dial == nil {
		opts.Dial = func(addr string, timeout time.Duration) (Conn, error) {
			return Dial(addr, timeout)
		}
	}
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}
	host, _, err := net.SplitHostPort(s.Address)
	if err != nil {
		return nil, fmt.Errorf("bot: endereço %q inválido: %w", s.Address, err)
	}
	if !opts.AllowRemote && !isLoopback(host) {
		return nil, fmt.Errorf("bot: alvo %q não é local; use -allow-remote para um ambiente autorizado", s.Address)
	}
	for _, b := range s.Bots {
		p, err := LookupProfile(b.Profile)
		if err != nil {
			return nil, err
		}
		if err := p.Validate(); err != nil {
			return nil, err
		}
	}
	return &Runner{scen: s, opts: opts, metrics: NewMetrics()}, nil
}

// Metrics devolve o coletor da execução (para o endpoint de métricas).
func (r *Runner) Metrics() *Metrics { return r.metrics }

// Run executa o cenário até o tempo acabar ou o contexto ser cancelado.
func (r *Runner) Run(ctx context.Context) (Report, error) {
	started := time.Now()
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(r.scen.DurationSeconds)*time.Second)
	defer cancel()

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		connected int
		failed    int
		final     = make(map[string]int)
		visited   = make(map[string]struct{})
	)

	transitions := DefaultTransitions(r.metrics)
	probe, err := NewMachine(StateConnecting, transitions)
	if err != nil {
		return Report{}, err
	}

	tick := time.Duration(r.scen.TickMillis) * time.Millisecond
	for i, spec := range r.scen.Bots {
		wg.Add(1)
		go func(i int, spec Spec) {
			defer wg.Done()
			seed := r.scen.Seed + int64(i)*7919
			st, err := r.runBot(runCtx, spec, seed, transitions, tick)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed++
				r.recordError(fmt.Sprintf("bot %s: %v", spec.Account, err))
			} else {
				connected++
			}
			final[string(st.state)]++
			for _, rec := range st.history {
				visited[string(rec.From)] = struct{}{}
				visited[string(rec.To)] = struct{}{}
			}
		}(i, spec)
	}
	wg.Wait()

	statesVisited := make([]string, 0, len(visited))
	for s := range visited {
		statesVisited = append(statesVisited, s)
	}
	sort.Strings(statesVisited)

	r.mu.Lock()
	errs := append([]string(nil), r.errors...)
	actions, ticks, frames := r.actions, r.ticks, r.frames
	r.mu.Unlock()

	rep := Report{
		ScenarioID:    r.scen.ID,
		Objective:     r.scen.Objective,
		Address:       r.scen.Address,
		Seed:          r.scen.Seed,
		StartedAt:     started,
		Duration:      time.Since(started).Round(time.Millisecond).String(),
		Bots:          len(r.scen.Bots),
		Connected:     connected,
		Failed:        failed,
		Ticks:         ticks,
		Actions:       actions,
		Errors:        int64(len(errs)),
		Frames:        frames,
		StatesVisited: statesVisited,
		FinalStates:   final,
		Counters:      r.metrics.Snapshot(),
		P99Millis:     r.metrics.Percentile("actions_latency", 99),
	}
	if len(errs) > 10 {
		errs = errs[:10]
	}
	rep.ErrorSamples = errs

	if len(r.scen.Success) > 0 {
		ok, why := Evaluate(r.scen.Success, rep)
		rep.Success = ok
		rep.SuccessWhy = why
	}
	if len(r.scen.Failure) > 0 {
		bad, why := Evaluate(r.scen.Failure, rep)
		rep.Failure = bad
		rep.FailureWhy = why
	}
	// Um critério de falha satisfeito reprova o cenário mesmo que os critérios
	// de sucesso passem: os dois são avaliados, não excludentes.
	if rep.Failure {
		rep.Success = false
	}
	_ = probe
	return rep, nil
}

// runBot executa um bot do começo ao fim e devolve a máquina com o histórico.
func (r *Runner) runBot(ctx context.Context, spec Spec, seed int64, transitions []Transition, tick time.Duration) (*Machine, error) {
	profile, err := LookupProfile(spec.Profile)
	if err != nil {
		return nil, err
	}
	m, err := NewMachine(StateConnecting, transitions)
	if err != nil {
		return nil, err
	}
	cctx := &Context{
		Slot:          spec.Slot,
		HP:            100,
		HPmax:         100,
		MP:            50,
		MPmax:         50,
		InventoryFree: 8,
		Potions:       5,
	}
	rng := NewRNG(seed)

	conn, err := r.opts.Dial(r.scen.Address, dialTimeout)
	if err != nil {
		return m, err
	}
	defer conn.Close()

	if err := fire(m, EventConnected, cctx); err != nil {
		return m, err
	}
	loginStart := time.Now()
	if err := conn.Login(spec.Account, spec.Password, "", r.opts.Version); err != nil {
		return m, err
	}
	r.metrics.Observe("actions_latency", time.Since(loginStart))
	if err := fire(m, EventAuthenticated, cctx); err != nil {
		return m, err
	}
	if err := conn.SelectCharacter(spec.Slot); err != nil {
		return m, err
	}
	if err := fire(m, EventCharacterReady, cctx); err != nil {
		return m, err
	}
	if err := fire(m, EventWorldEntered, cctx); err != nil {
		return m, err
	}

	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if tr, err := m.Fire(EventStop, cctx); err == nil {
				_ = tr
			}
			return m, nil
		case <-ticker.C:
			r.mu.Lock()
			r.ticks++
			r.mu.Unlock()
			cctx.Ticks++
			r.metrics.Inc("ticks")

			ev := nextEvent(m.State(), cctx, profile, rng)
			if ev == "" {
				r.metrics.Inc("idle_steps")
				continue
			}
			stepStart := time.Now()
			tr, err := m.Fire(ev, cctx)
			if err != nil {
				// Sem transição não é falha do servidor: é buraco no roteiro.
				r.metrics.Inc("no_transition")
				if cctx.LastError == "" {
					cctx.LastError = err.Error()
				}
				continue
			}
			_ = tr
			r.metrics.Observe("actions_latency", time.Since(stepStart))
			r.mu.Lock()
			r.actions++
			r.mu.Unlock()
			r.metrics.Inc("actions")
		}
	}
}

// fire aplica um evento obrigatório do ciclo de vida; erro significa roteiro
// quebrado, não comportamento do servidor.
func fire(m *Machine, ev Event, c *Context) error {
	if _, err := m.Fire(ev, c); err != nil {
		return fmt.Errorf("bot: ciclo de vida: %w", err)
	}
	return nil
}

// recordError guarda amostras de erro para o relatório.
func (r *Runner) recordError(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.errors) < 100 {
		r.errors = append(r.errors, msg)
	}
}

// isLoopback reconhece os alvos considerados locais.
func isLoopback(host string) bool {
	h := strings.Trim(host, "[]")
	if h == "" {
		return false
	}
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}
