#!/usr/bin/env node
/**
 * gen-seed.mjs — converte o conteúdo legado (`origin/main`) em migrações SQL.
 *
 * ADR-0009: o PostgreSQL é a fonte canônica; o legado é **insumo** deste gerador.
 * Nada é lido em runtime — o produto é `crates/wyd-db/migrations/00NN_seed_*.sql`.
 *
 * Uso:
 *   node tools/content/gen-seed.mjs             # gera 0009, 0010 e 0012
 *   node tools/content/gen-seed.mjs --dry-run   # só relatório (sem escrever)
 *
 * Sanitização (modelo-banco.md §6.6): decodificação UTF-8 estrita (byte inválido fora de
 * comentário aborta), CRLF→LF, BOM/invisíveis/controle removidos, espaço não separável
 * virando espaço, NFC como forma canônica, espaços colapsados em rótulos. Todo desvio é
 * contado no relatório; nenhum nome é "consertado" por chute.
 *
 * Classificação (docs/wyd/itens.md, docs/wyd/skills.md): só regra de nome documentada;
 * o que não casa fica NULL com `classification_source`, nunca um palpite.
 */
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdirSync, writeFileSync, statSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(HERE, "../..");
const MIGRATIONS = resolve(ROOT, "crates/wyd-db/migrations");
const DRY = process.argv.includes("--dry-run");

/** Cada insumo: caminho no `origin/main` e a migração que o carrega. */
const SRC = {
  itemCatalog: { path: "internal/migrations/seed_item_catalog.up.sql", loadedBy: "0009" },
  skillCatalog: { path: "internal/migrations/seed_skill_catalog.up.sql", loadedBy: "0010" },
  npcTemplate: { path: "internal/migrations/seed_npc_template.up.sql", loadedBy: "0012" },
  npcDefinition: { path: "internal/migrations/seed_npc_definition.up.sql", loadedBy: "0012" },
  npcShopItem: { path: "internal/migrations/seed_npc_shop_item.up.sql", loadedBy: "0012" },
  worldItemSeed: { path: "internal/migrations/seed_world_item_seed.up.sql", loadedBy: "0012" },
  guildZone: { path: "internal/migrations/seed_guild_zone.up.sql", loadedBy: "0012" },
  levelItem: { path: "tmserver/content/TMsrv/run/LevelItem.txt", loadedBy: "0012" },
  guard: { path: "tmserver/content/TMsrv/run/Guard.txt", loadedBy: "0012" },
  mobMerc: { path: "tmserver/content/Common/Settings/MobMerc.txt", loadedBy: "0012" },
  gameConfig: { path: "tmserver/content/TMsrv/run/gameconfig.txt", loadedBy: "0012" },
  regions: { path: "tmserver/content/TMsrv/run/Regions.txt", loadedBy: "0012" },
};

const CONFIRMED_EFFECTS = new Set([
  "EF_NOTRADE", "EF_CLASS", "EF_AC", "EF_HP", "EF_MP", "EF_ITEMLEVEL", "EF_VOLATILE", "EF_RANGE",
]);

const sanitizeStats = new Map();
const failures = [];
const bump = (step, n = 1) => sanitizeStats.set(step, (sanitizeStats.get(step) ?? 0) + n);
const fail = (message) => failures.push(message);
const assert = (condition, message) => {
  if (!condition) fail(message);
};

// ---------------------------------------------------------------- insumo bruto

const sourceCache = new Map();

function gitShow(path) {
  return execFileSync("git", ["show", `origin/main:${path}`], {
    cwd: ROOT,
    maxBuffer: 512 * 1024 * 1024,
  });
}

function readSource(key) {
  if (sourceCache.has(key)) return sourceCache.get(key);
  const def = SRC[key];
  const bytes = gitShow(def.path);
  const entry = {
    key,
    path: def.path,
    loadedBy: def.loadedBy,
    bytes,
    sha256: createHash("sha256").update(bytes).digest("hex"),
    size: bytes.length,
    text: decodeSource(key, bytes),
  };
  sourceCache.set(key, entry);
  return entry;
}

/**
 * UTF-8 estrito. Se houver byte inválido, ele só é tolerado em linha de comentário
 * (`#`/`//`), que o parser descarta — caso contrário o gerador aborta em vez de perder
 * dado silenciosamente. Os 4 bytes inválidos do LevelItem.txt estão em `#TK ... AN?`.
 */
function decodeSource(key, bytes) {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    const lenient = new TextDecoder("utf-8").decode(bytes);
    const offences = lenient
      .split(/\r?\n/)
      .filter((line) => line.includes("\uFFFD") && !line.trimStart().startsWith("#"));
    if (offences.length) {
      throw new Error(`${key}: UTF-8 inválido em linha de dado: ${offences.slice(0, 3).join(" | ")}`);
    }
    bump("utf8_byte_invalido_em_comentario", (lenient.match(/\uFFFD/g) ?? []).length);
    return lenient;
  }
}

// -------------------------------------------------------------- sanitização

const INVISIBLES = /[\u200B-\u200F\u2060\uFEFF\u00AD]/g;
const CONTROLS = /[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F-\u009F]/g;
const NBSP = /[\u00A0\u2007\u202F]/g;

/** Texto canônico do banco: LF, NFC, sem BOM/invisível/controle, sem NBSP. */
function sanitizeText(value, field = "") {
  const start = String(value ?? "");
  let text = start;
  const apply = (step, re, replacement) => {
    const next = text.replace(re, replacement);
    if (next !== text) {
      bump(step);
      text = next;
    }
  };
  apply("crlf_para_lf", /\r\n?/g, "\n");
  apply("bom_removido", /\uFEFF/g, "");
  apply("invisivel_removido", INVISIBLES, "");
  apply("controle_removido", CONTROLS, "");
  apply("nbsp_para_espaco", NBSP, " ");
  if (text !== text.normalize("NFC")) {
    text = text.normalize("NFC");
    bump("nfc_recomposto");
  }
  if (text.includes("\uFFFD")) fail(`${field || "(campo)"}: U+FFFD no dado`);
  return text;
}

