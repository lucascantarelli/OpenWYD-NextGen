//! Carga do conteúdo estático (ADR-0009) — adapter da porta `ports::content`.
//!
//! Lê as tabelas de **base** (somente leitura para a aplicação), mapeia para os tipos da
//! porta e valida o que o SQL não expressa. Uma única transação de leitura: ou o boot vê
//! um conjunto coerente, ou recebe o relatório do que está inconsistente.
//!
//! Nada aqui roda no tick: o chamador é o boot, **antes** de abrir socket.

use wyd_core::ports::{
    ContentError, ContentSource, ContentTables, ContentVersion, ItemRow, SettingRow, SkillRow,
};

use crate::error::DbError;
use crate::pool::pg::PgBackend;

/// Item da foto do catálogo — só o que o schema já tem (ADR-0022 §4).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CatalogItemRow {
    pub item_index: i64,
    pub name: String,
}

/// Foto do catálogo servida pelo gateway (`item_catalog/list_items`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CatalogSnapshot {
    pub version: String,
    pub items: Vec<CatalogItemRow>,
}

/// Canal de servidor (migration `0029`): o que o `tmserver` publica no `serv00.htm` e usa
/// no handshake. `channel = 0` é a linha de status; `1..10` são os hosts do `:8281`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ServerChannel {
    pub group_id: i64,
    pub channel: i64,
    pub host: String,
    pub status_url: String,
    pub enabled: bool,
}

impl PgBackend {
    /// Lista os canais de servidor ordenados por grupo e canal.
    ///
    /// O `tmserver` lê isto no boot: lista de canais não é arquivo local, é configuração do
    /// servidor no banco (ADR-0022 §6). Tabela ausente = migrations não aplicadas (`Transport`).
    pub async fn server_channels(&self) -> Result<Vec<ServerChannel>, DbError> {
        let rows = sqlx::query(
            "SELECT group_id, channel, host, status_url, enabled \
             FROM server_channel ORDER BY group_id, channel",
        )
        .fetch_all(self.pool())
        .await
        .map_err(|_| DbError::Transport {
            op: "channels.list",
        })?;

        let mut canais = Vec::with_capacity(rows.len());
        for row in &rows {
            use sqlx::Row;
            canais.push(ServerChannel {
                group_id: i64::from(row.try_get::<i16, _>("group_id").unwrap_or_default()),
                channel: i64::from(row.try_get::<i16, _>("channel").unwrap_or_default()),
                host: row.try_get::<String, _>("host").unwrap_or_default(),
                status_url: row.try_get::<String, _>("status_url").unwrap_or_default(),
                enabled: row.try_get::<bool, _>("enabled").unwrap_or(false),
            });
        }
        Ok(canais)
    }

    /// Lê versão de conteúdo + catálogo numa leitura só (boot do gateway, nunca no tick).
    ///
    /// Tabela ausente é `Transport { op }` — a mensagem do erro não carrega DSN nem SQL, e o
    /// chamador decide se sobe degradado ou falha (o gateway falha: health sem catálogo mentiria).
    pub async fn catalog_snapshot(&self) -> Result<CatalogSnapshot, DbError> {
        // `content_version.version` é INTEGER (migration 0007); o cast mantém o tipo do
        // contrato em texto sem espalhar `i32` pelo handler.
        let version = sqlx::query_scalar::<_, String>(
            "SELECT version::text FROM content_version WHERE id = 1",
        )
        .fetch_optional(self.pool())
        .await
        .map_err(|_| DbError::Transport {
            op: "catalog.version",
        })?
        .ok_or(DbError::NotFound {
            entity: "content_version",
            id: 1,
        })?;

        let rows = sqlx::query("SELECT item_index, name FROM item_catalog ORDER BY item_index")
            .fetch_all(self.pool())
            .await
            .map_err(|_| DbError::Transport {
                op: "catalog.items",
            })?;

        let mut items = Vec::with_capacity(rows.len());
        for row in &rows {
            use sqlx::Row;
            items.push(CatalogItemRow {
                item_index: i64::from(row.try_get::<i32, _>("item_index").unwrap_or_default()),
                name: row.try_get::<String, _>("name").unwrap_or_default(),
            });
        }
        Ok(CatalogSnapshot { version, items })
    }
}

/// Pré-requisitos de uma skill para `_MSG_ApplyBonus` (tipo 2).
///
/// `skill_point` vem de `skill_catalog`; `req_*` de `item_catalog` — a skill é o item
/// `5000 + índice`, exatamente o `g_pItemList[Detail]` que o legado consulta.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct SkillRule {
    pub skill_point: i64,
    pub req_level: i64,
    pub req_int: i64,
    pub req_dex: i64,
    pub req_con: i64,
}

