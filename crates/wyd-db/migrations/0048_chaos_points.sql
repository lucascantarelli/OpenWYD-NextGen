-- 0048_character_chaos: CP (Chaos Points) do personagem.
--
-- Tabela 1:1 própria porque a série proíbe `ALTER TABLE` no up (mesmo motivo de
-- `character_scores`, 0032). O banco guarda o CP real (−75..=75); o byte que o cliente lê é
-- `cp + 75` (`ChaosRate` em `pMob+0x11e`) e a borda faz o deslocamento, nunca o SQL.
-- Números e limiares: docs/wyd/pesquisa-internet-2026-11-10.md §2.4.

-- migrate:up
CREATE TABLE character_chaos (
    character_id  BIGINT PRIMARY KEY REFERENCES characters (id) ON DELETE CASCADE,
    cp            SMALLINT NOT NULL DEFAULT 0,
    CONSTRAINT character_chaos_cp CHECK (cp BETWEEN -75 AND 75)
);

-- migrate:down
DROP TABLE character_chaos;
