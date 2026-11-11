//! Submódulo de `bot` (ADR-0027).

use super::*;

/// Extrai `N` de `"Campo de Treinamento: etapa N/4"`.
#[must_use]
pub fn parse_etapa(texto: &str) -> Option<i16> {
    let resto = texto.split("etapa ").nth(1)?;
    let numero: String = resto.chars().take_while(char::is_ascii_digit).collect();
    numero.parse().ok()
}

impl Bot {
    /// `_MSG_Quest` no treinador `npc_index`; devolve o texto do painel de resposta.
    pub fn quest(&mut self, npc_index: i32) -> std::io::Result<String> {
        self.send(Opcode::Quest, Quest::WIRE_LEN, |frame| {
            Quest {
                npc_index,
                confirm: 1,
            }
            .encode_into(frame)
            .expect("corpo do Quest");
        })?;
        let limite = std::time::Instant::now() + Duration::from_secs(5);
        while std::time::Instant::now() < limite {
            let (header, frame) = self.recv()?;
            if Opcode::from_wire(header.message_type) == Some(Opcode::MessagePanel) {
                let painel = MessagePanel::decode(&frame)
                    .map_err(|erro| Error::new(ErrorKind::InvalidData, erro.to_string()))?;
                return Ok(String::from_utf8_lossy(painel.text_bytes()).to_string());
            }
        }
        Err(Error::new(ErrorKind::TimedOut, "sem MessagePanel da quest"))
    }

    /// Roda as quatro etapas na ordem; recusa painel de etapa errada.
    pub fn campaign(&mut self, treinadores: &[i32; 4]) -> std::io::Result<Vec<String>> {
        let mut paineis = Vec::with_capacity(4);
        for (indice, npc) in treinadores.iter().enumerate() {
            let texto = self.quest(*npc)?;
            let etapa = parse_etapa(&texto).ok_or_else(|| {
                Error::new(
                    ErrorKind::InvalidData,
                    format!("{}: painel sem etapa: {texto}", self.race.nome()),
                )
            })?;
            let esperada = i16::try_from(indice).unwrap_or(0) + 1;
            if etapa != esperada {
                return Err(Error::new(
                    ErrorKind::InvalidData,
                    format!(
                        "{}: etapa {etapa} fora de ordem (esperada {esperada}): {texto}",
                        self.race.nome()
                    ),
                ));
            }
            paineis.push(texto);
        }
        Ok(paineis)
    }
}
