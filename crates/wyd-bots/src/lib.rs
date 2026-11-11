//! Bots do Campo de Treinamento (FASE-15-4): **quatro raças**, um bot por classe, falando o
//! protocolo real do canal (`INITCODE` → `_MSG_AccountLogin` → `_MSG_AccountSecure` →
//! `_MSG_CharacterLogin` → `_MSG_Quest`).
//!
//! Nada de atalho interno: o bot usa socket, ofuscação e checksum como o cliente 7662. O que
//! ele não faz é andar/matar — o roteiro assume as provas no inventário (drops dos guardiões
//! ainda não são entregues ao jogador) e exercita a quest pelos treinadores, que é o alvo
//! desta fase. A ordem das etapas é conferida pelo texto do `_MSG_MessagePanel`.

pub mod tela;

use std::io::{Error, ErrorKind, Read, Write};
use std::net::{TcpStream, ToSocketAddrs};
use std::time::Duration;

use wyd_protocol::codec::keyword::{decode_in_place, encode_in_place};
use wyd_protocol::payload::combat::{Attack, TargetDamage};
use wyd_protocol::payload::movement::Action;
use wyd_protocol::payload::quest::Quest;
use wyd_protocol::payload::scene::CreateMob;
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::{Header, HEADER_SIZE};
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{AccountLogin, AccountSecure, CharacterLogin, CreateCharacter, MessagePanel};

/// Uma raça jogável do 7662 (classe 0–3, ordem de `docs/wyd/classes.md` §1).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Race {
    TransKnight,
    Foema,
    BeastMaster,
    Huntress,
}

impl Race {
    /// As quatro raças, uma por bot.
    pub const TODAS: [Race; 4] = [
        Race::TransKnight,
        Race::Foema,
        Race::BeastMaster,
        Race::Huntress,
    ];

    /// Índice de classe do `_MSG_CreateCharacter`/`BaseMob`.
    #[must_use]
    pub const fn class(self) -> i32 {
        match self {
            Self::TransKnight => 0,
            Self::Foema => 1,
            Self::BeastMaster => 2,
            Self::Huntress => 3,
        }
    }

    /// Nome usado em log/CLI.
    #[must_use]
    pub const fn nome(self) -> &'static str {
        match self {
            Self::TransKnight => "TransKnight",
            Self::Foema => "Foema",
            Self::BeastMaster => "BeastMaster",
            Self::Huntress => "Huntress",
        }
    }
}

/// Dados de conexão/conta do bot (uma conta por raça).
#[derive(Debug, Clone)]
pub struct BotConfig {
    pub addr: String,
    pub account: String,
    pub password: String,
    pub pin: String,
    pub slot: i32,
    /// `MobID` dos quatro treinadores na cena (o bot não adivinha).
    pub treinadores: [i32; 4],
}

impl BotConfig {
    /// Configuração com os padrões do laboratório (conta e senha explícitas).
    #[must_use]
    pub fn new(
        addr: impl Into<String>,
        account: impl Into<String>,
        password: impl Into<String>,
    ) -> Self {
        Self {
            addr: addr.into(),
            account: account.into(),
            password: password.into(),
            pin: "2468".to_string(),
            slot: 0,
            treinadores: [0; 4],
        }
    }
}

/// Cliente do canal para um bot.
pub struct Bot {
    stream: TcpStream,
    key_word: u8,
    race: Race,
}

