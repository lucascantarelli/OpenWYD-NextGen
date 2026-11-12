//! Configuração remota assinada do launcher (CLIENT-11).
//!
//! A identidade embutida (`identidade.rs`, `include_str!`) continua sendo o padrão. Este
//! módulo permite que **marca, canais e URLs** venham do portal sem recompilar: o launcher
//! baixa `launcher.conf`, confere a assinatura e sobrepõe as chaves à identidade embutida.
//!
//! Formato (UTF-8, LF, `chave = valor`, `#` comenta), mesma disciplina do manifesto:
//!
//! ```text
//! # wyd-launcher-config v1
//! revisao = 7
//! portal.base = http://127.0.0.1:3000
//! launcher.manifesto = /api/launcher/manifesto
//! launcher.pacote = /api/launcher/pacote
//! cor.accent = #ff0000
//! servidor.2 = Novo
//! assinatura = <64 hex>
//! ```
//!
//! Regras que o parser faz valer:
//!
//! 1. `assinatura` é a **última linha com conteúdo** e o HMAC-SHA256 cobre os bytes exatos
//!    anteriores a ela — texto depois da assinatura seria texto não assinado, então é recusa;
//! 2. o HMAC usa a **chave simétrica da configuração** (`--key-file`, nunca embutida no
//!    binário) — o pacote de patch é outro domínio e usa Ed25519 (`distribuicao`);
//! 3. chave desconhecida, `revisao` ausente, `portal.base` fora de `http://host[:porta]` e
//!    caminho sem `/` reprovam **com o número da linha** — config errada não vira marca
//!    estranha silenciosa;
//! 4. a sobreposição passa pelo parser estrito da identidade (`Identidade::parse`) **e**
//!    pela validação de consumo (`validar`), então nome de canal acima de 8 caracteres,
//!    cor fora de `#rrggbb` e slot fora de 1..=10 reprovam no merge — não na skin;
//! 5. instalação atômica com `.bak` (`branding::install`): config meio escrita é pior que
//!    config velha;
//! 6. a janela resolve a identidade efetiva por `resolver`: arquivo ausente, chave ausente,
//!    assinatura inválida ou valor fora do contrato **caem para a embutida com motivo
//!    visível** — nunca `panic`, nunca tela muda, nunca config parcial.

use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};

use crate::branding;
use crate::identidade::{Identidade, IdentidadeError};
use crate::portal_client::PortalApi;
use crate::sha256;

/// Nome do arquivo instalado no diretório do jogo.
pub const CONF_NAME: &str = "launcher.conf";

/// Variável de ambiente com o caminho da chave usada pela janela para conferir a config.
pub const CHAVE_ENV: &str = "WYD_LAUNCHER_CONFIG_KEY";

/// Cabeçalho informativo da primeira linha (não validado, como no manifesto).
pub const FORMAT_V1: &str = "wyd-launcher-config v1";

/// Caminho de manifesto usado quando a config não traz `launcher.manifesto`.
pub const MANIFESTO_PADRAO: &str = "/api/launcher/manifesto";

/// Caminho de pacote usado quando a config não traz `launcher.pacote`.
pub const PACOTE_PADRAO: &str = "/api/launcher/pacote";

/// Chave reservada da assinatura.
const ASSINATURA: &str = "assinatura";

/// Teto de caminho/URL aceito (nada do contrato chega perto disto).
const URL_MAX: usize = 256;

/// URLs públicas que o launcher consome do portal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PortalRemoto {
    /// Base `http://host[:porta]` (mesma regra do `portal_client`).
    pub base: String,
    /// Caminho do manifesto de distribuição.
    pub manifesto: String,
    /// Caminho do pacote do launcher.
    pub pacote: String,
}

impl PortalRemoto {
    /// URL absoluta de um caminho do portal (base sem `/` final + caminho).
    #[must_use]
    pub fn url(&self, caminho: &str) -> String {
        format!("{}{caminho}", self.base.trim_end_matches('/'))
    }
}

/// Configuração remota já verificada.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Configuracao {
    /// Revisão publicada (serve de cache-buster e de diagnóstico).
    pub revisao: u64,
    /// URLs do portal.
    pub portal: PortalRemoto,
    sobreposicoes: Vec<(String, String)>,
}

impl Configuracao {
    /// Chaves de identidade que a config remota sobrepõe, na ordem do arquivo.
    #[must_use]
    pub fn sobreposicoes(&self) -> &[(String, String)] {
        &self.sobreposicoes
    }

