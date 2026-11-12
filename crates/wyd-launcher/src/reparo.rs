//! Reparo do cliente a partir do manifesto servido pelo portal (FASE-08-7).
//!
//! Fluxo: baixa o manifesto do cliente → confere o diretório → para cada arquivo
//! divergente baixa o byte-exato do portal e confere tamanho + SHA-256 antes de
//! escrever → pós-verificação. Hash divergente continua bloqueando o Jogar até o
//! reparo ficar verde (fail-closed do `manifest::verify`).
//!
//! O transporte não autentica o manifesto (sem assinatura): o portal é a fonte
//! do operador e o reparo nunca aceita conteúdo fora do manifesto. Assinar o
//! manifesto é o passo seguinte (chave privada fora do repositório).

use std::fmt;
use std::path::{Path, PathBuf};

use crate::manifest::{self, Manifest};
use crate::patch::{self, NewFile};
use crate::{portal_client, sha256};

/// Rota do manifesto do cliente no portal.
pub const MANIFESTO_PATH: &str = "/api/launcher/cliente/manifesto";
/// Prefixo da rota de arquivo do cliente no portal.
pub const ARQUIVO_PATH: &str = "/api/launcher/cliente/arquivo/";

/// Erros tipados do reparo.
#[derive(Debug)]
pub enum ReparoError {
    Portal(portal_client::PortalError),
    Manifesto(manifest::ManifestError),
    /// Conteúdo baixado não confere com o manifesto (nada escrito).
    Divergente(String),
    Io(String),
    Aplicar(String),
    /// Pós-verificação ainda encontrou problemas.
    Verificacao(Vec<String>),
}

impl fmt::Display for ReparoError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Portal(e) => write!(f, "portal: {e}"),
            Self::Manifesto(e) => write!(f, "manifesto do cliente inválido: {e}"),
            Self::Divergente(p) => write!(f, "arquivo baixado diverge do manifesto: {p}"),
            Self::Io(e) => write!(f, "erro de E/S: {e}"),
            Self::Aplicar(e) => write!(f, "falha ao aplicar o reparo: {e}"),
            Self::Verificacao(falhas) => {
                write!(f, "pós-verificação falhou:")?;
                for falha in falhas {
                    write!(f, "\n  {falha}")?;
                }
                Ok(())
            }
        }
    }
}

impl std::error::Error for ReparoError {}

impl From<portal_client::PortalError> for ReparoError {
    fn from(e: portal_client::PortalError) -> Self {
        Self::Portal(e)
    }
}

impl From<manifest::ManifestError> for ReparoError {
    fn from(e: manifest::ManifestError) -> Self {
        Self::Manifesto(e)
    }
}

/// Resultado do reparo.
#[derive(Debug, Clone)]
pub struct RelatorioReparo {
    pub build: u32,
    /// Arquivos que estavam divergentes e foram rebaixados.
    pub reparados: Vec<String>,
    pub backup: PathBuf,
}

/// Percent-encoding dos segmentos do caminho (espaços e não-ASCII do cliente).
#[must_use]
pub fn codificar_caminho(caminho: &str) -> String {
    let mut saida = String::with_capacity(caminho.len());
    for b in caminho.bytes() {
        match b {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' | b'/' => {
                saida.push(b as char)
            }
            outro => saida.push_str(&format!("%{outro:02X}")),
        }
    }
    saida
}

