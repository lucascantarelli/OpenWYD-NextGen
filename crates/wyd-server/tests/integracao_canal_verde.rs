//! Cidadania e canal verde (FASE-03-10): o papel do processo e o bônus de drop do canal.
//!
//! Três provas, todas determinísticas e sem banco:
//!
//! 1. **política do papel** — canal verde nunca liga PvP e mantém loja pessoal aberta; canal
//!    normal segue o mapa. É a decisão que o handler de PvP (checklist 4.5) e o de loja (7.2)
//!    consultam; aqui ela é provada como regra, sem inventar pacote que o cliente não manda;
//! 2. **bônus de cidadania no drop** — `citizen_bonus_bps` do `game_setting` entra como
//!    parâmetro: com a **mesma seed** e o mesmo roll, o cidadão usa a taxa corrigida e o
//!    visitante continua na taxa do legado. É o teste do risco declarado na issue (\"bônus
//!    global\" por derivação);
//! 3. **config inválida não vira taxa** — bônus negativo ou acima do teto recusa o snapshot
//!    (o processo continua no último bom), em vez de aplicar número inventado.

use std::sync::Arc;

use wyd_db::ChannelRole;
use wyd_db::{ConfigRateRow, DropConfigRow};
use wyd_server::channels::{effective_role, role_overrides_from_env, ChannelRules, ROLE_ENV};
use wyd_server::config::{GameConfig, LiveConfig};
use wyd_server::gameplay::Gameplay;
use wyd_sim::formula::drop::{CITIZEN_BONUS_MAX, DROP_ROLL_MAX};
use wyd_sim::roll::XorShift64;

fn rate(domain: &str, key: &str, value: i64) -> ConfigRateRow {
    ConfigRateRow {
        domain: domain.to_string(),
        key: key.to_string(),
        value,
        unit: "bps".to_string(),
        source: "seed".to_string(),
        overridden: false,
    }
}

fn drop_row(weight: i64) -> DropConfigRow {
    DropConfigRow {
        template_name: "wolf".to_string(),
        item_index: 413,
        item_name: "Poeira de Lactolerium".to_string(),
        weight,
        min_qty: 1,
        max_qty: 1,
        source: "seed".to_string(),
    }
}

/// Configuração mínima com 15 passos de refino válidos (o snapshot exige a tabela inteira) e o
/// bônus de cidadania como o banco o entrega: linha `setting` `Channel Settings.citizen_bonus_bps`.
fn config(citizen_bonus_bps: i64, weight: i64) -> Result<GameConfig, String> {
    let mut rates: Vec<ConfigRateRow> = (1..=15)
        .map(|step| {
            rate(
                "refine",
                &step.to_string(),
                i64::from(wyd_sim::formula::refine::REFINE_RATE_BPS[step - 1]),
            )
        })
        .collect();
    for index in 0..=15 {
        rates.push(rate("sanc", &format!("PO:{index}"), 10));
        rates.push(rate("sanc", &format!("PL:{index}"), 5));
        rates.push(rate("sanc", &format!("AMAGO:{index}"), 100));
    }
    rates.push(rate("combine", "Odin.PLUS12_BASE", 100));
    rates.push(rate("multiplier", "drop", 10_000));
    rates.push(rate(
        "setting",
        "Channel Settings.citizen_bonus_bps",
        citizen_bonus_bps,
    ));
    GameConfig::from_rows(
        1,
        &rates,
        Vec::new(),
        vec![drop_row(weight)],
        Vec::new(),
        Vec::new(),
    )
    .map_err(|error| error.to_string())
}

fn gameplay(citizen_bonus_bps: i64, weight: i64) -> Arc<Gameplay> {
    let foto = config(citizen_bonus_bps, weight).expect("configuração válida");
    Arc::new(Gameplay::new(Arc::new(LiveConfig::new(foto))))
}

/// Canal verde: sem PvP em qualquer ponto, loja pessoal aberta — o inverso da regra completa
/// só no eixo do PvP, porque loja é serviço comum nos dois papéis (`docs/wyd/canais-e-cidadania.md` §3).
#[test]
fn canal_verde_recusa_pvp_e_mantem_loja() {
    let verde = ChannelRules::new(ChannelRole::Verde);
    assert!(
        !verde.pvp_allowed(true),
        "mapa de PvP não liga PvP no canal verde"
    );
    assert!(!verde.pvp_allowed(false));
    assert!(verde.personal_shop_allowed(), "loja pessoal permitida");

    let normal = ChannelRules::new(ChannelRole::Normal);
    assert!(normal.pvp_allowed(true), "canal normal segue o mapa");
    assert!(
        !normal.pvp_allowed(false),
        "mapa sem PvP continua sem no canal normal"
    );
    assert!(normal.personal_shop_allowed());
}

