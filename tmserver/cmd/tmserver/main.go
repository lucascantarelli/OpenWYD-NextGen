// Command tmserver is the WYD game server (TMSrv): it speaks the legacy CPSock
// wire protocol to the unmodified WYD.exe 7662 client (tmserver/internal/protocol)
// and owns the in-memory world state through a single game-loop goroutine
// (tmserver/internal/world).
//
// This entrypoint only does wiring (guidelines §3): flags, logging, the gRPC
// client connections to dbServer/binServer, the listener and graceful shutdown.
// Without -dbserver the persistence falls back to a no-op (local bring-up).
// The wiring itself lives in app.go (REFACTOR-01), so a test can mount the world
// without opening a port.
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
