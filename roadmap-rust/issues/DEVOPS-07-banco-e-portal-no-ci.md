# DEVOPS-07 — Testes de banco e portal no CI (fim do skip silencioso)

> **ID canônico:** `DEVOPS-07`
> **ID histórico:** `DEVOPS-07`
> **Estado 2026-11-13:** 🚧 · ver `roadmap-rust/index.md`.

**Tipo:** DevOps · **Esforço:** M · **Depende de:** DEVOPS-04 · **Bloqueia:** DEVOPS-08

## Objetivo

Fazer o CI executar de verdade o que hoje passa por omissão: os testes de sistema do
`wyd-server` contra Postgres real, a suíte do portal (`tsc`, `vitest`, `next build`) e uma regra
que torne **impossível** um teste obrigatório pular sem que o run fique vermelho.

## Contexto / evidência

- `crates/wyd-server/tests/common/mod.rs` resolve `WYD_TEST_DSN`/`WYD_TEST_DATABASE_URL`/
  `DATABASE_URL` e, sem nenhum, imprime `SKIP: ... não executado` e **retorna sucesso**. No CI só
  o job `seed-boot` cria Postgres, e apenas para `sistema_seed`; logo `tests/sistema/*`
  (login, selchar, entrada no mundo, refino/piedade, lotação, cena) roda e pula.
- O portal não é construído nem testado no CI: `npx tsc --noEmit`, `npx vitest run` e
  `next build` existem (`make portal-test`, `make portal-build`) e nenhum job os chama —
  inclusive a suíte Playwright de 6 specs da WEB-04.
- `@vitest/coverage-v8` está instalado e sem script de cobertura (ver `DEVOPS-08`).
- Análise completa em `docs/rust/10-devops-ci.md` §2/§3 (achados B2, B3).
- **Medido em 2026-11-12:** a suíte do portal está **verde localmente** — `make portal-test`
  exit 0 (`tsc --noEmit` + vitest com 29 arquivos) e `make portal-build` exit 0 (`next build`,
  34 páginas estáticas em 35,6 s). Ligar o job `portal` no CI não herda dívida: é só não haver
  job hoje. O lado do banco segue não medido nesta sessão (nenhum teste com `WYD_TEST_DSN` foi
  executado), então a regra anti-skip continua sendo o item a provar.

## Progresso (2026-11-13)

- Jobs no `rust.yml`: `portal` (`npm run typecheck`, `npm run test`, `next build`) e `sistema`
  (Postgres 17 efêmero → `--migrate` → `cargo test -p wyd-server -p wyd-admin --all-targets -j 1`
  com `WYD_TEST_REQUIRED=1`).
- Anti-skip implementado em `crates/wyd-server/tests/common/mod.rs`: sem DSN, `SKIP` continua valendo
  em execução local opcional; com `WYD_TEST_REQUIRED=1` o teste **falha**. Provado em 2026-11-13 com
  `sistema_entrada_mundo`: `SKIP` (verde) × `FAILED — WYD_TEST_REQUIRED=1 sem DSN … não pode ser pulado`.
- Defeito real encontrado de passagem: `package.json` pedia `typescript ^7.0.2` com lock em 5.9.3 —
  `npm ci` abortava com `EUSAGE` antes de qualquer verificação. Lock sincronizado; `tsc --noEmit` e
  `next build` verdes em 7.0.2; nenhum alvo usa mais `npx tsc` (que baixava um pacote homônimo).
- Ainda falta: o primeiro run do job `sistema` com Postgres efêmero (nada executou no GitHub) e um
  resumo com a contagem de testes por suíte no log do job.

## Passos

1. Job `sistema` com serviço Postgres 17 (healthcheck), `--migrate` para a série completa e
   `WYD_TEST_DSN` apontando para o banco; rodar `cargo test -p wyd-server --all-targets` e
   `cargo test -p wyd-admin --all-targets` com `-j 1` (os testes compartilham estado).
2. Regra anti-skip: variável obrigatória (ex.: `WYD_TEST_REQUIRED=1`) que faz `common::` falhar
   em vez de pular quando o DSN falta. No CI ela está sempre ligada; localmente fica opcional.
3. Job `portal`: `npm ci`, `npx tsc --noEmit`, `npx vitest run`, `npm run build`. E2E Playwright
   entra como job separado com `playwright install --with-deps chromium` e o compose mínimo
   (gateway + db) — se o custo for alto, marcar `continue-on-error` **não** é opção: ou entra
   com `needs` correto, ou fica fora com issue própria e registro.
4. Publicar contagem de testes por suíte no log (`--format`/`--reporter` resumido) para que
   "verde" tenha número ao lado.
5. `make sistema-test` (ou equivalente) para reproduzir localmente o job com o compose de dev.
6. Docs: `docs/rust/02-execucao.md` §1/§4 (o que o CI cobre), `docs/rust/10-devops-ci.md` §2.

## Arquivos

- Alterados: `.github/workflows/rust.yml`, `crates/wyd-server/tests/common/mod.rs` (regra
  anti-skip), `Makefile`, `docs/rust/02-execucao.md`, `docs/rust/10-devops-ci.md`,
  `docs/rust/05-stack-docker.md` §5 (job de CI descrito).
- Proibido: marcar teste como `#[ignore]` para o job ficar verde; afrouxar asserção.

## Testes / verificação

```bash
# prova local do job
docker compose up -d db
make db-migrate
WYD_TEST_DSN=postgres://wyd:<senha>@127.0.0.1:5432/wyd cargo test -p wyd-server --all-targets -j 1
cd wyd-portal && npx tsc --noEmit && npx vitest run && npm run build
```

Saída esperada: zero `SKIP` nos testes de sistema; `146 passed` (baseline GAME-18) ou mais;
portal com tsc/vitest/build verdes no CI.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-testing — harness, banco, carga.
- .agents/skills/rust-test-quality — nomes, gate de banco e ausência de skip silencioso.
- .agents/skills/wyd-web-admin-ui — suíte e build do portal.

## Documentação de referencia

- `docs/rust/10-devops-ci.md` §2.
- `docs/rust/02-execucao.md` §1, §4, §5.
- `docs/arquitetura/adr/0024-migrations-no-boot.md`.

## Critérios de aceite

- [ ] Testes de sistema rodam contra Postgres no CI e nenhum deles imprime `SKIP`.
- [ ] Portal com `tsc --noEmit`, `vitest` e `next build` no CI.
- [ ] Regra anti-skip ativa no CI, provada por run sem DSN que fica vermelho.
- [ ] Contagem de testes publicada por suíte no log do run.

## Riscos / rollback

- Risco: testes de sistema com estado compartilhado ficarem flaky em paralelo → mitigação:
  `-j 1` e banco por job (não por step).
- Risco: E2E Playwright instável no runner → mitigação: job separado, `retries: 1` só nele, e
  escopo reduzido ao caminho de login/entrega.
- Rollback: desligar o job novo; os testes continuam executáveis localmente.
