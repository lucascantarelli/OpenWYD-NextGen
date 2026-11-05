-- Auditoria do banco (issue 58, fase 4). Somente leitura — pode rodar em produção.
--
--   docker exec -i openwyd-db-1 psql -U postgres -d postgres -f - < scripts/db-audit.sql
--
-- Cobre o que a revisão de banco precisa provar com número, não com opinião:
-- tamanho, tabelas sem PK, FK sem índice, índice morto, índice duplicado,
-- cobertura de comentário, recursos do Postgres em uso, padronização de tipo,
-- nomenclatura e planos ruins.

\echo '## 1. Tamanho e acesso por tabela'
SELECT relname AS tabela,
       n_live_tup AS linhas,
       pg_size_pretty(pg_total_relation_size(relid)) AS total,
       pg_size_pretty(pg_indexes_size(relid)) AS indices,
       seq_scan,
       idx_scan
FROM pg_stat_user_tables
ORDER BY pg_total_relation_size(relid) DESC;

\echo '## 2. Tabelas sem PRIMARY KEY'
SELECT c.relname AS tabela
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'r'
  AND NOT EXISTS (SELECT 1 FROM pg_index i WHERE i.indrelid = c.oid AND i.indisprimary)
ORDER BY 1;

\echo '## 3. FK de coluna única sem índice de origem (causa lock/scan em cascata)'
SELECT con.conrelid::regclass AS tabela,
       a.attname AS coluna,
       con.confrelid::regclass AS referencia
FROM pg_constraint con
JOIN pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = con.conkey[1]
WHERE con.contype = 'f' AND array_length(con.conkey, 1) = 1
  AND NOT EXISTS (
    SELECT 1 FROM pg_index i
    WHERE i.indrelid = con.conrelid AND i.indkey[0] = con.conkey[1] AND i.indisvalid)
ORDER BY 1, 2;

\echo '## 4. Índices nunca usados (candidatos a remoção; exclui os de constraint)'
SELECT s.relname AS tabela, s.indexrelname AS indice, s.idx_scan,
       pg_size_pretty(pg_relation_size(s.indexrelid)) AS tamanho
FROM pg_stat_user_indexes s
WHERE s.idx_scan = 0
  AND NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conindid = s.indexrelid)
ORDER BY pg_relation_size(s.indexrelid) DESC;

\echo '## 5. Índices com a mesma chave (duplicados/redundantes)'
SELECT indrelid::regclass AS tabela, array_agg(indexrelid::regclass) AS indices
FROM pg_index
GROUP BY indrelid, indkey
HAVING count(*) > 1
ORDER BY 1;

\echo '## 6. Cobertura de comentários (tabela e colunas)'
SELECT c.relname AS tabela,
       (obj_description(c.oid) IS NOT NULL) AS comentada,
       (SELECT count(*) FROM pg_attribute a
         WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped) AS colunas,
       (SELECT count(*) FROM pg_attribute a
         WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
           AND col_description(c.oid, a.attnum) IS NOT NULL) AS comentadas
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'r'
ORDER BY 1;

\echo '## 7. Constraints por tipo (p = PK, f = FK, c = CHECK, u = UNIQUE)'
SELECT contype, count(*) FROM pg_constraint GROUP BY 1 ORDER BY 2 DESC;

\echo '## 8. Recursos do Postgres em uso'
SELECT 'funções' AS recurso, count(*) FROM pg_proc p
  JOIN pg_namespace n ON n.oid = p.pronamespace WHERE n.nspname = 'public'
UNION ALL SELECT 'views', count(*) FROM pg_views WHERE schemaname = 'public'
UNION ALL SELECT 'materialized views', count(*) FROM pg_matviews WHERE schemaname = 'public'
UNION ALL SELECT 'triggers', count(*) FROM pg_trigger WHERE NOT tgisinternal
UNION ALL SELECT 'sequences', count(*) FROM information_schema.sequences WHERE sequence_schema = 'public'
UNION ALL SELECT 'enums', count(*) FROM pg_type t
  JOIN pg_namespace n ON n.oid = t.typnamespace WHERE t.typtype = 'e' AND n.nspname = 'public'
UNION ALL SELECT 'extensions', count(*) FROM pg_extension;

\echo '## 9. Distribuição de tipos de coluna'
SELECT data_type, count(*) FROM information_schema.columns
WHERE table_schema = 'public' GROUP BY 1 ORDER BY 2 DESC;

\echo '## 10. Mesma coluna com tipos diferentes entre tabelas (padronização)'
SELECT column_name, array_agg(DISTINCT data_type) AS tipos, count(*) AS ocorrencias
FROM information_schema.columns
WHERE table_schema = 'public'
GROUP BY 1
HAVING count(DISTINCT data_type) > 1
ORDER BY 1;

\echo '## 11. Nomenclatura suspeita (maiúsculas ou palavra reservada)'
SELECT table_name, column_name
FROM information_schema.columns
WHERE table_schema = 'public'
  AND (column_name ~ '[A-Z]'
       OR column_name IN ('user','order','desc','group','limit','offset','check',
                          'default','references','unique','primary','key','value',
                          'type','name','level','class','int','con','date','time','timestamp','position'))
ORDER BY 1, 2;

\echo '## 12. Identidade vs serial'
SELECT table_name, column_name, is_identity, column_default
FROM information_schema.columns
WHERE table_schema = 'public' AND (column_default LIKE 'nextval%' OR is_identity = 'YES')
ORDER BY 1, 2;

\echo '## 13. Ações de FK (a=no action, r=restrict, c=cascade, n=set null, d=set default)'
SELECT conrelid::regclass AS tabela, conname AS constraint, confdeltype, convalidated
FROM pg_constraint WHERE contype = 'f' ORDER BY 1, 2;

\echo '## 14. Tabelas grandes com muito seq_scan (falta de índice de leitura)'
SELECT relname AS tabela, seq_scan, idx_scan, n_live_tup AS linhas
FROM pg_stat_user_tables
WHERE n_live_tup > 1000
ORDER BY seq_scan DESC
LIMIT 10;

\echo '## 15. Bloat/estatística velha'
SELECT relname AS tabela, last_vacuum, last_autovacuum, last_analyze, last_autoanalyze
FROM pg_stat_user_tables
WHERE n_live_tup > 1000
ORDER BY relname;
