# DEVOPS-05 — Cache, timeouts e custo do pipeline

> **ID canônico:** `DEVOPS-05`
> **ID histórico:** `DEVOPS-05`
> **Estado 2026-11-13:** 🚧 · ver `roadmap-rust/index.md`.
> **Run real:** [43354688714](https://github.com/lucascantarelli/wyd/actions/runs/43354688714).

**Tipo:** DevOps · **Esforço:** S/M · **Depende de:** DEVOPS-04 · **Bloqueia:** DEVOPS-06

## Objetivo

Reduzir o custo e o tempo do CI sem perder cobertura: cache de dependências entre execuções,
`timeout-minutes` em todo job, fim da compilação duplicada do bench, filtro de caminho em PR e
`stable` pinado de forma reprodutível.

## Contexto / evidência

- Cinco jobs compilam o workspace (`rust`, `seed-boot`, `bench`, `release`, e `supply-chain` só
  instala ferramentas) **sem nenhum cache**: não há `Swatinem/rust-cache`, `actions/cache` de
  `~/.cargo`/`target` nem `cache-to=type:gha` nos `docker build`.
- Nenhum job declara `timeout-minutes` → default de 6 h para teste pendurado.
- `pull_request` não tem filtro de `path` (só `push` tem), então PR de documentação executa o
  pipeline inteiro.
- Job `bench`: `cargo bench --workspace --no-run` **e** `bench-gate.mjs` (que roda
  `cargo bench -p wyd-core --bench tick`) **e** `cargo bench -p wyd-net --bench limits` — o mesmo
  bench é compilado duas vezes.
- `rust-toolchain.toml` usa `channel = "stable"` (flutuante); a skill `wyd-rust-toolchain` afirma
  que "stable é pinado". **Medido em 2026-11-12 (DEVOPS-01):** dentro da base `rust:1.99-bookworm`
  o rustup baixou `1.100.0` e avisou `the missing active toolchain stable-... has been
  auto-installed` — fixar a imagem não fixa o compilador, e o build de imagem passa a depender
  de rede e da data.
- Limiares do `bench-gate` calibrados no host do dono (Windows, 2,2×–5,3× de folga), com p99
  variando ~4× na mesma máquina: risco de flake em runner compartilhado.
- Análise completa em `docs/rust/10-devops-ci.md` §1 (achados B4, C4, C5, C8).

## Run real (2026-11-13)

- Cache ativo e observável: `Cache restored successfully` do `Swatinem/rust-cache` no job `benches` do
  run [43354688714](https://github.com/lucascantarelli/wyd/actions/runs/43354688714), e
  `Build (cache gha entre execuções)` nas quatro imagens do run
  [43352177698](https://github.com/lucascantarelli/wyd/actions/runs/43352177698).
- `timeout-minutes` nos 16 jobs e bench compilado uma vez (`--no-run` + `bench-gate` no mesmo binário,
  job verde em ~2 min).
- O run provou **na prática** por que pinar a toolchain importa: o clippy do runner (1.100) acusou
  `needless_range_loop` que o local (1.99) não via; a toolchain local foi atualizada para 1.100.
- Falta: medir o ganho (minutos com/sem cache) e `bench-gate` estável em 5 execuções; e pinar
  `rust-toolchain.toml`, que segue `channel = "stable"`.

## Medição de cache (2026-11-13)

- Gate monolítico com cache quente: **3 min 17 s** (run 43417374963).
- Grupos em paralelo com `shared-key` novo (cache frio): 5 min 39 s; segundo run com cache quente:
  **4 min 29 s** — mas o caminho crítico era `bench` (4,3 min), não cache.
- Conclusão registrada: cache não é o gargalo; o gargalo é compilar o workspace no perfil de bench
  em todo push. `bench` saiu para `bench.yml` (gatilho por caminho + agenda) e o teto do gate caiu
  para `build+test` (~2,6 min).
- `bench.yml` **exercitado** por `workflow_dispatch`: run
  [43427765855](https://github.com/lucascantarelli/wyd/actions/runs/43427765855) verde em 5 min 10 s
  (compilar benches + `bench-gate` + `limits` do `wyd-net`). Observação de qualidade: o
  `bench-gate` é silencioso quando passa — publicar o resumo por cenário no log é o próximo passo
  para a régua ter evidência legível no run.

## Progresso (2026-11-13)

- Cache centralizado: `.github/actions/rust-setup` (toolchain + `Swatinem/rust-cache` com chave de
  `Cargo.lock`/`rust-toolchain.toml`) e `.github/actions/node-setup` (Node 22 + `npm ci` com cache);
  `images.yml` usa `cache-from/to: type=gha` por imagem.
- `timeout-minutes` em **todos** os 16 jobs dos cinco workflows; bench compila uma vez
  (`bench --no-run` + `bench-gate` sobre o binário já construído); `paths-ignore` em `pull_request`
  (docs/roadmap/`*.md`/`.agents`) e `cancel-in-progress` só em PR — `main`/`develop` não cancelam,
  para nenhum commit de base ficar sem gate.
- Ainda falta: medir o ganho real (comparar runs com e sem cache e publicar os minutos),
  `bench-gate` estável em 5 execuções, e pinar `rust-toolchain.toml` — hoje ainda `channel = "stable"`.

## Passos

1. Cache: `Swatinem/rust-cache@v2` nos jobs que compilam (chave por `Cargo.lock` +
   `rust-toolchain.toml`), e `cache-from/cache-to: type=gha,mode=max` nos `docker build`.
2. `timeout-minutes` por job (proporcional ao job; o `release` é o mais longo) e
   `strategy.fail-fast: false` onde fizer sentido.
3. `pull_request` com `paths-ignore` para `docs/**`, `roadmap-rust/**`, `*.md` — mantendo um job
   barato (`docs-check`) para esses PRs.
4. Bench: uma compilação só (o `bench-gate` já compila; remover o `--no-run` ou fazer o gate
   consumir a saída do `--no-run`).
5. `bench-gate`: separar ruído de regressão (mediana de N execuções ou `--test-threads` fixo) e
   registrar a máquina de referência na saída, para o limite não depender de hardware silencioso.
6. Toolchain: pinar `channel = "1.9x.y"` (versão real usada hoje) e atualizar por PR explícito;
   corrigir a afirmação da skill.
7. Medir e publicar: duração por job antes/depois e total de minutos consumidos (artefato no
   run). Sem número, não há ganho declarado (AGENTS.md: performance exige medição).

## Arquivos

- Alterados: `.github/workflows/rust.yml`, `rust-toolchain.toml`, `tools/dev/bench-gate.mjs`,
  `.agents/skills/wyd-rust-toolchain/SKILL.md`, `docs/rust/10-devops-ci.md` §1/§5.
- Proibido: desligar job para "ficar rápido"; `continue-on-error` em gate.

## Testes / verificação

```bash
# local: mesmo efeito do gate, com o binário compilado uma vez
cargo bench -p wyd-core --bench tick --no-run
node tools/dev/bench-gate.mjs
cargo test --workspace --all-targets
```

Saída esperada: dois runs consecutivos do CI com a segunda execução mais rápida (cache hit),
registrada no artefato; `bench-gate` sem flake em 5 execuções seguidas.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain — toolchain, CI e medição.

## Documentação de referencia

- `docs/rust/10-devops-ci.md` §1.
- `docs/rust/01-dependencias.md` — política de versão/MSRV.
- `docs/rust/02-execucao.md` §4 — baseline de bench.

## Critérios de aceite

- [ ] Cache ativo nos jobs que compilam, com ganho medido e publicado.
- [x] `timeout-minutes` em todos os jobs.
- [ ] Bench compilado uma vez; `bench-gate` estável em 5 execuções.
- [ ] Toolchain pinado por versão e skill corrigida.

## Riscos / rollback

- Risco: cache envenenado mascarar build quebrado → mitigação: chave inclui `Cargo.lock` e
  `rust-toolchain.toml`; job `release` pode rodar sem cache (`save-if`/`cache-all-crates`).
- Rollback: remover os passos de cache — volta ao estado atual.
