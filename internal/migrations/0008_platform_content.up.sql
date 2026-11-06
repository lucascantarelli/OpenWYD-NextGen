-- Issue 59: os quatro arquivos de configuração de jogo sem leitor Go vindo para
-- o banco, mais Regions.txt (52 retângulos) e a proveniência de import.
--
-- Regra de modelagem: só vira coluna o que o arquivo prova. Onde a semântica
-- ainda é UNVERIFIED (o cabeçalho do Guard.txt declara 13 campos por regra, o
-- arquivo traz 8), o dado fica em `rules`/`raw` em vez de colunas inventadas.
-- Nada aqui implementa comportamento: guarda, item por nível e restock de
-- mercador continuam sem consumidor até a issue de cada sistema.

CREATE TABLE IF NOT EXISTS content_source_file (
    path        text PRIMARY KEY,
    sha256      text NOT NULL,
    size_bytes  bigint NOT NULL,
    rows        integer NOT NULL,
    imported_at timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE content_source_file IS
    'Proveniência do import: hash e contagem por arquivo de conteúdo (issue 50, passo 1).';

-- Guard.txt: "<índice>,<gerador>" seguido de blocos [a,b,c,d,e,f,g,h].
-- Os 8 campos não casam com o cabeçalho legado (13), então ficam como jsonb.
CREATE TABLE IF NOT EXISTS guard_spawn (
    guard_index     smallint PRIMARY KEY,
    generator_index integer NOT NULL,
    rules           jsonb NOT NULL DEFAULT '[]'::jsonb,
    raw             text NOT NULL DEFAULT ''
);

-- LevelItem.txt: "<sequência> <nível> <classe> <tipo> <item> [extras...]".
-- `extra` guarda os números depois do item (refino/efeito não documentados).
CREATE TABLE IF NOT EXISTS level_item (
    seq        smallint NOT NULL,
    level      smallint NOT NULL,
    class_id   smallint NOT NULL,
    type_id    smallint NOT NULL,
    item_index integer NOT NULL,
    extra      jsonb NOT NULL DEFAULT '[]'::jsonb,
    raw        text NOT NULL DEFAULT '',
    -- A chave é a POSIÇÃO no arquivo. O arquivo repete blocos inteiros (a linha
    -- 13 e a 256 têm o mesmo seq 0 / nível 29 / item 1147), então (class, seq)
    -- não identifica linha — 318 linhas para 159 pares distintos. Importar por
    -- posição preserva o arquivo e continua idempotente.
    ordinal    smallint PRIMARY KEY
);
CREATE INDEX IF NOT EXISTS level_item_level_class_idx ON level_item (level, class_id);

-- MobMerc.txt: blocos "#[n]" com MobName/GenerateIndex/Item_n/RenewTime/RebornTime.
CREATE TABLE IF NOT EXISTS mob_merc (
    generator_index integer PRIMARY KEY,
    template_name   text NOT NULL DEFAULT '',
    renew_seconds   integer NOT NULL DEFAULT 0,
    reborn_seconds  integer NOT NULL DEFAULT 0,
    items           jsonb NOT NULL DEFAULT '[]'::jsonb,
    raw             text NOT NULL DEFAULT ''
);

-- gameconfig.txt: pares chave/valor por seção. `kind` distingue int, list,
-- string e raw (seção Treasure, guardada inteira sem interpretação).
CREATE TABLE IF NOT EXISTS game_setting (
    key        text PRIMARY KEY,
    section    text NOT NULL DEFAULT '',
    kind       text NOT NULL DEFAULT 'int',
    value_int  bigint,
    value_text text
);

-- Matriz "Item Drop Bonus" do gameconfig: 4 linhas por 16 colunas no arquivo
-- (medido em 06/11), não 8x8 como supunha a issue. Índices preservam a ordem
-- do arquivo e a contagem é conferida pelo -check.
CREATE TABLE IF NOT EXISTS game_drop_bonus (
    row_index smallint NOT NULL,
    col_index smallint NOT NULL,
    pct       smallint NOT NULL,
    PRIMARY KEY (row_index, col_index)
);

-- Regions.txt: "x1, y1, x2, y2 = Nome".
CREATE TABLE IF NOT EXISTS region (
    name    text PRIMARY KEY,
    x1      integer NOT NULL,
    y1      integer NOT NULL,
    x2      integer NOT NULL,
    y2      integer NOT NULL,
    source  text NOT NULL DEFAULT 'Regions.txt'
);

COMMENT ON TABLE region IS
    'Retângulos de Regions.txt. O worldatlas ainda lê o arquivo; passa a ler aqui na próxima fatia.';
