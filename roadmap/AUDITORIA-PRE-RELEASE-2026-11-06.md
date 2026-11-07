# Auditoria pré-release — 06/11/2026 (FASE 1)

Auditoria do estado real do projeto exigida pela missão `pre-release.md` (FASE 1) e base para
`pre-release-bots.md`. Cada número abaixo foi **medido neste checkout** com o comando ao lado; o que
não foi medido está marcado como tal.

Documentos irmãos:

- [`PRE-RELEASE-PROGRAM.md`](PRE-RELEASE-PROGRAM.md) — as 15 fases e a meta de bots mapeadas em issues.
- [`index.md`](index.md) — fila viva (fonte da verdade do progresso).
- [`REVIEW-12`](issues/REVIEW-12.md) … [`REVIEW-19`](issues/REVIEW-19.md) — revisão contínua por domínio.

## 1. Diagnóstico do roadmap

O roadmap tem três camadas que **não estavam reconciliadas**:

| Camada | Arquivo | Estado encontrado |
|---|---|---|
| Fila de trabalho | `roadmap/issues/` | 69 herdadas + 4 BOT abertas + 8 REVIEW + 2 MATH = **83** (issue-08 e issue-09 saíram para `test/` no fechamento da onda 1) |
| Fechadas com evidência | `roadmap/done/` | 30 herdadas + BOT-01/BOT-02 = **32** |
| Esperando só teste humano | `roadmap/test/` | **12** (issue-08 e issue-09 entraram em 06/11, com evidência datada) |

O problema não é falta de issue: é que **o estado publicado envelheceu**. Medições desta auditoria:

| Afirmação publicada | Medição de hoje | Onde está |
|---|---|---|
| "Toolchain Go no host ❌ ausente" | `go version` → `go1.27.3 windows/amd64` | `roadmap/index.md` §Estado atual |
| "Cliente apontando para o servidor local ⏳ pendente" | issue 05/06/07 fechadas em 04/11 com login, movimento e compra no cliente | `roadmap/index.md` §Estado atual |
| "hoje o roadmap tem 17 issues ⏳ e 13 🟡" | fila mudou: 18 issues foram fechadas depois daquele texto | `PLANO-2026-11.md` |
| `TEST-01` fechada: "42 `t.Skip`" | `grep -rIn 't\.Skip' --include='*_test.go'` → **43** | `done/TEST-01.md` |
| Issues 08 e 09 "⏳" | QA de 05/11 criou 4 personagens (HT/BM/FM/TK) com BaseMob, equip inicial e 1M de gold | `roadmap/index.md` §Fase 2 |
| "Sem e2e de navegador no portal: 0 Playwright" | confirmado: `portal/package.json` não tem Playwright | REVIEW-03/TEST-03 |

**Conclusão:** o roadmap está correto no método (issue com evidência, `done/` × `test/`) e
desatualizado na fotografia. Nenhuma issue foi encontrada fechada sem evidência; as divergências são de
**estado publicado**, não de trabalho inventado.

## 2. Inventário de issues

Por domínio (contagem de arquivos em `roadmap/issues/` após esta auditoria):

| Domínio | Issues abertas | Observação |
|---|---:|---|
| Jogo/servidor (10–13, 49, QA-01..07, MENSAGENS-01, CANAL-01/03) | 19 | QA in-game é o maior bloco; 3 dependem de cliente Windows |
| Banco e conteúdo (33, 35, 37, 50, 59, 60, REFACTOR-01/03) | 8 | 50/59/60 são a mesma frente em três fatias |
| CI/CD, DevOps, observabilidade (CI-03, TEST-03, issue-55, REVIEW-04/09) | 5 | CI-03 bloqueado por mapas fora do git |
| Portal (39, 53, 54, REVIEW-08, PORTAL/UI) | 4 | e2e de navegador inexistente |
| Cliente/launcher (51, 52) | 2 | offsets e launcher |
| Laboratório de bots (BOT-03..08) | 6 | novo nesta rodada; BOT-01/02 já fechadas |
| Matemática (MATH-01/02) | 2 | novo nesta rodada; fase 10 não tinha dono |
| Revisão contínua (REVIEW-01..04, 08..10, 12..19) | 15 | guarda-chuvas, não trabalho paralelo |

