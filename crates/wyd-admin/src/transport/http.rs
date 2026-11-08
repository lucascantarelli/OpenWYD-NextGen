//! HTTP/1.1 mínimo em std para o Cold Path (ADR-0008).
//!
//! Loopback, sem keep-alive, `Content-Length` obrigatório em POST, corpo
//! limitado. Só lê head + corpo e escreve uma resposta; o roteamento mora em
//! `transport::mod`. Nenhum `unsafe` e nenhum `panic!` com entrada de rede.

use std::io::{self, BufRead, Write};
use std::net::TcpListener;

/// Corpo máximo aceito/emitido (o maior payload do contrato é o mapa de atributos).
pub const MAX_BODY: usize = 32 * 1024 * 1024;

/// Head máximo (linha de request + headers), para não crescer memória sem limite.
pub const MAX_HEAD: usize = 16 * 1024;

/// Requisição já lida do socket.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Request {
    pub method: String,
    pub path: String,
    pub headers: Vec<(String, String)>,
    pub body: Vec<u8>,
}

impl Request {
    /// Header por nome, sem diferenciar maiúsculas.
    #[must_use]
    pub fn header(&self, name: &str) -> Option<&str> {
        self.headers
            .iter()
            .find(|(key, _)| key.eq_ignore_ascii_case(name))
            .map(|(_, value)| value.as_str())
    }

    /// `x-trace-id` da chamada (vazio quando o cliente não manda).
    #[must_use]
    pub fn trace_id(&self) -> &str {
        self.header("x-trace-id").unwrap_or("")
    }
}

/// Falha de transporte (vira status HTTP, nunca `result` de negócio).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum HttpError {
    BadRequest,
    MethodNotAllowed,
    PayloadTooLarge,
    HeadersTooLarge,
}

impl HttpError {
    #[must_use]
    pub const fn status(self) -> u16 {
        match self {
            Self::BadRequest => 400,
            Self::MethodNotAllowed => 405,
            Self::PayloadTooLarge => 413,
            Self::HeadersTooLarge => 431,
        }
    }
}

/// Lê uma requisição completa (request line + headers + corpo).
pub fn read_request<R: BufRead>(reader: &mut R, max_body: usize) -> Result<Request, HttpError> {
    let mut head = Vec::new();
    let mut total = 0usize;
    loop {
        let mut line = Vec::new();
        let read = reader
            .read_until(b'\n', &mut line)
            .map_err(|_| HttpError::BadRequest)?;
        if read == 0 {
            return Err(HttpError::BadRequest);
        }
        total = total.saturating_add(read);
        if total > MAX_HEAD {
            return Err(HttpError::HeadersTooLarge);
        }
        let is_blank = line == b"\r\n" || line == b"\n";
        head.extend_from_slice(&line);
        if is_blank {
            break;
        }
    }
    let text = std::str::from_utf8(&head).map_err(|_| HttpError::BadRequest)?;
    let mut lines = text.split("\r\n").flat_map(|l| l.split('\n'));
    let request_line = lines.next().ok_or(HttpError::BadRequest)?;
    let mut parts = request_line.split(' ');
    let (Some(method), Some(path), Some(version), None) =
        (parts.next(), parts.next(), parts.next(), parts.next())
    else {
        return Err(HttpError::BadRequest);
    };
    if !version.starts_with("HTTP/1.") || method.is_empty() || path.is_empty() {
        return Err(HttpError::BadRequest);
    }

    let mut headers: Vec<(String, String)> = Vec::new();
    let mut content_length: Option<usize> = None;
    for line in lines {
        if line.is_empty() {
            continue;
        }
        let (name, value) = line.split_once(':').ok_or(HttpError::BadRequest)?;
        let name = name.trim().to_ascii_lowercase();
        let value = value.trim();
        if name.is_empty() {
            return Err(HttpError::BadRequest);
        }
        if name == "transfer-encoding" {
            // v1 do contrato não aceita chunked.
            return Err(HttpError::BadRequest);
        }
        if name == "content-length" {
            let parsed = value.parse::<usize>().map_err(|_| HttpError::BadRequest)?;
            if content_length.is_some() {
                return Err(HttpError::BadRequest);
            }
            content_length = Some(parsed);
        }
        headers.push((name, value.to_string()));
    }

    if method != "POST" {
        return Err(HttpError::MethodNotAllowed);
    }
    let length = content_length.ok_or(HttpError::BadRequest)?;
    if length > max_body {
        return Err(HttpError::PayloadTooLarge);
    }
    let mut body = vec![0u8; length];
    io::Read::read_exact(reader, &mut body).map_err(|_| HttpError::BadRequest)?;
    Ok(Request {
        method: method.to_string(),
        path: path.to_string(),
        headers,
        body,
    })
}

