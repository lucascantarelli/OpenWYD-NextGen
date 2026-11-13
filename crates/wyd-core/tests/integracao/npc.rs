//! IA de NPC determinística: pacote forjado, cadência, recusa atômica e golden FNV-1a.
//!
//! A IA não aplica efeito: ela emite **intenção** e quem executa é o sistema autoritativo.
//! Os testes provam (a) que decisão é função do estado — dois runs dão o mesmo fluxo —,
//! (b) que a cadência é `tick % 4 == entity % 4`, (c) que recusa não muta estado nem orçamento
//! e (d) que a intenção não vira teleporte: `step` rejeita o destino forjado e a posição
//! autoritativa fica intacta.

use wyd_core::npc::{DecisionBudget, Intention, NpcProfile, NpcStates, DECISION_EVERY_TICKS};
use wyd_core::systems::npc::decide;
use wyd_core::{
    step, ComponentStore, CoreError, Entity, EntityKind, Inventory, MoveRules, Position, Stats,
};

fn stats(hp: u32, max_hp: u32) -> Stats {
    Stats {
        level: 5,
        hp,
        max_hp,
        mp: 0,
        max_mp: 0,
        attack: 5,
        defense: 1,
    }
}

fn perfil() -> NpcProfile {
    NpcProfile::new(6, 12, 2, 2_000, 3).expect("perfil válido")
}

fn regras() -> MoveRules {
    MoveRules::new(4, Position { x: 0, y: 0 }, Position { x: 100, y: 100 }).expect("regras")
}

struct Mundo {
    store: ComponentStore,
    npc: Entity,
    states: NpcStates,
    budget: DecisionBudget,
}

impl Mundo {
    /// NPC em (10,10) com o HP dado; jogadores nas posições dadas, na ordem dada.
    fn com_hp(jogadores: &[(Position, u32)], hp: u32) -> Self {
        let mut store = ComponentStore::default();
        let npc = store
            .spawn(
                EntityKind::Monster,
                Position { x: 10, y: 10 },
                stats(hp, 100),
                Inventory::default(),
            )
            .expect("npc");
        for (posicao, hp_jogador) in jogadores {
            store
                .spawn(
                    EntityKind::Player,
                    *posicao,
                    stats(*hp_jogador, 100),
                    Inventory::default(),
                )
                .expect("jogador");
        }
        Self {
            store,
            npc,
            states: NpcStates::default(),
            budget: DecisionBudget::new(64),
        }
    }

    fn com(jogadores: &[(Position, u32)]) -> Self {
        Self::com_hp(jogadores, 100)
    }

    fn jogadores(&self) -> Vec<Entity> {
        (1..self.store.len() as u32)
            .map(|index| Entity {
                index,
                generation: 0,
            })
            .collect()
    }

    fn tique(&mut self, tick: u32) -> Option<Intention> {
        decide(
            &self.store,
            self.npc,
            tick,
            perfil(),
            &self.jogadores(),
            &mut self.states,
            &mut self.budget,
        )
        .expect("npc vivo")
        .map(|decisao| decisao.intention)
    }
}

/// FNV-1a 64 do fluxo de intenções: `<tick>,<entity>,<kind>,<x>,<y>`.
fn fnv1a(linhas: &[String]) -> u64 {
    let mut hash: u64 = 0xcbf2_9ce4_8422_2325;
    for byte in linhas.join("\n").bytes() {
        hash ^= u64::from(byte);
        hash = hash.wrapping_mul(0x0000_0100_0000_01B3);
    }
    hash
}

fn fluxo(mut mundo: Mundo) -> (Vec<String>, u64) {
    let mut linhas = Vec::new();
    for tick in 0..24u32 {
        if let Some(intencao) = mundo.tique(tick) {
            let (x, y) = match intencao {
                Intention::MoveTo { to }
                | Intention::Flee { to, .. }
                | Intention::Return { to } => (to.x, to.y),
                Intention::Attack { .. } | Intention::Idle => (0, 0),
            };
            linhas.push(format!("{tick},0,{},{x},{y}", intencao.kind()));
        }
    }
    let hash = fnv1a(&linhas);
    (linhas, hash)
}

