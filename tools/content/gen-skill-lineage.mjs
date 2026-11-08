#!/usr/bin/env node
/**
 * Gera a migração que fecha `skill_catalog.class_code` / `lineage_code` para as skills
 * cuja linhagem está **publicada em fonte oficial**.
 *
 * Regra desta ferramenta: nada entra por dedução. Cada linha da tabela `curated` precisa
 * citar (a) o nome exato da skill no seed e (b) a fonte que a coloca naquela linhagem.
 * Nome que não existir no seed é reportado e ignorado — jamais "corrigido" por aproximação.
 *
 * Três nomes publicados pela Kersef não existiam no seed do cliente (`Roubar_Dinheiro`,
 * `Evocar_Lobo_Selvagem`, `Metamorfose_Superior`). Eles deixaram de ser "sem correspondência":
 * viraram linhas de catálogo em `0024_seed_skill_catalog_kersef.sql`, com índice provisório e
 * nota de pendência, porque criar linha é mais honesto do que fingir que a skill não existe.
 *
 * Uso: node tools/content/gen-skill-lineage.mjs
 * Saída: crates/wyd-db/migrations/0021_seed_skill_lineage.sql
 */
import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "..", "..");
const SEED = path.join(ROOT, "crates", "wyd-db", "migrations", "0019_seed_skill_catalog.sql");
const OUT = path.join(ROOT, "crates", "wyd-db", "migrations", "0021_seed_skill_lineage.sql");

const KERS = "https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/";

/**
 * Fonte de cada bloco:
 * - `kersef:9th`   → "Nona Skill (9th)", lista por classe e linhagem
 * - `kersef:10th`  → "10th Skill", lista por classe e linhagem
 * - `kersef:12th`  → tabela de 12ª por linhagem em `docs/wyd/classes.md` §2
 * - `rule:name_confirmed` → já mapeado no seed (mantido, não reescrevemos)
 *
 * `role: null` é permitido: há skill cuja **linhagem** está confirmada e cujo efeito não está
 * descrito na fonte (ex. `Freezing`). Preencher `skill_role` por dedução seria inventar dado.
 */
