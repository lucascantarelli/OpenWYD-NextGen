# 06 — Catálogo de protocolos e pacotes

> **Fonte de verdade:** `crates/wyd-protocol/wire/{header,frame,opcode}.rs`,
> `crates/wyd-protocol/src/wire/generated/opcode_table.rs` (198 opcodes),
> `docs/arquitetura/contrato-dispatch-opcode.md` (63 handlers C++), `docs/cliente/pacotes.md`
> (desassemblage do 7662) e `docs/arquitetura/protocolo.md`.
> **Nada aqui foi inventado.** Onde o campo não é conhecido, está escrito **Desconhecido**.

## 1. Transporte (CPSock)

| Peça | Valor | Status |
|---|---|---|
| Porta do canal | **8281** TCP fixa | Confirmado |
| Status do canal | HTTP `GET http://IP:80/serv00.htm` | Confirmado |
| HEADER | **12 B**, little-endian | Confirmado |
| Layout do HEADER | `Size u16 @0`, `KeyWord u8 @2`, `CheckSum u8 @3`, `Type u16 @4`, `ID u16 @6`, `Tick u32 @8` | Offsets de campo: **Fortemente indicado** (offsets declarados UNVERIFIED até captura) |
| `INITCODE` | `0x1F11F311` | Confirmado (bytes `11 f3 11 1f` em `.data`) |
| Tabela `pKeyWord` | 512 B, SHA-256 `e47996fe5e92de5d…` | Confirmado (bloco byte-idêntico) |
| Transformação | mod-4 sobre `[4, Size)`, índice `pKeyWord[rst*2+1]` | Confirmado por paridade de tabela |
| Faixa de `Size` | `[12, 8192]` | Confirmado no crate; teto 8192 é candidato no cliente |
| `CheckSum` | `Sum2 - Sum1`; **não rejeitante** | Confirmado (3 sites em `EB` no disco) |
| Versão enviada | **12000** | Confirmado em runtime |
| Validação de `Size` por opcode no cliente | **não existe** | Confirmado (decoder) |

**Regra prática:** o cliente não reclama de tamanho. Emitir frame curto demais não gera erro
visível — gera leitura de lixo. Por isso `require_ported` recusa opcode sem `declared_len`.

## 2. Flags de família (do `Basedef.h`)

| Flag | Valor | Direção | Entra no dispatch de cliente? |
|---|---:|---|---|
| `FLAG_CLIENT2GAME` | `0x0200` | C→S | **sim** |
| `FLAG_GAME2CLIENT` | `0x0100` | S→C | não (saída) |
| `FLAG_GAME2DB` | `0x0800` | jogo→DB | **não** (porta 7514 no legado) |
| `FLAG_DB2GAME` | `0x0400` | DB→jogo | não |
| `FLAG_NEW` | `0x4000` | ranking | não |

Valor de fio = base + flag (`_MSG_AccountLogin = 13 | 0x0200 = 0x020D`).

## 3. Catálogo — identidade e handler

