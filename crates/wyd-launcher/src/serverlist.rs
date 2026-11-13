//! `serverlist.bin`: o arquivo que o cliente 7662 lê para montar a lista de
//! servidores/canais.
//!
//! Formato provado nos bytes do cliente (`docs/cliente/formatos.md` §4):
//! grade `MAX_SERVERGROUP(10) × MAX_SERVERNUMBER(11) × 64 B` — `sizeof(g_pServerList)`
//! do legado (`Source/Code/Basedef.cpp:6874`) — em que cada célula é ASCII
//! terminado em NUL com cada byte somado de `obfuscationKey[63 - i]`
//! (`Basedef.cpp:6927`).
//!
//! Papel das células de um grupo (servidor): a célula 0 guarda a **URL de status**
//! (no máximo 32 caracteres úteis, senão o cliente não lista nada) e as células
//! 1..10 guardam o **host do canal** — o cliente fixa a porta `8281`, então cada
//! canal precisa de um endereço próprio.
//!
//! Este módulo é a **única** implementação de formato do lado Rust: geração,
//! leitura e instalação saem daqui. O gerador Node (`tools/client/serverlist.mjs`)
//! é a referência de operação e a paridade é cobrada por teste
//! (`tests/serverlist_paridade.rs`).

use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};

/// `MAX_SERVERGROUP` (`Basedef.h:109`): quantos servidores o cliente lista.
pub const GROUP_COUNT: usize = 10;
/// `MAX_SERVERNUMBER = MAX_SERVER + 1` (`Basedef.h:108-110`): canais por servidor.
pub const CHANNEL_COUNT: usize = 11;
/// Largura fixa de uma célula (`g_pServerList[..][..][64]`).
pub const ADDR_LEN: usize = 64;
/// `sizeof(g_pServerList)` — tamanho exato do arquivo gerado.
pub const BIN_SIZE: usize = GROUP_COUNT * CHANNEL_COUNT * ADDR_LEN;
/// Limite de caracteres úteis da URL de status (célula 0). Medido no cliente.
pub const STATUS_URL_MAX: usize = 32;
/// Nome do arquivo dentro do diretório do jogo.
pub const FILE_NAME: &str = "serverlist.bin";
/// Backup gravado ao substituir um arquivo diferente.
pub const BACKUP_NAME: &str = "serverlist.bin.bak";
/// Sufixo do arquivo temporário da instalação atômica.
const TMP_NAME: &str = "serverlist.bin.tmp";

/// Versão legível (`host:8281` por canal) instalada junto do `.bin`.
pub const TXT_NAME: &str = "serverlist.txt";
const TXT_TMP_NAME: &str = "serverlist.txt.tmp";

/// Tabela estática que o legado soma a cada byte (`Basedef.cpp:6927`).
pub const OBFUSCATION_KEY: [u8; ADDR_LEN] = [
    0xA4, 0xA1, 0xA4, 0xA4, 0xA4, 0xA7, 0xA4, 0xA9, 0xA4, 0xB1, 0xA4, 0xB2, 0xA4, 0xB5, 0xA4, 0xB7,
    0xA4, 0xB8, 0xA4, 0xBA, 0xA4, 0xBB, 0xA4, 0xBC, 0xA4, 0xBD, 0xA4, 0xBE, 0xA4, 0xBF, 0xA4, 0xC1,
    0xA4, 0xC3, 0xA4, 0xC5, 0xA4, 0xC7, 0xA4, 0xCB, 0xA4, 0xCC, 0xA4, 0xD0, 0xA4, 0xD1, 0xA4, 0xD3,
    0xA4, 0xBF, 0xA4, 0xC4, 0xA4, 0xD3, 0xA4, 0xC7, 0xA4, 0xCC, 0xB0, 0xA1, 0xB3, 0xAA, 0xB4, 0xD9,
];

