//! Integração da entrega do `client.zip`: fonte HTTP local, hash, cache e falhas.
//!
//! Sobe um servidor TCP mínimo no loopback (sem dependências externas), serve o ZIP
//! e conta requisições. Prova que a segunda instalação usa o cache sem nova conexão,
//! e que hash errado, arquivo corrompido e fonte fora do ar recusam sem deixar
//! `.part` no cache.
use std::fs;
use std::io::{Read, Write};
use std::net::TcpListener;
use std::path::PathBuf;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::Duration;

use wyd_launcher::cliente_zip::{self, ClienteZipError};

fn tmpdir(tag: &str) -> PathBuf {
    use std::sync::atomic::AtomicU64;
    static N: AtomicU64 = AtomicU64::new(0);
    let d = std::env::temp_dir().join(format!(
        "wyd-integracao-zip-{tag}-{}-{}",
        std::process::id(),
        N.fetch_add(1, Ordering::Relaxed)
    ));
    let _ = fs::remove_dir_all(&d);
    fs::create_dir_all(&d).unwrap();
    d
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

/// ZIP mínimo com uma entrada **stored**.
fn zip_stored(nome: &str, dados: &[u8]) -> Vec<u8> {
    let mut out = Vec::new();
    out.extend_from_slice(&[0x50, 0x4b, 0x03, 0x04]);
    out.extend_from_slice(&20u16.to_le_bytes());
    out.extend_from_slice(&0u16.to_le_bytes());
    out.extend_from_slice(&0u16.to_le_bytes());
    out.extend_from_slice(&0u32.to_le_bytes());
    out.extend_from_slice(&crc32(dados).to_le_bytes());
    out.extend_from_slice(&(dados.len() as u32).to_le_bytes());
    out.extend_from_slice(&(dados.len() as u32).to_le_bytes());
    out.extend_from_slice(&(nome.len() as u16).to_le_bytes());
    out.extend_from_slice(&0u16.to_le_bytes());
    out.extend_from_slice(nome.as_bytes());
    out.extend_from_slice(dados);
    let inicio = 0u32;
    let mut central = Vec::new();
    central.extend_from_slice(&[0x50, 0x4b, 0x01, 0x02]);
    central.extend_from_slice(&20u16.to_le_bytes());
    central.extend_from_slice(&20u16.to_le_bytes());
    central.extend_from_slice(&0u16.to_le_bytes());
    central.extend_from_slice(&0u16.to_le_bytes());
    central.extend_from_slice(&0u32.to_le_bytes());
    central.extend_from_slice(&crc32(dados).to_le_bytes());
    central.extend_from_slice(&(dados.len() as u32).to_le_bytes());
    central.extend_from_slice(&(dados.len() as u32).to_le_bytes());
    central.extend_from_slice(&(nome.len() as u16).to_le_bytes());
    central.extend_from_slice(&0u16.to_le_bytes());
    central.extend_from_slice(&0u16.to_le_bytes());
    central.extend_from_slice(&0u16.to_le_bytes());
    central.extend_from_slice(&0u16.to_le_bytes());
    central.extend_from_slice(&0u32.to_le_bytes());
    central.extend_from_slice(&inicio.to_le_bytes());
    central.extend_from_slice(nome.as_bytes());
    let offset_central = out.len() as u32;
    out.extend_from_slice(&central);
    out.extend_from_slice(&[0x50, 0x4b, 0x05, 0x06]);
    out.extend_from_slice(&0u16.to_le_bytes());
    out.extend_from_slice(&0u16.to_le_bytes());
    out.extend_from_slice(&1u16.to_le_bytes());
    out.extend_from_slice(&1u16.to_le_bytes());
    out.extend_from_slice(&(central.len() as u32).to_le_bytes());
    out.extend_from_slice(&offset_central.to_le_bytes());
    out.extend_from_slice(&0u16.to_le_bytes());
    out
}

fn preparar(tag: &str, conteudo: &[u8]) -> PathBuf {
    let dir = tmpdir(tag);
    fs::write(dir.join("jogo.dat"), b"antigo").unwrap();
    fs::write(
        dir.join("canais.txt"),
        "# canais\n0 0 http://127.0.0.1/serv00.htm\n0 1 127.0.0.1\n",
    )
    .unwrap();
    let texto = format!(
        "build 7662\nprotocol 12000\n{} {} jogo.dat\n",
        wyd_launcher::sha256::hex_of(conteudo),
        conteudo.len()
    );
    fs::write(dir.join("manifest.txt"), texto).unwrap();
    dir
}

/// Servidor loopback que responde `respostas` requisições com o mesmo corpo.
fn servir(corpo: Vec<u8>, respostas: usize) -> (String, Arc<AtomicUsize>, thread::JoinHandle<()>) {
    let ouvinte = TcpListener::bind("127.0.0.1:0").unwrap();
    let endereco = ouvinte.local_addr().unwrap();
    let contador = Arc::new(AtomicUsize::new(0));
    let visto = Arc::clone(&contador);
    let handle = thread::spawn(move || {
        for _ in 0..respostas {
            let Ok((mut conexao, _)) = ouvinte.accept() else {
                break;
            };
            let mut pedido = [0u8; 2048];
            let _ = conexao.read(&mut pedido);
            visto.fetch_add(1, Ordering::SeqCst);
            let cabecalho = format!(
                "HTTP/1.1 200 OK\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
                corpo.len()
            );
            let _ = conexao.write_all(cabecalho.as_bytes());
            let _ = conexao.write_all(&corpo);
        }
    });
    (format!("http://{endereco}/client.zip"), contador, handle)
}

fn partes_de_cache(dir: &std::path::Path) -> Vec<String> {
    let cache = dir.join(cliente_zip::CACHE_DIR);
    match fs::read_dir(cache) {
        Ok(entradas) => entradas
            .filter_map(|e| e.ok())
            .map(|e| e.file_name().to_string_lossy().into_owned())
            .filter(|nome| nome.ends_with(".part"))
            .collect(),
        Err(_) => Vec::new(),
    }
}

#[test]
fn baixa_valida_instala_e_reusa_cache_na_segunda_execucao() {
    let conteudo = b"cliente v2";
    let dir = preparar("sucesso", conteudo);
    let zip = zip_stored("jogo.dat", conteudo);
    let hash = wyd_launcher::sha256::hex_of(&zip);
    let (url, requisicoes, servidor) = servir(zip.clone(), 1);

    let primeira = cliente_zip::instalar(&dir, &url, None, Some(&hash)).unwrap();
    assert!(!primeira.do_cache, "primeira execução baixa da fonte");
    assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), conteudo);
    assert_eq!(
        fs::read(cliente_zip::cache_path(&dir, &hash)).unwrap(),
        zip,
        "pacote verificado fica no cache nomeado pelo hash"
    );

    let segunda = cliente_zip::instalar(&dir, &url, None, Some(&hash)).unwrap();
    assert!(segunda.do_cache, "segunda execução reusa o cache");
    assert_eq!(
        requisicoes.load(Ordering::SeqCst),
        1,
        "cache evita nova requisição"
    );
    assert!(partes_de_cache(&dir).is_empty());
    servidor.join().unwrap();
}

