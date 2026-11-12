# Mapa de skills — Go → Rust

> Resultado da PROC-07 (2026-11-12) e da auditoria de instruções de IA de 2026-11-12
> (evidência: `docs/rust/evidencias/AUDIT-IA-2026-11-12.md`).
> Roteador de uso para agentes: `.agents/skills/README.md`. Instruções sempre ativas: `AGENTS.md`.

## 1. Inventário vigente (23 `SKILL.md`)

**Domínio Rust (10):** `wyd-rust-core`, `wyd-rust-net`, `wyd-rust-db`, `wyd-rust-sim`,
`wyd-rust-obs`, `wyd-rust-toolchain`, `wyd-rust-testing`, `wyd-launcher`,
`wyd-client-reverso`, `wyd-server-ops`.

**Jogo, portal e cliente (4):** `wyd-client-connect`, `wyd-game-admin`, `wyd-economy`,
`wyd-web-admin-ui`.

**Transversais (6):** `rust-code-quality`, `rust-test-quality`, `software-architecture`,
`solid-design-patterns`, `frontend-design`, `github-actions-docs`.

**Processo (3):** `documentation-writer`, `writing-plans`, `skill-creator`.

Cada uma tem `name`, `description` com gatilho de uso, regras acionáveis e referências que
existem hoje. O roteador escolhe **uma principal por tarefa**; o corpo só é carregado sob demanda.

## 2. Histórico de sucessão e limpeza

| Skill | Destino | Motivo |
|---|---|---|
| `wyd-server-dev` | **removida** (2026-11-12) | stub Go; sucessoras `wyd-rust-core`/`-net`/`-db` |
| `wyd-testing-bots` | **removida** (2026-11-12) | alias; sucessora `wyd-rust-testing` |
| `wyd-database` | **fundida** em `wyd-rust-db` (2026-11-12) | mesma responsabilidade: persistência técnica + operação (schema, migrations, backup, diagnóstico) |
| `ui-ux-practices` | **removida** (2026-11-12) | conteúdo de outro projeto (`packages/app/ui`, `gp100-*`, `github-flow`); UI do WYD fica em `frontend-design` + `wyd-web-admin-ui` + `wyd-portal/README.md` |
| `finishing-a-development-branch` | **removida** (2026-11-12) | fluxo de commit/PR não usado e conflitante com a regra de não integrar sem pedido; ciclo em `docs/rust/02-execucao.md` |
| `writing-plans` | **reescrita** (2026-11-12) | removidos caminhos externos (`docs/superpowers/`, `superpowers:*`) e duplicação com o ciclo; agora decompõe a issue no próprio arquivo |
| `wyd-server-ops` | **atualizada** (2026-11-12) | stack real: gateway, dois canais, portal e monitoramento; removido `tmserver` |
| `wyd-rust-core`/`-net`/`-sim`/`-testing` | **atualizadas** (2026-11-12) | removidas notas de sucessão para stubs que já não existem |
| `wyd-client-connect`, `wyd-game-admin`, `wyd-economy`, `wyd-web-admin-ui` | **atualizadas** (PROC-18) | já apontam para servidor/portal Rust |
| `wyd-client-reverso` | **criada** (CLIENT-04) | RE do cliente fechado: PE/hashes, hooks, structs e evidência presa ao build |

## 3. Regras de manutenção

1. Skill nova só com lacuna demonstrada; uma skill = uma responsabilidade.
2. `AGENTS.md` não replica catálogo nem regra detalhada: aponta para este mapa e para o roteador.
3. Skill removida/fundida exige atualização do roteador, deste mapa, de `AGENTS.md` e de todas as
   issues que a citam, com busca provando zero caminho morto.
4. Nenhuma skill vigente pode citar `tmserver/*.go`, `make run-local`, `go test` ou gate Go como
   fluxo atual.

## 4. Verificação

```bash
grep -rlnE 'tmserver/.*\.go|make run-local|go test|\.agents/skills/(ui-ux-practices|finishing-a-development-branch|wyd-database)' \
  .agents/skills AGENTS.md roadmap-rust docs || echo SKILLS-LIMPAS
find .agents/skills -name SKILL.md | wc -l   # 23
find .agents/skills -type f | wc -l          # 44 (23 SKILL.md + 21 de apoio)
```
