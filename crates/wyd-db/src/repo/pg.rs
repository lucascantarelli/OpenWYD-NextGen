//! Implementação `pg` das portas — o adapter vive aqui, as portas nos arquivos irmãos.
//!
//! Cada operação é **uma** transação curta e devolve erro tipado: violação de constraint
//! vira `ConstraintViolation` (o chamador corrige o dado), linha ausente vira `NotFound`
//! (nunca `None` silencioso). Nenhuma string de SQL é montada com entrada do usuário:
//! tudo passa por bind.
//!
//! `delivery_queue` é lida **e baixada** aqui (`pending`/`mark_claimed`): a web enfileira,
//! o servidor drena — a única ponte permitida (`AGENTS.md` §3.7).

use crate::error::DbError;
use crate::model::{Account, Character, Role};
use crate::pool::pg::{map_error, PgBackend};
use crate::repo::accounts::CredentialRecord;
use crate::repo::{Accounts, Characters, Grant, Grants};

use sqlx::postgres::PgRow;
use sqlx::Row;

impl Accounts for PgBackend {
    async fn find_by_login(&self, login: &str) -> Result<Account, DbError> {
        let row = sqlx::query("SELECT id, login, role FROM accounts WHERE login = $1")
            .bind(login)
            .fetch_optional(self.pool())
            .await
            .map_err(map_error("accounts.find_by_login", "accounts"))?;
        row.map(|r| account_row(&r))
            .transpose()?
            .ok_or(DbError::NotFoundKey { entity: "account" })
    }

    async fn find_by_id(&self, id: i64) -> Result<Account, DbError> {
        let row = sqlx::query("SELECT id, login, role FROM accounts WHERE id = $1")
            .bind(id)
            .fetch_optional(self.pool())
            .await
            .map_err(map_error("accounts.find_by_id", "accounts"))?;
        row.map(|r| account_row(&r))
            .transpose()?
            .ok_or(DbError::NotFound {
                entity: "account",
                id,
            })
    }

    async fn create_account(&self, login: &str, role: Role) -> Result<Account, DbError> {
        Account::validate_login(login)?;
        // Hash de senha é responsabilidade da borda de autenticação (argon2id, AGENTS §3.6).
        // O placeholder abaixo existe só porque a coluna é NOT NULL: `create` aqui é a
        // criação administrativa de conta **sem credencial** (portal cria e o jogador define
        // a senha no primeiro login). Nunca é senha em texto plano de usuário real.
        let row = sqlx::query(
            "INSERT INTO accounts (login, password_hash, role) \
             VALUES ($1, $2, $3) RETURNING id, login, role",
        )
        .bind(login)
        .bind(NO_CREDENTIAL)
        .bind(role.as_db())
        .fetch_one(self.pool())
        .await
        .map_err(map_error("accounts.create_account", "accounts"))?;
        account_row(&row)
    }

    async fn credentials(&self, login: &str) -> Result<CredentialRecord, DbError> {
        let row =
            sqlx::query("SELECT id, login, role, password_hash FROM accounts WHERE login = $1")
                .bind(login)
                .fetch_optional(self.pool())
                .await
                .map_err(map_error("accounts.credentials", "accounts"))?;
        let Some(row) = row else {
            return Err(DbError::NotFoundKey { entity: "account" });
        };
        let password_hash: String =
            row.try_get("password_hash")
                .map_err(|_| DbError::Transport {
                    op: "accounts.credentials.decode",
                })?;
        Ok(CredentialRecord {
            account: account_row(&row)?,
            password_hash,
        })
    }

    async fn set_credentials(
        &self,
        login: &str,
        password_hash: &str,
        role: Role,
    ) -> Result<Account, DbError> {
        Account::validate_login(login)?;
        // Upsert de propósito: o seed de staff roda no `up` da stack e precisa ser
        // idempotente (rodar de novo rotaciona a senha, não duplica a conta).
        let row = sqlx::query(
            "INSERT INTO accounts (login, password_hash, role) VALUES ($1, $2, $3) \
             ON CONFLICT (login) DO UPDATE \
             SET password_hash = EXCLUDED.password_hash, role = EXCLUDED.role \
             RETURNING id, login, role",
        )
        .bind(login)
        .bind(password_hash)
        .bind(role.as_db())
        .fetch_one(self.pool())
        .await
        .map_err(map_error("accounts.set_credentials", "accounts"))?;
        account_row(&row)
    }
}

