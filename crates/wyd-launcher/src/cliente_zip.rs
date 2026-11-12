//! Instalação/atualização do cliente a partir do `client.zip` (CLIENT-07/08-8).
//!
//! O `client.zip` é o pacote publicado pelo operador (GitHub Releases, Drive ou
//! qualquer HTTP que sirva o arquivo). A origem é **configurável** por argumento
//! ou `WYD_LAUNCHER_CLIENT_ZIP`; o manifesto que manda nos hashes vem do próprio
//! `client/` ou de `WYD_LAUNCHER_CLIENT_MANIFEST`. Nada é inventado: cada entrada
//! do ZIP precisa existir no manifesto e conferir tamanho + SHA-256.
//!
//! Depois de aplicar, o `serverlist.bin` é **sempre** regenerado a partir do
//! `canais.txt` do diretório (regra da entrega): o cliente não fica com uma lista
//! de canais velha depois de uma atualização.
//!
//! O download usa `curl` do sistema quando a origem é URL. É deliberado: o
//! launcher não carrega stack TLS própria, e `curl` existe no Windows 10+ e em
//! Linux/macOS. Falha de `curl` é erro tipado, nunca arquivo vazio aceito.
//!
//! ## Fonte, hash e cache
//!
//! Precedência da origem: argumento (`instalar <dir> <zip|URL>`) > `--zip` >
//! `WYD_LAUNCHER_CLIENT_ZIP`. O hash esperado do pacote vem de `--sha256` >
//! `WYD_LAUNCHER_CLIENT_SHA256`; URL remota **exige** hash (fail-closed).
//! Antes de qualquer requisição, o cache `<dir>/.launcher-cache/client-<sha>.zip`
//! é validado pelo hash; se bater, o launcher reusa sem rede. Download vai para
//! `*.part` no mesmo diretório, é verificado e só então renomeado — nunca deixa
//! arquivo parcial no lugar do pacote bom.

use std::fmt;
use std::fs;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::process::Command;

use flate2::read::DeflateDecoder;

use crate::manifest::{self, Manifest};
use crate::patch::{self, NewFile};
use crate::sha256;
use crate::{portal_client, serverlist};

/// Variável com a URL/caminho do `client.zip`.
pub const ZIP_ENV: &str = "WYD_LAUNCHER_CLIENT_ZIP";
/// Variável com a URL/caminho do manifesto (padrão: `<dir>/manifest.txt`).
pub const MANIFESTO_ENV: &str = "WYD_LAUNCHER_CLIENT_MANIFEST";
/// Variável com o SHA-256 esperado do `client.zip` (obrigatório para URL).
pub const ZIP_SHA256_ENV: &str = "WYD_LAUNCHER_CLIENT_SHA256";
/// Subdiretório do jogo que guarda o pacote baixado.
pub const CACHE_DIR: &str = ".launcher-cache";

/// Erros tipados da instalação do cliente.
#[derive(Debug)]
pub enum ClienteZipError {
    Io(String),
    Manifesto(manifest::ManifestError),
    /// ZIP fora do formato suportado (leiaute, zip64, criptografia).
    Zip(String),
    /// Arquivo do manifesto não está no ZIP.
    Ausente(String),
    /// Entrada do ZIP diverge do manifesto (nada é aplicado).
    Divergente(String),
    /// Não foi possível baixar a origem.
    Download(String),
    /// URL remota sem hash esperado (não há como validar integridade).
    HashAusente,
    /// Bytes baixados/lidos não conferem com o hash publicado.
    HashDivergente {
        esperado: String,
        obtido: String,
    },
    /// Pós-verificação do diretório falhou (backup restaurado).
    PosVerificacao(String),
    Serverlist(serverlist::ServerlistError),
}

impl fmt::Display for ClienteZipError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Io(e) => write!(f, "erro de E/S: {e}"),
            Self::Manifesto(e) => write!(f, "manifesto inválido: {e}"),
            Self::Zip(e) => write!(f, "client.zip inválido: {e}"),
            Self::Ausente(p) => write!(f, "arquivo do manifesto ausente no client.zip: {p}"),
            Self::Divergente(p) => write!(f, "entrada do client.zip diverge do manifesto: {p}"),
            Self::Download(e) => write!(f, "download do client.zip falhou: {e}"),
            Self::HashAusente => write!(
                f,
                "client.zip remoto exige hash: defina --sha256 ou {ZIP_SHA256_ENV}"
            ),
            Self::HashDivergente { esperado, obtido } => write!(
                f,
                "client.zip corrompido: SHA-256 esperado {esperado}, obtido {obtido}"
            ),
            Self::PosVerificacao(p) => {
                write!(f, "pós-verificação falhou, backup restaurado: {p}")
            }
            Self::Serverlist(e) => write!(f, "serverlist recusado: {e}"),
        }
    }
}

