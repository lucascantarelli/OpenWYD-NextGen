# Observabilidade — operação e runbooks (QA-08)

Como subir a stack de monitoramento, quais métricas existem (contrato da QA-04) e o
que fazer em cada alerta. Nada aqui cria métrica nova: a fonte é
`crates/wyd-core/src/metrics/` + `crates/wyd-net/src/metrics.rs` (`--features metrics`), e o
mapa de serviços é `docs/rust/05-stack-docker.md`.

## 1. Subir e derrubar

```bash
make stack-config        # valida o compose (base+dev e prod) sem subir nada
make monitoring-rules    # promtool (config/regras/disparo-resolução) + amtool (rota do Alertmanager)
make monitoring-up       # Prometheus :9090, Alertmanager :9093, Grafana :3001, node/blackbox :9100/:9115
make monitoring-check    # smoke: ready, alvos, regras, datasource, dashboard, Alertmanager e watchdog
make monitoring-alertas  # prova ponta a ponta: alerta injetado chega ao receptor
make monitoring-down
```

Arquivos versionados (tudo dentro do `docker-compose.yml`; não existe mais
`docker-compose.monitoring.yml`): `docker/monitoring/prometheus/` (`prometheus.yml`,
`alerts.yml`, `alerts.test.yml`, `targets/wyd-server.json`),
`docker/monitoring/alertmanager/alertmanager.yml` (rota/receiver),
`docker/monitoring/blackbox/blackbox.yml` (sonda do portal),
`docker/monitoring/loki/` + `docker/monitoring/alloy/` (logs) e `docker/monitoring/grafana/`
(`provisioning/` + `dashboards/`). O receptor local de prova/bootstrap é
`tools/dev/alert-sink.mjs`, serviço `alert-sink` (o destino de operação é Slack/e-mail — §4).
Segredos (`GRAFANA_PASSWORD`, `SLACK_API_URL`, `SMTP_PASSWORD`) vêm do `.env`, nunca destes
arquivos.

**Alvos do motor:** `docker/monitoring/prometheus/targets/wyd-server.json` lista **um alvo por
canal** (`server-canal-1:9898` e `server-canal-2:9898`), com labels `canal` e `service`; o
Prometheus recarrega o `file_sd` em 30 s. Se o job `wyd-server` não aparecer, confira primeiro
se o container monta `docker/monitoring/prometheus` (o caminho antigo `monitoring/prometheus`
foi removido) e recrie só o Prometheus com `docker compose up -d --no-deps prometheus`.

## 2. Contrato de métricas (QA-04, fechado)

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
3. Verificar se os alvos em `docker/monitoring/prometheus/targets/wyd-server.json` apontam para os
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
3. Checar se o build é `--features metrics` (contadores custam ~24 ns/pacote, medido na QA-04).
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

`process_resident_memory_bytes{job="wyd-server"} > 5e8` (512 MB) por 15 min, por canal.

**De onde vem o teto:** a amostragem de RSS entrou de verdade em 2026-11-12 (thread de 1 Hz no
boot, ver `WydSemAmostraDeMemoria`); os dois canais operam em **~37 MB** (36,9 MB no canal 1 e
36,0 MB no canal 2, medidos no Prometheus). O limiar é ~14× esse baseline: não alerta em carga
normal e avisa muito antes do limite do container. Antes dessa correção o RSS era sempre `0`,
então esta regra existia mas **não podia disparar** — se hoje o valor voltar a `0`, o alerta a
seguir é o `WydSemAmostraDeMemoria`, não este.

1. Confirmar o valor atual e a tendência por canal:
   `process_resident_memory_bytes{job="wyd-server"}` e `deriv(...[30m])` — crescimento
   monotônico é vazamento; degrau único costuma ser carga (jogadores/NPCs a mais).
2. Comparar com `wyd_entities_active`: memória crescendo **sem** entidades = vazamento (fila,
   cache, histórico de chat, save pendente).
3. Conferir as filas antes de culpar o motor: `save_queue` (dreno a cada 5 s) e a
   `delivery_queue` de grants — dreno parado aparece como memória subindo com banco lento.
4. Reinício é paliativo e derruba sessões: registrar no log de operação e abrir issue com a
   curva do gráfico (`make monitoring-check` verde não substitui a curva).
5. Validar a regra sem esperar 15 min: `promtool test rules` cobre disparo em 600 MB e
   resolução ao voltar para 40 MB (`make monitoring-rules`).

