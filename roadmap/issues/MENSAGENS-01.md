# MENSAGENS-01 — Auditoria das mensagens cliente ↔ servidor

**Fase:** correção in-game · **Depende de:** issue 32 (painel de mensagem) · **Esforço:** M · **Estado:** 🔄 em andamento (fatias 1 e 2 entregues em 06/11/2026)

## Objetivo

Nenhuma recusa do servidor pode ser invisível no cliente: toda ação recusada responde com o texto
de `Language.txt`, no formato que o cliente 7662 realmente desenha. A regra vale para loja, NPC,
quest, grupo, guerra, comércio e avisos globais.

## O que foi medido (06/11/2026)

O caminho de texto do cliente é **um só**: `MSG_MessagePanel` com `HEADER.ID = 0`
(`SendClientMessage`, `SendFunc.cpp:27-43`; o cliente descarta painel com ID ≠ 0 —
`TMScene.cpp:1371`). O helper `notify()` do port enviava `MSG_MessageBoxOk` com **4 bytes**, mas o
struct é `{int;int;char String[96]}` (`Basedef.h:1534-1541`): a caixa abria **vazia** — ou não
abria. Resultado: dezenas de recusas corretas no servidor pareciam "nada acontece".

Sintoma relatado ao vivo: na **loja pessoal**, comprar sem gold não comprava e não dizia nada. O
handler já chamava `NoticeNotEnoughMoney`; o aviso é que sumia no caminho.

## Fatia 1 (entregue)

- `notify()` passou a mandar **texto** por `sendClientMessage` quando existe `noticeText(n)`
  confirmado em `Language.txt`; o corpo numérico ficou como último recurso e agora loga
  `notice without text` em Debug, para a lacuna aparecer na auditoria.
- Textos mapeados (verbatim de `tmserver/content/TMsrv/run/Language.txt`):
  `_NN_Not_Enough_Money` (113), `_NN_You_Have_No_Space_To_Trade` (86), `_NN_Not_Connected` (91),
  `_NN_ItemSold` (171), `_NN_OnlyVillage` (172), `_NN_Cant_Drop_Here` (117).
- Testes: `TestAutoTradeBuyWithoutGoldShowsMessage` (compra sem gold → painel ID 0 com o texto 113)
  e `TestNoticeTextCoversShopAndTradeTexts` (trava os seis mapeamentos).
- Efeito colateral **correto**: `_NN_ItemSold` agora chega ao vendedor quando um item sai da loja —
  antes era invisível.

## Fatia 2 (entregue) — o QA in-game apontou o que ainda estava mudo

Relatos: pedra adamantita, âmagos de montaria, esferas da sorte nos Perzens e bolsa do andarilho.

- **Refino (adamantita, beril, tectita, spinner):** o handler já chamava
  `NoticeRefineSuccess`/`NoticeFailToRefine`; o que faltava era o **texto** (176/76). Com o registro da
  fatia 1, a mensagem passou a sair no painel visível — nenhuma mudança de fluxo.
- **Âmagos de montaria:** já falavam por `sendRefineMessage` (`refine_feedback.go`) e
  `refineDustReject`; conferidos contra 74/75/76/176.
- **Perzens (místico e arcano):** a troca da esfera da sorte entregava a montaria em silêncio, e sem o
  item exigido o `return` era mudo. Agora confirma a troca e pede o item (`perzenDoneText`,
  `perzenNeedText`), ambos por `sendClientMessage` (painel ID 0).
- **Bolsa do Andarilho:** só o teto das duas bolsas avisava (`NoticeMaxBag`, 531); a ativação não dizia
  nada. Agora confirma (`bagActivatedText`).
- **Bônus na mesma linha:** jóia da recuperação sem efeito a limpar e selo do guerreiro (+10 Fama)
  também confirmam/avisam.
- **Testes:** `TestWandererBagUnlocksFirstExtraPage` (painel com o texto da bolsa),
  `TestPerzenMissingItem` (pedido do item), `readNotice` (aceita painel **ou** caixa numérica e
  traduz o texto de volta para o `Notice` — texto desconhecido reprova). `TestBuyRejectsMissingPrice`
  e `TestBuyNegativePriceSpeaks` passaram a exigir `_NN_Wrong_Trade_Packet1` em vez de silêncio.
- Textos que o `Language.txt` não tem (`bagActivatedText`, `perzenDoneText`, `perzenNeedText`,
  `noCleanseText`, `fameGainedText`) são **de projeto**, curtos, e só informam sucesso — nunca
  substituem a tabela em recusas.

## Lacunas conhecidas (a fechar)

| Grupo | Avisos ainda sem texto | Onde |
|---|---|---|
| Loja pessoal | `NoticeCantGetMore2G`, `NoticeCantAutoTrade` | `autotrade.go` |
| Skills | `NoticeNotEnoughSkillPoint`, `NoticeOnlyOneEighthSkill`, `NoticeLearnPrereq`, `NoticeAlreadyLearned`, `NoticeNotEnoughCoin`, `NoticeMaxPoint`, `NoticeOtherClassSkill` | `skill.go` |
| Refino/pó | `NoticeOnlyToEquips`, `NoticeCantRefineMore`, `NoticeFailToRefine`, `NoticeIncubated`, `NoticeIncuWaitMore`, `NoticeMountLevel` | `refine_*.go` |
| Inventário | `NoticeCargoFull`, `NoticeNoEmptySlot`, `NoticeMaxBag`, `NoticeCantEatMore`, `NoticeReqNotMet` | `item.go` |
| Login/sistema | `NoticeDBError`, `NoticeBillingDenied`, `NoticeDeletingWait`, `NoticeSelectCharacter` | `login.go`, `character.go` |

Regra para fechar cada linha: copiar o texto **da tabela** (nunca inventar), adicionar ao
`noticeText` e escrever o teste do fluxo que o dispara. Um aviso novo sem texto é regressão —
o teste `TestNoticeTextCoversShopAndTradeTexts` deve crescer junto.

## Auditar antes de portar

- `grep -c 'MSG_MessageBoxOk' tmserver/` deve cair a zero quando todos tiverem texto.
- Todo `notify(` novo precisa de entrada em `noticeText`; o log `notice without text` é a métrica
  de dívida (procurar em `docker compose logs tmserver`).

## Critérios de aceite

- [x] Compra na loja pessoal sem gold mostra "Não possui gold suficiente." no cliente.
- [ ] Todos os `Notice` usados por handler têm texto confirmado (tabela acima zerada).
- [ ] Nenhum handler recusa uma ação sem responder nada (`return` silencioso revisado) — loja
  pessoal, compra de NPC, Perzen e bolsa já cobertos; faltam as lacunas da tabela acima.
- [ ] Avisos de guerra/global conferidos contra `_MSG_MessageChat` e `MSG_MagicTrumpet`.
