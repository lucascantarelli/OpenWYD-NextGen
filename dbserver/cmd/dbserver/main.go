// Command dbserver is the persistence service (DBSrv). It ships the one-shot
// account-file converter and the gRPC AccountService (api/db/v1) that tmServer
// consumes over gRPC+mTLS (migration-plan.md §3.5).
//
// Usage:
//
//	dbserver convert -accounts <dir> [-dsn <postgres-url>]
//	dbserver serve [-addr :7514] -dsn <postgres-url> [-tls-cert … -tls-key … -tls-ca …]
//	dbserver seed-account -name <login> -pass <password> [-dsn <postgres-url>]
//
// convert: without -dsn it is a dry run; with -dsn it applies migrations and
// persists each converted account. serve: applies migrations then serves gRPC.
// seed-account: creates a single password-only account (no characters) for local
// testing; the player creates characters in the client. Idempotent.
package main

import (
	"errors"
	"flag"
	"log/slog"
	"os"

	"openwyd-nextgen/internal/metrics"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Healthcheck do compose: a imagem final é distroless (sem shell), então o
	// próprio binário consulta /healthz e sai 0/1.
	metrics.HealthcheckArgs(":9102")
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "convert":
		err = runConvert(os.Args[2:], logger)
	case "serve":
		err = runServe(os.Args[2:], logger)
	case "seed-account":
		err = runSeedAccount(os.Args[2:], logger)
	default:
		usage()
		os.Exit(2)
	}
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		logger.Error(os.Args[1]+" failed", "err", err)
		os.Exit(1)
	}
}
