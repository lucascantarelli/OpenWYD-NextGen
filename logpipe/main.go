// Command logpipe expõe, por HTTP e somente leitura, o stdout/stderr dos
// containers da stack para o portal que roda dentro do Docker.
//
// Por que existe: a tela /logs do portal lia os logs pelo Docker CLI do host.
// Containerizado, o portal não tem CLI nem socket, e montar
// /var/run/docker.sock direto no portal é proibido (AGENTS.md / issue 40).
// Este processo é o único que monta o socket — read-only, sem nenhuma operação
// de escrita, sem `docker exec`, sem acesso a arquivos — e publica só dois
// endpoints na rede interna do compose:
//
//	GET /healthz           → 200 "ok"
//	GET /status            → {"ok":true,"services":[{service,state,health}]}
//	GET /logs/<alvo>       → texto puro (streaming) com "<serviço> | <linha>"
//
// <alvo> é um dos serviços conhecidos ou "all", que agrega todos. O portal
// autentica a sessão antes de repassar; este processo nunca é publicado para
// fora da rede do compose.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// known é a whitelist única. Nada fora dela é lido, mesmo que exista container.
var known = []string{"tmserver", "tmserver2", "dbserver", "binserver", "webserver", "db", "portal"}

var sincePattern = regexp.MustCompile(`^\d{1,4}[smh]$`)

// docker é um cliente mínimo da Engine API sobre o unix socket.
type docker struct {
	client *http.Client
}

func newDocker(socket string) *docker {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 3 * time.Second}
			return dialer.DialContext(ctx, "unix", socket)
		},
		MaxIdleConns:        8,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	return &docker{client: &http.Client{Transport: transport}}
}

func (d *docker) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker"+path, nil)
	if err != nil {
		return nil, err
	}
	res, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker API indisponível: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 256))
		res.Body.Close()
		return nil, fmt.Errorf("docker API %s: %s: %s", path, res.Status, strings.TrimSpace(string(body)))
	}
	return res, nil
}

type container struct {
	ID    string   `json:"Id"`
	Names []string `json:"Names"`
	State string   `json:"State"`
}

// resolve mapeia serviço → container. O nome vem do compose
// (`openwyd-tmserver-1`, `openwyd_tmserver_1`), então comparamos por token, e
// preferimos o container em execução quando existem réplicas/parados.
func (d *docker) resolve(ctx context.Context) (map[string]container, error) {
	res, err := d.get(ctx, "/containers/json?all=1")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var list []container
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("lista de containers inválida: %w", err)
	}

	out := make(map[string]container, len(known))
	for _, service := range known {
		var best *container
		for i := range list {
			name := ""
			if len(list[i].Names) > 0 {
				name = strings.TrimPrefix(list[i].Names[0], "/")
			}
			if !matchesService(name, service) {
				continue
			}
			if best == nil || (list[i].State == "running" && best.State != "running") {
				best = &list[i]
			}
		}
		if best != nil {
			out[service] = *best
		}
	}
	return out, nil
}

func matchesService(name, service string) bool {
	for _, part := range strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	}) {
		if part == service {
			return true
		}
	}
	return false
}

type serviceStatus struct {
	Service string `json:"service"`
	State   string `json:"state"`
	Health  string `json:"health"`
}

func (d *docker) status(ctx context.Context) ([]serviceStatus, error) {
	found, err := d.resolve(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]serviceStatus, 0, len(known))
	for _, service := range known {
		found, ok := found[service]
		if !ok {
			out = append(out, serviceStatus{Service: service, State: "ausente"})
			continue
		}

		state := found.State
		health := ""
		if res, err := d.get(ctx, "/containers/"+found.ID+"/json"); err == nil {
			var info struct {
				State struct {
					Status string `json:"Status"`
					Health *struct {
						Status string `json:"Status"`
					} `json:"Health"`
				} `json:"State"`
			}
			err = json.NewDecoder(res.Body).Decode(&info)
			res.Body.Close()
			if err == nil {
				if info.State.Status != "" {
					state = info.State.Status
				}
				if info.State.Health != nil {
					health = info.State.Health.Status
				}
			}
		}
		out = append(out, serviceStatus{Service: service, State: state, Health: health})
	}
	return out, nil
}

func (d *docker) logsPath(id string, tail int, since *time.Time) string {
	path := fmt.Sprintf("/containers/%s/logs?stdout=1&stderr=1&follow=1&tail=%d", id, tail)
	if since != nil {
		path += "&since=" + strconv.FormatInt(since.Unix(), 10)
	}
	return path
}

// pump lê o stream do Docker (multiplexado, com header de 8 bytes por frame) e
// entrega linha por linha. Container com TTY responde cru: nesse caso o próprio
// header é texto e o resto segue como fluxo simples.
func pump(body io.Reader, emit func([]byte) error) error {
	reader := bufio.NewReaderSize(body, 64*1024)
	pending := make([]byte, 0, 8192)

	writeLines := func(chunk []byte) error {
		pending = append(pending, chunk...)
		for {
			index := bytes.IndexByte(pending, '\n')
			if index < 0 {
				break
			}
			if err := emit(pending[:index]); err != nil {
				return err
			}
			pending = append(pending[:0], pending[index+1:]...)
		}
		if len(pending) > 64*1024 {
			if err := emit(pending); err != nil {
				return err
			}
			pending = pending[:0]
		}
		return nil
	}

	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(reader, header); err != nil {
			break
		}
		size := int(binary.BigEndian.Uint32(header[4:]))
		if header[0] > 2 || size <= 0 || size > 1<<20 {
			if err := writeLines(header); err != nil {
				return err
			}
			chunk := make([]byte, 32*1024)
			for {
				n, err := reader.Read(chunk)
				if n > 0 {
					if werr := writeLines(chunk[:n]); werr != nil {
						return werr
					}
				}
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return err
				}
			}
			break
		}
		frame := make([]byte, size)
		if _, err := io.ReadFull(reader, frame); err != nil {
			break
		}
		if err := writeLines(frame); err != nil {
			return err
		}
	}

	if len(pending) > 0 {
		return emit(pending)
	}
	return nil
}

