# Testes — taxonomia, organização e auditoria

> Norma vigente desde 2026-11-11. Complementa `estrutura-e-padroes.md` §7 e
> `docs/rust/02-execucao.md` §3 (DoD). Vale para todo teste novo e para a revisão
> de teste existente; mudança de taxonomia exige ADR (rito FASE-13-1).

## 1. Escopo e método da auditoria

Auditoria feita em 2026-11-11 sobre a árvore inteira (`crates/*/src`, `crates/*/tests`,
`benches/`), com inventário automatizado de arquivos, contagem de `#[test]`, uso de
`#[ignore]`/`unsafe`, presença de DSN, socket, processo, env e `sleep`.

| Métrica | Valor |
|---|---|
| Arquivos com teste | 173 |
| Testes (`#[test]`) | 731 |
| Unidade (`src`, `#[cfg(test)]`) | 116 arquivos / 465 testes |
| Em `tests/` | 57 arquivos / 266 testes |
| `#[ignore]` | 0 |
| `unsafe` em teste | 0 |

Distribuição por tipo depois da reorganização:

| Tipo | Arquivos | Testes |
|---|---|---|
| `unidade` | 116 | 465 |
| `integracao` | 15 | 100 |
| `contrato` | 10 | 60 |
| `sistema` | 27 | 78 |
| `golden` | 5 | 28 |

## 2. Taxonomia

O tipo é decidido pelo que o teste **precisa para rodar** e pelo que ele **prova**,
não pela pasta em que nasceu.

| Tipo | Prefixo | Onde mora | Precisa de | Prova | Entra no `cargo test` padrão |
|---|---|---|---|---|---|
| Unidade | — | `src/**` em `#[cfg(test)] mod tests` | nada além da CPU | regra local, bordas, erro tipado | sim |
| Integração | `integracao_` | `crates/<crate>/tests/` | só a API pública da crate | colaboração entre módulos, fluxo sem I/O | sim |
| Contrato | `contrato_` | `tests/` | arquivos do repo (doc, SQL, TS, assets) | documentação/schema/rota não divergiu | sim |
| Golden | `golden_` | `tests/` | fixture/seed versionada | bytes ou valor exato com procedência | sim |
| Sistema | `sistema_` | `tests/` | Postgres, socket CPSock, processo, FS | fluxo real ponta-a-ponta | opt-in por env |

Regras de fronteira:

- Unidade **não** abre socket, banco, processo, arquivo do repo, nem dorme. Se precisa,
  é integração ou sistema.
- Integração **não** aceita DSN nem porta: se o teste falha por falta de infraestrutura,
  ele está no tipo errado.
- Contrato compara duas fontes de verdade (doc ↔ código, SQL ↔ catálogo, Rust ↔ Node/TS);
  não substitui teste de comportamento.
- Golden falha por **um byte/valor** e cita a origem no nome, no comentário ou no caminho
  da fixture (`include_bytes!`, seed `00xx`, captura datada).
- Sistema é **opt-in explícito**. Nunca silencioso (§4).

## 3. Nomenclatura e localização

1. Arquivo: `tests/<tipo>_<tema>.rs`, tema em `snake_case` e no singular quando possível
   (`sistema_login.rs`, `contrato_catalogo.rs`, `golden_pacotes.rs`).
2. Prefixo em vez de subpasta: o Cargo descobre `tests/*.rs` automaticamente; subpastas
   exigiriam um `[[test]]` por arquivo no `Cargo.toml` e escondem o tipo na listagem.
3. `tests/common/mod.rs` é o único arquivo sem `#[test]` permitido em `tests/`; o Cargo
   não o trata como alvo. Helpers específicos de um arquivo ficam nele mesmo.
4. Nome do teste descreve **estado esperado + condição**, em português, sem prefixo
   `test_`: `senha_errada_e_recusada_sem_sessao`, `skip_so_quando_dsn_ausente`.
5. Um arquivo tem um tema. Ao passar de ~400 linhas ou de dois tipos de infraestrutura,
   divida por tema (não por número de asserts).
6. Teste novo que muda contrato de wire/schema entra no mesmo commit da mudança e é
   citado na evidência da issue (`docs/rust/03-progresso.md`).

## 4. Higiene obrigatória

### 4.1 Gate de infraestrutura

Teste de sistema sem a variável de ambiente **passa com aviso**, nunca em silêncio e
nunca com `panic`:

```rust
macro_rules! exigir_dsn {
    () => {
        match dsn_teste() {
            Some(dsn) => dsn,
            None => {
                eprintln!("SKIP: WYD_TEST_DSN/WYD_TEST_DATABASE_URL/DATABASE_URL ausente — <o que não rodou>");
                return;
            }
        }
    };
}
```

