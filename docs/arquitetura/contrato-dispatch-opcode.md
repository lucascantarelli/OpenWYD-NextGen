# Contrato de porte — dispatch por opcode

> **Data:** 2026-11-08 · **Fonte de verdade:** `Source/Code/TMSrv/ProcessClientMessage.cpp`
> (switch do servidor original) + `Source/Code/Basedef.h` (valores `_MSG_*`) e a reimplementação
> Go em `tmserver/internal/handler/dispatch.go` + `tmserver/internal/protocol/types.go`.
> **Autoridade:** o **C++** para *o que o servidor original tratava*; o **Go** para *o que a
> reescrita anterior já tinha decodificado* (contrato de handler); o **cliente alvo**
> (`client/WYD.exe` 7662) para *o que pode legitimamente chegar*.
> **Produto:** a tabela que `crates/wyd-protocol` + `crates/wyd-net/src/dispatch/` devem
> implementar. Nada de código nesta passada.
>
> ## Normas do projeto que regem este contrato
>
> | Norma | O que já fixa (citado, não redefinido) |
> |---|---|
> | `AGENTS.md` §3 invariante 3 | servidor é autoridade; fórmulas vêm de FASE-10 aprovada, não inventadas no core |
> | `AGENTS.md` §3 invariante 4 | pacote autoritativo de cena com `HEADER.ID = ESCENE_FIELD` (30000) |
> | `AGENTS.md` §3 invariante 5 | campo UNVERIFIED não é gate nem vira tipo; seguir `require_ported`/`Unported` |
> | `AGENTS.md` §7 | estrutura antes de código; árvore e nomenclatura de ADR-0002 |
> | `.agents/skills/wyd-rust-net/SKILL.md` | regras 1–4: validar tamanho → opcode → payload, sem `panic`, `ESCENE_FIELD`, rate-limit desde o handshake; árvore `dispatch/` com `router` |
> | `docs/arquitetura/protocolo.md` §2/§4 | valor de fio = base + bits de direção; `Opcode::wire()` devolve o valor exato |
> | `docs/arquitetura/estrutura-e-padroes.md` §11 | `dispatch/router.rs`: opcode → handler com `match` exaustivo |
>
> **Medição nova deste documento:** o cruzamento dos 63 `case` do C++ com as 61 rotas Go e os 198
> opcodes Rust, a ausência de rota para `CombineItemOdin2`/`UpdateScore`, o estado de
> `_MSG_SplitItem` e o que o cliente alvo envia (ou não).

## 1. Como o legado decide o que tratar

- O C++ roteia por `switch (std->Type)` no início do processamento de pacote de cliente
  (`TMSrv/ProcessClientMessage.cpp:66`), com **63 `case`** (`:68`–`:307`). Cada `case` chama um
  arquivo `_MSG_*.cpp` — há **58 arquivos `_MSG_*.cpp`** em `Source/Code/TMSrv/`, porque alguns
  arquivos atendem mais de um `case` (`_MSG_Action` cobre `Action`/`Action2`/`Action3`;
  `_MSG_Attack` cobre `Attack`/`AttackOne`/`AttackTwo`).
- Os valores de fio são `const short _MSG_X = (N | FLAG_...)` em `Basedef.h:1292`–`:2547`
  (`FLAG_*` em `:1212-1221`). Exemplo: `_MSG_AccountLogin = (13 | FLAG_CLIENT2GAME)` = `0x020D`
  (`Basedef.h:1543`).
- O Go troca o `switch` por um mapa de rotas (`dispatch.routes[h.Type]`, `dispatch.go:468-477`)
  com **61 rotas registradas** (`dispatch.go:387-463`).

## 2. Inventário cruzado (63 casos C++ × Go × Rust)

Colunas: handler do C++ com a linha do `case`; `Type` **calculado** a partir de
`Basedef.h` (base + flags); rota Go (nome da constante em `protocol/types.go`); opcode já
existente em `crates/wyd-protocol/src/wire/generated/opcode_table.rs`.