type handler struct {
	docker *docker
	mu     sync.Mutex
	cache  struct {
		at     time.Time
		status []serviceStatus
		ok     bool
	}
}

func (h *handler) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, "ok\n")
}

func (h *handler) status(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	cached := h.cache
	h.mu.Unlock()
	if !cached.at.IsZero() && time.Since(cached.at) < 5*time.Second {
		writeStatus(w, cached.ok, cached.status, "")
		return
	}

	status, err := h.docker.status(r.Context())
	if err != nil {
		logf("status: %v", err)
		writeStatus(w, false, nil, err.Error())
		h.mu.Lock()
		h.cache.at = time.Now()
		h.cache.status, h.cache.ok = nil, false
		h.mu.Unlock()
		return
	}

	h.mu.Lock()
	h.cache.at = time.Now()
	h.cache.status, h.cache.ok = status, true
	h.mu.Unlock()
	writeStatus(w, true, status, "")
}

func writeStatus(w http.ResponseWriter, ok bool, services []serviceStatus, message string) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.Header().Set("cache-control", "no-store")
	payload := map[string]any{"ok": ok, "services": services}
	if message != "" {
		payload["error"] = message
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// logs atende /logs/<alvo>. "all" agrega todos os serviços conhecidos.
func (h *handler) logs(w http.ResponseWriter, r *http.Request) {
	target := strings.TrimPrefix(r.URL.Path, "/logs/")
	if target == "" || strings.Contains(target, "/") {
		http.Error(w, "alvo inválido", http.StatusNotFound)
		return
	}
	if target != "all" && !isKnown(target) {
		http.Error(w, "serviço desconhecido: "+target, http.StatusNotFound)
		return
	}

	tail := 200
	if raw := r.URL.Query().Get("tail"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			tail = min(max(n, 1), 2000)
		}
	}

	var since *time.Time
	if raw := r.URL.Query().Get("since"); raw != "" && sincePattern.MatchString(raw) {
		unit := map[byte]time.Duration{'s': time.Second, 'm': time.Minute, 'h': time.Hour}[raw[len(raw)-1]]
		seconds, err := strconv.Atoi(raw[:len(raw)-1])
		if err == nil && unit > 0 {
			at := time.Now().Add(-time.Duration(seconds) * unit)
			since = &at
		}
	}

	found, err := h.docker.resolve(r.Context())
	if err != nil {
		logf("logs %s: %v", target, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.Header().Set("cache-control", "no-store, no-transform")
	w.Header().Set("x-accel-buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)

	if target != "all" {
		container, ok := found[target]
		if !ok {
			fmt.Fprintf(w, "%s | [logpipe] container não encontrado\n", target)
			return
		}
		res, err := h.docker.get(r.Context(), h.docker.logsPath(container.ID, tail, since))
		if err != nil {
			fmt.Fprintf(w, "%s | [logpipe] %v\n", target, err)
			return
		}
		defer res.Body.Close()
		_ = pump(res.Body, func(line []byte) error {
			if _, err := fmt.Fprintf(w, "%s | %s\n", target, line); err != nil {
				return err
			}
			if flusher != nil {
				flusher.Flush()
			}
			return nil
		})
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var mu sync.Mutex
	var wait sync.WaitGroup
	for _, service := range known {
		container, ok := found[service]
		if !ok {
			continue
		}
		wait.Add(1)
		go func(service, id string) {
			defer wait.Done()
			res, err := h.docker.get(ctx, h.docker.logsPath(id, tail, since))
			if err != nil {
				mu.Lock()
				fmt.Fprintf(w, "%s | [logpipe] %v\n", service, err)
				if flusher != nil {
					flusher.Flush()
				}
				mu.Unlock()
				return
			}
			defer res.Body.Close()
			_ = pump(res.Body, func(line []byte) error {
				mu.Lock()
				defer mu.Unlock()
				if _, err := fmt.Fprintf(w, "%s | %s\n", service, line); err != nil {
					return err
				}
				if flusher != nil {
					flusher.Flush()
				}
				return nil
			})
		}(service, container.ID)
	}
	wait.Wait()
}

func isKnown(value string) bool {
	for _, service := range known {
		if service == value {
			return true
		}
	}
	return false
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[logpipe] "+format+"\n", args...)
}

func main() {
	addr := flag.String("addr", ":7788", "endereço HTTP na rede do compose")
	socket := flag.String("socket", "/var/run/docker.sock", "socket do Docker Engine")
	healthcheck := flag.Bool("healthcheck", false, "consulta /healthz e sai (healthcheck do compose)")
	flag.Parse()

	base := "http://127.0.0.1" + *addr
	if *healthcheck {
		client := &http.Client{Timeout: 3 * time.Second}
		res, err := client.Get(base + "/healthz")
		if err != nil {
			logf("healthcheck: %v", err)
			os.Exit(1)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		return
	}

	h := &handler{docker: newDocker(*socket)}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.healthz)
	mux.HandleFunc("/status", h.status)
	mux.HandleFunc("/logs/", h.logs)

	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	logf("listening addr=%s socket=%s services=%s", *addr, *socket, strings.Join(known, ","))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logf("servidor encerrado: %v", err)
		os.Exit(1)
	}
}
