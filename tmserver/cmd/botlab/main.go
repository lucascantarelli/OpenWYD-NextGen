// Comando botlab executa o laboratório de bots: valida cenários, roda a
// população artificial contra um tmServer local e publica métricas.
//
// Uso:
//
//	botlab validate -scenario cenarios/smoke.json
//	botlab run -scenario cenarios/smoke.json [-metrics :9200] [-json]
//	botlab metrics
//
// O comando recusa alvo que não seja loopback; para um ambiente autorizado e
// isolado, -allow-remote libera explicitamente.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"openwyd-nextgen/tmserver/internal/bot"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "botlab: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "validate":
		return cmdValidate(args[1:])
	case "run":
		return cmdRun(args[1:])
	case "metrics":
		fmt.Print("# métricas disponíveis em cenários Success/Failure:\n")
		for _, m := range bot.KnownMetrics() {
			fmt.Printf("- %s\n", m)
		}
		return nil
	case "help", "-h", "--help":
		return usage()
	default:
		return fmt.Errorf("subcomando %q desconhecido", args[0])
	}
}

func usage() error {
	fmt.Fprint(os.Stderr, `botlab — laboratório de bots do Dolorestec WYD2

  botlab validate -scenario <arquivo.json>
  botlab run      -scenario <arquivo.json> [-metrics :9200] [-json] [-allow-remote]
  botlab metrics

O cenário declara população, semente, duração, endereço e critérios de
sucesso/falha. Sem -allow-remote o alvo precisa ser loopback.
`)
	return nil
}

func cmdValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	path := fs.String("scenario", "", "caminho do cenário em JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("validate: -scenario é obrigatório")
	}
	s, err := bot.LoadScenario(*path)
	if err != nil {
		return err
	}
	fmt.Printf("cenário %s válido: %d bots, %ds, semente %d, alvo %s\n",
		s.ID, len(s.Bots), s.DurationSeconds, s.Seed, s.Address)
	return nil
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	path := fs.String("scenario", "", "caminho do cenário em JSON")
	metricsAddr := fs.String("metrics", "", "endereço de exposição Prometheus (ex.: 127.0.0.1:9200)")
	asJSON := fs.Bool("json", false, "imprime o relatório em JSON")
	allowRemote := fs.Bool("allow-remote", false, "libera alvo fora de loopback (ambiente autorizado)")
	version := fs.Int("version", 0, "versão de protocolo do login (0 = padrão do projeto)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("run: -scenario é obrigatório")
	}
	s, err := bot.LoadScenario(*path)
	if err != nil {
		return err
	}

	logf := func(format string, a ...any) {
		fmt.Fprintf(os.Stderr, "botlab: "+format+"\n", a...)
	}
	r, err := bot.NewRunner(s, bot.Options{
		AllowRemote: *allowRemote,
		Version:     int32(*version),
		Logf:        logf,
	})
	if err != nil {
		return err
	}

	if *metricsAddr != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			fmt.Fprint(w, r.Metrics().Prometheus())
		})
		ln, err := net.Listen("tcp", *metricsAddr)
		if err != nil {
			return fmt.Errorf("run: métricas em %s: %w", *metricsAddr, err)
		}
		go func() {
			if err := http.Serve(ln, mux); err != nil {
				logf("métricas encerradas: %v", err)
			}
		}()
		logf("métricas em http://%s/metrics", ln.Addr().String())
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rep, err := r.Run(ctx)
	if err != nil {
		return err
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			return err
		}
	} else {
		printHuman(rep)
	}
	if rep.Failure || (len(s.Success) > 0 && !rep.Success) {
		return fmt.Errorf("cenário %s reprovado", rep.ScenarioID)
	}
	return nil
}

func printHuman(rep bot.Report) {
	fmt.Printf("cenário: %s (%s)\n", rep.ScenarioID, rep.Objective)
	fmt.Printf("alvo: %s · semente: %d · duração: %s\n", rep.Address, rep.Seed, rep.Duration)
	fmt.Printf("bots: %d (conectados %d, falhas %d)\n", rep.Bots, rep.Connected, rep.Failed)
	fmt.Printf("passos: %d · ações: %d · frames: %d · erros: %d · p99: %.1f ms\n",
		rep.Ticks, rep.Actions, rep.Frames, rep.Errors, rep.P99Millis)
	if len(rep.FinalStates) > 0 {
		fmt.Printf("estados finais: %s\n", formatStates(rep.FinalStates))
	}
	if len(rep.ErrorSamples) > 0 {
		fmt.Printf("amostras de erro:\n  %s\n", strings.Join(rep.ErrorSamples, "\n  "))
	}
	for _, why := range rep.SuccessWhy {
		fmt.Printf("sucesso não satisfeito: %s\n", why)
	}
	for _, why := range rep.FailureWhy {
		fmt.Printf("critério de falha satisfeito: %s\n", why)
	}
	if rep.Success {
		fmt.Println("veredito: PASS")
	} else if rep.Failure || len(rep.SuccessWhy) > 0 {
		fmt.Println("veredito: FAIL")
	} else {
		fmt.Println("veredito: sem critério declarado")
	}
}

func formatStates(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return strings.Join(out, " ")
}

// sortStrings ordena sem puxar dependência extra no comando.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
