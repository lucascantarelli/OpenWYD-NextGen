# ADR-0027 — Árvore de `wyd-bots/src` por domínio

- **Estado:** aceita — implementada em 2026-11-11
- **Data:** 2026-11-11
- **Decisores:** dono do projeto + agente
- **Governa:** `crates/wyd-bots` (árvore §17 de `estrutura-e-padroes.md`), GAME-23

## Contexto

`crates/wyd-bots/src` tinha três arquivos na raiz: `lib.rs` (845 linhas com conexão,
autenticação, leitura de cena, quest, comércio, combate e movimento), `tela.rs` (223) e
`main.rs` (87). É o mesmo sintoma que o `wyd-server` tinha antes da ADR-0026: um arquivo
com várias fronteiras, difícil de revisar e sem dono de domínio, enquanto os demais
crates já usam diretório por domínio com `mod.rs` só declarando/re-exportando.

O crate é consumido pelo `wyd-launcher`, pelos testes de sistema do `wyd-server`
(`campo_vivo`, `cena_armia`, `gold_inicial`, `loja_viva`) e por `tests/sistema/cena_campo.rs`
via `wyd_bots::{Bot, BotConfig, Race, SceneMob}` e `wyd_bots::tela::*`. Esses caminhos
públicos não podem mudar.

## Decisão

Adotar a árvore por domínio, mantendo a fachada pública:

```
crates/wyd-bots/
├── Cargo.toml
└── src/
    ├── lib.rs              # fachada: //! + `mod bot` + re-exports + testes de unidade
    ├── main.rs             # CLI: monta BotConfig e chama a campanha
    ├── bot/
    │   ├── mod.rs          # socket, handshake, autenticação, seleção e helpers de frame
    │   ├── config.rs       # Race + BotConfig
    │   ├── cena.rs         # SceneMob + nome_do_campo (CreateMob)
    │   ├── campanha.rs     # campaign + quest + parse_etapa
    │   ├── comercio.rs     # loja, compra e SendItem
    │   ├── combate.rs      # Attack/AttackOne/AttackTwo e kill
    │   ├── movimento.rs    # Action/walk_to/open_gate
    │   └── recusa.rs       # motivo_da_recusa (texto acionável)
    └── tela/
        └── mod.rs          # renderização/entrada da tela de conexão
```

Regras:

- `lib.rs` só declara `mod bot`/`pub mod tela` e re-exporta `Bot`, `BotConfig`, `Race`,
  `SceneMob`, `parse_etapa`, `nome_do_campo` e `motivo_da_recusa`; os testes de unidade
  continuam nele.
- Cada submódulo de `bot/` contém apenas métodos de `impl Bot` do seu domínio; os campos
  do socket continuam privados e visíveis aos filhos por serem descendentes do módulo.
- O header de imports compartilhados fica em `bot/mod.rs` como `pub(crate) use`, então os
  filhos usam `use super::*` sem duplicar dependências.
- `tela.rs` virou `tela/mod.rs`; `wyd_bots::tela::*` e `main.rs` não mudam.
- Nenhuma lógica foi reescrita: os corpos foram movidos integralmente (`git mv` +
  divisão mecânica), e os gates provam o mesmo comportamento.

## Alternativas descartadas

| Alternativa | Por que não |
|---|---|
| Manter `lib.rs` plano | Repete o problema da ADR-0026 e mistura rede com regra de campanha |
| Criar crate separado (`wyd-bots-core`) | O crate é de teste/QA; separar só adiciona fronteira e build |
| Mover `tela` para `bot/tela` | `tela` não é protocolo de jogo; é a UI de conexão e já é pública |
| Reexportar cada método de `Bot` | Métodos inerentes não precisam de re-export; a API é o tipo |

## Consequências

- **Positivas:** um arquivo por domínio, revisão e testes localizados, alinhamento com
  `wyd-server`/`wyd-admin`; a fachada pública e os caminhos de `tela` permanecem.
- **Negativas/Riscos:** imports compartilhados no `bot/mod.rs`; mitigado por
  `pub(crate) use` + `use super::*` e pelo gate `clippy -D warnings`.
- **Prova:** `cargo fmt --all --check`, `cargo clippy -p wyd-bots --all-targets -- -D warnings`
  e `cargo test -p wyd-bots` verdes em 2026-11-11.
