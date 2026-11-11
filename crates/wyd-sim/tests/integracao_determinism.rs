//! Determinismo de tick e de replay — o contrato que separa laboratório de chute.
//!
//! Três provas, nesta ordem de força:
//!
//! 1. **Mesma seed → mesma sequência** de rolls (gerador sem relógio, sem estado oculto).
//! 2. **Mesma seed → mesmo tick**: efeitos, regeneração, golpes e ameaça repetem bit a bit.
//! 3. **Rolls gravados → mesma partida**: trocar o gerador por um reprodutor de rolls devolve
//!    exatamente o mesmo resultado, provando que a decisão está nas fórmulas e não no sorteio.
//!
//! Seed diferente mudando o resultado também é verificado — senão o item 2 seria verdade
//! para uma constante, não para uma simulação.

use wyd_sim::{
    crit_chance_bps, crits, hits, mul_bps, resolve_attack, ticks_for_seconds, AttackOutcome,
    AttackProfile, AttrPoints, AttrWeight, Bps, Combatant, Die, EffectSet, ExpCurve, GearBonus,
    GrowthProfile, LevelState, RegenAccumulator, ResistProfile, SimError, ThreatTable, TimedEffect,
    XorShift64,
};

/// Cenário fixo: dois combatentes com números redondos e nenhuma dependência externa.
fn cenario() -> (Combatant, Combatant, AttackProfile) {
    let peso = AttrWeight {
        hp_per_constitution: 3,
        mp_per_intelligence: 2,
        attack_per_strength: 2,
        attack_per_dexterity: 1,
        defense_per_constitution: 1,
    };
    let crescimento = GrowthProfile {
        hp_per_level: 10,
        mp_per_level: 4,
        attack_per_level: 2,
        defense_per_level: 1,
        weight: peso,
    };

    let atacante = Combatant {
        attrs: AttrPoints {
            strength: 40,
            intelligence: 10,
            dexterity: 20,
            constitution: 30,
        },
        weight: peso,
        growth: crescimento,
        gear: GearBonus {
            attack: 120,
            ..GearBonus::default()
        },
        crit_tenths: 70,
        ..Combatant::of_level(1, 60)
    };
    let defensor = Combatant {
        attrs: AttrPoints {
            constitution: 50,
            ..AttrPoints::ZERO
        },
        weight: peso,
        growth: crescimento,
        absorb: 40,
        evasion_bps: Bps::from_percent(15),
        crit_resist_bps: Bps::from_percent(2),
        resist: ResistProfile {
            fire_bps: Bps::from_percent(20),
            frost_bps: Bps::ZERO,
        },
        ..Combatant::of_level(2, 62)
    };
    let golpe = AttackProfile {
        skill_bps: Bps::from_percent(130),
        pierce: 30,
        ..AttackProfile::new(200)
    };
    (atacante, defensor, golpe)
}

fn transcreve(seed: u64, ticks: usize) -> Vec<AttackOutcome> {
    let (atacante, defensor, golpe) = cenario();
    let mut die = XorShift64::from_seed(seed);
    (0..ticks)
        .map(|_| resolve_attack(&atacante, &defensor, &golpe, &mut die))
        .collect()
}

#[test]
fn same_seed_produces_the_same_sequence_of_rolls() {
    let mut a = XorShift64::from_seed(0xC0FFEE);
    let mut b = XorShift64::from_seed(0xC0FFEE);
    let seq_a: Vec<(u32, u32)> = (0..64).map(|_| (a.roll_bps(), a.state() as u32)).collect();
    let seq_b: Vec<(u32, u32)> = (0..64).map(|_| (b.roll_bps(), b.state() as u32)).collect();
    assert_eq!(seq_a, seq_b);
    assert_eq!(a.state(), b.state(), "estado final idêntico");
}

#[test]
fn same_seed_produces_the_same_tick_transcript() {
    assert_eq!(transcreve(42, 200), transcreve(42, 200));
}

#[test]
fn different_seed_changes_the_outcome() {
    assert_ne!(transcreve(42, 200), transcreve(43, 200));
}

/// Reprodutor de rolls gravados: a mesma lista tem de devolver a mesma partida.
struct RecordedDie {
    rolls: Vec<u32>,
    cursor: usize,
}

impl Die for RecordedDie {
    fn roll_1_to(&mut self, max: u32) -> Result<u32, SimError> {
        if max == 0 {
            return Err(SimError::RollOutOfRange { roll: 0, max });
        }
        let valor = self.rolls.get(self.cursor).copied().unwrap_or(0);
        self.cursor += 1;
        Ok((valor % max).saturating_add(1))
    }

    fn roll_bps(&mut self) -> u32 {
        let valor = self.rolls.get(self.cursor).copied().unwrap_or(0);
        self.cursor += 1;
        valor % 10_000
    }
}

