//! Golden cross-language: a assinatura Ed25519 produzida por
//! `tools/launcher/sign-package.mjs` (Node/OpenSSL) tem de verificar no launcher
//! com a chave pública embutida (`distribuicao::CHAVE_PUBLICA_HEX`).
//!
//! A fixture `tests/golden/pacote_ed25519/` é pública (manifesto, payload e
//! assinatura; nenhum segredo). A chave privada que a assinou não está no
//! repositório — se este teste falhar, ou a constante mudou sem re-assinar a
//! fixture, ou o verificador divergiu do emissor.

use std::path::Path;

use wyd_launcher::distribuicao;

#[test]
fn pacote_assinado_pelo_node_verifica_com_a_chave_embutida() {
    let pkg = Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/golden/pacote_ed25519");
    let chave = distribuicao::chave_embutida().expect("chave pública embutida válida");
    let m = distribuicao::verify_package(&pkg, &chave).expect("assinatura do Node deve verificar");
    assert_eq!(m.build, 7662);
    assert_eq!(m.files.len(), 1);
    assert_eq!(m.files[0].path, "jogo.dat");
}

#[test]
fn pacote_do_node_com_payload_adulterado_e_recusado() {
    // Mesmo pacote, payload trocado: a assinatura cobre o manifesto, então a
    // recusa vem da verificação de hash — prova que as duas camadas valem.
    let origem = Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/golden/pacote_ed25519");
    let copia = std::env::temp_dir().join(format!("wyd-golden-{}", std::process::id()));
    std::fs::create_dir_all(copia.join("arquivos")).unwrap();
    for nome in ["manifesto", "assinatura"] {
        std::fs::copy(origem.join(nome), copia.join(nome)).unwrap();
    }
    std::fs::write(copia.join("arquivos/jogo.dat"), b"outro-conteudo").unwrap();

    let chave = distribuicao::chave_embutida().unwrap();
    let r = distribuicao::verify_package(&copia, &chave);
    assert!(
        matches!(r, Err(distribuicao::DistError::PayloadCorrupt { .. })),
        "payload trocado devia ser recusado: {r:?}"
    );
    std::fs::remove_dir_all(&copia).ok();
}
