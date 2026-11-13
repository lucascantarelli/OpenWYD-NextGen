-- 0036_character_special: maestria (`Special[4]`) e pontos de maestria do personagem.
-- `_MSG_ApplyBonus` tipo 1 (`_MSG_ApplyBonus.cpp:85-123`) incrementa
-- `BaseScore.Special[detail]` e debita `SpecialBonus`. O `BaseMob` das quatro classes traz
-- `Special[4] = 0` e `SpecialBonus = 0` (conferido byte a byte em `assets/basemob/`), então
-- personagem sem linha vale zero — exatamente o template, sem materialização.
--
-- Teto do legado: 200 por maestria, 255 quando a skill de linha correspondente está
-- aprendida (`Detail` 1/2/3 → bits 7/15/23 do `LearnedSkill`). O CHECK cobre o teto máximo
-- (255); o teto dinâmico por nível é regra de servidor.

-- migrate:up
CREATE TABLE character_special (
    character_id  BIGINT PRIMARY KEY REFERENCES characters (id) ON DELETE CASCADE,
    special_0     SMALLINT NOT NULL DEFAULT 0,
    special_1     SMALLINT NOT NULL DEFAULT 0,
    special_2     SMALLINT NOT NULL DEFAULT 0,
    special_3     SMALLINT NOT NULL DEFAULT 0,
    special_bonus SMALLINT NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT character_special_range CHECK (
        special_0 BETWEEN 0 AND 255
        AND special_1 BETWEEN 0 AND 255
        AND special_2 BETWEEN 0 AND 255
        AND special_3 BETWEEN 0 AND 255
        AND special_bonus >= 0
    )
);

-- migrate:down
DROP TABLE character_special;
