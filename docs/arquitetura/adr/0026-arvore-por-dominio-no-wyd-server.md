# ADR-0026 — Árvore de `wyd-server/src` por domínio

- **Estado:** aceita — implementada em 2026-11-11 (etapas 1–5; ver *Execução*)
- **Data:** 2026-11-11
- **Decisores:** dono do projeto + agente
- **Governa:** `crates/wyd-server` (árvore §16 de `estrutura-e-padroes.md`), FASE-03/07/08

## Contexto

`crates/wyd-server/src` é o único crate que ainda não segue o padrão dos demais
(`wyd-core`, `wyd-db`, `wyd-admin`, `wyd-net`, `wyd-protocol`, `wyd-sim`: diretório por
domínio, `mod.rs` só declarando e re-exportando). Hoje são 14 arquivos planos e 12 822
linhas:

| Arquivo | LOC | Conteúdo misturado |
|---|---:|---|
| `cpsock.rs` | 2 862 | aceite, handshake, login/PIN, cerca de sessão, limite por IP, seleção, dispatch de item/ataque/diálogo |
| `world.rs` | 2 363 | tick, cena, NPC, combate, entrada/saída, save, métricas |
| `characters.rs` | 1 654 | Create/Delete/CharacterLogin + regras de slot/nome |
| `personagens.rs` | 1 431 | `STRUCT_SELCHAR`/`STRUCT_MOB`, leitura do banco |
| `gameplay.rs` | 1 162 | refino, loot, EXP, quest, loja |
| `config.rs` | 1 091 | snapshot, carga, validação, NPC/template |
| `main.rs` | 563 | boot, env, migrations, listeners, loops, CLI |
| `session.rs` | 562 | sessão única entre canais |
| `channels.rs` | 309 | canais/papéis |
| `status.rs` | 303 | HTTP/health + sonda UDP |
| `auth.rs` | 243 | senha + PIN |
| `metrics.rs` | 206 | métricas do processo |
| `lib.rs`/`logs.rs` | 73 | fachada/log |

Um arquivo com três fronteiras (wire, persistência, jogo) não pode ser testado por
unidade sem arrastar o processo inteiro; é a causa direta dos testes de 1 000+ linhas
(`sistema_selchar.rs`, `sistema_cena_mobs.rs`) e da dificuldade de achar o dono de cada
regra. A árvore atual foi desenhada quando o crate era fino (ADR-0022); o crescimento
por fases estourou o desenho.

Restrições que a mudança **não** pode quebrar: Hot Path monolítico no mesmo processo,
sem I/O no tick; um processo por canal; `main.rs` só faz boot/wiring; nenhuma regra de
jogo migra para o Cold Path; erros tipados por fronteira; testes de unidade ao lado da
regra.

## Decisão

Adotar em `wyd-server/src/` a mesma árvore por domínio dos demais crates:

```
src/
├── lib.rs                 # fachada: re-exports + invariantes (sem lógica)
├── main.rs                # só CLI/boot: chama boot::run()
├── boot/
│   ├── mod.rs             # orquestra env → migrations → pool → listeners → loops
│   ├── env.rs             # leitura/validação de variáveis
│   └── healthcheck.rs     # `--healthcheck`
├── cpsock/
│   ├── mod.rs             # accept loop, Stage/Scene e respostas (sem regra de domínio)
│   ├── handshake.rs       # INITCODE/versão/checksum
│   ├── login.rs           # AccountLogin + senha/PIN + sessão
│   ├── selection.rs       # seleção/entrada no mundo
│   ├── action.rs          # _MSG_Action/UseItem/Attack → portas do mundo
│   └── limits.rs          # limite por IP/rate-limit
├── world/
│   ├── mod.rs             # World/Engine, tipos, comandos e porta `EngineHandle`
│   ├── tick.rs            # EngineThread/run_loop, movimento e avanço do relógio
│   ├── scene.rs           # entrada/saída, CreateMob/RemoveMob, ESCENE_FIELD
│   ├── npc.rs             # sync_npcs, diálogo
│   ├── combat.rs          # golpe autoritativo
│   └── save.rs            # fila de save/periódico
├── character/
│   ├── mod.rs
│   ├── crud.rs            # Create/Delete/CharacterLogin
│   ├── snapshot.rs        # STRUCT_SELCHAR/STRUCT_MOB
│   └── session.rs         # sessão única entre canais
├── gameplay/
│   ├── mod.rs             # fachada do domínio
│   └── service.rs         # refino/loot/EXP/quest/portões/inventário (ver Execução)
├── config/
│   ├── mod.rs             # fachada do domínio
│   └── service.rs         # snapshot/revisão/carga/validação/NPC (ver Execução)
├── auth/
│   ├── mod.rs
│   ├── password.rs
│   └── pin.rs
├── status/
│   ├── mod.rs             # fachada do domínio
│   └── service.rs         # /serv00.htm, /health e sonda UDP :8281
├── channels.rs
├── metrics.rs
└── logs.rs
```