impl std::error::Error for ClienteZipError {}

impl From<manifest::ManifestError> for ClienteZipError {
    fn from(e: manifest::ManifestError) -> Self {
        Self::Manifesto(e)
    }
}

impl From<serverlist::ServerlistError> for ClienteZipError {
    fn from(e: serverlist::ServerlistError) -> Self {
        Self::Serverlist(e)
    }
}

/// Relatório de uma instalação/atualização concluída.
#[derive(Debug, Clone)]
pub struct RelatorioInstalacao {
    pub build: u32,
    pub arquivos: usize,
    pub backup: PathBuf,
    /// Bytes do `serverlist.bin` regenerado (sempre presente).
    pub serverlist_bytes: usize,
    /// Caminho da origem usada (URL ou arquivo).
    pub origem: String,
    /// SHA-256 real dos bytes instalados (igual ao esperado quando configurado).
    pub sha256: String,
    /// `true` quando o pacote veio do cache local (sem nova requisição).
    pub do_cache: bool,
}

/// Uma entrada de arquivo do ZIP (só o necessário para ler o payload).
struct EntradaZip {
    metodo: u16,
    comprimido: usize,
    tamanho: u64,
    inicio_dados: usize,
}

fn u16_le(b: &[u8], off: usize) -> Option<u16> {
    Some(u16::from_le_bytes(b.get(off..off + 2)?.try_into().ok()?))
}

fn u32_le(b: &[u8], off: usize) -> Option<u32> {
    Some(u32::from_le_bytes(b.get(off..off + 4)?.try_into().ok()?))
}

/// Lê o diretório central do ZIP em um mapa `nome → entrada`.
fn diretorio(zip: &[u8]) -> Result<Vec<(String, EntradaZip)>, ClienteZipError> {
    let eocd = (0..zip.len().saturating_sub(3))
        .rev()
        .find(|&i| zip[i..i + 4] == [0x50, 0x4b, 0x05, 0x06])
        .ok_or_else(|| ClienteZipError::Zip("fim do diretório central ausente".into()))?;
    let total = u16_le(zip, eocd + 10).ok_or_else(|| ClienteZipError::Zip("EOCD curto".into()))?;
    let mut off =
        u32_le(zip, eocd + 16).ok_or_else(|| ClienteZipError::Zip("EOCD curto".into()))? as usize;
    let mut saida = Vec::with_capacity(total as usize);
    for _ in 0..total {
        if zip.get(off..off + 4) != Some(&[0x50, 0x4b, 0x01, 0x02]) {
            return Err(ClienteZipError::Zip("cabeçalho central inválido".into()));
        }
        let flags =
            u16_le(zip, off + 8).ok_or_else(|| ClienteZipError::Zip("central curto".into()))?;
        if flags & 0x1 != 0 {
            return Err(ClienteZipError::Zip("entrada criptografada".into()));
        }
        let metodo =
            u16_le(zip, off + 10).ok_or_else(|| ClienteZipError::Zip("central curto".into()))?;
        let comprimido = u32_le(zip, off + 20)
            .ok_or_else(|| ClienteZipError::Zip("central curto".into()))?
            as usize;
        let tamanho = u32_le(zip, off + 24)
            .ok_or_else(|| ClienteZipError::Zip("central curto".into()))?
            as u64;
        let nome_len = u16_le(zip, off + 28)
            .ok_or_else(|| ClienteZipError::Zip("central curto".into()))?
            as usize;
        let extra_len = u16_le(zip, off + 30)
            .ok_or_else(|| ClienteZipError::Zip("central curto".into()))?
            as usize;
        let comentario_len = u16_le(zip, off + 32)
            .ok_or_else(|| ClienteZipError::Zip("central curto".into()))?
            as usize;
        let local = u32_le(zip, off + 42)
            .ok_or_else(|| ClienteZipError::Zip("central curto".into()))?
            as usize;
        let nome = zip
            .get(off + 46..off + 46 + nome_len)
            .ok_or_else(|| ClienteZipError::Zip("nome fora do arquivo".into()))?;
        let nome = String::from_utf8_lossy(nome).into_owned();
        // Cabeçalho local: nome/extra podem ter tamanhos diferentes do central.
        if zip.get(local..local + 4) != Some(&[0x50, 0x4b, 0x03, 0x04]) {
            return Err(ClienteZipError::Zip(format!(
                "cabeçalho local inválido: {nome}"
            )));
        }
        let nome_local = u16_le(zip, local + 26)
            .ok_or_else(|| ClienteZipError::Zip("local curto".into()))?
            as usize;
        let extra_local = u16_le(zip, local + 28)
            .ok_or_else(|| ClienteZipError::Zip("local curto".into()))?
            as usize;
        let inicio_dados = local + 30 + nome_local + extra_local;
        if inicio_dados + comprimido > zip.len() {
            return Err(ClienteZipError::Zip(format!("dados truncados: {nome}")));
        }
        if !nome.ends_with('/') {
            saida.push((
                nome,
                EntradaZip {
                    metodo,
                    comprimido,
                    tamanho,
                    inicio_dados,
                },
            ));
        }
        off += 46 + nome_len + extra_len + comentario_len;
    }
    Ok(saida)
}

