//! Janela do launcher GUI (CLIENT-08) — HTML de uma página gerado da identidade **efetiva**.
//!
//! A GUI ainda não existe como app Tauri; o que existe é o **contrato visual**: este módulo
//! emite a janela (login, notícias, canais, botão Jogar) usando **só** `cor.*`, `espaco.*`,
//! `raio*`, `tipografia.*` e `servidor.N` da identidade efetiva. A identidade efetiva vem do
//! mesmo `configuracao` que o CLI usa (`Efetiva`): quando há `launcher.conf` assinado e válido,
//! marca, canais e URLs do portal saem dele; quando não há (ausente, sem chave, adulterado ou
//! fora do contrato), sai a identidade embutida **com o motivo no rodapé** — nunca tela muda,
//! nunca `panic`, nunca config parcial.
//!
//! O teste do módulo reprova qualquer cor fora da identidade, então a GUI nasce na mesma paleta
//! do portal e da skin do 7662 (`branding.rs`) em vez de virar uma terceira.
//!
//! `Launcher.hat` continua sendo o arquivo que o `WYDLauncher.exe` de 7662 abre; este HTML tem
//! outro nome (`LauncherGUI.html`) para não ser carregado por engano por aquele binário.

use std::fs;
use std::io;
use std::path::Path;

use crate::branding::{self, InstallReport};
use crate::configuracao::{Efetiva, Origem};
use crate::identidade::IdentidadeError;

/// Nome do arquivo gerado no diretório do jogo.
pub const GUI_NAME: &str = "LauncherGUI.html";

/// Escapa texto para HTML (nome de canal e motivo do fallback vêm de fora do binário).
fn escapar(texto: &str) -> String {
    let mut saida = String::with_capacity(texto.len());
    for caractere in texto.chars() {
        match caractere {
            '&' => saida.push_str("&amp;"),
            '<' => saida.push_str("&lt;"),
            '>' => saida.push_str("&gt;"),
            '"' => saida.push_str("&quot;"),
            '\'' => saida.push_str("&#39;"),
            outro => saida.push(outro),
        }
    }
    saida
}

fn espacos(efetiva: &Efetiva) -> Result<Vec<u32>, IdentidadeError> {
    let id = &efetiva.identidade;
    (1..=7).map(|i| id.numero(&format!("espaco.{i}"))).collect()
}

/// Notícias exibidas na janela e o motivo quando a rota não respondeu.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Noticias {
    pub itens: Vec<crate::portal_client::Noticia>,
    pub aviso: Option<String>,
}

/// HTML completo da janela (contrato mínimo, sem notícias).
pub fn html(efetiva: &Efetiva) -> Result<String, IdentidadeError> {
    html_com_noticias(efetiva, &Noticias::default())
}