    /// Aplica as sobreposições sobre a identidade embutida.
    ///
    /// A validação final é a do `Identidade::parse`: nada aqui afrouxa as regras de cor,
    /// número ou nome de canal que o resto do launcher já obedece.
    pub fn mesclar(&self, base: &Identidade) -> Result<Identidade, IdentidadeError> {
        let mut pares: Vec<(String, String)> = base.pares().to_vec();
        for (chave, valor) in &self.sobreposicoes {
            match pares
                .iter_mut()
                .find(|(existente, _)| existente.as_str() == chave.as_str())
            {
                Some((_, atual)) => *atual = valor.clone(),
                None => pares.push((chave.clone(), valor.clone())),
            }
        }
        let mut texto = String::new();
        for (chave, valor) in &pares {
            texto.push_str(chave);
            texto.push_str(" = ");
            texto.push_str(valor);
            texto.push('\n');
        }
        let id = Identidade::parse(&texto)?;
        validar(&id)?;
        Ok(id)
    }
}

/// De onde saiu a identidade que a janela está desenhando.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Origem {
    /// `launcher.conf` verificado e aplicado.
    Remota {
        /// Revisão publicada.
        revisao: u64,
    },
    /// Identidade embutida no binário (`include_str!`).
    Embutida,
}

/// Identidade efetiva da janela: marca/canais, URLs do portal e a origem.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Efetiva {
    /// Identidade já mesclada e validada — a única que a janela desenha.
    pub identidade: Identidade,
    /// URLs do portal quando a config remota foi aceita.
    pub portal: Option<PortalRemoto>,
    /// Origem (observável na janela e no log do CLI).
    pub origem: Origem,
    /// Motivo do fallback; `None` quando a config remota foi aplicada.
    pub motivo: Option<String>,
}

impl Efetiva {
    /// Identidade embutida com o motivo do fallback (o caso "sem config confiável").
    pub fn embutida(motivo: impl Into<String>) -> Result<Self, IdentidadeError> {
        Ok(Self {
            identidade: crate::identidade::padrao()?,
            portal: None,
            origem: Origem::Embutida,
            motivo: Some(motivo.into()),
        })
    }

    /// Identidade remota já mesclada (quem autenticou os bytes foi o `verificar`).
    #[must_use]
    pub fn remota(identidade: Identidade, portal: PortalRemoto, revisao: u64) -> Self {
        Self {
            identidade,
            portal: Some(portal),
            origem: Origem::Remota { revisao },
            motivo: None,
        }
    }
}

/// Resolve a identidade efetiva exatamente como a CLI e a janela Tauri (CLIENT-08).
///
/// `config` é o caminho de operador e é **estrito** (assinatura inválida é erro, não
/// fallback). Sem ele, resolve `<dir>/launcher.conf` com `key_file` e cai para a
/// identidade embutida com motivo observável.
pub fn resolver_flags(
    dir: &Path,
    config: Option<&str>,
    key_file: Option<&str>,
) -> Result<Efetiva, String> {
    match config {
        Some(config) => {
            let Some(key_file) = key_file else {
                return Err(format!(
                    "--config exige --key-file (ou {} com o caminho da chave)",
                    CHAVE_ENV
                ));
            };
            let chave =
                carregar_chave(Path::new(key_file)).map_err(|e| format!("chave inválida: {e}"))?;
            let texto =
                ler(Path::new(config)).map_err(|e| format!("configuração ilegível: {e}"))?;
            let remota =
                verificar(&texto, &chave).map_err(|e| format!("configuração recusada: {e}"))?;
            let base =
                crate::identidade::padrao().map_err(|e| format!("identidade inválida: {e}"))?;
            let id = remota
                .mesclar(&base)
                .map_err(|e| format!("identidade remota inválida: {e}"))?;
            Ok(Efetiva::remota(id, remota.portal, remota.revisao))
        }
        None => {
            let chave = match key_file {
                Some(arquivo) => Some(
                    carregar_chave(Path::new(arquivo))
                        .map_err(|e| format!("chave inválida: {e}"))?,
                ),
                None => None,
            };
            resolver(dir, chave.as_deref())
                .map_err(|e| format!("identidade embutida inválida: {e}"))
        }
    }
}

/// Valida o que os consumidores exigem, para a config não passar e quebrar depois.
///
/// `Identidade::parse` valida sintaxe; cor, número e nome de canal só são
/// checados na leitura. Aqui a checagem é antecipada: config remota inválida é
/// recusada no `config`, não na hora de pintar a skin ou montar o `sn.bin`.
fn validar(id: &Identidade) -> Result<(), IdentidadeError> {
    for (chave, _) in id.pares() {
        if chave.starts_with("cor.") {
            id.cor(chave)?;
        } else if chave == "tipografia.altura" {
            id.decimal(chave)?;
        } else if chave == "tipografia.familia" || chave == "tipografia.mono" {
            id.get(chave)?;
        } else if chave == "tipografia.tamanho"
            || chave == "tipografia.titulo"
            || chave == "tipografia.legenda"
            || chave == "raio"
            || chave == "raio.pequeno"
            || chave.starts_with("espaco.")
        {
            id.numero(chave)?;
        }
    }
    id.servidores()?;
    Ok(())
}

