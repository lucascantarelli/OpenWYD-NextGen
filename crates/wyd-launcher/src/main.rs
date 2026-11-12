//! CLI fina do launcher: `check`, `launch`, `serverlist`, `branding` e `update`.
//! Lógica mora na biblioteca; aqui só parsing de argumentos e exit codes.

use std::path::Path;
use std::process::ExitCode;
use std::time::Instant;

use wyd_launcher::{
    bootstrap, branding, cliente_zip, configuracao, distribuicao, gui, identidade, lancamento,
    manifest, portal_client, reparo, serverlist, snbin, wyt,
};

const EXE_PADRAO: &str = "WYD.exe";

fn uso() -> String {
    format!(
        "uso:\n  {n} check <dir-do-jogo> <manifesto>\n  {n} config <dir-do-jogo> <base-do-portal> --key-file <arq> [--dry-run]\n  {n} launch <dir-do-jogo> <manifesto> <registro-de-canais> [--instances N] [--dry-run] [--config <arq> --key-file <arq>] [-- args...]\n  {n} serverlist <registro-de-canais> <dir-do-jogo> [--dry-run] [--config <arq> --key-file <arq>]\n  {n} branding <dir-do-jogo> [--dry-run] [--config <arq> --key-file <arq>]\n  {n} gui <dir-do-jogo> [--dry-run] [--config <arq> --key-file <arq>]\n  {n} retheme <origem.wyt> <destino.wyt> [--dry-run] [--config <arq> --key-file <arq>]\n  {n} update <dir-do-jogo> <pacote> [--dry-run]\n  {n} instalar <dir-do-jogo> [<client.zip|URL>] [<manifesto|URL>]\n  {n} reparo <dir-do-jogo> <base-do-portal>\n  {n} icone <arquivo.ico>\n  (instalar usa WYD_LAUNCHER_CLIENT_ZIP/WYD_LAUNCHER_CLIENT_MANIFEST quando o argumento falta; reparo baixa manifesto e arquivos do portal)\n  (branding/gui/serverlist/launch/retheme resolvem <dir>/launcher.conf com --key-file ou WYD_LAUNCHER_CONFIG_KEY; sem config confiável, cai para a identidade embutida com motivo; update confere o pacote com a chave pública Ed25519 embutida)",
        n = wyd_launcher::LAUNCHER_NAME
    )
}

/// Base do portal para a telemetria: `launcher.conf` remoto primeiro, env depois.
fn base_do_portal(remoto: Option<&configuracao::PortalRemoto>) -> Option<String> {
    remoto.map(|portal| portal.base.clone()).or_else(|| {
        std::env::var(portal_client::PORTAL_ENV)
            .ok()
            .filter(|valor| !valor.is_empty())
    })
}

/// Telemetria best-effort: falha no envio nunca muda o exit code nem a tela.
fn telemetria(
    portal: Option<&str>,
    operacao: portal_client::Operacao,
    ok: bool,
    inicio: Instant,
    arquivos: u32,
    falhas: Vec<portal_client::FalhaTelemetria>,
) {
    let Some(base) = portal.map(str::to_string).or_else(|| base_do_portal(None)) else {
        return;
    };
    let duracao_ms = u32::try_from(inicio.elapsed().as_millis())
        .unwrap_or(600_000)
        .min(600_000);
    let evento = portal_client::Telemetria {
        versao: portal_client::VERSAO_LAUNCHER,
        build_cliente: portal_client::BUILD_CLIENTE,
        operacao,
        ok,
        duracao_ms,
        arquivos,
        falhas,
    };
    if let Err(e) = portal_client::enviar_telemetria(&base, &evento) {
        eprintln!("telemetria: {e}");
    }
}

