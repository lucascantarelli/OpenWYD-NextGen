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

use crate::session::PolicyOverrides;
use wyd_db::{ChannelRole, ServerChannel};

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

/// Sobrescritas de política do canal por processo (FASE-06-3): o banco é a fonte,
/// o env sobrescreve. `WYD_SERVER_MAX_PLAYERS` fixa o teto de sessões vivas;
/// `WYD_SERVER_MAINTENANCE` (`1`/`true`/`yes`/`on` liga, qualquer outro valor não vazio
/// desliga) recusa entrada nova sem derrubar sessão viva. Ausente, vazio ou com parse que
/// falha = env ignorado (vale o banco), nunca `panic!`.
#[must_use]
pub fn policy_overrides_from_env() -> PolicyOverrides {
    let limpo = |nome: &str| {
        std::env::var(nome)
            .ok()
            .map(|texto| texto.trim().to_string())
            .filter(|texto| !texto.is_empty())
    };
    let max_players = limpo("WYD_SERVER_MAX_PLAYERS").and_then(|texto| texto.parse::<u32>().ok());
    let maintenance = limpo("WYD_SERVER_MAINTENANCE").map(|texto| {
        matches!(
            texto.to_ascii_lowercase().as_str(),
            "1" | "true" | "yes" | "on"
        )
    });
    PolicyOverrides {
        max_players,
        maintenance,
    }
}

/// Env que fixa o papel do canal por processo (`normal`/`verde`), sobrescrevendo o banco.
///
/// Existe para operação e teste (subir o mesmo build como canal novato sem migração nova);
/// valor desconhecido é **ignorado** — vale o banco —, nunca `panic!`.
pub const ROLE_ENV: &str = "WYD_SERVER_CHANNEL_ROLE";

/// Papel lido do env, se houver valor reconhecido.
#[must_use]
pub fn role_overrides_from_env() -> Option<ChannelRole> {
    std::env::var(ROLE_ENV)
        .ok()
        .map(|texto| texto.trim().to_string())
        .filter(|texto| !texto.is_empty())
        .and_then(|texto| ChannelRole::parse(&texto))
}

/// Papel efetivo do processo: env sobrescreve o banco; sem nenhum dos dois, `normal`.
#[must_use]
pub fn effective_role(db: Option<ChannelRole>, env: Option<ChannelRole>) -> ChannelRole {
    env.or(db).unwrap_or_default()
}

/// Regras que o papel do canal decide (FASE-03-10).
///
/// O papel é do **processo**, não do jogador (`docs/wyd/canais-e-cidadania.md` §3): o canal
/// verde é o canal novato do legado — PvP desligado e loja pessoal aberta. Canal normal segue
/// a regra completa, em que o mapa decide o PvP. Nada aqui olha mapa, `AttributeMap` ou
/// personagem: a política é do canal.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ChannelRules {
    role: ChannelRole,
}

impl ChannelRules {
    #[must_use]
    pub const fn new(role: ChannelRole) -> Self {
        Self { role }
    }

    #[must_use]
    pub const fn role(self) -> ChannelRole {
        self.role
    }

    /// Nome do papel como aparece no `/health`, no log e no banco.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        self.role.as_db()
    }

    /// PvP permitido neste ponto do mundo?
    ///
    /// Canal verde nunca permite, mesmo em mapa de PvP; canal normal só onde o mapa permite
    /// (`map_allows_pvp`). O `attribute` do mapa continua sendo a fonte da zona — o canal só
    /// pode **restringir**, nunca liberar além dele.
    #[must_use]
    pub const fn pvp_allowed(self, map_allows_pvp: bool) -> bool {
        match self.role {
            ChannelRole::Normal => map_allows_pvp,
            ChannelRole::Verde => false,
        }
    }

    /// Loja pessoal é permitida? Sim nos dois papéis: no canal verde ela é parte da regra do
    /// novato, no normal é serviço comum de cidade.
    #[must_use]
    pub const fn personal_shop_allowed(self) -> bool {
        true
    }
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

    #[test]
    fn papel_verde_desliga_pvp_e_mantem_loja() {
        let verde = ChannelRules::new(ChannelRole::Verde);
        assert!(!verde.pvp_allowed(true), "canal verde nunca liga PvP");
        assert!(!verde.pvp_allowed(false));
        assert!(verde.personal_shop_allowed(), "loja pessoal segue aberta");
        assert_eq!(verde.as_str(), "verde");

        let normal = ChannelRules::new(ChannelRole::Normal);
        assert!(normal.pvp_allowed(true), "canal normal segue o mapa");
        assert!(!normal.pvp_allowed(false), "mapa sem PvP continua sem");
        assert!(normal.personal_shop_allowed());
        assert_eq!(normal.role(), ChannelRole::Normal);
    }

    #[test]
    fn env_sobrescreve_o_banco_e_valor_ruim_cai_no_default() {
        // Banco "normal" + env "verde" = verde (teste/operação sem migração nova).
        assert_eq!(
            effective_role(Some(ChannelRole::Normal), Some(ChannelRole::Verde)),
            ChannelRole::Verde
        );
        // Sem env vale o banco; sem banco nem env, `normal` (nada muda).
        assert_eq!(
            effective_role(Some(ChannelRole::Verde), None),
            ChannelRole::Verde
        );
        assert_eq!(effective_role(None, None), ChannelRole::Normal);
        assert_eq!(ChannelRole::parse(" VERDE "), Some(ChannelRole::Verde));
        assert_eq!(ChannelRole::parse("novato"), None);
    }

    #[test]
    fn env_de_papel_invalido_nao_derruba_o_processo() {
        std::env::set_var(ROLE_ENV, "novato");
        assert_eq!(
            role_overrides_from_env(),
            None,
            "valor desconhecido vale o banco"
        );
        std::env::set_var(ROLE_ENV, " Verde ");
        assert_eq!(role_overrides_from_env(), Some(ChannelRole::Verde));
        std::env::set_var(ROLE_ENV, "  ");
        assert_eq!(role_overrides_from_env(), None, "env vazio vale o banco");
        std::env::remove_var(ROLE_ENV);
        assert_eq!(role_overrides_from_env(), None);
    }
}
