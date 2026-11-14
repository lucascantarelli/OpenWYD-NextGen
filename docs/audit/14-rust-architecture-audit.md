# 14 — Auditoria de arquitetura e qualidade do Rust

> Medições de 2026-11-13: 9 crates, 202 arquivos `src`, 55.675 linhas, 810 testes.
> Contagens por `grep`: `unsafe` real **0** (5 menções são comentários), `unwrap()` **124** em
> `src` + 36 em `tests`, `TODO/FIXME/todo!/unimplemented!` **16** (todos em testes de contrato).
> **Nenhum `cargo check`/`clippy` foi executado nesta sessão** — o julgamento é estrutural.

## 1. Fronteiras versus ADR-0002

| Fronteira esperada | Onde está | Desvio | Severidade |
|---|---|---|---|
| Wire puro | `wyd-protocol` | — | — |
| Engine sem I/O | `wyd-core` | — (não importa `sqlx`/`reqwest`) | — |
| Transporte | `wyd-net` **e** `wyd-server::cpsock` | transporte real no crate de composição | **Média** |
| Persistência | `wyd-db` | — | — |
| Control plane | `wyd-admin` (HTTP/JSON) | portal é BFF puro (bom) | — |
| Composição | `wyd-server` | 13.225 linhas (24% do total) | **Média** |
| Fórmulas | `wyd-sim` | há lógica simples duplicada em `core/systems/combat.rs` | Média |
| Distribuição | `wyd-launcher` | — | — |

O invariante “Hot Path sem I/O no tick” é respeitado: a thread do tick tem parada limpa, sem
`tokio` no laço (`world/tick.rs:240`) e a persistência entra por canais fora do tick.

## 2. Concorrência e estado

| Decisão | Registro | Risco observado |
|---|---|---|
| Async adotado só no Cold Path (`sqlx`) | ADR-0021 | baixo |
| Tick em thread própria por canal | `world/tick.rs` | um processo por canal — escala horizontal, não vertical |
| Estado compartilhado por `Arc`/canais | composição | `unwrap()` em `src` pode virar queda se invariante falhar |
| Sessão única por conta | DB `0030` | depende de limpeza correta no crash (há caminho de ociosidade/queda) |
| `delivery_queue` com `idempotency_key` | `0006` | bom: entrega é retry-safe |

## 3. Tratamento de erro

| Item | Estado | Comentário |
|---|---|---|
| Erros tipados por crate (`error.rs` em 5 crates) | ✅ | sem `anyhow` no caminho de rede |
| `require_ported` | ✅ | gate explícito contra layout UNVERIFIED — **exemplar** |
| `panic!` em entrada de rede | **nenhum encontrado** | `AGENTS.md` §2.8 cumprido |
| `unwrap()` em `src` | **124** | não são `panic!` de rede, mas são pontos de queda; priorizar os de parse/IO |
| `expect` com contexto | parcial | — |
| Erro engolido (`.ok()`, `let _ =`) | não medido | `UNK-24` |

## 4. Qualidade de código

| Métrica | Valor | Leitura |
|---|---:|---|
| Arquivos `src` por crate | 7–34 | granularidade boa (sem monólitos de arquivo) |
| Maior crate | `wyd-server` 34 arquivos/13.225 linhas | fronteira mais frágil |
| Menor crate | `wyd-net` 7/833 | subdimensionado para o papel |
| Duplicação conhecida | documentada em `docs/rust/09-duplicidades.md` | PROC-01 aberta |
| `TODO` em produção | **0** | bom sinal |
| `unsafe` | **0** | bom sinal |
| Testes | 810, distribuídos | cobertura alta, mas assimétrica no S→C |

## 5. Persistência e integridade

