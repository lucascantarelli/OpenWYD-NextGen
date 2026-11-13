//! Frente Tauri mínima do launcher (CLIENT-08, ADR-0007).
//!
//! A janela não tem marca própria: ela pede `pagina` ao núcleo, que devolve o
//! HTML da identidade **efetiva** já resolvida (`configuracao::resolver_flags`),
//! com notícias do portal ou o aviso do fallback. O botão Jogar chama `jogar`,
//! que usa o mesmo pipeline `lancamento` da CLI — fail-closed e sem escrita
//! própria no diretório do jogo.
//!
//! Modos de linha de comando:
//! - `--instalar <dir>`: copia este executável para `<dir>/WYDLauncher.exe` (atômico, com backup) e sai.
//! - `--dir <dir>`: pasta do jogo (padrão: pasta deste executável).
//! - `--check`: resolve a identidade e imprime origem/pasta sem abrir janela.

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::path::{Path, PathBuf};
use std::sync::Mutex;

use wyd_launcher::{configuracao, gui, lancamento, portal_client};

/// Estado resolvido uma vez no boot: a janela nunca resolve identidade no meio do uso.
struct Estado {
    efetiva: configuracao::Efetiva,
    dir: PathBuf,
    /// Executável aberto pelo Jogar (padrão `WYD.exe`; `--exe` troca em teste).
    exe: String,
    noticias: gui::Noticias,
}

/// HTML da janela: identidade efetiva + notícias (ou o motivo do fallback).
#[tauri::command]
fn pagina(estado: tauri::State<'_, Mutex<Estado>>) -> Result<String, String> {
    let estado = estado
        .lock()
        .map_err(|_| "estado indisponível".to_string())?;
    gui::html_com_noticias(&estado.efetiva, &estado.noticias).map_err(|e| e.to_string())
}

/// Abre o jogo pelo pipeline compartilhado (manifesto → serverlist → sn.bin → spawn).
#[tauri::command]
fn jogar(estado: tauri::State<'_, Mutex<Estado>>) -> Result<String, String> {
    let (dir, exe, identidade) = {
        let estado = estado
            .lock()
            .map_err(|_| "estado indisponível".to_string())?;
        (
            estado.dir.clone(),
            estado.exe.clone(),
            estado.efetiva.identidade.clone(),
        )
    };
    let relatorio = lancamento::iniciar(
        &dir,
        &dir.join(lancamento::MANIFESTO_PADRAO),
        &dir.join(lancamento::REGISTRO_PADRAO),
        &identidade,
        &exe,
        &[],
        1,
    )
    .map_err(|e| {
        format!(
            "{e} (pasta {}, manifesto {}, executável {})",
            dir.display(),
            dir.join(lancamento::MANIFESTO_PADRAO).display(),
            exe
        )
    })?;
    Ok(format!(
        "jogo iniciado: {} janela(s) · build {} · serverlist {} byte(s)",
        relatorio.pids.len(),
        relatorio.build,
        relatorio.instalacao.bytes
    ))
}

fn arg_valor(args: &[String], nome: &str) -> Option<String> {
    let posicao = args.iter().position(|a| a == nome)?;
    args.get(posicao + 1).cloned()
}

/// Pasta do jogo quando nenhuma foi pedida: a pasta deste executável.
fn pasta_padrao() -> PathBuf {
    std::env::current_exe()
        .ok()
        .and_then(|exe| exe.parent().map(Path::to_path_buf))
        .unwrap_or_else(|| PathBuf::from("."))
}