/// Recusas da configuração remota.
#[derive(Debug)]
pub enum ConfigError {
    /// Arquivo ilegível.
    Io(io::Error),
    /// Linha que não é `chave = valor`, ou chave fora do alfabeto.
    BadLine { line: usize, reason: &'static str },
    /// Chave repetida.
    DuplicateKey { line: usize, key: String },
    /// Chave que o launcher não conhece (provável erro de digitação).
    UnknownKey { line: usize, key: String },
    /// Chave obrigatória ausente.
    Missing { key: &'static str },
    /// Valor fora do contrato.
    BadValue { key: String, value: String },
    /// Sem linha `assinatura`.
    Unsigned,
    /// Conteúdo depois da assinatura (não seria coberto pelo HMAC).
    SignatureNotLast { line: usize },
    /// HMAC não confere.
    BadSignature,
    /// Arquivo da chave HMAC não existe.
    ChaveAusente(PathBuf),
    /// Sobreposição inválida para a identidade única.
    Identidade(IdentidadeError),
}

impl fmt::Display for ConfigError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Io(e) => write!(f, "erro de E/S: {e}"),
            Self::BadLine { line, reason } => write!(f, "configuração linha {line}: {reason}"),
            Self::DuplicateKey { line, key } => {
                write!(f, "configuração linha {line}: chave `{key}` repetida")
            }
            Self::UnknownKey { line, key } => {
                write!(f, "configuração linha {line}: chave `{key}` desconhecida")
            }
            Self::Missing { key } => write!(f, "configuração sem a chave `{key}`"),
            Self::BadValue { key, value } => {
                write!(f, "configuração `{key}`: valor `{value}` fora do contrato")
            }
            Self::Unsigned => write!(f, "configuração sem assinatura (recusada)"),
            Self::SignatureNotLast { line } => write!(
                f,
                "configuração linha {line}: conteúdo depois da assinatura não é coberto por ela"
            ),
            Self::BadSignature => write!(f, "assinatura da configuração inválida (recusada)"),
            Self::ChaveAusente(p) => {
                write!(f, "arquivo de chave não encontrado: {}", p.display())
            }
            Self::Identidade(e) => write!(f, "identidade remota inválida: {e}"),
        }
    }
}

impl std::error::Error for ConfigError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Io(e) => Some(e),
            Self::Identidade(e) => Some(e),
            _ => None,
        }
    }
}

struct Analisada {
    revisao: Option<u64>,
    base: Option<String>,
    manifesto: Option<String>,
    pacote: Option<String>,
    sobreposicoes: Vec<(String, String)>,
    inicio: Option<usize>,
    assinatura: Option<[u8; 32]>,
}

/// Chaves de identidade que a config pode sobrepor (as demais são do contrato).
fn chave_de_identidade(chave: &str) -> bool {
    matches!(chave, "raio" | "raio.pequeno")
        || chave.starts_with("cor.")
        || chave.starts_with("espaco.")
        || chave.starts_with("tipografia.")
        || chave.starts_with("servidor.")
}

/// Caminho público aceito: absoluto, sem espaço, sem barra invertida e curto.
fn caminho_valido(valor: &str) -> bool {
    valor.starts_with('/')
        && valor.len() <= URL_MAX
        && !valor.contains(char::is_whitespace)
        && !valor.contains('\\')
        && !valor.contains('"')
}

/// Comparação em tempo constante (mesmo número de operações com 1 ou 32 bytes certos).
fn iguais(a: &[u8; 32], b: &[u8; 32]) -> bool {
    let mut diferenca = 0u8;
    for (x, y) in a.iter().zip(b.iter()) {
        diferenca |= x ^ y;
    }
    diferenca == 0
}

