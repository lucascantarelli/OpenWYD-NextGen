# Protocolo CPSock do WYD (documento normativo de fio)

> Build de referência: cliente 7662. Fontes canônicas: `protocol-spec.md` §1,
> `Basedef.h` (198 constantes `_MSG_*`), `CPSock.cpp`. Implementação:
> `crates/wyd-protocol/` (árvore §3 de `estrutura-e-padroes.md`).
> O que está aqui é contrato; o que não está é UNVERIFIED (§6) e não vira tipo.

## 1. HEADER (12 bytes, little-endian, sem padding)

| Offset | Campo | Tipo | Regra |
|---:|---|---|---|
| 0 | `Size` | u16 | tamanho total do pacote; válido em [12, 8192] (`CPSock.h:38`, `CPSock.cpp:397-406`) |
| 2 | `KeyWord` | u8 | índice 0–255 da semente em `pKeyWord` (gravado pelo emissor) |
| 3 | `CheckSum` | u8 | `soma(cifrado) − soma(claro)` com wrap (gravado pelo emissor) |
| 4 | `Type` | u16 | valor de fio = base + bits de direção (§2) |
| 6 | `ID` | u16 | índice de conexão/jogador (`pUser[]`) |
| 8 | `ClientTick` | u32 | timestamp do cliente; `235543242` (`SKIPCHECKTICK`) nunca é válido |

## 2. Enquadramento e direção

- `INITCODE = 0x1F11F311`: primeiros 4 bytes de toda conexão nova (`CPSock.h:40`).
- Corte por `Size`: `validate_size` rejeita fora de [12, 8192]; `split_frame`
  exige o frame inteiro antes de qualquer parse.
- Bits de direção no `Type` (`Basedef.h`): `Game2Client 0x0100` (J→C),
  `Client2Game 0x0200` (C→J), `Db2Game 0x0400` (D→J), `Game2Db 0x0800` (J→D),
  `Db2Np 0x1000` (D→N), `Np2Db 0x2000` (N→D), `New 0x4000` (novo).
  `Opcode::wire()` devolve o valor exato que trafega; `Flow` documenta a direção.

## 3. Ofuscação e checksum

- `Size`, `KeyWord`, `CheckSum` (offsets 0..4) ficam em claro; a transformação
  começa no offset 4 e vai até `Size` (`CPSock.cpp:428-453`, `:556-584`).
- Semente: `KEY_TABLE[key_word * 2]`; byte de transformação por posição:
  `KEY_TABLE[rst * 2 + 1]` (`pKeyWord[512] = { // 7.xx keys }`).
- Aritmética em 8 bits com wrap (equivale ao `unsigned char` do C); não é
  cifragem — a chave é pública e o resultado precisa ser idêntico ao legado.
- Limite conhecido: cada byte contribui com `±f(trans)` para a diferença, logo
  o checksum é função da chave, não do conteúdo — tamper de 1 byte do corpo o
  preserva. Ele detecta corrupção do campo/emissor divergente, não adulteração
  de conteúdo (prova em `tests/roundtrip.rs`: `tampered_body_fails_strict_checksum`).
- **Decisão da stack nova**: entrada de rede usa política `Strict` (divergência
  é erro tipado e rejeita o pacote); `Tolerant` só em captura/diagnóstico/RE.
  O legado tolerava mismatch (`CPSock.cpp:457-466`) — não repetimos isso no
  caminho de produção.

## 4. Catálogo de opcodes (gerado de `Basedef.h`, 198 constantes)

