//! `_MSG_Quest` (`0x028B`): interação de quest no NPC (Campo de Treinamento).
//!
//! Layout: `MSG_STANDARDPARM2` = `HEADER` (12 B) + `int Parm1` + `int Parm2` = **20 B**.
//! O legado lê `Parm1` como o `npcIndex` clicado
//! (`origin/main:Source/Code/TMSrv/_MSG_Quest.cpp`, `switch (p->Parm1)`), e `Parm2` é o
//! `confirm` do cliente. O servidor **não** usa `Parm2` para escolher recompensa: o NPC
//! (merchant) e o estágio `Newbie` decidem, nunca o corpo do pacote.
//!
//! O cliente não tem janela de quest: a resposta é `_MSG_MessagePanel`/`SendSay`.

use super::{check_len, PayloadError, WireField, WirePayload};
use crate::wire::opcode::Opcode;

wire_payload!(Quest, Quest, 20, [(12, npc_index, i32), (16, confirm, i32)]);

#[cfg(test)]
mod tests {
    use super::super::{declared_len, round_trip};
    use super::*;
    use crate::wire::opcode::Opcode;

    #[test]
    fn quest_round_trip() {
        let quest = Quest {
            npc_index: 1042,
            confirm: 1,
        };
        assert_eq!(Quest::WIRE_LEN, 20, "MSG_STANDARDPARM2: 12 de HEADER + 8");
        assert_eq!(round_trip(quest), quest);
        assert_eq!(declared_len(Opcode::Quest), Some(20));
        // Corpo curto/longo é recusado sem panic (entrada de rede nunca derruba).
        assert!(Quest::decode(&[0u8; 19]).is_err());
        assert!(Quest::decode(&[0u8; 21]).is_err());
    }
}
