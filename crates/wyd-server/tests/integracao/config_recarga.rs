//! Recarga de configuração viva (WEB-06): portal → `config_revision` → snapshot.
//!
//! Sem banco externo: `MemDb` faz o papel das tabelas, **com as mesmas validações** de
//! `config_set_rate`, e incrementa a revisão na mesma "transação" (escrita + bump). O teste
//! prova o contrato que o processo de jogo precisa cumprir:
//!
//! 1. revisão nova vira snapshot novo e o valor editado aparece no resolvedor;
//! 2. leitor que já tinha a foto antiga continua vendo a antiga (troca atômica);
//! 3. reset volta ao seed histórico;
//! 4. escrita inválida é recusada no portal **e**, se chegar ao banco, a recarga recusa a
//!    foto inteira e mantém a última boa;
//! 5. falha de leitura mantém a última boa;
//! 6. a thread `wyd-config` aplica sozinha, sem tocar no tick.

#[path = "../common/mod.rs"]
mod common;

use std::collections::BTreeMap;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex};
use std::time::Duration;

use wyd_db::{AccessoryRuleRow, ConfigRateRow, DropConfigRow, NpcConfigRow, RouteConfigRow};
use wyd_server::config::{
    spawn_reloader, ConfigSource, GameConfig, LiveConfig, ReloadOutcome, DEFAULT_MULTIPLIER_BPS,
};
use wyd_server::personagens::Mob;
use wyd_sim::formula::refine::{Powder, REFINE_RATE_BPS};

/// `STRUCT_MOB` de 816 B para o NPC de teste: EXP, nível e atributos como o `npc_template`
/// entrega ao servidor.
fn npc_template(exp: i64, max_hp: i32) -> Vec<u8> {
    let mut mob = Mob {
        exp,
        ..Mob::default()
    };
    mob.current_score.level = 50;
    mob.current_score.max_hp = max_hp;
    mob.current_score.hp = max_hp;
    mob.current_score.damage = 120;
    mob.current_score.ac = 20;
    mob.name[..8].copy_from_slice(b"Orc\0\0\0\0\0");
    mob.encode()
}

/// Banco de mentira com o estado efetivo que `config_rates()` devolveria.
struct MemDb {
    revision: Mutex<i64>,
    rates: Mutex<BTreeMap<(String, String), i64>>,
    drops: Mutex<BTreeMap<(String, i32), DropConfigRow>>,
    npcs: Mutex<Vec<NpcConfigRow>>,
    routes: Mutex<Vec<RouteConfigRow>>,
    accessories: Mutex<Vec<AccessoryRuleRow>>,
    fail: AtomicBool,
}

impl MemDb {
    fn seeded() -> Self {
        let mut rates = BTreeMap::new();
        let mut seed = |domain: &str, key: &str, value: i64| {
            rates.insert((domain.to_string(), key.to_string()), value);
        };
        for (indice, taxa) in REFINE_RATE_BPS.iter().enumerate() {
            seed("refine", &(indice + 1).to_string(), i64::from(*taxa));
        }
        // PO 10%, PL 5%, AMAGO 100% (a tabela do legado).
        seed("sanc", "PO:5", 10);
        seed("sanc", "PL:5", 5);
        seed("sanc", "AMAGO:10", 100);
        seed("combine", "Odin.PLUS12_BASE", 100);
        seed("combine", "Aylin.BASE", 1);
        seed("setting", "event.exp_bonus", 0);

        Self {
            revision: Mutex::new(3),
            rates: Mutex::new(rates),
            drops: Mutex::new(BTreeMap::new()),
            npcs: Mutex::new(vec![NpcConfigRow {
                id: 7,
                slug: "aylin".to_string(),
                display_name: "Aylin".to_string(),
                template_name: "Orc_Cavaleiro".to_string(),
                template: npc_template(5_000, 900),
                map_id: 1,
                pos_x: 40,
                pos_y: 40,
                enabled: true,
                merchant: 0,
            }]),
            routes: Mutex::new(vec![RouteConfigRow {
                id: 1,
                from_map_id: 1,
                from_x: 40,
                from_y: 40,
                to_map_id: 2,
                to_x: 10,
                to_y: 10,
                npc_id: Some(7),
                cost_gold: 1_000,
                req_level: 5,
                enabled: true,
            }]),
            accessories: Mutex::new(vec![AccessoryRuleRow {
                kind: "necklace".to_string(),
                recipe: "Amunra Crystal".to_string(),
                step_from: 0,
                step_to: 1,
                rate_bps: 4_000,
                fail_mode: "keep".to_string(),
                fail_bonus_bps: 0,
                gold_cost: 50_000,
                material_name: "Amunra Crystal".to_string(),
                material_qty: 1,
                source: "captura".to_string(),
                notes: String::new(),
            }]),
            fail: AtomicBool::new(false),
        }
    }

