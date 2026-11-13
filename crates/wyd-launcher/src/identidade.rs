//! Identidade visual do projeto: leitura da **fonte única** `docs/arquitetura/identidade.tokens`.
//!
//! O launcher não tem paleta própria. Tudo o que ele desenha — a skin da tela de entrada
//! (`branding.rs`) e os nomes de servidor que o cliente pinta no `sn.bin` (`snbin.rs`) — sai
//! deste arquivo, o mesmo que o portal referencia em `globals.css`. O teste
//! `tests/identidade.rs` compara o arquivo com o CSS do portal e com os assets instalados:
//! divergência é falha de teste, não aviso de revisão.
//!
//! Formato estrito, mesma disciplina do manifesto: `chave = valor`, `#` comenta, linha sem
//! `=`, chave fora de `[a-z0-9.-]` e chave repetida reprovam **com o número da linha**.

use core::fmt;

/// Arquivo canônico, embutido no binário: o launcher não lê identidade em runtime.
pub const FONTE: &str = include_str!("../../../docs/arquitetura/identidade.tokens");

/// Maior nome de servidor que cabe no campo de 9 B do `sn.bin` (NUL no fim).
pub const NOME_SERVIDOR_MAX: usize = 8;

/// Cor de 24 bits, como o BMP a grava (a ordem de byte da escrita é B, G, R).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Rgb {
    /// Canal vermelho.
    pub r: u8,
    /// Canal verde.
    pub g: u8,
    /// Canal azul.
    pub b: u8,
}

impl Rgb {
    /// Lê `#rrggbb` (o `#` é opcional, o tamanho não).
    pub fn from_hex(valor: &str) -> Result<Self, IdentidadeError> {
        let hex = valor.strip_prefix('#').unwrap_or(valor);
        if hex.len() != 6 || !hex.bytes().all(|byte| byte.is_ascii_hexdigit()) {
            return Err(IdentidadeError::BadColor {
                key: String::new(),
                value: valor.to_string(),
            });
        }
        let par = |inicio: usize| -> Result<u8, IdentidadeError> {
            u8::from_str_radix(&hex[inicio..inicio + 2], 16).map_err(|_| {
                IdentidadeError::BadColor {
                    key: String::new(),
                    value: valor.to_string(),
                }
            })
        };
        Ok(Self {
            r: par(0)?,
            g: par(2)?,
            b: par(4)?,
        })
    }

    /// Forma canônica `#rrggbb` (a mesma do CSS).
    #[must_use]
    pub fn hex(self) -> String {
        format!("#{:02x}{:02x}{:02x}", self.r, self.g, self.b)
    }

    /// Interpolação linear de canais, usada no gradiente da skin.
    #[must_use]
    pub fn lerp(self, outro: Self, t: f32) -> Self {
        let t = t.clamp(0.0, 1.0);
        let misturar = |a: u8, b: u8| -> u8 {
            let a = f32::from(a);
            let b = f32::from(b);
            let v = (a + (b - a) * t).round().clamp(0.0, 255.0);
            // `clamp` garante 0..=255: o cast não trunca valor válido.
            #[allow(clippy::cast_possible_truncation)]
            let byte = v as u8;
            byte
        };
        Self {
            r: misturar(self.r, outro.r),
            g: misturar(self.g, outro.g),
            b: misturar(self.b, outro.b),
        }
    }
}

/// Recusas de leitura da identidade.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum IdentidadeError {
    /// Linha que não é `chave = valor`, ou chave fora do alfabeto.
    BadLine { line: usize, reason: &'static str },
    /// Chave repetida no arquivo.
    DuplicateKey { line: usize, key: String },
    /// Chave que o consumidor precisa e o arquivo não tem.
    Missing { key: String },
    /// Valor que não é `#rrggbb`.
    BadColor { key: String, value: String },
    /// Valor que não é número (ou fora do esperado).
    BadNumber { key: String, value: String },
    /// Chave de servidor inválida (slot ou nome).
    BadServer { key: String, reason: &'static str },
}

impl fmt::Display for IdentidadeError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::BadLine { line, reason } => {
                write!(f, "identidade linha {line}: {reason}")
            }
            Self::DuplicateKey { line, key } => {
                write!(f, "identidade linha {line}: chave `{key}` repetida")
            }
            Self::Missing { key } => write!(f, "identidade sem a chave `{key}`"),
            Self::BadColor { key, value } => {
                write!(f, "identidade `{key}`: cor `{value}` não é #rrggbb")
            }
            Self::BadNumber { key, value } => {
                write!(f, "identidade `{key}`: `{value}` não é número")
            }
            Self::BadServer { key, reason } => {
                write!(f, "identidade `{key}`: {reason}")
            }
        }
    }
}

impl std::error::Error for IdentidadeError {}

/// Identidade já validada: lista de `(chave, valor)` na ordem do arquivo.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Identidade {
    valores: Vec<(String, String)>,
}

