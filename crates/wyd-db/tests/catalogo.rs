//! Catálogo do banco ↔ migrations, sem Postgres.
//!
//! Prende `docs/arquitetura/banco-catalogo.md` ao SQL que realmente cria o schema: tabela,
//! coluna e PK precisam bater, e a série não aceita `ALTER TABLE` no `up` (coluna nova nasce
//! na criação da tabela). A prova contra Postgres de verdade é outro passo, com banco vivo:
//! `node tools/content/verify-schema.mjs`, que compara `pg_attribute` com este mesmo documento.
//!
//! Este teste existe porque documentação de 59 tabelas desatualiza em silêncio — e catálogo
//! errado é pior que catálogo ausente: alguém modela código em cima dele.

use std::collections::{BTreeMap, BTreeSet};
use std::path::{Path, PathBuf};

fn raiz() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
}

fn dir_migrations() -> PathBuf {
    raiz().join("migrations")
}

fn caminho_catalogo() -> PathBuf {
    raiz()
        .join("..")
        .join("..")
        .join("docs")
        .join("arquitetura")
        .join("banco-catalogo.md")
}

fn migrations() -> Vec<(u32, PathBuf)> {
    let mut arquivos: Vec<(u32, PathBuf)> = std::fs::read_dir(dir_migrations())
        .expect("diretório de migrations legível")
        .map(|entrada| entrada.expect("entrada legível").path())
        .filter(|p| p.extension().is_some_and(|ext| ext == "sql"))
        .map(|p| {
            let nome = p.file_name().expect("nome").to_string_lossy().to_string();
            let numero: u32 = nome[..4].parse().expect("NNNN numérico no nome");
            (numero, p)
        })
        .collect();
    arquivos.sort();
    assert!(!arquivos.is_empty(), "há migrations versionadas");
    arquivos
}

fn bloco_up(sql: &str) -> &str {
    // Marcador precisa ocupar a linha inteira: o cabeçalho cita `-- migrate:up` na convenção.
    let inicio = sql
        .find("-- migrate:up\n")
        .map(|i| i + "-- migrate:up\n".len())
        .expect("marcador up");
    let fim = sql.rfind("-- migrate:down").expect("marcador down");
    assert!(inicio < fim, "up antes do down");
    &sql[inicio..fim]
}

struct TabelaSql {
    colunas: Vec<String>,
    pk: Vec<String>,
    arquivo: String,
}

/// Separa o corpo da tabela por vírgula de nível zero (constraint longa quebra em várias
/// linhas; um leitor linha-a-linha trataria `'mortal', 'god'` como coluna).
fn elementos(corpo: &str) -> Vec<String> {
    let texto: String = corpo
        .lines()
        .map(|l| match l.find("--") {
            Some(i) => &l[..i],
            None => l,
        })
        .collect::<Vec<_>>()
        .join("\n");
    let mut partes = Vec::new();
    let mut atual = String::new();
    let mut profundidade = 0i32;
    let mut aspas = false;
    for ch in texto.chars() {
        if ch == '\'' {
            aspas = !aspas;
        }
        if !aspas {
            if ch == '(' {
                profundidade += 1;
            } else if ch == ')' {
                profundidade -= 1;
            } else if ch == ',' && profundidade == 0 {
                partes.push(atual.clone());
                atual.clear();
                continue;
            }
        }
        atual.push(ch);
    }
    if !atual.trim().is_empty() {
        partes.push(atual);
    }
    partes
        .into_iter()
        .map(|p| p.split_whitespace().collect::<Vec<_>>().join(" "))
        .filter(|p| !p.is_empty())
        .collect()
}

fn eh_constraint(elemento: &str) -> bool {
    [
        "CONSTRAINT",
        "PRIMARY KEY",
        "UNIQUE",
        "FOREIGN KEY",
        "CHECK",
    ]
    .iter()
    .any(|prefixo| elemento.starts_with(prefixo))
}

/// Lê o estilo da série: `CREATE TABLE x (` e `);` sozinhos, elementos separados por vírgula.
fn tabelas_das_migrations() -> BTreeMap<String, TabelaSql> {
    let mut tabelas = BTreeMap::new();
    for (_numero, caminho) in migrations() {
        let arquivo = caminho
            .file_name()
            .expect("nome")
            .to_string_lossy()
            .to_string();
        let sql = std::fs::read_to_string(&caminho).expect("migration legível");
        let up = bloco_up(&sql);
        assert!(
            !up.lines().any(|l| l.starts_with("ALTER TABLE")),
            "{arquivo}: ALTER TABLE no up — consolide a coluna na criação da tabela"
        );

        let mut cursor = up;
        while let Some(pos) = cursor.find("CREATE TABLE ") {
            let depois = &cursor[pos + "CREATE TABLE ".len()..];
            let nome: String = depois
                .chars()
                .take_while(|c| c.is_ascii_alphanumeric() || *c == '_')
                .collect();
            let abre = depois.find('(').expect("parêntese de abertura");
            let fecha = depois.find("\n);").expect("fechamento `);` sozinho");
            let corpo = &depois[abre + 1..fecha];

            let mut colunas = Vec::new();
            let mut pk: Vec<String> = Vec::new();
            for elemento in elementos(corpo) {
                if eh_constraint(&elemento) {
                    if let Some(inicio) = elemento.find("PRIMARY KEY (") {
                        let dentro = &elemento[inicio + "PRIMARY KEY (".len()..];
                        let fim = dentro.find(')').expect("PK fechada");
                        pk.extend(dentro[..fim].split(',').map(|c| c.trim().to_string()));
                    }
                    continue;
                }
                let coluna = elemento
                    .split_whitespace()
                    .next()
                    .expect("nome da coluna")
                    .to_string();
                if elemento.contains("PRIMARY KEY") {
                    pk.push(coluna.clone());
                }
                colunas.push(coluna);
            }
            assert!(
                !colunas.is_empty(),
                "{arquivo}: tabela {nome} sem colunas lidas"
            );
            assert!(
                tabelas
                    .insert(
                        nome.clone(),
                        TabelaSql {
                            colunas,
                            pk,
                            arquivo: arquivo.clone(),
                        },
                    )
                    .is_none(),
                "tabela {nome} criada duas vezes"
            );
            cursor = &depois[fecha + "\n);".len()..];
        }
    }
    tabelas
}