/// HTML da janela com o painel de notícias (ou o aviso de indisponibilidade).
///
/// A janela nunca fica muda nem falha por causa da vitrine: sem itens e com
/// aviso, o painel existe e mostra o motivo; sem nenhum dos dois, o painel nem
/// é criado.
pub fn html_com_noticias(
    efetiva: &Efetiva,
    noticias: &Noticias,
) -> Result<String, IdentidadeError> {
    let id = &efetiva.identidade;
    let hex = |chave: &str| id.cor(chave).map(|cor| cor.hex());
    let bg = hex("cor.bg")?;
    let surface = hex("cor.surface")?;
    let surface2 = hex("cor.surface-2")?;
    let border = hex("cor.border")?;
    let border_forte = hex("cor.border-strong")?;
    let text = hex("cor.text")?;
    let muted = hex("cor.muted")?;
    let accent = hex("cor.accent")?;
    let accent_ink = hex("cor.accent-ink")?;
    let ok = hex("cor.ok")?;
    let warn = hex("cor.warn")?;
    let danger = hex("cor.danger")?;
    let embed = hex("cor.embed")?;
    let fundo = branding::fundo_data_uri();
    let e = espacos(efetiva)?;
    let raio = id.numero("raio")?;
    let raio_pequeno = id.numero("raio.pequeno")?;
    let familia = id.get("tipografia.familia")?;
    let mono = id.get("tipografia.mono")?;
    let tamanho = id.numero("tipografia.tamanho")?;
    let altura = id.decimal("tipografia.altura")?;
    let titulo = id.numero("tipografia.titulo")?;
    let legenda = id.numero("tipografia.legenda")?;
    let servidores = id.servidores()?;

    let canais: String = servidores
        .iter()
        .map(|(slot, nome)| {
            format!(
                "<li class=\"canal\"><span class=\"canal__num\">{slot}</span>\
<span class=\"canal__nome\">{nome}</span><span class=\"canal__estado\">aguardando</span></li>",
                nome = escapar(nome)
            )
        })
        .collect();

    let origem_label = match efetiva.origem {
        Origem::Remota { revisao } => format!("revisão {revisao}"),
        Origem::Embutida => "identidade embutida".to_string(),
    };

    let portal = match &efetiva.portal {
        Some(portal) => format!(
            "<p class=\"aviso\">Portal: <a href=\"{base}\">{base}</a></p>\
<p class=\"aviso\">Distribuição: <a href=\"{manifesto}\">manifesto</a> · \
<a href=\"{pacote}\">pacote do launcher</a></p>",
            base = escapar(&portal.base),
            manifesto = escapar(&portal.url(&portal.manifesto)),
            pacote = escapar(&portal.url(&portal.pacote)),
        ),
        None => String::new(),
    };

    let motivo = match &efetiva.motivo {
        Some(motivo) => format!(
            "<p class=\"aviso aviso--motivo\">identidade embutida: {}</p>",
            escapar(motivo)
        ),
        None => String::new(),
    };

    let (css_noticias, painel_noticias) = if noticias.itens.is_empty() && noticias.aviso.is_none() {
        (String::new(), String::new())
    } else {
        let itens: String = noticias
            .itens
            .iter()
            .take(5)
            .map(|n| {
                format!(
                    "<li class=\"noticia\"><a class=\"noticia__titulo\" href=\"{url}\">{titulo}</a>\
<span class=\"noticia__data\">{data} · {tag}</span>\
<p class=\"noticia__resumo\">{resumo}</p></li>",
                    url = escapar(&n.url),
                    titulo = escapar(&n.titulo),
                    data = escapar(&n.data),
                    tag = escapar(&n.tag),
                    resumo = escapar(&n.resumo),
                )
            })
            .collect();
        let aviso = noticias
            .aviso
            .as_ref()
            .map(|motivo| format!("<p class=\"aviso aviso--motivo\">{}</p>", escapar(motivo)))
            .unwrap_or_default();
        let css = ".painel--largo {{ grid-column: 1 / -1; }}\n\
.noticias {{ list-style: none; margin: 0; padding: 0; display: grid; gap: var(--space-3); }}\n\
.noticia {{ display: grid; gap: var(--space-1); }}\n\
.noticia__titulo {{ font-weight: 600; }}\n\
.noticia__data {{ color: var(--muted); font-size: var(--legend-size); font-family: var(--mono); }}\n\
.noticia__resumo {{ margin: 0; color: var(--muted); }}\n";
        (
            css.to_string(),
            format!(
                "<section class=\"painel painel--largo\"><h2>Notícias</h2>\
<ul class=\"noticias\">{itens}</ul>{aviso}</section>"
            ),
        )
    };

    let mut out = String::new();
    out.push_str(&format!(
        "<!doctype html>\n<html lang=\"pt-BR\">\n<head>\n<meta charset=\"utf-8\">\n\
<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">\n\
<title>WYD · launcher</title>\n<style>\n\
:root {{\n  --bg: {bg};\n  --surface: {surface};\n  --surface-2: {surface2};\n\
  --border: {border};\n  --border-strong: {border_forte};\n  --text: {text};\n\
  --muted: {muted};\n  --accent: {accent};\n  --accent-ink: {accent_ink};\n\
  --ok: {ok};\n  --warn: {warn};\n  --danger: {danger};\n  --embed: {embed};\n\
  --space-1: {e1}px;\n  --space-2: {e2}px;\n  --space-3: {e3}px;\n\
  --space-4: {e4}px;\n  --space-5: {e5}px;\n  --space-6: {e6}px;\n  --space-7: {e7}px;\n\
  --radius: {raio}px;\n  --radius-sm: {raio_pequeno}px;\n\
  --familia: {familia};\n  --mono: {mono};\n  --title-size: {titulo}px;\n\
  --legend-size: {legenda}px;\n  --line: {altura};\n  --fundo: url(\"{fundo}\");\n}}\n\
* {{ box-sizing: border-box; }}\n\
html, body {{ margin: 0; min-height: 100%; }}\n\
body {{ background-color: var(--bg); color: var(--text); font-family: var(--familia);\n\
  font-size: {tamanho}px; line-height: var(--line);\n\
  background-image: linear-gradient(rgba(5, 7, 10, 0.72), rgba(5, 7, 10, 0.88)), var(--fundo);\n\
  background-size: cover; background-position: center; background-repeat: no-repeat;\n\
  background-attachment: fixed; }}\n\
a {{ color: var(--accent); }}\n\
.shell {{ max-width: 1040px; margin: 0 auto; padding: var(--space-5);\n\
  display: grid; gap: var(--space-5); }}\n\
.topo {{ display: flex; align-items: baseline; justify-content: space-between;\n\
  gap: var(--space-3); padding-bottom: var(--space-3);\n\
  border-bottom: 1px solid var(--border); }}\n\
.marca {{ margin: 0; font-size: var(--title-size); letter-spacing: 0.12em;\n\
  color: var(--accent); font-weight: 700; }}\n\
.versao {{ color: var(--muted); font-size: var(--legend-size); font-family: var(--mono); }}\n\
.grade {{ display: grid; grid-template-columns: minmax(260px, 340px) 1fr;\n\
  gap: var(--space-4); align-items: start; }}\n\
.painel {{ background: var(--surface); border: 1px solid var(--border);\n\
  border-radius: var(--radius); padding: var(--space-4); }}\n\
.painel h2 {{ margin: 0 0 var(--space-3); font-size: {legenda}px;\n\
  text-transform: uppercase; letter-spacing: 0.08em; color: var(--muted);\n\
  font-weight: 600; }}\n\
.campo {{ display: grid; gap: var(--space-1); margin-bottom: var(--space-3); }}\n\
.campo > span {{ color: var(--muted); font-size: var(--legend-size); }}\n\
.campo input {{ background: var(--bg); border: 1px solid var(--border-strong);\n\
  border-radius: var(--radius-sm); color: var(--text); padding: var(--space-2)\n\
  var(--space-3); font-size: {tamanho}px; min-height: 32px; width: 100%; }}\n\
.campo input:focus-visible {{ outline: 2px solid var(--accent); outline-offset: 1px; }}\n\
.btn {{ appearance: none; border: 1px solid var(--border-strong);\n\
  background: var(--surface-2); color: var(--text); border-radius: var(--radius-sm);\n\
  padding: var(--space-2) var(--space-3); font-size: {legenda}px; cursor: pointer;\n\
  min-height: 32px; }}\n\
.btn--primario {{ background: var(--accent); border-color: var(--accent);\n\
  color: var(--accent-ink); font-weight: 600; width: 100%; }}\n\
.btn--primario:hover {{ border-color: var(--accent-ink); }}\n\
.aviso {{ margin: var(--space-2) 0 0; color: var(--muted); font-size: var(--legend-size); }}\n\
.aviso--motivo {{ color: var(--warn); }}\n\
.canais {{ list-style: none; margin: 0; padding: 0; display: grid;\n\
  gap: var(--space-2); }}\n\
.canal {{ display: grid; grid-template-columns: 28px 1fr auto; gap: var(--space-2);\n\
  align-items: center; background: var(--surface-2); border: 1px solid var(--border);\n\
  border-radius: var(--radius-sm); padding: var(--space-2) var(--space-3); }}\n\
.canal__num {{ color: var(--muted); font-family: var(--mono); font-size: {legenda}px; }}\n\
.canal__nome {{ font-weight: 600; }}\n\
.canal__estado {{ color: var(--warn); font-size: {legenda}px; }}\n\
.rodape {{ display: flex; flex-wrap: wrap; gap: var(--space-3);\n\
  align-items: center; justify-content: space-between; color: var(--muted);\n\
  font-size: var(--legend-size); border-top: 1px solid var(--border);\n\
  padding-top: var(--space-3); }}\n\
.legenda {{ display: flex; gap: var(--space-3); }}\n\
.ok {{ color: var(--ok); }}\n\
.erro {{ color: var(--danger); }}\n\
</style>\n</head>\n<body>\n<main class=\"shell\">\n\
<header class=\"topo\"><h1 class=\"marca\">WYD</h1>\n\
<span class=\"versao\">launcher 0.1.0 · {origem_label}</span></header>\n\
<section class=\"grade\">\n\
<form class=\"painel\" action=\"#\" method=\"post\">\n\
<h2>Entrar</h2>\n\
<label class=\"campo\"><span>Conta</span>\n\
<input name=\"conta\" autocomplete=\"username\" maxlength=\"16\" required></label>\n\
<label class=\"campo\"><span>Senha</span>\n\
<input name=\"senha\" type=\"password\" autocomplete=\"current-password\"\n\
maxlength=\"64\" required></label>\n\
<button class=\"btn btn--primario\" type=\"submit\">Jogar</button>\n\
<p class=\"aviso\">A conta é a mesma do jogo; o launcher não guarda senha.</p>\n\
</form>\n\
<div class=\"painel\"><h2>Canais</h2>\n\
<ul class=\"canais\">{canais}</ul>\n\
<p class=\"aviso\">Nomes e ordem vêm da identidade e do registro de canais;\n\
o status é lido do servidor no boot.</p>{portal}</div>\n\
</section>\n\
<footer class=\"rodape\"><span class=\"legenda\">\n\
<span class=\"ok\">verificação</span><span class=\"erro\">patch</span>\n\
<span>serverlist.bin</span></span>\n\
<span>identidade única: portal · launcher · cliente</span>{motivo}</footer>\n\
</main>\n</body>\n</html>\n",
        e1 = e[0],
        e2 = e[1],
        e3 = e[2],
        e4 = e[3],
        e5 = e[4],
        e6 = e[5],
        e7 = e[6],
    ));
    if !css_noticias.is_empty() {
        out = out.replacen("</style>", &format!("{css_noticias}</style>"), 1);
    }
    if !painel_noticias.is_empty() {
        out = out.replacen(
            "</section>\n<footer",
            &format!("{painel_noticias}</section>\n<footer"),
            1,
        );
    }
    Ok(out)
}

/// Nome do launcher dentro da pasta do jogo.
pub const LAUNCHER_EXE: &str = "WYDLauncher.exe";

/// Copia o próprio launcher para a pasta do jogo como `WYDLauncher.exe`.
///
/// Leitura + escrita atômica com backup (mesma rotina dos outros assets); quando
/// o destino já é idêntico nada é reescrito.
pub fn instalar_launcher(dir: &Path, origem: &Path) -> Result<InstallReport, io::Error> {
    let bytes = fs::read(origem)?;
    if bytes.is_empty() {
        return Err(io::Error::new(
            io::ErrorKind::InvalidData,
            "executável do launcher vazio",
        ));
    }
    let mut relatorios = branding::install(dir, &[(LAUNCHER_EXE, &bytes)])?;
    Ok(relatorios.remove(0))
}

/// Grava a janela de forma atômica, com backup (mesma rotina dos outros assets).
pub fn install(dir: &Path, conteudo: &str) -> Result<InstallReport, io::Error> {
    let mut relatorios = branding::install(dir, &[(GUI_NAME, conteudo.as_bytes())])?;
    Ok(relatorios.remove(0))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::BTreeSet;

    #[test]
    fn janela_so_usa_cores_e_nomes_da_identidade() {
        let efetiva = Efetiva::embutida("teste").expect("identidade");
        let id = &efetiva.identidade;
        let html = html(&efetiva).expect("html");
        let paleta: BTreeSet<String> = id
            .pares()
            .iter()
            .filter(|(chave, _)| chave.starts_with("cor."))
            .map(|(_, valor)| valor.to_lowercase())
            .collect();
        let baixo = html.to_lowercase();
        let mut vistas = BTreeSet::new();
        let mut indice = 0;
        while let Some(posicao) = baixo[indice..].find('#') {
            let inicio = indice + posicao + 1;
            let hex: String = baixo[inicio..]
                .chars()
                .take_while(char::is_ascii_hexdigit)
                .take(6)
                .collect();
            if hex.len() == 6 {
                vistas.insert(format!("#{hex}"));
            }
            indice = inicio;
        }
        assert!(!vistas.is_empty(), "janela sem cor nenhuma?");
        for cor in &vistas {
            assert!(paleta.contains(cor), "cor {cor} fora da identidade única");
        }
        for (slot, nome) in id.servidores().expect("servidores") {
            assert!(
                html.contains(&nome),
                "canal {slot} (`{nome}`) não aparece na janela"
            );
        }
        for indice in 1..=7 {
            let valor = id.numero(&format!("espaco.{indice}")).expect("espaço");
            assert!(
                html.contains(&format!("--space-{indice}: {valor}px;")),
                "espaço {indice} ausente da janela"
            );
        }
        assert!(html.contains("var(--space-4)") && html.contains("var(--space-5)"));
        assert!(html.contains("var(--radius)") && html.contains("var(--radius-sm)"));
        assert!(html.contains(&format!(
            "--familia: {};",
            id.get("tipografia.familia").unwrap()
        )));
        assert!(html.contains(&format!(
            "--title-size: {}px;",
            id.numero("tipografia.titulo").expect("título")
        )));
    }

    #[test]
    fn janela_nao_carrega_recurso_externo() {
        let efetiva = Efetiva::embutida("teste").expect("identidade");
        let html = html(&efetiva).expect("html");
        assert!(
            !html.contains("<script") && !html.contains("<link"),
            "janela não pode carregar recurso de terceiro"
        );
        assert!(!html.contains("src=\"http"));
        assert!(
            !html.contains("href=\"http"),
            "sem config remota não há link absoluto para o portal"
        );
    }

    #[test]
    fn fallback_aparece_no_rodape_para_nao_ficar_tela_muda() {
        let efetiva =
            Efetiva::embutida("launcher.conf recusado: assinatura inválida").expect("identidade");
        let html = html(&efetiva).expect("html");
        assert!(html.contains("identidade embutida:"));
        assert!(html.contains("assinatura inválida"), "motivo não aparece");
        assert!(html.contains("identidade embutida</span>"));
    }

    fn noticia(titulo: &str) -> crate::portal_client::Noticia {
        crate::portal_client::Noticia {
            slug: "primeira".to_string(),
            titulo: titulo.to_string(),
            data: "2026-11-12".to_string(),
            tag: "Jogo".to_string(),
            resumo: "resumo da notícia".to_string(),
            url: "/noticias/primeira".to_string(),
        }
    }

    #[test]
    fn noticias_do_portal_aparecem_na_janela() {
        let efetiva = Efetiva::embutida("teste").expect("identidade");
        let noticias = Noticias {
            itens: vec![noticia("Servidor em festa")],
            aviso: None,
        };
        let html = html_com_noticias(&efetiva, &noticias).expect("html");
        assert!(html.contains("<h2>Notícias</h2>"));
        assert!(html.contains("Servidor em festa"));
        assert!(html.contains("2026-11-12 · Jogo"));
        assert!(html.contains("resumo da notícia"));
        assert!(html.contains("href=\"/noticias/primeira\""));
        assert!(!html.contains("indisponíveis"), "sucesso não mostra aviso");
    }

    #[test]
    fn falha_da_vitrine_vira_aviso_sem_esconder_o_resto() {
        let efetiva = Efetiva::embutida("teste").expect("identidade");
        let noticias = Noticias {
            itens: Vec::new(),
            aviso: Some("notícias indisponíveis: timeout".to_string()),
        };
        let html = html_com_noticias(&efetiva, &noticias).expect("html");
        assert!(html.contains("<h2>Notícias</h2>"));
        assert!(html.contains("notícias indisponíveis: timeout"));
        assert!(
            html.contains("<h2>Canais</h2>"),
            "a janela continua completa offline"
        );
        assert!(!html.contains("class=\"noticia\""));
    }

    #[test]
    fn texto_da_noticia_e_escapado() {
        let efetiva = Efetiva::embutida("teste").expect("identidade");
        let noticias = Noticias {
            itens: vec![noticia("<script>alert(1)</script>")],
            aviso: None,
        };
        let html = html_com_noticias(&efetiva, &noticias).expect("html");
        assert!(!html.contains("<script"), "html do portal não executa");
        assert!(html.contains("&lt;script&gt;"));
    }

    #[test]
    fn sem_noticias_o_painel_nem_e_criado() {
        let efetiva = Efetiva::embutida("teste").expect("identidade");
        let html = html(&efetiva).expect("html");
        assert!(!html.contains("Notícias"));
        assert!(!html.contains("noticia__"));
    }

    #[test]
    fn launcher_e_copiado_para_o_jogo_como_wydlauncher() {
        let dir = std::env::temp_dir().join(format!(
            "wyd-launcher-copia-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .expect("relógio")
                .as_nanos()
        ));
        std::fs::create_dir_all(&dir).expect("tmp");
        let origem = dir.join("origem.exe");
        std::fs::write(&origem, b"MZ launcher falso").expect("origem");
        let primeiro = instalar_launcher(&dir, &origem).expect("copia");
        assert!(primeiro.changed);
        assert_eq!(primeiro.path, dir.join(LAUNCHER_EXE));
        assert_eq!(
            std::fs::read(dir.join(LAUNCHER_EXE)).expect("copiado"),
            b"MZ launcher falso"
        );
        let segundo = instalar_launcher(&dir, &origem).expect("recopia");
        assert!(!segundo.changed, "idêntico não reescreve nem cria backup");
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn launcher_vazio_e_recusado() {
        let dir = std::env::temp_dir().join(format!("wyd-launcher-vazio-{}", std::process::id()));
        std::fs::create_dir_all(&dir).expect("tmp");
        let origem = dir.join("vazio.exe");
        std::fs::write(&origem, b"").expect("origem");
        assert!(instalar_launcher(&dir, &origem).is_err());
        assert!(!dir.join(LAUNCHER_EXE).exists());
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn janela_usa_o_fundo_da_marca() {
        let efetiva = Efetiva::embutida("teste").expect("identidade");
        let html = html(&efetiva).expect("html");
        assert!(
            html.contains("data:image/jpeg;base64,"),
            "fundo não embutido"
        );
        assert!(html.contains("--fundo: url("));
        assert!(
            html.contains("background-size: cover"),
            "fundo sem recorte responsivo"
        );
        assert!(html.contains("linear-gradient(rgba(5, 7, 10, 0.72)"));
    }

    #[test]
    fn janela_lista_no_maximo_cinco_noticias() {
        let efetiva = Efetiva::embutida("teste").expect("identidade");
        let itens = (0..8).map(|i| noticia(&format!("noticia {i}"))).collect();
        let html = html_com_noticias(&efetiva, &Noticias { itens, aviso: None }).expect("html");
        assert_eq!(html.matches("class=\"noticia\"").count(), 5);
    }
}
