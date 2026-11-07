# Issue 08-1 — Tabela do protocolo (headers, opcodes, tamanhos, payload)

**Fase:** 8 · **Depende de:** 01-2, 02-4 · **Bloqueia:** 08-2 · **Esforço:** M

## Objetivo

Publicar em `docs/arquitetura/` a tabela exata do protocolo e entregar a crate
`wyd-protocol` **organizada na árvore planejada**, com opcodes tipados,
payloads documentados e round-trip testado.

## Contexto / evidência

`pre-release-rust.md` FASE 8 §8.1. Fontes canônicas: `protocol-spec.md`,
`handlers/`, `data-formats.md` (em `.deprecated-go/docs/migration/`),
`Basedef.h` (198 constantes `_MSG_*`) e `CPSock.cpp` (`pKeyWord[512]`).

**Estado real (2026-11-07):** existe um protótipo em `crates/wyd-protocol/src/`
(`header.rs`, `payload.rs`, `obfuscation.rs`, `opcodes.rs`, `keyword.rs`) que
decodifica/codifica, mas está **fora da árvore planejada** (ADR-0002):
`payload.rs` mistura login/combate/comércio/UI, `opcodes.rs` (1.256 linhas
geradas) está na raiz de `src/`, não há `tests/` de integração nem `generated/`.
Foi congelado; a reorganização é o primeiro passo desta issue.

## Árvore alvo (de `docs/arquitetura/estrutura-e-padroes.md` §3)

```
src/lib.rs | error.rs
src/wire/{header.rs, frame.rs, opcode.rs, generated/opcode_table.rs}
src/codec/{keyword.rs, checksum.rs, generated/keyword_table.rs}
src/payload/{auth.rs, character.rs, movement.rs, combat.rs, inventory.rs, commerce.rs, ui.rs}
tests/{common/mod.rs, roundtrip.rs}
```

## Passos

1. **Reorganizar sem mudar comportamento**: mover cada item para a árvore alvo,
   `mod.rs` só declarativo, `error.rs` com um `ProtocolError` único (Header/
   Frame/Payload/Transform), `generated/` para tabelas.
2. Gerar as tabelas com `node tools/protocol/gen-protocol.mjs && cargo fmt --all`
   (fonte: `Basedef.h` + `CPSock.cpp`), versionadas com cabeçalho "GERADO" — o
   gerador não formata a saída, o `cargo fmt` é obrigatório.
3. Publicar `docs/arquitetura/protocolo.md` com: HEADER, flags de direção,
   catálogo de opcodes (nome, valor, direção), tabela de payloads portados
   (offsets/tamanho) e a lista **UNVERIFIED** (`MSG_Trade`, `MSG_SetHpMp`,
   `_AUTH_GAME`) que não vira tipo.
4. Cobertura de round-trip: por payload portado + header + ofuscação/checksum +
   rejeição de tamanho inválido; integração em `tests/roundtrip.rs`.
5. `make check` verde; sem `unwrap`/`todo!` no caminho de parse.

## Arquivos

- Criados: `src/{error.rs, wire/**, codec/**, payload/**}`, `tests/**`,
  `docs/arquitetura/protocolo.md`, `tools/protocol/gen-protocol.mjs` (já existe).
- Alterados: `src/lib.rs` (fachada), `Cargo.toml` da crate (sem dependências novas).
- Proibido tocar: `.deprecated-go/` (leitura), `client/`, qualquer `unwrap` em
  parse, tipo para layout UNVERIFIED.

## Testes / verificação

```bash
cargo test -p wyd-protocol
make check
```

Saída esperada: `test result: ok` em unitários + `tests/roundtrip.rs`; gate
completo verde (`fmt`, `clippy -D warnings`, `build`, `test`).

## Skills Relacionadas do freebuff

- .agents/skills/software-architecture - árvore, fronteiras, ADR-0002.
- .agents/skills/solid-design-patterns - SRP por domínio em `payload/`.
- .agents/skills/wyd-rust-net - contrato de fio e hardening.
- .agents/skills/rust-code-quality - erros tipados, testes, sem stub.
- .agents/skills/documentation-writer - `docs/arquitetura/protocolo.md`.

## Documentação de referencia

- docs/arquitetura/estrutura-e-padroes.md - árvore normativa.
- docs/arquitetura/adr/0002-estrutura-modulos.md - decisão da estrutura.
- .deprecated-go/docs/migration/protocol-spec.md - espec de fio.
- .deprecated-go/Source/Code/Basedef.h / CPSock.cpp - fonte canônica gerada.

## Subtasks

- [x] Codificar issue (protótipo congelado, sem árvore)
- [ ] Reorganizar na árvore alvo (passo 1)
- [ ] Publicar `docs/arquitetura/protocolo.md` (passo 3)
- [ ] Testar issue (round-trip por payload + integração)
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue

## Critérios de aceite

- [ ] Crate segue a árvore de `estrutura-e-padroes.md` §3 (sem arquivo de
  domínio na raiz de `src/`).
- [ ] Tabela publicada com fonte por linha; UNVERIFIED marcado e sem tipo.
- [ ] Round-trip verde para todo payload portado; gate verde.
- [ ] Evidência datada em `docs/rust/03-progresso.md`.

## Riscos / rollback

- Risco: reorganizar e perder comportamento → mitigação: mover + testes verdes a
  cada passo do movimento (mesmo commit).
- Risco: tratar hipótese como fato → mitigação: UNVERIFIED nunca vira requisito.
- Rollback: `git checkout -- crates/wyd-protocol/`.
