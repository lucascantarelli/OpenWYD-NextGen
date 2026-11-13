//! `tmserver` — binário do canal. Toda a orquestração vive em [`wyd_server::boot`].
//!
//! `--healthcheck` faz `GET /health` no próprio processo; sem a flag, sobe o canal:
//! CPSock `:8281`, status `/serv00.htm`/`/health`, `/metrics` `:9898` e a thread de tick.

use std::process::ExitCode;

fn main() -> ExitCode {
    wyd_server::boot::main()
}
