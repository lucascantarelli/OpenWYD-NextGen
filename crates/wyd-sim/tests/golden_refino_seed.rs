//! Escada de refino e `SancRate` contra os **seeds** que o Postgres aplica.
//!
//! Os dois lados são compilados para dentro do teste (`include_str!`): divergência entre a
//! fórmula e o dado quebra o gate, não a produção. Fronteira respeitada: `wyd-sim` **não**
//! depende de `wyd-db`; a leitura é do compilador, não I/O em runtime.
//!
//! Fonte da regra: `docs/wyd/sistemas/refino-auditoria.md` (público × legado × Rust).

use std::collections::BTreeMap;

use wyd_sim::{
    anct_rate_bps, blessing_total_bps, plus12_rate_bps, powder_success_bps, refine_step_kind,
    refine_success_bps, AnctRules, Powder, RefineKind, RefinePolicy,
};

const SEED_REFINE: &str = include_str!("../../wyd-db/migrations/0022_seed_refino.sql");
const SEED_SANC_FILE: &str = include_str!("../../wyd-db/migrations/0025_seed_sanc_rate.sql");
const SEED_SANC_DEFAULTS: &str =
    include_str!("../../wyd-db/migrations/0026_seed_sanc_rate_defaults.sql");

/// Uma linha de `refine_rate` como o banco a recebe.
#[derive(Debug, Clone, Copy)]
struct SeedStep {
    step: u8,
    rate_bps: u16,
    kind: RefineKind,
}

/// Lê o `INSERT INTO refine_rate` do seed até o `ON CONFLICT`.
fn seed_steps() -> Vec<SeedStep> {
    let corpo = SEED_REFINE
        .split("INSERT INTO refine_rate")
        .nth(1)
        .expect("seed deve conter o INSERT de refine_rate");
    corpo
        .lines()
        .filter_map(|linha| {
            let linha = linha.trim().strip_prefix('(')?;
            let mut campos = linha.split(',');
            let step: u8 = campos.next()?.trim().parse().ok()?;
            let rate_bps: u16 = campos.next()?.trim().parse().ok()?;
            let kind = match campos.next()?.trim().trim_matches('\'') {
                "dust" => RefineKind::Dust,
                "compose" => RefineKind::Compose,
                outro => panic!("kind inesperado no seed: {outro}"),
            };
            Some(SeedStep {
                step,
                rate_bps,
                kind,
            })
        })
        .collect()
}

/// Tabela efetiva `(pó, índice) → percentual`, como o servidor a lê: as linhas do arquivo
/// e as que o loader do legado mantém do default compilado.
fn sanc_table() -> BTreeMap<(&'static str, u8), u8> {
    let mut tabela = BTreeMap::new();
    for seed in [SEED_SANC_FILE, SEED_SANC_DEFAULTS] {
        let corpo = seed
            .split("INSERT INTO sanc_rate (powder, refine_index, percent")
            .nth(1)
            .expect("seed deve conter o INSERT de sanc_rate");
        for linha in corpo.lines() {
            let Some(linha) = linha.trim().strip_prefix('(') else {
                continue;
            };
            if linha.starts_with("1,") {
                break; // começou o INSERT de content_version
            }
            let mut campos = linha.split(',');
            let powder = match campos.next().unwrap_or_default().trim().trim_matches('\'') {
                "PO" => "PO",
                "PL" => "PL",
                "AMAGO" => "AMAGO",
                _ => continue,
            };
            let Some(index) = campos.next().and_then(|c| c.trim().parse::<u8>().ok()) else {
                continue;
            };
            let Some(percent) = campos.next().and_then(|c| c.trim().parse::<u8>().ok()) else {
                continue;
            };
            tabela.insert((powder, index), percent);
        }
    }
    tabela
}

