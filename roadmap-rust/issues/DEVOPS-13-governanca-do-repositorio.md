# DEVOPS-13 — Governança do repositório: settings, proteção de branch, Slack e organização

> **ID canônico:** `DEVOPS-13`
> **ID histórico:** `DEVOPS-13`
> **Estado 2026-11-13:** 🚧 · ver `roadmap-rust/index.md`.

**Tipo:** DevOps · **Esforço:** S · **Depende de:** DEVOPS-11 · **Bloqueia:** —

## Objetivo

Deixar o repositório com as travas que faltavam para o pipeline valer alguma coisa: metadados,
proteção de branch exigindo o `gate`, um único método de merge, donos por área (`CODEOWNERS`),
canal de segurança (`SECURITY.md`) e notificação de CI no Slack.

## Contexto / evidência (levantado em 2026-11-13)

- `description` vazia, `topics` vazios, wiki ligada (a documentação é MkDocs), `delete_branch_on_merge`
  desligado e **três** métodos de merge habilitados.
- `develop` e `main` **sem proteção** (API de protection devolvia 404): qualquer push direto passava,
  então o `gate` verde era opcional.
- Sem `CODEOWNERS` e sem `SECURITY.md`.
- Job `notify` do `rust.yml` existe e fica `skipped` porque `SLACK_CI_WEBHOOK` não está configurado.
- Migrar para a organização `dolorestec` estava em avaliação: repositório **público** já tem Actions
  ilimitado em runner padrão, então o ganho seria governança (times, secrets/runner de organização),
  não cota.

## Progresso (2026-11-13)

- `CODEOWNERS` por área (engine, sim, net, protocolo, db/admin/server, launcher, portal, infra/CI,
  docs/roadmap) e `SECURITY.md` com canal privado de reporte.
- `paths-ignore` do `pull_request` removido: check obrigatório só é satisfeito se o workflow rodar —
  PR de documentação ficaria sem `gate` e travado para sempre.
- Settings aplicados via API: description, topics, wiki off, `delete_branch_on_merge`, squash como
  único método de merge (evidência no changelog do commit).
- Proteção de `develop` e `main` exigindo o check `gate (agrega os grupos)`, sem push direto, sem
  force-push e sem deleção.
- `bench.yml` exercitado por `workflow_dispatch`: run
  [43427765855](https://github.com/lucascantarelli/wyd/actions/runs/43427765855) verde (5 min 10 s),
  então nenhum workflow do repositório segue sem execução comprovada.
- `main` e `develop` passam a exigir `gate (agrega os grupos)`; como `enforce_admins=true`, nem o
  dono empurra direto — o caminho é PR com squash.

## Critérios de aceite

- [x] `CODEOWNERS` e `SECURITY.md` no repositório.
- [x] `develop` e `main` protegidas exigindo `gate`, com push direto bloqueado.
- [x] Settings de merge e metadados aplicados.
- [ ] `SLACK_CI_WEBHOOK` configurado pelo dono (canal sugerido `#wyd-ci`) — sem o segredo, `notify`
      permanece `skipped` por desenho.
- [ ] Decisão registrada sobre migrar (ou não) para a organização `dolorestec`.

## Riscos / rollback

- Risco: proteção com `enforce_admins` bloqueia o dono de corrigir `develop` direto. Rollback:
  `gh api -X DELETE repos/lucascantarelli/wyd/branches/develop/protection` (documentado em
  `docs/rust/10-devops-ci.md` §13).
- Risco: exigir o `gate` com nome errado trava o merge. O contexto usado é exatamente o job do
  orquestrador: `gate (agrega os grupos)`.
