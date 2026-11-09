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
use crate::model::{Account, AccountModeration, Character, Role};
use crate::pool::pg::{map_error, PgBackend};
use crate::repo::accounts::CredentialRecord;
use crate::repo::audit::{AuditRow, AuditTrail, NewAuditEntry};
use crate::repo::economy::{CityTaxTotal, Economy, EconomySummary, GrantTotal};
use crate::repo::sessions::ClaimOutcome;
use crate::repo::{Accounts, Characters, Grant, Grants, Sessions};

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
        // O bloqueio de moderação entra na mesma consulta da credencial: a borda de
        // autenticação não decide sozinha se a conta pode entrar.
        let row = sqlx::query(
            "SELECT a.id, a.login, a.role, a.password_hash, \
             COALESCE(m.blocked, false) AS blocked \
             FROM accounts a \
             LEFT JOIN account_moderation m ON m.account_id = a.id \
             WHERE a.login = $1",
        )
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
        let blocked: bool = row.try_get("blocked").map_err(|_| DbError::Transport {
            op: "accounts.credentials.blocked",
        })?;
        Ok(CredentialRecord {
            account: account_row(&row)?,
            password_hash,
            blocked,
        })
    }

    async fn pin_hash(&self, login: &str) -> Result<Option<String>, DbError> {
        let row = sqlx::query("SELECT pin_hash FROM accounts WHERE login = $1")
            .bind(login)
            .fetch_optional(self.pool())
            .await
            .map_err(map_error("accounts.pin_hash", "accounts"))?;
        let Some(row) = row else {
            return Err(DbError::NotFoundKey { entity: "account" });
        };
        row.try_get::<Option<String>, _>("pin_hash")
            .map_err(|_| DbError::Transport {
                op: "accounts.pin_hash.decode",
            })
    }

    async fn pin_guard(&self, login: &str) -> Result<(i64, i64), DbError> {
        let row = sqlx::query(
            "SELECT pin_failures::bigint AS falhas, \
             COALESCE(GREATEST(EXTRACT(EPOCH FROM (pin_locked_until - now()))::bigint, 0), 0) AS restante \
             FROM accounts WHERE login = $1",
        )
        .bind(login)
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("accounts.pin_guard", "accounts"))?;
        let Some(row) = row else {
            return Err(DbError::NotFoundKey { entity: "account" });
        };
        let falhas: i64 = row.try_get("falhas").map_err(|_| DbError::Transport {
            op: "accounts.pin_guard.falhas",
        })?;
        let restante: i64 = row.try_get("restante").map_err(|_| DbError::Transport {
            op: "accounts.pin_guard.restante",
        })?;
        Ok((falhas, restante))
    }

    async fn pin_failure(
        &self,
        login: &str,
        limite: i64,
        segundos: i64,
    ) -> Result<(i64, i64), DbError> {
        let row = sqlx::query(
            "UPDATE accounts SET \
               pin_failures = pin_failures + 1, \
               pin_locked_until = CASE \
                 WHEN pin_failures + 1 >= $2 THEN now() + ($3 * interval '1 second') \
                 ELSE pin_locked_until END \
             WHERE login = $1 \
             RETURNING pin_failures::bigint AS falhas, \
               COALESCE(GREATEST(EXTRACT(EPOCH FROM (pin_locked_until - now()))::bigint, 0), 0) AS restante",
        )
        .bind(login)
        .bind(i16::try_from(limite).unwrap_or(i16::MAX))
        .bind(segundos)
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("accounts.pin_failure", "accounts"))?;
        let Some(row) = row else {
            return Err(DbError::NotFoundKey { entity: "account" });
        };
        Ok((
            row.try_get("falhas").map_err(|_| DbError::Transport {
                op: "accounts.pin_failure.falhas",
            })?,
            row.try_get("restante").map_err(|_| DbError::Transport {
                op: "accounts.pin_failure.restante",
            })?,
        ))
    }

    async fn pin_success(&self, login: &str) -> Result<(), DbError> {
        let feito = sqlx::query(
            "UPDATE accounts SET pin_failures = 0, pin_locked_until = NULL WHERE login = $1",
        )
        .bind(login)
        .execute(self.pool())
        .await
        .map_err(map_error("accounts.pin_success", "accounts"))?;
        if feito.rows_affected() == 0 {
            return Err(DbError::NotFoundKey { entity: "account" });
        }
        Ok(())
    }

    async fn set_pin_hash(&self, login: &str, pin_hash: &str) -> Result<(), DbError> {
        let feito = sqlx::query("UPDATE accounts SET pin_hash = $2 WHERE login = $1")
            .bind(login)
            .bind(pin_hash)
            .execute(self.pool())
            .await
            .map_err(map_error("accounts.set_pin_hash", "accounts"))?;
        if feito.rows_affected() == 0 {
            return Err(DbError::NotFoundKey { entity: "account" });
        }
        Ok(())
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

    fn moderation_state<'a>(
        &'a self,
        login: &'a str,
    ) -> std::pin::Pin<
        Box<dyn std::future::Future<Output = Result<AccountModeration, DbError>> + Send + 'a>,
    > {
        Box::pin(async move {
            let sql =
                format!("SELECT {MODERATION_COLUMNS} FROM {MODERATION_FROM} WHERE a.login = $1");
            let row = sqlx::query(&sql)
                .bind(login)
                .fetch_optional(self.pool())
                .await
                .map_err(map_error("accounts.moderation_state", "accounts"))?;
            let Some(row) = row else {
                return Err(DbError::NotFoundKey { entity: "account" });
            };
            moderation_row(&row)
        })
    }

    fn set_role<'a>(
        &'a self,
        login: &'a str,
        role: Role,
    ) -> std::pin::Pin<
        Box<dyn std::future::Future<Output = Result<AccountModeration, DbError>> + Send + 'a>,
    > {
        Box::pin(async move {
            // Uma instrução: o UPDATE e a leitura do estado novo na mesma foto.
            let sql = format!(
                "WITH alvo AS (\
                   UPDATE accounts SET role = $2 WHERE login = $1 RETURNING id, login, role\
                 ) \
                 SELECT {MODERATION_COLUMNS} FROM alvo a \
                 LEFT JOIN account_moderation m ON m.account_id = a.id"
            );
            let row = sqlx::query(&sql)
                .bind(login)
                .bind(role.as_db())
                .fetch_optional(self.pool())
                .await
                .map_err(map_error("accounts.set_role", "accounts"))?;
            let Some(row) = row else {
                return Err(DbError::NotFoundKey { entity: "account" });
            };
            moderation_row(&row)
        })
    }

    fn set_blocked<'a>(
        &'a self,
        login: &'a str,
        blocked: bool,
        reason: &'a str,
        by: i64,
    ) -> std::pin::Pin<
        Box<dyn std::future::Future<Output = Result<AccountModeration, DbError>> + Send + 'a>,
    > {
        Box::pin(async move {
            // `blocked`/`reason`/`blocked_at` saem do `RETURNING` do upsert, não da
            // tabela: no PostgreSQL todas as sub-instruções de um `WITH` veem o
            // mesmo snapshot, então reler `account_moderation` aqui devolveria o
            // estado **anterior** (bloqueio respondia `blocked:false`).
            let sql = "WITH alvo AS (SELECT id FROM accounts WHERE login = $1), \
                 gravado AS (\
                   INSERT INTO account_moderation (account_id, blocked, reason, blocked_by, blocked_at) \
                   SELECT id, $2, $3, $4, CASE WHEN $2 THEN now() ELSE NULL END FROM alvo \
                   ON CONFLICT (account_id) DO UPDATE \
                   SET blocked = EXCLUDED.blocked, reason = EXCLUDED.reason, \
                       blocked_by = EXCLUDED.blocked_by, blocked_at = EXCLUDED.blocked_at \
                   RETURNING account_id, blocked, reason, \
                     (EXTRACT(EPOCH FROM blocked_at) * 1000)::bigint AS blocked_at_ms\
                 ) \
                 SELECT a.id, a.login, a.role, g.blocked, g.reason, g.blocked_at_ms \
                 FROM accounts a JOIN gravado g ON g.account_id = a.id";
            let row = sqlx::query(sql)
                .bind(login)
                .bind(blocked)
                .bind(reason)
                .bind(by)
                .fetch_optional(self.pool())
                .await
                .map_err(map_error("accounts.set_blocked", "accounts"))?;
            let Some(row) = row else {
                return Err(DbError::NotFoundKey { entity: "account" });
            };
            moderation_row(&row)
        })
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
    "id, account_id, slot, name, class, level, exp, hp, mp, pos_x, pos_y, \
     gold, hp_max, mp_max, pvp_points";

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
        let mut personagens: Vec<Character> =
            rows.iter().map(character_row).collect::<Result<_, _>>()?;
        // Guilda é relação (`guild_members`), não coluna: um SELECT só para a conta.
        let guildas = sqlx::query(
            "SELECT gm.character_id, g.id, g.name FROM guild_members gm \
             JOIN guilds g ON g.id = gm.guild_id \
             JOIN characters c ON c.id = gm.character_id \
             WHERE c.account_id = $1 AND c.deleted_at IS NULL",
        )
        .bind(account_id)
        .fetch_all(self.pool())
        .await
        .map_err(map_error(
            "characters.list_by_account.guilds",
            "guild_members",
        ))?;
        for linha in &guildas {
            let id: i64 = linha.get("character_id");
            if let Some(personagem) = personagens.iter_mut().find(|p| p.id == id) {
                personagem.guild_id = Some(linha.get("id"));
                personagem.guild_name = linha.get("name");
            }
        }
        Ok(personagens)
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

    async fn search_by_name(&self, term: &str, limit: usize) -> Result<Vec<Character>, DbError> {
        let sql = format!(
            "SELECT {CHARACTER_COLUMNS} FROM characters \
             WHERE deleted_at IS NULL AND name ILIKE '%' || $1 || '%' \
             ORDER BY name ASC LIMIT $2"
        );
        let rows = sqlx::query(&sql)
            .bind(term)
            .bind(i64::try_from(limit).unwrap_or(i64::MAX))
            .fetch_all(self.pool())
            .await
            .map_err(map_error("characters.search_by_name", "characters"))?;
        rows.iter().map(character_row).collect()
    }

    async fn create_character(&self, character: &Character) -> Result<Character, DbError> {
        character.validate()?;
        let sql = format!(
            "INSERT INTO characters \
             (account_id, slot, name, class, level, exp, hp, mp, pos_x, pos_y, \
              gold, hp_max, mp_max, pvp_points) \
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) \
             RETURNING {CHARACTER_COLUMNS}"
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
            .bind(character.gold)
            .bind(i32::try_from(character.hp_max).unwrap_or(0))
            .bind(i32::try_from(character.mp_max).unwrap_or(0))
            .bind(i32::try_from(character.pvp_points).unwrap_or(0))
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
        gold: row.try_get("gold").map_err(decode)?,
        hp_max: i64::from(row.try_get::<i32, _>("hp_max").map_err(decode)?),
        mp_max: i64::from(row.try_get::<i32, _>("mp_max").map_err(decode)?),
        pvp_points: i64::from(row.try_get::<i32, _>("pvp_points").map_err(decode)?),
        ..Character::default()
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

    async fn enqueue(&self, key: &str, by: i64, grant: &Grant) -> Result<(i64, bool), DbError> {
        let (kind, character_id, item_id, qty, gold_amount, level_delta) = match grant {
            Grant::Item {
                character_id,
                item_id,
                qty,
            } => (
                "item",
                *character_id,
                Some(*item_id),
                Some(*qty),
                None,
                None,
            ),
            Grant::Gold {
                character_id,
                amount,
            } => ("gold", *character_id, None, None, Some(*amount), None),
            Grant::Level {
                character_id,
                delta,
            } => ("level", *character_id, None, None, None, Some(*delta)),
        };
        // A conta vem do personagem (nunca do corpo do pedido): o grant pertence a
        // quem é dono do destino. Personagem inexistente/apagado é `404` de domínio.
        let conta =
            sqlx::query("SELECT account_id FROM characters WHERE id = $1 AND deleted_at IS NULL")
                .bind(character_id)
                .fetch_optional(self.pool())
                .await
                .map_err(map_error("grants.enqueue", "characters"))?;
        let Some(conta) = conta else {
            return Err(DbError::NotFoundKey {
                entity: "character",
            });
        };
        let account_id: i64 = conta
            .try_get("account_id")
            .map_err(|_| DbError::Transport {
                op: "grants.enqueue.decode",
            })?;
        let inserido = sqlx::query(
            "INSERT INTO delivery_queue \
               (kind, account_id, character_id, item_id, qty, gold_amount, level_delta, \
                idempotency_key, created_by) \
             VALUES ($1, $2, $3, $4::int, $5::int, $6, $7::int, $8, $9) \
             ON CONFLICT (idempotency_key) DO NOTHING \
             RETURNING id",
        )
        .bind(kind)
        .bind(account_id)
        .bind(character_id)
        .bind(item_id)
        .bind(qty)
        .bind(gold_amount)
        .bind(level_delta)
        .bind(key)
        .bind(by)
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("grants.enqueue", "delivery_queue"))?;
        if let Some(row) = inserido {
            let id = row.try_get("id").map_err(|_| DbError::Transport {
                op: "grants.enqueue.decode",
            })?;
            return Ok((id, false));
        }
        // Conflito: a chave já existia **antes** deste pedido — replay.
        let existente = sqlx::query("SELECT id FROM delivery_queue WHERE idempotency_key = $1")
            .bind(key)
            .fetch_optional(self.pool())
            .await
            .map_err(map_error("grants.enqueue", "delivery_queue"))?;
        let Some(row) = existente else {
            return Err(DbError::NotFoundKey {
                entity: "delivery_queue",
            });
        };
        let id = row.try_get("id").map_err(|_| DbError::Transport {
            op: "grants.enqueue.decode",
        })?;
        Ok((id, true))
    }

    async fn claim_gold(&self, id: i64) -> Result<Option<(i64, i64)>, DbError> {
        let row = sqlx::query(
            "WITH alvo AS (\
               UPDATE delivery_queue \
               SET status = 'claimed', claimed_at = now() \
               WHERE id = $1 AND status = 'pending' AND kind = 'gold' \
               RETURNING character_id, gold_amount\
             ) \
             UPDATE characters c \
             SET gold = c.gold + alvo.gold_amount, updated_at = now() \
             FROM alvo WHERE c.id = alvo.character_id \
             RETURNING c.id, c.gold",
        )
        .bind(id)
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("grants.claim_gold", "delivery_queue"))?;
        let Some(row) = row else {
            return Ok(None);
        };
        let character_id = row.try_get("id").map_err(|_| DbError::Transport {
            op: "grants.claim_gold.decode",
        })?;
        let gold = row.try_get("gold").map_err(|_| DbError::Transport {
            op: "grants.claim_gold.decode",
        })?;
        Ok(Some((character_id, gold)))
    }

    async fn list_for_character(
        &self,
        character_id: i64,
        limit: usize,
    ) -> Result<Vec<crate::repo::QueuedGrantRow>, DbError> {
        self.queued_for_character(character_id, limit).await
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

/// Colunas do estado de moderação, na ordem de todo `SELECT` deste módulo.
const MODERATION_COLUMNS: &str = "a.id, a.login, a.role, COALESCE(m.blocked, false) AS blocked, \
     COALESCE(m.reason, '') AS reason, \
     (EXTRACT(EPOCH FROM m.blocked_at) * 1000)::bigint AS blocked_at_ms";

/// `FROM` comum: conta + moderação (linha ausente = conta liberada).
const MODERATION_FROM: &str = "accounts a LEFT JOIN account_moderation m ON m.account_id = a.id";

fn moderation_row(row: &PgRow) -> Result<AccountModeration, DbError> {
    let blocked: bool = row.try_get("blocked").map_err(|_| DbError::Transport {
        op: "moderation.decode",
    })?;
    let reason: String = row.try_get("reason").map_err(|_| DbError::Transport {
        op: "moderation.decode",
    })?;
    let blocked_at_ms: Option<i64> =
        row.try_get("blocked_at_ms")
            .map_err(|_| DbError::Transport {
                op: "moderation.decode",
            })?;
    Ok(AccountModeration {
        account: account_row(row)?,
        blocked,
        reason,
        blocked_at_ms: blocked_at_ms.and_then(|ms| u64::try_from(ms).ok()),
    })
}

impl AuditTrail for PgBackend {
    fn record<'a>(
        &'a self,
        entry: &'a NewAuditEntry,
    ) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), DbError>> + Send + 'a>> {
        Box::pin(async move {
            // `actor` mantém a coluna TEXT da 0005: `id:role`, separável na leitura.
            let actor = format!("{}:{}", entry.actor_id, entry.actor_role.as_db());
            sqlx::query(
                "INSERT INTO audit_log (actor, action, target, detail) VALUES ($1, $2, $3, $4)",
            )
            .bind(actor)
            .bind(&entry.action)
            .bind(&entry.target)
            .bind(&entry.detail)
            .execute(self.pool())
            .await
            .map_err(map_error("audit.record", "audit_log"))?;
            Ok(())
        })
    }

    fn recent(
        &self,
        limit: usize,
    ) -> std::pin::Pin<
        Box<dyn std::future::Future<Output = Result<Vec<AuditRow>, DbError>> + Send + '_>,
    > {
        Box::pin(async move {
            // `actor` legado (`system`) não tem id: a máscara evita cast inválido.
            let rows = sqlx::query(
                "SELECT (EXTRACT(EPOCH FROM at) * 1000)::bigint AS at_ms, \
                 CASE WHEN actor ~ '^[0-9]+:' THEN split_part(actor, ':', 1)::bigint ELSE 0 END AS actor_id, \
                 CASE WHEN actor ~ '^[0-9]+:' THEN split_part(actor, ':', 2) ELSE actor END AS actor_role, \
                 action, target, detail \
                 FROM audit_log ORDER BY at DESC, id DESC LIMIT $1",
            )
            .bind(i64::try_from(limit).unwrap_or(i64::MAX))
            .fetch_all(self.pool())
            .await
            .map_err(map_error("audit.recent", "audit_log"))?;
            rows.iter()
                .map(|row| {
                    let at_ms: i64 = row.try_get("at_ms").map_err(|_| DbError::Transport {
                        op: "audit.recent.decode",
                    })?;
                    Ok(AuditRow {
                        at_ms: u64::try_from(at_ms).unwrap_or(0),
                        actor_id: row.try_get("actor_id").map_err(|_| DbError::Transport {
                            op: "audit.recent.decode",
                        })?,
                        actor_role: row.try_get("actor_role").map_err(|_| DbError::Transport {
                            op: "audit.recent.decode",
                        })?,
                        action: row.try_get("action").map_err(|_| DbError::Transport {
                            op: "audit.recent.decode",
                        })?,
                        target: row.try_get("target").map_err(|_| DbError::Transport {
                            op: "audit.recent.decode",
                        })?,
                        detail: row.try_get("detail").map_err(|_| DbError::Transport {
                            op: "audit.recent.decode",
                        })?,
                    })
                })
                .collect()
        })
    }
}