/// A escada do seed é exatamente a da fórmula, passo a passo, com a família certa.
#[test]
fn escada_do_seed_bate_com_a_formula_nos_quinze_passos() {
    let passos = seed_steps();
    assert_eq!(passos.len(), 15, "seed publica 15 passos\n{passos:#?}");

    for passo in &passos {
        assert_eq!(
            refine_success_bps(passo.step),
            Some(passo.rate_bps),
            "passo {} diverge entre seed e fórmula",
            passo.step
        );
        assert_eq!(
            refine_step_kind(passo.step),
            Some(passo.kind),
            "família do passo {} diverge",
            passo.step
        );
    }
}

/// A tabela efetiva é a do legado: 36 células (3 pós × 12 índices), com o arquivo
/// sobrescrevendo o default e o default preenchendo o que o arquivo não lista.
#[test]
fn tabela_sanc_efetiva_tem_as_36_celulas_do_legado() {
    let tabela = sanc_table();
    assert_eq!(tabela.len(), 36, "tabela efetiva incompleta: {tabela:#?}");

    // Basedef.cpp g_pSancRate com o SancRate.txt aplicado por cima.
    let esperado: [(&str, [u8; 12]); 3] = [
        ("PO", [100, 100, 100, 85, 70, 40, 0, 0, 0, 0, 0, 0]),
        ("PL", [100, 100, 100, 100, 100, 100, 80, 70, 70, 10, 0, 0]),
        ("AMAGO", [100, 80, 60, 40, 20, 10, 10, 10, 5, 5, 5, 5]),
    ];
    for (powder, linha) in esperado {
        for (indice, percent) in linha.iter().enumerate() {
            assert_eq!(
                tabela.get(&(powder, indice as u8)),
                Some(percent),
                "{powder} índice {indice}"
            );
        }
    }
}

/// O caminho de pó consome a célula em `nível + 1`, não em `nível` — o erro de índice
/// mais fácil de cometer e o que deixaria a taxa deslocada em um passo inteiro.
#[test]
fn formula_le_a_celula_em_nivel_mais_um() {
    let tabela = sanc_table();
    let policy = RefinePolicy::LEGACY;

    for powder in [Powder::Ori, Powder::Lac] {
        for nivel in 0u8..=9 {
            let Some(percent) = tabela.get(&(powder.code(), nivel + 1)).copied() else {
                continue;
            };
            assert_eq!(
                powder_success_bps(policy, powder, nivel, 0, percent),
                Some(u16::from(percent) * 100),
                "{} em +{nivel} (índice {})",
                powder.code(),
                nivel + 1
            );
        }
    }

    // +4→+5 com Ori: 40% da célula PO 5. Se alguém indexar pelo nível, lê 70% (PO 4).
    let po5 = tabela.get(&("PO", 5)).copied().expect("PO 5 no seed");
    assert_eq!(
        powder_success_bps(policy, Powder::Ori, 4, 0, po5),
        Some(4_000)
    );
}

/// Os passos especiais não consultam a tabela: Âmago é 100% e +10→+11 é 15%.
#[test]
fn passos_especiais_ignoram_a_tabela() {
    let policy = RefinePolicy::LEGACY;
    assert_eq!(
        powder_success_bps(policy, Powder::Amago, 9, 0, 0),
        Some(10_000)
    );
    assert_eq!(
        powder_success_bps(policy, Powder::Lac, 10, 20, 100),
        Some(1_500),
        "a linha Âmago 10 (5%) não é a taxa do passo +10 → +11"
    );
}

/// As réguas de composição usadas na auditoria continuam valendo: Anct dos dois lados,
/// quatro bênçãos +9 e +11→+12 com quatro pedras secretas.
#[test]
fn reguas_de_composicao_da_auditoria_continuam_valendo() {
    assert_eq!(
        anct_rate_bps(AnctRules::LEGACY, &[7, 8, 9]),
        Some(1_700),
        "legado: 1% + 2 + 4 + 10"
    );
    assert_eq!(
        anct_rate_bps(AnctRules::PUBLISHED, &[10, 10, 10, 10]),
        Some(5_300),
        "publicado: 5% + 4 × 12%"
    );
    assert_eq!(blessing_total_bps(&[9, 9, 9, 9]), Some(4_000));
    assert_eq!(plus12_rate_bps(11, 4, &[]), Some(800));
}
