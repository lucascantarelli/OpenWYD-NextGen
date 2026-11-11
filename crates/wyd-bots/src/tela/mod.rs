//! Tela de conexão (renderizador único): traduz a recusa do handshake para a frase que o
//! jogador lê. O **mesmo** `render` é usado pelo modo headless (CLI) e pelo caminho
//! GUI/launcher — não existe texto duplicado em cada frente.
//!
//! O protocolo só manda o opcode `*Fail` (sem corpo com motivo), então o que o servidor não
//! distingue no fio é classificado aqui pelo contexto da etapa que falhou; nunca devolve
//! string vazia.

use std::io::ErrorKind;

use wyd_protocol::wire::opcode::Opcode;

/// Recusa que a tela sabe explicar. `Desconhecida` é o último recurso acionável.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Recusa {
    SenhaIncorreta,
    PinIncorreto,
    PinBloqueado,
    SelecaoInvalida,
    PersonagemInexistente,
    ContaEmUso,
    ZonaIndisponivel,
    Desconhecida(String),
}

/// Texto da tela para a recusa — sempre em português, sempre com próximo passo.
#[must_use]
pub fn render(recusa: &Recusa) -> String {
    match recusa {
        Recusa::SenhaIncorreta => {
            "Conta ou senha incorreta. Confira o nome da conta e a senha e tente novamente."
                .to_string()
        }
        Recusa::PinIncorreto => {
            "PIN incorreto. Digite os 6 dígitos do PIN e tente novamente.".to_string()
        }
        Recusa::PinBloqueado => {
            "PIN bloqueado por várias tentativas. Aguarde alguns minutos e tente novamente."
                .to_string()
        }
        Recusa::SelecaoInvalida => {
            "Seleção de personagem inválida. Escolha o personagem pela lista e tente de novo."
                .to_string()
        }
        Recusa::PersonagemInexistente => {
            "Personagem indisponível nesse slot. Escolha outro slot ou crie um personagem."
                .to_string()
        }
        Recusa::ContaEmUso => {
            "Sua conta está em uso em outra sessão. Feche o outro cliente e tente novamente."
                .to_string()
        }
        Recusa::ZonaIndisponivel => {
            "Zona indisponível no momento. Tente novamente em instantes ou escolha outro canal."
                .to_string()
        }
        Recusa::Desconhecida(motivo) => {
            format!("Não foi possível conectar ({motivo}). Verifique os dados e tente novamente.")
        }
    }
}

/// Classifica um texto de erro/recusa vindo da borda. Reconhece o que o servidor manda e cai
/// no genérico acionável quando o motivo é vazio ou não mapeado.
#[must_use]
pub fn classificar(motivo: &str) -> Recusa {
    let texto = motivo.to_lowercase();
    if texto.trim().is_empty() {
        return Recusa::Desconhecida("sem motivo informado".to_string());
    }
    // Prefixo estável da borda: `recusa:<wire>:<texto>`. Classificar pelo texto em português
    // misturaria casos (a frase de PIN cita bloqueio e viraria PinBloqueado).
    if let Some(resto) = texto.strip_prefix("recusa:") {
        let mut partes = resto.split(':');
        if let (Some(wire), Some(codigo)) = (partes.next(), partes.next()) {
            // Código do servidor manda; 0/ausente cai na classificação por opcode.
            if let Ok(numero) = codigo.parse::<u8>() {
                if numero != 0 {
                    return por_codigo(numero);
                }
            }
            if let Ok(numero) = wire.parse::<u16>() {
                if let Some(opcode) = Opcode::from_wire(numero) {
                    return do_opcode(opcode);
                }
            }
        }
    }
    if texto.contains("senha numérica") || texto.contains("pin") {
        return if texto.contains("bloque") || texto.contains("aguarde") || texto.contains("lockout")
        {
            Recusa::PinBloqueado
        } else {
            Recusa::PinIncorreto
        };
    }
    if texto.contains("em uso") || texto.contains("sess") {
        return Recusa::ContaEmUso;
    }
    if texto.contains("slot") || texto.contains("personagem") {
        return Recusa::PersonagemInexistente;
    }
    if texto.contains("payload") || texto.contains("tamanho") {
        return Recusa::SelecaoInvalida;
    }
    if texto.contains("zona") || texto.contains("cena") || texto.contains("canal") {
        return Recusa::ZonaIndisponivel;
    }
    if texto.contains("senha") || texto.contains("conta") {
        return Recusa::SenhaIncorreta;
    }
    Recusa::Desconhecida(motivo.trim().to_string())
}

