-- Overlay dos efeitos base de item (issue 44, metade de escrita).
--
-- `Release/Common/ItemList.csv` guarda os adds base como pares
-- `EF_<nome>,<valor>`, e a quantidade varia por item (a 1188 tem seis: classe,
-- espaço, defesa, velocidade, regen. de MP e nível). O servidor os indexa pelo
-- CÓDIGO do efeito (internal/content.BaseEffects), não por posição, então a chave
-- da edição também é o código: uma linha substitui o valor de UM efeito e a
-- ausência de linha significa "vale o conteúdo".
--
-- A web NÃO reescreve o CSV: `Release/` é conteúdo versionado e o tmServer o lê no
-- boot. O tmServer só consulta esta tabela atrás de flag
-- (W2PP_ITEM_EFFECT_EDITING); sem ela segue lendo o CSV, que é o caminho de hoje.
CREATE TABLE item_effect_override (
    item_index  INTEGER  NOT NULL,              -- índice do item em ItemList.csv
    effect      SMALLINT NOT NULL CHECK (effect >= 0 AND effect <= 127), -- ItemEffect.h
    value       INTEGER  NOT NULL,              -- STRUCT_EFFECT grava um short
    updated_by  BIGINT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (item_index, effect)
);

COMMENT ON TABLE item_effect_override IS
    'Overlay dos efeitos base de item (issue 44): uma linha por efeito editado; sem linha vale o ItemList.csv.';

-- Mesma forma de character_admin_audit (0035) e npc_audit (0005): toda escrita
-- deixa before/after + quem fez, porque efeito errado só aparece como bug em jogo
-- depois que alguém equipou o item.
CREATE TABLE item_effect_audit (
    id           BIGSERIAL PRIMARY KEY,
    moderator_id BIGINT NOT NULL,               -- account.id do moderador
    item_index   INTEGER NOT NULL,              -- não é FK: sobrevive a reimport de conteúdo
    effect       SMALLINT NOT NULL,
    action       TEXT NOT NULL,                 -- 'upsert' | 'delete'
    before       JSONB,                         -- NULL = não havia linha
    after        JSONB,                         -- NULL = linha removida
    reason       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX item_effect_audit_item_idx
    ON item_effect_audit(item_index, created_at DESC);
CREATE INDEX item_effect_audit_moderator_idx
    ON item_effect_audit(moderator_id, created_at DESC);

COMMENT ON TABLE item_effect_audit IS
    'Auditoria das edições de efeito base de item (issue 44): before/after por efeito.';
