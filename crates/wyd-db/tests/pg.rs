//! Integração real: portas `pg` e fila de save contra um Postgres descartável.
//!
//! Não roda no `cargo test` sem banco. A ausência de `WYD_TEST_DSN` **pula** com
//! mensagem explícita (nada de fake-pass silencioso); a evidência da FASE-04-3 foi
//! produzida com `postgres:17-alpine` + migrations aplicadas:
//!
//! ```bash
//! docker run -d --rm --name wyd-pg43 -e POSTGRES_PASSWORD=dev -e POSTGRES_DB=wyd \
//!   -p 55432:5432 postgres:17-alpine
//! # aplica os ups de migrations/*.sql (awk entre os marcadores) e seeds/dev.sql
//! WYD_TEST_DSN=postgres://postgres:dev@127.0.0.1:55432/wyd cargo test -p wyd-db --test pg
//! ```
//!
//! Cada teste usa login/nome único (sufixo de nanossegundos) e não depende de ordem:
//! o banco é compartilhado só entre estes casos.

use std::sync::Arc;
use std::time::Instant;

use wyd_db::{
    Accounts, Character, Characters, DbError, Grant, Grants, PgBackend, PoolConfig, Role,
    SavePolicy, SaveQueue, SaveRequest,
};

fn dsn() -> Option<String> {
    match std::env::var("WYD_TEST_DSN") {
        Ok(v) if !v.trim().is_empty() => Some(v),
        _ => {
            eprintln!("SKIP: WYD_TEST_DSN ausente — integração com Postgres não executada");
            None
        }
    }
}

async fn backend() -> Option<PgBackend> {
    let dsn = dsn()?;
    Some(
        PgBackend::connect(&dsn, PoolConfig::local())
            .await
            .expect("conecta no Postgres de teste"),
    )
}

fn suffix() -> u128 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .expect("relógio")
        .as_nanos()
}

fn character(account_id: i64, id: i64, name: String, hp: i64) -> Character {
    Character {
        id,
        account_id,
        slot: 0,
        name,
        class: 0,
        level: 12,
        exp: 340,
        hp,
        mp: 20,
        pos_x: 2100,
        pos_y: 2200,
    }
}

#[tokio::test]
async fn contas_criam_buscam_e_recusam_login_duplicado() {
    let Some(backend) = backend().await else {
        return;
    };
    let login = format!("t{}", suffix());

    let created = backend
        .create_account(&login, Role::Moderator)
        .await
        .expect("cria conta");
    assert!(created.id > 0);
    assert_eq!(created.login, login);
    assert_eq!(created.role, Role::Moderator);

    let by_login = backend
        .find_by_login(&login)
        .await
        .expect("busca por login");
    assert_eq!(by_login, created);
    let by_id = backend.find_by_id(created.id).await.expect("busca por id");
    assert_eq!(by_id, created);

    // UNIQUE de login chega como erro tipado, não como pânico nem default.
    assert!(matches!(
        backend.create_account(&login, Role::Player).await,
        Err(DbError::ConstraintViolation {
            table: "accounts",
            ..
        })
    ));
    assert!(matches!(
        backend.find_by_login(&format!("{login}-nao-existe")).await,
        Err(DbError::NotFoundKey { entity: "account" })
    ));
    assert!(matches!(
        backend.find_by_id(i64::MAX).await,
        Err(DbError::NotFound { .. })
    ));
}

