-- 0050_seed_correcao_catalogo_game12_15: fecha a classificação de aglutinação (GAME-12/GAME-15).
--
-- O gerador (`tools/content/gen-seed.mjs`) foi corrigido para a próxima regeneração; o seed
-- `0018` versionado permanece histórico porque o ledger recusa reescrever migration aplicada.
-- Esta migração aplica a curadoria em qualquer banco e fecha o `stackable` nulo usando a prova
-- que a view `item_stack_cap` já carrega. 406 é mana (EF_MP 200, preço igual ao 401), 416 é
-- consumível, 419/420 são materiais e 3385/2433–2435/4045 são placeholders explícitos:
--   * item sem `kind` com `EF_AMOUNT > 0` empilha (pack);
--   * os demais itens sem prova ficam em `false` (teto 1), nunca em um `true` sem fonte.
--
-- `kind` continua NULL quando não há prova; a classificação de teto não inventa categoria.
--
-- O `down` restaura o estado anterior à correção (linhas nulas + valores do seed antigo). O
-- `down` do `0018` regenerado derruba a tabela, então o ciclo completo continua fechando em 0.

-- migrate:up

UPDATE item_catalog
   SET name = 'Poção_de_Mana_M.',
       kind = 'consumable',
       stackable = true,
       classification_source = 'curadoria:game12_15',
       confidence = 100
 WHERE item_index = 406;

UPDATE item_catalog
   SET kind = 'consumable',
       stackable = true,
       classification_source = 'curadoria:game12_15',
       confidence = 100
 WHERE item_index = 416;

UPDATE item_catalog
   SET kind = 'material',
       stackable = true,
       classification_source = 'curadoria:game12_15',
       confidence = 100
 WHERE item_index IN (419, 420);

-- Placeholder é placeholder mesmo com preço/EF_AMOUNT: o nome é a marca explícita do slot.
UPDATE item_catalog
   SET stackable = false,
       classification_source = 'curadoria:game12_15',
       confidence = 100
 WHERE item_index IN (3385, 2433, 2434, 2435, 4045);

-- Materializa a view antes do UPDATE: `item_stack_cap` lê `item_catalog`, e o CTE
-- MATERIALIZED evita que a atualização mude o conjunto que está sendo lido.
WITH caps AS MATERIALIZED (
    SELECT item_index, cap FROM item_stack_cap
)
UPDATE item_catalog c
   SET stackable = (caps.cap = 120),
       classification_source = CASE
           WHEN caps.cap = 120 THEN 'derivado:ef_amount'
           ELSE 'derivado:conservador'
       END,
       confidence = 80
  FROM caps
 WHERE caps.item_index = c.item_index
   AND c.stackable IS NULL;

-- migrate:down

UPDATE item_catalog
   SET stackable = NULL,
       classification_source = 'legacy',
       confidence = NULL
 WHERE classification_source IN ('derivado:ef_amount', 'derivado:conservador');

UPDATE item_catalog
   SET name = 'Poção_de_Cura_M.',
       kind = 'consumable',
       stackable = true,
       classification_source = 'rule:name_v1',
       confidence = 60
 WHERE item_index = 406;

UPDATE item_catalog
   SET kind = NULL,
       stackable = NULL,
       classification_source = 'legacy',
       confidence = NULL
 WHERE item_index IN (416, 419, 420, 3385, 2433, 2434, 2435, 4045);