/// Bytes descomprimidos de uma entrada (stored ou deflate).
fn conteudo(zip: &[u8], nome: &str, e: &EntradaZip) -> Result<Vec<u8>, ClienteZipError> {
    let bruto = zip
        .get(e.inicio_dados..e.inicio_dados + e.comprimido)
        .ok_or_else(|| ClienteZipError::Zip(format!("dados truncados: {nome}")))?;
    let bytes = match e.metodo {
        0 => bruto.to_vec(),
        8 => {
            let mut saida = Vec::with_capacity(e.tamanho as usize);
            DeflateDecoder::new(bruto)
                .read_to_end(&mut saida)
                .map_err(|erro| ClienteZipError::Zip(format!("deflate de {nome}: {erro}")))?;
            saida
        }
        outro => {
            return Err(ClienteZipError::Zip(format!(
                "método de compressão {outro} não suportado: {nome}"
            )))
        }
    };
    if bytes.len() as u64 != e.tamanho {
        return Err(ClienteZipError::Divergente(nome.to_string()));
    }
    Ok(bytes)
}

/// Normaliza e valida um SHA-256 publicado (64 dígitos hex).
pub fn normalizar_sha(texto: &str) -> Result<String, ClienteZipError> {
    let sha = texto.trim().to_ascii_lowercase();
    if sha.len() != 64 || !sha.bytes().all(|b| b.is_ascii_hexdigit()) {
        return Err(ClienteZipError::Download(format!(
            "hash SHA-256 inválido (esperado 64 hex): {texto}"
        )));
    }
    Ok(sha)
}

/// Caminho do pacote no cache local, nomeado pelo hash esperado.
#[must_use]
pub fn cache_path(dir: &Path, sha: &str) -> PathBuf {
    dir.join(CACHE_DIR).join(format!("client-{sha}.zip"))
}

fn conferir_hash(bytes: &[u8], esperado: &str) -> Result<(), ClienteZipError> {
    let obtido = sha256::hex_of(bytes);
    if obtido != esperado {
        return Err(ClienteZipError::HashDivergente {
            esperado: esperado.to_string(),
            obtido,
        });
    }
    Ok(())
}

