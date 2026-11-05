--
-- PostgreSQL database dump
--


-- Dumped from database version 17.11
-- Dumped by pg_dump version 17.11

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
-- (linha `SELECT pg_catalog.set_config('search_path', '', false);` do pg_dump
-- removida de propósito: é SET de sessão, sobrevive à transação da migration e
-- deixaria o `INSERT INTO schema_migrations` do runner sem schema. O dump
-- qualifica os objetos, então não precisa dela.)
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: item_owner_kind; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.item_owner_kind AS ENUM (
    'char_equip',
    'char_carry',
    'account_cargo'
);


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: account; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account (
    id bigint NOT NULL,
    name text NOT NULL,
    pass_hash text NOT NULL,
    pin_hash text DEFAULT ''::text NOT NULL,
    block_pass_hash text DEFAULT ''::text NOT NULL,
    real_name text DEFAULT ''::text NOT NULL,
    email text DEFAULT ''::text NOT NULL,
    telephone text DEFAULT ''::text NOT NULL,
    address text DEFAULT ''::text NOT NULL,
    ssn1 integer DEFAULT 0 NOT NULL,
    ssn2 integer DEFAULT 0 NOT NULL,
    donate_balance integer DEFAULT 0 NOT NULL,
    cargo_coin integer DEFAULT 0 NOT NULL,
    is_blocked boolean DEFAULT false NOT NULL,
    year integer DEFAULT 0 NOT NULL,
    year_day integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    role text DEFAULT 'player'::text NOT NULL,
    role_changed_at timestamp with time zone,
    role_changed_by bigint,
    blocked_reason text DEFAULT ''::text NOT NULL
);


--
-- Name: TABLE account; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.account IS 'Conta de jogador: credenciais argon2id, papel administrativo (issue 32) e saldos premium. Dado de jogador — nunca entra em seed.';


--
-- Name: COLUMN account.role; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.account.role IS 'Papel administrativo: player, moderator ou admin (issue 32).';


--
-- Name: COLUMN account.blocked_reason; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.account.blocked_reason IS 'Motivo do último bloqueio (vazio quando a conta está ativa).';


