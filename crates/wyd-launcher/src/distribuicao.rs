//! Distribuição: busca com resume, assinatura de pacote e `update` (FASE-08-3).
//!
//! Formato do pacote (tudo relativo à raiz do pacote):
//! `manifesto` (formato de `manifest::parse`), `assinatura` (HMAC-SHA256 da
//! chave sobre os bytes exatos do `manifesto`, em 64 hex) e `arquivos/<...>`
//! (payloads descritos no manifesto). A assinatura é simétrica e provisória:
//! protege integridade em trânsito/espelho, mas a chave é compartilhada —
//! trocar por assinatura assimétrica quando houver acesso a registry.
//!
//! Transporte remoto (HTTPS) está fora desta passada: a fonte do pacote é um
//! caminho local (espelho, pasta de rede, mídia). O pipeline buscar →
//! verificar → aplicar é o mesmo que um transporte remoto usará.

use std::fmt;
use std::fs;
use std::io::{self, Seek, SeekFrom};
use std::path::{Path, PathBuf};

use crate::manifest::{self, join_inside, Manifest};
use crate::patch::{self, NewFile};
use crate::sha256;

/// Erros tipados de distribuição (pacote ruim é o caso normal, não exceção).
#[derive(Debug)]
pub enum DistError {
    Io(io::Error),
    Manifest(manifest::ManifestError),
    Patch(patch::PatchError),
    KeyMissing(PathBuf),
    BadSignature,
    /// Payload no pacote não confere com o manifesto (nada foi aplicado).
    PayloadCorrupt {
        path: String,
    },
    /// Pós-verificação do diretório falhou (backup já restaurado).
    PostVerify {
        path: String,
    },
}

impl fmt::Display for DistError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Io(e) => write!(f, "erro de E/S: {e}"),
            Self::Manifest(e) => write!(f, "manifesto do pacote inválido: {e}"),
            Self::Patch(e) => write!(f, "falha ao aplicar: {e}"),
            Self::KeyMissing(p) => write!(f, "arquivo de chave não encontrado: {}", p.display()),
            Self::BadSignature => write!(f, "assinatura do pacote inválida (pacote recusado)"),
            Self::PayloadCorrupt { path } => {
                write!(f, "payload corrompido no pacote, nada aplicado: {path}")
            }
            Self::PostVerify { path } => {
                write!(f, "pós-verificação falhou, backup restaurado: {path}")
            }
        }
    }
}

impl std::error::Error for DistError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Io(e) => Some(e),
            Self::Manifest(e) => Some(e),
            Self::Patch(e) => Some(e),
            _ => None,
        }
    }
}

impl From<io::Error> for DistError {
    fn from(e: io::Error) -> Self {
        Self::Io(e)
    }
}

impl From<manifest::ManifestError> for DistError {
    fn from(e: manifest::ManifestError) -> Self {
        Self::Manifest(e)
    }
}

impl From<patch::PatchError> for DistError {
    fn from(e: patch::PatchError) -> Self {
        Self::Patch(e)
    }
}

/// HMAC-SHA256 (RFC 2104) sobre `sha256` em `std` puro.
#[must_use]
pub fn hmac_sha256(key: &[u8], msg: &[u8]) -> [u8; 32] {
    const BLOCO: usize = 64;
    let mut kpad = [0u8; BLOCO];
    if key.len() > BLOCO {
        let d = sha256::digest(key);
        kpad[..32].copy_from_slice(&d);
    } else {
        kpad[..key.len()].copy_from_slice(key);
    }
    let mut ipad = [0x36u8; BLOCO];
    let mut opad = [0x5cu8; BLOCO];
    for i in 0..BLOCO {
        ipad[i] ^= kpad[i];
        opad[i] ^= kpad[i];
    }
    let mut dentro = Vec::with_capacity(BLOCO + msg.len());
    dentro.extend_from_slice(&ipad);
    dentro.extend_from_slice(msg);
    let h_dentro = sha256::digest(&dentro);
    let mut fora = Vec::with_capacity(BLOCO + 32);
    fora.extend_from_slice(&opad);
    fora.extend_from_slice(&h_dentro);
    sha256::digest(&fora)
}