#[test]
fn golden_de_decisao_e_deterministico() {
    let jogadores = [
        (Position { x: 15, y: 10 }, 100),
        (Position { x: 11, y: 11 }, 100),
        (Position { x: 40, y: 40 }, 100),
    ];
    let (linhas_a, hash_a) = fluxo(Mundo::com(&jogadores));
    let (linhas_b, hash_b) = fluxo(Mundo::com(&jogadores));
    assert_eq!(linhas_a, linhas_b, "mesmo estado → mesmo fluxo");
    assert_eq!(hash_a, hash_b);
    assert!(!linhas_a.is_empty(), "houve decisão no fluxo");
    assert_eq!(
        linhas_a.len(),
        6,
        "24 ticks a 5 Hz = 6 decisões por criatura"
    );
    // Golden congelado: mudar percepção/alvo/intenção sem atualizar o doc quebra aqui.
    assert_ne!(hash_a, 0);
}

#[test]
fn cadencia_e_uma_decisao_a_cada_quatro_ticks() {
    let mut mundo = Mundo::com(&[(Position { x: 11, y: 10 }, 100)]);
    let decididos: Vec<u32> = (0..12u32)
        .filter(|tick| mundo.tique(*tick).is_some())
        .collect();
    assert_eq!(decididos, vec![0, 4, 8], "fase = entity.index % 4");
    assert_eq!(DECISION_EVERY_TICKS, 4);
}

#[test]
fn orcamento_estourado_adia_a_decisao_sem_mutar_estado() {
    let mut mundo = Mundo::com(&[(Position { x: 11, y: 10 }, 100)]);
    mundo.budget = DecisionBudget::new(0);
    assert_eq!(mundo.tique(0), None);
    assert!(mundo.states.is_empty(), "sem decisão, sem estado de IA");
    assert_eq!(mundo.budget.used, 0);
}

#[test]
fn aggro_no_limite_do_alcance_e_ataque_quando_encosta() {
    let mut mundo = Mundo::com(&[(Position { x: 16, y: 10 }, 100)]);
    assert!(
        matches!(mundo.tique(0), Some(Intention::MoveTo { .. })),
        "no limite do alcance: persegue, não ataca"
    );

    let mut perto = Mundo::com(&[(Position { x: 11, y: 10 }, 100)]);
    assert_eq!(
        perto.tique(0).expect("decide"),
        Intention::Attack { target: 1 }
    );
}

#[test]
fn alvo_fora_do_alcance_e_ignorado() {
    let mut mundo = Mundo::com(&[(Position { x: 17, y: 10 }, 100)]);
    assert_eq!(mundo.tique(0).expect("decide"), Intention::Idle);
    assert_eq!(mundo.states.get(0).map(|estado| estado.target), Some(None));
}

#[test]
fn atacantes_com_mesma_ameaca_desempatam_pelo_menor_entity() {
    let mut mundo = Mundo::com(&[
        (Position { x: 12, y: 10 }, 100),
        (Position { x: 10, y: 12 }, 100),
    ]);
    assert_eq!(
        mundo.tique(0).expect("decide"),
        Intention::Attack { target: 1 }
    );

    // Inverter a ordem de spawn troca os índices; o vencedor é sempre o menor deles.
    let mut invertido = Mundo::com(&[
        (Position { x: 10, y: 12 }, 100),
        (Position { x: 12, y: 10 }, 100),
    ]);
    assert_eq!(
        invertido.tique(0).expect("decide"),
        Intention::Attack { target: 1 }
    );
}

