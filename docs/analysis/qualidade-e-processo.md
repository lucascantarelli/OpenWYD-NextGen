# Qualidade, arquitetura e processo — plano de profissionalização

Fotografia de 2026-11-04 e o plano para os próximos 90 dias. O que já está bom não é reescrito; o que
é dívida vira issue com critério de aceite.

## 1. Fotografia do código

| Métrica | Valor | Leitura |
|---------|------:|---------|
| Linhas de Go (sem testes) | ~91.100 | porte já é maior que o legado (99 `.cpp`) em volume de regra |
| Arquivos `_test.go` | 258 | cobertura existe, concentrada em protocolo/mundo |
| Testes do portal | 9 arquivos (`vitest`) | bom nos `lib/`, raso em páginas |
| CI | `.github/workflows/ci.yml`: gofmt, build, testes, portal | base pronta; falta integração com banco, race em tudo, cobertura e imagem |
| Migrações | 49 arquivos, versão `0042` | disciplina boa; falta teste de `down` e de migração em base vazia no CI |
| Serviços | 5 Go + portal + logpipe | fronteiras claras (`api/*`), sem `pkg` genérico |

## 2. Arquitetura: o que está certo e o que é dívida

**Certo (manter):**

- **Um dono do estado do mundo** (`world.World.Run`) — decisão que evita a classe inteira de bugs de
  duplicação; é a maior qualidade estrutural do projeto.
- **BFF**: o browser nunca fala gRPC; `adminCall` centraliza sessão, autorização injetada, deadline e
  auditoria. Padrão repetível e já repetido em 20 telas.
- **Overlay + auditoria** para conteúdo, em vez de reescrever CSV: migração incremental sem parada.
- **Contratos gRPC versionáveis** (`api/*/v1`) com `AdminResult` no corpo — erros de negócio não
  viram erro de transporte.
- **Segredos**: argon2id, `SESSION_SECRET` obrigatório, nada de hash no wire.

**Dívida priorizada:**

| # | Dívida | Por que importa | Correção |
|---|--------|-----------------|----------|
| D1 | conteúdo lido de CSV + overlay | duas verdades; deploy depende de volume | issue 50 |
| D2 | `serverlist.bin` versionado | vaza topologia, viola `AGENTS.md` | issue 56 |
| D3 | documentação sem porta de entrada/ADRs | onboarding e decisões | issue 54 |
| D4 | teste de integração apaga schema (`resetTestSchema`) e depende de banco manual | ninguém roda; store sem rede de segurança | issue 55 |
| D5 | `cmd/*/main.go` com wiring grande | difícil testar composição | issue 55 |
| D6 | validação duplicada Go/TS (login, motivo, item) | mensagens divergem | issue 55 |
| D7 | só logs, sem métricas/tracing | diagnóstico lento | issue 55 |
| D8 | `openspec/`, `project-analizer/` sem uso | ruído | issue 56 |
| D9 | ~~doc de versão do cliente inconsistente (7640 × 12000)~~ | — | **resolvido em 04/11** (issue 06: 12000 medido) |

## 3. Pirâmide de testes (alvo)

```
        e2e (poucos, caros)          login headless, fluxo de mundo, contas, faturamento
      ─────────────────────
      integração (store+gRPC)        Postgres efêmero por teste (testcontainers/ephemeral schema)
    ───────────────────────────
    unidade (muitos, baratos)        protocolo, fórmulas, RNG, validações, libs do portal
```

Regras práticas:

1. **Cada bug in-game vira teste** antes da correção (foi o que salvou B12/B16: golden case primeiro).
2. **Paridade de RNG** é teste de contrato: qualquer mudança em ordem de `rand()` quebra a suíte.
3. **Integração com banco efêmero**: subir um schema temporário por teste (`create schema test_x`),
   rodar `Migrate`, derrubar no fim. Elimina o risco de `resetTestSchema` apagar dados de dev.
4. **Cobertura mínima por pacote crítico** (`world`, `handler`, `store`, `protocol`): 70% como piso,
   não como meta.