`roadmap/test/` (10): CANAL-02, 24, 25, 34, 36, 43, 44, 45, 46, 48 — todas com código pronto e
dependência exclusiva de evidência no cliente real.

## 3. Duplicidades e inconsistências

Nenhuma issue duplicada de trabalho foi encontrada; o que existe é **sobreposição de guarda-chuva**,
que precisa ser lida na hierarquia certa:

| Par | Relação | Ação |
|---|---|---|
| REVIEW-01 × issue-49 | ambos "fechar a fila 🟡"; REVIEW-01 é o método, 49 é o lote | REVIEW-01 aponta para 49 |
| REVIEW-03 × TEST-03 | REVIEW-03 cobre testes; TEST-03 é o Playwright | TEST-03 é filha; não abrir outra |
| REVIEW-04 × CI-01/CI-03/issue-55 | REVIEW-04 audita CI; CI-01 fechada, CI-03 e 55 executam | manter REVIEW-04 como registro |
| REVIEW-09 × OBS-01..04 | OBS-01..04 fechadas; REVIEW-09 vira registro permanente | não reabrir OBS |
| issue-50 × 59 × 62 | "conteúdo no banco" em três fatias (CSV, configs, catálogos do webserver) | 62 fechada; 50 e 59 vivas |
| issue-54 × DOC-01 | hub de documentação × site mkdocs entregue | 54 reduzida a ADRs/runbooks restantes |
| issue-52 × REVIEW-07 | offsets do cliente × auditoria do legado C++ | REVIEW-07 fechada; 52 executa |

Inconsistências de processo encontradas:

1. **`TODO` solto** em `tmserver/internal/handler/autotrade.go:115` (`TODO(#115)`), contra
   `development-guidelines/processo.md` §8. Registrado em REVIEW-12.
2. **Contagem de skip** divergente entre `TEST-01` e a árvore (42 × 43).
3. **`pre-release*.md` na raiz** não é roadmap: são documentos de missão. Passam a ter mapa em
   `PRE-RELEASE-PROGRAM.md` para não virarem "pendência informal".
4. **Duas fotos de estado** (índice + PLANO) com datas diferentes e números que já não batem.

## 4. Issues concluídas que precisam de regularização

| Issue | Situação | Regularização | Ação desta rodada |
|---|---|---|---|
| issue-08 (criação de personagem) | ✅ código; criação provada em 05/11 (4 classes, equip, 1M gold) | mover para `test/` com a evidência datada e o critério humano que falta | ✅ regularizada em 06/11 (`roadmap/test/issue-08.md`) |
| issue-09 (persistência) | inventário/gold/última cidade documentados no `SESSION-PRIMER`; falta evidência SQL + restart + cargo/affects | mover para `test/` com o que existe e o que falta | ✅ regularizada em 06/11 (`roadmap/test/issue-09.md`) |
| CANAL-02 | em `test/` com código pronto | falta só evidência no cliente (ESC → Servidor) | ligado a CANAL-03 |
| issue-32 | fechada, mas "falta a recusa no login do cliente" | a recusa foi vista no cliente em 04/11 | corrigir o texto da issue |
| TEST-01 | fechada com 42 skips; recontagem de 06/11 fecha em **47** (36 `conteúdo`, 9 `ambiente`, 2 `feature`) — 43 na onda 1, +1 do teste de paridade de conteúdo e +3 dos testes de paridade do worldatlas | corrigida na issue com data e comando de prova | ✅ registrado em `done/TEST-01.md` e no índice |
| OBS-02/03 | fechadas após o review do lote | contador de review em 2 de 10, abaixo do marco | sem review extra, conforme regra |

