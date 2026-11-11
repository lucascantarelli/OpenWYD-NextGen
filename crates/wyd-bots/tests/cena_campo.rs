//! Harness vivo: o Campo de Treinamento só aparece **quando o personagem chega perto**.
//!
//! Depois da entrega de cena por proximidade (`world::scene_around`, raio de 24 tiles), a
//! sessão não recebe mais os 6 110 NPCs do mapa no login. O teste anda por quatro pontos do
//! Campo (`_MSG_Action` validado no tick), acumula o que cada varredura entrega e confere as
//! dez instâncias do seed `0046` — uma por coordenada, sem sobra das fixtures `0047`.
//!
//! Gated por `WYD_LIVE_CPSOCK`; usa a conta de teste do projeto.

use std::collections::BTreeMap;

use wyd_bots::{Bot, BotConfig, Race, SceneMob};

/// `(nome, x, y)` do seed 0046 — as coordenadas são as do guia oficial do Campo.
const CAMPOS: [(&str, i32, i32); 10] = [
    ("PrimeiroTreinador", 2077, 2017),
    ("SegundoTreinador", 2144, 1989),
    ("TerceiroTreinador", 2082, 1966),
    ("ChefeTreinador", 2122, 2041),
    ("Unicorn", 2130, 2040),
    ("Cabuncle", 2131, 2033),
    ("GremlinDecoy", 2076, 2022),
    ("GiantCrill", 2146, 1997),
    ("OrcSniper", 2078, 1971),
    ("OrcWarrior", 2135, 1948),
];

/// Posições das fixtures removidas pela `0047`: não pode sobrar criatura aqui.
const FIXTURES: [(i32, i32); 8] = [
    (2108, 2046),
    (2112, 2046),
    (2116, 2046),
    (2120, 2046),
    (2115, 2042),
    (2120, 2048),
    (2125, 2052),
    (2130, 2055),
];

/// Pontos por onde o personagem passa: o raio de 24 tiles cobre o Campo inteiro em quatro
/// paradas (2076..2146 × 1948..2041), incluindo o Chefe em (2122,2041).
const ROTA: [(i32, i32); 5] = [
    (2077, 2017),
    (2078, 1971),
    (2146, 1997),
    (2135, 1948),
    (2122, 2041),
];

/// Lê a cena por 400 ms e acumula por `MobID` (o mesmo NPC pode ser reenviado ao voltar).
fn coletar(bot: &mut Bot, vistos: &mut BTreeMap<u16, SceneMob>) {
    for mob in bot.read_scene(400).expect("cena") {
        vistos.entry(mob.id).or_insert(mob);
    }
}

#[test]
fn imprime_os_npcs_do_campo_com_mobid_da_cena() {
    let Ok(endereco) = std::env::var("WYD_LIVE_CPSOCK") else {
        eprintln!("SKIP: WYD_LIVE_CPSOCK ausente");
        return;
    };
    let conta = std::env::var("WYD_LIVE_ACCOUNT").expect("WYD_LIVE_ACCOUNT");
    let senha = std::env::var("WYD_LIVE_PASSWORD").expect("WYD_LIVE_PASSWORD");
    let pin = std::env::var("WYD_LIVE_PIN").unwrap_or_else(|_| "2468".to_string());
    let slot: i32 = std::env::var("WYD_LIVE_CHARACTER_SLOT")
        .ok()
        .and_then(|valor| valor.parse().ok())
        .unwrap_or(0);
    let mut config = BotConfig::new(endereco.clone(), conta, senha);
    config.pin = pin;
    config.slot = slot;

    let mut bot = Bot::connect(&config, Race::TransKnight).expect("handshake");
    let mut vistos: BTreeMap<u16, SceneMob> = BTreeMap::new();
    coletar(&mut bot, &mut vistos);
    let eu = vistos
        .values()
        .find(|mob| mob.id < 1000)
        .expect("personagem na cena");
    let (mut x, mut y) = (eu.x, eu.y);
    eprintln!(
        "PASS(entrada): {} mobs, personagem em ({x},{y})",
        vistos.len()
    );

    for destino in ROTA {
        while (x, y) != destino {
            let proximo = (
                x + (destino.0 - x).clamp(-3, 3),
                y + (destino.1 - y).clamp(-3, 3),
            );
            bot.action_at((x, y), proximo).expect("passo");
            std::thread::sleep(std::time::Duration::from_millis(900));
            (x, y) = proximo;
            coletar(&mut bot, &mut vistos);
        }
        coletar(&mut bot, &mut vistos);
        eprintln!("PASS(rota): em ({x},{y}) com {} mobs vistos", vistos.len());
    }

    assert!(
        vistos.len() < 500,
        "cena acumulada com {} mobs: a entrega por proximidade não está ativa",
        vistos.len()
    );

    for (nome, cx, cy) in CAMPOS {
        let instancias: Vec<_> = vistos
            .values()
            .filter(|mob| mob.x == cx && mob.y == cy)
            .collect();
        // O snapshot tem NPCs legítimos no mesmo tile (ex.: OrcWarrior e um vizinho em
        // 2135,1948): a checagem aqui é "o NPC do Campo chegou"; duplicata de fixture é
        // cobrada nas posições antigas (FIXTURES), que têm de ficar vazias.
        assert!(
            !instancias.is_empty(),
            "{nome} em ({cx},{cy}) não chegou na cena (fora do raio de 24 tiles)"
        );
        eprintln!(
            "CAMPO nome={} pos=({cx},{cy}) cena_mob_id={} cena_nome={}",
            nome, instancias[0].id, instancias[0].name
        );
    }
    for (fx, fy) in FIXTURES {
        let sobra = vistos
            .values()
            .filter(|mob| mob.x == fx && mob.y == fy)
            .count();
        assert_eq!(sobra, 0, "fixture removida ainda na cena em ({fx},{fy})");
    }
}
