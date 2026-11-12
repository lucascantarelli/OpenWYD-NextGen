//! Cliente da API pública do portal (`/api/launcher/*`, LAUNCHER-01).
//!
//! O launcher lê do portal a versão esperada e a grade de eventos sem segunda
//! senha: é leitura pública de dado de jogo, com cache curto no servidor. Sem
//! TLS aqui (`http://` de loopback/rede local): o `https://` entra com o canal
//! de distribuição assinado, fora deste módulo.
//!
//! Sem dependências: HTTP/1.0 em `TcpStream` e extração de campos JSON só do
//! que o launcher consome — resposta fora do contrato é erro tipado, nunca
//! valor inventado.

use std::io::{Read, Write};
use std::net::{TcpStream, ToSocketAddrs};
use std::time::Duration;

/// Timeout de cada chamada (rede local/loopback; portal responde em ms).
const TIMEOUT: Duration = Duration::from_secs(10);

/// Timeout curto das rotas de vitrine/telemetria.
///
/// Elas nunca podem atrasar a abertura: portal lento vira aviso na janela, não
/// tela parada esperando 10 s.
pub const TIMEOUT_PUBLICO: Duration = Duration::from_millis(700);

/// Versão do launcher anunciada na telemetria (contrato de `launcher-telemetria`).
pub const VERSAO_LAUNCHER: u32 = 1;

/// Build do cliente que o launcher distribui (7662, `docs/arquitetura/protocolo.md`).
pub const BUILD_CLIENTE: u32 = 7662;

/// Variável de ambiente com a base do portal quando não há `launcher.conf` remoto.
pub const PORTAL_ENV: &str = "WYD_LAUNCHER_PORTAL";

/// Rota pública da configuração assinada (`configuracao::verificar` autentica).
pub const CONFIGURACAO_PATH: &str = "/api/launcher/configuracao";

/// Erro tipado da leitura do portal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum PortalError {
    /// Base fora de `http://host[:porta]`.
    BaseInvalida,
    /// Sem conexão com o portal (fora do ar ou endereço errado).
    Conexao,
    /// O portal aceitou a conexão e não respondeu a tempo.
    Timeout,
    /// Resposta HTTP fora de 200 (o código viaja junto para diagnóstico).
    Http(u16),
    /// Corpo sem separador cabeçalho/corpo ou status ilegível.
    Resposta,
    /// Campo obrigatório ausente ou com tipo errado no JSON.
    Campo(&'static str),
}

impl std::fmt::Display for PortalError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::BaseInvalida => {
                write!(f, "endereço do portal inválido (use http://host[:porta])")
            }
            Self::Conexao => write!(f, "portal inacessível"),
            Self::Timeout => write!(f, "portal não respondeu a tempo"),
            Self::Http(codigo) => write!(f, "portal respondeu HTTP {codigo}"),
            Self::Resposta => write!(f, "resposta do portal ilegível"),
            Self::Campo(campo) => write!(f, "portal não informou `{campo}`"),
        }
    }
}

impl std::error::Error for PortalError {}

/// Versão que o portal anuncia (`GET /api/launcher/versao`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Versao {
    /// Build do cliente esperado (7662 documentado).
    pub client_build: u32,
    /// Versão de fio (`wyd-protocol::PROTOCOL_VERSION`).
    pub protocol_version: u32,
    /// Revisão de configuração vigente (`None` com gateway fora do ar).
    pub config_revision: Option<i64>,
    /// `true` quando o portal alcançou o gateway nesta leitura.
    pub gateway_up: bool,
}

/// Evento na grade pública (`GET /api/launcher/eventos`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Evento {
    pub code: String,
    pub name: String,
    pub enabled: bool,
}

/// Notícia da vitrine pública (`GET /api/launcher/noticias`).
///
/// Só o recorte de vitrine: o corpo markdown fica no portal. `url` chega
/// relativa (`/noticias/<slug>`) e a janela resolve contra a base.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Noticia {
    pub slug: String,
    pub titulo: String,
    pub data: String,
    pub tag: String,
    pub resumo: String,
    pub url: String,
}

