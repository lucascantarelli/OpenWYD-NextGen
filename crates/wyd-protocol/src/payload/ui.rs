//! Interface: painel de mensagens e clima (`_MSG_MessagePanel`, `_MSG_UpdateWeather`).

use super::{check_len, nul_trimmed, PayloadError, WireField, WirePayload};
use crate::wire::opcode::Opcode;

wire_payload!(MessagePanel, MessagePanel, 140, [(12, text, [u8; 128])]);

impl MessagePanel {
    /// Texto do painel até o primeiro `NUL`.
    #[must_use]
    pub fn text_bytes(&self) -> &[u8] {
        nul_trimmed(&self.text)
    }
}

wire_payload!(
    UpdateWeather,
    UpdateWeather,
    16,
    [(12, current_weather, i32)]
);

#[cfg(test)]
mod tests {
    use super::super::round_trip;
    use super::*;

    #[test]
    fn message_panel_and_weather_round_trip() {
        let mut text = [0u8; 128];
        text[..5].copy_from_slice(b"hello");
        let panel = MessagePanel { text };
        let decoded = round_trip(panel);
        assert_eq!(decoded, panel);
        assert_eq!(decoded.text_bytes(), b"hello");
        assert_eq!(MessagePanel::WIRE_LEN, 140);

        let weather = UpdateWeather { current_weather: 3 };
        assert_eq!(UpdateWeather::WIRE_LEN, 16);
        assert_eq!(round_trip(weather), weather);
    }
}
