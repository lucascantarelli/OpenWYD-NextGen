-- Snapshot das flags de processo do tmServer (issue 36, aba "Flags").
--
-- As flags vivem no `command:` do docker-compose.yaml e no ambiente (`W2PP_*`);
-- a web não as edita. O tmServer grava aqui, no boot, com que valores ele subiu —
-- é a única forma de o moderador ver a configuração real do processo em execução
-- em vez de adivinhar pelo arquivo.
--
-- Guarda histórico (uma linha por boot): comparar o snapshot atual com o anterior
-- é como se descobre que um restart mudou algo sem ninguém avisar.
CREATE TABLE server_instance_flags (
    id          BIGSERIAL PRIMARY KEY,
    server      TEXT NOT NULL,               -- 'tmserver' (cabe dbserver/binserver no futuro)
    flags       JSONB NOT NULL,              -- [{name,value,restart_required,hint}]
    captured_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX server_instance_flags_server_idx
    ON server_instance_flags(server, captured_at DESC);

COMMENT ON TABLE server_instance_flags IS
    'Flags de processo com que cada serviço subiu (issue 36): snapshot read-only, uma linha por boot.';