const curated = [
  // --- 9ª skill: uma por linhagem, publicada pela Kersef ---
  { name: "Anti_Magia", class: "beastmaster", lineage: "elemental", role: "buff", source: "kersef:9th", confidence: 95 },
  { name: "Unidade_Mental", class: "beastmaster", lineage: "evocation", role: "passive", source: "kersef:9th", confidence: 95 },
  { name: "Concha_Resistente", class: "beastmaster", lineage: "nature", role: "passive", source: "kersef:9th", confidence: 95 },
  { name: "Proteção_Divina", class: "transknight", lineage: "faith", role: "buff", source: "kersef:9th", confidence: 95 },
  { name: "Mãos_Sangrentas", class: "transknight", lineage: "trans", role: "passive", source: "kersef:9th", confidence: 95 },
  { name: "Contra_Ataque", class: "huntress", lineage: "survival", role: "passive", source: "kersef:9th", confidence: 90 },
  { name: "Ponto_do_Mestre", class: "huntress", lineage: "trade", role: "passive", source: "kersef:9th", confidence: 95 },
  { name: "Cenote", class: "foema", lineage: "white_magic", role: "passive", source: "kersef:9th", confidence: 95 },
  { name: "Magia_Misteriosa", class: "foema", lineage: "black_magic", role: "buff", source: "kersef:9th", confidence: 95 },

  // --- 10ª skill: exige 4000 de fama, uma por linhagem ---
  { name: "Proteção_Absoluta", class: "foema", lineage: "white_magic", role: "buff", source: "kersef:10th", confidence: 95 },
  { name: "Congelamento_Proficiente", class: "foema", lineage: "black_magic", role: "passive", source: "kersef:10th", confidence: 95 },
  { name: "Incapacitador", class: "foema", lineage: "special_magic", role: "debuff", source: "kersef:10th", confidence: 90 },
  { name: "Bênção_Divina", class: "transknight", lineage: "faith", role: "passive", source: "kersef:10th", confidence: 95 },
  { name: "Mestre_das_Armas", class: "transknight", lineage: "trans", role: "passive", source: "kersef:10th", confidence: 90 },
  { name: "Conexão_de_Gelo", class: "transknight", lineage: "magic_sword", role: "passive", source: "kersef:10th", confidence: 95 },
  { name: "Chama_Resistente", class: "beastmaster", lineage: "elemental", role: "buff", source: "kersef:10th", confidence: 95 },
  { name: "Invocação_Final", class: "beastmaster", lineage: "evocation", role: "summon", source: "kersef:10th", confidence: 90 },
  { name: "Espinhos_Fortalecidos", class: "beastmaster", lineage: "nature", role: "passive", source: "kersef:10th", confidence: 90 },
  { name: "Ataque_Rápido_Proficiente", class: "huntress", lineage: "survival", role: "passive", source: "kersef:10th", confidence: 95 },

  // --- listas oficiais por linhagem (páginas de classe da Kersef / WYD International) ---
  // Aqui o nome EN da fonte foi conferido contra o nome PT do seed um a um; quando a
  // correspondência não é literal, a linha NÃO entra (confiança menor não autoriza chute).
  { name: "Cura", class: "foema", lineage: "white_magic", role: "heal", source: "kersef:class-foema", confidence: 80 },
  { name: "Recuperar", class: "foema", lineage: "white_magic", role: "heal", source: "kersef:class-foema", confidence: 80 },
  { name: "Renascimento", class: "foema", lineage: "white_magic", role: "heal", source: "kersef:class-foema", confidence: 80 },
  { name: "Escudo_Mágico", class: "foema", lineage: "white_magic", role: "buff", source: "kersef:class-foema", confidence: 75 },
  { name: "Ataque_de_Fogo", class: "foema", lineage: "black_magic", role: "damage", source: "kersef:class-foema", confidence: 85 },
  { name: "Relâmpago", class: "foema", lineage: "black_magic", role: "damage", source: "kersef:class-foema", confidence: 85 },
  { name: "Lança_de_Gelo", class: "foema", lineage: "black_magic", role: "damage", source: "kersef:class-foema", confidence: 85 },
  { name: "Nevasca", class: "foema", lineage: "black_magic", role: "damage", source: "kersef:class-foema", confidence: 80 },
  { name: "Inferno", class: "foema", lineage: "black_magic", role: "damage", source: "kersef:class-foema", confidence: 80 },
  { name: "Cancelamento", class: "foema", lineage: "special_magic", role: "debuff", source: "kersef:class-foema", confidence: 85 },
  { name: "Controle_de_Mana", class: "foema", lineage: "special_magic", role: "debuff", source: "kersef:class-foema", confidence: 80 },
  { name: "Remover_Memória", class: "foema", lineage: "special_magic", role: "debuff", source: "wyd-international:9-12", confidence: 85 },
  { name: "Destino", class: "transknight", lineage: "faith", role: "damage", source: "kersef:class-transknight", confidence: 85 },
  { name: "Toque_Sagrado", class: "transknight", lineage: "faith", role: "debuff", source: "kersef:class-transknight", confidence: 80 },
  { name: "Carga", class: "transknight", lineage: "trans", role: "damage", source: "kersef:class-transknight", confidence: 85 },
  { name: "Assalto", class: "transknight", lineage: "trans", role: "damage", source: "kersef:class-transknight", confidence: 85 },
  { name: "Golpe_Mortal", class: "transknight", lineage: "trans", role: "damage", source: "kersef:class-transknight", confidence: 80 },
  { name: "Espada_da_Fênix", class: "transknight", lineage: "magic_sword", role: "damage", source: "kersef:class-transknight", confidence: 75 },
  { name: "Tempestade_de_Gelo", class: "transknight", lineage: "magic_sword", role: "damage", source: "wyd-international:9-12", confidence: 85 },
  { name: "Flecha_Mágica", class: "huntress", lineage: "survival", role: "damage", source: "kersef:class-huntress", confidence: 80 },
  { name: "Garra_Habilidosa", class: "huntress", lineage: "survival", role: "damage", source: "kersef:class-huntress", confidence: 80 },
  { name: "Alquimia", class: "huntress", lineage: "trade", role: "craft", source: "kersef:class-huntress", confidence: 80 },
  { name: "Extração", class: "huntress", lineage: "trade", role: "craft", source: "kersef:class-huntress", confidence: 75 },
  { name: "Golpe_Felino", class: "huntress", lineage: "capture", role: "damage", source: "kersef:class-huntress", confidence: 75 },
  { name: "Ligação_Espectral", class: "huntress", lineage: "capture", role: "debuff", source: "kersef:class-huntress", confidence: 75 },
  { name: "Fera_Flamejante", class: "beastmaster", lineage: "elemental", role: "summon", source: "kersef:class-beastmaster", confidence: 75 },
  { name: "Proteção_Elemental", class: "beastmaster", lineage: "elemental", role: "buff", source: "kersef:class-beastmaster", confidence: 75 },
  { name: "Evocar_Condor", class: "beastmaster", lineage: "evocation", role: "summon", source: "kersef:class-beastmaster", confidence: 80 },
  { name: "Evocar_Urso_Selvagem", class: "beastmaster", lineage: "evocation", role: "summon", source: "kersef:class-beastmaster", confidence: 80 },
  { name: "Evocar_Grande_Tigre", class: "beastmaster", lineage: "evocation", role: "summon", source: "kersef:class-beastmaster", confidence: 80 },
  { name: "Evocar_Gorila_Gigante", class: "beastmaster", lineage: "evocation", role: "summon", source: "kersef:class-beastmaster", confidence: 80 },
  { name: "Evocar_Dragão_Negro", class: "beastmaster", lineage: "evocation", role: "summon", source: "kersef:class-beastmaster", confidence: 80 },
  { name: "Evocar_Succubus", class: "beastmaster", lineage: "evocation", role: "summon", source: "kersef:class-beastmaster", confidence: 80 },

  // --- 9ª que faltava: Espada Mágica (docs/wyd/classes.md §2, tabela do TransKnight) ---
  { name: "Espelho_Mágico", class: "transknight", lineage: "magic_sword", role: "passive", source: "kersef:9th", confidence: 90 },

  // --- 12ª skill por linhagem (docs/wyd/classes.md §2: "9ª/10ª/11ª/12ª — CONFIRMADO") ---
  // A 11ª não entra: nenhuma fonte publica a lista de 11ª por linhagem (só a regra de
  // nível 281+/1.000 fama/livro), então não há nome para casar com o seed.
  { name: "Clod_Attack", class: "transknight", lineage: "faith", role: "passive", source: "kersef:12th", confidence: 90 },
  { name: "WindStorm", class: "transknight", lineage: "trans", role: "passive", source: "kersef:12th", confidence: 90 },
  { name: "Seal_Master", class: "transknight", lineage: "magic_sword", role: "passive", source: "kersef:12th", confidence: 90 },
  { name: "Holy_Power", class: "foema", lineage: "white_magic", role: "passive", source: "kersef:12th", confidence: 90 },
  { name: "Thunder_Lising", class: "foema", lineage: "black_magic", role: "passive", source: "kersef:12th", confidence: 90 },
  { name: "Another_Change", class: "foema", lineage: "special_magic", role: "buff", source: "kersef:12th", confidence: 85 },
  { name: "Manaburn_Master", class: "beastmaster", lineage: "elemental", role: "passive", source: "kersef:12th", confidence: 90 },
  { name: "Ice_Armor", class: "beastmaster", lineage: "evocation", role: "passive", source: "kersef:12th", confidence: 90 },
  // Nome e linhagem confirmados; efeito não descrito na fonte → sem `role` (não inventar).
  { name: "Freezing", class: "huntress", lineage: "survival", role: null, source: "kersef:12th", confidence: 85 },
  { name: "Reinforce_Soul", class: "huntress", lineage: "trade", role: null, source: "kersef:12th", confidence: 85 },
  // Natureza (BeastMaster) e Captura (Huntress) estão marcadas "não confirmada" na fonte.
];

