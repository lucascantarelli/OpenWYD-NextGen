//! Comércio com NPC (`_MSG_Buy`, `_MSG_Sell`, `_MSG_ShopList`, `_MSG_REQShopList`).
//!
//! Troca entre jogadores (`MSG_Trade`) tem layout UNVERIFIED e fica fora até a
//! captura real (CLIENT-02).

use super::{check_len, PayloadError, WireField, WirePayload};
use crate::wire::opcode::Opcode;

wire_payload!(
    Buy,
    Buy,
    22,
    [
        (12, target_id, u16),
        (14, target_inven_pos, i16),
        (16, my_inven_pos, i16),
        (18, coin, i32),
    ]
);

wire_payload!(
    Sell,
    Sell,
    18,
    [(12, target_id, u16), (14, my_type, i16), (16, my_pos, i16),]
);

/// `MAX_SHOPLIST` (`Basedef.h`): 27 posições por página de loja.
pub const MAX_SHOPLIST: usize = 27;

/// `STRUCT_ITEM` no fio: `sIndex` (i16) + 3 pares `(cEffect, cValue)`.
pub type WireItem = [u8; 8];

/// `_MSG_ShopList` (0x017C, S→C): catálogo aberto pelo mercador.
///
/// Layout do `Basedef.h` v7662 (`struct MSG_ShopList`): `_MSG` (12 B), `int ShopType`,
/// `STRUCT_ITEM List[MAX_SHOPLIST]` e `int Tax`, somando **236 B**. O `SendShopList` do
/// legado manda `HEADER.ID = ESCENE_FIELD` (30000), grava 0 em `ShopType` para loja de NPC
/// e o imposto da cidade em `Tax` (0 neste servidor: não há zona de guilda). Cada `List[i]`
/// é a posição `i` do catálogo, e `sIndex == 0` é posição vazia.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ShopList {
    pub shop_type: i32,
    pub items: [WireItem; MAX_SHOPLIST],
    pub tax: i32,
}

impl WirePayload for ShopList {
    const OPCODE: Opcode = Opcode::ShopList;
    const WIRE_LEN: usize = 236;

    fn decode(packet: &[u8]) -> Result<Self, PayloadError> {
        check_len(Self::WIRE_LEN, packet.len())?;
        let mut items = [[0u8; 8]; MAX_SHOPLIST];
        for (indice, item) in items.iter_mut().enumerate() {
            let inicio = 16 + indice * 8;
            item.copy_from_slice(&packet[inicio..inicio + 8]);
        }
        Ok(Self {
            shop_type: <i32 as WireField>::read(packet, 12),
            items,
            tax: <i32 as WireField>::read(packet, 232),
        })
    }

    fn encode_into(&self, packet: &mut [u8]) -> Result<(), PayloadError> {
        check_len(Self::WIRE_LEN, packet.len())?;
        <i32 as WireField>::write(self.shop_type, packet, 12);
        for (indice, item) in self.items.iter().enumerate() {
            let inicio = 16 + indice * 8;
            packet[inicio..inicio + 8].copy_from_slice(item);
        }
        <i32 as WireField>::write(self.tax, packet, 232);
        Ok(())
    }
}

// `_MSG_REQShopList` (0x027B, C→J): pedido de catálogo.
//
// Layout do `Basedef.h` v7662 (`struct MSG_REQShopList`): `_MSG` (12) +
// `unsigned short Target` + `unsigned short Unk` = **16 B**. `Target` é o `MobID`
// da criatura clicada (o mesmo espaço de `scene::CreateMob`).
wire_payload!(
    ReqShopList,
    REQShopList,
    16,
    [(12, target, u16), (14, unk, u16)]
);

#[cfg(test)]
mod tests {
    use super::super::round_trip;
    use super::*;

    #[test]
    fn buy_sell_round_trip() {
        let buy = Buy {
            target_id: 12,
            target_inven_pos: -1,
            my_inven_pos: 7,
            coin: 5000,
        };
        assert_eq!(Buy::WIRE_LEN, 22);
        assert_eq!(round_trip(buy), buy);

        let sell = Sell {
            target_id: 12,
            my_type: 2,
            my_pos: 8,
        };
        assert_eq!(Sell::WIRE_LEN, 18);
        assert_eq!(round_trip(sell), sell);
    }

    #[test]
    fn shop_list_round_trip_e_tamanho() {
        let mut items = [[0u8; 8]; MAX_SHOPLIST];
        items[0] = [0x2f, 0x00, 43, 37, 0, 0, 0, 0];
        let pacote = ShopList {
            shop_type: 0,
            items,
            tax: 0,
        };
        assert_eq!(ShopList::WIRE_LEN, 236, "12 + 4 + 27*8 + 4");
        assert_eq!(round_trip(pacote), pacote);
        // O primeiro item começa em +16 e o `Tax` fecha em +232.
        let mut cru = vec![0u8; ShopList::WIRE_LEN];
        pacote.encode_into(&mut cru).expect("corpo do ShopList");
        assert_eq!(&cru[16..24], &items[0]);
        assert_eq!(&cru[232..236], &0i32.to_le_bytes());
    }

    #[test]
    fn req_shop_list_round_trip() {
        let pedido = ReqShopList {
            target: 4784,
            unk: 0,
        };
        assert_eq!(ReqShopList::WIRE_LEN, 16);
        assert_eq!(round_trip(pedido), pedido);
    }
}
