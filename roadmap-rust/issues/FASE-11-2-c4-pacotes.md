# Issue 11-2 — C4 + especificações de pacotes de rede ✅ CONCLUÍDA

**Fase:** 11 · **Depende de:** 08-1 ✅ · **Bloqueia:** — · **Esforço:** M · **Status:** ✅ concluída (2026-11-07)

## Objetivo

Publicar diagramas C4 (níveis 1–3) fiéis ao código e as especificações de
pacotes vigentes em `docs/arquitetura/`.

## O que foi executado

1. **Criado `docs/arquitetura/c4-pacotes.md`**: N1 (contexto: cliente 7662 → servidor
   Rust → Postgres + portal), N2 (containers ancorados em `Cargo.toml` members e
   services do compose `db`/`portal`), N3 (só componentes existentes: tick/ECS/spatial
   + portas, codec/wire/payloads, limits, model/repo/pool/transfer, api/auth/audit/service).
2. **Transporte futuro declarado, não desenhado**: framing/sessão `8281/tcp` é FASE-07
   (seção marcada, sem caixa inventada).
3. **Specs de pacotes ancoradas, não duplicadas**: catálogo segue normativo em
   `protocolo.md` §4–§6; §4 deste doc ancora fluxos (login `0x020D`, CNF `0x010A`–`0x011D`,
   cena `ESCENE_FIELD`, combines `0x02C3`–`0x02C4`, grants via `delivery_queue`) nos
   módulos que os produzem/consomem, com valores conferidos em
   `wire/generated/opcode_table.rs`.
4. **Rastreabilidade §5**: cada caixa cita caminho existente; regra de manutenção
   (mudança de árvore exige ADR, doc muda na mesma entrega).
5. **Correção colateral**: `visao-pastas.md` listava `docs/arquitetura/` incompleta
   (faltavam `protocolo.md`, `modelo-banco.md`); linha atualizada incluindo `c4-pacotes.md`.

## Evidência (passe 2026-11-07)

```text
cargo fmt --all -- --check                                  → FMT-OK
cargo clippy --workspace --all-targets -- -D warnings       → 0 warnings
cargo test --workspace                                     → 20 suites ok, 0 failed
grep -rln 'tmserver|Source/Code|.deprecated-go|docs/migration' docs/arquitetura/c4-pacotes.md → limpo (sem match)
opcode 0x020d/0x010a-0x011d/0x02c3-0x02c4 conferidos em wire/generated/opcode_table.rs
```

## Critérios de aceite

- [x] C4 1–3 publicado com rastreabilidade; specs de pacotes vigentes.

## Notas

- Sem `crates/*/src/` tocado (só docs). Sem mermaid: diagramas em texto + tabelas
  (repo não tem precedente de renderer; fonte como texto versionável).
- Próxima de menor esforço sugerida: **FASE-11-1** (guia de contribuição + manuais —
  documentação, sem código; `docs/manual-usuario/` hoje vazia por decisão registrada).
