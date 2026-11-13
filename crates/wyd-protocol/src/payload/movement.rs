//! Movimento e ação no mundo (`_MSG_Action`).

use super::{check_len, PayloadError, WireField, WirePayload};
use crate::wire::opcode::Opcode;

wire_payload!(
    Action,
    Action,
    52,
    [
        (12, pos_x, i16),
        (14, pos_y, i16),
        (16, effect, i32),
        (20, speed, i32),
        (24, route, [u8; 24]),
        (48, target_x, i16),
        (50, target_y, i16),
    ]
);

#[cfg(test)]
mod tests {
    use super::super::round_trip;
    use super::*;

    #[test]
    fn action_round_trip_keeps_route_bytes() {
        let mut route = [0u8; 24];
        route[..3].copy_from_slice(&[1, 2, 3]);
        let action = Action {
            pos_x: 100,
            pos_y: -200,
            effect: 1,
            speed: 4,
            route,
            target_x: 150,
            target_y: -260,
        };
        let decoded = round_trip(action);
        assert_eq!(decoded, action);
        assert_eq!(decoded.pos_y, -200);
        assert_eq!(&decoded.route[..3], &[1, 2, 3]);
        assert_eq!(Action::WIRE_LEN, 52);
    }
}