/// Operação medida pela telemetria (`operacao` do contrato da rota).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Operacao {
    Check,
    Config,
    Patch,
    Launch,
}

impl Operacao {
    #[must_use]
    pub fn como_str(self) -> &'static str {
        match self {
            Self::Check => "check",
            Self::Config => "config",
            Self::Patch => "patch",
            Self::Launch => "launch",
        }
    }
}

/// Motivo de falha aceito pelo contrato (`falhas[].motivo`).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MotivoFalha {
    Hash,
    Io,
    Assinatura,
    Rede,
    Desconhecido,
}

impl MotivoFalha {
    #[must_use]
    pub fn como_str(self) -> &'static str {
        match self {
            Self::Hash => "hash",
            Self::Io => "io",
            Self::Assinatura => "assinatura",
            Self::Rede => "rede",
            Self::Desconhecido => "desconhecido",
        }
    }
}

/// Falha por arquivo enviada na telemetria (nome de arquivo, nunca caminho).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FalhaTelemetria {
    pub arquivo: String,
    pub motivo: MotivoFalha,
}

impl FalhaTelemetria {
    #[must_use]
    pub fn nova(arquivo: impl Into<String>, motivo: MotivoFalha) -> Self {
        Self {
            arquivo: arquivo.into(),
            motivo,
        }
    }
}

/// Evento de telemetria no contrato estrito de `/api/launcher/telemetria`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Telemetria {
    pub versao: u32,
    pub build_cliente: u32,
    pub operacao: Operacao,
    pub ok: bool,
    pub duracao_ms: u32,
    pub arquivos: u32,
    pub falhas: Vec<FalhaTelemetria>,
}

impl Telemetria {
    /// JSON exato do contrato: sem campo a mais (a rota é `strictObject`).
    #[must_use]
    pub fn json(&self) -> String {
        let falhas: Vec<String> = self
            .falhas
            .iter()
            .take(10)
            .map(|falha| {
                format!(
                    "{{\"arquivo\":\"{}\",\"motivo\":\"{}\"}}",
                    escapar_json(&falha.arquivo),
                    falha.motivo.como_str()
                )
            })
            .collect();
        format!(
            "{{\"versao\":{},\"build_cliente\":{},\"operacao\":\"{}\",\"ok\":{},\"duracao_ms\":{},\"arquivos\":{},\"falhas\":[{}]}}",
            self.versao,
            self.build_cliente,
            self.operacao.como_str(),
            self.ok,
            self.duracao_ms,
            self.arquivos,
            falhas.join(",")
        )
    }
}

/// Escapa texto para JSON (nome de arquivo vem do manifesto, então não é confiável).
fn escapar_json(texto: &str) -> String {
    let mut saida = String::with_capacity(texto.len());
    for caractere in texto.chars() {
        match caractere {
            '"' => saida.push_str("\\\""),
            '\\' => saida.push_str("\\\\"),
            '\n' => saida.push_str("\\n"),
            '\r' => saida.push_str("\\r"),
            '\t' => saida.push_str("\\t"),
            controle if controle < ' ' => saida.push(' '),
            outro => saida.push(outro),
        }
    }
    saida
}

/// Envia um evento de telemetria; erro é do chamador (o CLI só loga).
pub fn enviar_telemetria(base: &str, evento: &Telemetria) -> Result<(), PortalError> {
    let api = PortalApi::new(base)?;
    api.post_json("/api/launcher/telemetria", &evento.json())?;
    Ok(())
}

/// API do portal a partir de uma base `http://host[:porta]`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PortalApi {
    host: String,
}

impl PortalApi {
    /// Valida a base uma vez; o erro sai aqui, não no meio do patch.
    pub fn new(base: &str) -> Result<Self, PortalError> {
        let resto = base
            .trim()
            .strip_prefix("http://")
            .ok_or(PortalError::BaseInvalida)?;
        let host = resto.split('/').next().unwrap_or("").trim();
        if host.is_empty() || host.contains(char::is_whitespace) {
            return Err(PortalError::BaseInvalida);
        }
        Ok(Self {
            host: host.to_string(),
        })
    }

