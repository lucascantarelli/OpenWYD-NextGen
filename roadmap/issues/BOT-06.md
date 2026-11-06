# BOT-06 — Simulação econômica e indicadores de inflação

**Fase:** BOT · **Depende de:** BOT-03, BOT-04 · **Bloqueia:** — · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Medir o que uma população de farmers e comerciantes faz com a economia: gold gerado/consumido,
distribuição de riqueza, preços médios, taxa de refino e inflação ao longo do tempo.

## Contexto / evidência

- `sapphire_balance` (manto), `SancRate` (refino) e `item_catalog.price` já são os pontos de preço do
  servidor; o refino para em +6 com Ori e vai a +9 com outro pó (medido em QA in-game).
- O portal tem `/faturamento` (donate) e `/drops`; economia de **gold in-game** não tem painel ainda.
- O documento de missão exige: dropar dados de economia, não alterar a economia a partir de simulação
  isolada.

## Passos

1. Definir os indicadores e como cada um é medido (fonte: snapshot do servidor, não o modelo do bot).
2. Rodar cenário de N farmers por H horas simuladas e registrar gold/h, itens/h e descarte por venda.
3. Rodar cenário de comerciantes e medir concentração de riqueza (Gini) e preço médio praticado.
4. Rodar refino em lote e medir taxa efetiva por faixa contra `SancRate`.
5. Publicar relatório com limitações explícitas; nenhuma mudança de balanceamento sem aprovação.

## Arquivos

- `tmserver/internal/bot/economy.go` (novo)
- `tmserver/internal/bot/scenarios/economia.json`
- `docs/analysis/economia-simulada.md` (relatório)

## Testes / verificação

```
go run ./tmserver/cmd/botlab run -scenario tmserver/internal/bot/scenarios/economia.json -json > rel.json
```

Saída esperada: relatório com gold gerado/consumido, Gini, preço médio, taxa de refino por faixa e o
comando SQL usado para conferir cada número contra o banco.

## Critérios de aceite

- [ ] Cada indicador tem a consulta/lido de origem declarado
- [ ] Cenário reproduzível por semente e duração
- [ ] Relatório separa medido de estimado
- [ ] Nenhuma alteração automática de preço, drop ou taxa

## Riscos / rollback

Risco: conclusão de balanceamento a partir de amostra pequena. Mitigação: declarar N e intervalo de
confiança. Rollback: nenhum estado do servidor é alterado por esta issue.
