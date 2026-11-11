# Pacotes do caminho login → campo

> Tabela viva do contrato C↔S com o estado de cada pacote. `GERADO` = golden sintético
> (`fixtures/cliente/*.json`) validado por `captura.mjs`. `PENDENTE` = só fechado com a
> captura ao vivo (roteiro em `depuracao.md`). Tamanhos e fluxos vêm de
> `protocol-spec.md` §3.5 + `flows.md`; o crate usa os mesmos valores (`payload.rs`).

## C→S (gerados)

| Opcode | Nome | Tamanho | Fixture | Status |
|---|---|---:|---|---|
| `0x020D` | AccountLogin | 116 | `account_login.json` | **GERADO** |
| `0x020F` | CreateCharacter | 36 | `create_character.json` | **GERADO** |
| `0x0213` | CharacterLogin | 20 (cliente 7662: 36) | `character_login.json` | **GERADO** |
| `0x036C` | Action | 52 | `action.json` | **GERADO** |
| `0x0333` | `MessageChat` | 140 | `text[128] @12` | **RE** (sender `0x4695b3`; receptor `0x48241f` lê `HEADER.ID` = falante; `/cp` + chat social) |

## S→C (aguardando captura)

| Opcode | Nome | Tamanho esperado | Status |
|---|---|---:|---|
| `0x010A` | CNFAccountLogin | 2008 | **PENDENTE** (conferir `Size` no wire) |
| `0x0110` | CNFNewCharacter | STRUCT_SELCHAR | **PENDENTE** |
| `0x0112` | CNFDeleteCharacter | `STRUCT_SELCHAR` (844) | **CORPO PORTADO** (mesmo layout do `CNFNewCharacter`) |
| `0x0116` | CNFCharacterLogout | só header | **PORTADO** (resposta ao `0x0215`) |
| `0x0114` | CNFCharacterLogin | STRUCT_MOB + cauda | **PENDENTE** |
| `0x0364` | CreateMob | 232 | **CORPO PORTADO + PROVADO NO FIO** (`Basedef.h` + porte Go; sessão de jogo real decodificou 6100 frames; captura 08-6 pendente) |
| `0x0165` | RemoveMob | 16 | **CORPO PORTADO + PROVADO NO FIO** (`Basedef.h` + porte Go; delta de revisão removendo e recriando NPC na mesma sessão; captura 08-6 pendente) |
| `0x7530` | ESCENE_FIELD | cena autoritativa | **PENDENTE** |

## O que o build 7662 realmente trata (desassemble 2026-11-11)

O roteador de recebimento compara `HEADER.Type` (`WORD [frame+4]`) com um conjunto finito de
opcodes. Extraindo **todos** os `cmp reg, imm` do `WYD.exe` e cruzando com a tabela
`crates/wyd-protocol/src/wire/generated/opcode_table.rs`, os opcodes do nosso catálogo que o
cliente realmente compara são 91 (de `0x0101 MessagePanel` a `0x0FDE AccountSecure`).

Consequências medidas — não deduzidas:

| Opcode | Nome | Presente no cliente? | Efeito prático |
|---|---|---|---|
| `0x0101` | MessagePanel | **sim** | única via de texto livre no jogo |
| `0x0102` | MessageBoxOk | compara, mas cai no mesmo bloco que só trata `0x0101/0x0105/0x0106` | não desenha caixa por si |
| `0x0110` | CNFNewCharacter | **sim** | copia 0x348 (840 B) de `frame+0x10` (sel@4 do corpo) |
| `0x0112` | CNFDeleteCharacter | **sim** | mesmo `memcpy` 840 B de `frame+0x10` → corpo de **844 B** (`sel@4`) |
| `0x0117` | NewAccountFail | **NÃO** (zero ocorrências) | senha errada com esse pacote não mostra nada — é o caso relatado no teste manual |
| `0x0119` / `0x011a` / `0x011b` | CharacterLoginFail / NewCharacterFail / DeleteCharacterFail | **sim** | cada um exibe mensagem própria do cliente |

O `CNFDeleteCharacter` com `Size = 852` (12 de header + 844) está **correto**: o cliente lê os
840 B a partir de `frame+16`, exatamente `sel@4` no corpo. Já o `_MSG_NewAccountFail` (0x0117)
não existe no binário; para a senha errada aparecer na tela é preciso usar um opcode que o
cliente desenhe (`MessagePanel`) com o layout provado — **não** foi trocado às cegas nesta
entrega.

O canal responde hoje com corpo em `0x010A` (`MSG_DBCNFAccountLogin`, 1996 B), `0x0110`
(`STRUCT_SELCHAR`, 844 B), `0x0112` (o mesmo `STRUCT_SELCHAR` depois da remoção) e `0x0114`
(`MSG_CNFClientCharacterLogin`, 1820 B); os `*Fail` do domínio continuam só de cabeçalho. O
resultado de cada operação é decidido no Postgres e o corpo é o mesmo estado que o banco
serializa — a diferença para a captura ao vivo é de campo, não de fluxo.