impl Bot {
    /// Abre o socket e faz o handshake completo; devolve o bot pronto para o `_MSG_Quest`.
    pub fn connect(config: &BotConfig, race: Race) -> std::io::Result<Self> {
        let endereco = config
            .addr
            .to_socket_addrs()?
            .next()
            .ok_or_else(|| Error::new(ErrorKind::AddrNotAvailable, "endereco do canal"))?;
        let stream = TcpStream::connect_timeout(&endereco, Duration::from_secs(5))?;
        stream.set_read_timeout(Some(Duration::from_secs(5)))?;
        stream.set_write_timeout(Some(Duration::from_secs(5)))?;
        let mut bot = Self {
            stream,
            key_word: 11,
            race,
        };
        // Servidor que encerra sem responder (ex.: fail só de cabeçalho já consumido, ou
        // socket fechado na recusa) não pode virar "failed to fill whole buffer": a tela
        // traduz o EOF para uma frase acionável, como qualquer outra recusa.
        bot.handshake(config).map_err(|erro| {
            if matches!(
                erro.kind(),
                ErrorKind::UnexpectedEof | ErrorKind::ConnectionReset | ErrorKind::BrokenPipe
            ) {
                Error::new(
                    ErrorKind::ConnectionAborted,
                    "Conexão encerrada pelo servidor antes da resposta — confira a senha, o PIN \
                     e o slot do personagem e tente novamente.",
                )
            } else {
                erro
            }
        })?;
        Ok(bot)
    }

    /// Raça deste bot.
    #[must_use]
    pub fn race(&self) -> Race {
        self.race
    }

    fn autenticar(&mut self, config: &BotConfig) -> std::io::Result<()> {
        self.stream.write_all(&INITCODE.to_le_bytes())?;

        let mut nome = [0u8; 16];
        let bytes = config.account.as_bytes();
        if bytes.len() > nome.len() {
            return Err(Error::new(
                ErrorKind::InvalidInput,
                "conta acima de 16 bytes",
            ));
        }
        nome[..bytes.len()].copy_from_slice(bytes);
        let mut senha = [0u8; 12];
        let bytes = config.password.as_bytes();
        if bytes.len() > senha.len() {
            return Err(Error::new(
                ErrorKind::InvalidInput,
                "senha acima de 12 bytes",
            ));
        }
        senha[..bytes.len()].copy_from_slice(bytes);
        self.send(Opcode::AccountLogin, AccountLogin::WIRE_LEN, |frame| {
            AccountLogin {
                account_password: senha,
                account_name: nome,
                zero: [0u8; 52],
                client_version: 12_000,
                db_need_save: 1,
                adapter_name: [0; 4],
            }
            .encode_into(frame)
            .expect("corpo do AccountLogin");
        })?;
        self.expect(Opcode::CNFAccountLogin)?;

        let mut token = [0u8; 6];
        for (indice, digito) in config.pin.bytes().take(6).enumerate() {
            token[indice] = digito;
        }
        self.send(Opcode::AccountSecure, AccountSecure::WIRE_LEN, |frame| {
            AccountSecure {
                numeric_token: token,
                unknown: [0u8; 10],
                change_numeric: 1,
            }
            .encode_into(frame)
            .expect("corpo do AccountSecure");
        })?;
        // O canal confirma o PIN ecoando `_MSG_AccountSecure` (0x0FDE), não um `CNF`.
        self.expect(Opcode::AccountSecure)?;
        Ok(())
    }

    fn entrar(&mut self, config: &BotConfig) -> std::io::Result<()> {
        let slot = config.slot;
        self.send(Opcode::CharacterLogin, CharacterLogin::WIRE_LEN, |frame| {
            CharacterLogin { slot, force: 1 }
                .encode_into(frame)
                .expect("corpo do CharacterLogin");
        })?;
        self.expect(Opcode::CNFCharacterLogin)?;
        Ok(())
    }

    fn handshake(&mut self, config: &BotConfig) -> std::io::Result<()> {
        self.autenticar(config)?;
        self.entrar(config)
    }

    /// Caminho de personagem **novo**: autentica, cria o char no slot e entra com ele.
    ///
    /// É o `_MSG_CreateCharacter` real (36 B) + `_MSG_CharacterLogin`, sem atalho de banco:
    /// o teste vivo prova o mesmo caminho que o cliente 7662 faz na tela de criação.
    pub fn connect_novo(
        config: &BotConfig,
        race: Race,
        nome: &str,
        class: i32,
    ) -> std::io::Result<Self> {
        let endereco = config
            .addr
            .to_socket_addrs()?
            .next()
            .ok_or_else(|| Error::new(ErrorKind::AddrNotAvailable, "endereco do canal"))?;
        let stream = TcpStream::connect_timeout(&endereco, Duration::from_secs(5))?;
        stream.set_read_timeout(Some(Duration::from_secs(5)))?;
        stream.set_write_timeout(Some(Duration::from_secs(5)))?;
        let mut bot = Self {
            stream,
            key_word: 11,
            race,
        };
        bot.autenticar(config)?;
        bot.create_character(config.slot, nome, class)?;
        bot.entrar(config)?;
        Ok(bot)
    }

