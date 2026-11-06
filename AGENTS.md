# AGENTS.md — Dolorestec WYD2

Instruções para agentes de IA que trabalham neste repositório. Leia antes de qualquer tarefa:
define o que é o projeto, como subir a stack, quais invariantes não podem ser quebradas e onde
está o conhecimento já produzido.

## 1. O que é este projeto

Servidor local do WYD2 (With Your Destiny) escrito em Go, o portal administrativo em Next.js e o
cliente Windows usado para conectar — tudo num único repositório.

| Pasta | O que é | Stack |
|-------|---------|-------|
| `tmserver/` | Servidor do jogo (CPSock, dono do estado do mundo) | Go 1.26 |
| `dbserver/` `binserver/` `webserver/` | Persistência, billing e web-api por gRPC | Go 1.26, gRPC |
| `internal/` | Store, migrations, domínio, argon2id — compartilhados | Go, pgx v5 |
| `portal/` | UI de administração: BFF puro sobre o web-api, sem tocar estado vivo | Next.js 15, TypeScript |
| `client/` | Cliente Windows modificado (WYD.exe, proteções, assets) — **ignorado no git** | Binários fechados |
| `roadmap/` | Planejamento: o que está feito, pendente e em que ordem | Markdown |
| `tmserver/content/` | Conteúdo do servidor: taxas, mapas, BaseMob/BaseSummon (LEGACY-02) | Dados |
| `Source/` | Fontes C++ legados, referência de wire/format/gameplay | C++ |
| `docs/` | Engenharia reversa (`docs/migration/`), integrações e planos | Markdown |
| `.agents/skills/` | Skills de operação/desenvolvimento (WYD + UI) | Markdown |

O objetivo: **servidor local do WYD2 operacional, com login pelo cliente modificado, personagens,
batalha, comércio e quests funcionando**, mais uma camada de automação por IA para administrar e
evoluir o servidor.

`roadmap/index.md` é a fonte da verdade do planejamento. Para arquitetura Go detalhada, comandos e
convenções, veja `CLAUDE.md`; este arquivo cobre o repositório inteiro (servidor + portal + cliente
+ roadmap + skills).

## 2. Caminho feliz (bring-up do zero)

```bash
# 0. Conteúdo que ainda vive em arquivo: taxas, mapas e BaseMob. Catálogos de item/skill,
#    templates de NPC e spawns vêm do banco (baseline da issue 58).
test -f tmserver/content/Common/serv00.htm && test -d tmserver/content/DBsrv/run/BaseMob || echo "CONTEÚDO FALTANDO"
test -f tmserver/content/TMsrv/run/HeightMap.dat || echo "MAPAS AUSENTES: make content-sync"

# 1. Stack completa (Postgres + dbServer + binServer + webServer + portal + tmServer)
make run-local          # = ./scripts/run-local.sh  (conta test / test123)

# 2. Ver o servidor de jogo e o portal respondendo
docker compose logs -f tmserver        # "tmserver listening" + "NPCs spawned"
curl -s http://127.0.0.1/ | head       # status HTTP :80 (serv00.htm) que o cliente sonda
#    portal: http://localhost:3000     (login com conta de papel moderator/admin)

# 3. Portal em modo desenvolvimento (hot reload), sem container:
make portal-install     # npm ci
make portal-dev         # http://127.0.0.1:3000

# 4. Apontar o cliente para cá
#    "serverlist bin/serverlist editor.exe" -> IP:8281 -> copiar serverlist.bin
#    para "client/" (fazendo backup do original) e abrir WYD.exe
```

Detalhes, diagnóstico e alternativas: skills `wyd-server-ops`, `wyd-client-connect` e
`wyd-web-admin-ui`.

## 3. Invariantes que não podem ser quebradas

