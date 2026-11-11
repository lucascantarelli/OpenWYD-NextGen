//! Implementação `pg` das portas — o adapter vive aqui, as portas nos arquivos irmãos.
//!
//! Cada operação é **uma** transação curta e devolve erro tipado: violação de constraint
//! vira `ConstraintViolation` (o chamador corrige o dado), linha ausente vira `NotFound`
//! (nunca `None` silencioso). Nenhuma string de SQL é montada com entrada do usuário:
//! tudo passa por bind.
//!
//! `delivery_queue` é lida **e baixada** aqui (`pending`/`mark_claimed`): a web enfileira,
//! o servidor drena — a única ponte permitida (`AGENTS.md` §3.7).

use std::collections::HashMap;

use crate::error::DbError;
use crate::model::{
    Account, AccountModeration, Character, ItemRow, Progression, Role, SHORT_SKILL_LEN,
};
use crate::pool::pg::{map_error, PgBackend};
use crate::repo::accounts::CredentialRecord;
use crate::repo::audit::{AuditRow, AuditTrail, NewAuditEntry};
use crate::repo::content::{ChannelCapacity, ChannelRole, SkillRule, SkillRules};
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

    async fn store_vault_items(&self, account_id: i64, items: &[ItemRow]) -> Result<(), DbError> {
        if items.is_empty() {
            return Ok(());
        }
        let mut tx = self.begin().await?;
        for item in items {
            sqlx::query(
                "INSERT INTO vault_items \
                 (account_id, slot, item_id, qty, flags, refine, ancient, gem_effect, gem_value) \
                 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) \
                 ON CONFLICT (account_id, slot) DO UPDATE SET \
                 item_id = EXCLUDED.item_id, qty = EXCLUDED.qty, flags = EXCLUDED.flags, \
                 refine = EXCLUDED.refine, ancient = EXCLUDED.ancient, \
                 gem_effect = EXCLUDED.gem_effect, gem_value = EXCLUDED.gem_value",
            )
            .bind(account_id)
            .bind(i16::try_from(item.slot).unwrap_or(i16::MAX))
            .bind(i32::try_from(item.item_id).unwrap_or(0))
            .bind(i32::try_from(item.qty).unwrap_or(1))
            .bind(i32::try_from(item.flags).unwrap_or(0))
            .bind(i16::try_from(item.refine).unwrap_or(0))
            .bind(item.ancient)
            .bind(i16::try_from(item.gem_effect).unwrap_or(0))
            .bind(i16::try_from(item.gem_value).unwrap_or(0))
            .execute(&mut *tx)
            .await
            .map_err(map_error("accounts.store_vault_items", "vault_items"))?;
        }
        tx.commit()
            .await
            .map_err(map_error("accounts.store_vault_items", "vault_items"))?;
        Ok(())
    }

    async fn set_account_coin(&self, account_id: i64, coin: i64) -> Result<(), DbError> {
        if !(0..=i64::from(i32::MAX)).contains(&coin) {
            return Err(DbError::ConstraintViolation {
                table: "account_wallet",
                constraint: "account_wallet_coin",
            });
        }
        sqlx::query(
            "INSERT INTO account_wallet (account_id, coin) VALUES ($1, $2) \
             ON CONFLICT (account_id) DO UPDATE SET coin = EXCLUDED.coin",
        )
        .bind(account_id)
        .bind(coin)
        .execute(self.pool())
        .await
        .map_err(map_error("accounts.set_account_coin", "account_wallet"))?;
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

    async fn vault_items(&self, account_id: i64) -> Result<Vec<ItemRow>, DbError> {
        let rows = sqlx::query(
            "SELECT slot, item_id, qty, flags, refine, ancient, gem_effect, gem_value \
             FROM vault_items WHERE account_id = $1 ORDER BY slot",
        )
        .bind(account_id)
        .fetch_all(self.pool())
        .await
        .map_err(map_error("accounts.vault_items", "vault_items"))?;
        rows.iter().map(item_row).collect()
    }

    async fn account_coin(&self, account_id: i64) -> Result<Option<i64>, DbError> {
        let row = sqlx::query("SELECT coin FROM account_wallet WHERE account_id = $1")
            .bind(account_id)
            .fetch_optional(self.pool())
            .await
            .map_err(map_error("accounts.account_coin", "account_wallet"))?;
        row.map(|row| {
            row.try_get::<i64, _>("coin")
                .map_err(|_| DbError::Transport {
                    op: "account_wallet.decode",
                })
        })
        .transpose()
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
        // Atributos moram na tabela 1:1 `character_scores` (0032): um SELECT só para a
        // conta. Personagem antigo sem linha fica com os defaults zero, nunca com lixo.
        let scores = sqlx::query(
            "SELECT cs.character_id, cs.strength, cs.intelligence, cs.dexterity, \
             cs.constitution, cs.score_bonus, cs.skill_bonus \
             FROM character_scores cs JOIN characters c ON c.id = cs.character_id \
             WHERE c.account_id = $1 AND c.deleted_at IS NULL",
        )
        .bind(account_id)
        .fetch_all(self.pool())
        .await
        .map_err(map_error("characters.list_scores", "character_scores"))?;
        let mut por_id = HashMap::with_capacity(scores.len());
        for row in &scores {
            let decode = |_| DbError::Transport {
                op: "character_scores.decode",
            };
            por_id.insert(
                row.try_get::<i64, _>("character_id").map_err(decode)?,
                (
                    i64::from(row.try_get::<i16, _>("strength").map_err(decode)?),
                    i64::from(row.try_get::<i16, _>("intelligence").map_err(decode)?),
                    i64::from(row.try_get::<i16, _>("dexterity").map_err(decode)?),
                    i64::from(row.try_get::<i16, _>("constitution").map_err(decode)?),
                    i64::from(row.try_get::<i32, _>("score_bonus").map_err(decode)?),
                    i64::from(row.try_get::<i32, _>("skill_bonus").map_err(decode)?),
                ),
            );
        }
        for personagem in &mut personagens {
            if let Some((forca, int, dex, con, pontos, skills)) = por_id.get(&personagem.id) {
                personagem.strength = *forca;
                personagem.intelligence = *int;
                personagem.dexterity = *dex;
                personagem.constitution = *con;
                personagem.score_bonus = *pontos;
                personagem.skill_bonus = *skills;
            }
        }
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
        let mut tx = self.begin().await?;
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
            .fetch_one(&mut *tx)
            .await
            .map_err(map_error("characters.create_character", "characters"))?;
        let id: i64 = row
            .try_get("id")
            .map_err(map_error("characters.create_character.id", "characters"))?;
        sqlx::query(
            "INSERT INTO character_scores \
             (character_id, strength, intelligence, dexterity, constitution, \
              score_bonus, skill_bonus) \
             VALUES ($1, $2, $3, $4, $5, $6, $7)",
        )
        .bind(id)
        .bind(i16::try_from(character.strength).unwrap_or(0))
        .bind(i16::try_from(character.intelligence).unwrap_or(0))
        .bind(i16::try_from(character.dexterity).unwrap_or(0))
        .bind(i16::try_from(character.constitution).unwrap_or(0))
        .bind(i32::try_from(character.score_bonus).unwrap_or(0))
        .bind(i32::try_from(character.skill_bonus).unwrap_or(0))
        .execute(&mut *tx)
        .await
        .map_err(map_error("characters.create_scores", "character_scores"))?;
        tx.commit()
            .await
            .map_err(map_error("characters.create_commit", "characters"))?;
        let mut criado = character_row(&row)?;
        criado.strength = character.strength;
        criado.intelligence = character.intelligence;
        criado.dexterity = character.dexterity;
        criado.constitution = character.constitution;
        criado.score_bonus = character.score_bonus;
        criado.skill_bonus = character.skill_bonus;
        Ok(criado)
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

    async fn items(&self, character_id: i64) -> Result<Vec<ItemRow>, DbError> {
        let rows = sqlx::query(
            "SELECT slot, item_id, qty, flags, refine, ancient, gem_effect, gem_value \
             FROM character_items WHERE character_id = $1 ORDER BY slot",
        )
        .bind(character_id)
        .fetch_all(self.pool())
        .await
        .map_err(map_error("characters.items", "character_items"))?;
        rows.iter().map(item_row).collect()
    }

    async fn insert_items(&self, character_id: i64, items: &[ItemRow]) -> Result<(), DbError> {
        if items.is_empty() {
            return Ok(());
        }
        // Uma transação para o conjunto: ou o inventário inicial entra inteiro, ou nada.
        let mut tx = self.begin().await?;
        for item in items {
            sqlx::query(
                "INSERT INTO character_items \
                 (character_id, slot, item_id, qty, flags, refine, ancient, gem_effect, gem_value) \
                 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) \
                 ON CONFLICT (character_id, slot) DO NOTHING",
            )
            .bind(character_id)
            .bind(i16::try_from(item.slot).unwrap_or(i16::MAX))
            .bind(i32::try_from(item.item_id).unwrap_or(0))
            .bind(i32::try_from(item.qty).unwrap_or(1))
            .bind(i32::try_from(item.flags).unwrap_or(0))
            .bind(i16::try_from(item.refine).unwrap_or(0))
            .bind(item.ancient)
            .bind(i16::try_from(item.gem_effect).unwrap_or(0))
            .bind(i16::try_from(item.gem_value).unwrap_or(0))
            .execute(&mut *tx)
            .await
            .map_err(map_error("characters.insert_items", "character_items"))?;
        }
        tx.commit()
            .await
            .map_err(map_error("characters.insert_items", "character_items"))?;
        Ok(())
    }

    async fn refine_pity(&self, character_id: i64, slot: i64) -> Result<u8, DbError> {
        let slot = i16::try_from(slot).map_err(|_| DbError::Transport {
            op: "characters.refine_pity",
        })?;
        let pity: Option<i16> = sqlx::query_scalar(
            "SELECT pity FROM refine_pity WHERE character_id = $1 AND slot = $2",
        )
        .bind(character_id)
        .bind(slot)
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("characters.refine_pity", "refine_pity"))?;
        Ok(pity.unwrap_or(0).clamp(0, 20) as u8)
    }

    async fn replace_items(
        &self,
        character_id: i64,
        items: &[ItemRow],
        pity: &[(i64, u8)],
    ) -> Result<(), DbError> {
        let mut tx = self.begin().await?;
        sqlx::query("DELETE FROM character_items WHERE character_id = $1")
            .bind(character_id)
            .execute(&mut *tx)
            .await
            .map_err(map_error("characters.replace_items", "character_items"))?;
        for item in items {
            sqlx::query(
                "INSERT INTO character_items \
                 (character_id, slot, item_id, qty, flags, refine, ancient, gem_effect, gem_value) \
                 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
            )
            .bind(character_id)
            .bind(i16::try_from(item.slot).unwrap_or(i16::MAX))
            .bind(i32::try_from(item.item_id).unwrap_or(0))
            .bind(i32::try_from(item.qty).unwrap_or(1))
            .bind(i32::try_from(item.flags).unwrap_or(0))
            .bind(i16::try_from(item.refine).unwrap_or(0))
            .bind(item.ancient)
            .bind(i16::try_from(item.gem_effect).unwrap_or(0))
            .bind(i16::try_from(item.gem_value).unwrap_or(0))
            .execute(&mut *tx)
            .await
            .map_err(map_error("characters.replace_items", "character_items"))?;
        }
        // Poda: slots que saíram do inventário (pó consumido, item destruído) não
        // deixam piedade órfã; a mesma transação grava item + pó + piedade.
        sqlx::query(
            "DELETE FROM refine_pity WHERE character_id = $1 AND slot NOT IN \
             (SELECT slot FROM character_items WHERE character_id = $1)",
        )
        .bind(character_id)
        .execute(&mut *tx)
        .await
        .map_err(map_error("characters.replace_items", "refine_pity"))?;
        for (slot, valor) in pity {
            let slot = i16::try_from(*slot).map_err(|_| DbError::Transport {
                op: "characters.replace_items",
            })?;
            if *valor == 0 {
                sqlx::query("DELETE FROM refine_pity WHERE character_id = $1 AND slot = $2")
                    .bind(character_id)
                    .bind(slot)
                    .execute(&mut *tx)
                    .await
                    .map_err(map_error("characters.replace_items", "refine_pity"))?;
            } else {
                sqlx::query(
                    "INSERT INTO refine_pity (character_id, slot, pity) VALUES ($1, $2, $3) \
                     ON CONFLICT (character_id, slot) DO UPDATE SET pity = EXCLUDED.pity",
                )
                .bind(character_id)
                .bind(slot)
                .bind(i16::from((*valor).min(20)))
                .execute(&mut *tx)
                .await
                .map_err(map_error("characters.replace_items", "refine_pity"))?;
            }
        }
        tx.commit()
            .await
            .map_err(map_error("characters.replace_items", "character_items"))?;
        Ok(())
    }

    async fn short_skill(
        &self,
        character_id: i64,
    ) -> Result<Option<[u8; SHORT_SKILL_LEN]>, DbError> {
        let row =
            sqlx::query("SELECT short_skill FROM character_short_skill WHERE character_id = $1")
                .bind(character_id)
                .fetch_optional(self.pool())
                .await
                .map_err(map_error("characters.short_skill", "character_short_skill"))?;
        let Some(row) = row else {
            return Ok(None);
        };
        let bytes: Vec<u8> = row.try_get("short_skill").map_err(|_| DbError::Transport {
            op: "character_short_skill.decode",
        })?;
        // O CHECK do banco garante 16 bytes; contagem divergente é dado corrompido.
        let skill: [u8; SHORT_SKILL_LEN] =
            bytes.try_into().map_err(|_| DbError::ConstraintViolation {
                table: "character_short_skill",
                constraint: "character_short_skill_len",
            })?;
        Ok(Some(skill))
    }

    async fn set_short_skill(
        &self,
        character_id: i64,
        skill: &[u8; SHORT_SKILL_LEN],
    ) -> Result<(), DbError> {
        sqlx::query(
            "INSERT INTO character_short_skill (character_id, short_skill) VALUES ($1, $2) \
             ON CONFLICT (character_id) DO UPDATE SET short_skill = EXCLUDED.short_skill",
        )
        .bind(character_id)
        .bind(skill.as_slice())
        .execute(self.pool())
        .await
        .map_err(map_error(
            "characters.set_short_skill",
            "character_short_skill",
        ))?;
        Ok(())
    }

    async fn progression(&self, character_id: i64) -> Result<Progression, DbError> {
        let row = sqlx::query(
            // Atributos e pontos moram em `character_scores` (0032); vitais e ouro em
            // `characters`; `LearnedSkill` (0035) e maestria/`SpecialBonus` (0036) em
            // tabelas 1:1. Sem linha, maestria e pontos valem zero — o `BaseMob`.
            "SELECT cs.strength::bigint AS strength, cs.intelligence::bigint AS intelligence, \
                    cs.dexterity::bigint AS dexterity, cs.constitution::bigint AS constitution, \
                    c.hp_max::bigint AS hp_max, c.mp_max::bigint AS mp_max, \
                    c.hp::bigint AS hp, c.mp::bigint AS mp, c.gold, \
                    c.level::bigint AS level, c.exp, \
                    cs.score_bonus::bigint AS score_bonus, cs.skill_bonus::bigint AS skill_bonus, \
                    COALESCE(s.learned_skill, 0) AS learned_skill, \
                    COALESCE(sp.special_0, 0)::bigint AS special_0, \
                    COALESCE(sp.special_1, 0)::bigint AS special_1, \
                    COALESCE(sp.special_2, 0)::bigint AS special_2, \
                    COALESCE(sp.special_3, 0)::bigint AS special_3, \
                    COALESCE(sp.special_bonus, 0)::bigint AS special_bonus \
             FROM characters c \
             JOIN character_scores cs ON cs.character_id = c.id \
             LEFT JOIN character_learned_skill s ON s.character_id = c.id \
             LEFT JOIN character_special sp ON sp.character_id = c.id \
             WHERE c.id = $1 AND c.deleted_at IS NULL",
        )
        .bind(character_id)
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("characters.progression", "characters"))?;
        let Some(row) = row else {
            return Err(DbError::NotFound {
                entity: "character",
                id: character_id,
            });
        };
        let op = "characters.progression.decode";
        let decode = |_| DbError::Transport { op };
        Ok(Progression {
            level: row.try_get("level").map_err(decode)?,
            exp: row.try_get("exp").map_err(decode)?,
            strength: row.try_get("strength").map_err(decode)?,
            intelligence: row.try_get("intelligence").map_err(decode)?,
            dexterity: row.try_get("dexterity").map_err(decode)?,
            constitution: row.try_get("constitution").map_err(decode)?,
            hp_max: row.try_get("hp_max").map_err(decode)?,
            mp_max: row.try_get("mp_max").map_err(decode)?,
            hp: row.try_get("hp").map_err(decode)?,
            mp: row.try_get("mp").map_err(decode)?,
            gold: row.try_get("gold").map_err(decode)?,
            score_bonus: row.try_get("score_bonus").map_err(decode)?,
            skill_bonus: row.try_get("skill_bonus").map_err(decode)?,
            learned_skill: row.try_get("learned_skill").map_err(decode)?,
            special: [
                row.try_get("special_0").map_err(decode)?,
                row.try_get("special_1").map_err(decode)?,
                row.try_get("special_2").map_err(decode)?,
                row.try_get("special_3").map_err(decode)?,
            ],
            special_bonus: row.try_get("special_bonus").map_err(decode)?,
        })
    }

    async fn quest_state(
        &self,
        character_id: i64,
        quest_code: &str,
    ) -> Result<Option<crate::model::QuestState>, DbError> {
        let row = sqlx::query(
            "SELECT step, progress, (completed_at IS NOT NULL) AS completed \
             FROM character_quest WHERE character_id = $1 AND quest_code = $2",
        )
        .bind(character_id)
        .bind(quest_code)
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("characters.quest_state", "character_quest"))?;
        let Some(row) = row else {
            return Ok(None);
        };
        let op = "characters.quest_state.decode";
        let decode = |_| DbError::Transport { op };
        let step: i16 = row.try_get("step").map_err(decode)?;
        Ok(Some(crate::model::QuestState {
            quest_code: quest_code.to_string(),
            // `step` é 1-based no banco; `stage` é o `Newbie` 0-based do legado.
            stage: step.saturating_sub(1),
            completed: row.try_get("completed").map_err(decode)?,
            reward_mask: row.try_get("progress").map_err(decode)?,
        }))
    }

    async fn store_quest_state(
        &self,
        character_id: i64,
        state: &crate::model::QuestState,
    ) -> Result<(), DbError> {
        sqlx::query(
            "INSERT INTO character_quest \
             (character_id, quest_code, step, progress, completed_at) \
             VALUES ($1, $2, $3, $4, CASE WHEN $5 THEN now() ELSE NULL END) \
             ON CONFLICT (character_id, quest_code) DO UPDATE SET \
             step = EXCLUDED.step, progress = EXCLUDED.progress, \
             completed_at = COALESCE(EXCLUDED.completed_at, character_quest.completed_at)",
        )
        .bind(character_id)
        .bind(&state.quest_code)
        .bind(state.stage.saturating_add(1))
        .bind(state.reward_mask)
        .bind(state.completed)
        .execute(self.pool())
        .await
        .map_err(map_error("characters.store_quest_state", "character_quest"))?;
        Ok(())
    }

    async fn store_progression(
        &self,
        character_id: i64,
        progression: &Progression,
    ) -> Result<(), DbError> {
        let mut tx = self.begin().await?;
        let feito = sqlx::query(
            "UPDATE characters SET hp_max = $2::int, mp_max = $3::int, hp = $4::int, \
                    mp = $5::int, gold = $6::bigint, level = $7::int, exp = $8::bigint, \
                    updated_at = now() \
             WHERE id = $1 AND deleted_at IS NULL",
        )
        .bind(character_id)
        .bind(i32::try_from(progression.hp_max).unwrap_or(0))
        .bind(i32::try_from(progression.mp_max).unwrap_or(0))
        .bind(i32::try_from(progression.hp).unwrap_or(0))
        .bind(i32::try_from(progression.mp).unwrap_or(0))
        .bind(progression.gold)
        .bind(i32::try_from(progression.level).unwrap_or(1))
        .bind(progression.exp)
        .execute(&mut *tx)
        .await
        .map_err(map_error("characters.store_progression", "characters"))?;
        if feito.rows_affected() == 0 {
            return Err(DbError::NotFound {
                entity: "character",
                id: character_id,
            });
        }
        let atualizado = sqlx::query(
            "UPDATE character_scores SET strength = $2::int, intelligence = $3::int, \
                    dexterity = $4::int, constitution = $5::int, score_bonus = $6::int, \
                    skill_bonus = $7::int \
             WHERE character_id = $1",
        )
        .bind(character_id)
        .bind(i32::try_from(progression.strength).unwrap_or(0))
        .bind(i32::try_from(progression.intelligence).unwrap_or(0))
        .bind(i32::try_from(progression.dexterity).unwrap_or(0))
        .bind(i32::try_from(progression.constitution).unwrap_or(0))
        .bind(i32::try_from(progression.score_bonus).unwrap_or(0))
        .bind(i32::try_from(progression.skill_bonus).unwrap_or(0))
        .execute(&mut *tx)
        .await
        .map_err(map_error(
            "characters.store_progression",
            "character_scores",
        ))?;
        if atualizado.rows_affected() == 0 {
            return Err(DbError::NotFound {
                entity: "character_scores",
                id: character_id,
            });
        }
        sqlx::query(
            "INSERT INTO character_learned_skill (character_id, learned_skill) VALUES ($1, $2) \
             ON CONFLICT (character_id) DO UPDATE SET \
             learned_skill = EXCLUDED.learned_skill, updated_at = now()",
        )
        .bind(character_id)
        .bind(progression.learned_skill)
        .execute(&mut *tx)
        .await
        .map_err(map_error(
            "characters.store_progression",
            "character_learned_skill",
        ))?;
        sqlx::query(
            "INSERT INTO character_special \
             (character_id, special_0, special_1, special_2, special_3, special_bonus) \
             VALUES ($1, $2, $3, $4, $5, $6) \
             ON CONFLICT (character_id) DO UPDATE SET \
             special_0 = EXCLUDED.special_0, special_1 = EXCLUDED.special_1, \
             special_2 = EXCLUDED.special_2, special_3 = EXCLUDED.special_3, \
             special_bonus = EXCLUDED.special_bonus, updated_at = now()",
        )
        .bind(character_id)
        .bind(i16::try_from(progression.special[0]).unwrap_or(0))
        .bind(i16::try_from(progression.special[1]).unwrap_or(0))
        .bind(i16::try_from(progression.special[2]).unwrap_or(0))
        .bind(i16::try_from(progression.special[3]).unwrap_or(0))
        .bind(i16::try_from(progression.special_bonus).unwrap_or(0))
        .execute(&mut *tx)
        .await
        .map_err(map_error(
            "characters.store_progression",
            "character_special",
        ))?;
        tx.commit()
            .await
            .map_err(map_error("characters.store_progression", "characters"))?;
        Ok(())
    }
}

