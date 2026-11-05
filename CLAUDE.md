# CLAUDE.md

Guia para agentes que trabalham neste repositório. O documento **principal** é `AGENTS.md`
(estrutura, invariantes, verificação, portal e skills) — leia-o primeiro. Este arquivo resume o
essencial de arquitetura e comandos; não repita conteúdo nele.

## O que é

Reescrita big-bang em Go do servidor WYD (With Your Destiny), mirando o **`WYD.exe` 7662 sem
modificação** (o build usado envia `ClientVersion` **12000** no login — medido em 04/11; `Basedef.h`
documenta 7640), mais o portal de administração em Next.js (`portal/`). Os fontes
legados em C++ ficam em `Source/`, o conteúdo do jogo em `Release/`, e a engenharia reversa em
`docs/migration/` (leia antes de mudar wire/format/gameplay; pontos não confirmados estão marcados
**UNVERIFIED**).

Module path: `openwyd-nextgen` (Go 1.26, Linux/Docker).

## Comandos

```bash
make build          # go build ./...
make binaries       # binários em bin/ (tmserver, dbserver, binserver, webserver)
make test           # go test -race -cover ./...
make lint           # golangci-lint run
make vet            # go vet ./...
make fmt            # gofmt + goimports
make proto          # regen gRPC de api/*.proto
make certs          # certs mTLS de dev (gitignored)

make run            # tmserver sozinho, persistência no-op
make run-local      # stack completa (db, dbServer, binServer, webServer, portal, tmServer) + conta test/test123

make portal-install # npm ci em portal/
make portal-dev     # portal em http://127.0.0.1:3000 (hot reload)
make portal-test    # tsc --noEmit + vitest
make portal-build   # next build (standalone)
```

Testes pontuais:

```bash
go test -run TestChecksum ./tmserver/internal/protocol
go test -race ./tmserver/internal/world
go test -tags=integration ./internal/store/...   # precisa de Postgres
```

## Arquitetura (resumo)

Quatro serviços Go; só a borda client↔tmServer fala o protocolo legado, os links internos são gRPC
(+mTLS):

- **tmServer** (`:8281` jogo, `:80` status) — dono de todo o estado do mundo numa única goroutine,
  sem locks.
- **dbServer** (`:7514`) — persistência PostgreSQL/pgx v5 (`api/db/v1`).
- **binServer** (`:3000`) — billing (`api/bin/v1`).
- **webServer** (`:7600`) — web-api (`api/web/v1`) consumido pelo portal server-side.
- **portal** (`:3000`) — Next.js como BFF puro; nunca fala gRPC do browser nem escreve estado vivo.

Pacotes compartilhados: `internal/store`, `internal/migrations`, `internal/domain`, `internal/secret`
(argon2id). `cmd/<svc>/main.go` só faz wiring; lógica em `internal/`.

A invariante que sustenta tudo: **estado do mundo em uma goroutine** (`world.World.Run`), handlers
rodam dentro dela, chamadas bloqueantes saem via `World.Go`. Detalhes e as demais invariantes
(offsets binários, RNG, `ESCENE_FIELD`, UNVERIFIED, web sem escrita em personagem) estão em
`AGENTS.md §3`.

## Convenções

Siga `development-guidelines/Go-development-guidelines.md`. Em resumo: sem pacotes `util`/`common`,
nomenclatura idiomática, erros com `%w`, `context` primeiro, comentário do *porquê*, GoDoc em
exportados, testes table-driven com `-race`, segredos só via ambiente.