/** Rótulo: mesma sanitização + espaços colapsados (nome, slug, região, seção). */
function sanitizeLabel(value, field = "") {
  const text = sanitizeText(value, field);
  const collapsed = text.replace(/\s+/g, " ").trim();
  if (collapsed !== text) bump("espacos_colapsados");
  return collapsed;
}

/** Mesma dobra de `content_fold()` no SQL (só para relatório e checagem de duplicata). */
function foldName(value) {
  const from = "áàâãäåéèêëíìîïóòôõöúùûüçñýÿªº";
  const to = "aaaaaaeeeeiiiiooooouuuucnyyao";
  let out = "";
  for (const ch of String(value).toLowerCase()
    .replace(/\u00e6/g, "ae").replace(/\u0153/g, "oe").replace(/\u00df/g, "ss")) {
    const index = from.indexOf(ch);
    out += index >= 0 ? to[index] : ch;
  }
  return out.replace(/[^a-z0-9]+/g, " ").trim();
}

/**
 * Assinatura de CP949 lido como ISO-8859-1: letras que não aparecem em nome PT/EN.
 * `Foema_Anciã` e `Troll_Caçador@@` são português legítimo e ficam de fora.
 */
const CP949_SIGNATURE = new Set([
  0x00c7, 0x00c4, 0x00c5, 0x00b5, 0x00bb, 0x00bd, 0x00b9, 0x00ae, 0x00d0, 0x00ca,
  0x00cf, 0x00b1, 0x00b4, 0x00ec, 0x00f1, 0x00b8, 0x00e4, 0x00d7, 0x00ce, 0x00ef,
  0x00e5, 0x00b7, 0x00a3,
]);

/**
 * Mojibake coreano = tem letra da assinatura, ou os bytes recuperam Hangul inteiro.
 * A chave não é renomeada (é referência de cliente e de `npc_definition`); a leitura
 * em coreano vai em `name_recovered` quando a recuperação é total.
 */
function isCp949Mojibake(value) {
  const codepoints = [...value].map((ch) => ch.codePointAt(0));
  if (!codepoints.some((cp) => cp > 0x7f)) return false;
  if (codepoints.some((cp) => CP949_SIGNATURE.has(cp))) return true;
  return recoverCp949(value) !== null;
}

/**
 * Nome coreano (CP949) que foi decodificado como ISO-8859-1 antes de virar UTF-8: o texto
 * atual é latin1 de bytes coreanos. Recuperação só é aceita se os bytes formarem EUC-KR
 * válido com sílaba Hangul — nomes portugueses (`Foema_Anciã`, `Troll_Caçador`) não passam.
 */
function recoverCp949(value) {
  const codepoints = [...value].map((ch) => ch.codePointAt(0));
  if (codepoints.some((cp) => cp > 0xff) || codepoints.every((cp) => cp < 0x80)) return null;
  const decoded = new TextDecoder("euc-kr").decode(Buffer.from(codepoints));
  if (decoded.includes("\uFFFD") || !/[\uAC00-\uD7A3]/.test(decoded)) return null;
  return decoded;
}

// ------------------------------------------------------------------- helpers SQL

const sqlText = (v) => `'${String(v).replace(/'/g, "''")}'`;
const sqlJson = (v) => `'${JSON.stringify(v).replace(/'/g, "''")}'::jsonb`;
const sqlInt = (v) => (v === null || v === undefined ? "NULL" : String(v));
const sqlBool = (v) => (v === null || v === undefined ? "NULL" : v ? "true" : "false");
const sqlTextArray = (arr) =>
  arr.length ? `ARRAY[${arr.map(sqlText).join(", ")}]::text[]` : "'{}'::text[]";
const sqlIntArray = (arr) => `ARRAY[${arr.map((n) => String(n)).join(", ")}]::smallint[]`;

function jsonRows(sql) {
  const rows = [];
  const re = /\$json\$(\{[\s\S]*?\})\$json\$::json/g;
  let match;
  while ((match = re.exec(sql)) !== null) rows.push(JSON.parse(match[1]));
  return rows;
}

/** Um INSERT por lote (statement único grande, sem parâmetro): menos verboso no diff. */
function insertBatches(table, columns, rows, size = 200) {
  const out = [];
  for (let i = 0; i < rows.length; i += size) {
    out.push(`INSERT INTO ${table} (${columns}) VALUES`);
    out.push(`${rows.slice(i, i + size).join(",\n")};`, "");
  }
  return out;
}

function groupBy(list, keyFn) {
  const map = new Map();
  for (const item of list) {
    const key = keyFn(item);
    if (!map.has(key)) map.set(key, []);
    map.get(key).push(item);
  }
  return map;
}

// ------------------------------------------------------- classificação (0009/0010)