    fn revision(&self) -> i64 {
        *self.revision.lock().expect("revisão")
    }

    fn bump(&self) {
        *self.revision.lock().expect("revisão") += 1;
    }

    /// Caminho do portal: valida antes de gravar e sobe a revisão na mesma transação.
    fn set_rate(&self, domain: &str, key: &str, value: i64) -> Result<(), String> {
        match domain {
            "refine" => {
                let step: u8 = key.parse().map_err(|_| "passo inválido".to_string())?;
                if !(1..=15).contains(&step) || !(0..=10_000).contains(&value) {
                    return Err("taxa de refino fora de 1..=15 passos / 0..=10000".to_string());
                }
            }
            "sanc" => {
                let (powder, index) = key.split_once(':').ok_or("chave de sanc inválida")?;
                if !matches!(powder, "PO" | "PL" | "AMAGO")
                    || index.parse::<u8>().is_err()
                    || !(0..=100).contains(&value)
                {
                    return Err("sanc inválido".to_string());
                }
            }
            "combine" => {
                if !key.contains('.') || value < 0 {
                    return Err("combine inválido".to_string());
                }
            }
            "multiplier" => {
                if !matches!(key, "exp" | "drop" | "gold" | "skill_point")
                    || !(0..=1_000_000).contains(&value)
                {
                    return Err("multiplicador inválido".to_string());
                }
            }
            "setting" => {
                if !key.contains('.') {
                    return Err("chave de setting inválida".to_string());
                }
            }
            outro => return Err(format!("domínio desconhecido: {outro}")),
        }
        self.rates
            .lock()
            .expect("taxas")
            .insert((domain.to_string(), key.to_string()), value);
        self.bump();
        Ok(())
    }

    /// Grava sem validar, como um INSERT direto: prova a defesa do lado do servidor.
    fn set_rate_unchecked(&self, domain: &str, key: &str, value: i64) {
        self.rates
            .lock()
            .expect("taxas")
            .insert((domain.to_string(), key.to_string()), value);
        self.bump();
    }

    /// Reset volta ao seed histórico (o `DELETE` do override + `COALESCE` do seed).
    fn reset_rate(&self, domain: &str, key: &str, seed: i64) {
        self.rates
            .lock()
            .expect("taxas")
            .insert((domain.to_string(), key.to_string()), seed);
        self.bump();
    }

    fn remove_rate(&self, domain: &str, key: &str) {
        self.rates
            .lock()
            .expect("taxas")
            .remove(&(domain.to_string(), key.to_string()));
        self.bump();
    }

    fn set_drop(&self, template: &str, item: i32, weight: i64) {
        self.drops.lock().expect("drops").insert(
            (template.to_string(), item),
            DropConfigRow {
                template_name: template.to_string(),
                item_index: item,
                item_name: "Espada".to_string(),
                weight,
                min_qty: 1,
                max_qty: 2,
                source: "portal".to_string(),
            },
        );
        self.bump();
    }

    /// Edição de NPC pelo portal (posição/visibilidade) + template de conteúdo novo.
    fn set_npc(&self, id: i32, map_id: i16, x: i16, y: i16, enabled: bool, template: Vec<u8>) {
        {
            let mut npcs = self.npcs.lock().expect("npcs");
            if let Some(npc) = npcs.iter_mut().find(|npc| npc.id == id) {
                npc.map_id = map_id;
                npc.pos_x = x;
                npc.pos_y = y;
                npc.enabled = enabled;
                npc.template = template;
            }
        }
        self.bump();
    }

