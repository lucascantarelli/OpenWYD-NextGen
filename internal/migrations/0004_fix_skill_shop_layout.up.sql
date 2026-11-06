-- Corrige a loja dos NPCs de skill que traziam item no lugar de skill. O template
-- `Foema_Ancian` (ids 152 e 263) vendia 5024..5030 + `Garnet` (2444) no slot 7 e só
-- 5042..5046 da magia especial; `Cap.Cavaleiros` (id 37) vendia 14 skills espalhadas
-- entre poções e equipamento. O layout correto é o dos NPCs de skill íntegros do
-- mesmo conjunto: 24 skills em 3 famílias de 8, slots 0-7, 9-16 e 18-25.
--
-- A cópia é feita do NPC de referência (Kara = 23 para a Foema, Cap.Cavaleiros = 30
-- para o TK) em vez de repetir os 24 pares na mão: se o catálogo mudar, a migration
-- não fica com uma segunda cópia do dado. Num banco vazio ela é no-op (as linhas de
-- referência ainda não existem) e os seeds já trazem o layout corrigido.
DELETE FROM npc_shop_item WHERE npc_id IN (37, 152, 263);

INSERT INTO npc_shop_item (npc_id, slot, item_index, quantity, eff1, effv1, eff2, effv2, eff3, effv3)
SELECT t.npc, s.slot, s.item_index, s.quantity, s.eff1, s.effv1, s.eff2, s.effv2, s.eff3, s.effv3
FROM npc_shop_item s
CROSS JOIN (VALUES (152::bigint), (263::bigint)) AS t(npc)
WHERE s.npc_id = 23;

INSERT INTO npc_shop_item (npc_id, slot, item_index, quantity, eff1, effv1, eff2, effv2, eff3, effv3)
SELECT 37, s.slot, s.item_index, s.quantity, s.eff1, s.effv1, s.eff2, s.effv2, s.eff3, s.effv3
FROM npc_shop_item s
WHERE s.npc_id = 30;