function norm(value) {
  return String(value)
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

const ITEM_COLUMNS = `item_index, name, price, grade, unique_id, extra, pos, req_level,
    req_str, req_int, req_dex, req_con, volatile, attack_range, effects, effect_names, fields,
    kind, primary_system, event_only, tradeable, stackable, class_mask,
    classification_source, confidence`;

const SKILL_COLUMNS = `skill_index, name, skill_point, target_type, mana_spent, delay_ticks,
    range_tiles, instance_type, instance_value, tick_type, tick_value, affect_type, affect_value,
    affect_time, instance_attribute, tick_attribute, aggressive, max_target, party, affect_resist,
    passive, class_code, lineage_code, primary_system, skill_role,
    classification_source, confidence`;

const KIND_RULES = [
  [/\b(escudo|shield|broquel|buckler)\b/, "shield"],
  [/\b(elmo|capuz|capacete|helmet|cap|hood|coroa)\b/, "helmet"],
  [/\b(botas?|sapatos?|boots?|grevas)\b/, "boots"],
  [/\b(luvas?|gloves?|manoplas?)\b/, "gloves"],
  [/\b(capa|manto|cloak|cape)\b/, "cloak"],
  [/\b(armadura|peitoral|tunica|robe|vestido|skirt|calca|pants|brace|body|suit|traje)\b/, "armor"],
  [/\b(anel|amuleto|bracelete|pingente|ring|amulet|necklace|earring|brinco)\b/, "accessory"],
  [/\b(espada|machado|arco|garra|clava|cajado|staff|vara|lan[c\u00e7]a|adaga|martelo|sword|axe|bow|claw|mace|dagger|hammer|spear|wand|scimitar|katana|foice|scythe)\b/, "weapon"],
  [/\b(pocao|potion|kit|elixir|ervas?|herb|comida|food|bebida)\b/, "consumable"],
  [/\b(flecha|arrow|municao|ammo|dardo)\b/, "ammo"],
  [/\b(pergaminho|scroll|chave|key|mapa|map|bilhete|letter|carta)\b/, "quest"],
  [/\b(cupom|coupon|ticket|ingresso|comemorativ|evento|event)\b/, "cash"],
  [/\b(caixa|box|pacote|package|bau|chest|saco|bag)\b/, "container"],
  [/\b(asa|wings?|aura|efeito)\b/, "cosmetic"],
  [/\b(po|poeira|powder|dust|ori|lac|minerio|ore|sucata|scrap|essencia|essence|pedra|stone|cristal|crystal|joia|jewel|safira|sapphire|gema|gem|lingote|barra|bar)\b/, "material"],
];

function classifyKind(name) {
  const n = ` ${norm(name).replace(/_/g, " ")} `;
  for (const [re, kind] of KIND_RULES) if (re.test(n)) return kind;
  return null;
}

function classifySystem(name, kind, effects) {
  const n = ` ${norm(name).replace(/_/g, " ")} `;
  if (/\b(evento|event|cupom|coupon|ticket|ingresso)\b/.test(n)) return "evento";
  if (/\b(safira|sapphire|manto de|hekalotia|akeronia|akelonia)\b/.test(n)) return "reino";
  if (/\b(poeira|po|powder|ori|lac|pedra|stone|sabio|wise|lunar|replecao)\b/.test(n)) return "refino";
  if (/\b(pergaminho|scroll|chave|key)\b/.test(n)) return "quest";
  if (/\b(joia|jewel|gema|gem)\b/.test(n)) return "craft";
  if (effects.includes("EF_NOTRADE") && kind === "consumable") return "donate";
  if (["weapon", "shield", "armor", "helmet", "boots", "gloves", "cloak", "accessory"].includes(kind)) {
    return "combate";
  }
  if (kind === "consumable" || kind === "ammo") return "combate";
  if (kind === "quest") return "quest";
  if (kind === "cash") return "evento";
  return null;
}

function classifyEventOnly(name) {
  const n = ` ${norm(name).replace(/_/g, " ")} `;
  return /[\b](cupom|coupon|ticket|ingresso|evento|event|comemorativ|aniversario|natal|halloween)[\b]/.test(n)
    ? true
    : null;
}

const STACKABLE = new Set(["consumable", "material", "ammo", "quest", "cash", "container"]);

const SKILL_MAP = {
  ataque_fatal: { cls: "huntress", lineage: "survival", role: "damage", system: "combate" },
  carga: { cls: "transknight", lineage: "trans", role: "damage", system: "combate" },
  assalto: { cls: "transknight", lineage: "trans", role: "buff", system: "combate" },
  bencao_divina: { cls: "transknight", lineage: "faith", role: "passive", system: "combate" },
  aura_da_vida: { cls: "transknight", lineage: "faith", role: "buff", system: "combate" },
  choque_divino: { cls: "transknight", lineage: "faith", role: "damage", system: "combate" },
  clod_attack: { cls: "transknight", lineage: "magic_sword", role: "debuff", system: "combate" },
  blaze_luncher: { cls: "transknight", lineage: "magic_sword", role: "debuff", system: "combate" },
  another_change: { cls: "foema", lineage: "special_magic", role: "passive", system: "combate" },
  chama_resistente: { cls: "beastmaster", lineage: "elemental", role: "summon", system: "combate" },
  chamas_etereas: { cls: "beastmaster", lineage: "elemental", role: "damage", system: "combate" },
  ambush: { cls: "beastmaster", lineage: "nature", role: "debuff", system: "combate" },
};

// --------------------------------------------------------------------- parsers

/**
 * LevelItem.txt: "# SEQUENCIA, LEVEL, CLASSE, TYPE, ITEM" + 6 extras.
 * O bloco inteiro do arquivo aparece repetido (318 linhas → 161 de conteúdo); a chave de
 * deduplicação inclui `seq`, porque dois `seq` têm duas linhas distintas no insumo.
 */
function parseLevelItem(text) {
  const seen = new Set();
  const rows = [];
  let duplicated = 0;
  text.split("\n").forEach((line, index) => {
    const raw = line.trim();
    if (!raw || raw.startsWith("#")) return;
    const numbers = raw.split(/\s+/).map(Number);
    if (numbers.length < 5 || numbers.some((n) => !Number.isFinite(n))) {
      throw new Error(`LevelItem linha ${index + 1}: ${raw}`);
    }
    const [seq, level, classId, typeId, itemIndex, ...extras] = numbers;
    const key = [seq, level, classId, typeId, itemIndex, ...extras].join("|");
    if (seen.has(key)) {
      duplicated += 1;
      return;
    }
    seen.add(key);
    rows.push({ seq, level, classId, typeId, itemIndex, extras, sourceLine: index + 1 });
  });
  bump("level_item_linha_repetida", duplicated);
  return rows.map((row, i) => ({ ...row, ordinal: i + 1 }));
}

/** Guard.txt: "<guard_index>,<npc_index>" + N blocos "[8 números]" (bloco UNVERIFIED). */
function parseGuard(text) {
  const rows = [];
  text.split("\n").forEach((line, index) => {
    const raw = line.trim();
    if (!raw || raw.startsWith("#")) return;
    const comma = raw.indexOf(",");
    const open = raw.indexOf("[");
    if (comma < 0 || open < 0) throw new Error(`Guard linha ${index + 1}: ${raw}`);
    const blocks = [...raw.slice(comma + 1).matchAll(/\[([^\]]*)\]/g)].map((m) =>
      m[1].split(",").map((v) => Number(v.trim())),
    );
    for (const block of blocks) {
      if (block.length !== 8 || block.some((n) => !Number.isFinite(n))) {
        throw new Error(`Guard linha ${index + 1}: bloco de ${block.length} valores`);
      }
    }
    rows.push({
      guardIndex: Number(raw.slice(0, comma)),
      npcIndex: Number(raw.slice(comma + 1, open)),
      blocks,
      sourceLine: index + 1,
    });
  });
  return rows;
}

