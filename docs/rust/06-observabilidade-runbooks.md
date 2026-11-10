# Observabilidade — operação e runbooks (FASE-12-1)

Como subir a stack de monitoramento, quais métricas existem (contrato da FASE-06-1) e o
que fazer em cada alerta. Nada aqui cria métrica nova: a fonte é
`crates/wyd-core/src/metrics/` + `crates/wyd-net/src/metrics.rs` (`--features metrics`), e o
mapa de serviços é `docs/rust/05-stack-docker.md`.

## 1. Subir e derrubar

```bash
make stack-config        # valida o compose (base+dev e prod) sem subir nada
make monitoring-rules    # promtool: config + regras + teste de disparo/resolução
make monitoring-up       # Prometheus :9090, Grafana :3001, node-exporter :9100 (só loopback)
make monitoring-check    # smoke: /-/ready, alvos, datasource e dashboard provisionados
make monitoring-down
```

Arquivos versionados: `docker-compose.monitoring.yml`, `monitoring/prometheus/`
(`prometheus.yml`, `alerts.yml`, `alerts.test.yml`, `targets/wyd-server.json`), e
`monitoring/grafana/` (`provisioning/` + `dashboards/wyd-saude.json`). Segredos
(`GRAFANA_PASSWORD`) vêm do `.env`, nunca destes arquivos.

**Alvos do motor:** `monitoring/prometheus/targets/wyd-server.json` está **vazio** de
propósito — não existe binário de servidor de jogo no workspace (só `wyd-launcher` tem
`[[bin]]`), logo não há `/metrics` para raspar. Quando o serviço `tmserver` existir, o
arquivo passa a listar `tmserver:9898` (um alvo por canal) e o Prometheus recarrega em 30 s.

## 2. Contrato de métricas (FASE-06-1, fechado)

| Métrica | Tipo | Significado |
|---|---|---|
| `wyd_ticks_total` | counter | ticks executados |
| `wyd_tick_duration_microseconds` | histogram | duração do tick (buckets 0,5 ms–1 s) |
| `wyd_entities_active` | gauge | entidades vivas no ECS |
| `process_resident_memory_bytes` | gauge | RSS do processo (amostrado só em Linux) |
| `wyd_memory_samples_total` | counter | amostras de RSS coletadas |
| `wyd_packets_total` | counter | pacotes aceitos |
| `wyd_packets_rejected_total{reason}` | counter | recusas: `rate`, `ip`, `total`, `timeout` |
| `wyd_handshakes_total` | counter | handshakes concluídos |
| `wyd_connections_pending` | gauge | conexões pendentes |
| `wyd_sessions_open` | gauge | sessões abertas |
| `wyd_gateway_requests_total{result}` | counter | gateway: `ok` (200 result=1), `business` (200 result≠1), `transport` (5xx/4xx fora do envelope), `unauthorized` (401) — ADR-0022 |
| `wyd_gateway_uptime_seconds` | gauge | segundos desde o start do gateway |

O gateway é scraped pelo job `wyd-gateway` (alvo fixo `gateway:7600`); o motor pelos alvos de
`targets/wyd-server.json`. Métrica de um job nunca é lida como se fosse do outro.

## 3. Runbooks

### WydGatewayForaDoAr

`up{job="wyd-gateway"} == 0` por 5 min — com ele fora, o portal responde 502 em toda página que
chama o backend.

1. `docker compose ps gateway` e `docker compose logs --tail=200 gateway`.
2. `curl -s 127.0.0.1:7600/health` — 200 prova banco; 503 prova banco fora (o processo está vivo).
3. `docker compose exec gateway /usr/local/bin/wyd-gateway --healthcheck; echo $?` — é o mesmo
   comando do healthcheck do container.
4. Se o processo morreu no boot, o log diz qual dependência faltou: `DATABASE_URL` (`fatal`),
   `WEB_API_TOKEN` ausente ou `catálogo indisponível (migrations aplicadas?)`. Rode `make db-migrate`
   antes de reiniciar.
5. Nunca "consertar" o alerta: sem gateway o portal fica sem backend de verdade.

### WydGatewayErroDeTransporte

`rate(wyd_gateway_requests_total{result="transport"}[5m]) > 0` por 10 min.

1. `docker compose logs gateway | grep audit` mostra a rota chamada e o autor.
2. `transport` também conta `4xx` de rota: 404 significa contrato violado (nome fora de
   `SERVICOS`) e 501 significa rota declarada sem handler — os dois são diagnóstico, não queda.
3. 5xx no handler = futuro pendente de runtime ou `503` de catálogo não carregado: reiniciar
   resolve o segundo; o primeiro é issue.

### WydServidorForaDoAr

`up{job="wyd-server"} == 0` por 5 min.

1. `docker compose ps` / `docker compose logs --tail=200 <serviço do motor>`.
2. `curl -s 127.0.0.1:9898/metrics | head` no host do motor — se não responde, o processo morreu
   ou o `MetricsServer` não foi iniciado.
3. Verificar se os alvos em `monitoring/prometheus/targets/wyd-server.json` apontam para os
   containers certos (um alvo por canal: `server:9898` e `server-canal-2:9898`).
4. O processo de um canal morre sem derrubar o outro: `docker compose ps server server-canal-2`
   e `docker logs wyd-canal-<n>` — o log de boot nomeia `channel`, `channels published` e a
   versão esperada do gate. `docker exec wyd-canal-<n> /usr/local/bin/tmserver --healthcheck`
   só sai 0 com `/health` 200 (que faz `SELECT 1`).
