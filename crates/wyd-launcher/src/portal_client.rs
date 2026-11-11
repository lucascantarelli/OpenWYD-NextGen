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

/// Erro tipado da leitura do portal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum PortalError {
    /// Base fora de `http://host[:porta]`.
    BaseInvalida,
    /// Sem conexão com o portal (fora do ar ou endereço errado).
    Conexao,
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

    /// `GET` bloqueante em HTTP/1.0; devolve o corpo com 200 ou o código.
    fn get(&self, caminho: &str) -> Result<String, PortalError> {
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
            TcpStream::connect_timeout(&socket, TIMEOUT).map_err(|_| PortalError::Conexao)?;
        conexao
            .set_read_timeout(Some(TIMEOUT))
            .map_err(|_| PortalError::Conexao)?;
        let pedido = format!(
            "GET {caminho} HTTP/1.0\r\nHost: {}\r\nConnection: close\r\n\r\n",
            self.host
        );
        conexao
            .write_all(pedido.as_bytes())
            .map_err(|_| PortalError::Conexao)?;
        let mut bruto = Vec::new();
        conexao
            .read_to_end(&mut bruto)
            .map_err(|_| PortalError::Conexao)?;
        let texto = String::from_utf8(bruto).map_err(|_| PortalError::Resposta)?;
        let (cabeca, corpo) = texto.split_once("\r\n\r\n").ok_or(PortalError::Resposta)?;
        let codigo = cabeca
            .lines()
            .next()
            .and_then(|linha| linha.split_whitespace().nth(1))
            .and_then(|numero| numero.parse::<u16>().ok())
            .ok_or(PortalError::Resposta)?;
        if codigo != 200 {
            return Err(PortalError::Http(codigo));
        }
        Ok(corpo.to_string())
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
}
