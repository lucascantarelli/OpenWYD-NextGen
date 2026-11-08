# Mapa do cliente — usuário do `wyd-launcher`

> **Data:** 2026-11-07 · **Build de referência:** 7662 (`04c9d2a1…27ddf8`)
> Este é o mapa de navegação do dossiê: o que há em `client/`, o que está documentado,
> o que falta, por que o cliente é o consumidor do `wyd-launcher` e como ele será
> modernizado. Detalhe técnico mora nos documentos apontados — aqui não se duplica evidência.

## 1. O que há em `client/` (resumo)

`~344 MB` fora do git: `WYD.exe` (PE32/i386, 2.347.008 B) + `WYDLauncher.exe` + 5 DLLs de
proteção/gráficos + XTrap (6 arquivos) + drivers `jlock*` + catálogos (`.bin`/`.dat`) +
350 telas `.wyt` + 5.361 malhas `.msa` + áudio. Inventário completo com hashes, pastas e
revalidação em [`inventario.md`](inventario.md).

## 2. Documentado vs lacuna

| Item | Estado | Onde |
|---|---|---|
| Identidade do build (hash, PE, entry, imports) | ✅ documentado | `inventario.md` §1, `analise-estatica.md` §2 |
| 6 âncoras de hook + instrução exata | ✅ CONFIRMADO | `analise-estatica.md` §3, `superficie-conhecida.md` §3 |
| 3 sites de checksum (já `EB` no disco) | ✅ CONFIRMADO | `superficie-conhecida.md` §2 |
| Tabela CPSock + INITCODE no binário | ✅ CONFIRMADO | `analise-estatica.md` §5 |
| `.wyt` (magic, versão, dimensões candidatas) | ✅ parcial | `formatos.md` §1 |
| `.msa` (versão, textura @34) | ✅ parcial | `formatos.md` §2 |
| `sn.bin` (13 nomes, 1º byte opaco) | ✅ parcial | `formatos.md` §3 |
| `serverlist.bin` (codificação) | ✅ decodificado: grade `10×11×64` + tabela própria | `formatos.md` §4, `tools/client/serverlist.mjs` |
| Structs de jogo (`STRUCT_MOB`/`SELCHAR`/CreateMob) | ❌ lacuna | 08-6 (trace dinâmico) |
| Offsets de campo do header 12 B | ❌ lacuna | 08-6 (trace dinâmico) |
| S→C (CNFAccountLogin, CreateMob, ESCENE_FIELD) | ❌ lacuna | 08-6 (captura ao vivo) |
| Geometria/animação `.msa`, controles `.wyt`, `.xt` | ❌ lacuna | fora do caminho crítico |

## 3. Por que o cliente é o consumidor do `wyd-launcher`

A relação é unidirecional: o launcher **valida → atualiza → inicia** o `WYD.exe`; o jogo
nunca chama o launcher de volta.

```
wyd-launcher ──manifesto──▶ client/ (hashes de inventario.md §1–§4)
     │──patch atômico (tmp + rename)──▶ arquivos do jogo
     └──bootstrap──▶ WYD.exe ──▶ INITCODE → 0x020D → 0x010A → seleção → campo
```

Contrato que o launcher precisa e onde mora:

| Precisa de | Fonte |
|---|---|
| quais arquivos + SHA-256 + versão | `inventario.md` §1–§4 |
| o que um patch pode tocar sem quebrar | `superficie-conhecida.md` §2–§3 (sites e pontos de extensão) |
| como iniciar e o que observar | `depuracao.md` §2–§3 |
| estratégia de patch e licença (GPL-3.0) | ADR-0003 + `customizacao.md` |

Issues: `08-7` ✅ (verificação/patch/bootstrap) e `08-3` ✅ (distribuição dos pacotes — updater substituto, sobre a 08-7). Skill de
operação do dia a dia: `wyd-client-connect`; skill de investigação: `wyd-client-reverso`.

## 4. Roadmap de modernização / atualização / customização

| Etapa | Issue | Status | Entrega |
|---|---|---|---|
| Dossiê, inventário, baseline | 08-4 | ✅ | `docs/cliente/` + hashes |
| RE estático (PE, âncoras, formatos) | 08-5 | ✅ | `tools/client/` + `gerado/` |
| Trace dinâmico + goldens | 08-6 | 🚧 | `depuracao.md`, `fixtures/`; captura ao vivo bloqueada (sem servidor + VM) |
| Distribuição do launcher/patch (updater) | 08-3 | ✅ | construída sobre a 08-7 (entrega + assinatura) |
| Customização + manifesto | 08-7 | ✅ | implementado e testado (19/19); spawn real pendente de ambiente com jogo |

Atualizar o cliente = trocar bytes com endereço provado (`superficie-conhecida.md`) e
manifesto versionado; nunca editar binário à mão, nunca versionar binário.

## 5. Plano de desassemble pesado — quadro consolidado

Objetivo: máximo controle e conhecimento das mecânicas do jogo fechado, com cada achado
preso a build + hash + endereço. Plano completo em [`engenharia-reversa.md`](engenharia-reversa.md);
roteiro executável em [`depuracao.md`](depuracao.md).

**Confirmado até aqui (08-5):**

| Símbolo | VA | Instrução | Papel |
|---|---|---|---|
| `ReadMessage` | `0x4252D6` | `jge` | todo pacote recebido |
| `SendChat` | `0x4676C5` | `jne` | chat digitado |
| `SendPack` | `0x40126A` | `push ecx` | envio pelo socket |
| `SendPacket` | `0x54DA23` | `push ebp` (entrada) | envio original |
| `AddVolatileMessageItem` / `SetVolatileMessageBoxText` | `0x422007` / `0x41FB30` | `cmp` | item/diálogo |
| checksum ×3 | `0x53AC6A/52/7E` | `jmp` (já `EB`) | checagem desativada no disco |
| `pKeyWord[512]` + `INITCODE` | `.data` | — | transporte confirmado |

**Ferramentas e ambiente:** Node 26 + `objdump` (disponíveis); sem Ghidra/IDA/Python neste
ambiente — decompilação e xrefs ficam para 08-6 com ferramenta adequada. Execução sempre em
**VM descartável**, x64dbg 32-bit + ScyllaHide.

**Fases:** M0 baseline ✅ · M1 mapa de âncoras ✅ · M2 structs ⬜ (08-6) · M3–M5 trace/correlação
🚧 (roteiro pronto, captura bloqueada) · M6 customização ✅ (implementado; spawn real pendente).

**Higiene (innegociável):** backup antes de escrever; nada de credencial no repo; `.pcap` e
dumps fora do git; fixture real só higienizada (`GOLDEN` + `sanitized: true`); não redistribuir
binário/asset; XTrap é diagnóstico, não combate.

**Pronto quando:** todo alvo T1–T8 tem documento com build + hash + evidência; servidor
reproduz os pacotes cobertos byte a byte; launcher valida por manifesto e patch reversível;
nenhum UNVERIFIED virou gate; nada do cliente entrou no git.
