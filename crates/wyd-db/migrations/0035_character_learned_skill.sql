-- 0035_character_learned_skill: `LearnedSkill` do personagem (`_MSG_ApplyBonus` tipo 2).
-- Tabela 1:1 própria porque a série proíbe `ALTER TABLE` no up; personagem anterior a esta
-- entrega fica sem linha e o servidor lê 0 (nenhuma skill aprendida), sem inventar bit.
--
-- `STRUCT_MOB.LearnedSkill@780` é máscara de 24 bits (`g_pSpell[24 * classe + pos]`); o
-- `Learn` do `MSG_UpdateEtc` viaja como `long long` (8 B), mas o legado só usa os 24 bits.

-- migrate:up
CREATE TABLE character_learned_skill (
    character_id  BIGINT PRIMARY KEY REFERENCES characters (id) ON DELETE CASCADE,
    learned_skill BIGINT NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT character_learned_skill_bits CHECK (learned_skill >= 0 AND learned_skill <= 16777215)
);

-- migrate:down
DROP TABLE character_learned_skill;
