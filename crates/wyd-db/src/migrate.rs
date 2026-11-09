//! Runner de migrations no boot (ADR-0024).
//!
//! Gateway e canais aplicam o schema **antes** de abrir o pool; não existe job nem
//! container extra no compose. Regras:
//!
//! - lê `MIGRATIONS_DIR` (default `/migrations`, copiado nas imagens) e ordena `*.sql`;
//! - ledger `schema_migrations(version, checksum, applied_at)`: reexecutar não reaplica;
//! - **advisory lock de sessão**: gateway e canal subindo juntos não migram ao mesmo tempo;
//! - **uma transação por arquivo**: falha reverte só aquele arquivo;
//! - **checksum** do bloco `up`: arquivo já aplicado que mudou é erro, não skip;
//! - **baseline**: volume anterior ao ledger com schema completo é marcado como aplicado;
//! - ao fim exige `item_catalog` — health sem catálogo mentiria.
//!
//! DDL nunca roda no tick: quem chama é o boot do processo Cold Path.

use std::collections::BTreeSet;
use std::fmt;
use std::path::{Path, PathBuf};

use sha2::{Digest, Sha256};
use sqlx::postgres::PgPoolOptions;
use sqlx::{Connection, PgConnection};

/// Variável com o diretório da série.
pub const MIGRATIONS_DIR_ENV: &str = "MIGRATIONS_DIR";
/// Default dentro das imagens distroless (ver `docker/Dockerfile.*`).
pub const DEFAULT_MIGRATIONS_DIR: &str = "/migrations";
/// Tabela de controle: uma linha por arquivo aplicado.
const LEDGER: &str = "schema_migrations";
/// Chave do advisory lock: um migrador por banco, não um por processo.
const LOCK_KEY: i64 = 0x5759_4401;

/// Resultado de uma execução (puladas = já aplicadas com o mesmo checksum).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MigrationReport {
    pub applied: usize,
    pub skipped: usize,
    pub version: Option<String>,
    pub baseline: bool,
}

impl fmt::Display for MigrationReport {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "applied={} skipped={} version={} baseline={}",
            self.applied,
            self.skipped,
            self.version.as_deref().unwrap_or("-"),
            self.baseline
        )
    }
}

/// Um arquivo da série já parseado.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MigrationFile {
    pub version: String,
    pub checksum: String,
    pub up: String,
}

/// Erro próprio da fronteira de migrations (nenhuma variante carrega DSN).
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum MigrationError {
    /// Diretório ausente/ilegível.
    Dir(String),
    /// Nenhum `*.sql` no diretório.
    Empty,
    /// Arquivo sem `-- migrate:up` / `-- migrate:down`.
    MissingMarkers { file: String },
    /// Duas versões iguais na série.
    DuplicateVersion { version: String },
    /// Arquivo já aplicado com outro conteúdo.
    ChecksumMismatch { version: String },
    /// DSN fora do esquema `postgres://`.
    InvalidDsn,
    /// Falha do Postgres em um arquivo (ou no ledger).
    Sql { version: String, detail: String },
    /// Série aplicada não produziu o catálogo.
    MissingCatalog,
}

impl fmt::Display for MigrationError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Dir(detail) => write!(f, "série de migrations ilegível: {detail}"),
            Self::Empty => write!(f, "nenhum arquivo de migration no diretório"),
            Self::MissingMarkers { file } => {
                write!(f, "{file}: sem linha -- migrate:up / -- migrate:down")
            }
            Self::DuplicateVersion { version } => {
                write!(f, "versão {version} duplicada na série")
            }
            Self::ChecksumMismatch { version } => write!(
                f,
                "{version} já aplicada com outro conteúdo (checksum divergente): não reescreva migration aplicada"
            ),
            Self::InvalidDsn => write!(f, "DSN inválido (esperado esquema postgres://)"),
            Self::Sql { version, detail } => {
                write!(f, "migration {version} falhou: {detail}")
            }
            Self::MissingCatalog => {
                write!(f, "schema sem item_catalog depois de aplicar a série")
            }
        }
    }
}

impl std::error::Error for MigrationError {}

/// Diretório configurado por ambiente (a imagem define `/migrations`).
#[must_use]
pub fn dir_from_env() -> PathBuf {
    std::env::var(MIGRATIONS_DIR_ENV)
        .ok()
        .filter(|value| !value.trim().is_empty())
        .map(PathBuf::from)
        .unwrap_or_else(|| PathBuf::from(DEFAULT_MIGRATIONS_DIR))
}