/// Uma célula preenchida: grupo (servidor), canal e conteúdo em ASCII.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Cell {
    /// Índice do servidor na grade (`0..GROUP_COUNT`).
    pub group: usize,
    /// `0` = URL de status; `1..=10` = host do canal.
    pub channel: usize,
    /// Conteúdo da célula (URL ou host), sem o NUL.
    pub addr: String,
}

impl Cell {
    /// Célula 0: URL de status (`http://host/serv00.htm`).
    #[must_use]
    pub fn is_status(&self) -> bool {
        self.channel == 0
    }
}

/// Erros de registro/geração — nunca `panic` em dado de entrada.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ServerlistError {
    /// Registro sem nenhuma célula: a lista ficaria vazia.
    Empty,
    /// Linha que não é `<grupo> <canal> <endereço>`.
    BadLine { line: usize, reason: String },
    /// Grupo fora de `0..GROUP_COUNT`.
    GroupOutOfRange { line: usize, group: usize },
    /// Canal fora de `0..CHANNEL_COUNT`.
    ChannelOutOfRange { line: usize, channel: usize },
    /// Conteúdo maior que os 63 bytes úteis de uma célula.
    AddrTooLong { line: usize, len: usize },
    /// URL de status acima dos 32 caracteres que o cliente desenha.
    StatusUrlTooLong {
        line: usize,
        len: usize,
        addr: String,
    },
    /// URL de status que não começa com `http://`.
    StatusUrlNotHttp { line: usize, addr: String },
    /// Endereço de canal com porta — o cliente fixa 8281.
    ChannelWithPort {
        line: usize,
        addr: String,
        reason: &'static str,
    },
    /// Dois canais do mesmo grupo com o mesmo endereço.
    DuplicateChannelAddress { group: usize, addr: String },
    /// Arquivo de entrada com tamanho diferente de `BIN_SIZE`.
    SizeMismatch { found: usize, expected: usize },
    /// Falha de E/S na leitura, geração ou instalação.
    Io(String),
    /// A identidade única (`docs/arquitetura/identidade.tokens`) recusou o nome de servidor.
    Identidade(String),
}

impl fmt::Display for ServerlistError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Empty => write!(f, "registro de canais vazio: a lista ficaria sem servidor"),
            Self::BadLine { line, reason } => {
                write!(
                    f,
                    "linha {line}: esperado `<grupo> <canal> <endereço>` ({reason})"
                )
            }
            Self::GroupOutOfRange { line, group } => write!(
                f,
                "linha {line}: grupo {group} fora de 0..{}",
                GROUP_COUNT - 1
            ),
            Self::ChannelOutOfRange { line, channel } => write!(
                f,
                "linha {line}: canal {channel} fora de 0..{}",
                CHANNEL_COUNT - 1
            ),
            Self::AddrTooLong { line, len } => write!(
                f,
                "linha {line}: endereço com {len} bytes; a célula comporta {}",
                ADDR_LEN - 1
            ),
            Self::StatusUrlTooLong { line, len, addr } => write!(
                f,
                "linha {line}: URL de status com {len} caracteres ({addr}); \
                 o cliente só desenha a lista até {STATUS_URL_MAX} — use `http://host/serv00.htm` \
                 sem porta (o status é servido no :80)"
            ),
            Self::StatusUrlNotHttp { line, addr } => write!(
                f,
                "linha {line}: a célula 0 é a URL de status e precisa de `http://` (veio `{addr}`)"
            ),
            Self::ChannelWithPort { line, addr, reason } => write!(
                f,
                "linha {line}: endereço de canal `{addr}` inválido ({reason}); \
                 o cliente fixa a porta 8281, use um IP por canal"
            ),
            Self::DuplicateChannelAddress { group, addr } => write!(
                f,
                "grupo {group}: dois canais com o mesmo endereço `{addr}` — um IP por canal"
            ),
            Self::SizeMismatch { found, expected } => {
                write!(f, "arquivo com {found} bytes; esperado {expected}")
            }
            Self::Io(e) => write!(f, "erro de E/S: {e}"),
            Self::Identidade(e) => write!(f, "identidade: {e}"),
        }
    }
}

