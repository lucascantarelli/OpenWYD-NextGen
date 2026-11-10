-- 0039_classificacao_e_teto: baldes do catálogo e teto de pilha derivado.
--
-- Contexto (2026-11-10): 1.532 linhas de `item_catalog` sem `kind` — e também sem `stackable`
-- e sem `EF_AMOUNT`. Elas misturam três coisas (ruído de importação, item real não
-- classificado, placeholder), e a regra "sem classificação ⇒ teto 1" fazia item real
-- (Antídoto 416, Resto de Oriharucon 419) não empilhar.
--
-- Duas views derivadas (não materializadas: nunca ficam velhas em relação ao catálogo):
--   item_bucket     — cada linha em `ruido` / `real` / `placeholder`, com motivo auditável;
--   item_stack_cap  — teto por item (120 aglutinável, 1 resto), `contained` = `EF_AMOUNT`
--                     (conteúdo de pack) e a fonte de cada decisão.
--
-- Critérios (nenhum item real é rebaixado sem evidência):
--   placeholder  nome normalizado em `noname` / `not used` / `n a` — é como o próprio
--                cliente marca o slot vazio;
--   ruido        nome igual a NPC, região, classe ou template de monstro, E preço 0, E
--                unique_id 0, E item não usado em loja, prêmio de nível, seed de mundo,
--                doação, recompensa diária, override de drop, refino de gema, equipamento
--                de monstro ou teleporte;
--   real         todo o resto — inclusive o que não tem uso conhecido (conservador: um item
--                real rebaixado a ruído some do jogo; um ruído tratado como real ocupa um
--                slot a mais).
--
-- Os nomes externos e o conjunto de itens "usados" entram como **conjuntos pré-computados**
-- (`nome_externo`, `usado`), não como `EXISTS` correlacionado: a versão correlacionada fazia
-- `content_fold` por par (3.220 itens × ~8.000 nomes) e levava minutos. Aqui é um join.
--
-- `item_catalog.fields` é ARRAY plano `[índice, nome, peso, "0.0.0.0.0", flag, preço, 0, 0, 0,
-- "EF_*", valor, ...]`, não objeto: `fields->>'EF_AMOUNT'` devolve NULL em toda linha, e a
-- extração correta é pelo PAR rótulo→valor (`EF_AMOUNT` seguido do valor).

-- migrate:up

CREATE OR REPLACE VIEW item_bucket AS
WITH nomes AS (
    SELECT content_fold(display_name) AS nome, 'npc' AS origem
      FROM npc_definition
     WHERE display_name IS NOT NULL AND display_name <> ''
    UNION
    SELECT content_fold(template_name), 'npc'
      FROM npc_definition
     WHERE template_name IS NOT NULL AND template_name <> ''
    UNION
    SELECT content_fold(template_name), 'npc'
      FROM npc_template
     WHERE template_name IS NOT NULL AND template_name <> ''
    UNION
    SELECT content_fold(name_recovered), 'npc'
      FROM npc_template
     WHERE name_recovered IS NOT NULL AND name_recovered <> ''
    UNION
    SELECT content_fold(name), 'regiao' FROM region
    UNION
    SELECT content_fold(label), 'classe' FROM game_class
    UNION
    SELECT content_fold(template_name), 'monstro'
      FROM mob_template_stat
     WHERE template_name IS NOT NULL AND template_name <> ''
),
nome_externo AS (
    SELECT nome, min(origem) AS origem FROM nomes GROUP BY nome
),
usado AS (
    SELECT item_index FROM npc_shop_item
    UNION SELECT item_index FROM level_item
    UNION SELECT item_index FROM world_item_seed
    UNION SELECT item_index FROM donate_shop_item
    UNION SELECT item_index FROM daily_reward_item
    UNION SELECT item_index FROM drop_rate_override
    UNION SELECT item_index FROM refine_gem_effect
    UNION SELECT item_index FROM mob_template_equip
    UNION SELECT req_item_index FROM teleport_route WHERE req_item_index IS NOT NULL
)
SELECT c.item_index,
       CASE
           WHEN c.name_fold IN ('noname', 'not used', 'n a') THEN 'placeholder'
           WHEN u.item_index IS NOT NULL THEN 'real'
           WHEN n.nome IS NOT NULL AND c.price = 0 AND c.unique_id = 0 THEN 'ruido'
           ELSE 'real'
       END AS bucket,
       CASE
           WHEN c.name_fold IN ('noname', 'not used', 'n a') THEN 'nome-placeholder'
           WHEN u.item_index IS NOT NULL THEN 'usado-em-tabela'
           WHEN n.nome IS NOT NULL AND c.price = 0 AND c.unique_id = 0
               THEN 'nome-de-' || n.origem
           WHEN c.price > 0 OR c.unique_id > 0 THEN 'preco-ou-unique'
           ELSE 'sem-uso-conhecido'
       END AS reason
  FROM item_catalog c
  LEFT JOIN nome_externo n ON n.nome = c.name_fold
  LEFT JOIN usado u ON u.item_index = c.item_index;

CREATE OR REPLACE VIEW item_stack_cap AS
WITH pares AS (
    -- Uma passada no array; `lead` entrega o valor que segue cada rótulo. Unir o array a si
    -- mesmo (`ord + 1`) fazia produto cartesiano de ~20 × 20 elementos por item e levava
    -- minutos com o `item_stack_cap` dentro de um join.
    SELECT c.item_index,
           e.value,
           lead(e.value) OVER (PARTITION BY c.item_index ORDER BY e.ord) AS seguinte
      FROM item_catalog c,
           jsonb_array_elements_text(c.fields) WITH ORDINALITY e(value, ord)
),
amt AS (
    SELECT item_index,
           max(CASE WHEN value = 'EF_AMOUNT' THEN seguinte::int END) AS contained
      FROM pares
     GROUP BY item_index
)
SELECT c.item_index,
       CASE
           WHEN c.stackable THEN 120
           WHEN c.stackable = false THEN 1
           WHEN b.bucket <> 'real' THEN 1
           WHEN a.contained > 0 THEN 120
           WHEN c.name_fold LIKE '%poeira%'
             OR c.name_fold LIKE 'resto%'
             OR c.name_fold LIKE 'fragmento%' THEN 120
           ELSE 1
       END AS cap,
       a.contained,
       CASE
           WHEN c.stackable THEN 'catalogo:aglutinavel'
           WHEN c.stackable = false THEN 'catalogo:nao_aglutinavel'
           WHEN b.bucket = 'placeholder' THEN 'catalogo:placeholder'
           WHEN b.bucket = 'ruido' THEN 'catalogo:ruido'
           WHEN a.contained > 0 THEN 'legado:ef_amount'
           WHEN c.name_fold LIKE '%poeira%'
             OR c.name_fold LIKE 'resto%'
             OR c.name_fold LIKE 'fragmento%' THEN 'familia:material'
           ELSE 'default:sem-prova'
       END AS source
  FROM item_catalog c
  JOIN item_bucket b ON b.item_index = c.item_index
  LEFT JOIN amt a ON a.item_index = c.item_index;

-- migrate:down
DROP VIEW IF EXISTS item_stack_cap;
DROP VIEW IF EXISTS item_bucket;