| # | Handler C++ (`ProcessClientMessage.cpp`) | `Type` | Go (`types.go`) | Rust (`opcode_table.rs`) |
|---:|---|---:|---|---|
| 1 | `_MSG_AccountLogin` (L68) | 0x020D | `MsgAccountLogin` | `AccountLogin` |
| 2 | `_MSG_CreateCharacter` (L84) | 0x020F | `MsgCreateCharacter` | `CreateCharacter` |
| 3 | `_MSG_DeleteCharacter` (L80) | 0x0211 | `MsgDeleteCharacter` | `DeleteCharacter` |
| 4 | `_MSG_CharacterLogin` (L72) | 0x0213 | `MsgCharacterLogin` | `CharacterLogin` |
| 5 | `_MSG_CharacterLogout` (L76) | 0x0215 | `MsgCharacterLogout` | `CharacterLogout` |
| 6 | `_MSG_GetItem` (L202) | 0x0270 | `MsgGetItem` | `GetItem` |
| 7 | `_MSG_DropItem` (L198) | 0x0272 | `MsgDropItem` | `DropItem` |
| 8 | `_MSG_ApplyBonus` (L214) | 0x0277 | `MsgApplyBonus` | `ApplyBonus` |
| 9 | `_MSG_REQShopList` (L136) | 0x027B | `MsgREQShopList` | `REQShopList` |
| 10 | `_MSG_Restart` (L116) | 0x0289 | `MsgRestart` | `Restart` |
| 11 | `_MSG_Quest` (L184) | 0x028B | `MsgQuest` | `Quest` |
| 12 | `_MSG_Deprivate` (L120) | 0x028C | `MsgDeprivate` | `Deprivate` |
| 13 | `_MSG_Challange` (L124) | 0x028E | `MsgChallange` | `Challange` |
| 14 | `_MSG_ChallangeConfirm` (L128) | 0x028F | `MsgChallangeConfirm` | `ChallangeConfirm` |
| 15 | `_MSG_ReqTeleport` (L132) | 0x0290 | `MsgReqTeleport` | `ReqTeleport` |
| 16 | `_MSG_ChangeCity` (L168) | 0x0291 | `MsgChangeCity` | `ChangeCity` |
| 17 | `_MSG_CombineItemLindy` (L283) | 0x02C3 | `MsgCombineItemLindy` | `CombineItemLindy` |
| 18 | `_MSG_CombineItemShany` (L254) | 0x02C4 | `MsgCombineItemShany` | `CombineItemShany` |
| 19 | `_MSG_CapsuleInfo` (L303) | 0x02CD | `MsgCapsuleInfo` | `CapsuleInfo` |
| 20 | `_MSG_CombineItemOdin` (L266) | 0x02D2 | `MsgCombineItemOdin` | `CombineItemOdin` |
| 21 | `_MSG_CombineItemEhre` (L246) | 0x02D3 | `MsgCombineItemEhre` | `CombineItemEhre` |
| 22 | `_MSG_CombineItemExtracao` (L291) | 0x02D4 | `MsgCombineItemExtracao` | `CombineItemExtracao` |
| 23 | `_MSG_CombineItemAlquimia` (L287) | 0x02E1 | `MsgCombineItemAlquimia` | `CombineItemAlquimia` |
| 24 | `_MSG_CombineItemOdin2` (L267) | 0x02E2 | **—** (sem constante) | `CombineItemOdin2` |
| 25 | `_MSG_DeleteItem` (L271) | 0x02E4 | `MsgDeleteItem` | `DeleteItem` |
| 26 | `_MSG_MessageChat` (L92) | 0x0333 | `MsgMessageChat` | `MessageChat` |
| 27 | `_MSG_MessageWhisper` (L164) | 0x0334 | `MsgMessageWhisper` | `MessageWhisper` |
| 28 | `_MSG_UpdateScore` (L106) | 0x0336 | `MsgUpdateScore` (sem rota) | `UpdateScore` |
| 29 | `_MSG_Action2` (L97) | 0x0366 | `MsgAction2` | `Action2` |
| 30 | `_MSG_Attack` (L192) | 0x0367 | `MsgAttack` | `Attack` |
| 31 | `_MSG_Action3` (L98) | 0x0368 | `MsgAction3` | `Action3` |
| 32 | `_MSG_NoViewMob` (L112) | 0x0369 | `MsgNoViewMob` | `NoViewMob` |
| 33 | `_MSG_Motion` (L102) | 0x036A | `MsgMotion` | `Motion` |
| 34 | `_MSG_Action` (L96) | 0x036C | `MsgAction` | `Action` |
| 35 | `_MSG_UseItem` (L210) | 0x0373 | `MsgUseItem` | `UseItem` |
| 36 | `_MSG_UpdateItem` (L180) | 0x0374 | `MsgUpdateItem` | `UpdateItem` |
| 37 | `_MSG_TradingItem` (L160) | 0x0376 | `MsgTradingItem` | `TradingItem` |
| 38 | `_MSG_SetShortSkill` (L188) | 0x0378 | `MsgSetShortSkill` | `SetShortSkill` |
| 39 | `_MSG_Buy` (L226) | 0x0379 | `MsgBuy` | `Buy` |
| 40 | `_MSG_Sell` (L230) | 0x037A | `MsgSell` | `Sell` |
| 41 | `_MSG_RemoveParty` (L148) | 0x037E | `MsgRemoveParty` | `RemoveParty` |
| 42 | `_MSG_SendReqParty` (L152) | 0x037F | `MsgSendReqParty` | `SendReqParty` |
| 43 | `_MSG_Trade` (L234) | 0x0383 | `MsgTrade` | `Trade` |
| 44 | `_MSG_QuitTrade` (L206) | 0x0384 | `MsgQuitTrade` | `QuitTrade` |
| 45 | `_MSG_Withdraw` (L144) | 0x0387 | `MsgWithdraw` | `Withdraw` |
| 46 | `_MSG_Deposit` (L140) | 0x0388 | `MsgDeposit` | `Deposit` |
| 47 | `_MSG_SendAutoTrade` (L218) | 0x0397 | `MsgSendAutoTrade` | `SendAutoTrade` |
| 48 | `_MSG_ReqBuy` (L222) | 0x0398 | `MsgReqBuy` | `ReqBuy` |
| 49 | `_MSG_PKMode` (L172) | 0x0399 | `MsgPKMode` | `PKMode` |
| 50 | `_MSG_ReqTradeList` (L176) | 0x039A | `MsgReqTradeList` | `ReqTradeList` |
| 51 | `_MSG_AttackOne` (L193) | 0x039D | `MsgAttackOne` | `AttackOne` |
| 52 | `_MSG_AttackTwo` (L194) | 0x039E | `MsgAttackTwo` | `AttackTwo` |
| 53 | `_MSG_ReqRanking` (L242) | 0x039F | `MsgReqRanking` | `ReqRanking` |
| 54 | `_MSG_CombineItem` (L238) | 0x03A6 | `MsgCombineItem` | `CombineItem` |
| 55 | `_MSG_AcceptParty` (L156) | 0x03AB | `MsgAcceptParty` | `AcceptParty` |
| 56 | `_MSG_CombineItemAilyn` (L258) | 0x03B5 | `MsgCombineItemAilyn` | `CombineItemAilyn` |
| 57 | `_MSG_CombineItemAgatha` (L262) | 0x03BA | `MsgCombineItemAgatha` | `CombineItemAgatha` |
| 58 | `_MSG_CombineItemTiny` (L250) | 0x03C0 | `MsgCombineItemTiny` | `CombineItemTiny` |
| 59 | `_MSG_PutoutSeal` (L307) | 0x03CC | `MsgPutoutSeal` | `PutoutSeal` |
| 60 | `_MSG_InviteGuild` (L275) | 0x03D5 | `MsgInviteGuild` | `InviteGuild` |
| 61 | `_MSG_War` (L299) | 0x0E0E | `MsgWar` | `War` |
| 62 | `_MSG_GuildAlly` (L295) | 0x0E12 | `MsgGuildAlly` | `GuildAlly` |
| 63 | `_MSG_AccountSecure` (L88) | 0x0FDE | `MsgAccountSecure` | `AccountSecure` |

