//! Bots do Campo de Treinamento (GAME-23): quatro raças, um bot por classe,
//! falando o protocolo real do canal (ADR-0027). A implementação vive em `bot/`.

mod bot;
pub mod tela;

pub use bot::{motivo_da_recusa, nome_do_campo, parse_etapa, Bot, BotConfig, Race, SceneMob};

#[cfg(test)]
mod tests {
    use super::*;
    use wyd_protocol::wire::opcode::Opcode;

    #[test]
    fn quatro_racas_com_classes_distintas() {
        let classes: Vec<i32> = Race::TODAS.iter().map(|raca| raca.class()).collect();
        assert_eq!(classes, vec![0, 1, 2, 3]);
        let nomes: Vec<&str> = Race::TODAS.iter().map(|raca| raca.nome()).collect();
        assert_eq!(
            nomes,
            vec!["TransKnight", "Foema", "BeastMaster", "Huntress"]
        );
    }

    #[test]
    fn parse_etapa_le_o_painel_do_servidor() {
        assert_eq!(parse_etapa("Campo de Treinamento: etapa 1/4"), Some(1));
        assert_eq!(
            parse_etapa("Campo de Treinamento: etapa 4/4 (concluido)"),
            Some(4)
        );
        assert_eq!(parse_etapa("falta a prova 451"), None);
    }

    #[test]
    fn toda_recusa_tem_mensagem_em_portugues() {
        for opcode in [
            Opcode::NewAccountFail,
            Opcode::AccountSecureFail,
            Opcode::CharacterLoginFail,
            Opcode::AccountLogin,
        ] {
            let texto = motivo_da_recusa(opcode);
            assert!(!texto.trim().is_empty(), "{opcode:?} sem mensagem");
            let minusculo = texto.to_lowercase();
            assert!(
                minusculo.contains("tente")
                    || minusculo.contains("escolha")
                    || minusculo.contains("confira"),
                "mensagem sem próximo passo: {texto}"
            );
        }
        assert!(motivo_da_recusa(Opcode::AccountSecureFail).contains("PIN"));
        assert!(motivo_da_recusa(Opcode::CharacterLoginFail).contains("Personagem"));
        assert!(motivo_da_recusa(Opcode::NewAccountFail).contains("senha"));
    }

    #[test]
    fn nome_do_create_mob_perde_o_preenchimento_nulo() {
        let mut nome = [0u8; 16];
        nome[..14].copy_from_slice(b"PrimeiroMestre");
        assert_eq!(nome_do_campo(&nome), "PrimeiroMestre");
        assert_eq!(nome_do_campo(&[b'A'; 16]), "AAAAAAAAAAAAAAAA");
    }

    #[test]
    fn campanha_usa_os_quatro_treinadores_na_ordem() {
        let config = BotConfig::new("127.0.0.1:8281", "qa", "qa12345678");
        assert_eq!(config.treinadores, [0; 4]);
        // A ordem é a dos `Merchant` 36/40/41/100 (sequência A).
        assert_eq!(config.slot, 0);
        assert_eq!(config.pin, "2468");
    }
}
