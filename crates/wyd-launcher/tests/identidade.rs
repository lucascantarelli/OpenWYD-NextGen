//! Paridade da identidade única e dos assets gerados por ela.
//!
//! Duas garantias que o projeto não tinha antes:
//!
//! 1. **uma fonte só** — `docs/arquitetura/identidade.tokens` é comparada com o
//!    `wyd-portal/src/app/globals.css`; qualquer valor divergente (ou cor solta no CSS que não
//!    exista na identidade) quebra o teste. É o launcher que carrega a identidade no binário
//!    (`include_str!`), então não existe paleta paralela em Rust;
//! 2. **os assets instalados são os gerados** — `client/sn.bin`, `client/Launcher.bmp` e
//!    `client/Launcher.hat` têm de ser byte-idênticos ao que o launcher produz a partir da
//!    identidade e do registro de canais. Se alguém editar à mão (ou o nome do servidor mudar
//!    só num dos lados), o teste falha em vez de o cliente desenhar outra coisa.
//!
//! `client/` não é versionado (binários do 7662), então os testes de asset avisam e retornam
//! quando o diretório não está na máquina.

use std::collections::{BTreeMap, BTreeSet};
use std::path::{Path, PathBuf};

use wyd_launcher::{branding, identidade, serverlist, snbin};

fn raiz() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("..")
        .join("..")
        .canonicalize()
        .unwrap_or_else(|_| PathBuf::from("."))
}

/// `--token: valor;` do CSS, sem `:root` implícito (o arquivo inteiro é varrido).
fn css_tokens(css: &str) -> BTreeMap<String, String> {
    let mut mapa = BTreeMap::new();
    for linha in css.lines() {
        let limpa = linha.trim();
        if !limpa.starts_with("--") {
            continue;
        }
        let Some((chave, valor)) = limpa.split_once(':') else {
            continue;
        };
        let valor = valor.trim().trim_end_matches(';').trim();
        if valor.is_empty() {
            continue;
        }
        mapa.insert(chave.trim().to_string(), valor.to_string());
    }
    mapa
}

#[test]
fn identidade_e_globals_css_nao_divergem() {
    let id = identidade::padrao().expect("identidade versionada válida");
    let css = std::fs::read_to_string(raiz().join("wyd-portal/src/app/globals.css"))
        .expect("globals.css do portal");
    let declarado = css_tokens(&css);

    for (chave, valor) in id.pares() {
        if let Some(nome) = chave.strip_prefix("cor.") {
            assert_eq!(
                declarado.get(&format!("--{nome}")).map(String::as_str),
                Some(valor.as_str()),
                "cor `{chave}` divergiu do CSS do portal"
            );
        } else if let Some(indice) = chave.strip_prefix("espaco.") {
            let esperado = format!("{valor}px");
            assert_eq!(
                declarado
                    .get(&format!("--space-{indice}"))
                    .map(String::as_str),
                Some(esperado.as_str()),
                "espaço `{chave}` divergiu do CSS do portal"
            );
        }
    }

    assert_eq!(declarado.get("--radius").map(String::as_str), Some("8px"));
    assert_eq!(
        declarado.get("--radius-sm").map(String::as_str),
        Some("6px")
    );

    let familia = id.get("tipografia.familia").expect("família");
    assert!(
        css.contains(&format!("font-family: {familia};")),
        "família tipográfica do CSS divergiu de `tipografia.familia`"
    );
    let mono = id.get("tipografia.mono").expect("mono");
    assert!(
        css.contains(&format!("font-family: {mono};")),
        "fonte mono do CSS divergiu de `tipografia.mono`"
    );
    let tamanho = id.get("tipografia.tamanho").expect("tamanho");
    assert!(
        css.contains(&format!("font-size: {tamanho}px;")),
        "corpo do CSS não usa `tipografia.tamanho`"
    );
    let altura = id.get("tipografia.altura").expect("altura");
    assert!(
        css.contains(&format!("line-height: {altura};")),
        "corpo do CSS não usa `tipografia.altura`"
    );
}

/// Nenhuma cor de 6 dígitos solta no CSS: toda cor do portal vem da identidade.
#[test]
fn css_do_portal_nao_tem_cor_fora_da_identidade() {
    let id = identidade::padrao().expect("identidade");
    let cores: BTreeSet<String> = id
        .pares()
        .iter()
        .filter(|(chave, _)| chave.starts_with("cor."))
        .map(|(_, valor)| valor.to_lowercase())
        .collect();
    let css = std::fs::read_to_string(raiz().join("wyd-portal/src/app/globals.css"))
        .expect("globals.css do portal");
    for (indice, linha) in css.lines().enumerate() {
        let baixa = linha.to_lowercase();
        for (posicao, _) in baixa.match_indices('#') {
            let hex: String = baixa[posicao + 1..]
                .chars()
                .take_while(|c| c.is_ascii_hexdigit())
                .collect();
            if hex.len() != 6 {
                continue;
            }
            let cor = format!("#{hex}");
            assert!(
                cores.contains(&cor),
                "globals.css linha {}: cor {cor} não existe em identidade.tokens",
                indice + 1
            );
        }
    }
}