/// Repara `dir` contra o manifesto servido por `base` (`http://host[:porta]`).
pub fn reparar(dir: &Path, base: &str) -> Result<RelatorioReparo, ReparoError> {
    let api = portal_client::PortalApi::new(base)?;
    let texto = api.fetch_bytes(MANIFESTO_PATH)?;
    let texto = String::from_utf8(texto)
        .map_err(|_| ReparoError::Io("manifesto do portal não é UTF-8".into()))?;
    let m: Manifest = manifest::parse(&texto)?;
    let rel = manifest::verify(&m, dir);
    if rel.clean() {
        return Ok(RelatorioReparo {
            build: m.build,
            reparados: Vec::new(),
            backup: dir.to_path_buf(),
        });
    }
    let mut novos: Vec<NewFile> = Vec::new();
    for falha in rel.failures() {
        let entry = m
            .files
            .iter()
            .find(|e| e.path == falha.path)
            .ok_or_else(|| ReparoError::Io(format!("manifesto sem {}", falha.path)))?;
        let caminho = format!("{ARQUIVO_PATH}{}", codificar_caminho(&entry.path));
        let bytes = api.fetch_bytes(&caminho)?;
        if bytes.len() as u64 != entry.size || sha256::digest(&bytes) != entry.sha256 {
            return Err(ReparoError::Divergente(entry.path.clone()));
        }
        novos.push(NewFile {
            path: entry.path.clone(),
            size: entry.size,
            sha256: entry.sha256,
            contents: bytes,
        });
    }
    let backup = dir.with_extension(format!(
        "reparo-bak-{}-{}",
        m.build,
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_secs())
            .unwrap_or(0)
    ));
    let reparados: Vec<String> = novos.iter().map(|n| n.path.clone()).collect();
    patch::apply(dir, &novos, &backup).map_err(|e| ReparoError::Aplicar(e.to_string()))?;
    let depois = manifest::verify(&m, dir);
    if !depois.clean() {
        patch::rollback(dir, &backup).ok();
        let falhas = depois
            .failures()
            .map(|f| format!("{}: {}", f.path, f.status))
            .collect();
        return Err(ReparoError::Verificacao(falhas));
    }
    Ok(RelatorioReparo {
        build: m.build,
        reparados,
        backup,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::io::{Read, Write};
    use std::net::TcpListener;
    use std::path::PathBuf;

    fn tmpdir(tag: &str) -> PathBuf {
        use std::sync::atomic::{AtomicU64, Ordering};
        static N: AtomicU64 = AtomicU64::new(0);
        let d = std::env::temp_dir().join(format!(
            "wyd-reparo-{tag}-{}-{}",
            std::process::id(),
            N.fetch_add(1, Ordering::Relaxed)
        ));
        let _ = fs::remove_dir_all(&d);
        fs::create_dir_all(&d).unwrap();
        d
    }

    /// Servidor HTTP/1.0 mínimo: manifest + arquivos servidos do mapa.
    fn servidor(rotas: Vec<(String, Vec<u8>)>) -> (String, std::thread::JoinHandle<()>) {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let porta = listener.local_addr().unwrap().port();
        let handle = std::thread::spawn(move || {
            for fluxo in listener.incoming() {
                let Ok(mut fluxo) = fluxo else { continue };
                let mut buf = [0u8; 4096];
                let n = fluxo.read(&mut buf).unwrap_or(0);
                let pedido = String::from_utf8_lossy(&buf[..n]);
                let caminho = pedido
                    .lines()
                    .next()
                    .and_then(|l| l.split_whitespace().nth(1))
                    .unwrap_or("/");
                match rotas.iter().find(|(rota, _)| rota == caminho) {
                    Some((_, corpo)) => {
                        let cabeca =
                            format!("HTTP/1.0 200 OK\r\nContent-Length: {}\r\n\r\n", corpo.len());
                        let _ = fluxo.write_all(cabeca.as_bytes());
                        let _ = fluxo.write_all(corpo);
                    }
                    None => {
                        let _ = fluxo.write_all(b"HTTP/1.0 404 Not Found\r\n\r\n");
                    }
                }
            }
        });
        (format!("http://127.0.0.1:{porta}"), handle)
    }

    #[test]
    fn baixa_arquivo_divergente_e_restaura_o_hash() {
        let dir = tmpdir("restaura");
        let bom = b"conteudo-bom".to_vec();
        let texto = format!(
            "build 7662\nprotocol 12000\n{} {} jogo.dat\n",
            sha256::hex_of(&bom),
            bom.len()
        );
        fs::write(dir.join("manifest.txt"), &texto).unwrap();
        fs::write(dir.join("jogo.dat"), b"adulterado").unwrap();
        let (base, _srv) = servidor(vec![
            (MANIFESTO_PATH.to_string(), texto.clone().into_bytes()),
            (format!("{ARQUIVO_PATH}jogo.dat"), bom.clone()),
        ]);
        let rel = reparar(&dir, &base).unwrap();
        assert_eq!(rel.build, 7662);
        assert_eq!(rel.reparados, vec!["jogo.dat".to_string()]);
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), bom);
        assert!(manifest::parse(&texto)
            .map(|m| manifest::verify(&m, &dir).clean())
            .unwrap());
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn conteudo_errado_no_portal_nao_escreve() {
        let dir = tmpdir("ruim");
        let esperado = b"conteudo-bom".to_vec();
        let texto = format!(
            "build 7662\nprotocol 12000\n{} {} jogo.dat\n",
            sha256::hex_of(&esperado),
            esperado.len()
        );
        fs::write(dir.join("jogo.dat"), b"adulterado").unwrap();
        let (base, _srv) = servidor(vec![
            (MANIFESTO_PATH.to_string(), texto.into_bytes()),
            (format!("{ARQUIVO_PATH}jogo.dat"), b"outro".to_vec()),
        ]);
        let erro = reparar(&dir, &base).unwrap_err();
        assert!(matches!(erro, ReparoError::Divergente(_)), "{erro}");
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"adulterado");
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn codifica_espacos_e_acentos_do_caminho() {
        let codado = codificar_caminho("Mesh/MeshTextureList - Cópia.bin");
        assert_eq!(codado, "Mesh/MeshTextureList%20-%20C%C3%B3pia.bin");
    }
}
