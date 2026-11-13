//! Piedade de refino persistida + `SendItem` (GAME-16).
//!
//! Cobre o caminho de pó ponta a ponta contra um Postgres descartável: a piedade
//! acumula na falha qualificada, zera no sucesso, vai no fio empacotada no par
//! EF_SANC (`encode_sanc_cvalue`) e item com `EF_NOSANC` no catálogo é vetado sem
//! mutação. O `SendItem` (0x0182) é verificado byte a byte no corpo do frame.
//!
//! Não roda sem banco: a ausência de `WYD_TEST_DSN` **pula** com mensagem explícita.
//! Suba o Postgres e aplique a série + seeds como em `wyd-db/tests/pg.rs`:
//!
//! ```bash
//! docker run -d --rm --name wyd-pg43 -e POSTGRES_PASSWORD=dev -e POSTGRES_DB=wyd \
//!   -p 55432:5432 postgres:17-alpine
//! # aplica os ups de migrations/*.sql (awk entre os marcadores) e seeds/dev.sql
//! WYD_TEST_DSN=postgres://postgres:dev@127.0.0.1:55432/wyd cargo test -p wyd-server --test refino_piedade
//! ```

#[path = "../common/mod.rs"]
mod common;

use std::sync::Arc;

use wyd_db::{
    Accounts, Character, Characters, ConfigRateRow, ItemRow, PgBackend, PoolConfig, Role,
};
use wyd_protocol::payload::WirePayload;
use wyd_protocol::{SendItem, UseItem};
use wyd_server::characters::{Selected, SelectionStore};
use wyd_server::config::{GameConfig, LiveConfig};
use wyd_server::gameplay::UseItemRolls;
use wyd_server::personagens::Item;
use wyd_sim::formula::refine::EF_SANC;

fn suffix() -> u128 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .expect("relógio")
        .as_nanos()
}

fn rate(domain: &str, key: &str, value: i64) -> ConfigRateRow {
    ConfigRateRow {
        domain: domain.to_string(),
        key: key.to_string(),
        value,
        unit: if domain == "sanc" { "%" } else { "bps" }.to_string(),
        source: "seed".to_string(),
        overridden: false,
    }
}

