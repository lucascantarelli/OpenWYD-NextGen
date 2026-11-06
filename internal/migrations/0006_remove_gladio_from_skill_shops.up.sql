-- Limpeza fina do Gladio (item 867): os NPCs de skill 49 e 50 vendiam o item no
-- slot 17, que é o separador entre as famílias de magia (0-7, 9-16, 18-25) e
-- portanto deve ficar vazio. O defeito vinha do seed importado; o seed curado já
-- foi corrigido no mesmo commit, e esta migration conserta a base viva.
DELETE FROM npc_shop_item
 WHERE npc_id IN (49, 50)
   AND slot = 17
   AND item_index = 867;
