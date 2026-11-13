# Skills do projeto — roteador

> **Regra:** carregue **uma** skill principal por tarefa. As transversais são apoio curto; não
> substituem a skill de domínio. O `SKILL.md` de cada pasta é a fonte de verdade.
> Inventário e histórico de remoções: `docs/manual-técnico/skills.md`.
>
> **Orçamento de contexto:** 1 principal + no máximo 2 transversais. Se a tarefa pede uma
> terceira skill de domínio, ela está grande demais: divida em subtasks antes de continuar.
> Não carregue o catálogo inteiro nem repita no roteador o corpo que já está no `SKILL.md`.

## Código Rust e operação

| Tarefa | Skill |
|---|---|
| Tick, ECS, spatial grid, combate, inventário, movimento | `wyd-rust-core` |
| Framing, sessão, rate-limit, parsers, protocolo | `wyd-rust-net` |
| Persistência, schema, migrations, seeds, backup, `delivery_queue` | `wyd-rust-db` |
| Fórmulas, balanceamento, cenários determinísticos | `wyd-rust-sim` |
| Tracing, métricas, hardening, limites | `wyd-rust-obs` |
| Cargo, toolchain, fmt/clippy, CI, imagens, compose e deploy local | `wyd-rust-toolchain` |
| Harness, goldens, carga, bots sem cliente gráfico | `wyd-rust-testing` |
| Distribuição, patch, manifesto, launcher | `wyd-launcher` |
| RE do cliente fechado, inventário, offsets | `wyd-client-reverso` |
| Subir stack, logs, banco, diagnóstico Docker | `wyd-server-ops` |

## Jogo, portal e cliente

| Tarefa | Skill |
|---|---|
| Economia, drops, refino, EXP, preços | `wyd-economy` |
| GM, bans, grants, auditoria de staff | `wyd-game-admin` |
| Portal Next.js, BFF, sessão, telas | `wyd-web-admin-ui` |
| Cliente 7662 não lista servidor / login / versão | `wyd-client-connect` |

## Qualidade, arquitetura e processo

| Tarefa | Skill / doc |
|---|---|
| Revisão de código Rust, ownership, erros tipados | `rust-code-quality` |
| Tipo, nome e higiene de testes | `rust-test-quality` |
| Fronteiras, Hot/Cold, DOD/ECS, ADR, SOLID e patterns | `software-architecture` |
| Direção visual e revisão de UI do portal | `wyd-web-admin-ui` + `wyd-portal/README.md` §Identidade e desenho |
| Guia/relatório e documentação do projeto | `docs/index.md` (convenções) + `docs/rust/02-execucao.md` §5 |
| Planejar e decompor uma issue multi-etapa | `writing-plans` |
| Criar ou revisar skill | `docs/manual-técnico/skills.md` + `tools/dev/check-skills.mjs` |

## Manutenção

- Toda skill precisa de `name`, `description` com gatilho de uso, regras acionáveis e referências
  que existam hoje. Regra duplicada entre skill e `AGENTS.md` deve ficar na skill e ser linkada.
- Ao remover ou fundir uma skill, atualize este roteador, `AGENTS.md`, `docs/manual-técnico/skills.md`
  e todas as issues que a citam; registre a evidência no changelog.
