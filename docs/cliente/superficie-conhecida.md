# Superfície conhecida do cliente

> Endereços valem **apenas** para o build
> `04c9d2a1235cba57f7b588347fae935a2466824ddf9d7d92e541e0cc8a27ddf8` (`inventario.md` §1).
> A coluna **Status** vem de `docs/cliente/analise-estatica.md` (2026-11-07): cada linha foi
> conferida por bytes + fronteira de instrução, não por herança de documentação.
> Fonte primária: `Source/Code/ClientPatch_v7662/` e `Source/Code/CPSock.cpp`; consolidação
> histórica em `.deprecated-go/docs/migration/client-patch-hooks.md` e `protocol-spec.md`.

## 1. Transporte (CPSock)

| Peça | Valor | Evidência no binário | Status |
|---|---|---|---|
| Header | 12 B: `Size(2) KeyWord(1) CheckSum(1) Type(2) ID(2) Tick(4)` | tabela de 512 B embutida + INITCODE + limite 8192 | **CONFIRMADO INDIRETO** (offsets de campo UNVERIFIED até 08-6) |
| INITCODE | `0x1F11F311` | constante `11 f3 11 1f` presente em `.data` | **CONFIRMADO (dado)**; referência em código **NÃO LOCALIZADA** |
| Tabela `pKeyWord` | 512 B, `SHA-256 e47996fe5e92de5d…` | bloco byte-idêntico localizado | **CONFIRMADO** |
| Ofuscação | mod-4 sobre `4..Size`, `pKeyWord[rst*2+1]` | implementada e testada no crate `wyd-protocol` | CONFIRMADO por paridade de tabela |
| Checksum | `Sum2 - Sum1`, **não rejeitante** | os 3 sites de checagem estão **desativados no próprio `.exe`** | **CONFIRMADO** |
| Tamanho máximo | 8192 B | candidatos `cmp reg, 0x2000` registrados em `gerado/evidencia.json` | candidato, não conclusivo |
| Versão enviada no login | **12000** | medido no handshake (skill `wyd-client-connect` §3) | CONFIRMADO por runtime |

O laço de ofuscação começa em **4**, então `Type/ID/ClientTick` também são transformados — um
pacote só de header passa pela transformação. O crate espelha isso e tem teste de ida-e-volta.

## 2. Patches de checksum — o build já vem desativado

| Endereço | Bytes no disco | Instrução | Status |
|---|---|---|---|
| `0x53AC6A` | `eb 04 33 c0 eb 35` | `jmp 0x53AC70` | **CONFIRMADO** (já é `EB`) |
| `0x53AD52` | `eb 07 33 c0 e9 5e 01 00` | `jmp 0x53AD5B` | **CONFIRMADO** (já é `EB`) |
| `0x53AE7E` | `eb 04 33 c0 eb 35` | `jmp 0x53AE84` | **CONFIRMADO** (já é `EB`) |

O `ClientPatch_v7662` descreve `jz → jmp` em runtime; o binário distribuído **já contém o salto
incondicional**. Consequência: o servidor Rust deve continuar com checksum não rejeitante para
este build; “rejeitar por padrão” quebraria o cliente de referência.

## 3. Hooks — endereços confirmados, natureza corrigida

| Símbolo | Endereço | Instrução no local | Entrada de função? | Status |
|---|---|---|---|---|
| `ReadMessage` | `0x4252D6` | `jge 0x4253CC` | não | **CONFIRMADO** |
| `SendChat` | `0x4676C5` | `jne 0x467740` | não | **CONFIRMADO** |
| `SendPack` | `0x40126A` | `push ecx` | não | **CONFIRMADO** |
| `SendPacket` | `0x54DA23` | `push ebp` | **sim** (prólogo completo) | **CONFIRMADO** |
| `AddVolatileMessageItem` | `0x422007` | `cmp [ebp+0xc], 0x15e2` | não | **CONFIRMADO** |
| `SetVolatileMessageBoxText` | `0x41FB30` | `cmp [ebp-0x1ac], 0xd3` | não | **CONFIRMADO** |

Correção de nomenclatura herdada: cinco dos seis “endereços de função” são **instruções internas
reescritas pelo hook** (o trampolim preserva a condição), não o início da rotina. Só
`SendPacket` é entrada de função. O `ClientPatch` continua sendo o mapa de referência, mas o
rótulo correto é “ponto de hook”.

**Pontos de extensão reais** (não há fonte): receber/reescrever pacote, comando de chat,
diálogo/UI e envio de pacote. Qualquer feature de cliente passa por um destes quatro.

## 4. Fluxo de entrada do jogo

```
WYD.exe → serverlist.bin → GET http://IP:80/serv00.htm → TCP IP:8281
        → INITCODE → header 12 B → MSG_AccountLogin (0x020D, ClientVersion)
        → seleção de personagem → ESCENE_FIELD (30000) como pacote autoritativo de cena
```

Strings no `.data` confirmam o caminho relativo `./serverlist.bin` e a leitura de
`.\ItemList.bin`; o launcher (`WYDLauncher.exe`) não participa do fluxo de jogo.

## 5. Formatos

Estrutura inicial documentada em `docs/cliente/formatos.md`:

| Formato | O que ficou comprovado | O que seguiu opaco |
|---|---|---|
| `.wyt` | magic `WT10`, versão `0x00020000`, candidatos a largura/altura, campo @20 variável | controles, fontes, sprites, eventos |
| `.msa` | `0x112` @0, `0x20` @4, `1` @8, nome de textura ASCII @34 | geometria, esqueleto, animação |
| `sn.bin` | 13 nomes NUL-separados a partir do offset 1 | primeiro byte `0xd6` |
| `serverlist.bin` | nenhuma hipótese sobreviveu (CPSock, XOR 1 byte, XOR tabela) | codificação e layout de canal |

## 6. Divergências conhecidas com forks

`wyd-client-feasibility.md` comparou este build com o cliente OSS `alanpetry/Wyd` (v769):
`MAX_EQUIP` 16 vs 18, `MAX_CARGO` 128 vs 120, `STRUCT_SCORE.Level` int vs short,
`STRUCT_SELCHAR` 840 vs 904 B, `MSG_CNFAccountLogin` 2008 vs ~1920 B. Transporte é idêntico;
**struct é o dialeto do build** e não se mistura. O servidor fala 7662 — nunca outro fork.

## 7. Trace dinâmico (08-6, em curso)

- Roteiro publicado em `docs/cliente/depuracao.md` (breakpoints, fases, reprodução do freeze).
- Goldens **gerados** do caminho C→S validados: `fixtures/cliente/*.json` (5/5, `captura.mjs --check`).
- Pendente da captura ao vivo: offsets de campo do header, structs S→C (`STRUCT_MOB`/`SELCHAR`/CreateMob/ESCENE_FIELD),
  codificação de `serverlist.bin`. Catálogo em `docs/cliente/pacotes.md`.