impl SkillRules for PgBackend {
    async fn skill_rule(&self, item_index: i64) -> Result<Option<SkillRule>, DbError> {
        let row = sqlx::query(
            "SELECT s.skill_point::bigint AS skill_point, \
                    COALESCE(i.req_level, 0)::bigint AS req_level, \
                    COALESCE(i.req_int, 0)::bigint AS req_int, \
                    COALESCE(i.req_dex, 0)::bigint AS req_dex, \
                    COALESCE(i.req_con, 0)::bigint AS req_con \
             FROM skill_catalog s \
             LEFT JOIN item_catalog i ON i.item_index = 5000 + s.skill_index \
             WHERE s.skill_index = $1 - 5000",
        )
        .bind(i32::try_from(item_index).unwrap_or(-1))
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("content.skill_rule", "skill_catalog"))?;
        let Some(row) = row else {
            return Ok(None);
        };
        let op = "content.skill_rule.decode";
        let decode = |_| DbError::Transport { op };
        Ok(Some(SkillRule {
            skill_point: row.try_get("skill_point").map_err(decode)?,
            req_level: row.try_get("req_level").map_err(decode)?,
            req_int: row.try_get("req_int").map_err(decode)?,
            req_dex: row.try_get("req_dex").map_err(decode)?,
            req_con: row.try_get("req_con").map_err(decode)?,
        }))
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

/// Linha de item de `character_items`/`vault_items` (colunas idênticas).
fn item_row(row: &PgRow) -> Result<ItemRow, DbError> {
    let op = "items.decode";
    let decode = |_| DbError::Transport { op };
    Ok(ItemRow {
        slot: i64::from(row.try_get::<i16, _>("slot").map_err(decode)?),
        item_id: i64::from(row.try_get::<i32, _>("item_id").map_err(decode)?),
        qty: i64::from(row.try_get::<i32, _>("qty").map_err(decode)?),
        flags: i64::from(row.try_get::<i32, _>("flags").map_err(decode)?),
        refine: i64::from(row.try_get::<i16, _>("refine").map_err(decode)?),
        ancient: row.try_get("ancient").map_err(decode)?,
        gem_effect: i64::from(row.try_get::<i16, _>("gem_effect").map_err(decode)?),
        gem_value: i64::from(row.try_get::<i16, _>("gem_value").map_err(decode)?),
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

    /// Primeiro slot livre de `Carry` (16–79) ou `None` com a mochila cheia.
    async fn claim_item(&self, id: i64) -> Result<Option<(i64, i32)>, DbError> {
        const CARRY_FIRST: i16 = 16;
        const CARRY_LAST: i16 = 79;
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "grants.claim_item",
        })?;
        let linha: Option<(i64, i32, i32)> = sqlx::query_as(
            "SELECT character_id, item_id, qty FROM delivery_queue \
             WHERE id = $1 AND status = 'pending' AND kind = 'item' FOR UPDATE",
        )
        .bind(id)
        .fetch_optional(&mut *tx)
        .await
        .map_err(map_error("grants.claim_item.read", "delivery_queue"))?;
        let Some((character_id, item_id, qty)) = linha else {
            return Ok(None);
        };
        let ocupados: Vec<i16> =
            sqlx::query_scalar("SELECT slot FROM character_items WHERE character_id = $1")
                .bind(character_id)
                .fetch_all(&mut *tx)
                .await
                .map_err(map_error("grants.claim_item.read", "character_items"))?;
        let livre = (CARRY_FIRST..=CARRY_LAST).find(|slot| !ocupados.contains(slot));
        let Some(slot) = livre else {
            // Mochila cheia: a linha fica pendente para a próxima varredura do dreno
            // (o staff vê `attempts` crescer na tela de entregas e libera espaço).
            sqlx::query("UPDATE delivery_queue SET attempts = attempts + 1 WHERE id = $1")
                .bind(id)
                .execute(&mut *tx)
                .await
                .map_err(map_error("grants.claim_item.write", "delivery_queue"))?;
            tx.commit().await.map_err(|_e| DbError::Transport {
                op: "grants.claim_item.commit",
            })?;
            return Ok(None);
        };
        let inserido = sqlx::query(
            "INSERT INTO character_items \
               (character_id, slot, item_id, qty, flags, refine, ancient, gem_effect, gem_value) \
             VALUES ($1, $2, $3, $4, 0, 0, false, 0, 0) \
             ON CONFLICT (character_id, slot) DO NOTHING",
        )
        .bind(character_id)
        .bind(slot)
        .bind(item_id)
        .bind(qty)
        .execute(&mut *tx)
        .await
        .map_err(map_error("grants.claim_item.write", "character_items"))?;
        if inserido.rows_affected() == 0 {
            // Corrida no slot (outro dreno/consumo): tenta de novo na próxima varredura.
            tx.rollback().await.map_err(|_e| DbError::Transport {
                op: "grants.claim_item.rollback",
            })?;
            return Ok(None);
        }
        sqlx::query(
            "UPDATE delivery_queue SET status = 'claimed', claimed_at = now() WHERE id = $1",
        )
        .bind(id)
        .execute(&mut *tx)
        .await
        .map_err(map_error("grants.claim_item.write", "delivery_queue"))?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "grants.claim_item.commit",
        })?;
        Ok(Some((character_id, i32::from(slot))))
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

