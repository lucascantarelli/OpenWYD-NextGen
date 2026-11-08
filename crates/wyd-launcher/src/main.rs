//! CLI fina do launcher: `check`, `launch`, `serverlist` e `update`. Lógica mora na biblioteca.

use std::path::Path;
use std::process::ExitCode;

use wyd_launcher::{bootstrap, distribuicao, manifest, serverlist};

const EXE_PADRAO: &str = "WYD.exe";

fn uso() -> String {
    format!(
        "uso:\n  {n} check <dir-do-jogo> <manifesto>\n  {n} launch <dir-do-jogo> <manifesto> <registro-de-canais> [--dry-run] [-- args...]\n  {n} serverlist <registro-de-canais> <dir-do-jogo> [--dry-run]\n  {n} update <dir-do-jogo> <pacote> --key-file <arq> [--dry-run]",
        n = wyd_launcher::LAUNCHER_NAME
    )
}

fn cmd_check(dir: &str, manifesto: &str) -> ExitCode {
    let m = match manifest::load(Path::new(manifesto)) {
        Ok(m) => m,
        Err(e) => {
            eprintln!("manifesto inválido: {e}");
            return ExitCode::from(2);
        }
    };
    let rel = manifest::verify(&m, Path::new(dir));
    let mut falhas = 0;
    for r in &rel.files {
        if r.status == manifest::FileStatus::Ok {
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
    if falhas == 0 {
        ExitCode::SUCCESS
    } else {
        ExitCode::from(1)
    }
}

/// Gera e instala o `serverlist.bin` a partir do registro de canais.
///
/// Devolve o relatório quando instalou; erro tipado (sem instalar nada) quando o
/// registro é inválido.
fn instalar_serverlist(
    dir: &str,
    registro: &str,
    dry_run: bool,
) -> Result<serverlist::InstallReport, serverlist::ServerlistError> {
    let bytes = serverlist::generate_from_registry(Path::new(registro))?;
    let cells = serverlist::decode(&bytes)?;
    println!(
        "registro {registro}: {} célula(s) → {} bytes",
        cells.len(),
        bytes.len()
    );
    for c in &cells {
        println!("  grupo {} canal {} → {}", c.group, c.channel, c.addr);
    }
    if dry_run {
        return Ok(serverlist::InstallReport {
            path: Path::new(dir).join(serverlist::FILE_NAME),
            bytes: bytes.len(),
            backup: None,
            changed: false,
        });
    }
    let r = serverlist::install(Path::new(dir), &bytes)?;
    Ok(r)
}

fn relatar_instalacao(r: &serverlist::InstallReport) {
    if r.changed {
        match &r.backup {
            Some(b) => println!(
                "serverlist instalado: {} ({} bytes, backup {})",
                r.path.display(),
                r.bytes,
                b.display()
            ),
            None => println!(
                "serverlist instalado: {} ({} bytes, sem backup: não havia arquivo)",
                r.path.display(),
                r.bytes
            ),
        }
    } else {
        println!(
            "serverlist já estava idêntico: {} ({} bytes, nada reescrito)",
            r.path.display(),
            r.bytes
        );
    }
}

fn cmd_serverlist(registro: &str, dir: &str, dry_run: bool) -> ExitCode {
    match instalar_serverlist(dir, registro, dry_run) {
        Ok(r) => {
            if dry_run {
                println!("dry-run: geraria {} ({} bytes)", r.path.display(), r.bytes);
            } else {
                relatar_instalacao(&r);
            }
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("serverlist recusado: {e}");
            ExitCode::from(1)
        }
    }
}

fn cmd_launch(
    dir: &str,
    manifesto: &str,
    registro: &str,
    dry_run: bool,
    extras: &[String],
) -> ExitCode {
    // Etapa 1 — portão do manifesto (ADR-0003). Falhou aqui, nada é escrito:
    // o `serverlist.bin` existente e o jogo ficam como estavam.
    let m = match manifest::load(Path::new(manifesto)) {
        Ok(m) => m,
        Err(e) => {
            eprintln!("manifesto inacessível/inválido — nada será instalado nem iniciado: {e}");
            return ExitCode::from(2);
        }
    };
    let dir_p = Path::new(dir);
    let rel = manifest::verify(&m, dir_p);
    if !rel.clean() {
        eprintln!("verificação falhou — serverlist não é tocado e o jogo não será iniciado:");
        for f in rel.failures() {
            eprintln!("  {}: {}", f.path, f.status);
        }
        return ExitCode::from(1);
    }

    // Etapa 2 — gerar/validar o serverlist a partir do registro versionado.
    let instalacao = match instalar_serverlist(dir, registro, dry_run) {
        Ok(r) => r,
        Err(e) => {
            eprintln!("serverlist recusado — jogo não será iniciado: {e}");
            return ExitCode::from(1);
        }
    };

    // Etapa 3 — instalação atômica com backup.
    if dry_run {
        println!(
            "dry-run: instalaria {} ({} bytes)",
            instalacao.path.display(),
            instalacao.bytes
        );
    } else {
        relatar_instalacao(&instalacao);
    }

    // Etapa 4 — lançamento.
    let plano = match bootstrap::plan(dir_p, EXE_PADRAO, extras) {
        Ok(p) => p,
        Err(e) => {
            eprintln!("lançamento impossível: {e}");
            return ExitCode::from(1);
        }
    };
    if dry_run {
        println!("dry-run: {}", bootstrap::describe(&plano));
        return ExitCode::SUCCESS;
    }
    match bootstrap::spawn(&plano) {
        Ok(filho) => {
            println!("jogo iniciado (pid {})", filho.id());
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("falha ao iniciar: {e}");
            ExitCode::from(1)
        }
    }
}

fn cmd_update(dir: &str, pacote: &str, key_file: &str, dry_run: bool) -> ExitCode {
    let chave = match distribuicao::load_key(Path::new(key_file)) {
        Ok(k) => k,
        Err(e) => {
            eprintln!("chave inválida: {e}");
            return ExitCode::from(2);
        }
    };
    if dry_run {
        match distribuicao::plan(Path::new(dir), Path::new(pacote), &chave) {
            Ok(p) => {
                println!(
                    "plano (build {}): {} arquivo(s) mudariam",
                    p.build,
                    p.mudariam.len()
                );
                for m in &p.mudariam {
                    println!("  {m}");
                }
                ExitCode::SUCCESS
            }
            Err(e) => {
                eprintln!("pacote recusado: {e}");
                ExitCode::from(1)
            }
        }
    } else {
        match distribuicao::update(Path::new(dir), Path::new(pacote), &chave) {
            Ok(r) => {
                println!(
                    "update aplicado: {} arquivo(s), build {}, backup {}",
                    r.aplicados,
                    r.build,
                    r.backup.display()
                );
                ExitCode::SUCCESS
            }
            Err(e) => {
                eprintln!("update falhou: {e}");
                ExitCode::from(1)
            }
        }
    }
}

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().collect();
    if args.len() < 4 {
        eprintln!("{}", uso());
        return ExitCode::from(2);
    }
    match args[1].as_str() {
        "check" => cmd_check(&args[2], &args[3]),
        "serverlist" => {
            let dry_run = args[4..].iter().any(|a| a == "--dry-run");
            cmd_serverlist(&args[2], &args[3], dry_run)
        }
        "update" => {
            let mut key_file = String::new();
            let mut dry_run = false;
            let mut resto = args[4..].iter();
            while let Some(a) = resto.next() {
                if a == "--key-file" {
                    key_file = resto.next().cloned().unwrap_or_default();
                } else if a == "--dry-run" {
                    dry_run = true;
                }
            }
            if key_file.is_empty() {
                eprintln!("falta --key-file <arq>");
                eprintln!("{}", uso());
                return ExitCode::from(2);
            }
            cmd_update(&args[2], &args[3], &key_file, dry_run)
        }
        "launch" => {
            let Some(registro) = args.get(4).filter(|a| !a.starts_with("--")) else {
                eprintln!("falta <registro-de-canais>");
                eprintln!("{}", uso());
                return ExitCode::from(2);
            };
            let mut dry_run = false;
            let mut extras: Vec<String> = Vec::new();
            let mut apos_travessao = false;
            for a in &args[5..] {
                if apos_travessao {
                    extras.push(a.clone());
                } else if a == "--dry-run" {
                    dry_run = true;
                } else if a == "--" {
                    apos_travessao = true;
                }
            }
            cmd_launch(&args[2], &args[3], registro, dry_run, &extras)
        }
        _ => {
            eprintln!("{}", uso());
            ExitCode::from(2)
        }
    }
}