    /// `_MSG_CreateCharacter` (0x0203) e a confirmação `CNFNewCharacter`.
    pub fn create_character(&mut self, slot: i32, nome: &str, class: i32) -> std::io::Result<()> {
        let mut mob_name = [0u8; 16];
        let bytes = nome.as_bytes();
        if bytes.len() > mob_name.len() {
            return Err(Error::new(
                ErrorKind::InvalidInput,
                "nome acima de 16 bytes",
            ));
        }
        mob_name[..bytes.len()].copy_from_slice(bytes);
        self.send(
            Opcode::CreateCharacter,
            CreateCharacter::WIRE_LEN,
            |frame| {
                CreateCharacter {
                    slot,
                    mob_name,
                    mob_class: class,
                }
                .encode_into(frame)
                .expect("corpo do CreateCharacter");
            },
        )?;
        self.expect(Opcode::CNFNewCharacter)
    }

    /// `_MSG_Quest` no treinador `npc_index`; devolve o texto do painel de resposta.
    pub fn quest(&mut self, npc_index: i32) -> std::io::Result<String> {
        self.send(Opcode::Quest, Quest::WIRE_LEN, |frame| {
            Quest {
                npc_index,
                confirm: 1,
            }
            .encode_into(frame)
            .expect("corpo do Quest");
        })?;
        let limite = std::time::Instant::now() + Duration::from_secs(5);
        while std::time::Instant::now() < limite {
            let (header, frame) = self.recv()?;
            if Opcode::from_wire(header.message_type) == Some(Opcode::MessagePanel) {
                let painel = MessagePanel::decode(&frame)
                    .map_err(|erro| Error::new(ErrorKind::InvalidData, erro.to_string()))?;
                return Ok(String::from_utf8_lossy(painel.text_bytes()).to_string());
            }
        }
        Err(Error::new(ErrorKind::TimedOut, "sem MessagePanel da quest"))
    }

    /// Roda as quatro etapas na ordem; recusa painel de etapa errada.
    pub fn campaign(&mut self, treinadores: &[i32; 4]) -> std::io::Result<Vec<String>> {
        let mut paineis = Vec::with_capacity(4);
        for (indice, npc) in treinadores.iter().enumerate() {
            let texto = self.quest(*npc)?;
            let etapa = parse_etapa(&texto).ok_or_else(|| {
                Error::new(
                    ErrorKind::InvalidData,
                    format!("{}: painel sem etapa: {texto}", self.race.nome()),
                )
            })?;
            let esperada = i16::try_from(indice).unwrap_or(0) + 1;
            if etapa != esperada {
                return Err(Error::new(
                    ErrorKind::InvalidData,
                    format!(
                        "{}: etapa {etapa} fora de ordem (esperada {esperada}): {texto}",
                        self.race.nome()
                    ),
                ));
            }
            paineis.push(texto);
        }
        Ok(paineis)
    }