    fn reset_drop(&self, template: &str, item: i32) {
        self.drops
            .lock()
            .expect("drops")
            .remove(&(template.to_string(), item));
        self.bump();
    }

    fn fail_reads(&self, fail: bool) {
        self.fail.store(fail, Ordering::Relaxed);
    }

    fn rate_rows(&self) -> Vec<ConfigRateRow> {
        self.rates
            .lock()
            .expect("taxas")
            .iter()
            .map(|((domain, key), value)| ConfigRateRow {
                domain: domain.clone(),
                key: key.clone(),
                value: *value,
                unit: if domain == "sanc" { "%" } else { "bps" }.to_string(),
                source: "seed".to_string(),
                overridden: false,
            })
            .collect()
    }
}

impl ConfigSource for MemDb {
    fn revision(&self) -> Result<i64, String> {
        if self.fail.load(Ordering::Relaxed) {
            return Err("pool fechado (simulado)".to_string());
        }
        Ok(self.revision())
    }

    fn load(&self, revision: i64) -> Result<GameConfig, String> {
        if self.fail.load(Ordering::Relaxed) {
            return Err("timeout de leitura (simulado)".to_string());
        }
        GameConfig::from_rows(
            revision,
            &self.rate_rows(),
            self.accessories.lock().expect("acessórios").clone(),
            self.drops
                .lock()
                .expect("drops")
                .values()
                .cloned()
                .collect(),
            self.npcs.lock().expect("npcs").clone(),
            self.routes.lock().expect("rotas").clone(),
        )
        .map_err(|error| error.to_string())
    }
}

fn boot(source: &MemDb) -> LiveConfig {
    let revision = ConfigSource::revision(source).expect("revisão");
    LiveConfig::new(source.load(revision).expect("snapshot inicial"))
}