const seed = readFileSync(SEED, "utf8");

/** Divide uma tupla de `VALUES` por vírgula respeitando aspas simples (`'a, b'`). */
function campos(texto) {
  const saida = [];
  let atual = "";
  let dentro = false;
  for (const ch of texto) {
    if (ch === "'") {
      dentro = !dentro;
      atual += ch;
    } else if (ch === "," && !dentro) {
      saida.push(atual.trim());
      atual = "";
    } else {
      atual += ch;
    }
  }
  saida.push(atual.trim());
  return saida;
}

const semAspas = (valor) => {
  if (valor === "NULL") return null;
  return valor.startsWith("'") ? valor.slice(1, -1).replace(/''/g, "'") : valor;
};

/**
 * Estado **anterior** de cada linha do seed (0019), lido da própria migração.
 *
 * É dele que sai o `down` seletivo: reverter a 0021 tem de devolver a classificação que a 0019
 * publicou, não apagá-la. As colunas finais de cada tupla são, nesta ordem, `class_code`,
 * `lineage_code`, `primary_system`, `skill_role`, `classification_source`, `confidence`.
 */
const anterior = new Map();
for (const linha of seed.split("\n")) {
  const texto = linha.trim();
  if (!texto.startsWith("(")) continue;
  const c = campos(texto.replace(/\),?$/, "").slice(1));
  const index = Number(c[0]);
  if (!Number.isInteger(index)) continue;
  const [classe, linhagem, sistema, papel, fonte, confianca] = c.slice(-6).map(semAspas);
  anterior.set(index, {
    name: semAspas(c[1]),
    class: classe,
    lineage: linhagem,
    system: sistema,
    role: papel,
    source: fonte,
    confidence: confianca,
  });
}

