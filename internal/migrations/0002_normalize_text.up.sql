-- Normalização de texto das tabelas de conteúdo (06/11/2026).
--
-- O que faz: aplica a mesma regra do pacote `internal/textnorm` aos dados que já
-- estão no banco — NFC (forma composta), sem espaço nas pontas e com espaço
-- interno colapsado. É idempotente: rodar de novo não muda nada.
--
-- Por que migration e não só código: `internal/textnorm` protege o que ENTRA
-- (importcontent). Um banco restaurado de dump antigo, ou um seed regerado antes
-- do pacote existir, continuaria com NFD/espaço invisível. A migration fecha o
-- que já estava gravado.
--
-- O que NÃO está aqui, de propósito:
--   - Reparo de duplo encoding ("AmÃ©m" → "Amém"): é julgamento sobre bytes, não
--     cabe em SQL. Fica no importador (`textnorm.RepairMojibake`), com teste.
--   - Tabelas de conta/personagem/auditoria: texto de jogador é dado do usuário,
--     não conteúdo nosso; normalizar ali mudaria o que a pessoa digitou.
--   - CHECK CONSTRAINT: só entra depois de uma rodada com dados reais de
--     produção; hoje a garantia é o importador + este saneamento.

DO $$
DECLARE
  tabela text;
  coluna text;
  regra text;
BEGIN
  FOREACH tabela IN ARRAY ARRAY[
    'item_catalog',
    'skill_catalog',
    'npc_template',
    'npc_definition',
    'npc_shop_item',
    'world_item_seed',
    'npc_config_meta',
    'world_event_meta'
  ]
  LOOP
    FOR coluna IN
      SELECT c.column_name
      FROM information_schema.columns c
      WHERE c.table_schema = 'public'
        AND c.table_name = tabela
        AND c.data_type IN ('text', 'character varying')
    LOOP
      regra := format(
        'btrim(regexp_replace(normalize(%I, NFC), ''\s+'', '' '', ''g''))',
        coluna
      );

      EXECUTE format(
        'UPDATE %I SET %I = %s WHERE %I IS NOT NULL AND %I IS DISTINCT FROM %s',
        tabela, coluna, regra, coluna, coluna, regra
      );
    END LOOP;
  END LOOP;
END $$;
