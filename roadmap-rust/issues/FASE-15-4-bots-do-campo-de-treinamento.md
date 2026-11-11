# FASE-15-4 — Bots autônomos do Campo de Treinamento (um por raça)

**Fase:** 15 · **Depende de:** FASE-15-3 (bots jogáveis) ⬜, FASE-03-18 (quest) ⬜ · **Bloqueia:** — · **Esforço:** M

## Objetivo

Quatro bots autônomos, **um de cada raça** (TK, Foema, BeastMaster, Huntress), criam personagem,
entram no mundo e executam o Campo de Treinamento inteiro sozinhos — servindo de teste de
regressão jogável da primeira experiência.

## Contexto / evidência

`crates/wyd-bots` está planejado (`ADR-0010`) e ainda não existe; hoje o harness cobre protocolo,
não sessão jogável (checklist FASE-15-3 ⬜). O porte Go já tem `tmserver/internal/bot/client.go`
com login/seleção e leitura de `CNFCharacterLogin`, que serve de referência de máquina de
estados. As etapas da quest e os itens estão em `FASE-03-18`.

## Estado real (2026-11-10)

**Entregue:** crate `crates/wyd-bots` (workspace), com `Race` (TK/Foema/BeastMaster/Huntress),
cliente de protocolo real (`INITCODE` → `AccountLogin` → `AccountSecure` → `CharacterLogin`),
`Bot::quest` (`_MSG_Quest`) e `Bot::campaign` que exige os quatro treinadores na ordem lida do
`_MSG_MessagePanel` (`parse_etapa`). CLI `wyd-bots --conta --senha --raca --npc 1,2,3,4`.
Testes: 3 (quatro classes distintas, parse do painel, ordem dos treinadores).

**Aberto:** o roteiro ainda não anda/mata. As provas 451/452/453/524 precisam existir no
inventário — o abate resolve o loot mas ainda **não entrega** ao jogador, e os NPCs do Campo
não estão no snapshot (ver `FASE-03-18`, aberto 1 e 2). Por isso a campanha viva exige
`--npc` e inventário preparado; a coleta automática dos drops entra quando o loot virar
entrega e o Campo tiver templates.

## Passos

1. Extrair a máquina de estados do bot do harness de protocolo: conectar → login (senha + PIN)
   → roster → criar personagem (se preciso) → `CharacterLogin` → ler cena.
2. Navegação por tiles até NPC/alvo usando somente pacotes provados (o movimento no fio ainda é
   resíduo da FASE-03-9; se o `_MSG_Action` for o único caminho provado, usar o que existe e
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
- `.agents/skills/wyd-database` (legado, só contexto de dados) — nada de SQL no bot.

## Documentação de referencia

- `docs/rust/07-ia-e-bots.md`, `docs/arquitetura/adr/0010-ia-npc-e-bots.md`.
- `docs/wyd/pesquisa-campo-de-treinamento-2026-11-10.md`.
- `main:tmserver/internal/bot/client.go` (referência de estados).

## Subtasks

- [ ] Codificar cliente-bot jogável
- [ ] Roteiro da quest por raça
- [ ] Rodar 4 em paralelo e comparar relatórios
- [ ] Documentar e fechar issue

## Critérios de aceite

- [ ] 4 bots, 4 raças, cada um conclui a quest sem intervenção.
- [ ] Nenhum bot usa atalho interno (tudo pelo fio).
- [ ] Relatório por etapa disponível para regressão.

## Riscos / rollback

- Risco: bots disputarem as mesmas chaves/NPCs → mitigação: contas e instâncias separadas por
  bot e fila de NPC; o drop da chave é por personagem.
- Rollback: os bots são um binário de teste; removê-lo não afeta o servidor.
