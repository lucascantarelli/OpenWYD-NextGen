//! Aplicação atômica de patch com backup e rollback (ADR-0003).
//!
//! Ordem obrigatória: validar fontes → backup → tmp + rename por arquivo →
//! verificação pós-escrita. Falha em qualquer ponto restaura o backup.

use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};

use crate::manifest::join_inside;
use crate::sha256;

/// Um arquivo novo a aplicar: conteúdo + hash/tamanho esperados.
#[derive(Debug, Clone)]
pub struct NewFile {
    pub path: String,
    pub size: u64,
    pub sha256: [u8; 32],
    pub contents: Vec<u8>,
}

/// Erros tipados de patch.
#[derive(Debug)]
pub enum PatchError {
    Io(io::Error),
    UnsafePath(String),
    /// O conteúdo entregue não confere com tamanho/hash (nada foi tocado).
    SourceCorrupt {
        path: String,
    },
    /// Leitura pós-escrita divergiu (backup já restaurado).
    VerifyFailed {
        path: String,
    },
}

impl fmt::Display for PatchError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Io(e) => write!(f, "erro de E/S: {e}"),
            Self::UnsafePath(p) => write!(f, "caminho inseguro: {p}"),
            Self::SourceCorrupt { path } => {
                write!(f, "fonte corrompida (tamanho/hash), nada aplicado: {path}")
            }
            Self::VerifyFailed { path } => {
                write!(
                    f,
                    "verificação pós-escrita falhou, backup restaurado: {path}"
                )
            }
        }
    }
}

impl std::error::Error for PatchError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Io(e) => Some(e),
            _ => None,
        }
    }
}

impl From<crate::manifest::ManifestError> for PatchError {
    fn from(e: crate::manifest::ManifestError) -> Self {
        match e {
            crate::manifest::ManifestError::Io(io) => Self::Io(io),
            crate::manifest::ManifestError::UnsafePath(p) => Self::UnsafePath(p),
            other => Self::Io(io::Error::other(other.to_string())),
        }
    }
}

static TMP_SEQ: AtomicU64 = AtomicU64::new(0);

fn tmp_name(target: &Path) -> PathBuf {
    let n = TMP_SEQ.fetch_add(1, Ordering::Relaxed);
    target.with_extension(format!("tmp-{}-{}", std::process::id(), n))
}

fn apply_one(dir: &Path, f: &NewFile) -> Result<bool, PatchError> {
    let target = join_inside(dir, &f.path)?;
    if let Some(parent) = target.parent() {
        fs::create_dir_all(parent).map_err(PatchError::Io)?;
    }
    let existed = target.exists();
    let tmp = tmp_name(&target);
    fs::write(&tmp, &f.contents).map_err(PatchError::Io)?;
    let back = fs::read(&tmp).map_err(PatchError::Io)?;
    if back.len() as u64 != f.size || sha256::digest(&back) != f.sha256 {
        fs::remove_file(&tmp).ok();
        return Err(PatchError::VerifyFailed {
            path: f.path.clone(),
        });
    }
    fs::rename(&tmp, &target).map_err(PatchError::Io)?;
    Ok(!existed)
}

/// Aplica `files` em `dir`, guardando originais em `backup_dir`.
///
/// Retorna a quantidade aplicada. Em falha, restaura o backup e remove
/// arquivos criados pelo patch — nunca deixa aplicação parcial como sucesso.
pub fn apply(dir: &Path, files: &[NewFile], backup_dir: &Path) -> Result<usize, PatchError> {
    for f in files {
        if f.contents.len() as u64 != f.size || sha256::digest(&f.contents) != f.sha256 {
            return Err(PatchError::SourceCorrupt {
                path: f.path.clone(),
            });
        }
    }
    fs::create_dir_all(backup_dir).map_err(PatchError::Io)?;

    for f in files {
        let target = join_inside(dir, &f.path)?;
        if target.exists() {
            let b = join_inside(backup_dir, &f.path)?;
            if let Some(parent) = b.parent() {
                fs::create_dir_all(parent).map_err(PatchError::Io)?;
            }
            fs::copy(&target, &b).map_err(PatchError::Io)?;
        }
    }

    let mut created: Vec<PathBuf> = Vec::new();
    let mut done = 0;
    for f in files {
        match apply_one(dir, f) {
            Ok(was_new) => {
                if was_new {
                    created.push(join_inside(dir, &f.path)?);
                }
                done += 1;
            }
            Err(e) => {
                for c in &created {
                    fs::remove_file(c).ok();
                }
                rollback(dir, backup_dir).ok();
                return Err(e);
            }
        }
    }
    Ok(done)
}

