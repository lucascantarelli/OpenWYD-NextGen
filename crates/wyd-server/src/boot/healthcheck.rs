//! `--healthcheck`: `GET /health` no próprio processo; só 200 é saudável.

use std::io::{BufRead, BufReader, Write};
use std::net::TcpStream;
use std::process::ExitCode;
use std::time::Duration;

use super::env::DEFAULT_HEALTHCHECK;

pub(crate) fn healthcheck() -> ExitCode {
    let addr = std::env::var("WYD_SERVER_HEALTHCHECK_ADDR")
        .unwrap_or_else(|_| DEFAULT_HEALTHCHECK.to_string());
    let Ok(mut stream) = TcpStream::connect(&addr) else {
        return ExitCode::FAILURE;
    };
    let _ = stream.set_read_timeout(Some(Duration::from_secs(5)));
    let request = "GET /health HTTP/1.1\r\nhost: localhost\r\nconnection: close\r\n\r\n";
    if stream.write_all(request.as_bytes()).is_err() {
        return ExitCode::FAILURE;
    }
    let mut reader = BufReader::new(stream);
    let mut status_line = String::new();
    if reader.read_line(&mut status_line).is_err() {
        return ExitCode::FAILURE;
    }
    if status_line.starts_with("HTTP/1.1 200") {
        ExitCode::SUCCESS
    } else {
        ExitCode::FAILURE
    }
}
