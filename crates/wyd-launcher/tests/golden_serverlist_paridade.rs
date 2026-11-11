//! Paridade de bytes entre o gerador Rust e o gerador de referência Node.
//!
//! O `serverlist.bin` tem uma implementação de formato só no Rust
//! (`crates/wyd-launcher/src/serverlist.rs`); o Node (`tools/client/serverlist.mjs`)
//! é a referência de operação que o time usa para inspecionar/gerar à mão. Se os
//! dois divergirem, um `.bin` gerado pela operação deixa de ser o que o launcher
//! instalaria — este teste falha antes disso chegar no cliente.
//!
//! Roda `node` de verdade sobre o registro versionado. Sem `node` no PATH o teste
//! avisa e retorna (é dependência de ferramenta, não de runtime do jogo); com
//! `node` presente, qualquer diferença de byte é falha.

use std::path::{Path, PathBuf};
use std::process::Command;

use wyd_launcher::serverlist;

fn raiz() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("..")
        .join("..")
        .canonicalize()
        .unwrap_or_else(|_| PathBuf::from("."))
}

fn registro() -> PathBuf {
    raiz().join("tools/client/serverlist-servers.txt")
}

fn tmpdir(tag: &str) -> PathBuf {
    let d = std::env::temp_dir().join(format!("wyd-paridade-{tag}-{}", std::process::id()));
    std::fs::create_dir_all(&d).unwrap();
    d
}

#[test]
fn bytes_do_rust_sao_identicos_aos_do_gerador_node() {
    let reg = registro();
    let rust = serverlist::generate_from_registry(&reg).expect("registro versionado válido");
    assert_eq!(rust.len(), 7040, "grade 10×11×64");

    let dir = tmpdir("geral");
    let saida = dir.join("node-serverlist.bin");
    let status = Command::new("node")
        .arg("tools/client/serverlist.mjs")
        .arg("--in")
        .arg(&reg)
        .arg("--out")
        .arg(&saida)
        .current_dir(raiz())
        .status();
    let status = match status {
        Ok(s) => s,
        Err(e) => {
            eprintln!("node indisponível ({e}) — paridade não executada neste ambiente");
            return;
        }
    };
    assert!(status.success(), "gerador Node falhou: {status:?}");

    let node = std::fs::read(&saida).expect("bin do node");
    assert_eq!(rust.len(), node.len(), "tamanhos divergentes");
    let diferencias: Vec<usize> = rust
        .iter()
        .zip(node.iter())
        .enumerate()
        .filter(|(_, (a, b))| a != b)
        .map(|(i, _)| i)
        .collect();
    assert!(
        diferencias.is_empty(),
        "bytes divergentes em {} posição(ões), começando em {:?}",
        diferencias.len(),
        &diferencias[..diferencias.len().min(8)]
    );
    std::fs::remove_dir_all(&dir).ok();
}

#[test]
fn limite_de_32_caracteres_e_cobrado_nos_dois_lados() {
    // O Rust recusa a URL de 37 caracteres com erro nomeado...
    let ruim = "0 0 http://192.168.100.3:8281/serv00.htm\n0 1 192.168.100.3\n";
    let dir = tmpdir("limite");
    let entrada = dir.join("ruim.txt");
    std::fs::write(&entrada, ruim).unwrap();
    let erro = serverlist::generate_from_registry(&entrada).unwrap_err();
    assert!(erro.to_string().contains("32"), "{erro}");

    // ...e o Node também, para os dois lados contarem a mesma história.
    let status = Command::new("node")
        .arg("tools/client/serverlist.mjs")
        .arg("--in")
        .arg(&entrada)
        .arg("--out")
        .arg(dir.join("x.bin"))
        .current_dir(raiz())
        .status();
    match status {
        Ok(s) => assert!(!s.success(), "Node aceitou URL acima do limite"),
        Err(e) => eprintln!("node indisponível ({e}) — lado Node não verificado"),
    }
    std::fs::remove_dir_all(&dir).ok();
}

/// `serverlist.txt` legível: URL de status + **um host:8281 por canal**, na ordem das células.
/// Divergir do `.bin` aqui confunde a operação (foi o que aconteceu com dois canais: o `.bin`
/// tinha as duas células e o `.txt` continuava com um host só).
#[test]
fn txt_legivel_tem_um_host_por_canal_igual_ao_bin() {
    use wyd_launcher::serverlist::{self, Cell};

    let cells = vec![
        Cell {
            group: 1,
            channel: 0,
            addr: "http://192.168.100.3/serv00.htm".into(),
        },
        Cell {
            group: 1,
            channel: 1,
            addr: "192.168.100.3".into(),
        },
        Cell {
            group: 1,
            channel: 2,
            addr: "192.168.100.4".into(),
        },
    ];
    let bin = serverlist::build(&cells);
    let txt = serverlist::render_txt(&serverlist::decode(&bin).expect("bin válido"));

    let linhas: Vec<&str> = txt.lines().collect();
    assert_eq!(linhas[0], "http://192.168.100.3/serv00.htm");
    assert_eq!(&linhas[1..], &["192.168.100.3:8281", "192.168.100.4:8281"]);

    // Paridade célula a célula: todo canal 1..10 do bin aparece no texto na mesma ordem.
    let mut canais: Vec<_> = serverlist::decode(&bin)
        .expect("bin válido")
        .into_iter()
        .filter(|c| c.channel >= 1)
        .collect();
    canais.sort_by_key(|c| (c.group, c.channel));
    assert_eq!(canais.len(), linhas.len() - 1);
    for (canal, linha) in canais.iter().zip(&linhas[1..]) {
        assert_eq!(format!("{}:8281", canal.addr), *linha);
    }
}
