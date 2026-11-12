---
name: wyd-game-admin
description: Administra o jogo (papéis, bans, grants, auditoria) via portal e comandos GM no servidor Rust. Use para dar item/nível, banir/desbanir, mover jogadores ou preparar conta de teste.
---

# Administração do jogo (Rust)

Norma: `docs/arquitetura/adr/0023-moderacao-auditoria-economia.md`,
`docs/arquitetura/modelo-banco.md` e `wyd-portal/README.md` (adminCall/sessão/auditoria).
A autorização e o envelope estão nesses docs; não os reimplemente na skill.

## Decisões específicas

- Papel é `account.role` no banco, lido pela sessão/`wyd-admin`; `moderator_id` vem da
  sessão, nunca do corpo. Negação é auditada; papel novo vale no próximo login.
- Toda ação administrativa passa por `adminCall`; item/gold/nível vão por `delivery_queue`
  e são drenados no login. Não edite `character`/`item` à mão.
- Grant repetido não é idempotente por natureza: registre antes/depois e use a chave de
  idempotência quando o contrato oferecer. SQL direto só offline, com backup e sem BFF.
- Comandos GM de chat: `/gm kick|notice|goto|summon|spawn|item|setlevel|setgold|ban|unban|guildname|guildfame`
  e comandos de jogador (`/cp`, `/nick`, `/gritar`, teleportes, `/buffs`). Índice de item
  vem de `item_catalog`, nunca de chute.
- Confirme o efeito no estado **e** no pacote (`MSG_UpdateScore`/painel), não no chat.

## Prova

Ação pelo portal + auditoria em `audit_log`; para lote, harness reproduzível em vez de
digitação repetida.