fn collect_files(root: &Path) -> io::Result<Vec<PathBuf>> {
    let mut out = Vec::new();
    let mut stack = vec![root.to_path_buf()];
    while let Some(d) = stack.pop() {
        for entry in fs::read_dir(&d)? {
            let p = entry?.path();
            if p.is_dir() {
                stack.push(p);
            } else {
                out.push(p);
            }
        }
    }
    Ok(out)
}

/// Restaura em `dir` tudo que está em `backup_dir`. Não remove arquivos novos —
/// só devolve os originais. Retorna a quantidade restaurada.
pub fn rollback(dir: &Path, backup_dir: &Path) -> Result<usize, PatchError> {
    let mut done = 0;
    for full in collect_files(backup_dir).map_err(PatchError::Io)? {
        let rel = full
            .strip_prefix(backup_dir)
            .map_err(|_| PatchError::Io(io::Error::other("backup fora da base esperada")))?;
        let rel_s = rel.to_string_lossy().replace('\\', "/");
        let target = join_inside(dir, &rel_s)?;
        if let Some(parent) = target.parent() {
            fs::create_dir_all(parent).map_err(PatchError::Io)?;
        }
        fs::copy(&full, &target).map_err(PatchError::Io)?;
        done += 1;
    }
    Ok(done)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn tmpdir(tag: &str) -> PathBuf {
        static N: AtomicU64 = AtomicU64::new(1000);
        let d = std::env::temp_dir().join(format!(
            "wyd-launcher-patch-{tag}-{}-{}",
            std::process::id(),
            N.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(&d).unwrap();
        d
    }

    fn novo(path: &str, contents: &[u8]) -> NewFile {
        NewFile {
            path: path.to_string(),
            size: contents.len() as u64,
            sha256: sha256::digest(contents),
            contents: contents.to_vec(),
        }
    }

    #[test]
    fn aplica_arquivo_novo_e_reserva_backup() {
        let dir = tmpdir("novo");
        let bak = dir.join("bak");
        let n = apply(&dir, &[novo("sub/jogo.dat", b"v2")], &bak).unwrap();
        assert_eq!(n, 1);
        assert_eq!(fs::read(dir.join("sub/jogo.dat")).unwrap(), b"v2");
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn substitui_e_rollback_manual_devolve_original() {
        let dir = tmpdir("troca");
        let bak = dir.join("bak");
        fs::write(dir.join("a.dat"), b"v1").unwrap();
        apply(&dir, &[novo("a.dat", b"v2")], &bak).unwrap();
        assert_eq!(fs::read(dir.join("a.dat")).unwrap(), b"v2");
        assert_eq!(rollback(&dir, &bak).unwrap(), 1);
        assert_eq!(fs::read(dir.join("a.dat")).unwrap(), b"v1");
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn fonte_corrompida_nao_toca_em_nada() {
        let dir = tmpdir("corrompida");
        let bak = dir.join("bak");
        fs::write(dir.join("a.dat"), b"v1").unwrap();
        let mut ruim = novo("a.dat", b"v2");
        ruim.contents = b"adulterado".to_vec(); // hash não confere
        let r = apply(&dir, &[ruim], &bak);
        assert!(matches!(r, Err(PatchError::SourceCorrupt { .. })));
        assert_eq!(fs::read(dir.join("a.dat")).unwrap(), b"v1");
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn falha_no_meio_restaura_tudo() {
        let dir = tmpdir("meio");
        let bak = dir.join("bak");
        fs::write(dir.join("a.dat"), b"v1").unwrap();
        let mut ruim = novo("b.dat", b"v2");
        ruim.sha256 = [0u8; 32]; // fonte inválida de propósito
        let r = apply(&dir, &[novo("a.dat", b"v1-nova"), ruim], &bak);
        assert!(r.is_err());
        assert_eq!(fs::read(dir.join("a.dat")).unwrap(), b"v1");
        assert!(!dir.join("b.dat").exists());
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn rejeita_fuga_de_diretorio() {
        let dir = tmpdir("fuga");
        let bak = dir.join("bak");
        let r = apply(&dir, &[novo("../fora.dat", b"x")], &bak);
        assert!(matches!(r, Err(PatchError::UnsafePath(_))));
        fs::remove_dir_all(&dir).ok();
    }
}
