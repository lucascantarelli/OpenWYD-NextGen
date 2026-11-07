// Command dbserver é o serviço de persistência (DBSrv). Ele traz o conversor
// one-shot de arquivo de conta e o AccountService gRPC (api/db/v1) que o tmServer
// consome via gRPC+mTLS (migration-plan.md §3.5).
//
// Uso:
//
//	dbserver convert -accounts <dir> [-dsn <postgres-url>]
//	dbserver serve [-addr :7514] -dsn <postgres-url> [-tls-cert … -tls-key … -tls-ca …]
//	dbserver seed-account -name <login> -pass <password> [-dsn <postgres-url>]
//
// convert: sem -dsn é simulação; com -dsn aplica as migrations e persiste cada
// conta convertida. serve: aplica as migrations e então serve gRPC.
// seed-account: cria uma conta só com senha (sem personagens) para teste local;
// o jogador cria personagens no cliente. Idempotente.
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