## 5. Lacunas identificadas (medidas)

| Lacuna | Medição | Dono |
|---|---|---|
| Fórmulas sem nível de confiança nem catálogo | não existe `docs/sistemas/matematica.md`; 12 páginas de sistema, nenhuma de matemática | MATH-01 |
| Hits em velocidade alta não investigados | nenhum teste/simulador de velocidade no repo | MATH-02 |
| Núcleo de bots | **inexistente até hoje**: nenhum cliente headless reutilizável (só testes e2e com cliente embutido no teste) | BOT-02 ✅ |
| Carga/soak/orquestração | nenhum container ou controlador de bots | BOT-05 |
| Economia simulada | nenhum indicador de gold/Gini/inflação | BOT-06 |
| e2e de navegador no portal | 0 Playwright; 15 arquivos de teste unitário | TEST-03 |
| Cobertura de `t.Skip` | 47 skips, todos com motivo (política de TEST-01) | REVIEW-12 |
| `TODO` no código | 1 (autotrade EF_NOTRADE) | REVIEW-12 |
| Dashboard consolidado de projeto | dashboards existem para infra (Grafana), não para roadmap/CI/cobertura | REVIEW-14 |
| Boot completo no CI | mapas 17 MiB fora do git | CI-03 |
| Segurança formal | sem revisão de domínio registrada até hoje | REVIEW-15 |

## 6. Proposta de reorganização

Manter o método atual (uma issue = um resultado verificável, `done/` com evidência, `test/` para o que
só falta humano) e mudar apenas a **camada de navegação**:

1. **Fila por domínio**, não por número: jogo, banco/conteúdo, plataforma (CI/obs/segurança), portal,
   cliente, bots, matemática. O índice ganha essa visão no topo ("por onde começar").
2. **Guarda-chuva separado da execução:** `REVIEW-*` registram; as filhas executam. Nenhuma REVIEW
   fecha enquanto o domínio precisar de acompanhamento.
3. **Documento de missão com mapa:** `PRE-RELEASE-PROGRAM.md` liga cada fase dos dois `.md` de missão
   a issues concretas, para não existir trabalho sem dono.
4. **Fotografia com data e comando:** todo número publicado no índice vem com data e comando de
   medição, para a próxima auditoria conferir em um minuto.

## 7. Nova estrutura de prioridades

| Prioridade | Critério | Fila |
|---|---|---|
| **P0 (agora)** | destrava release e custa pouco | CI-03 (boot no CI), CANAL-03 (ESC → servidor), regularizar issues 08/09, corrigir fatos envelhecidos do índice |
| **P1** | risco estrutural e segurança | REVIEW-15 varredura, `check-secrets` ampliado, issue-50/59 (conteúdo no banco), issue-55 (CI completo) |
| **P2** | produto e capacidade | TEST-03 (Playwright), issue-53 (feedback), BOT-03 (ações reais) → BOT-05 (carga), MATH-01 |
| **P3** | evolução | BOT-04 (navegação), BOT-06 (economia), BOT-07 (social/LLM), MATH-02 (velocidade), issue-51/52 (launcher/cliente), dashboards |

Regra herdada que continua valendo: **nada de frente nova antes de a fila de evidência andar** — hoje
10 issues (‑`test/`) esperam só o cliente real.

## 8. Plano de execução (ondas)