/** MobMerc.txt: blocos "# [n]" com MobName/GenerateIndex/Item_n/RenewTime/RebornTime. */
function parseMobMerc(text) {
  const rows = [];
  let current = null;
  text.split("\n").forEach((line, index) => {
    const raw = line.trim();
    if (/^#\s*\[\s*\d+\s*\]/.test(raw)) {
      if (current) rows.push(current);
      current = { templateName: "", generatorIndex: 0, renew: 0, reborn: 0, items: [], sourceLine: index + 1 };
      return;
    }
    if (!current || !raw || raw.startsWith("//")) return;
    const pair = raw.match(/^([A-Za-z_0-9]+):\s*(.*)$/);
    if (!pair) return;
    const [, key, value] = pair;
    if (key === "MobName") current.templateName = value.trim();
    else if (key === "GenerateIndex") current.generatorIndex = Number(value.trim());
    else if (key === "RenewTime") current.renew = Number(value.trim());
    else if (key === "RebornTime") current.reborn = Number(value.trim());
    else if (key.startsWith("Item_")) {
      current.items.push({
        slot: Number(key.slice(5)),
        values: value.split(/\s+/).map((v) => (v === "-" ? null : Number(v))),
      });
    }
  });
  if (current) rows.push(current);
  return rows;
}

/** Seções matriciais do gameconfig: quantas linhas formam um bloco. */
const MATRIX_SECTIONS = { "Item Drop Bonus Settings": 1, "Treasure Settings": 2 };

/**
 * gameconfig.txt: seções "<Nome> Settings:" com pares chave/valor. `evstart` tem 4
 * números, então o valor inteiro só existe quando sobra um token; o resto vai para
 * `value_text` — o arquivo é a verdade, a semântica fica para quem portar a regra.
 */
function parseGameConfig(text) {
  const settings = [];
  const matrices = [];
  let section = null;
  let blockSize = 1;
  let pending = [];

  const flush = () => {
    pending.forEach((entry) => {
      matrices.push({
        section: entry.section,
        blockIndex: Math.floor(entry.index / entry.blockSize),
        lineIndex: entry.index % entry.blockSize,
        values: entry.values,
        sourceLine: entry.line,
      });
    });
    pending = [];
  };

  text.split("\n").forEach((line, index) => {
    const sourceLine = index + 1;
    const raw = line.trim();
    if (!raw) return;
    if (raw.endsWith("Settings:")) {
      flush();
      section = sanitizeLabel(raw.replace(/:\s*$/, ""), "gameconfig.seção");
      blockSize = MATRIX_SECTIONS[section] ?? 1;
      return;
    }
    if (!section) throw new Error(`gameconfig linha ${sourceLine} fora de seção: ${raw}`);
    if (section in MATRIX_SECTIONS) {
      const values = raw.split(/\s+/).map(Number);
      if (values.some((n) => !Number.isFinite(n))) {
        throw new Error(`gameconfig linha ${sourceLine} não numérica: ${raw}`);
      }
      pending.push({ section, blockSize, index: pending.length, values, line: sourceLine });
      return;
    }
    // Uma linha traz VÁRIAS chaves: "evindex 0 evdelete 0 ... evstart 0 0 0 1".
    // O valor de cada chave são os números seguintes; `evstart` tem quatro.
    let key = null;
    let values = [];
    const commit = () => {
      if (key === null) return;
      settings.push({
        section,
        key: sanitizeLabel(key, `${section}.chave`),
        valueInt: values.length === 1 ? values[0] : null,
        valueText: values.join(" "),
        sourceLine,
      });
    };
    for (const token of raw.split(/\s+/)) {
      const numeric = Number.isFinite(Number(token));
      if (numeric) {
        if (key === null) throw new Error(`gameconfig linha ${sourceLine}: valor sem chave (${token})`);
        values.push(Number(token));
        continue;
      }
      if (key !== null && values.length === 0) {
        throw new Error(`gameconfig linha ${sourceLine}: chave sem valor (${key})`);
      }
      commit();
      key = token;
      values = [];
    }
    commit();
  });
  flush();
  return { settings, matrices };
}

