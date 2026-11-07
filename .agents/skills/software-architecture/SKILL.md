---
name: software-architecture
description: Aplica boas práticas de arquitetura e engenharia no OpenWYD (Hot Path vs Cold Path, DOD/ECS, baixo acoplamento, C4, ADRs, rastreabilidade). Use ao desenhar módulos, dividir crates, definir fronteiras entre engine/rede/persistência/portal ou antes de qualquer refatoração estrutural.
---

# Arquitetura e engenharia (boas práticas)

Referências: `start-rust.md` §§1–5, `pre-release-rust.md` §§0–1,
`roadmap-rust/issues/FASE-13.md`, `docs/arquitetura/adr/`.

## Princípios

1. **Hot Path vs Cold Path.** Tick, física, combate e visibilidade vivem juntos
   no mesmo espaço de memória (`wyd-core`); auth, billing, portal e batch vivem
   fora, async. Nenhum salto de rede no loop do jogo — latência é requisito,
   não preferência.
2. **Dados antes de objetos.** DOD/ECS com componentes contíguos; sistemas
   pequenos sobre dados, sem hierarquias profundas nem estado escondido em
   singletons globais.
3. **Fronteiras explícitas.** Cada crate tem um dono e um contrato: `wyd-protocol`
   (bytes), `wyd-net` (transporte), `wyd-core` (regras), `wyd-db` (persistência),
   `wyd-admin` (controle), `wyd-sim` (matemática), `wyd-launcher` (distribuição).
   Dependência só na direção do estável para o volátil; sem ciclos entre crates.
4. **Mudança estrutural via ADR.** Nova fronteira, novo threading, novo formato
   ou nova dependência pesada exige ADR em `docs/arquitetura/adr/` (contexto → decisão →
   consequências) antes do código.
5. **Rastreabilidade.** Problema → issue (`roadmap-rust/`) → diagnóstico →
   solução → teste → docs. Sem implementação órfã de issue.

## Checklist de desenho

- [ ] O tick continua sem I/O bloqueante, sem lock global e sem alocação quente?
- [ ] A fronteira nova é testável isolada (contrato + mocks/fakes)?
- [ ] Existe ADR? O diagrama C4 (nível 3–4) foi atualizado?
- [ ] O que é observável (métrica/log) para operar isso em produção?
