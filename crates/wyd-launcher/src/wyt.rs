//! Re-tematização dos `.wyt` da interface do cliente 7662 na paleta da identidade única.
//!
//! O formato foi decodificado em `tools/client/wyt.mjs` e está em `docs/cliente/formatos.md`
//! §1: `WT10`, cabeçalho de 22 B (largura @16, altura @18, bits/pixel @20), raster sem
//! compressão (`w*h*(bits/8)`, BGR ou BGRA) e 8 B finais. Este módulo **não** inventa formato:
//! reescreve só os bytes de pixel, mantendo cabeçalho, dimensões, canal alfa e trailer — se a
//! conta de tamanho não fechar, recusa em vez de gerar arquivo que o cliente não abre.
//!
//! A cor não vem de literal nenhum: cada pixel tem a luminância medida e é remapeado por uma
//! rampa montada com `cor.bg`, `cor.surface-2`, `cor.border-strong` e `cor.accent` da
//! identidade. O mesmo token que pinta portal, skin do launcher e `LauncherGUI.html` pinta a
//! tela de seleção.

use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};

use crate::branding::InstallReport;
use crate::identidade::{Identidade, IdentidadeError, Rgb};

/// Magic do pacote de UI do 7662.
pub const MAGIC: &[u8; 4] = b"WT10";
/// Cabeçalho medido (magic, versão, dois reservados, largura, altura, bits, flag).
pub const HEADER: usize = 22;
/// Marcador final medido (`00`×8 nas 259 amostras).
pub const TRAILER: usize = 8;
/// Largura em pixels de `NUI/ServerList.wyt` (medida no arquivo real).
pub const SERVERLIST_LARGURA: u16 = 287;
/// Altura em pixels de `NUI/ServerList.wyt`.
pub const SERVERLIST_ALTURA: u16 = 280;

/// Recusas de leitura/escrita do `.wyt`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum WytError {
    /// Arquivo menor que cabeçalho + trailer.
    Curto { bytes: usize },
    /// Primeiros 4 bytes não são `WT10`.
    Magic,
    /// Bits por pixel fora de 24/32 (existem 8 e 16 bpp no cliente; não são re-tematizados).
    Bits { bits: u8 },
    /// Dimensão zero.
    Dimensao,
    /// `22 + w*h*bpp + 8` não é o tamanho do arquivo.
    Tamanho { esperado: usize, real: usize },
    /// Identidade sem alguma cor usada na rampa.
    Identidade(IdentidadeError),
}

impl fmt::Display for WytError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Curto { bytes } => write!(f, "arquivo de {bytes} B: menor que 30 B"),
            Self::Magic => write!(f, "magic diferente de \"WT10\""),
            Self::Bits { bits } => write!(f, "{bits} bits por pixel (só 24/32 são re-tematizados)"),
            Self::Dimensao => write!(f, "dimensão zero no cabeçalho"),
            Self::Tamanho { esperado, real } => {
                write!(
                    f,
                    "tamanho {real} B não fecha o leiaute (esperado {esperado} B)"
                )
            }
            Self::Identidade(erro) => write!(f, "identidade: {erro}"),
        }
    }
}

impl std::error::Error for WytError {}

impl From<IdentidadeError> for WytError {
    fn from(erro: IdentidadeError) -> Self {
        Self::Identidade(erro)
    }
}

/// Raster decodificado, com o cabeçalho e o trailer preservados byte a byte.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Raster {
    /// Cabeçalho de 22 B, intocado.
    pub cabecalho: Vec<u8>,
    /// Largura em pixels (@16).
    pub largura: u16,
    /// Altura em pixels (@18).
    pub altura: u16,
    /// Bits por pixel (@20): 24 (BGR) ou 32 (BGRA).
    pub bits: u8,
    /// Pixels BGR/BGRA na ordem do arquivo.
    pub pixels: Vec<u8>,
    /// Marcador final de 8 B, intocado.
    pub trailer: Vec<u8>,
}

