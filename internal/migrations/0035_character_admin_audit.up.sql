-- Audit trail for moderator character actions (issue 43).
--
-- The web never writes live character state: grants go through delivery_queue
-- and every change has to be reconstructable (before/after + reason), which is
-- what this table is for. It mirrors npc_audit's shape (migration 0005) but is
-- keyed by character_id, because a character can move between accounts during a
-- support action.
CREATE TABLE character_admin_audit (
    id            BIGSERIAL PRIMARY KEY,
    moderator_id  BIGINT NOT NULL,              -- account.id of the moderator
    character_id  BIGINT NOT NULL,              -- not an FK: survives deletes
    action        TEXT NOT NULL,                -- 'grant_item' | 'grant_coin' | 'set_field' | …
    before        JSONB,                        -- value before the action (NULL = created)
    after         JSONB,                        -- value/queued rows after the action
    reason        TEXT NOT NULL DEFAULT '',     -- moderator's typed justification
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX character_admin_audit_character_idx
    ON character_admin_audit(character_id, created_at DESC);
CREATE INDEX character_admin_audit_moderator_idx
    ON character_admin_audit(moderator_id, created_at DESC);

COMMENT ON TABLE character_admin_audit IS
    'Moderator character actions (issue 43): every grant/edit with before/after and the typed reason.';
