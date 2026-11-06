# Quests, RuneQuest e recompensa diária

Três coisas diferentes que a interface do cliente apresenta parecido:

| Nome | O que é | Onde o servidor trata |
|---|---|---|
| Quest | condição de matar/coletar que o cliente acompanha | `newbiequest.go`, `newbie.go` |
| RuneQuest | cadeia de quests por runas, com progresso persistido | `runequest.go`, `runequest_kills.go` |
| Diária | recompensa reivindicável uma vez por dia | RPC do web-api + tela `/diaria` |

## Quest e RuneQuest

O cliente é quem desenha o texto da quest; o servidor conta **progresso**. Quando um mob morre,
`runequest_kills.go` incrementa o contador do personagem se aquele mob interessa à runa ativa. Ao
completar, a recompensa é entregue com EXP pelo **painel** (não como fala no chat) e o estado é
persistido. `quest_area_reset.go` zera progresso por área quando a regra manda.

## Recompensa diária

O estado é por personagem e por dia, no servidor. Reivindicar duas vezes no mesmo dia é recusado — não
é o cliente que decide se já foi pego. A tela do portal (`/diaria`) lê e ajusta a configuração; ela
**não** entrega recompensa a ninguém (entrega de item/EXP ao personagem vivo passa por
`delivery_queue`).

## Onde está

| Peça | Arquivo |
|---|---|
| Quest inicial | `newbiequest.go`, `newbie.go` |
| RuneQuest e contagem de kill | `runequest.go`, `runequest_kills.go` |
| Reset por área | `quest_area_reset.go` |
| Diária | RPC do web-api, tela `/diaria` no portal |
| Taxa de quest | `tmserver/content/Common/Settings/QuestsRate.txt` |

## Como verificar

```bash
go test ./tmserver/internal/handler/ -run 'Quest|RuneQuest|Newbie'
```

No jogo: começar uma RuneQuest, matar o mob pedido e conferir o contador subindo; completar e conferir
que a recompensa cai **uma** vez. Reivindicar a diária duas vezes deve falhar na segunda.

!!! warning "UNVERIFIED"
    O texto e a ordem das quests normais vêm do cliente; o repositório não tem uma tabela de "quest N
    pede M mortes". O que é verificável aqui é a contagem, a persistência e a entrega — não o roteiro
    narrativo.

## Ponto de atenção

EXP de quest já saiu como `MSG_MessageChat` no passado (o personagem "falava" o valor). O caminho
correto é o pacote de painel; teste `TestApplyMobExpUsesExpPanel` trava essa regressão para o ganho de
mob, e o mesmo cuidado vale para quest.
