//! Rastreabilidade migration ↔ modelo sem precisar de banco vivo.
//!
//! Valida estaticamente: marcadores up/down, reversibilidade (toda tabela
//! criada no up cai no down), FKs apontando para tabelas criadas e seeds só
//! tocando tabelas existentes e tabela criada duas vezes. A aplicação contra
//! Postgres real (up → down → up, com contagens) é `make migrations-apply`
//! (`tools/content/migrate-cycle.mjs`); este teste garante a mesma disciplina
//! no `cargo test`, sem depender de banco vivo.

use std::collections::{BTreeMap, BTreeSet};
use std::path::PathBuf;

fn db_dir() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
}

fn migration_files() -> Vec<PathBuf> {
    let mut files: Vec<PathBuf> = std::fs::read_dir(db_dir().join("migrations"))
        .expect("diretório de migrations legível")
        .map(|e| e.expect("entrada legível").path())
        .filter(|p| p.extension().is_some_and(|x| x == "sql"))
        .collect();
    files.sort();
    assert!(!files.is_empty(), "há migrations versionadas");
    files
}

fn split_up_down(sql: &str, name: &str) -> (String, String) {
    // Âncora no fim de linha: o cabeçalho do arquivo MENCIONA os marcadores;
    // só a linha própria vale ("-- migrate:up\n").
    let up = sql
        .find("-- migrate:up\n")
        .unwrap_or_else(|| panic!("{name} tem marcador up em linha própria"));
    let down = sql
        .find("-- migrate:down\n")
        .unwrap_or_else(|| panic!("{name} tem marcador down em linha própria"));
    assert!(up < down, "{name}: up vem antes do down");
    (sql[up..down].to_string(), sql[down..].to_string())
}

/// `CREATE TABLE x` / `DROP TABLE x` (forma usada nas migrations do repo).
fn tables_created(sql: &str) -> BTreeSet<String> {
    sql.lines()
        .filter_map(|l| {
            let t = l.trim();
            t.strip_prefix("CREATE TABLE")
                .map(|r| r.split_whitespace().next().unwrap_or("").to_string())
        })
        .collect()
}

fn tables_dropped(sql: &str) -> BTreeSet<String> {
    sql.lines()
        .filter_map(|l| {
            let t = l.trim();
            t.strip_prefix("DROP TABLE")
                .map(|r| r.trim().split(';').next().unwrap_or("").trim().to_string())
        })
        .collect()
}

/// Alvos de `REFERENCES tabela (` — um por linha nas migrations do repo.
fn fk_targets(sql: &str) -> Vec<String> {
    sql.lines()
        .filter_map(|l| {
            let t = l.trim().to_ascii_lowercase();
            t.find("references").map(|i| {
                t[i + "references".len()..]
                    .trim()
                    .split([' ', '('])
                    .next()
                    .unwrap_or("")
                    .to_string()
            })
        })
        .collect()
}

#[test]
fn migrations_are_numbered_reversible_and_fk_consistent() {
    let files = migration_files();
    let mut known: BTreeSet<String> = BTreeSet::new();
    let mut prev = 0u32;
    let mut downs: BTreeMap<String, BTreeSet<String>> = BTreeMap::new();

    for path in &files {
        let name = path
            .file_name()
            .expect("nome")
            .to_string_lossy()
            .to_string();
        let num: u32 = name[..4].parse().expect("NNNN numérico");
        assert!(num > prev, "migrations em ordem crescente: {name}");
        prev = num;

        let sql = std::fs::read_to_string(path).expect("migration legível");
        let (up, down) = split_up_down(&sql, &name);
        let created = tables_created(&up);
        // DDL cria tabela ou evolui uma existente (`ALTER TABLE`); seed (`NNNN_seed_*`) só
        // insere, então não pode ser exigido dele.
        if !name.contains("_seed_") {
            assert!(
                !created.is_empty() || up.contains("ALTER TABLE"),
                "{name} cria ou altera ao menos uma tabela"
            );
        }
        for t in tables_dropped(&down) {
            downs.entry(t).or_default().insert(name.clone());
        }
        for target in fk_targets(&up) {
            assert!(
                known.contains(&target) || created.contains(&target),
                "{name}: FK para tabela desconhecida `{target}`"
            );
        }
        // CHECKs de domínio: SÓ a tabela de contas enumera o papel de acesso. Buscar a
        // substring `role` reprovava 0007 por causa de `skill_role` e de `game_class.role`
        // (texto livre, sem domínio) — nenhum dos dois é papel de acesso.
        if created.contains("accounts") {
            assert!(
                up.contains("IN ('player', 'moderator', 'admin')"),
                "{name}: CHECK de role"
            );
        }
        for table in &created {
            assert!(
                known.insert(table.clone()),
                "{name}: tabela `{table}` criada duas vezes"
            );
        }
    }

    // Reversibilidade total: toda tabela criada cai em algum down.
    for t in &known {
        assert!(
            downs.contains_key(t),
            "tabela `{t}` sem DROP em nenhum down"
        );
    }
    // O modelo cresceu por domínio (24 migrations); o que não pode sumir é o núcleo de jogo.
    for core in [
        "accounts",
        "characters",
        "character_items",
        "vault_items",
        "guilds",
        "guild_members",
        "audit_log",
        "delivery_queue",
    ] {
        assert!(known.contains(core), "modelo mantém a tabela core `{core}`");
    }
}

#[test]
fn dev_seed_touches_only_known_tables_without_secrets() {
    let files = migration_files();
    let mut known = BTreeSet::new();
    for path in &files {
        let sql = std::fs::read_to_string(path).expect("migration legível");
        let (up, _) = split_up_down(&sql, "seed-check");
        known.extend(tables_created(&up));
    }
    let seed = std::fs::read_to_string(db_dir().join("seeds/dev.sql")).expect("seed legível");
    for line in seed.lines() {
        let t = line.trim();
        if let Some(rest) = t.strip_prefix("INSERT INTO ") {
            let table = rest.split_whitespace().next().unwrap_or("");
            assert!(
                known.contains(table),
                "seed insere em tabela conhecida: {table}"
            );
        }
    }
    let lower = seed.to_lowercase();
    assert!(!lower.contains("postgres://"), "seed sem DSN");
    assert!(
        seed.contains("DEV-ONLY") || seed.contains("dev"),
        "seed marcado como desenvolvimento"
    );
}