/// Extrai o bloco `up`; `None` quando os marcadores não existem.
#[must_use]
pub fn up_block(sql: &str) -> Option<String> {
    let mut up = String::new();
    let mut inside = false;
    let mut seen_up = false;
    for line in sql.lines() {
        let marker = line.trim_end();
        if marker == "-- migrate:up" {
            inside = true;
            seen_up = true;
            continue;
        }
        if marker == "-- migrate:down" {
            inside = false;
            continue;
        }
        if inside {
            up.push_str(line);
            up.push('\n');
        }
    }
    if seen_up && sql.lines().any(|line| line.trim_end() == "-- migrate:down") {
        Some(up)
    } else {
        None
    }
}

/// sha256 do bloco `up` em hexadecimal (detecta reescrita de migration aplicada).
#[must_use]
pub fn checksum(up: &str) -> String {
    let digest = Sha256::digest(up.as_bytes());
    let mut out = String::with_capacity(digest.len() * 2);
    for byte in digest {
        out.push_str(&format!("{byte:02x}"));
    }
    out
}

/// Lê e valida a série do diretório (ordem de aplicação = ordem dos nomes).
pub fn load(dir: &Path) -> Result<Vec<MigrationFile>, MigrationError> {
    let entries = std::fs::read_dir(dir)
        .map_err(|error| MigrationError::Dir(format!("{}: {error}", dir.display())))?;
    let mut paths: Vec<PathBuf> = entries
        .filter_map(Result::ok)
        .map(|entry| entry.path())
        .filter(|path| path.extension().is_some_and(|ext| ext == "sql"))
        .collect();
    paths.sort();
    if paths.is_empty() {
        return Err(MigrationError::Empty);
    }

    let mut files = Vec::with_capacity(paths.len());
    let mut versions = BTreeSet::new();
    for path in paths {
        let file = path
            .file_name()
            .map(|name| name.to_string_lossy().to_string())
            .unwrap_or_default();
        let sql = std::fs::read_to_string(&path)
            .map_err(|error| MigrationError::Dir(format!("{file}: {error}")))?;
        let up =
            up_block(&sql).ok_or_else(|| MigrationError::MissingMarkers { file: file.clone() })?;
        let version = file.split('_').next().unwrap_or_default().to_string();
        if version.is_empty() {
            return Err(MigrationError::MissingMarkers { file });
        }
        if !versions.insert(version.clone()) {
            return Err(MigrationError::DuplicateVersion { version });
        }
        files.push(MigrationFile {
            version,
            checksum: checksum(&up),
            up,
        });
    }
    Ok(files)
}

/// Aplica a série do diretório padrão (`MIGRATIONS_DIR`).
pub async fn run(dsn: &str) -> Result<MigrationReport, MigrationError> {
    run_dir(dsn, &dir_from_env(), false).await
}