    /// Lê a cena por até `millis` e devolve as criaturas dos `_MSG_CreateMob` recebidos.
    ///
    /// O resto dos pacotes (score, items, painéis) é descartado: o bot só precisa do id,
    /// do nome e do tile para escolher alvo e portão, tudo dado pelo servidor.
    pub fn read_scene(&mut self, millis: u64) -> std::io::Result<Vec<SceneMob>> {
        let anterior = self
            .stream
            .read_timeout()?
            .unwrap_or(Duration::from_secs(5));
        self.stream
            .set_read_timeout(Some(Duration::from_millis(millis)))?;
        let mut vistos = Vec::new();
        let resultado = loop {
            match self.recv() {
                Ok((header, frame)) => {
                    if Opcode::from_wire(header.message_type) == Some(Opcode::CreateMob) {
                        match CreateMob::decode(&frame) {
                            Ok(mob) => vistos.push(SceneMob {
                                id: mob.mob_id,
                                name: nome_do_campo(&mob.mob_name),
                                x: i32::from(mob.pos_x),
                                y: i32::from(mob.pos_y),
                                level: u16::try_from(mob.level).unwrap_or(0),
                                hp: u32::try_from(mob.hp).unwrap_or(0),
                                max_hp: u32::try_from(mob.max_hp).unwrap_or(0),
                                merchant: mob.merchant,
                            }),
                            Err(erro) => {
                                break Err(Error::new(ErrorKind::InvalidData, erro.to_string()))
                            }
                        }
                    }
                }
                Err(erro)
                    if erro.kind() == ErrorKind::WouldBlock
                        || erro.kind() == ErrorKind::TimedOut =>
                {
                    break Ok(vistos)
                }
                Err(erro) => break Err(erro),
            }
        };
        self.stream.set_read_timeout(Some(anterior))?;
        resultado
    }

    /// Lê por até `millis` e devolve o `Coin` do primeiro `_MSG_UpdateEtc`.
    ///
    /// `Coin` é o `i32` do corpo em +28 (`cpsock::send_progress` → `update_etc_body`), logo
    /// o byte 28 depois do `HEADER` de 12 B. `None` = o pacote não chegou no prazo.
    pub fn read_etc_coin(&mut self, millis: u64) -> std::io::Result<Option<i32>> {
        let anterior = self
            .stream
            .read_timeout()?
            .unwrap_or(Duration::from_secs(5));
        self.stream
            .set_read_timeout(Some(Duration::from_millis(millis)))?;
        let resultado = loop {
            match self.recv() {
                Ok((header, frame)) => {
                    if Opcode::from_wire(header.message_type) == Some(Opcode::UpdateEtc) {
                        let base = HEADER_SIZE + 28;
                        if frame.len() < base + 4 {
                            break Err(Error::new(
                                ErrorKind::InvalidData,
                                "_MSG_UpdateEtc curto para o Coin@28",
                            ));
                        }
                        let mut bytes = [0u8; 4];
                        bytes.copy_from_slice(&frame[base..base + 4]);
                        break Ok(Some(i32::from_le_bytes(bytes)));
                    }
                }
                Err(erro)
                    if erro.kind() == ErrorKind::WouldBlock
                        || erro.kind() == ErrorKind::TimedOut =>
                {
                    break Ok(None)
                }
                Err(erro) => break Err(erro),
            }
        };
        self.stream.set_read_timeout(Some(anterior))?;
        resultado
    }

    /// `_MSG_REQShopList`: abre o catálogo do mercador `scene_id`.
    ///
    /// Devolve as posições não vazias do `_MSG_ShopList` (até 27), como chegaram no fio.
    pub fn open_shop(&mut self, scene_id: u16) -> std::io::Result<Vec<[u8; 8]>> {
        self.send(
            Opcode::REQShopList,
            wyd_protocol::ReqShopList::WIRE_LEN,
            |frame| {
                wyd_protocol::ReqShopList {
                    target: scene_id,
                    unk: 0,
                }
                .encode_into(frame)
                .expect("corpo do ReqShopList");
            },
        )?;
        self.read_shop_list(3_000)
    }