/// HMAC-SHA256 (RFC 2104) sobre `sha256` em `std` puro.
///
/// Assina a configuração remota; a chave é simétrica e própria deste domínio (o
/// pacote de patch não usa HMAC: usa Ed25519 com chave pública embutida).
#[must_use]
pub fn hmac_sha256(key: &[u8], msg: &[u8]) -> [u8; 32] {
    const BLOCO: usize = 64;
    let mut kpad = [0u8; BLOCO];
    if key.len() > BLOCO {
        let d = crate::sha256::digest(key);
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
    let h_dentro = crate::sha256::digest(&dentro);
    let mut fora = Vec::with_capacity(BLOCO + 32);
    fora.extend_from_slice(&opad);
    fora.extend_from_slice(&h_dentro);
    crate::sha256::digest(&fora)
}

/// Lê a chave HMAC da configuração (bytes crus, sem CR/LF final).
///
/// É a chave simétrica exigida por `verificar` (`--key-file` ou
/// `WYD_LAUNCHER_CONFIG_KEY`), separada da chave Ed25519 do pacote.
pub fn carregar_chave(path: &Path) -> Result<Vec<u8>, ConfigError> {
    if !path.exists() {
        return Err(ConfigError::ChaveAusente(path.to_path_buf()));
    }
    let mut k = fs::read(path).map_err(ConfigError::Io)?;
    while k.last().is_some_and(|b| *b == b'\n' || *b == b'\r') {
        k.pop();
    }
    Ok(k)
}

/// Analisa o texto sem autenticar (uso interno do `verificar`).
fn analisar(texto: &str) -> Result<Analisada, ConfigError> {
    let mut revisao: Option<u64> = None;
    let mut base: Option<String> = None;
    let mut manifesto: Option<String> = None;
    let mut pacote: Option<String> = None;
    let mut sobreposicoes: Vec<(String, String)> = Vec::new();
    let mut vistas: Vec<String> = Vec::new();
    let mut inicio: Option<usize> = None;
    let mut assinatura: Option<[u8; 32]> = None;
    let mut depois_da_assinatura = false;
    let mut offset = 0usize;

    for (indice, bruta) in texto.split_inclusive('\n').enumerate() {
        let line = indice + 1;
        let inicio_linha = offset;
        offset += bruta.len();
        let limpa = bruta.trim_end_matches(['\r', '\n']).trim();
        if limpa.is_empty() {
            continue;
        }
        if depois_da_assinatura {
            return Err(ConfigError::SignatureNotLast { line });
        }
        if limpa.starts_with('#') {
            continue;
        }
        let Some((chave, valor)) = limpa.split_once('=') else {
            return Err(ConfigError::BadLine {
                line,
                reason: "esperado `chave = valor`",
            });
        };
        let chave = chave.trim();
        let valor = valor.trim();
        if chave.is_empty() || valor.is_empty() {
            return Err(ConfigError::BadLine {
                line,
                reason: "chave ou valor vazio",
            });
        }
        if !chave
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'.' || b == b'-')
        {
            return Err(ConfigError::BadLine {
                line,
                reason: "chave fora de [a-z0-9.-]",
            });
        }
        if vistas.iter().any(|vista| vista.as_str() == chave) {
            return Err(ConfigError::DuplicateKey {
                line,
                key: chave.to_string(),
            });
        }
        vistas.push(chave.to_string());

        if chave == ASSINATURA {
            inicio = Some(inicio_linha);
            assinatura = Some(sha256::parse_hex(valor).ok_or(ConfigError::BadSignature)?);
            depois_da_assinatura = true;
            continue;
        }

        match chave {
            "revisao" => {
                revisao = Some(valor.parse().map_err(|_| ConfigError::BadValue {
                    key: chave.to_string(),
                    value: valor.to_string(),
                })?);
            }
            "portal.base" => {
                PortalApi::new(valor).map_err(|_| ConfigError::BadValue {
                    key: chave.to_string(),
                    value: valor.to_string(),
                })?;
                base = Some(valor.to_string());
            }
            "launcher.manifesto" => {
                if !caminho_valido(valor) {
                    return Err(ConfigError::BadValue {
                        key: chave.to_string(),
                        value: valor.to_string(),
                    });
                }
                manifesto = Some(valor.to_string());
            }
            "launcher.pacote" => {
                if !caminho_valido(valor) {
                    return Err(ConfigError::BadValue {
                        key: chave.to_string(),
                        value: valor.to_string(),
                    });
                }
                pacote = Some(valor.to_string());
            }
            outra if chave_de_identidade(outra) => {
                sobreposicoes.push((outra.to_string(), valor.to_string()));
            }
            outra => {
                return Err(ConfigError::UnknownKey {
                    line,
                    key: outra.to_string(),
                });
            }
        }
    }

    Ok(Analisada {
        revisao,
        base,
        manifesto,
        pacote,
        sobreposicoes,
        inicio,
        assinatura,
    })
}

