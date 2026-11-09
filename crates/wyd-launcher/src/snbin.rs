//! `sn.bin` — os nomes que o cliente desenha na **tela de entrada** (server select).
//!
//! Leiaute medido (143 B, `docs/cliente/formatos.md` §3): 11 campos de 9 B começando no
//! offset 0 e 11 contadores `int32` little-endian a partir do offset 99.
//!
//! - slot 0 é o cabeçalho reservado: primeiro byte `0xd6` (o `R` de `Reserved` sobrescrito),
//!   `contador = canais + 1`;
//! - slots 1..=10 são servidores (o slot **é** o grupo do `serverlist.bin`), com
//!   `contador = grupo + 1` quando aquele grupo tem canal. Zerar o contador derruba a linha;
//! - o rótulo final é `sprintf("%s-%d", nome, canal)` no cliente — o nome vem daqui, da
//!   identidade única (`servidor.N`), nunca de constante no código.
//!
//! Sem o slot 0 intacto o cliente monta `"Canal N Grupp"`; é por isso que este módulo
//! preserva o byte medido em vez de escrever `"Reserved"` limpo.

use std::fs;
use std::io;
use std::path::{Path, PathBuf};

use crate::branding::InstallReport;
use crate::identidade::{Identidade, IdentidadeError};
use crate::serverlist::Cell;

/// Slots do arquivo (cabeçalho reservado + 10 servidores).
pub const SLOTS: usize = 11;
/// Tamanho de cada nome (NUL incluído).
pub const NAME_LEN: usize = 9;
/// Início dos contadores `int32` (11 × 9).
pub const COUNTER_OFFSET: usize = SLOTS * NAME_LEN;
/// Tamanho total medido.
pub const SIZE: usize = COUNTER_OFFSET + SLOTS * 4;
/// Nome do arquivo no diretório do cliente.
pub const FILE_NAME: &str = "sn.bin";
/// Primeiro byte do slot reservado (medido no cliente 7662).
pub const RESERVED_FIRST_BYTE: u8 = 0xd6;

/// Gera os 143 bytes a partir da identidade e da contagem de canais por grupo.
///
/// `canais_por_grupo[i]` é quantos canais o grupo `i` publica no `serverlist.bin`
/// (`0` = grupo sem canal, contador zerado).
pub fn build(id: &Identidade, canais_por_grupo: &[usize; 10]) -> Result<Vec<u8>, IdentidadeError> {
    let servidores = id.servidores()?;
    let mut bin = vec![0u8; SIZE];
    // Slot 0 reservado: byte opaco medido + "eserved", para o cliente não cair em
    // "Canal N Grupp".
    bin[0] = RESERVED_FIRST_BYTE;
    bin[1..NAME_LEN - 1].copy_from_slice(b"eserved");
    let total: usize = canais_por_grupo.iter().sum();
    escrever_contador(&mut bin, 0, total + 1);
    for (slot, nome) in servidores {
        // O slot do arquivo **é** o grupo do `serverlist.bin` (slot 0 é o cabeçalho
        // reservado), e o contador é `grupo + 1` — é o que faz aquele slot desenhar o
        // grupo. Nome sem canal publicado não é desenhado (contador zerado).
        if slot == 0 || slot >= canais_por_grupo.len() || canais_por_grupo[slot] == 0 {
            continue;
        }
        let inicio = slot * NAME_LEN;
        bin[inicio..inicio + nome.len()].copy_from_slice(nome.as_bytes());
        escrever_contador(&mut bin, slot, slot + 1);
    }
    Ok(bin)
}

/// Conta canais por grupo a partir das células do `serverlist.bin`.
#[must_use]
pub fn canais_por_grupo(cells: &[Cell]) -> [usize; 10] {
    let mut contagem = [0usize; 10];
    for cell in cells {
        if cell.channel >= 1 && cell.group < contagem.len() {
            contagem[cell.group] += 1;
        }
    }
    contagem
}