| Onda | Conteúdo | Saída verificável |
|---|---|---|
| 0 (esta) | auditoria + reorganização + núcleo de bots | este documento, índice atualizado, `go test ./tmserver/internal/bot/...` verde |
| 1 | P0: fatos corrigidos, issues 08/09 regularizadas, CI cobrindo bots | índice sem afirmação que a medição contradiz (✅ 06/11). Estado do CI medido no `Makefile`: `ci: gate check-skips check-comments guias-check vet build test portal-test`, e `test` é `go test -race -cover ./...` — que **já inclui** os pacotes de bots; `bot-test` é a rodada focada. Levar o alvo dedicado e o dashboard ao pipeline segue como BOT-08, sem job duplicado. `make ci` **não foi executado**: este host não tem `make`, então rodou-se `go` + os scripts (`gate`, `check-*`) direto, com os resultados da §11 |
| 2 | P1: segurança e conteúdo no banco | ✅ 06/11: `check-secrets` ampliado (5 checagens, vermelho/verde provados), ferramentas de segurança medidas, issue 59 entregue até a fronteira do banco (§12); escrita no Postgres vivo em `test/issue-59.md` |
| 3 | P2: bots com ações reais + Playwright | bot mata mob com invariante lida do servidor; 5 fluxos e2e no portal |
| 4 | P3: capacidade, economia, matemática | relatório de carga, indicadores econômicos, catálogo de fórmulas |

Revisão técnica consolidada **a cada 10 issues fechadas** (regra da missão e do `REVIEW-11`); o
contador está em 2 de 10.

## 9. Issues de revisão contínua

Criadas nesta rodada, uma por domínio, com histórico datado e critério por rodada:

| Issue | Domínio | Primeiro achado registrado |
|---|---|---|
| [REVIEW-12](issues/REVIEW-12.md) | arquitetura e código Go | `TODO(#115)`; 47 skips |
| [REVIEW-13](issues/REVIEW-13.md) | banco e persistência | 23 migrações; conteúdo restante em 59/60 |
| [REVIEW-14](issues/REVIEW-14.md) | CI/CD e observabilidade | boot completo fora do CI; bots fora do `make ci` |
| [REVIEW-15](issues/REVIEW-15.md) | segurança | rotas fora do middleware (corrigido) |
| [REVIEW-16](issues/REVIEW-16.md) | portal | 0 Playwright; portal do jogador sem dono único |
| [REVIEW-17](issues/REVIEW-17.md) | cliente e engenharia reversa | ESC → Servidor (CANAL-03); offsets (52) |
| [REVIEW-18](issues/REVIEW-18.md) | matemática e balanceamento | nenhum catálogo com nível de confiança |
| [REVIEW-19](issues/REVIEW-19.md) | IA, skills e documentação | índice afirmando toolchain ausente |

## 10. Atualização das Skills

| Skill | Mudança | Motivo (medido) |
|---|---|---|
| `wyd-testing-bots` | aponta para `tmserver/internal/bot` e `tmserver/cmd/botlab`, com os alvos `bot-test`/`botlab`; registro do limite (`sim_*`) | antes descrevia "bot próprio" sem citar o pacote, que passou a existir |
| `wyd-server-dev` | sem mudança nesta rodada | — |
| `wyd-server-ops` | sem mudança nesta rodada | — |
| `wyd-database` | pendente: incorporar a regra de "migração = remoção do legado" no checklist | REVIEW-13 |
| `wyd-economy` | pendente: apontar para BOT-06 quando houver indicadores | BOT-06 |

Skills que **não** serão criadas: `release-engineering` e `observability` separadas. O conteúdo já
cabe em `processo.md` + REVIEW-14; skill nova sem conteúdo próprio é duplicação.

## 11. Verificação executada (onda 1, 06/11/2026)

Módulo inteiro medido neste host (`go1.27.3 windows/amd64`), fatiado para que nenhum comando chegasse
perto do teto de execução. Tempos de relógio de parede, com `-count=1` (nada veio de cache de teste).

