# BOT-08 — CI, dashboards e regressão do laboratório de bots

**Fase:** BOT · **Depende de:** BOT-02, BOT-05 · **Bloqueia:** — · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Levar o laboratório para o pipeline e para a observabilidade: teste do núcleo em todo PR, cenário
rápido em integração, carga/soak periódicos e dashboard próprio.

## Contexto / evidência

- `make ci` hoje roda `gate check-skips check-comments guias-check vet build test portal-test`;
  `bot-test` já existe mas **não** está em `make ci`.
- `metrics.go` publica `openwyd_bot_*`; o Prometheus do projeto já coleta 7 alvos e tem 4 alertas.
- Não existe e2e de navegador no portal (0 Playwright) — mesma lacuna de "prova automatizada", tratada
  em TEST-03.
- Teste de carga não pode rodar em PR: custo, duração e instabilidade do ambiente.

## Passos

1. Incluir `bot-test` em `make ci` (é offline e roda em segundos).
2. Criar job de integração com banco efêmero que roda `cenarios/smoke.json` contra um tmServer real.
3. Agendar (cron semanal) o cenário de população e um soak curto, publicando o relatório como artefato.
4. Dashboard Grafana do laboratório: bots ativos, conexões/s, p95/p99, erros por perfil, estados.
5. Transformar cada bug reproduzido por bot em cenário de regressão versionado.

## Arquivos

- `Makefile` (`ci: … bot-test`)
- `.github/workflows/_go.yml` (passo do núcleo de bots)
- `.github/workflows/bots.yml` (cron)
- `monitoring/` (dashboard e alertas)

## Testes / verificação

```
make ci                       # inclui bot-test
go run ./tmserver/cmd/botlab run -scenario tmserver/internal/bot/scenarios/smoke.json
```

Saída esperada: pipeline verde; relatório de carga arquivado; painel com séries `openwyd_bot_*`.

## Critérios de aceite

- [ ] `bot-test` em `make ci` e no workflow de Go
- [ ] Cenário rápido contra servidor real em integração
- [ ] Carga/soak em execução periódica, nunca em PR
- [ ] Dashboard com bots ativos, latência e erros por perfil
- [ ] Todo bug de bot virou cenário de regressão

## Riscos / rollback

Risco: carga agendada coincidir com uso manual do servidor. Mitigação: janela declarada e alvo
separado. Rollback: desativar o cron; o núcleo continua coberto por teste offline.
