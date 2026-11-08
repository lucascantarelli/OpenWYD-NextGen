//! Porta de conteúdo estático (ADR-0009): o core pede, o `wyd-db` entrega.
//!
//! O banco garante o que é declarativo (PK/FK/CHECK). Aqui moram as regras que SQL **não**
//! expressa — coerência cruzada, vocabulário e contagem mínima — para o boot **abortar**
//! com relatório por tabela/pk/coluna em vez de subir com meio mundo carregado.
//!
//! Nada aqui faz I/O: são dados imutáveis + validação pura. O load acontece **antes** de
//! abrir socket e nunca dentro do tick (`AGENTS.md` §3 invariante 1). Uma coluna cuja
//! semântica ainda não foi portada é `UNVERIFIED` e **não** vira gate (invariante 5).

use std::collections::BTreeSet;
use std::fmt;
use std::future::Future;

/// Teto de problemas reportados (o boot lista, não despeja o banco inteiro no log).
pub const MAX_CONTENT_ISSUES: usize = 20;

/// Versão do conteúdo carregado (`content_version`, linha única).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ContentVersion {
    /// Versão incremental gravada pela última migração de seed.
    pub version: i32,
    /// Nota humana da migração (qual domínio entrou).
    pub note: String,
}

/// Linha de `item_catalog` que o tick precisa (o resto continua no banco, para o portal).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ItemRow {
    /// `item_index` do legado (PK, > 0).
    pub item_index: u32,
    /// Nome canônico (NFC).
    pub name: String,
    /// Preço de catálogo (o preço efetivo vem do overlay `item_price`).
    pub price: i64,
    /// Nível mínimo de uso.
    pub req_level: u16,
    /// Categoria (`item_kind.code`), quando classificada.
    pub kind: Option<String>,
}

/// Linha de `skill_catalog` que o tick precisa.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SkillRow {
    /// Índice da skill (< 248 no CPSock).
    pub skill_index: u16,
    /// Nome; vazio é permitido (skill ainda não casada com fonte oficial).
    pub name: String,
    /// Classe (`game_class.code`), quando classificada.
    pub class_code: Option<String>,
    /// Linhagem (`skill_lineage.code`), quando classificada.
    pub lineage_code: Option<String>,
}

/// Linha de `game_setting` (gameconfig legado, sem semântica inventada).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SettingRow {
    /// Seção do arquivo (`[section]`).
    pub section: String,
    /// Chave dentro da seção.
    pub key: String,
    /// Valor numérico, quando o insumo era numérico.
    pub value_int: Option<i64>,
    /// Valor textual cru (o legado põe várias chaves por linha).
    pub value_text: String,
}

/// Tudo o que o boot precisa do conteúdo estático, já tipado.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ContentTables {
    pub version: ContentVersion,
    pub items: Vec<ItemRow>,
    pub skills: Vec<SkillRow>,
    pub settings: Vec<SettingRow>,
    /// `skill_lineage.code` existentes (para a checagem cruzada de `SkillRow`).
    pub lineages: Vec<String>,
}

/// Contagem por tabela — o que o boot loga junto da versão (determinismo).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ContentCounts {
    pub items: usize,
    pub skills: usize,
    pub settings: usize,
    pub lineages: usize,
}

/// Um problema de conteúdo, no formato que o boot imprime.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ContentIssue {
    /// Tabela de origem (`item_catalog`, `skill_catalog`, …).
    pub table: &'static str,
    /// Chave primária da linha (`-` quando não há linha única).
    pub pk: String,
    /// Coluna ou relação culpada.
    pub column: &'static str,
    /// O que está errado, em português, sem jargão de SQL.
    pub reason: &'static str,
}

impl fmt::Display for ContentIssue {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "{} pk={} coluna={}: {}",
            self.table, self.pk, self.column, self.reason
        )
    }
}

/// Conteúdo inconsistente: o boot imprime e sai com código != 0.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ContentError {
    /// Até [`MAX_CONTENT_ISSUES`] problemas (os excedentes são contados, não listados).
    pub issues: Vec<ContentIssue>,
    /// `true` quando havia mais problemas que o teto.
    pub truncated: bool,
}

impl ContentError {
    /// Erro de um único problema (atalho para o loader).
    #[must_use]
    pub fn single(
        table: &'static str,
        pk: impl Into<String>,
        column: &'static str,
        reason: &'static str,
    ) -> Self {
        Self {
            issues: vec![ContentIssue {
                table,
                pk: pk.into(),
                column,
                reason,
            }],
            truncated: false,
        }
    }
}

