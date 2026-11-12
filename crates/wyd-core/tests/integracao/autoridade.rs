//! Autoridade do servidor (GAME-05): pacote forjado é recusado **sem** alterar estado.
//!
//! Usa só a API pública do core, como a borda de rede faria: o teste entrega intenção (destino,
//! alvo) e confere que o resultado é do servidor. Nenhum teste aqui passa dano ou posição
//! "confirmada pelo cliente" — isso nem existe na assinatura.

use wyd_core::{
    attack, cast, step, AttackRules, ComponentStore, CoreError, DamageModel, DomainEvent, Entity,
    EntityKind, Inventory, Item, MoveRules, Position, SkillBook, SkillDef, Stats,
};

fn stats(hp: u32, attack: u32) -> Stats {
    Stats {
        level: 40,
        hp,
        max_hp: hp,
        mp: 50,
        max_mp: 50,
        attack,
        defense: 10,
    }
}

fn spawn(store: &mut ComponentStore, pos: Position, s: Stats) -> Entity {
    store
        .spawn(EntityKind::Player, pos, s, Inventory::default())
        .expect("spawn válido")
}

/// Modelo de dano de teste: fecha o contrato do trait sem inventar fórmula de jogo.
struct DanoFixo(u32);

impl DamageModel for DanoFixo {
    fn damage(&self, _attacker: Stats, _defender: Stats) -> u32 {
        self.0
    }
}

fn regras_movimento() -> MoveRules {
    MoveRules::new(4, Position { x: 0, y: 0 }, Position { x: 1_000, y: 1_000 })
        .expect("regras de movimento")
}

fn regras_combate() -> AttackRules {
    AttackRules::new(2, 800).expect("regras de combate")
}

/// Teleporte (andar 100 tiles em 250 ms) é recusado: posição, fila e cooldown intactos.
#[test]
fn teleporte_forjado_nao_move_nem_publica() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, Position { x: 10, y: 10 }, stats(100, 20));
    let antes = store.position_of(e);

    assert_eq!(
        step(
            &mut store,
            e,
            Position { x: 110, y: 10 },
            250,
            regras_movimento()
        ),
        Err(CoreError::MoveTooFar {
            distance: 100,
            allowed: 1
        })
    );
    assert_eq!(store.position_of(e), antes);
    assert!(store.events().is_empty(), "recusa não vira evento");
}

/// Passo dentro do orçamento é aceito e a posição gravada é a **do servidor**.
#[test]
fn passo_dentro_do_orcamento_e_gravado_pelo_servidor() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, Position { x: 10, y: 10 }, stats(100, 20));
    let destino = Position { x: 12, y: 11 };

    assert_eq!(
        step(&mut store, e, destino, 500, regras_movimento()),
        Ok(destino)
    );
    assert_eq!(store.position_of(e), Some(destino));
    assert_eq!(
        store.drain_events(),
        vec![DomainEvent::Moved {
            entity: e,
            from: Position { x: 10, y: 10 },
            to: destino
        }]
    );
}

/// Dano "informado pelo cliente" não existe: o HP cai pelo modelo do servidor, não por um
/// número entregue junto do pedido.
#[test]
fn dano_aplicado_vem_do_modelo_do_servidor() {
    let mut store = ComponentStore::default();
    let atacante = spawn(&mut store, Position { x: 10, y: 10 }, stats(100, 999));
    let alvo = spawn(&mut store, Position { x: 11, y: 10 }, stats(100, 5));

    let golpe = attack(
        &mut store,
        atacante,
        alvo,
        1_000,
        regras_combate(),
        &DanoFixo(6),
    )
    .expect("ataque válido");
    assert_eq!((golpe.amount, golpe.remaining_hp), (6, 94));
    assert_eq!(store.stats_of(alvo).expect("vivo").hp, 94);
}

/// Ataque repetido dentro do cooldown é recusado com estado, HP e fila inalterados.
#[test]
fn ataque_em_cooldown_nao_altera_estado() {
    let mut store = ComponentStore::default();
    let atacante = spawn(&mut store, Position { x: 10, y: 10 }, stats(100, 20));
    let alvo = spawn(&mut store, Position { x: 10, y: 11 }, stats(100, 20));

    attack(
        &mut store,
        atacante,
        alvo,
        1_000,
        regras_combate(),
        &DanoFixo(5),
    )
    .expect("primeiro");
    store.drain_events();

    assert_eq!(
        attack(
            &mut store,
            atacante,
            alvo,
            1_200,
            regras_combate(),
            &DanoFixo(5)
        ),
        Err(CoreError::AttackCooldown { remaining_ms: 600 })
    );
    assert_eq!(store.stats_of(alvo).expect("vivo").hp, 95);
    assert_eq!(
        store.next_attack_ms(atacante),
        Some(1_800),
        "cooldown é o do primeiro golpe"
    );
    assert!(store.events().is_empty());
}