#[tokio::test]
async fn personagens_crud_soft_delete_e_ranking() {
    let Some(backend) = backend().await else {
        return;
    };
    let tag = suffix();
    let account = backend
        .create_account(&format!("c{tag}"), Role::Player)
        .await
        .expect("cria conta");

    let mut base = character(account.id, 0, format!("Hero{tag}"), 90);
    base.slot = 0;
    let first = backend
        .create_character(&base)
        .await
        .expect("cria personagem");
    assert!(first.id > 0);
    assert_eq!(first.account_id, account.id);

    let mut second = character(account.id, 0, format!("Alt{tag}"), 50);
    second.slot = 1;
    second.level = 30;
    let second = backend
        .create_character(&second)
        .await
        .expect("cria segundo");

    let list = backend
        .list_by_account(account.id)
        .await
        .expect("lista por conta");
    assert_eq!(list.len(), 2);
    assert_eq!(list[0].slot, 0);
    assert_eq!(list[1].slot, 1);

    // Slot repetido na mesma conta: o índice único do banco recusa.
    let mut dup = character(account.id, 0, format!("Dup{tag}"), 10);
    dup.slot = 0;
    assert!(matches!(
        backend.create_character(&dup).await,
        Err(DbError::ConstraintViolation { .. })
    ));

    let top = backend.top_by_level(5).await.expect("ranking");
    assert!(!top.is_empty());
    let levels: Vec<i64> = top.iter().map(|c| c.level).collect();
    let mut sorted = levels.clone();
    sorted.sort_unstable_by(|a, b| b.cmp(a));
    assert_eq!(levels, sorted, "ranking desce por nível");

    backend.soft_delete(first.id).await.expect("apaga");
    let list = backend
        .list_by_account(account.id)
        .await
        .expect("lista depois do delete");
    assert_eq!(list.len(), 1);
    assert_eq!(list[0].id, second.id);
    assert!(matches!(
        backend.soft_delete(first.id).await,
        Err(DbError::NotFound {
            entity: "character",
            ..
        })
    ));
}

#[tokio::test]
async fn grants_pendentes_sao_tipados_e_baixados() {
    let Some(backend) = backend().await else {
        return;
    };
    let tag = suffix();
    let account = backend
        .create_account(&format!("g{tag}"), Role::Player)
        .await
        .expect("cria conta");
    let hero = backend
        .create_character(&character(account.id, 0, format!("G{tag}"), 100))
        .await
        .expect("cria personagem");

    // A web enfileira direto na tabela (papel do portal); o servidor só lê e baixa.
    for (kind, item, gold, level) in [
        ("item", Some((1001_i32, 2_i32)), None, None),
        ("gold", None, Some(5_000_i64), None),
        ("level", None, None, Some(1_i32)),
    ] {
        sqlx::query(
            "INSERT INTO delivery_queue \
             (kind, account_id, character_id, item_id, qty, gold_amount, level_delta) \
             VALUES ($1, $2, $3, $4, $5, $6, $7)",
        )
        .bind(kind)
        .bind(account.id)
        .bind(hero.id)
        .bind(item.map(|(i, _)| i))
        .bind(item.map(|(_, q)| q))
        .bind(gold)
        .bind(level)
        .execute(backend.pool())
        .await
        .expect("enfileira grant");
    }

    let pending = backend.pending(50).await.expect("lê pendentes");
    let mine: Vec<&(i64, Grant)> = pending
        .iter()
        .filter(|(_, g)| match g {
            Grant::Item { character_id, .. }
            | Grant::Gold { character_id, .. }
            | Grant::Level { character_id, .. } => *character_id == hero.id,
        })
        .collect();
    assert_eq!(mine.len(), 3, "três grants criados");
    assert!(mine
        .iter()
        .any(|(_, g)| matches!(g, Grant::Item { qty: 2, .. })));
    assert!(mine
        .iter()
        .any(|(_, g)| matches!(g, Grant::Gold { amount: 5_000, .. })));
    assert!(mine
        .iter()
        .any(|(_, g)| matches!(g, Grant::Level { delta: 1, .. })));

    // Baixa idempotente por id: a segunda tentativa é `NotFound`, nunca dupla aplicação.
    let (id, _) = mine[0];
    backend.mark_claimed(*id).await.expect("marca claimed");
    assert!(matches!(
        backend.mark_claimed(*id).await,
        Err(DbError::NotFound {
            entity: "delivery_queue",
            ..
        })
    ));
    let after = backend.pending(50).await.expect("relê pendentes");
    assert!(
        !after.iter().any(|(pending_id, _)| pending_id == id),
        "grant baixado sai dos pendentes"
    );
}