#[test]
fn portal_edits_chegam_ao_resolvedor_e_reset_volta_ao_seed() {
    let source = MemDb::seeded();
    let live = boot(&source);

    // Foto inicial: seed histórico, sem override.
    let inicial = live.snapshot();
    assert_eq!(inicial.revision(), 3);
    assert_eq!(inicial.refine_bps(12), Some(400));
    assert_eq!(inicial.sanc_percent(Powder::Ori, 5), Some(10));
    assert_eq!(inicial.combine_value("Odin", "PLUS12_BASE"), Some(100));
    assert_eq!(inicial.multiplier_bps("exp"), DEFAULT_MULTIPLIER_BPS);
    assert_eq!(inicial.apply_multiplier("exp", 1_000), 1_000);
    assert_eq!(inicial.drop_chance_bps("wolf", 413), None);
    assert_eq!(inicial.npc(7).map(|npc| npc.slug.as_str()), Some("aylin"));
    assert_eq!(inicial.route(1).map(|route| route.to_map_id), Some(2));
    assert_eq!(
        inicial.accessory_rate_bps("necklace", "Amunra Crystal", 0),
        Some(4_000)
    );
    assert_eq!(inicial.refine_steps(), 15);

    // Todas as famílias editáveis pelo portal, na ordem da tela.
    source.set_rate("refine", "12", 9_555).expect("refino");
    source.set_rate("sanc", "PO:5", 77).expect("sanc");
    source
        .set_rate("combine", "Odin.PLUS12_BASE", 8_888)
        .expect("combine");
    source
        .set_rate("multiplier", "exp", 25_000)
        .expect("multiplicador");
    source
        .set_rate("setting", "event.exp_bonus", 7)
        .expect("setting");
    source.set_drop("wolf", 413, 500_000);
    let nova_revisao = source.revision();
    assert_eq!(nova_revisao, 9, "cada escrita sobe a revisão");

    let resultado = live.refresh(&source);
    assert_eq!(
        resultado,
        ReloadOutcome::Applied {
            from: 3,
            to: nova_revisao
        }
    );

    let novo = live.snapshot();
    assert_eq!(novo.revision(), nova_revisao);
    assert_eq!(novo.refine_bps(12), Some(9_555));
    assert_eq!(novo.sanc_percent(Powder::Ori, 5), Some(77));
    assert_eq!(novo.combine_value("Odin", "PLUS12_BASE"), Some(8_888));
    assert_eq!(novo.multiplier_bps("exp"), 25_000);
    assert_eq!(novo.apply_multiplier("exp", 1_000), 2_500);
    assert_eq!(novo.setting_int("event", "exp_bonus"), Some(7));
    assert_eq!(novo.drop_chance_bps("wolf", 413), Some(5_000));
    assert_eq!(novo.npc_count(), 1);
    assert_eq!(novo.route_count(), 1);
    assert_eq!(novo.accessory_count(), 1);

    // Formula do refino com pó vem da tabela nova (política do `wyd-sim`, não cópia local).
    assert!(novo.powder_rate_bps(Powder::Ori, 4, 0).is_some());

    // A foto antiga continua íntegra: quem já tinha o `Arc` não vê meio snapshot.
    assert_eq!(inicial.refine_bps(12), Some(400));
    assert_eq!(inicial.sanc_percent(Powder::Ori, 5), Some(10));
    assert_eq!(live.reloads(), 1);
    assert_eq!(live.failures(), 0);

    // Prova observável: o scrape mostra a revisão publicada e o que está carregado.
    let metricas = live.render_metrics();
    assert!(metricas.contains(&format!("wyd_config_revision {nova_revisao}")));
    assert!(metricas.contains("wyd_config_reloads_total 1"));
    assert!(metricas.contains("wyd_config_reload_failures_total 0"));
    assert!(metricas.contains("wyd_config_refine_steps 15"));
    assert!(metricas.contains("wyd_config_npcs 1"));

    // Sem mudança de revisão não há recarga.
    assert_eq!(live.refresh(&source), ReloadOutcome::Unchanged);

    // Reset: refino, SancRate e composição voltam ao seed; multiplicador volta ao neutro
    // e o drop sai da tabela (sem override = sem linha).
    source.reset_rate("refine", "12", i64::from(REFINE_RATE_BPS[11]));
    source.reset_rate("sanc", "PO:5", 10);
    source.reset_rate("combine", "Odin.PLUS12_BASE", 100);
    source.remove_rate("multiplier", "exp");
    source.reset_drop("wolf", 413);
    let reset = source.revision();
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied {
            from: nova_revisao,
            to: reset
        }
    );
    let depois = live.snapshot();
    assert_eq!(depois.refine_bps(12), Some(400));
    assert_eq!(depois.multiplier_bps("exp"), DEFAULT_MULTIPLIER_BPS);
    assert_eq!(depois.drop_chance_bps("wolf", 413), None);
    assert_eq!(depois.sanc_percent(Powder::Ori, 5), Some(10));
    assert_eq!(depois.combine_value("Odin", "PLUS12_BASE"), Some(100));
    // O que não foi editado permanece no seed.
    assert_eq!(depois.combine_value("Aylin", "BASE"), Some(1));
}