/// Verifica a assinatura e devolve a configuração autenticada.
///
/// A chave é a **simétrica da configuração** (`carregar_chave` + `--key-file`), distinta da
/// chave Ed25519 que assina o pacote de patch: são domínios diferentes e a config pode ser
/// re-assinada pelo operador sem tocar no par de distribuição (ADR-0003).
pub fn verificar(texto: &str, chave: &[u8]) -> Result<Configuracao, ConfigError> {
    let analisada = analisar(texto)?;
    let inicio = analisada.inicio.ok_or(ConfigError::Unsigned)?;
    let assinatura = analisada.assinatura.ok_or(ConfigError::BadSignature)?;
    if !iguais(
        &hmac_sha256(chave, &texto.as_bytes()[..inicio]),
        &assinatura,
    ) {
        return Err(ConfigError::BadSignature);
    }
    Ok(Configuracao {
        revisao: analisada
            .revisao
            .ok_or(ConfigError::Missing { key: "revisao" })?,
        portal: PortalRemoto {
            base: analisada
                .base
                .ok_or(ConfigError::Missing { key: "portal.base" })?,
            manifesto: analisada
                .manifesto
                .unwrap_or_else(|| MANIFESTO_PADRAO.to_string()),
            pacote: analisada
                .pacote
                .unwrap_or_else(|| PACOTE_PADRAO.to_string()),
        },
        sobreposicoes: analisada.sobreposicoes,
    })
}

/// Lê o `launcher.conf` instalado (texto cru: quem autentica é `verificar`).
pub fn ler(arquivo: &Path) -> Result<String, ConfigError> {
    fs::read_to_string(arquivo).map_err(ConfigError::Io)
}

/// Instala o texto assinado em `dir`, de forma atômica e com backup.
pub fn install(dir: &Path, texto: &str) -> Result<branding::InstallReport, ConfigError> {
    let mut relatorios =
        branding::install(dir, &[(CONF_NAME, texto.as_bytes())]).map_err(ConfigError::Io)?;
    relatorios
        .pop()
        .ok_or_else(|| ConfigError::Io(io::Error::other("instalação sem relatório")))
}

