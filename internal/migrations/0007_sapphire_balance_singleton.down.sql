-- Rollback da 0007: remove apenas a linha ainda intacta nos defaults. Se uma
-- compra já avançou a revisão ou os custos, o preço é dado de jogo e não deve
-- ser apagado por um down de migration.
DELETE FROM sapphire_balance
 WHERE singleton
   AND version = 1
   AND hekalotia_cost = 8
   AND akelonia_cost = 8;
