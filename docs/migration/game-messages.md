# Mensagens de jogo: canais, regra e auditoria

> Como o servidor Go fala com o cliente 7662. Toda mensagem que o jogador deveria ver passa por um
> destes cinco caminhos; nenhum outro formato é desenhado pelo cliente.

## 1. Canais (o que o cliente desenha)

| Caminho | Frame | Quando usar | Onde |
|---|---|---|---|
| Painel de texto | `MSG_MessagePanel` com `HEADER.ID = 0` (`SendFunc.cpp:27-43`) | avisos e recusas: "não possui gold", "apenas na cidade", resultado de refino | `handler/notice.go` (`notify`/`notifyf`) e `sendClientMessage` |
| Fala no chat | `MSG_MessageChat` | linha de chat/`SendSay` de NPC, comandos que respondem texto (`/cp`, `/gritar`) | `handler/chat.go` (`sendChatText`, `sendNPCChatText`) |
| Sussurro/canais | `MSG_MessageWhisper` | privado, grupo (`=`), guilda (`-`, `Color = 3`) | `handler/chat.go` |
| Painel de EXP | `MSG_Exp_Msg_Panel_` (0x5000) | ganho de EXP (o cliente pinta como notificação, não como fala) | `handler/chat.go` (`sendMsgExp`) |
| Grito global | `MSG_MagicTrumpet` com `HEADER.ID = 0` | `/gritar` (Trombeta Mágica), anúncios para todos online | `handler/chat.go` (`magicTrumpet`) |

**O que NÃO funciona:** `MSG_MessageBoxOk` com 4 bytes numéricos. É o que o port usava como
fallback de aviso: o `String[96]` fica vazio, então o cliente 7662 abre uma caixa em branco ou
simplesmente não desenha nada. Foi a causa raiz do "comprei sem gold e não aconteceu nada" (QA
05/11 e loja pessoal 06/11) e de dezenas de recusas silenciosas.

## 2. Regra do projeto

1. **Todo aviso tem texto.** Um `Notice` só existe se houver a linha correspondente em
   `tmserver/content/TMsrv/run/Language.txt` e uma entrada em `noticeTexts`.
2. **Texto copiado da tabela**, nos bytes Windows-1252 dela (`\xe3` = ã, `\xe7` = ç). Inventar
   string é pior que a caixa vazia, porque parece funcionar.
3. **Aviso com valor usa `notifyf`** (`Você precisa de %d Gold.`, Language.txt 204) — nunca um
   texto genérico que esconde o número.
4. **Recusa fala.** Se uma ação pode ser recusada (sem gold, inventário cheio, caixa do armazém no
   teto de 2G, anti-tamper da loja), o caminho de recusa envia texto. `return` silencioso só vale
   para pacote malformado ou estado inconsistente, nunca para regra de jogo.
5. **Lacuna aparece no log.** Aviso sem texto cai no fallback numérico com `log.Warn`
   ("aviso sem texto: a caixa numérica é invisível no cliente 7662"), para a auditoria achar o que
   falta. `TestEveryLiveNoticeHasText` reprova qualquer código novo sem texto.

## 3. Catálogo aviso → Language.txt

