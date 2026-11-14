# QA-13 — Validação manual com dois clientes: sessão, combate, loot, refino e logout

> **ID canônico:** `QA-13`
> **ID histórico:** `QA-13`
> **Estado 2026-11-13:** ⬜ · ver `roadmap-rust/index.md`.

**Tipo:** Teste manual · **Esforço:** M · **Depende de:** `QA-12` (entrada funcionando), duas contas de teste · **Bloqueia:** confiança em `GAME-05`, `GAME-11`, `GAME-17`

## Objetivo

Provar o que só existe com **dois jogadores reais**: sessão única, limite por IP, visibilidade
de movimento entre clientes, combate com morte/drop, refino e logout. Os testes de sistema usam
harness; este usa o cliente gráfico.

## Contexto / evidência

- `tests/sistema/sessao_unica.rs`, `limite_ip.rs`, `lotacao.rs` provam o servidor, não o cliente.
- Cena autoritativa emite `ESCENE_FIELD`/`CreateMob`/`RemoveMob`; render no 7662 segue pendente
  (`GAME-17`).
- Drop entrega no inventário com `_MSG_SendItem`; caixa no chão **não** existe (`GAME-11`).

## Pré-condições

1. Dois PCs ou duas VMs com o cliente 7662, ou dois usuários do Windows (o limite por IP é 6).
2. Duas contas com personagem no mesmo canal (`server-canal-1`, `127.0.0.1:8281`).
3. Backup de `client/` antes de abrir.

## Passos

| # | Caso | Ação | Esperado | Evidência |
|---:|---|---|---|---|
| 1 | Sessão única | logar a conta A no cliente 1 e no cliente 2 | segundo é recusado; o primeiro continua | log `result=conflict` + prints |
| 2 | Limite por IP | abrir 7 conexões do mesmo IP | a 7ª é recusada (limite 6) | métrica `wyd_ip_limit_refusals_total` |
| 3 | Visibilidade | A anda; B observa | B vê A se mover sem “piscar” | prints dos dois lados |
| 4 | Entrada em cena | A entra no mapa de B | A aparece para B e vice-versa | prints |
| 5 | Combate | A ataca um NPC até a morte | NPC desaparece para os dois; EXP sobe | log `event=death` + print |
| 6 | Loot | abrir inventário após o abate | item recebido (pilha/slot) | log `event=loot` + print |
| 7 | Refino | usar pedra em item | item muda de `+N` e persiste após relog | log `SendItem` + print antes/depois |
| 8 | Logout | A sai explicitamente | B vê A sair da cena | print + `event=logout reason=explicit` |
| 9 | Queda | matar o processo do cliente A | sessão libera e B vê a saída | log `reason=socket` |

## Arquivos

- Atualizar: `docs/cliente/roteiro-prova-visual-7662.md`, `GAME-17`, `GAME-11`, `GAME-05`.
- Proibido: usar duas contas do dono; rodar com o servidor em manutenção.

## Critérios de aceite

- [ ] Sessão única e limite por IP provados **no cliente** (não só no harness).
- [ ] Movimento e cena visíveis nos dois lados, com print simultâneo.
- [ ] Morte, loot e refino com log do canal + estado no cliente.
- [ ] Logout e queda deixam a cena consistente para o outro jogador.

## Se falhar

Registre no arquivo: caso, horário, print dos dois clientes, trecho de log do canal e se o
servidor ficou em estado inconsistente (personagem fantasma, item duplicado, sessão presa).
