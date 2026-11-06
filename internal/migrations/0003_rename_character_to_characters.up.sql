-- DB-04: `character` é palavra reservada do SQL e obriga aspas em toda query, ao
-- contrário do resto do schema (`account`, `guild`, `npc_definition`...). Renomear
-- agora, antes de existir dado de jogador que importe preservar em produção, tira o
-- nome reservado do código, do BFF e de toda query futura.
--
-- Os índices acompanham o nome da tabela; as constraints que apontam para ela
-- (`*_character_id_fkey`) ficam como estão: renomeá-las mexeria em tabelas de
-- terceiros sem ganho de leitura.
ALTER TABLE public."character" RENAME TO characters;

ALTER INDEX public.character_account_id_idx RENAME TO characters_account_id_idx;
ALTER INDEX public.character_exp_ranking_idx RENAME TO characters_exp_ranking_idx;
ALTER INDEX public.character_name_class_master_idx RENAME TO characters_name_class_master_idx;