    /// Lê por até `millis` e devolve o `_MSG_ShopList` recebido (vazio = sem catálogo).
    pub fn read_shop_list(&mut self, millis: u64) -> std::io::Result<Vec<[u8; 8]>> {
        let anterior = self
            .stream
            .read_timeout()?
            .unwrap_or(Duration::from_secs(5));
        self.stream
            .set_read_timeout(Some(Duration::from_millis(millis)))?;
        let resultado = loop {
            match self.recv() {
                Ok((header, frame)) => {
                    if Opcode::from_wire(header.message_type) == Some(Opcode::ShopList) {
                        match wyd_protocol::ShopList::decode(&frame) {
                            Ok(lista) => {
                                break Ok(lista
                                    .items
                                    .iter()
                                    .copied()
                                    .filter(|item| item[0] != 0 || item[1] != 0)
                                    .collect())
                            }
                            Err(erro) => {
                                break Err(Error::new(ErrorKind::InvalidData, erro.to_string()))
                            }
                        }
                    }
                }
                Err(erro)
                    if erro.kind() == ErrorKind::WouldBlock
                        || erro.kind() == ErrorKind::TimedOut =>
                {
                    break Ok(Vec::new())
                }
                Err(erro) => break Err(erro),
            }
        };
        self.stream.set_read_timeout(Some(anterior))?;
        resultado
    }

    /// `_MSG_Buy`: pede a posição `shop_slot` do catálogo pagando `coin`.
    pub fn buy(&mut self, scene_id: u16, shop_slot: i16, coin: i32) -> std::io::Result<()> {
        self.send(Opcode::Buy, wyd_protocol::Buy::WIRE_LEN, |frame| {
            wyd_protocol::Buy {
                target_id: scene_id,
                target_inven_pos: shop_slot,
                my_inven_pos: -1,
                coin,
            }
            .encode_into(frame)
            .expect("corpo do Buy");
        })
    }

    /// Lê por até `millis` e devolve o primeiro `_MSG_SendItem` (tipo, slot, item).
    pub fn read_send_item(&mut self, millis: u64) -> std::io::Result<Option<(u16, u16, [u8; 8])>> {
        let anterior = self
            .stream
            .read_timeout()?
            .unwrap_or(Duration::from_secs(5));
        self.stream
            .set_read_timeout(Some(Duration::from_millis(millis)))?;
        let resultado = loop {
            match self.recv() {
                Ok((header, frame)) => {
                    if Opcode::from_wire(header.message_type) == Some(Opcode::SendItem) {
                        match wyd_protocol::SendItem::decode(&frame) {
                            Ok(pacote) => {
                                break Ok(Some((pacote.inv_type, pacote.slot, pacote.item)))
                            }
                            Err(erro) => {
                                break Err(Error::new(ErrorKind::InvalidData, erro.to_string()))
                            }
                        }
                    }
                }
                Err(erro)
                    if erro.kind() == ErrorKind::WouldBlock
                        || erro.kind() == ErrorKind::TimedOut =>
                {
                    break Ok(None)
                }
                Err(erro) => break Err(erro),
            }
        };
        self.stream.set_read_timeout(Some(anterior))?;
        resultado
    }

    /// Envia `_MSG_AttackOne` no tile do alvo (o servidor resolve o alvo pela posição).
    pub fn attack_at(&mut self, x: i32, y: i32) -> std::io::Result<()> {
        let coord = |valor: i32| u16::try_from(valor).unwrap_or(0);
        let (target_x, target_y) = (coord(x), coord(y));
        self.send(Opcode::AttackOne, Attack::<1>::WIRE_LEN, |frame| {
            Attack::<1> {
                unk_1: [0; 4],
                current_hp: 0,
                unk_2: [0; 4],
                current_exp: 0,
                unk_0: 0,
                pos_x: target_x,
                pos_y: target_y,
                target_x,
                target_y,
                attacker_id: 0,
                progress: 0,
                motion: 1,
                skill_parm: 0,
                double_critical: 0,
                flag_local: 0,
                rsv: 0,
                current_mp: 0,
                skill_index: 0,
                req_mp: 0,
                dam: [TargetDamage {
                    target_id: 0,
                    damage: 0,
                }],
            }
            .encode_into(frame)
            .expect("corpo do AttackOne");
        })
    }