const rows = [...anterior].map(([index, row]) => ({ index, name: row.name }));
const byName = new Map(rows.map((r) => [r.name, r]));

const applied = [];
const missing = [];
const duplicates = [];
const byIndex = new Map();
for (const entry of curated) {
  const row = byName.get(entry.name);
  if (!row) {
    missing.push(entry.name);
    continue;
  }
  // Dois nomes podem resolver para o mesmo registro do seed (sinônimo/erro de curadoria).
  // Emitir as duas linhas faria o UPDATE aplicar a última em silêncio: aqui a duplicata é
  // reportada e a primeira vence, para o número de cobertura continuar verdadeiro.
  const previous = byIndex.get(row.index);
  if (previous) {
    duplicates.push(`${entry.name} → skill_index ${row.index} (já usado por ${previous})`);
    continue;
  }
  byIndex.set(row.index, entry.name);
  applied.push({ ...entry, index: row.index, anterior: anterior.get(row.index) });
}

const esc = (value) => value.replace(/'/g, "''");
const roleSql = (role) => (role === null ? "NULL" : `'${esc(role)}'`);
const values = applied
  .map(
    (a) =>
      `    (${a.index}, '${esc(a.class)}', '${esc(a.lineage)}', ${roleSql(a.role)}, '${esc(a.source)}', ${a.confidence})`,
  )
  .join(",\n");

// Linhas que a 0019 já classificava: o `down` da 0021 tem de restaurar exatamente estas.
const existing = new Set(
  [...anterior]
    .filter(([, row]) => row.class !== null)
    .map(([index]) => index),
);
// Cobertura é a UNIÃO (seed já mapeado ∪ curadoria), não a soma das duas: linha curada que
// repete o seed não aumenta o total. Era essa soma que anunciava 61 com 60 no banco.
const repeated = applied.filter((a) => existing.has(a.index));
const fresh = applied.length - repeated.length;
const mapped = new Set([...existing, ...applied.map((a) => a.index)]);
const stillUnmapped = rows.length - mapped.size;

const restauradas = applied.filter((a) => a.anterior.class !== null);
const novas = applied.filter((a) => a.anterior.class === null);
const valorSql = (valor) => (valor === null ? "NULL" : `'${esc(valor)}'`);

// `down` seletivo: cada linha volta ao estado que a 0019 publicou. As colunas restauradas são
// exatamente as que o `up` escreve — `primary_system` nunca é tocada por esta migração.
const restaurarValues = restauradas
  .map(
    (a) =>
      `    (${a.index}, '${esc(a.anterior.class)}', '${esc(a.anterior.lineage)}', ${valorSql(a.anterior.role)}, ${valorSql(a.anterior.source)}, ${a.anterior.confidence})`,
  )
  .join(",\n");
const restaurarSql =
  restauradas.length === 0
    ? ""
    : `UPDATE skill_catalog AS s
SET class_code = v.class_code,
    lineage_code = v.lineage_code,
    skill_role = v.skill_role,
    classification_source = v.source,
    confidence = v.confidence,
    updated_at = now()
FROM (VALUES
${restaurarValues}
) AS v (skill_index, class_code, lineage_code, skill_role, source, confidence)
WHERE s.skill_index = v.skill_index;

`;

const sql = `-- 0021_seed_skill_lineage: classe e linhagem das skills com fonte oficial (docs/wyd/decisoes-servidor.md §6).
-- GERADO por tools/content/gen-skill-lineage.mjs — NÃO EDITAR À MÃO.
-- Só entram skills cuja linhagem está publicada (Kersef 9ª/10ª e a tabela de 12ª de
-- docs/wyd/classes.md §2). O resto continua
-- NULL com classification_source = 'unmapped': ausência de fonte não vira chute.
-- Cobertura: ${mapped.size} de ${rows.length} skills (${existing.size} já mapeadas no seed +
-- ${fresh} novas nesta migração; ${repeated.length} linhas curadas repetiam o seed).
-- Down seletivo: as ${restauradas.length} linhas que a 0019 já classificava voltam ao estado da
-- 0019 (valores restaurados na própria migração, lidos do seed pelo gerador); as ${novas.length}
-- classificadas pela primeira vez aqui voltam a 'unmapped'. Nada é apagado, então reverter só
-- esta migração é íntegro e o ciclo up→down→up reproduz o mesmo estado.

-- migrate:up
UPDATE skill_catalog AS s
SET class_code = v.class_code,
    lineage_code = v.lineage_code,
    skill_role = v.skill_role,
    classification_source = v.source,
    confidence = v.confidence,
    updated_at = now()
FROM (VALUES
${values}
) AS v (skill_index, class_code, lineage_code, skill_role, source, confidence)
WHERE s.skill_index = v.skill_index;

-- migrate:down
${restaurarSql}UPDATE skill_catalog AS s
SET class_code = NULL,
    lineage_code = NULL,
    skill_role = NULL,
    classification_source = 'unmapped',
    confidence = NULL,
    updated_at = now()
WHERE s.skill_index IN (${novas.map((a) => a.index).join(", ") || "NULL"});
`;

writeFileSync(OUT, sql);
console.log(
  `gen-skill-lineage: ${applied.length} skills mapeadas → ${path.relative(ROOT, OUT)}`,
);
if (missing.length > 0) {
  console.log(`  sem correspondência no seed (ignoradas): ${missing.join(", ")}`);
}
if (duplicates.length > 0) {
  console.log(`  DUPLICATAS (segunda descartada): ${duplicates.join("; ")}`);
}
if (repeated.length > 0) {
  console.log(
    `  já mapeadas (repetidas, sem efeito): ${repeated.map((a) => a.name).join(", ")}`,
  );
}
console.log(
  `  cobertura: ${mapped.size} de ${rows.length} (${existing.size} já + ${fresh} novas) | continuam unmapped: ${stillUnmapped}`,
);
