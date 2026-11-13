# 08 — Engenharia reversa: ferramentas, método e estratégia

> **Alvo:** `client/WYD.exe` build 7662 — **PE32 / i386**, image base `0x400000`, `.text`
> ~2 MB, `TimeDateStamp` 2015-07-02, D3D9 (`d3d9.dll`, `dinput8`, `dsound`, `ws2_32`,
> `wsock32`, `wininet`), 268 imports, sem packer (seções legíveis, imports completos, strings em
> claro), **sem Authenticode verificado**.
> **Escopo:** análise de artefato próprio, para interoperabilidade e preservação. Ver §6.

## 1. O que já foi feito neste repositório (sem decompilador)

| Técnica | Ferramenta usada | Resultado |
|---|---|---|
| Cabeçalho PE, seções, entry point | leitura binária + `pe.mjs` | Confirmado (`docs/cliente/analise-estatica.md`) |
| Strings e âncoras | `strings.mjs` | 6 âncoras + 3 sites de checksum |
| Desassemblage de trechos | **`objdump -d -M intel`** | provou direção de `CreateItem`/`GetItem`, offsets de campo, `memcpy` de 840 B |
| Comparação de tabela de ofuscação | hash de bloco | `pKeyWord` byte-idêntica |
| Validação de goldens C→S | `captura.mjs` | 5/5 |
| Inventário de formatos | `formats.mjs`, `wyt.mjs` | `.wyt`, `.msa`, `sn.bin`, `serverlist.bin` |
| Artefatos determinísticos | `evidence.mjs` | `docs/cliente/gerado/*.json` |

**O que falta e por quê:** grafo de chamadas (precisa de decompilador), structs S→C (precisa de
captura), comportamento em execução (precisa de GUI/VM). O repositório **não** tem Ghidra/IDA
instalados — a análise foi por `objdump` e leitura de bytes.

## 2. Comparação de ferramentas para **este** alvo

| Ferramenta | Licença/custo | Encaixe no PE32/D3D9 | Para que usar aqui | Veredito |
|---|---|---|---|---|
| **Ghidra** | livre (Apache 2.0) | x86 32-bit + decompilador; headless via script | grafo de chamadas, renomear funções, procurar referências a opcodes/constantes | **Recomendada** como base estática |
| **IDA Free / Pro** | free / pago | x86 OK; decompilador completo é do Pro | análise interativa, FLIRT, mapas de struct | Boa se já houver licença |
| **Binary Ninja** | pago | x86 OK, IL boa | análise rápida, plugins | Opcional |
| **x64dbg** | livre | 32/64-bit, breakpoints, memória, patches | trace dinâmico, hook de `SendPacket 0x54DA23` | **Recomendada** como debugger |
| OllyDbg | livre, abandonado | 32-bit apenas | compatível com a “era” do cliente | Só se houver plugin legado |
| WinDbg | livre | user/kernel, minidumps | crash dump, análise de exceção | Apoio |
| Cheat Engine | livre | scan de memória, tabela de endereços | **análise autorizada** de estruturas em memória | Apoio (não para burlar terceiros) |
| Detect It Easy (DIE) | livre | assinatura de packer/compilador | confirmar ausência de packer, MSVC | **Recomendada** (rápida) |
| PE-bear / CFF Explorer | livre | edição de header/recursos | inspecionar `.rsrc`, imports, TLS | Apoio |
| dnSpy / ILSpy | livre | **só .NET** | — | **Não aplicável** (binário nativo) |
| radare2 / rizin | livre | CLI + scripting | automação em lote, diff de builds | Apoio |
| **Frida** | livre | instrumentação dinâmica 32-bit | interceptar `send`/`recv`, ler structs em runtime | **Recomendada** para captura dirigida |
| Process Monitor | livre | arquivo/registro/rede por processo | descobrir assets carregados, ordem de leitura | **Recomendada** |
| Process Explorer | livre | processos/DLLs/handles | confirmar se `XTrap`/`jlock` carregam | Apoio |
| **Wireshark** | livre | TCP bruto | captura 8281 antes/depois de ações | **Recomendada** (com dissector próprio) |

### 2.1 Combinação recomendada (ordem de custo/benefício)

1. **Wireshark + dissector Lua/Python** para CPSock (12 B + ofuscação): transforma a captura em
   pacotes legíveis. É o maior ganho por hora.
2. **Frida** ou **x64dbg** com breakpoint em `SendPacket (0x54DA23)` e no receptor: captura o
   **frame antes da ofuscação**, que é o dado que interessa.
3. **Ghidra headless** para o grafo de chamadas e para achar todos os `cmp` de opcode (foi feito
   à mão com `objdump`; automatizar reduz erro).
4. **ProcMon** para o mapa de assets (o que abre, em que ordem, com que fallback).
5. **DIE + PE-bear** uma vez, para registrar o build no dossiê.

**Custo/limite:** tudo roda em Windows local; Ghidra exige JVM; Frida exige injetar em processo
próprio — se o anticheat (XTrap/jlock) estiver ativo, a instrumentação pode ser detectada ou
bloqueada. **Investigar isso antes** com ProcExp (quais drivers/serviços carregam).

## 3. As quatro abordagens

### A — Engenharia reversa do executável