Resumo medido: **63/63** valores existem no catálogo Rust (`opcode_table.rs`, 198 entradas
`Opcode::X => 0x…`); **61** têm rota no Go; **62** têm constante Go; **1** (`CombineItemOdin2`)
não existe em Go; `_MSG_UpdateScore` tem constante Go mas **não** está roteado (é S→C no fluxo
real — ver §4).

## 3. O que é tráfego de rede local (e não de cliente)

Metade dos opcodes do legado **nunca passa pelo socket do cliente**: são mensagens
TMSrv ↔ DBSrv/BISrv na porta 7514 (`Basedef.h:105`, `TMSrv/Server.cpp:4012-4017`).
Elas usam as flags `FLAG_GAME2DB`/`FLAG_DB2GAME` e um `switch` próprio
(`ProcessDBMessage.cpp`, 1.483 linhas; espelhado por 48 constantes `_MSG_DB*` em
`Basedef.h:1292-1513`).

| Classe | Flag | Onde é tratado | Entra no `dispatch` de cliente? |
|---|---|---|---|
| Cliente → jogo | `FLAG_CLIENT2GAME` (0x0200) | `ProcessClientMessage.cpp:66` | **sim** |
| Jogo → cliente | `FLAG_GAME2CLIENT` (0x0100) | `SendFunc.cpp` (2.145 linhas) | não (é saída) |
| Jogo ↔ DB | `FLAG_GAME2DB`/`FLAG_DB2GAME` (0x0800/0x0400) | `ProcessDBMessage.cpp` | **não** — não é rede de jogador |
| DB ↔ NP | 0x1000/0x2000 | `DBSrv`/`NPTool` | não |
| família "new" | `FLAG_NEW` (0x4000) | ranking (grind/exp) | não |

