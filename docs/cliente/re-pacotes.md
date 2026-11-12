# RE do cliente — pacotes e protocolo

> Gerado por `node apps/tools/bin/wyd-tools.mjs pacotes` — não editar à mão.
> Catálogo e handlers vêm do código Rust; capturas de `tmp/` entram como evidência e são marcadas quando não decodificadas.

## Resumo executivo

- **198** opcodes no catálogo (`opcode_table.rs`).
- **34** com referência de handler em `crates/wyd-server`; **164** sem referência.
- **1** com estrutura de campos declarada em `crates/wyd-protocol/src/payload`.
- **1** com opcode confirmado por captura em `tmp/`.
- **4** arquivo(s) de captura em `tmp/` (3 frames declarados).

## Opcodes

Handler = qualquer referência `Opcode::X` em `crates/wyd-server` (inclui o envio de respostas CNF, não só o `match` de entrada).

| Wire | Opcode | Direção | Handler no servidor | Estrutura (payload) | Captura |
| --- | --- | --- | --- | --- | --- |
| 0x0101 | MessagePanel | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0102 | MessageBoxOk | S→C | não encontrado | — | — |
| 0x010a | CNFAccountLogin | S→C | crates/wyd-server/src/cpsock/handshake.rs | — | tmp/captura/diff-selchar.json |
| 0x0110 | CNFNewCharacter | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0112 | CNFDeleteCharacter | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0114 | CNFCharacterLogin | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0116 | CNFCharacterLogout | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0117 | NewAccountFail | S→C | crates/wyd-server/src/cpsock/handshake.rs, crates/wyd-server/src/cpsock/mod.rs | — | — |
| 0x0119 | CharacterLoginFail | S→C | crates/wyd-server/src/cpsock/action.rs, crates/wyd-server/src/cpsock/mod.rs | — | — |
| 0x011a | NewCharacterFail | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x011b | DeleteCharacterFail | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x011c | AlreadyPlaying | S→C | não encontrado | — | — |
| 0x011d | StillPlaying | S→C | não encontrado | — | — |
| 0x0165 | RemoveMob | S→C | crates/wyd-server/src/cpsock/action.rs, crates/wyd-server/src/cpsock/mod.rs | — | — |
| 0x0166 | PKInfo | S→C | não encontrado | — | — |
| 0x016f | DecayItem | S→C | não encontrado | — | — |
| 0x0171 | CNFGetItem | S→C | não encontrado | — | — |
| 0x0175 | CNFDropItem | S→C | não encontrado | — | — |
| 0x017c | ShopList | S→C | crates/wyd-server/src/cpsock/action.rs | ShopList (236 B) | — |
| 0x0181 | SetHpMp | S→C | não encontrado | — | — |
| 0x0182 | SendItem | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0185 | UpdateCarry | S→C | não encontrado | — | — |
| 0x018a | SetHpDam | S→C | não encontrado | — | — |
| 0x018b | UpdateWeather | S→C | não encontrado | — | — |
| 0x018d | ReqChallange | S→C | não encontrado | — | — |
| 0x0193 | SetClan | S→C | não encontrado | — | — |
| 0x0196 | CloseShop | S→C | não encontrado | — | — |
| 0x020d | AccountLogin | C→S | crates/wyd-server/src/cpsock/handshake.rs | — | — |
| 0x020f | CreateCharacter | C→S | crates/wyd-server/src/cpsock/action.rs, crates/wyd-server/src/cpsock/login.rs | — | — |
| 0x0211 | DeleteCharacter | C→S | crates/wyd-server/src/cpsock/action.rs, crates/wyd-server/src/cpsock/login.rs | — | — |
| 0x0213 | CharacterLogin | C→S | crates/wyd-server/src/cpsock/action.rs, crates/wyd-server/src/cpsock/login.rs | — | — |
| 0x0215 | CharacterLogout | C→S | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x026e | CreateItem | C→S | não encontrado | — | — |
| 0x0270 | GetItem | C→S | não encontrado | — | — |
| 0x0272 | DropItem | C→S | não encontrado | — | — |
| 0x0277 | ApplyBonus | C→S | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x027b | REQShopList | C→S | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0289 | Restart | C→S | não encontrado | — | — |
| 0x028b | Quest | C→S | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x028c | Deprivate | C→S | não encontrado | — | — |
| 0x028e | Challange | C→S | não encontrado | — | — |
| 0x028f | ChallangeConfirm | C→S | não encontrado | — | — |
| 0x0290 | ReqTeleport | C→S | não encontrado | — | — |
| 0x0291 | ChangeCity | C→S | não encontrado | — | — |
| 0x0292 | SetHpMode | C→S | não encontrado | — | — |
| 0x02c3 | CombineItemLindy | C→S | não encontrado | — | — |
| 0x02c4 | CombineItemShany | C→S | não encontrado | — | — |
| 0x02cd | CapsuleInfo | C→S | não encontrado | — | — |
| 0x02d2 | CombineItemOdin | C→S | não encontrado | — | — |
| 0x02d3 | CombineItemEhre | C→S | não encontrado | — | — |
| 0x02d4 | CombineItemExtracao | C→S | não encontrado | — | — |
| 0x02d5 | CombineItemLoki | C→S | não encontrado | — | — |
| 0x02e1 | CombineItemAlquimia | C→S | não encontrado | — | — |
| 0x02e2 | CombineItemOdin2 | C→S | não encontrado | — | — |
| 0x02e4 | DeleteItem | C→S | não encontrado | — | — |
| 0x02e5 | SplitItem | C→S | não encontrado | — | — |
| 0x02e6 | CombineDedekinto | C→S | não encontrado | — | — |
| 0x02e9 | CombineDedekinto2 | C→S | não encontrado | — | — |
| 0x0333 | MessageChat | S→C | crates/wyd-server/src/cpsock/action.rs, crates/wyd-server/src/cpsock/mod.rs | — | — |
| 0x0334 | MessageWhisper | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0336 | UpdateScore | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0337 | UpdateEtc | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0338 | CNFMobKill | S→C | não encontrado | — | — |
| 0x0339 | UpdateCargoCoin | S→C | não encontrado | — | — |
| 0x034d | DisableExpMsg | S→C | não encontrado | — | — |
| 0x0363 | CreateMobTrade | S→C | não encontrado | — | — |
| 0x0364 | CreateMob | S→C | crates/wyd-server/src/cpsock/action.rs, crates/wyd-server/src/cpsock/mod.rs | — | — |
| 0x0366 | Action2 | S→C | não encontrado | — | — |
| 0x0367 | Attack | S→C | crates/wyd-server/src/cpsock/action.rs, crates/wyd-server/src/gameplay/service.rs | — | — |
| 0x0368 | Action3 | S→C | não encontrado | — | — |
| 0x0369 | NoViewMob | S→C | não encontrado | — | — |
| 0x036a | Motion | S→C | não encontrado | — | — |
| 0x036b | UpdateEquip | S→C | não encontrado | — | — |
| 0x036c | Action | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x0373 | UseItem | S→C | crates/wyd-server/src/cpsock/action.rs, crates/wyd-server/src/gameplay/service.rs | — | — |
| 0x0374 | UpdateItem | S→C | não encontrado | — | — |
| 0x0376 | TradingItem | S→C | não encontrado | — | — |
| 0x0378 | SetShortSkill | S→C | não encontrado | — | — |
| 0x0379 | Buy | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x037a | Sell | S→C | não encontrado | — | — |
| 0x037d | CNFAddParty | S→C | não encontrado | — | — |
| 0x037e | RemoveParty | S→C | não encontrado | — | — |
| 0x037f | SendReqParty | S→C | não encontrado | — | — |
| 0x0383 | Trade | S→C | não encontrado | — | — |
| 0x0384 | QuitTrade | S→C | não encontrado | — | — |
| 0x0386 | CNFCheck | S→C | não encontrado | — | — |
| 0x0387 | Withdraw | S→C | não encontrado | — | — |
| 0x0388 | Deposit | S→C | não encontrado | — | — |
| 0x0397 | SendAutoTrade | S→C | não encontrado | — | — |
| 0x0398 | ReqBuy | S→C | não encontrado | — | — |
| 0x0399 | PKMode | S→C | não encontrado | — | — |
| 0x039a | ReqTradeList | S→C | não encontrado | — | — |
| 0x039b | ItemSold | S→C | não encontrado | — | — |
| 0x039d | AttackOne | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x039e | AttackTwo | S→C | crates/wyd-server/src/cpsock/action.rs | — | — |
| 0x039f | ReqRanking | S→C | não encontrado | — | — |
| 0x03a0 | Ping | S→C | não encontrado | — | — |
| 0x03a1 | StartTime | S→C | não encontrado | — | — |
| 0x03a2 | EnvEffect | S→C | não encontrado | — | — |
| 0x03a3 | SoundEffect | S→C | não encontrado | — | — |
| 0x03a4 | GuildDisable | S→C | não encontrado | — | — |
| 0x03a5 | GuildBoard | S→C | não encontrado | — | — |
| 0x03a6 | CombineItem | S→C | não encontrado | — | — |
| 0x03a7 | CombineComplete | S→C | não encontrado | — | — |
| 0x03a8 | SendWarInfo | S→C | não encontrado | — | — |
| 0x03ab | AcceptParty | S→C | não encontrado | — | — |
| 0x03ac | SendCastleState | S→C | não encontrado | — | — |
| 0x03ad | SendCastleState2 | S→C | não encontrado | — | — |
| 0x03b0 | MobLeft | S→C | não encontrado | — | — |
| 0x03b4 | SendArchEffect | S→C | não encontrado | — | — |
| 0x03b5 | CombineItemAilyn | S→C | não encontrado | — | — |
| 0x03b9 | SendAffect | S→C | não encontrado | — | — |
| 0x03ba | CombineItemAgatha | S→C | não encontrado | — | — |
| 0x03bb | MobCount | S→C | não encontrado | — | — |
| 0x03c0 | CombineItemTiny | S→C | não encontrado | — | — |
| 0x03cc | PutoutSeal | S→C | não encontrado | — | — |
| 0x03d5 | InviteGuild | S→C | não encontrado | — | — |
| 0x0401 | DBMessagePanel | DB→S | não encontrado | — | — |
| 0x0402 | DBMessageBoxOk | DB→S | não encontrado | — | — |
| 0x040a | DBSavingQuit | DB→S | não encontrado | — | — |
| 0x040b | DBCNFAccountLogOut | DB→S | não encontrado | — | — |
| 0x040e | DBCNFArchCharacterSucess | DB→S | não encontrado | — | — |
| 0x040f | DBCNFArchCharacterFail | DB→S | não encontrado | — | — |
| 0x0410 | DBCNFCapsuleCharacterFail | DB→S | não encontrado | — | — |
| 0x0411 | DBCNFCapsuleCharacterFail2 | DB→S | não encontrado | — | — |
| 0x0412 | DBCNFCapsuleSucess | DB→S | não encontrado | — | — |
| 0x0413 | DBClientMessage | DB→S | não encontrado | — | — |
| 0x0414 | DBCheckPrimaryAccount | DB→S | não encontrado | — | — |
| 0x0416 | DBCNFAccountLogin | DB→S | não encontrado | — | — |
| 0x0417 | DBCNFCharacterLogin | DB→S | não encontrado | — | — |
| 0x0418 | DBCNFNewCharacter | DB→S | não encontrado | — | — |
| 0x0419 | DBCNFDeleteCharacter | DB→S | não encontrado | — | — |
| 0x041a | DBNewAccountFail | DB→S | não encontrado | — | — |
| 0x041c | DBCharacterLoginFail | DB→S | não encontrado | — | — |
| 0x041d | DBNewCharacterFail | DB→S | não encontrado | — | — |
| 0x041e | DBDeleteCharacterFail | DB→S | não encontrado | — | — |
| 0x041f | DBAlreadyPlaying | DB→S | não encontrado | — | — |
| 0x0420 | DBStillPlaying | DB→S | não encontrado | — | — |
| 0x0421 | DBAccountLoginFailAccount | DB→S | não encontrado | — | — |
| 0x0422 | DBAccountLoginFailPass | DB→S | não encontrado | — | — |
| 0x0423 | DBSetIndex | DB→S | não encontrado | — | — |
| 0x0424 | DBAccountLoginFailBlock | DB→S | não encontrado | — | — |
| 0x0425 | DBAccountLoginFailDisable | DB→S | não encontrado | — | — |
| 0x0426 | DBOnlyOncePerDay | DB→S | não encontrado | — | — |
| 0x0427 | GuildReport | DB→S | não encontrado | — | — |
| 0x052a | DBCNFServerChange | S→C | não encontrado | — | — |
| 0x052b | DBServerSend1 | S→C | não encontrado | — | — |
| 0x07a9 | TransperCharacter | S→C | não encontrado | — | — |
| 0x0801 | DBNewAccount | S→DB | não encontrado | — | — |
| 0x0802 | DBCreateCharacter | S→DB | não encontrado | — | — |
| 0x0803 | DBAccountLogin | S→DB | não encontrado | — | — |
| 0x0804 | DBCharacterLogin | S→DB | não encontrado | — | — |
| 0x0805 | DBNoNeedSave | S→DB | não encontrado | — | — |
| 0x0806 | SavingQuit | S→DB | não encontrado | — | — |
| 0x0807 | DBSaveMob | S→DB | não encontrado | — | — |
| 0x0809 | DBDeleteCharacter | S→DB | não encontrado | — | — |
| 0x080a | DBUpdateSapphire | S→DB | não encontrado | — | — |
| 0x080c | MessageDBRecord | S→DB | não encontrado | — | — |
| 0x080d | GuildZoneReport | S→DB | não encontrado | — | — |
| 0x0814 | DBServerChange | S→DB | não encontrado | — | — |
| 0x0815 | DBItemDayLog | S→DB | não encontrado | — | — |
| 0x0816 | DBActivatePinCode | S→DB | não encontrado | — | — |
| 0x0817 | DBPrimaryAccount | S→DB | não encontrado | — | — |
| 0x081f | DBCreateArchCharacter | S→DB | não encontrado | — | — |
| 0x0ad9 | MasterGriff | C→S | não encontrado | — | — |
| 0x0c0b | MessageDBImple | DB→S | não encontrado | — | — |
| 0x0c0f | DBSendItem | DB→S | não encontrado | — | — |
| 0x0c11 | DBSendDonate | DB→S | não encontrado | — | — |
| 0x0c3c | DBCapsuleInfo | DB→S | não encontrado | — | — |
| 0x0c3d | DBPutInCapsule | DB→S | não encontrado | — | — |
| 0x0c3e | DBOutCapsule | DB→S | não encontrado | — | — |
| 0x0d1d | MagicTrumpet | S→C | não encontrado | — | — |
| 0x0d1e | DBNotice | S→C | não encontrado | — | — |
| 0x0d1f | CNFDBCapsuleInfo | S→C | não encontrado | — | — |
| 0x0e0e | War | C→S | não encontrado | — | — |
| 0x0e12 | GuildAlly | C→S | não encontrado | — | — |
| 0x0e13 | GuildInfo | C→S | não encontrado | — | — |
| 0x0faa | ReqTransper | S→C | não encontrado | — | — |
| 0x0fde | AccountSecure | S→C | crates/wyd-server/src/cpsock/action.rs, crates/wyd-server/src/cpsock/login.rs | — | — |
| 0x0fdf | AccountSecureFail | S→C | crates/wyd-server/src/cpsock/login.rs, crates/wyd-server/src/cpsock/mod.rs | — | — |
| 0x1001 | NPReqIDPASS | outra | não encontrado | — | — |
| 0x1004 | NPNotFound | outra | não encontrado | — | — |
| 0x1005 | NPAccountInfo | outra | não encontrado | — | — |
| 0x1c10 | NPAppeal | DB→S | não encontrado | — | — |
| 0x2002 | NPIDPASS | outra | não encontrado | — | — |
| 0x2003 | NPReqAccount | outra | não encontrado | — | — |
| 0x2006 | NPReqSaveAccount | outra | não encontrado | — | — |
| 0x3007 | NPDisable | outra | não encontrado | — | — |
| 0x3008 | NPEnable | outra | não encontrado | — | — |
| 0x300a | NPState | outra | não encontrado | — | — |
| 0x300b | NPCreateCharacter | outra | não encontrado | — | — |
| 0x300c | NPCreateCharacterReply | outra | não encontrado | — | — |
| 0x300d | NPDonate | outra | não encontrado | — | — |
| 0x3409 | NPNotice | DB→S | não encontrado | — | — |
| 0x4501 | GrindRankingData | S→C | não encontrado | — | — |
| 0x4d02 | UpdateExpRanking | S→C | não encontrado | — | — |
| 0x5000 | ExpMsgPanel | outra | não encontrado | — | — |
| 0x5001 | BATTLEWOTE | outra | não encontrado | — | — |

