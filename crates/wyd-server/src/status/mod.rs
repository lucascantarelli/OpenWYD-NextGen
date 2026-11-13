//! Status do canal: `serv00.htm`, `/health` e sonda UDP do seletor do cliente.
//!
//! ADR-0026 (etapa 5): a fachada só declara o domínio; HTTP e UDP ficam em
//! [`service`].

mod service;

pub use service::*;