Consequência para o Rust: **`dispatch/` só trata `FLAG_CLIENT2GAME`**; o caminho
jogo↔banco é a porta de persistência (`wyd-db`), não um socket próprio (ADR-0001/0009) — a
topologia de 3 processos do legado não é reproduzida (decisão já tomada, aqui só se registra
o que o legado fazia).

## 4. O que o cliente alvo realmente envia — cruzamento com `client/`

Medições e leituras sobre o binário alvo (`client/WYD.exe`, 2.347.008 B, SHA-256 `04c9d2a1…`):

| Verificação | Resultado | Evidência |
|---|---|---|
| Constante `INITCODE` presente | **sim** — `11 F3 11 1F` em 3 offsets (150059, 150556, 151488) | `grep -oba $'\x11\xf3\x11\x1f' client/WYD.exe` |
| Classe de transporte | mesma família CPSock (header 12 B + keyword table de 512 B) | `docs/cliente/superficie-conhecida.md` §1 (tabela `pKeyWord` byte-idêntica) |
| Versão enviada no login | **12000** (medido em runtime no legado Go) × `APP_VERSION` do C++ = **7640** | `docs/cliente/superficie-conhecida.md` §1 vs `Basedef.h:102` |
| Opcodes tratados pelo servidor | 63 `case` no C++, todos presentes no catálogo Rust | §2 |

**Divergências que o porte precisa decidir (não ignorar):**

1. **Versão de cliente (D-D1 / U-A) — resolvida por ADR.** O C++ exige `ClientVersion == 7640`
   (`Basedef.h:102`; o Go repete em `dispatch.go`). O cliente oficial é `client/` (build 7662) e
   envia **12000**, medido no runtime (`superficie-conhecida.md:20`) no campo de offset 92
   (`payload/auth.rs:14`). `docs/arquitetura/adr/0020-versao-de-cliente-no-login.md` decide:
   `APP_VERSION = 7640` é **constante histórica** (nunca comparada) e o gate usa
   `PROTOCOL_VERSION = 12_000` (`wyd-protocol/src/lib.rs:40`), configurável no boot, com erro
   tipado + métrica por versão recebida. O gate ainda não existe (zero consumidores de
   `APP_VERSION`); quando nascer, nasce com essa regra — não com 7640.
2. **`_MSG_UpdateScore` (0x0336).** Existe `case` no C++ (L106), mas o fluxo é S→C (score do
   jogador). Tratar como entrada de cliente é convite a exploit; no Rust ele é **saída**
   (`SendFunc`-equivalente), não rota.
3. **`_MSG_CombineItemOdin2` (0x02E2).** C++ trata, Go não tem constante nem rota. Ou o Go
   perdeu um opcode real, ou o C++ tem código morto. **UNVERIFIED (U-B):** *o cliente 7662
   envia 0x02E2?* — fecha com busca de imediato `0x02E2` no `WYD.exe` e captura da janela de
   combine.
4. **`MsgDeprivate` (0x028C).** O Go documenta que **não** é autotrade e sim destituição de
   sub-líder de guilda (`types.go:136-140`, citando `Server.cpp:8694`); o C++ tem `case`
   (L120). O Rust deve tratá-lo como ação de guilda, com autorização de líder.
