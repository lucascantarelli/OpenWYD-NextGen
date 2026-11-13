//! Submódulo de `bot` (ADR-0027).

use super::*;

impl Bot {
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
}
