-- Auditoria das edições de guilda feitas pela web (issue 45, escritas).
--
-- Guildas, filiações e relações são estado que o tmServer carrega no boot, então
-- uma edição aqui só chega ao jogo depois de um restart — mas o registro do que
-- foi mudado, por quem e por quê precisa existir de qualquer forma: trocar o
-- mestre de uma guilda é exatamente o tipo de ação que vira disputa depois.
--
-- Mesmo formato de character_admin_audit (0035) e item_effect_audit (0036):
-- before/after em JSONB lidos da própria tabela, nunca do que o cliente mandou.
CREATE TABLE guild_admin_audit (
    id           BIGSERIAL PRIMARY KEY,
    moderator_id BIGINT NOT NULL,               -- account.id do moderador
    guild_id     INTEGER NOT NULL,              -- não é FK: sobrevive a delete da guilda
    action       TEXT NOT NULL,                 -- 'set_master' | 'remove_member' | 'relation'
    target       JSONB,                         -- personagem/relação alvo
    before       JSONB,                         -- retrato da guilda antes
    after        JSONB,                         -- retrato depois (NULL quando nada sobrou)
    reason       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX guild_admin_audit_guild_idx ON guild_admin_audit(guild_id, created_at DESC);
CREATE INDEX guild_admin_audit_moderator_idx ON guild_admin_audit(moderator_id, created_at DESC);

COMMENT ON TABLE guild_admin_audit IS
    'Auditoria das edições de guilda/relação pela web (issue 45): before/after por ação.';