5. **`MsgAccountSecure` (0x0FDE).** PIN, com flags de todas as direções
   (`Basedef.h:1586`); o Go usa `0x0FDE` (`types.go:79`). É fluxo de segurança — não pode
   cair no mesmo caminho de um handler comum sem rate limit.

## 5. Contrato de implementação no Rust

### 5.1 Fronteiras (ADR-0002 / `visao-pastas.md` §2)

| Peça | Crate/módulo | Regra |
|---|---|---|
| catálogo `Type` ↔ nome ↔ flag | `wyd-protocol::wire::generated::opcode_table` | já existe (198); **fonte congelada**, não editar à mão |
| decodificação tipada do payload | `wyd-protocol::payload::{auth,character,combat,commerce,inventory,movement,ui}` | um domínio por arquivo; payload não decodificado fica `Unported` |
| roteamento `Type → handler` | `wyd-net::dispatch::router` (`match` exaustivo — árvore §11) | nenhuma regra de jogo aqui |
| efeito de jogo | `wyd-core::systems::{movement,combat,skill,inventory,npc}` + `rules/` | recebe **intenção**, nunca resultado; fórmula só de FASE-10 (`AGENTS.md` §3 invariante 3) |
| resposta ao cliente | `wyd-protocol` + `wyd-net::session` | monta S→C por opcode do catálogo; cena vai com `HEADER.ID = ESCENE_FIELD` (`AGENTS.md` §3 invariante 4, skill regra 3) |

### 5.2 Regras obrigatórias do dispatch

1. **Default = recusar e logar**, nunca `panic` (AGENTS §3). Opcode desconhecido ou
   `UNVERIFIED` ⇒ segue o caminho conservador (`require_ported`) e registra `Type` em hex —
   exatamente o que o Go fazia como diagnóstico (`dispatch.go:471-474`).
2. **Filtro de direção:** só `FLAG_CLIENT2GAME`. Um `Type` de S→C chegando do cliente é
   anomalia (log + métrica), não rota.
3. **`ID` não é identidade:** o slot vem da sessão (socket), nunca do header (contrato de
   transporte §6). Handler só recebe a sessão autenticada.
4. **Ordem de validação igual à do legado:** sessão → estado do jogador (`USER_PLAY`) →
   payload → regra → efeito. O C++ usa `pUser[conn].Mode != USER_PLAY` como guarda em vários
   pontos (`Server.cpp:810`, `:1028`, `:1064`) — pular essa guarda reabre bypass de ação.
5. **Um opcode = um handler tipado**; `Action`/`Action2`/`Action3` e
   `Attack`/`AttackOne`/`AttackTwo` mapeiam para a **mesma** intenção de domínio (`move`,
   `attack`), com o `Type` original preservado só para telemetria — foi assim que o C++
   organizou os arquivos.
6. **Combine tem 10 variantes** (`CombineItem`, `…Odin`, `…Odin2`, `…Ehre`, `…Tiny`, `…Shany`,
   `…Ailyn`, `…Agatha`, `…Lindy`, `…Alquimia`, `…Extracao`): no legado, **um arquivo por
   variante** em `TMSrv/_MSG_CombineItem*.cpp`; no Rust é **uma** entrada de domínio com o
   catálogo de receita vindo do banco (ADR-0009), não 10 caminhos divergentes.
7. **Pacote autoritativo de cena sai com `HEADER.ID = ESCENE_FIELD`** (30000), nunca com o id da
   conexão do dono (`AGENTS.md` §3 invariante 4; skill `wyd-rust-net` regra 3) — senão barra de EXP/HP não
   atualiza no cliente.
8. **Rate-limit e telemetria desde o handshake**, sem `println!` como observabilidade
   (skill `wyd-rust-net` regra 4; árvore §11 `telemetry/metrics.rs`). Opcode fora do catálogo,
   payload truncado e `Type` de direção errada são contados por motivo — não descartados em
   silêncio (`dispatch.go:471-474` fazia log; o Rust conta e loga via `tracing`, ADR-0018).

### 5.3 Cobertura mínima por lote (ordem sugerida, alinhada ao caminho crítico)