| Comando | Escopo | Resultado | Tempo |
|---|---|---|---|
| `go build ./...` | 85 pacotes | exit 0 | 37 s |
| `go vet ./...` | 85 pacotes | exit 0 | 10 s |
| `go test -race ./internal/...` | 19 pacotes compartilhados | ok | 138 s |
| `go test -race ./tmserver/internal/...` (A) | 19 pacotes (sem `handler`/`world`) | ok | 151 s |
| `go test -race ./tmserver/internal/world/` | `world` (teste mais longo 75,9 s) | ok | 191 s |
| `go test -race ./webserver/internal/...` (A) | 12 pacotes | ok | 147 s |
| `go test -race ./webserver/internal/...` (B) | 14 pacotes restantes | ok | 185 s |
| `go test -race ./api/... ./binserver/... ./dbserver/... ./client-tools/... ./logpipe/... ./tmserver/cmd/... ./webserver/cmd/...` | 7 pacotes de comando | ok | 210 s |
| `go test -race ./tmserver/internal/handler/` (8 blocos de ~104 testes) | 826 testes | ok (após correção, ver abaixo) | 35 + 20 + 47 + 48 + 104 + 128 + 79 + 53 s |

O pacote `handler` é o único que não cabia num comando: 101 arquivos `_test.go`, 826 funções. A fatia
por nome de teste (`-run '^(TestA|TestB|…)$'`) rodou cada bloco com `-race`.

**Falha real encontrada e corrigida:** `TestTransformExpiryRevertsMesh` falhou (`transform_test.go:399`,
"missing transform UpdateEquip"). Não era bug de produto: o sweep de affects toca o jogador só na fase
`s.Conn%8`, e com `Time: 1` o revert podia disparar **entre** os dois logins — o pacote ia para quem já
estava em visão (o dono), e o observador legitimamente não recebia nada. Correção na mesma data: o
harness ganhou `startServerAffectTickGated`, que segura o tick até os dois clientes entrarem em visão,
tornando o cenário determinístico sem afrouxar a asserção. Prova: teste isolado 5× com `-race` (ok,
1,6 s) e bloco completo reexecutado (ok, 44,5 s).

**Skips:** 43 no módulo, todos com `motivo:` declarado (`check-skips` verde); 32 de conteúdo, 9 de
ambiente, 2 de feature — ver [TEST-01](done/TEST-01.md). A onda 2 fechou com **47** (36 `conteúdo`, 9 `ambiente`, 2 `feature`): o skip da paridade de conteúdo
e os três do worldatlas (dois de conteúdo e um de banco ausente, todos com `motivo:`).

**Governança:** `check-comments` 3 diretórios limpos · `check-docs-links` 141 links, 0 quebrados ·
`check-skips` 47/47 com motivo.

## 12. Onda 2 — plataforma verificada (06/11/2026)

| Comando | Escopo | Resultado | Tempo |
|---|---|---|---|
| `go build ./...` | 85 pacotes | exit 0 | 34 s |
| `go vet ./...` | 85 pacotes | exit 0 | 15 s |
| `go test -race -count=1 ./internal/...` | 19 pacotes | ok | 59 s |
| `go test -race -count=1 ./tmserver/internal/content/ ./tmserver/cmd/importcontent/` | conteúdo + importador | ok | 26 s |
| `bash scripts/check-secrets.sh` | árvore + `.gitleaks.toml` | verde; vermelho provado com `.pem` versionado (RC 1) e token `ghp_` plantado (RC 1) | ~15 s/rodada |
| `gitleaks dir . --config .gitleaks.toml` | 42,27 MB | `no leaks found` (2 arquivos > 5 MB ignorados, ver REVIEW-15) | 15–30 s |
| `govulncheck ./...` | módulo inteiro | 0 afetando o código; 3 em módulos exigidos, não chamados | 33 s (instalação 121 s) |
| `cd portal && npm audit --omit=dev` | portal | `found 0 vulnerabilities` | 20 s |
| `npx vitest run src/middleware.test.ts` | guarda de rotas | 11 testes passando | 11 s |
| `go run ./tmserver/cmd/importcontent -content tmserver/content` | dry-run do importador | `guards=5 level_items=318 mob_mercs=1 settings=26 drop_matrix=64 regions=52` | 4 s |