## Caixa de item no chão e pickup — direção e pack lidos do build 7662

**Como foi obtido:** análise estática de `client/WYD.exe` (7662, `SHA-256 04c9d2a1…27ddf8`)
com `objdump -d -M intel client/WYD.exe` sobre `.text`; cada linha abaixo cita a instrução pelo VA.
**Não é captura de rede**, e isso é deliberado: a captura ao vivo
(`tools/client/captura-live.mjs --alvo`) exige o cliente aberto na GUI **e** um servidor que
emita o pacote — o canal Rust ainda não emite `CreateItem` nem `GetItem`, então não haveria byte
a capturar. Para as duas perguntas em aberto (direção e pack) o binário responde melhor que uma
amostra: mostra o código que **envia** e o que **recebe**.

Tabela do `Basedef.h` (mesma fonte de `CreateMob`/`RemoveMob`), mantida como referência — o que o
binário confirma e o que ele corrige está nas seções seguintes. Offsets a partir do início do
frame; `STRUCT_ITEM` = 8 B (`short sIndex` + `stEffect[3]`, cada efeito 2 B):

| Opcode | Nome | Direção no `Basedef.h` | Corpo | Campos |
|---|---|---|---:|---|
| `0x026E` | CreateItem | C→J (declarado) | 18 B | `GridX`+12, `GridY`+14, `ItemID`+16, `STRUCT_ITEM`+18, `Rotate`+26, `State`+27, `Height`+28, `Create`+29 |
| `0x016F` | DecayItem | J→C | 4 B | `ItemID`+12 (`short`), `unk`+14 |
| `0x0270` | GetItem | C→J | 16 B | `DestType`+12, `DestPos`+16, `ItemID`+20, `GridX`+22, `GridY`+24 |
| `0x0171` | CNFGetItem | J→C | 16 B | `DestType`+12, `DestPos`+16, `STRUCT_ITEM`+20 |
| `0x0272` | DropItem | C→J | 20 B | `SourType`+12, `SourPos`+16, `Rotate`+20, `GridX`+24, `GridY`+26 |
| `0x0175` | CNFDropItem | J→C | 16 B | `SourType`+12, `SourPos`+16, `Rotate`+20, `GridX`+24, `GridY`+26 |

### Direção — o `Basedef.h` erra no `CreateItem`

| Opcode | Nome | `Basedef.h` declara | Build 7662 | Veredito |
|---|---|---|---|---|
| `0x026E` | CreateItem | C→J | **recebido**: `0x495dc5 cmp edx,0x26e` → `call 0x486c25` | **S→C** |
| `0x016F` | DecayItem | J→C | recebido: `0x495e4d` → `call 0x487959` | S→C |
| `0x0270` | GetItem | C→J | **enviado**: `0x4fee1b`..`0x4fee7e` (`memset` 0x1c + `SendPacket` 0x54da23) | **C→S** |
| `0x0171` | CNFGetItem | J→C | recebido: `0x495e09` → `call 0x487580` | S→C |
| `0x0175` | CNFDropItem | J→C | recebido: `0x495de7` → `call 0x4872ba` | S→C |

O roteador de recebimento (`0x495d..`) é uma cadeia de `cmp <Type>,imm` + `call <handler>`: só
entra ali o que **chega** do servidor. `GetItem` não está nessa cadeia — ele é montado no cliente e
entregue a `SendPacket`. Consequência para o nosso lado: a entrada `CreateItem` de
`crates/wyd-protocol/src/wire/generated/opcode_table.rs` está como `Flow::Client2Game`, o oposto do
binário; corrigir tabela congelada é rito próprio e **não** foi feito nesta entrega.

### `_MSG_CreateItem` (0x026E, S→C) — 30 B, **packed**

Handler `0x486c25`; a mensagem é `[ebp+0x8]`, guardada em `[ebp-0x14]`:

| Offset | Largura | Instrução (VA) | Campo |
|---|---|---|---|
| +0x0C | u16 | `486d73` / `486ef5` `mov ..,[..+0xc]` | `GridX` |
| +0x0E | u16 | `486d91` / `486eeb` `mov ..,[..+0xe]` | `GridY` |
| +0x10 | u16 | `486c51 mov dx,[ecx+0x10]` (vai a lookup de template, `[…+0x38]`) | `ItemID` |
| +0x12 | 8 B | `486c6e` / `486eb3` `add ecx,0x12`; lido em dois dwords (`+0x12`, `+0x16`) | `STRUCT_ITEM` (`sIndex` + `stEffect[3]`) |
| +0x1A | u8 | `486d48 mov cl,[eax+0x1a]` (usado como ângulo: `fild`/`fmul`/`fdiv`) | `Rotate` |
| +0x1B | u8 | `486eff` / `486f16` `mov dl,[..+0x1b]` | `State` |
| +0x1C | **i8** | `486ec5 movsx eax,[edx+0x1c]` | `Height` (lido com sinal) |
| +0x1D | u8 | `487154 mov cl,[eax+0x1d]` + `487157 cmp ecx,0x1` | `Create` (flag comparado com 1) |

