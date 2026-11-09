//! Skin da tela de entrada do launcher: `Launcher.bmp` + `Launcher.hat` gerados da identidade.
//!
//! A cadeia foi medida no cliente 7662: `WYDLauncher.exe` abre `%s/Launcher.hat`
//! (string única no binário) e o `.hat` é HTML que aponta para `Launcher.bmp`
//! (`<img src="Launcher.bmp">`). Os dois arquivos são formatos totalmente decodificáveis —
//! BMP 24 bits sem compressão e HTML —, então esta é a superfície da tela inicial que dá para
//! pintar **sem** RE do `.wyt` (a tela de seleção de servidor do `WYD.exe`, que fica para a
//! FASE-08-6).
//!
//! A escrita é atômica com backup (`.bak`), igual ao `serverlist.bin`: nunca um arquivo meio
//! escrito no lugar do válido.

use std::fs;
use std::io;
use std::path::{Path, PathBuf};

use crate::identidade::{Identidade, IdentidadeError, Rgb};

/// Nome do skin carregado pelo `.hat`.
pub const BMP_NAME: &str = "Launcher.bmp";
/// HTML da janela de entrada do launcher.
pub const HAT_NAME: &str = "Launcher.hat";
/// Largura medida no `client/Launcher.bmp` original (BITMAPINFOHEADER, offset 0x18).
pub const LARGURA: u32 = 598;
/// Altura medida no arquivo original (offset 0x1C).
pub const ALTURA: u32 = 369;

/// Altura da faixa de destaque, em pixels, acima da base da skin.
const FAIXA_ALTURA: u32 = 6;
/// Distância da faixa à base.
const FAIXA_MARGEM: u32 = 40;
/// Escala dos glifos do wordmark (5×7 → 40×56).
const GLIFO_ESCALA: u32 = 8;
/// Espaço entre glifos.
const GLIFO_ESPACO: u32 = 16;

/// Glifos 5×7 do wordmark (bit 4 = coluna esquerda).
const GLIFO_W: [[u8; 5]; 7] = [
    [1, 0, 0, 0, 1],
    [1, 0, 0, 0, 1],
    [1, 0, 0, 0, 1],
    [1, 0, 1, 0, 1],
    [1, 0, 1, 0, 1],
    [1, 1, 0, 1, 1],
    [1, 0, 0, 0, 1],
];
const GLIFO_Y: [[u8; 5]; 7] = [
    [1, 0, 0, 0, 1],
    [1, 0, 0, 0, 1],
    [0, 1, 0, 1, 0],
    [0, 0, 1, 0, 0],
    [0, 0, 1, 0, 0],
    [0, 0, 1, 0, 0],
    [0, 0, 1, 0, 0],
];
const GLIFO_D: [[u8; 5]; 7] = [
    [1, 1, 1, 1, 0],
    [1, 0, 0, 0, 1],
    [1, 0, 0, 0, 1],
    [1, 0, 0, 0, 1],
    [1, 0, 0, 0, 1],
    [1, 0, 0, 0, 1],
    [1, 1, 1, 1, 0],
];

/// Relatório de instalação de um asset (mesma forma do `serverlist::InstallReport`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct InstallReport {
    /// Caminho gravado.
    pub path: PathBuf,
    /// Bytes gravados.
    pub bytes: usize,
    /// Backup criado (só quando havia um arquivo diferente).
    pub backup: Option<PathBuf>,
    /// Falso quando o arquivo já era idêntico.
    pub changed: bool,
}

/// Gera o `Launcher.bmp` (24 bits, 598×369) na paleta da identidade.
///
/// Desenho: gradiente vertical de `cor.surface` para `cor.bg`, faixa em `cor.accent` acima da
/// base e o wordmark `WYD` no mesmo dourado. Nenhuma cor fora da identidade e nenhum
/// timestamp — o mesmo arquivo de entrada gera os mesmos bytes.
pub fn bmp(id: &Identidade) -> Result<Vec<u8>, IdentidadeError> {
    let topo = id.cor("cor.surface")?;
    let fundo = id.cor("cor.bg")?;
    let destaque = id.cor("cor.accent")?;

    let stride = stride(LARGURA);
    let mut pixels = vec![0u8; stride * ALTURA as usize];
    for y in 0..ALTURA {
        #[allow(clippy::cast_precision_loss)]
        let t = y as f32 / (ALTURA - 1) as f32;
        let cor = topo.lerp(fundo, t);
        for x in 0..LARGURA {
            pintar(&mut pixels, stride, x, y, cor);
        }
    }
    let base_faixa = ALTURA - FAIXA_MARGEM;
    for y in base_faixa..base_faixa + FAIXA_ALTURA {
        for x in 0..LARGURA {
            pintar(&mut pixels, stride, x, y, destaque);
        }
    }
    let marca = [GLIFO_W, GLIFO_Y, GLIFO_D];
    let largura_marca = 3 * 5 * GLIFO_ESCALA + 2 * GLIFO_ESPACO;
    let x0 = (LARGURA - largura_marca) / 2;
    let y0 = ALTURA / 3;
    for (indice, glifo) in marca.iter().enumerate() {
        let base_x = x0 + indice as u32 * (5 * GLIFO_ESCALA + GLIFO_ESPACO);
        for (linha, bits) in glifo.iter().enumerate() {
            for (coluna, bit) in bits.iter().enumerate() {
                if *bit == 0 {
                    continue;
                }
                for dy in 0..GLIFO_ESCALA {
                    for dx in 0..GLIFO_ESCALA {
                        pintar(
                            &mut pixels,
                            stride,
                            base_x + coluna as u32 * GLIFO_ESCALA + dx,
                            y0 + linha as u32 * GLIFO_ESCALA + dy,
                            destaque,
                        );
                    }
                }
            }
        }
    }

    let mut out = Vec::with_capacity(54 + pixels.len());
    let tamanho_arquivo = 54u32 + pixels.len() as u32;
    out.extend_from_slice(b"BM");
    out.extend_from_slice(&tamanho_arquivo.to_le_bytes());
    out.extend_from_slice(&0u32.to_le_bytes());
    out.extend_from_slice(&54u32.to_le_bytes());
    out.extend_from_slice(&40u32.to_le_bytes());
    out.extend_from_slice(&(LARGURA as i32).to_le_bytes());
    out.extend_from_slice(&(ALTURA as i32).to_le_bytes());
    out.extend_from_slice(&1u16.to_le_bytes());
    out.extend_from_slice(&24u16.to_le_bytes());
    out.extend_from_slice(&0u32.to_le_bytes());
    out.extend_from_slice(&0u32.to_le_bytes());
    out.extend_from_slice(&2834i32.to_le_bytes());
    out.extend_from_slice(&2834i32.to_le_bytes());
    out.extend_from_slice(&0u32.to_le_bytes());
    out.extend_from_slice(&0u32.to_le_bytes());
    out.extend_from_slice(&pixels);
    Ok(out)
}