| Lote | Opcodes | Por quê primeiro |
|---|---|---|
| L1 sessão | `0x020D`, `0x020F`, `0x0211`, `0x0213`, `0x0215`, `0x0289`, `0x0FDE` | sem login não há teste ponta-a-ponta |
| L2 movimento/visão | `0x036C`, `0x0366`, `0x0368`, `0x0369`, `0x036A`, `0x0290`, `0x0291` | é o que o cliente manda em todo tick |
| L3 combate | `0x0367`, `0x039D`, `0x039E`, `0x0270`, `0x0272`, `0x0373` | núcleo da autoridade (FASE-03-5) |
| L4 inventário/comércio | `0x0374`, `0x02E4`, `0x0379`, `0x037A`, `0x027B`, `0x0383`, `0x0384`, `0x0376`, `0x0387`, `0x0388` | comércio é item do objetivo |
| L5 social | `0x0333`, `0x0334`, `0x037E`, `0x037F`, `0x03AB`, `0x03D5`, `0x0E12`, `0x0E0E`, `0x028C` | party/guilda |
| L6 progressão | `0x0277`, `0x0336` (saída), `0x0378`, `0x039F`, `0x02CD`, `0x02E5` | skills/pontos/ranking |
| L7 conteúdo | `0x028B`, `0x03A6` + 9 combines, `0x03CC`, `0x028E`, `0x028F` | quests/refino |
| L8 eventos | `0x0397`, `0x0398`, `0x039A`, `0x02C3`/`0x02C4` | loja pessoal/autotrade |

**UNVERIFIED (U-C):** `_MSG_SplitItem = (229 | FLAG_CLIENT2GAME) = 0x02E5`
(`Basedef.h:2380`) tem arquivo `TMSrv/_MSG_SplitItem.cpp` e rota no Go
(`dispatch.go:411` → `d.splitItem`; `types.go:96`), mas **não** aparece entre os 63 `case` de
`ProcessClientMessage.cpp`. Pergunta: *o C++ trata 0x02E5 em outro ponto (dentro de
`_MSG_UpdateItem`/`_MSG_UseItem`) ou o handler foi desligado do switch?* Fecha com xrefs de
`SplitItem` em `ProcessClientMessage.cpp` + captura de dividir pilha no cliente.

## 6. O que o Rust já tem e o que falta

| Item | Estado | Evidência |
|---|---|---|
| catálogo de `Type` (198) com flags e `wire()` | ✅ | `opcode_table.rs` (1.272 linhas, 198 mapeamentos) |
| payloads tipados de 7 domínios | ✅ parcial | `wyd-protocol/src/payload/{auth,character,combat,commerce,inventory,movement,ui}.rs` |
| validação de tamanho → opcode → payload (`require_ported`) | ✅ | FASE-08-2 |
| `dispatch/router.rs` (Type → intenção → `systems`) | ❌ | hoje o crate tem `lib.rs`, `limits/` e `src/metrics.rs`; `error.rs`, `session/`, `transport/`, `dispatch/` e `telemetry/` são a árvore §11 e ainda não existem |
| resposta S→C por opcode (`ESCENE_FIELD` na cena) | ❌ | nenhum emissor; `SendFunc` do legado tem 2.145 linhas de referência |
| rota por sessão autenticada (`USER_PLAY`) | ❌ | árvore prevê `session/{state,handshake}`; nada existe ainda |

## 7. Reprodução

```bash
# valores e handlers do C++
grep -nE "^const short[[:space:]]+_MSG_" /tmp/legacy/Source/Code/Basedef.h | wc -l   # 197
grep -cE "^[[:space:]]*case _MSG_" /tmp/legacy/Source/Code/TMSrv/ProcessClientMessage.cpp  # 63
# rotas do Go
grep -cE "routes\[protocol\.Msg" /tmp/legacy/tmserver/internal/handler/dispatch.go   # 61
grep -nE "^[[:space:]]*Msg[A-Za-z0-9_]+ +Type = 0x" /tmp/legacy/tmserver/internal/protocol/types.go | wc -l  # 107
# catálogo Rust
grep -cE "Opcode::[A-Za-z0-9_]+ => 0x" crates/wyd-protocol/src/wire/generated/opcode_table.rs  # 198
# cliente alvo
LC_ALL=C grep -oba $'\x11\xf3\x11\x1f' client/WYD.exe | head
```