impl PgBackend {
    async fn queued_for_character(
        &self,
        character_id: i64,
        limit: usize,
    ) -> Result<Vec<crate::repo::QueuedGrantRow>, DbError> {
        let rows = sqlx::query(
            "SELECT id, kind, item_id, qty, gold_amount, level_delta, status, \
                    idempotency_key, created_by, \
                    (extract(epoch FROM created_at) * 1000)::bigint AS at_ms \
             FROM delivery_queue WHERE character_id = $1 ORDER BY id DESC LIMIT $2",
        )
        .bind(character_id)
        .bind(i64::try_from(limit).unwrap_or(i64::MAX))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("grants.list_for_character", "delivery_queue"))?;
        let op = "grants.list_for_character.decode";
        let decode = |_| DbError::Transport { op };
        rows.iter()
            .map(|row| {
                Ok(crate::repo::QueuedGrantRow {
                    id: row.try_get("id").map_err(decode)?,
                    kind: row.try_get("kind").map_err(decode)?,
                    item_id: row.try_get("item_id").map_err(decode)?,
                    qty: row.try_get("qty").map_err(decode)?,
                    gold: row
                        .try_get::<Option<i64>, _>("gold_amount")
                        .map_err(decode)?
                        .unwrap_or(0),
                    level_delta: row.try_get("level_delta").map_err(decode)?,
                    status: row.try_get("status").map_err(decode)?,
                    idempotency_key: row.try_get("idempotency_key").map_err(decode)?,
                    created_by: row.try_get("created_by").map_err(decode)?,
                    at_ms: row.try_get("at_ms").map_err(decode)?,
                })
            })
            .collect()
    }
}