/// Papel vem do dado (ou do env para teste), nunca do build; valor desconhecido não derruba.
#[test]
fn papel_do_canal_e_configuravel_e_invalido_cai_no_default() {
    assert_eq!(
        effective_role(Some(ChannelRole::Verde), Some(ChannelRole::Normal)),
        ChannelRole::Normal,
        "env sobrescreve o banco"
    );
    assert_eq!(
        effective_role(Some(ChannelRole::Verde), None),
        ChannelRole::Verde,
        "sem env vale o banco"
    );
    assert_eq!(
        effective_role(None, None),
        ChannelRole::Normal,
        "canal sem linha nem env segue normal"
    );

    std::env::set_var(ROLE_ENV, "verde");
    assert_eq!(role_overrides_from_env(), Some(ChannelRole::Verde));
    std::env::set_var(ROLE_ENV, "canal-de-novato");
    assert_eq!(
        role_overrides_from_env(),
        None,
        "papel desconhecido é ignorado, não é recusa"
    );
    std::env::remove_var(ROLE_ENV);
}

/// Mesma seed, mesmo roll: o bônus do canal de cidadania muda a decisão, e só para o cidadão.
///
/// Peso 500.000 (50%) com bônus de 50% vira 750.000: o roll 600.000 não cai sem cidadania e
/// cai com ela. É a prova de que o bônus é **entrada** aplicada sobre a taxa, não constante
/// escondida no cálculo.
#[test]
fn bonus_de_cidadania_muda_o_drop_na_mesma_seed() {
    let cidadao = gameplay(CITIZEN_BONUS_MAX.raw() as i64, 500_000);
    let visitante = gameplay(0, 500_000);

    let com_bonus = cidadao
        .loot("wolf", &[(413, 600_000)], true)
        .expect("tabela do snapshot");
    assert_eq!(com_bonus.len(), 1, "60% de 750.000 cai");

    let sem_bonus = cidadao
        .loot("wolf", &[(413, 600_000)], false)
        .expect("tabela do snapshot");
    assert!(
        sem_bonus.is_empty(),
        "visitante não recebe o bônus do canal"
    );
    assert!(
        visitante
            .loot("wolf", &[(413, 600_000)], true)
            .expect("tabela")
            .is_empty(),
        "canal com bônus 0 mantém a taxa do legado"
    );

    // Sorteio do abate: mesma seed, resultado diferente só pelo bônus do canal.
    let mut die_cidadao = XorShift64::from_seed(2_026);
    let mut die_visitante = XorShift64::from_seed(2_026);
    let morte_cidadao = cidadao
        .loot_at_death("wolf", &mut die_cidadao, true)
        .expect("tabela");
    let morte_visitante = cidadao
        .loot_at_death("wolf", &mut die_visitante, false)
        .expect("tabela");
    assert_eq!(
        morte_cidadao.rolls, morte_visitante.rolls,
        "a seed produz o mesmo roll: só a taxa mudou"
    );
    let (_, roll) = morte_cidadao.rolls[0];
    assert!(
        (1..=DROP_ROLL_MAX).contains(&roll),
        "roll na escala do original: {roll}"
    );
    assert_eq!(
        !morte_cidadao.drops.is_empty(),
        roll <= 750_000,
        "cidadão usa 500.000 + 50% = 750.000"
    );
    assert_eq!(
        !morte_visitante.drops.is_empty(),
        roll <= 500_000,
        "visitante usa o peso cru do legado"
    );
}

/// Bônus fora da faixa não vira taxa: o snapshot é recusado (o processo fica no último bom).
#[test]
fn bonus_fora_da_faixa_recusa_o_snapshot() {
    assert!(config(-1, 500_000).is_err(), "bônus negativo não existe");
    assert!(config(5_001, 500_000).is_err(), "acima do teto de 50%");
    assert!(config(5_000, 500_000).is_ok(), "teto é aceito");
    assert!(config(0, 500_000).is_ok(), "default do seed é aceito");
}
