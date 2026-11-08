---
name: wyd-client-reverso
description: Inspect and reverse engineer the closed WYD 7662 Windows client (client/): PE/binary inventory, hashes, CPSock wire pipeline, ClientPatch hook addresses, struct layout confirmation, static disassembly with Ghidra, dynamic tracing with x64dbg, and turning findings into documented, build-pinned evidence. Use when adding client features, hunting a client-side crash/freeze, aligning packets between client and Rust server, producing the launcher integrity manifest, or whenever an address, offset or struct layout of WYD.exe must be established or verified.
---

# Engenharia reversa do cliente (`client/`)

O cliente é **fechado** (HanbitSoft/JoyImpact, build 7662, PE32/i386) e **não é versionado**.
Não existe fonte: todo conhecimento vem de desassemble, captura de pacote e da DLL `ClientPatch_v7662` em `client/` + capturas. Dossiê canônico: `docs/cliente/`.

## 1. Antes de qualquer coisa: fixe o build

```bash
sha256sum client/WYD.exe
# esperado: 04c9d2a1235cba57f7b588347fae935a2466824ddf9d7d92e541e0cc8a27ddf8
```

Hash diferente = **outro build**. Todo endereço de `docs/cliente/superficie-conhecida.md`
deixa de valer e o dossiê precisa ser reaberto. Nunca "ajuste" um offset no chute.

## 2. O que já é conhecido (não redescubra)

| Peça | Onde |
|---|---|
| Inventário, hashes, pastas, assets | `docs/cliente/inventario.md` |
| CPSock, patches, hooks, pontos de extensão | `docs/cliente/superficie-conhecida.md` |
| Plano, alvos T1–T8, marcos M0–M6 | `docs/cliente/engenharia-reversa.md` |
| Mapa client↔launcher, modernização, plano de desassemble | `docs/cliente/mapa.md` |
| Estratégia de customização e manifesto do launcher | `docs/cliente/customizacao.md` |

Endereços-âncora: `ReadMessage 0x4252D6`, `SendChat 0x4676C5`, `SendPack 0x40126A`,
`SendPacket 0x54DA23`, checksum off `0x53AC6A/0x53AD52/0x53AE7E`. Se o desassemble não
reproduzir esses endereços, **pare**: o binário ou o mapa está errado.

## 3. Regras de evidência

1. Toda afirmação leva **build (hash) + endereço + trecho/instrução**. Sem isso, `UNVERIFIED`.
2. `UNVERIFIED` **não é gate** (`AGENTS.md` §3.5): o servidor segue conservador e loga.
3. Duas evidências independentes (xref + assinatura, ou estático + dinâmico) antes de nomear
   uma função.
4. O servidor é autoridade: nada que o cliente envia vira regra de jogo.
5. Achado novo → atualizar `docs/cliente/` **na mesma entrega**, não depois.

## 4. Fluxo de trabalho

1. **Estático primeiro**: Ghidra (headless para lote), `tools/client/*.mjs` para PE/strings.
2. **Depois dinâmico**: x64dbg **32-bit** + ScyllaHide, em VM descartável; breakpoints nos
   hooks conhecidos; trace de opcode/tamanho/hash por evento.
3. **Correlacione** com `crates/wyd-protocol`: o teste byte-exato é a prova de entendimento.
4. **Documente** em `docs/cliente/{mapa-funcoes,structs,pacotes}.md` conforme o marco (M1–M5). Roteiro de trace e fixtures em `docs/cliente/depuracao.md` + `tools/client/captura.mjs` (FASE-08-6).

## 5. Higiene e limites

- **Nunca** versione `WYD.exe`, DLL, `.sys`, dump, `WYD.log` ou `serverlist.bin` com IP privado.
- **Backup antes de escrever** em `client/`; patch só por ferramenta, aplicação atômica.
- **Não redistribua** binário/asset; registre hash e endereço, nunca conteúdo.
- **XTrap/JProtect**: diagnóstico, não combate. Se estiver ativo, teste em VM e documente o
  efeito; não escreva bypass.
- **Licença**: `ClientPatch_v7662` é GPL-3.0 (© Victor Klafke/Charles TheHouse); herdar código
  mantém a obrigação. Decisão de estratégia de patch vai em ADR (FASE-08-7).

## 6. Verificação mínima

```bash
sha256sum client/WYD.exe                                   # build correto
node tools/client/pe.mjs client/WYD.exe | head            # seções/imports (08-5)
cargo test -p wyd-protocol                                 # golden dos pacotes
cargo fmt --all -- --check && cargo clippy --workspace -- -D warnings && cargo test --workspace
```