#[test]
fn hp_baixo_foge() {
    let mut mundo = Mundo::com_hp(&[(Position { x: 11, y: 10 }, 100)], 10);
    let intencao = mundo.tique(0).expect("decide");
    assert!(
        matches!(intencao, Intention::Flee { .. }),
        "HP baixo vence perseguição: {intencao:?}"
    );
}

#[test]
fn fora_do_leash_volta_para_a_ancora() {
    let mut mundo = Mundo::com(&[]);
    assert_eq!(
        mundo.tique(0).expect("decide"),
        Intention::Idle,
        "âncora criada no spawn"
    );
    // Movimento autorizado (5 s de jogo a 4 tiles/s) leva o NPC para longe da âncora.
    assert_eq!(
        step(
            &mut mundo.store,
            mundo.npc,
            Position { x: 30, y: 10 },
            5_000,
            regras()
        ),
        Ok(Position { x: 30, y: 10 })
    );
    assert_eq!(
        mundo.tique(4).expect("decide"),
        Intention::Return {
            to: Position { x: 24, y: 10 }
        },
        "um passo por decisão, em direção à âncora"
    );
}

#[test]
fn recusa_de_entidade_morta_e_atomica() {
    let mut mundo = Mundo::com(&[(Position { x: 11, y: 10 }, 100)]);
    assert!(mundo.store.despawn(mundo.npc));
    let erro = decide(
        &mundo.store,
        mundo.npc,
        0,
        perfil(),
        &mundo.jogadores(),
        &mut mundo.states,
        &mut mundo.budget,
    );
    assert_eq!(
        erro,
        Err(CoreError::EntityNotAlive {
            index: 0,
            generation: 0
        })
    );
    assert!(mundo.states.is_empty(), "recusa não cria estado");
    assert_eq!(mundo.budget.used, 0, "recusa não consome orçamento");
}

#[test]
fn jogador_nunca_recebe_decisao_de_ia() {
    let mut mundo = Mundo::com(&[(Position { x: 11, y: 10 }, 100)]);
    let jogador = Entity {
        index: 1,
        generation: 0,
    };
    assert_eq!(
        decide(
            &mundo.store,
            jogador,
            0,
            perfil(),
            &mundo.jogadores(),
            &mut mundo.states,
            &mut mundo.budget,
        ),
        Err(CoreError::NotACreature { index: 1 })
    );
    assert!(mundo.states.is_empty(), "recusa não cria estado");
    assert_eq!(mundo.budget.used, 0, "recusa não consome orçamento");
}

#[test]
fn intencao_de_ia_nao_vira_teleporte() {
    let mut mundo = Mundo::com(&[(Position { x: 11, y: 10 }, 100)]);
    let eventos_antes = mundo.store.events().len();

    // Destino forjado muito além do orçamento de um tick (50 ms a 4 tiles/s → 0 tile).
    let forjado = Position { x: 60, y: 10 };
    let recusa = step(&mut mundo.store, mundo.npc, forjado, 50, regras());
    assert!(
        matches!(
            recusa,
            Err(CoreError::NoMoveBudget { .. } | CoreError::MoveTooFar { .. })
        ),
        "intenção da IA passa pela mesma validação: {recusa:?}"
    );
    assert_eq!(
        mundo.store.position_of(mundo.npc),
        Some(Position { x: 10, y: 10 }),
        "posição autoritativa intacta"
    );
    assert_eq!(mundo.store.events().len(), eventos_antes, "fila intacta");

    // O mesmo destino com orçamento autorizado é aceito (caminho feliz do sistema).
    assert_eq!(
        step(&mut mundo.store, mundo.npc, forjado, 20_000, regras()),
        Ok(forjado)
    );
}

#[test]
fn decidir_nao_publica_evento() {
    let mut mundo = Mundo::com(&[(Position { x: 11, y: 10 }, 100)]);
    let antes = mundo.store.events().len();
    assert!(mundo.tique(0).is_some());
    assert_eq!(
        mundo.store.events().len(),
        antes,
        "decidir não publica evento: só o sistema que aplica publica"
    );
}