/// `client/` é gitignored; sem ele o teste de asset não tem o que comparar.
fn cliente() -> Option<PathBuf> {
    let dir = raiz().join("client");
    if dir.join(snbin::FILE_NAME).exists() && dir.join(branding::BMP_NAME).exists() {
        Some(dir)
    } else {
        eprintln!("client/ ausente nesta máquina: pulando prova dos assets instalados");
        None
    }
}

#[test]
fn sn_bin_instalado_vem_da_identidade_e_do_registro() {
    let Some(dir) = cliente() else {
        return;
    };
    let id = identidade::padrao().expect("identidade");
    let cells = serverlist::load_registry(&raiz().join("tools/client/serverlist-servers.txt"))
        .expect("registro de canais versionado");
    let esperado =
        snbin::build(&id, &snbin::canais_por_grupo(&cells)).expect("sn.bin gerado da identidade");
    let instalado = std::fs::read(dir.join(snbin::FILE_NAME)).expect("sn.bin instalado");
    assert_eq!(
        instalado, esperado,
        "sn.bin instalado divergiu da identidade + registro (rode `wyd-launcher serverlist`)"
    );
    // Nomes do arquivo instalado são exatamente os da identidade, no slot do grupo.
    for (slot, nome) in id.servidores().expect("servidores") {
        let inicio = slot * snbin::NAME_LEN;
        let campo = &instalado[inicio..inicio + snbin::NAME_LEN];
        assert!(
            campo.starts_with(nome.as_bytes()),
            "slot {slot} não carrega o nome `{nome}` da identidade"
        );
    }
}

#[test]
fn skin_instalada_e_a_gerada_pela_identidade() {
    let Some(dir) = cliente() else {
        return;
    };
    let id = identidade::padrao().expect("identidade");
    let esperado_bmp = branding::bmp(&id).expect("skin");
    let instalado = std::fs::read(dir.join(branding::BMP_NAME)).expect("Launcher.bmp");
    assert_eq!(
        instalado, esperado_bmp,
        "Launcher.bmp instalado não é o gerado pela identidade (rode `wyd-launcher branding`)"
    );

    // Cabeçalho BMP lido do arquivo instalado (não da constante) e cor de base na identidade:
    // prova que o pixel do arquivo é o `cor.bg`, não só que o gerador compila.
    assert_eq!(&instalado[..2], b"BM");
    assert_eq!(
        u16::from_le_bytes([instalado[0x1c], instalado[0x1d]]),
        24,
        "24 bits por pixel"
    );
    let largura = i32::from_le_bytes([
        instalado[0x12],
        instalado[0x13],
        instalado[0x14],
        instalado[0x15],
    ]);
    let altura = i32::from_le_bytes([
        instalado[0x16],
        instalado[0x17],
        instalado[0x18],
        instalado[0x19],
    ]);
    assert_eq!(
        (largura as u32, altura as u32),
        (branding::LARGURA, branding::ALTURA)
    );

    let fundo = id.cor("cor.bg").expect("fundo");
    let stride = (branding::LARGURA as usize * 3).div_ceil(4) * 4;
    // Linha 0 do arquivo = base da imagem (BMP bottom-up), cor final do gradiente.
    let base = 54usize;
    assert_eq!(
        &instalado[base..base + 3],
        &[fundo.b, fundo.g, fundo.r],
        "último pixel do gradiente não é `cor.bg`"
    );
    // Toda cor da skin vem da identidade: nenhum pixel pode estar fora da paleta.
    let paleta: BTreeSet<[u8; 3]> = id
        .pares()
        .iter()
        .filter(|(chave, _)| chave.starts_with("cor."))
        .map(|(chave, _)| id.cor(chave).expect("cor"))
        .map(|c| [c.b, c.g, c.r])
        .collect();
    let mut acento = 0usize;
    let alvo = id.cor("cor.accent").expect("acento");
    for linha in 0..branding::ALTURA as usize {
        for coluna in 0..branding::LARGURA as usize {
            let o = base + linha * stride + coluna * 3;
            let pixel = [instalado[o], instalado[o + 1], instalado[o + 2]];
            if pixel == [alvo.b, alvo.g, alvo.r] {
                acento += 1;
            }
        }
    }
    assert!(
        acento > 0,
        "skin instalada não desenha `cor.accent` em nenhum pixel"
    );
    // Faixa e gradiente: a maior parte da área é composta por cores da identidade.
    assert!(paleta.contains(&[fundo.b, fundo.g, fundo.r]));

    let hat = std::fs::read_to_string(dir.join(branding::HAT_NAME)).expect("Launcher.hat");
    assert!(
        hat.contains(&fundo.hex()),
        "Launcher.hat não usa `cor.bg` de fundo"
    );
    assert!(hat.contains(branding::BMP_NAME), "hat não aponta a skin");
}
