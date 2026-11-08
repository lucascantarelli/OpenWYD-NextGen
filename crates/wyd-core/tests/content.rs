//! Conteúdo inconsistente **aborta o boot** (ADR-0009), com relatório por tabela/pk/coluna.
//!
//! Estes testes cobrem o que o SQL não expressa: vocabulário, coerência cruzada e
//! contagem mínima. O caminho com banco real (migrations + seed) fica em
//! `crates/wyd-db/tests/pg.rs`; aqui a validação pura, sem I/O.

use wyd_core::ports::{
    ContentError, ContentTables, ContentVersion, ItemRow, SettingRow, SkillRow, MAX_CONTENT_ISSUES,
};

fn item(index: u32, name: &str) -> ItemRow {
    ItemRow {
        item_index: index,
        name: name.to_string(),
        price: 10,
        req_level: 1,
        kind: None,
    }
}

fn skill(index: u16, name: &str, lineage: Option<&str>) -> SkillRow {
    SkillRow {
        skill_index: index,
        name: name.to_string(),
        class_code: None,
        lineage_code: lineage.map(str::to_string),
    }
}

fn setting(section: &str, key: &str) -> SettingRow {
    SettingRow {
        section: section.to_string(),
        key: key.to_string(),
        value_int: Some(1),
        value_text: String::new(),
    }
}

fn healthy() -> ContentTables {
    ContentTables {
        version: ContentVersion {
            version: 3,
            note: "taxonomia + catálogo de itens".to_string(),
        },
        items: vec![item(1, "Espada Curta"), item(2, "Poção de Vida")],
        skills: vec![skill(0, "Faith", Some("faith")), skill(1, "Corte", None)],
        settings: vec![setting("server", "exp_rate"), setting("pvp", "enabled")],
        lineages: vec!["faith".to_string(), "trans".to_string()],
    }
}

fn issues(tables: &ContentTables) -> ContentError {
    tables
        .validate()
        .expect_err("conteúdo deveria ser rejeitado")
}

#[test]
fn conteudo_coerente_passa_e_conta_por_tabela() {
    let tables = healthy();
    assert!(tables.validate().is_ok());

    let counts = tables.counts();
    assert_eq!(counts.items, 2);
    assert_eq!(counts.skills, 2);
    assert_eq!(counts.settings, 2);
    assert_eq!(counts.lineages, 2);
}

#[test]
fn indice_de_item_repetido_aponta_tabela_pk_e_coluna() {
    let mut tables = healthy();
    tables.items.push(item(2, "Poção de Vida (duplicada)"));
    let err = issues(&tables);

    assert_eq!(err.issues.len(), 1);
    let issue = &err.issues[0];
    assert_eq!(issue.table, "item_catalog");
    assert_eq!(issue.pk, "2");
    assert_eq!(issue.column, "item_index");
    assert!(err.to_string().contains("item_catalog pk=2"));
}

#[test]
fn nome_em_branco_ou_com_controle_rejeita() {
    let mut tables = healthy();
    tables.items[0].name = "   ".to_string();
    assert_eq!(issues(&tables).issues[0].reason, "nome vazio");

    let mut tables = healthy();
    tables.items[1].name = "Poção\u{7}".to_string();
    assert_eq!(
        issues(&tables).issues[0].reason,
        "nome com caractere de controle"
    );
}

#[test]
fn linhagem_inexistente_e_erro_cruzado() {
    let mut tables = healthy();
    tables.skills[0].lineage_code = Some("inventada".to_string());
    let err = issues(&tables);

    assert_eq!(err.issues.len(), 1);
    assert_eq!(err.issues[0].table, "skill_catalog");
    assert_eq!(err.issues[0].pk, "0");
    assert_eq!(err.issues[0].column, "lineage_code");
    assert_eq!(
        err.issues[0].reason,
        "linhagem inexistente em skill_lineage"
    );
}

#[test]
fn catalogo_vazio_e_versao_invalida_rejeitam() {
    let mut tables = healthy();
    tables.items.clear();
    tables.skills.clear();
    tables.version.version = 0;
    let err = issues(&tables);

    assert_eq!(err.issues.len(), 3);
    assert!(err.issues.iter().any(|i| i.table == "item_catalog"));
    assert!(err.issues.iter().any(|i| i.table == "skill_catalog"));
    assert!(err.issues.iter().any(|i| i.table == "content_version"));
}

#[test]
fn chave_de_setting_repetida_e_vazia_rejeitam() {
    let mut tables = healthy();
    tables.settings.push(setting("server", "exp_rate"));
    let err = issues(&tables);
    assert_eq!(err.issues.len(), 1);
    assert_eq!(err.issues[0].column, "key");

    let mut tables = healthy();
    tables.settings.push(setting("", "sem_secao"));
    let err = issues(&tables);
    assert_eq!(err.issues[0].column, "section");
}

#[test]
fn relatorio_tem_teto_e_marca_truncado() {
    let mut tables = healthy();
    tables.items = (1..=(MAX_CONTENT_ISSUES as u32 + 5))
        .map(|i| item(i, "   "))
        .collect();

    let err = issues(&tables);
    assert_eq!(err.issues.len(), MAX_CONTENT_ISSUES);
    assert!(err.truncated);
    assert!(err.to_string().contains("lista truncada"));
}