#[tokio::test]
async fn save_grava_o_ultimo_snapshot_e_shutdown_nao_perde() {
    let Some(backend) = backend().await else {
        return;
    };
    let tag = suffix();
    let account = backend
        .create_account(&format!("s{tag}"), Role::Player)
        .await
        .expect("cria conta");

    let hero = backend
        .create_character(&character(account.id, 0, format!("S{tag}"), 100))
        .await
        .expect("cria personagem");

    let (queue, mut drain) = SaveQueue::bounded(SavePolicy::local()).expect("política válida");
    // Dois snapshots do mesmo personagem entre drenos: vence o mais novo.
    queue
        .enqueue(SaveRequest::Character(character(
            account.id,
            hero.id,
            hero.name.clone(),
            90,
        )))
        .expect("cabe");
    queue
        .enqueue(SaveRequest::Character(character(
            account.id,
            hero.id,
            hero.name.clone(),
            70,
        )))
        .expect("cabe");

    let report = drain.flush(&backend).await.expect("flush");
    assert_eq!(report.written, 1);
    assert_eq!(report.coalesced, 1);
    assert_eq!(report.missing, 0);

    let reloaded = backend
        .list_by_account(account.id)
        .await
        .expect("relê personagem");
    assert_eq!(reloaded[0].hp, 70, "gravou o último snapshot");

    // Shutdown: dois personagens distintos, produtor derrubado, dreno esvazia.
    let mut other = character(account.id, 0, format!("S{tag}b"), 40);
    other.slot = 1;
    other.level = 20;
    let other = backend
        .create_character(&other)
        .await
        .expect("cria segundo");

    for (id, name, hp) in [
        (hero.id, hero.name.clone(), 66),
        (other.id, other.name.clone(), 33),
    ] {
        queue
            .enqueue(SaveRequest::Character(character(account.id, id, name, hp)))
            .expect("cabe");
    }
    drop(queue); // fecha o canal: é o sinal de "não vem mais nada".

    let report = drain.shutdown(&backend).await.expect("shutdown flush");
    assert_eq!(report.written, 2, "nada perdido no shutdown");
    assert_eq!(report.missing, 0);
    assert_eq!(drain.policy().max_batch, SavePolicy::local().max_batch);

    let reloaded = backend
        .list_by_account(account.id)
        .await
        .expect("relê os dois");
    assert_eq!(reloaded[0].hp, 66);
    assert_eq!(reloaded[1].hp, 33);

    // Personagem apagado entre enfileirar e drenar: contado como `missing`, não silêncio.
    let (queue2, mut drain2) = SaveQueue::bounded(SavePolicy::local()).expect("política válida");
    queue2
        .enqueue(SaveRequest::Character(character(
            account.id,
            other.id,
            other.name.clone(),
            11,
        )))
        .expect("cabe");
    backend.soft_delete(other.id).await.expect("apaga");
    let report = drain2.flush(&backend).await.expect("flush do apagado");
    assert_eq!(report.missing, 1);
    assert_eq!(report.written, 0);
}

#[tokio::test]
async fn enqueue_do_hot_path_nao_espera_o_banco() {
    let Some(backend) = backend().await else {
        return;
    };
    let tag = suffix();
    let account = backend
        .create_account(&format!("l{tag}"), Role::Player)
        .await
        .expect("cria conta");
    let hero = backend
        .create_character(&character(account.id, 0, format!("L{tag}"), 100))
        .await
        .expect("cria personagem");

    let policy = SavePolicy {
        queue_capacity: 65_536,
        ..SavePolicy::local()
    };
    let (queue, mut drain) = SaveQueue::bounded(policy).expect("política válida");

    // Dreno em paralelo, gravando de verdade: o custo do lado do tick não pode depender dele.
    let backend = Arc::new(backend);
    let drain_backend = Arc::clone(&backend);
    let drainer = tokio::spawn(async move { drain.flush(&drain_backend).await });

    const N: usize = 20_000;
    let snapshot = character(account.id, hero.id, hero.name.clone(), 100);
    let mut samples = Vec::with_capacity(N);
    for i in 0..N {
        let mut s = snapshot.clone();
        s.hp = (i % 100) as i64;
        let start = Instant::now();
        queue
            .enqueue(SaveRequest::Character(s))
            .expect("capacidade suficiente");
        samples.push(start.elapsed().as_nanos());
    }
    assert_eq!(queue.pending(), N);

    samples.sort_unstable();
    let pct = |p: usize| samples[(samples.len() - 1) * p / 100];
    let p50 = pct(50);
    let p99 = pct(99);
    let max = *samples.last().expect("amostras");
    eprintln!(
        "enqueue (Hot Path) N={N}: p50={p50} ns, p99={p99} ns, max={max} ns \
         (orçamento do tick 50 ms)"
    );
    // Limite folgado e proposital: 1 ms é 2% de um tick de 20 Hz. Medido fica em
    // centenas de nanossegundos; o assert existe para pegar regressão grosseira
    // (ex.: alguém trocar `try_send` por `send().await` não compila no tick, mas
    // alocar dentro do snapshot passaria despercebido).
    assert!(p99 < 1_000_000, "enqueue p99 {p99} ns estourou 1 ms");

    // O dreno rodou concorrente com o produtor: o resultado dele não pode passar do
    // que foi enfileirado, e o contador (única disputa entre as duas metades) também não.
    let flushed = drainer.await.expect("tarefa de dreno").expect("flush");
    assert!(flushed.received <= N as u32, "dreno leu {flushed:?}");
    assert!(queue.pending() <= N, "contador {}", queue.pending());
}

