-- Overlay da tabela de drop por slot (issue 34, fatia de leitura/escrita).
--
-- A chance de drop não está em arquivo de configuração: `DropRate[64]` é código
-- (`tmserver/internal/loot/loot.go`, espelho de Basedef.cpp:222) e o valor é
-- **divisor** — quanto maior, mais raro. O slot é a chave: quem define a raridade
-- de um item é a posição dele no `Carry[]` do template, não o item.
--
-- Ausência de linha significa "vale o valor do código"; o tmserver só lê isto no
-- boot, atrás de `-drop-config-editing` (OFF = comportamento atual, byte a byte).
CREATE TABLE drop_rate_override (
    slot        INTEGER     PRIMARY KEY CHECK (slot BETWEEN 0 AND 63),
    value       INTEGER     NOT NULL CHECK (value BETWEEN 1 AND 1000000),
    updated_by  BIGINT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE drop_rate_override IS
    'Overlay de DropRate[64] (issue 34): divisor por slot de Carry; sem linha vale o código em loot.go.';

-- Mesmo formato das auditorias 0035/0036/0037: before/after lidos da tabela.
-- Uma taxa digitada ao contrário só aparece como economia quebrada em jogo.
CREATE TABLE drop_rate_audit (
    id           BIGSERIAL PRIMARY KEY,
    moderator_id BIGINT NOT NULL,
    slot         INTEGER NOT NULL,
    action       TEXT    NOT NULL, -- 'upsert' | 'reset'
    before       JSONB,            -- NULL = não havia linha
    after        JSONB,            -- NULL = voltou ao código
    reason       TEXT    NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX drop_rate_audit_slot_idx ON drop_rate_audit(slot, created_at DESC);
CREATE INDEX drop_rate_audit_moderator_idx ON drop_rate_audit(moderator_id, created_at DESC);

COMMENT ON TABLE drop_rate_audit IS
    'Auditoria das edições de DropRate (issue 34): before/after por slot.';