    async fn live_session_of(&self, login: &str) -> Result<Option<i16>, DbError> {
        let row = sqlx::query(
            "SELECT channel FROM character_session \
             WHERE account_login = $1 AND released_at IS NULL \
             ORDER BY id DESC LIMIT 1",
        )
        .bind(login)
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("sessions.live_session_of", "character_session"))?;
        row.map(|linha| {
            linha
                .try_get::<i16, _>("channel")
                .map_err(|_| DbError::Transport {
                    op: "sessions.live_session_of.decode",
                })
        })
        .transpose()
    }

    async fn release_account_sessions(&self, login: &str) -> Result<u64, DbError> {
        let res = sqlx::query(
            "UPDATE character_session SET released_at = now() \
             WHERE account_login = $1 AND released_at IS NULL",
        )
        .bind(login)
        .execute(self.pool())
        .await
        .map_err(map_error("sessions.release_account", "character_session"))?;
        Ok(res.rows_affected())
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

    async fn release_stale_sessions(&self, ttl_secs: i64) -> Result<u64, DbError> {
        let resultado = sqlx::query(
            "UPDATE character_session SET released_at = now() \
             WHERE released_at IS NULL AND renewed_at < now() - make_interval(secs => $1)",
        )
        .bind(ttl_secs as f64)
        .execute(self.pool())
        .await
        .map_err(map_error("sessions.reap", "character_session"))?;
        Ok(resultado.rows_affected())
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

    async fn channel_capacity(
        &self,
        group_id: i16,
        channel: i16,
    ) -> Result<ChannelCapacity, DbError> {
        let linha = sqlx::query(
            "SELECT max_players, maintenance FROM channel_capacity \
             WHERE group_id = $1 AND channel = $2",
        )
        .bind(group_id)
        .bind(channel)
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("sessions.capacity", "channel_capacity"))?;
        let Some(linha) = linha else {
            return Ok(ChannelCapacity::permissive());
        };
        let maximo: i32 = linha
            .try_get("max_players")
            .map_err(|_| DbError::Transport {
                op: "sessions.decode",
            })?;
        let manutencao: bool = linha
            .try_get("maintenance")
            .map_err(|_| DbError::Transport {
                op: "sessions.decode",
            })?;
        Ok(ChannelCapacity {
            max_players: u32::try_from(maximo).unwrap_or(0),
            maintenance: manutencao,
        })
    }

    async fn channel_role(
        &self,
        group_id: i16,
        channel: i16,
    ) -> Result<Option<ChannelRole>, DbError> {
        let linha =
            sqlx::query("SELECT role FROM channel_role WHERE group_id = $1 AND channel = $2")
                .bind(group_id)
                .bind(channel)
                .fetch_optional(self.pool())
                .await
                .map_err(map_error("sessions.role", "channel_role"))?;
        let Some(linha) = linha else {
            return Ok(None);
        };
        let texto: String = linha.try_get("role").map_err(|_| DbError::Transport {
            op: "sessions.decode",
        })?;
        // Valor fora do CHECK é dado corrompido: devolve `None` para o chamador cair no
        // default `normal` em vez de derrubar o boot do canal.
        Ok(ChannelRole::parse(&texto))
    }
}
