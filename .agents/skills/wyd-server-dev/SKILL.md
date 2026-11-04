---
name: wyd-server-dev
description: Desenvolve e altera o servidor Go do WYD2 (tmserver, dbserver, binserver, webserver) respeitando o loop de dono único, codecs binários por offset, paridade de RNG e o contrato do protocolo CPSock. Use ao adicionar ou corrigir handler _MSG_*, mudar pacote/struct de rede, criar migração de banco, ajustar combate/IA/skill, ou revisar código do servidor antes de commitar.
---

# Desenvolvimento do servidor Go

`` é a reescrita big-bang em Go. `AGENTS.md` e `CLAUDE.md` são
autoritativos; esta skill é o atalho operacional para não quebrar as invariantes.

## 1. Mapa do código

```

├── tmserver/            # jogo: borda CPSock + estado do mundo
│   ├── cmd/tmserver/    # wiring: flags, listeners, loaders de conteúdo
│   └── internal/
│       ├── protocol/    # HEADER, framing, Types, codecs (pack(1)) — testes byte-exatos
│       ├── world/       # World.Run (dono único), grid, sessões, tick
│       ├── handler/     # dispatch.go + um arquivo por domínio (login, combat, item, shop…)
│       ├── combat/ refine/ combine/ loot/ level/ rng/ route/ content/ npctemplate/
│       └── dbclient/ binclient/ worldcfg/ npccfg/ mobstat/
├── dbserver/            # gRPC de persistência (Postgres/pgx v5) + seed-account/convert
├── binserver/           # billing (allow-all por padrão)
├── webserver/           # web-api (api/web/v1) para a plataforma Next.js
├── api/                 # .proto (db, bin, web) — make proto
├── internal/            # store, migrations, domain, secure (mTLS), secret (argon2id)
├── Release/             # conteúdo carregado no boot (ver skill wyd-server-ops)
└── docs/migration/      # engenharia reversa: leia antes de mudar wire/gameplay
```

## 2. As quatro regras que quebram o servidor se ignoradas

1. **Dono único.** `world.World.Run` é a única goroutine que toca estado do mundo. Handler roda
   dentro dela (mutação direta, sem lock). Chamada bloqueante → `World.Go` e retorno por canal.
   Se você precisar de `sync.Mutex` no estado de jogo, o desenho está errado.
2. **Offset explícito.** wire `pack(1)`, save `alinhamento natural MSVC x86`. Nada de
   `unsafe`/`binary.Read` em struct Go. `STRUCT_MOB`=816, `STRUCT_ITEM`=8, `STRUCT_SELCHAR`=840,
   `STRUCT_ACCOUNTFILE`=7952. Todo codec novo merece teste byte-exato.
3. **Ordem do RNG.** `rng.MSVC` é o `rand()` do MSVC. Em drop/refino/crítico a **ordem** das
   chamadas é o contrato. Não introduza sorteio novo no meio de um caminho já coberto por golden.
   Sorteios de evento global usam stream dedicado (`Dispatcher.eventRNG`) justamente para não
   invalidar esses goldens.
4. **Autoridade de cena.** Pacote S→C autoritativo (attack, score) vai com `HEADER.ID = ESCENE_FIELD`
   (30000, `protocol.IDScene`), não com o id da conexão do dono. Errado = barra de EXP parada e
   personagem que "não upa" mesmo com o servidor contando certo.

## 3. Adicionar ou corrigir um handler

1. Ache o contrato do `_MSG_*` em `docs/migration/handlers/` e o `Type` em
   `tmserver/internal/protocol/types.go`. Se não houver contrato, o pacote é UNVERIFIED: implemente o
   caminho conservador e logue.
2. Implemente no arquivo de domínio certo (`handler/login.go`, `combat.go`, `item.go`, `shop.go`,
   `trade.go`, `quest.go`, `chat.go`, `gm.go`, …). Não crie arquivo novo se já existe domínio.
3. Registre a rota em `dispatch.go` (`d.routes[protocol.MsgX] = d.x`). Rota não registrada =
   pacote chega, loga `routed=false` e é ignorado — é o primeiro lugar a olhar quando "o cliente
   manda e nada acontece".
