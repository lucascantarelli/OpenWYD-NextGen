//! Unidades inteiras do laboratório — newtype para não confundir bps com inteiro cru.
//!
//! Regra de arredondamento do crate (uma só, documentada em
//! `docs/arquitetura/matematica-sim.md` §3): **truncamento (piso)** na divisão inteira,
//! como no legado C++. Não existe arredondamento "para o mais próximo" escondido em
//! fórmula — quem quiser outra política constrói o número antes de entrar aqui.
//!
//! Nada de `f32`/`f64`: percentual é [`Bps`] e grandeza decimal de item é [`Tenths`].

/// Basis point: `10_000` = 100%, `1` = 0,01%.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Bps(u32);

impl Bps {
    /// Nenhum por cento (`0`).
    pub const ZERO: Self = Self(0);
    /// Todos os por cento (`10_000`) — identidade multiplicativa.
    pub const FULL: Self = Self(10_000);

    /// Basis points crus (`9000` = 90%).
    #[must_use]
    pub const fn new(raw: u32) -> Self {
        Self(raw)
    }

    /// Valor cru em basis points.
    #[must_use]
    pub const fn raw(self) -> u32 {
        self.0
    }

    /// Converte pontos percentuais (`40` = 40%) em basis points.
    #[must_use]
    pub const fn from_percent(points: u32) -> Self {
        Self(points.saturating_mul(100))
    }

    /// Aplica o percentual a um valor inteiro, truncando (piso).
    #[must_use]
    pub const fn apply(self, value: u32) -> u32 {
        mul_bps(value, self.0)
    }

    /// Complemento em relação a 100% (`1_000` → `9_000`); satura em zero.
    #[must_use]
    pub const fn complement(self) -> Self {
        Self(10_000u32.saturating_sub(self.0))
    }

    /// Limita o percentual a uma faixa fechada — usado onde "mais de 100%" não tem sentido.
    #[must_use]
    pub fn clamp(self, min: Self, max: Self) -> Self {
        Self(self.0.clamp(min.0, max.0))
    }
}

/// Multiplica por basis points com truncamento e saturação em [`u32::MAX`].
///
/// A conta é feita em `u64` para não estourar `u32` no produto intermediário; o resultado
/// nunca passa de `u32::MAX` (saturação explícita, não *wrap*).
#[must_use]
pub const fn mul_bps(value: u32, bps: u32) -> u32 {
    let produto = (value as u64) * (bps as u64);
    let saida = produto / 10_000;
    if saida > u32::MAX as u64 {
        u32::MAX
    } else {
        saida as u32
    }
}

/// Décimos de ponto percentual — a unidade em que o cliente exibe grandeza decimal
/// (`7.0` de crítico = `70` décimos).
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Tenths(u32);

impl Tenths {
    /// Décimos crus (`70` = 7,0).
    #[must_use]
    pub const fn new(raw: u32) -> Self {
        Self(raw)
    }

    /// Valor cru em décimos.
    #[must_use]
    pub const fn raw(self) -> u32 {
        self.0
    }

    /// Converte em basis points dado quantos bps vale um décimo (`10` → `70` = `700` bps).
    #[must_use]
    pub const fn to_bps(self, bps_per_tenth: u32) -> Bps {
        Bps(self.0.saturating_mul(bps_per_tenth))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn full_is_identity_and_zero_annihilates() {
        assert_eq!(Bps::FULL.apply(1_234), 1_234);
        assert_eq!(Bps::ZERO.apply(1_234), 0);
    }

    #[test]
    fn truncation_is_floor_not_round() {
        // 1% de 99 = 0,99 → 0 (piso). Arredondar "para o mais próximo" daria 1.
        assert_eq!(mul_bps(99, 100), 0);
        // 90% de 15 = 13,5 → 13.
        assert_eq!(mul_bps(15, 9_000), 13);
    }

    #[test]
    fn saturates_instead_of_wrapping() {
        assert_eq!(mul_bps(u32::MAX, 20_000), u32::MAX);
    }

    #[test]
    fn percent_and_tenths_convert() {
        assert_eq!(Bps::from_percent(40).raw(), 4_000);
        assert_eq!(Tenths::new(70).to_bps(10).raw(), 700);
    }

    #[test]
    fn complement_saturates_at_zero() {
        assert_eq!(Bps::new(2_000).complement().raw(), 8_000);
        assert_eq!(Bps::new(20_000).complement().raw(), 0);
    }
}
