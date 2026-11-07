// Command tmserver é o servidor de jogo do WYD (TMSrv): fala o protocolo de wire
// CPSock legado com o cliente WYD.exe 7662 sem modificação (tmserver/internal/protocol)
// e é dono do estado do mundo em memória por uma única goroutine de laço de jogo
// (tmserver/internal/world).
//
// Este entrypoint só faz ligação (guidelines §3): flags, logging, as conexões
// de cliente gRPC para dbServer/binServer, o listener e o shutdown gracioso.
// Sem -dbserver a persistência cai para no-op (bring-up local).
// A ligação em si vive em app.go (REFACTOR-01), para um teste poder montar o mundo
// sem abrir uma porta.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"openwyd-nextgen/internal/metrics"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Healthcheck do compose: a imagem final é distroless (sem shell), então o
	// próprio binário consulta /healthz e sai 0/1.
	metrics.HealthcheckArgs(":9101")
	cfg, err := parseConfig(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		logger.Error("tmserver: invalid configuration", "err", err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := newApp(ctx, cfg, logger)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info("tmserver stopped")
			return
		}
		logger.Error("tmserver stopped with error", "err", err)
		os.Exit(1)
	}
	if err := a.run(ctx, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("tmserver stopped with error", "err", err)
		os.Exit(1)
	}
	logger.Info("tmserver stopped")
}