| ID | Nome | Direção | Tamanho | Campos conhecidos | Handler legado | Handler Rust | Cliente | Confiança |
|---|---|---|---:|---|---|---|---|---|
| PKT-0x020D | AccountLogin | C→S | 116 | id, senha, ClientVersion | `_MSG_AccountLogin.cpp` | `cpsock::handshake` | envia | Confirmado |
| PKT-0x010A | CNFAccountLogin | S→C | 2008 (corpo 1996 nosso) | roster/selchar | `_MSG_AccountLogin.cpp` | `cpsock::reply_with_body` | recebe | Fortemente indicado |
| PKT-0x0FDE | AccountSecure | C→S | variável (`change_numeric` 1/2) | PIN, modo | `_MSG_AccountSecure` | `cpsock::authorize_pin` | envia | Confirmado |
| PKT-0x0FDF | AccountSecureFail | S→C | header | motivo | idem | `secure_pin` recusa | recebe | Confirmado |
| PKT-0x0117 | NewAccountFail | S→C | — | — | `_MSG_NewAccountFail` | **não usado** | **AUSENTE no binário** | Divergente |
| PKT-0x020F | CreateCharacter | C→S | 36 | nome, classe, slot | `_MSG_CreateCharacter.cpp` | `cpsock::create_character` | envia | Confirmado |
| PKT-0x0110 | CNFNewCharacter | S→C | corpo 844 | `STRUCT_SELCHAR` | idem | `character_body` | copia 840 B de `frame+0x10` | Confirmado |
| PKT-0x0211 | DeleteCharacter | C→S | — | nome/slot | `_MSG_DeleteCharacter.cpp` | `delete_character` | envia | Confirmado |
| PKT-0x0112 | CNFDeleteCharacter | S→C | frame 852 | `STRUCT_SELCHAR` | idem | mesma serialização | `memcpy` 840 B | Confirmado |
| PKT-0x0119/1a/1b | *LoginFail / *CharacterFail / *DeleteFail | S→C | header | motivo | vários | recusas | **presentes** | Confirmado |
| PKT-0x0213 | CharacterLogin | C→S | 20 (7662: **36**) | slot/id | `_MSG_CharacterLogin.cpp` | `login_character` | envia | Confirmado |
| PKT-0x0114 | CNFCharacterLogin | S→C | 1820 | `STRUCT_MOB` + cauda | idem | `personagens::StructMob` | recebe | Fortemente indicado |
| PKT-0x0215 | CharacterLogout | C→S | header | — | `_MSG_CharacterLogout.cpp` | `logout` | envia | Confirmado |
| PKT-0x0116 | CNFCharacterLogout | S→C | header | — | idem | ✅ | recebe | Confirmado |
| PKT-0x7530 | ESCENE_FIELD | S→C | variável | cena autoritativa (`HEADER.ID = 30000`) | `SendFunc.cpp` | `world::scene_snapshot` | recebe | Confirmado (servidor) |
| PKT-0x0364 | CreateMob | S→C | 232 | mob + posição + HP + nome + guilda | `SendFunc.cpp` | `send_create_mob` | recebe | Fortemente indicado |
| PKT-0x0165 | RemoveMob | S→C | 16 | id + `RemoveType` 0/1/2 | `SendFunc.cpp` | ✅ | recebe | Confirmado |
| PKT-0x0101 | MessagePanel | S→C | variável | texto | `SendFunc.cpp` | `send_message` | **única via de texto livre** | Confirmado |
| PKT-0x0102 | MessageBoxOk | S→C | — | — | idem | — | compara, cai no bloco 0x0101 | Confirmado |
| PKT-0x0336 | UpdateScore | S→C | — | HP/MP/EXP/ouro | `_MSG_UpdateScore` | `sendScoreSelf` | recebe | Fortemente indicado |
| PKT-0x036C | Action | C→S | 52 | posição, tipo, parâmetros | `_MSG_Action.cpp` | `handle_action` | envia | Confirmado |
| PKT-0x036A | Motion | C→S | — | direção/estado | `_MSG_Motion.cpp` | — | envia | Desconhecido |
| PKT-0x0367 | Attack | C→S | — | alvo | `_MSG_Attack.cpp` | `handle_attack` | envia | Confirmado |
| PKT-0x039D/0x039E | AttackOne/AttackTwo | C→S | — | alvo | `_MSG_Attack.cpp` | ✅ | envia | Confirmado (opcode) |
| PKT-0x0369 | NoViewMob | C→S | — | id | `_MSG_NoViewMob.cpp` | — | envia | Desconhecido |
| PKT-0x0373 | UseItem | C→S | — | slot/posição | `_MSG_UseItem.cpp` | `handle_use_item` → `refine` | envia | Confirmado |
| PKT-0x0182 | SendItem | S→C | — | slot + `STRUCT_ITEM` | `SendFunc.cpp` | ✅ | recebe | Fortemente indicado |
| PKT-0x026E | CreateItem | S→C | **30** packed | `GridX+0x0C`, `GridY+0x0E`, `ItemID+0x10`, `STRUCT_ITEM+0x12`, `Rotate+0x1A`, `State+0x1B`, `Height+0x1C` (i8), `Create+0x1D` | `_MSG_GetItem`/`DropItem` | ❌ | **recebe** | **Confirmado** (desassemblage) |
| PKT-0x016F | DecayItem | S→C | corpo 4 | `ItemID+0x0C`, unk `+0x0E` | `SendFunc.cpp` | ❌ | recebe | Fortemente indicado |
| PKT-0x0270 | GetItem | C→S | **28** | `Type+0x04`, `ID+0x06`, `DestType+0x0C = 1`, `DestPos+0x10` | `_MSG_GetItem.cpp` | ❌ | **envia** | **Confirmado** |
| PKT-0x0171 | CNFGetItem | S→C | corpo 16 | `DestType+0x0C`, `DestPos+0x10`, `STRUCT_ITEM+0x14` | idem | ❌ | recebe | Confirmado |
| PKT-0x0272 | DropItem | C→S | corpo 20 | `SourType`, `SourPos`, `Rotate`, `GridX/Y` | `_MSG_DropItem.cpp` | ❌ | envia | Confirmado |
| PKT-0x0175 | CNFDropItem | S→C | corpo 16 | `SourType`, `SourPos`, `Rotate`, `GridX/Y` | idem | ❌ | recebe | Confirmado |
| PKT-0x0374 | UpdateItem | C→S | — | slot de origem/destino | `_MSG_UpdateItem.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x027B | REQShopList | C→S | — | NPC | `_MSG_REQShopList.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x0379/0x037A | Buy / Sell | C→S | — | NPC, slot, qtd | `_MSG_Buy.cpp`/`_MSG_Sell.cpp` | parcial | envia | Confirmado (opcode) |
| PKT-0x0398 | ReqBuy | C→S | — | item/preço | `_MSG_ReqBuy.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x0383 | Trade | C→S | 156 (×154 no Go) | par, itens, ouro | `_MSG_Trade.cpp` | ❌ (layout candidato) | envia | **Divergente** |
| PKT-0x0376 | TradingItem | C→S | — | item da janela | `_MSG_TradingItem.cpp` | ❌ | envia | Desconhecido |
| PKT-0x0384 | QuitTrade | C→S | — | — | `_MSG_QuitTrade.cpp` | ❌ | envia | Confirmado |
| PKT-0x028B | Quest | C→S | 20 | NPC, opção | `_MSG_Quest.cpp` | `handle_quest` | envia | Confirmado |
| PKT-0x0333 | MessageChat | C→S | 140 | `text[128] @12` | `_MSG_MessageChat.cpp` | ❌ | envia (`0x4695b3`) | Confirmado |
| PKT-0x0334 | MessageWhisper | C→S | — | alvo + texto | `_MSG_MessageWhisper.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x0277 | ApplyBonus | C→S | — | atributo | `_MSG_ApplyBonus.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x0290 | ReqTeleport | C→S | — | destino | `_MSG_ReqTeleport.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x0291 | ChangeCity | C→S | — | cidade | `_MSG_ChangeCity.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x0399 | PKMode | C→S | — | modo | `_MSG_PKMode.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x039F | ReqRanking | C→S | — | tipo | `_MSG_ReqRanking.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x03D5/0x0E12 | InviteGuild / GuildAlly | C→S | — | guilda/alvo | `_MSG_InviteGuild.cpp`/`GuildAlly.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x0E0E | War | C→S | — | — | `_MSG_War.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x02C3…0x02E2 | Combine* (11 variantes) | C→S | — | itens, NPC | `_MSG_CombineItem*.cpp` | só `compose` interno | envia | Confirmado (opcode) |
| PKT-0x02E4 | DeleteItem | C→S | — | slot | `_MSG_DeleteItem.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x02CD | CapsuleInfo | C→S | — | — | `_MSG_CapsuleInfo.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x03CC | PutoutSeal | C→S | — | item | `_MSG_PutoutSeal.cpp` | ❌ | envia | Confirmado (opcode) |
| PKT-0x0289/0x028C/0x028E/0x028F | Restart / Deprivate / Challange / Confirm | C→S | — | — | vários | ❌ | envia | Desconhecido |
| PKT-0x0397/0x039A | SendAutoTrade / ReqTradeList | C→S | — | loja pessoal | `_MSG_SendAutoTrade.cpp` | ❌ | envia | Desconhecido |

## 4. Catálogo — cabeçalho, serialização, validação e erro

| ID | Cabeçalho / corpo | Serialização | Validações | Resposta esperada | Erros possíveis | Evidência |
|---|---|---|---|---|---|---|
| PKT-0x020D | header + 104 B | LE, strings NUL | tamanho, versão, senha | `0x010A` ou recusa | senha, versão, bloqueio | `handshake.rs`, `login.rs` |
| PKT-0x010A | header + corpo | `STRUCT_SELCHAR` | — | — | cliente ignora corpo inválido | `selchar.rs` |
| PKT-0x0FDE | header + PIN + modo | LE | 4–6 dígitos, tentativas | ok / `0x0FDF` | PIN errado, bloqueio 10 min | `pin_*.rs` |
| PKT-0x020F | header + 24 B | LE | nome, classe, slot, limite | `0x0110` ou `0x011a` | nome inválido, slot cheio | `personagens_crud.rs` |
| PKT-0x0213 | header + 24 B (7662) | LE | slot do dono | `0x0114` | slot inválido | `selchar.rs` |
| PKT-0x7530 | `ID=30000` | deltas de cena | teto por refresh | — | — | `world::tests` |
| PKT-0x0364 | 232 B | packed | id na faixa de criatura | — | cliente ignora id fora de faixa | `cena_mobs.rs` |
| PKT-0x026E | **packed**, sem padding | LE | — | — | leitura de lixo se `< 30` | `pacotes.md` §CreateItem |
| PKT-0x0270 | `memset(0x1c)` + campos | LE | `DestType=1` fixo | `0x0171` | — | `pacotes.md` §GetItem |
| PKT-0x0383 | 156 B candidato | packed/natural em disputa | — | confirmação | duplicação se errar | `trade-layout.mjs` |
| PKT-0x0333 | `text[128] @12` | CP1252 provável | tamanho, filtro | eco aos vizinhos | texto vazio | `pacotes.md` |
| Não catalogados S→C | — | — | — | — | — | **Desconhecido** |

## 5. Divergências de catálogo (não corrigidas nesta auditoria)

| ID | Divergência | Impacto | Ação |
|---|---|---|---|
| `DIV-02` | `opcode_table.rs` marca `CreateItem` como `Client2Game`; o binário prova S→C | contrato e dispatch errados para item no chão | rito próprio de tabela congelada |
| `DIV-01` | `0x0117 NewAccountFail` no C++ e no catálogo, **ausente** no cliente | mensagem de senha errada invisível | usar `0x0101` |
| `DIV-04` | `MSG_Trade` 156 B (natural) × 154 B (Go `pack(1)`) | trade não pode ser implementado às cegas | captura |
| `DIV-03` | `0x0213` 20 B no `Basedef.h` × **36 B** no 7662 | leitura de 16 B a mais | já tratado no crate |
| — | `_COMBINEITEMODIN2` (`0x02E2`) sem rota no Go | opcode “órfão” no porte anterior | decidir na implementação |

## 6. O que falta para o catálogo ser “completo”

1. **Sentido S→C**: `STRUCT_MOB` (1820 B), corpo de `ESCENE_FIELD`, `UpdateScore`,
   `CNFGetItem`, lista de loja. Sem captura, é `Desconhecido` — não é lacuna de esforço, é de
   observação.
2. **Semântica de `Motion`/`NoViewMob`/`TradingItem`**: opcode confirmado, corpo não.
3. **Família `Combine*`**: 11 opcodes, um resolvedor. Mapear qual NPC usa qual é necessário antes
   de implementar mix no fio.
4. **`ESCENE_FIELD` de resposta**: o cliente responde à cena? Não há evidência. `UNK-06`.

## 7. Regras de ouro para quem for implementar

1. **Nunca** emitir opcode que o cliente não compara (91 de 198) esperando reação.
2. **Nunca** preencher struct sem `declared_len` provado; `require_ported` existe para isso.
3. **Sempre** ofuscar a partir do offset 4 — inclusive pacotes só de header.
4. **Nunca** tornar o checksum rejeitante: o build alvo já vem com os 3 sites em `EB`.
5. **Nunca** assumir que o cliente valida `Size`: frame curto = lixo lido, não erro.
