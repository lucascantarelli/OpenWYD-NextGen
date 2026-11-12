//! Distribuição: pacote assinado com Ed25519 e `update` com rollback (CLIENT-03).
//!
//! Formato do pacote (tudo relativo à raiz do pacote):
//! `manifesto` (formato de `manifest::parse`), `assinatura` (Ed25519 da chave
//! privada do operador sobre os bytes exatos do `manifesto`, em 128 hex) e
//! `arquivos/<...>` (payloads descritos no manifesto).
//!
//! A verificação usa a chave **pública** embutida (`CHAVE_PUBLICA_HEX`): o
//! binário não guarda segredo, então copiar o executável não permite re-assinar
//! um pacote adulterado — é essa a diferença para o HMAC simétrico anterior.
//! Rotacionar a chave é publicar um launcher novo (recompila com a chave nova);
//! a chave privada vive só no CI/secret e assina via `tools/launcher/sign-package.mjs`.
//!
//! Transporte remoto (HTTPS) está fora desta passada: a fonte do pacote é um
//! caminho local (espelho, pasta de rede, mídia). O pipeline buscar →
//! verificar → aplicar é o mesmo que um transporte remoto usará.

use std::fmt;
use std::fs;
use std::io::{self, Seek, SeekFrom};
use std::path::{Path, PathBuf};

use ed25519_dalek::{Signature, VerifyingKey};

use crate::manifest::{self, join_inside, Manifest};
use crate::patch::{self, NewFile};
use crate::sha256;

/// Chave pública Ed25519 (32 bytes em hex) que assina os pacotes de patch.
///
/// Par gerado com `node tools/launcher/sign-package.mjs keygen <chave.pem>` em
/// 2026-11-11; a metade privada não está no repositório. Trocar esta constante
/// é uma entrega de launcher novo, nunca um passo de operação.
pub const CHAVE_PUBLICA_HEX: &str =
    "f49843b32e50a342b14f29751fbb7177cec0dd6ec6fe80df01cb699e68d19b18";

