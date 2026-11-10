-- 0033_character_short_skill: `ShortSkill[16]` persistido do personagem.
-- Tabela 1:1 própria porque a série proíbe `ALTER TABLE` no up; o personagem antigo fica
-- sem linha e o servidor materializa `0xFF` (o `memset(ShortSkill, -1, 16)` da criação em
-- `CFileDB.cpp`) na primeira leitura, em vez de inventar valor no pacote.
--
-- O campo vai no `MSG_CNFClientCharacterLogin` em `ShortSkill[16]@1034`
-- (`docs/arquitetura/protocolo.md` §5.1).

-- migrate:up
CREATE TABLE character_short_skill (
    character_id BIGINT PRIMARY KEY REFERENCES characters (id) ON DELETE CASCADE,
    short_skill  BYTEA NOT NULL,
    CONSTRAINT character_short_skill_len CHECK (octet_length(short_skill) = 16)
);

-- migrate:down
DROP TABLE character_short_skill;