### WydRssCrescendo

`deriv(process_resident_memory_bytes{job="wyd-server"}[1h]) * 3600 > 67108864` por 30 min, por
canal — RSS subindo mais de **64 MB/h** (1 MiB/min).

Este é o alerta de **vazamento**, não de nível: o `WydRssAlto` só fala quando o processo passa de
512 MB. Hoje o RSS é plano em ~37 MB; a 64 MB/h o teto levaria ~7 h, então sobra tempo para agir.

1. Olhar o formato da curva (`process_resident_memory_bytes{job="wyd-server"}` em 6 h): rampa
   monotônica = vazamento; degrau que estabiliza = carga (mapa, NPCs, jogadores) — o `for: 30m`
   já descarta as rampas curtas de boot.
2. Cruzar com `wyd_entities_active` e `rate(wyd_ticks_total[5m])`: memória subindo com entidades
   e tick estáveis é vazamento no motor (fila, cache, histórico de chat).
3. Antes de culpar o motor, checar o que roda **fora** do tick: dreno do save (5 s) e
   `delivery_queue` de grants — dreno travado por banco lento aparece como RSS subindo.
4. Confirmado o vazamento, abrir issue com a curva e o commit; reinício é paliativo e derruba
   sessões.
5. Validar a regra sem esperar uma hora: `promtool test rules` cobre +90 MB/h disparando em
   55 min e resolvendo após estabilizar (`make monitoring-rules`).

### WydSemAmostraDeMemoria

`wyd_memory_samples_total` parado com o motor no ar.

1. Esperado fora de Linux: a amostragem lê `/proc/self/statm` e devolve `None` nas demais
   plataformas (lacuna declarada na QA-04).
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
4. Confirmar que a métrica não foi renomeada: `wyd_ticks_total` é contrato da QA-04 e só
   existe quando o motor roda (ADR-0022 §7).

### WydPortalForaDoAr

`probe_success{job="wyd-portal"} == 0` por 3 min — o portal não respondeu **200** em
`/api/health`. BFF fora = staff sem painel, mesmo com o jogo rodando.

1. `docker compose ps portal` e `docker compose logs --tail=100 portal`.
2. `curl -s -o /dev/null -w '%{http_code}' 127.0.0.1:3000/api/health` — **502** significa
   portal vivo e gateway fora (o alerta certo é `WydGatewayForaDoAr`); **000** significa
   processo morto ou porta errada.
3. `docker compose exec blackbox-exporter wget -qO- 'http://127.0.0.1:9115/probe?module=http_2xx&target=http%3A%2F%2Fportal%3A3000%2Fapi%2Fhealth'`:
   `probe_success 0` com `probe_http_status_code 0` = rede/DNS interno, não HTTP.
4. Se o portal respondeu 307/302, a sonda está apontando para rota protegida: `/api/health`
   precisa ser público (200 sem cookie). Não “conserte” afrouxando a sonda.

### WydDiscoQuaseCheio

Menos de 10% livre em algum filesystem do host por 10 min — o volume do Postgres é o primeiro
a sofrer.

1. `docker system df` e `docker compose exec db df -h /var/lib/postgresql/data`.
2. Maiores consumidores típicos: `wyd-prometheus-data` (7 d), log `json-file` de container
   sem teto, `target/`, imagens antigas (`docker image prune` **só** com autorização).
3. Nada de `down -v`: libere espaço por retenção (`--storage.tsdb.retention.time`) e por
   exportação de log (Loki já retém 14 d), não apagando volume.
4. Depois de liberar, o alerta resolve sozinho na próxima avaliação (sem `for` de resolução).

### WydCpuAltaHost

Mais de 90% de CPU no host por 15 min. O tick e o Postgres disputam o mesmo núcleo.

1. `docker stats --no-stream` — qual container consome? Motor, banco ou build em paralelo?
2. Se for o motor, cruze com `WydTickP99Alto`/`WydTickAcimaDoOrcamento` para saber se o
   problema é carga de jogo ou contenção do host.
3. `docker compose up -d --build` rodando junto do jogo é causa comum: build não é operação de
   runtime. Agende.

### WydMemoriaAltaHost

Mais de 95% de memória em uso por 15 min; o OOM killer pode matar o motor ou o Postgres.

