// Command binserver é o serviço de cobrança (BISrv), desenhado do zero como
// serviço gRPC, com API (api/bin/v1) e política (binserver/internal/billing)
// próprias. O link legado _AUTH_GAME de 196 bytes não foi verificado e não é
// reproduzido de propósito.
//
// Uso:
//
//	binserver [-addr :3000] [-allow-unknown] [-tls-cert … -tls-key … -tls-ca …]
//
// Serve o BillingService por gRPC (+mTLS quando os caminhos de certificado estão
// definidos) até SIGINT/SIGTERM.
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
	metrics.HealthcheckArgs(":9103")
	cfg, err := parseConfig(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		logger.Error("binserver: invalid configuration", "err", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	a, err := newApp(ctx, cfg, logger)
	if err != nil {
		logger.Error("binserver failed", "err", err)
		os.Exit(1)
	}
	if err := a.run(ctx, logger); err != nil {
		logger.Error("binserver stopped with error", "err", err)
		os.Exit(1)
	}
	logger.Info("binserver stopped")
}
