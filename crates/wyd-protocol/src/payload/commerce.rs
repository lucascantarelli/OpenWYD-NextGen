//! Comércio com NPC (`_MSG_Buy`, `_MSG_Sell`).
//!
//! Troca entre jogadores (`MSG_Trade`) tem layout UNVERIFIED e fica fora até a
//! captura real (FASE-08-2).

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
}
