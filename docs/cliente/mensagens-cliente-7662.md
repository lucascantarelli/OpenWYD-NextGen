# Mensagens que o build 7662 reconhece (desassemble 2026-11-11)

> Gerado por `objdump -d -M intel client/WYD.exe` + cruzamento com
> `crates/wyd-protocol/src/wire/generated/opcode_table.rs`. A coluna **canal** diz se o
> opcode aparece em `crates/wyd-server/src/cpsock.rs` (o canal **trata ou emite**).
> `não` significa que o cliente sabe desenhar/aceitar o pacote e o servidor ainda não o
> manda — é a fila de mensagens em falta, não um bug de layout.

| Opcode | Nome | Canal |
|---|---|---|
| `0x0101` | `MessagePanel` | sim |
| `0x0102` | `MessageBoxOk` | não |
| `0x010a` | `CNFAccountLogin` | sim |
| `0x0110` | `CNFNewCharacter` | sim |
| `0x0112` | `CNFDeleteCharacter` | sim |
| `0x0114` | `CNFCharacterLogin` | sim |
| `0x0116` | `CNFCharacterLogout` | sim |
| `0x0119` | `CharacterLoginFail` | sim |
| `0x011a` | `NewCharacterFail` | sim |
| `0x011b` | `DeleteCharacterFail` | sim |
| `0x011c` | `AlreadyPlaying` | não |
| `0x011d` | `StillPlaying` | não |
| `0x0165` | `RemoveMob` | sim |
| `0x016f` | `DecayItem` | não |
| `0x0171` | `CNFGetItem` | não |
| `0x0175` | `CNFDropItem` | não |
| `0x017c` | `ShopList` | sim |
| `0x0181` | `SetHpMp` | não |
| `0x0182` | `SendItem` | sim |
| `0x0185` | `UpdateCarry` | não |
| `0x018a` | `SetHpDam` | não |
| `0x018b` | `UpdateWeather` | não |
| `0x018d` | `ReqChallange` | não |
| `0x0193` | `SetClan` | não |
| `0x0196` | `CloseShop` | não |
| `0x020f` | `CreateCharacter` | sim |
| `0x0211` | `DeleteCharacter` | sim |
| `0x0213` | `CharacterLogin` | sim |
| `0x0215` | `CharacterLogout` | sim |
| `0x026e` | `CreateItem` | não |
| `0x0292` | `SetHpMode` | não |
| `0x0333` | `MessageChat` | sim (`/cp`, chat social) |
| `0x0334` | `MessageWhisper` | sim |
| `0x0336` | `UpdateScore` | sim |
| `0x0337` | `UpdateEtc` | sim |
| `0x0338` | `CNFMobKill` | não |
| `0x0339` | `UpdateCargoCoin` | não |
| `0x0363` | `CreateMobTrade` | não |
| `0x0364` | `CreateMob` | sim |
| `0x0366` | `Action2` | não |
| `0x0367` | `Attack` | sim |
| `0x0368` | `Action3` | não |
| `0x0369` | `NoViewMob` | não |
| `0x036a` | `Motion` | não |
| `0x036b` | `UpdateEquip` | não |
| `0x036c` | `Action` | sim |
| `0x0374` | `UpdateItem` | não |
| `0x0376` | `TradingItem` | não |
| `0x0378` | `SetShortSkill` | não |
| `0x0379` | `Buy` | sim |
| `0x037a` | `Sell` | não |
| `0x037d` | `CNFAddParty` | não |
| `0x037e` | `RemoveParty` | não |
| `0x037f` | `SendReqParty` | não |
| `0x0383` | `Trade` | não |
| `0x0384` | `QuitTrade` | não |
| `0x0386` | `CNFCheck` | não |
| `0x0387` | `Withdraw` | não |
| `0x0388` | `Deposit` | não |
| `0x0397` | `SendAutoTrade` | não |
| `0x039b` | `ItemSold` | não |
| `0x039d` | `AttackOne` | sim |
| `0x039e` | `AttackTwo` | sim |
| `0x039f` | `ReqRanking` | não |
| `0x03a1` | `StartTime` | não |
| `0x03a2` | `EnvEffect` | não |
| `0x03a3` | `SoundEffect` | não |
| `0x03a4` | `GuildDisable` | não |
| `0x03a7` | `CombineComplete` | não |
| `0x03a8` | `SendWarInfo` | não |
| `0x03ac` | `SendCastleState` | não |
| `0x03ad` | `SendCastleState2` | não |
| `0x03b0` | `MobLeft` | não |
| `0x03b4` | `SendArchEffect` | não |
| `0x03b9` | `SendAffect` | não |
| `0x03bb` | `MobCount` | não |
| `0x0423` | `DBSetIndex` | não |
| `0x0424` | `DBAccountLoginFailBlock` | não |
| `0x052a` | `DBCNFServerChange` | não |
| `0x07a9` | `TransperCharacter` | não |
| `0x0d1d` | `MagicTrumpet` | não |
| `0x0faa` | `ReqTransper` | não |
| `0x0fde` | `AccountSecure` | sim |
| `0x0fdf` | `AccountSecureFail` | sim |
| `0x1001` | `NPReqIDPASS` | não |
| `0x1004` | `NPNotFound` | não |
| `0x1005` | `NPAccountInfo` | não |
