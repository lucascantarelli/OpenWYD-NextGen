// Wiring do modo `serve` do dbserver, separado do `main` para ser testável sem
// `os.Args`, sem abrir socket e sem banco real (REFACTOR-01).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/dbserver/internal/grpcsrv"
	"openwyd-nextgen/internal/metrics"
	"openwyd-nextgen/internal/secure"
	"openwyd-nextgen/internal/store"
)

// serveConfig é a configuração resolvida do subcomando `serve`.
type serveConfig struct {
	addr        string
	dsn         string
	tlsCert     string
	tlsKey      string
	tlsCA       string
	metricsAddr string
}

// parseServeConfig lê as flags do `serve` (defaults vindos do ambiente) para o
// `serveConfig`, sem encerrar o processo quando uma flag é inválida.
func parseServeConfig(args []string) (serveConfig, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	var cfg serveConfig
	fs.StringVar(&cfg.addr, "addr", ":7514", "gRPC listen address")
	fs.StringVar(&cfg.dsn, "dsn", envOr("W2PP_DB_DSN", ""), "PostgreSQL DSN (or W2PP_DB_DSN)")
	fs.StringVar(&cfg.tlsCert, "tls-cert", os.Getenv("W2PP_TLS_CERT"), "server certificate (PEM)")
	fs.StringVar(&cfg.tlsKey, "tls-key", os.Getenv("W2PP_TLS_KEY"), "server private key (PEM)")
	fs.StringVar(&cfg.tlsCA, "tls-ca", os.Getenv("W2PP_TLS_CA"), "client CA (PEM) for mTLS")
	fs.StringVar(&cfg.metricsAddr, "metrics-addr", envOr("W2PP_METRICS_ADDR", ":9102"), "endpoint de métricas/health (Prometheus) em porta separada; vazio desliga")
	if err := fs.Parse(args); err != nil {
		return serveConfig{}, err
	}
	return cfg, nil
}

// app é o servidor montado e pronto para servir; o listener só existe em `run`,
// então construir o `app` num teste não abre porta nenhuma.
type app struct {
	srv  *grpc.Server
	pool *pgxpool.Pool
	addr string
	mtls bool
}

// newApp abre o pool, aplica as migrations, exige o catálogo de NPCs e registra
// os serviços gRPC. Não abre porta e não bloqueia.
func newApp(ctx context.Context, cfg serveConfig, logger *slog.Logger) (*app, error) {
	if cfg.dsn == "" {
		return nil, fmt.Errorf("-dsn (or W2PP_DB_DSN) is required")
	}
	pool, err := pgxpool.New(ctx, cfg.dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := store.MigrateWithEnv(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	st := store.New(pool)
	// O catálogo de spawn vem do banco e SÓ dele (issue 57/58): NPCGener.txt saiu
	// do conteúdo na fase 5. O antigo reconcile a cada boot era justamente o que
	// fazia o banco divergir do seed (npc_shop_item 6.711 × 3.538).
	// Falha fechada no requisito real: ALGUMA fonte precisa ter produzido o
	// catálogo. Subir com zero spawns parece um servidor saudável que nunca
	// invoca monstro, o que é bem mais difícil de diagnosticar que uma recusa.
	n, cerr := st.ContentNPCDefinitionCount(ctx)
	if cerr != nil {
		pool.Close()
		return nil, fmt.Errorf("count NPC catalogue: %w", cerr)
	}
	if n == 0 {
		pool.Close()
		return nil, fmt.Errorf("no NPC catalogue in the database: seed it (tmserver/cmd/seedgen) — the world would boot with no monsters")
	}
	logger.Info("npc catalogue ready", "content_definitions", n)

	metricsReg := metrics.New("dbserver")
	go metrics.Serve(ctx, cfg.metricsAddr, metricsReg, logger)
	metrics.RegisterPGXPool(metricsReg, "dbserver", pool)
	metrics.RegisterTableRows(metricsReg, "dbserver", "delivery_queue_rows", "Linhas na delivery_queue.", "SELECT count(*) FROM delivery_queue", pool)

	srv, err := newServer(cfg, st, metricsReg, logger)
	if err != nil {
		pool.Close()
		return nil, err
	}
	return &app{srv: srv, pool: pool, addr: cfg.addr, mtls: cfg.tlsCert != ""}, nil
}

// newServer monta o gRPC (credenciais, interceptor e registro dos serviços) a
// partir de um store já resolvido. É o passo sem I/O de banco que o teste usa.
func newServer(cfg serveConfig, st *store.Store, metricsReg *metrics.Registry, logger *slog.Logger) (*grpc.Server, error) {
	creds, err := secure.ServerCreds(secure.Config{CertFile: cfg.tlsCert, KeyFile: cfg.tlsKey, CAFile: cfg.tlsCA})
	if err != nil {
		return nil, err
	}
	srv := grpc.NewServer(grpc.Creds(creds), grpc.ChainUnaryInterceptor(metrics.UnaryServerInterceptor("dbserver", metricsReg, logger)))
	dbv1.RegisterAccountServiceServer(srv, grpcsrv.New(st).WithCrossoverKey(os.Getenv("W2PP_CROSSOVER_KEY")))
	dbv1.RegisterNpcConfigServiceServer(srv, grpcsrv.NewNpcConfig(st))
	// Catálogos de conteúdo (issue 58): o tmServer lê itens/skills/templates/
	// objetos do mundo daqui em vez dos arquivos de tmserver/content/.
	dbv1.RegisterContentCatalogServiceServer(srv, grpcsrv.NewContentCatalog(st))
	dbv1.RegisterTeleportConfigServiceServer(srv, grpcsrv.NewTeleportConfig(st))
	dbv1.RegisterItemEffectConfigServiceServer(srv, grpcsrv.NewItemEffectConfig(st))
	dbv1.RegisterContentRateConfigServiceServer(srv, grpcsrv.NewContentRateConfig(st))
	dbv1.RegisterInstanceFlagsServiceServer(srv, grpcsrv.NewInstanceFlags(st))
	dbv1.RegisterDropConfigServiceServer(srv, grpcsrv.NewDropConfig(st))
	dbv1.RegisterWorldEventConfigServiceServer(srv, grpcsrv.NewWorldEventConfig(st))
	return srv, nil
}

// run abre o listener e serve até o ctx ser cancelado (SIGINT/SIGTERM).
func (a *app) run(ctx context.Context, logger *slog.Logger) error {
	defer a.pool.Close()
	ln, err := net.Listen("tcp", a.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", a.addr, err)
	}
	logger.Info("dbserver serving", "addr", a.addr, "mtls", a.mtls)

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

// runServe resolve a configuração do `serve`, monta o app e serve até o
// SIGINT/SIGTERM. O sinal vive aqui, não no wiring, para que o teste possa
// construir o app com um ctx qualquer.
func runServe(args []string, logger *slog.Logger) error {
	cfg, err := parseServeConfig(args)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	a, err := newApp(ctx, cfg, logger)
	if err != nil {
		return err
	}
	return a.run(ctx, logger)
}

// usage imprime os três subcomandos; o `main` só o chama quando falta comando ou
// ele é desconhecido.
func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  dbserver convert -accounts <dir> [-dsn <postgres-url>]")
	fmt.Fprintln(os.Stderr, "  dbserver serve [-addr :7514] -dsn <postgres-url> [-tls-cert -tls-key -tls-ca]")
	fmt.Fprintln(os.Stderr, "  dbserver seed-account -name <login> -pass <password> [-dsn <postgres-url>]")
}

// envOr devolve o valor de ambiente de key, ou def quando não definida.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
