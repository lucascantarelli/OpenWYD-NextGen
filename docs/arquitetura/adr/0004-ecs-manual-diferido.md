# ADR-0004 — ECS manual (SoA) agora; crate ECS diferida

- **Estado:** aceita
- **Data:** 2026-11-07
- **Decisores:** dono do projeto + agente (registrado na issue FASE-13-2)
- **Governa:** `roadmap-rust/issues/FASE-03-2-ecs.md` (sistemas, bench, migração futura)

## Contexto

A FASE-03 entregou `ComponentStore` SoA manual (`crates/wyd-core/src/ecs/components.rs`):
`Vec` contíguo por componente, spawn com geração, `validate()` estrutural. Os candidatos
`hecs`/`bevy_ecs` (+ `bumpalo`) exigem rede/vendor + ADR e estão indisponíveis offline
(`docs/rust/01-dependencias.md` linhas 19–20; `pre-release-rust.md` §0.2). A engine precisa
compilar, testar e medir **agora**, sem travar o caminho crítico em download.

## Decisão

Adotamos SoA manual em `wyd-core` até o bench de guerra mandar o contrário, porque
zero dependência externa desbloqueia FASE-03/06-2/10 hoje com disciplina equivalente
(contíguo, sem alloc no quente, sem `unsafe`).

## Consequências

- **Positivas:** build offline, API mínima sob nosso controle, `ComponentStore` já medido
  (guerra 5000 em ~8,6 ms — `docs/rust/03-progresso.md` FASE-03).
- **Negativas:** sem archetypes/queryDSL prontos; sistemas futuros (`systems/`) nascem manuais.
- **Invariantes afetados:** nenhum `dyn` no quente; componente novo entra como `Vec` contíguo
  + teste de determinismo (não como struct aninhada).

## Alternativas consideradas

| Alternativa | Por que foi descartada (por enquanto) |
|---|---|
| `hecs` | exige rede/vendor; reavaliar quando houver bench comparativo |
| `bevy_ecs` | idem, + superfície maior que a necessária ao Hot Path |
| `bumpalo` (arena) | idem; `reserve()` + reuso de slot cobrem o frio atual |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/issues/FASE-03-2-ecs.md`
- Docs atualizadas nesta ADR: nenhuma (decisão registra o estado vigente)
- Evidência: `cargo test -p wyd-core` (12 lib + 3 determinismo, guerra medida)

## Revisão

- Revisar quando: bench de guerra degradar, ou rede/vendor liberarem `hecs`/`bevy_ecs`.
- Sinais de que a decisão envelheceu: sistema novo pedindo query ad-hoc complexa; alloc
  quente voltando por falta de arena.