/** Regions.txt: "<x1>,<y1>,<x2>,<y2> = Nome" (o nome pode repetir com outro retângulo). */
function parseRegions(text) {
  const rows = [];
  let inverted = 0;
  let outside = 0;
  text.split("\n").forEach((line, index) => {
    const raw = line.trim();
    if (!raw || raw.startsWith("//")) return;
    const match = raw.match(/^(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*=\s*(.+)$/);
    if (!match) throw new Error(`Regions linha ${index + 1}: ${raw}`);
    const x1 = Number(match[1]);
    const y1 = Number(match[2]);
    const x2 = Number(match[3]);
    const y2 = Number(match[4]);
    // Retângulo invertido é o MESMO conjunto de tiles: canonizar é perda zero.
    const canon = [Math.min(x1, x2), Math.min(y1, y2), Math.max(x1, x2), Math.max(y1, y2)];
    if (canon[0] !== x1 || canon[1] !== y1 || canon[2] !== x2 || canon[3] !== y2) inverted += 1;
    // O cliente trabalha em 0..4095; uma região do legado passa disso (Dugeon_1_Andar_Hidra).
    if (Math.max(x1, x2) > 4095 || Math.max(y1, y2) > 4095) outside += 1;
    rows.push({
      x1: canon[0], y1: canon[1], x2: canon[2], y2: canon[3],
      name: sanitizeLabel(match[5], "region.name"),
      sourceLine: index + 1,
    });
  });
  if (inverted) bump("region_retangulo_invertido_canonizado", inverted);
  if (outside) bump("region_acima_do_espaco_do_cliente", outside);
  return rows;
}

// ------------------------------------------------------------------ migração 0009

function buildItems() {
  const source = readSource("itemCatalog");
  const rows = jsonRows(source.text);
  const effectNames = new Set();
  const items = rows.map((row) => {
    const rawFields = Array.isArray(row.fields) ? row.fields : [];
    const fieldEffects = rawFields.filter((f) => typeof f === "string" && f.startsWith("EF_"));
    const effects = Array.isArray(row.effects) ? row.effects : [];
    const names = [...new Set([...fieldEffects, ...(Array.isArray(row.effect_names) ? row.effect_names : [])])];
    names.forEach((n) => effectNames.add(n));
    const name = sanitizeLabel(row.name, `item_catalog#${row.item_index}.name`);
    const kind = classifyKind(name);
    return {
      ...row,
      name,
      names,
      effects,
      rawFields,
      kind,
      system: classifySystem(name, kind, names),
      eventOnly: classifyEventOnly(name),
      tradeable: !names.includes("EF_NOTRADE"),
      stackable: kind === null ? null : STACKABLE.has(kind),
      source: kind === null ? "legacy" : "rule:name_v1",
      confidence: kind === null ? null : 60,
    };
  });

  const duplicates = groupBy(items, (it) => foldName(it.name));
  const accidental = groupBy(
    items,
    (it) => [foldName(it.name), it.grade, it.price, JSON.stringify(it.effects), JSON.stringify(it.rawFields)].join("|"),
  );
  return {
    source,
    items,
    effectNames,
    report: {
      linhas: items.length,
      nomes_distintos: new Set(items.map((it) => foldName(it.name))).size,
      nomes_repetidos_grupos: [...duplicates.values()].filter((v) => v.length > 1).length,
      nomes_repetidos_itens: [...duplicates.values()].filter((v) => v.length > 1).reduce((s, v) => s + v.length, 0),
      duplicata_acidental: [...accidental.values()].filter((v) => v.length > 1).length,
    },
  };
}

function emitItemSql({ source, items, effectNames }) {
  const effectLines = [...effectNames].sort().map((name) => {
    const label = name.replace(/^EF_/, "").replace(/_/g, " ").toLowerCase();
    const category = name === "EF_NOTRADE" ? "economia" : name === "EF_CLASS" ? "restricao" : "score";
    return `    (${sqlText(name)}, ${sqlText(label)}, ${sqlText(category)}, ${CONFIRMED_EFFECTS.has(name) ? "true" : "false"})`;
  });

  const values = items.map((it) => `(${[
    sqlInt(it.item_index), sqlText(it.name), sqlInt(it.price ?? 0), sqlInt(it.grade ?? 0),
    sqlInt(it.unique_id ?? 0), sqlInt(it.extra ?? 0), sqlInt(it.pos ?? 0),
    sqlInt(it.req_level ?? 0), sqlInt(it.req_str ?? 0), sqlInt(it.req_int ?? 0),
    sqlInt(it.req_dex ?? 0), sqlInt(it.req_con ?? 0), sqlInt(it.volatile),
    sqlInt(it.attack_range), sqlJson(it.effects), sqlTextArray(it.names), sqlJson(it.rawFields),
    it.kind ? sqlText(it.kind) : "NULL", it.system ? sqlText(it.system) : "NULL",
    sqlBool(it.eventOnly), sqlBool(it.tradeable), sqlBool(it.stackable), "NULL",
    sqlText(it.source), sqlInt(it.confidence),
  ].join(", ")})`);

  return [
    "-- 0017_seed_item_catalog: catálogo de itens do legado (3.220) + vocabulário EF_* (ADR-0009).",
    "-- GERADO por tools/content/gen-seed.mjs — NÃO EDITAR À MÃO.",
    `-- Fonte: origin/main:${source.path} (sha256 ${source.sha256}).`,
    "-- Sanitização: NFC, LF, sem BOM/invisível/controle (modelo-banco §6.6).",
    "",
    "-- migrate:up",
    "INSERT INTO item_effect (name, label, category, confirmed) VALUES",
    effectLines.join(",\n"),
    "ON CONFLICT (name) DO NOTHING;",
    "",
    ...insertBatches("item_catalog", ITEM_COLUMNS, values, 200),
    "UPDATE content_version SET version = 2, note = 'item_catalog + item_effect (3.220 itens)' WHERE id = 1;",
    "",
    "-- migrate:down",
    "DELETE FROM item_catalog;",
  ].join("\n");
}

// ------------------------------------------------------------------ migração 0010

function buildSkills() {
  const source = readSource("skillCatalog");
  const skills = jsonRows(source.text).map((row) => {
    const mapped = SKILL_MAP[norm(row.name)] ?? null;
    return {
      ...row,
      name: sanitizeLabel(row.name ?? "", `skill_catalog#${row.skill_index}.name`),
      cls: mapped?.cls ?? null,
      lineage: mapped?.lineage ?? null,
      role: mapped?.role ?? null,
      system: mapped?.system ?? null,
      source: mapped ? "rule:name_confirmed" : "unmapped",
      confidence: mapped ? 80 : null,
    };
  });
  return { source, skills };
}

function emitSkillSql({ source, skills }) {
  const values = skills.map((sk) => `(${[
    sqlInt(sk.skill_index), sqlText(sk.name), sqlInt(sk.skill_point ?? 0),
    sqlInt(sk.target_type ?? 0), sqlInt(sk.mana_spent ?? 0), sqlInt(sk.delay_ticks ?? 0),
    sqlInt(sk.range_tiles ?? 0), sqlInt(sk.instance_type ?? 0), sqlInt(sk.instance_value ?? 0),
    sqlInt(sk.tick_type ?? 0), sqlInt(sk.tick_value ?? 0), sqlInt(sk.affect_type ?? 0),
    sqlInt(sk.affect_value ?? 0), sqlInt(sk.affect_time ?? 0), sqlInt(sk.instance_attribute ?? 0),
    sqlInt(sk.tick_attribute ?? 0), sqlInt(sk.aggressive ?? 0), sqlInt(sk.max_target ?? 0),
    sqlInt(sk.party ?? 0), sqlInt(sk.affect_resist ?? 0), sqlInt(sk.passive ?? 0),
    sk.cls ? sqlText(sk.cls) : "NULL", sk.lineage ? sqlText(sk.lineage) : "NULL",
    sk.system ? sqlText(sk.system) : "NULL", sk.role ? sqlText(sk.role) : "NULL",
    sqlText(sk.source), sqlInt(sk.confidence),
  ].join(", ")})`);

  return [
    "-- 0018_seed_skill_catalog: catálogo de skills do legado (151) (ADR-0009).",
    "-- GERADO por tools/content/gen-seed.mjs — NÃO EDITAR À MÃO.",
    `-- Fonte: origin/main:${source.path} (sha256 ${source.sha256}).`,
    "",
    "-- migrate:up",
    ...insertBatches("skill_catalog", SKILL_COLUMNS, values, 200),
    "UPDATE content_version SET version = 3, note = 'skill_catalog (151 skills)' WHERE id = 1;",
    "",
    "-- migrate:down",
    "DELETE FROM skill_catalog;",
  ].join("\n");
}

// ------------------------------------------------------------------ migração 0012

function buildWorld() {
  const templates = jsonRows(readSource("npcTemplate").text).map((row) => {
    const templateName = sanitizeLabel(row.template_name, "npc_template.template_name");
    const hex = String(row.blob ?? "").replace(/^\\x/, "");
    const blob = Buffer.from(hex, "hex");
    const computed = createHash("sha256").update(blob).digest("hex");
    assert(blob.length === 816, `npc_template ${templateName}: blob de ${blob.length} bytes`);
    assert(computed === row.sha256, `npc_template ${templateName}: sha256 diverge do blob`);
    const encodingOrigin = isCp949Mojibake(templateName) ? "cp949-as-latin1" : "utf8";
    const recovered = encodingOrigin === "cp949-as-latin1" ? recoverCp949(templateName) : null;
    return {
      templateName,
      fileName: sanitizeLabel(row.file_name, "npc_template.file_name"),
      sourceSize: row.source_size,
      sha256: row.sha256,
      blob: `\\x${hex}`,
      encodingOrigin,
      recovered,
    };
  });

  const definitions = jsonRows(readSource("npcDefinition").text).map((row) => ({
    id: row.id,
    slug: sanitizeLabel(row.slug, `npc_definition#${row.id}.slug`),
    templateName: sanitizeLabel(row.template_name, `npc_definition#${row.id}.template_name`),
    displayName: sanitizeLabel(row.display_name ?? "", `npc_definition#${row.id}.display_name`),
    enabled: row.enabled,
    mapId: row.map_id,
    posX: row.pos_x,
    posY: row.pos_y,
    routeType: row.route_type,
    merchant: row.merchant,
    origin: row.origin,
    generatorIndex: row.generator_index,
    followerTemplate: sanitizeLabel(row.follower_template ?? "", `npc_definition#${row.id}.follower_template`),
    minuteGenerate: row.minute_generate,
    minGroup: row.min_group,
    maxGroup: row.max_group,
    maxNumMob: row.max_num_mob,
    formation: row.formation,
    generatorData: row.generator_data ?? {},
  }));

  const shopItems = jsonRows(readSource("npcShopItem").text).map((row) => ({
    npcId: row.npc_id,
    slot: row.slot,
    itemIndex: row.item_index,
    eff1: row.eff1, effv1: row.effv1,
    eff2: row.eff2, effv2: row.effv2,
    eff3: row.eff3, effv3: row.effv3,
    quantity: row.quantity,
  }));

  const worldItems = jsonRows(readSource("worldItemSeed").text).map((row) => ({
    id: row.id, itemIndex: row.item_index, posX: row.pos_x, posY: row.pos_y,
    rotate: row.rotate, enabled: row.enabled,
  }));

  const guildZones = jsonRows(readSource("guildZone").text).map((row) => ({
    clan: row.clan, zone: row.zone, victory: row.victory, cityTax: row.city_tax,
    taxVault: row.tax_vault, chargeGuild: row.charge_guild,
    challengeGuild: row.challenge_guild, challengeMoney: row.challenge_money,
  }));

  const levelItems = parseLevelItem(readSource("levelItem").text);
  const guards = parseGuard(readSource("guard").text);
  const mobMercs = parseMobMerc(readSource("mobMerc").text).map((row) => ({
    ...row,
    templateName: sanitizeLabel(row.templateName, "mob_merc.template_name"),
  }));
  const { settings, matrices } = parseGameConfig(readSource("gameConfig").text);
  const regions = parseRegions(readSource("regions").text);

  // ---- invariantes (o gerador não emite seed que o banco vá recusar)
  const templateNames = new Set(templates.map((t) => t.templateName));
  assert(templates.length === 1991, `npc_template: ${templates.length} linhas (esperado 1991)`);
  assert(definitions.length === 6100, `npc_definition: ${definitions.length} linhas (esperado 6100)`);
  assert(shopItems.length === 6706, `npc_shop_item: ${shopItems.length} linhas (esperado 6706)`);
  for (const def of definitions) {
    if (!templateNames.has(def.templateName)) fail(`npc_definition#${def.id}: template ausente ${def.templateName}`);
  }
  const definitionIds = new Set(definitions.map((d) => d.id));
  for (const item of shopItems) {
    if (!definitionIds.has(item.npcId)) fail(`npc_shop_item: npc ${item.npcId} inexistente`);
    if (item.slot < 0 || item.slot > 26) fail(`npc_shop_item ${item.npcId}/${item.slot}: slot fora de 0..26`);
    if (item.quantity < 1 || item.quantity > 255) fail(`npc_shop_item ${item.npcId}/${item.slot}: quantity ${item.quantity}`);
  }
  const shopKeys = groupBy(shopItems, (s) => `${s.npcId}/${s.slot}`);
  assert([...shopKeys.values()].every((v) => v.length === 1), "npc_shop_item: chave (npc_id, slot) duplicada");
  for (const guard of guards) {
    if (!definitionIds.has(guard.npcIndex)) fail(`guard_spawn ${guard.guardIndex}: npc ${guard.npcIndex} inexistente`);
  }
  for (const row of levelItems) {
    if (row.classId < 0 || row.classId > 3) fail(`level_item ${row.ordinal}: classe ${row.classId}`);
    if (row.typeId < -1 || row.typeId > 3) fail(`level_item ${row.ordinal}: type ${row.typeId}`);
    if (row.level < 1 || row.level > 255) fail(`level_item ${row.ordinal}: level ${row.level}`);
    if (row.itemIndex <= 0) fail(`level_item ${row.ordinal}: item ${row.itemIndex}`);
  }
  for (const region of regions) {
    if (!(region.x1 <= region.x2 && region.y1 <= region.y2)) fail(`region ${region.name}: retângulo invertido`);
    for (const v of [region.x1, region.y1, region.x2, region.y2]) {
      if (v < 0) fail(`region ${region.name}: coordenada negativa ${v}`);
    }
  }
  const settingKeys = groupBy(settings, (s) => `${s.section}|${s.key}`);
  for (const [key, rows] of settingKeys) if (rows.length > 1) fail(`game_setting duplicada: ${key}`);

  if (process.argv.includes("--verbose")) {
    console.error("matrizes:", JSON.stringify(matrices.map((m) => [m.section, m.blockIndex, m.lineIndex, m.values.length])));
    console.error(
      "mojibake:",
      templates.filter((t) => t.encodingOrigin === "cp949-as-latin1").map((t) => t.templateName).join(" | "),
    );
  }

  return {
    templates, definitions, shopItems, worldItems, guildZones,
    levelItems, guards, mobMercs, settings, matrices, regions,
  };
}

function emitWorldSql(world) {
  const out = [
    "-- 0019_seed_mundo: mundo legado no banco (ADR-0009) — NPCs, lojas, prêmios por nível,",
    "-- spawns de guarda, mercador, configuração de jogo e regiões.",
    "-- GERADO por tools/content/gen-seed.mjs — NÃO EDITAR À MÃO.",
    "-- Insumos e hashes: tabela content_source (gravada por esta migração).",
    "-- Sanitização: NFC, CRLF→LF, sem BOM/invisível/controle (modelo-banco §6.6).",
    "",
    "-- migrate:up",
  ];

  // proveniência: todos os insumos que alimentaram migrações de conteúdo
  const transformSeed = "utf-8 estrito; NFC; LF; JSON $json$ íntegro";
  const transformText = "utf-8 estrito (byte inválido só em comentário); CRLF→LF; comentários descartados; NFC";
  const provenance = [
    ["itemCatalog", 3220, transformSeed],
    ["skillCatalog", 151, transformSeed],
    ["npcTemplate", 1991, transformSeed],
    ["npcDefinition", 6100, transformSeed],
    ["npcShopItem", 6706, transformSeed],
    ["worldItemSeed", 49, transformSeed],
    ["guildZone", 5, transformSeed],
    ["levelItem", world.levelItems.length, transformText],
    ["guard", world.guards.length, transformText],
    ["mobMerc", world.mobMercs.length, transformText],
    ["gameConfig", world.settings.length + world.matrices.length, transformText],
    ["regions", world.regions.length, transformText],
  ].map(([key, rows, transform]) => {
    const source = readSource(key);
    return `(${[
      sqlText(`origin/main:${source.path}`), sqlText(source.sha256), sqlInt(source.size),
      sqlInt(rows), sqlText(source.loadedBy), sqlText(transform),
    ].join(", ")})`;
  });
  out.push(...insertBatches(
    "content_source",
    "path, sha256, size_bytes, row_count, loaded_by, transform",
    provenance,
    50,
  ));

  // npc_template
  out.push(...insertBatches(
    "npc_template",
    "template_name, file_name, source_size, sha256, blob, encoding_origin, name_recovered",
    world.templates.map((t) => `(${[
      sqlText(t.templateName), sqlText(t.fileName), sqlInt(t.sourceSize), sqlText(t.sha256),
      `${sqlText(t.blob)}::bytea`, sqlText(t.encodingOrigin),
      t.recovered ? sqlText(t.recovered) : "NULL",
    ].join(", ")})`),
    100,
  ));

  // npc_definition
  out.push(...insertBatches(
    "npc_definition",
    `id, slug, template_name, display_name, enabled, map_id, pos_x, pos_y, route_type,
     merchant, origin, generator_index, follower_template, minute_generate, min_group,
     max_group, max_num_mob, formation, generator_data`,
    world.definitions.map((d) => `(${[
      sqlInt(d.id), sqlText(d.slug), sqlText(d.templateName), sqlText(d.displayName),
      sqlBool(d.enabled), sqlInt(d.mapId), sqlInt(d.posX), sqlInt(d.posY), sqlInt(d.routeType),
      sqlInt(d.merchant), sqlText(d.origin), sqlInt(d.generatorIndex), sqlText(d.followerTemplate),
      sqlInt(d.minuteGenerate), sqlInt(d.minGroup), sqlInt(d.maxGroup), sqlInt(d.maxNumMob),
      sqlInt(d.formation), sqlJson(d.generatorData),
    ].join(", ")})`),
    150,
  ));

  // npc_shop_item
  out.push(...insertBatches(
    "npc_shop_item",
    "npc_id, slot, item_index, eff1, effv1, eff2, effv2, eff3, effv3, quantity",
    world.shopItems.map((s) => `(${[
      sqlInt(s.npcId), sqlInt(s.slot), sqlInt(s.itemIndex), sqlInt(s.eff1), sqlInt(s.effv1),
      sqlInt(s.eff2), sqlInt(s.effv2), sqlInt(s.eff3), sqlInt(s.effv3), sqlInt(s.quantity),
    ].join(", ")})`),
    200,
  ));

  // level_item
  out.push(...insertBatches(
    "level_item",
    "ordinal, source_seq, source_line, level, class_id, type_id, item_index, extras",
    world.levelItems.map((r) => `(${[
      sqlInt(r.ordinal), sqlInt(r.seq), sqlInt(r.sourceLine), sqlInt(r.level),
      sqlInt(r.classId), sqlInt(r.typeId), sqlInt(r.itemIndex), sqlJson(r.extras),
    ].join(", ")})`),
    100,
  ));

  // guard_spawn + guard_spawn_reward
  out.push(...insertBatches(
    "guard_spawn",
    "guard_index, npc_id, source_line",
    world.guards.map((g) => `(${[sqlInt(g.guardIndex), sqlInt(g.npcIndex), sqlInt(g.sourceLine)].join(", ")})`),
    50,
  ));
  const rewards = world.guards.flatMap((g) =>
    g.blocks.map((values, slot) => `(${[sqlInt(g.guardIndex), sqlInt(slot), sqlIntArray(values)].join(", ")})`),
  );
  out.push(...insertBatches(
    "guard_spawn_reward",
    "guard_index, req_slot, values",
    rewards,
    50,
  ));

  // mob_merc
  out.push(...insertBatches(
    "mob_merc",
    "generator_index, template_name, renew_seconds, reborn_seconds, items",
    world.mobMercs.map((m) => `(${[
      sqlInt(m.generatorIndex), sqlText(m.templateName), sqlInt(m.renew), sqlInt(m.reborn),
      sqlJson(m.items),
    ].join(", ")})`),
    50,
  ));

  // game_setting + game_matrix
  out.push(...insertBatches(
    "game_setting",
    "section, key, value_int, value_text, source_line",
    world.settings.map((s) => `(${[
      sqlText(s.section), sqlText(s.key), sqlInt(s.valueInt), sqlText(s.valueText), sqlInt(s.sourceLine),
    ].join(", ")})`),
    100,
  ));
  out.push(...insertBatches(
    "game_matrix",
    "section, block_index, line_index, values, source_line",
    world.matrices.map((m) => `(${[
      sqlText(m.section), sqlInt(m.blockIndex), sqlInt(m.lineIndex), sqlIntArray(m.values),
      sqlInt(m.sourceLine),
    ].join(", ")})`),
    50,
  ));

  // region (id é IDENTITY: sem coluna de id no INSERT)
  out.push(...insertBatches(
    "region",
    "name, x1, y1, x2, y2, source_line",
    world.regions.map((r) => `(${[
      sqlText(r.name), sqlInt(r.x1), sqlInt(r.y1), sqlInt(r.x2), sqlInt(r.y2), sqlInt(r.sourceLine),
    ].join(", ")})`),
    100,
  ));

  // world_item_seed + guild_zone
  out.push(...insertBatches(
    "world_item_seed",
    "id, item_index, pos_x, pos_y, rotate, enabled",
    world.worldItems.map((w) => `(${[
      sqlInt(w.id), sqlInt(w.itemIndex), sqlInt(w.posX), sqlInt(w.posY), sqlInt(w.rotate), sqlBool(w.enabled),
    ].join(", ")})`),
    50,
  ));
  out.push(...insertBatches(
    "guild_zone",
    `clan, zone, victory, city_tax, tax_vault, charge_guild, challenge_guild, challenge_money`,
    world.guildZones.map((z) => `(${[
      sqlInt(z.clan), sqlInt(z.zone), sqlInt(z.victory), sqlInt(z.cityTax), sqlInt(z.taxVault),
      sqlInt(z.chargeGuild), sqlInt(z.challengeGuild), sqlInt(z.challengeMoney),
    ].join(", ")})`),
    50,
  ));

  out.push(
    "UPDATE content_version SET version = 4, note = 'mundo: npc/lojas/prêmios/regiões (1991 templates, 6100 NPCs)' WHERE id = 1;",
    "",
    "-- migrate:down",
    "DELETE FROM guild_zone;",
    "DELETE FROM world_item_seed;",
    "DELETE FROM region;",
    "DELETE FROM game_matrix;",
    "DELETE FROM game_setting;",
    "DELETE FROM mob_merc;",
    "DELETE FROM guard_spawn_reward;",
    "DELETE FROM guard_spawn;",
    "DELETE FROM level_item;",
    "DELETE FROM npc_shop_item;",
    "DELETE FROM npc_definition;",
    "DELETE FROM npc_template;",
    "DELETE FROM content_source;",
  );
  return out.join("\n");
}

// ------------------------------------------------------------------------- main

function main() {
  const item = buildItems();
  const skill = buildSkills();
  const world = buildWorld();

  const skillMapped = skill.skills.filter((s) => s.cls !== null).length;
  const report = {
    fontes: [...sourceCache.values()].map((s) => ({
      insumo: s.path,
      bytes: s.size,
      sha256: s.sha256.slice(0, 16),
      migracao: s.loadedBy,
    })),
    itens: item.report,
    skills: { linhas: skill.skills.length, mapeadas: skillMapped },
    mundo: {
      npc_template: world.templates.length,
      npc_template_nome_coreano: world.templates.filter((t) => t.encodingOrigin === "cp949-as-latin1").length,
      npc_template_nome_recuperado: world.templates.filter((t) => t.recovered).length,
      npc_definition: world.definitions.length,
      npc_shop_item: world.shopItems.length,
      level_item: world.levelItems.length,
      guard_spawn: world.guards.length,
      guard_spawn_reward: world.guards.reduce((s, g) => s + g.blocks.length, 0),
      mob_merc: world.mobMercs.length,
      game_setting: world.settings.length,
      game_matrix: world.matrices.length,
      region: world.regions.length,
      world_item_seed: world.worldItems.length,
      guild_zone: world.guildZones.length,
    },
    sanitizacao: Object.fromEntries([...sanitizeStats.entries()].sort()),
    falhas: failures,
  };

  if (DRY || failures.length) {
    console.log(JSON.stringify(report, null, 2));
    if (failures.length) {
      console.error(`\n${failures.length} falha(s) de integridade — nada foi escrito.`);
      process.exit(1);
    }
    return;
  }

  mkdirSync(MIGRATIONS, { recursive: true });
  const files = [
    ["0017_seed_item_catalog.sql", emitItemSql(item)],
    ["0018_seed_skill_catalog.sql", emitSkillSql(skill)],
    ["0019_seed_mundo.sql", emitWorldSql(world)],
  ];
  for (const [name, sql] of files) {
    const path = resolve(MIGRATIONS, name);
    writeFileSync(path, `${sql}\n`);
    console.log(`${name}: ${(statSync(path).size / 1024).toFixed(0)} KiB`);
  }
  console.log(JSON.stringify(report, null, 2));
}

main();