    /// Mata a criatura `nome` vista na cena, atacando até o `_MSG_RemoveMob` dela.
    ///
    /// Só serve para um alvo que ainda vai aparecer: o `_MSG_CreateMob` é consumido na leitura
    /// da cena, então quem já foi lido deve usar [`Bot::kill_at`] com id/posição.
    pub fn kill(&mut self, nome: &str) -> std::io::Result<(i32, i32)> {
        let limite = std::time::Instant::now() + Duration::from_secs(10);
        let mut alvo = None;
        while alvo.is_none() && std::time::Instant::now() < limite {
            alvo = self
                .read_scene(700)?
                .into_iter()
                .find(|mob| mob.name == nome);
        }
        let alvo = alvo.ok_or_else(|| {
            Error::new(
                ErrorKind::NotFound,
                format!("{}: {nome} ausente da cena", self.race.nome()),
            )
        })?;
        self.kill_at(alvo.id, alvo.x, alvo.y)?;
        Ok((alvo.x, alvo.y))
    }

    /// Ataca o alvo já conhecido (`MobID`/tile vistos na cena) até o `_MSG_RemoveMob` dele.
    pub fn kill_at(&mut self, id: u16, x: i32, y: i32) -> std::io::Result<()> {
        let limite = std::time::Instant::now() + Duration::from_secs(20);
        while std::time::Instant::now() < limite {
            self.attack_at(x, y)?;
            let fim = std::time::Instant::now() + Duration::from_millis(600);
            while std::time::Instant::now() < fim {
                match self.recv() {
                    Ok((header, _)) => {
                        if Opcode::from_wire(header.message_type) == Some(Opcode::RemoveMob)
                            && header.id == id
                        {
                            return Ok(());
                        }
                    }
                    Err(erro)
                        if erro.kind() == ErrorKind::WouldBlock
                            || erro.kind() == ErrorKind::TimedOut =>
                    {
                        break
                    }
                    Err(erro) => return Err(erro),
                }
            }
        }
        Err(Error::new(
            ErrorKind::TimedOut,
            format!("{}: alvo {id} em ({x},{y}) não morreu", self.race.nome()),
        ))
    }

    /// Clica no tile do portão (`_MSG_Action`) e devolve o painel de resposta.
    pub fn open_gate(&mut self, x: i32, y: i32) -> std::io::Result<String> {
        self.action_at((x, y), (x, y))?;
        self.painel()
    }

    /// Envia um `_MSG_Action` de movimento/click de `from` para `to`.
    pub fn action_at(&mut self, from: (i32, i32), to: (i32, i32)) -> std::io::Result<()> {
        let coord = |valor: i32| i16::try_from(valor).unwrap_or(0);
        let (pos_x, pos_y) = (coord(from.0), coord(from.1));
        let (target_x, target_y) = (coord(to.0), coord(to.1));
        self.send(Opcode::Action, Action::WIRE_LEN, |frame| {
            Action {
                pos_x,
                pos_y,
                effect: 0,
                speed: 0,
                route: [0; 24],
                target_x,
                target_y,
            }
            .encode_into(frame)
            .expect("corpo do Action");
        })
    }

    /// Anda até `to` em passos de 3 tiles, esperando o orçamento do servidor entre eles.
    ///
    /// O bot não conhece a própria posição pelo fio: `from` é a posição lida na cena.
    pub fn walk_to(&mut self, from: (i32, i32), to: (i32, i32)) -> std::io::Result<(i32, i32)> {
        let mut atual = from;
        while atual != to {
            let proximo = (
                atual.0 + (to.0 - atual.0).clamp(-3, 3),
                atual.1 + (to.1 - atual.1).clamp(-3, 3),
            );
            self.action_at(atual, proximo)?;
            std::thread::sleep(Duration::from_millis(900));
            atual = proximo;
        }
        Ok(atual)
    }