/// Nome de arquivo para a telemetria (sem diretório, só o alfabeto do contrato).
fn nome_de_arquivo(caminho: &str) -> String {
    let base = caminho.rsplit('/').next().unwrap_or(caminho);
    let limpo: String = base
        .chars()
        .map(|c| {
            if c.is_ascii_alphanumeric() || matches!(c, '.' | '_' | '-') {
                c
            } else {
                '_'
            }
        })
        .take(120)
        .collect();
    if limpo.is_empty() {
        "arquivo".to_string()
    } else {
        limpo
    }
}

fn motivo_de(erro: &distribuicao::DistError) -> portal_client::MotivoFalha {
    use distribuicao::DistError as E;
    use portal_client::MotivoFalha as M;
    match erro {
        E::BadSignature | E::ChaveEmbutida => M::Assinatura,
        E::PayloadCorrupt { .. } | E::PostVerify { .. } => M::Hash,
        E::Io(_) | E::Patch(_) => M::Io,
        E::Manifest(_) => M::Desconhecido,
    }
}

fn cmd_check(dir: &str, manifesto: &str, portal: Option<&str>) -> ExitCode {
    let inicio = Instant::now();
    let m = match manifest::load(Path::new(manifesto)) {
        Ok(m) => m,
        Err(e) => {
            eprintln!("manifesto inválido: {e}");
            telemetria(
                portal,
                portal_client::Operacao::Check,
                false,
                inicio,
                0,
                vec![portal_client::FalhaTelemetria::nova(
                    "manifesto",
                    portal_client::MotivoFalha::Io,
                )],
            );
            return ExitCode::from(2);
        }
    };
    let rel = manifest::verify(&m, Path::new(dir));
    let mut falhas = 0;
    let mut telemetria_falhas = Vec::new();
    for r in &rel.files {
        if r.status == manifest::FileStatus::Ok {
            println!("OK      {}", r.path);
        } else {
            println!("FALHA   {}: {}", r.path, r.status);
            falhas += 1;
            telemetria_falhas.push(portal_client::FalhaTelemetria::nova(
                nome_de_arquivo(&r.path),
                portal_client::MotivoFalha::Hash,
            ));
        }
    }
    println!(
        "manifesto build {}: {} íntegros, {falhas} com problema",
        m.build,
        rel.files.len() - falhas
    );
    telemetria(
        portal,
        portal_client::Operacao::Check,
        falhas == 0,
        inicio,
        rel.files.len() as u32,
        telemetria_falhas,
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
    identidade: &identidade::Identidade,
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
    let sn = snbin::build(identidade, &snbin::canais_por_grupo(&cells))
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

fn cmd_serverlist(
    registro: &str,
    dir: &str,
    dry_run: bool,
    efetiva: &configuracao::Efetiva,
) -> ExitCode {
    relatar_identidade(efetiva);
    match instalar_serverlist(dir, registro, dry_run, &efetiva.identidade) {
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

/// Gera o ícone da marca (`icon.ico`) no caminho pedido.
fn cmd_icone(destino: &str) -> ExitCode {
    let path = Path::new(destino);
    let (Some(dir), Some(nome)) = (
        path.parent().filter(|p| !p.as_os_str().is_empty()),
        path.file_name().and_then(|n| n.to_str()),
    ) else {
        eprintln!("destino inválido: {destino}");
        return ExitCode::from(2);
    };
    let id = match identidade::padrao() {
        Ok(id) => id,
        Err(e) => {
            eprintln!("identidade inválida: {e}");
            return ExitCode::from(2);
        }
    };
    let bytes = match branding::ico(&id) {
        Ok(bytes) => bytes,
        Err(e) => {
            eprintln!("ícone: {e}");
            return ExitCode::from(1);
        }
    };
    match branding::install(dir, &[(nome, bytes.as_slice())]) {
        Ok(relatorios) => {
            for r in &relatorios {
                relatar_asset(r);
            }
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("ícone não gravado: {e}");
            ExitCode::from(1)
        }
    }
}

fn cmd_branding(dir: &str, dry_run: bool, efetiva: &configuracao::Efetiva) -> ExitCode {
    relatar_identidade(efetiva);
    let id = &efetiva.identidade;
    let bmp = match branding::bmp(id) {
        Ok(bytes) => bytes,
        Err(e) => {
            eprintln!("skin: {e}");
            return ExitCode::from(1);
        }
    };
    let hat = match branding::hat(id) {
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
    efetiva: &configuracao::Efetiva,
) -> ExitCode {
    relatar_identidade(efetiva);
    let inicio = Instant::now();
    let portal = efetiva.portal.as_ref().map(|portal| portal.base.as_str());
    let dir_p = Path::new(dir);
    // Fail-closed (ADR-0003): manifesto vermelho = nada escrito e jogo não abre.
    // Pipeline compartilhado com a janela Tauri (`lancamento`).
    let plano = match lancamento::planejar(
        dir_p,
        Path::new(manifesto),
        Path::new(registro),
        &efetiva.identidade,
        EXE_PADRAO,
        extras,
    ) {
        Ok(plano) => plano,
        Err(e) => return falha_de_lancamento(portal, inicio, &e),
    };
    println!(
        "manifesto build {}: {} célula(s) do registro {registro}",
        plano.build,
        plano.cells.len()
    );
    for c in &plano.cells {
        println!("  grupo {} canal {} → {}", c.group, c.channel, c.addr);
    }
    if dry_run {
        println!(
            "dry-run: instalaria {} ({} bytes) e {} ({} bytes) em {dir}",
            dir_p.join(serverlist::FILE_NAME).display(),
            plano.serverlist.len(),
            dir_p.join(snbin::FILE_NAME).display(),
            plano.sn.len()
        );
        println!("dry-run: {}", bootstrap::describe(&plano.launch));
        for janela in 1..=instancias {
            println!("dry-run: janela {janela} de {instancias} (mesmo executável)");
        }
        return ExitCode::SUCCESS;
    }
    let preparo = match lancamento::preparar(dir_p, &plano) {
        Ok(preparo) => preparo,
        Err(e) => return falha_de_lancamento(portal, inicio, &e),
    };
    relatar_instalacao(&preparo.instalacao);
    relatar_asset(&preparo.sn);
    let pids = match lancamento::abrir(&plano, instancias) {
        Ok(pids) => pids,
        Err(e) => return falha_de_lancamento(portal, inicio, &e),
    };
    let lista = pids
        .iter()
        .map(u32::to_string)
        .collect::<Vec<_>>()
        .join(", ");
    println!("jogo iniciado: {} janela(s) (pids: {lista})", pids.len());
    telemetria(
        portal,
        portal_client::Operacao::Launch,
        true,
        inicio,
        1,
        Vec::new(),
    );
    ExitCode::SUCCESS
}

/// Log + telemetria + exit code de uma recusa do pipeline (fail-closed).
fn falha_de_lancamento(
    portal: Option<&str>,
    inicio: Instant,
    erro: &lancamento::LancamentoError,
) -> ExitCode {
    eprintln!("{erro}");
    telemetria(
        portal,
        portal_client::Operacao::Launch,
        false,
        inicio,
        0,
        vec![portal_client::FalhaTelemetria::nova(
            "lancamento",
            motivo_do_lancamento(erro),
        )],
    );
    ExitCode::from(codigo_do_lancamento(erro))
}

fn motivo_do_lancamento(erro: &lancamento::LancamentoError) -> portal_client::MotivoFalha {
    use lancamento::LancamentoError as E;
    use portal_client::MotivoFalha as M;
    match erro {
        E::Verificacao(_) => M::Hash,
        E::Snbin(_) => M::Assinatura,
        E::Manifesto(_) | E::Serverlist(_) | E::Bootstrap(_) => M::Io,
    }
}

fn codigo_do_lancamento(erro: &lancamento::LancamentoError) -> u8 {
    match erro {
        lancamento::LancamentoError::Manifesto(_) => 2,
        _ => 1,
    }
}

fn cmd_update(dir: &str, pacote: &str, dry_run: bool, portal: Option<&str>) -> ExitCode {
    let inicio = Instant::now();
    // A chave do pacote é pública e vem embutida: não existe --key-file para o jogador errar.
    let chave = match distribuicao::chave_embutida() {
        Ok(k) => k,
        Err(e) => {
            eprintln!("launcher inconsistente: {e}");
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
                telemetria(
                    portal,
                    portal_client::Operacao::Patch,
                    true,
                    inicio,
                    u32::try_from(r.aplicados).unwrap_or(u32::MAX),
                    Vec::new(),
                );
                ExitCode::SUCCESS
            }
            Err(e) => {
                eprintln!("update falhou: {e}");
                telemetria(
                    portal,
                    portal_client::Operacao::Patch,
                    false,
                    inicio,
                    0,
                    vec![portal_client::FalhaTelemetria::nova(
                        nome_de_arquivo(pacote),
                        motivo_de(&e),
                    )],
                );
                ExitCode::from(1)
            }
        }
    }
}

/// Notícias da janela: sem portal ou com rota fora, a janela abre com o aviso.
fn noticias_da_janela(efetiva: &configuracao::Efetiva) -> gui::Noticias {
    let Some(base) = base_do_portal(efetiva.portal.as_ref()) else {
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

/// Gera a janela do launcher GUI (contrato visual da FASE-08-8) na identidade efetiva.
fn cmd_gui(dir: &str, dry_run: bool, efetiva: &configuracao::Efetiva) -> ExitCode {
    relatar_identidade(efetiva);
    let noticias = noticias_da_janela(efetiva);
    if let Some(motivo) = &noticias.aviso {
        eprintln!("{motivo}");
    } else {
        println!("notícias: {} item(ns)", noticias.itens.len());
    }
    let html = match gui::html_com_noticias(efetiva, &noticias) {
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
fn cmd_retheme(
    origem: &str,
    destino: &str,
    dry_run: bool,
    efetiva: &configuracao::Efetiva,
) -> ExitCode {
    relatar_identidade(efetiva);
    let bytes = match std::fs::read(origem) {
        Ok(bytes) => bytes,
        Err(e) => {
            eprintln!("origem ilegível ({origem}): {e}");
            return ExitCode::from(2);
        }
    };
    let saida = match wyt::retheme(&bytes, &efetiva.identidade) {
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

/// Valor da flag `--nome`, quando presente (o valor vem logo depois dela).
fn flag_valor(args: &[String], nome: &str) -> Option<String> {
    let posicao = args.iter().position(|a| a == nome)?;
    args.get(posicao + 1).cloned()
}

/// Identidade efetiva do diretório: remota assinada quando dá, embutida com motivo quando não.
///
/// `--config <arquivo>` é o caminho de operador e é **estrito**: assinatura inválida ou arquivo
/// ilegível são erro, não fallback. Sem `--config`, a janela resolve `<dir>/launcher.conf`
/// (`configuracao::resolver`) com a chave de `--key-file` ou `WYD_LAUNCHER_CONFIG_KEY` e cai
/// para a identidade embutida com motivo observável.
fn identidade_efetiva(dir: &str, flags: &[String]) -> Result<configuracao::Efetiva, String> {
    let key_file = flag_valor(flags, "--key-file").or_else(|| {
        std::env::var(configuracao::CHAVE_ENV)
            .ok()
            .filter(|p| !p.is_empty())
    });
    configuracao::resolver_flags(
        Path::new(dir),
        flag_valor(flags, "--config").as_deref(),
        key_file.as_deref(),
    )
}

/// Registra na saída de qual identidade a janela desenha (origem + motivo do fallback).
fn relatar_identidade(efetiva: &configuracao::Efetiva) {
    match &efetiva.origem {
        configuracao::Origem::Remota { revisao } => {
            let base = efetiva
                .portal
                .as_ref()
                .map_or("?", |portal| portal.base.as_str());
            println!("identidade: remota revisão {revisao} · portal {base}");
        }
        configuracao::Origem::Embutida => println!(
            "identidade: embutida ({})",
            efetiva.motivo.as_deref().unwrap_or("sem motivo")
        ),
    }
}

/// Busca, verifica e instala a configuração assinada publicada pelo portal.
fn cmd_config(dir: &str, base: &str, key_file: &str, dry_run: bool) -> ExitCode {
    let inicio = Instant::now();
    let chave = match configuracao::carregar_chave(Path::new(key_file)) {
        Ok(k) => k,
        Err(e) => {
            eprintln!("chave inválida: {e}");
            return ExitCode::from(2);
        }
    };
    let api = match portal_client::PortalApi::new(base) {
        Ok(api) => api,
        Err(e) => {
            eprintln!("portal inválido: {e}");
            return ExitCode::from(2);
        }
    };
    let texto = match api.fetch_configuracao() {
        Ok(texto) => texto,
        Err(e) => {
            eprintln!("configuração não obtida: {e}");
            telemetria(
                Some(base),
                portal_client::Operacao::Config,
                false,
                inicio,
                0,
                vec![portal_client::FalhaTelemetria::nova(
                    "launcher.conf",
                    portal_client::MotivoFalha::Rede,
                )],
            );
            return ExitCode::from(1);
        }
    };
    let remota = match configuracao::verificar(&texto, &chave) {
        Ok(remota) => remota,
        Err(e) => {
            eprintln!("configuração recusada: {e}");
            telemetria(
                Some(base),
                portal_client::Operacao::Config,
                false,
                inicio,
                0,
                vec![portal_client::FalhaTelemetria::nova(
                    "launcher.conf",
                    portal_client::MotivoFalha::Assinatura,
                )],
            );
            return ExitCode::from(1);
        }
    };
    // Só instala o que o launcher consegue usar: o merge é a validação final da marca/canais.
    let base_id = match identidade::padrao() {
        Ok(id) => id,
        Err(e) => {
            eprintln!("identidade embutida inválida: {e}");
            return ExitCode::from(2);
        }
    };
    let id = match remota.mesclar(&base_id) {
        Ok(id) => id,
        Err(e) => {
            eprintln!("configuração recusada: {e}");
            telemetria(
                Some(base),
                portal_client::Operacao::Config,
                false,
                inicio,
                0,
                vec![portal_client::FalhaTelemetria::nova(
                    "launcher.conf",
                    portal_client::MotivoFalha::Assinatura,
                )],
            );
            return ExitCode::from(1);
        }
    };
    println!(
        "configuração revisão {} de {}: {} sobreposição(ões), {} servidor(es), manifesto {}",
        remota.revisao,
        remota.portal.base,
        remota.sobreposicoes().len(),
        id.servidores().map(|s| s.len()).unwrap_or(0),
        remota.portal.manifesto
    );
    if dry_run {
        println!("dry-run: gravaria {} em {dir}", configuracao::CONF_NAME);
        return ExitCode::SUCCESS;
    }
    match configuracao::install(Path::new(dir), &texto) {
        Ok(r) => {
            relatar_asset(&r);
            println!(
                "janela: resolve {} com --key-file ou {}",
                configuracao::CONF_NAME,
                configuracao::CHAVE_ENV
            );
            telemetria(
                Some(base),
                portal_client::Operacao::Config,
                true,
                inicio,
                1,
                Vec::new(),
            );
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("falha ao gravar a configuração: {e}");
            telemetria(
                Some(base),
                portal_client::Operacao::Config,
                false,
                inicio,
                0,
                vec![portal_client::FalhaTelemetria::nova(
                    configuracao::CONF_NAME,
                    portal_client::MotivoFalha::Io,
                )],
            );
            ExitCode::from(1)
        }
    }
}

/// Instala/atualiza o cliente a partir do `client.zip` e regenera o `serverlist.bin`.
fn cmd_instalar(dir: &str, zip: &str, manifesto: Option<&str>, portal: Option<&str>) -> ExitCode {
    let inicio = Instant::now();
    match cliente_zip::instalar(Path::new(dir), zip, manifesto) {
        Ok(r) => {
            println!(
                "cliente instalado: build {} · {} arquivo(s) · serverlist {} byte(s) · backup {}",
                r.build,
                r.arquivos,
                r.serverlist_bytes,
                r.backup.display()
            );
            println!("origem: {}", r.origem);
            telemetria(
                portal,
                portal_client::Operacao::Patch,
                true,
                inicio,
                u32::try_from(r.arquivos).unwrap_or(u32::MAX),
                Vec::new(),
            );
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("instalação recusada: {e}");
            telemetria(
                portal,
                portal_client::Operacao::Patch,
                false,
                inicio,
                0,
                vec![portal_client::FalhaTelemetria::nova(
                    "client.zip",
                    portal_client::MotivoFalha::Hash,
                )],
            );
            ExitCode::from(1)
        }
    }
}

/// Repara o diretório contra o manifesto servido pelo portal.
fn cmd_reparo(dir: &str, base: &str, portal: Option<&str>) -> ExitCode {
    let inicio = Instant::now();
    match reparo::reparar(Path::new(dir), base) {
        Ok(r) if r.reparados.is_empty() => {
            println!("reparo: nada a fazer (build {} já íntegro)", r.build);
            ExitCode::SUCCESS
        }
        Ok(r) => {
            println!(
                "reparo: {} arquivo(s) restaurado(s) do portal (build {}, backup {})",
                r.reparados.len(),
                r.build,
                r.backup.display()
            );
            for caminho in &r.reparados {
                println!("  restaurado: {caminho}");
            }
            telemetria(
                portal,
                portal_client::Operacao::Patch,
                true,
                inicio,
                u32::try_from(r.reparados.len()).unwrap_or(u32::MAX),
                Vec::new(),
            );
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("reparo falhou: {e}");
            telemetria(
                portal,
                portal_client::Operacao::Patch,
                false,
                inicio,
                0,
                vec![portal_client::FalhaTelemetria::nova(
                    "reparo",
                    portal_client::MotivoFalha::Hash,
                )],
            );
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
            (Some(dir), Some(manifesto)) => {
                let flags = args.get(4..).unwrap_or(&[]);
                let efetiva = identidade_efetiva(dir, flags).ok();
                let portal = efetiva
                    .as_ref()
                    .and_then(|e| e.portal.as_ref())
                    .map(|p| p.base.as_str());
                cmd_check(dir, manifesto, portal)
            }
            _ => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
        "serverlist" => match (args.get(2), args.get(3)) {
            (Some(registro), Some(dir)) => {
                let flags = args.get(4..).unwrap_or(&[]);
                let dry_run = flags.iter().any(|a| a == "--dry-run");
                match identidade_efetiva(dir, flags) {
                    Ok(efetiva) => cmd_serverlist(registro, dir, dry_run, &efetiva),
                    Err(e) => {
                        eprintln!("{e}");
                        ExitCode::from(2)
                    }
                }
            }
            _ => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
        "config" => match (args.get(2), args.get(3)) {
            (Some(dir), Some(base)) => {
                let flags = args.get(4..).unwrap_or(&[]);
                let key_file = flag_valor(flags, "--key-file").unwrap_or_default();
                if key_file.is_empty() {
                    eprintln!("falta --key-file <arq>");
                    eprintln!("{}", uso());
                    return ExitCode::from(2);
                }
                let dry_run = flags.iter().any(|a| a == "--dry-run");
                cmd_config(dir, base, &key_file, dry_run)
            }
            _ => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
        "update" => match (args.get(2), args.get(3)) {
            (Some(dir), Some(pacote)) => {
                let flags = args.get(4..).unwrap_or(&[]);
                let dry_run = flags.iter().any(|a| a == "--dry-run");
                let efetiva = identidade_efetiva(dir, flags).ok();
                let portal = efetiva
                    .as_ref()
                    .and_then(|e| e.portal.as_ref())
                    .map(|p| p.base.as_str());
                cmd_update(dir, pacote, dry_run, portal)
            }
            _ => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
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
            match identidade_efetiva(&args[2], &args[5..]) {
                Ok(efetiva) => cmd_launch(
                    &args[2], &args[3], registro, dry_run, &extras, instancias, &efetiva,
                ),
                Err(e) => {
                    eprintln!("{e}");
                    ExitCode::from(2)
                }
            }
        }
        "instalar" => match args.get(2) {
            Some(dir) => {
                let zip = args
                    .get(3)
                    .cloned()
                    .or_else(|| {
                        std::env::var(cliente_zip::ZIP_ENV)
                            .ok()
                            .filter(|v| !v.is_empty())
                    });
                let Some(zip) = zip else {
                    eprintln!(
                        "falta o client.zip: passe <client.zip|URL> ou defina {}",
                        cliente_zip::ZIP_ENV
                    );
                    return ExitCode::from(2);
                };
                let manifesto = args.get(4).cloned().or_else(|| {
                    std::env::var(cliente_zip::MANIFESTO_ENV)
                        .ok()
                        .filter(|v| !v.is_empty())
                });
                let efetiva = identidade_efetiva(dir, &args[5..]).ok();
                let portal = efetiva
                    .as_ref()
                    .and_then(|e| e.portal.as_ref())
                    .map(|p| p.base.as_str());
                cmd_instalar(dir, &zip, manifesto.as_deref(), portal)
            }
            None => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
        "reparo" => match (args.get(2), args.get(3)) {
            (Some(dir), Some(base)) => {
                let efetiva = identidade_efetiva(dir, &args[4..]).ok();
                let portal = efetiva
                    .as_ref()
                    .and_then(|e| e.portal.as_ref())
                    .map(|p| p.base.as_str());
                cmd_reparo(dir, base, portal)
            }
            _ => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
        "icone" => match args.get(2) {
            Some(destino) => cmd_icone(destino),
            None => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
        "branding" => match args.get(2) {
            Some(dir) => {
                let flags = args.get(3..).unwrap_or(&[]);
                let dry_run = flags.iter().any(|a| a == "--dry-run");
                match identidade_efetiva(dir, flags) {
                    Ok(efetiva) => cmd_branding(dir, dry_run, &efetiva),
                    Err(e) => {
                        eprintln!("{e}");
                        ExitCode::from(2)
                    }
                }
            }
            None => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
        "gui" => match args.get(2) {
            Some(dir) => {
                let flags = args.get(3..).unwrap_or(&[]);
                let dry_run = flags.iter().any(|a| a == "--dry-run");
                match identidade_efetiva(dir, flags) {
                    Ok(efetiva) => cmd_gui(dir, dry_run, &efetiva),
                    Err(e) => {
                        eprintln!("{e}");
                        ExitCode::from(2)
                    }
                }
            }
            None => {
                eprintln!("{}", uso());
                ExitCode::from(2)
            }
        },
        "retheme" => match (args.get(2), args.get(3)) {
            (Some(origem), Some(destino)) => {
                let flags = args.get(4..).unwrap_or(&[]);
                let dry_run = flags.iter().any(|a| a == "--dry-run");
                let dir = Path::new(destino)
                    .parent()
                    .and_then(|p| p.to_str())
                    .unwrap_or(".");
                match identidade_efetiva(dir, flags) {
                    Ok(efetiva) => cmd_retheme(origem, destino, dry_run, &efetiva),
                    Err(e) => {
                        eprintln!("{e}");
                        ExitCode::from(2)
                    }
                }
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
