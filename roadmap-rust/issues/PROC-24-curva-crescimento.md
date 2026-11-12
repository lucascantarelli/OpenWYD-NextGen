# PROC-24 — Reorganização do roadmap: caminho crítico, curva de crescimento e entregas reais

> **ID canônico:** `PROC-24`
> **ID histórico:** `ROAD-01`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Tipo:** Processo · **Esforço:** M · **Depende de:** PROC-01 · **Bloqueia:** todas as FASEs abertas

## Objetivo

Mapear todos os caminhos/issues (quests, sistemas, client, launcher, portal, bots) e publicar a ordem que entrega valor real primeiro, com curva de crescimento natural do servidor e sem bloqueios.

## Contexto / evidência

- `roadmap-rust/index.md`: 15 fases + fila de entregas rápidas por tamanho/desbloqueio; caminho crítico `00 → 01 → CLIENT-01 → CLIENT-02 → 03 → 04 → 07` e `03 → 10 → 15`.
- Abertos reais: `PROC-02` (ação do dono), `GAME-18` 🟡, `WEB-01` 🚧, `CLIENT-06` 🚧/⛔, `CLIENT-08` 🚧, `CLIENT-10` ⛔, `GAME-19` 🚧, `GAME-20/3/4` ⬜, `QA-09/2/3/4` ⬜/🟡. `PROC-05` e `QA-07` foram concluídas em 2026-11-12 e estão no changelog.
- `prompt.md` pede: caminho mais célere, execução sem bloqueios, maior entrega real primeira, foco em base (comunicação, login, acesso ao mundo, mensagens/notificações), testes manuais com conta admin + bots de carga.

## Passos

1. Publicar matriz caminho × desbloqueio × esforço × valor (login → mundo → combate → persistência → portal → launcher → bots → balanceamento → release).
2. Definir onda 1 (base sólida): `CLIENT-02` ✅ + `GAME-07/GAME-08/GAME-09/GAME-17` ✅ + `WEB-01` 🚧 + `QA-05/QA-06` ✅ + `CLIENT-09` ✅ → login, PIN, seleção, entrada no mundo, cena `ESCENE_FIELD`, logout explícito. Prova: conta admin entra, anda, sai, reloga sem timeout.
3. Onda 2 (mundo jogável): drops ligados ao inventário ✅ + `SendItem` ✅ + refino no fio ✅ + EXP no fio ✅ → caçar, upar, refinar com goldens.
4. Onda 3 (operação): portal leitura/escrita auditada (`WEB-05/WEB-06` ✅) + monitoramento custom (WEB-07) + contas (WEB-08) + premium/cash (WEB-09).
5. Onda 4 (escala): `wyd-bots` jogável (`QA-11/GAME-23` 🟡) + suíte concorrência (`QA-09`) + carga/banco/release (`QA-10`) + laboratório `wyd-sim` (`GAME-21`).
6. Congelar `CLIENT-06` (captura ao vivo, exige VM+cliente) e `CLIENT-10` (trade, layout UNVERIFIED) como ⛔ com critério de desbloqueio explícito — nada abaixo espera por eles.

## Arquivos

- Alterados: `roadmap-rust/index.md` (ordem + ondas), `docs/rust/03-progresso.md` (status), `docs/rust/08-checklist-funcionalidades.md` (mapa do que falta para jogável), `docs/rust/04-revisao-roadmap.md` (revisão).
- Proibido: mudar código nesta issue.

## Testes / verificação

```bash
node tools/content/check-issue-status.mjs
```

Saída esperada: `ok: status do índice e do painel batem`.

## Skills relacionadas

- writing-plans — plano por ondas com DoD.
- software-architecture — dependências Hot/Cold.
- wyd-rust-testing — prova por onda (conta admin + bots).

## Documentação de referência

- `roadmap-rust/index.md`, `docs/rust/03-progresso.md`, `docs/rust/08-checklist-funcionalidades.md`
- `docs/rust/02-execucao.md` §1–§4, `docs/arquitetura/adr/0022-binarios-de-servico.md`

## Subtasks

- [ ] Matriz caminho × desbloqueio publicada
- [ ] Ondas 1–4 com DoD por onda
- [ ] `CLIENT-06/CLIENT-10` congeladas com critério de desbloqueio
- [ ] Índice + painel reconciliados

## Critérios de aceite

- [ ] Ordem sem bloqueio: nenhuma issue da onda N espera captura/decisão externa.
- [ ] Onda 1 demonstrável com conta admin no cliente 7662 (login → mundo → logout → relogin).
- [ ] Cada issue aberta tem dono (onda), esforço (S/M/L) e prova nomeada.

## Riscos / rollback

- Risco: reordenar sem prova e esconder bloqueio → mitigação: bloqueio ⛔ só com evidência de dependência externa.
- Rollback: reverter `index.md` (processo, sem código).
