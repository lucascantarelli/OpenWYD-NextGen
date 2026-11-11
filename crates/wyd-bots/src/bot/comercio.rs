//! Submódulo de `bot` (ADR-0027).

use super::*;

impl Bot {
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
}