<!-- GERADO:catalogo-opcodes (tools/protocol/gen-protocol.mjs de Basedef.h) -->
| Constante | Fio | Direção |
|---|---|---|
| `_MSG_MessagePanel` | 0x0101 | J→C |
| `_MSG_MessageBoxOk` | 0x0102 | J→C |
| `_MSG_CNFAccountLogin` | 0x010A | J→C |
| `_MSG_CNFNewCharacter` | 0x0110 | J→C |
| `_MSG_CNFDeleteCharacter` | 0x0112 | J→C |
| `_MSG_CNFCharacterLogin` | 0x0114 | J→C |
| `_MSG_CNFCharacterLogout` | 0x0116 | J→C |
| `_MSG_NewAccountFail` | 0x0117 | J→C |
| `_MSG_CharacterLoginFail` | 0x0119 | J→C |
| `_MSG_NewCharacterFail` | 0x011A | J→C |
| `_MSG_DeleteCharacterFail` | 0x011B | J→C |
| `_MSG_AlreadyPlaying` | 0x011C | J→C |
| `_MSG_StillPlaying` | 0x011D | J→C |
| `_MSG_RemoveMob` | 0x0165 | J→C |
| `_MSG_PKInfo` | 0x0166 | J→C |
| `_MSG_DecayItem` | 0x016F | J→C |
| `_MSG_CNFGetItem` | 0x0171 | J→C |
| `_MSG_CNFDropItem` | 0x0175 | J→C |
| `_MSG_ShopList` | 0x017C | J→C |
| `_MSG_SetHpMp` | 0x0181 | J→C |
| `_MSG_SendItem` | 0x0182 | J→C |
| `_MSG_UpdateCarry` | 0x0185 | J→C |
| `_MSG_SetHpDam` | 0x018A | J→C |
| `_MSG_UpdateWeather` | 0x018B | J→C |
| `_MSG_ReqChallange` | 0x018D | J→C |
| `_MSG_SetClan` | 0x0193 | J→C |
| `_MSG_CloseShop` | 0x0196 | J→C |
| `_MSG_AccountLogin` | 0x020D | C→J |
| `_MSG_CreateCharacter` | 0x020F | C→J |
| `_MSG_DeleteCharacter` | 0x0211 | C→J |
| `_MSG_CharacterLogin` | 0x0213 | C→J |
| `_MSG_CharacterLogout` | 0x0215 | C→J |
| `_MSG_CreateItem` | 0x026E | C→J |
| `_MSG_GetItem` | 0x0270 | C→J |
| `_MSG_DropItem` | 0x0272 | C→J |
| `_MSG_ApplyBonus` | 0x0277 | C→J |
| `_MSG_REQShopList` | 0x027B | C→J |
| `_MSG_Restart` | 0x0289 | C→J |
| `_MSG_Quest` | 0x028B | C→J |
| `_MSG_Deprivate` | 0x028C | C→J |
| `_MSG_Challange` | 0x028E | C→J |
| `_MSG_ChallangeConfirm` | 0x028F | C→J |
| `_MSG_ReqTeleport` | 0x0290 | C→J |
| `_MSG_ChangeCity` | 0x0291 | C→J |
| `_MSG_SetHpMode` | 0x0292 | C→J |
| `_MSG_CombineItemLindy` | 0x02C3 | C→J |
| `_MSG_CombineItemShany` | 0x02C4 | C→J |
| `_MSG_CapsuleInfo` | 0x02CD | C→J |
| `_MSG_CombineItemOdin` | 0x02D2 | C→J |
| `_MSG_CombineItemEhre` | 0x02D3 | C→J |
| `_MSG_CombineItemExtracao` | 0x02D4 | C→J |
| `_MSG_CombineItemLoki` | 0x02D5 | C→J |
| `_MSG_CombineItemAlquimia` | 0x02E1 | C→J |
| `_MSG_CombineItemOdin2` | 0x02E2 | C→J |
| `_MSG_DeleteItem` | 0x02E4 | C→J |
| `_MSG_SplitItem` | 0x02E5 | C→J |
| `_MSG_CombineDedekinto` | 0x02E6 | C→J |
| `_MSG_CombineDedekinto2` | 0x02E9 | C→J |
| `_MSG_MessageChat` | 0x0333 | J→C, C→J |
| `_MSG_MessageWhisper` | 0x0334 | J→C, C→J |
| `_MSG_UpdateScore` | 0x0336 | J→C, C→J |
| `_MSG_UpdateEtc` | 0x0337 | J→C, C→J |
| `_MSG_CNFMobKill` | 0x0338 | J→C, C→J |
| `_MSG_UpdateCargoCoin` | 0x0339 | J→C, C→J |
| `_MSG_DisableExpMsg` | 0x034D | J→C |
| `_MSG_CreateMobTrade` | 0x0363 | J→C, C→J |
| `_MSG_CreateMob` | 0x0364 | J→C, C→J |
| `_MSG_Action2` | 0x0366 | J→C, C→J |
| `_MSG_Attack` | 0x0367 | J→C, C→J |
| `_MSG_Action3` | 0x0368 | J→C, C→J |
| `_MSG_NoViewMob` | 0x0369 | J→C, C→J |
| `_MSG_Motion` | 0x036A | J→C, C→J |
| `_MSG_UpdateEquip` | 0x036B | J→C, C→J |
| `_MSG_Action` | 0x036C | J→C, C→J |
| `_MSG_UseItem` | 0x0373 | J→C, C→J |
| `_MSG_UpdateItem` | 0x0374 | J→C, C→J |
| `_MSG_TradingItem` | 0x0376 | J→C, C→J |
| `_MSG_SetShortSkill` | 0x0378 | J→C, C→J |
| `_MSG_Buy` | 0x0379 | J→C, C→J |
| `_MSG_Sell` | 0x037A | J→C, C→J |
| `_MSG_CNFAddParty` | 0x037D | J→C, C→J |
| `_MSG_RemoveParty` | 0x037E | J→C, C→J |
| `_MSG_SendReqParty` | 0x037F | J→C, C→J |
| `_MSG_Trade` | 0x0383 | J→C, C→J |
| `_MSG_QuitTrade` | 0x0384 | J→C, C→J |
| `_MSG_CNFCheck` | 0x0386 | J→C, C→J |
| `_MSG_Withdraw` | 0x0387 | J→C, C→J |
| `_MSG_Deposit` | 0x0388 | J→C, C→J |
| `_MSG_SendAutoTrade` | 0x0397 | J→C, C→J |
| `_MSG_ReqBuy` | 0x0398 | J→C, C→J |
| `_MSG_PKMode` | 0x0399 | J→C, C→J |
| `_MSG_ReqTradeList` | 0x039A | J→C, C→J |
| `_MSG_ItemSold` | 0x039B | J→C, C→J |
| `_MSG_AttackOne` | 0x039D | J→C, C→J |
| `_MSG_AttackTwo` | 0x039E | J→C, C→J |
| `_MSG_ReqRanking` | 0x039F | J→C, C→J |
| `_MSG_Ping` | 0x03A0 | J→C, C→J |
| `_MSG_StartTime` | 0x03A1 | J→C, C→J |
| `_MSG_EnvEffect` | 0x03A2 | J→C, C→J |
| `_MSG_SoundEffect` | 0x03A3 | J→C, C→J |
| `_MSG_GuildDisable` | 0x03A4 | J→C, C→J |
| `_MSG_GuildBoard` | 0x03A5 | J→C, C→J |
| `_MSG_CombineItem` | 0x03A6 | J→C, C→J |
| `_MSG_CombineComplete` | 0x03A7 | J→C, C→J |
| `_MSG_SendWarInfo` | 0x03A8 | J→C, C→J |
| `_MSG_AcceptParty` | 0x03AB | J→C, C→J |
| `_MSG_SendCastleState` | 0x03AC | J→C, C→J |
| `_MSG_SendCastleState2` | 0x03AD | J→C, C→J |
| `_MSG_MobLeft` | 0x03B0 | J→C, C→J |
| `_MSG_SendArchEffect` | 0x03B4 | J→C, C→J |
| `_MSG_CombineItemAilyn` | 0x03B5 | J→C, C→J |
| `_MSG_SendAffect` | 0x03B9 | J→C, C→J |
| `_MSG_CombineItemAgatha` | 0x03BA | J→C, C→J |
| `_MSG_MobCount` | 0x03BB | J→C, C→J |
| `_MSG_CombineItemTiny` | 0x03C0 | J→C, C→J |
| `_MSG_PutoutSeal` | 0x03CC | J→C, C→J |
| `_MSG_InviteGuild` | 0x03D5 | J→C, C→J |
| `_MSG_DBMessagePanel` | 0x0401 | D→J |
| `_MSG_DBMessageBoxOk` | 0x0402 | D→J |
| `_MSG_DBSavingQuit` | 0x040A | D→J |
| `_MSG_DBCNFAccountLogOut` | 0x040B | D→J |
| `_MSG_DBCNFArchCharacterSucess` | 0x040E | D→J |
| `_MSG_DBCNFArchCharacterFail` | 0x040F | D→J |
| `_MSG_DBCNFCapsuleCharacterFail` | 0x0410 | D→J |
| `_MSG_DBCNFCapsuleCharacterFail2` | 0x0411 | D→J |
| `_MSG_DBCNFCapsuleSucess` | 0x0412 | D→J |
| `_MSG_DBClientMessage` | 0x0413 | D→J |
| `_MSG_DBCheckPrimaryAccount` | 0x0414 | D→J |
| `_MSG_DBCNFAccountLogin` | 0x0416 | D→J |
| `_MSG_DBCNFCharacterLogin` | 0x0417 | D→J |
| `_MSG_DBCNFNewCharacter` | 0x0418 | D→J |
| `_MSG_DBCNFDeleteCharacter` | 0x0419 | D→J |
| `_MSG_DBNewAccountFail` | 0x041A | D→J |
| `_MSG_DBCharacterLoginFail` | 0x041C | D→J |
| `_MSG_DBNewCharacterFail` | 0x041D | D→J |
| `_MSG_DBDeleteCharacterFail` | 0x041E | D→J |
| `_MSG_DBAlreadyPlaying` | 0x041F | D→J |
| `_MSG_DBStillPlaying` | 0x0420 | D→J |
| `_MSG_DBAccountLoginFail_Account` | 0x0421 | D→J |
| `_MSG_DBAccountLoginFail_Pass` | 0x0422 | D→J |
| `_MSG_DBSetIndex` | 0x0423 | D→J |
| `_MSG_DBAccountLoginFail_Block` | 0x0424 | D→J |
| `_MSG_DBAccountLoginFail_Disable` | 0x0425 | D→J |
| `_MSG_DBOnlyOncePerDay` | 0x0426 | D→J |
| `_MSG_GuildReport` | 0x0427 | D→J |
| `_MSG_DBCNFServerChange` | 0x052A | J→C, D→J |
| `_MSG_DBServerSend1` | 0x052B | J→C, D→J |
| `_MSG_TransperCharacter` | 0x07A9 | J→C, C→J, D→J |
| `_MSG_DBNewAccount` | 0x0801 | J→D |
| `_MSG_DBCreateCharacter` | 0x0802 | J→D |
| `_MSG_DBAccountLogin` | 0x0803 | J→D |
| `_MSG_DBCharacterLogin` | 0x0804 | J→D |
| `_MSG_DBNoNeedSave` | 0x0805 | J→D |
| `_MSG_SavingQuit` | 0x0806 | J→D |
| `_MSG_DBSaveMob` | 0x0807 | J→D |
| `_MSG_DBDeleteCharacter` | 0x0809 | J→D |
| `_MSG_DBUpdateSapphire` | 0x080A | J→D |
| `_MSG_MessageDBRecord` | 0x080C | J→D |
| `_MSG_GuildZoneReport` | 0x080D | J→D |
| `_MSG_DBServerChange` | 0x0814 | J→D |
| `_MSG_DBItemDayLog` | 0x0815 | J→D |
| `_MSG_DBActivatePinCode` | 0x0816 | J→D |
| `_MSG_DBPrimaryAccount` | 0x0817 | J→D |
| `_MSG_DBCreateArchCharacter` | 0x081F | J→D |
| `_MSG_MasterGriff` | 0x0AD9 | C→J, J→D |
| `_MSG_MessageDBImple` | 0x0C0B | D→J, J→D |
| `_MSG_DBSendItem` | 0x0C0F | D→J, J→D |
| `_MSG_DBSendDonate` | 0x0C11 | D→J, J→D |
| `_MSG_DBCapsuleInfo` | 0x0C3C | D→J, J→D |
| `_MSG_DBPutInCapsule` | 0x0C3D | D→J, J→D |
| `_MSG_DBOutCapsule` | 0x0C3E | D→J, J→D |
| `_MSG_MagicTrumpet` | 0x0D1D | J→C, D→J, J→D |
| `_MSG_DBNotice` | 0x0D1E | J→C, D→J, J→D |
| `_MSG_CNFDBCapsuleInfo` | 0x0D1F | J→C, D→J, J→D |
| `_MSG_War` | 0x0E0E | C→J, D→J, J→D |
| `_MSG_GuildAlly` | 0x0E12 | C→J, D→J, J→D |
| `_MSG_GuildInfo` | 0x0E13 | C→J, D→J, J→D |
| `_MSG_ReqTransper` | 0x0FAA | J→C, C→J, D→J, J→D |
| `_MSG_AccountSecure` | 0x0FDE | J→C, C→J, D→J, J→D |
| `_MSG_AccountSecureFail` | 0x0FDF | J→C, C→J, D→J, J→D |
| `_MSG_NPReqIDPASS` | 0x1001 | D→N |
| `_MSG_NPNotFound` | 0x1004 | D→N |
| `_MSG_NPAccountInfo` | 0x1005 | D→N |
| `_MSG_NPAppeal` | 0x1C10 | D→J, J→D, D→N |
| `_MSG_NPIDPASS` | 0x2002 | N→D |
| `_MSG_NPReqAccount` | 0x2003 | N→D |
| `_MSG_NPReqSaveAccount` | 0x2006 | N→D |
| `_MSG_NPDisable` | 0x3007 | D→N, N→D |
| `_MSG_NPEnable` | 0x3008 | D→N, N→D |
| `_MSG_NPState` | 0x300A | D→N, N→D |
| `_MSG_NPCreateCharacter` | 0x300B | D→N, N→D |
| `_MSG_NPCreateCharacter_Reply` | 0x300C | D→N, N→D |
| `_MSG_NPDonate` | 0x300D | D→N, N→D |
| `_MSG_NPNotice` | 0x3409 | D→J, D→N, N→D |
| `_MSG_GrindRankingData` | 0x4501 | J→C, D→J, novo |
| `_MSG_UpdateExpRanking` | 0x4D02 | J→C, D→J, J→D, novo |
| `_MSG_Exp_Msg_Panel_` | 0x5000 | — |
| `_MSG_BATTLEWOTE` | 0x5001 | — |
<!-- /GERADO:catalogo-opcodes -->

