// Wiring do binserver, separado do `main` para ser testável sem `os.Args`, sem
// abrir socket e sem depender do processo ter subido (REFACTOR-01).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"

	binv1 "openwyd-nextgen/api/bin/v1"
	"openwyd-nextgen/binserver/internal/billing"
	"openwyd-nextgen/binserver/internal/grpcsrv"
	"openwyd-nextgen/internal/metrics"
	"openwyd-nextgen/internal/secure"
)

// config é a configuração resolvida de uma execução: tudo o que `newApp`
// precisa, sem nenhum acesso a `flag` ou ao ambiente.
type config struct {
	addr         string
	allowUnknown bool
	tlsCert      string
	tlsKey       string
	tlsCA        string
	metricsAddr  string
}

// parseConfig lê as flags (e os defaults vindos do ambiente) para o `config`.
// Flag inválida devolve erro em vez de encerrar o processo: quem decide o código
// de saída é o `main`, como em qualquer outro erro de boot.
func parseConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("binserver", flag.ContinueOnError)
	var cfg config
	fs.StringVar(&cfg.addr, "addr", ":3000", "gRPC listen address")
	fs.BoolVar(&cfg.allowUnknown, "allow-unknown", true, "free-to-play: allow accounts with no billing record")
	fs.StringVar(&cfg.tlsCert, "tls-cert", os.Getenv("W2PP_TLS_CERT"), "server certificate (PEM)")
	fs.StringVar(&cfg.tlsKey, "tls-key", os.Getenv("W2PP_TLS_KEY"), "server private key (PEM)")
	fs.StringVar(&cfg.tlsCA, "tls-ca", os.Getenv("W2PP_TLS_CA"), "client CA (PEM) for mTLS")
	fs.StringVar(&cfg.metricsAddr, "metrics-addr", envOr("W2PP_METRICS_ADDR", ":9103"), "endpoint de métricas/health (Prometheus) em porta separada; vazio desliga")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	return cfg, nil
}

// app é o serviço montado e pronto para servir. O listener só existe em `run`,
// então construir o `app` num teste não abre porta nenhuma.
type app struct {
	srv          *grpc.Server
	addr         string
	allowUnknown bool
	mtls         bool
}

// newApp monta o gRPC do binserver a partir do `config`: credenciais, métricas e
// o registro do BillingService. Não abre porta e não bloqueia.
func newApp(ctx context.Context, cfg config, logger *slog.Logger) (*app, error) {
	metricsReg := metrics.New("binserver")
	go metrics.Serve(ctx, cfg.metricsAddr, metricsReg, logger)

	creds, err := secure.ServerCreds(secure.Config{CertFile: cfg.tlsCert, KeyFile: cfg.tlsKey, CAFile: cfg.tlsCA})
	if err != nil {
		return nil, err
	}
	srv := grpc.NewServer(grpc.Creds(creds), grpc.ChainUnaryInterceptor(metrics.UnaryServerInterceptor("binserver", metricsReg, logger)))
	binv1.RegisterBillingServiceServer(srv, grpcsrv.New(billing.New(cfg.allowUnknown), nil))
	return &app{srv: srv, addr: cfg.addr, allowUnknown: cfg.allowUnknown, mtls: cfg.tlsCert != ""}, nil
}

// run abre o listener e serve até SIGINT/SIGTERM (ctx cancelado), quando para de
// forma graciosa. Um erro do `Serve` que não seja o encerramento normal volta
// para o `main`.
func (a *app) run(ctx context.Context, logger *slog.Logger) error {
	ln, err := net.Listen("tcp", a.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", a.addr, err)
	}
	logger.Info("binserver serving", "addr", a.addr, "allow_unknown", a.allowUnknown, "mtls", a.mtls)

	errCh := make(chan error, 1)
	go func() { errCh <- a.srv.Serve(ln) }()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
		a.srv.GracefulStop()
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}
}

// envOr devolve o valor de ambiente de key, ou def quando não está definida.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
