-- DB-01: separa o privilégio do banco em três papéis.
--
--   openwyd_migrator  dono dos objetos, CREATE no schema — só migrations e seedgen
--   openwyd_app       DML (SELECT/INSERT/UPDATE/DELETE) — dbserver e webserver
--   openwyd_ro        SELECT — leitura/relatório, nunca é usada por serviço hoje
--
-- Idempotente: pode rodar a cada `docker compose up`. As senhas nunca aparecem no
-- arquivo: vêm do ambiente por `\getenv` (psql 15+).
\set ON_ERROR_STOP on

\getenv migrator_password OPENWYD_MIGRATOR_PASSWORD
\getenv app_password OPENWYD_APP_PASSWORD
\getenv ro_password OPENWYD_RO_PASSWORD

SELECT 'CREATE ROLE openwyd_migrator LOGIN'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openwyd_migrator')\gexec
SELECT 'CREATE ROLE openwyd_app LOGIN'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openwyd_app')\gexec
SELECT 'CREATE ROLE openwyd_ro LOGIN'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openwyd_ro')\gexec

ALTER ROLE openwyd_migrator LOGIN PASSWORD :'migrator_password';
ALTER ROLE openwyd_app LOGIN PASSWORD :'app_password';
ALTER ROLE openwyd_ro LOGIN PASSWORD :'ro_password';

-- Migrations rodam como migrator: ele precisa criar tabela/índice no schema.
GRANT CREATE, USAGE ON SCHEMA public TO openwyd_migrator;
GRANT USAGE ON SCHEMA public TO openwyd_app, openwyd_ro;

-- O banco existente nasceu como superusuário; transfere a posse para o migrator
-- para que uma migration futura possa fazer ALTER sem superusuário.
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' LOOP
        EXECUTE format('ALTER TABLE public.%I OWNER TO openwyd_migrator', r.tablename);
    END LOOP;
    FOR r IN SELECT sequencename FROM pg_sequences WHERE schemaname = 'public' LOOP
        EXECUTE format('ALTER SEQUENCE public.%I OWNER TO openwyd_migrator', r.sequencename);
    END LOOP;
END $$;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO openwyd_app;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO openwyd_app;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO openwyd_ro;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO openwyd_ro;

-- Tabela criada por uma migration futura já nasce com os grants certos.
ALTER DEFAULT PRIVILEGES FOR ROLE openwyd_migrator IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO openwyd_app;
ALTER DEFAULT PRIVILEGES FOR ROLE openwyd_migrator IN SCHEMA public
    GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO openwyd_app;
ALTER DEFAULT PRIVILEGES FOR ROLE openwyd_migrator IN SCHEMA public
    GRANT SELECT ON TABLES TO openwyd_ro;
