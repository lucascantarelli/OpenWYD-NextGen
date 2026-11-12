-- 0049_seed_campo_sephira: NPC Sephira +9 do Campo de Treinamento (D-8, 2026-11-11).
--
-- Complementa o seed `0046` (que já está aplicado e não pode ser reescrito sem quebrar o
-- checksum do `sqlx`). A decisão D-8 pede o NPC de armas Sephira +9 ao lado do Unicorn
-- oficial; o template `Refinacoes` é o NPC de refino real do snapshot, então o fluxo de
-- refino do +9 encontra uma definição de NPC existente, sem blob inventado.
--
-- Id 6311 fica logo depois dos dez NPCs do Campo (6301..6310).

-- migrate:up
INSERT INTO npc_definition
    (id, slug, template_name, display_name, enabled, map_id, pos_x, pos_y, merchant, origin)
VALUES
    (6311, 'campo-sephira-9', 'Refinacoes', 'Sephira +9', true, 0, 2133, 2040, 0, 'custom')
ON CONFLICT (id) DO NOTHING;

-- migrate:down
DELETE FROM npc_definition WHERE id = 6311 AND slug = 'campo-sephira-9';