/// Lê a chave de um arquivo (bytes crus, sem `\n`/`\r` final).
pub fn load_key(path: &Path) -> Result<Vec<u8>, DistError> {
    if !path.exists() {
        return Err(DistError::KeyMissing(path.to_path_buf()));
    }
    let mut k = fs::read(path)?;
    while k.last().is_some_and(|b| *b == b'\n' || *b == b'\r') {
        k.pop();
    }
    Ok(k)
}

/// Verifica a assinatura do pacote e devolve o manifesto validado.
///
/// Ordem: manifesto parseado → HMAC confere → payloads conferem. Qualquer
/// divergência recusa o pacote inteiro antes de tocar no diretório do jogo.
pub fn verify_package(pkg: &Path, key: &[u8]) -> Result<Manifest, DistError> {
    let raw = fs::read(join_inside(pkg, "manifesto").map_err(DistError::Manifest)?)?;
    let m = manifest::parse(std::str::from_utf8(&raw).map_err(|_| {
        DistError::Manifest(manifest::ManifestError::BadLine {
            line: 1,
            reason: "manifesto não é UTF-8",
        })
    })?)?;
    let sig_path = join_inside(pkg, "assinatura").map_err(DistError::Manifest)?;
    let sig_txt = fs::read_to_string(&sig_path)?.trim().to_string();
    let esperada = sha256::parse_hex(&sig_txt).ok_or(DistError::BadSignature)?;
    if hmac_sha256(key, &raw) != esperada {
        return Err(DistError::BadSignature);
    }
    let base = join_inside(pkg, "arquivos").map_err(DistError::Manifest)?;
    for e in &m.files {
        let full = join_inside(&base, &e.path).map_err(DistError::Manifest)?;
        let bytes = fs::read(&full).map_err(|err| {
            if err.kind() == io::ErrorKind::NotFound {
                DistError::PayloadCorrupt {
                    path: e.path.clone(),
                }
            } else {
                DistError::Io(err)
            }
        })?;
        if bytes.len() as u64 != e.size || sha256::digest(&bytes) != e.sha256 {
            return Err(DistError::PayloadCorrupt {
                path: e.path.clone(),
            });
        }
    }
    Ok(m)
}

/// Baixa (copia) um payload para `staging`, retomando de `.part` parcial.
///
/// Só finaliza com rename quando tamanho + hash conferem.
pub fn fetch_one(
    src: &Path,
    staging: &Path,
    entry: &manifest::Entry,
) -> Result<PathBuf, DistError> {
    let fin = join_inside(staging, &entry.path).map_err(DistError::Manifest)?;
    if let Some(p) = fin.parent() {
        fs::create_dir_all(p)?;
    }
    let part = PathBuf::from(format!("{}.part", fin.display()));
    let mut offset = 0u64;
    if part.exists() {
        let n = fs::metadata(&part)?.len();
        if n < entry.size {
            offset = n; // resume: continua de onde parou
        } else {
            fs::remove_file(&part)?; // parcial inválida: recomeça
        }
    }
    let mut sf = fs::File::open(src)?;
    sf.seek(SeekFrom::Start(offset))?;
    {
        let mut pf = fs::OpenOptions::new()
            .create(true)
            .append(true)
            .open(&part)?;
        io::copy(&mut sf, &mut pf)?;
    }
    let bytes = fs::read(&part)?;
    if bytes.len() as u64 != entry.size || sha256::digest(&bytes) != entry.sha256 {
        return Err(DistError::PayloadCorrupt {
            path: entry.path.clone(),
        });
    }
    fs::rename(&part, &fin)?;
    Ok(fin)
}

/// O que o `update` mudaria, sem escrever nada (para `--dry-run`).
#[derive(Debug, Clone)]
pub struct UpdatePlan {
    pub build: u32,
    /// Caminhos que divergem do manifesto do pacote (seriam aplicados).
    pub mudariam: Vec<String>,
}

