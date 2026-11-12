# Skills do projeto — roteador

> **Regra:** carregue **uma** skill principal por tarefa. As transversais são apoio curto; não
> substituem a skill de domínio. O `SKILL.md` de cada pasta é a fonte de verdade.
> Inventário e histórico de remoções: `docs/manual-técnico/skills.md`.

## Código Rust e operação

| Tarefa | Skill |
|---|---|
| Tick, ECS, spatial grid, combate, inventário, movimento | `wyd-rust-core` |
| Framing, sessão, rate-limit, parsers, protocolo | `wyd-rust-net` |
| Persistência, schema, migrations, seeds, backup, `delivery_queue` | `wyd-rust-db` |
| Fórmulas, balanceamento, cenários determinísticos | `wyd-rust-sim` |
| Tracing, métricas, hardening, limites | `wyd-rust-obs` |
| Cargo, toolchain, fmt/clippy, CI, dependências | `wyd-rust-toolchain` |
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

| Tarefa | Skill |
|---|---|
| Revisão de código Rust, ownership, erros tipados | `rust-code-quality` |
| Tipo, nome e higiene de testes | `rust-test-quality` |
| Fronteiras, Hot/Cold, DOD/ECS, ADR | `software-architecture` |
| SOLID e patterns idiomáticos | `solid-design-patterns` |
| Direção visual e revisão de UI do portal | `frontend-design` |
| Workflows GitHub Actions | `github-actions-docs` |
| Guia/relatório em Diátaxis | `documentation-writer` |
| Planejar e decompor uma issue multi-etapa | `writing-plans` |
| Criar ou revisar skill | `skill-creator` |

## Manutenção

- Toda skill precisa de `name`, `description` com gatilho de uso, regras acionáveis e referências
  que existam hoje. Regra duplicada entre skill e `AGENTS.md` deve ficar na skill e ser linkada.
- Ao remover ou fundir uma skill, atualize este roteador, `AGENTS.md`, `docs/manual-técnico/skills.md`
  e todas as issues que a citam; registre a evidência no changelog.