Todos os campos caem nos offsets crus do `Basedef.h`, sem folga entre +0x12 e +0x1A (o
`STRUCT_ITEM` é lido em dois dwords consecutivos) → **packed**, corpo de 18 B e frame de **30 B**.
O decoder do cliente não valida `Size` por opcode, então o binário prova a posição de cada campo e,
por ela, o mínimo de 30 B — não prova que o servidor legado não mande 32 com padding.

### `_MSG_GetItem` (0x0270, C→S) — 28 B (14 B de corpo + 2 B de padding)

`0x4fee1b`: `push 0x1c` / `push 0` / `lea [ebp-0x4c]` / `call 0x5c5b80` (`memset(buf,0,28)`);
`0x4fee78`: `push 0x1c` / `lea [ebp-0x4c]` / `call 0x54da23` (`SendPacket(ptr, 28)`). Base = `ebp-0x4c`:

| Offset | Largura | Instrução (VA) | Campo |
|---|---|---|---|
| +0x04 | u16 | `4fee36 mov WORD [ebp-0x48],0x270` | `Type` |
| +0x06 | u16 | `4fee32 mov WORD [ebp-0x46],cx` | `ID` (id do jogador, `obj+0x20`) |
| +0x0C | i32 | `4fee47 mov DWORD [ebp-0x40],0x1` | `DestType` (constante **1** neste caminho) |
| +0x10 | i32 | `4fee57 mov DWORD [ebp-0x3c],ecx` | `DestPos` (`y*5 + x`, par vindo de chamada virtual `[…+0x9c]`) |
| +0x14 | u16 | `4fee43 mov WORD [ebp-0x38],ax` | `ItemID` |
| +0x16 | u16 | `4fee65 mov WORD [ebp-0x36],ax` | `GridX` (float do item → inteiro) |
| +0x18 | u16 | `4fee74 mov WORD [ebp-0x34],ax` | `GridY` (idem) |
| +0x1A..+0x1B | 2 B | zerado pelo `memset` | **padding** |

Os offsets de campo do `Basedef.h` estão certos; o que ele não diz é o tamanho — o buffer vai a
**28 B** (struct alinhado a 4, 26 → 28) e os 2 últimos bytes vão zerados. Um decoder que exigir
26 B rejeita o pacote real.

### O que continua UNVERIFIED

1. `DestType` de `GetItem` só foi visto como **1** (um caminho do cliente). Equipar, baú e venda
   precisam de segunda ocorrência no binário ou de captura.
2. `CreateItem` fecharia em 30 B pelo mapa de campos; um frame real confirmaria que o servidor
   legado também não usa padding (32 B). O item 1 acima e este são os únicos dois pontos que a
   captura ao vivo ainda acrescenta.
3. Faixa de ids de caixa: o original usa ids próprios, separados de jogador (`MAX_USER` 1000) e de
   NPC (`MAX_MOB` 25000); o valor exato segue sem fonte.

**Menor caminho auditável para consumir drops por esse fluxo** (proposta, em ordem):

1. **Morte resolve o loot** (já implementado): `Opcode::Attack*` → `Engine::attack_at` tira a
   criatura do mundo → `SelectionStore::attack` → `Gameplay::loot_at_death(template, die)` com um
   roll por item da tabela do snapshot; os rolls e os itens vão para `event=death`/`event=loot`.
2. **Materializar a caixa**: por item que caiu, criar **um** `CreateItem` (0x026E) na posição da
   morte, com um id de caixa do canal (tabela em memória `id → (item, posição, instante)`) e
   `STRUCT_ITEM` do catálogo. Reusa `reply_with_body`, igual a `CreateMob`.
3. **Coletar**: tratar `GetItem` (0x0270) — validar distância contra a posição autoritativa,
   gravar o item em `character_items` e responder `CNFGetItem` (0x0171) + `DecayItem` (0x016F)
   para a caixa sumir da vista de todos.
4. **Expirar**: `DecayItem` por timeout, sem tabela nova (o mapa do canal é a fonte).

Cada passo é uma entrega com teste próprio: o passo 1 não depende de layout; os passos 2–4 já têm
layout confirmado no binário (acima), faltando só a faixa de ids de caixa.

### Reprodução da leitura

