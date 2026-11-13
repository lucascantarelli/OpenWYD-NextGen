# PROC-11 — Revisão fria da árvore real (2026-11-12)

> **Escopo:** confrontar `docs/arquitetura/estrutura-e-padroes.md` com a árvore executada
> das crates, os `Cargo.toml` e os alvos de teste. Nenhum código de jogo foi alterado.
> Divergência que exige comportamento novo vira issue; divergência documental foi corrigida
> neste mesmo commit de working tree.
>
> **Método:** `find crates/*/src -type f -name '*.rs'`, `find crates/*/tests`, leitura dos
> `Cargo.toml` e `cargo metadata`/`make check`. A árvore executada é a seção §18 do documento
> normativo; as seções marcadas “alvo” continuam sendo plano, não inventário.

## 1. DAG real (medido nos `Cargo.toml`)

```text
wyd-launcher ──► wyd-bots ──► wyd-protocol
wyd-server   ──► wyd-protocol, wyd-core, wyd-net, wyd-db, wyd-sim, wyd-admin
wyd-admin    ──► wyd-db, wyd-protocol
wyd-db       ──► wyd-core
wyd-core     ──► wyd-protocol
wyd-net      ──► wyd-protocol
wyd-sim      ──► (independente; não usa protocol)
wyd-bots     ──► wyd-protocol
```

- O documento dizia `wyd-launcher` sem dependência interna, `wyd-admin` só `wyd-db`,
  `wyd-db` independente e `wyd-sim → wyd-protocol`. Os quatro estavam errados.
- `wyd-bots → wyd-sim/wyd-net` era “PLANEJADO” e continua planejado; o crate só depende de
  `wyd-protocol` hoje.
- Não há ciclo. `wyd-db → wyd-core` é a inversão deliberada da ADR-0009 (o core define a
  porta de conteúdo; o db entrega os tipos). O texto agora diz isso em vez de negar a aresta.

## 2. Divergências por crate

| Crate | Documento dizia | Árvore real | Decisão |
|---|---|---|---|
| `wyd-launcher` | sem subdeps; §9 listava 14 arquivos | depende de `wyd-bots`; tem `cliente_zip.rs`, `portal_client.rs`, `reparo.rs`, `wyt.rs` além dos listados; testes `golden/assinatura_ed25519.rs` e `integracao/cliente_zip.rs` | Documentação corrigida; são módulos executados e testados. |
| `wyd-core` | §10 “alvo” com `rules/`, `world/`, `systems/visibility.rs`, `ports/persistence.rs` | sem esses módulos; tem `metrics/` e `telemetry.rs`; `systems/compose.rs` existe | §18 registra a árvore executada; os ausentes continuam **alvo**, não código. |
| `wyd-net` | §11 “alvo” com `transport/`, `dispatch/`, `session/state.rs` | só `session/handshake.rs`, `limits/`, `metrics.rs` | §18 registra o executado; transporte/dispatch continuam alvo. |
| `wyd-db` | §12 listava repo sem `config.rs`/`sessions.rs` | ambos existem; `seeds/` não existe | Documentação corrigida; `seeds/` segue previsto. |
| `wyd-admin` | §13 sem `api/daily.rs`, `donate.rs`, `event.rs`, `metrics.rs`, `auth/password.rs`, `service/pin.rs` | todos existem | Documentação corrigida. |
| `wyd-protocol` | §3 sem `payload/quest.rs` | existe (`_MSG_Quest`) | Documentação corrigida. |
| `wyd-sim` | §14 com `formula/speed.rs`, `bot/`; sem `formula/chaos.rs` | `chaos.rs` existe; `speed.rs` e `bot/` são reserva/plano | §18 registra o executado; reservas continuam explícitas. |
| `wyd-server` | §16 dizia que o abate “não rola loot” e que o sorteio não tinha call site | `cpsock::handle_attack` chama `loot_at_death` e `grant_loot` | Texto stale corrigido; caixa/pickup continuam sem call site. |
| `apps/launcher-ui` | §9.1 sem `assets/` e `gen/` | ambos existem | Documentação corrigida; `gen/` é saída do Tauri. |
| `apps/tools` | §15 só descrevia `tools/` | `apps/tools` é o app Node com testes (`make tools-test`) | Documentação corrigida; ADR-0028 é a referência. |

## 3. O que foi corrigido no documento

1. §1: DAG real e regra da aresta `wyd-db → wyd-core` (ADR-0009).
2. §3: `payload/quest.rs`.
3. §9: arquivos e testes reais do launcher.
4. §10/§11/§14: nota de que são **alvo**, não inventário; §18 é a árvore executada.
5. §12/§13: módulos reais que faltavam.
6. §15: `apps/tools` e o conteúdo real de `tools/dev`.
7. §16: removida a afirmação de que o abate não rola loot; o call site existe desde a
   entrega de 2026-11-11.
8. §18: árvore executada por crate, com testes e ressalvas.

## 4. O que continua aberto (não é documentação)

| Item | Por que não foi corrigido aqui | Dono |
|---|---|---|
| `wyd-net` sem `transport/`/`dispatch/`/`state.rs` | exige implementação de transporte e dispatch, fora do escopo documental | `GAME-05`/CLIENT-06 |
| `wyd-core` sem `rules/`, `world/`, `ports/persistence.rs` | são módulos-alvo ainda não necessários ao caminho atual | `GAME-19`/`GAME-21` |
| `wyd-sim` sem `formula/speed.rs` e `bot/` | reserva explícita; `GAME-20`/`QA-11` | issues citadas |
| `wyd-db/seeds/` ausente | previsto; seeds atuais são migrations | `PROC-15`/`PROC-24` |
| `apps/launcher-ui` sem prova de janela | depende do Windows/ambiente do jogador | `CLIENT-08` |
| `B6` posição exata × spawn por cidade | decisão de produto, não árvore | decisão do dono |

## 5. Verificação

```text
make check -> 0
issue-status: 36/36 antes do fechamento
docs: 36 issues, 9 crates, 10 membros, 48 migrations, 181 docs, 264 links
tools: 7/7
```

A revisão foi feita por agente nesta sessão, com medição direta da árvore; a aprovação humana
do desenho continua recomendada, mas não há mais divergência documental conhecida entre o
texto e os arquivos executados.
