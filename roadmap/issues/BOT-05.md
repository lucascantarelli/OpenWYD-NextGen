# BOT-05 — Orquestração, containers e testes de carga controlados

**Fase:** BOT · **Depende de:** BOT-03 · **Bloqueia:** BOT-06, BOT-08 · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Executar uma população grande com baixo custo: controlador central, workers, alvos por cenário, carga
progressiva/constante/pico/soak e interrupção de emergência — sem criar um container por bot.

## Contexto / evidência

- Hoje `botlab run` é um processo por cenário, com N goroutines dentro: para centenas de bots isso já
  é o modelo certo (processo > container por bot).
- O laço do mundo é único (`world.World.Run`): carga de bot é carga de jogo. Testar carga na stack que
  alguém usa à mão é proibido pelo processo.
- O compose já tem perfis e redes internas (`openwyd-*`), e Prometheus/Grafana no perfil
  `monitoring`.
- O executor recusa alvo não loopback sem `-allow-remote`; o container precisa manter essa guarda.

## Passos

1. Controlador com subcomandos (`start`, `stop`, `status`, `report`) e arquivo de estado por execução.
2. Perfis de carga: progressiva (10→50→100→250→500→1000), constante, pico e soak, com limite máximo
   declarado por cenário.
3. Interrupção de emergência: `SIGTERM` cancela todos os bots e grava relatório parcial.
4. Imagem do laboratório (`botlab`) no compose, em perfil próprio (`bots`), sem porta publicada além
   do `/metrics`.
5. Cenários de recuperação: derrubar o tmServer no meio e medir reconexão/erros.

## Arquivos

- `tmserver/cmd/botlab/main.go` (subcomandos)
- `tmserver/internal/bot/orchestrator.go` (novo)
- `docker-compose.yaml`/`docker-compose.override.yaml` (perfil `bots`)
- `Dockerfile` (estágio do botlab)

## Testes / verificação

```
docker compose --profile bots run --rm botlab run -scenario /scenarios/populacao.json
curl -s localhost:9200/metrics | grep openwyd_bot
```

Saída esperada: relatório com bots ativos, conexões/s, p95/p99, erros e ponto de degradação; `SIGTERM`
encerra com relatório parcial e sem processo órfão.

## Critérios de aceite

- [ ] Carga progressiva com ponto de degradação identificado e registrado
- [ ] Soak sem vazamento (RSS estável) e sem desconexão em cascata
- [ ] Interrupção de emergência funciona e grava relatório
- [ ] Um processo serve N bots; nenhum container por bot
- [ ] Guarda de alvo não autorizado mantida dentro do container

## Riscos / rollback

Risco: derrubar a stack de outro teste. Mitigação: perfil separado + aviso + limite de bots.
Rollback: remover o perfil `bots` do compose; a CLI continua funcionando no host.
