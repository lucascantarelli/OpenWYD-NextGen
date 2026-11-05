# REFACTOR-01 — Wiring dos `main.go`: extrair `newApp` por serviço

**Fase:** revisão · **Depende de:** — · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

Cada `cmd/*/main.go` só lê flags, chama `newApp(...)` e roda. O wiring (store → serviços → gRPC)
passa a ser testável sem `os.Args`, sem abrir socket e sem banco real.

## Contexto/evidência (05/11/2026)

- `webserver/cmd/webserver/main.go` tem **256 linhas** e concentra: flags, credenciais TLS, pool,
  migração, **todos** os construtores de serviço, o carregamento de catálogo e o registro de ~20
  serviços gRPC. Nada disso é exercitável em teste — e foi exatamente aí que a issue 62 nasceu
  (o wiring escaneava arquivo que não existe mais).
- O mesmo padrão se repete em `tmserver/cmd/tmserver`, `dbserver/cmd/dbserver`,
  `binserver/cmd/binserver`.
- Consequência prática: qualquer mudança de wiring só é verificada subindo container (build + boot
  de minutos) em vez de um teste de milissegundos.

## Passos

1. Definir a forma: `type app struct { srv *grpc.Server; ln net.Listener; stop func() }` e
   `newApp(ctx, cfg) (*app, error)` por serviço, recebendo um `cfg` explícito (nada de `flag` dentro).
2. `main` fica: `flags → cfg → newApp → run(ctx) → graceful stop`.
3. Um teste por serviço que monta `newApp` com DSN de teste (ou `store` falso) e verifica que os
   serviços gRPC registrados respondem a um `Health`/`Ping` via `bufconn` (o padrão já usado em
   `dbserver/internal/grpcsrv/bufconn_test.go`).
4. Mover para `newApp` o carregamento de catálogo (itens, skills, templates, drops) para que a
   ausência de conteúdo no banco seja um erro testável, não um WARN de boot.
5. Medir: `wc -l cmd/*/main.go` antes/depois; a meta é `main.go` < 80 linhas em cada serviço.

## Arquivos

- `tmserver/cmd/tmserver/main.go`, `dbserver/cmd/dbserver/main.go`,
  `binserver/cmd/binserver/main.go`, `webserver/cmd/webserver/main.go`
- `*/cmd/*/app.go` (novo) e `app_test.go` (novo)
- `docs/development-guidelines/Go-comments.md` (referência de estilo, criado na REFACTOR-03)

## Testes/verificação

```bash
go build ./...
go test ./webserver/cmd/webserver/ ./dbserver/cmd/dbserver/ -run TestNewApp -v
wc -l */cmd/*/main.go    # cada um < 80
```

## Critérios de aceite

- [ ] Nenhum `cmd/*/main.go` com mais de 80 linhas.
- [ ] `newApp` coberto por teste que não abre porta real (bufconn/socket em memória).
- [ ] Boot do compose continua idêntico (mesmas flags, mesmas variáveis).
- [ ] Catálogo vazio no banco produz erro explícito de `newApp`, não WARN silencioso.

## Riscos/rollback

Risco: refatorar wiring e mudar ordem de registro/migração, quebrando boot de um serviço. Mitigação:
um serviço por commit, com o compose subindo entre eles. Rollback: cada `main.go` é independente.