impl std::error::Error for ServerlistError {}

fn byte_len(text: &str) -> usize {
    text.len()
}

/// Analisa o registro em texto: `<grupo> <canal> <endereço>`, `#` comenta.
///
/// Valida os dois limites medidos **com erro nomeado**, em vez de truncar:
/// URL de status até [`STATUS_URL_MAX`] caracteres e endereço de canal sem porta
/// (com um endereço distinto por canal).
pub fn parse(text: &str) -> Result<Vec<Cell>, ServerlistError> {
    let mut cells: Vec<Cell> = Vec::new();
    for (idx, raw) in text.lines().enumerate() {
        let line = idx + 1;
        let trimmed = raw.trim();
        if trimmed.is_empty() || trimmed.starts_with('#') {
            continue;
        }
        let mut fields = trimmed.split_whitespace();
        let (group_s, channel_s, addr) =
            match (fields.next(), fields.next(), fields.next(), fields.next()) {
                (Some(g), Some(c), Some(a), None) => (g, c, a),
                _ => {
                    return Err(ServerlistError::BadLine {
                        line,
                        reason: "número de campos diferente de três".to_string(),
                    })
                }
            };
        let group: usize = group_s.parse().map_err(|_| ServerlistError::BadLine {
            line,
            reason: "grupo não é inteiro".to_string(),
        })?;
        let channel: usize = channel_s.parse().map_err(|_| ServerlistError::BadLine {
            line,
            reason: "canal não é inteiro".to_string(),
        })?;
        if group >= GROUP_COUNT {
            return Err(ServerlistError::GroupOutOfRange { line, group });
        }
        if channel >= CHANNEL_COUNT {
            return Err(ServerlistError::ChannelOutOfRange { line, channel });
        }
        if byte_len(addr) >= ADDR_LEN {
            return Err(ServerlistError::AddrTooLong {
                line,
                len: byte_len(addr),
            });
        }
        if channel == 0 {
            if !addr.starts_with("http://") {
                return Err(ServerlistError::StatusUrlNotHttp {
                    line,
                    addr: addr.to_string(),
                });
            }
            if addr.chars().count() > STATUS_URL_MAX {
                return Err(ServerlistError::StatusUrlTooLong {
                    line,
                    len: addr.chars().count(),
                    addr: addr.to_string(),
                });
            }
        } else {
            if addr.contains(':') {
                return Err(ServerlistError::ChannelWithPort {
                    line,
                    addr: addr.to_string(),
                    reason: "contém porta",
                });
            }
            if addr.contains('/') {
                return Err(ServerlistError::ChannelWithPort {
                    line,
                    addr: addr.to_string(),
                    reason: "contém caminho",
                });
            }
            let repetido = cells
                .iter()
                .any(|c| c.group == group && c.channel != 0 && c.addr == addr);
            if repetido {
                return Err(ServerlistError::DuplicateChannelAddress {
                    group,
                    addr: addr.to_string(),
                });
            }
        }
        cells.push(Cell {
            group,
            channel,
            addr: addr.to_string(),
        });
    }
    if cells.is_empty() {
        return Err(ServerlistError::Empty);
    }
    Ok(cells)
}

/// Gera os 7.040 bytes a partir das células já validadas.
///
/// Entrada posterior vence, como o `strcpy` sequencial do carregador legado.
#[must_use]
pub fn build(cells: &[Cell]) -> Vec<u8> {
    let mut bin = vec![0u8; BIN_SIZE];
    for cell in cells {
        let slot = (cell.group * CHANNEL_COUNT + cell.channel) * ADDR_LEN;
        let bytes = cell.addr.as_bytes();
        bin[slot..slot + bytes.len()].copy_from_slice(bytes);
    }
    for slot in (0..BIN_SIZE).step_by(ADDR_LEN) {
        for i in 0..ADDR_LEN {
            bin[slot + i] = bin[slot + i].wrapping_add(OBFUSCATION_KEY[ADDR_LEN - 1 - i]);
        }
    }
    bin
}