#[tokio::test]
async fn constraints_do_banco_chegam_tipadas() {
    let Some(backend) = backend().await else {
        return;
    };
    // FK inexistente: personagem com conta que não existe.
    let ghost = character(i64::MAX - 1, 0, format!("X{}", suffix()), 10);
    assert!(matches!(
        backend.create_character(&ghost).await,
        Err(DbError::ConstraintViolation { .. })
    ));

    // CHECK de slot (0–3) antes mesmo de chegar ao banco.
    let mut bad = character(1, 0, format!("Y{}", suffix()), 10);
    bad.slot = 9;
    assert!(matches!(
        backend.create_character(&bad).await,
        Err(DbError::ConstraintViolation {
            constraint: "slot/level/exp",
            ..
        })
    ));
}

#[tokio::test]
async fn conteudo_estatico_carrega_e_aborta_com_dado_corrompido() {
    use wyd_core::ports::ContentSource;

    let Some(backend) = backend().await else {
        return;
    };

    // Caminho feliz: o seed das migrations é coerente e a versão é a última aplicada.
    let tables = backend
        .load_content()
        .await
        .expect("conteúdo das migrations deve ser válido");
    assert!(tables.version.version >= 1, "{:?}", tables.version);
    assert!(tables.items.len() > 100, "itens: {}", tables.items.len());
    assert!(!tables.skills.is_empty(), "skill_catalog vazio");
    assert!(!tables.lineages.is_empty(), "skill_lineage vazio");
    assert!(
        tables
            .skills
            .iter()
            .filter_map(|s| s.lineage_code.as_deref())
            .all(|code| tables.lineages.iter().any(|l| l == code)),
        "toda linhagem carregada existe em skill_lineage"
    );
    eprintln!(
        "conteúdo v{} \"{}\": {} itens, {} skills, {} settings, {} linhagens",
        tables.version.version,
        tables.version.note,
        tables.counts().items,
        tables.counts().skills,
        tables.counts().settings,
        tables.counts().lineages
    );

    // Dado corrompido que o SQL não barra (sem CHECK de controle em `skill_catalog.name`):
    // o loader tem de abortar o boot, não subir com nome inválido.
    let victim = tables.skills[0].skill_index;
    let original = tables.skills[0].name.clone();
    let update = "UPDATE skill_catalog SET name = $1 WHERE skill_index = $2";
    sqlx::query(update)
        .bind("nome\u{1}corrompido")
        .bind(i32::from(victim))
        .execute(backend.pool())
        .await
        .expect("corrompe a linha");
    let err = backend
        .load_content()
        .await
        .expect_err("conteúdo corrompido deve ser rejeitado");
    // Restaura antes de afirmar qualquer coisa: o banco é compartilhado entre os casos.
    sqlx::query(update)
        .bind(&original)
        .bind(i32::from(victim))
        .execute(backend.pool())
        .await
        .expect("restaura a linha");

    assert!(
        err.issues
            .iter()
            .any(|i| i.table == "skill_catalog" && i.column == "name"),
        "relatório sem a linha culpada: {err}"
    );
    assert!(
        backend.load_content().await.is_ok(),
        "restauração não voltou ao válido"
    );
}
