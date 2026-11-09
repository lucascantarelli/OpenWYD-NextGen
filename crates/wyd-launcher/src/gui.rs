//! Janela do launcher GUI (FASE-08-8) — HTML de uma página gerado da identidade única.
//!
//! A GUI ainda não existe como app Tauri; o que existe é o **contrato visual**: este módulo
//! emite a janela (login, notícias, canais, botão Jogar) usando **só** `cor.*`, `espaco.*`,
//! `raio*`, `tipografia.*` e `servidor.N` de `docs/arquitetura/identidade.tokens`. O teste do
//! módulo reprova qualquer cor fora da identidade, então a GUI nasce na mesma paleta do portal
//! e da skin do 7662 (`branding.rs`) em vez de virar uma terceira.
//!
//! `Launcher.hat` continua sendo o arquivo que o `WYDLauncher.exe` de 7662 abre; este HTML tem
//! outro nome (`LauncherGUI.html`) para não ser carregado por engano por aquele binário.

use std::io;
use std::path::Path;

use crate::branding::{self, InstallReport};
use crate::identidade::{Identidade, IdentidadeError};

/// Nome do arquivo gerado no diretório do jogo.
pub const GUI_NAME: &str = "LauncherGUI.html";

fn espacos(id: &Identidade) -> Result<Vec<u32>, IdentidadeError> {
    (1..=7).map(|i| id.numero(&format!("espaco.{i}"))).collect()
}

/// HTML completo da janela, com todos os valores vindos da identidade.
pub fn html(id: &Identidade) -> Result<String, IdentidadeError> {
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
    let e = espacos(id)?;
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
<span class=\"canal__nome\">{nome}</span><span class=\"canal__estado\">aguardando</span></li>"
            )
        })
        .collect();

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
  --legend-size: {legenda}px;\n  --line: {altura};\n}}\n\
* {{ box-sizing: border-box; }}\n\
html, body {{ margin: 0; min-height: 100%; }}\n\
body {{ background: var(--bg); color: var(--text); font-family: var(--familia);\n\
  font-size: {tamanho}px; line-height: var(--line); }}\n\
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
<span class=\"versao\">launcher 0.1.0</span></header>\n\
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
o status é lido do servidor no boot.</p></div>\n\
</section>\n\
<footer class=\"rodape\"><span class=\"legenda\">\n\
<span class=\"ok\">verificação</span><span class=\"erro\">patch</span>\n\
<span>serverlist.bin</span></span>\n\
<span>identidade única: portal · launcher · cliente</span></footer>\n\
</main>\n</body>\n</html>\n",
        e1 = e[0],
        e2 = e[1],
        e3 = e[2],
        e4 = e[3],
        e5 = e[4],
        e6 = e[5],
        e7 = e[6],
    ));
    Ok(out)
}

/// Grava a janela de forma atômica, com backup (mesma rotina dos outros assets).
pub fn install(dir: &Path, conteudo: &str) -> Result<InstallReport, io::Error> {
    let mut relatorios = branding::install(dir, &[(GUI_NAME, conteudo.as_bytes())])?;
    Ok(relatorios.remove(0))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identidade::padrao;
    use std::collections::BTreeSet;

    #[test]
    fn janela_so_usa_cores_e_nomes_da_identidade() {
        let id = padrao().expect("identidade");
        let html = html(&id).expect("html");
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
        let id = padrao().expect("identidade");
        let html = html(&id).expect("html");
        assert!(!html.contains("http://") && !html.contains("https://"));
    }
}
