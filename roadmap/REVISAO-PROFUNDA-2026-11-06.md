# Revisão profunda do roadmap — 06/11/2026

Segunda passada sobre a série `REVIEW-01..10`, com **medição no checkout**, correção das afirmações
que envelheceram e uma nova ordem de execução orientada a fechar barato primeiro.

Substitui a leitura de 05/11 como referência de prioridade; `index.md` continua sendo a tabela de
estado e cada `issues/REVIEW-*.md` continua sendo a especificação da frente.

## 1. Como esta revisão foi feita

Tudo abaixo saiu de comando executado nesta árvore (não de memória nem de estimativa):

```bash
find . -name '*.go' | wc -l                       # 584
find . -name '*_test.go' | wc -l                  # 270
find . -name '*.go' -print0 | xargs -0 cat | wc -l # 149.305
grep -rn --include=*.go -E '// (The|This|Returns|Sets|Creates|Handles|Loads|Parses|Builds|Reads|Writes|Keeps|Sends|Marks|Applies|Uses) ' . | wc -l   # 396
grep -rn --include=*.go -E 'issue [0-9]|roadmap/|docs/migration' . | grep -v '\.pb\.go' | wc -l   # 109
grep -rn --include=*_test.go 't\.Skip' . | wc -l  # 42
find docs -name '*.md' | wc -l                    # 174
find portal/src/app -name 'page.tsx' | wc -l      # 30
ls .github/workflows                              # ci.yml (1)
ls internal/migrations/*.up.sql | wc -l           # 13
```

## 2. Inventário medido (06/11/2026)

| Dimensão | Número | Leitura |
|---|---:|---|
| Código Go | 584 arquivos / 149.305 linhas | volume alto, concentrado em `tmserver` |
| Testes Go | 270 arquivos (46% dos `.go`) | proporção boa; qualidade é o problema, não a quantidade |
| Comentário em inglês | **396** | a convenção do projeto não é cumprida há meses |
| Citação de issue/caminho de doc dentro de `.go` | **109** (fora `.pb.go`) | histórico acoplado ao código |
| `t.Skip` | **42** | precisa de triagem: conteúdo ausente × feature × morto |
| Docs `.md` | 174 em `docs/` (423 no repo) | alto, sem porta de entrada (sem `mkdocs.yml`) |
| Páginas do portal | 30 `page.tsx`, **0 e2e** | só `vitest`; nenhuma prova em navegador |
| Workflows CI | 1 (`ci.yml`, 5 jobs) | sem Postgres, sem cobertura com piso, sem SBOM |
| Migrations | 13 `*.up.sql` | baseline squasheado; falta job de base vazia |
| Arquivos soltos na raiz | 15 | aceitável, mas `docs/` é o lugar de 3 deles |

## 3. Achados novos (e o que nas issues está desatualizado)

| # | Achado | Evidência | Ação |
|---|---|---|---|
| N1 | **`portal/src/middleware.ts` existe** e valida JWT (`jwtVerify`, HS256) — a PORTAL-01 afirma "0 `middleware.ts`" | `portal/src/middleware.ts` lido nesta revisão | issue corrigida; falta o teste de 403 por rota |
| N2 | **`/precos` e `/atlas` estavam fora do `matcher`**: a rota existe, o link está no menu, e o middleware não cobria | `matcher` listava 16 rotas; `GUARDED` listava 18 | **corrigido agora** (as duas entraram no matcher) |
| N3 | `GUARDED` era **código morto** (`void GUARDED`): documentava uma intenção que o `matcher` não cumpria — o pior tipo de drift | mesmo arquivo | array removido, aviso escrito no docblock |
| N4 | **`.claude/skills` não existe** (só `.claude/commands` e `settings.json`) — a REVIEW-10 fala em "dois espelhos que divergem" | `ls -R .claude` | afirmação corrigida; sobra decidir `commands/opsx` |
| N5 | **`openspec/` já não existe** (arquivado em release anterior) — a REVIEW-10 ainda pede "adotar ou arquivar" | `ls openspec` vazio | critério N5 fechado por fato consumado |
| N6 | `.agents/skills` tem **15** skills, não 22 | `ls .agents/skills` | número corrigido |
| N7 | Não existe `docs/development-guidelines/` — o `AGENTS.md` aponta para `development-guidelines/` na raiz | `ls` do caminho | a DoD nasce em `development-guidelines/processo.md` |

N2/N3 são o padrão de defeito que a revisão existe para achar: **documentação que afirma uma proteção
que o código não aplica**. Um teste de rota (item da PORTAL-01) pega os dois.

