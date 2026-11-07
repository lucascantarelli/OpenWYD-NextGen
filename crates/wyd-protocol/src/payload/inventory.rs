//! Inventário: uso, troca de posição e bônus (`_MSG_UseItem`, `_MSG_TradingItem`, `_MSG_ApplyBonus`).

use super::{check_len, PayloadError, WireField, WirePayload};
use crate::wire::opcode::Opcode;

wire_payload!(
    UseItem,
    UseItem,
    34,
    [
        (12, sour_type, i32),
        (16, sour_pos, i32),
        (20, dest_type, i32),
        (24, dest_pos, i32),
        (28, grid_x, u16),
        (30, grid_y, u16),
        (32, warp_id, u16),
    ]
);

wire_payload!(
    TradingItem,
    TradingItem,
    20,
    [
        (12, dest_place, u8),
        (13, dest_slot, u8),
        (14, src_place, u8),
        (15, src_slot, u8),
        (16, warp_id, i32),
    ]
);

wire_payload!(
    ApplyBonus,
    ApplyBonus,
    18,
    [
        (12, bonus_type, i16),
        (14, detail, i16),
        (16, target_id, u16),
    ]
);

#[cfg(test)]
mod tests {
    use super::super::round_trip;
    use super::*;

    #[test]
    fn use_item_and_trading_item_round_trip() {
        let use_item = UseItem {
            sour_type: 1,
            sour_pos: 2,
            dest_type: 3,
            dest_pos: 4,
            grid_x: 5,
            grid_y: 6,
            warp_id: 7,
        };
        assert_eq!(UseItem::WIRE_LEN, 34);
        assert_eq!(round_trip(use_item), use_item);

        let trading = TradingItem {
            dest_place: 1,
            dest_slot: 2,
            src_place: 3,
            src_slot: 4,
            warp_id: 99,
        };
        assert_eq!(TradingItem::WIRE_LEN, 20);
        assert_eq!(round_trip(trading), trading);
    }

    #[test]
    fn apply_bonus_round_trip() {
        let bonus = ApplyBonus {
            bonus_type: 0,
            detail: 3,
            target_id: 55,
        };
        assert_eq!(ApplyBonus::WIRE_LEN, 18);
        assert_eq!(round_trip(bonus), bonus);
    }
}