/// Notícias da vitrine; falha vira aviso na janela (nunca impede abrir o jogo).
fn noticias(efetiva: &configuracao::Efetiva) -> gui::Noticias {
    let base = efetiva
        .portal
        .as_ref()
        .map(|portal| portal.base.clone())
        .or_else(|| {
            std::env::var(portal_client::PORTAL_ENV)
                .ok()
                .filter(|valor| !valor.is_empty())
        });
    let Some(base) = base else {
        return gui::Noticias {
            itens: Vec::new(),
            aviso: Some("notícias: sem portal configurado".to_string()),
        };
    };
    match portal_client::PortalApi::new(&base).and_then(|api| api.fetch_noticias()) {
        Ok(itens) => gui::Noticias { itens, aviso: None },
        Err(e) => gui::Noticias {
            itens: Vec::new(),
            aviso: Some(format!("notícias indisponíveis: {e}")),
        },
    }
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    if let Some(destino) = arg_valor(&args, "--instalar") {
        let origem = std::env::current_exe().expect("executável do launcher");
        match gui::instalar_launcher(Path::new(&destino), &origem) {
            Ok(r) => println!(
                "launcher instalado: {} ({} bytes, {})",
                r.path.display(),
                r.bytes,
                if r.changed { "novo" } else { "já idêntico" }
            ),
            Err(e) => {
                eprintln!("launcher não instalado: {e}");
                std::process::exit(1);
            }
        }
        return;
    }

    // `--dir`/`--exe` vêm da linha de comando; as mesmas chaves por ambiente
    // existem para o harness WebDriver, que não consegue injetar argv com
    // segurança no app empacotado.
    let dir = arg_valor(&args, "--dir")
        .filter(|valor| !valor.starts_with("--"))
        .or_else(|| std::env::var("WYD_LAUNCHER_DIR").ok())
        .map(PathBuf::from)
        .unwrap_or_else(pasta_padrao);
    let chave = std::env::var(configuracao::CHAVE_ENV)
        .ok()
        .filter(|valor| !valor.is_empty());
    let efetiva = configuracao::resolver_flags(&dir, None, chave.as_deref())
        .or_else(configuracao::Efetiva::embutida)
        .expect("identidade embutida sempre válida");

    if args.iter().any(|a| a == "--check") {
        println!("identidade: {:?}", efetiva.origem);
        println!("pasta do jogo: {}", dir.display());
        println!("portal: {:?}", efetiva.portal.as_ref().map(|p| &p.base));
        // O `--check` do jogador confere o manifesto real: é o mesmo portão do
        // Jogar, sem abrir jogo e sem escrever nada.
        let manifesto = dir.join(lancamento::MANIFESTO_PADRAO);
        let m = match wyd_launcher::manifest::load(&manifesto) {
            Ok(m) => m,
            Err(e) => {
                eprintln!("manifesto inválido ({}): {e}", manifesto.display());
                std::process::exit(2);
            }
        };
        let rel = wyd_launcher::manifest::verify(&m, &dir);
        let mut falhas = 0usize;
        for r in &rel.files {
            if r.status == wyd_launcher::manifest::FileStatus::Ok {
                println!("OK      {}", r.path);
            } else {
                println!("FALHA   {}: {}", r.path, r.status);
                falhas += 1;
            }
        }
        println!(
            "manifesto build {}: {} íntegros, {falhas} com problema",
            m.build,
            rel.files.len() - falhas
        );
        std::process::exit(if falhas == 0 { 0 } else { 1 });
    }

    let exe = arg_valor(&args, "--exe")
        .filter(|valor| !valor.starts_with("--"))
        .or_else(|| std::env::var("WYD_LAUNCHER_EXE").ok())
        .filter(|valor| !valor.is_empty())
        .unwrap_or_else(|| "WYD.exe".to_string());
    let estado = Estado {
        noticias: noticias(&efetiva),
        efetiva,
        dir,
        exe,
    };

    tauri::Builder::default()
        .manage(Mutex::new(estado))
        .invoke_handler(tauri::generate_handler![pagina, jogar])
        .run(tauri::generate_context!())
        .expect("erro ao abrir a janela do launcher");
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn argumento_vem_depois_da_flag() {
        let args = vec![
            "WYDLauncher".to_string(),
            "--dir".to_string(),
            "C:/jogo".to_string(),
        ];
        assert_eq!(arg_valor(&args, "--dir").as_deref(), Some("C:/jogo"));
        assert_eq!(arg_valor(&args, "--instalar"), None);
    }

    #[test]
    fn sem_portal_noticias_viram_aviso_sem_travar() {
        let efetiva = configuracao::Efetiva::embutida("teste").expect("identidade");
        // Sem `WYD_LAUNCHER_PORTAL` no ambiente de teste, a vitrine só pode avisar.
        if std::env::var(portal_client::PORTAL_ENV).is_err() {
            let noticias = noticias(&efetiva);
            assert!(noticias.itens.is_empty());
            assert!(noticias
                .aviso
                .as_deref()
                .unwrap_or_default()
                .contains("sem portal"));
        }
    }
}