/// Obtém o `client.zip`: cache validado → arquivo local → URL (gravada de forma atômica).
///
/// Devolve `(bytes, veio_do_cache)`. Se `esperado` existe, o cache em
/// `<dir>/.launcher-cache/client-<sha>.zip` é a primeira tentativa e não há nova
/// requisição quando o hash confere. URL sem hash é recusada antes da rede.
pub fn obter_zip(
    dir: &Path,
    origem: &str,
    esperado: Option<&str>,
) -> Result<(Vec<u8>, bool), ClienteZipError> {
    let esperado = esperado.map(normalizar_sha).transpose()?;
    let remoto = origem.starts_with("http://") || origem.starts_with("https://");
    // Cache só para fonte remota: um caminho local explícito é override e é lido do disco.
    if remoto {
        if let Some(sha) = esperado.as_deref() {
            let cache = cache_path(dir, sha);
            if let Ok(bytes) = fs::read(&cache) {
                if sha256::hex_of(&bytes) == sha {
                    return Ok((bytes, true));
                }
            }
        }
    }

    if !remoto {
        let bytes = fs::read(origem).map_err(|e| {
            ClienteZipError::Download(format!("arquivo local indisponível ({origem}): {e}"))
        })?;
        if let Some(sha) = esperado.as_deref() {
            conferir_hash(&bytes, sha)?;
        }
        return Ok((bytes, false));
    }

    let sha = esperado.as_deref().ok_or(ClienteZipError::HashAusente)?;
    let cache_dir = dir.join(CACHE_DIR);
    fs::create_dir_all(&cache_dir).map_err(|e| ClienteZipError::Io(e.to_string()))?;
    let temp = cache_dir.join(format!(
        "client-{}-{}.part",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_nanos())
            .unwrap_or(0)
    ));
    let saida = Command::new("curl")
        .args(["-fsSL", "--retry", "2", "--max-time", "600", "-o"])
        .arg(&temp)
        .arg(origem)
        .output()
        .map_err(|e| ClienteZipError::Download(format!("curl indisponível: {e}")))?;
    if !saida.status.success() {
        let _ = fs::remove_file(&temp);
        return Err(ClienteZipError::Download(format!(
            "fonte indisponível ({}): {}",
            saida.status,
            String::from_utf8_lossy(&saida.stderr).trim()
        )));
    }
    let bytes = match fs::read(&temp) {
        Ok(bytes) => bytes,
        Err(e) => {
            let _ = fs::remove_file(&temp);
            return Err(ClienteZipError::Io(e.to_string()));
        }
    };
    if let Err(e) = conferir_hash(&bytes, sha) {
        let _ = fs::remove_file(&temp);
        return Err(e);
    }
    let destino = cache_path(dir, sha);
    if destino.exists() {
        let _ = fs::remove_file(&destino);
    }
    if let Err(e) = fs::rename(&temp, &destino) {
        let _ = fs::remove_file(&temp);
        return Err(ClienteZipError::Io(format!(
            "gravação atômica do cache falhou: {e}"
        )));
    }
    Ok((bytes, false))
}

/// Carrega o manifesto do arquivo do diretório ou de uma origem remota/local.
pub fn carregar_manifesto(dir: &Path, origem: Option<&str>) -> Result<Manifest, ClienteZipError> {
    match origem {
        None | Some("") => Ok(manifest::load(
            &dir.join(crate::lancamento::MANIFESTO_PADRAO),
        )?),
        Some(origem) if origem.starts_with("http://") => {
            let (base, caminho) = dividir_url(origem)?;
            let api = portal_client::PortalApi::new(&base)
                .map_err(|e| ClienteZipError::Download(e.to_string()))?;
            let bytes = api
                .fetch_bytes(&caminho)
                .map_err(|e| ClienteZipError::Download(e.to_string()))?;
            let texto = String::from_utf8(bytes)
                .map_err(|_| ClienteZipError::Zip("manifesto não é UTF-8".into()))?;
            manifest::parse(&texto).map_err(ClienteZipError::Manifesto)
        }
        Some(origem) => {
            let texto =
                fs::read_to_string(origem).map_err(|e| ClienteZipError::Io(e.to_string()))?;
            manifest::parse(&texto).map_err(ClienteZipError::Manifesto)
        }
    }
}

/// `http://host[:porta]/caminho` → (`http://host[:porta]`, `/caminho`).
fn dividir_url(url: &str) -> Result<(String, String), ClienteZipError> {
    let resto = url.strip_prefix("http://").ok_or_else(|| {
        ClienteZipError::Download("só http:// é suportado no portal local".into())
    })?;
    match resto.find('/') {
        Some(pos) => Ok((
            format!("http://{}", &resto[..pos]),
            resto[pos..].to_string(),
        )),
        None => Err(ClienteZipError::Download("URL sem caminho".into())),
    }
}