/// Erros tipados de distribuição (pacote ruim é o caso normal, não exceção).
#[derive(Debug)]
pub enum DistError {
    Io(io::Error),
    Manifest(manifest::ManifestError),
    Patch(patch::PatchError),
    /// A constante embutida não é uma chave Ed25519 válida (build quebrado).
    ChaveEmbutida,
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
            Self::ChaveEmbutida => write!(
                f,
                "chave pública embutida inválida (build do launcher inconsistente)"
            ),
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

/// Um dígito hex (maiúsculo ou minúsculo) em valor.
fn valor_hex(c: u8) -> Option<u8> {
    match c {
        b'0'..=b'9' => Some(c - b'0'),
        b'a'..=b'f' => Some(c - b'a' + 10),
        b'A'..=b'F' => Some(c - b'A' + 10),
        _ => None,
    }
}

/// Hex de exatamente `N` bytes (espaços/quebras nas pontas são ignorados).
///
/// Devolve `None` para tamanho errado ou caractere não-hex: assinatura truncada
/// é `BadSignature`, nunca `panic` nem índice fora da faixa.
fn de_hex<const N: usize>(texto: &str) -> Option<[u8; N]> {
    let bytes = texto.trim().as_bytes();
    if bytes.len() != N * 2 {
        return None;
    }
    let mut out = [0u8; N];
    for (i, par) in bytes.as_chunks::<2>().0.iter().enumerate() {
        out[i] = (valor_hex(par[0])? << 4) | valor_hex(par[1])?;
    }
    Some(out)
}

/// Chave pública embutida, pronta para verificar pacotes.
///
/// `Err` aqui só acontece com a constante corrompida (build quebrado), nunca
/// por causa de entrada de rede ou de pacote.
pub fn chave_embutida() -> Result<VerifyingKey, DistError> {
    let bytes = de_hex::<32>(CHAVE_PUBLICA_HEX).ok_or(DistError::ChaveEmbutida)?;
    VerifyingKey::from_bytes(&bytes).map_err(|_| DistError::ChaveEmbutida)
}

/// Verifica a assinatura do pacote e devolve o manifesto validado.
///
/// Ordem: manifesto parseado → assinatura Ed25519 confere → payloads conferem.
/// Qualquer divergência recusa o pacote inteiro antes de tocar no diretório do
/// jogo. `verify_strict` recusa assinaturas malleáveis (metade da ordem + L).
pub fn verify_package(pkg: &Path, chave: &VerifyingKey) -> Result<Manifest, DistError> {
    let raw = fs::read(join_inside(pkg, "manifesto").map_err(DistError::Manifest)?)?;
    let m = manifest::parse(std::str::from_utf8(&raw).map_err(|_| {
        DistError::Manifest(manifest::ManifestError::BadLine {
            line: 1,
            reason: "manifesto não é UTF-8",
        })
    })?)?;
    let sig_path = join_inside(pkg, "assinatura").map_err(DistError::Manifest)?;
    let sig_txt = fs::read_to_string(&sig_path)?;
    let bruta = de_hex::<64>(&sig_txt).ok_or(DistError::BadSignature)?;
    chave
        .verify_strict(&raw, &Signature::from_bytes(&bruta))
        .map_err(|_| DistError::BadSignature)?;
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
pub fn plan(dir: &Path, pkg: &Path, chave: &VerifyingKey) -> Result<UpdatePlan, DistError> {
    let m = verify_package(pkg, chave)?;
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

/// Diretório de staging único por execução (pid + segundo + contador).
///
/// Sem o contador, dois `update` no mesmo processo e no mesmo segundo (testes em
/// paralelo, duas janelas) compartilhariam o staging e um apagaria o arquivo do
/// outro — falha intermitente que não existe em produção, mas existe no harness.
fn staging_dir() -> PathBuf {
    use std::sync::atomic::{AtomicU64, Ordering};
    static SEQ: AtomicU64 = AtomicU64::new(0);
    std::env::temp_dir().join(format!(
        "wyd-update-{}-{}-{}",
        std::process::id(),
        epoch_s(),
        SEQ.fetch_add(1, Ordering::Relaxed)
    ))
}

/// Aplica o pacote em `dir`: verifica assinatura → baixa → patch atômico → reverifica.
///
/// O rollback é do `patch::apply` (restaura os alvos já trocados quando uma
/// escrita falha no meio) e do `PostVerify` aqui: se o diretório não ficar
/// idêntico ao manifesto, o backup volta antes de devolver o erro.
pub fn update(dir: &Path, pkg: &Path, chave: &VerifyingKey) -> Result<UpdateReport, DistError> {
    let m = verify_package(pkg, chave)?;
    let base = join_inside(pkg, "arquivos").map_err(DistError::Manifest)?;

    let staging = staging_dir();
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
    use ed25519_dalek::{Signer, SigningKey};
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

    /// Chave de teste determinística (nunca a embutida, que não tem privada aqui).
    fn chave_de_teste() -> SigningKey {
        SigningKey::from_bytes(&[7u8; 32])
    }

    fn hex(bytes: &[u8]) -> String {
        const D: &[u8; 16] = b"0123456789abcdef";
        let mut s = String::with_capacity(bytes.len() * 2);
        for b in bytes {
            s.push(D[(b >> 4) as usize] as char);
            s.push(D[(b & 15) as usize] as char);
        }
        s
    }

    /// Monta um pacote válido em `pkg` com um arquivo `jogo.dat`, assinado pela chave de teste.
    fn pacote_valido(pkg: &Path, conteudo: &[u8]) -> Manifest {
        let m_txt = format!(
            "# wyd-launcher-manifest v1\nbuild 7662\nprotocol 12000\n{} {} jogo.dat\n",
            sha256::hex_of(conteudo),
            conteudo.len()
        );
        fs::write(pkg.join("manifesto"), &m_txt).unwrap();
        let sig = chave_de_teste().sign(m_txt.as_bytes());
        fs::write(pkg.join("assinatura"), hex(&sig.to_bytes())).unwrap();
        let arq = pkg.join("arquivos");
        fs::create_dir_all(&arq).unwrap();
        fs::write(arq.join("jogo.dat"), conteudo).unwrap();
        manifest::parse(&m_txt).unwrap()
    }

    #[test]
    fn sha256_cobre_mensagem_de_2_blocos() {
        // Vetor NIST de 56 bytes (força o caminho multi-bloco).
        let got = sha256::hex_of(b"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq");
        assert_eq!(
            got,
            "248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1"
        );
    }

    #[test]
    fn chave_embutida_e_uma_chave_ed25519_valida() {
        let k = chave_embutida().expect("CHAVE_PUBLICA_HEX deve ser 32 bytes válidos");
        // A chave privada correspondente não está no repositório: só a pública entra.
        assert_ne!(hex(k.as_bytes()), "0".repeat(64));
    }

    #[test]
    fn pacote_valido_aplica_e_fica_verde() {
        let pkg = tmpdir("pkg-ok");
        let dir = tmpdir("dir-ok");
        pacote_valido(&pkg, b"v2-do-pacote");
        fs::write(dir.join("jogo.dat"), b"v1-antiga").unwrap();
        let rel = update(&dir, &pkg, &chave_de_teste().verifying_key()).unwrap();
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
    fn assinatura_de_outra_chave_recusa_antes_de_tocar_no_disco() {
        let pkg = tmpdir("pkg-sig");
        let dir = tmpdir("dir-sig");
        pacote_valido(&pkg, b"v2");
        // Assinatura legítima, mas de quem não é o dono da chave embutida.
        let manifesto = fs::read(pkg.join("manifesto")).unwrap();
        let outra = SigningKey::from_bytes(&[9u8; 32]);
        fs::write(
            pkg.join("assinatura"),
            hex(&outra.sign(&manifesto).to_bytes()),
        )
        .unwrap();
        fs::write(dir.join("jogo.dat"), b"v1").unwrap();
        let r = update(&dir, &pkg, &chave_de_teste().verifying_key());
        assert!(matches!(r, Err(DistError::BadSignature)));
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"v1");
        fs::remove_dir_all(&pkg).ok();
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn manifesto_alterado_depois_de_assinar_recusa() {
        let pkg = tmpdir("pkg-tamper");
        let dir = tmpdir("dir-tamper");
        pacote_valido(&pkg, b"v2");
        // Altera um valor que o parser aceita (build) sem re-assinar: o manifesto
        // continua válido sintaticamente, então quem tem de recusar é a assinatura.
        let texto = fs::read_to_string(pkg.join("manifesto")).unwrap();
        let adulterado = texto.replace("build 7662", "build 7663");
        assert_ne!(texto, adulterado);
        fs::write(pkg.join("manifesto"), adulterado).unwrap();
        fs::write(dir.join("jogo.dat"), b"v1").unwrap();
        let r = update(&dir, &pkg, &chave_de_teste().verifying_key());
        assert!(matches!(r, Err(DistError::BadSignature)));
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"v1");
        fs::remove_dir_all(&pkg).ok();
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn assinatura_truncada_e_erro_tipado_sem_panic() {
        let pkg = tmpdir("pkg-curta");
        pacote_valido(&pkg, b"v2");
        fs::write(pkg.join("assinatura"), "62e3a807\n").unwrap();
        let r = verify_package(&pkg, &chave_de_teste().verifying_key());
        assert!(matches!(r, Err(DistError::BadSignature)));
        fs::remove_dir_all(&pkg).ok();
    }

    #[test]
    fn payload_adulterado_recusa_sem_aplicar() {
        let pkg = tmpdir("pkg-pay");
        let dir = tmpdir("dir-pay");
        pacote_valido(&pkg, b"v2");
        // adultera o payload mantendo o tamanho (passa no size, falha no hash)
        fs::write(pkg.join("arquivos/jogo.dat"), b"vX").unwrap();
        fs::write(dir.join("jogo.dat"), b"v1").unwrap();
        let r = update(&dir, &pkg, &chave_de_teste().verifying_key());
        assert!(matches!(r, Err(DistError::PayloadCorrupt { .. })));
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"v1");
        fs::remove_dir_all(&pkg).ok();
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn falha_no_meio_do_patch_restaura_o_backup() {
        let pkg = tmpdir("pkg-rb");
        let dir = tmpdir("dir-rb");
        let (a, b) = (b"novo-a".as_slice(), b"novo-b".as_slice());
        let m_txt = format!(
            "# wyd-launcher-manifest v1\nbuild 7662\nprotocol 12000\n{} {} a.dat\n{} {} bloco/sub.dat\n",
            sha256::hex_of(a),
            a.len(),
            sha256::hex_of(b),
            b.len()
        );
        fs::write(pkg.join("manifesto"), &m_txt).unwrap();
        let sig = chave_de_teste().sign(m_txt.as_bytes());
        fs::write(pkg.join("assinatura"), hex(&sig.to_bytes())).unwrap();
        let arq = pkg.join("arquivos");
        fs::create_dir_all(arq.join("bloco")).unwrap();
        fs::write(arq.join("a.dat"), a).unwrap();
        fs::write(arq.join("bloco/sub.dat"), b).unwrap();

        fs::write(dir.join("a.dat"), b"antigo-a").unwrap();
        // `bloco` é arquivo, não diretório: a segunda escrita falha depois de
        // `a.dat` já ter sido trocado — é o cenário que exige rollback.
        fs::write(dir.join("bloco"), b"arquivo-no-lugar-de-dir").unwrap();

        let r = update(&dir, &pkg, &chave_de_teste().verifying_key());
        assert!(r.is_err(), "update devia falhar no segundo arquivo");
        assert_eq!(
            fs::read(dir.join("a.dat")).unwrap(),
            b"antigo-a",
            "rollback devolveu o arquivo já aplicado"
        );
        assert!(dir.join("bloco").is_file(), "nada foi criado no lugar");
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
        let p = plan(&dir, &pkg, &chave_de_teste().verifying_key()).unwrap();
        assert_eq!(p.build, 7662);
        assert_eq!(p.mudariam.len(), 1);
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"v1"); // intacto
        fs::remove_dir_all(&pkg).ok();
        fs::remove_dir_all(&dir).ok();
    }
}