    /// Lê a versão anunciada.
    pub fn fetch_versao(&self) -> Result<Versao, PortalError> {
        let corpo = self.get("/api/launcher/versao")?;
        Ok(Versao {
            client_build: extrair_u32(&corpo, "client_build")
                .ok_or(PortalError::Campo("client_build"))?,
            protocol_version: extrair_u32(&corpo, "protocol_version")
                .ok_or(PortalError::Campo("protocol_version"))?,
            config_revision: extrair_i64(&corpo, "config_revision"),
            gateway_up: extrair_bool(&corpo, "gateway_up").unwrap_or(false),
        })
    }

    /// Lê a configuração assinada como **texto cru**.
    ///
    /// Este módulo só transporta bytes: quem decide aceitar é o
    /// `configuracao::verificar`, que tem a chave. Assim a config nunca é
    /// interpretada antes de a assinatura conferir.
    pub fn fetch_configuracao(&self) -> Result<String, PortalError> {
        self.get(CONFIGURACAO_PATH)
    }

    /// Lê a grade de eventos (só código/nome/estado: o detalhe fica no portal).
    pub fn fetch_eventos(&self) -> Result<Vec<Evento>, PortalError> {
        let corpo = self.get("/api/launcher/eventos")?;
        let mut eventos = Vec::new();
        for objeto in objetos_do_array(&corpo, "events").ok_or(PortalError::Campo("events"))? {
            eventos.push(Evento {
                code: extrair_texto(&objeto, "code").ok_or(PortalError::Campo("code"))?,
                name: extrair_texto(&objeto, "name").unwrap_or_default(),
                enabled: extrair_bool(&objeto, "enabled").unwrap_or(false),
            });
        }
        Ok(eventos)
    }

    /// Lê a vitrine pública de notícias (recorte sem corpo).
    ///
    /// Lista vazia é resposta válida (`{"noticias":[]}`); array ausente é erro
    /// nomeado, nunca lista inventada.
    pub fn fetch_noticias(&self) -> Result<Vec<Noticia>, PortalError> {
        let corpo = self.get_publico("/api/launcher/noticias")?;
        let mut noticias = Vec::new();
        for objeto in objetos_do_array(&corpo, "noticias").ok_or(PortalError::Campo("noticias"))? {
            noticias.push(Noticia {
                slug: extrair_texto(&objeto, "slug").ok_or(PortalError::Campo("slug"))?,
                titulo: extrair_texto(&objeto, "titulo").ok_or(PortalError::Campo("titulo"))?,
                data: extrair_texto(&objeto, "data").unwrap_or_default(),
                tag: extrair_texto(&objeto, "tag").unwrap_or_default(),
                resumo: extrair_texto(&objeto, "resumo").unwrap_or_default(),
                url: extrair_texto(&objeto, "url").unwrap_or_default(),
            });
        }
        Ok(noticias)
    }

    /// `GET` de bytes crus (manifesto e arquivos do cliente, reparo).
    ///
    /// O corpo não passa por `String`: binário do cliente não é UTF-8.
    pub fn fetch_bytes(&self, caminho: &str) -> Result<Vec<u8>, PortalError> {
        self.request_bytes("GET", caminho, None, TIMEOUT)
    }

    /// `GET` com o timeout normal (config/versão/eventos).
    fn get(&self, caminho: &str) -> Result<String, PortalError> {
        self.request("GET", caminho, None, TIMEOUT)
    }

    /// `GET` com timeout curto (vitrine): portal lento não segura a janela.
    fn get_publico(&self, caminho: &str) -> Result<String, PortalError> {
        self.request("GET", caminho, None, TIMEOUT_PUBLICO)
    }