## 5. Payloads portados (offsets explícitos, `protocol-spec.md` §3.5)

| Tipo | Opcode (fio) | Tamanho | Corpo (offsets) |
|---|---|---:|---|
| `AccountLogin` | 0x020D | 116 | senha[12]+12, conta[16]+24, zero[52]+40, versão i32+92, save i32+96, adapter[4×i32]+100 |
| `CreateCharacter` | 0x020F | 36 | slot+12, nome[16]+16, classe+32 |
| `DeleteCharacter` | 0x0211 | 44 | slot+12, nome[16]+16, senha[12]+32 |
| `CharacterLogin` | 0x0213 | 20 | slot+12, force+16 |
| `Action` | 0x036C | 52 | pos(2×i16)+12, effect+16, speed+20, route[24]+24, alvo(2×i16)+48 |
| `UseItem` | 0x0373 | 34 | origem(tipo/pos)+12, destino(tipo/pos)+20, grade(2×u16)+28, warp+32 |
| `TradingItem` | 0x0376 | 20 | lugares/slots(4×u8)+12, warp+16 |
| `Buy` | 0x0379 | 22 | npc+12, pos inv.(2×i16)+14, moeda+18 |
| `Sell` | 0x037A | 18 | npc+12, tipo/pos(2×i16)+14 |
| `ApplyBonus` | 0x0277 | 18 | tipo/detalhe(2×i16)+12, alvo+16 |
| `AccountSecure` | 0x0FDE | 32 | token[6]+12, desconhecido[10]+18, troca i32+28 |
| `MessagePanel` | 0x0101 | 140 | texto[128]+12 |
| `UpdateWeather` | 0x018B | 16 | clima i32+12 |
| `Attack` (13 alvos) | 0x0367 | 164 | corpo 60 + 13×`STRUCT_DAM`(8) |
| `AttackOne` | 0x039D | 68 | corpo 60 + 1×`STRUCT_DAM` |
| `AttackTwo` | 0x039E | 76 | corpo 60 + 2×`STRUCT_DAM` |

`STRUCT_DAM` = `target_id` i32 + `damage` i32. Strings são `char[N]` em bytes
crus (nunca UTF-8 assumido); leitura até o primeiro `NUL`.

## 6. UNVERIFIED (não vira tipo até a captura real, FASE-08-2)

- `MSG_Trade` (troca entre jogadores) — layout não documentado.
- `MSG_SetHpMp` — layout não documentado.
- `_AUTH_GAME` — layout não documentado.

## 7. Fontes por seção

| Seção | Fonte |
|---|---|
| §1 HEADER | `protocol-spec.md` §1.1, `CPSock.h:38` |
| §2 enquadramento/direção | `CPSock.h:40`, `CPSock.cpp:397-406`, `Basedef.h` (flags) |
| §3 ofuscação | `CPSock.cpp:428-466`, `:556-584`, `pKeyWord[512]` |
| §4 catálogo | `Basedef.h` via `tools/protocol/gen-protocol.mjs` |
| §5 payloads | `protocol-spec.md` §3.5, `data-formats.md` |