--
-- Name: account_admin_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account_admin_audit (
    id bigint NOT NULL,
    actor_id bigint NOT NULL,
    target_id bigint NOT NULL,
    action text NOT NULL,
    old_value text,
    new_value text,
    reason text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE account_admin_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.account_admin_audit IS 'Alterações de papel/bloqueio/criação de conta (issue 32): valor anterior e novo, autor e motivo.';


--
-- Name: account_admin_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.account_admin_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: account_admin_audit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.account_admin_audit_id_seq OWNED BY public.account_admin_audit.id;


--
-- Name: account_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.account ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.account_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: account_play_code; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account_play_code (
    code_hash bytea NOT NULL,
    account_id bigint NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE account_play_code; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.account_play_code IS 'Código de login de uso único (play code): guarda só o hash, com validade curta.';


--
-- Name: affect; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.affect (
    id bigint NOT NULL,
    character_id bigint NOT NULL,
    type smallint NOT NULL,
    value smallint NOT NULL,
    level integer NOT NULL,
    "time" bigint NOT NULL
);


--
-- Name: TABLE affect; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.affect IS 'Buffs/efeitos ativos de um personagem (Affect[32] do STRUCT_MOB): type, value e o campo time do fio.';


--
-- Name: affect_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.affect ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.affect_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: castle_quest_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.castle_quest_state (
    id smallint NOT NULL,
    level integer DEFAULT 0 NOT NULL,
    time_left integer DEFAULT 0 NOT NULL,
    clear boolean DEFAULT false NOT NULL,
    leader_name text DEFAULT ''::text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT castle_quest_state_id_check CHECK ((id = 1))
);


--
-- Name: TABLE castle_quest_state; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.castle_quest_state IS 'Estado do Castle Quest (CastleQuest.txt): nível, tempo restante, líder e flag de clear. Linha única.';


--
-- Name: character; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public."character" (
    id bigint NOT NULL,
    account_id bigint NOT NULL,
    slot smallint NOT NULL,
    name text NOT NULL,
    class smallint DEFAULT 0 NOT NULL,
    clan smallint DEFAULT 0 NOT NULL,
    guild_id integer DEFAULT 0 NOT NULL,
    guild_level smallint DEFAULT 0 NOT NULL,
    level integer DEFAULT 0 NOT NULL,
    exp bigint DEFAULT 0 NOT NULL,
    coin integer DEFAULT 0 NOT NULL,
    str smallint DEFAULT 0 NOT NULL,
    "int" smallint DEFAULT 0 NOT NULL,
    dex smallint DEFAULT 0 NOT NULL,
    con smallint DEFAULT 0 NOT NULL,
    score_bonus integer DEFAULT 0 NOT NULL,
    special_bonus integer DEFAULT 0 NOT NULL,
    skill_bonus integer DEFAULT 0 NOT NULL,
    max_hp integer DEFAULT 0 NOT NULL,
    max_mp integer DEFAULT 0 NOT NULL,
    hp integer DEFAULT 0 NOT NULL,
    mp integer DEFAULT 0 NOT NULL,
    critical smallint DEFAULT 0 NOT NULL,
    regen_hp integer DEFAULT 0 NOT NULL,
    regen_mp integer DEFAULT 0 NOT NULL,
    resist_fire smallint DEFAULT 0 NOT NULL,
    resist_ice smallint DEFAULT 0 NOT NULL,
    resist_thunder smallint DEFAULT 0 NOT NULL,
    resist_magic smallint DEFAULT 0 NOT NULL,
    learned_skill integer DEFAULT 0 NOT NULL,
    sec_learned_skill integer DEFAULT 0 NOT NULL,
    magic bigint DEFAULT 0 NOT NULL,
    save_x integer DEFAULT 0 NOT NULL,
    save_y integer DEFAULT 0 NOT NULL,
    citizen smallint DEFAULT 0 NOT NULL,
    class_master smallint DEFAULT 0 NOT NULL,
    soul smallint DEFAULT 0 NOT NULL,
    skill_bar smallint[] DEFAULT '{}'::smallint[] NOT NULL,
    short_skill smallint[] DEFAULT '{}'::smallint[] NOT NULL,
    last_city smallint DEFAULT 0 NOT NULL,
    special smallint[] DEFAULT '{}'::smallint[] NOT NULL,
    celestial_lv40 smallint DEFAULT 0 NOT NULL,
    celestial_lv90 smallint DEFAULT 0 NOT NULL,
    celestial_circle smallint DEFAULT 0 NOT NULL,
    fame integer DEFAULT 0 NOT NULL,
    terra_mistica smallint DEFAULT 0 NOT NULL,
    pk_point smallint DEFAULT 75 NOT NULL,
    guilty smallint DEFAULT 0 NOT NULL,
    cur_kill smallint DEFAULT 0 NOT NULL,
    tot_kill integer DEFAULT 0 NOT NULL,
    arch_lv355 smallint DEFAULT 0 NOT NULL,
    arch_lv370 smallint DEFAULT 0 NOT NULL,
    mortal_level smallint DEFAULT 0 NOT NULL,
    celestial_arch_level smallint DEFAULT 0 NOT NULL,
    arch_crystal_stage smallint DEFAULT 0 NOT NULL,
    kefra_ticket integer DEFAULT 0 NOT NULL,
    nightmare_entries integer DEFAULT 0 NOT NULL,
    last_nightmare_use bigint DEFAULT 0 NOT NULL,
    base_str smallint,
    base_int smallint,
    base_dex smallint,
    base_con smallint,
    base_max_hp integer,
    base_max_mp integer,
    newbie_quest smallint DEFAULT 0 NOT NULL,
    molar_gargula smallint DEFAULT 0 NOT NULL,
    pilula_orc smallint DEFAULT 0 NOT NULL,
    celestial_reset smallint DEFAULT 0 NOT NULL,
    celestial_add120 smallint DEFAULT 0 NOT NULL,
    celestial_add150 smallint DEFAULT 0 NOT NULL,
    celestial_add180 smallint DEFAULT 0 NOT NULL,
    celestial_add200 smallint DEFAULT 0 NOT NULL,
    save_celestial bytea DEFAULT '\x'::bytea NOT NULL,
    celestial_level smallint DEFAULT 0 NOT NULL,
    sub_celestial_level smallint DEFAULT 0 NOT NULL,
    CONSTRAINT character_arch_crystal_stage_check CHECK (((arch_crystal_stage >= 0) AND (arch_crystal_stage <= 4))),
    CONSTRAINT character_slot_check CHECK (((slot >= 0) AND (slot <= 3)))
);


--
-- Name: TABLE "character"; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public."character" IS 'Personagem: espelho de STRUCT_MOB + progressão (níveis, celestial, PK). `character` é palavra reservada: sempre entre aspas no SQL.';


--
-- Name: COLUMN "character".guild_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public."character".guild_id IS '0 = sem guilda; sem FK porque o registro mínimo de guilda pode não existir (issue 131).';


--
-- Name: COLUMN "character".soul; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public."character".soul IS 'MobExtra.Soul: alma permanente; alimenta o affect 29.';


--
-- Name: COLUMN "character".fame; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public."character".fame IS 'Fama do personagem (MobExtra.Fame).';


--
-- Name: COLUMN "character".pk_point; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public."character".pk_point IS 'Contador de caos (PKPoint do fio): 75 = neutro e o valor exibido é pk_point-75; faixa 1..150.';


--
-- Name: COLUMN "character".celestial_arch_level; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public."character".celestial_arch_level IS 'Banda de ArchLevel celestial (1..5) do QuestInfo.';


--
-- Name: character_admin_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.character_admin_audit (
    id bigint NOT NULL,
    moderator_id bigint NOT NULL,
    character_id bigint NOT NULL,
    action text NOT NULL,
    before jsonb,
    after jsonb,
    reason text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE character_admin_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.character_admin_audit IS 'Moderator character actions (issue 43): every grant/edit with before/after and the typed reason.';


--
-- Name: character_admin_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.character_admin_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: character_admin_audit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.character_admin_audit_id_seq OWNED BY public.character_admin_audit.id;


--
-- Name: character_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public."character" ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.character_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: character_pvp_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.character_pvp_stats (
    character_id bigint NOT NULL,
    wins integer DEFAULT 0 NOT NULL,
    losses integer DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE character_pvp_stats; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.character_pvp_stats IS 'Placar de PvP (vitórias/derrotas) por personagem.';


--
-- Name: content_rate_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.content_rate_audit (
    id bigint NOT NULL,
    moderator_id bigint NOT NULL,
    file text NOT NULL,
    family text NOT NULL,
    name text NOT NULL,
    action text NOT NULL,
    before jsonb,
    after jsonb,
    reason text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE content_rate_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.content_rate_audit IS 'Auditoria das edições de taxa de conteúdo (issue 36): before/after por célula.';


--
-- Name: content_rate_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.content_rate_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: content_rate_audit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.content_rate_audit_id_seq OWNED BY public.content_rate_audit.id;


--
-- Name: content_rate_override; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.content_rate_override (
    file text NOT NULL,
    family text NOT NULL,
    name text NOT NULL,
    value integer NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT content_rate_override_value_check CHECK ((value >= 0))
);


--
-- Name: TABLE content_rate_override; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.content_rate_override IS 'Overlay das taxas de conteúdo (issue 36): uma linha por célula editada; sem linha vale o arquivo em Release/Common/Settings.';


--
-- Name: daily_reward_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.daily_reward_audit (
    id bigint NOT NULL,
    reward_item_id bigint,
    account_id bigint NOT NULL,
    action text NOT NULL,
    before jsonb,
    after jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE daily_reward_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.daily_reward_audit IS 'Auditoria das edições de recompensa diária: before/after em jsonb.';


--
-- Name: daily_reward_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.daily_reward_audit ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.daily_reward_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: daily_reward_claim; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.daily_reward_claim (
    id bigint NOT NULL,
    account_id bigint NOT NULL,
    reward_item_id bigint,
    claim_date date NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE daily_reward_claim; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.daily_reward_claim IS 'Reivindicação de recompensa diária por conta e dia: o UNIQUE é o que impede crédito duplicado.';


--
-- Name: daily_reward_claim_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.daily_reward_claim ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.daily_reward_claim_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: daily_reward_item; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.daily_reward_item (
    id bigint NOT NULL,
    item_index integer NOT NULL,
    eff1 smallint DEFAULT 0 NOT NULL,
    effv1 smallint DEFAULT 0 NOT NULL,
    eff2 smallint DEFAULT 0 NOT NULL,
    effv2 smallint DEFAULT 0 NOT NULL,
    eff3 smallint DEFAULT 0 NOT NULL,
    effv3 smallint DEFAULT 0 NOT NULL,
    title text DEFAULT ''::text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    expires_days integer DEFAULT 0 NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT daily_reward_item_expires_days_check CHECK ((expires_days >= 0))
);


--
-- Name: TABLE daily_reward_item; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.daily_reward_item IS 'Item da recompensa diária (login diário), com efeitos e validade.';


--
-- Name: daily_reward_item_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.daily_reward_item ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.daily_reward_item_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: delivery_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.delivery_queue (
    id bigint NOT NULL,
    account_id bigint NOT NULL,
    character_id bigint,
    kind text NOT NULL,
    payload jsonb NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    source text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE delivery_queue; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.delivery_queue IS 'Fila de entregas web→jogo (issue 30): o tmServer drena e marca status; a web nunca escreve item direto.';


--
-- Name: COLUMN delivery_queue.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.delivery_queue.status IS 'Estado do drenado pelo tmServer: pending até ser aplicado.';


--
-- Name: delivery_queue_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.delivery_queue ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.delivery_queue_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: donate_payer_profile; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.donate_payer_profile (
    account_id bigint NOT NULL,
    name text NOT NULL,
    cpf text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE donate_payer_profile; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.donate_payer_profile IS 'Dados do pagador (nome/CPF) exigidos pelo gateway de doação; separado de account para não vazar em listagens.';


--
-- Name: donate_shop_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.donate_shop_audit (
    id bigint NOT NULL,
    shop_item_id bigint,
    account_id bigint NOT NULL,
    action text NOT NULL,
    before jsonb,
    after jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE donate_shop_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.donate_shop_audit IS 'Auditoria das edições da loja de doação: before/after em jsonb.';


--
-- Name: donate_shop_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.donate_shop_audit ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.donate_shop_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: donate_shop_item; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.donate_shop_item (
    id bigint NOT NULL,
    item_index integer NOT NULL,
    eff1 smallint DEFAULT 0 NOT NULL,
    effv1 smallint DEFAULT 0 NOT NULL,
    eff2 smallint DEFAULT 0 NOT NULL,
    effv2 smallint DEFAULT 0 NOT NULL,
    eff3 smallint DEFAULT 0 NOT NULL,
    effv3 smallint DEFAULT 0 NOT NULL,
    price integer NOT NULL,
    title text DEFAULT ''::text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    expires_days integer DEFAULT 0 NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT donate_shop_item_expires_days_check CHECK ((expires_days >= 0)),
    CONSTRAINT donate_shop_item_price_check CHECK ((price > 0))
);


--
-- Name: TABLE donate_shop_item; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.donate_shop_item IS 'Item da loja de doação (comprado com donate_balance), com efeitos e validade.';


--
-- Name: donate_shop_item_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.donate_shop_item ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.donate_shop_item_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: donate_topup_order; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.donate_topup_order (
    id bigint NOT NULL,
    external_reference text NOT NULL,
    account_id bigint NOT NULL,
    credits integer NOT NULL,
    amount_cents bigint NOT NULL,
    payment_method smallint NOT NULL,
    provider text,
    status smallint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    confirmed_at timestamp with time zone,
    CONSTRAINT donate_topup_order_amount_cents_check CHECK ((amount_cents > 0)),
    CONSTRAINT donate_topup_order_credits_check CHECK ((credits > 0))
);


--
-- Name: TABLE donate_topup_order; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.donate_topup_order IS 'Pedido de recarga de créditos no gateway: referência externa única, valor em centavos e status.';


--
-- Name: donate_topup_order_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.donate_topup_order ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.donate_topup_order_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: drop_rate_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.drop_rate_audit (
    id bigint NOT NULL,
    moderator_id bigint NOT NULL,
    slot smallint NOT NULL,
    action text NOT NULL,
    before jsonb,
    after jsonb,
    reason text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE drop_rate_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.drop_rate_audit IS 'Auditoria das edições de DropRate (issue 34): before/after por slot.';


--
-- Name: drop_rate_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.drop_rate_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: drop_rate_audit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.drop_rate_audit_id_seq OWNED BY public.drop_rate_audit.id;


--
-- Name: drop_rate_override; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.drop_rate_override (
    slot smallint NOT NULL,
    value integer NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT drop_rate_override_slot_check CHECK (((slot >= 0) AND (slot <= 63))),
    CONSTRAINT drop_rate_override_value_check CHECK (((value >= 1) AND (value <= 1000000)))
);


--
-- Name: TABLE drop_rate_override; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.drop_rate_override IS 'Overlay de DropRate[64] (issue 34): divisor por slot de Carry; sem linha vale o código em loot.go.';


--
-- Name: guild; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.guild (
    id integer NOT NULL,
    name text NOT NULL,
    clan smallint DEFAULT 0 NOT NULL,
    fame integer DEFAULT 0 NOT NULL,
    citizen smallint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT guild_id_check CHECK (((id > 0) AND (id < 65536)))
);


--
-- Name: TABLE guild; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.guild IS 'Guilda: nome, clã, fama e cidadania do líder.';


--
-- Name: guild_admin_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.guild_admin_audit (
    id bigint NOT NULL,
    moderator_id bigint NOT NULL,
    guild_id integer NOT NULL,
    action text NOT NULL,
    target jsonb,
    before jsonb,
    after jsonb,
    reason text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE guild_admin_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.guild_admin_audit IS 'Auditoria das edições de guilda/relação pela web (issue 45): before/after por ação.';


--
-- Name: guild_admin_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.guild_admin_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: guild_admin_audit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.guild_admin_audit_id_seq OWNED BY public.guild_admin_audit.id;


--
-- Name: guild_member; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.guild_member (
    guild_id integer NOT NULL,
    character_id bigint NOT NULL,
    account_id bigint NOT NULL,
    slot smallint NOT NULL,
    name text NOT NULL,
    guild_level smallint NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT guild_member_guild_level_check CHECK (((guild_level >= 0) AND (guild_level <= 9))),
    CONSTRAINT guild_member_slot_check CHECK (((slot >= 0) AND (slot <= 3)))
);


--
-- Name: TABLE guild_member; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.guild_member IS 'Membro de guilda por slot (até 12): liga personagem/conta ao papel dentro da guilda.';


--
-- Name: guild_relation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.guild_relation (
    guild_id integer NOT NULL,
    target_guild_id integer NOT NULL,
    kind smallint NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT guild_relation_check CHECK ((guild_id <> target_guild_id)),
    CONSTRAINT guild_relation_kind_check CHECK ((kind = ANY (ARRAY[1, 2])))
);


--
-- Name: TABLE guild_relation; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.guild_relation IS 'Relação entre guildas: kind 0 = aliada, 1 = inimiga.';


--
-- Name: guild_tower_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.guild_tower_state (
    id smallint NOT NULL,
    owner_guild integer DEFAULT 0 NOT NULL,
    updated_at_unix bigint DEFAULT 0 NOT NULL,
    CONSTRAINT guild_tower_state_id_check CHECK ((id = 1))
);


--
-- Name: TABLE guild_tower_state; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.guild_tower_state IS 'Dono atual da torre de guilda: guild e timestamp unix da última troca.';


--
-- Name: guild_zone; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.guild_zone (
    zone smallint NOT NULL,
    charge_guild integer DEFAULT 0 NOT NULL,
    challenge_guild integer DEFAULT 0 NOT NULL,
    clan smallint DEFAULT 0 NOT NULL,
    victory smallint DEFAULT 0 NOT NULL,
    city_tax smallint DEFAULT 0 NOT NULL,
    challenge_money bigint DEFAULT 0 NOT NULL,
    tax_vault bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT guild_zone_city_tax_check CHECK (((city_tax >= 0) AND (city_tax <= 30))),
    CONSTRAINT guild_zone_zone_check CHECK (((zone >= 0) AND (zone <= 4)))
);


--
-- Name: TABLE guild_zone; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.guild_zone IS 'Estado das zonas de disputa de guilda (GuildZone[5]): desafiante, taxa, vitória e cofre.';


--
-- Name: item; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.item (
    id bigint NOT NULL,
    owner_kind public.item_owner_kind NOT NULL,
    account_id bigint,
    character_id bigint,
    slot smallint NOT NULL,
    item_index integer NOT NULL,
    eff1 smallint DEFAULT 0 NOT NULL,
    effv1 smallint DEFAULT 0 NOT NULL,
    eff2 smallint DEFAULT 0 NOT NULL,
    effv2 smallint DEFAULT 0 NOT NULL,
    eff3 smallint DEFAULT 0 NOT NULL,
    effv3 smallint DEFAULT 0 NOT NULL,
    expires_at timestamp with time zone,
    CONSTRAINT item_check CHECK (((owner_kind = 'account_cargo'::public.item_owner_kind) = ((account_id IS NOT NULL) AND (character_id IS NULL)))),
    CONSTRAINT item_check1 CHECK (((owner_kind = ANY (ARRAY['char_equip'::public.item_owner_kind, 'char_carry'::public.item_owner_kind])) = (character_id IS NOT NULL)))
);


--
-- Name: TABLE item; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.item IS 'Item possuído: Equip[16]/Carry[64] da personagem e Cargo[128] da conta (STRUCT_ITEM, 8 B). eff/effv são smallint.';


--
-- Name: COLUMN item.item_index; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.item.item_index IS 'Índice em item_catalog (STRUCT_ITEMLIST, int); é o mesmo número do CSV.';


--
-- Name: item_catalog; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.item_catalog (
    item_index integer NOT NULL,
    name text NOT NULL,
    price integer DEFAULT 0 NOT NULL,
    grade smallint DEFAULT 0 NOT NULL,
    unique_id smallint DEFAULT 0 NOT NULL,
    extra integer DEFAULT 0 NOT NULL,
    pos smallint DEFAULT 0 NOT NULL,
    req_level smallint DEFAULT 0 NOT NULL,
    req_str smallint DEFAULT 0 NOT NULL,
    req_int smallint DEFAULT 0 NOT NULL,
    req_dex smallint DEFAULT 0 NOT NULL,
    req_con smallint DEFAULT 0 NOT NULL,
    volatile smallint,
    attack_range smallint,
    effects jsonb DEFAULT '[]'::jsonb NOT NULL,
    fields jsonb NOT NULL,
    sha256 text DEFAULT ''::text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT item_catalog_effects_array CHECK ((jsonb_typeof(effects) = 'array'::text)),
    CONSTRAINT item_catalog_item_index_check CHECK ((item_index > 0)),
    CONSTRAINT item_catalog_price_check CHECK ((price >= 0))
);


--
-- Name: TABLE item_catalog; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.item_catalog IS 'Catálogo de itens (ItemList.csv) no banco — issue 58. Substitui a leitura do CSV no boot; o arquivo vira insumo do importador.';


--
-- Name: COLUMN item_catalog.price; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.item_catalog.price IS 'Preço base vindo do ItemList.csv.';


--
-- Name: COLUMN item_catalog.effects; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.item_catalog.effects IS 'Efeitos de score (EF_* mapeados em content.efName) já resolvidos para id numérico, como em content.ItemList.BaseEffects.';


--
-- Name: COLUMN item_catalog.fields; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.item_catalog.fields IS 'Colunas cruas da linha de ItemList.csv, na ordem original: permite reimportar sem perda de coluna não modelada.';


--
-- Name: CONSTRAINT item_catalog_effects_array ON item_catalog; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON CONSTRAINT item_catalog_effects_array ON public.item_catalog IS 'effects é lista (nunca jsonb null): o seed relê a coluna por json_populate_record, onde null vira SQL NULL.';


--
-- Name: item_effect_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.item_effect_audit (
    id bigint NOT NULL,
    moderator_id bigint NOT NULL,
    item_index integer NOT NULL,
    effect smallint NOT NULL,
    action text NOT NULL,
    before jsonb,
    after jsonb,
    reason text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE item_effect_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.item_effect_audit IS 'Auditoria das edições de efeito base de item (issue 44): before/after por efeito.';


--
-- Name: item_effect_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.item_effect_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: item_effect_audit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.item_effect_audit_id_seq OWNED BY public.item_effect_audit.id;


--
-- Name: item_effect_override; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.item_effect_override (
    item_index integer NOT NULL,
    effect smallint NOT NULL,
    value integer NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT item_effect_override_effect_check CHECK (((effect >= 0) AND (effect <= 127)))
);


--
-- Name: TABLE item_effect_override; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.item_effect_override IS 'Overlay dos efeitos base de item (issue 44): uma linha por efeito editado; sem linha vale o ItemList.csv.';


--
-- Name: item_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.item ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.item_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: item_price; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.item_price (
    item_index integer NOT NULL,
    price bigint NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE item_price; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.item_price IS 'Preço de venda por item: overlay do preço base de item_catalog (usado pelo mercador).';


--
-- Name: COLUMN item_price.price; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.item_price.price IS 'Preço efetivo em moeda do jogo (bigint); o preço base do catálogo é integer porque vem do CSV.';


--
-- Name: kefra_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.kefra_state (
    id smallint NOT NULL,
    defeated boolean NOT NULL,
    next_spawn_unix bigint NOT NULL,
    last_spawn_unix bigint NOT NULL,
    revision bigint NOT NULL,
    CONSTRAINT kefra_state_check CHECK (((last_spawn_unix >= 0) AND (last_spawn_unix < next_spawn_unix))),
    CONSTRAINT kefra_state_id_check CHECK ((id = 1)),
    CONSTRAINT kefra_state_next_spawn_unix_check CHECK ((next_spawn_unix > 0)),
    CONSTRAINT kefra_state_revision_check CHECK ((revision > 0))
);


--
-- Name: TABLE kefra_state; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.kefra_state IS 'Estado do Kefra: derrotado, próximo spawn e revisão. Linha única.';


--
-- Name: mob_carry_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mob_carry_audit (
    id bigint NOT NULL,
    moderator_id bigint NOT NULL,
    template_name text NOT NULL,
    action text NOT NULL,
    before jsonb,
    after jsonb,
    reason text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE mob_carry_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mob_carry_audit IS 'Auditoria das edições de Carry[] (issue 34): array antes/depois por template.';


--
-- Name: mob_carry_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.mob_carry_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: mob_carry_audit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.mob_carry_audit_id_seq OWNED BY public.mob_carry_audit.id;


--
-- Name: mob_carry_override; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mob_carry_override (
    template_name text NOT NULL,
    slot smallint NOT NULL,
    item_index integer NOT NULL,
    eff1 smallint DEFAULT 0 NOT NULL,
    effv1 smallint DEFAULT 0 NOT NULL,
    eff2 smallint DEFAULT 0 NOT NULL,
    effv2 smallint DEFAULT 0 NOT NULL,
    eff3 smallint DEFAULT 0 NOT NULL,
    effv3 smallint DEFAULT 0 NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT mob_carry_override_eff1_check CHECK (((eff1 >= 0) AND (eff1 <= 255))),
    CONSTRAINT mob_carry_override_eff2_check CHECK (((eff2 >= 0) AND (eff2 <= 255))),
    CONSTRAINT mob_carry_override_eff3_check CHECK (((eff3 >= 0) AND (eff3 <= 255))),
    CONSTRAINT mob_carry_override_effv1_check CHECK (((effv1 >= 0) AND (effv1 <= 255))),
    CONSTRAINT mob_carry_override_effv2_check CHECK (((effv2 >= 0) AND (effv2 <= 255))),
    CONSTRAINT mob_carry_override_effv3_check CHECK (((effv3 >= 0) AND (effv3 <= 255))),
    CONSTRAINT mob_carry_override_item_index_check CHECK (((item_index >= 1) AND (item_index <= 32767))),
    CONSTRAINT mob_carry_override_slot_check CHECK (((slot >= 0) AND (slot <= 63)))
);


--
-- Name: TABLE mob_carry_override; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mob_carry_override IS 'Overlay do Carry[64] de um template (issue 34): slot define a raridade do drop; salvar substitui a tabela do template.';


--
-- Name: mob_template_equip; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mob_template_equip (
    template_name text NOT NULL,
    slot smallint NOT NULL,
    item_index integer NOT NULL,
    eff1 smallint DEFAULT 0 NOT NULL,
    effv1 smallint DEFAULT 0 NOT NULL,
    eff2 smallint DEFAULT 0 NOT NULL,
    effv2 smallint DEFAULT 0 NOT NULL,
    eff3 smallint DEFAULT 0 NOT NULL,
    effv3 smallint DEFAULT 0 NOT NULL,
    CONSTRAINT mob_template_equip_slot_check CHECK (((slot >= 0) AND (slot <= 15)))
);


--
-- Name: TABLE mob_template_equip; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mob_template_equip IS 'Equipamento inicial por template de monstro (BaseMob): slot → item e efeitos.';


--
-- Name: mob_template_stat; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mob_template_stat (
    template_name text NOT NULL,
    display_name text DEFAULT ''::text NOT NULL,
    clan smallint DEFAULT 0 NOT NULL,
    merchant smallint DEFAULT 0 NOT NULL,
    class smallint DEFAULT 0 NOT NULL,
    coin integer DEFAULT 0 NOT NULL,
    exp bigint DEFAULT 0 NOT NULL,
    spx integer DEFAULT 0 NOT NULL,
    spy integer DEFAULT 0 NOT NULL,
    level integer DEFAULT 0 NOT NULL,
    ac integer DEFAULT 0 NOT NULL,
    damage integer DEFAULT 0 NOT NULL,
    chaos_rate smallint DEFAULT 0 NOT NULL,
    attack_run smallint DEFAULT 0 NOT NULL,
    direction smallint DEFAULT 0 NOT NULL,
    str smallint DEFAULT 0 NOT NULL,
    intel smallint DEFAULT 0 NOT NULL,
    dex smallint DEFAULT 0 NOT NULL,
    con smallint DEFAULT 0 NOT NULL,
    special1 smallint DEFAULT 0 NOT NULL,
    special2 smallint DEFAULT 0 NOT NULL,
    special3 smallint DEFAULT 0 NOT NULL,
    special4 smallint DEFAULT 0 NOT NULL,
    max_hp integer DEFAULT 0 NOT NULL,
    hp integer DEFAULT 0 NOT NULL,
    max_mp integer DEFAULT 0 NOT NULL,
    mp integer DEFAULT 0 NOT NULL,
    learned_skill integer DEFAULT 0 NOT NULL,
    score_bonus integer DEFAULT 0 NOT NULL,
    skill_bar1 smallint DEFAULT 0 NOT NULL,
    skill_bar2 smallint DEFAULT 0 NOT NULL,
    skill_bar3 smallint DEFAULT 0 NOT NULL,
    skill_bar4 smallint DEFAULT 0 NOT NULL,
    regen_hp integer DEFAULT 0 NOT NULL,
    regen_mp integer DEFAULT 0 NOT NULL,
    resist1 smallint DEFAULT 0 NOT NULL,
    resist2 smallint DEFAULT 0 NOT NULL,
    resist3 smallint DEFAULT 0 NOT NULL,
    resist4 smallint DEFAULT 0 NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE mob_template_stat; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mob_template_stat IS 'Estatísticas editáveis de um template de monstro (STRUCT_MOB destrinchado); o blob canônico vive em npc_template.';


--
-- Name: COLUMN mob_template_stat.updated_by; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.mob_template_stat.updated_by IS 'Último editor (account.id, ON DELETE SET NULL); NULL = veio do arquivo/import.';


--
-- Name: npc_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.npc_audit (
    id bigint NOT NULL,
    npc_id bigint,
    account_id bigint NOT NULL,
    action text NOT NULL,
    before jsonb,
    after jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE npc_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.npc_audit IS 'Auditoria das edições de NPC feitas pelo portal: before/after em jsonb.';


--
-- Name: npc_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.npc_audit ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.npc_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: npc_config_meta; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.npc_config_meta (
    id boolean DEFAULT true NOT NULL,
    version bigint DEFAULT 0 NOT NULL,
    CONSTRAINT npc_config_meta_id_check CHECK (id)
);


--
-- Name: TABLE npc_config_meta; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.npc_config_meta IS 'Metadados do catálogo de NPCs: id booleano garante linha única; version invalida cache a cada edição.';


--
-- Name: npc_definition; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.npc_definition (
    id bigint NOT NULL,
    slug text NOT NULL,
    template_name text NOT NULL,
    display_name text DEFAULT ''::text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    map_id integer DEFAULT 0 NOT NULL,
    pos_x smallint DEFAULT 0 NOT NULL,
    pos_y smallint DEFAULT 0 NOT NULL,
    route_type smallint DEFAULT 0 NOT NULL,
    merchant smallint DEFAULT 0 NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    origin text DEFAULT 'custom'::text NOT NULL,
    generator_index integer,
    follower_template text DEFAULT ''::text NOT NULL,
    minute_generate integer DEFAULT 0 NOT NULL,
    min_group integer DEFAULT 0 NOT NULL,
    max_group integer DEFAULT 0 NOT NULL,
    max_num_mob integer DEFAULT 1 NOT NULL,
    formation smallint DEFAULT 0 NOT NULL,
    generator_data jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT npc_definition_origin_check CHECK ((origin = ANY (ARRAY['content'::text, 'custom'::text])))
);


--
-- Name: TABLE npc_definition; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.npc_definition IS 'Definição de NPC/gerador no mundo (substituiu NPCGener.txt, issue 57): posição, template, grupo e origem (content/edit).';


--
-- Name: COLUMN npc_definition.pos_x; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.npc_definition.pos_x IS 'Coordenada no grid do mapa (0..4095).';


--
-- Name: COLUMN npc_definition.pos_y; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.npc_definition.pos_y IS 'Coordenada no grid do mapa (0..4095).';


--
-- Name: COLUMN npc_definition.origin; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.npc_definition.origin IS 'content = importado do conteúdo; edit = criado/editado no portal (o import não sobrescreve).';


--
-- Name: npc_definition_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.npc_definition ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.npc_definition_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: npc_shop_item; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.npc_shop_item (
    npc_id bigint NOT NULL,
    slot smallint NOT NULL,
    item_index integer NOT NULL,
    eff1 smallint DEFAULT 0 NOT NULL,
    effv1 smallint DEFAULT 0 NOT NULL,
    eff2 smallint DEFAULT 0 NOT NULL,
    effv2 smallint DEFAULT 0 NOT NULL,
    eff3 smallint DEFAULT 0 NOT NULL,
    effv3 smallint DEFAULT 0 NOT NULL,
    quantity smallint DEFAULT 1 NOT NULL,
    CONSTRAINT npc_shop_item_quantity_check CHECK (((quantity >= 1) AND (quantity <= 255))),
    CONSTRAINT npc_shop_item_slot_check CHECK (((slot >= 0) AND (slot <= 26)))
);


--
-- Name: TABLE npc_shop_item; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.npc_shop_item IS 'Item vendido por um NPC (loja): slot, item, efeitos e quantidade.';


--
-- Name: npc_template; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.npc_template (
    template_name text NOT NULL,
    file_name text NOT NULL,
    source_size smallint NOT NULL,
    sha256 text NOT NULL,
    blob bytea NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT npc_template_blob_check CHECK ((length(blob) = 816))
);


--
-- Name: TABLE npc_template; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.npc_template IS 'Templates STRUCT_MOB do diretório npc/ no banco — issue 58. O blob é sempre canônico (816 B); layouts legados são normalizados no import.';


--
-- Name: COLUMN npc_template.source_size; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.npc_template.source_size IS 'Tamanho do arquivo original em disco (756/816/920), preservado para auditoria da normalização.';


--
-- Name: sapphire_balance; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sapphire_balance (
    singleton boolean DEFAULT true NOT NULL,
    version bigint DEFAULT 1 NOT NULL,
    hekalotia_cost integer DEFAULT 8 NOT NULL,
    akelonia_cost integer DEFAULT 8 NOT NULL,
    CONSTRAINT sapphire_balance_akelonia_cost_check CHECK (((akelonia_cost >= 4) AND (akelonia_cost <= 32))),
    CONSTRAINT sapphire_balance_hekalotia_cost_check CHECK (((hekalotia_cost >= 4) AND (hekalotia_cost <= 32))),
    CONSTRAINT sapphire_balance_singleton_check CHECK (singleton)
);


--
-- Name: TABLE sapphire_balance; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.sapphire_balance IS 'Custo em safiras das soluções de Hekalotia/Akelonia; linha única (singleton).';


--
-- Name: server_instance_flags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.server_instance_flags (
    id bigint NOT NULL,
    server text NOT NULL,
    flags jsonb NOT NULL,
    captured_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE server_instance_flags; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.server_instance_flags IS 'Flags de processo com que cada serviço subiu (issue 36): snapshot read-only, uma linha por boot.';


--
-- Name: server_instance_flags_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.server_instance_flags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: server_instance_flags_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.server_instance_flags_id_seq OWNED BY public.server_instance_flags.id;


--
-- Name: skill_catalog; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.skill_catalog (
    skill_index integer NOT NULL,
    name text DEFAULT ''::text NOT NULL,
    skill_point smallint DEFAULT 0 NOT NULL,
    target_type smallint DEFAULT 0 NOT NULL,
    mana_spent smallint DEFAULT 0 NOT NULL,
    delay_ticks smallint DEFAULT 0 NOT NULL,
    range_tiles smallint DEFAULT 0 NOT NULL,
    instance_type smallint DEFAULT 0 NOT NULL,
    instance_value smallint DEFAULT 0 NOT NULL,
    tick_type smallint DEFAULT 0 NOT NULL,
    tick_value smallint DEFAULT 0 NOT NULL,
    affect_type smallint DEFAULT 0 NOT NULL,
    affect_value smallint DEFAULT 0 NOT NULL,
    affect_time smallint DEFAULT 0 NOT NULL,
    instance_attribute smallint DEFAULT 0 NOT NULL,
    tick_attribute smallint DEFAULT 0 NOT NULL,
    aggressive smallint DEFAULT 0 NOT NULL,
    max_target smallint DEFAULT 0 NOT NULL,
    party smallint DEFAULT 0 NOT NULL,
    affect_resist smallint DEFAULT 0 NOT NULL,
    passive smallint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT skill_catalog_skill_index_check CHECK (((skill_index >= 0) AND (skill_index < 248)))
);


--
-- Name: TABLE skill_catalog; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.skill_catalog IS 'Catálogo de skills (SkillData.csv) no banco — issue 58. Substitui a leitura do CSV pelo tmServer e pelo portal (nomes de skill).';


--
-- Name: teleport_route; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.teleport_route (
    id bigint NOT NULL,
    src_x integer NOT NULL,
    src_y integer NOT NULL,
    dst_x integer NOT NULL,
    dst_y integer NOT NULL,
    cost integer DEFAULT 0 NOT NULL,
    requirement text DEFAULT 'none'::text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT teleport_route_cost_check CHECK ((cost >= 0)),
    CONSTRAINT teleport_route_dst_x_check CHECK (((dst_x >= 0) AND (dst_x < 4096))),
    CONSTRAINT teleport_route_dst_y_check CHECK (((dst_y >= 0) AND (dst_y < 4096))),
    CONSTRAINT teleport_route_requirement_check CHECK ((requirement = ANY (ARRAY['none'::text, 'fairy'::text, 'kefra'::text]))),
    CONSTRAINT teleport_route_src_x_check CHECK (((src_x >= 0) AND (src_x < 4096) AND ((src_x % 4) = 0))),
    CONSTRAINT teleport_route_src_y_check CHECK (((src_y >= 0) AND (src_y < 4096) AND ((src_y % 4) = 0)))
);


--
-- Name: TABLE teleport_route; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.teleport_route IS 'Moderator-editable teleport overlay (issue 42); rows override world.teleportTable by origin tile.';


--
-- Name: teleport_route_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.teleport_route_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: teleport_route_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.teleport_route_id_seq OWNED BY public.teleport_route.id;


--
-- Name: world_event_audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.world_event_audit (
    id bigint NOT NULL,
    account_id bigint NOT NULL,
    action text NOT NULL,
    before jsonb,
    after jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE world_event_audit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.world_event_audit IS 'Auditoria das edições do evento de mundo: before/after em jsonb.';


--
-- Name: world_event_audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.world_event_audit ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.world_event_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: world_event_config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.world_event_config (
    id boolean DEFAULT true NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    item_index integer DEFAULT 0 NOT NULL,
    rate integer DEFAULT 0 NOT NULL,
    start_index integer DEFAULT 0 NOT NULL,
    current_index integer DEFAULT 0 NOT NULL,
    end_index integer DEFAULT 0 NOT NULL,
    indexed boolean DEFAULT false NOT NULL,
    notice_enabled boolean DEFAULT true NOT NULL,
    double_exp_enabled boolean DEFAULT false NOT NULL,
    newbie_event_enabled boolean DEFAULT false NOT NULL,
    updated_by bigint,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT world_event_config_current_index_check CHECK ((current_index >= 0)),
    CONSTRAINT world_event_config_end_index_check CHECK ((end_index >= 0)),
    CONSTRAINT world_event_config_id_check CHECK (id),
    CONSTRAINT world_event_config_item_index_check CHECK ((item_index >= 0)),
    CONSTRAINT world_event_config_rate_check CHECK ((rate >= 0)),
    CONSTRAINT world_event_config_start_index_check CHECK ((start_index >= 0))
);


--
-- Name: TABLE world_event_config; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.world_event_config IS 'Configuração do evento de mundo (item sorteado, taxa, janela de índices e flags de aviso). Linha única.';


--
-- Name: world_event_meta; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.world_event_meta (
    id boolean DEFAULT true NOT NULL,
    version bigint DEFAULT 0 NOT NULL,
    CONSTRAINT world_event_meta_id_check CHECK (id)
);


--
-- Name: TABLE world_event_meta; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.world_event_meta IS 'Metadados do evento de mundo: id booleano + version para invalidação de cache.';


--
-- Name: world_item_seed; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.world_item_seed (
    id bigint NOT NULL,
    item_index integer NOT NULL,
    pos_x smallint NOT NULL,
    pos_y smallint NOT NULL,
    rotate smallint DEFAULT 0 NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE world_item_seed; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.world_item_seed IS 'Objetos estáticos do mundo (InitItem.csv) no banco — issue 58. Reimport idempotente pela chave natural (item_index, pos_x, pos_y).';


--
-- Name: world_item_seed_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.world_item_seed_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: world_item_seed_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.world_item_seed_id_seq OWNED BY public.world_item_seed.id;


--
-- Name: account_admin_audit id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_admin_audit ALTER COLUMN id SET DEFAULT nextval('public.account_admin_audit_id_seq'::regclass);


--
-- Name: character_admin_audit id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.character_admin_audit ALTER COLUMN id SET DEFAULT nextval('public.character_admin_audit_id_seq'::regclass);


--
-- Name: content_rate_audit id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.content_rate_audit ALTER COLUMN id SET DEFAULT nextval('public.content_rate_audit_id_seq'::regclass);


--
-- Name: drop_rate_audit id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.drop_rate_audit ALTER COLUMN id SET DEFAULT nextval('public.drop_rate_audit_id_seq'::regclass);


--
-- Name: guild_admin_audit id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_admin_audit ALTER COLUMN id SET DEFAULT nextval('public.guild_admin_audit_id_seq'::regclass);


--
-- Name: item_effect_audit id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_effect_audit ALTER COLUMN id SET DEFAULT nextval('public.item_effect_audit_id_seq'::regclass);


--
-- Name: mob_carry_audit id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mob_carry_audit ALTER COLUMN id SET DEFAULT nextval('public.mob_carry_audit_id_seq'::regclass);


--
-- Name: server_instance_flags id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.server_instance_flags ALTER COLUMN id SET DEFAULT nextval('public.server_instance_flags_id_seq'::regclass);


--
-- Name: teleport_route id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teleport_route ALTER COLUMN id SET DEFAULT nextval('public.teleport_route_id_seq'::regclass);


--
-- Name: world_item_seed id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_item_seed ALTER COLUMN id SET DEFAULT nextval('public.world_item_seed_id_seq'::regclass);


--
-- Name: account_admin_audit account_admin_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_admin_audit
    ADD CONSTRAINT account_admin_audit_pkey PRIMARY KEY (id);


--
-- Name: account account_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account
    ADD CONSTRAINT account_name_key UNIQUE (name);


--
-- Name: account account_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account
    ADD CONSTRAINT account_pkey PRIMARY KEY (id);


--
-- Name: account_play_code account_play_code_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_play_code
    ADD CONSTRAINT account_play_code_pkey PRIMARY KEY (code_hash);


--
-- Name: affect affect_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.affect
    ADD CONSTRAINT affect_pkey PRIMARY KEY (id);


--
-- Name: castle_quest_state castle_quest_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.castle_quest_state
    ADD CONSTRAINT castle_quest_state_pkey PRIMARY KEY (id);


--
-- Name: character character_account_id_slot_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."character"
    ADD CONSTRAINT character_account_id_slot_key UNIQUE (account_id, slot);


--
-- Name: character_admin_audit character_admin_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.character_admin_audit
    ADD CONSTRAINT character_admin_audit_pkey PRIMARY KEY (id);


--
-- Name: character character_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."character"
    ADD CONSTRAINT character_pkey PRIMARY KEY (id);


--
-- Name: character_pvp_stats character_pvp_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.character_pvp_stats
    ADD CONSTRAINT character_pvp_stats_pkey PRIMARY KEY (character_id);


--
-- Name: content_rate_audit content_rate_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.content_rate_audit
    ADD CONSTRAINT content_rate_audit_pkey PRIMARY KEY (id);


--
-- Name: content_rate_override content_rate_override_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.content_rate_override
    ADD CONSTRAINT content_rate_override_pkey PRIMARY KEY (file, family, name);


--
-- Name: daily_reward_audit daily_reward_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daily_reward_audit
    ADD CONSTRAINT daily_reward_audit_pkey PRIMARY KEY (id);


--
-- Name: daily_reward_claim daily_reward_claim_account_id_claim_date_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daily_reward_claim
    ADD CONSTRAINT daily_reward_claim_account_id_claim_date_key UNIQUE (account_id, claim_date);


--
-- Name: daily_reward_claim daily_reward_claim_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daily_reward_claim
    ADD CONSTRAINT daily_reward_claim_pkey PRIMARY KEY (id);


--
-- Name: daily_reward_item daily_reward_item_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daily_reward_item
    ADD CONSTRAINT daily_reward_item_pkey PRIMARY KEY (id);


--
-- Name: delivery_queue delivery_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.delivery_queue
    ADD CONSTRAINT delivery_queue_pkey PRIMARY KEY (id);


--
-- Name: donate_payer_profile donate_payer_profile_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.donate_payer_profile
    ADD CONSTRAINT donate_payer_profile_pkey PRIMARY KEY (account_id);


--
-- Name: donate_shop_audit donate_shop_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.donate_shop_audit
    ADD CONSTRAINT donate_shop_audit_pkey PRIMARY KEY (id);


--
-- Name: donate_shop_item donate_shop_item_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.donate_shop_item
    ADD CONSTRAINT donate_shop_item_pkey PRIMARY KEY (id);


--
-- Name: donate_topup_order donate_topup_order_external_reference_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.donate_topup_order
    ADD CONSTRAINT donate_topup_order_external_reference_key UNIQUE (external_reference);


--
-- Name: donate_topup_order donate_topup_order_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.donate_topup_order
    ADD CONSTRAINT donate_topup_order_pkey PRIMARY KEY (id);


--
-- Name: drop_rate_audit drop_rate_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.drop_rate_audit
    ADD CONSTRAINT drop_rate_audit_pkey PRIMARY KEY (id);


--
-- Name: drop_rate_override drop_rate_override_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.drop_rate_override
    ADD CONSTRAINT drop_rate_override_pkey PRIMARY KEY (slot);


--
-- Name: guild_admin_audit guild_admin_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_admin_audit
    ADD CONSTRAINT guild_admin_audit_pkey PRIMARY KEY (id);


--
-- Name: guild_member guild_member_character_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_member
    ADD CONSTRAINT guild_member_character_id_key UNIQUE (character_id);


--
-- Name: guild_member guild_member_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_member
    ADD CONSTRAINT guild_member_pkey PRIMARY KEY (guild_id, character_id);


--
-- Name: guild guild_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild
    ADD CONSTRAINT guild_name_key UNIQUE (name);


--
-- Name: guild guild_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild
    ADD CONSTRAINT guild_pkey PRIMARY KEY (id);


--
-- Name: guild_relation guild_relation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_relation
    ADD CONSTRAINT guild_relation_pkey PRIMARY KEY (guild_id, kind);


--
-- Name: guild_tower_state guild_tower_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_tower_state
    ADD CONSTRAINT guild_tower_state_pkey PRIMARY KEY (id);


--
-- Name: guild_zone guild_zone_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_zone
    ADD CONSTRAINT guild_zone_pkey PRIMARY KEY (zone);


--
-- Name: item_catalog item_catalog_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_catalog
    ADD CONSTRAINT item_catalog_pkey PRIMARY KEY (item_index);


--
-- Name: item_effect_audit item_effect_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_effect_audit
    ADD CONSTRAINT item_effect_audit_pkey PRIMARY KEY (id);


--
-- Name: item_effect_override item_effect_override_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_effect_override
    ADD CONSTRAINT item_effect_override_pkey PRIMARY KEY (item_index, effect);


--
-- Name: item item_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item
    ADD CONSTRAINT item_pkey PRIMARY KEY (id);


--
-- Name: item_price item_price_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_price
    ADD CONSTRAINT item_price_pkey PRIMARY KEY (item_index);


--
-- Name: kefra_state kefra_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kefra_state
    ADD CONSTRAINT kefra_state_pkey PRIMARY KEY (id);


--
-- Name: mob_carry_audit mob_carry_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mob_carry_audit
    ADD CONSTRAINT mob_carry_audit_pkey PRIMARY KEY (id);


--
-- Name: mob_carry_override mob_carry_override_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mob_carry_override
    ADD CONSTRAINT mob_carry_override_pkey PRIMARY KEY (template_name, slot);


--
-- Name: mob_template_equip mob_template_equip_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mob_template_equip
    ADD CONSTRAINT mob_template_equip_pkey PRIMARY KEY (template_name, slot);


--
-- Name: mob_template_stat mob_template_stat_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mob_template_stat
    ADD CONSTRAINT mob_template_stat_pkey PRIMARY KEY (template_name);


--
-- Name: npc_audit npc_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.npc_audit
    ADD CONSTRAINT npc_audit_pkey PRIMARY KEY (id);


--
-- Name: npc_config_meta npc_config_meta_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.npc_config_meta
    ADD CONSTRAINT npc_config_meta_pkey PRIMARY KEY (id);


--
-- Name: npc_definition npc_definition_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.npc_definition
    ADD CONSTRAINT npc_definition_pkey PRIMARY KEY (id);


--
-- Name: npc_definition npc_definition_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.npc_definition
    ADD CONSTRAINT npc_definition_slug_key UNIQUE (slug);


--
-- Name: npc_shop_item npc_shop_item_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.npc_shop_item
    ADD CONSTRAINT npc_shop_item_pkey PRIMARY KEY (npc_id, slot);


--
-- Name: npc_template npc_template_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.npc_template
    ADD CONSTRAINT npc_template_pkey PRIMARY KEY (template_name);


--
-- Name: sapphire_balance sapphire_balance_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sapphire_balance
    ADD CONSTRAINT sapphire_balance_pkey PRIMARY KEY (singleton);


--
-- Name: server_instance_flags server_instance_flags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.server_instance_flags
    ADD CONSTRAINT server_instance_flags_pkey PRIMARY KEY (id);


--
-- Name: skill_catalog skill_catalog_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.skill_catalog
    ADD CONSTRAINT skill_catalog_pkey PRIMARY KEY (skill_index);


--
-- Name: teleport_route teleport_route_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teleport_route
    ADD CONSTRAINT teleport_route_pkey PRIMARY KEY (id);


--
-- Name: teleport_route teleport_route_src_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teleport_route
    ADD CONSTRAINT teleport_route_src_key UNIQUE (src_x, src_y);


--
-- Name: world_event_audit world_event_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_event_audit
    ADD CONSTRAINT world_event_audit_pkey PRIMARY KEY (id);


--
-- Name: world_event_config world_event_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_event_config
    ADD CONSTRAINT world_event_config_pkey PRIMARY KEY (id);


--
-- Name: world_event_meta world_event_meta_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_event_meta
    ADD CONSTRAINT world_event_meta_pkey PRIMARY KEY (id);


--
-- Name: world_item_seed world_item_seed_item_index_pos_x_pos_y_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_item_seed
    ADD CONSTRAINT world_item_seed_item_index_pos_x_pos_y_key UNIQUE (item_index, pos_x, pos_y);


--
-- Name: world_item_seed world_item_seed_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_item_seed
    ADD CONSTRAINT world_item_seed_pkey PRIMARY KEY (id);


--
-- Name: account_admin_audit_actor_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_admin_audit_actor_idx ON public.account_admin_audit USING btree (actor_id, id DESC);


--
-- Name: account_admin_audit_target_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_admin_audit_target_idx ON public.account_admin_audit USING btree (target_id, id DESC);


--
-- Name: account_name_prefix_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_name_prefix_idx ON public.account USING btree (name text_pattern_ops);


--
-- Name: account_play_code_account_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_play_code_account_idx ON public.account_play_code USING btree (account_id);


--
-- Name: affect_character_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX affect_character_id_idx ON public.affect USING btree (character_id);


--
-- Name: character_account_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX character_account_id_idx ON public."character" USING btree (account_id);


--
-- Name: character_admin_audit_character_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX character_admin_audit_character_idx ON public.character_admin_audit USING btree (character_id, created_at DESC);


--
-- Name: character_admin_audit_moderator_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX character_admin_audit_moderator_idx ON public.character_admin_audit USING btree (moderator_id, created_at DESC);


--
-- Name: character_exp_ranking_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX character_exp_ranking_idx ON public."character" USING btree ((
CASE class_master
    WHEN 2 THEN 1
    WHEN 1 THEN 2
    ELSE (class_master)::integer
END) DESC, exp DESC, level DESC, name) WHERE (level < 1000);


--
-- Name: character_name_class_master_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX character_name_class_master_idx ON public."character" USING btree (name, class_master);


--
-- Name: content_rate_audit_key_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX content_rate_audit_key_idx ON public.content_rate_audit USING btree (file, family, name, created_at DESC);


--
-- Name: content_rate_audit_moderator_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX content_rate_audit_moderator_idx ON public.content_rate_audit USING btree (moderator_id, created_at DESC);


--
-- Name: daily_reward_audit_reward_item_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX daily_reward_audit_reward_item_id_idx ON public.daily_reward_audit USING btree (reward_item_id);


--
-- Name: daily_reward_claim_reward_item_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX daily_reward_claim_reward_item_idx ON public.daily_reward_claim USING btree (reward_item_id);


--
-- Name: daily_reward_item_updated_by_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX daily_reward_item_updated_by_idx ON public.daily_reward_item USING btree (updated_by);


--
-- Name: delivery_queue_account_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX delivery_queue_account_status_idx ON public.delivery_queue USING btree (account_id, status);


--
-- Name: donate_shop_audit_action_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX donate_shop_audit_action_created_idx ON public.donate_shop_audit USING btree (action, created_at DESC);


--
-- Name: donate_shop_audit_shop_item_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX donate_shop_audit_shop_item_id_idx ON public.donate_shop_audit USING btree (shop_item_id);


--
-- Name: donate_shop_audit_subject_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX donate_shop_audit_subject_idx ON public.donate_shop_audit USING btree (COALESCE((NULLIF((after ->> 'account_id'::text), ''::text))::bigint, account_id)) WHERE (action = ANY (ARRAY['purchase'::text, 'credit_balance'::text]));


--
-- Name: donate_shop_item_updated_by_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX donate_shop_item_updated_by_idx ON public.donate_shop_item USING btree (updated_by);


--
-- Name: donate_topup_order_account_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX donate_topup_order_account_idx ON public.donate_topup_order USING btree (account_id);


--
-- Name: donate_topup_order_confirmed_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX donate_topup_order_confirmed_idx ON public.donate_topup_order USING btree (confirmed_at DESC) WHERE (status = 2);


--
-- Name: donate_topup_order_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX donate_topup_order_created_idx ON public.donate_topup_order USING btree (created_at DESC);


--
-- Name: drop_rate_audit_moderator_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX drop_rate_audit_moderator_idx ON public.drop_rate_audit USING btree (moderator_id, created_at DESC);


--
-- Name: drop_rate_audit_slot_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX drop_rate_audit_slot_idx ON public.drop_rate_audit USING btree (slot, created_at DESC);


--
-- Name: guild_admin_audit_guild_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX guild_admin_audit_guild_idx ON public.guild_admin_audit USING btree (guild_id, created_at DESC);


--
-- Name: guild_admin_audit_moderator_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX guild_admin_audit_moderator_idx ON public.guild_admin_audit USING btree (moderator_id, created_at DESC);


--
-- Name: guild_member_account_slot_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX guild_member_account_slot_idx ON public.guild_member USING btree (account_id, slot);


--
-- Name: guild_member_level_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX guild_member_level_idx ON public.guild_member USING btree (guild_id, guild_level);


--
-- Name: guild_relation_target_guild_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX guild_relation_target_guild_id_idx ON public.guild_relation USING btree (target_guild_id);


--
-- Name: item_account_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_account_id_idx ON public.item USING btree (account_id);


--
-- Name: item_catalog_grade_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_catalog_grade_idx ON public.item_catalog USING btree (grade);


--
-- Name: item_catalog_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_catalog_name_idx ON public.item_catalog USING btree (lower(name));


--
-- Name: item_character_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_character_id_idx ON public.item USING btree (character_id);


--
-- Name: item_effect_audit_item_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_effect_audit_item_idx ON public.item_effect_audit USING btree (item_index, created_at DESC);


--
-- Name: item_effect_audit_moderator_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_effect_audit_moderator_idx ON public.item_effect_audit USING btree (moderator_id, created_at DESC);


--
-- Name: item_price_updated_by_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_price_updated_by_idx ON public.item_price USING btree (updated_by);


--
-- Name: mob_carry_audit_moderator_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX mob_carry_audit_moderator_idx ON public.mob_carry_audit USING btree (moderator_id, created_at DESC);


--
-- Name: mob_carry_audit_template_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX mob_carry_audit_template_idx ON public.mob_carry_audit USING btree (template_name, created_at DESC);


--
-- Name: mob_template_equip_template_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX mob_template_equip_template_idx ON public.mob_template_equip USING btree (template_name);


--
-- Name: mob_template_stat_updated_by_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX mob_template_stat_updated_by_idx ON public.mob_template_stat USING btree (updated_by);


--
-- Name: npc_audit_npc_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX npc_audit_npc_id_idx ON public.npc_audit USING btree (npc_id);


--
-- Name: npc_definition_generator_index_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX npc_definition_generator_index_idx ON public.npc_definition USING btree (generator_index) WHERE (generator_index IS NOT NULL);


--
-- Name: npc_definition_updated_by_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX npc_definition_updated_by_idx ON public.npc_definition USING btree (updated_by);


--
-- Name: npc_shop_item_npc_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX npc_shop_item_npc_id_idx ON public.npc_shop_item USING btree (npc_id);


--
-- Name: npc_template_file_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX npc_template_file_name_idx ON public.npc_template USING btree (file_name);


--
-- Name: server_instance_flags_server_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX server_instance_flags_server_idx ON public.server_instance_flags USING btree (server, captured_at DESC);


--
-- Name: skill_catalog_class_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX skill_catalog_class_idx ON public.skill_catalog USING btree (((skill_index / 24)));


--
-- Name: skill_catalog_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX skill_catalog_name_idx ON public.skill_catalog USING btree (lower(name));


--
-- Name: world_event_config_updated_by_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX world_event_config_updated_by_idx ON public.world_event_config USING btree (updated_by);


--
-- Name: world_item_seed_pos_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX world_item_seed_pos_idx ON public.world_item_seed USING btree (pos_x, pos_y);


--
-- Name: account_play_code account_play_code_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_play_code
    ADD CONSTRAINT account_play_code_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: affect affect_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.affect
    ADD CONSTRAINT affect_character_id_fkey FOREIGN KEY (character_id) REFERENCES public."character"(id) ON DELETE CASCADE;


--
-- Name: character character_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."character"
    ADD CONSTRAINT character_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: character_pvp_stats character_pvp_stats_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.character_pvp_stats
    ADD CONSTRAINT character_pvp_stats_character_id_fkey FOREIGN KEY (character_id) REFERENCES public."character"(id) ON DELETE CASCADE;


--
-- Name: daily_reward_claim daily_reward_claim_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daily_reward_claim
    ADD CONSTRAINT daily_reward_claim_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: daily_reward_claim daily_reward_claim_reward_item_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daily_reward_claim
    ADD CONSTRAINT daily_reward_claim_reward_item_id_fkey FOREIGN KEY (reward_item_id) REFERENCES public.daily_reward_item(id) ON DELETE SET NULL;


--
-- Name: daily_reward_item daily_reward_item_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daily_reward_item
    ADD CONSTRAINT daily_reward_item_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.account(id) ON DELETE SET NULL;


--
-- Name: delivery_queue delivery_queue_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.delivery_queue
    ADD CONSTRAINT delivery_queue_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: donate_payer_profile donate_payer_profile_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.donate_payer_profile
    ADD CONSTRAINT donate_payer_profile_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: donate_shop_item donate_shop_item_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.donate_shop_item
    ADD CONSTRAINT donate_shop_item_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.account(id) ON DELETE SET NULL;


--
-- Name: donate_topup_order donate_topup_order_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.donate_topup_order
    ADD CONSTRAINT donate_topup_order_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: guild_member guild_member_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_member
    ADD CONSTRAINT guild_member_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: guild_member guild_member_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_member
    ADD CONSTRAINT guild_member_character_id_fkey FOREIGN KEY (character_id) REFERENCES public."character"(id) ON DELETE CASCADE;


--
-- Name: guild_member guild_member_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_member
    ADD CONSTRAINT guild_member_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guild(id) ON DELETE CASCADE;


--
-- Name: guild_relation guild_relation_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_relation
    ADD CONSTRAINT guild_relation_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guild(id) ON DELETE CASCADE;


--
-- Name: guild_relation guild_relation_target_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.guild_relation
    ADD CONSTRAINT guild_relation_target_guild_id_fkey FOREIGN KEY (target_guild_id) REFERENCES public.guild(id) ON DELETE CASCADE;


--
-- Name: item item_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item
    ADD CONSTRAINT item_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: item item_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item
    ADD CONSTRAINT item_character_id_fkey FOREIGN KEY (character_id) REFERENCES public."character"(id) ON DELETE CASCADE;


--
-- Name: item_price item_price_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_price
    ADD CONSTRAINT item_price_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.account(id) ON DELETE SET NULL;


--
-- Name: mob_template_equip mob_template_equip_template_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mob_template_equip
    ADD CONSTRAINT mob_template_equip_template_name_fkey FOREIGN KEY (template_name) REFERENCES public.mob_template_stat(template_name) ON DELETE CASCADE;


--
-- Name: mob_template_stat mob_template_stat_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mob_template_stat
    ADD CONSTRAINT mob_template_stat_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.account(id) ON DELETE SET NULL;


--
-- Name: npc_definition npc_definition_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.npc_definition
    ADD CONSTRAINT npc_definition_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.account(id) ON DELETE SET NULL;


--
-- Name: npc_shop_item npc_shop_item_npc_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.npc_shop_item
    ADD CONSTRAINT npc_shop_item_npc_id_fkey FOREIGN KEY (npc_id) REFERENCES public.npc_definition(id) ON DELETE CASCADE;


--
-- Name: world_event_config world_event_config_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.world_event_config
    ADD CONSTRAINT world_event_config_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.account(id) ON DELETE SET NULL;


--
-- PostgreSQL database dump complete
--


