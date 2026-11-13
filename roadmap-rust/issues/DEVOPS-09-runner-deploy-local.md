# DEVOPS-09 — Runner auto-hospedado e deploy local após CI verde

> **ID canônico:** `DEVOPS-09`
> **ID histórico:** `DEVOPS-09`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Tipo:** DevOps · **Esforço:** M · **Depende de:** DEVOPS-02, DEVOPS-04 · **Bloqueia:** —

## Objetivo

Entregar **CD local**: quando o CI de `develop` fica verde, a máquina de desenvolvimento
reconstrói e sobe a stack do compose sozinha, com as mesmas provas de contrato da rotina de
fechamento — usando um runner auto-hospedado, sem nunca executar código de pull request e sem
colocar segredo no GitHub.

## Contexto / evidência

- Hoje o deploy local é manual (`docker compose up -d --build --wait` / `make local-rebuild`) e
  depende de o dono lembrar de rodar depois do merge. Nenhum workflow tem job de deploy.
- A máquina já tem tudo o que o deploy precisa (Docker, `.env`, `client/`, portas 8281/3000/7600)
  — e é exatamente por isso que um runner ali é uma superfície de execução de código.
- `AGENTS.md` §0/§7 proíbem reiniciar serviço durante teste humano e proíbem `down -v`/reset de
  volume: o job precisa respeitar as duas regras.
- Desenho completo (passos, travas e alternativa sem runner) em `docs/rust/10-devops-ci.md` §9.

## Passos

1. Registrar runner de **repositório** na máquina, labels `self-hosted,wyd-local`, executando
   como serviço com conta Windows dedicada **sem** privilégio de administrador. Documentar a
   instalação (comando, pasta, serviço, como remover) em `docs/rust/10-devops-ci.md` §9.
2. Workflow `deploy-local.yml`: `on: workflow_dispatch` + `workflow_run` do workflow `Rust`
   concluído com `success`, `if: github.ref == 'refs/heads/develop'`,
   `runs-on: [self-hosted, wyd-local]`, `environment: local`, `timeout-minutes` curto.
3. Passos: `git fetch` do commit validado → `docker compose build` → `docker compose up -d
   --wait --wait-timeout 180` → provas (`/login` 200, `/entregas` 307, `gateway/health` com
   `schema` igual a `SCHEMA_VERSION`, `/api/health` do portal 200) → log como artefato.
4. Travas de segurança: nenhum trigger `pull_request`; `permissions: contents: read`; segredos
   só no `.env` do host; o job não faz `push`, não faz `down -v`, não toca `client/`.
5. Trava operacional: `concurrency` exclusivo do deploy local (um por vez) e um arquivo-sentinela
   (`tmp/teste-humano.flag`) que faz o job sair com aviso em vez de reiniciar a stack no meio de
   um teste manual. Registrar a convenção no doc e na rotina de fechamento.
6. Notificação do resultado (passou/falhou) no mesmo destino do Alertmanager (`DEVOPS-02`).
7. Alternativa documentada (sem runner): polling `git pull` + `docker compose build` na máquina,
   com o mesmo bloco de provas — sabidamente sem certeza de CI verde.

## Arquivos

- Criados: `.github/workflows/deploy-local.yml`, `docs/rust/10-devops-ci.md` §9 (instalação e
  operação do runner), script de provas do deploy em `tools/dev/`.
- Alterados: `Makefile` (`deploy-local` como atalho equivalente), `docs/rust/02-execucao.md` §5
  (rotina de fechamento passa a citar o flag de teste humano), `AGENTS.md` §3 se surgir comando novo.
- Proibido: `pull_request` nesse runner; segredo em `secrets` do GitHub; conta admin.

## Testes / verificação

```bash
# no runner, com o CI verde em develop
docker compose config --quiet
docker compose build
docker compose up -d --wait --wait-timeout 180
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/login     # 200
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/entregas  # 307
curl -s http://127.0.0.1:7600/health
```

Saída esperada: job verde só com as quatro provas OK; um PR de fork **não** dispara o runner;
com `tmp/teste-humano.flag` presente o job aborta com mensagem clara.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-server-ops — subir/validar a stack e diagnosticar boot.
- .agents/skills/wyd-rust-toolchain — workflow, cache e toolchain do runner.

## Documentação de referencia

- `docs/rust/10-devops-ci.md` §9.
- `docs/rust/05-stack-docker.md` §5 (rotina de fechamento).
- `docs/rust/02-execucao.md` §2 (branch/fluxo).

## Critérios de aceite

- [ ] Push verde em `develop` reconstrói e sobe a stack local sem intervenção humana.
- [ ] Job falha (e não "quase passa") quando qualquer prova de contrato falha.
- [ ] Nenhum trigger de `pull_request`; conta sem privilégio; segredos fora do GitHub.
- [ ] Instalação/remoção do runner e a trava de teste humano documentadas.
- [ ] Resultado do deploy notificado no destino do Alertmanager.

## Riscos / rollback

- Risco: runner comprometido executa código no host com Docker → mitigação: só `push` em
  `develop`, revisão obrigatória de PR, conta dedicada, sem `secrets` de escrita.
- Risco: deploy automático atropelar teste manual → mitigação: flag/sentinela + `workflow_dispatch`
  como freio.
- Rollback: remover o runner do repositório (Settings → Actions → Runners) e desligar o workflow;
  a stack continua operável pelos comandos manuais.