impl Identidade {
    /// Analisa o arquivo inteiro; a primeira linha ruim interrompe com o número dela.
    pub fn parse(texto: &str) -> Result<Self, IdentidadeError> {
        let mut valores: Vec<(String, String)> = Vec::new();
        for (indice, bruta) in texto.lines().enumerate() {
            let line = indice + 1;
            let limpa = bruta.trim();
            if limpa.is_empty() || limpa.starts_with('#') {
                continue;
            }
            let Some((chave, valor)) = limpa.split_once('=') else {
                return Err(IdentidadeError::BadLine {
                    line,
                    reason: "esperado `chave = valor`",
                });
            };
            let chave = chave.trim();
            let valor = valor.trim();
            if chave.is_empty() || valor.is_empty() {
                return Err(IdentidadeError::BadLine {
                    line,
                    reason: "chave ou valor vazio",
                });
            }
            if !chave
                .bytes()
                .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'.' || b == b'-')
            {
                return Err(IdentidadeError::BadLine {
                    line,
                    reason: "chave fora de [a-z0-9.-]",
                });
            }
            if valores.iter().any(|(existente, _)| existente == chave) {
                return Err(IdentidadeError::DuplicateKey {
                    line,
                    key: chave.to_string(),
                });
            }
            valores.push((chave.to_string(), valor.to_string()));
        }
        if valores.is_empty() {
            return Err(IdentidadeError::BadLine {
                line: 1,
                reason: "arquivo sem nenhuma chave",
            });
        }
        Ok(Self { valores })
    }

    /// Valor bruto da chave.
    pub fn get(&self, chave: &str) -> Result<&str, IdentidadeError> {
        self.valores
            .iter()
            .find(|(existente, _)| existente == chave)
            .map(|(_, valor)| valor.as_str())
            .ok_or_else(|| IdentidadeError::Missing {
                key: chave.to_string(),
            })
    }

    /// Cor `#rrggbb` da chave.
    pub fn cor(&self, chave: &str) -> Result<Rgb, IdentidadeError> {
        let valor = self.get(chave)?;
        Rgb::from_hex(valor).map_err(|_| IdentidadeError::BadColor {
            key: chave.to_string(),
            value: valor.to_string(),
        })
    }

    /// Inteiro não negativo da chave (espaços, raio).
    pub fn numero(&self, chave: &str) -> Result<u32, IdentidadeError> {
        let valor = self.get(chave)?;
        valor.parse().map_err(|_| IdentidadeError::BadNumber {
            key: chave.to_string(),
            value: valor.to_string(),
        })
    }

    /// Decimal da chave (`tipografia.altura`).
    pub fn decimal(&self, chave: &str) -> Result<f32, IdentidadeError> {
        let valor = self.get(chave)?;
        valor.parse().map_err(|_| IdentidadeError::BadNumber {
            key: chave.to_string(),
            value: valor.to_string(),
        })
    }

    /// Pares na ordem do arquivo (usado pelos testes de paridade com o CSS).
    #[must_use]
    pub fn pares(&self) -> &[(String, String)] {
        &self.valores
    }

    /// Servidores da tela de entrada, ordenados por slot (`1..=10`).
    pub fn servidores(&self) -> Result<Vec<(usize, String)>, IdentidadeError> {
        let mut servidores = Vec::new();
        for (chave, valor) in &self.valores {
            let Some(resto) = chave.strip_prefix("servidor.") else {
                continue;
            };
            let slot: usize = resto.parse().map_err(|_| IdentidadeError::BadServer {
                key: chave.clone(),
                reason: "slot não é inteiro",
            })?;
            if !(1..=10).contains(&slot) {
                return Err(IdentidadeError::BadServer {
                    key: chave.clone(),
                    reason: "slot fora de 1..=10",
                });
            }
            if valor.len() > NOME_SERVIDOR_MAX {
                return Err(IdentidadeError::BadServer {
                    key: chave.clone(),
                    reason: "nome acima de 8 caracteres (campo de 9 B do sn.bin)",
                });
            }
            if !valor.bytes().all(|b| (0x21..=0x7e).contains(&b)) {
                return Err(IdentidadeError::BadServer {
                    key: chave.clone(),
                    reason: "nome fora de ASCII imprimível",
                });
            }
            servidores.push((slot, valor.clone()));
        }
        servidores.sort_by_key(|(slot, _)| *slot);
        Ok(servidores)
    }
}

/// Identidade embutida (a mesma que a skin e o `sn.bin` usam).
pub fn padrao() -> Result<Identidade, IdentidadeError> {
    Identidade::parse(FONTE)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn arquivo_canonico_analisa_e_tem_a_paleta() {
        let id = padrao().expect("identidade versionada válida");
        assert_eq!(id.cor("cor.accent").expect("acento").hex(), "#c8a24a");
        assert_eq!(id.numero("espaco.7").expect("espaço 7"), 84);
        assert_eq!(id.decimal("tipografia.altura").expect("altura"), 1.5);
        assert_eq!(id.servidores().expect("servidores")[0].0, 1);
    }

    #[test]
    fn linha_ruim_chave_repetida_e_cor_invalida_reprovam() {
        assert!(Identidade::parse("cor.bg").is_err());
        assert!(Identidade::parse("cor.bg = #0f1115\ncor.bg = #000000").is_err());
        let id = Identidade::parse("cor.bg = #0f1115").expect("uma chave");
        assert!(id.cor("cor.bg").is_ok());
        assert!(id.get("cor.text").is_err());
        let ruim = Identidade::parse("cor.text = azul").expect("chave válida");
        assert!(ruim.cor("cor.text").is_err());
    }

    #[test]
    fn nome_de_servidor_longo_reprova() {
        let nove = Identidade::parse("servidor.1 = NoveChars").expect("sintaxe válida");
        assert!(
            nove.servidores().is_err(),
            "9 caracteres não cabem no campo de 9 B"
        );
        let oito = Identidade::parse("servidor.1 = OitoChar").expect("sintaxe válida");
        assert_eq!(oito.servidores().expect("8 caracteres")[0].1, "OitoChar");
        let slot = Identidade::parse("servidor.11 = WYD").expect("sintaxe válida");
        assert!(slot.servidores().is_err(), "slot 11 não existe");
    }
}
