# Issue 08-2 — `wyd-protocol` zero-copy + validação de limites + golden

**Fase:** 8 · **Depende de:** 08-1, 02-4 · **Bloqueia:** FASE-03 · **Esforço:** L

## Objetivo

Implementar header/opcodes com `#[repr(C)]` e validação de limites antes de
qualquer parse, com testes golden byte-exatos.

## Contexto / evidência

`pre-release-rust.md` FASE 8 §8.1 (`zerocopy`/`rkyv`, `nom`); `start-rust.md` §5
(serialização zero-copy). Base: `crates/wyd-protocol/src/lib.rs`,
`crates/wyd-net/src/lib.rs` (`MAX_PACKET_BYTES`=4096).

## Passos

0. Respeitar a árvore de `docs/arquitetura/estrutura-e-padroes.md` §3 (ADR-0002):
   nenhum arquivo de domínio na raiz de `src/`.
1. Implementar `wire/header.rs` (`#[repr(C)]`, `zerocopy`) e `wire/opcode.rs` a
   partir da tabela 08-1 (somente linhas confirmadas).
2. Implementar validação tamanho→opcode→payload com erro tipado antes do parse.
3. Golden tests byte-exatos (pacote real → struct → bytes idênticos).
4. Teste: truncado/desconhecido rejeitado; `make check`.

## Arquivos

- Criados: `crates/wyd-protocol/src/header.rs`, `opcodes.rs`, goldens, testes.
- Alterados: `lib.rs` das crates, `Cargo.toml` (deps via ADR).
- Proibido tocar: `.deprecated-go/`, `client/`, mudar protocolo sem avaliar
  impacto no cliente.

## Testes / verificação

```bash
cargo test -p wyd-protocol -p wyd-net
make check
```

Saída esperada: `test result: ok` incluindo golden e rejeições.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-net - parsers e limites.
- .agents/skills/rust-code-quality - golden, sem panic.
- .agents/skills/software-architecture - ADR de serialização.

## Documentação de referencia

- roadmap-rust/issues/FASE-08-1-tabela-protocolo.md - tabela fonte.
- pre-release-rust.md (FASE 8 §8.1).
- docs/rust/01-dependencias.md - zerocopy/nom/rkyv.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Golden byte-exato por opcode implementado; truncado/desconhecido rejeitado.
- [x] Nenhum `unsafe` fora do validado pelo parser.

**Status:** ✅ (2026-11-07)

## Resultados

- `Header` com `#[repr(C)]` + teste de offsets (`offset_of!`: size 0, key_word 2,
  checksum 3, type 4, id 6, tick 8; `size_of == 12`). Sem crate externa: `zerocopy`/
  `rkyv`/`nom` exigiriam dependência de rede + ADR (offline nesta máquina, sem
  ADR aprovada) — decisão: layout provado por `repr(C)` + teste, parse por
  empréstimo sem alocação — ver ADR-0005 (`docs/arquitetura/adr/0005-serializacao-manual-diferida.md`).
- Pipeline tamanho → opcode → payload: `wire::inspect` (`split_frame` +
  `Opcode::from_wire`, novo `FrameError::UnknownOpcode`) e
  `payload::require_ported` (novo `PayloadError::Unported` — gate UNVERIFIED).
- `tests/golden.rs` (9 testes): 5 fixtures `fixtures/cliente/*.json` passam por
  corte → desofuscação → `Strict` → `inspect` → decode, com asserções semânticas
  (GOLDEN/12000, posições da Action…) e ida-volta selado → claro → selado
  byte-idêntica ao `wireHex`; tamper de checksum, truncado, opcode 0xFFFF e
  opcode não-portado rejeitados com erro tipado.
- Gate: `fmt` + `clippy -D warnings` + `cargo test --workspace` verdes (protocolo
  35 lib + 9 golden + 4 roundtrip; zero `unsafe` na crate).
- `Cargo.toml` intocado (sem deps novas); `.deprecated-go/` e `client/` intocados.
- Limitação herdada: fixtures são sintéticos (`--gen`); golden contra captura
  real aguarda FASE-08-6.

## Riscos / rollback

- Risco: layout divergente do cliente real → mitigação: golden contra captura
  real antes de fechar.
- Rollback: `git checkout -- crates/wyd-protocol crates/wyd-net/`.