    /// `POST` JSON com timeout curto (telemetria é best-effort).
    fn post_json(&self, caminho: &str, corpo: &str) -> Result<String, PortalError> {
        self.request("POST", caminho, Some(corpo), TIMEOUT_PUBLICO)
    }

    /// `GET`/`POST` bloqueante em HTTP/1.0; devolve o corpo em 2xx ou o código.
    fn request(
        &self,
        metodo: &str,
        caminho: &str,
        corpo: Option<&str>,
        timeout: Duration,
    ) -> Result<String, PortalError> {
        let bytes = self.request_bytes(metodo, caminho, corpo, timeout)?;
        String::from_utf8(bytes).map_err(|_| PortalError::Resposta)
    }

    fn request_bytes(
        &self,
        metodo: &str,
        caminho: &str,
        corpo: Option<&str>,
        timeout: Duration,
    ) -> Result<Vec<u8>, PortalError> {
        let endereco = if self.host.contains(':') {
            self.host.clone()
        } else {
            format!("{}:80", self.host)
        };
        let socket = endereco
            .to_socket_addrs()
            .map_err(|_| PortalError::Conexao)?
            .next()
            .ok_or(PortalError::Conexao)?;
        let mut conexao =
            TcpStream::connect_timeout(&socket, timeout).map_err(erro_de_io_do_portal)?;
        conexao
            .set_read_timeout(Some(timeout))
            .map_err(erro_de_io_do_portal)?;
        conexao
            .set_write_timeout(Some(timeout))
            .map_err(erro_de_io_do_portal)?;
        let extra = match corpo {
            Some(corpo) => format!(
                "Content-Type: application/json\r\nContent-Length: {}\r\n",
                corpo.len()
            ),
            None => String::new(),
        };
        let pedido = format!(
            "{metodo} {caminho} HTTP/1.0\r\nHost: {}\r\nConnection: close\r\n{extra}\r\n{}",
            self.host,
            corpo.unwrap_or("")
        );
        conexao
            .write_all(pedido.as_bytes())
            .map_err(erro_de_io_do_portal)?;
        let mut bruto = Vec::new();
        conexao
            .read_to_end(&mut bruto)
            .map_err(erro_de_io_do_portal)?;
        // Separa cabeçalho e corpo no primeiro `\r\n\r\n` (o corpo é binário).
        let separador = bruto
            .windows(4)
            .position(|janela| janela == b"\r\n\r\n")
            .ok_or(PortalError::Resposta)?;
        let cabeca = String::from_utf8_lossy(&bruto[..separador]);
        let codigo = cabeca
            .lines()
            .next()
            .and_then(|linha| linha.split_whitespace().nth(1))
            .and_then(|numero| numero.parse::<u16>().ok())
            .ok_or(PortalError::Resposta)?;
        if !(200..300).contains(&codigo) {
            return Err(PortalError::Http(codigo));
        }
        Ok(bruto[separador + 4..].to_vec())
    }
}

/// Erro de socket: tempo esgotado é `Timeout`; o resto é `Conexao`.
fn erro_de_io_do_portal(erro: std::io::Error) -> PortalError {
    match erro.kind() {
        std::io::ErrorKind::TimedOut | std::io::ErrorKind::WouldBlock => PortalError::Timeout,
        _ => PortalError::Conexao,
    }
}

/// Valor bruto após `"chave":`, sem espaços à esquerda.
fn bruto_apos<'a>(json: &'a str, chave: &str) -> Option<&'a str> {
    let marcador = format!("\"{chave}\"");
    let mut resto = json;
    loop {
        let pos = resto.find(marcador.as_str())?;
        let apos = resto[pos + marcador.len()..].trim_start();
        if let Some(valor) = apos.strip_prefix(':') {
            return Some(valor.trim_start());
        }
        resto = &resto[pos + marcador.len()..];
    }
}

fn extrair_i64(json: &str, chave: &str) -> Option<i64> {
    let bruto = bruto_apos(json, chave)?;
    if bruto.starts_with("null") {
        return None;
    }
    let fim = bruto.find(|c: char| !c.is_ascii_digit() && c != '-')?;
    bruto[..fim].parse::<i64>().ok()
}