4. Escreva teste **no mesmo pacote**, table-driven, com `-race`, no padrão dos existentes
   (`handler/*_test.go`, `protocol/*_test.go`). Teste de handler precisa espelhar o **wiring de
   produção**: se o comportamento depende de `Config.Spells`/`BaseMobs`/`ItemEffects`, monte o
   dispatcher com esses campos ligados. Foi exatamente esse buraco que escondeu uma regressão em que
   todo dano player→mob morreu.
5. Se o handler mudar estado persistido, atualize a migração/proto/save e o `SESSION-PRIMER`
   (a lista de "o que já funciona" é viva).

## 4. Persistência

- Schema: `internal/migrations/NNNN_nome.up.sql` (+ `.down.sql`). Aplicado no boot por
  `store.Migrate`; nunca "crie tabela" à mão.
- RPC: `api/db/v1/db.proto` → `make proto` (precisa `protoc` + `protoc-gen-go[-grpc]`).
- **Depois de mexer em `.proto` ou migração**: `docker compose build --no-cache dbserver tmserver`.
  Layer em cache já causou "o banco tem o valor certo e o serviço devolve antigo".
- Save de personagem é assíncrono (`World.Go`); para logout-para-seleção use
  `World.SaveCharacterThen` (confirma ao cliente só depois do commit). `shutdown()` espera saves em
  voo (`saveWG`).
- Senha/PIN sempre `internal/secret` (argon2id). Texto plano é proibido, inclusive em log.

## 5. Ciclo de verificação

```bash
# (comandos rodam na raiz do repositório)
gofmt -l . && go vet ./...            # ou: make fmt / make vet
go build ./...
go test -race ./tmserver/internal/... # foco no pacote que você tocou
go test -race ./...                   # antes de fechar a tarefa
```

Sem Go no host:

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine \
  sh -c "go build ./... && go test -race ./tmserver/internal/..."
```

`make lint` exige `golangci-lint` v2. Rode o que existir e **relate o que não foi possível rodar** em
vez de declarar verde.

## 6. Convenções do repositório

Siga `development-guidelines/Go-development-guidelines.md`. Em resumo:

- `cmd/<bin>/main.go` só faz wiring; lógica em `internal/`.
- Nomes idiomáticos (`MixedCaps`), arquivos `snake_case.go`, getter sem prefixo `Get`.
- Erro é o último retorno, com `%w` e contexto; ignorar erro exige comentário justificando.
- `context.Context` primeiro parâmetro.
- Comente o **porquê** (principalmente quirk de paridade), não o **o quê**.
- Testes table-driven, mesmo pacote, `-race`.

## 7. Armadilhas conhecidas (custaram horas de debug)

| Sintoma | Causa real | Prevenção |
|---------|-----------|-----------|
| Servidor calcula certo, cliente mostra errado | valor errado **no pacote S→C** | logue o pacote, não só o estado |
| Dano player→mob zera, mob→player ok | gate em campo UNVERIFIED (sentinela de skill) | tolerar + logar; nunca gatear em não confirmado |
| Barra de EXP parada | `HEADER.ID` da conexão em vez de `ESCENE_FIELD` | regra §2.4 |
| Mudança "não pega" no container | imagem em cache | `build --no-cache` |
| Login trava em "conectando" | link gRPC tmServer↔dbServer morto | reiniciar os dois; ver logs |
| Buff vaza entre personagens da mesma conexão | `Entity.ResetAffects()` só no login/logout | resetar em toda troca de char |
| Affects/score somando duas vezes após relogin | valor cacheado bakeado no score flat | manter contribuição read-time (`Aff*`/`Rsv`) |

## 8. Antes de commitar

- `git diff` revisado; sem `Release/` alterado por acidente, sem `serverlist.bin` de IP privado, sem
  segredo.
- Se mudou comportamento observável, atualize `docs/migration/SESSION-PRIMER.md` e/ou
  `docs/game.md` (✅/⏳).
- Se fechou uma issue do `roadmap/`, marque o status em `roadmap/index.md`.
