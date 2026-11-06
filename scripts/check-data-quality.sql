-- Auditoria de qualidade de texto do Postgres do OpenWYD.
--
-- Uso:
--   docker compose exec -T db psql -U postgres -d postgres -f /scripts/check-data-quality.sql
--   (ou: docker compose exec -T db psql -U postgres -d postgres < scripts/check-data-quality.sql)
--
-- O que ele mede, por coluna de texto do schema `public`:
--   1. NFC       — grafema equivalente em forma decomposta (NFD) quebra igualdade,
--                  UNIQUE e busca: "José" em NFD != "José" em NFC.
--   2. btrim     — espaço no começo/fim, invisível na tela e em ORDER BY.
--   3. controle  — C0/C1/NUL dentro do texto (vindo de dump ou CSV truncado).
--   4. espaço duplo — ruído de importação que atrapalha busca por prefixo.
--   5. mojibake  — duplo encoding UTF-8/Latin-1 (“Ã©” em vez de “é”). A marca
--                  precisa ser uma SEQUÊNCIA de duplo encoding, não “Â” solto:
--                  “Âmbar” e “Âmago” são português correto (falso positivo real
--                  da primeira versão desta consulta: 37 itens acusados à toa).
--   6. vazio     — string vazia onde o campo é nome (não é o mesmo que NULL).
--
-- Ele NÃO corrige nada: só relata. A correção é migration
-- (`internal/migrations/0002_normalize_text.up.sql`) e, daqui pra frente,
-- `internal/textnorm` no caminho de importação.

\pset pager off
\echo '== contagem por tabela =='
SELECT relname AS tabela, n_live_tup AS linhas
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC, relname;

\echo ''
\echo '== colunas de texto com problema =='
DO $$
DECLARE
  r record;
  bad bigint;
  moji bigint;
  total int := 0;
BEGIN
  FOR r IN
    SELECT c.table_name, c.column_name
    FROM information_schema.columns c
    JOIN information_schema.tables t
      ON t.table_schema = c.table_schema
     AND t.table_name = c.table_name
     AND t.table_type = 'BASE TABLE'
    WHERE c.table_schema = 'public'
      AND c.data_type IN ('text', 'character varying')
    ORDER BY c.table_name, c.column_name
  LOOP
    EXECUTE format(
      'SELECT count(*) FROM %I WHERE %I IS NOT NULL AND (%I <> normalize(%I, NFC) OR %I <> btrim(%I) OR %I ~ ''[[:cntrl:]]'' OR %I ~ ''  '')',
      r.table_name, r.column_name,
      r.column_name, r.column_name,
      r.column_name, r.column_name,
      r.column_name,
      r.column_name
    ) INTO bad;

    EXECUTE format(
      'SELECT count(*) FROM %I WHERE %I ~ %L',
      r.table_name, r.column_name,
      -- Sequências completas de duplo encoding (não “Ã”/“Â” sozinhos).
      '(Ã©|Ã¡|Ã­|Ã³|Ãº|Ã£|Ãµ|Ã§|Ã¢|Ãª|Ã´|Ã |Ã‰|Ã‡|Ãƒ|â€“|â€”|â€™|â€œ|â€|Âº|Âª|Â£|Â»|Â«)'
    ) INTO moji;

    IF bad > 0 OR moji > 0 THEN
      total := total + 1;
      RAISE NOTICE 'PROBLEMA %.%: normalizacao/espaco=% mojibake=%', r.table_name, r.column_name, bad, moji;
    END IF;
  END LOOP;

  IF total = 0 THEN
    RAISE NOTICE 'OK: nenhuma coluna de texto com problema';
  ELSE
    RAISE NOTICE 'TOTAL: % coluna(s) de texto com problema', total;
  END IF;
END $$;

\echo ''
\echo '== nomes vazios onde o campo e identificador =='
DO $$
DECLARE
  r record;
  vazios bigint;
BEGIN
  FOR r IN
    SELECT c.table_name, c.column_name
    FROM information_schema.columns c
    WHERE c.table_schema = 'public'
      AND c.data_type IN ('text', 'character varying')
      AND c.column_name IN ('name', 'slug', 'login', 'template_name', 'title', 'label')
    ORDER BY 1, 2
  LOOP
    EXECUTE format(      'SELECT count(*) FROM %I WHERE %I IS NOT NULL AND btrim(%I) = %L', r.table_name, r.column_name, r.column_name, '') INTO vazios;
    IF vazios > 0 THEN
      RAISE NOTICE 'VAZIO %.%: %', r.table_name, r.column_name, vazios;
    END IF;
  END LOOP;
END $$;