impl Economy for PgBackend {
    fn summary(
        &self,
    ) -> std::pin::Pin<
        Box<dyn std::future::Future<Output = Result<EconomySummary, DbError>> + Send + '_>,
    > {
        Box::pin(async move {
            let cidades = sqlx::query(
                "SELECT city_code, COALESCE(SUM(amount), 0)::bigint AS total \
                 FROM city_tax_ledger GROUP BY city_code ORDER BY city_code",
            )
            .fetch_all(self.pool())
            .await
            .map_err(map_error("economy.tax", "city_tax_ledger"))?;
            let tax_by_city = cidades
                .iter()
                .map(|row| {
                    Ok(CityTaxTotal {
                        city_code: row.try_get("city_code").map_err(|_| DbError::Transport {
                            op: "economy.tax.decode",
                        })?,
                        total: row.try_get("total").map_err(|_| DbError::Transport {
                            op: "economy.tax.decode",
                        })?,
                    })
                })
                .collect::<Result<Vec<_>, DbError>>()?;

            let grants = sqlx::query(
                "SELECT kind, count(*)::bigint AS total, \
                 COALESCE(SUM(gold_amount), 0)::bigint AS gold \
                 FROM delivery_queue GROUP BY kind ORDER BY kind",
            )
            .fetch_all(self.pool())
            .await
            .map_err(map_error("economy.grants", "delivery_queue"))?;
            let grants = grants
                .iter()
                .map(|row| {
                    Ok(GrantTotal {
                        kind: row.try_get("kind").map_err(|_| DbError::Transport {
                            op: "economy.grants.decode",
                        })?,
                        count: row.try_get("total").map_err(|_| DbError::Transport {
                            op: "economy.grants.decode",
                        })?,
                        gold: row.try_get("gold").map_err(|_| DbError::Transport {
                            op: "economy.grants.decode",
                        })?,
                    })
                })
                .collect::<Result<Vec<_>, DbError>>()?;

            Ok(EconomySummary {
                tax_by_city,
                grants,
            })
        })
    }
}