- A mensagem nomeia **todas** as variáveis aceitas e o que deixou de ser provado.
- Um único helper por crate (`tests/common/mod.rs`), não um `dsn()` copiado por arquivo.
- Helper que conecta devolve `Result` e falha com contexto; não use `.expect("conecta")`
  dentro de helper chamado antes do gate.
- `#[ignore]` não é gate: esconde a existência do teste. Proibido.

### 4.2 Sincronização

- Proibido `sleep` fixo para sincronizar. Use poll com prazo:
  `esperar_ate(Duration::from_secs(5), || condicao())`, falhando com diagnóstico.
- Exceção documentada: prova viva que **mantém** sessão aberta de propósito
  (`WYD_LIVE_HOLD_SECS`) para um scrape do Prometheus. O sleep é o objetivo do teste,
  não sincronização; fica explícito no nome/comentário e no doc.
- Timeout de socket sempre finito; teste não pode pendurar o CI.

### 4.3 Isolamento e efeitos

- Teste de sistema cria os próprios dados com sufixo único (timestamp/aleatório) e
  remove no fim; nunca depende de conta/estado deixado por execução anterior.
- Escrita em disco só em diretório temporário do teste ou sob flag explícita
  (`WYD_CAPTURE_DIR`); nunca em `tmp/` do repositório por padrão.
- Porta efêmera (`bind("127.0.0.1:0")`) em vez de porta fixa.
- Teste não altera `.env`, banco de desenvolvimento compartilhado sem aviso, nem
  derruba serviço do usuário.

### 4.4 Asserções

- Toda asserção compara valor concreto (byte, campo, erro, contagem). `assert!(x.is_ok())`
  sozinho não prova contrato.
- Teste negativo é obrigatório na fronteira: entrada inválida → erro tipado, sem `panic`.
- Não enfraqueça asserção para o CI passar; se o ambiente não permite, o teste é opt-in
  com `SKIP` explícito.
- Determinismo: seed/relógio injetados; duas execuções produzem o mesmo resultado.

### 4.5 Unidade em `src`

- `#[cfg(test)] mod tests` no fim do arquivo da regra, não em `lib.rs` central.
- Não duplique a lógica no teste para "provar" a si mesmo; chame a função real.
- Binário (`src/main.rs`) não é lugar de teste: extraia a lógica para módulo da lib e
  teste lá. Teste em `main.rs` não roda em `cargo test --lib` e mistura env/processo.

## 5. Matriz por crate

| Crate | Unidade | Integração | Contrato | Golden | Sistema | Total |
|---|---|---|---|---|---|---|
| `wyd-admin` | 21/64 | — | 6/47 | — | 1/1 | 28/112 |
| `wyd-bots` | 2/9 | — | — | — | 1/1 | 3/10 |
| `wyd-core` | 22/102 | 7/58 | — | 1/4 | 1/1 | 31/165 |
| `wyd-db` | 10/18 | — | 2/5 | — | 3/10 | 15/33 |
| `wyd-launcher` | 13/60 | — | 1/4 | 1/3 | — | 15/67 |
| `wyd-net` | 5/14 | 2/7 | — | — | — | 7/21 |
| `wyd-protocol` | 16/44 | 1/4 | — | 1/10 | — | 18/58 |
| `wyd-server` | 9/54 | 4/23 | 1/4 | — | 21/65 | 35/146 |
| `wyd-sim` | 18/100 | 1/8 | — | 2/11 | — | 21/119 |

Formato `arquivos/testes`. Contagem de 2026-11-11; atualize ao mover teste de tipo.

## 6. Auditoria caso a caso (`tests/`)

Legenda: **OK** = tipo correto e higiene adequada; **P0** = gate mentiroso/risco de
falso verde; **P1** = tipo misturado, monólito ou duplicação; **P2** = higiene
(sleep, tamanho, artefato, doc).

### 6.1 `wyd-server` (26 arquivos)

