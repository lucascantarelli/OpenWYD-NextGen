---
name: wyd-server-dev
description: APOSENTADA (FASE-09-2). Não usar como norma. Sucessoras por domínio: wyd-rust-core (engine/Hot Path), wyd-rust-net (transporte/protocolo), wyd-rust-db (persistência).
---

# APOSENTADA — `wyd-server-dev` (FASE-09-2 ✅)

Não usar como norma de implementação. O corpo anterior (servidor legado) foi removido junto com a árvore legada.

Sucessoras por domínio:

- Engine/Hot Path (tick, ECS, spatial): `wyd-rust-core`.
- Transporte/protocolo (framing, sessão, rate-limit, parsers): `wyd-rust-net`.
- Persistência (modelo, repo, migrations, delivery_queue): `wyd-rust-db`.
- Harness/goldens/carga: `wyd-rust-testing`.

Conhecimento atemporal extraído e preservado nas sucessoras: servidor é autoridade (posição, dano, inventário); layout binário por offset explícito com política Strict; pacote autoritativo de cena com ID 30000; UNVERIFIED nunca vira tipo nem gate; a web nunca escreve estado vivo (grants via delivery_queue auditada).

Fonte do destino: `docs/manual-técnico/skills.md`.
