# REFACTOR-01 — Wiring dos `main.go`: extrair `newApp` por serviço

**Fase:** revisão · **Depende de:** — · **Esforço:** M · **Estado:** ✅ entregue em 06/11/2026

## Objetivo

Cada `cmd/*/main.go` só lê flags, chama `newApp(...)` e roda. O wiring (store → serviços → gRPC)
passa a ser testável sem `os.Args`, sem abrir socket e sem banco real.

## Entregue (06/11/2026)

Os quatro serviços ganharam um `app.go` com a mesma forma: `config` (valores já resolvidos),
`parseConfig(args)` (flags + defaults do ambiente), `newApp(ctx, cfg, logger)` (I/O de boot e
montagem) e o método `run(ctx, logger)` (listener e `Serve`). O `main.go` ficou só com
healthcheck → `parseConfig` → `newApp` → `run`, com o código de saída decidido por ele.

| Serviço | `main.go` antes | `main.go` depois | Wiring em |
|---|---:|---:|---|
| `binserver` | 95 | **51** | `binserver/cmd/binserver/app.go` |
| `dbserver` | 301 | **54** | `dbserver/cmd/dbserver/app.go` (+ `convert.go`, `seedaccount.go`) |
| `tmserver` | 874 | **56** | `tmserver/cmd/tmserver/app.go` |
| `webserver` | 285 | **33** | `webserver/cmd/webserver/app.go` |
| **total** | **1.555** | **194** | |

Mudanças de comportamento, todas exigidas pelos critérios abaixo:

- **`tmserver`**: o servidor HTTP de status (`:80`) e o `net.Listen` do CPSock saíram do wiring para
  `run`. As conexões gRPC abertas no `newApp` ficam numa lista de `closers` fechada no shutdown (antes
  eram `defer` dentro de `run`, que também só fechava na saída — mesmo comportamento, agora explícito).
- **`webserver`**: catálogo de itens vazio no banco deixou de ser `WARN` e passou a ser **erro** de
  `newApp` ("item_catalog is empty: seed the content catalogue"). Era o único dos quatro que subia
  "saudável" com o picker vazio.
- **`dbserver`**: `convert` e `seed-account` saíram para arquivos próprios; `serve` virou
  `parseServeConfig` + `newApp` + `run`. A checagem de catálogo de NPCs (`ContentNPCDefinitionCount`)
  continua fail-closed e agora fecha o pool quando falha.
- **`binserver`**: inalterado em comportamento; só o wiring mudou de lugar.

### Costura de teste

`newApp` continua sendo quem faz o I/O (pool, migrations, catálogos). Para testar a montagem sem
Postgres, cada serviço expõe o passo seguinte como função separada — `newAppWithDeps` (webserver) e
`newServer` (dbserver), ambos chamados por `newApp` — de modo que o teste exercita **o mesmo código de
registro** que a produção, com dependências já resolvidas. Os testes não abrem porta: montam o gRPC e
falam com ele por `bufconn`, ou conferem `GetServiceInfo()`.

## Contexto/evidência (05/11/2026, antes da entrega)

- `webserver/cmd/webserver/main.go` tinha **256 linhas** e concentrava: flags, credenciais TLS, pool,
  migração, **todos** os construtores de serviço, o carregamento de catálogo e o registro de ~20
  serviços gRPC. Nada disso era exercitável em teste — e foi exatamente aí que a issue 62 nasceu
  (o wiring escaneava arquivo que não existe mais).
- O mesmo padrão se repetia em `tmserver/cmd/tmserver`, `dbserver/cmd/dbserver`,
  `binserver/cmd/binserver`.
- Consequência prática: qualquer mudança de wiring só era verificada subindo container (build + boot
  de minutos) em vez de um teste de milissegundos.

## Passos (todos concluídos)

1. ~~Definir a forma: `type app struct { srv *grpc.Server; ln net.Listener; stop func() }` e
   `newApp(ctx, cfg) (*app, error)` por serviço, recebendo um `cfg` explícito (nada de `flag` dentro).~~
   Feito: `app` guarda `srv`/`w`/`addr`/`closers`, e `newApp` não abre listener.
2. ~~`main` fica: `flags → cfg → newApp → run(ctx) → graceful stop`.~~ Feito nos quatro.
3. ~~Um teste por serviço que monta `newApp` e verifica os serviços gRPC registrados via `bufconn`.~~
   Feito: `TestNewAppServesBillingOverBufconn`, `TestNewServerRegistersEveryService`,
   `TestNewAppWithDepsServesItemCatalogOverBufconn`, e no tmserver
   `TestNewAppBuildsWorldWithoutListening` (prova a porta ainda livre depois do `newApp`).