/// Desfaz [`build`]: devolve as células preenchidas de um arquivo do cliente.
pub fn decode(bin: &[u8]) -> Result<Vec<Cell>, ServerlistError> {
    if bin.len() != BIN_SIZE {
        return Err(ServerlistError::SizeMismatch {
            found: bin.len(),
            expected: BIN_SIZE,
        });
    }
    let mut cells = Vec::new();
    for group in 0..GROUP_COUNT {
        for channel in 0..CHANNEL_COUNT {
            let slot = (group * CHANNEL_COUNT + channel) * ADDR_LEN;
            let mut bytes = [0u8; ADDR_LEN];
            for (i, byte) in bytes.iter_mut().enumerate() {
                *byte = bin[slot + i].wrapping_sub(OBFUSCATION_KEY[ADDR_LEN - 1 - i]);
            }
            let end = bytes.iter().position(|b| *b == 0).unwrap_or(ADDR_LEN);
            if end == 0 {
                continue;
            }
            cells.push(Cell {
                group,
                channel,
                addr: String::from_utf8_lossy(&bytes[..end]).into_owned(),
            });
        }
    }
    Ok(cells)
}

/// `serverlist.txt` legível: uma linha de URL de status (canal 0) e **um `host:8281` por
/// canal de jogo**, na mesma ordem das células do `.bin`. É referência humana — quem manda
/// no cliente é o `.bin` —, mas divergir dele confunde operação, então a paridade é testada.
#[must_use]
pub fn render_txt(cells: &[Cell]) -> String {
    let mut out = String::new();
    if let Some(status) = cells
        .iter()
        .filter(|c| c.channel == 0 && !c.addr.is_empty())
        .min_by_key(|c| c.group)
    {
        out.push_str(&status.addr);
        out.push('\n');
    }
    let mut canais: Vec<&Cell> = cells.iter().filter(|c| c.channel >= 1).collect();
    canais.sort_by_key(|c| (c.group, c.channel));
    for canal in canais {
        out.push_str(&canal.addr);
        out.push_str(":8281\n");
    }
    out
}

/// Resultado da instalação no diretório do jogo.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct InstallReport {
    /// Caminho de `serverlist.bin`.
    pub path: PathBuf,
    /// Bytes instalados.
    pub bytes: usize,
    /// Backup criado (só quando havia um arquivo diferente).
    pub backup: Option<PathBuf>,
    /// Falso quando o arquivo já estava idêntico (nada foi reescrito).
    pub changed: bool,
}

/// Instala o `serverlist.bin` de forma **atômica com backup**.
///
/// Escreve `serverlist.bin.tmp` no mesmo diretório, copia o arquivo anterior para
/// `serverlist.bin.bak` quando ele difere e só então renomeia por cima — um
/// arquivo meio escrito nunca fica no lugar do válido. Se o conteúdo já é o
/// mesmo, nada é reescrito nem é criado backup novo.
pub fn install(dir: &Path, bin: &[u8]) -> Result<InstallReport, ServerlistError> {
    if bin.len() != BIN_SIZE {
        return Err(ServerlistError::SizeMismatch {
            found: bin.len(),
            expected: BIN_SIZE,
        });
    }
    let path = dir.join(FILE_NAME);
    // O `.txt` legível acompanha o `.bin` mesmo quando o `.bin` já está idêntico — foi assim
    // que ele ficou para trás com um host só depois de entrar o segundo canal.
    let txt_path = dir.join(TXT_NAME);
    let txt = render_txt(&decode(bin)?);
    if fs::read_to_string(&txt_path).ok().as_deref() != Some(txt.as_str()) {
        let txt_tmp = dir.join(TXT_TMP_NAME);
        fs::write(&txt_tmp, txt.as_bytes()).map_err(|e| ServerlistError::Io(e.to_string()))?;
        fs::rename(&txt_tmp, &txt_path).map_err(|e| {
            let _ = fs::remove_file(&txt_tmp);
            ServerlistError::Io(e.to_string())
        })?;
    }

    let atual = fs::read(&path).ok();
    if atual.as_deref() == Some(bin) {
        return Ok(InstallReport {
            path,
            bytes: bin.len(),
            backup: None,
            changed: false,
        });
    }
    let tmp = dir.join(TMP_NAME);
    fs::write(&tmp, bin).map_err(|e| ServerlistError::Io(e.to_string()))?;
    let mut backup = None;
    if atual.is_some() {
        let destino = dir.join(BACKUP_NAME);
        fs::copy(&path, &destino).map_err(|e| ServerlistError::Io(e.to_string()))?;
        backup = Some(destino);
    }
    fs::rename(&tmp, &path).map_err(|e| {
        let _ = fs::remove_file(&tmp);
        ServerlistError::Io(e.to_string())
    })?;
    Ok(InstallReport {
        path,
        bytes: bin.len(),
        backup,
        changed: true,
    })
}

