//! Erro único do crate: uma variante por fronteira.
//!
//! Os erros específicos continuam nos seus módulos; aqui há conversão `From`
//! para quem prefere tratar tudo como [`ProtocolError`].

use core::fmt;

use crate::codec::keyword::TransformError;
use crate::payload::PayloadError;
use crate::wire::frame::FrameError;
use crate::wire::header::HeaderError;

/// Erro único do `wyd-protocol`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProtocolError {
    /// Falha no cabeçalho de 12 bytes.
    Header(HeaderError),
    /// Falha no enquadramento por `Size`.
    Frame(FrameError),
    /// Falha na transformação/ofuscação.
    Transform(TransformError),
    /// Falha no layout de um payload.
    Payload(PayloadError),
}

impl fmt::Display for ProtocolError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Header(e) => write!(f, "cabeçalho: {e}"),
            Self::Frame(e) => write!(f, "enquadramento: {e}"),
            Self::Transform(e) => write!(f, "transformação: {e}"),
            Self::Payload(e) => write!(f, "payload: {e}"),
        }
    }
}

impl std::error::Error for ProtocolError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Header(e) => Some(e),
            Self::Frame(e) => Some(e),
            Self::Transform(e) => Some(e),
            Self::Payload(e) => Some(e),
        }
    }
}

impl From<HeaderError> for ProtocolError {
    fn from(e: HeaderError) -> Self {
        Self::Header(e)
    }
}

impl From<FrameError> for ProtocolError {
    fn from(e: FrameError) -> Self {
        Self::Frame(e)
    }
}

impl From<TransformError> for ProtocolError {
    fn from(e: TransformError) -> Self {
        Self::Transform(e)
    }
}

impl From<PayloadError> for ProtocolError {
    fn from(e: PayloadError) -> Self {
        Self::Payload(e)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn every_boundary_converts_into_protocol_error() {
        let h: ProtocolError = HeaderError::SizeOutOfRange { size: 11 }.into();
        assert!(matches!(h, ProtocolError::Header(_)));
        let f: ProtocolError = FrameError::SizeOutOfRange { size: 11 }.into();
        assert!(matches!(f, ProtocolError::Frame(_)));
        let t: ProtocolError = TransformError::InvalidSize { size: 11 }.into();
        assert!(matches!(t, ProtocolError::Transform(_)));
        let p: ProtocolError = PayloadError::BadLength {
            expected: 116,
            got: 11,
        }
        .into();
        assert!(matches!(p, ProtocolError::Payload(_)));
        assert!(format!("{h}").contains("cabeçalho"));
    }
}
