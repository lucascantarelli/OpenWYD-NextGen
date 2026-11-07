-- 0006_delivery_queue: grants do wyd-portal/admin para dreno do servidor (única ponte web→jogo).
-- Convenção: -- migrate:up / -- migrate:down (ver docs/arquitetura/modelo-banco.md §4).
-- Grant pendente impede apagar conta/personagem (RESTRICT: falha explícita, sem item órfão).

-- migrate:up
CREATE TABLE delivery_queue (
    id              BIGSERIAL PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    kind            TEXT NOT NULL,
    account_id      BIGINT REFERENCES accounts (id) ON DELETE RESTRICT,
    character_id    BIGINT REFERENCES characters (id) ON DELETE RESTRICT,
    item_id         INTEGER,
    qty             INTEGER,
    gold_amount     BIGINT,
    level_delta     INTEGER,
    status          TEXT NOT NULL DEFAULT 'pending',
    attempts        INTEGER NOT NULL DEFAULT 0,
    claimed_at      TIMESTAMPTZ,
    CONSTRAINT delivery_queue_kind CHECK (kind IN ('item', 'gold', 'level')),
    CONSTRAINT delivery_queue_status CHECK (status IN ('pending', 'claimed', 'failed')),
    CONSTRAINT delivery_queue_attempts CHECK (attempts >= 0),
    CONSTRAINT delivery_queue_item_coherent CHECK (
        (kind <> 'item') OR (item_id IS NOT NULL AND qty IS NOT NULL AND qty >= 1)
    ),
    CONSTRAINT delivery_queue_gold_coherent CHECK (
        (kind <> 'gold') OR (gold_amount IS NOT NULL AND gold_amount >= 0)
    ),
    CONSTRAINT delivery_queue_level_coherent CHECK (
        (kind <> 'level') OR (level_delta IS NOT NULL AND level_delta > 0)
    )
);
CREATE INDEX delivery_queue_pending_idx ON delivery_queue (status) WHERE status = 'pending';

-- migrate:down
DROP INDEX delivery_queue_pending_idx;
DROP TABLE delivery_queue;
