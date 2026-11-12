---
name: software-architecture
description: Aplica arquitetura e engenharia no WYD (Hot Path vs Cold Path, DOD/ECS, baixo acoplamento, SOLID/patterns idiomáticos, C4, ADRs, rastreabilidade). Use ao desenhar módulos, dividir crates, modelar domínios, refatorar condicionais ou definir fronteiras entre engine/rede/persistência/portal.
---

# Arquitetura e engenharia (boas práticas)

Norma: `docs/arquitetura/estrutura-e-padroes.md` §1 (DAG), §5 (patterns), §6 (SOLID),
§8 (fluxo), §10–§17 (árvores) e `docs/arquitetura/adr/` (decisões). Não copie DAG,
tabela de patterns nem SOLID para a skill.

## Decisões específicas

- Fronteira nova, threading novo, formato novo ou dependência pesada: **ADR aceita antes**
  do código. Contexto → decisão → consequências; sem ADR não há “exceção temporária”.
- Dependência só na direção estável → volátil, sem ciclo. `wyd-core` não conhece SQL,
  socket ou portal; Cold Path fica fora do tick (AGENTS §2.1).
- Escolha um pattern já adotado no §5/§6; não introduza DI framework, `dyn` no Hot Path,
  herança simulada ou service locator. Genérico só com ganho medido.
- Arquivo novo só na árvore do §10–§17 e listado na issue. `mod.rs` sem lógica; gerado em
  `generated/`; teste no tipo/local do §7.
- Se a mudança alterar fronteira, atualize o C4 (`docs/arquitetura/c4-pacotes.md`) e a
  documentação viva na mesma entrega.

## Checklist

- [ ] ADR existe e foi aceita antes da mudança estrutural?
- [ ] DAG continua acíclico e a crate dona da regra não ganhou I/O?
- [ ] A solução usa pattern/SOLID já documentado, sem abstração nova sem medida?
- [ ] Árvore, C4 e issue listam os mesmos arquivos?