## Lacunas de handler

164 opcode(s) do catálogo sem referência em `crates/wyd-server`:

`0x0102` MessageBoxOk (S→C) · `0x011c` AlreadyPlaying (S→C) · `0x011d` StillPlaying (S→C) · `0x0166` PKInfo (S→C) · `0x016f` DecayItem (S→C) · `0x0171` CNFGetItem (S→C) · `0x0175` CNFDropItem (S→C) · `0x0181` SetHpMp (S→C) · `0x0185` UpdateCarry (S→C) · `0x018a` SetHpDam (S→C) · `0x018b` UpdateWeather (S→C) · `0x018d` ReqChallange (S→C) · `0x0193` SetClan (S→C) · `0x0196` CloseShop (S→C) · `0x026e` CreateItem (C→S) · `0x0270` GetItem (C→S) · `0x0272` DropItem (C→S) · `0x0289` Restart (C→S) · `0x028c` Deprivate (C→S) · `0x028e` Challange (C→S) · `0x028f` ChallangeConfirm (C→S) · `0x0290` ReqTeleport (C→S) · `0x0291` ChangeCity (C→S) · `0x0292` SetHpMode (C→S) · `0x02c3` CombineItemLindy (C→S) · `0x02c4` CombineItemShany (C→S) · `0x02cd` CapsuleInfo (C→S) · `0x02d2` CombineItemOdin (C→S) · `0x02d3` CombineItemEhre (C→S) · `0x02d4` CombineItemExtracao (C→S) · `0x02d5` CombineItemLoki (C→S) · `0x02e1` CombineItemAlquimia (C→S) · `0x02e2` CombineItemOdin2 (C→S) · `0x02e4` DeleteItem (C→S) · `0x02e5` SplitItem (C→S) · `0x02e6` CombineDedekinto (C→S) · `0x02e9` CombineDedekinto2 (C→S) · `0x0338` CNFMobKill (S→C) · `0x0339` UpdateCargoCoin (S→C) · `0x034d` DisableExpMsg (S→C) · `0x0363` CreateMobTrade (S→C) · `0x0366` Action2 (S→C) · `0x0368` Action3 (S→C) · `0x0369` NoViewMob (S→C) · `0x036a` Motion (S→C) · `0x036b` UpdateEquip (S→C) · `0x0374` UpdateItem (S→C) · `0x0376` TradingItem (S→C) · `0x0378` SetShortSkill (S→C) · `0x037a` Sell (S→C) · `0x037d` CNFAddParty (S→C) · `0x037e` RemoveParty (S→C) · `0x037f` SendReqParty (S→C) · `0x0383` Trade (S→C) · `0x0384` QuitTrade (S→C) · `0x0386` CNFCheck (S→C) · `0x0387` Withdraw (S→C) · `0x0388` Deposit (S→C) · `0x0397` SendAutoTrade (S→C) · `0x0398` ReqBuy (S→C) · `0x0399` PKMode (S→C) · `0x039a` ReqTradeList (S→C) · `0x039b` ItemSold (S→C) · `0x039f` ReqRanking (S→C) · `0x03a0` Ping (S→C) · `0x03a1` StartTime (S→C) · `0x03a2` EnvEffect (S→C) · `0x03a3` SoundEffect (S→C) · `0x03a4` GuildDisable (S→C) · `0x03a5` GuildBoard (S→C) · `0x03a6` CombineItem (S→C) · `0x03a7` CombineComplete (S→C) · `0x03a8` SendWarInfo (S→C) · `0x03ab` AcceptParty (S→C) · `0x03ac` SendCastleState (S→C) · `0x03ad` SendCastleState2 (S→C) · `0x03b0` MobLeft (S→C) · `0x03b4` SendArchEffect (S→C) · `0x03b5` CombineItemAilyn (S→C) · `0x03b9` SendAffect (S→C) · `0x03ba` CombineItemAgatha (S→C) · `0x03bb` MobCount (S→C) · `0x03c0` CombineItemTiny (S→C) · `0x03cc` PutoutSeal (S→C) · `0x03d5` InviteGuild (S→C) · `0x0401` DBMessagePanel (DB→S) · `0x0402` DBMessageBoxOk (DB→S) · `0x040a` DBSavingQuit (DB→S) · `0x040b` DBCNFAccountLogOut (DB→S) · `0x040e` DBCNFArchCharacterSucess (DB→S) · `0x040f` DBCNFArchCharacterFail (DB→S) · `0x0410` DBCNFCapsuleCharacterFail (DB→S) · `0x0411` DBCNFCapsuleCharacterFail2 (DB→S) · `0x0412` DBCNFCapsuleSucess (DB→S) · `0x0413` DBClientMessage (DB→S) · `0x0414` DBCheckPrimaryAccount (DB→S) · `0x0416` DBCNFAccountLogin (DB→S) · `0x0417` DBCNFCharacterLogin (DB→S) · `0x0418` DBCNFNewCharacter (DB→S) · `0x0419` DBCNFDeleteCharacter (DB→S) · `0x041a` DBNewAccountFail (DB→S) · `0x041c` DBCharacterLoginFail (DB→S) · `0x041d` DBNewCharacterFail (DB→S) · `0x041e` DBDeleteCharacterFail (DB→S) · `0x041f` DBAlreadyPlaying (DB→S) · `0x0420` DBStillPlaying (DB→S) · `0x0421` DBAccountLoginFailAccount (DB→S) · `0x0422` DBAccountLoginFailPass (DB→S) · `0x0423` DBSetIndex (DB→S) · `0x0424` DBAccountLoginFailBlock (DB→S) · `0x0425` DBAccountLoginFailDisable (DB→S) · `0x0426` DBOnlyOncePerDay (DB→S) · `0x0427` GuildReport (DB→S) · `0x052a` DBCNFServerChange (S→C) · `0x052b` DBServerSend1 (S→C) · `0x07a9` TransperCharacter (S→C) · `0x0801` DBNewAccount (S→DB) · `0x0802` DBCreateCharacter (S→DB) · `0x0803` DBAccountLogin (S→DB) · `0x0804` DBCharacterLogin (S→DB) · `0x0805` DBNoNeedSave (S→DB) · `0x0806` SavingQuit (S→DB) · `0x0807` DBSaveMob (S→DB) · `0x0809` DBDeleteCharacter (S→DB) · `0x080a` DBUpdateSapphire (S→DB) · `0x080c` MessageDBRecord (S→DB) · `0x080d` GuildZoneReport (S→DB) · `0x0814` DBServerChange (S→DB) · `0x0815` DBItemDayLog (S→DB) · `0x0816` DBActivatePinCode (S→DB) · `0x0817` DBPrimaryAccount (S→DB) · `0x081f` DBCreateArchCharacter (S→DB) · `0x0ad9` MasterGriff (C→S) · `0x0c0b` MessageDBImple (DB→S) · `0x0c0f` DBSendItem (DB→S) · `0x0c11` DBSendDonate (DB→S) · `0x0c3c` DBCapsuleInfo (DB→S) · `0x0c3d` DBPutInCapsule (DB→S) · `0x0c3e` DBOutCapsule (DB→S) · `0x0d1d` MagicTrumpet (S→C) · `0x0d1e` DBNotice (S→C) · `0x0d1f` CNFDBCapsuleInfo (S→C) · `0x0e0e` War (C→S) · `0x0e12` GuildAlly (C→S) · `0x0e13` GuildInfo (C→S) · `0x0faa` ReqTransper (S→C) · `0x1001` NPReqIDPASS (outra) · `0x1004` NPNotFound (outra) · `0x1005` NPAccountInfo (outra) · `0x1c10` NPAppeal (DB→S) · `0x2002` NPIDPASS (outra) · `0x2003` NPReqAccount (outra) · `0x2006` NPReqSaveAccount (outra) · `0x3007` NPDisable (outra) · `0x3008` NPEnable (outra) · `0x300a` NPState (outra) · `0x300b` NPCreateCharacter (outra) · `0x300c` NPCreateCharacterReply (outra) · `0x300d` NPDonate (outra) · `0x3409` NPNotice (DB→S) · `0x4501` GrindRankingData (S→C) · `0x4d02` UpdateExpRanking (S→C) · `0x5000` ExpMsgPanel (outra) · `0x5001` BATTLEWOTE (outra)