impl Raster {
    /// Bytes por pixel.
    #[must_use]
    pub fn bpp(&self) -> usize {
        usize::from(self.bits / 8)
    }

    /// Serializa de volta no leiaute medido.
    #[must_use]
    pub fn render(&self) -> Vec<u8> {
        let mut out = Vec::with_capacity(HEADER + self.pixels.len() + TRAILER);
        out.extend_from_slice(&self.cabecalho);
        out.extend_from_slice(&self.pixels);
        out.extend_from_slice(&self.trailer);
        out
    }
}

/// Decodifica o pacote; qualquer divergência é erro tipado (nunca "melhor esforço").
pub fn parse(bytes: &[u8]) -> Result<Raster, WytError> {
    if bytes.len() < HEADER + TRAILER {
        return Err(WytError::Curto { bytes: bytes.len() });
    }
    if &bytes[0..4] != MAGIC {
        return Err(WytError::Magic);
    }
    let largura = u16::from_le_bytes([bytes[16], bytes[17]]);
    let altura = u16::from_le_bytes([bytes[18], bytes[19]]);
    let bits = bytes[20];
    if bits != 24 && bits != 32 {
        return Err(WytError::Bits { bits });
    }
    if largura == 0 || altura == 0 {
        return Err(WytError::Dimensao);
    }
    let bpp = usize::from(bits / 8);
    let esperado = HEADER + usize::from(largura) * usize::from(altura) * bpp + TRAILER;
    if esperado != bytes.len() {
        return Err(WytError::Tamanho {
            esperado,
            real: bytes.len(),
        });
    }
    let fim = bytes.len() - TRAILER;
    Ok(Raster {
        cabecalho: bytes[..HEADER].to_vec(),
        largura,
        altura,
        bits,
        pixels: bytes[HEADER..fim].to_vec(),
        trailer: bytes[fim..].to_vec(),
    })
}

/// Luminância inteira (77 R + 150 G + 29 B, soma 256 → deslocamento de 8 bits).
fn luma(r: u8, g: u8, b: u8) -> u8 {
    let soma = 77 * u32::from(r) + 150 * u32::from(g) + 29 * u32::from(b);
    // A soma de pesos é 256, então `>> 8` devolve exatamente 0..=255.
    #[allow(clippy::cast_possible_truncation)]
    let valor = (soma >> 8) as u8;
    valor
}

/// Rampa: `cor.bg` → `cor.surface-2` → `cor.border-strong` → `cor.accent`.
fn grau(entrada: u8, id: &Identidade) -> Result<Rgb, WytError> {
    let fundo = id.cor("cor.bg")?;
    let superficie = id.cor("cor.surface-2")?;
    let borda = id.cor("cor.border-strong")?;
    let acento = id.cor("cor.accent")?;
    let t = f32::from(entrada) / 255.0;
    Ok(if t < 0.55 {
        fundo.lerp(superficie, t / 0.55)
    } else if t < 0.85 {
        superficie.lerp(borda, (t - 0.55) / 0.30)
    } else {
        borda.lerp(acento, (t - 0.85) / 0.15)
    })
}

/// Reescreve os pixels na identidade, preservando cabeçalho, alfa e trailer.
pub fn retheme(origem: &[u8], id: &Identidade) -> Result<Vec<u8>, WytError> {
    let mut raster = parse(origem)?;
    let bpp = raster.bpp();
    for pixel in raster.pixels.chunks_exact_mut(bpp) {
        let (b, g, r) = (pixel[0], pixel[1], pixel[2]);
        let cor = grau(luma(r, g, b), id)?;
        pixel[0] = cor.b;
        pixel[1] = cor.g;
        pixel[2] = cor.r;
        // pixel[3] (alfa) fica como está: o cliente usa o alfa do original.
    }
    Ok(raster.render())
}

