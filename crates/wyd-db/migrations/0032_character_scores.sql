-- 0032_character_scores: atributos-base e pontos livres do personagem.
-- Tabela 1:1 própria porque a série proíbe `ALTER TABLE` no up; a identidade em
-- `characters` não engorda e personagens antigos ficam com os defaults (0).
--
-- Fonte: `origin/main:Source/Code/Basedef.cpp` (`BaseSIDCHM`, `g_pIncrementHp/Mp`,
-- `BASE_GetHpMp`, `BASE_GetBonusScorePoint`, `BASE_GetBonusSkillPoint`) e o porte Go
-- em `origin/main:tmserver/internal/level/level.go`.

-- migrate:up
CREATE TABLE character_scores (
    character_id  BIGINT PRIMARY KEY REFERENCES characters (id) ON DELETE CASCADE,
    strength      SMALLINT NOT NULL DEFAULT 0,
    intelligence  SMALLINT NOT NULL DEFAULT 0,
    dexterity     SMALLINT NOT NULL DEFAULT 0,
    constitution  SMALLINT NOT NULL DEFAULT 0,
    score_bonus   INTEGER NOT NULL DEFAULT 0,
    skill_bonus   INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT character_scores_strength CHECK (strength >= 0),
    CONSTRAINT character_scores_intelligence CHECK (intelligence >= 0),
    CONSTRAINT character_scores_dexterity CHECK (dexterity >= 0),
    CONSTRAINT character_scores_constitution CHECK (constitution >= 0),
    CONSTRAINT character_scores_score_bonus CHECK (score_bonus >= 0),
    CONSTRAINT character_scores_skill_bonus CHECK (skill_bonus >= 0)
);

-- migrate:down
DROP TABLE character_scores;