/// Instala o `sn.bin` de forma atômica, com backup (`sn.bin.bak`).
pub fn install(dir: &Path, bin: &[u8]) -> Result<InstallReport, io::Error> {
    let path: PathBuf = dir.join(FILE_NAME);
    let atual = fs::read(&path).ok();
    if atual.as_deref() == Some(bin) {
        return Ok(InstallReport {
            path,
            bytes: bin.len(),
            backup: None,
            changed: false,
        });
    }
    let tmp = dir.join(format!("{FILE_NAME}.tmp"));
    fs::write(&tmp, bin)?;
    let mut backup = None;
    if atual.is_some() {
        let destino = dir.join(format!("{FILE_NAME}.bak"));
        fs::copy(&path, &destino)?;
        backup = Some(destino);
    }
    if let Err(erro) = fs::rename(&tmp, &path) {
        let _ = fs::remove_file(&tmp);
        return Err(erro);
    }
    Ok(InstallReport {
        path,
        bytes: bin.len(),
        backup,
        changed: true,
    })
}

fn escrever_contador(bin: &mut [u8], slot: usize, valor: usize) {
    let inicio = COUNTER_OFFSET + slot * 4;
    let bytes = i32::try_from(valor).unwrap_or(i32::MAX).to_le_bytes();
    bin[inicio..inicio + 4].copy_from_slice(&bytes);
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identidade::padrao;

    fn dois_canais() -> [usize; 10] {
        // Grupo 1 (o slot do arquivo é o grupo; 0 é o cabeçalho reservado).
        let mut contagem = [0usize; 10];
        contagem[1] = 2;
        contagem
    }

    #[test]
    fn sn_bin_tem_o_leiaute_medido() {
        let id = padrao().expect("identidade");
        let bin = build(&id, &dois_canais()).expect("sn.bin");
        assert_eq!(bin.len(), SIZE);
        assert_eq!(bin[0], RESERVED_FIRST_BYTE);
        assert_eq!(&bin[1..8], b"eserved");
        // Nome do slot 1 vem da identidade, não de constante.
        assert_eq!(&bin[NAME_LEN..NAME_LEN + 3], b"WYD");
        // Contadores: cabeçalho = canais + 1; slot 1 = grupo + 1.
        let contador = |slot: usize| {
            let i = COUNTER_OFFSET + slot * 4;
            i32::from_le_bytes([bin[i], bin[i + 1], bin[i + 2], bin[i + 3]])
        };
        assert_eq!(contador(0), 3);
        assert_eq!(contador(1), 2);
        assert_eq!(contador(2), 0);
    }

    #[test]
    fn grupo_sem_canal_nao_escreve_nome_nem_contador() {
        let id = padrao().expect("identidade");
        let bin = build(&id, &[0usize; 10]).expect("sn.bin");
        assert_eq!(&bin[NAME_LEN..NAME_LEN + 3], &[0, 0, 0]);
        let i = COUNTER_OFFSET + 4;
        assert_eq!(
            i32::from_le_bytes([bin[i], bin[i + 1], bin[i + 2], bin[i + 3]]),
            0
        );
    }

    #[test]
    fn contagem_por_grupo_ignora_celula_de_status() {
        let cells = vec![
            Cell {
                group: 1,
                channel: 0,
                addr: "http://192.168.100.3/serv00.htm".to_string(),
            },
            Cell {
                group: 1,
                channel: 1,
                addr: "192.168.100.3".to_string(),
            },
            Cell {
                group: 1,
                channel: 2,
                addr: "192.168.100.4".to_string(),
            },
        ];
        let contagem = canais_por_grupo(&cells);
        assert_eq!(contagem[1], 2);
        assert_eq!(contagem[0], 0);
        // Slot do arquivo = grupo do `.bin`: o grupo 1 escreve nome e contador 2.
        let bin = build(&padrao().expect("identidade"), &contagem).expect("sn.bin");
        assert_eq!(&bin[NAME_LEN..NAME_LEN + 3], b"WYD");
        let i = COUNTER_OFFSET + 4;
        assert_eq!(
            i32::from_le_bytes([bin[i], bin[i + 1], bin[i + 2], bin[i + 3]]),
            2
        );
    }
}
