-- Overlay das taxas de conteúdo (issue 36, fatia de edição).
--
-- SancRate/CompRate/QuestsRate vivem em `Release/Common/Settings/`, que é montado
-- ":ro" no container — a web não reescreve conteúdo. A edição mora aqui: uma linha
-- por célula e a ausência de linha significa "vale o arquivo".
--
-- A chave é (file, family, name) e não uma posição numérica: `CompRate.txt` tem
-- VÁRIAS chaves por família ("Compositor Item_+7", "Item_+8", …), então uma coluna
-- `pos` faria todas caírem na mesma célula. `name` é estável contra o arquivo
-- ganhar linhas novas:
--   SancRate   '0'..'11'         (índice do refino)
--   CompRate   'Item_+7'         (chave do arquivo)
--   QuestsRate '0.2'             (tier 0, arch min)
CREATE TABLE content_rate_override (
    file        TEXT    NOT NULL,                    -- 'SancRate' | 'CompRate' | 'QuestsRate'
    family      TEXT    NOT NULL,                    -- PO/PL/AMAGO · família do CompRate · EXP/COIN/LEVEL
    name        TEXT    NOT NULL,                    -- índice, chave ou "<tier>.<coluna>"
    value       INTEGER NOT NULL CHECK (value >= 0), -- o limite real é por coluna (ratetables.Row)
    updated_by  BIGINT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (file, family, name)
);

COMMENT ON TABLE content_rate_override IS
    'Overlay das taxas de conteúdo (issue 36): uma linha por célula editada; sem linha vale o arquivo em Release/Common/Settings.';

-- Mesmo formato das auditorias 0035/0036/0037: before/after lidos da tabela,
-- porque uma taxa errada só aparece como bug em jogo depois de um refino/combine.
CREATE TABLE content_rate_audit (
    id           BIGSERIAL PRIMARY KEY,
    moderator_id BIGINT NOT NULL,
    file         TEXT   NOT NULL,
    family       TEXT   NOT NULL,
    name         TEXT   NOT NULL,
    action       TEXT   NOT NULL,                    -- 'upsert' | 'reset'
    before       JSONB,                              -- NULL = não havia linha
    after        JSONB,                              -- NULL = voltou ao arquivo
    reason       TEXT   NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX content_rate_audit_key_idx ON content_rate_audit(file, family, name, created_at DESC);
CREATE INDEX content_rate_audit_moderator_idx ON content_rate_audit(moderator_id, created_at DESC);

COMMENT ON TABLE content_rate_audit IS
    'Auditoria das edições de taxa de conteúdo (issue 36): before/after por célula.';