/// Calcula o plano: pacote íntegro + diff contra o diretório do jogo.
pub fn plan(dir: &Path, pkg: &Path, key: &[u8]) -> Result<UpdatePlan, DistError> {
    let m = verify_package(pkg, key)?;
    let rel = manifest::verify(&m, dir);
    let mudariam = rel
        .failures()
        .map(|r| format!("{}: {}", r.path, r.status))
        .collect();
    Ok(UpdatePlan {
        build: m.build,
        mudariam,
    })
}

/// Relatório de um `update` aplicado.
#[derive(Debug, Clone)]
pub struct UpdateReport {
    pub aplicados: usize,
    pub backup: PathBuf,
    pub build: u32,
}

fn epoch_s() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

/// Aplica o pacote em `dir`: verifica assinatura → baixa → patch atômico → reverifica.
pub fn update(dir: &Path, pkg: &Path, key: &[u8]) -> Result<UpdateReport, DistError> {
    let m = verify_package(pkg, key)?;
    let base = join_inside(pkg, "arquivos").map_err(DistError::Manifest)?;

    let staging =
        std::env::temp_dir().join(format!("wyd-update-{}-{}", std::process::id(), epoch_s()));
    fs::create_dir_all(&staging)?;
    let resultado = (|| -> Result<UpdateReport, DistError> {
        let mut novos: Vec<NewFile> = Vec::with_capacity(m.files.len());
        for e in &m.files {
            let src = join_inside(&base, &e.path).map_err(DistError::Manifest)?;
            let fin = fetch_one(&src, &staging, e)?;
            let contents = fs::read(&fin)?;
            novos.push(NewFile {
                path: e.path.clone(),
                size: e.size,
                sha256: e.sha256,
                contents,
            });
        }
        let backup = dir.with_extension(format!("bak-{}-{}", m.build, epoch_s()));
        let aplicados = patch::apply(dir, &novos, &backup)?;
        let rel = manifest::verify(&m, dir);
        if !rel.clean() {
            let primeiro = rel
                .failures()
                .next()
                .map(|r| r.path.clone())
                .unwrap_or_else(|| "?".to_string());
            patch::rollback(dir, &backup).ok();
            return Err(DistError::PostVerify { path: primeiro });
        }
        Ok(UpdateReport {
            aplicados,
            backup,
            build: m.build,
        })
    })();
    fs::remove_dir_all(&staging).ok();
    resultado
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU64, Ordering};

    fn tmpdir(tag: &str) -> PathBuf {
        static N: AtomicU64 = AtomicU64::new(5000);
        let d = std::env::temp_dir().join(format!(
            "wyd-launcher-dist-{tag}-{}-{}",
            std::process::id(),
            N.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(&d).unwrap();
        d
    }

    const CHAVE: &[u8] = b"chave-de-teste-08-3";

    /// Monta um pacote válido em `pkg` com um arquivo `jogo.dat`.
    fn pacote_valido(pkg: &Path, conteudo: &[u8]) -> Manifest {
        let m_txt = format!(
            "# wyd-launcher-manifest v1\nbuild 7662\nprotocol 12000\n{} {} jogo.dat\n",
            sha256::hex_of(conteudo),
            conteudo.len()
        );
        fs::write(pkg.join("manifesto"), &m_txt).unwrap();
        let sig = hmac_sha256(CHAVE, m_txt.as_bytes());
        fs::write(pkg.join("assinatura"), sha256_hex(&sig)).unwrap();
        let arq = pkg.join("arquivos");
        fs::create_dir_all(&arq).unwrap();
        fs::write(arq.join("jogo.dat"), conteudo).unwrap();
        manifest::parse(&m_txt).unwrap()
    }

    fn sha256_hex(d: &[u8; 32]) -> String {
        const HEX: &[u8; 16] = b"0123456789abcdef";
        let mut s = String::with_capacity(64);
        for b in d {
            s.push(HEX[(b >> 4) as usize] as char);
            s.push(HEX[(b & 15) as usize] as char);
        }
        s
    }

    #[test]
    fn sha256_cobre_mensagem_de_2_blocos() {
        // Vetor NIST de 56 bytes (força o caminho multi-bloco que o HMAC usa).
        let got = sha256::hex_of(b"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq");
        assert_eq!(
            got,
            "248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1"
        );
    }

    #[test]
    fn hmac_confere_vetor_rfc4231() {
        // Caso 1 da RFC 4231: chave 0x0b×20, dados "Hi There". Valor conferido
        // contra implementação independente (Node crypto/OpenSSL) em 2026-11-07.
        let got = hmac_sha256(&[0x0bu8; 20], b"Hi There");
        assert_eq!(
            sha256_hex(&got),
            "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7"
        );
    }

    #[test]
    fn pacote_valido_aplica_e_fica_verde() {
        let pkg = tmpdir("pkg-ok");
        let dir = tmpdir("dir-ok");
        pacote_valido(&pkg, b"v2-do-pacote");
        fs::write(dir.join("jogo.dat"), b"v1-antiga").unwrap();
        let rel = update(&dir, &pkg, CHAVE).unwrap();
        assert_eq!(rel.aplicados, 1);
        assert_eq!(rel.build, 7662);
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"v2-do-pacote");
        let m = manifest::load(&pkg.join("manifesto")).unwrap();
        assert!(manifest::verify(&m, &dir).clean());
        fs::remove_dir_all(&pkg).ok();
        fs::remove_dir_all(&dir).ok();
        fs::remove_dir_all(&rel.backup).ok();
    }

    #[test]
    fn assinatura_errada_recusa_antes_de_tocar_no_disco() {
        let pkg = tmpdir("pkg-sig");
        let dir = tmpdir("dir-sig");
        pacote_valido(&pkg, b"v2");
        fs::write(pkg.join("assinatura"), "0".repeat(64)).unwrap();
        fs::write(dir.join("jogo.dat"), b"v1").unwrap();
        let r = update(&dir, &pkg, CHAVE);
        assert!(matches!(r, Err(DistError::BadSignature)));
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"v1");
        fs::remove_dir_all(&pkg).ok();
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn payload_adulterado_recusa_sem_aplicar() {
        let pkg = tmpdir("pkg-pay");
        let dir = tmpdir("dir-pay");
        pacote_valido(&pkg, b"v2");
        // adultera o payload mantendo o tamanho (passa no size, falha no hash)
        fs::write(pkg.join("arquivos/jogo.dat"), b"vX").unwrap();
        fs::write(dir.join("jogo.dat"), b"v1").unwrap();
        let r = update(&dir, &pkg, CHAVE);
        assert!(matches!(r, Err(DistError::PayloadCorrupt { .. })));
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"v1");
        fs::remove_dir_all(&pkg).ok();
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn resume_completa_parcial() {
        let pkg = tmpdir("pkg-res");
        pacote_valido(&pkg, b"0123456789abcdef");
        let m = manifest::load(&pkg.join("manifesto")).unwrap();
        let staging = tmpdir("stg-res");
        let fin = join_inside(&staging, "jogo.dat").unwrap();
        let part = PathBuf::from(format!("{}.part", fin.display()));
        fs::create_dir_all(fin.parent().unwrap()).unwrap();
        fs::write(&part, b"01234567").unwrap(); // metade
        let src = pkg.join("arquivos/jogo.dat");
        let out = fetch_one(&src, &staging, &m.files[0]).unwrap();
        assert_eq!(out, fin);
        assert_eq!(fs::read(&fin).unwrap(), b"0123456789abcdef");
        assert!(!part.exists());
        fs::remove_dir_all(&pkg).ok();
        fs::remove_dir_all(&staging).ok();
    }

    #[test]
    fn plano_nao_escreve_nada() {
        let pkg = tmpdir("pkg-plan");
        let dir = tmpdir("dir-plan");
        pacote_valido(&pkg, b"v2");
        fs::write(dir.join("jogo.dat"), b"v1").unwrap();
        let p = plan(&dir, &pkg, CHAVE).unwrap();
        assert_eq!(p.build, 7662);
        assert_eq!(p.mudariam.len(), 1);
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"v1"); // intacto
        fs::remove_dir_all(&pkg).ok();
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn chave_ausente_e_erro_tipado() {
        let r = load_key(Path::new("chave-que-nao-existe.key"));
        assert!(matches!(r, Err(DistError::KeyMissing(_))));
    }
}
