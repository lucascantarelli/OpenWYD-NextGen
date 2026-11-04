-- Overlay do Carry[] por template de mob (issue 34, fatia 3).
--
-- O `Carry[]` do STRUCT_MOB (64 slots, offset 268) é a tabela de drop do mob: o
-- **slot** decide a raridade (o divisor vem de DropRate[64]) e o item decide o
-- que cai. Editar isso pelo portal é um overlay, nunca uma escrita no arquivo
-- em Release/ — o tmserver aplica no carregamento do template, uma vez no boot,
-- atrás de `-mob-carry-overlay` (OFF = arquivos puros).
--
-- Semântica autoritativa, igual ao Equip[]: salvar substitui a tabela inteira do
-- template; slot ausente é slot vazio. `Reset` apaga as linhas do template e ele
-- volta a ser exatamente o arquivo.
CREATE TABLE mob_carry_override (
    template_name TEXT    NOT NULL,
    slot          INTEGER NOT NULL CHECK (slot BETWEEN 0 AND 63),
    item_index    INTEGER NOT NULL CHECK (item_index BETWEEN 1 AND 32767),
    eff1          INTEGER NOT NULL DEFAULT 0 CHECK (eff1 BETWEEN 0 AND 255),
    effv1         INTEGER NOT NULL DEFAULT 0 CHECK (effv1 BETWEEN 0 AND 255),
    eff2          INTEGER NOT NULL DEFAULT 0 CHECK (eff2 BETWEEN 0 AND 255),
    effv2         INTEGER NOT NULL DEFAULT 0 CHECK (effv2 BETWEEN 0 AND 255),
    eff3          INTEGER NOT NULL DEFAULT 0 CHECK (eff3 BETWEEN 0 AND 255),
    effv3         INTEGER NOT NULL DEFAULT 0 CHECK (effv3 BETWEEN 0 AND 255),
    updated_by    BIGINT,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (template_name, slot)
);

COMMENT ON TABLE mob_carry_override IS
    'Overlay do Carry[64] de um template (issue 34): slot define a raridade do drop; salvar substitui a tabela do template.';

-- Mesmo formato das auditorias 0035/0036/0037/0040: before/after lidos da tabela,
-- uma linha por operação com o array completo do template.
CREATE TABLE mob_carry_audit (
    id            BIGSERIAL PRIMARY KEY,
    moderator_id  BIGINT NOT NULL,
    template_name TEXT   NOT NULL,
    action        TEXT   NOT NULL, -- 'set' | 'reset'
    before        JSONB,           -- NULL = não havia override
    after         JSONB,           -- NULL = voltou ao arquivo
    reason        TEXT   NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX mob_carry_audit_template_idx ON mob_carry_audit(template_name, created_at DESC);
CREATE INDEX mob_carry_audit_moderator_idx ON mob_carry_audit(moderator_id, created_at DESC);

COMMENT ON TABLE mob_carry_audit IS
    'Auditoria das edições de Carry[] (issue 34): array antes/depois por template.';