/// Gravador: delega ao gerador real e anota cada roll entregue, na ordem do consumo.
struct TeeDie {
    inner: XorShift64,
    rolls: Vec<u32>,
}

impl Die for TeeDie {
    fn roll_1_to(&mut self, max: u32) -> Result<u32, SimError> {
        let valor = self.inner.roll_1_to(max)?;
        self.rolls.push(valor);
        Ok(valor)
    }

    fn roll_bps(&mut self) -> u32 {
        let valor = self.inner.roll_bps();
        self.rolls.push(valor);
        valor
    }
}

#[test]
fn recorded_rolls_replay_the_exact_same_combat() {
    let (atacante, defensor, golpe) = cenario();
    let mut gravador = TeeDie {
        inner: XorShift64::from_seed(7),
        rolls: Vec::new(),
    };
    let esperado: Vec<AttackOutcome> = (0..50)
        .map(|_| resolve_attack(&atacante, &defensor, &golpe, &mut gravador))
        .collect();

    // O gravador anota o que foi **consumido**, não o que se supôs consumir:
    // golpe perdido gasta um roll, golpe acertado gasta dois.
    let mut grants = gravador.rolls;
    let total_rolls = grants.len();
    let mut replay = RecordedDie {
        rolls: std::mem::take(&mut grants),
        cursor: 0,
    };
    let obtido: Vec<AttackOutcome> = (0..50)
        .map(|_| resolve_attack(&atacante, &defensor, &golpe, &mut replay))
        .collect();

    assert_eq!(esperado, obtido, "replay com rolls gravados");
    assert_eq!(replay.cursor, total_rolls, "consumiu exatamente o gravado");
}

#[test]
fn tick_state_is_reproducible_across_every_system() {
    let roda = || {
        let mut efeitos = EffectSet::new();
        let mut regen = RegenAccumulator::default();
        let mut ameaca = ThreatTable::new();
        let mut hp = 0u32;

        efeitos
            .apply(TimedEffect {
                affect: 64,
                magnitude: 10,
                remaining_ticks: ticks_for_seconds(64, 20).expect("cadência válida"),
            })
            .expect("cabe");

        for tick in 0..80u32 {
            let expirados = efeitos.tick();
            let ganho = wyd_sim::regen::per_tick(2_000, 20, &mut regen).expect("cadência");
            hp = wyd_sim::regen::apply_to(hp, 1_000, ganho);
            if tick % 7 == 0 {
                ameaca.add(hp % 3 + 1, hp, tick).expect("cabe");
            }
            let _ = expirados;
        }

        (
            hp,
            efeitos.len(),
            ameaca.select_target(79, 0),
            ameaca.threat_of(2),
        )
    };

    assert_eq!(roda(), roda());
    // 80 ticks a 20 Hz = 4 s a 2.000 HP/s = 8.000, travado no teto de 1.000.
    assert_eq!(roda().0, 1_000);
}

#[test]
fn formulas_are_pure_functions_of_their_inputs() {
    let (atacante, defensor, golpe) = cenario();
    let a = atacante.crit_chance(defensor);
    let b = atacante.crit_chance(defensor);
    assert_eq!(a, b);

    let chance = crit_chance_bps(
        Bps::from_percent(7),
        atacante.crit_chance_mod_bps,
        defensor.crit_resist_bps,
    );
    assert_eq!(chance, a, "mesma conta, mesma saída");

    // Nenhuma fórmula depende de chamada anterior: repetir altera nada.
    let golpe_1 = resolve_attack(&atacante, &defensor, &golpe, &mut XorShift64::from_seed(3));
    let golpe_2 = resolve_attack(&atacante, &defensor, &golpe, &mut XorShift64::from_seed(3));
    assert_eq!(golpe_1, golpe_2);
}

#[test]
fn level_up_replay_takes_the_same_path() {
    let curva = ExpCurve::new(vec![0, 1_000, 3_000, 6_000, 10_000]).expect("curva válida");
    let passo = |estado: LevelState, lote: u64| {
        let mut atual = estado;
        for _ in 0..10 {
            atual = wyd_sim::grant_exp(atual, lote, &curva);
        }
        atual
    };
    let inicial = LevelState {
        level: 1,
        exp_into_level: 0,
    };
    assert_eq!(passo(inicial, 500), passo(inicial, 500));
    assert_eq!(passo(inicial, 500).level, 3);
}

#[test]
fn hit_and_crit_rolls_are_half_open() {
    assert!(hits(9_999, Bps::FULL));
    assert!(!hits(0, Bps::ZERO));
    assert!(crits(0, Bps::new(1)));
    assert!(!crits(1, Bps::new(1)));
    assert_eq!(mul_bps(100, 9_000), 90);
}