| Arquivo | Tipo | Testes | Infra | Veredito |
|---|---|---|---|---|
| `contrato_monitoramento.rs` | contrato | 4 | arquivos do repo | OK — doc ↔ consultas/rotas; usa `CARGO_MANIFEST_DIR` |
| `integracao_canal_verde.rs` | integração | 4 | nenhuma | OK |
| `integracao_config_gameplay.rs` | integração | 9 | nenhuma | P2 — 842 linhas; dividir refino/loot/EXP quando crescer |
| `integracao_config_recarga.rs` | integração | 6 | nenhuma | P2 — 2 `sleep` de 5/10 ms viram poll |
| `integracao_tick_processo.rs` | integração | 4 | nenhuma | P2 — 2 `sleep` (300/120 ms) viram poll com prazo |
| `sistema_campo_treinamento.rs` | sistema | 12 | DB/FS? | OK — fluxo de quest completo, 12 casos |
| `sistema_campo_vivo.rs` | sistema | 1 | DB + CPSock | OK — `SKIP` explícito nas duas variáveis |
| `sistema_cena_armia.rs` | sistema | 1 | DB + CPSock | OK — idem |
| `sistema_cena_mobs.rs` | sistema | 1 | DB + CPSock | **P1** — 442 linhas/1 teste com 24 asserts; separar handshake, cena e diálogo |
| `sistema_conta_de_teste.rs` | sistema | 1 | DB + CPSock | OK |
| `sistema_entrada_mundo.rs` | sistema | 1 | DB | OK |
| `sistema_gold_inicial.rs` | sistema | 1 | DB + CPSock | OK |
| `sistema_handshake.rs` | sistema | 5 | CPSock vivo | OK — dois gates nomeados |
| `sistema_limite_ip.rs` | sistema | 1 | socket local | P2 — 2 `sleep` (200/50 ms) viram poll |
| `sistema_login.rs` | sistema | 7 | DB + socket | P2 — gate por macro, mas helper `servidor()` ainda faz `.expect` |
| `sistema_loja_viva.rs` | sistema | 1 | DB + CPSock | OK |
| `sistema_lotacao.rs` | sistema | 5 | DB | OK — macro de gate nomeada |
| `sistema_personagens_crud.rs` | sistema | 2 | DB + socket | **P1** — 428 linhas, DSN duplicado, `skip:` minúsculo (L339) |
| `sistema_pin_persistencia.rs` | sistema | 3 | DB | OK |
| `sistema_pin_troca.rs` | sistema | 1 | DB | OK |
| `sistema_portal_pin.rs` | sistema | 1 | DB | OK |
| `sistema_refino_piedade.rs` | sistema | 5 | DB | OK — gate único `WYD_TEST_DSN` |
| `sistema_selchar.rs` | sistema | 9 | DB + socket | **P1** — 1279 linhas, contrato+DB+logout; **P2** escreve `tmp/captura` |
| `sistema_selecao.rs` | sistema | 4 | CPSock + Prometheus | **P0** — `return;` sem mensagem (L327); 3 `sleep` de 2/5/1 s |
| `sistema_sessao_unica.rs` | sistema | 1 | 2 canais CPSock | P2 — `SKIP` ok, mas `sleep` de 400 ms vira poll |
| `sistema_status_udp.rs` | sistema | 2 | UDP local | OK — porta efêmera |

### 6.2 `wyd-admin` (7 arquivos)

| Arquivo | Tipo | Testes | Infra | Veredito |
|---|---|---|---|---|
| `contrato_autorizacao.rs` | contrato | 4 | nenhuma | OK |
| `contrato_login.rs` | contrato | 4 | nenhuma | OK |
| `contrato_painel.rs` | contrato | 6 | nenhuma | OK |
| `contrato_transporte.rs` | contrato | 14 | nenhuma | OK — paridade com o cliente TS (ADR-0008) |
| `contrato_conteudo.rs` | contrato | 7 | fake + DB real | **P1** — 2 testes de Postgres no arquivo de contrato; separar `sistema_conteudo.rs` |
| `contrato_operacao.rs` | contrato | 12 | fake + DB real | **P1** — 1281 linhas; 5 testes reais de DB no contrato |
| `sistema_seed.rs` | sistema | 1 | processo + DB | OK — `SKIP` explícito |

### 6.3 `wyd-db`, `wyd-core`, `wyd-net`, `wyd-protocol`, `wyd-sim`, `wyd-launcher`, `wyd-bots`

