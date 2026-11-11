//! CP (Chaos Points): faixa, cor do nome no cliente e penalidade de kill.
//!
//! Fonte dos números: `docs/wyd/pesquisa-internet-2026-11-10.md` §2.4 (Kersef + WYD Int).
//! O cliente não recebe "CP": ele lê o byte `ChaosRate` = `CP + 75` em `pMob+0x11e` e
//! interpola a tabela de cores do nome (`WYD.exe` `0x4e2393`, tabela `0x6188a4`).

/// Piso do CP: também o limiar em que o PK fica bloqueado.
pub const CP_MIN: i16 = -75;
/// Teto do CP.
pub const CP_MAX: i16 = 75;
/// Deslocamento do CP para o byte `ChaosRate` do cliente (`0..=150`).
pub const CP_WIRE_OFFSET: i16 = 75;

/// Prende o CP na faixa publicada.
#[must_use]
pub fn clamp_cp(cp: i16) -> i16 {
    cp.clamp(CP_MIN, CP_MAX)
}

/// Byte `ChaosRate` do fio: é ele que define a cor do nome no cliente.
#[must_use]
pub fn chaos_rate(cp: i16) -> u8 {
    u8::try_from(clamp_cp(cp) + CP_WIRE_OFFSET).unwrap_or(0)
}

/// Penalidade de caos por matar outro jogador.
///
/// Vítima já listada (CP negativo) não dá penalidade; vítima com CP positivo rende 2 ou 3
/// pontos (3 a partir de 25). A leitura de "2 a 3 dependendo do CP da vítima" é a de
/// `docs/wyd/pesquisa-internet-2026-11-10.md` §2.4; o valor exato é decisão de balanceamento.
#[must_use]
pub fn penalidade_de_kill(cp_vitima: i16) -> i16 {
    if cp_vitima < 0 {
        0
    } else if cp_vitima >= 25 {
        3
    } else {
        2
    }
}

/// Autor no piso do CP (`-75`) não inicia PK (WYD Int, guia 46 + patch 38).
#[must_use]
pub fn pode_iniciar_pk(cp: i16) -> bool {
    cp > CP_MIN
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn cor_do_nome_usa_cp_deslocado_em_75() {
        assert_eq!(chaos_rate(0), 75, "CP 0 = 0x4B, o neutro do cliente");
        assert_eq!(chaos_rate(-75), 0, "piso");
        assert_eq!(chaos_rate(75), 150, "teto");
        assert_eq!(chaos_rate(200), 150, "acima do teto satura");
        assert_eq!(chaos_rate(-200), 0, "abaixo do piso satura");
    }

    #[test]
    fn penalidade_ignora_vitima_ja_listada_e_cresce_com_cp_positivo() {
        assert_eq!(penalidade_de_kill(-1), 0);
        assert_eq!(penalidade_de_kill(-75), 0);
        assert_eq!(penalidade_de_kill(0), 2);
        assert_eq!(penalidade_de_kill(24), 2);
        assert_eq!(penalidade_de_kill(25), 3);
        assert_eq!(penalidade_de_kill(75), 3);
    }

    #[test]
    fn pk_bloqueado_no_piso_do_cp() {
        assert!(!pode_iniciar_pk(-75));
        assert!(pode_iniciar_pk(-74));
        assert!(pode_iniciar_pk(0));
    }
}