1. **Dono único do estado do mundo.** Todo o estado de jogo vive em **uma goroutine**
   (`world.World.Run`). Handlers em `tmserver/internal/handler/` rodam *dentro* dessa goroutine e
   podem mutar estado direto. Chamadas bloqueantes (dbServer/billing) vão para fora via
   `World.Go` e o resultado volta por canal. **Nunca** adicione locks no estado do mundo, nem
   mutação fora do loop — é o que evita duplicação de item.
2. **Layout binário é por offset explícito.** Wire usa `pack(1)`; structs de save usam
   alinhamento natural MSVC x86. Não confie no alinhamento do Go. Tamanhos-chave:
   `STRUCT_MOB`=816, `STRUCT_ITEM`=8, `STRUCT_SELCHAR`=840, `STRUCT_ACCOUNTFILE`=7952.
3. **Paridade de RNG.** O `rand()` do MSVC é um LCG reimplementado em `tmserver/internal/rng`.
   Preserve a **ordem** das chamadas em qualquer código de gameplay (drop, refino, crítico).
4. **Pacote autoritativo de cena vai com `HEADER.ID = ESCENE_FIELD`** (30000), não com o id da
   conexão do dono — senão barra de EXP/HP não atualiza no cliente.
5. **Campo UNVERIFIED não é gate.** Para comportamento de cliente não capturado: tolerar, logar e
   seguir pelo caminho conservador. Nunca rejeitar uma ação server-authoritative por causa de um
   sentinela não confirmado.
6. **Senhas e PINs com argon2id** (`internal/secret`). Texto plano é proibido, inclusive em log.
7. **A web nunca escreve estado de personagem vivo.** Grants vão por `delivery_queue` que o
   tmServer drena (ver `docs/migration/web-platform-plan.md`). O portal é BFF: só lê e envia
   intenções pelo web-api gRPC.
8. **`tmserver/content/` é conteúdo, não lixo.** A config pequena é versionada com o código; os dois
   mapas (17 MiB) ficam **fora do git** e entram por `make content-sync`, conferidos pelo SHA-256 de
   `tmserver/content/content.sha256` (`make content-check`). `tmserver/content/_legacy/` é arquivo
   morto do servidor Win32: ignorado, fora da imagem e nunca lido. Nunca escreva no conteúdo pela web.
9. **`client/` não é versionado.** São ~344 MB de binários proprietários; fica no `.gitignore`.
   Nunca commite, nunca inclua no contexto de build do Docker (já está no `.dockerignore`).

### Regras do portal

- **Toda ação admin passa por `adminCall`** (`portal/src/server/admin.ts`): sessão de staff,
  `moderator_id` vindo da sessão (nunca do corpo), deadline, mapeamento `AdminResult` → HTTP e
  auditoria. Page/action não chama o cliente gRPC admin direto.
- O browser nunca fala gRPC nem vê certificado, DSN ou stack: `toResponse` devolve só `{ error }`
  com `cache-control: no-store`.
- `SESSION_SECRET` é obrigatório fora de desenvolvimento (>= 32 caracteres). O default do compose
  é **apenas para dev**.
- O portal não escreve em `tmserver/content/` nem em tabelas de personagem vivo; overlays têm auditoria.

## 4. Árvore de conteúdo (`tmserver/content/`)

O conteúdo mora **dentro da árvore do servidor** e é a única fonte no boot: o default de `-content`
é `tmserver/content` (nos containers, `/content`). Taxas são **obrigatórias** (erro fatal se
quebrarem); mapas e BaseMob são opcionais (warning).

**Versionamento (LEGACY-02):** a config pequena é versionada junto do código. Os dois mapas (17 MiB)
ficam **fora do git** e entram por `make content-sync` (`W2PP_CONTENT_SOURCE=<árvore legada>`), que
confere o SHA-256 de `tmserver/content/content.sha256`. O resíduo do servidor Win32 (DLLs, `.bat`,
dumps, catálogos `.bin`, estado de servidor de teste) foi arquivado em
`tmserver/content/_legacy/` — fora do git, nunca lido pelo servidor, nunca copiado para a imagem.