struct TabelaDoc {
    colunas: Vec<String>,
    pk: Vec<String>,
}

/// Lê a seção 2 do catálogo — o formato é gerado, então dá para exigir dele.
fn tabelas_do_catalogo(markdown: &str) -> BTreeMap<String, TabelaDoc> {
    let inicio = markdown
        .find("## 2. Catálogo")
        .expect("catálogo tem a seção 2");
    let fim = markdown
        .find("## 3. Decisões provisórias")
        .expect("catálogo tem a seção 3");
    let mut tabelas = BTreeMap::new();
    let mut atual: Option<(String, TabelaDoc)> = None;

    for linha in markdown[inicio..fim].lines() {
        let linha = linha.trim();
        if let Some(titulo) = linha.strip_prefix("### `") {
            if let Some((nome, doc)) = atual.take() {
                tabelas.insert(nome, doc);
            }
            let nome = titulo.trim_end_matches('`').to_string();
            atual = Some((
                nome,
                TabelaDoc {
                    colunas: Vec::new(),
                    pk: Vec::new(),
                },
            ));
            continue;
        }
        let Some((_, doc)) = atual.as_mut() else {
            continue;
        };
        if let Some(coluna) = linha.strip_prefix("| `") {
            let nome = coluna
                .split('`')
                .next()
                .expect("coluna fechada")
                .to_string();
            doc.colunas.push(nome);
        } else if let Some(lista) = linha.strip_prefix("- **PK:** ") {
            doc.pk = lista
                .split(" · ")
                .flat_map(|parte| parte.split(", "))
                .filter_map(|token| token.trim().strip_prefix('`'))
                .map(|token| token.trim_end_matches('`').to_string())
                .collect();
        }
    }
    if let Some((nome, doc)) = atual.take() {
        tabelas.insert(nome, doc);
    }
    tabelas
}

#[test]
fn catalogo_bate_com_as_migrations() {
    let sql = tabelas_das_migrations();
    let texto = std::fs::read_to_string(caminho_catalogo()).expect("catálogo legível");
    let doc = tabelas_do_catalogo(&texto);

    let so_sql: BTreeSet<&String> = sql.keys().filter(|t| !doc.contains_key(*t)).collect();
    let so_doc: BTreeSet<&String> = doc.keys().filter(|t| !sql.contains_key(*t)).collect();
    assert!(
        so_sql.is_empty(),
        "tabelas criadas nas migrations e fora do catálogo: {so_sql:?}"
    );
    assert!(
        so_doc.is_empty(),
        "tabelas no catálogo que nenhuma migration cria: {so_doc:?}"
    );

    for (nome, tabela) in &sql {
        let documentada = doc.get(nome).expect("tabela documentada");
        assert_eq!(
            documentada.colunas, tabela.colunas,
            "{nome}: colunas do catálogo divergem de {}",
            tabela.arquivo
        );
        let mut pk_doc = documentada.pk.clone();
        let mut pk_sql = tabela.pk.clone();
        pk_doc.sort();
        pk_sql.sort();
        assert_eq!(pk_doc, pk_sql, "{nome}: PK do catálogo diverge");
    }
}

#[test]
fn serie_e_contigua_e_schema_version_e_a_ultima() {
    let arquivos = migrations();
    for (indice, (numero, caminho)) in arquivos.iter().enumerate() {
        assert_eq!(
            *numero as usize,
            indice + 1,
            "numeração contígua a partir de 0001: {}",
            caminho.file_name().expect("nome").to_string_lossy()
        );
    }
    let ultima = arquivos.last().expect("há migrations").0;
    assert_eq!(
        wyd_db::SCHEMA_VERSION,
        ultima,
        "SCHEMA_VERSION deve ser o número da última migration"
    );
}

#[test]
fn primeiro_seed_vem_depois_do_ultimo_ddl() {
    let arquivos = migrations();
    let ultimo_ddl = arquivos
        .iter()
        .filter(|(_, caminho)| !nome(caminho).contains("_seed_"))
        .map(|(numero, _)| *numero)
        .max()
        .expect("há DDL");
    let primeiro_seed = arquivos
        .iter()
        .filter(|(_, caminho)| nome(caminho).contains("_seed_"))
        .map(|(numero, _)| *numero)
        .min()
        .expect("há seed");
    assert!(
        primeiro_seed > ultimo_ddl,
        "seed {primeiro_seed:04} antes do último DDL {ultimo_ddl:04}"
    );
}

fn nome(caminho: &Path) -> String {
    caminho
        .file_name()
        .expect("nome")
        .to_string_lossy()
        .to_string()
}