| `Notice` | id | Texto |
|---|---|---|
| `NoticeVersionMismatch` | 36 | Versão do jogo incompatível com os servidores… |
| `Notice3WrongPass` | 37 | 3 tentativas de senha incorreta ocasiona bloqueio… |
| `NoticeBadPass` | 133 | Senha incorreta. |
| `NoticeNoAccount` | 131 | Conta inexistente. |
| `NoticeBlocked` | 132 | Conta bloqueada Grade X… |
| `NoticeSelectCharacter` | 34 | Por favor selecione um personagem. |
| `NoticeDeletingWait` | 469 | Excluindo personagem. Aguarde… |
| `NoticeLoginNow` | 470 | Realizando login. Aguarde… |
| `NoticeDBError` | 476 | Resultou em falha. |
| `NoticeBillingDenied` | 126 | Você usou todo o tempo de serviço. |
| `NoticeNotConnected` | 91 | O jogador não está conectado. |
| `NoticeDenyWhisper` | 165 | Não é possível sussurrar para este jogador. |
| `NoticeCargoFull` | 273 | Não é possível armazenar mais que 2 bilhões de Gold. |
| `NoticeCantGetMore2G` | 273 | idem |
| `NoticeCantDropHere` | 117 | Não é possível jogar itens aqui. |
| `NoticeCantAutoTrade` | 170 | Não é possível durante a auto venda. |
| `NoticeOnlyVillage` | 172 | Apenas na cidade. |
| `NoticeNotEnoughMoney` | 113 | Não possui gold suficiente. |
| `NoticeNoSpaceToTrade` | 86 | Não há espaço no inventário para realizar transações. |
| `NoticeItemSold` | 171 | O item foi vendido. |
| `NoticeReqNotMet` | 298 | Nível Insuficiente. Isto não pode ser utilizado. |
| `NoticeCantUseHere` | 96 | Não está disponível nesta área. |
| `NoticeCantEatMore` | 345 | Não é possível comer mais. |
| `NoticeOnlyToEquips` | 74 | Possível somente com armas e armaduras equipadas. |
| `NoticeCantRefineMore` | 75 | Este item não pode ser mais refinado. |
| `NoticeFailToRefine` | 76 | Refinação falhou. |
| `NoticeRefineSuccess` | 176 | Obteve sucesso na refinação. |
| `NoticeIncubated` | 253 | Incubação sucedida. |
| `NoticeIncuWaitMore` | 261 | Incubação não está pronta. |
| `NoticeMountLevel` | 263 | Sua montaria cresceu. |
| `NoticeNoEmptySlot` | 307 | Não há espaço para o Arch. Exclua um personagem. |
| `NoticeMaxBag` | 531 | Você já possui as 2 bolsas ativadas. |
| `NoticeNoKey` | 79 | Você não possui uma chave. |
| `NoticeSetWarp` | 186 | Localização foi armazenada. |
| `NoticeOtherClassSkill` | 107 | Não pode aprender habilidades de outra classe. |
| `NoticeNotEnoughSkillPoint` | 108 | Não possui pontos suficientes. |
| `NoticeAlreadyLearned` | 109 | Você já aprendeu essa habilidade. |
| `NoticeOnlyOneEighthSkill` | 402 | 8º Skill pode ser somente da 1ª Classe. |
| `NoticeLearnPrereq` | 403 | É necessário aprender todas as skills antes da 8ª Skill. |
| `NoticeMaxPoint` | 105 | Não pode colocar mais pontos. |
| `NoticeNotEnoughCoin` | 204 | Você precisa de %d Gold. |
| `NoticeAlreadyDone` | 71 | Você já completou esta Quest. |

Obsoletos (nada envia, mantidos para não deslocar o `iota`): `NoticeDuelBattleInProgress`,
`NoticeDuelWin`, `NoticeDuelLose`, `NoticeDuelDraw`, `NoticeDuelInCity`,
`NoticeReinoCapeRequired`.

Textos enviados direto por `sendClientMessage` (não são `Notice`, mas seguem a mesma regra):
login/conta (36/37/131/132/133), comércio (`onlyMerchantSells` 112, `onlyMerchantBuyYours` 116,
deposit/withdraw 44/45, `tradeWrongPacket` 80), PvP/duelo (143/165/193/194/196/197/208/209/331),
grupo (91/119/120/121/122/215), level up (53), EXP (473), cidadania (427).

## 4. Auditoria: o que ainda não chega ao cliente

O que existe e fala corretamente hoje: login/conta, seleção de personagem, billing, chat público,
sussurro/grupo/guilda, `/gritar` (grito global do canal), loja de NPC (compra/venda), loja pessoal
(abrir, listar, comprar), refino, montaria/incubação, skills, gates com chave, armazém, comandos
(`/cp`, `/nick`, `/nt`, `/day`, teleportes, guilda).

Lacunas conhecidas, em ordem de impacto:

