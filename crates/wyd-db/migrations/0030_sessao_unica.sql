-- 0030_sessao_unica: sessão viva por conta — o mesmo personagem não joga em dois canais.
--
-- Canal é processo, mas o banco é o mesmo entre os canais do grupo
-- (`docs/wyd/canais-e-cidadania.md` §5). Sem esta tabela, o mesmo personagem abria sessão
-- no canal 1 e no canal 2 ao mesmo tempo: dois mundos independentes com o mesmo personagem.
--
-- O caminho natural do jogo (transitar entre canais) continua: sair de um canal libera a
-- sessão, e a próxima entrada — em qualquer canal do grupo — reencontra o personagem no
-- banco com todos os dados. O legado faz isso pelo `_MSG_DBServerChange` (0x0814) /
-- `_MSG_DBCNFServerChange` (0x052A) porque cada canal tinha o **próprio** DB; aqui os canais
-- compartilham o Postgres, então trocar de canal é relogar (não há dado a mover).
--
-- `character_id` já existe para a seleção de personagem (FASE-08-2): hoje a cerca é por
-- `account_login` porque o handshake ainda não autentica a conta. A sessão que a autenticação
-- provar (account_id) substitui a chave sem mudar as garantias.

-- migrate:up
CREATE TABLE character_session (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_login  TEXT NOT NULL CHECK (char_length(account_login) BETWEEN 1 AND 16),
    character_id   BIGINT REFERENCES characters (id) ON DELETE CASCADE,
    group_id       SMALLINT NOT NULL CHECK (group_id BETWEEN 0 AND 9),
    channel        SMALLINT NOT NULL CHECK (channel BETWEEN 1 AND 10),
    client_version INTEGER NOT NULL,
    opened_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    renewed_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at    TIMESTAMPTZ,
    CONSTRAINT character_session_release_after_open
        CHECK (released_at IS NULL OR released_at >= opened_at)
);

COMMENT ON TABLE character_session IS
    'Sessão viva do jogador por conta/canal; índice parcial garante no máximo uma por conta.';

COMMENT ON COLUMN character_session.renewed_at IS
    'Último sinal de vida do processo dono; sessão ociosa além do TTL é retomada, não travada.';

-- Uma sessão viva por conta (o mesmo personagem não entra em dois canais ao mesmo tempo).
CREATE UNIQUE INDEX character_session_uma_por_conta
    ON character_session (account_login) WHERE released_at IS NULL;

-- Uma sessão viva por personagem, quando a seleção de personagem passar a preencher a coluna.
CREATE UNIQUE INDEX character_session_uma_por_personagem
    ON character_session (character_id)
    WHERE released_at IS NULL AND character_id IS NOT NULL;

-- Consulta de ocupação por canal (status/metrics e diagnóstico de mundo separado).
CREATE INDEX character_session_por_canal
    ON character_session (group_id, channel) WHERE released_at IS NULL;

-- migrate:down
DROP INDEX character_session_por_canal;
DROP INDEX character_session_uma_por_personagem;
DROP INDEX character_session_uma_por_conta;
DROP TABLE character_session;