## 4. Aprofundamento por review

### REVIEW-02 — código Go, idioma e organização

O que medimos muda a natureza do trabalho: **396** comentários em inglês e **109** citações de issue
não são um "polimento", são uma migração de convenção. Fazer isso "por pacote, com teste verde" é o
único caminho que não mente no comentário.

Decisões propostas (para virar `development-guidelines/Go-style.md`):

1. **Comentário em português, `doc` começando pelo nome do símbolo** — a regra do `golint`/`revive`
   continua valendo, só o idioma muda. Referência: [Effective Go](https://go.dev/doc/effective_go) e
   [Go Style Guide do Google](https://google.github.io/styleguide/go/decisions).
2. **`revive` com `exported` + `godot`** só por diretório já limpo (lista explícita no CI), porque
   ligar repo-wide hoje deixaria o job vermelho para sempre — e gate permanentemente vermelho é gate
   desligado.
3. **Organização de pastas**: `tmserver/internal/handler` tem 130+ arquivos num pacote só. Não é
   errado em Go (pacote = unidade de compilação), mas `handler/` já mistura domínios (combate, item,
   guilda, evento). Proposta: **não** quebrar em subpacotes agora (custo alto, ganho discutível com
   `world` como dono do estado) e sim **agrupar por prefixo de arquivo** (`combine_*`, `guild_*`,
   `rune*`) — o que já acontece parcialmente — e usar `doc.go` por domínio. Referência:
   [Organizing a Go module](https://go.dev/doc/modules/layout) e
   [Go blog: Package names](https://go.dev/blog/package-names).
4. **`cmd/*/main.go`**: `tmserver` tem 845 linhas de wiring. `newApp` por serviço é a menor mudança
   que torna o wiring testável; não é DDD, é separação de `os.Args` do grafo de dependências.
5. **SOLID/DDD/Clean Architecture**: aplicar onde paga — `world` (dono único do estado) já é a
   fronteira; `handler` é camada de aplicação; `dbclient`/`store` são infraestrutura. Introduzir
   camada de domínio adicional aqui criaria indireção sem teste que justifique.

### REVIEW-03 — testes

Medição que muda a conversa: **270 arquivos de teste, 0 e2e de navegador, 42 `t.Skip`**.

- Triagem dos 42 `t.Skip` (TEST-01) é barata e de alto retorno: cada skip é uma promessa não cumprida.
  Classificar em `conteúdo ausente` (legítimo, com caminho), `feature ausente` (vira issue) e `morto`
  (apagar).
- **Playwright**: o guia do Next 15 recomenda testar contra o **build de produção** e usar `webServer`
  para subir/esperar o servidor ([Next.js — Playwright](https://nextjs.org/docs/15/app/guides/testing/playwright),
  [Playwright — web server](https://playwright.dev/docs/test-webserver)). Autenticação por
  `storageState` com um projeto `setup` e arquivos em `playwright/.auth` **não versionados**
  ([Playwright — Auth](https://playwright.dev/docs/auth)); um estado por papel (`admin`, `moderator`,
  `player`) cobre o 403 sem repetir login em cada teste.
- Para o caso de middleware, o Next 15 separa autenticação de autorização e **avisa que middleware não
  deve ser a única proteção**: a checagem tem de existir perto do dado
  ([Next.js — Authentication](https://nextjs.org/docs/15/app/guides/authentication)). Isso confirma o
  desenho do nosso BFF (`adminCall`) e explica por que o teste de 403 precisa exercitar **os dois**
  níveis: rota (middleware) e RPC (web-api).
- Sharding só depois de estabilizar: `--shard=x/y` com `fullyParallel: true`
  ([Playwright — Sharding](https://playwright.dev/docs/test-sharding)).
- **Pirâmide por tipo, não por pasta nova**: unidade em `*_test.go`; integração atrás de
  `//go:build integration` (já existe); e2e no portal. Criar `tests/e2e/` no Go sem um runner que
  suba o stack vira pasta vazia com README — o e2e do Go é o `TestE2ESmokeLogin` com tag `e2e`.

### REVIEW-04 — CI/CD

Reuso e banco são as duas lacunas reais:

- `workflow_call` para `_go.yml`/`_portal.yml` **exige** `on: workflow_call`
  ([GitHub — Reuse workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows));
  cache de Go deve vir do `actions/setup-go` (`cache-dependency-path` quando o `go.sum` não está na
  raiz) — [GitHub — Go](https://docs.github.com/en/actions/tutorials/build-and-test-code/go).
- Postgres em CI: `services.postgres` com `--health-cmd pg_isready` e, só depois de pronto, aplicar as
  migrations ([GitHub — PostgreSQL service](https://docs.github.com/en/actions/tutorials/use-containerized-services/create-postgresql-service-containers)).
  O job precisa **falhar** se a contagem por tabela divergir do baseline — é o mesmo gate do
  `scripts/db-baseline.sh`, em modo não destrutivo.
- SBOM por Syft (`anchore/sbom-action`, SPDX) e vulnerabilidade **de código Go** por `govulncheck`
  ([Syft](https://github.com/anchore/syft), [anchore/sbom-action](https://github.com/anchore/sbom-action),
  [Go — Vulnerability Management](https://go.dev/doc/security/vuln/)). São complementares: SBOM olha
  o que entrou na imagem, `govulncheck` olha o que o nosso código chama.
- `make ci` deve ser **o mesmo comando** do job (já existe `make gate`; falta o alvo guarda-chuva).

### REVIEW-05 — banco

A parte de roles/DSN continua válida; o que mudou é o escopo. O bloco do usuário acrescenta uma
frente de **qualidade de dados** (skills da FM com item no lugar de skill, erros de `relation/column
does not exist` no log) que não é a mesma coisa que roles e índices — precisa de issue própria:

- **Incoerência catálogo × classe** (Magia Branca/Negra/Especial da FM) é bug de **seed**, não de
  schema. O caminho é um verificador: para cada `npc_skill`/`npc_shop_item`, validar que o
  `item_index` está na faixa de skill esperada e que a classe do NPC bate com a família da skill.
- Erros `relation "instance_flag" does not exist` / `column "version" does not exist` no log do
  Postgres são **consultas do portal** apontando para schema antigo: sinal de contrato solto, não de
  migration faltando. Cada um vira teste de contrato contra o schema atual.
- Normalização: nomes repetidos/despadronizados em `item_catalog` pedem um relatório (dado, não
  opinião) antes de qualquer remodelagem; `security_invoker` em views quando elas existirem.

### REVIEW-06 — documentação

423 `.md` e nenhuma porta de entrada. O movimento certeiro é **um site com navegação derivada do
`AGENTS.md` §6**, que já é o melhor mapa do projeto. `mkdocs build --strict` é o gate (link quebrado
reprova). Página por sistema de jogo é o maior buraco: hoje o conhecimento está em
`docs/migration/handlers/` (58/58) mas não há uma página "como funciona a guilda" para quem joga ou
administra.

Estado em 06/11: `openspec` já foi arquivado e `docs/archive/` existe — o trabalho de organização já
começou; falta a camada publicada.

### REVIEW-07 — legado C++

A classificação já está feita em boa parte (`Source/Code/_legacy/` com README de substituição). O que
falta é a **decisão registrada** (manter `Source/` como referência de wire × mover para repositório
separado) e o mapa hook-a-hook do `ClientPatch_v7662`, que é dependência do cliente e não do servidor.

### REVIEW-08 — portal

Separação `(public)` × `(admin)` está iniciada; o que a medição mostra é que **a proteção de rota
existe mas estava furada em duas rotas** (N2/N3). Antes de qualquer tela pública nova, o teste de
autorização por rota precisa estar no CI — abrir superfície pública sem esse teste é como dirigir sem
freio.

Referências para o desenho de autorização: o guia do Next 15 (401 × 403, checagem perto do dado) e a
[OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
(negar por padrão, verificar no servidor, nunca confiar em UI).

### REVIEW-09 — observabilidade

Três decisões técnicas, todas com fonte:

1. `/metrics` com `promhttp.Handler()` ([Prometheus — Go](https://prometheus.io/docs/guides/go-application/)).
2. gRPC instrumentado por interceptors (`go-grpc-middleware/providers/prometheus`) — o
   `go-grpc-prometheus` antigo está depreciado
   ([grpc-middleware](https://github.com/grpc-ecosystem/go-grpc-middleware)).
3. **`trace_id` não é label de métrica**: cardinalidade explodida; para ligar métrica a trace use
   *exemplars* ([Prometheus — naming](https://prometheus.io/docs/practices/naming/)). O `trace_id`
   vai no **log** e no contexto OpenTelemetry (`otelgrpc.NewServerHandler()`,
   `trace.SpanContextFromContext`)
   ([OTel Go](https://opentelemetry.io/docs/languages/go/instrumentation)).
4. Segredos e sessão: OWASP ASVS V7 (sessão) e V13.3 (segredos)
   ([Secrets Management](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html),
   [Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)).
   Nosso `SESSION_SECRET` obrigatório fora de dev já é metade; falta o gate que reprova o default.

### REVIEW-10 — limpeza e IA

Corrigido por medição: não há dois espelhos de skills (N4) e `openspec` já saiu (N5). O que resta é
`CLEAN-05` (DoD única — **feito nesta rodada**: `development-guidelines/processo.md`) e a auditoria de
`scripts/` (10 arquivos; `check-build.sh`/`check-eol.sh` já são usados por gates).

## 5. Nova ordem de execução

Critério: **primeiro o que fecha com teste local em menos de um dia**, depois o que reduz risco
estrutural, depois produto. Issue grande só entra quando a fila barata estiver vazia.

### Fila A — fechar barato (S, teste local, sem humano)

> **Estado no fim de 06/11:** A1 ✅ (DoD), A2 ✅ (teste de rota, com 4 rotas abertas corrigidas), A3 ✅
> (42 skips classificados + gate), A4 ✅ (`make ci` + workflows reutilizáveis), A5 🟡 (fatia de 3
> diretórios + gate; dívida medida na issue), A6 🟡 (site `--strict` verde + 6 ADRs; falta página por
> sistema e `doc_legacy/`). Detalhe por item nas próprias issues.

| Ordem | Issue | Por quê agora | Aceite |
|---:|---|---|---|
| A1 | `CLEAN-05` | DoD única destrava o critério de "pronto" de todas as outras | ✅ `development-guidelines/processo.md` + `AGENTS.md` §8 apontando |
| A2 | `PORTAL-01` (parte 1) | middleware existia furado em 2 rotas; correção feita, falta o teste | teste de rota sem cookie → 307, com papel errado → 403 |
| A3 | `TEST-01` | 42 `t.Skip` são 42 promessas; triagem é leitura + decisão | cada skip com dono: conteúdo, issue ou remoção |
| A4 | `CI-01` | workflows reutilizáveis + path filters | `ci.yml` só orquestra; `_go.yml`/`_portal.yml` |
| A5 | `REFACTOR-03` (fatia) | 396 comentários: começar por `internal/*` (mais consumido) | `grep` zerado no diretório escolhido |
| A6 | `DOC-01` | `mkdocs build --strict` verde num site mínimo viável | site sobe no compose, nav do `AGENTS.md` §6 |
| A7 | `NORMALIZE-01/02` | já fechadas; só registrar evidência no índice | ✅ |
| A8 | `BUILD-01` | já fechada | ✅ |

### Fila B — risco estrutural (M)

| Ordem | Issue | Aceite |
|---:|---|---|
| B1 | `CI-02` + `DB-01` | Postgres de serviço aplicando migrations em base vazia; DSN sem superusuário |
| B2 | `OBS-01` | `/metrics` nos 4 serviços, sem bloquear o laço do mundo |
| B3 | `PORTAL-01` (parte 2) | testes de autorização por rota no CI |
| B4 | `REFACTOR-01` | `newApp` por serviço, `main` só flags |
| B5 | `TEST-03` | 5 fluxos Playwright com `storageState` por papel |
| B6 | `CLEAN-01/04` | `.claude/commands` decidido; `scripts/` sem órfão |

### Fila C — depende de humano/cliente

| Ordem | Issue | Depende de |
|---:|---|---|
| C1 | `issue-49` (QA in-game) | cliente Windows + sessão dedicada |
| C2 | `issue-05/06/07` (evidência final) | vídeo/log do cliente |
| C3 | `PORTAL-02/03` (site público, loja) | decisão de produto + provider de pagamento |

## 6. Primeiras três execuções recomendadas

1. **A2** — fechar a guarda de rota com teste (a correção já está no working tree; falta o teste que
   prova que `/precos` sem cookie redireciona).
2. **A3** — triagem dos 42 `t.Skip` mexendo só em comentário/issue, sem risco de código.
3. **A4** — CI reutilizável, porque todo o resto depende de um gate confiável.

## 7. Riscos desta revisão

| Risco | Mitigação |
|---|---|
| Fechar issue por leitura de código sem executar | o aceite exige comando + saída; leitura só vira achado (como N2), não fechamento |
| Reorganização de pastas quebrar imports e fixar `go build` vermelho | nenhuma movimentação sem `go build ./...` no mesmo commit |
| "Aplicar SOLID/DDD" virar camada decorativa | só entra abstração que tenha teste que falhe sem ela |
| Playwright flaky desligado na primeira falha | poucos fluxos, `webServer` controlado, retry só de rede |
