---
title: Ouro inicial: 5 milhões para quem começa
date: 2026-11-11
author: Equipe WYD2
tag: Jogo
summary: O personagem novo entra com o ouro do template da classe, copiado do BaseMob original — e o valor agora chega à tela, não só ao banco.
---

Todo personagem criado entra no mundo com **5.000.000 de ouro**. O valor não foi
escolhido a dedo: vem do `Coin` do `BaseMob` de cada classe, o mesmo template que o
servidor original copiava na criação.

## O que estava errado

O banco gravava o valor certo desde o começo. O que faltava era o pacote que desenha o
ouro na tela: a entrada no mundo mandava a barra de vida e experiência, mas não o
`_MSG_UpdateEtc`, que carrega a moeda. Resultado: 5 milhões no banco, zero na interface.

A correção envia o par completo na entrada e tem prova viva: o teste lê o pacote e
confere o mesmo valor persistido no banco.
