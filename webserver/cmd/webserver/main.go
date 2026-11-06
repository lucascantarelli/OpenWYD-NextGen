// Command webserver é o web-api: a borda gRPC que o BFF do Next.js chama no
// servidor. É dono dos fluxos de conta da plataforma web (cadastro, conferência
// de credencial) sobre a mesma tabela `account` e o mesmo argon2id do dbServer,
// mas é um serviço SEPARADO do AccountService legado do dbServer.
//
// Uso:
//
//	webserver [-addr :7600] -dsn <postgres-url> [-tls-cert … -tls-key … -tls-ca …] [-content <tmserver/content/>]
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
	metrics.HealthcheckArgs(":9104")
	err := runFromArgs(os.Args[1:], logger)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		logger.Error("webserver failed", "err", err)
		os.Exit(1)
	}
}