/// Item no slot inexistente, gema falsa e entidade morta continuam recusados pelo core:
/// a autoridade cobre inventário além de movimento/combate.
#[test]
fn inventario_forjado_e_recusado_sem_alterar_item() {
    let mut store = ComponentStore::default();
    let mut inv = Inventory::default();
    inv.equip[3] = Item {
        id: 1668,
        qty: 1,
        refine: 13,
        ..Item::default()
    };
    inv.bag[0] = Item {
        id: 3387,
        qty: 1,
        ..Item::default()
    };
    let e = store
        .spawn(EntityKind::Player, Position::default(), stats(100, 20), inv)
        .expect("spawn válido");

    assert_eq!(
        store.apply_gem(e, 99, 3387),
        Err(CoreError::SlotOutOfRange { slot: 99, max: 15 })
    );
    assert_eq!(
        store.apply_gem(e, 3, 2441),
        Err(CoreError::UnknownGem { item_index: 2441 })
    );
    assert_eq!(
        store.inventory(e).expect("vivo").bag_qty(3387),
        1,
        "recusa não gasta a gema"
    );
}

/// Alvo despawnado (id obsoleto) não recebe golpe nem gera evento.
#[test]
fn ataque_em_entidade_morta_e_recusado() {
    let mut store = ComponentStore::default();
    let atacante = spawn(&mut store, Position { x: 10, y: 10 }, stats(100, 20));
    let alvo = spawn(&mut store, Position { x: 11, y: 10 }, stats(100, 20));
    assert!(store.despawn(alvo));

    assert_eq!(
        attack(
            &mut store,
            atacante,
            alvo,
            0,
            regras_combate(),
            &DanoFixo(5)
        ),
        Err(CoreError::EntityNotAlive {
            index: alvo.index,
            generation: alvo.generation
        })
    );
    assert!(store.events().is_empty());
}

/// Toque de dados novos: `Stats` inválido continua barrado no spawn (fronteira fria).
#[test]
fn spawn_invalido_nao_cria_entidade() {
    let mut store = ComponentStore::default();
    assert!(store
        .spawn(
            EntityKind::Player,
            Position::default(),
            Stats {
                level: 1,
                hp: 10,
                max_hp: 0,
                mp: 0,
                max_mp: 0,
                attack: 1,
                defense: 1,
            },
            Inventory::default(),
        )
        .is_err());
    assert_eq!(store.live_count(), 0);
}

/// Skill: o custo é do servidor; skill não aprendida não cobra MP nem publica evento.
#[test]
fn skill_nao_aprendida_e_recusada_sem_cobrar_mp() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, Position { x: 10, y: 10 }, stats(100, 20));
    let alvo = spawn(&mut store, Position { x: 11, y: 11 }, stats(100, 20));
    let livro = SkillBook::new(vec![SkillDef {
        id: 32,
        mana_cost: 7,
        cooldown_ms: 500,
        range_tiles: 5,
    }])
    .expect("livro válido");

    assert_eq!(
        cast(&mut store, e, 32, &[], Some(alvo), 0, &livro),
        Err(CoreError::SkillNotLearned { skill: 32 })
    );
    assert_eq!(store.stats_of(e).expect("vivo").mp, 50, "MP intacto");
    assert!(store.events().is_empty());
}

/// Skill aprendida: MP e cooldown são cobrados pelo servidor e o evento carrega o custo.
#[test]
fn skill_aprendida_cobra_mp_e_publica_evento() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, Position { x: 10, y: 10 }, stats(100, 20));
    let alvo = spawn(&mut store, Position { x: 12, y: 10 }, stats(100, 20));
    let livro = SkillBook::new(vec![SkillDef {
        id: 32,
        mana_cost: 7,
        cooldown_ms: 500,
        range_tiles: 5,
    }])
    .expect("livro válido");

    let uso = cast(&mut store, e, 32, &[32], Some(alvo), 1_000, &livro).expect("skill válida");
    assert_eq!((uso.mana_left, uso.next_skill_ms), (43, 1_500));
    assert_eq!(
        store.drain_events(),
        vec![DomainEvent::SkillCast {
            caster: e,
            skill: 32,
            mana_spent: 7,
            mana_left: 43
        }]
    );
}

/// Alvo fora do alcance da skill: recusa sem cobrar MP nem armar cooldown.
#[test]
fn skill_fora_de_alcance_nao_cobra() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, Position { x: 10, y: 10 }, stats(100, 20));
    let alvo = spawn(&mut store, Position { x: 40, y: 10 }, stats(100, 20));
    let livro = SkillBook::new(vec![SkillDef {
        id: 32,
        mana_cost: 7,
        cooldown_ms: 500,
        range_tiles: 5,
    }])
    .expect("livro válido");

    assert_eq!(
        cast(&mut store, e, 32, &[32], Some(alvo), 0, &livro),
        Err(CoreError::TargetOutOfRange {
            distance: 30,
            range: 5
        })
    );
    assert_eq!(store.stats_of(e).expect("vivo").mp, 50);
    assert_eq!(store.next_skill_ms(e), Some(0));
}