/// Classifica o opcode de recusa do servidor para a etapa correspondente.
#[must_use]
pub fn do_opcode(opcode: Opcode) -> Recusa {
    match opcode {
        Opcode::NewAccountFail => Recusa::SenhaIncorreta,
        Opcode::AccountSecureFail => Recusa::PinIncorreto,
        Opcode::CharacterLoginFail => Recusa::PersonagemInexistente,
        outro => Recusa::Desconhecida(format!("opcode {}", outro.wire())),
    }
}

/// Classifica um erro de I/O do handshake (socket fechado sem resposta).
#[must_use]
pub fn do_erro(erro: &std::io::Error) -> Recusa {
    if matches!(
        erro.kind(),
        ErrorKind::UnexpectedEof | ErrorKind::ConnectionReset | ErrorKind::BrokenPipe
    ) {
        Recusa::SenhaIncorreta
    } else {
        classificar(&erro.to_string())
    }
}

/// Códigos do trailer de motivo (`wyd_protocol::motivo::CodigoRecusa`) → recusa da tela.
#[must_use]
pub fn por_codigo(codigo: u8) -> Recusa {
    match codigo {
        1 => Recusa::SenhaIncorreta,
        2 => Recusa::PinIncorreto,
        3 => Recusa::PinBloqueado,
        4 => Recusa::SelecaoInvalida,
        5 => Recusa::PersonagemInexistente,
        6 => Recusa::ContaEmUso,
        7 => Recusa::ZonaIndisponivel,
        _ => Recusa::Desconhecida("motivo não informado pelo servidor".to_string()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn todas() -> Vec<Recusa> {
        vec![
            Recusa::SenhaIncorreta,
            Recusa::PinIncorreto,
            Recusa::PinBloqueado,
            Recusa::SelecaoInvalida,
            Recusa::PersonagemInexistente,
            Recusa::ContaEmUso,
            Recusa::ZonaIndisponivel,
            Recusa::Desconhecida("timeout".to_string()),
        ]
    }

    #[test]
    fn cada_recusa_tem_frase_propria_e_acionavel() {
        let mut vistas = std::collections::BTreeSet::new();
        for recusa in todas() {
            let texto = render(&recusa);
            assert!(!texto.trim().is_empty(), "{recusa:?} vazia");
            assert!(vistas.insert(texto.clone()), "texto repetido: {texto}");
        }
    }

    #[test]
    fn motivo_vazio_ou_desconhecido_nao_vira_tela_em_branco() {
        for motivo in ["", "   ", "algo nunca visto"] {
            let texto = render(&classificar(motivo));
            assert!(!texto.trim().is_empty());
            assert!(texto.contains("Não foi possível conectar") || texto.contains("tente"));
        }
    }

    #[test]
    fn os_sete_codigos_e_o_ausente_passam_pelo_mesmo_render() {
        let mut vistas = std::collections::BTreeSet::new();
        for codigo in 0u8..=7 {
            let texto = render(&por_codigo(codigo));
            assert!(!texto.trim().is_empty(), "codigo {codigo} sem texto");
            assert!(
                vistas.insert(texto.clone()),
                "codigo {codigo} repetiu: {texto}"
            );
        }
        assert_eq!(
            vistas.len(),
            8,
            "oito frases distintas (7 causas + fallback)"
        );
    }

    #[test]
    fn classificacao_distingue_pin_errado_de_bloqueado() {
        assert_eq!(
            classificar("senha numérica não confere"),
            Recusa::PinIncorreto
        );
        assert_eq!(
            classificar("pin bloqueado, aguarde 300s"),
            Recusa::PinBloqueado
        );
        assert_eq!(
            classificar("payload inválido: tamanho"),
            Recusa::SelecaoInvalida
        );
    }
}