/// Instala/atualiza o cliente: manifesto → ZIP → patch atômico → pós-verificação → serverlist.
///
/// Fail-closed: qualquer entrada divergente ou ausente recusa o pacote inteiro
/// antes de escrever; falha na pós-verificação restaura o backup.
pub fn instalar(
    dir: &Path,
    origem_zip: &str,
    origem_manifesto: Option<&str>,
    sha256_esperado: Option<&str>,
) -> Result<RelatorioInstalacao, ClienteZipError> {
    let m = carregar_manifesto(dir, origem_manifesto)?;
    let (zip, do_cache) = obter_zip(dir, origem_zip, sha256_esperado)?;
    let entradas = diretorio(&zip)?;
    let mut novos: Vec<NewFile> = Vec::with_capacity(m.files.len());
    for e in &m.files {
        let (nome, entrada) = entradas
            .iter()
            .find(|(nome, _)| nome == &e.path)
            .ok_or_else(|| ClienteZipError::Ausente(e.path.clone()))?;
        let bytes = conteudo(&zip, nome, entrada)?;
        if bytes.len() as u64 != e.size || sha256::digest(&bytes) != e.sha256 {
            return Err(ClienteZipError::Divergente(e.path.clone()));
        }
        novos.push(NewFile {
            path: e.path.clone(),
            size: e.size,
            sha256: e.sha256,
            contents: bytes,
        });
    }
    let backup = dir.with_extension(format!(
        "zip-bak-{}-{}",
        m.build,
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_secs())
            .unwrap_or(0)
    ));
    let arquivos =
        patch::apply(dir, &novos, &backup).map_err(|e| ClienteZipError::Io(e.to_string()))?;
    let rel = manifest::verify(&m, dir);
    if !rel.clean() {
        let primeiro = rel
            .failures()
            .next()
            .map(|r| r.path.clone())
            .unwrap_or_else(|| "?".into());
        patch::rollback(dir, &backup).ok();
        return Err(ClienteZipError::PosVerificacao(primeiro));
    }
    // Regra do launcher: `serverlist.bin` sempre regenerado do `canais.txt` do diretório.
    let canais = dir.join(crate::lancamento::REGISTRO_PADRAO);
    let bin = serverlist::generate_from_registry(&canais)?;
    let instalacao = serverlist::install(dir, &bin)?;
    Ok(RelatorioInstalacao {
        build: m.build,
        arquivos,
        backup,
        serverlist_bytes: instalacao.bytes,
        origem: origem_zip.to_string(),
        sha256: sha256::hex_of(&zip),
        do_cache,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    fn tmpdir(tag: &str) -> PathBuf {
        use std::sync::atomic::{AtomicU64, Ordering};
        static N: AtomicU64 = AtomicU64::new(0);
        let d = std::env::temp_dir().join(format!(
            "wyd-cliente-zip-{tag}-{}-{}",
            std::process::id(),
            N.fetch_add(1, Ordering::Relaxed)
        ));
        let _ = fs::remove_dir_all(&d);
        fs::create_dir_all(&d).unwrap();
        d
    }

    /// ZIP mínimo com entradas **stored** (método 0), suficiente para o leitor.
    fn zip_stored(arquivos: &[(&str, &[u8])]) -> Vec<u8> {
        let mut out = Vec::new();
        let mut central = Vec::new();
        for (nome, dados) in arquivos {
            let inicio = out.len();
            out.extend_from_slice(&[0x50, 0x4b, 0x03, 0x04]);
            out.extend_from_slice(&20u16.to_le_bytes());
            out.extend_from_slice(&0u16.to_le_bytes());
            out.extend_from_slice(&0u16.to_le_bytes()); // stored
            out.extend_from_slice(&0u32.to_le_bytes());
            out.extend_from_slice(&(crc32(dados)).to_le_bytes());
            out.extend_from_slice(&(dados.len() as u32).to_le_bytes());
            out.extend_from_slice(&(dados.len() as u32).to_le_bytes());
            out.extend_from_slice(&(nome.len() as u16).to_le_bytes());
            out.extend_from_slice(&0u16.to_le_bytes());
            out.extend_from_slice(nome.as_bytes());
            out.extend_from_slice(dados);

            central.extend_from_slice(&[0x50, 0x4b, 0x01, 0x02]);
            central.extend_from_slice(&20u16.to_le_bytes());
            central.extend_from_slice(&20u16.to_le_bytes());
            central.extend_from_slice(&0u16.to_le_bytes());
            central.extend_from_slice(&0u16.to_le_bytes()); // stored
            central.extend_from_slice(&0u32.to_le_bytes());
            central.extend_from_slice(&(crc32(dados)).to_le_bytes());
            central.extend_from_slice(&(dados.len() as u32).to_le_bytes());
            central.extend_from_slice(&(dados.len() as u32).to_le_bytes());
            central.extend_from_slice(&(nome.len() as u16).to_le_bytes());
            central.extend_from_slice(&0u16.to_le_bytes());
            central.extend_from_slice(&0u16.to_le_bytes());
            central.extend_from_slice(&0u16.to_le_bytes());
            central.extend_from_slice(&0u16.to_le_bytes());
            central.extend_from_slice(&0u32.to_le_bytes());
            central.extend_from_slice(&(inicio as u32).to_le_bytes());
            central.extend_from_slice(nome.as_bytes());
        }
        let inicio_central = out.len();
        out.extend_from_slice(&central);
        out.extend_from_slice(&[0x50, 0x4b, 0x05, 0x06]);
        out.extend_from_slice(&0u16.to_le_bytes());
        out.extend_from_slice(&0u16.to_le_bytes());
        out.extend_from_slice(&(arquivos.len() as u16).to_le_bytes());
        out.extend_from_slice(&(arquivos.len() as u16).to_le_bytes());
        out.extend_from_slice(&(central.len() as u32).to_le_bytes());
        out.extend_from_slice(&(inicio_central as u32).to_le_bytes());
        out.extend_from_slice(&0u16.to_le_bytes());
        out
    }

    fn crc32(dados: &[u8]) -> u32 {
        let mut crc = 0xFFFF_FFFFu32;
        for b in dados {
            crc ^= *b as u32;
            for _ in 0..8 {
                crc = (crc >> 1) ^ (0xEDB8_8320 & (crc & 1).wrapping_neg());
            }
        }
        !crc
    }

    fn preparar(tag: &str, conteudo: &[u8]) -> (PathBuf, Manifest) {
        let dir = tmpdir(tag);
        fs::write(dir.join("jogo.dat"), b"antigo").unwrap();
        fs::write(
            dir.join("canais.txt"),
            "# canais\n0 0 http://127.0.0.1/serv00.htm\n0 1 127.0.0.1\n",
        )
        .unwrap();
        let texto = format!(
            "build 7662\nprotocol 12000\n{} {} jogo.dat\n",
            sha256::hex_of(conteudo),
            conteudo.len()
        );
        fs::write(dir.join("manifest.txt"), &texto).unwrap();
        (dir, manifest::parse(&texto).unwrap())
    }

    #[test]
    fn instala_zip_e_sempre_regenera_serverlist() {
        let conteudo = b"cliente v2";
        let (dir, _) = preparar("ok", conteudo);
        let zip = zip_stored(&[
            ("jogo.dat", conteudo),
            ("ignorado.bin", b"nao esta no manifesto"),
        ]);
        let zpath = dir.join("client.zip");
        fs::write(&zpath, &zip).unwrap();
        let rel = instalar(&dir, zpath.to_str().unwrap(), None, None).unwrap();
        assert_eq!(rel.arquivos, 1);
        assert_eq!(rel.build, 7662);
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), conteudo);
        assert_eq!(
            fs::read(dir.join(serverlist::FILE_NAME)).unwrap().len(),
            serverlist::BIN_SIZE
        );
        assert!(
            !dir.join("ignorado.bin").exists(),
            "só o manifesto é instalado"
        );
    }

    #[test]
    fn entrada_divergente_recusa_antes_de_escrever() {
        let (dir, _) = preparar("tamper", b"v2");
        let zip = zip_stored(&[("jogo.dat", b"vX")]);
        let zpath = dir.join("client.zip");
        fs::write(&zpath, zip).unwrap();
        let erro = instalar(&dir, zpath.to_str().unwrap(), None, None).unwrap_err();
        assert!(matches!(erro, ClienteZipError::Divergente(_)), "{erro}");
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"antigo");
    }

    #[test]
    fn arquivo_do_manifesto_ausente_no_zip_e_erro() {
        let (dir, _) = preparar("falta", b"v2");
        let zip = zip_stored(&[("outro.dat", b"v2")]);
        let zpath = dir.join("client.zip");
        fs::write(&zpath, zip).unwrap();
        let erro = instalar(&dir, zpath.to_str().unwrap(), None, None).unwrap_err();
        assert!(matches!(erro, ClienteZipError::Ausente(_)), "{erro}");
    }

    #[test]
    fn zip_truncado_e_erro_tipado_sem_panic() {
        let (dir, _) = preparar("trunc", b"v2");
        let zpath = dir.join("client.zip");
        fs::write(&zpath, b"PK\x03\x04lixo").unwrap();
        assert!(instalar(&dir, zpath.to_str().unwrap(), None, None).is_err());
    }

    /// ZIP de uma entrada só, com o método informado (0 stored, 8 deflate).
    fn zip_uma_entrada(nome: &str, dados: &[u8], comprimido: &[u8], metodo: u16) -> Vec<u8> {
        let mut zip = Vec::new();
        zip.extend_from_slice(&[0x50, 0x4b, 0x03, 0x04]);
        zip.extend_from_slice(&20u16.to_le_bytes());
        zip.extend_from_slice(&0u16.to_le_bytes());
        zip.extend_from_slice(&metodo.to_le_bytes());
        zip.extend_from_slice(&0u16.to_le_bytes()); // hora
        zip.extend_from_slice(&0u16.to_le_bytes()); // data
        zip.extend_from_slice(&crc32(dados).to_le_bytes());
        zip.extend_from_slice(&(comprimido.len() as u32).to_le_bytes());
        zip.extend_from_slice(&(dados.len() as u32).to_le_bytes());
        zip.extend_from_slice(&(nome.len() as u16).to_le_bytes());
        zip.extend_from_slice(&0u16.to_le_bytes());
        zip.extend_from_slice(nome.as_bytes());
        zip.extend_from_slice(comprimido);
        let central = zip.len();
        zip.extend_from_slice(&[0x50, 0x4b, 0x01, 0x02]);
        zip.extend_from_slice(&20u16.to_le_bytes());
        zip.extend_from_slice(&20u16.to_le_bytes());
        zip.extend_from_slice(&0u16.to_le_bytes());
        zip.extend_from_slice(&metodo.to_le_bytes());
        zip.extend_from_slice(&0u16.to_le_bytes());
        zip.extend_from_slice(&0u16.to_le_bytes());
        zip.extend_from_slice(&crc32(dados).to_le_bytes());
        zip.extend_from_slice(&(comprimido.len() as u32).to_le_bytes());
        zip.extend_from_slice(&(dados.len() as u32).to_le_bytes());
        zip.extend_from_slice(&(nome.len() as u16).to_le_bytes());
        zip.extend_from_slice(&0u16.to_le_bytes()); // extra
        zip.extend_from_slice(&0u16.to_le_bytes()); // comentário
        zip.extend_from_slice(&0u16.to_le_bytes()); // disco
        zip.extend_from_slice(&0u16.to_le_bytes()); // atributos internos
        zip.extend_from_slice(&0u32.to_le_bytes()); // atributos externos
        zip.extend_from_slice(&0u32.to_le_bytes()); // offset local
        zip.extend_from_slice(nome.as_bytes());
        let central_len = zip.len() - central;
        zip.extend_from_slice(&[0x50, 0x4b, 0x05, 0x06]);
        zip.extend_from_slice(&0u16.to_le_bytes());
        zip.extend_from_slice(&0u16.to_le_bytes());
        zip.extend_from_slice(&1u16.to_le_bytes());
        zip.extend_from_slice(&1u16.to_le_bytes());
        zip.extend_from_slice(&(central_len as u32).to_le_bytes());
        zip.extend_from_slice(&(central as u32).to_le_bytes());
        zip.extend_from_slice(&0u16.to_le_bytes());
        zip
    }

    #[test]
    fn detecta_entrada_deflate() {
        // Prova o caminho deflate: comprime "v2-deflate" e monta o ZIP com método 8.
        use flate2::write::DeflateEncoder;
        let dados = b"v2-deflate";
        let mut enc = DeflateEncoder::new(Vec::new(), flate2::Compression::default());
        enc.write_all(dados).unwrap();
        let comprimido = enc.finish().unwrap();
        let (dir, _) = preparar("deflate", dados);
        let zpath = dir.join("client.zip");
        fs::write(&zpath, zip_uma_entrada("jogo.dat", dados, &comprimido, 8)).unwrap();
        let rel = instalar(&dir, zpath.to_str().unwrap(), None, None).unwrap();
        assert_eq!(rel.arquivos, 1);
        assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), dados);
    }
}