## Opcodes vistos em captura

Confirmados pelo diff de captura (opcode já decodificado no artefato; os frames crus seguem cifrados):

| Wire | Opcode | Arquivo |
| --- | --- | --- |
| 0x010a | CNFAccountLogin | tmp/captura/diff-selchar.json |

## Estruturas conhecidas

### ShopList (0x017c)

`struct ShopList` — 236 B de fio

- `shop_type: i32`
- `items: [WireItem; MAX_SHOPLIST]`
- `tax: i32`

## Evidência no cliente

Imports de rede em WYD.exe: **WSOCK32.dll** (13), **WININET.dll** (8), **WS2_32.dll** (6).

Strings de rede/protocolo classificadas: **27**; amostra:

- `_MSG_CombineAmunlaStone` (0x6235a8)
- `_MSG_CombineCube` (0x6235d8)
- `_MSG_CombineGod2Item` (0x623614)
- `_MSG_CombineGodUnSeal` (0x62362c)
- `_MSG_CombineItem` (0x623544)
- `_MSG_CombineRuneStone` (0x6235c0)
- `_MSG_CombineStone` (0x6235ec)
- `_MSG_FreeMaxLevel` (0x623594)
- `_MSG_MergeItem` (0x623558)
- `_MSG_ReturnItem` (0x623534)
- `_MSG_TransUpgrade` (0x623568)
- `_MSG_TransWeaponUpgrade` (0x62357c)
- `./serverlist.bin` (0x622938)
- `#define ITEM_CHECKSUM %d` (0x6222f4)
- `#define ITEM_CHECKSUM2 %d` (0x622310)
- `#define SKILL_CHECKSUM %d` (0x622270)
- `CheckSum Error` (0x6158c4)
- `CheckSum Error : %d,%d   m_nCheckSum=%d  nCheckSum = %d` (0x61b374)
- `CheckSum Error | %d,%d  TMGround::m_nCheckSum=%d nCheckSum=%d` (0x61b498)
- `Header` (0x6252a8)
- `incorrect header check` (0x5fce44)
- `Initialize single socket fail` (0x61027c)
- `Initialize socket fai` (0x610244)
- `Initialize socket fail` (0x6101dc)
- `Initialize Socket Failed` (0x622ff8)
- `sn.bin` (0x6230c4)
- `was compiled with header files that match` (0x6105cd)

## Capturas em `tmp/`

| Arquivo | Bytes | Frames/tokens | Tipos textuais |
| --- | --- | --- | --- |
| tmp/captura/diff-selchar.json | 414 | 0 | — |
| tmp/captura/diff.json | 143 | 1 | S2C 0XE628 |
| tmp/captura/selchar-a.json | 4056 | 1 | — |
| tmp/captura/selchar-b.json | 4056 | 1 | — |

Os frames crus guardam bytes cifrados; decodificar o `Type` exige a keyword da sessão, que não está no arquivo. Por isso a coluna de tipos só mostra o que já está em texto/diff (ex.: `diff-selchar.json`, que traz `opcode: 0x010a`) — o restante é **não decodificado**, não inferido.

## Como regenerar

```bash
node apps/tools/bin/wyd-tools.mjs pacotes
```

