---
title: Lojas de NPC abertas — comprar e receber item
date: 2026-11-11
author: Equipe WYD2
tag: Jogo
summary: Os 32 mercadores de Armia respondem ao pedido de catálogo e entregam o item no slot certo, com o preço vindo do catálogo do servidor.
---

O jogador agora abre a loja de um NPC, vê o catálogo e compra. A compra é decidida no
servidor: o preço vem do catálogo de itens, o slot é conferido e o item entra na mochila.

## Como foi feito

- O pacote `_MSG_REQShopList` abre a loja e o `_MSG_ShopList` devolve os itens do
  mercador — 236 bytes de catálogo por resposta.
- O `_MSG_Buy` passa pela autoridade do servidor: preço divergente é recusado, item
  fora do catálogo também.
- A prova viva cria personagem, anda até o mercador, lê o catálogo e compra uma runa,
  conferindo o item no slot 6 da mochila.

O log do canal registra cada passo (`event=shop result=list`, `bought` e `refused`),
para a operação conseguir responder a um jogador sem adivinhar.
