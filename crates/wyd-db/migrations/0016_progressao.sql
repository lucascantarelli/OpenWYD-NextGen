-- 0016_progressao: pontos, atributos, fama e maestria por personagem — o que o save precisa
-- além de level/exp/hp/mp que já moram em `characters`.
-- Fonte: docs/wyd/progressao.md §2–§3 (6 pontos de status e 2 de aprendizado por nível até
-- 200, respec por safira, maestria por linhagem com teto 200/255/320, `arch_of_id`) e
-- docs/wyd/skills.md §1 (skill se aprende pagando pontos, com teto de maestria e limite de
-- 25 skills).
-- Por que arquivo próprio e depois de 0007: `character_lineage` e `character_skills`
-- referenciam `skill_lineage`/`skill_catalog`, que só existem a partir de 0007. Referência
-- para tabela ainda não criada é o defeito que esta série não aceita.

-- migrate:up
-- Bloco de status do personagem: atributos distribuíveis, pontos livres das três moedas e
-- fama. Uma linha por personagem (1:1 com `characters`), não colunas soltas na tabela quente.
CREATE TABLE character_status (
    character_id  BIGINT PRIMARY KEY REFERENCES characters (id) ON DELETE CASCADE,
    attr_str      INTEGER NOT NULL DEFAULT 0 CHECK (attr_str >= 0),
    attr_int      INTEGER NOT NULL DEFAULT 0 CHECK (attr_int >= 0),
    attr_dex      INTEGER NOT NULL DEFAULT 0 CHECK (attr_dex >= 0),
    attr_con      INTEGER NOT NULL DEFAULT 0 CHECK (attr_con >= 0),
    points_status INTEGER NOT NULL DEFAULT 0 CHECK (points_status >= 0),
    points_learn  INTEGER NOT NULL DEFAULT 0 CHECK (points_learn >= 0),
    points_skill  INTEGER NOT NULL DEFAULT 0 CHECK (points_skill >= 0),
    -- Fama tem uso declarado (skills de topo 4.000/1.000/12.000, aposta de cidade 100):
    -- progressao.md §3. Não é pontuação cosmética.
    fame          BIGINT NOT NULL DEFAULT 0 CHECK (fame >= 0),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE character_status IS
    'Pontos de status/aprendizado/skill, atributos distribuídos e fama por personagem (1:1 com characters).';

-- Maestria por linhagem: o teto depende da classe (200/255/320 segundo progressao.md §3);
-- o CHECK cobre o maior teto conhecido e o teto por classe é regra de jogo, não do banco.
CREATE TABLE character_lineage (
    character_id BIGINT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    lineage_code TEXT NOT NULL REFERENCES skill_lineage (code),
    mastery      SMALLINT NOT NULL DEFAULT 0 CHECK (mastery BETWEEN 0 AND 320),
    PRIMARY KEY (character_id, lineage_code)
);

COMMENT ON COLUMN character_lineage.mastery IS
    'Pontos de maestria na linhagem; teto efetivo (200/255/320) é regra de classe, validada no core.';

-- Skills aprendidas. `skill_index` SEM FK de propósito: o catálogo cobre 154 dos 248 índices
-- possíveis do cliente (0007/0019/0024) e uma FK barraria skill válida que ainda não foi
-- catalogada — é a mesma decisão de `npc_shop_item.item_index` (0008).
CREATE TABLE character_skills (
    character_id BIGINT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    skill_index  INTEGER NOT NULL CHECK (skill_index BETWEEN 0 AND 247),
    mastery      SMALLINT NOT NULL DEFAULT 0 CHECK (mastery >= 0),
    learned_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (character_id, skill_index)
);

CREATE INDEX character_skills_skill_idx ON character_skills (skill_index);

COMMENT ON COLUMN character_skills.skill_index IS
    'Índice da skill no cliente (0..247). Sem FK: catálogo parcial por decisão registrada.';

-- migrate:down
DROP INDEX character_skills_skill_idx;
DROP TABLE character_skills;
DROP TABLE character_lineage;
DROP TABLE character_status;
