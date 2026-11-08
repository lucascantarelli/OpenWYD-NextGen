# Mapa de skills — Go → Rust

> Resultado da FASE-01-3 (`roadmap-rust/issues/FASE-01-3-inventario-skills.md`).
> Insumo obrigatório da FASE-09 (auditoria profunda em 09-1, execução em 09-2).
> Fonte de verdade para agentes: `AGENTS.md` §0.

## 1. Skills Rust vigentes (não mexer sem ADR)

`wyd-rust-core`, `wyd-rust-net`, `wyd-rust-db`, `wyd-rust-sim`, `wyd-rust-obs`,
`wyd-rust-toolchain`, `wyd-rust-testing` — cobrem engine/Hot Path, transporte/protocolo,
persistência, laboratório matemático, observabilidade/hardening e build/CI.

Boas práticas transversais: `rust-code-quality`, `software-architecture`,
`solid-design-patterns`.

## 2. Skills de domínio herdadas do Go

| Skill | Destino | Motivo | Conflito atual |
|---|---|---|---|
| `wyd-server-dev` | **aposentada** (09-2 ✅) | Go do tmserver; sucessoras `wyd-rust-core`/`-net`/`-db` | — |
| `wyd-server-ops` | **suceder** → `wyd-rust-ops` (09-2) | operar stack, portas, logs, migrations, backup continua necessário, mas sobre Compose Rust | comandos `docker compose` de serviços Go, `make run-local` |
| `wyd-database` | **suceder** → `wyd-rust-db` + apêndice Postgres | schema/migrations/backup seguem válidos; ferramenta passa a `sqlx`/`refinery` | migrations em `internal/migrations/` (Go) |
| `wyd-client-connect` | **atualizar** (mesmo nome) | serverlist, versão de protocolo e diagnóstico do cliente seguem reais; muda o servidor alvo | descreve handshake do tmserver Go |
| `wyd-client-reverso` (nova) | **criada na 08-4** | RE do cliente fechado: PE/hashes, hooks por endereço, structs, Ghidra/x64dbg, evidência presa ao build | não existia no legado Go |
| `wyd-testing-bots` | **sucedida** → `wyd-rust-testing` (09-2 ✅) | harness/bots/carga em Rust | — |
| `wyd-game-admin` | **atualizar** | comandos GM e papéis são regra de jogo; backend passa a `wyd-admin` | assume binServer/webserver Go |
| `wyd-economy` | **atualizar** | preços, drops, refino e curva de EXP são domínio atemporal; ferramenta passa a `wyd-sim` | scripts/consultas no Postgres antigo |
| `wyd-web-admin-ui` | **atualizar** | o portal continua Next.js; BFF passa a falar com `wyd-admin` | `adminCall` aponta para web-api Go |

## 3. Skills de engenharia (genéricas — manter)

`documentation-writer`, `finishing-a-development-branch`, `frontend-design`,
`github-actions-docs`, `skill-creator`, `ui-ux-practices`, `writing-plans`.

Nenhuma delas cita stack Go como vigente; são reaproveitáveis como estão.

## 4. Regras para a FASE-09

1. Skill sucessora nasce com frontmatter (`name`, `description`), regras
   numeradas e referências a caminhos que existem hoje.
2. Skill aposentada recebe nota de sucessão no topo (link para a nova) — o
   conhecimento atemporal é extraído antes, nunca apagado sem registro.
3. Nenhuma skill vigente pode citar `tmserver/*.go`, `make run-local` ou gate Go
   como fluxo atual (grep de conferência em 09-2).
4. Uma skill = uma responsabilidade; sobreposição é resolvida movendo a regra,
   não duplicando.

## 5. Verificação (executada em 09-2)

```bash
grep -rln 'tmserver/.*\.go\|make run-local' .agents/skills/wyd-rust-* .agents/skills/rust-* .agents/skills/software-* .agents/skills/solid-* || echo SKILLS-LIMPAS
```
