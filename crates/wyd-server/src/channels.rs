//! Canais do servidor: a lista que o cliente vê vem do Postgres (`server_channel`).
//!
//! Duas superfícies consomem esta foto:
//!
//! - `GET :80/serv00.htm` — corpo CSV que o cliente lê com `strtok`/`sscanf`
//!   (`docs/cliente/formatos.md` §4.1). O formato medido no probe é a lista de hosts dos
//!   canais `1..10`, separada por vírgula, sem espaços: `192.168.100.3,192.168.100.4`;
//! - handshake CPSock — o servidor registra quantos canais publicou e recusa subir com a
//!   tabela vazia (health sem canal mentiria sobre "servidor pronto").
//!
//! O corpo tem teto de **1.023 caracteres**: o cliente faz UMA `InternetReadFile` de `0x400`
//! bytes. Lista maior é truncada por canal inteiro (nunca corta host no meio) e o corte vai
//! para o log.

use wyd_db::ServerChannel;

/// Teto do corpo de `serv00.htm` (uma leitura de `0x400` B com NUL no fim).
pub const MAX_STATUS_BODY: usize = 1023;

/// Foto carregada no boot: canais habilitados + URL de status do grupo.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct World {
    channels: Vec<ServerChannel>,
}

impl World {
    /// Monta a foto recusando lista sem canal de jogo (nada para o cliente discar).
    pub fn new(channels: Vec<ServerChannel>) -> Result<Self, &'static str> {
        let habilitados: Vec<ServerChannel> =
            channels.into_iter().filter(|canal| canal.enabled).collect();
        if habilitados.is_empty() {
            return Err("server_channel sem canal habilitado");
        }
        if habilitados
            .iter()
            .all(|canal| canal.channel == 0 || canal.host.is_empty())
        {
            return Err("server_channel só tem linha de status: nenhum canal 1..10");
        }
        Ok(Self {
            channels: habilitados,
        })
    }

    /// Canais de jogo (`channel >= 1`), na ordem do banco.
    pub fn game_channels(&self) -> impl Iterator<Item = &ServerChannel> {
        self.channels.iter().filter(|canal| canal.channel >= 1)
    }

    /// Quantos canais de jogo a foto publica.
    #[must_use]
    pub fn game_channel_count(&self) -> usize {
        self.game_channels().count()
    }

    /// URL de status do grupo (`channel = 0`), quando configurada.
    pub fn status_url(&self) -> Option<&str> {
        self.channels
            .iter()
            .find(|canal| canal.channel == 0 && !canal.status_url.is_empty())
            .map(|canal| canal.status_url.as_str())
    }
}

/// Corpo CSV de `serv00.htm`: hosts dos canais de jogo, na ordem do banco.
///
/// Sem espaços e sem quebra de linha — o cliente lê o corpo inteiro de uma vez e o consome
/// com `strtok(",")`; CRLF entraria no último campo.
#[must_use]
pub fn render_status_csv(world: &World) -> String {
    let mut corpo = String::new();
    for canal in world.game_channels() {
        if canal.host.is_empty() {
            continue;
        }
        let extra = if corpo.is_empty() {
            canal.host.len()
        } else {
            canal.host.len() + 1
        };
        if corpo.len() + extra > MAX_STATUS_BODY {
            break;
        }
        if !corpo.is_empty() {
            corpo.push(',');
        }
        corpo.push_str(&canal.host);
    }
    corpo
}

/// URL de status publicada (atalho para log/health).
#[must_use]
pub fn status_url(world: &World) -> &str {
    world.status_url().unwrap_or("")
}

#[cfg(test)]
mod tests {
    use super::*;

    fn canal(group: i64, channel: i64, host: &str, url: &str) -> ServerChannel {
        ServerChannel {
            group_id: group,
            channel,
            host: host.to_string(),
            status_url: url.to_string(),
            enabled: true,
        }
    }

    #[test]
    fn csv_tem_so_os_hosts_de_jogo_na_ordem_do_banco() {
        let world = World::new(vec![
            canal(1, 0, "192.168.100.3", "http://192.168.100.3/serv00.htm"),
            canal(1, 1, "192.168.100.3", ""),
            canal(1, 2, "192.168.100.4", ""),
        ])
        .expect("foto válida");
        assert_eq!(render_status_csv(&world), "192.168.100.3,192.168.100.4");
        assert_eq!(status_url(&world), "http://192.168.100.3/serv00.htm");
        assert_eq!(world.game_channel_count(), 2);
    }

    #[test]
    fn canal_desabilitado_nao_entra_no_corpo() {
        let mut desligado = canal(1, 2, "192.168.100.4", "");
        desligado.enabled = false;
        let world = World::new(vec![canal(1, 1, "10.0.0.1", ""), desligado]).expect("foto válida");
        assert_eq!(render_status_csv(&world), "10.0.0.1");
    }

    #[test]
    fn corpo_respeita_o_teto_de_uma_leitura_do_cliente() {
        // 11 canais de 100 chars: 10 caberiam em 1023, o 11º não.
        let canais: Vec<ServerChannel> = (1..=10)
            .map(|i| canal(1, i, &format!("{i:0>100}"), ""))
            .collect();
        let world = World::new(canais).expect("foto válida");
        let corpo = render_status_csv(&world);
        assert!(corpo.len() <= MAX_STATUS_BODY, "corpo {} B", corpo.len());
    }

    #[test]
    fn sem_canal_de_jogo_a_foto_e_recusada() {
        assert!(World::new(vec![canal(1, 0, "10.0.0.1", "http://x/serv00.htm")]).is_err());
        assert!(World::new(Vec::new()).is_err());
    }
}
