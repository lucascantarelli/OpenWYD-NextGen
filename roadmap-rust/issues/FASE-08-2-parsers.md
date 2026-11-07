# Issue 08-2 — `wyd-protocol` zero-copy + validação de limites + golden

**Fase:** 8 · **Depende de:** 08-1 · **Bloqueia:** FASE-03 · **Esforço:** L

## Objetivo

Implementar header/opcodes com `#[repr(C)]` e validação de limites antes de
qualquer parse, com testes golden byte-exatos.

## Contexto / evidência

`pre-release-rust.md` FASE 8 §8.1 (`zerocopy`/`rkyv`, `nom`); `start-rust.md` §5
(serialização zero-copy). Base: `crates/wyd-protocol/src/lib.rs`,
`crates/wyd-net/src/lib.rs` (`MAX_PACKET_BYTES`=4096).

## Passos

1. Implementar `header.rs` (`#[repr(C)]`, `zerocopy`) e `opcodes.rs` a partir da
   tabela 08-1 (somente linhas confirmadas).
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

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Golden byte-exato por opcode implementado; truncado/desconhecido rejeitado.
- [ ] Nenhum `unsafe` fora do validado pelo parser.

## Riscos / rollback

- Risco: layout divergente do cliente real → mitigação: golden contra captura
  real antes de fechar.
- Rollback: `git checkout -- crates/wyd-protocol crates/wyd-net/`.