**Catálogos não moram mais aqui** (issue 58, fase 5): `item_catalog`, `skill_catalog`,
`npc_template`, `world_item_seed` e `npc_definition` vêm do Postgres, pelo gRPC do dbserver, e a
ausência deles recusa o boot. O `importcontent` continua existindo só para reprocessar uma árvore de
conteúdo externa (backup).

```
tmserver/content/Common/            serv00.htm,
                                    Settings/{CompRate,SancRate,QuestsRate,CastleQuest,
                                    MobMerc,RefineRate}.txt
tmserver/content/TMsrv/run/         Regions.txt, Guard.txt, LevelItem.txt, gameconfig.txt,
                                    Language.txt, ItemEffect.h,
                                    HeightMap.dat (16 MiB, 4096² — fora do git),
                                    AttributeMap.dat (1 MiB, 1024² — fora do git)
tmserver/content/DBsrv/run/BaseMob/ {TK,FM,BM,HT} — template por classe (816 B) com equip inicial
tmserver/content/TMsrv/run/BaseSummon/ templates de invocação do BM
tmserver/content/_legacy/           arquivo morto do servidor Win32 (não lido)
```

Os mapas alimentam `route.Bake` (atributo → altura 127 = bloqueado) e o pathfinding dos mobs.
Sem eles o servidor sobe, mas os mobs andam "cego" (passo Chebyshev) e o multiplicador de EXP do
Campo de Treino fica desligado.

## 5. Verificação (escada mínima, do mais barato ao mais caro)

```bash
# 1. compila e testa (host precisa de Go 1.26; senão use o container abaixo)
go build ./... && go test -race ./tmserver/internal/...

# 2. mesma coisa sem Go no host
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine \
  sh -c "go build ./... && go test ./tmserver/internal/protocol/ ./tmserver/internal/world/"

# 3. login headless (não precisa do cliente Windows) — SEMPRE antes de culpar código de jogo
docker run --rm --network host -v "$PWD":/src -w /src golang:1.26-alpine \
  go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v
#   env: W2PP_E2E_ADDR=127.0.0.1:8281 W2PP_E2E_ACCOUNT=test W2PP_E2E_PASSWORD=test123
#        W2PP_E2E_VERSION=12000   (sem isso o servidor responde 0x102 = version mismatch)

# 4. portal (o que o CI roda para ele)
make portal-test        # tsc --noEmit + vitest
make portal-build       # next build (valida rotas e o standalone)
```

**Windows + Git Bash (verificado nesta máquina):** `-w /src` é reescrito pelo MSYS para o diretório
de instalação do Git e o docker aborta com `working directory 'C:/…/resources/git/src' is invalid`.
Prefixe `MSYS_NO_PATHCONV=1` e passe o projeto em caminho Windows. No Docker Desktop não existe
`--network host`: use a rede do compose e o nome do serviço.

```bash
MSYS_NO_PATHCONV=1 docker run --rm --network openwyd_default -v "D:/dolorestec/wyd2/OpenWYD:/src" -e W2PP_E2E_ADDR=tmserver:8281 -e W2PP_E2E_ACCOUNT=test -e W2PP_E2E_PASSWORD=test123 -e W2PP_E2E_VERSION=12000 golang:1.26-alpine sh -c "cd /src && go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v"
```

Não reinicie serviços no meio de um teste do usuário: o restart dispara save-on-shutdown e
reconexão gRPC, gerando falhas transitórias que parecem bug. Faça o deploy, **espere** o teste
humano terminar, e só então mexa.

## 6. Onde está o conhecimento

