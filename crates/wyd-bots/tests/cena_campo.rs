//! Harness vivo: imprime a cena que o servidor implantado entrega, com o `MobID` de cada
//! criatura do Campo (nomes das fixtures `teste-campo-*`).
//!
//! Gated por `WYD_LIVE_CPSOCK`; usa a conta de teste do projeto. Não altera a cena.

use wyd_bots::{Bot, BotConfig, Race};

const CAMPOS: [(&str, i32, i32); 8] = [
    ("PrimeiroTreinador", 2108, 2046),
    ("SegundoTreinador", 2112, 2046),
    ("TerceiroTreinador", 2116, 2046),
    ("ChefeTreinador", 2120, 2046),
    ("GremlinDecoy", 2115, 2042),
    ("GiantCrill", 2120, 2048),
    ("OrcSniper", 2125, 2052),
    ("OrcWarrior", 2130, 2055),
];

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
    // A cena chega na entrada; colher até estabilizar.
    let mut cena = bot.read_scene(1500).expect("cena");
    cena.extend(bot.read_scene(1500).expect("cena 2"));
    let total = cena.len();
    eprintln!("PASS(campo): {total} mobs na cena de {endereco}");
    for (nome, x, y) in CAMPOS {
        match cena
            .iter()
            .find(|mob| mob.name == nome || (mob.x == x && mob.y == y))
        {
            Some(mob) => eprintln!(
                "CAMPO nome={} fixture_pos=({x},{y}) cena_mob_id={} cena_nome={} cena_pos=({},{})",
                nome, mob.id, mob.name, mob.x, mob.y
            ),
            None => eprintln!("CAMPO nome={nome} fixture_pos=({x},{y}) AUSENTE"),
        }
    }
}