impl Sessions for PgBackend {
    async fn claim_session(
        &self,
        login: &str,
        group_id: i16,
        channel: i16,
        client_version: i32,
        stale_after_secs: i64,
    ) -> Result<ClaimOutcome, DbError> {
        // Uma transação: primeiro libera a sessão órfã (dono parou de renovar), depois tenta
        // abrir. O índice parcial `character_session_uma_por_conta` é quem decide o conflito —
        // nenhuma checagem em memória pode substituí-lo, porque são processos diferentes.
        let mut tx = self
            .pool()
            .begin()
            .await
            .map_err(map_error("sessions.begin", "character_session"))?;

        sqlx::query(
            "UPDATE character_session SET released_at = now() \
             WHERE account_login = $1 AND released_at IS NULL \
               AND renewed_at < now() - make_interval(secs => $2)",
        )
        .bind(login)
        .bind(stale_after_secs)
        .execute(&mut *tx)
        .await
        .map_err(map_error("sessions.expire", "character_session"))?;

        let aberta = sqlx::query(
            "INSERT INTO character_session (account_login, group_id, channel, client_version) \
             VALUES ($1, $2, $3, $4) \
             ON CONFLICT (account_login) WHERE released_at IS NULL DO NOTHING \
             RETURNING id",
        )
        .bind(login)
        .bind(group_id)
        .bind(channel)
        .bind(client_version)
        .fetch_optional(&mut *tx)
        .await
        .map_err(map_error("sessions.claim", "character_session"))?;

        if let Some(row) = aberta {
            let id: i64 = row.try_get("id").map_err(|_| DbError::Transport {
                op: "sessions.decode",
            })?;
            tx.commit()
                .await
                .map_err(map_error("sessions.commit", "character_session"))?;
            return Ok(ClaimOutcome::Acquired { id });
        }

        // Conflito: quem está vivo hoje nesta conta (só o canal entra no log/recusa).
        let viva = sqlx::query(
            "SELECT channel FROM character_session \
             WHERE account_login = $1 AND released_at IS NULL ORDER BY id DESC LIMIT 1",
        )
        .bind(login)
        .fetch_optional(&mut *tx)
        .await
        .map_err(map_error("sessions.conflict", "character_session"))?;
        tx.commit()
            .await
            .map_err(map_error("sessions.commit", "character_session"))?;

        let channel = viva
            .map(|row| row.try_get::<i16, _>("channel"))
            .transpose()
            .map_err(|_| DbError::Transport {
                op: "sessions.decode",
            })?
            .unwrap_or(channel);
        Ok(ClaimOutcome::Conflict { channel })
    }