#[test]
fn escrita_invalida_e_falha_de_leitura_mantem_o_ultimo_snapshot_bom() {
    let source = MemDb::seeded();
    let live = boot(&source);
    source
        .set_rate("refine", "12", 9_555)
        .expect("edição válida");
    assert!(matches!(
        live.refresh(&source),
        ReloadOutcome::Applied { .. }
    ));
    let bom = live.snapshot();
    assert_eq!(bom.refine_bps(12), Some(9_555));

    // 1. O portal recusa antes de gravar (mesma faixa da constraint).
    assert!(source.set_rate("refine", "12", 20_000).is_err());
    assert!(source.set_rate("sanc", "PO:5", 101).is_err());
    assert!(source.set_rate("multiplier", "exp", 1_000_001).is_err());
    assert_eq!(source.revision(), 4, "escrita recusada não sobe a revisão");
    assert_eq!(live.refresh(&source), ReloadOutcome::Unchanged);

    // 2. Mesmo que um INSERT direto passe, o servidor recusa a foto inteira.
    source.set_rate_unchecked("refine", "12", 20_000);
    let falha = live.refresh(&source);
    assert!(
        matches!(falha, ReloadOutcome::Failed { .. }),
        "revisão inválida precisa ser recusada: {falha:?}"
    );
    assert_eq!(live.snapshot().refine_bps(12), Some(9_555));
    assert_eq!(live.snapshot().revision(), 4);
    assert_eq!(live.failures(), 1);
    assert!(live.last_error().is_some());

    // 3. Snapshot incompleto (sem um passo de refino) também não é completado com zero.
    source.reset_rate("refine", "12", 400);
    source.remove_rate("refine", "7");
    let incompleto = live.refresh(&source);
    assert!(matches!(incompleto, ReloadOutcome::Failed { .. }));
    assert_eq!(live.snapshot().refine_steps(), 15);
    assert_eq!(
        live.snapshot().refine_bps(7),
        REFINE_RATE_BPS.get(6).copied()
    );
    assert_eq!(live.failures(), 2);

    // 4. Banco indisponível: o último snapshot bom continua servindo.
    source.fail_reads(true);
    let banco_fora = live.refresh(&source);
    assert!(matches!(banco_fora, ReloadOutcome::Failed { .. }));
    assert_eq!(live.snapshot().revision(), 4);
    assert_eq!(live.snapshot().refine_bps(12), Some(9_555));
    assert_eq!(live.failures(), 3);
    source.fail_reads(false); // Recuperado (com o passo que faltava de volta), a próxima revisão válida publica.
    source.reset_rate("refine", "7", i64::from(REFINE_RATE_BPS[6]));
    source
        .set_rate("refine", "12", 5_000)
        .expect("edição válida");
    assert!(matches!(
        live.refresh(&source),
        ReloadOutcome::Applied { .. }
    ));
    assert_eq!(live.snapshot().refine_bps(12), Some(5_000));
    assert_eq!(live.reloads(), 2);
}

#[test]
fn thread_de_recarga_aplica_sem_acao_do_tick() {
    let source = Arc::new(MemDb::seeded());
    let live = Arc::new(boot(&source));
    let stop = Arc::new(AtomicBool::new(false));
    let thread = spawn_reloader(
        Arc::clone(&live),
        Arc::clone(&source) as Arc<dyn ConfigSource>,
        Duration::from_millis(10),
        Arc::clone(&stop),
    )
    .expect("thread de config");

    source.set_rate("sanc", "PL:5", 88).expect("edição");
    let alvo = source.revision();
    assert!(
        common::esperar_ate(Duration::from_secs(5), || live.revision() >= alvo),
        "recarga não publicou a revisão {alvo}"
    );
    assert_eq!(live.snapshot().sanc_percent(Powder::Lac, 5), Some(88));
    assert_eq!(live.revision(), alvo);
    assert!(live.reloads() >= 1);

    stop.store(true, Ordering::Relaxed);
    assert!(
        common::esperar_ate(Duration::from_secs(5), || thread.is_finished()),
        "thread de recarga não parou"
    );
    thread.join().expect("thread de config");
}

#[test]
fn snapshot_sem_refino_ou_com_revisao_zero_e_recusado() {
    let source = MemDb::seeded();
    let completo = source.rate_rows();
    let acessorios = source.accessories.lock().expect("acessórios").clone();

    assert!(GameConfig::from_rows(
        0,
        &completo,
        acessorios.clone(),
        Vec::new(),
        Vec::new(),
        Vec::new()
    )
    .is_err());

    let sem_refino: Vec<_> = completo
        .iter()
        .filter(|row| row.domain != "refine")
        .cloned()
        .collect();
    assert!(matches!(
        GameConfig::from_rows(
            3,
            &sem_refino,
            acessorios.clone(),
            Vec::new(),
            Vec::new(),
            Vec::new()
        ),
        Err(wyd_server::config::ConfigError::Missing { .. })
    ));
    let sem_sanc: Vec<_> = completo
        .iter()
        .filter(|row| row.domain != "sanc")
        .cloned()
        .collect();
    assert!(matches!(
        GameConfig::from_rows(
            3,
            &sem_sanc,
            acessorios.clone(),
            Vec::new(),
            Vec::new(),
            Vec::new()
        ),
        Err(wyd_server::config::ConfigError::Empty("sanc"))
    ));
    assert!(
        GameConfig::from_rows(3, &completo, acessorios, Vec::new(), Vec::new(), Vec::new()).is_ok()
    );
}