/// Aplica a série do diretório informado; `reset` recria o schema `public`.
pub async fn run_dir(
    dsn: &str,
    dir: &Path,
    reset: bool,
) -> Result<MigrationReport, MigrationError> {
    if !crate::is_postgres_dsn(dsn) {
        return Err(MigrationError::InvalidDsn);
    }
    let files = load(dir)?;

    let pool = PgPoolOptions::new()
        .max_connections(1)
        .connect(dsn)
        .await
        .map_err(|error| sql_error("-", error))?;
    let mut conn = pool
        .acquire()
        .await
        .map_err(|error| sql_error("-", error))?;

    // Exclusão mútua entre gateway e canal: o segundo espera o primeiro terminar.
    sqlx::query("SELECT pg_advisory_lock($1)")
        .bind(LOCK_KEY)
        .execute(&mut *conn)
        .await
        .map_err(|error| sql_error("-", error))?;

    if reset {
        exec(
            &mut conn,
            "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;",
            "reset",
        )
        .await?;
    }

    exec(
        &mut conn,
        &format!(
            "CREATE TABLE IF NOT EXISTS {LEDGER} (
                version    TEXT PRIMARY KEY,
                checksum   TEXT NOT NULL,
                applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
            )"
        ),
        "ledger",
    )
    .await?;

    // Baseline de volume anterior ao ledger (schema já completo, sem histórico).
    let mut baseline = false;
    if !reset && ledger_count(&mut conn).await? == 0 && exists(&mut conn, "public.accounts").await?
    {
        baseline = true;
        for file in &files {
            sqlx::query(&format!(
                "INSERT INTO {LEDGER} (version, checksum) VALUES ($1, $2) ON CONFLICT DO NOTHING"
            ))
            .bind(&file.version)
            .bind(&file.checksum)
            .execute(&mut *conn)
            .await
            .map_err(|error| sql_error(&file.version, error))?;
        }
    }

    let mut applied = 0usize;
    let mut skipped = 0usize;
    for file in &files {
        let existing: Option<String> =
            sqlx::query_scalar(&format!("SELECT checksum FROM {LEDGER} WHERE version = $1"))
                .bind(&file.version)
                .fetch_optional(&mut *conn)
                .await
                .map_err(|error| sql_error(&file.version, error))?;
        if let Some(recorded) = existing {
            if recorded != file.checksum {
                return Err(MigrationError::ChecksumMismatch {
                    version: file.version.clone(),
                });
            }
            skipped += 1;
            continue;
        }

        let mut tx = conn
            .begin()
            .await
            .map_err(|error| sql_error(&file.version, error))?;
        sqlx::raw_sql(&file.up)
            .execute(&mut *tx)
            .await
            .map_err(|error| sql_error(&file.version, error))?;
        sqlx::query(&format!(
            "INSERT INTO {LEDGER} (version, checksum) VALUES ($1, $2)"
        ))
        .bind(&file.version)
        .bind(&file.checksum)
        .execute(&mut *tx)
        .await
        .map_err(|error| sql_error(&file.version, error))?;
        tx.commit()
            .await
            .map_err(|error| sql_error(&file.version, error))?;
        applied += 1;
    }

    if !exists(&mut conn, "public.item_catalog").await? {
        return Err(MigrationError::MissingCatalog);
    }
    let version: Option<String> = sqlx::query_scalar(&format!("SELECT max(version) FROM {LEDGER}"))
        .fetch_one(&mut *conn)
        .await
        .map_err(|error| sql_error("-", error))?;

    let _ = sqlx::query("SELECT pg_advisory_unlock($1)")
        .bind(LOCK_KEY)
        .execute(&mut *conn)
        .await;

    Ok(MigrationReport {
        applied,
        skipped,
        version,
        baseline,
    })
}

async fn exec(conn: &mut PgConnection, sql: &str, version: &str) -> Result<(), MigrationError> {
    sqlx::raw_sql(sql)
        .execute(conn)
        .await
        .map_err(|error| sql_error(version, error))?;
    Ok(())
}

async fn ledger_count(conn: &mut PgConnection) -> Result<i64, MigrationError> {
    sqlx::query_scalar(&format!("SELECT count(*) FROM {LEDGER}"))
        .fetch_one(conn)
        .await
        .map_err(|error| sql_error("-", error))
}

async fn exists(conn: &mut PgConnection, regclass: &str) -> Result<bool, MigrationError> {
    let found: Option<String> = sqlx::query_scalar("SELECT to_regclass($1)::text")
        .bind(regclass)
        .fetch_one(conn)
        .await
        .map_err(|error| sql_error("-", error))?;
    Ok(found.is_some())
}

fn sql_error(version: &str, error: sqlx::Error) -> MigrationError {
    MigrationError::Sql {
        version: version.to_string(),
        detail: error.to_string(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const SAMPLE: &str =
        "-- 0001_teste\n-- migrate:up\nCREATE TABLE a (id INT);\n-- migrate:down\nDROP TABLE a;\n";

    #[test]
    fn up_block_ignora_o_down() {
        let up = up_block(SAMPLE).expect("marcadores presentes");
        assert!(up.contains("CREATE TABLE a"));
        assert!(!up.contains("DROP TABLE"));
    }

    #[test]
    fn sem_marcadores_e_erro() {
        assert_eq!(up_block("CREATE TABLE a (id INT);"), None);
        assert_eq!(up_block("-- migrate:up\nCREATE TABLE a (id INT);"), None);
    }

    #[test]
    fn checksum_muda_quando_o_up_muda() {
        let base = up_block(SAMPLE).expect("marcadores");
        let mut alterado = base.clone();
        alterado.push_str("CREATE TABLE b (id INT);\n");
        assert_ne!(checksum(&base), checksum(&alterado));
        assert_eq!(checksum(&base).len(), 64);
    }
}