4. ~~Mover para `newApp` o carregamento de catálogo para que a ausência de conteúdo seja erro
   testável, não WARN de boot.~~ Feito no `webserver` (erro) e coberto por
   `TestNewAppWithDepsRejectsEmptyItemCatalog`; `dbserver` e `tmserver` já eram fail-closed.
5. ~~Medir: `wc -l cmd/*/main.go` antes/depois; a meta é `main.go` < 80 linhas em cada serviço.~~
   Feito: 51/54/56/33 (tabela acima).

## Resultado medido (06/11/2026)

| Prova | Comando | Resultado |
|---|---|---|
| compila | `go build ./...` | RC 0 (38 s) |
| vet | `go vet ./...` | RC 0 (10 s) |
| testes dos quatro pacotes | `go test -race -count=1 ./binserver/cmd/binserver/ ./dbserver/cmd/dbserver/ ./webserver/cmd/webserver/ ./tmserver/cmd/tmserver/` | RC 0 (35 s): 4 × `ok` |
| tamanho dos mains | `wc -l */cmd/*/main.go` | serviços: 51, 54, 56, 33 |
| paridade das flags | nomes extraídos de `git show HEAD:<main.go>` × `app.go` | webserver 8=8, binserver 6=6, tmserver 33=33, dbserver `serve` 6=6 (todas idênticas); `convert` (2) e `seed-account` (3) mantiveram as suas em arquivos próprios |
| skip com motivo | `bash scripts/check-skips.sh` | RC 0: 47 skip(s), todos com motivo (nenhum novo) |
| comentários | `bash scripts/check-comments.sh` | RC 0: 3 diretório(s) limpos |
| links de docs | `bash scripts/check-docs-links.sh` | RC 0 |
| formatação | `gofmt -l` nos arquivos alterados | vazio (fora dos dois `_test.go` antigos em CRLF, anteriores a esta entrega) |

O boot do compose não foi executado nesta rodada (o host é a máquina do usuário e a stack não estava
em uso); a paridade é por construção e foi medida pela comparação nome a nome das flags, com os
mesmos defaults e os mesmos fallbacks `W2PP_*` (`grep -c 'W2PP_'`: 11/4/7/35 nos quatro `app.go`).

## Critérios de aceite

- [x] Nenhum `cmd/*/main.go` de **serviço** com mais de 80 linhas — 33 a 56 nos quatro.
      As ferramentas de linha de comando (`importcontent` 567, `worldatlas` 656, `seedgen` 176,
      `botlab` 206, `exptool` 68, `importteleports` 81, `itemiconpack` 44, `itemiconupload` 57) não
      são serviços e não aparecem em "Arquivos": nelas o `main` **é** o comando, não o wiring.
- [x] `newApp` coberto por teste que não abre porta real: três serviços falam por `bufconn`; no
      tmserver o teste confere que a porta configurada continua livre depois do `newApp`.
- [x] Boot do compose idêntico: mesmas flags, mesmos nomes, mesmos defaults e mesmas variáveis de
      ambiente (comprovado por comparação automática contra `HEAD`).
- [x] Catálogo vazio no banco produz erro explícito de `newApp`, não `WARN` silencioso — no
      `webserver` isso passou a valer nesta entrega; `dbserver` e `tmserver` já recusavam o boot.

## Arquivos

- `tmserver/cmd/tmserver/{main,app}.go`, `dbserver/cmd/dbserver/{main,app,convert,seedaccount}.go`,
  `binserver/cmd/binserver/{main,app}.go`, `webserver/cmd/webserver/{main,app}.go`
- `{binserver,dbserver,webserver,tmserver}/cmd/*/app_test.go` (novos)
- `roadmap/index.md`, `roadmap/PRE-RELEASE-PROGRAM.md`, `.agents/skills/wyd-server-ops/SKILL.md`,
  `docs/migration/mob-template-editing-plan.md` (caminhos de wiring que mudaram de arquivo)

## Riscos/rollback

Risco: refatorar wiring e mudar ordem de registro/migração, quebrando boot de um serviço. Mitigação
aplicada: um serviço por vez, com `go build`/`go vet` entre eles, e a ordem de registro preservada
linha a linha (só mudou de arquivo). Rollback: cada `main.go` é independente; nada foi commitado.
