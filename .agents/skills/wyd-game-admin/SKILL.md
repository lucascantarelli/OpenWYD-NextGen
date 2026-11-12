---
name: wyd-game-admin
description: Administra o jogo (papéis, bans, grants, auditoria) via portal e comandos GM no servidor Rust. Use para dar item/nível, banir/desbanir, mover jogadores ou preparar conta de teste.
---

# Administração do jogo (Rust)

> Sucessão PROC-18: mantido o nome; backend migrado para `wyd-admin` + `delivery_queue`. Fonte: `crates/wyd-admin/`, `docs/arquitetura/modelo-banco.md`.

## Regras

1. **Autoridade é `account.role`.** `moderator`/`admin` no banco, lido no login; comando negado é silencioso; sessão web expira em 8h (papel novo vale no próximo login).
2. **Toda ação passa por `adminCall`.** `moderator_id` vem da sessão, nunca do corpo; `AdminResult` → HTTP; `cache-control: no-store`; auditoria de quem/o quê/quando (`crates/wyd-admin/src/audit/`).
3. **A web nunca escreve estado vivo.** Item/gold/nível vão por `delivery_queue` (`source` + motivo auditado) drenada no login; nunca edite `character`/`item` à mão — o próximo save sobrescreve.
4. **Grants não são idempotentes por repetição.** Anote antes/depois; `setgold`/item repetido duplica.
5. **Emergência sem portal.** SQL direto só offline, sem auditoria do BFF; prefira `/contas` (mesma transação: papel + `role_changed_*` + `account_admin_audit`).

## Comandos GM (via chat do personagem)

`/gm kick|notice|goto|summon|spawn|item|setlevel|setgold|ban|unban|guildname|guildfame` + comandos de jogador (`/cp`, `/nick`, `/gritar`, teleportes, `/buffs`). Índices de item vêm de `item_catalog` (nunca chute).

## Auditoria

- Procure `gm` nos logs do servidor; confirme efeito no estado **e** no pacote (`MSG_UpdateScore` no painel, nunca chat).
- Lote = teste/harness reproduzível, não digitação repetida.