/// Escreve a resposta com os headers fixos do contrato (`no-store`, `close`).
pub fn write_response<W: Write>(
    out: &mut W,
    status: u16,
    trace_id: &str,
    body: &str,
) -> io::Result<()> {
    let mut head = String::with_capacity(192);
    head.push_str("HTTP/1.1 ");
    head.push_str(&status.to_string());
    head.push(' ');
    head.push_str(reason(status));
    head.push_str("\r\ncontent-type: application/json\r\ncache-control: no-store\r\n");
    head.push_str("x-trace-id: ");
    head.push_str(trace_id);
    head.push_str("\r\ncontent-length: ");
    head.push_str(&body.len().to_string());
    head.push_str("\r\nconnection: close\r\n\r\n");
    out.write_all(head.as_bytes())?;
    out.write_all(body.as_bytes())?;
    out.flush()
}

const fn reason(status: u16) -> &'static str {
    match status {
        200 => "OK",
        400 => "Bad Request",
        401 => "Unauthorized",
        404 => "Not Found",
        405 => "Method Not Allowed",
        413 => "Payload Too Large",
        431 => "Request Header Fields Too Large",
        _ => "Internal Server Error",
    }
}

/// Abre o listener **somente** em loopback (ADR-0008 §2: nunca `0.0.0.0`).
pub fn bind_loopback(port: u16) -> io::Result<TcpListener> {
    TcpListener::bind(("127.0.0.1", port))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Cursor;

    fn request(raw: &str) -> Result<Request, HttpError> {
        let mut cursor = Cursor::new(raw.as_bytes().to_vec());
        read_request(&mut cursor, 1024)
    }

    #[test]
    fn reads_post_with_body_and_headers() {
        let raw = "POST /admin/v1/accounts/find HTTP/1.1\r\nHost: x\r\nAuthorization: Bearer t\r\nContent-Length: 2\r\n\r\n{}";
        let req = request(raw).expect("request válida");
        assert_eq!(req.path, "/admin/v1/accounts/find");
        assert_eq!(req.header("authorization"), Some("Bearer t"));
        assert_eq!(req.body, b"{}");
    }

    #[test]
    fn rejects_get_missing_length_and_chunked() {
        assert_eq!(
            request("GET /admin/v1/x/y HTTP/1.1\r\n\r\n"),
            Err(HttpError::MethodNotAllowed)
        );
        assert_eq!(
            request("POST /admin/v1/x/y HTTP/1.1\r\n\r\n"),
            Err(HttpError::BadRequest)
        );
        assert_eq!(
            request("POST / HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Length: 0\r\n\r\n"),
            Err(HttpError::BadRequest)
        );
    }

    #[test]
    fn rejects_body_above_limit() {
        let raw = "POST / HTTP/1.1\r\nContent-Length: 99999\r\n\r\n";
        assert_eq!(request(raw), Err(HttpError::PayloadTooLarge));
    }

    #[test]
    fn writes_no_store_and_trace() {
        let mut out = Vec::new();
        write_response(&mut out, 200, "trace-1", "{\"result\":1}").expect("escreve");
        let text = String::from_utf8(out).expect("utf-8");
        assert!(text.starts_with("HTTP/1.1 200 OK\r\n"));
        assert!(text.contains("cache-control: no-store"));
        assert!(text.contains("x-trace-id: trace-1"));
        assert!(text.ends_with("{\"result\":1}"));
    }

    #[test]
    fn bind_is_loopback_only() {
        let listener = bind_loopback(0).expect("porta efêmera");
        let addr = listener.local_addr().expect("endereço local");
        assert!(addr.ip().is_loopback());
    }
}
