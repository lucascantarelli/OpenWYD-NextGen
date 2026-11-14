# GAME-10 — Cidadania: jogador não sabe se é cidadão do canal

> **ID canônico:** `GAME-10`
> **ID histórico:** `GAME-10`
> **Estado 2026-11-13:** ⬜ · **reaberta** · ver `roadmap-rust/index.md`.

**Tipo:** Bug / requisito de produto · **Esforço:** S · **Depende de:** `GAME-17` (entrada no mundo estável) · **Bloqueia:** clareza da onda 1

## Objetivo

Ao entrar no mundo, o jogador recebe **uma mensagem dizendo se é cidadão daquele canal** —
“você é cidadão do canal” ou “você não é cidadão deste canal” — e a cidadania aparece na
interface. Hoje não aparece nada.

## Contexto / evidência

- **Relato do dono (2026-11-13):** “não aparece cidadania; quando loga tem que dizer se é cidadão
  do canal ou não, uma msg: você não é cidadão do canal”.
- `GAME-10` foi fechada em 2026-11-12 provando regra (bônus de caça/drop, recusa de NPC de outra
  cidade, canal verde) com testes de `wyd-core`/`wyd-sim`/`canal_verde` — **sem** nenhuma
  mensagem ao jogador e sem prova visual.
- `_MSG_MessagePanel` (0x0101) é o único texto livre que o 7662 desenha (usado nas recusas de
  login); é o caminho natural para a mensagem.

## Passos

1. Definir o texto e o momento: no `CNFCharacterLogin`/entrada no mundo, enviar `MessagePanel`
   com “Você é cidadão do canal N” ou “Você não é cidadão deste canal”.
2. Exibir a cidadania no `STRUCT_SELCHAR`/`UpdateScore` conforme o campo que o cliente lê
   (capturar no 7662 qual campo a tela usa).
3. Cobrir com teste de fio (pacote enviado, texto exato) e provar no cliente.

## Critérios de aceite

- [ ] Mensagem de cidadania enviada no login, com texto definido e teste de bytes.
- [ ] Print do cliente mostrando a mensagem e o indicador de cidadania.
- [ ] Cidadão e não-cidadão produzem mensagens distintas.