5. Se o log disser `server_channel indisponível`, o banco não tem a migration `0029`:
   `make db-migrate` antes de reiniciar — nunca subir com lista de canais inventada.

### WydTickP99Alto

P99 do tick acima de 40% do orçamento do processo por 10 min. O orçamento vem de
`wyd_tick_budget_microseconds` (`1e6 / WYD_TICK_HZ`): 50 ms a 20 Hz, 20 ms a 50 Hz.

1. Confirmar no dashboard "Tick P50/P95/P99" que é o P99 e não um pico isolado.
2. Comparar com `wyd_entities_active` e `wyd_packets_total`: crescimento de entidades ou
   pacotes normalmente explica o aumento.
3. Checar se o build é `--features metrics` (contadores custam ~24 ns/pacote, medido na 06-1).
4. Regra de ouro do projeto: nada de I/O, alocação ou lock no tick. Se a causa for nova
   chamada no laço, reverter antes de qualquer ajuste de bucket.

### WydPacotesRecusados

Mais de 10% dos pacotes recusados por 5 min.

1. Painel "Recusas por motivo": `rate` = rate limit (jogador atacando rápido demais), `ip` =
   muitas conexões do mesmo IP, `total` = teto global, `timeout` = sessão ociosa.
2. `rate` alto em jogador legítimo pode ser limite mal calibrado; em rajada de um IP só, é
   ataque — manter e registrar.
3. Nunca aumentar limite sem medir o caminho normal: ajuste é decisão registrada (ADR/issue),
   não reação no dashboard.

### WydHandshakeFalhando

Pacotes entrando e nenhum handshake concluído por 10 min.

1. `wyd_packets_total` sobe mas `wyd_handshakes_total` não: clientes estão conectando e não
   concluindo o `INITCODE`/versão.
2. Conferir o gate de versão (`PROTOCOL_VERSION` 12 000, ADR-0020) — cliente com versão
   diferente é recusado com erro tipado e métrica de versão.
3. Conferir MTU/fragmentação e o checksum `Strict` (divergência é erro tipado, não silêncio).

### WydRssAlto

RSS acima de 1,5 GB por 15 min.

1. Confirmar que o processo é o motor (`process_resident_memory_bytes{job="wyd-server"}`).
2. Comparar com `wyd_entities_active`: memória crescendo sem entidades = vazamento (fila,
   cache, histórico de chat).
3. Reinício é paliativo e derruba sessões: registrar no log de operação e abrir issue com a
   curva do gráfico.

### WydSemAmostraDeMemoria

`wyd_memory_samples_total` parado com o motor no ar.

1. Esperado fora de Linux: a amostragem lê `/proc/self/statm` e devolve `None` nas demais
   plataformas (lacuna declarada na 06-1).
2. Se o motor roda em container Linux e a métrica está parada, é bug de amostragem — abrir
   issue; não mascarar o alerta.

### WydTickAcimaDoOrcamento

Regra: P99 acima de 100% de `wyd_tick_budget_microseconds` (gauge publicado pelo
processo; o limiar acompanha `WYD_TICK_HZ`, não um valor fixo).

1. Olhar `wyd_tick_duration_microseconds` por canal no painel “saúde do motor”: se só um canal
   estourou, o problema é do processo (GC de SO, swap, CPU throttling), não do grupo.
2. Comparar com `rate(wyd_ticks_total[1m])`: ticks/s abaixo do `WYD_TICK_HZ` do container =
   perda de passo real; acima = intervalo configurado divergente entre canais.
3. `docker stats` no container do canal; se RSS/CPU estiverem normais, verificar
   `WydSemAmostraDeMemoria` e o log `event=tick p99_us=…` da thread.
4. Causa comum de P99 alto sem carga: stall de VM/host (o `Scheduler` descarta o atraso, então
   o tick não “corre atrás” — o sintoma é P99 alto e taxa normal).

### WydMotorSemTick

1. `up{job="wyd-server"}` continua 1: o processo responde `/metrics`, então a thread
   `wyd-tick` parou ou nunca subiu.
2. Ver no log do container `event=boot`/`event=tick`: ausência de `event=tick` = thread não
   iniciou; boot com `reason=WYD_TICK_HZ` = configuração recusada (faixa 20–50).
3. `docker logs <canal> | grep event=tick`: se havia `ticks=` e parou, o processo travou no
   laço (ver stack com `SIGQUIT`/profiler) — abrir issue com o horário.
4. Confirmar que a métrica não foi renomeada: `wyd_ticks_total` é contrato da FASE-06-1 e só
   existe quando o motor roda (ADR-0022 §7).

## 4. Lacunas declaradas

- **Série do motor depende da imagem em execução:** os painéis do motor ficam vazios enquanto
  os containers `server`/`server-canal-2` rodarem imagem anterior à FASE-03-7, mesmo com
  `up{job="wyd-server"} == 1`. Recriar só esses containers (`docker compose up -d --build
  server server-canal-2`) publica `wyd_ticks_total`, `wyd_tick_duration_microseconds`,
  `wyd_tick_budget_microseconds` e `wyd_entities_active`. O gateway (`wyd_gateway_*`) não
  depende disso.
- **Sem métrica de memória do gateway:** `process_resident_memory_bytes` é do exporter do motor;
  no Cold Path a leitura viria de `/proc` (mesma lacuna de plataforma da 06-1) — declarado, não inventado.
- **Sem alerta de migration:** a 12-1 pede "falha de migration", mas a 06-1 não expõe métrica
  de migration. Criar a métrica é mudança de contrato (ADR/issue dona), não deste entrega.