/// Grava o `.wyt` re-tematizado de forma atômica, com backup ao lado do destino.
pub fn install(destino: &Path, bytes: &[u8]) -> Result<InstallReport, io::Error> {
    let atual = fs::read(destino).ok();
    if atual.as_deref() == Some(bytes) {
        return Ok(InstallReport {
            path: destino.to_path_buf(),
            bytes: bytes.len(),
            backup: None,
            changed: false,
        });
    }
    let tmp: PathBuf = destino.with_extension("wyt.tmp");
    fs::write(&tmp, bytes)?;
    let mut backup = None;
    if atual.is_some() {
        let destino_bak = destino.with_extension("wyt.bak");
        fs::copy(destino, &destino_bak)?;
        backup = Some(destino_bak);
    }
    if let Err(erro) = fs::rename(&tmp, destino) {
        let _ = fs::remove_file(&tmp);
        return Err(erro);
    }
    Ok(InstallReport {
        path: destino.to_path_buf(),
        bytes: bytes.len(),
        backup,
        changed: true,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identidade::padrao;

    fn raster_de_dois(cores: [(u8, u8, u8); 2]) -> Vec<u8> {
        let mut bytes = vec![0u8; HEADER + 2 * 4 + TRAILER];
        bytes[..4].copy_from_slice(MAGIC);
        bytes[4..8].copy_from_slice(&0x0002_0000u32.to_le_bytes());
        bytes[16..18].copy_from_slice(&2u16.to_le_bytes());
        bytes[18..20].copy_from_slice(&1u16.to_le_bytes());
        bytes[20] = 32;
        bytes[21] = 8;
        for (indice, (r, g, b)) in cores.iter().enumerate() {
            let base = HEADER + indice * 4;
            bytes[base] = *b;
            bytes[base + 1] = *g;
            bytes[base + 2] = *r;
            bytes[base + 3] = 0x80;
        }
        bytes
    }

    #[test]
    fn leiaute_do_client_e_preservado() {
        let origem = raster_de_dois([(0, 0, 0), (255, 255, 255)]);
        let id = padrao().expect("identidade");
        let saida = retheme(&origem, &id).expect("re-tematiza");
        assert_eq!(saida.len(), origem.len());
        assert_eq!(&saida[..HEADER], &origem[..HEADER], "cabeçalho intocado");
        assert_eq!(&saida[saida.len() - TRAILER..], &[0u8; TRAILER]);
        // Alfa preservado nas duas amostras.
        assert_eq!(saida[HEADER + 3], 0x80);
        assert_eq!(saida[HEADER + 7], 0x80);
    }

    #[test]
    fn preto_vira_fundo_e_branco_vira_acento_da_identidade() {
        let origem = raster_de_dois([(0, 0, 0), (255, 255, 255)]);
        let id = padrao().expect("identidade");
        let saida = retheme(&origem, &id).expect("re-tematiza");
        let fundo = id.cor("cor.bg").expect("fundo");
        let acento = id.cor("cor.accent").expect("acento");
        assert_eq!(&saida[HEADER..HEADER + 3], &[fundo.b, fundo.g, fundo.r]);
        assert_eq!(
            &saida[HEADER + 4..HEADER + 7],
            &[acento.b, acento.g, acento.r]
        );
    }

    #[test]
    fn arquivo_que_nao_fecha_a_conta_e_recusado() {
        let id = padrao().expect("identidade");
        assert_eq!(retheme(&[0u8; 10], &id), Err(WytError::Curto { bytes: 10 }));
        let mut curto_magic = raster_de_dois([(0, 0, 0), (0, 0, 0)]);
        curto_magic[0] = b'X';
        assert_eq!(retheme(&curto_magic, &id), Err(WytError::Magic));
        let mut bits = raster_de_dois([(0, 0, 0), (0, 0, 0)]);
        bits[20] = 16;
        assert_eq!(retheme(&bits, &id), Err(WytError::Bits { bits: 16 }));
        let mut tamanho = raster_de_dois([(0, 0, 0), (0, 0, 0)]);
        tamanho.push(0);
        assert!(matches!(
            retheme(&tamanho, &id),
            Err(WytError::Tamanho { .. })
        ));
    }
}