```bash
sha256sum client/WYD.exe                      # 04c9d2a1…27ddf8
objdump -d -M intel client/WYD.exe > target/wyd-text.asm
grep -nE "(^|[ ,])0x26e([ ,]|$)" target/wyd-text.asm   # roteador + handler
grep -nE "(^|[ ,])0x270([ ,]|$)" target/wyd-text.asm   # site de envio
```

`target/` é ignorado no git; o `.asm` de 31 MB não entra no repositório.

## Fluxo de seleção no canal (`crates/wyd-server`)

| Passo | Pacote | O que o servidor faz |
|---|---|---|
| 1 | `_MSG_AccountLogin` | INITCODE → frame → checksum `Strict` → gate de versão → cerca de sessão → `CNFAccountLogin` (só cabeçalho) |
| 2 | `_MSG_CreateCharacter` | valida slot/classe/nome, exige conta em `accounts`, grava em `characters` → `CNFNewCharacter` ou `NewCharacterFail` |
| 3 | `_MSG_CharacterLogin` | resolve conta+slot, loga `stage=in_game` → `CNFCharacterLogin` ou `CharacterLoginFail` |
| 2b | `_MSG_CreateCharacter` com o slot ocupado | aloca o **primeiro slot livre** de 0–3 (nunca sobrescreve); conta cheia → `NewCharacterFail` |
| 4 | `_MSG_DeleteCharacter` | exige nome igual ao do slot, soft-delete → `CNFDeleteCharacter` com o `STRUCT_SELCHAR` do roster restante |
| 4b | `_MSG_CharacterLogout` (`0x0215`) | responde `0x0116`, sai do mundo, salva e libera a sessão **na hora** (fechar a janela não espera timeout); queda sem pacote libera igual (`reason=socket`). Prova: `tests/sistema/selchar.rs::logout_explicito_fecha_a_conexao_e_libera_a_sessao` |
| 4c | `_MSG_MessageWhisper` (`0x0334`, canal do `/getout`) | **tolerado** com `event=citizen action=getout`: o corpo não tem layout provado, então nada é lido nem executado e a sessão continua (invariante 5) |
| 5 | qualquer outro | **tolerado** (`result=tolerated`), sessão continua — antes o primeiro pacote encerrava a conexão |

Só `Size` inválido ou falha de I/O encerram a conexão. A senha de 12 bytes do
`_MSG_DeleteCharacter` não é conferida nesta entrega (pertence à autenticação de conta, que
ainda não existe); a remoção é do dono da sessão e o log grava `password_checked=false`.

## Regras

1. Golden gerado nunca vira prova de comportamento do cliente — só de codec.
2. Divergência entre golden gerado e captura real = o servidor está errado, nunca o contrário.
3. Layout com `UNVERIFIED` não entra em fixture tipada: entra como esquema em texto, sem bytes.4. Tamanho de `0x010A`/`0x0114` só é fato depois da captura (o previsto acima é a
expectativa a confirmar, não o contrato).
5. `0x0364`/`0x0165` têm layout **documentado** (não capturado): `MSG_CreateMob` = `_MSG` (12 B)
   + `PosX/PosY`+12, `MobID`+16, `MobName[16]`+18, `Equip[16]`+34, `Affect[32]`+66,
   `Guild`+130, `GuildMemberType`+132, `Unknow[3]`+133, `STRUCT_SCORE`+136, `CreateType`+184,
   `AnctCode[16]`+186, `Tab[26]`+202, `Hold`+228 (232 B), `HEADER.ID = ESCENE_FIELD` (30000);
   `MSG_RemoveMob` = `_MSG` + `RemoveType`+12 (16 B), `HEADER.ID` = id da criatura
   (`RemoveType` 0 = fora de vista, 1 = morte, 2 = logout). Fonte: `Basedef.h` e o porte Go do
   mesmo alvo 7662 (`tmserver/internal/protocol/createmob.go`, `createMobSize = 232`),
   implementados em `crates/wyd-protocol/src/payload/scene.rs` (ADR-0025). **Provado no fio**
   por `crates/wyd-server/tests/sistema/cena_mobs.rs` contra o canal em execução: sessão autenticada
   (`INITCODE` → `_MSG_AccountLogin` → `_MSG_AccountSecure` → `_MSG_CharacterLogin`) recebeu
   **6100** `CreateMob` de 232 B (`HEADER.ID=30000`, ids únicos, nome e posição do snapshot) e
   o `_MSG_Action` do NPC respondeu `_MSG_MessagePanel` com o `display_name`; na troca de
   revisão o servidor emitiu `RemoveMob` (16 B) do NPC desligado e `CreateMob` de volta ao
   religá-lo, sem duplicata e sem mob órfão (`creates=6101 removes=1 ativos=6100`). O desenho
   desses bytes pelo cliente 7662 continua pendente da FASE-08-6.