5. **BDD só onde há regra de negócio compartilhada** com a operação (ex.: "último admin não pode ser
   removido", "bloqueio vale no próximo login") — em `testdata/*.feature` ou em teste table-driven com
   nome de cenário. Não introduzir Cucumber no runtime.

## 4. Processo (ágil, sem teatro)

- **Fluxo**: issue do roadmap → branch curta → PR com evidência → merge. Uma issue = um resultado
  verificável (regra que o roadmap já tem).
- **Definition of Done** (já implícita no `AGENTS.md`): código + teste + doc + evidência + rollback.
- **Sprint de 1 semana**, com uma meta de fechamento de issues por sprint (2–4) e uma "fila in-game"
  reservada (issue 49) para não travar em dependência humana.
- **Revisão**: PR pequeno (<400 linhas) revisado por outra pessoa/agente com foco em invariantes
  (`AGENTS.md §3`), não em estilo (gofmt/prettier cuidam).
- **ADR** para toda decisão que custa reverter: Go, overlay, `delivery_queue`, `logpipe`, banco como
  fonte de conteúdo, launcher. Formato curto: contexto, decisão, consequências, alternativas.
- **Postmortem leve** para incidente: o que quebrou, por que passou, qual teste faltava.

## 5. CI/CD e operação

Hoje o CI cobre gofmt/build/testes/portal. Evolução (issue 55):

1. **Build reprodutível** das 5 imagens + portal (cache de camada, SBOM).
2. **Integração com Postgres de serviço** no CI (`services: postgres`) e migração em base vazia.
3. **`go test -race`** em todos os pacotes, não só mundo.
4. **Cobertura publicada** com piso por pacote crítico.
5. **Portal**: `tsc`, `vitest`, `next build`, lint.
6. **E2E opcional** (nightly): sobe compose, roda login headless + eventos + contas.
7. **Observabilidade**: `/metrics` (Prometheus) no `tmserver`/`webserver`, `trace_id` nos logs, health
   por serviço (já existe healthcheck), alerta de fila (`delivery_queue` crescendo) e de migração.
8. **Segurança**: `govulncheck`, `npm audit --omit=dev`, rotação de `SESSION_SECRET`, backup testado
   (restore mensal).

Não migrar para Kubernetes antes de existir medição: hoje 1 canal, 1 banco. Compose é adequado.

## 6. Arquivos de IA (`.agents/`, `.claude/`, `openspec/`, `project-analizer/`)

Diagnóstico:

- `.agents/skills/`: **20 skills**, 41 arquivos. As específicas do projeto (`wyd-*`, 6) são boas e
  usadas. As **genéricas** (`documentation-writer`, `skill-creator`, `writing-plans`,
  `finishing-a-development-branch`, `github-actions-docs`, `frontend-design`, `ui-ux-practices`) são
  úteis mas **duplicam** o que já existe em `.claude/skills` — verificar sobreposição e manter uma
  cópia só.
- `openspec-*`: 6 skills para um fluxo com 5 mudanças paradas e **zero** referência no código.
- `.claude/`: `commands`, `settings.json`, `skills` — espelho parcial do `.agents`; risco de divergir.
- `project-analizer/`: plugin de terceiro, sem uso desde 2026-06.
- `docs/agents/`: 3 relatórios do legado C++ (junho/2026) — valor histórico, não operacional.

Plano:

1. **Fonte única**: manter `.agents/skills` como canônico e `.claude/skills` como symlink/espelho
   gerado (ou remover o segundo).
2. **Deduplicar skills genéricas** — uma por assunto.
3. **Decidir openspec**: adotar para mudanças grandes (com `config.yaml` preenchido com contexto do
   projeto) **ou** arquivar em `docs/archive/openspec/` e remover as 6 skills. Não deixar meio-termo.
4. **Remover `project-analizer/`** do repo (ou mover para `tools/` com README explicando que é
   plugin externo), mantendo `docs/agents/` como arquivo.
5. **Skills novas** que faltam: `wyd-content-import` (issue 50), `wyd-launcher` (51), `wyd-qa-ingame`
   (49) e `wyd-release` (build/deploy/rollback).

## 7. Marcos

| Marco | Entrega | Critério |
|-------|---------|----------|
| 30 dias | Fila in-game zerada + `serverlist.bin` fora do git + hub de docs | issues 49, 54, 56; 13 issues 🟡 fechadas ou reclassificadas com evidência |
| 60 dias | Conteúdo no banco + CI com banco efêmero + métricas | issues 50, 55; `tmserver` sem CSV em produção |
| 90 dias | Launcher fase 1–2 + feedback de jogadores + mapa de offsets do 12000 | issues 51, 53, 52 |
