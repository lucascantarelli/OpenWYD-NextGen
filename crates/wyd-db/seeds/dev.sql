-- seeds/dev.sql — dados de DESENVOLVIMENTO. Valores obviamente falsos (DEV-ONLY);
-- nunca commitar segredo real aqui. Produção usa runbook próprio (FASE-05), não este arquivo.

INSERT INTO accounts (login, password_hash, role)
VALUES ('dev', 'argon2id$dev$inseguro', 'admin');

INSERT INTO characters (account_id, slot, name, class, level, exp, hp, mp, pos_x, pos_y)
VALUES (1, 0, 'Dev', 0, 1, 0, 100, 10, 2100, 2100);

INSERT INTO character_items (character_id, slot, item_id, qty)
VALUES (1, 0, 1, 1);

INSERT INTO vault_items (account_id, slot, item_id, qty)
VALUES (1, 0, 2, 5);

INSERT INTO guilds (name, master_character_id)
VALUES ('DevGuild', 1);

INSERT INTO guild_members (guild_id, character_id, rank)
VALUES (1, 1, 9);

INSERT INTO delivery_queue (kind, account_id, character_id, gold_amount)
VALUES ('gold', 1, 1, 1000);