#[test]
fn npc_do_snapshot_traz_template_e_a_recarga_move_o_mundo() {
    let source = MemDb::seeded();
    let live = boot(&source);
    let inicial = live.snapshot();

    // O snapshot carrega o `STRUCT_MOB` do `npc_template`: EXP, nível e atributos reais.
    let npc = inicial.npc(7).expect("NPC do snapshot");
    assert_eq!(npc.template_name, "Orc_Cavaleiro");
    assert_eq!(npc.exp, 5_000);
    assert_eq!(npc.level, 50);
    assert_eq!((npc.max_hp, npc.attack, npc.defense), (900, 120, 20));
    assert!(
        npc.monster,
        "EXP no template e sem loja = criatura de combate"
    );

    // Spawn: a foto do mapa 1 vira `NpcSpec` com nome do portal, posição e Stats.
    let specs = wyserv_world::npc_specs(&inicial, 1);
    assert_eq!(specs.len(), 1);
    assert_eq!((specs[0].id, specs[0].name.as_str()), (7, "Aylin"));
    assert_eq!(
        (specs[0].position.x, specs[0].position.y),
        (40, 40),
        "posição da definição"
    );
    assert_eq!(specs[0].stats.max_hp, 900);
    assert_eq!(specs[0].kind, wyd_core::EntityKind::Monster);
    assert_eq!(specs[0].exp, 5_000);
    assert!(
        wyserv_world::npc_specs(&inicial, 2).is_empty(),
        "mapa sem NPC não materializa nada"
    );

    // Editar a posição e trocar o template muda a foto; a revisão sobe e o mundo reconcilia.
    source.set_npc(7, 1, 50, 60, true, npc_template(7_500, 1_200));
    assert!(matches!(
        live.refresh(&source),
        ReloadOutcome::Applied { .. }
    ));
    let novo = live.snapshot();
    let npc = novo.npc(7).expect("NPC após a recarga");
    assert_eq!((npc.pos_x, npc.pos_y), (50, 60));
    assert_eq!(npc.exp, 7_500);
    assert_eq!(npc.max_hp, 1_200);
    let specs = wyserv_world::npc_specs(&novo, 1);
    assert_eq!((specs[0].position.x, specs[0].position.y), (50, 60));
    assert_eq!(specs[0].exp, 7_500);

    // Desabilitar no portal tira o NPC do mundo na mesma recarga.
    source.set_npc(7, 1, 50, 60, false, npc_template(7_500, 1_200));
    assert!(matches!(
        live.refresh(&source),
        ReloadOutcome::Applied { .. }
    ));
    assert!(wyserv_world::npc_specs(&live.snapshot(), 1).is_empty());
}

#[test]
fn template_de_npc_invalido_recusa_a_foto_inteira() {
    let source = MemDb::seeded();
    let live = boot(&source);
    assert_eq!(live.snapshot().npc(7).map(|npc| npc.exp), Some(5_000));

    // Template curto (não é STRUCT_MOB de 816 B): a foto inteira é recusada.
    source.set_npc(7, 1, 40, 40, true, vec![0u8; 16]);
    let falha = live.refresh(&source);
    assert!(
        matches!(falha, ReloadOutcome::Failed { .. }),
        "template inválido não pode ser publicado: {falha:?}"
    );
    assert_eq!(live.snapshot().npc(7).map(|npc| npc.exp), Some(5_000));
}

/// Apelido local para não repetir o caminho completo nos asserts acima.
mod wyserv_world {
    pub use wyd_server::world::npc_specs;
}
