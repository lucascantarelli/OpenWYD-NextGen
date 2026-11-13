//! Submódulo de `bot` (ADR-0027): raça e dados de conta.

/// Uma raça jogável do 7662 (classe 0–3, ordem de `docs/wyd/classes.md` §1).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Race {
    TransKnight,
    Foema,
    BeastMaster,
    Huntress,
}

impl Race {
    /// As quatro raças, uma por bot.
    pub const TODAS: [Race; 4] = [
        Race::TransKnight,
        Race::Foema,
        Race::BeastMaster,
        Race::Huntress,
    ];

    /// Índice de classe do `_MSG_CreateCharacter`/`BaseMob`.
    #[must_use]
    pub const fn class(self) -> i32 {
        match self {
            Self::TransKnight => 0,
            Self::Foema => 1,
            Self::BeastMaster => 2,
            Self::Huntress => 3,
        }
    }

    /// Nome usado em log/CLI.
    #[must_use]
    pub const fn nome(self) -> &'static str {
        match self {
            Self::TransKnight => "TransKnight",
            Self::Foema => "Foema",
            Self::BeastMaster => "BeastMaster",
            Self::Huntress => "Huntress",
        }
    }
}

/// Dados de conexão/conta do bot (uma conta por raça).
#[derive(Debug, Clone)]
pub struct BotConfig {
    pub addr: String,
    pub account: String,
    pub password: String,
    pub pin: String,
    pub slot: i32,
    /// `MobID` dos quatro treinadores na cena (o bot não adivinha).
    pub treinadores: [i32; 4],
}

impl BotConfig {
    /// Configuração com os padrões do laboratório (conta e senha explícitas).
    #[must_use]
    pub fn new(
        addr: impl Into<String>,
        account: impl Into<String>,
        password: impl Into<String>,
    ) -> Self {
        Self {
            addr: addr.into(),
            account: account.into(),
            password: password.into(),
            pin: "2468".to_string(),
            slot: 0,
            treinadores: [0; 4],
        }
    }
}