| Preciso de… | Leia |
|-------------|------|
| Topologia, portas, configs, hardcodes | `docs/migration/config-ops.md` |
| Protocolo CPSock, Types, structs | `docs/migration/protocol-spec.md` |
| Formatos de save, alinhamento | `docs/migration/data-formats.md` |
| Fórmulas (dano, exp, drop, refino) | `docs/migration/game-rules.md` |
| Contrato de um `_MSG_*` | `docs/migration/handlers/` (58/58) |
| Fluxos ponta-a-ponta | `docs/migration/flows.md` |
| Estado real do que funciona | `docs/migration/SESSION-PRIMER.md` |
| Comandos in-game e de GM | `docs/game.md` |
| Histórico de bugs in-game | `docs/migration/ingame-bugs.md` |
| Portal: BFF, telas, overlays, sessão | `portal/README.md` + `.agents/skills/wyd-web-admin-ui` |
| Banco: auditoria, tipos, índices, squash | `docs/analysis/banco-de-dados.md` + skill `wyd-database` |
| Plano de execução deste projeto | `roadmap/index.md` + `roadmap/PLANO-2026-11.md` |
| Análise de conjunto (estado, legado, cliente, banco, processo) | `docs/analysis/` |

Antes de mexer em wire/format/gameplay, leia o doc correspondente. Ele cita evidência como
`Source/Code/.../arquivo.cpp:linha` e marca como **UNVERIFIED** o que não foi confirmado — respeite
essa marcação em vez de tratá-la como fato.

## 7. Skills disponíveis (`.agents/skills/`)

| Skill | Use quando |
|-------|-----------|
| `wyd-server-ops` | subir/parar/inspecionar a stack local, portas, logs, banco, backup, migrations |
| `wyd-database` | auditar/migrar/backup do Postgres, escrever migration, squash de migrations, seeds de referência |
| `wyd-client-connect` | gerar `serverlist.bin`, alinhar versão de protocolo, diagnosticar cliente que não loga |
| `wyd-game-admin` | GM commands, contas, papéis, bans, dar item/gold/nível, auditoria |
| `wyd-economy` | preços, drops, refino, combines, curva de EXP, balanceamento e ferramentas de dados |
| `wyd-server-dev` | escrever/alterar código Go do servidor (handlers, protocolo, persistência) |
| `wyd-testing-bots` | testes e2e headless, harness de protocolo, bots e carga |
| `wyd-web-admin-ui` | construir ou alterar o portal (telas, BFF gRPC, sessão/papéis, overlays) |
| `frontend-design`, `ui-ux-practices` | direção visual e revisão de UI/UX do portal |
| ~~`openspec-*`~~ | arquivado em `docs/archive/openspec/`; o plano vive no `roadmap/` |

## 8. Convenções de trabalho

- **Ao migrar uma funcionalidade, remova o caminho legado na mesma entrega** (arquivo, flag, coluna,
  tela, tabela). Migração sem remoção deixa duas fontes vivas e o defeito nasce da convivência: foi
  assim que 20 linhas do seed curado (0006) ficaram sombreadas pelos blocos importados e duplicaram
  os mercadores de Armia (B18, 04/11). Se a migração ainda não terminou, a remoção fica registrada
  como issue de fechamento (ex.: 57), não como pendência informal.
- Siga `development-guidelines/Go-development-guidelines.md` para Go.
- **Definition of Done única:** `development-guidelines/processo.md` (as seis linhas de evidência, a
  regra de issue, a regra de comentário e a DoD por tipo de mudança). Em divergência com este arquivo,
  vale o `processo.md`.
- Prefira editar o que existe a criar arquivo novo; o roadmap já diz onde cada coisa mora.
- Docs, UI e mensagens em português; commits pequenos e descritivos em português.
- Nunca versione segredos, `serverlist.bin` gerado com IP privado, dumps de conta, binários do
  cliente (`client/`) ou o `project-id` do Freebuff (`.freebuff/`).
- Ao concluir uma issue do roadmap, atualize o status em `roadmap/index.md` e a linha
  correspondente em `docs/migration/SESSION-PRIMER.md` quando o comportamento mudar.
