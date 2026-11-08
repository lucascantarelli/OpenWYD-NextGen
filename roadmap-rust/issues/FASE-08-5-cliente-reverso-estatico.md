# Issue 08-5 — Engenharia reversa estática do cliente

**Fase:** 8 · **Depende de:** 08-4 · **Bloqueia:** 08-6, 08-7 · **Esforço:** L · **Status:** ✅

## Objetivo

Produzir o mapa estático do `WYD.exe` 7662: funções nomeadas por assinatura, estruturas
confirmadas e cada endereço herdado classificado como **CONFIRMADO** ou **REFUTADO** com
evidência do próprio binário — sem supor offset.

## Contexto / evidência

Só existiam **11 patches** e **7 hooks** conhecidos (`docs/cliente/superficie-conhecida.md`,
herdado de `ClientPatch_v7662`). Não havia mapa, RTTI nem structs confirmadas. Ambiente sem
Python, Ghidra ou rabin2; `objdump` (mingw64) e Node 26 disponíveis.

## Resultados (2026-11-07)

| Entrega | Evidência |
|---|---|
| Pipeline reproduzível | `tools/client/analyse.mjs` — um comando, exit 1 se o hash do build divergir |
| Parser PE próprio | `tools/client/pe.mjs`: PE32/i386, entry `0x5f3c66`, 4 seções, 18 DLLs/268 imports, export `WYDBR.exe!fcEXP`, `TimeDateStamp` 2015-07-02 |
| Índice de strings | 4.426 strings, 495 relevantes com VA/seção (`gerado/strings.json`) |
| Âncoras herdadas | **6/6 CONFIRMADO** como fronteira de instrução; só `SendPacket` (`0x54DA23`) é entrada de função |
| Patches de checksum | **3/3 CONFIRMADO**; os três já estão em `EB` **no disco** (build distribuído sem checagem) |
| CPSock | tabela de 512 B localizada (`SHA-256 e47996fe…`); INITCODE presente como dado; header de 12 B **CONFIRMADO INDIRETO** |
| Formatos | `.wyt` (magic `WT10`, versão, candidatos a largura/altura), `.msa` (0x112 @0, textura @34), `sn.bin` (13 nomes) documentados; `serverlist.bin` **opaco** após 4 famílias de tentativas |
| Determinismo | duas execuções produzem artefatos byte-idênticos |

**Achado que corrigiu a documentação herdada:** cinco dos seis “endereços de função” do
`ClientPatch` são **instruções internas** que o hook reescreve (`jge` em `0x4252D6`, `jne` em
`0x4676C5`, `push ecx` em `0x40126A`), não prólogos.

**Redução de escopo registrada:** T2 (structs `STRUCT_MOB`/`ITEM`/`SELCHAR`) **não** fecha com
método puramente estático sem decompilador. Transferido para a 08-6, que já dependia desta issue.

## Arquivos

- Criados: `tools/client/{README.md,pe.mjs,strings.mjs,keytable.mjs,evidence.mjs,formats.mjs,analyse.mjs}`,
  `docs/cliente/analise-estatica.md`, `docs/cliente/formatos.md`,
  `docs/cliente/gerado/{pe.json,strings.json,evidencia.json,formatos.json,ancoras.txt}`.
- Alterados: `docs/cliente/{README.md,inventario.md,superficie-conhecida.md,engenharia-reversa.md}`,
  `roadmap-rust/index.md`, `docs/rust/03-progresso.md`.
- Proibido tocar: `client/` (somente leitura), `crates/`.

## Testes / verificação

```bash
node tools/client/analyse.mjs
sha256sum client/WYD.exe   # 04c9d2a1…27ddf8
cargo fmt --all -- --check && cargo clippy --workspace -- -D warnings && cargo test --workspace
```

Saída esperada: `6/6 CONFIRMADO`, `3/3 CONFIRMADO`, artefatos gravados; segunda execução com
hashes iguais; gate com exit 0.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-client-reverso - método e higiene de RE.
- .agents/skills/wyd-rust-net - contrato de transporte.
- .agents/skills/documentation-writer - formato dos mapas.
- .agents/skills/rust-code-quality - scripts testáveis.

## Documentação de referencia

- docs/cliente/analise-estatica.md - resultado completo desta issue.
- docs/cliente/formatos.md - estrutura inicial dos formatos.
- docs/arquitetura/protocolo.md §5 - alinhamento esperado das structs.

## Subtasks

- [x] Codificar issue (pipeline)
- [x] Testar issue (execução + determinismo + gate)
- [x] Revisar issue (bytes + fronteira de instrução)
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Um comando documentado gera os artefatos sem erro e falha se o build divergir.
- [x] Cada endereço herdado classificado CONFIRMADO/REFUTADO com bytes e desassemble.
- [x] `.wyt`, `.msa`, `sn.bin` com estrutura inicial e amostra bruta; `serverlist.bin` declarado opaco.
- [x] `docs/cliente/` reflete os achados com data e evidência.
- [ ] T2 (structs de jogo) — movido para 08-6 por limitação de ferramenta, registrado aqui.

## Riscos / rollback

- Risco: confundir instrução interna com entrada de função → mitigação: classificação separa
  `instructionBoundary` de `functionEntry`.
- Risco: build diferente do inventariado → mitigação: hash conferido antes de qualquer análise.
- Rollback: `git checkout -- tools/client docs/cliente roadmap-rust/index.md`.