1. `docker stats --no-stream` e `free -h` no host.
2. Procurar vazamento: `process_resident_memory_bytes{job="wyd-server"}` subindo sem
   `wyd_entities_active` acompanhar é vazamento, não carga.
3. Mitigação imediata: reiniciar **o** container culpado (nunca a stack inteira) e registrar o
   horário; abrir issue com a série do painel.

### WydHostSemMetricas

`up{job="node"} == 0` por 5 min — sem `node-exporter` o host fica invisível e os alertas de
CPU/RAM/disco param de ter dado.

1. `docker compose ps node-exporter` e `docker compose logs --tail=50 node-exporter`.
2. `curl -s 127.0.0.1:9100/metrics | head -3` (o override publica em loopback).
3. No Docker Desktop o exporter vê a VM Linux, não o Windows: número de disco/CPU é do VM.
   Se a coleta falhar por `--path.rootfs`, recrie só o serviço.

### WydAlertmanagerForaDoAr

`up{job="alertmanager"} == 0` por 5 min — as regras continuam avaliando e **ninguém** é avisado.
É o alerta que mais importa: sem ele, todo o resto é silencioso.

1. `docker compose ps alertmanager` + `docker compose logs --tail=100 alertmanager`.
2. `curl -s -o /dev/null -w '%{http_code}' 127.0.0.1:9093/-/ready`; se o container está de pé e
   a rota é 200, o problema é só o scrape (rede `wyd`).
3. `amtool check-config` via `make monitoring-rules` antes de recriar: config inválida impede o boot.
4. Depois de voltar, confirme com `make monitoring-check` (o `WydWatchdog` tem que reaparecer).

### WydWatchdog

Não é incidente: é o heartbeat **sempre em firing** que prova que a cadeia
`Prometheus → Alertmanager → receptor` está inteira. Receber o aviso periódico é o esperado.

1. Ausência do aviso (e do alerta no Alertmanager) = cadeia quebrada; investigue
   `WydAlertmanagerForaDoAr`, o receiver no log do Alertmanager e a rota `alerting:` do Prometheus.
2. Durante manutenção, **silencie** no Alertmanager (UI `:9093` ou `amtool silence add`), com
   prazo e comentário — nunca apague a regra.

## 4. Notificação: onde o alerta chega (DEVOPS-02)

```text
motor/gateway/host/portal → Prometheus (scrape) → regras (alerts.yml)
                          → Alertmanager (:9093) → slack | email
```

**Destino real:** a rota padrão entrega em `slack` (default) ou `email`, escolhido por
`ALERTMANAGER_DESTINO`. O segredo **nunca** está no arquivo versionado: o container renderiza a
config no boot (`docker/monitoring/alertmanager/render-config.sh` — fonte única, também usada
por `make monitoring-rules --check`) e escreve `/tmp/wyd-secrets/slack_api_url` (ou
`smtp_password`) a partir do ambiente.

| Variável (`.env`) | Papel |
|---|---|
| `ALERTMANAGER_DESTINO` | `slack` (default) ou `email` |
| `SLACK_API_URL` | Incoming Webhook do Slack — **segredo** |
| `SLACK_CHANNEL` | Canal (default `#wyd-alertas`) |
| `SMTP_TO` / `SMTP_FROM` / `SMTP_SMARTHOST` / `SMTP_USER` / `SMTP_PASSWORD` | Destino por e-mail |

```bash
# Slack
printf '%s\n' 'SLACK_API_URL=https://hooks.slack.com/services/...' >> .env
docker compose up -d alertmanager   # recria com o webhook

# E-mail
#   ALERTMANAGER_DESTINO=email, SMTP_SMARTHOST=smtp.exemplo:587, SMTP_FROM/SMTP_TO/SMTP_USER
#   + SMTP_PASSWORD
```

**Mensagem no Slack (templates):** `docker/monitoring/alertmanager/templates/wyd.tmpl`
define `wyd.titulo` e `wyd.texto`. O título traz status, severidade com ícone, alerta e canal;
o corpo traz, por alerta, gravidade, `canal`/alvo/job, **desde quando** e o **runbook** da regra
(link quando a anotação já é URL). O render é provado sem esperar alerta real:

```bash
docker compose exec -T alertmanager amtool template render \
  --template.glob='/etc/alertmanager/templates/*.tmpl' \
  --template.text='{{ template "wyd.texto" . }}' \
  --template.data=/etc/alertmanager/slack-exemplo.json
```

