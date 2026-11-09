//! Canais do servidor: a lista que o cliente vê vem do Postgres (`server_channel`).
//!
//! Duas superfícies consomem esta foto:
//!
//! - `GET :80/serv00.htm` — corpo com **um número por canal de jogo** (jogadores online
//!   naquele canal), um por linha: `0\n0\n`. O cliente lê o corpo com
//!   `sscanf(corpo, "%d", …)` em 10 destinos inicializados em `-1` (`WYD.exe`
//!   0x46d9d6–0x46da54, ver `docs/cliente/formatos.md` §4.1): cada número é o contador da
//!   linha do canal e o destino que continua `-1` é desenhado como `Inexistente`. O
//!   separador tem de ser espaço/quebra de linha (vírgula para o `sscanf`). Os **hosts**
//!   não vêm daqui — vêm da célula do `serverlist.bin`;
//! - handshake CPSock — o servidor registra quantos canais publicou e recusa subir com a
//!   tabela vazia (health sem canal mentiria sobre "servidor pronto").
//!
//! O corpo tem teto de **1.023 caracteres**: o cliente faz UMA `InternetReadFile` de `0x400`
//! bytes. Lista maior é truncada por canal inteiro e o corte vai para o log.

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

/// Corpo de `serv00.htm`: **um número por canal de jogo**, na ordem do banco, separados
/// por quebra de linha — a contagem de sessões vivas **daquele canal** (`character_session`
/// filtrada por grupo e canal), nunca um total do grupo: um canal não enxerga a presença do
/// outro (`docs/wyd/canais-e-cidadania.md` §5.2).
///
/// O separador não é vírgula: o cliente faz `sscanf(corpo, "%d")` (10 destinos) e o
/// `sscanf` pula espaços/quebras de linha entre as conversões, mas **para no primeiro
/// caractere que não seja número nem espaço** — com vírgula só o primeiro canal é lido e
/// os demais ficam em `-1` (`Inexistente`). É o que o servidor de referência serve
/// (`w2pp-OpenWYD`, `Release/Common/serv00.htm`): um valor por linha.
#[must_use]
pub fn render_status_body(world: &World, online: &[u64]) -> String {
    let mut corpo = String::new();
    for (indice, _) in world.game_channels().enumerate() {
        // Presença do canal_i, na mesma ordem de `game_channels()`; canal sem leitura é 0.
        let valor = online.get(indice).copied().unwrap_or(0).to_string();
        // Valor + '\n': o `sscanf` do cliente pula espaço/quebra de linha entre números.
        if corpo.len() + valor.len() + 1 > MAX_STATUS_BODY {
            break;
        }
        corpo.push_str(&valor);
        corpo.push('\n');
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
    fn corpo_tem_um_numero_por_canal_de_jogo() {
        let world = World::new(vec![
            canal(1, 0, "192.168.100.3", "http://192.168.100.3/serv00.htm"),
            canal(1, 1, "192.168.100.3", ""),
            canal(1, 2, "192.168.100.4", ""),
        ])
        .expect("foto válida");
        assert_eq!(render_status_body(&world, &[0, 0]), "0\n0\n");
        assert_eq!(status_url(&world), "http://192.168.100.3/serv00.htm");
        assert_eq!(world.game_channel_count(), 2);
    }

    #[test]
    fn canal_desabilitado_nao_entra_no_corpo() {
        let mut desligado = canal(1, 2, "192.168.100.4", "");
        desligado.enabled = false;
        let world = World::new(vec![canal(1, 1, "10.0.0.1", ""), desligado]).expect("foto válida");
        assert_eq!(render_status_body(&world, &[0]), "0\n");
    }

    #[test]
    fn corpo_respeita_o_teto_de_uma_leitura_do_cliente() {
        let canais: Vec<ServerChannel> = (1..=10).map(|i| canal(1, i, "10.0.0.1", "")).collect();
        let world = World::new(canais).expect("foto válida");
        let corpo = render_status_body(&world, &[0; 10]);
        assert!(corpo.len() <= MAX_STATUS_BODY, "corpo {} B", corpo.len());
        assert_eq!(corpo, "0\n0\n0\n0\n0\n0\n0\n0\n0\n0\n");
    }

    #[test]
    fn contagem_de_um_canal_nao_aparece_no_outro() {
        // A presença é por canal: o número da linha 1 é do canal 1 e o da linha 2 é do canal 2.
        // Um total do grupo nas duas linhas (vazamento de presença) quebraria este teste.
        let world = World::new(vec![
            canal(1, 0, "192.168.100.3", "http://192.168.100.3/serv00.htm"),
            canal(1, 1, "192.168.100.3", ""),
            canal(1, 2, "192.168.100.4", ""),
        ])
        .expect("foto válida");
        assert_eq!(render_status_body(&world, &[0, 3]), "0\n3\n");
        assert_eq!(render_status_body(&world, &[2, 0]), "2\n0\n");
    }

    #[test]
    fn sem_canal_de_jogo_a_foto_e_recusada() {
        assert!(World::new(vec![canal(1, 0, "10.0.0.1", "http://x/serv00.htm")]).is_err());
        assert!(World::new(Vec::new()).is_err());
    }
}