/// Resolve a identidade efetiva do diretório do jogo — sem `panic` e com motivo.
///
/// Ordem: `dir/launcher.conf` existe? → chave disponível? → assinatura confere? →
/// merge/validação passam? Qualquer resposta negativa devolve a **identidade embutida**
/// com motivo legível, que é o que a janela mostra no rodapé. Config parcial nunca é
/// aplicada: ou o arquivo inteiro vale, ou nada dele vale.
pub fn resolver(dir: &Path, chave: Option<&[u8]>) -> Result<Efetiva, IdentidadeError> {
    let arquivo = dir.join(CONF_NAME);
    let texto = match fs::read_to_string(&arquivo) {
        Ok(texto) => texto,
        Err(e) if e.kind() == io::ErrorKind::NotFound => {
            return Efetiva::embutida(format!("{CONF_NAME} ausente em {}", dir.display()));
        }
        Err(e) => return Efetiva::embutida(format!("{CONF_NAME} ilegível: {e}")),
    };
    let Some(chave) = chave else {
        return Efetiva::embutida(format!(
            "{CONF_NAME} sem chave para conferir ({CHAVE_ENV} ou --key-file)"
        ));
    };
    let remota = match verificar(&texto, chave) {
        Ok(remota) => remota,
        Err(e) => return Efetiva::embutida(format!("{CONF_NAME} recusado: {e}")),
    };
    let base = crate::identidade::padrao()?;
    let revisao = remota.revisao;
    let portal = remota.portal.clone();
    match remota.mesclar(&base) {
        Ok(identidade) => Ok(Efetiva::remota(identidade, portal, revisao)),
        Err(e) => Efetiva::embutida(format!("{CONF_NAME} recusado: {e}")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identidade::padrao;
    use std::path::PathBuf;
    use std::sync::atomic::{AtomicU64, Ordering};

    const CHAVE: &[u8] = b"chave-de-teste-config";

    fn corpo() -> String {
        "# wyd-launcher-config v1\n\
         revisao = 3\n\
         portal.base = http://127.0.0.1:3000\n\
         cor.accent = #ff0000\n\
         servidor.2 = Novo\n"
            .to_string()
    }

    /// Hex de um resumo já calculado (`sha256::hex_of` resumiria a entrada de novo).
    fn hex32(digest: &[u8; 32]) -> String {
        const HEX: &[u8; 16] = b"0123456789abcdef";
        let mut s = String::with_capacity(64);
        for b in digest {
            s.push(HEX[(b >> 4) as usize] as char);
            s.push(HEX[(b & 15) as usize] as char);
        }
        s
    }

    /// Corpo + linha de assinatura calculada como o publicador (`sign-config.mjs`) faz.
    fn assinar(corpo: &str) -> String {
        let digest = hmac_sha256(CHAVE, corpo.as_bytes());
        format!("{corpo}assinatura = {}\n", hex32(&digest))
    }

    #[test]
    fn configuracao_assinada_traz_revisao_urls_e_marca() {
        let cfg = verificar(&assinar(&corpo()), CHAVE).expect("config válida");
        assert_eq!(cfg.revisao, 3);
        assert_eq!(cfg.portal.base, "http://127.0.0.1:3000");
        assert_eq!(cfg.portal.manifesto, MANIFESTO_PADRAO);
        assert_eq!(cfg.portal.pacote, PACOTE_PADRAO);
        assert_eq!(cfg.sobreposicoes().len(), 2);

        let id = cfg.mesclar(&padrao().expect("identidade")).expect("merge");
        assert_eq!(id.cor("cor.accent").expect("acento").hex(), "#ff0000");
        let servidores = id.servidores().expect("servidores");
        assert!(servidores.contains(&(2, "Novo".to_string())));
        assert!(
            servidores.iter().any(|(slot, _)| *slot == 1),
            "sobrepor um slot não apaga os demais"
        );
    }

    #[test]
    fn caminhos_personalizados_substituem_os_padroes() {
        let texto = "# wyd-launcher-config v1\n\
                     revisao = 9\n\
                     portal.base = http://127.0.0.1:8080\n\
                     launcher.manifesto = /dist/manifest.txt\n\
                     launcher.pacote = /dist/wyd-launcher.exe\n";
        let cfg = verificar(&assinar(texto), CHAVE).expect("config válida");
        assert_eq!(cfg.portal.manifesto, "/dist/manifest.txt");
        assert_eq!(cfg.portal.pacote, "/dist/wyd-launcher.exe");
    }

    #[test]
    fn assinatura_de_outra_chave_recusa() {
        let texto = assinar(&corpo());
        assert!(matches!(
            verificar(&texto, b"outra-chave"),
            Err(ConfigError::BadSignature)
        ));
    }

    #[test]
    fn corpo_alterado_depois_de_assinar_recusa() {
        let texto = assinar(&corpo()).replace("revisao = 3", "revisao = 4");
        assert!(matches!(
            verificar(&texto, CHAVE),
            Err(ConfigError::BadSignature)
        ));
    }

    #[test]
    fn sem_assinatura_recusa() {
        assert!(matches!(
            verificar(&corpo(), CHAVE),
            Err(ConfigError::Unsigned)
        ));
    }

    #[test]
    fn conteudo_depois_da_assinatura_recusa() {
        let texto = format!("{}cor.bg = #000000\n", assinar(&corpo()));
        assert!(matches!(
            verificar(&texto, CHAVE),
            Err(ConfigError::SignatureNotLast { .. })
        ));
    }

    #[test]
    fn chave_desconhecida_recusa_com_linha() {
        let texto = "# wyd-launcher-config v1\nrevisao = 1\nportal.base = http://127.0.0.1:3000\n\
                     portal.bse = http://127.0.0.1:3000\n";
        match verificar(&assinar(texto), CHAVE) {
            Err(ConfigError::UnknownKey { line, key }) => {
                assert_eq!(line, 4);
                assert_eq!(key, "portal.bse");
            }
            outro => panic!("esperava chave desconhecida, veio {outro:?}"),
        }
    }

    #[test]
    fn revisao_ausente_recusa() {
        let texto = "# wyd-launcher-config v1\nportal.base = http://127.0.0.1:3000\n";
        assert!(matches!(
            verificar(&assinar(texto), CHAVE),
            Err(ConfigError::Missing { key: "revisao" })
        ));
    }

    #[test]
    fn base_https_ou_caminho_ruim_reprovam() {
        for ruim in [
            "portal.base = https://127.0.0.1:3000",
            "portal.base = 127.0.0.1:3000",
            "launcher.pacote = api/launcher/pacote",
            "launcher.manifesto = /api/with space",
        ] {
            let texto = format!("# wyd-launcher-config v1\nrevisao = 1\n{ruim}\n");
            assert!(
                matches!(
                    verificar(&assinar(&texto), CHAVE),
                    Err(ConfigError::BadValue { .. })
                ),
                "esperava recusa para `{ruim}`"
            );
        }
    }

    #[test]
    fn chave_repetida_recusa() {
        let texto = "# wyd-launcher-config v1\nrevisao = 1\nrevisao = 2\n\
                     portal.base = http://127.0.0.1:3000\n";
        assert!(matches!(
            verificar(&assinar(texto), CHAVE),
            Err(ConfigError::DuplicateKey { line: 3, .. })
        ));
    }

    #[test]
    fn sobreposicao_invalida_nao_quebra_a_identidade_boa() {
        // Assinada corretamente, mas fora do que o `sn.bin` aceita (8 caracteres).
        let texto = "# wyd-launcher-config v1\nrevisao = 1\n\
                     portal.base = http://127.0.0.1:3000\nservidor.1 = NoveChars\n";
        let cfg = verificar(&assinar(texto), CHAVE).expect("assinatura boa");
        assert!(
            cfg.mesclar(&padrao().expect("identidade")).is_err(),
            "nome de 9 caracteres não pode passar pelo merge"
        );
    }

    fn tmpdir(tag: &str) -> PathBuf {
        static N: AtomicU64 = AtomicU64::new(9000);
        let d = std::env::temp_dir().join(format!(
            "wyd-launcher-conf-{tag}-{}-{}",
            std::process::id(),
            N.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(&d).expect("tmp");
        d
    }

    #[test]
    fn instala_com_backup_e_nao_reescreve_igual() {
        let dir = tmpdir("install");
        let primeiro = assinar(&corpo());
        let r1 = install(&dir, &primeiro).expect("instala");
        assert!(
            r1.changed && r1.backup.is_none(),
            "primeira escrita sem backup"
        );
        assert_eq!(
            fs::read_to_string(dir.join(CONF_NAME)).expect("lê"),
            primeiro
        );

        let r2 = install(&dir, &primeiro).expect("reinstala");
        assert!(!r2.changed, "conteúdo idêntico não é reescrito");

        let segundo = assinar(&corpo().replace("revisao = 3", "revisao = 4"));
        let r3 = install(&dir, &segundo).expect("troca");
        assert!(r3.changed && r3.backup.is_some(), "troca guarda backup");
        assert_eq!(
            fs::read_to_string(dir.join(CONF_NAME)).expect("lê"),
            segundo
        );
        let backup = fs::read_to_string(dir.join(format!("{CONF_NAME}.bak"))).expect("bak");
        assert_eq!(backup, primeiro);
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn ler_arquivo_ausente_e_erro_tipado() {
        let erro = ler(Path::new("launcher-que-nao-existe.conf")).expect_err("falha");
        assert!(matches!(erro, ConfigError::Io(_)));
    }

    /// Instala a config de teste assinada em `dir` (o mesmo que o `config` faz).
    fn instalar_assinado(dir: &Path) {
        install(dir, &assinar(&corpo())).expect("instala");
    }

    #[test]
    fn resolver_aplica_a_config_remota_com_urls() {
        let dir = tmpdir("resolver-remoto");
        instalar_assinado(&dir);
        let efetiva = resolver(&dir, Some(CHAVE)).expect("resolve");
        assert!(matches!(efetiva.origem, Origem::Remota { revisao: 3 }));
        assert_eq!(efetiva.motivo, None);
        assert_eq!(
            efetiva.identidade.cor("cor.accent").expect("acento").hex(),
            "#ff0000"
        );
        assert!(efetiva
            .identidade
            .servidores()
            .expect("canais")
            .contains(&(2, "Novo".to_string())));
        let portal = efetiva.portal.expect("portal remoto");
        assert_eq!(
            portal.url(MANIFESTO_PADRAO),
            "http://127.0.0.1:3000/api/launcher/manifesto"
        );
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn resolver_sem_arquivo_cai_para_a_embutida_com_motivo() {
        let dir = tmpdir("resolver-ausente");
        let efetiva = resolver(&dir, Some(CHAVE)).expect("resolve");
        assert_eq!(efetiva.origem, Origem::Embutida);
        assert!(efetiva.portal.is_none());
        assert!(efetiva.motivo.as_deref().unwrap_or("").contains("ausente"));
        assert_eq!(efetiva.identidade, padrao().expect("identidade"));
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn resolver_sem_chave_nao_confia_no_arquivo() {
        let dir = tmpdir("resolver-sem-chave");
        instalar_assinado(&dir);
        let efetiva = resolver(&dir, None).expect("resolve");
        assert_eq!(efetiva.origem, Origem::Embutida);
        assert!(efetiva.motivo.as_deref().unwrap_or("").contains("chave"));
        assert_eq!(
            efetiva.identidade.cor("cor.accent").expect("acento").hex(),
            "#c8a24a"
        );
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn resolver_recusa_arquivo_adulterado() {
        let dir = tmpdir("resolver-adulterado");
        instalar_assinado(&dir);
        let caminho = dir.join(CONF_NAME);
        let texto = fs::read_to_string(&caminho)
            .expect("lê")
            .replace("#ff0000", "#00ff00");
        fs::write(&caminho, texto).expect("escreve");
        let efetiva = resolver(&dir, Some(CHAVE)).expect("resolve");
        assert_eq!(efetiva.origem, Origem::Embutida);
        assert!(efetiva
            .motivo
            .as_deref()
            .unwrap_or("")
            .contains("assinatura"));
        assert!(efetiva.portal.is_none());
        assert_eq!(efetiva.identidade, padrao().expect("identidade"));
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn resolver_recusa_config_assinada_mas_fora_do_contrato() {
        let dir = tmpdir("resolver-contrato");
        let texto = "# wyd-launcher-config v1\nrevisao = 1\n\
                     portal.base = http://127.0.0.1:3000\ncor.accent = azul\n";
        fs::write(dir.join(CONF_NAME), assinar(texto)).expect("escreve");
        let efetiva = resolver(&dir, Some(CHAVE)).expect("resolve");
        assert_eq!(efetiva.origem, Origem::Embutida);
        assert!(efetiva.motivo.as_deref().unwrap_or("").contains("recusado"));
        assert_eq!(efetiva.identidade, padrao().expect("identidade"));
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn resolver_nao_entra_em_panico_com_arquivo_binario() {
        let dir = tmpdir("resolver-binario");
        fs::write(dir.join(CONF_NAME), [0xff, 0xfe, 0x00, 0x01]).expect("escreve");
        let efetiva = resolver(&dir, Some(CHAVE)).expect("resolve");
        assert_eq!(efetiva.origem, Origem::Embutida);
        assert!(efetiva.motivo.is_some());
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn janela_desenha_marca_canais_e_urls_da_config_remota() {
        let dir = tmpdir("janela-remota");
        instalar_assinado(&dir);
        let efetiva = resolver(&dir, Some(CHAVE)).expect("resolve");
        let html = crate::gui::html(&efetiva).expect("html");
        assert!(html.contains("#ff0000"), "acento remoto não renderizou");
        assert!(html.contains("Novo"), "canal remoto não renderizou");
        assert!(html.contains("http://127.0.0.1:3000/api/launcher/manifesto"));
        assert!(html.contains("revisão 3"));
        assert!(!html.contains("identidade embutida:"));
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn trocar_launcher_conf_muda_a_janela_sem_recompilar() {
        let dir = tmpdir("janela-troca");
        instalar_assinado(&dir);
        let antes = crate::gui::html(&resolver(&dir, Some(CHAVE)).expect("resolve")).expect("html");

        let outro = "# wyd-launcher-config v1\nrevisao = 8\n\
                     portal.base = http://127.0.0.1:9999\ncor.accent = #00ff00\n\
                     servidor.5 = Outro\n";
        install(&dir, &assinar(outro)).expect("instala outro");
        let depois =
            crate::gui::html(&resolver(&dir, Some(CHAVE)).expect("resolve")).expect("html");

        assert_ne!(antes, depois, "trocar o arquivo tem de mudar a janela");
        assert!(depois.contains("#00ff00") && depois.contains("Outro"));
        assert!(depois.contains("http://127.0.0.1:9999/api/launcher/pacote"));
        assert!(depois.contains("revisão 8"));
        assert!(!depois.contains("#ff0000"), "marca antiga não pode sobrar");
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn janela_cai_para_embutida_com_motivo_visivel() {
        let dir = tmpdir("janela-fallback");
        let efetiva = resolver(&dir, Some(CHAVE)).expect("resolve");
        let html = crate::gui::html(&efetiva).expect("html");
        assert!(html.contains("identidade embutida:"));
        assert!(html.contains("launcher.conf ausente"));
        assert!(html.contains("#c8a24a"), "acento embutido não renderizou");
        assert!(!html.contains("href=\"http"));
        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn nome_de_canal_remoto_e_escapado_no_html() {
        let dir = tmpdir("janela-escape");
        let texto = "# wyd-launcher-config v1\nrevisao = 1\n\
                     portal.base = http://127.0.0.1:3000\nservidor.3 = </b>W<\n";
        install(&dir, &assinar(texto)).expect("instala");
        let efetiva = resolver(&dir, Some(CHAVE)).expect("resolve");
        let html = crate::gui::html(&efetiva).expect("html");
        assert!(html.contains("&lt;/b&gt;W&lt;"), "nome não foi escapado");
        assert!(!html.contains("</b>W<"));
        fs::remove_dir_all(&dir).ok();
    }
}