/// Marca de conta criada sem credencial (primeiro login define a senha).
///
/// Não é hash de senha e **não** autentica: a borda de login compara `password_hash`
/// com argon2id e recusa qualquer coisa que não comece com `$argon2`.
pub const NO_CREDENTIAL: &str = "$sem-credencial$definir-no-primeiro-login";

fn account_row(row: &PgRow) -> Result<Account, DbError> {
    let role = row
        .try_get::<String, _>("role")
        .map_err(|_| DbError::Transport {
            op: "accounts.decode",
        })?;
    Ok(Account {
        id: row.try_get("id").map_err(|_| DbError::Transport {
            op: "accounts.decode",
        })?,
        login: row.try_get("login").map_err(|_| DbError::Transport {
            op: "accounts.decode",
        })?,
        role: role_from_db(&role)?,
    })
}

fn role_from_db(role: &str) -> Result<Role, DbError> {
    match role {
        "player" => Ok(Role::Player),
        "moderator" => Ok(Role::Moderator),
        "admin" => Ok(Role::Admin),
        // CHECK do banco enumera os três; valor fora dele é dado corrompido, não default.
        _ => Err(DbError::ConstraintViolation {
            table: "accounts",
            constraint: "accounts_role",
        }),
    }
}

/// Colunas do personagem na ordem usada por todo `SELECT`/`RETURNING` deste módulo.
const CHARACTER_COLUMNS: &str =
    "id, account_id, slot, name, class, level, exp, hp, mp, pos_x, pos_y";

impl Characters for PgBackend {
    async fn list_by_account(&self, account_id: i64) -> Result<Vec<Character>, DbError> {
        let sql = format!(
            "SELECT {CHARACTER_COLUMNS} FROM characters \
             WHERE account_id = $1 AND deleted_at IS NULL ORDER BY slot"
        );
        let rows = sqlx::query(&sql)
            .bind(account_id)
            .fetch_all(self.pool())
            .await
            .map_err(map_error("characters.list_by_account", "characters"))?;
        rows.iter().map(character_row).collect()
    }

    async fn top_by_level(&self, limit: usize) -> Result<Vec<Character>, DbError> {
        let sql = format!(
            "SELECT {CHARACTER_COLUMNS} FROM characters \
             WHERE deleted_at IS NULL ORDER BY level DESC, name ASC LIMIT $1"
        );
        let rows = sqlx::query(&sql)
            .bind(i64::try_from(limit).unwrap_or(i64::MAX))
            .fetch_all(self.pool())
            .await
            .map_err(map_error("characters.top_by_level", "characters"))?;
        rows.iter().map(character_row).collect()
    }

    async fn create_character(&self, character: &Character) -> Result<Character, DbError> {
        character.validate()?;
        let sql = format!(
            "INSERT INTO characters \
             (account_id, slot, name, class, level, exp, hp, mp, pos_x, pos_y) \
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING {CHARACTER_COLUMNS}"
        );
        let row = sqlx::query(&sql)
            .bind(character.account_id)
            .bind(i16::try_from(character.slot).unwrap_or(i16::MAX))
            .bind(&character.name)
            .bind(i16::try_from(character.class).unwrap_or(0))
            .bind(i32::try_from(character.level).unwrap_or(1))
            .bind(character.exp)
            .bind(i32::try_from(character.hp).unwrap_or(1))
            .bind(i32::try_from(character.mp).unwrap_or(0))
            .bind(i32::try_from(character.pos_x).unwrap_or(0))
            .bind(i32::try_from(character.pos_y).unwrap_or(0))
            .fetch_one(self.pool())
            .await
            .map_err(map_error("characters.create_character", "characters"))?;
        character_row(&row)
    }

