# Issue 08 — Engenharia reversa: protocolo, rede e launcher

**Fase:** 8 · **Depende de:** FASE-01 · **Bloqueia:** FASE-03 · **Esforço:** L

## Objetivo

Cristalizar o contrato binário exato em `wyd-protocol` (`#[repr(C)]`, validação de
limites, zero-copy) com transporte em `wyd-net`, e entregar o launcher mínimo
(`wyd-launcher`) com verificação de integridade.

## Contexto / evidência

Toda a verdade do wire vive em `.deprecated-go/docs/migration/` (`protocol-spec.md`,
`data-formats.md`, `handlers/` 58/58, `flows.md`) e nas `STRUCT_*` do legado
(`STRUCT_MOB`=816, `STRUCT_ITEM`=8, `STRUCT_SELCHAR`=840, `STRUCT_ACCOUNTFILE`=7952).
Esqueleto atual: `crates/wyd-protocol/src/lib.rs` (PROTOCOL_VERSION=12000,
ESCENE_FIELD=30000) e `crates/wyd-net/src/lib.rs` (`MAX_PACKET_BYTES`=4096) —
`cargo test -p wyd-protocol -p wyd-net` verde. Launcher: `wyd-launcher` imprime
bootstrap; falta hash/patch (`pre-release-rust.md` FASE 8).

## Passos

1. Extrair de `protocol-spec.md` + `handlers/`: HEADER, opcodes, tamanhos e
   payload — tabela em `docs/arquitetura/` (nada UNVERIFIED vira gate).
2. Implementar `crates/wyd-protocol/src/header.rs` (layout `#[repr(C)]`,
   `zerocopy`) e `opcodes.rs`, com testes byte-exatos (golden contra capturas).
3. Implementar validação de limites em `wyd-net` (tamanho → opcode → payload),
   rejeitando com erro tipado antes de qualquer parse.
4. Launcher: `wyd-launcher` verifica hash dos arquivos, baixa patch e inicia o
   cliente; `main.rs` usa `bootstrap_message()`.
5. `make check`.

## Arquivos

- Criados: `crates/wyd-protocol/src/header.rs`, `opcodes.rs`,
  `crates/wyd-net/src/*.rs`, testes golden, doc do contrato em `docs/arquitetura/`.
- Alterados: `crates/wyd-protocol/src/lib.rs`, `crates/wyd-net/src/lib.rs`,
  `Cargo.toml` das crates (deps `zerocopy`/`nom` via ADR).
- Proibido tocar: `.deprecated-go/` (leitura apenas), `client/` (binários),
   mudar protocolo sem avaliar impacto no cliente existente.

## Testes / verificação

```bash
cargo test -p wyd-protocol -p wyd-net -p wyd-launcher
make check
```

Saída esperada: `test result: ok` incluindo golden byte-exato; pacote truncado ou
opcode desconhecido é rejeitado (teste dedicado).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-net - transporte, parsers, limites, DoS.
- .agents/skills/wyd-rust-core - consumo do contrato pela engine.
- .agents/skills/wyd-client-connect - serverlist, versão de protocolo, diagnóstico.
- .agents/skills/rust-code-quality - golden tests, sem panic.

## Documentação de referencia

- .deprecated-go/docs/migration/protocol-spec.md - HEADER, Types, framing.
- .deprecated-go/docs/migration/data-formats.md - alinhamentos e tamanhos.
- .deprecated-go/docs/migration/handlers/ - contrato de cada `_MSG_*`.
- .deprecated-go/docs/migration/flows.md - fluxos ponta a ponta.
- pre-release-rust.md (FASE 8) - launcher e zero-copy.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Opcodes/tamanhos documentados; parsers com golden byte-exato.
- [ ] Launcher verifica integridade e versiona arquivos.
- [ ] Comportamento UNVERIFIED segue caminho conservador + log (nunca gate).

## Riscos / rollback

- Risco: tratar hipótese como fato e quebrar compatibilidade com o cliente →
  mitigação: marcação UNVERIFIED obrigatória + teste contra captura real.
- Rollback: contrato versionado (`PROTOCOL_VERSION`); `git checkout -- crates/wyd-protocol crates/wyd-net`.