### 4.1 Fluxos que voltaram a falar (06/11, QA in-game)

| Fluxo | Antes | Agora |
|---|---|---|
| Pedra Adamantita / Beril / Tectita / Spinner no item | `NoticeRefineSuccess`/`NoticeFailToRefine` iam como caixa **numérica** (invisível): "joguei e não avisou nada" | painel de texto: 176 "Obteve sucesso na refinação." / 76 "Refinação falhou." |
| Âmagos / anvil de refino (Ori, Lac, Âmago, Lactolerium_100) | já falava por `sendRefineMessage`/`refineDustReject` | sem mudança; coberto por 74/75/76/176 |
| Esferas da sorte nos Perzens (místico e arcano) | a troca entregava a montaria **em silêncio**, e sem o item exigido o NPC não dizia nada | "Troca realizada: sua recompensa está na mochila." e "Você precisa trazer o item exigido por este NPC." |
| Bolsa do Andarilho | só o teto das 2 bolsas avisava (531); a ativação não dizia nada | "Bolsa do Andarilho ativada: mais 15 espaços no inventário." |
| Jóia da Recuperação sem buff para limpar | consumia em silêncio | "Nenhum efeito para remover." |
| Selo do Guerreiro (+10 Fama, amuleto de reino) | entrava sem confirmação | "Você ganhou 10 pontos de Fama." |
| Custo em gold recusado (guilda, 8ª skill) | caixa numérica (invisível) | `Você precisa de %d Gold.` (204) com o valor real |
| Loja pessoal: anti-tamper e compra de NPC sem preço | `return` calado ("o servidor travou") | `Erro na transação.` (80) |

Textos de confirmação que o Language.txt não tem são **de projeto** (`bagActivatedText`,
`perzenDoneText`, `perzenNeedText`, `noCleanseText`, `fameGainedText`): curtos, no mesmo encoding e
usados só para informar — nunca para impedir ação nem para recusas cujo texto o cliente compara.

### 4.2 Lacunas

| Sistema | Situação | Por quê |
|---|---|---|
| Grito global entre canais | `MSG_MagicTrumpet` alcança só o canal atual | o legado relayava pelo DBSrv (`CFileDB.cpp:1616`); falta RPC de anúncio no dbServer |
| Guerra de guildas / reino / canais | sem anúncio (`_SS_War_*`, `_NN_Guild_Battle_Notice*`, `_DN_ChannelWar*`) | sistema de guerra não modelado; os textos já estão na tabela |
| Batalha Real / quiz / leilão | sem anúncio (`_NN_BR_*`, `_DD_QUIZ`, `_SN_AUCTION_*`) | eventos não implementados |
| Quests | só as mensagens de conclusão existentes; sem o "quest em progresso" (`_NN_NoHave_Quest`) | quests pontuais; o motor de quests não existe |
| Mute de chat (`MuteChat`) | não modelado | `/gm` ainda não tem mute; "Chat foi bloqueado pelo GM" (352) fica sem uso |
| Morte/PvP | fala por `sendClientMessage`, sem texto de perda de EXP localizado | `_DN_lose_D_exp` (142) exige formatação; hoje o handler usa chat |

Regra para fechar cada linha: implementar o sistema e emitir pelo canal da §1 — não criar um
formato novo, e não usar `MSG_MessageBoxOk`.

## 5. Como adicionar uma mensagem

1. Ache o id e o texto em `tmserver/content/TMsrv/run/Language.txt` (o número é o `Language.h`).
2. Se for recusa/aviso: declare o `Notice` **no fim** do bloco (o `iota` é a ordem histórica),
   copie o texto em bytes Windows-1252 para `noticeTexts` e chame `notify`/`notifyf`.
3. Se for fala de NPC ou resposta de comando: `sendClientMessage` (painel) ou `sendChatText`
   (chat) — o NPC usa `sendNPCChatText` com o id do NPC no header.
4. Acrescente o código em `TestEveryLiveNoticeHasText` e rode
   `go test ./tmserver/internal/handler/ -run 'Notice|AutoTrade|Cargo|Gate' -v`.
