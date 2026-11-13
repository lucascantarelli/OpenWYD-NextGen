# Mapa de skills — Go → Rust

> Resultado da PROC-07 (2026-11-12) e das auditorias de instruções de IA de 2026-11-12
> (evidências: `docs/rust/evidencias/AUDIT-IA-2026-11-12.md` e
> `docs/rust/evidencias/AUDIT-SKILLS-2026-11-12.md`).
> Roteador de uso para agentes: `.agents/skills/README.md`. Instruções sempre ativas: `AGENTS.md`.

## 1. Inventário vigente (18 `SKILL.md`)

**Domínio Rust (10):** `wyd-rust-core`, `wyd-rust-net`, `wyd-rust-db`, `wyd-rust-sim`,
`wyd-rust-obs`, `wyd-rust-toolchain`, `wyd-rust-testing`, `wyd-launcher`,
`wyd-client-reverso`, `wyd-server-ops`.

**Jogo, portal e cliente (4):** `wyd-client-connect`, `wyd-game-admin`, `wyd-economy`,
`wyd-web-admin-ui`.

**Transversais (3):** `rust-code-quality`, `rust-test-quality` e `software-architecture`
(inclui SOLID/patterns).

**Processo (1):** `writing-plans`.

Cada uma tem `name`, `description` com gatilho de uso, regras acionáveis e referências que
existem hoje. O roteador escolhe **uma principal por tarefa**; o corpo só é carregado sob demanda.

## 2. Histórico de sucessão e limpeza

| Skill | Destino | Motivo |
|---|---|---|
| `wyd-server-dev` | **removida** (2026-11-12) | stub Go; sucessoras `wyd-rust-core`/`-net`/`-db` |
| `wyd-testing-bots` | **removida** (2026-11-12) | alias; sucessora `wyd-rust-testing` |
| `wyd-database` | **fundida** em `wyd-rust-db` (2026-11-12) | mesma responsabilidade: persistência técnica + operação (schema, migrations, backup, diagnóstico) |
| `ui-ux-practices` | **removida** (2026-11-12) | conteúdo de outro projeto (`packages/app/ui`, `gp100-*`, `github-flow`); UI do WYD fica em `wyd-web-admin-ui` + `wyd-portal/README.md` |
| `finishing-a-development-branch` | **removida** (2026-11-12) | fluxo de commit/PR não usado e conflitante com a regra de não integrar sem pedido; ciclo em `docs/rust/02-execucao.md` |
| `writing-plans` | **reescrita** (2026-11-12) | removidos caminhos externos (`docs/superpowers/`, `superpowers:*`) e duplicação com o ciclo; agora decompõe a issue no próprio arquivo |
| `wyd-server-ops` | **atualizada** (2026-11-12) | stack real: gateway, dois canais, portal e monitoramento; removido `tmserver` |
| `wyd-rust-core`/`-net`/`-sim`/`-testing` | **atualizadas** (2026-11-12) | removidas notas de sucessão para stubs que já não existem |
| `wyd-client-connect`, `wyd-game-admin`, `wyd-economy`, `wyd-web-admin-ui` | **atualizadas** (PROC-18) | já apontam para servidor/portal Rust |
| `wyd-client-reverso` | **criada** (CLIENT-04) | RE do cliente fechado: PE/hashes, hooks, structs e evidência presa ao build |
| 3 skills genéricas de UI, documentação e manutenção | **removidas** (2026-11-12, 3ª auditoria) | não tinham regra do WYD; as versões de conta continuam disponíveis. UI fica em `wyd-web-admin-ui` + `wyd-portal/README.md`; docs em `docs/index.md` + `docs/rust/02-execucao.md` §5; manutenção em `docs/manual-técnico/skills.md` + `tools/dev/check-skills.mjs` |
| `rust-test-quality` | **encurtada** (2026-11-12) | taxonomia/nomenclatura/higiene detalhadas agora vivem só em `docs/arquitetura/testes.md`; a skill guarda gate, comandos e checklist |
| `solid-design-patterns` | **fundida** em `software-architecture` (2026-11-12, 2ª auditoria) | mesma responsabilidade de desenho; SOLID/patterns agora vivem na skill de arquitetura, sem duas fontes |
| `github-actions-docs` | **removida** (2026-11-12, 2ª auditoria) | skill genérica de documentação externa; CI e pinning ficam em `wyd-rust-toolchain` + `.github/workflows/rust.yml` |
| `writing-plans/plan-document-reviewer-prompt.md` | **removido** (2026-11-12, 2ª auditoria) | arquivo de apoio órfão: nenhuma skill, issue ou doc o referenciava; o formato de plano já está no `SKILL.md` |

> **Conteúdo e genéricas (2026-11-12):** 18 skills de domínio/processo passaram a linkar a
> norma em vez de repetir AGENTS/docs; as 3 genéricas foram removidas do projeto. Detalhes e métricas:
> [`AUDIT-SKILLS-2026-11-12.md`](../rust/evidencias/AUDIT-SKILLS-2026-11-12.md) §6–§7.

## 3. Regras de manutenção

1. Skill nova só com lacuna demonstrada; uma skill = uma responsabilidade.
2. `AGENTS.md` não replica catálogo nem regra detalhada: aponta para este mapa e para o roteador.
3. Skill removida/fundida exige atualização do roteador, deste mapa, de `AGENTS.md` e de todas as
   issues que a citam, com busca provando zero caminho morto.
4. Nenhuma skill vigente pode citar `tmserver/*.go`, `make run-local`, `go test` ou gate Go como
   fluxo atual.

## 4. Verificação

```bash
make skills-check                            # frontmatter, 500 linhas, recursos e referências
find .agents/skills -name SKILL.md | wc -l   # 18
find .agents/skills -type f | wc -l          # 19 (18 SKILL.md + 1 README)
```