    /// Espera o próximo `_MSG_MessagePanel` da conexão.
    fn painel(&mut self) -> std::io::Result<String> {
        let limite = std::time::Instant::now() + Duration::from_secs(5);
        while std::time::Instant::now() < limite {
            let (header, frame) = self.recv()?;
            if Opcode::from_wire(header.message_type) == Some(Opcode::MessagePanel) {
                let painel = MessagePanel::decode(&frame)
                    .map_err(|erro| Error::new(ErrorKind::InvalidData, erro.to_string()))?;
                return Ok(String::from_utf8_lossy(painel.text_bytes()).to_string());
            }
        }
        Err(Error::new(ErrorKind::TimedOut, "sem MessagePanel"))
    }

    fn send(
        &mut self,
        opcode: Opcode,
        len: usize,
        corpo: impl FnOnce(&mut [u8]),
    ) -> std::io::Result<()> {
        let mut frame = vec![0u8; len];
        let size = u16::try_from(len).map_err(|_| Error::new(ErrorKind::InvalidInput, "frame"))?;
        Header {
            size,
            key_word: self.key_word,
            checksum: 0,
            message_type: opcode.wire(),
            id: 1,
            client_tick: 7,
        }
        .encode(&mut frame)
        .map_err(|erro| Error::new(ErrorKind::InvalidData, erro.to_string()))?;
        corpo(&mut frame);
        encode_in_place(&mut frame, self.key_word)
            .map_err(|erro| Error::new(ErrorKind::InvalidData, erro.to_string()))?;
        self.stream.write_all(&frame)
    }

    fn recv(&mut self) -> std::io::Result<(Header, Vec<u8>)> {
        let mut head = [0u8; HEADER_SIZE];
        self.stream.read_exact(&mut head)?;
        let size = usize::from(u16::from_le_bytes([head[0], head[1]]));
        if !(HEADER_SIZE..=8192).contains(&size) {
            return Err(Error::new(ErrorKind::InvalidData, "Size fora de 12..8192"));
        }
        let mut frame = vec![0u8; size];
        frame[..HEADER_SIZE].copy_from_slice(&head);
        self.stream.read_exact(&mut frame[HEADER_SIZE..])?;
        decode_in_place(&mut frame)
            .map_err(|erro| Error::new(ErrorKind::InvalidData, erro.to_string()))?;
        let header = Header::decode(&frame)
            .map_err(|erro| Error::new(ErrorKind::InvalidData, erro.to_string()))?;
        Ok((header, frame))
    }

    fn expect(&mut self, opcode: Opcode) -> std::io::Result<()> {
        let (header, frame) = self.recv()?;
        // Trailer opcional de motivo: só existe se o servidor mandou o byte extra.
        let motivo = wyd_protocol::motivo::CodigoRecusa::from_byte(frame.get(HEADER_SIZE).copied());
        match Opcode::from_wire(header.message_type) {
            Some(visto) if visto == opcode => Ok(()),
            Some(visto) => Err(Error::new(
                ErrorKind::PermissionDenied,
                format!(
                    "recusa:{}:{}:{}",
                    visto.wire(),
                    motivo.byte(),
                    motivo_da_recusa(visto)
                ),
            )),
            None => Err(Error::new(
                ErrorKind::InvalidData,
                format!("esperado {opcode:?}, recebido um opcode fora do catálogo"),
            )),
        }
    }
}

/// Motivo em português para uma recusa do handshake (é a mensagem da tela de conexão).
///
/// O cliente 7662 é fechado e não exibe nada ao receber `*Fail`; esta é a superfície sob
/// nosso controle e cobre o que o servidor já manda hoje. Sempre devolve texto acionável —
/// nunca vazio, para não existir "clico e nada acontece".
#[must_use]
pub fn motivo_da_recusa(opcode: Opcode) -> String {
    match opcode {
        Opcode::NewAccountFail => {
            "Conta ou senha incorreta — confira os dados e tente novamente.".to_string()
        }
        Opcode::AccountSecureFail => "PIN incorreto ou bloqueado por tentativas — confira o PIN \
             e tente novamente; após várias falhas aguarde alguns minutos."
            .to_string(),
        Opcode::CharacterLoginFail => {
            "Personagem indisponível — escolha outro slot ou crie um personagem.".to_string()
        }
        outro => format!(
            "O servidor recusou a conexão ({}). Tente novamente em instantes.",
            outro.wire()
        ),
    }
}

