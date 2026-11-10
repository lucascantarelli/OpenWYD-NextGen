# Missão wyd2.md — Fase 7: encerramento (2026-11-10)

> Aplica as normalizações da Fase 5 §1–§2 na checklist e mapeia os 9 itens aos
> issues existentes (sem duplicar, conforme `wyd2.md` §12). Só este arquivo e a
> checklist foram tocados; nada commitado, nenhuma issue criada.

## 1. Diffs aplicados na checklist (`docs/rust/08-checklist-funcionalidades.md`)

1. Linha 1.13: "pacote de saída não tratado" → handler `cpsock.rs:1012-1023`
   (`CNFCharacterLogout` + `leave_world` + `Drop`); mantido "sem teste dedicado".
2. Tabela "sem prova automatizada": ganhou coluna `Estado` com 🚧 nas 4 linhas;
   texto da 1.13 corrigido (handler existe, falta teste + golden); demais textos
   mantidos (corretos e atuais).

## 2. Mapeamento 9 itens → issues existentes (nenhuma issue nova necessária)

| # | Item Fase 3 | Issue existente | Estado |
|---|---|---|---|
| 1 | Caixa + pickup + rito tabela | `FASE-03-11-drop-inventario-e-caixa.md` | cobre (depósito, `GroundLoot`, layouts E2 citados) |
| 2 | Cena `ESCENE_FIELD` | `FASE-03-9-cena-autoritativa.md` | cobre |
| 3 | Logout com teste | `FASE-08-9-logout-explicito.md` | cobre (corpo já descreve o correto) |
| 4 | Cadeia de dano/goldens | `FASE-10-1-equacoes.md` | cobre |
| 5 | EXP por flags | `FASE-10-4-exp-por-flags.md` ✅ aberta (flags default off) | ⬜ |
| 6 | Pity + `UpdateItem` | `FASE-03-16-piedade-updateitem.md` ✅ aberta | ⬜ |
| 7 | Trade atômica | `FASE-08-10-trade-atomica.md` ✅ aberta como ⛔ bloqueada-em-layout | ⛔ |
| 8 | Catálogo B1–B17+§6 | `FASE-06-4-catalogo-ameacas.md` ✅ aberta | ⬜ |
| 9 | Harness LCG / bots | `FASE-15-1-suite-concorrencia.md`, `FASE-15-3-bots-carga.md` | cobre |

Itens 5/6/8 abertos como ⬜ (entregas testáveis próprias, sem duplicar as
existentes); item 7 aberto como ⛔ com o bloqueio explícito no título — codar
contra layout chutado continua proibido. Nenhuma duplicata criada: FASE-03-12
trata só de aglutinação/stack, FASE-06-2 só de hardening.

## 3. Missão encerrada

Fases 1–6 entregues em `docs/wyd/pesquisa-wyd2-missao-fase{1-6}-2026-11-10.md`;
Fase 7 aplica os diffs e o mapeamento. Pesquisa completa sem número, layout ou
fórmula inventados. Próximo dono: captura 08-6 → issues 7/8 → implementação.
