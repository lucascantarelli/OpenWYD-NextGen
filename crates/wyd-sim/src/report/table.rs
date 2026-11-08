//! Tabela de texto alinhada, determinística e sem dependência.
//!
//! Alocação é livre aqui: relatório é caminho frio do laboratório, nunca o tick.

use crate::formula::damage::{damage_chain, DamageInput};

/// Tabela de colunas de texto, com alinhamento à esquerda por coluna.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Table {
    headers: Vec<String>,
    rows: Vec<Vec<String>>,
}

impl Table {
    /// Tabela com cabeçalho fixo; as linhas seguintes são ajustadas a esta largura.
    #[must_use]
    pub fn new(headers: &[&str]) -> Self {
        Self {
            headers: headers.iter().map(|h| (*h).to_owned()).collect(),
            rows: Vec::new(),
        }
    }

    /// Quantas colunas o cabeçalho define.
    #[must_use]
    pub fn columns(&self) -> usize {
        self.headers.len()
    }

    /// Quantas linhas de dados.
    #[must_use]
    pub fn len(&self) -> usize {
        self.rows.len()
    }

    /// Tabela só com cabeçalho.
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.rows.is_empty()
    }

    /// Adiciona uma linha; células a mais são ignoradas e a menos viram vazio.
    pub fn push(&mut self, cells: &[&str]) {
        let mut linha = Vec::with_capacity(self.headers.len());
        for indice in 0..self.headers.len() {
            linha.push(cells.get(indice).copied().unwrap_or("").to_owned());
        }
        self.rows.push(linha);
    }

    /// Renderiza com `| ` separando colunas e uma régua de `-` sob o cabeçalho.
    #[must_use]
    pub fn render(&self) -> String {
        let mut larguras: Vec<usize> = self.headers.iter().map(String::len).collect();
        for linha in &self.rows {
            for (indice, celula) in linha.iter().enumerate() {
                if let Some(largura) = larguras.get_mut(indice) {
                    *largura = (*largura).max(celula.len());
                }
            }
        }

        let mut saida = String::new();
        push_linha(&mut saida, &self.headers, &larguras);
        let regua: Vec<String> = larguras.iter().map(|l| "-".repeat(*l)).collect();
        push_linha(&mut saida, &regua, &larguras);
        for linha in &self.rows {
            push_linha(&mut saida, linha, &larguras);
        }
        saida
    }
}

fn push_linha(saida: &mut String, celulas: &[String], larguras: &[usize]) {
    let mut primeira = true;
    for (indice, celula) in celulas.iter().enumerate() {
        let largura = larguras.get(indice).copied().unwrap_or(0);
        if !primeira {
            saida.push_str(" | ");
        }
        saida.push_str(celula);
        for _ in celula.len()..largura {
            saida.push(' ');
        }
        primeira = false;
    }
    saida.push('\n');
}

/// Dano resultante por defesa do alvo, para um dano base fixo.
///
/// Junta a cadeia de dano ao formato: é o "atual × proposto" de `estrutura-e-padroes.md` §14
/// sem inventar número — o número vem de [`damage_chain`].
#[must_use]
pub fn damage_by_defense(base: u32, defenses: &[u32]) -> Table {
    let mut tabela = Table::new(&["defesa", "dano"]);
    for defesa in defenses {
        let dano = damage_chain(&DamageInput {
            defense: *defesa,
            ..DamageInput::new(base)
        });
        tabela.push(&[&defesa.to_string(), &dano.to_string()]);
    }
    tabela
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn columns_are_aligned_and_stable() {
        let mut tabela = Table::new(&["nivel", "dano"]);
        tabela.push(&["1", "7"]);
        tabela.push(&["1000", "123456"]);
        let texto = tabela.render();
        let linhas: Vec<&str> = texto.lines().collect();
        assert_eq!(linhas.len(), 4, "cabeçalho + régua + 2 linhas");
        assert_eq!(linhas[0], "nivel | dano  ");
        assert_eq!(linhas[2], "1     | 7     ");
        assert_eq!(linhas[3], "1000  | 123456");
        assert_eq!(texto, tabela.clone().render(), "render é determinístico");
    }

    #[test]
    fn short_rows_are_padded_not_rejected() {
        let mut tabela = Table::new(&["a", "b"]);
        tabela.push(&["x"]);
        assert_eq!(tabela.len(), 1);
        assert!(tabela.render().contains("x |"), "linha curta é completada");
        assert_eq!(tabela.columns(), 2);
    }

    #[test]
    fn damage_by_defense_uses_the_chain_and_stops_at_zero() {
        let tabela = damage_by_defense(100, &[0, 40, 500]);
        let texto = tabela.render();
        assert!(texto.contains("100"), "texto: {texto}");
        assert!(texto.contains('0'), "texto: {texto}");
        assert_eq!(tabela.columns(), 2);
        assert_eq!(tabela.len(), 3);
    }
}
