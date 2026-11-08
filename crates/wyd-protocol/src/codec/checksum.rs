//! Política de checksum do CPSock: soma (claro × cifrado) + decisão.
//!
//! O checksum é `soma(cifrado) - soma(claro)` com wrap de 8 bits, calculado por
//! [`decode_in_place`](super::keyword::decode_in_place). O legado **tolerava**
//! divergência (`CPSock.cpp:457-466`); a decisão da stack nova está registrada em
//! `docs/arquitetura/protocolo.md` §3: entrada de rede usa
//! [`ChecksumPolicy::Strict`], captura e diagnóstico usam [`ChecksumPolicy::Tolerant`].

use super::keyword::TransformError;

/// Política de verificação do checksum declarado no `HEADER`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum ChecksumPolicy {
    /// Divergência é erro tipado (entrada de rede do servidor).
    #[default]
    Strict,
    /// Divergência só é observável (captura, diagnóstico, RE).
    Tolerant,
}

/// Verifica o checksum calculado contra o declarado, conforme a política.
pub fn verify(computed: u8, declared: u8, policy: ChecksumPolicy) -> Result<(), TransformError> {
    match policy {
        ChecksumPolicy::Strict if computed != declared => {
            Err(TransformError::ChecksumMismatch { computed, declared })
        }
        _ => Ok(()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn strict_rejects_mismatch_and_tolerant_lets_it_pass() {
        assert_eq!(verify(0xAB, 0xAB, ChecksumPolicy::Strict), Ok(()));
        assert_eq!(
            verify(0xAB, 0xAC, ChecksumPolicy::Strict),
            Err(TransformError::ChecksumMismatch {
                computed: 0xAB,
                declared: 0xAC
            })
        );
        assert_eq!(verify(0xAB, 0xAC, ChecksumPolicy::Tolerant), Ok(()));
    }
}
