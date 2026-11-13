# ADR-0005 — Serialização manual (LE explícito + `repr(C)`) agora; `zerocopy`/`nom` diferidos

- **Estado:** aceita
- **Data:** 2026-11-07
- **Decisores:** dono do projeto + agente (registrado na issue PROC-22)
- **Governa:** `roadmap-rust/changelog.md` (parsers, goldens, migração futura)

## Contexto

A CLIENT-02 entregou parsers por leitura explícita little-endian + `#[repr(C)]` com offsets
provados por teste (`crates/wyd-protocol/src/wire/header.rs`), pipeline tamanho → opcode →
payload e 9 goldens byte-idênticos. Os candidatos `zerocopy`/`nom`/`rkyv` exigem rede/vendor
(`docs/rust/01-dependencias.md` linha 24; `pre-release-rust.md` §0.2) e estão indisponíveis
offline. O protocolo precisa estar congelado e testado **agora**.

## Decisão

Adotamos serialização manual com layout provado por teste até haver motivo medido para
trocar, porque 16 payloads + 198 opcodes já passam em goldens sem nenhuma dependência.

## Consequências

- **Positivas:** zero `unsafe` na crate, build offline, goldens como rede de proteção total.
- **Negativas:** cada payload novo exige offsets manuais + teste (custo linear, visível).
- **Invariantes afetados:** payload novo sem golden não entra; `repr(C)` + `offset_of!`
  obrigatórios em header de fio; `UNVERIFIED` nunca vira tipo sem captura real.

## Alternativas consideradas

| Alternativa | Por que foi descartada (por enquanto) |
|---|---|
| `zerocopy` | exige rede/vendor; reavaliar se goldens acusarem divergência de layout |
| `nom` | idem; o pipeline atual já é total sem combinadores |
| `rkyv` | idem, + formato próprio incompatível com o fio legado |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/changelog.md`
- Docs atualizadas nesta ADR: nenhuma (decisão registra o estado vigente)
- Evidência: `cargo test -p wyd-protocol` (35 lib + 9 golden + 4 integração)

## Revisão

- Revisar quando: payload novo com layout irregular (bitfields, compressão) ou rede/vendor
  liberarem os candidatos.
- Sinais de que a decisão envelheceu: goldens frágeis a cada payload; parse manual virando
  fonte de bugs em vez de proteção.
