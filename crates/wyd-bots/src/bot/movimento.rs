//! Submódulo de `bot` (ADR-0027).

use super::*;

impl Bot {
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
}