impl fmt::Display for ContentError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "conteúdo inconsistente: {} problema(s)",
            self.issues.len()
        )?;
        if self.truncated {
            write!(f, " (lista truncada)")?;
        }
        for issue in &self.issues {
            write!(f, "\n  - {issue}")?;
        }
        Ok(())
    }
}

impl std::error::Error for ContentError {}

/// Porta: quem entrega o conteúdo é a borda (`wyd-db`), não o core (DIP).
pub trait ContentSource {
    /// Carrega e valida o conteúdo. Erro = relatório do que está inconsistente.
    fn load_content(&self) -> impl Future<Output = Result<ContentTables, ContentError>> + Send;
}

impl ContentTables {
    /// Contagem por tabela (vai para o log de boot junto da versão).
    #[must_use]
    pub fn counts(&self) -> ContentCounts {
        ContentCounts {
            items: self.items.len(),
            skills: self.skills.len(),
            settings: self.settings.len(),
            lineages: self.lineages.len(),
        }
    }

    /// Valida o que o SQL não expressa. Lista até [`MAX_CONTENT_ISSUES`] problemas.
    ///
    /// # Errors
    ///
    /// [`ContentError`] quando qualquer regra falha — o chamador **não** sobe o servidor.
    pub fn validate(&self) -> Result<(), ContentError> {
        let mut report = Report::new();

        if self.version.version < 1 {
            report.push("content_version", "1", "version", "versão < 1");
        }
        if self.items.is_empty() {
            report.push(
                "item_catalog",
                "-",
                "item_index",
                "catálogo vazio: nenhum item carregado",
            );
        }
        if self.skills.is_empty() {
            report.push(
                "skill_catalog",
                "-",
                "skill_index",
                "catálogo vazio: nenhuma skill carregada",
            );
        }

        let lineages: BTreeSet<&str> = self.lineages.iter().map(String::as_str).collect();
        if lineages.len() != self.lineages.len() {
            report.push("skill_lineage", "-", "code", "linhagem repetida");
        }

        let mut seen_items = BTreeSet::new();
        for item in &self.items {
            let pk = item.item_index.to_string();
            if !seen_items.insert(item.item_index) {
                report.push("item_catalog", pk, "item_index", "índice repetido");
                continue;
            }
            if item.name.trim().is_empty() {
                report.push("item_catalog", &pk, "name", "nome vazio");
            } else if item.name.chars().any(char::is_control) {
                report.push(
                    "item_catalog",
                    &pk,
                    "name",
                    "nome com caractere de controle",
                );
            }
            if item.price < 0 {
                report.push("item_catalog", &pk, "price", "preço negativo");
            }
        }

        let mut seen_skills = BTreeSet::new();
        for skill in &self.skills {
            let pk = skill.skill_index.to_string();
            if !seen_skills.insert(skill.skill_index) {
                report.push("skill_catalog", pk, "skill_index", "índice repetido");
                continue;
            }
            if skill.name.chars().any(char::is_control) {
                report.push(
                    "skill_catalog",
                    &pk,
                    "name",
                    "nome com caractere de controle",
                );
            }
            if let Some(lineage) = &skill.lineage_code {
                if !lineages.contains(lineage.as_str()) {
                    report.push(
                        "skill_catalog",
                        &pk,
                        "lineage_code",
                        "linhagem inexistente em skill_lineage",
                    );
                }
            }
        }

        let mut seen_settings = BTreeSet::new();
        for setting in &self.settings {
            let pk = format!("{}:{}", setting.section, setting.key);
            if setting.section.trim().is_empty() {
                report.push("game_setting", &pk, "section", "seção vazia");
            }
            if setting.key.trim().is_empty() {
                report.push("game_setting", &pk, "key", "chave vazia");
            }
            if !seen_settings.insert((setting.section.as_str(), setting.key.as_str())) {
                report.push("game_setting", &pk, "key", "chave repetida na seção");
            }
        }

        report.finish()
    }
}

/// Acumulador com teto (private: o formato público é [`ContentError`]).
struct Report {
    issues: Vec<ContentIssue>,
    truncated: bool,
}

impl Report {
    fn new() -> Self {
        Self {
            issues: Vec::new(),
            truncated: false,
        }
    }

    fn push(
        &mut self,
        table: &'static str,
        pk: impl Into<String>,
        column: &'static str,
        reason: &'static str,
    ) {
        if self.issues.len() >= MAX_CONTENT_ISSUES {
            self.truncated = true;
            return;
        }
        self.issues.push(ContentIssue {
            table,
            pk: pk.into(),
            column,
            reason,
        });
    }

    fn finish(self) -> Result<(), ContentError> {
        if self.issues.is_empty() {
            Ok(())
        } else {
            Err(ContentError {
                issues: self.issues,
                truncated: self.truncated,
            })
        }
    }
}