/// Snapshot mínimo que o resolvedor de refino aceita: escada 1..=15 + células de pó.
fn foto() -> GameConfig {
    use wyd_sim::formula::refine::REFINE_RATE_BPS;
    let mut rates: Vec<ConfigRateRow> = (1..=15)
        .map(|step| {
            rate(
                "refine",
                &step.to_string(),
                i64::from(REFINE_RATE_BPS[step - 1]),
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
    rates.push(rate("setting", "event.exp_bonus", 0));
    GameConfig::from_rows(1, &rates, Vec::new(), Vec::new(), Vec::new(), Vec::new())
        .expect("configuração de teste válida")
}

fn linha(slot: i64, item_id: i64, qty: i64, refine: i64) -> ItemRow {
    ItemRow {
        slot,
        item_id,
        qty,
        refine,
        ..ItemRow::default()
    }
}

fn pacote() -> UseItem {
    // Pó no `Carry[0]` (slot 16), alvo equipado no slot 1 — o caminho de pó clássico.
    UseItem {
        sour_type: 1,
        sour_pos: 0,
        dest_type: 0,
        dest_pos: 1,
        grid_x: 0,
        grid_y: 0,
        warp_id: 0,
    }
}

fn runtime() -> Arc<tokio::runtime::Runtime> {
    Arc::new(
        tokio::runtime::Builder::new_multi_thread()
            .worker_threads(2)
            .enable_all()
            .build()
            .expect("runtime"),
    )
}

struct Mundo {
    backend: Arc<PgBackend>,
    personagem: Character,
}

impl Mundo {
    async fn novo(dsn: &str, alvo: ItemRow) -> Option<Self> {
        let backend = Arc::new(
            PgBackend::connect(dsn, PoolConfig::local())
                .await
                .expect("conecta no Postgres de teste"),
        );
        let conta = backend
            .create_account(&format!("piedade{}", suffix()), Role::Player)
            .await
            .expect("conta de teste");
        let base = Character {
            account_id: conta.id,
            slot: 0,
            name: format!("P{}", suffix() % 1_000_000_000),
            class: 0,
            level: 20,
            exp: 0,
            hp: 120,
            hp_max: 120,
            mp: 60,
            mp_max: 60,
            pos_x: 2096,
            pos_y: 2096,
            ..Character::default()
        };
        let personagem = backend
            .create_character(&base)
            .await
            .expect("personagem de teste");
        // Pó de Ori com 2 cargas + alvo; semeia trocando o inventário inteiro.
        let itens = vec![linha(16, 412, 2, 0), alvo];
        backend
            .replace_items(personagem.id, &itens, &[])
            .await
            .expect("inventário de teste");
        Some(Self {
            backend,
            personagem,
        })
    }

    fn loja(&self, rt: &Arc<tokio::runtime::Runtime>) -> SelectionStore {
        SelectionStore::new(Arc::clone(&self.backend), Arc::clone(rt))
            .with_config(Arc::new(LiveConfig::new(foto())))
    }

    fn selecionado(&self, itens: Vec<ItemRow>) -> Selected {
        Selected {
            character: self.personagem.clone(),
            items: itens,
            short_skill: [0xFF; 16],
            progression: wyd_db::Progression::default(),
        }
    }

    async fn itens(&self) -> Vec<ItemRow> {
        self.backend
            .items(self.personagem.id)
            .await
            .expect("lê inventário")
    }
}

#[test]
fn piedade_acumula_na_falha_e_zera_no_sucesso() {
    let Some(dsn) = common::dsn_ou_skip("refino/piedade") else {
        return;
    };
    let rt = runtime();
    // O `SelectionStore` bloqueia o runtime por dentro (`block_on` na porta): cada passo
    // assíncrono roda num `block_on` próprio, nunca aninhado.
    let Some(mundo) = rt.block_on(Mundo::novo(&dsn, linha(1, 1_900, 1, 4))) else {
        return;
    };
    let loja = mundo.loja(&rt);
    // Falha qualificada: taxa 10% (PO:5 = 10), roll 99 erra, `pity_roll` 0 acumula.
    let mut selecionado = mundo.selecionado(rt.block_on(mundo.itens()));
    let falha = loja
        .use_item(
            &mut selecionado,
            &pacote(),
            UseItemRolls {
                roll: 99,
                pity_roll: 0,
            },
        )
        .expect("falha de refino resolve");
    assert!(!falha.resolution.success);
    assert_eq!(falha.resolution.next_pity, 1);
    assert_eq!(
        rt.block_on(mundo.backend.refine_pity(mundo.personagem.id, 1))
            .expect("piedade"),
        1,
        "a falha gravou piedade 1 no banco"
    );
    // O alvo continua +4 e o par sanc no fio carrega a piedade: 4 + 10×1.
    let alvo = selecionado
        .items
        .iter()
        .find(|item| item.slot == 1)
        .expect("alvo na sessão");
    let quadro = Item::from_row_with_pity(alvo, 1).to_bytes();
    assert_eq!(quadro[2], EF_SANC);
    assert_eq!(quadro[3], 14);

    // Sucesso com a piedade acumulada: sobe para +5 e zera a piedade no banco.
    let sucesso = loja
        .use_item(
            &mut selecionado,
            &pacote(),
            UseItemRolls {
                roll: 0,
                pity_roll: 3,
            },
        )
        .expect("sucesso de refino resolve");
    assert!(sucesso.resolution.success);
    assert_eq!(sucesso.resolution.to_level, 5);
    assert_eq!(sucesso.resolution.next_pity, 0);
    let depois = rt.block_on(mundo.itens());
    let alvo = depois
        .iter()
        .find(|item| item.slot == 1)
        .expect("alvo segue no inventário");
    assert_eq!(alvo.refine, 5);
    assert_eq!(
        rt.block_on(mundo.backend.refine_pity(mundo.personagem.id, 1))
            .expect("piedade"),
        0,
        "o sucesso zerou a piedade no banco"
    );
}

#[test]
fn send_item_do_refino_tem_layout_do_legado() {
    // +7 com 3 de piedade: corpo de 12 B com `inv_type`@12, `slot`@14 e o item@16.
    let quadro = Item::from_row_with_pity(&linha(1, 1_900, 1, 7), 3).to_bytes();
    let resposta = SendItem {
        inv_type: 0,
        slot: 1,
        item: quadro,
    };
    assert_eq!(SendItem::WIRE_LEN, 24);
    let mut cru = vec![0u8; SendItem::WIRE_LEN];
    resposta.encode_into(&mut cru).expect("codifica");
    assert_eq!(&cru[12..14], &[0, 0], "equipamento@12");
    assert_eq!(&cru[14..16], &[1, 0], "slot@14");
    assert_eq!(
        &cru[16..24],
        &[0x6c, 0x07, EF_SANC, 37, 0, 0, 0, 0],
        "sIndex 1900 + sanc(7 + 10×3)"
    );
    assert_eq!(
        SendItem::decode(&cru).expect("decodifica"),
        resposta,
        "round-trip do frame"
    );
}

#[test]
fn ef_nosanc_veta_sem_mutacao() {
    // Alma da Fênix (1741) carrega `EF_NOSANC` no catálogo semeado.
    let Some(dsn) = common::dsn_ou_skip("refino/piedade") else {
        return;
    };
    let rt = runtime();
    let Some(mundo) = rt.block_on(Mundo::novo(&dsn, linha(1, 1_741, 1, 0))) else {
        return;
    };
    let loja = mundo.loja(&rt);
    let mut selecionado = mundo.selecionado(rt.block_on(mundo.itens()));
    let erro = loja
        .use_item(
            &mut selecionado,
            &pacote(),
            UseItemRolls {
                roll: 0,
                pity_roll: 0,
            },
        )
        .expect_err("EF_NOSANC veta");
    let texto = erro.to_string();
    assert!(texto.contains("EF_NOSANC"), "veto de catálogo: {texto}");
    let depois = rt.block_on(mundo.itens());
    assert_eq!(depois.len(), 2, "nada foi consumido nem destruído");
    assert!(
        depois.iter().any(|item| item.slot == 16 && item.qty == 2),
        "o pó segue intacto"
    );
    assert_eq!(
        rt.block_on(mundo.backend.refine_pity(mundo.personagem.id, 1))
            .expect("piedade"),
        0,
        "veto não acumula piedade"
    );
}

#[test]
fn replace_items_poda_piedade_de_slot_que_saiu() {
    let Some(dsn) = common::dsn_ou_skip("refino/piedade") else {
        return;
    };
    let rt = runtime();
    rt.block_on(async {
        let backend = PgBackend::connect(&dsn, PoolConfig::local())
            .await
            .expect("conecta no Postgres de teste");
        let conta = backend
            .create_account(&format!("poda{}", suffix()), Role::Player)
            .await
            .expect("conta de teste");
        let personagem = backend
            .create_character(&Character {
                account_id: conta.id,
                slot: 0,
                name: format!("Q{}", suffix() % 1_000_000_000),
                class: 0,
                level: 20,
                exp: 0,
                hp: 120,
                hp_max: 120,
                mp: 60,
                mp_max: 60,
                pos_x: 2096,
                pos_y: 2096,
                ..Character::default()
            })
            .await
            .expect("personagem de teste");

        backend
            .replace_items(personagem.id, &[linha(1, 1_900, 1, 4)], &[(1, 6)])
            .await
            .expect("semeia piedade");
        assert_eq!(
            backend
                .refine_pity(personagem.id, 1)
                .await
                .expect("piedade"),
            6
        );
        // O item saiu do inventário (destruído/consumido): a piedade órfã é podada.
        backend
            .replace_items(personagem.id, &[linha(16, 412, 1, 0)], &[])
            .await
            .expect("troca o inventário");
        assert_eq!(
            backend
                .refine_pity(personagem.id, 1)
                .await
                .expect("piedade"),
            0,
            "slot fora do inventário não guarda piedade"
        );
    });
}

#[test]
fn item_has_effect_le_do_catalogo_semeado() {
    let Some(dsn) = common::dsn_ou_skip("refino/piedade") else {
        return;
    };
    let rt = runtime();
    rt.block_on(async {
        let backend = PgBackend::connect(&dsn, PoolConfig::local())
            .await
            .expect("conecta no Postgres de teste");
        assert!(
            !backend
                .item_has_effect(1_900, "EF_NOSANC")
                .await
                .expect("lê catálogo"),
            "bota conjuradora é refinável"
        );
        assert!(
            backend
                .item_has_effect(1_741, "EF_NOSANC")
                .await
                .expect("lê catálogo"),
            "alma da fênix carrega EF_NOSANC"
        );
        assert!(
            !backend
                .item_has_effect(999_999, "EF_NOSANC")
                .await
                .expect("lê catálogo"),
            "item inexistente não veta"
        );
    });
}
