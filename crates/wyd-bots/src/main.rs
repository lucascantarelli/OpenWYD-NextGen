//! CLI dos bots do Campo: `wyd-bots --conta X --senha Y --raca foema --npc 1,2,3,4`.
//!
//! Uma raça por processo (o roteiro é o mesmo); rodar as quatro em paralelo é uma conta por
//! bot. Sem `--npc` a campanha não tem alvo e o processo explica o que falta.

use std::process::ExitCode;

use wyd_bots::{Bot, BotConfig, Race};

fn argumento(nome: &str) -> Option<String> {
    let mut args = std::env::args().skip(1);
    while let Some(arg) = args.next() {
        if arg == nome {
            return args.next();
        }
    }
    None
}

fn raca(valor: &str) -> Option<Race> {
    Race::TODAS
        .into_iter()
        .find(|raca| raca.nome().eq_ignore_ascii_case(valor))
}

fn main() -> ExitCode {
    let Some(conta) = argumento("--conta") else {
        eprintln!("uso: wyd-bots --conta X --senha Y [--pin 2468] [--slot 0] [--raca foema] [--addr 127.0.0.1:8281] [--npc 1,2,3,4]");
        return ExitCode::from(2);
    };
    let Some(senha) = argumento("--senha") else {
        eprintln!("falta --senha");
        return ExitCode::from(2);
    };
    let nome_raca = argumento("--raca").unwrap_or_else(|| "TransKnight".to_string());
    let Some(raca) = raca(&nome_raca) else {
        eprintln!("raca desconhecida: {nome_raca}");
        return ExitCode::from(2);
    };
    let mut config = BotConfig::new(
        argumento("--addr").unwrap_or_else(|| "127.0.0.1:8281".to_string()),
        conta,
        senha,
    );
    if let Some(pin) = argumento("--pin") {
        config.pin = pin;
    }
    if let Some(slot) = argumento("--slot").and_then(|valor| valor.parse().ok()) {
        config.slot = slot;
    }
    if let Some(lista) = argumento("--npc") {
        let ids: Vec<i32> = lista
            .split(',')
            .filter_map(|valor| valor.trim().parse().ok())
            .collect();
        if ids.len() != 4 {
            eprintln!("--npc pede quatro MobID (treinadores 36/40/41/100)");
            return ExitCode::from(2);
        }
        config.treinadores = [ids[0], ids[1], ids[2], ids[3]];
    }
    if config.treinadores == [0; 4] {
        eprintln!("sem --npc: informe os quatro MobID dos treinadores na cena");
        return ExitCode::from(2);
    }

    match Bot::connect(&config, raca) {
        Ok(mut bot) => match bot.campaign(&config.treinadores) {
            Ok(paineis) => {
                for (indice, texto) in paineis.iter().enumerate() {
                    println!("{} etapa {}: {texto}", bot.race().nome(), indice + 1);
                }
                ExitCode::SUCCESS
            }
            Err(erro) => {
                eprintln!("{}: {erro}", bot.race().nome());
                ExitCode::from(1)
            }
        },
        Err(erro) => {
            // Mesma tela do caminho headless e do launcher: `tela::render` é a fonte única.
            let recusa = wyd_bots::tela::do_erro(&erro);
            eprintln!("{}: {}", raca.nome(), wyd_bots::tela::render(&recusa));
            ExitCode::from(1)
        }
    }
}
