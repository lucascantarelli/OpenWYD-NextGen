# ADR-0006 — Runtime async (`tokio`/`sqlx`) diferido; std agora

- **Estado:** aceita
- **Data:** 2026-11-07
- **Decisores:** dono do projeto + agente (registrado na issue PROC-22)
- **Governa:** `roadmap-rust/changelog.md`, `PROC-16-persistencia-async.md`, `QA-02-pipeline-ci.md`, `QA-04-telemetria.md`, transporte do `wyd-admin` (FASE-07)

## Contexto

`tokio`/`sqlx`/`refinery`/`tracing`/`prometheus` exigem rede/vendor + ADR e estão
indisponíveis offline (`docs/rust/01-dependencias.md` linhas 22–27). O que já funciona
em std: traits async nativas (`wyd-db/src/repo/`), `DeliveryQueue` (`mpsc` limitada),
`PoolConfig` validada, `TokenBucket`/`ConnectionGate` (`wyd-net/src/limits/`), migrations
aplicáveis por `psql -f` (prova FASE-04 em PG17 descartável). Travar o caminho crítico
em download seria trocar entrega por espera.

## Decisão

Adotamos std até a rede/vendor liberarem o ecossistema async, porque contratos (traits),
filas e limites já são testáveis sem runtime — o runtime entra como implementação das
portas, não como reescrita.

## Consequências

- **Positivas:** `cargo test --workspace` verde offline; fronteiras async já definidas
  (`async fn` em traits, `sync_channel` com backpressure).
- **Negativas:** sem pool real, sem servidor HTTP, sem telemetria exportada — PROC-16,
  QA-04, 07-transporte e QA-02 ficam explicitamente pendentes.
- **Invariantes afetados:** nenhum `await` no tick (regra permanente); DSN nunca versionado;
  `sqlx`/`tokio` entram juntos, via ADR filha, com vendor auditado.

## Alternativas consideradas

| Alternativa | Por que foi descartada (por enquanto) |
|---|---|
| `tokio` + `sqlx` agora | exige rede/vendor; sem eles não há build reproduzível |
| `refinery` embarcado | idem; `psql -f` cobre a validação até lá |
| reescrever sync depois | rejeitada: portas já são async; só a implementação muda |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/changelog.md`, `roadmap-rust/issues/FASE-07.md`
- Docs atualizadas nesta ADR: nenhuma (decisão registra o estado vigente)
- Evidência: `cargo test -p wyd-db` (8 lib + 2 constraints), `cargo test -p wyd-net`
  (4 lib + 6 flood), `cargo test -p wyd-admin` (11 lib + 4 integração)

## Revisão

- Revisar quando: rede/vendor liberarem `tokio`+`sqlx` (ou decisão de vendor próprio).
- Sinais de que a decisão envelheceu: fila `mpsc` virando gargalo medido; necessidade de
  servidor HTTP real para o portal operar sem Go (transporte já decidido na ADR-0008:
  HTTP/1.1 + JSON em std na v1; `tokio`/`axum` viram otimização, não requisito).
