//! Manifesto de integridade (`docs/cliente/customizacao.md` §2).
//!
//! Formato: UTF-8, uma entrada por linha, `#` = comentário, caminhos relativos
//! com `/` (sem espaços — nenhum arquivo do cliente os usa).

use std::fmt;
use std::fs;
use std::io;
use std::path::{Component, Path, PathBuf};

use crate::sha256;

/// Cabeçalho esperado na primeira linha de comentário (informativo, não validado).
pub const FORMAT_V1: &str = "wyd-launcher-manifest v1";

/// Uma entrada do manifesto: hash + tamanho + caminho relativo.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Entry {
    pub sha256: [u8; 32],
    pub size: u64,
    pub path: String,
}

/// Manifesto analisado: build + protocolo + arquivos esperados.
#[derive(Debug, Clone)]
pub struct Manifest {
    pub build: u32,
    pub protocol: u32,
    pub files: Vec<Entry>,
}

/// Erros tipados de manifesto (caminhos ruins são o caso normal, não exceção).
#[derive(Debug)]
pub enum ManifestError {
    Io(io::Error),
    BadLine { line: usize, reason: &'static str },
    MissingField(&'static str),
    DuplicateField(&'static str),
    DuplicatePath(String),
    UnsafePath(String),
}

impl fmt::Display for ManifestError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Io(e) => write!(f, "erro de E/S: {e}"),
            Self::BadLine { line, reason } => write!(f, "linha {line} inválida: {reason}"),
            Self::MissingField(k) => write!(f, "campo obrigatório ausente: {k}"),
            Self::DuplicateField(k) => write!(f, "campo duplicado: {k}"),
            Self::DuplicatePath(p) => write!(f, "caminho duplicado: {p}"),
            Self::UnsafePath(p) => write!(f, "caminho inseguro (absoluto ou com `..`): {p}"),
        }
    }
}

impl std::error::Error for ManifestError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Io(e) => Some(e),
            _ => None,
        }
    }
}

/// Junta `rel` a `base`, recusando absoluto, vazio, `..` e componentes especiais.
pub(crate) fn join_inside(base: &Path, rel: &str) -> Result<PathBuf, ManifestError> {
    let p = Path::new(rel);
    let normal_only = !rel.is_empty() && p.components().all(|c| matches!(c, Component::Normal(_)));
    if !normal_only {
        return Err(ManifestError::UnsafePath(rel.to_string()));
    }
    Ok(base.join(p))
}

/// Analisa o texto do manifesto (erro sempre com número da linha).
pub fn parse(text: &str) -> Result<Manifest, ManifestError> {
    let mut build: Option<u32> = None;
    let mut protocol: Option<u32> = None;
    let mut files: Vec<Entry> = Vec::new();

    for (idx, raw) in text.lines().enumerate() {
        let line_no = idx + 1;
        let line = raw.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        if let Some(rest) = line.strip_prefix("build ") {
            if build.is_some() {
                return Err(ManifestError::DuplicateField("build"));
            }
            build = Some(rest.trim().parse().map_err(|_| ManifestError::BadLine {
                line: line_no,
                reason: "`build` não é u32",
            })?);
            continue;
        }
        if let Some(rest) = line.strip_prefix("protocol ") {
            if protocol.is_some() {
                return Err(ManifestError::DuplicateField("protocol"));
            }
            protocol = Some(rest.trim().parse().map_err(|_| ManifestError::BadLine {
                line: line_no,
                reason: "`protocol` não é u32",
            })?);
            continue;
        }
        let mut parts = line.split_whitespace();
        let (hex, size_s, path) = match (parts.next(), parts.next(), parts.next(), parts.next()) {
            (Some(h), Some(s), Some(p), None) => (h, s, p),
            _ => {
                return Err(ManifestError::BadLine {
                    line: line_no,
                    reason: "esperado `<sha256hex> <tamanho> <caminho>`",
                });
            }
        };
        let sha256 = sha256::parse_hex(hex).ok_or(ManifestError::BadLine {
            line: line_no,
            reason: "sha256 deve ter 64 hexadecimais",
        })?;
        let size: u64 = size_s.parse().map_err(|_| ManifestError::BadLine {
            line: line_no,
            reason: "tamanho não é u64",
        })?;
        join_inside(Path::new("."), path).map_err(|_| ManifestError::BadLine {
            line: line_no,
            reason: "caminho inseguro",
        })?;
        if files.iter().any(|e: &Entry| e.path == path) {
            return Err(ManifestError::DuplicatePath(path.to_string()));
        }
        files.push(Entry {
            sha256,
            size,
            path: path.to_string(),
        });
    }

    Ok(Manifest {
        build: build.ok_or(ManifestError::MissingField("build"))?,
        protocol: protocol.ok_or(ManifestError::MissingField("protocol"))?,
        files,
    })
}

/// Lê e analisa um arquivo de manifesto.
pub fn load(path: &Path) -> Result<Manifest, ManifestError> {
    fs::read_to_string(path)
        .map_err(ManifestError::Io)
        .and_then(|t| parse(&t))
}

/// Estado de um arquivo verificado contra o manifesto.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FileStatus {
    Ok,
    Missing,
    SizeMismatch { expected: u64, found: u64 },
    HashMismatch,
    Unreadable(String),
}

