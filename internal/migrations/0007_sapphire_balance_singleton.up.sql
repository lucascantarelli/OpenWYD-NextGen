-- A tabela nasce com os defaults (revisão 1, custos 8/8), mas o baseline não
-- inseria a linha única. Sem ela, QuoteKingdomCape falhava com "no rows" e a
-- compra do manto não tinha revisão para comparar. ON CONFLICT preserva um
-- preço já ajustado por um banco existente.
INSERT INTO sapphire_balance (singleton)
VALUES (TRUE)
ON CONFLICT (singleton) DO NOTHING;