/// Leitura pontual das regras de skill (uma consulta por clique, fora do tick).
pub trait SkillRules {
    fn skill_rule(
        &self,
        item_index: i64,
    ) -> impl std::future::Future<Output = Result<Option<SkillRule>, crate::error::DbError>> + Send;
}

impl ContentSource for PgBackend {
    async fn load_content(&self) -> Result<ContentTables, ContentError> {
        let mut tx = self.begin().await.map_err(|_e| infra_failure())?;

        let version = sqlx::query("SELECT version, note FROM content_version WHERE id = 1")
            .fetch_optional(&mut *tx)
            .await
            .map_err(load_failure)?
            .map(|row| {
                use sqlx::Row;
                ContentVersion {
                    version: row.try_get("version").unwrap_or_default(),
                    note: row.try_get("note").unwrap_or_default(),
                }
            })
            .ok_or_else(|| {
                ContentError::single(
                    "content_version",
                    "1",
                    "version",
                    "linha ausente: migrations de conteúdo não aplicadas",
                )
            })?;

        let items = sqlx::query(
            "SELECT item_index, name, price, req_level, kind \
             FROM item_catalog ORDER BY item_index",
        )
        .fetch_all(&mut *tx)
        .await
        .map_err(load_failure)?
        .iter()
        .map(|row| {
            use sqlx::Row;
            ItemRow {
                item_index: row
                    .try_get::<i32, _>("item_index")
                    .unwrap_or_default()
                    .max(0) as u32,
                name: row.try_get("name").unwrap_or_default(),
                price: row.try_get::<i32, _>("price").unwrap_or_default().into(),
                req_level: row
                    .try_get::<i16, _>("req_level")
                    .unwrap_or_default()
                    .max(0) as u16,
                kind: row.try_get("kind").ok().flatten(),
            }
        })
        .collect::<Vec<_>>();

        let skills = sqlx::query(
            "SELECT skill_index, name, class_code, lineage_code \
             FROM skill_catalog ORDER BY skill_index",
        )
        .fetch_all(&mut *tx)
        .await
        .map_err(load_failure)?
        .iter()
        .map(|row| {
            use sqlx::Row;
            SkillRow {
                skill_index: row
                    .try_get::<i32, _>("skill_index")
                    .unwrap_or_default()
                    .max(0) as u16,
                name: row.try_get("name").unwrap_or_default(),
                class_code: row.try_get("class_code").ok().flatten(),
                lineage_code: row.try_get("lineage_code").ok().flatten(),
            }
        })
        .collect::<Vec<_>>();

        let settings = sqlx::query(
            "SELECT section, key, value_int, value_text FROM game_setting \
             ORDER BY section, key",
        )
        .fetch_all(&mut *tx)
        .await
        .map_err(load_failure)?
        .iter()
        .map(|row| {
            use sqlx::Row;
            SettingRow {
                section: row.try_get("section").unwrap_or_default(),
                key: row.try_get("key").unwrap_or_default(),
                value_int: row.try_get("value_int").ok().flatten(),
                value_text: row.try_get("value_text").unwrap_or_default(),
            }
        })
        .collect::<Vec<_>>();

        let lineages = sqlx::query("SELECT code FROM skill_lineage ORDER BY code")
            .fetch_all(&mut *tx)
            .await
            .map_err(load_failure)?
            .iter()
            .map(|row| {
                use sqlx::Row;
                row.try_get::<String, _>("code").unwrap_or_default()
            })
            .collect::<Vec<_>>();

        tx.commit().await.map_err(load_failure)?;

        let tables = ContentTables {
            version,
            items,
            skills,
            settings,
            lineages,
        };
        tables.validate()?;
        Ok(tables)
    }
}

/// Falha de leitura não é "conteúdo inconsistente": é infraestrutura. Vira um issue com
/// tabela `*` para o boot distinguir as duas coisas no relatório.
fn load_failure(_e: sqlx::Error) -> ContentError {
    infra_failure()
}

fn infra_failure() -> ContentError {
    ContentError::single("*", "-", "*", "falha ao ler o conteúdo no banco")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn falha_de_leitura_vira_issue_de_infraestrutura() {
        let err = load_failure(sqlx::Error::PoolClosed);
        assert_eq!(err.issues.len(), 1);
        assert_eq!(err.issues[0].table, "*");
        assert!(err.to_string().contains("falha ao ler"));
    }
}
