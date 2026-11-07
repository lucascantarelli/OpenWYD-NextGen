//! Login e segurança de conta (`_MSG_AccountLogin`, `_MSG_AccountSecure`).

use super::{check_len, nul_trimmed, PayloadError, WireField, WirePayload};
use crate::wire::opcode::Opcode;

wire_payload!(
    AccountLogin,
    AccountLogin,
    116,
    [
        (12, account_password, [u8; 12]),
        (24, account_name, [u8; 16]),
        (40, zero, [u8; 52]),
        (92, client_version, i32),
        (96, db_need_save, i32),
        (100, adapter_name, [i32; 4]),
    ]
);

impl AccountLogin {
    /// Nome de conta útil (sem padding NUL).
    #[must_use]
    pub fn account_name_bytes(&self) -> &[u8] {
        nul_trimmed(&self.account_name)
    }
}

wire_payload!(
    AccountSecure,
    AccountSecure,
    32,
    [
        (12, numeric_token, [u8; 6]),
        (18, unknown, [u8; 10]),
        (28, change_numeric, i32),
    ]
);

#[cfg(test)]
mod tests {
    use super::super::round_trip;
    use super::*;

    #[test]
    fn account_login_round_trip() {
        let mut account_name = [0u8; 16];
        account_name[..4].copy_from_slice(b"test");
        let login = AccountLogin {
            account_password: [1u8; 12],
            account_name,
            zero: [0u8; 52],
            client_version: 12_000,
            db_need_save: 1,
            adapter_name: [11, 22, 33, 44],
        };
        let mut buf = vec![0u8; AccountLogin::WIRE_LEN];
        login.encode_into(&mut buf).expect("tamanho correto");
        assert_eq!(&buf[24..28], b"test");
        assert_eq!(&buf[92..96], &12_000i32.to_le_bytes());

        let decoded = round_trip(login);
        assert_eq!(decoded, login);
        assert_eq!(decoded.account_name_bytes(), b"test");
        assert_eq!(AccountLogin::WIRE_LEN, 116);
        assert_eq!(AccountLogin::OPCODE.wire(), 525);
    }

    #[test]
    fn account_secure_round_trip() {
        let secure = AccountSecure {
            numeric_token: [1, 2, 3, 4, 5, 6],
            unknown: [0u8; 10],
            change_numeric: 1,
        };
        assert_eq!(AccountSecure::WIRE_LEN, 32);
        assert_eq!(round_trip(secure), secure);
    }
}