impl fmt::Display for FileStatus {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Ok => write!(f, "íntegro"),
            Self::Missing => write!(f, "ausente"),
            Self::SizeMismatch { expected, found } => {
                write!(
                    f,
                    "tamanho divergente (esperado {expected}, encontrado {found})"
                )
            }
            Self::HashMismatch => write!(f, "hash divergente"),
            Self::Unreadable(e) => write!(f, "ilegível: {e}"),
        }
    }
}

/// Um arquivo e seu estado.
#[derive(Debug, Clone)]
pub struct FileReport {
    pub path: String,
    pub status: FileStatus,
}

/// Resultado da verificação de um diretório contra o manifesto.
#[derive(Debug, Clone, Default)]
pub struct Report {
    pub files: Vec<FileReport>,
}

impl Report {
    /// Verdadeiro quando todos os arquivos estão íntegros.
    #[must_use]
    pub fn clean(&self) -> bool {
        self.files.iter().all(|r| r.status == FileStatus::Ok)
    }

    /// Itera só os arquivos com problema.
    pub fn failures(&self) -> impl Iterator<Item = &FileReport> {
        self.files.iter().filter(|r| r.status != FileStatus::Ok)
    }
}

/// Verifica `dir` contra o manifesto (tamanho primeiro, hash depois).
#[must_use]
pub fn verify(manifest: &Manifest, dir: &Path) -> Report {
    let mut out = Report {
        files: Vec::with_capacity(manifest.files.len()),
    };
    for e in &manifest.files {
        let status = match join_inside(dir, &e.path) {
            Err(err) => FileStatus::Unreadable(err.to_string()),
            Ok(full) => match fs::metadata(&full) {
                Err(err) if err.kind() == io::ErrorKind::NotFound => FileStatus::Missing,
                Err(err) => FileStatus::Unreadable(err.to_string()),
                Ok(meta) if meta.len() != e.size => FileStatus::SizeMismatch {
                    expected: e.size,
                    found: meta.len(),
                },
                Ok(_) => match fs::read(&full) {
                    Err(err) => FileStatus::Unreadable(err.to_string()),
                    Ok(bytes) if sha256::digest(&bytes) == e.sha256 => FileStatus::Ok,
                    Ok(_) => FileStatus::HashMismatch,
                },
            },
        };
        out.files.push(FileReport {
            path: e.path.clone(),
            status,
        });
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    fn tmpdir(tag: &str) -> PathBuf {
        use std::sync::atomic::{AtomicU64, Ordering};
        static N: AtomicU64 = AtomicU64::new(0);
        let d = std::env::temp_dir().join(format!(
            "wyd-launcher-manifest-{tag}-{}-{}",
            std::process::id(),
            N.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(&d).unwrap();
        d
    }

    const EXEMPLO: &str = "# wyd-launcher-manifest v1\nbuild 7662\nprotocol 12000\n\
        e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 0 vazio.dat\n\
        ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad 3 abc.dat\n";

    #[test]
    fn analisa_exemplo() {
        let m = parse(EXEMPLO).unwrap();
        assert_eq!((m.build, m.protocol), (7662, 12000));
        assert_eq!(m.files.len(), 2);
        assert_eq!(m.files[1].path, "abc.dat");
    }

    #[test]
    fn verifica_diretorio_integro() {
        let dir = tmpdir("ok");
        fs::write(dir.join("vazio.dat"), b"").unwrap();
        fs::write(dir.join("abc.dat"), b"abc").unwrap();
        let r = verify(&parse(EXEMPLO).unwrap(), &dir);
        assert!(r.clean());
        assert_eq!(r.failures().count(), 0);
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn detecta_ausente_tamanho_e_hash() {
        let dir = tmpdir("ruim");
        fs::write(dir.join("abc.dat"), b"abcd").unwrap(); // tamanho errado
        let r = verify(&parse(EXEMPLO).unwrap(), &dir);
        assert!(!r.clean());
        let falhas: Vec<_> = r.failures().collect();
        assert_eq!(falhas.len(), 2);
        assert_eq!(falhas[0].status, FileStatus::Missing);
        assert!(matches!(falhas[1].status, FileStatus::SizeMismatch { .. }));
        fs::write(dir.join("abc.dat"), b"abd").unwrap(); // tamanho certo, hash errado
        let r2 = verify(&parse(EXEMPLO).unwrap(), &dir);
        let abc = r2.files.iter().find(|r| r.path == "abc.dat").unwrap();
        assert_eq!(abc.status, FileStatus::HashMismatch);
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn rejeita_linhas_ruins_com_numero() {
        assert!(matches!(
            parse("build 1\nprotocol 2\nzzzz 3 x"),
            Err(ManifestError::BadLine { line: 3, .. })
        ));
        assert!(matches!(
            parse("build 1\n"),
            Err(ManifestError::MissingField("protocol"))
        ));
        assert!(matches!(
            parse("build 1\nbuild 2\nprotocol 3\n"),
            Err(ManifestError::DuplicateField("build"))
        ));
        assert!(matches!(
            parse("build 1\nprotocol 2\n../fuga 3 x"),
            Err(ManifestError::BadLine { .. })
        ));
    }
}
