-- 0008_seed_taxonomia: sistemas, classes, linhagens e categorias de item (docs/wyd/).
-- Convenção: -- migrate:up / -- migrate:down.
-- Dados curados e citados em docs/wyd/{sistemas,classes,itens}.md. Idempotente.

-- migrate:up
INSERT INTO game_system (code, label, category, description) VALUES
    ('progressao', 'Progressão', 'progressao', 'Level, EXP, pontos de status/aprendizado/skill, fama e transformação Arch.'),
    ('combate', 'Combate', 'combate', 'Dano físico e mágico, PvE e PvP (zonas PK, guerra de guilda, castelo, reinos).'),
    ('refino', 'Refino', 'combate', 'Refino +0..+15 com pó de Ori/Lac, Pedra do Sábio no +10 e combinação celestial/god.'),
    ('replecao', 'Repleção', 'combate', 'Pedras de repleção A-C em armaduras: gera opção adicional aleatória.'),
    ('mix', 'Mix', 'economia', 'Combinação de itens em receitas (Mixlist do cliente).'),
    ('craft', 'Craft', 'economia', 'Combinação de joias e craft de arma/armadura (linhagem Trade da Huntress).'),
    ('drop', 'Drop', 'economia', 'Drop roll por item: valor do item comparado ao sorteio (1000 = 0,1%).'),
    ('quest', 'Quests', 'progressao', 'Missões, chaves e pergaminhos (ex.: Pista das Runas, quests de Arch).'),
    ('diaria', 'Recompensa diária', 'progressao', 'Recompensa diária por conta/personagem (daily_reward).'),
    ('evento', 'Eventos', 'evento', 'Kefra semanal, cupons, tickets e itens comemorativos.'),
    ('guilda', 'Guilda', 'social', 'Criação, ranks, membros e relações entre guildas.'),
    ('guerra_guilda', 'Guerra de guilda', 'social', 'Confronto entre guildas e alianças.'),
    ('castelo', 'Castelo', 'social', 'Disputa da Coroa de Athila; até 1000 jogadores simultâneos.'),
    ('reino', 'Reinos', 'social', 'Hekalotia e Akeronia; entrada no nível 220 com safiras e mantos de reino.'),
    ('loja', 'Loja de NPC', 'economia', 'Mercadores: compra, venda e restoque (npc_shop_item).'),
    ('donate', 'Donate', 'economia', 'Loja premium e créditos de donate (donate_shop_item).'),
    ('teleporte', 'Teleporte', 'mundo', 'Rotas de teleporte e requisitos (teleport_route).'),
    ('armazem', 'Armazém', 'economia', 'Baú da conta (vault_items).'),
    ('montaria', 'Montaria', 'progressao', 'Montarias e seus atributos.'),
    ('mundo', 'Mundo', 'mundo', 'Mapas, regiões, atributos de terreno e minimapa.')
ON CONFLICT (code) DO NOTHING;

INSERT INTO game_class (code, label, role, description) VALUES
    ('transknight', 'TransKnight', 'tanque ou dano físico', 'Guardião do Altar dos Deuses em Armia; armas de uma e duas mãos.'),
    ('foema', 'Foema', 'dano mágico e controle', 'Maga elemental; magia branca, negra e especial.'),
    ('beastmaster', 'BeastMaster', 'invocador híbrido', 'Fera e magia natural; invocações até FinalVerid.'),
    ('huntress', 'Huntress', 'dano à distância', 'Arco, garra e armadilhas; sobrevivência, comércio e captura.'),
    ('arch', 'Arch', 'classe transcendente', 'Quatro classes Arch após duas quests; sem elmo, armas Arch.'),
    ('common', 'Comum', 'todas as classes', 'Skill/item sem restrição de classe.')
ON CONFLICT (code) DO NOTHING;

INSERT INTO skill_lineage (code, class_code, label, focus) VALUES
    ('faith', 'transknight', 'Faith', 'Magia sagrada, área e buff de HP.'),
    ('trans', 'transknight', 'Trans', 'Dano físico direto e buffs de ataque.'),
    ('magic_sword', 'transknight', 'Magic Sword', 'Híbrido arma+magia e controle.'),
    ('white_magic', 'foema', 'White Magic', 'Cura, proteção e invulnerabilidade.'),
    ('black_magic', 'foema', 'Black Magic', 'Dano elemental e debuffs.'),
    ('special_magic', 'foema', 'Special Magic', 'Utilidade, remoção de buff e controle.'),
    ('elemental', 'beastmaster', 'Elemental', 'Dano elemental e escudo de fogo.'),
    ('evocation', 'beastmaster', 'Evocation', 'Invocações de feras.'),
    ('nature', 'beastmaster', 'Nature', 'Reflexão, veneno e controle.'),
    ('survival', 'huntress', 'Survival', 'Dano com arco e sobrevivência.'),
    ('trade', 'huntress', 'Trade', 'Combinação de joias, comércio e craft.'),
    ('capture', 'huntress', 'Capture', 'Laços, controle e captura.'),
    ('common', 'common', 'Comum', 'Sem linhagem específica.')
ON CONFLICT (code) DO NOTHING;

INSERT INTO item_kind (code, label, slot, description) VALUES
    ('weapon', 'Arma', 'mão', 'Uma ou duas mãos; refina, não repleta.'),
    ('shield', 'Escudo', 'mão secundária', 'Defesa; refina, não repleta.'),
    ('armor', 'Armadura', 'corpo', 'Refina e aceita repleção.'),
    ('helmet', 'Elmo', 'cabeça', 'Defesa; Arch não usa.'),
    ('boots', 'Botas', 'pés', 'Defesa.'),
    ('gloves', 'Luvas', 'mãos', 'Defesa.'),
    ('cloak', 'Manto/Capa', 'costas', 'Defesa; mantos de reino (Hekalotia/Akeronia).'),
    ('accessory', 'Acessório', 'dedo/pescoço/pulso', 'Bônus; não refina nem repleta.'),
    ('consumable', 'Consumível', 'mochila', 'Poção, kit de cura, elixir.'),
    ('material', 'Material', 'mochila', 'Minério, sucata, pó, pedra, essência, joia.'),
    ('quest', 'Item de quest', 'mochila', 'Pergaminho, chave, item de missão.'),
    ('cash', 'Item premium', 'mochila', 'Item de donate/loja premium.'),
    ('ammo', 'Munição', 'mochila', 'Flechas.'),
    ('container', 'Caixa/Pacote', 'mochila', 'Contêiner de itens.'),
    ('cosmetic', 'Cosmético', '—', 'Asa, aura, visual.'),
    ('misc', 'Diversos', '—', 'Sem categoria conhecida.')
ON CONFLICT (code) DO NOTHING;

-- migrate:down
DELETE FROM item_kind;
DELETE FROM skill_lineage;
DELETE FROM game_class;
DELETE FROM game_system;
