# GAME-23 — Bots autônomos do Campo de Treinamento (um por raça)

> **ID canônico:** `GAME-23`
> **ID histórico:** `FASE-15-4`
> **Estado 2026-11-12:** 🟡 · ver `roadmap-rust/index.md`.

**Fase:** 15 · **Depende de:** QA-11 (bots jogáveis) 🟡, GAME-18 (quest) 🟡 · **Bloqueia:** — · **Esforço:** M

## Objetivo

Quatro bots autônomos, **um de cada raça** (TK, Foema, BeastMaster, Huntress), criam personagem,
entram no mundo e executam o Campo de Treinamento inteiro sozinhos — servindo de teste de
regressão jogável da primeira experiência.

## Contexto / evidência

`crates/wyd-bots` foi entregue em 2026-11-10 com cliente de protocolo real; o que falta é o
roteiro vivo (andar, matar, coletar) e a prova com os quatro bots em paralelo. O porte Go tem
`tmserver/internal/bot/client.go` com login/seleção e leitura de `CNFCharacterLogin`, que serviu
de referência de máquina de estados. As etapas da quest e os itens estão em `GAME-18` 🟡.

## Estado real (2026-11-10)

**Entregue:** crate `crates/wyd-bots` (workspace), com `Race` (TK/Foema/BeastMaster/Huntress),
cliente de protocolo real (`INITCODE` → `AccountLogin` → `AccountSecure` → `CharacterLogin`),
`Bot::quest` (`_MSG_Quest`) e `Bot::campaign` que exige os quatro treinadores na ordem lida do
`_MSG_MessagePanel` (`parse_etapa`). CLI `wyd-bots --conta --senha --raca --npc 1,2,3,4`.
Testes: 3 (quatro classes distintas, parse do painel, ordem dos treinadores).

**Aberto (2026-11-12):** o crate já tem `walk_to`/`action_at`, `attack_at`/`kill`,
`open_gate` e `open_shop`/`buy`; o teste vivo `tests/sistema/cena_campo.rs` lê os 10 NPCs do
seed `0046` por proximidade. O que falta é rodar **os quatro bots em paralelo** no roteiro
completo (NPC → guardião → chave → portão → boss → emblema), coletar os drops e comparar os
relatórios por raça. O loot do Campo já é entregue no inventário (`GAME-18`/`0049`), então a
preparação manual de inventário deixou de ser pré-requisito do caminho principal.

## Passos

1. Extrair a máquina de estados do bot do harness de protocolo: conectar → login (senha + PIN)
   → roster → criar personagem (se preciso) → `CharacterLogin` → ler cena.
2. Navegação por tiles até NPC/alvo usando somente pacotes provados (o movimento no fio ainda é
   resíduo da GAME-09; se o `_MSG_Action` for o único caminho provado, usar o que existe e
   declarar o limite).
3. Roteiro da quest por bot: falar com o treinador da etapa, esperar a recompensa, matar o
   guardião da chave, usar a chave no portão, repetir; no fim, boss + emblema + chefe.
4. Cada bot registra um relatório por etapa (tempo, itens recebidos, HP/MP) para o teste
   comparar entre raças.
5. Rodar os quatro em paralelo contra o canal do compose, com conta própria por bot e sem
   interferir em sessões humanas.

## Arquivos

- Criados: `crates/wyd-bots/` (árvore do ADR-0002/0010), `crates/wyd-bots/tests/campo.rs`.
- Alterados: `Cargo.toml` do workspace, `docs/rust/07-ia-e-bots.md`, checklist.
- Proibido: bot que lê banco por dentro ou pula validação do servidor.

## Testes / verificação

```bash
cargo test -p wyd-bots
cargo run -p wyd-bots -- --raca TK --conta bot-tk
```

Saída esperada: 4/4 bots concluem 0→4 e o relatório mostra a recompensa escolhida em cada raça.

## Skills Relacionadas do freebuff

- `.agents/skills/wyd-rust-testing` — harness e carga.
- `.agents/skills/wyd-rust-net` — handshake e pacotes provados.
- `.agents/skills/wyd-rust-db` (só contexto de dados) — nada de SQL no bot.

## Documentação de referencia

- `docs/rust/07-ia-e-bots.md`, `docs/arquitetura/adr/0010-ia-npc-e-bots.md`.
- `docs/wyd/pesquisa-campo-de-treinamento-2026-11-10.md`.
- `main:tmserver/internal/bot/client.go` (referência de estados).

## Subtasks

- [x] Codificar cliente-bot jogável (protocolo real, cena, combate, loja, portão).
- [x] Ler os 10 NPCs do Campo no seed `0046` pelo teste vivo de proximidade.
- [ ] Roteiro da quest por raça (quatro raças, drops reais, sem `--npc` preparado).
- [ ] Rodar 4 em paralelo e comparar relatórios.
- [ ] Documentar e fechar issue

## Critérios de aceite

- [ ] 4 bots, 4 raças, cada um conclui a quest sem intervenção.
- [x] Nenhum bot usa atalho interno (tudo pelo fio).
- [ ] Relatório por etapa disponível para regressão.

## Riscos / rollback

- Risco: bots disputarem as mesmas chaves/NPCs → mitigação: contas e instâncias separadas por
  bot e fila de NPC; o drop da chave é por personagem.
- Rollback: os bots são um binário de teste; removê-lo não afeta o servidor.