**Trocar/rotacionar a webhook:** revogue a antiga no app do Slack, escreva a nova em
`SLACK_API_URL` no `.env` (forka o arquivo, nunca em git/log), recrie só o alertmanager
(`docker compose up -d --force-recreate alertmanager`) e rode `make monitoring-alertas`. A prova
de que o valor carregado é o novo sai sem imprimir segredo: compare o hash de
`/tmp/wyd-secrets/slack_api_url` no container com o do valor no `.env`.

`make monitoring-rules` roda esse render nos dois estados — **disparo** (`[FIRING]`) e
**resolução** (`[RESOLVED]`) — então template quebrado falha no gate, não na madrugada.

**Resolução:** o receiver tem `send_resolved: true`, então quando o alerta sai de `firing` o
Slack recebe de novo com `[RESOLVED]` e, no corpo, `encerrado <hora>`. Sem essa mensagem o canal
fica com um alarme que “nunca acabou”. `make monitoring-alertas` prova **os dois** envios:
injeta, confirma a notificação de disparo, resolve pela API e confirma a de resolução (dois
incrementos em `alertmanager_notifications_total`, `failed_total` zerado).

**Bootstrap (sem `SLACK_API_URL`):** a integração Slack posta no receptor local `alert-sink`
(`/var/log/wyd-alerts/alertas.jsonl`, volume `alerts-out`). Isso mantém a stack funcionando e
prova o **formato** que o Slack recebe (`channel`/`username`/`attachments`); ao definir a URL
real, o mesmo POST passa a ir para o Slack — sem editar rota, inibição ou runbook.
O `alert-sink` **não** é destino de operação: ele recebe o bootstrap e o caminho de prova
(`prova=local`).

```bash
make monitoring-alertas   # rota real + entrega (bootstrap ou externo) com saída explicando qual foi
docker compose exec -T alertmanager amtool config routes test \
  --config.file=/tmp/wyd-secrets/alertmanager.yml alertname=WydTickP99Alto severity=critical
docker compose exec -T alert-sink tail -3 /var/log/wyd-alerts/alertas.jsonl
```

**Prova de entrega no destino externo:** o Alertmanager não loga sucesso em `info`, então a
ev idência objetiva são os contadores por integração (com `SLACK_API_URL` definido,
`make monitoring-alertas` os lê automaticamente):

```bash
docker compose exec -T alertmanager wget -qO- http://127.0.0.1:9093/metrics | \
  grep -E '^alertmanager_notifications_(total|failed_total)\{integration="slack"'
```

`notifications_total` subindo e `failed_total` parado = destino aceitou a notificação (HTTP 2xx).
Cada alerta gera **duas** entregas quando o ciclo fecha: disparo e resolução.
Falha de webhook inválido, canal inexistente ou SMTP recusado aparece em `failed_total` e no log
com `level=error`.

**Se o e-mail não sair:** o Alertmanager valida `to`/`from`/`smarthost` no boot (defaults
`alertas@localhost`/`127.0.0.1:587` existem só para a configuração ser válida). Destino `email`
sem SMTP real falha no envio, com o motivo no log — o alerta continua em `firing` e é reenviado
no `group_interval`.

Regras de convivência: `WydServidorForaDoAr` inibe os alertas do mesmo canal, `WydDiscoQuaseCheio`
inibe CPU/RAM do host e `WydGatewayForaDoAr` inibe `WydPortalForaDoAr` — o sintoma não é
notificado quando a causa já foi.

## 5. Lacunas declaradas

- **Série do motor:** com os canais atuais, `up{job="wyd-server"}` está 1/1 e os painéis de
  tick/entidades têm série. Se ficarem vazios, verifique se o container do motor roda imagem
  posterior à GAME-07 e se `wyd-server.json` aponta para os dois canais; recriar só os canais
  publica `wyd_ticks_total`, `wyd_tick_duration_microseconds`, `wyd_tick_budget_microseconds`
  e `wyd_entities_active`. O gateway (`wyd_gateway_*`) não depende disso.
- **Sem métrica de memória do gateway:** `process_resident_memory_bytes` é do exporter do motor;
  no Cold Path a leitura viria de `/proc` (mesma lacuna de plataforma da QA-04) — declarado, não inventado.
- **Sem alerta de migration:** a QA-08 pede "falha de migration", mas a QA-04 não expõe métrica
  de migration. Criar a métrica é mudança de contrato (ADR/issue dona), não deste entrega.