| Critério | Avaliação |
|---|---|
| Viabilidade | Alta para **dados** (offsets, structs, tabelas); média para **fluxo** (UI, estados) |
| Dificuldade | Alta — 2 MB de `.text`, sem símbolos |
| Ferramentas | Ghidra, x64dbg, Frida |
| Conhecimento | x86, Windows API, D3D9, CPSock |
| Limitações | Não recupera nomes nem intenção; structs dependem de `#pragma pack` |
| Riscos | anticheat; confundir instrução interna com entrada de função (já aconteceu: 5 dos 6 “hooks” não são entrada) |
| Benefício | Fecha layout de `SELCHAR`/`MOB` e a direção dos pacotes |
| Manutenção | Baixa (build congelado) |
| Aplicabilidade | **Alta** — é o único caminho para o S→C |

### B — Análise de comunicação

| Critério | Avaliação |
|---|---|
| Viabilidade | **Altíssima** — o servidor é nosso e pode emitir qualquer opcode |
| Dificuldade | Média (precisa de dissector + cliente aberto) |
| Ferramentas | Wireshark, Frida, `tools/client/captura*.mjs` |
| Limitações | Só observa o que o cliente **faz**; não diz o que ele **ignora** |
| Riscos | concluir “não recebeu” quando o cliente apenas não desenha |
| Benefício | Prova de frame real, tamanho efetivo, ordem |
| Aplicabilidade | **Máxima** — resolveria 4 das 6 frentes bloqueadas |

### C — Análise de recursos e arquivos

| Critério | Avaliação |
|---|---|
| Viabilidade | **Alta** — já há parsers próprios |
| Dificuldade | Baixa para `.wyt`/`sn.bin`/`serverlist`; alta para `.msa`/`.msh`/`.ani` |
| Ferramentas | `wyt.mjs`, `formats.mjs`, `extract-tables.mjs`, Blender (malhas) |
| Limitações | Layout de tabela sem contrato provado |
| Benefício | Catálogo de itens/skills do cliente, troca de textura/UI |
| Aplicabilidade | **Alta** para conteúdo e customização |

### D — Cliente independente

| Critério | Avaliação |
|---|---|
| Viabilidade | Média no longo prazo; baixa no curto |
| Dificuldade | **Muito alta** — render, rede, UI, física, 1.700+ malhas |
| Ferramentas | engine própria ou adaptação (há precedente OSS em WebGL) |
| Limitações | Reimplementar tudo que o `.exe` desenha (layout não está em arquivo) |
| Riscos | jurídicos (assets), escopo infinito, dois alvos de protocolo |
| Benefício | Liberdade total de UI/efeitos/plataforma |
| Aplicabilidade | **Não agora**; só depois de paridade no cliente original |

## 4. Estratégia combinada recomendada

```
Fase 1 (barata):  C + B  → fechar formatos que já têm parser e capturar frames com o servidor atual
Fase 2 (média):   B + A  → S→C com Frida em SendPacket; structs no Ghidra
Fase 3 (caro):    A      → UI/estados e hooks de extensão
Fase 4 (opcional): D     → só com paridade e decisão jurídica
```

**Regra de ouro:** toda descoberta entra em `docs/cliente/` com **build + hash + endereço/arquivo**.
Sem isso é UNVERIFIED e não vira regra (`docs/cliente/README.md` §3).

## 5. Riscos técnicos específicos deste alvo

| Risco | Detalhe |
|---|---|
| Anticheat | `XTrap/XTrapVa.dll` (2,7 MB) e `JProtectDll.dll` podem detectar debugger/injeção |
| Drivers `jlock101/102.sys` | Presence não implica carga; **não** instalar/assinar nada |
| Versão de build | Todo offset vale **só** para `04c9d2a1…27ddf8`; hash diferente = dossiê reaberto |
| D3D9 legado | Em Windows moderno, o cliente falha em criar *render target* (log local) — isso afeta captura visual, não a rede |
| Ofuscação | Capturar em `recv` cru mostra bytes ofuscados; o breakpoint em `SendPacket` mostra o frame claro |
| `msvcp140d.dll` | Runtime *debug* pode mudar timing/comportamento; registrar ao comparar |

## 6. Observação jurídica e de segurança

- O projeto é de **estudo e desenvolvimento interno**, sem distribuição pública de material
  proprietário. Isso **não** elimina direitos autorais, licenças ou contratos.
- O cliente é fechado (HanbitSoft/JoyImpact). `client/` não é versionado e **não deve** ser
  redistribuído; o dossiê registra hash/offset, nunca conteúdo.
- `ClientPatch_v7662` é **GPL-3.0**: serve de leitura de endereços, **não** de código; copiar
  lógica obriga revisão da ADR-0003 antes de qualquer merge.
- Nada nesta auditoria autoriza acessar infraestrutura de terceiros, contornar autenticação de
  servidor alheio ou extrair credenciais. A análise é do **nosso** artefato e do **nosso**
  servidor.
- Ferramentas de memória (Cheat Engine/Frida) aqui são usadas para **entender o próprio cliente**
  em ambiente controlado — não para burlar anticheat de servidor de terceiro.

## 7. Próximos passos concretos (CLIENT-06)

1. Confirmar com ProcExp/ProcMon se `XTrap`/`jlock` carregam no Windows atual (define se Frida é
   viável).
2. Escrever dissector CPSock no Wireshark (header 12 B + tabela `pKeyWord` já conhecida).
3. Rodar `captura-live.mjs` com o canal Rust emitindo `CreateMob`/`ESCENE_FIELD` (não precisa de
   novo código: já emite) e comparar com os goldens.
4. Instrumentar `SendPacket` para capturar o frame S→C claro; repetir para `STRUCT_MOB`,
   `SELCHAR`, `CNFEGetItem`.
5. Só então abrir `MSG_Trade` (CLIENT-10) e item no chão (GAME-11).