Achados da REVIEW-15 nesta rodada (tabela completa na issue): 6 achados, **5 corrigidos** e **1 aceito**
com justificativa registrada — gitleaks não escaneia arquivo acima de 5 MB (`seed_npc_definition.up.sql`
de 7 MB e `HeightMap.dat` de 16 MB), limite conhecido e igual ao do CI. As correções: `check-secrets`
cobrindo chave versionada, allowlist cega e tokens GitLab/Stripe/npm/PAT; default do `importcontent`
limitado ao que ainda é arquivo (os CSVs da 58 saíram da árvore); chave do `level_item` virou o ordinal
do arquivo (318 linhas, 159 pares, blocos repetidos); matriz de drop medida em 4×16; caminho real do
`MobMerc.txt` em `Common/Settings/`.

O que não se prova neste host: a escrita no Postgres (sem container) — roteiro com contagens esperadas,
idempotência e prova de drift em [`test/issue-59.md`](test/issue-59.md).

## Limitações desta auditoria

- A suíte Go foi executada por inteiro neste host, mas **nenhum teste de jogo foi executado**: cliente
  gráfico, banco vivo e stack em containers ficaram fora por decisão do usuário. O que depende de
  runtime (login real, persistência, comércio, quests) continua pendência humana em `test/`, com
  comando e critério escritos em cada issue.
- A varredura de `TODO`/`FIXME` cobre `*.go`, `*.ts`, `*.tsx`, `*.sql` e `*.sh`; markdown não foi
  varrido por marcadores.
- Contagem de issues muda a cada criação; os números do §2 valem para o commit desta auditoria.

## 13. Onda 3 — gate de lint fechado (06/11/2026)

O `make lint` era o único alvo sem prova: `golangci-lint` não existia no host. Instalado
(`go install …/golangci-lint/v2@latest`, v2.15.0), a primeira execução deu **55 achados** e o gate
ficou vermelho. Todos foram corrigidos, não silenciados:

| Achado | Quantos | Correção |
|---|---:|---|
| `misspell` acusando português correto | 37 | 24 palavras entraram em `ignore-rules` do `.golangci.yml`; nenhuma regra do linter foi desligada |
| `gofmt` (CRLF no worktree) | 3 | `gofmt -w` nos arquivos pendentes; `.gitattributes` já fixa `eol=lf`, então o conteúdo do commit não muda |
| `revive` | 9 | `print`→`printPath`, `max`→`maxVal`, `BotSpec`→`Spec` (stutter), `else` após `return` removido, `ctx` como 1º parâmetro, parâmetros não usados viraram `_` |
| `unused` | 4 | `guildOwnerTargetJSON` (função morta), `expPanelDefaultColor`, e dois campos de fake em testes |
| `staticcheck` | 1 | `grpc.WithInsecure` → `credentials/insecure` |
| `unconvert` | 1 | conversão `int32` desnecessária removida |

Resultado da onda: `golangci-lint run ./...` → **`0 issues.`** (RC 0). A varredura de
`TODO`/`FIXME`/`HACK`/`XXX` fechou em **0** achados: o único TODO restante
(`handler/autotrade.go`, EF_NOTRADE) virou código — a loja agora recusa item com o efeito 127,
com teste `TestAutoTradeOpenRejectsNoTrade` provando a recusa.

| Comando | Resultado | RC |
|---|---|---:|
| `golangci-lint run ./...` | 0 issues | 0 |
| `go build ./...` | ok | 0 |
| `go vet ./...` | ok | 0 |
| `cd portal && npx tsc --noEmit` | sem erro | 0 |
| `cd portal && npx vitest run` | 16 arquivos, 118 testes | 0 |
| `cd portal && npm run build` | sem aviso/deprecação (0 linhas `warn`/`deprecat`) | 0 |
| `bash scripts/check-comments.sh` | 15 diretórios limpos | 0 |
| `bash scripts/check-skips.sh` | 47 skips, todos com motivo | 0 |
| `bash scripts/check-docs-links.sh` | 141 links, 0 quebrados | 0 |
