---
name: wyd-client-reverso
description: Inspect and reverse engineer the closed WYD 7662 Windows client (client/): PE/binary inventory, hashes, CPSock wire pipeline, ClientPatch hook addresses, struct layout confirmation, static disassembly with Ghidra, dynamic tracing with x64dbg, and turning findings into documented, build-pinned evidence. Use when adding client features, hunting a client-side crash/freeze, aligning packets between client and Rust server, producing the launcher integrity manifest, or whenever an address, offset or struct layout of WYD.exe must be established or verified.
---

# Engenharia reversa do cliente (`client/`)

Norma: `docs/cliente/` — `inventario.md` (build/hash), `superficie-conhecida.md`
(CPSock/hooks), `engenharia-reversa.md` (marcos) e `depuracao.md` (trace). O cliente é
fechado, build 7662, PE32/i386 e não versionado.

## Decisões específicas

- Fixe o build pelo hash de `docs/cliente/inventario.md` antes de qualquer análise. Hash
  diferente invalida endereços e reabre o dossiê; não “ajuste” offset no chute.
- Toda afirmação leva build + endereço + trecho/instrução. Sem isso é `UNVERIFIED`, e
  `UNVERIFIED` não vira gate nem regra de servidor.
- Duas evidências independentes antes de nomear função (xref + assinatura ou estático +
  dinâmico). Estático primeiro; dinâmico em VM descartável, com breakpoints conhecidos.
- Correlacione com `crates/wyd-protocol` e prove com golden byte-exato; achado novo atualiza
  `docs/cliente/` na mesma entrega.
- Nunca versione/redistribua `WYD.exe`, DLL, dump, `WYD.log` ou `serverlist.bin` com IP
  privado. XTrap/JProtect: diagnostique e documente; não escreva bypass.

## Prova

`sha256sum client/WYD.exe` confere o inventário; `cargo test -p wyd-protocol` para o golden;
`make check` para o workspace.
