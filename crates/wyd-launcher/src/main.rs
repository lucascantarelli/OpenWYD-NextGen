//! CLI fina do launcher: `check`, `launch`, `serverlist`, `branding` e `update`.
//! Lógica mora na biblioteca; aqui só parsing de argumentos e exit codes.

use std::path::Path;
use std::process::ExitCode;

use wyd_launcher::{
    bootstrap, branding, distribuicao, gui, identidade, manifest, serverlist, snbin, wyt,
};

const EXE_PADRAO: &str = "WYD.exe";

fn uso() -> String {
    format!(
        "uso:\n  {n} check <dir-do-jogo> <manifesto>\n  {n} launch <dir-do-jogo> <manifesto> <registro-de-canais> [--instances N] [--dry-run] [-- args...]\n  {n} serverlist <registro-de-canais> <dir-do-jogo> [--dry-run]\n  {n} branding <dir-do-jogo> [--dry-run]\n  {n} gui <dir-do-jogo> [--dry-run]\n  {n} retheme <origem.wyt> <destino.wyt> [--dry-run]\n  {n} update <dir-do-jogo> <pacote> --key-file <arq> [--dry-run]",
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
    let identidade = identidade::padrao()
        .map_err(|erro| serverlist::ServerlistError::Identidade(erro.to_string()))?;
    let sn = snbin::build(&identidade, &snbin::canais_por_grupo(&cells))
        .map_err(|erro| serverlist::ServerlistError::Identidade(erro.to_string()))?;
    println!(
        "sn.bin: {} servidor(es) da identidade → {} bytes",
        identidade.servidores().map(|s| s.len()).unwrap_or(0),
        sn.len()
    );
    if dry_run {
        println!(
            "dry-run: geraria {} ({} bytes) e {} ({} bytes) em {dir}",
            Path::new(dir).join(serverlist::FILE_NAME).display(),
            bytes.len(),
            snbin::FILE_NAME,
            sn.len()
        );
        return Ok(serverlist::InstallReport {
            path: Path::new(dir).join(serverlist::FILE_NAME),
            bytes: bytes.len(),
            backup: None,
            changed: false,
        });
    }
    let r = serverlist::install(Path::new(dir), &bytes)?;
    // `sn.bin` anda junto: nomes da identidade única (`servidor.N`) e grupos do `.bin`
    // recém-gerado. Antes o nome era constante no gerador ("NextGen") — fontes divergentes.
    relatar_asset(&snbin::install(Path::new(dir), &sn)?);
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

fn cmd_branding(dir: &str, dry_run: bool) -> ExitCode {
    let id = match identidade::padrao() {
        Ok(id) => id,
        Err(e) => {
            eprintln!("identidade inválida: {e}");
            return ExitCode::from(2);
        }
    };
    let bmp = match branding::bmp(&id) {
        Ok(bytes) => bytes,
        Err(e) => {
            eprintln!("skin: {e}");
            return ExitCode::from(1);
        }
    };
    let hat = match branding::hat(&id) {
        Ok(html) => html,
        Err(e) => {
            eprintln!("html: {e}");
            return ExitCode::from(1);
        }
    };
    println!(
        "identidade: fundo {} · acento {} · skin {}×{} · {} servidor(es)",
        id.get("cor.bg").unwrap_or("?"),
        id.get("cor.accent").unwrap_or("?"),
        branding::LARGURA,
        branding::ALTURA,
        id.servidores().map(|s| s.len()).unwrap_or(0)
    );
    if dry_run {
        println!(
            "dry-run: geraria {} ({} bytes) e {} ({} bytes) em {dir}",
            branding::BMP_NAME,
            bmp.len(),
            branding::HAT_NAME,
            hat.len()
        );
        return ExitCode::SUCCESS;
    }
    match branding::install(
        Path::new(dir),
        &[
            (branding::BMP_NAME, bmp.as_slice()),
            (branding::HAT_NAME, hat.as_bytes()),
        ],
    ) {
        Ok(relatorios) => {
            for r in &relatorios {
                relatar_asset(r);
            }
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("skin recusado: {e}");
            ExitCode::from(1)
        }
    }
}

fn relatar_asset(r: &branding::InstallReport) {
    if r.changed {
        match &r.backup {
            Some(b) => println!(
                "instalado: {} ({} bytes, backup {})",
                r.path.display(),
                r.bytes,
                b.display()
            ),
            None => println!(
                "instalado: {} ({} bytes, sem backup: não havia arquivo)",
                r.path.display(),
                r.bytes
            ),
        }
    } else {
        println!(
            "já estava idêntico: {} ({} bytes)",
            r.path.display(),
            r.bytes
        );
    }
}

fn cmd_launch(
    dir: &str,
    manifesto: &str,
    registro: &str,
    dry_run: bool,
    extras: &[String],
    instancias: usize,
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
        for janela in 1..=instancias {
            println!("dry-run: janela {janela} de {instancias} (mesmo executável)");
        }
        return ExitCode::SUCCESS;
    }
    // Multiboxing é prática normal: N janelas independentes, sem trava de instância única.
    let mut pids = Vec::new();
    for janela in 1..=instancias {
        match bootstrap::spawn(&plano) {
            Ok(filho) => pids.push(filho.id()),
            Err(e) => {
                eprintln!("falha ao iniciar a janela {janela} de {instancias}: {e}");
                return ExitCode::from(1);
            }
        }
    }
    let lista = pids
        .iter()
        .map(u32::to_string)
        .collect::<Vec<_>>()
        .join(", ");
    println!("jogo iniciado: {} janela(s) (pids: {lista})", pids.len());
    ExitCode::SUCCESS
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

/// Gera a janela do launcher GUI (contrato visual da FASE-08-8) na identidade única.
fn cmd_gui(dir: &str, dry_run: bool) -> ExitCode {
    let identidade = match identidade::padrao() {
        Ok(id) => id,
        Err(e) => {
            eprintln!("identidade inválida: {e}");
            return ExitCode::from(2);
        }
    };
    let html = match gui::html(&identidade) {
        Ok(html) => html,
        Err(e) => {
            eprintln!("janela não gerada: {e}");
            return ExitCode::from(2);
        }
    };
    if dry_run {
        println!(
            "dry-run: geraria {} ({} bytes) em {dir}",
            gui::GUI_NAME,
            html.len()
        );
        return ExitCode::SUCCESS;
    }
    match gui::install(Path::new(dir), &html) {
        Ok(r) => {
            relatar_asset(&r);
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("falha ao escrever a janela: {e}");
            ExitCode::from(1)
        }
    }
}

/// Re-tematiza um `.wyt` do cliente na identidade única (o destino recebe `.bak`).
fn cmd_retheme(origem: &str, destino: &str, dry_run: bool) -> ExitCode {
    let bytes = match std::fs::read(origem) {
        Ok(bytes) => bytes,
        Err(e) => {
            eprintln!("origem ilegível ({origem}): {e}");
            return ExitCode::from(2);
        }
    };
    let identidade = match identidade::padrao() {
        Ok(id) => id,
        Err(e) => {
            eprintln!("identidade inválida: {e}");
            return ExitCode::from(2);
        }
    };
    let saida = match wyt::retheme(&bytes, &identidade) {
        Ok(saida) => saida,
        Err(e) => {
            eprintln!("wyt recusado: {e}");
            return ExitCode::from(1);
        }
    };
    println!(
        "re-tema: {origem} ({} B) → {destino} ({} B)",
        bytes.len(),
        saida.len()
    );
    if dry_run {
        return ExitCode::SUCCESS;
    }
    match wyt::install(Path::new(destino), &saida) {
        Ok(r) => {
            relatar_asset(&r);
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("falha ao gravar {destino}: {e}");
            ExitCode::from(1)
        }
    }
}

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().collect();
    if args.len() < 3 {
        eprintln!("{}", uso());
        return ExitCode::from(2);
    }
    match args[1].as_str() {
        "check" => match (args.get(2), args.get(3)) {
            (Some(dir), Some(manifesto)) => cmd_check(dir, manifesto),
            _ => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
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
            let mut instancias = 1usize;
            let mut indice = 5;
            while indice < args.len() {
                let a = &args[indice];
                if apos_travessao {
                    extras.push(a.clone());
                } else if a == "--dry-run" {
                    dry_run = true;
                } else if a == "--instances" {
                    let valor = args.get(indice + 1).cloned().unwrap_or_default();
                    match bootstrap::parse_instances(&valor) {
                        Ok(janelas) => instancias = janelas,
                        Err(erro) => {
                            eprintln!("{erro}");
                            return ExitCode::from(2);
                        }
                    }
                    indice += 1;
                } else if a == "--" {
                    apos_travessao = true;
                }
                indice += 1;
            }
            cmd_launch(&args[2], &args[3], registro, dry_run, &extras, instancias)
        }
        "branding" => {
            // `branding <dir> [--dry-run]`: só o diretório é posicional.
            let dry_run = args.iter().skip(3).any(|a| a == "--dry-run");
            cmd_branding(&args[2], dry_run)
        }
        "gui" => {
            let dry_run = args.iter().skip(3).any(|a| a == "--dry-run");
            cmd_gui(&args[2], dry_run)
        }
        "retheme" => match (args.get(2), args.get(3)) {
            (Some(origem), Some(destino)) => {
                let dry_run = args.iter().skip(4).any(|a| a == "--dry-run");
                cmd_retheme(origem, destino, dry_run)
            }
            _ => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
        _ => {
            eprintln!("{}", uso());
            ExitCode::from(2)
        }
    }
}
