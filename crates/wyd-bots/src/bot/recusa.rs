//! Submódulo de `bot` (ADR-0027).

use super::*;

/// Motivo em português para uma recusa do handshake (é a mensagem da tela de conexão).
///
/// O cliente 7662 é fechado e não exibe nada ao receber `*Fail`; esta é a superfície sob
/// nosso controle e cobre o que o servidor já manda hoje. Sempre devolve texto acionável —
/// nunca vazio, para não existir "clico e nada acontece".
#[must_use]
pub fn motivo_da_recusa(opcode: Opcode) -> String {
    match opcode {
        Opcode::NewAccountFail => {
            "Conta ou senha incorreta — confira os dados e tente novamente.".to_string()
        }
        Opcode::AccountSecureFail => "PIN incorreto ou bloqueado por tentativas — confira o PIN \
             e tente novamente; após várias falhas aguarde alguns minutos."
            .to_string(),
        Opcode::CharacterLoginFail => {
            "Personagem indisponível — escolha outro slot ou crie um personagem.".to_string()
        }
        outro => format!(
            "O servidor recusou a conexão ({}). Tente novamente em instantes.",
            outro.wire()
        ),
    }
}