/// Gera o `Launcher.hat` (HTML da janela de entrada) nas cores da identidade.
pub fn hat(id: &Identidade) -> Result<String, IdentidadeError> {
    let fundo = id.cor("cor.bg")?.hex();
    Ok(format!(
        "<body TOPMARGIN=0 LEFTMARGIN=0 MARGINWIDTH=0 MARGINHEIGHT=0 bgcolor=\"{fundo}\">\r\n  \
<table WIDTH=\"530\" HEIGHT=\"298\" BORDER=0 CELLPADDING=0 CELLSPACING=0>\r\n  \
<tr valign=\"center\"><td align=\"center\" bgcolor=\"{fundo}\"><img src=\"{BMP_NAME}\"></td></tr>\r\n  \
</table>\r\n</body>\n"
    ))
}

/// Instala os dois assets de forma atômica, com backup por arquivo.
pub fn install(dir: &Path, arquivos: &[(&str, &[u8])]) -> Result<Vec<InstallReport>, io::Error> {
    let mut relatorios = Vec::new();
    for (nome, conteudo) in arquivos {
        let path = dir.join(nome);
        let atual = fs::read(&path).ok();
        if atual.as_deref() == Some(*conteudo) {
            relatorios.push(InstallReport {
                path,
                bytes: conteudo.len(),
                backup: None,
                changed: false,
            });
            continue;
        }
        let tmp = dir.join(format!("{nome}.tmp"));
        fs::write(&tmp, conteudo)?;
        let mut backup = None;
        if atual.is_some() {
            let destino = dir.join(format!("{nome}.bak"));
            fs::copy(&path, &destino)?;
            backup = Some(destino);
        }
        if let Err(erro) = fs::rename(&tmp, &path) {
            let _ = fs::remove_file(&tmp);
            return Err(erro);
        }
        relatorios.push(InstallReport {
            path,
            bytes: conteudo.len(),
            backup,
            changed: true,
        });
    }
    Ok(relatorios)
}

fn stride(largura: u32) -> usize {
    // 24 bits por pixel, linha alinhada em 4 bytes (BMP).
    let bytes = largura as usize * 3;
    bytes.div_ceil(4) * 4
}

fn pintar(pixels: &mut [u8], stride: usize, x: u32, y: u32, cor: Rgb) {
    if x >= LARGURA || y >= ALTURA {
        return;
    }
    // BMP é bottom-up: a linha 0 do arquivo é a base da imagem.
    let linha = (ALTURA - 1 - y) as usize;
    let offset = linha * stride + x as usize * 3;
    pixels[offset] = cor.b;
    pixels[offset + 1] = cor.g;
    pixels[offset + 2] = cor.r;
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identidade::padrao;

    #[test]
    fn bmp_tem_cabecalho_24_bits_e_dimensao_do_original() {
        let id = padrao().expect("identidade");
        let bytes = bmp(&id).expect("skin");
        assert_eq!(&bytes[..2], b"BM");
        let largura = i32::from_le_bytes([bytes[0x12], bytes[0x13], bytes[0x14], bytes[0x15]]);
        let altura = i32::from_le_bytes([bytes[0x16], bytes[0x17], bytes[0x18], bytes[0x19]]);
        let bits = u16::from_le_bytes([bytes[0x1c], bytes[0x1d]]);
        assert_eq!((largura as u32, altura as u32, bits), (LARGURA, ALTURA, 24));
        assert_eq!(bytes.len(), 54 + stride(LARGURA) * ALTURA as usize);
    }

    #[test]
    fn hat_usa_a_cor_de_fundo_da_identidade() {
        let id = padrao().expect("identidade");
        let html = hat(&id).expect("hat");
        assert!(html.contains(&format!("bgcolor=\"{}\"", id.cor("cor.bg").unwrap().hex())));
        assert!(html.contains(BMP_NAME));
    }
}