#[test]
fn cache_corrompido_e_ignorado_e_o_pacote_e_baixado_de_novo() {
    let conteudo = b"cliente v2";
    let dir = preparar("cache-ruim", conteudo);
    let zip = zip_stored("jogo.dat", conteudo);
    let hash = wyd_launcher::sha256::hex_of(&zip);
    let (url, requisicoes, servidor) = servir(zip.clone(), 1);
    let cache = cliente_zip::cache_path(&dir, &hash);
    fs::create_dir_all(cache.parent().unwrap()).unwrap();
    fs::write(&cache, b"lixo no cache").unwrap();

    let rel = cliente_zip::instalar(&dir, &url, None, Some(&hash)).unwrap();
    assert!(!rel.do_cache, "cache inválido não pode ser aceito");
    assert_eq!(requisicoes.load(Ordering::SeqCst), 1);
    assert_eq!(fs::read(&cache).unwrap(), zip, "cache é reescrito correto");
    assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), conteudo);
    servidor.join().unwrap();
}

#[test]
fn hash_divergente_falha_sem_gravar_cache_nem_cliente() {
    let dir = preparar("hash", b"cliente v2");
    let zip = zip_stored("jogo.dat", b"cliente v2");
    let (url, requisicoes, servidor) = servir(zip, 1);
    let errado = "a".repeat(64);

    let erro = cliente_zip::instalar(&dir, &url, None, Some(&errado)).unwrap_err();
    assert!(
        matches!(erro, ClienteZipError::HashDivergente { .. }),
        "{erro}"
    );
    assert_eq!(requisicoes.load(Ordering::SeqCst), 1);
    assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"antigo");
    assert!(partes_de_cache(&dir).is_empty(), "sem arquivo parcial");
    assert!(!cliente_zip::cache_path(&dir, &errado).exists());
    servidor.join().unwrap();
}