Regras da árvore:

- `mod.rs` **só** declara módulos e re-exporta a API pública do domínio; nenhuma regra.
- Um arquivo, uma fronteira: wire (`cpsock/`), estado vivo (`world/`), persistência
  (`character/`, `gameplay/`), configuração (`config/`), processo (`boot/`, `status/`).
- Testes de unidade acompanham a regra no arquivo do domínio; teste que precisa de
  socket/banco continua em `tests/sistema/` (ADR-0002 + `docs/arquitetura/testes.md`).
- `main.rs` não ganha `#[cfg(test)]`: o que é testável desce para `boot/`/`auth/`.
- Re-exports temporários na `lib.rs` mantêm os caminhos antigos durante a migração; são
  removidos ao fim de cada etapa (sem `pub use` permanente de módulo interno).

## Alternativas descartadas

| Alternativa | Por que não |
|---|---|
| Manter plano e só dividir os arquivos grandes em `*_parte2.rs` | Não cria dono de domínio; agrava o `utils.rs` proibido |
| Criar crate novo (`wyd-server-core`) | Duplicaria a fronteira do monólito e exigiria passar estado por canal; ADR-0001/0022 |
| Mover regra para `wyd-core`/`wyd-db` | Mistura camadas: core não conhece wire; db não conhece sessão |
| Fazer tudo num único commit | 12,8 mil linhas sem revisão incremental; qualquer regressão fica invisível |

## Consequências

- **Positivas:** arquivo com um dono; teste de unidade possível sem subir processo; o
  teste de sistema encolhe porque o comportamento passa a ser provado na regra; revisão
  por domínio; alinhamento com os outros oito crates.
- **Negativas:** muitas movimentações em arquivos grandes; `git blame` migra (mitigado por
  `git mv` + commits separados); imports internos mudam de caminho.
- **Riscos:** quebrar o dispatch do tick ao mover `world`/`cpsock`; mitigação é mover sem
  editar lógica e rodar o gate após cada etapa (fmt + clippy + `cargo test -p wyd-server -j 1`).

## Plano de execução (uma etapa por commit/issue)

1. `boot/` + `main.rs` fino + `auth/` (senha/PIN) — menor risco, sem tocar no tick.
2. `cpsock/`: separar handshake/login/selection/action/limits, com o `dispatch` em
   `action.rs` e o `mod.rs` só com accept loop/`Stage`/respostas — **concluída em
   2026-11-11**; os testes `sistema_login`, `sistema_handshake`, `sistema_selecao` são o gate.
3. `world/`: tick, cena, NPC, combate e save (módulos `tick.rs`, `scene.rs`, `npc.rs`,
   `combat.rs`, `save.rs`) — **concluída em 2026-11-11**; gate `sistema_cena_mobs`,
   `sistema_entrada_mundo`, `integracao_tick_processo`.
4. `character/` + `gameplay/`: CRUD, snapshot, refino/loot/quest/EXP; gate
   `sistema_personagens_crud`, `sistema_selchar`, `sistema_refino_piedade` — **concluída em
   2026-11-11**.
5. `config/` + `status/` + limpeza dos re-exports; atualizar §16 e este ADR para
   **aceita (implementada)** — **concluída em 2026-11-11**. Os aliases `characters`,
   `personagens` e `session` foram mantidos na `lib.rs` porque testes/launchers externos
   ainda usam os caminhos antigos; removê-los é uma quebra de API fora do escopo desta ADR.

Cada etapa: `cargo fmt --all --check`, `cargo clippy -p wyd-server --all-targets -- -D warnings`,
`cargo test -p wyd-server -j 1`, e atualização do painel em `docs/rust/03-progresso.md`.

## Execução (2026-11-11)

A árvore entregue é a desta ADR, com uma consolidação deliberada: `gameplay/`, `config/`
e `status/` têm `mod.rs` de fachada e **um** `service.rs` com as regras do domínio, em vez
da subdivisão fina (`refine.rs`/`loot.rs`/`loader.rs`/`udp.rs`) prevista acima. A divisão
fina não tem consumidor/teste que a exija hoje e o `service.rs` já dá o dono de domínio
e a fronteira que faltavam; quando uma regra ganhar teste próprio, ela desce para o
arquivo do seu domínio sem mudar a fachada. `character/` foi dividido de fato em
`crud.rs`, `snapshot.rs` e `session.rs`.

A raiz de `src/` ficou apenas com `lib.rs`, `main.rs`, `channels.rs`, `metrics.rs` e
`logs.rs` (os três últimos previstos na árvore). O fallback `include_bytes!` de
`character/snapshot.rs` foi ajustado de `../assets` para `../../assets` por causa da
profundidade do novo diretório.