| Arquivo | Tipo | Testes | Veredito |
|---|---|---|---|
| `wyd-db/tests/contrato_catalogo.rs` | contrato | 3 | OK — doc ↔ migrations ↔ `SCHEMA_VERSION` |
| `wyd-db/tests/contrato_migrations.rs` | contrato | 2 | OK |
| `wyd-db/tests/sistema_pg.rs` | sistema | 8 | P2 — 9 gates repetidos; extrair `exigir_dsn!` |
| `wyd-db/tests/sistema_sessoes.rs` | sistema | 1 | P2 — mesmo gate |
| `wyd-db/tests/sistema_stack_caps.rs` | sistema | 1 | P2 — mesmo gate |
| `wyd-core/tests/integracao_autoridade.rs` | integração | 10 | OK — pacote forjado recusado |
| `wyd-core/tests/integracao_compose.rs` | integração | 11 | OK |
| `wyd-core/tests/integracao_content.rs` | integração | 7 | OK |
| `wyd-core/tests/integracao_determinism.rs` | integração | 3 | OK |
| `wyd-core/tests/integracao_gem.rs` | integração | 6 | OK |
| `wyd-core/tests/integracao_inventario.rs` | integração | 9 | OK |
| `wyd-core/tests/integracao_npc.rs` | integração | 12 | OK — determinismo com hash |
| `wyd-core/tests/golden_gema_seed.rs` | golden | 4 | OK — seed SQL citado |
| `wyd-core/tests/sistema_telemetry.rs` | sistema | 1 | OK — captura stdout e tick reais |
| `wyd-net/tests/integracao_flood.rs` | integração | 6 | OK |
| `wyd-net/tests/integracao_metrics.rs` | integração | 1 | OK — feature `metrics` |
| `wyd-protocol/tests/integracao_roundtrip.rs` | integração | 4 | OK |
| `wyd-protocol/tests/golden_pacotes.rs` | golden | 10 | OK — bytes reais |
| `wyd-sim/tests/integracao_determinism.rs` | integração | 8 | OK |
| `wyd-sim/tests/golden_formulas.rs` | golden | 6 | OK — valor exato por fórmula |
| `wyd-sim/tests/golden_refino_seed.rs` | golden | 5 | OK — paridade com seed |
| `wyd-launcher/tests/contrato_identidade.rs` | contrato | 4 | **P1** — 2 `return` mudos quando `client/` não existe (L150/L176) |
| `wyd-launcher/tests/golden_serverlist_paridade.rs` | golden | 3 | OK — skip com mensagem quando `node` falta |
| `wyd-bots/tests/sistema_cena_campo.rs` | sistema | 1 | P2 — `sleep` fixo no laço do bot |

### 6.4 Unidade (`src`)

| Arquivo | Testes | Veredito |
|---|---|---|
| `wyd-server/src/world.rs` | 18 | OK; arquivo grande, teste no fim do arquivo é o padrão |
| `wyd-core/src/systems/inventory.rs` | 19 | OK |
| `wyd-core/src/ecs/components.rs` | 19 | OK |
| `wyd-admin/src/main.rs` | 4 | **P1** — teste de binário com env/processo; mover regra para a lib |
| `wyd-core/src/telemetry.rs` | 1 | P2 — depende de env; isolar com subscriber injetado |
| `wyd-bots/src/lib.rs` | 5 | OK (o `sleep` é do laço de produção, não do teste) |

## 7. Dívida priorizada

### P0 — falso verde (corrigir antes de qualquer release)

1. `sistema_selecao.rs`: gate sem mensagem. Um operador não sabe que 4 casos não rodaram.
2. Padronizar mensagem única (`SKIP: ...`) e um macro por crate; eliminar `skip:`
   minúsculo e `return` mudo (`sistema_personagens_crud.rs`, `contrato_identidade.rs`).

### P1 — tipo misturado e monólito

1. `contrato_conteudo.rs`/`contrato_operacao.rs`: mover os testes com Postgres para
   `sistema_conteudo.rs`/`sistema_operacao.rs` (mesma suíte, outro gate).
2. `sistema_cena_mobs.rs`: quebrar em `sistema_handshake_mobs`, `sistema_cena_mobs` e
   `sistema_dialogo_npc` com helpers em `tests/common/mod.rs`.
3. `sistema_selchar.rs`: separar contrato de pacote, CRUD de personagem e logout;
   escrita em `tmp/` sob `WYD_CAPTURE_DIR`.
4. `wyd-admin/src/main.rs`: extrair `dotenv`/seed para módulo da lib e testar como unidade.

### P2 — higiene

1. Substituir os 12 `sleep` de sincronização por poll com prazo (7 arquivos).
2. Unificar DSN: `WYD_TEST_DSN` **ou** `WYD_TEST_DATABASE_URL`/`DATABASE_URL` via um
   `tests/common/mod.rs` por crate (`wyd-server`, `wyd-db`, `wyd-admin`).
3. Reduzir `integracao_config_gameplay.rs` e `contrato_operacao.rs` a ≤400 linhas.
4. Corrigir docs que citam arquivos inexistentes (`tests/movement.rs`, `tests/combat.rs`,
   `tests/logout.rs`) para o módulo/tema real.

## 8. Checklist de revisão

- [ ] O tipo declarado no nome corresponde ao que o teste precisa (sem DSN em
      `integracao_`, sem socket em `unidade`)?
- [ ] Gate nomeia todas as variáveis e o que não rodou? Sem `#[ignore]`?
- [ ] Nenhum `sleep` fixo de sincronização; timeout finito?
- [ ] Dados únicos e limpeza; porta efêmera; sem escrita no repo por padrão?
- [ ] Asserção de valor concreto, caso negativo e determinismo?
- [ ] Arquivo ≤~400 linhas e um tema; helper no `common` quando reutilizado?
- [ ] Referências em Makefile/CI/docs atualizadas no mesmo commit?
- [ ] `cargo fmt --all --check` e `cargo clippy --workspace --all-targets -- -D warnings`
      limpos; teste afetado executado (com `-j 1` em `wyd-server` no Windows)?