| Aspecto | Estado | Evidência |
|---|---|---|
| Migrations versionadas no boot | ✅ (ADR-0024) | `migrate.rs`, 51 arquivos |
| Transações em refino (item + piedade) | ✅ | `refino_piedade.rs` |
| Idempotência de entrega | ✅ | `delivery_queue.idempotency_key UNIQUE` |
| Save de personagem fora do tick | ✅ | `world/save.rs` |
| Integridade de inventário | 🚧 | sem baú, sem trade, sem durabilidade |
| Concorrência de update | parcial | `UNK-23`: não há teste de dois saves simultâneos da mesma conta |
| Recuperação após crash | parcial | sessão órfã depende de timeout/limpeza |

## 6. Segurança

| Controle | Estado | Nota |
|---|---|---|
| Hash de senha | ✅ Argon2id | antes do `CNF` |
| PIN | ✅ Argon2id + bloqueio persistido | 5 falhas/10 min |
| Rate-limit desde o handshake | ✅ | `wyd-net/limits` + `cpsock/limits` |
| Limite por IP | ✅ | 6 padrão |
| Autoridade server-side | ✅ | pacote forjado recusado (`integracao/autoridade.rs`) |
| Checksum | **não rejeitante, por exigência do build** | não “endurecer” |
| Segredo em log | ✅ testes afirmam ausência | `pin_e_senha_nunca_aparecem_no_log` |
| Portal | sessão httpOnly, `moderator_id` da sessão, auditoria | ADR-0008/0023 |
| Superfície de erro do portal | ✅ | JSON com limites de profundidade |
| TLS | **não adotado por decisão** (DEVOPS-03) | documentar como risco aceito |

## 7. Achados com severidade

| # | Achado | Sev | Evidência | Ação sugerida |
|---|---|---|---|---|
| A-1 | `wyd-net` não contém o transporte real | Média | 833 linhas × `cpsock` em `wyd-server` | ADR para mover ou renomear o papel |
| A-2 | `payload/` sem cobertura S→C (198 opcodes, 5 fixtures) | **Alta** | `tests/fixtures/cliente` | captura + fixtures por struct |
| A-3 | 4 resolvedores prontos sem opcode | **Alta** | checklist §5.11 | ligar skill/compose/inventory |
| A-4 | 124 `unwrap()` em `src` | Média | `grep` | trocar por `?`/`expect` nos caminhos de parse |
| A-5 | Regra de combate duplicada (`core` × `sim`) | Média | `systems/combat.rs` × `formula/damage.rs` | manter `sim` como fonte |
| A-6 | Sem colisão de jogador | **Alta** | `PAR-MOV-02` | derivar de `AttributeMap` |
| A-7 | Documentação divergente do código | Baixa | `DIV-08` | PROC-01 |
| A-8 | Sem teste de concorrência de save | Média | `UNK-23` | teste de dois saves |
| A-9 | Sem `Graphics.ini` no cliente (ambiente, não Rust) | Baixa | `ls client` | doc de setup |
| A-10 | `drop` por item × por slot do legado | Média | `DIV-05` | comparar curvas no `wyd-sim` |

## 8. O que a arquitetura faz bem (não mexer)

1. **`require_ported`** como gate de conhecimento — é a melhor defesa contra duplicação de item.
2. **Determinismo**: tick + `xorshift64` com seed provam repetibilidade (`integracao_determinism`).
3. **Fronteira Cold Path**: portal nunca fala SQL; tudo passa por `wyd-admin`.
4. **Entrega idempotente** por fila, com dreno e `attempts`.
5. **Zero `unsafe`, zero `panic!` de rede, zero TODO em produção**.

## 9. Recomendação de sequência arquitetural

```
1. captura S→C  → fixtures → payloads (A-2)
2. ligar skill/compose/inventory (A-3)
3. colisão de jogador (A-6)
4. mover/renomear transporte (A-1) — só depois de 1–2, para não misturar refactor com feature
5. `unwrap()` nos caminhos de parse (A-4)
6. teste de concorrência de save (A-8)
```

Refactor antes de paridade é o erro clássico: **feature primeiro, fronteira depois** — e sempre
com ADR quando mexer em fronteira (`AGENTS.md` §2.7).
