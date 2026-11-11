---
title: O primeiro gameplay está no ar
date: 2026-11-11
author: Equipe WYD2
tag: Servidor
summary: Login, criação de personagem e entrada no mundo funcionando de ponta a ponta no servidor em Rust, com o Campo de Treinamento cheio de NPCs e quatro bots de QA.
---

Depois de meses de porte, o servidor em Rust atende o cliente 7662 no caminho completo:
a conta entra, o personagem é criado e o jogador pisa no mundo.

## O que já funciona

- **Login e seleção de personagem** com a sessão do canal, a senha numérica e a
  reconexão recusando o segundo canal ocupado.
- **Criação de personagem** copiando o template real da classe: aparência, equipamento
  inicial, pontos de vida e o ouro de entrada.
- **Entrada no mundo** com a cena entregue por proximidade: os NPCs ao redor aparecem
  sem derrubar o cliente com 6 mil pacotes de uma vez.
- **Campo de Treinamento** com dez NPCs, quest de aceitação e um chefe, além de quatro
  bots de QA que entram, andam, atacam e conversam com os mercadores.

## Próximos passos

Distribuição dos pontos de atributo pelo cliente, efeito do baú de experiência e a
segunda leva de telas do portal. Cada passo entra com teste vivo contra o canal.