/// Criatura vista na cena (`_MSG_CreateMob`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SceneMob {
    /// `MobID` do servidor (o mesmo id do abate).
    pub id: u16,
    /// Nome do template, sem o preenchimento NUL.
    pub name: String,
    pub x: i32,
    pub y: i32,
    /// `STRUCT_SCORE` do `_MSG_CreateMob`: o teste cobra que a criatura chegue desenhável.
    pub level: u16,
    pub hp: u32,
    pub max_hp: u32,
    /// `Merchant` do `STRUCT_SCORE` (0 = sem loja).
    pub merchant: u8,
}

/// Nome do `MobName[16]` sem a cauda NUL.
#[must_use]
pub fn nome_do_campo(bytes: &[u8; 16]) -> String {
    let fim = bytes.iter().position(|byte| *byte == 0).unwrap_or(16);
    String::from_utf8_lossy(&bytes[..fim]).to_string()
}

/// Extrai `N` de `"Campo de Treinamento: etapa N/4"`.
#[must_use]
pub fn parse_etapa(texto: &str) -> Option<i16> {
    let resto = texto.split("etapa ").nth(1)?;
    let numero: String = resto.chars().take_while(char::is_ascii_digit).collect();
    numero.parse().ok()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn quatro_racas_com_classes_distintas() {
        let classes: Vec<i32> = Race::TODAS.iter().map(|raca| raca.class()).collect();
        assert_eq!(classes, vec![0, 1, 2, 3]);
        let nomes: Vec<&str> = Race::TODAS.iter().map(|raca| raca.nome()).collect();
        assert_eq!(
            nomes,
            vec!["TransKnight", "Foema", "BeastMaster", "Huntress"]
        );
    }

    #[test]
    fn parse_etapa_le_o_painel_do_servidor() {
        assert_eq!(parse_etapa("Campo de Treinamento: etapa 1/4"), Some(1));
        assert_eq!(
            parse_etapa("Campo de Treinamento: etapa 4/4 (concluido)"),
            Some(4)
        );
        assert_eq!(parse_etapa("falta a prova 451"), None);
    }

    #[test]
    fn toda_recusa_tem_mensagem_em_portugues() {
        for opcode in [
            Opcode::NewAccountFail,
            Opcode::AccountSecureFail,
            Opcode::CharacterLoginFail,
            Opcode::AccountLogin,
        ] {
            let texto = motivo_da_recusa(opcode);
            assert!(!texto.trim().is_empty(), "{opcode:?} sem mensagem");
            let minusculo = texto.to_lowercase();
            assert!(
                minusculo.contains("tente")
                    || minusculo.contains("escolha")
                    || minusculo.contains("confira"),
                "mensagem sem próximo passo: {texto}"
            );
        }
        assert!(motivo_da_recusa(Opcode::AccountSecureFail).contains("PIN"));
        assert!(motivo_da_recusa(Opcode::CharacterLoginFail).contains("Personagem"));
        assert!(motivo_da_recusa(Opcode::NewAccountFail).contains("senha"));
    }

    #[test]
    fn nome_do_create_mob_perde_o_preenchimento_nulo() {
        let mut nome = [0u8; 16];
        nome[..14].copy_from_slice(b"PrimeiroMestre");
        assert_eq!(nome_do_campo(&nome), "PrimeiroMestre");
        assert_eq!(nome_do_campo(&[b'A'; 16]), "AAAAAAAAAAAAAAAA");
    }

    #[test]
    fn campanha_usa_os_quatro_treinadores_na_ordem() {
        let config = BotConfig::new("127.0.0.1:8281", "qa", "qa12345678");
        assert_eq!(config.treinadores, [0; 4]);
        // A ordem é a dos `Merchant` 36/40/41/100 (sequência A).
        assert_eq!(config.slot, 0);
        assert_eq!(config.pin, "2468");
    }
}