#[test]
fn download_corrompido_nao_deixa_parcial_no_cache() {
    let dir = preparar("corrompido", b"cliente v2");
    let zip = zip_stored("jogo.dat", b"cliente v2");
    let hash_esperado = wyd_launcher::sha256::hex_of(&zip);
    let truncado = zip[..zip.len() / 2].to_vec();
    let (url, _, servidor) = servir(truncado, 1);

    let erro = cliente_zip::instalar(&dir, &url, None, Some(&hash_esperado)).unwrap_err();
    assert!(
        matches!(erro, ClienteZipError::HashDivergente { .. }),
        "{erro}"
    );
    assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"antigo");
    assert!(partes_de_cache(&dir).is_empty(), "sem arquivo parcial");
    servidor.join().unwrap();
}

#[test]
fn url_sem_hash_e_recusada_antes_da_rede() {
    let dir = preparar("sem-hash", b"cliente v2");
    let erro =
        cliente_zip::instalar(&dir, "http://127.0.0.1:9/client.zip", None, None).unwrap_err();
    assert!(matches!(erro, ClienteZipError::HashAusente), "{erro}");
    assert!(partes_de_cache(&dir).is_empty());
}

#[test]
fn fonte_indisponivel_falha_com_mensagem_e_sem_lixo() {
    let dir = preparar("fora", b"cliente v2");
    let hash = "b".repeat(64);
    // Sobe e derruba o listener para garantir porta fechada.
    let (url, _, servidor) = {
        let ouvinte = TcpListener::bind("127.0.0.1:0").unwrap();
        let endereco = ouvinte.local_addr().unwrap();
        drop(ouvinte);
        (
            format!("http://{endereco}/client.zip"),
            Arc::new(AtomicUsize::new(0)),
            thread::spawn(|| {}),
        )
    };
    thread::sleep(Duration::from_millis(20));
    let erro = cliente_zip::instalar(&dir, &url, None, Some(&hash)).unwrap_err();
    assert!(matches!(erro, ClienteZipError::Download(_)), "{erro}");
    assert!(partes_de_cache(&dir).is_empty(), "sem arquivo parcial");
    assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"antigo");
    servidor.join().unwrap();
}

/// Smoke real de HTTPS (GitHub Releases): asset público pequeno e hash fixo.
/// Fica `#[ignore]` para o `make check` não depender de rede; rode com
/// `cargo test -p wyd-launcher --test integracao_cliente_zip -- --ignored`.
#[test]
#[ignore = "rede: baixa asset público do GitHub Releases"]
fn baixa_asset_publico_https_com_hash_e_reusa_cache() {
    const URL: &str =
        "https://github.com/cli/cli/releases/download/v2.62.0/gh_2.62.0_checksums.txt";
    const HASH: &str = "89dc6f5225aa0c70d6f90950f5246225afff2f423f3d242f7d8813fe9993af4d";
    let dir = tmpdir("github");
    let (primeiro, cache) = cliente_zip::obter_zip(&dir, URL, Some(HASH)).unwrap();
    assert!(!cache, "primeira chamada baixa");
    assert_eq!(primeiro.len(), 1856);
    let (segundo, cache) = cliente_zip::obter_zip(&dir, URL, Some(HASH)).unwrap();
    assert!(cache, "segunda chamada vem do cache");
    assert_eq!(primeiro, segundo);
}

#[test]
fn override_local_dispensa_hash_e_nao_polui_cache() {
    let dir = preparar("local", b"cliente v2");
    let zip = zip_stored("jogo.dat", b"cliente v2");
    let origem = dir.join("pacote-local.zip");
    fs::write(&origem, &zip).unwrap();
    let rel = cliente_zip::instalar(&dir, origem.to_str().unwrap(), None, None).unwrap();
    assert!(!rel.do_cache);
    assert_eq!(fs::read(dir.join("jogo.dat")).unwrap(), b"cliente v2");
    assert!(!dir.join(cliente_zip::CACHE_DIR).exists());
}
