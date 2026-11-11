---
title: O Campo de Treinamento abriu
date: 2026-11-10
author: Equipe WYD2
tag: Jogo
summary: Dez NPCs, quest de entrada, drops e um chefe no mapa inicial — e a cena agora chega ao cliente por proximidade.
---

O mapa inicial ganhou vida: dez NPCs com nome e posição vindos do banco, a quest de
entrada do Campo de Treinamento e um chefe no fim da trilha.

## Cena por proximidade

O cliente 7662 não desenha 6 mil mobs de uma vez — a leitura da cena inteira no login
travava a tela. A entrega passou a ser por proximidade: o canal monta a vista ao redor
do personagem (24 tiles, ajustável por configuração) e recalcula a cada varredura, de
modo que andar troca o conjunto de NPCs sem reenviar o mapa inteiro.

O resultado é o que o jogador vê: NPCs na tela, sem travar na entrada.