/// Lê e analisa um registro de canais versionado.
pub fn load_registry(path: &Path) -> Result<Vec<Cell>, ServerlistError> {
    let text = fs::read_to_string(path).map_err(|e| ServerlistError::Io(e.to_string()))?;
    parse(&text)
}

/// Conveniência: lê o registro e devolve os bytes prontos para instalar.
pub fn generate_from_registry(path: &Path) -> Result<Vec<u8>, ServerlistError> {
    Ok(build(&load_registry(path)?))
}

/// Erro de E/S convertido para o tipo do módulo (usado pelo binário).
impl From<io::Error> for ServerlistError {
    fn from(e: io::Error) -> Self {
        Self::Io(e.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const REGISTRO: &str = "\
# comentário
1 0 http://192.168.100.3/serv00.htm
1 1 192.168.100.3
";

    fn tmpdir(tag: &str) -> PathBuf {
        use std::sync::atomic::{AtomicU64, Ordering};
        static N: AtomicU64 = AtomicU64::new(0);
        let d = std::env::temp_dir().join(format!(
            "wyd-launcher-serverlist-{tag}-{}-{}",
            std::process::id(),
            N.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(&d).unwrap();
        d
    }

    #[test]
    fn gera_arquivo_do_tamanho_do_cliente() {
        let cells = parse(REGISTRO).unwrap();
        let bin = build(&cells);
        assert_eq!(bin.len(), BIN_SIZE);
        assert_eq!(bin.len(), 7040);
    }

    #[test]
    fn round_trip_preserva_celulas() {
        let cells = parse(REGISTRO).unwrap();
        let volta = decode(&build(&cells)).unwrap();
        assert_eq!(volta, cells);
    }

    #[test]
    fn ofuscacao_bate_com_os_bytes_do_cliente() {
        // Célula `1 1 192.168.100.3`: cada byte recebe key[63-i].
        let bin = build(&parse(REGISTRO).unwrap());
        let slot = (CHANNEL_COUNT + 1) * ADDR_LEN;
        let esperado: Vec<u8> = "192.168.100.3"
            .bytes()
            .enumerate()
            .map(|(i, b)| b.wrapping_add(OBFUSCATION_KEY[ADDR_LEN - 1 - i]))
            .collect();
        assert_eq!(&bin[slot..slot + esperado.len()], &esperado[..]);
        assert_eq!(
            bin[slot + esperado.len()],
            OBFUSCATION_KEY[ADDR_LEN - 1 - esperado.len()]
        );
    }

    #[test]
    fn url_de_status_acima_de_32_e_erro_nomeado() {
        let longo = "1 0 http://192.168.100.3:8281/serv00.htm\n1 1 192.168.100.3\n";
        let err = parse(longo).unwrap_err();
        match &err {
            ServerlistError::StatusUrlTooLong { line, len, .. } => {
                assert_eq!((*line, *len), (1, 36));
            }
            outro => panic!("erro errado: {outro}"),
        }
        assert!(err.to_string().contains("32"));
    }

    #[test]
    fn url_de_status_no_limite_de_32_passa() {
        let no_limite = "0 0 http://127.0.0.1:8281/serv00.htm\n0 1 127.0.0.1\n";
        assert_eq!("http://127.0.0.1:8281/serv00.htm".chars().count(), 32);
        let cells = parse(no_limite).unwrap();
        assert_eq!(cells.len(), 2);
    }

    #[test]
    fn canal_com_porta_e_erro_nomeado() {
        let com_porta = "0 0 http://127.0.0.1/serv00.htm\n0 1 192.168.100.4:8282\n";
        let err = parse(com_porta).unwrap_err();
        assert!(matches!(
            err,
            ServerlistError::ChannelWithPort { line: 2, .. }
        ));
        assert!(err.to_string().contains("8281"));
    }

    #[test]
    fn dois_canais_com_o_mesmo_ip_e_erro() {
        let repetido = "0 0 http://127.0.0.1/serv00.htm\n0 1 192.168.100.3\n0 2 192.168.100.3\n";
        assert!(matches!(
            parse(repetido),
            Err(ServerlistError::DuplicateChannelAddress { group: 0, .. })
        ));
        // IPs distintos passam: é o modelo de canal (um IP por canal na 8281).
        let distinto = "0 0 http://127.0.0.1/serv00.htm\n0 1 192.168.100.3\n0 2 192.168.100.4\n";
        assert_eq!(parse(distinto).unwrap().len(), 3);
    }

    #[test]
    fn limites_da_grade_e_do_campo() {
        assert!(matches!(
            parse("10 1 127.0.0.1\n"),
            Err(ServerlistError::GroupOutOfRange { group: 10, .. })
        ));
        assert!(matches!(
            parse("0 11 127.0.0.1\n"),
            Err(ServerlistError::ChannelOutOfRange { channel: 11, .. })
        ));
        let grande = format!("0 1 {}\n", "9".repeat(64));
        assert!(matches!(
            parse(&grande),
            Err(ServerlistError::AddrTooLong { len: 64, .. })
        ));
        // 63 bytes cabem (63 úteis + NUL): o limite é inclusivo.
        let cabe = format!("0 1 {}\n", "9".repeat(63));
        assert_eq!(parse(&cabe).unwrap()[0].addr.len(), 63);
        assert!(matches!(
            parse("\n# só comentário\n"),
            Err(ServerlistError::Empty)
        ));
    }

    #[test]
    fn decode_recusa_tamanho_errado() {
        assert!(matches!(
            decode(&[0u8; 10]),
            Err(ServerlistError::SizeMismatch { found: 10, .. })
        ));
    }

    #[test]
    fn instala_com_backup_e_nao_reescreve_igual() {
        let dir = tmpdir("install");
        let bin_a = build(&parse(REGISTRO).unwrap());
        let primeiro = install(&dir, &bin_a).unwrap();
        assert!(primeiro.changed);
        assert!(
            primeiro.backup.is_none(),
            "primeira instalação não tem o que salvar"
        );

        let segundo = install(&dir, &bin_a).unwrap();
        assert!(!segundo.changed, "conteúdo idêntico não deve ser reescrito");
        assert!(segundo.backup.is_none());

        let outro = build(&parse("2 0 http://127.0.0.1/serv00.htm\n2 1 127.0.0.1\n").unwrap());
        let terceiro = install(&dir, &outro).unwrap();
        assert!(terceiro.changed);
        assert_eq!(
            fs::read(dir.join(BACKUP_NAME)).unwrap(),
            bin_a,
            "backup é o anterior"
        );
        assert_eq!(fs::read(dir.join(FILE_NAME)).unwrap(), outro);
        assert!(!dir.join(TMP_NAME).exists(), "temporário não pode sobrar");
        fs::remove_dir_all(&dir).ok();
    }
}