    async fn soft_delete(&self, id: i64) -> Result<(), DbError> {
        let res = sqlx::query(
            "UPDATE characters SET deleted_at = now(), updated_at = now() \
             WHERE id = $1 AND deleted_at IS NULL",
        )
        .bind(id)
        .execute(self.pool())
        .await
        .map_err(map_error("characters.soft_delete", "characters"))?;
        if res.rows_affected() == 0 {
            return Err(DbError::NotFound {
                entity: "character",
                id,
            });
        }
        Ok(())
    }
}

fn character_row(row: &PgRow) -> Result<Character, DbError> {
    let op = "characters.decode";
    let decode = |_| DbError::Transport { op };
    Ok(Character {
        id: row.try_get("id").map_err(decode)?,
        account_id: row.try_get("account_id").map_err(decode)?,
        slot: i64::from(row.try_get::<i16, _>("slot").map_err(decode)?),
        name: row.try_get("name").map_err(decode)?,
        class: i64::from(row.try_get::<i16, _>("class").map_err(decode)?),
        level: i64::from(row.try_get::<i32, _>("level").map_err(decode)?),
        exp: row.try_get("exp").map_err(decode)?,
        hp: i64::from(row.try_get::<i32, _>("hp").map_err(decode)?),
        mp: i64::from(row.try_get::<i32, _>("mp").map_err(decode)?),
        pos_x: i64::from(row.try_get::<i32, _>("pos_x").map_err(decode)?),
        pos_y: i64::from(row.try_get::<i32, _>("pos_y").map_err(decode)?),
    })
}

impl Grants for PgBackend {
    async fn pending(&self, limit: u32) -> Result<Vec<(i64, Grant)>, DbError> {
        let rows = sqlx::query(
            "SELECT id, kind, character_id, item_id, qty, gold_amount, level_delta \
             FROM delivery_queue WHERE status = 'pending' ORDER BY id LIMIT $1",
        )
        .bind(i64::from(limit))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("grants.pending", "delivery_queue"))?;
        rows.iter().map(grant_row).collect()
    }

    async fn mark_claimed(&self, id: i64) -> Result<(), DbError> {
        let res = sqlx::query(
            "UPDATE delivery_queue SET status = 'claimed', claimed_at = now() \
             WHERE id = $1 AND status = 'pending'",
        )
        .bind(id)
        .execute(self.pool())
        .await
        .map_err(map_error("grants.mark_claimed", "delivery_queue"))?;
        if res.rows_affected() == 0 {
            return Err(DbError::NotFound {
                entity: "delivery_queue",
                id,
            });
        }
        Ok(())
    }
}

fn grant_row(row: &PgRow) -> Result<(i64, Grant), DbError> {
    let op = "grants.decode";
    let decode = |_| DbError::Transport { op };
    let id: i64 = row.try_get("id").map_err(decode)?;
    let kind: String = row.try_get("kind").map_err(decode)?;
    let character_id: Option<i64> = row.try_get("character_id").map_err(decode)?;
    let coherent = |what: &'static str| DbError::ConstraintViolation {
        table: "delivery_queue",
        constraint: what,
    };
    let character_id = character_id.ok_or_else(|| coherent("delivery_queue_character"))?;
    let grant = match kind.as_str() {
        "item" => Grant::Item {
            character_id,
            item_id: row
                .try_get::<Option<i32>, _>("item_id")
                .map_err(decode)?
                .map(i64::from)
                .ok_or_else(|| coherent("delivery_queue_item_coherent"))?,
            qty: row
                .try_get::<Option<i32>, _>("qty")
                .map_err(decode)?
                .map(i64::from)
                .ok_or_else(|| coherent("delivery_queue_item_coherent"))?,
        },
        "gold" => Grant::Gold {
            character_id,
            amount: row
                .try_get::<Option<i64>, _>("gold_amount")
                .map_err(decode)?
                .ok_or_else(|| coherent("delivery_queue_gold_coherent"))?,
        },
        "level" => Grant::Level {
            character_id,
            delta: row
                .try_get::<Option<i32>, _>("level_delta")
                .map_err(decode)?
                .map(i64::from)
                .ok_or_else(|| coherent("delivery_queue_level_coherent"))?,
        },
        _ => return Err(coherent("delivery_queue_kind")),
    };
    Ok((id, grant))
}