fn extrair_u32(json: &str, chave: &str) -> Option<u32> {
    extrair_i64(json, chave).and_then(|valor| u32::try_from(valor).ok())
}

fn extrair_bool(json: &str, chave: &str) -> Option<bool> {
    let bruto = bruto_apos(json, chave)?;
    if bruto.starts_with("true") {
        Some(true)
    } else if bruto.starts_with("false") {
        Some(false)
    } else {
        None
    }
}

/// Texto de `"chave": "..."` com `\"`/`\\` resolvidos; `null` vira `None`.
fn extrair_texto(json: &str, chave: &str) -> Option<String> {
    let mut bruto = bruto_apos(json, chave)?.strip_prefix('"')?.chars();
    let mut saida = String::new();
    let mut fuga = false;
    for caractere in &mut bruto {
        if fuga {
            saida.push(caractere);
            fuga = false;
        } else if caractere == '\\' {
            fuga = true;
        } else if caractere == '"' {
            return Some(saida);
        } else {
            saida.push(caractere);
        }
    }
    None
}

/// Objetos de primeiro nível do array `"chave": [...]` (chaves respeitam texto).
fn objetos_do_array(json: &str, chave: &str) -> Option<Vec<String>> {
    let mut bruto = bruto_apos(json, chave)?
        .strip_prefix('[')?
        .chars()
        .peekable();
    let mut objetos = Vec::new();
    let mut atual = String::new();
    let mut profundidade = 0usize;
    let mut em_texto = false;
    let mut fuga = false;
    for caractere in &mut bruto {
        if fuga {
            if profundidade > 0 {
                atual.push(caractere);
            }
            fuga = false;
        } else if em_texto {
            if caractere == '\\' {
                fuga = true;
            } else if caractere == '"' {
                em_texto = false;
            }
            if profundidade > 0 {
                atual.push(caractere);
            }
        } else if caractere == '"' {
            em_texto = true;
            if profundidade > 0 {
                atual.push(caractere);
            }
        } else if caractere == '{' {
            profundidade += 1;
            atual.push(caractere);
        } else if caractere == '}' {
            if profundidade == 0 {
                return None;
            }
            profundidade -= 1;
            atual.push(caractere);
            if profundidade == 0 {
                objetos.push(std::mem::take(&mut atual));
            }
        } else if profundidade > 0 {
            atual.push(caractere);
        } else if caractere == ']' {
            return Some(objetos);
        }
    }
    None
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::net::TcpListener;

    /// Sobe um HTTP falso de uma resposta só; devolve base + handle.
    fn servidor_falso(resposta: &'static str) -> (String, std::thread::JoinHandle<()>) {
        let ouvinte = TcpListener::bind("127.0.0.1:0").expect("porta efêmera");
        let base = format!("http://{}", ouvinte.local_addr().expect("endereço"));
        let handle = std::thread::spawn(move || {
            let (mut fluxo, _) = ouvinte.accept().expect("aceita");
            let mut pedido = [0u8; 512];
            let _ = fluxo.read(&mut pedido);
            let corpo = format!(
                "HTTP/1.0 200 OK\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{resposta}",
                resposta.len()
            );
            let _ = fluxo.write_all(corpo.as_bytes());
        });
        (base, handle)
    }

    fn servidor_falso_status(status: u16) -> (String, std::thread::JoinHandle<()>) {
        let ouvinte = TcpListener::bind("127.0.0.1:0").expect("porta efêmera");
        let base = format!("http://{}", ouvinte.local_addr().expect("endereço"));
        let handle = std::thread::spawn(move || {
            let (mut fluxo, _) = ouvinte.accept().expect("aceita");
            let mut pedido = [0u8; 512];
            let _ = fluxo.read(&mut pedido);
            let _ = fluxo.write_all(
                format!("HTTP/1.0 {status} Erro\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
                    .as_bytes(),
            );
        });
        (base, handle)
    }

    #[test]
    fn base_fora_de_http_e_rejeitada_antes_de_conectar() {
        assert_eq!(
            PortalApi::new("https://host:3000"),
            Err(PortalError::BaseInvalida)
        );
        assert_eq!(
            PortalApi::new("host:3000/x"),
            Err(PortalError::BaseInvalida)
        );
        assert_eq!(PortalApi::new("http://"), Err(PortalError::BaseInvalida));
        assert!(PortalApi::new("http://127.0.0.1:3000/").is_ok());
    }

    #[test]
    fn versao_vem_do_portal_com_revisao() {
        let (base, handle) = servidor_falso(
            r#"{"client_build":7662,"protocol_version":12000,"config_revision":41,"gateway_up":true}"#,
        );
        let versao = PortalApi::new(&base)
            .expect("base")
            .fetch_versao()
            .expect("lê");
        handle.join().expect("servidor");
        assert_eq!(versao.client_build, 7662);
        assert_eq!(versao.protocol_version, 12_000);
        assert_eq!(versao.config_revision, Some(41));
        assert!(versao.gateway_up);
    }

    #[test]
    fn versao_sem_revisao_vira_none_sem_inventar_numero() {
        let (base, handle) = servidor_falso(
            r#"{"client_build":7662,"protocol_version":12000,"config_revision":null,"gateway_up":false}"#,
        );
        let versao = PortalApi::new(&base)
            .expect("base")
            .fetch_versao()
            .expect("lê");
        handle.join().expect("servidor");
        assert_eq!(versao.config_revision, None);
        assert!(!versao.gateway_up);
    }

    #[test]
    fn campo_obrigatorio_ausente_e_erro_nomeado() {
        let (base, handle) = servidor_falso(r#"{"protocol_version":12000}"#);
        let erro = PortalApi::new(&base)
            .expect("base")
            .fetch_versao()
            .expect_err("falha");
        handle.join().expect("servidor");
        assert_eq!(erro, PortalError::Campo("client_build"));
    }

    #[test]
    fn http_fora_de_200_vira_erro_com_codigo() {
        let (base, handle) = servidor_falso_status(503);
        let erro = PortalApi::new(&base)
            .expect("base")
            .fetch_versao()
            .expect_err("falha");
        handle.join().expect("servidor");
        assert_eq!(erro, PortalError::Http(503));
    }

    #[test]
    fn eventos_chegam_com_codigo_e_estado() {
        let (base, handle) = servidor_falso(
            r#"{"events":[{"code":"invasao","name":"Invasão","enabled":true},{"code":"xp2","name":"XP em dobro","enabled":false}]}"#,
        );
        let eventos = PortalApi::new(&base)
            .expect("base")
            .fetch_eventos()
            .expect("lê");
        handle.join().expect("servidor");
        assert_eq!(
            eventos,
            vec![
                Evento {
                    code: "invasao".to_string(),
                    name: "Invasão".to_string(),
                    enabled: true
                },
                Evento {
                    code: "xp2".to_string(),
                    name: "XP em dobro".to_string(),
                    enabled: false
                },
            ]
        );
    }

    #[test]
    fn configuracao_chega_como_texto_cru() {
        let corpo = "# wyd-launcher-config v1\nrevisao = 7\nassinatura = ";
        let (base, handle) = servidor_falso(corpo);
        let texto = PortalApi::new(&base)
            .expect("base")
            .fetch_configuracao()
            .expect("lê");
        handle.join().expect("servidor");
        assert_eq!(texto, corpo, "nada é reinterpretado antes da assinatura");
    }

    #[test]
    fn configuracao_nao_publicada_e_http_404() {
        let (base, handle) = servidor_falso_status(404);
        let erro = PortalApi::new(&base)
            .expect("base")
            .fetch_configuracao()
            .expect_err("falha");
        handle.join().expect("servidor");
        assert_eq!(erro, PortalError::Http(404));
    }

    #[test]
    fn grade_vazia_e_lista_vazia_nao_erro() {
        let (base, handle) = servidor_falso(r#"{"events":[]}"#);
        let eventos = PortalApi::new(&base)
            .expect("base")
            .fetch_eventos()
            .expect("lê");
        handle.join().expect("servidor");
        assert!(eventos.is_empty());
    }

    #[test]
    fn portal_fora_do_ar_e_erro_de_conexao() {
        let api = PortalApi::new("http://127.0.0.1:1").expect("base");
        assert_eq!(api.fetch_versao().expect_err("falha"), PortalError::Conexao);
    }

    fn noticia_esperada() -> Noticia {
        Noticia {
            slug: "a".to_string(),
            titulo: "A".to_string(),
            data: "2026-11-11".to_string(),
            tag: "Jogo".to_string(),
            resumo: "resumo".to_string(),
            url: "/noticias/a".to_string(),
        }
    }

    #[test]
    fn noticias_chegam_com_o_recorte_da_vitrine() {
        let (base, handle) = servidor_falso(
            r##"{"noticias":[{"slug":"a","titulo":"A","data":"2026-11-11","tag":"Jogo","resumo":"resumo","url":"/noticias/a","body":"corpo que nao deve sair"}]}"##,
        );
        let noticias = PortalApi::new(&base)
            .expect("base")
            .fetch_noticias()
            .expect("lê");
        handle.join().expect("servidor");
        assert_eq!(noticias, vec![noticia_esperada()]);
    }

    #[test]
    fn noticias_vazias_sao_lista_vazia() {
        let (base, handle) = servidor_falso(r#"{"noticias":[]}"#);
        let noticias = PortalApi::new(&base)
            .expect("base")
            .fetch_noticias()
            .expect("lê");
        handle.join().expect("servidor");
        assert!(noticias.is_empty());
    }

    #[test]
    fn noticias_sem_array_e_erro_nomeado() {
        let (base, handle) = servidor_falso(r#"{}"#);
        let erro = PortalApi::new(&base)
            .expect("base")
            .fetch_noticias()
            .expect_err("falha");
        handle.join().expect("servidor");
        assert_eq!(erro, PortalError::Campo("noticias"));
    }

    #[test]
    fn noticias_4xx_vira_http_com_codigo() {
        let (base, handle) = servidor_falso_status(404);
        let erro = PortalApi::new(&base)
            .expect("base")
            .fetch_noticias()
            .expect_err("falha");
        handle.join().expect("servidor");
        assert_eq!(erro, PortalError::Http(404));
    }

    /// Servidor que aceita e nunca responde: prova o timeout curto da vitrine.
    fn servidor_mudo() -> (String, std::thread::JoinHandle<()>) {
        let ouvinte = TcpListener::bind("127.0.0.1:0").expect("porta efêmera");
        let base = format!("http://{}", ouvinte.local_addr().expect("endereço"));
        let handle = std::thread::spawn(move || {
            if let Ok((mut fluxo, _)) = ouvinte.accept() {
                let mut pedido = [0u8; 512];
                let _ = fluxo.read(&mut pedido);
                std::thread::sleep(Duration::from_secs(2));
            }
        });
        (base, handle)
    }

    #[test]
    fn noticias_com_portal_mudo_estouram_o_timeout_curto() {
        let (base, handle) = servidor_mudo();
        let inicio = std::time::Instant::now();
        let erro = PortalApi::new(&base)
            .expect("base")
            .fetch_noticias()
            .expect_err("falha");
        assert_eq!(erro, PortalError::Timeout);
        assert!(
            inicio.elapsed() < Duration::from_secs(2),
            "vitrine não pode esperar o timeout longo"
        );
        let _ = handle.join();
    }

    /// Servidor que guarda o pedido cru e responde com `status`.
    fn servidor_captura(
        status: u16,
        corpo_resposta: &'static str,
    ) -> (
        String,
        std::sync::Arc<std::sync::Mutex<String>>,
        std::thread::JoinHandle<()>,
    ) {
        let ouvinte = TcpListener::bind("127.0.0.1:0").expect("porta efêmera");
        let base = format!("http://{}", ouvinte.local_addr().expect("endereço"));
        let guardado = std::sync::Arc::new(std::sync::Mutex::new(String::new()));
        let saida = std::sync::Arc::clone(&guardado);
        let handle = std::thread::spawn(move || {
            let (mut fluxo, _) = ouvinte.accept().expect("aceita");
            let mut bruto = Vec::new();
            let mut buffer = [0u8; 1024];
            // Lê cabeçalho e, se houver, o corpo declarado em Content-Length.
            loop {
                let lido = match fluxo.read(&mut buffer) {
                    Ok(0) | Err(_) => break,
                    Ok(n) => n,
                };
                bruto.extend_from_slice(&buffer[..lido]);
                if let Some(fim) = bruto.windows(4).position(|janela| janela == b"\r\n\r\n") {
                    let cabeca = String::from_utf8_lossy(&bruto[..fim]).to_lowercase();
                    let tamanho = cabeca
                        .lines()
                        .find_map(|linha| linha.strip_prefix("content-length:"))
                        .and_then(|valor| valor.trim().parse::<usize>().ok())
                        .unwrap_or(0);
                    if bruto.len() >= fim + 4 + tamanho {
                        break;
                    }
                }
            }
            *saida.lock().expect("lock") = String::from_utf8_lossy(&bruto).to_string();
            let _ = fluxo.write_all(
                format!(
                    "HTTP/1.0 {status} OK\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{corpo_resposta}",
                    corpo_resposta.len()
                )
                .as_bytes(),
            );
        });
        (base, guardado, handle)
    }

    fn evento() -> Telemetria {
        Telemetria {
            versao: VERSAO_LAUNCHER,
            build_cliente: BUILD_CLIENTE,
            operacao: Operacao::Patch,
            ok: true,
            duracao_ms: 250,
            arquivos: 3,
            falhas: vec![FalhaTelemetria::nova("jogo.dat", MotivoFalha::Hash)],
        }
    }

    #[test]
    fn telemetria_envia_post_com_o_contrato_exato() {
        let (base, pedido, handle) = servidor_captura(204, "");
        enviar_telemetria(&base, &evento()).expect("aceito");
        handle.join().expect("servidor");
        let cru = pedido.lock().expect("lock").clone();
        assert!(
            cru.starts_with("POST /api/launcher/telemetria HTTP/1.0\r\n"),
            "método/rota errados: {cru}"
        );
        let corpo = cru.split("\r\n\r\n").nth(1).expect("corpo");
        assert_eq!(corpo, evento().json());
        assert!(corpo.contains("\"operacao\":\"patch\""));
        assert!(!corpo.contains("login"), "nada de PII no corpo");
    }

    #[test]
    fn telemetria_recusada_vira_http_com_codigo() {
        let (base, _pedido, handle) = servidor_captura(400, r#"{"error":"fora do contrato"}"#);
        let erro = enviar_telemetria(&base, &evento()).expect_err("falha");
        handle.join().expect("servidor");
        assert_eq!(erro, PortalError::Http(400));
    }

    #[test]
    fn telemetria_com_portal_mudo_estoura_o_timeout_sem_travar() {
        let (base, handle) = servidor_mudo();
        let inicio = std::time::Instant::now();
        let erro = enviar_telemetria(&base, &evento()).expect_err("falha");
        assert_eq!(erro, PortalError::Timeout);
        assert!(inicio.elapsed() < Duration::from_secs(2));
        let _ = handle.join();
    }

    #[test]
    fn json_da_telemetria_escapa_nome_e_tem_no_maximo_dez_falhas() {
        let mut evento = evento();
        evento.falhas = (0..12)
            .map(|i| FalhaTelemetria::nova(format!("a\"{i}.dat"), MotivoFalha::Io))
            .collect();
        let json = evento.json();
        assert!(json.contains("\\\""), "aspas precisam ser escapadas");
        assert_eq!(json.matches("\"arquivo\"").count(), 10);
    }
}
