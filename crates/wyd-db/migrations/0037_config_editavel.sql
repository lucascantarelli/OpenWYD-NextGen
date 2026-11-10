-- 0037_config_editavel: taxa editável pelo portal sem perder o seed, revisão de config e
-- regra de refino de acessório (sistema posterior ao 7662).
--
-- Por que override em vez de UPDATE na tabela semeada: o seed continua sendo a fonte
-- histórica (0022/0025/0026/0027); a edição do staff vira uma segunda camada com autor e
-- fonte, e `reset` volta ao default documentado sem reescrever a migração. O loader do
-- servidor lê `COALESCE(override, base)` — nenhuma constante no core.
--
-- Fonte do refino de acessório: WYD Global/Bahamut (guia aberto em
-- https://wydglobal.raidhut.com/pt-br/3390) e a central Kersef
-- (https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/21072), que declara
-- acessórios não refináveis no sistema original +0..+11. Amuleto/ankh/pedra espiritual
-- não têm regra publicada na era oficial: a tabela aceita a linha, o seed não inventa.

-- migrate:up
-- Override de passo do refino com pó/composição (escala 0..10000 = 0..100%).
CREATE TABLE refine_rate_override (
    step           SMALLINT PRIMARY KEY CHECK (step BETWEEN 1 AND 15),
    rate_bps       INTEGER NOT NULL CHECK (rate_bps BETWEEN 0 AND 10000),
    content_source TEXT NOT NULL CHECK (content_source <> ''),
    notes          TEXT NOT NULL DEFAULT '',
    updated_by     BIGINT NOT NULL CHECK (updated_by > 0),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE refine_rate_override IS
    'Edição do staff sobre refine_rate (seed 0022); reset = apagar a linha.';

-- Override da célula de SancRate (pó × refino atual).
CREATE TABLE sanc_rate_override (
    powder         TEXT NOT NULL CHECK (powder IN ('PO', 'PL', 'AMAGO')),
    refine_index   SMALLINT NOT NULL CHECK (refine_index BETWEEN 0 AND 15),
    percent        SMALLINT NOT NULL CHECK (percent BETWEEN 0 AND 100),
    content_source TEXT NOT NULL CHECK (content_source <> ''),
    notes          TEXT NOT NULL DEFAULT '',
    updated_by     BIGINT NOT NULL CHECK (updated_by > 0),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (powder, refine_index)
);

-- Override de valor de composição (taxa percentual ou custo em moeda).
CREATE TABLE combine_value_override (
    family         TEXT NOT NULL CHECK (family <> ''),
    key            TEXT NOT NULL CHECK (key <> ''),
    value          INTEGER NOT NULL CHECK (value >= 0),
    unit           TEXT NOT NULL CHECK (unit IN ('percent', 'coin')),
    content_source TEXT NOT NULL CHECK (content_source <> ''),
    notes          TEXT NOT NULL DEFAULT '',
    updated_by     BIGINT NOT NULL CHECK (updated_by > 0),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (family, key)
);

-- Refino de acessório: uma linha por passo. `recipe` separa receitas do mesmo acessório
-- (Bahamut tem 1 Lac, 100 Lac e Torvald, com taxas e falhas diferentes).
-- `fail_mode`: keep = mantém o refino; reset = volta a +0; destroy = item perdido.
-- `fail_bonus_*` é a recompensa de consolação na falha (ex.: Bahamut Tear a 10%).
-- `material_item` não tem FK de propósito: cristal/poeira de servidor posterior pode não
-- existir no catálogo 7662; `material_name` preserva o nome publicado.
CREATE TABLE accessory_refine_rule (
    kind             TEXT NOT NULL CHECK (
                         kind IN ('earring', 'necklace', 'belt', 'amulet', 'ankh', 'spirit_stone')
                     ),
    recipe           TEXT NOT NULL CHECK (recipe <> ''),
    step_from        SMALLINT NOT NULL CHECK (step_from BETWEEN 0 AND 14),
    step_to          SMALLINT NOT NULL CHECK (step_to = step_from + 1),
    rate_bps         INTEGER NOT NULL CHECK (rate_bps BETWEEN 0 AND 10000),
    fail_mode        TEXT NOT NULL CHECK (fail_mode IN ('keep', 'reset', 'destroy')),
    fail_bonus_item  INTEGER,
    fail_bonus_bps   INTEGER NOT NULL DEFAULT 0 CHECK (fail_bonus_bps BETWEEN 0 AND 10000),
    gold_cost        BIGINT NOT NULL DEFAULT 0 CHECK (gold_cost >= 0),
    material_item    INTEGER,
    material_name    TEXT NOT NULL DEFAULT '',
    material_qty     SMALLINT NOT NULL DEFAULT 1 CHECK (material_qty > 0),
    content_source   TEXT NOT NULL CHECK (content_source <> ''),
    confidence       SMALLINT NOT NULL DEFAULT 80 CHECK (confidence BETWEEN 0 AND 100),
    notes            TEXT NOT NULL DEFAULT '',
    updated_by       BIGINT,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, recipe, step_from)
);

COMMENT ON TABLE accessory_refine_rule IS
    'Refino de acessório (brinco/colar/cinto/amuleto/ankh/pedra): passo, taxa e falha por receita.';

-- Revisão global da configuração: o servidor compara no boot/carga periódica e recarrega
-- fora do tick. Toda escrita de config faz UPDATE nesta linha na mesma transação.
CREATE TABLE config_revision (
    id         SMALLINT PRIMARY KEY CHECK (id = 1),
    version    BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by BIGINT NOT NULL DEFAULT 0 CHECK (updated_by >= 0),
    CONSTRAINT config_revision_singleton CHECK (id = 1)
);

INSERT INTO config_revision (id, version) VALUES (1, 1);

-- Trilha por mudança de configuração: quem, quando, o quê e o valor anterior, para
-- responder "quem mudou a taxa" sem depender do audit_log genérico.
CREATE TABLE config_change (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    domain     TEXT NOT NULL CHECK (domain <> ''),
    target     TEXT NOT NULL CHECK (target <> ''),
    old_value  TEXT NOT NULL DEFAULT '',
    new_value  TEXT NOT NULL DEFAULT '',
    actor_id   BIGINT NOT NULL CHECK (actor_id > 0),
    reason     TEXT NOT NULL DEFAULT '',
    at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX config_change_at_idx ON config_change (at DESC);

-- migrate:down
DROP INDEX config_change_at_idx;
DROP TABLE config_change;
DROP TABLE config_revision;
DROP TABLE accessory_refine_rule;
DROP TABLE combine_value_override;
DROP TABLE sanc_rate_override;
DROP TABLE refine_rate_override;