    async fn renew_session(&self, id: i64) -> Result<(), DbError> {
        sqlx::query(
            "UPDATE character_session SET renewed_at = now() WHERE id = $1 AND released_at IS NULL",
        )
        .bind(id)
        .execute(self.pool())
        .await
        .map_err(map_error("sessions.renew", "character_session"))?;
        Ok(())
    }

    async fn release_session(&self, id: i64) -> Result<(), DbError> {
        // Idempotente de propósito: a liberação roda no `Drop` do dono e pode repetir.
        sqlx::query(
            "UPDATE character_session SET released_at = now() \
             WHERE id = $1 AND released_at IS NULL",
        )
        .bind(id)
        .execute(self.pool())
        .await
        .map_err(map_error("sessions.release", "character_session"))?;
        Ok(())
    }

    async fn live_sessions(&self, group_id: i16, channel: i16) -> Result<u64, DbError> {
        let row = sqlx::query(
            "SELECT count(*) AS total FROM character_session \
             WHERE group_id = $1 AND channel = $2 AND released_at IS NULL",
        )
        .bind(group_id)
        .bind(channel)
        .fetch_one(self.pool())
        .await
        .map_err(map_error("sessions.live", "character_session"))?;
        let total: i64 = row.try_get("total").map_err(|_| DbError::Transport {
            op: "sessions.decode",
        })?;
        Ok(u64::try_from(total).unwrap_or(0))
    }
}
