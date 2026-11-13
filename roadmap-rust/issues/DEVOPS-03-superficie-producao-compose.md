# DEVOPS-03 — Superfície de produção do compose: TLS, tag, privilégio e limites

> **ID canônico:** `DEVOPS-03`
> **ID histórico:** `DEVOPS-03`
> **Estado 2026-11-13:** 🚧 · ver `roadmap-rust/index.md`.
> **Run real:** [43354688714](https://github.com/lucascantarelli/wyd/actions/runs/43354688714).

**Tipo:** DevOps · **Esforço:** M · **Depende de:** QA-03 ✅, WEB-01 ✅ · **Bloqueia:** release

## Objetivo

Tornar o perfil de produção (`docker-compose.yml` + `docker-compose.prod.yml`, sem override de
dev) seguro por construção: portal não exposto em HTTP puro, `TAG` obrigatório, observabilidade
fora da superfície pública, privilégio mínimo, log com teto e limite de recurso em todos os
serviços.

## Contexto / evidência

- `docker-compose.prod.yml` publica `3000:3000` (todas as interfaces) sem proxy/TLS: sessão de
  staff e cookie em claro.
- `image: wyd/*:${TAG:-dev}` também em prod — subir a imagem `dev` não gera erro, e o
  §7 de `docs/rust/05-stack-docker.md` proíbe tag sem TAG.
- `GF_AUTH_ANONYMOUS_ENABLED: true` e `GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_PASSWORD:-troque-aqui-dev}`
  estão no arquivo **base**, herdados por prod.
- `alloy` monta `/var/run/docker.sock:ro`: leitura do socket permite `inspect` e expõe o `env`
  de todos os containers (`POSTGRES_PASSWORD`, `SESSION_SECRET`, `WEB_API_TOKEN`).
- `logging` com `max-size` só nos dois canais; `healthcheck` ausente em prometheus, grafana,
  loki, alloy, node-exporter; `security_opt`/`cap_drop`/`read_only` em nenhum serviço; limites
  de recurso em prod só para `db`, `portal`, `mkdocs`.
- `docker-compose.reaper.yml` sem cabeçalho e sem menção na documentação.
- `.env` inteiro montado no gateway, que só consome `PORTAL_ADMIN_*`.
- Análise completa em `docs/rust/10-devops-ci.md` §6 (achados A3, A4, A5, C6–C15).

## Run real (2026-11-13)

- Run [43354688714](https://github.com/lucascantarelli/wyd/actions/runs/43354688714): job `compose`
  **verde** — `check-compose-hardening` (R1–R6) e `docker compose config` nos três modos (base+dev,
  prod, prod+.env). O CI não tem `.env`, então o job carrega `TAG`/`POSTGRES_PASSWORD`/
  `WEB_API_TOKEN`/`SESSION_SECRET` de validação; antes disso o `config` morria na interpolação.
- Continua 🚧 pelo que o gate **textual** não cobre: TLS na frente do portal, `healthcheck` dos cinco
  serviços de observabilidade, `cap_drop`/`security_opt` por serviço e log com teto nos oito restantes.

## Fechamento (2026-11-13)

- **TLS: não adotar** — decisão do dono e registrada: o stack é de uma máquina de dev, sem domínio
  nem terceiros; certificado somaria serviço novo, renovação e modo de falha de boot sem resolver
  risco real. Compensação: portal de produção passa a escutar **só em loopback**
  (`127.0.0.1:3000`), com túnel SSH/proxy como caminho remoto. Coberto por R8.
- **Log com teto em todos os 14 serviços**: âncoras `x-log-teto` (`json-file`, `10m` × 3 arquivos) e
  `x-seguranca` (`no-new-privileges:true`) aplicadas por serviço; R7 reprova serviço sem `logging`.
- `cap_drop` por serviço **não** foi aplicado: cada imagem exigiria prova de boot (Grafana cria
  volume como uid 472, Postgres precisa de `CHOWN`/`SETUID`) e a stack é local. Fica declarado como
  não adotado, com o motivo — em vez de um `cap_drop: [ALL]` que quebra o próximo `up`.
- Gate: `check-compose-hardening` agora cobre R1–R8 e foi provado **negativo** (portal em
  `0.0.0.0:3000` → exit 1 com mensagem apontando R8); `docker compose config` verde em base+dev e
  prod, e falhando sem `TAG` como deve.

## Progresso (2026-11-13)

- `docker-compose.prod.yml` reescrito como overlay seguro por construção: `${TAG:?defina TAG}` nas
  quatro imagens (portal, gateway e os dois canais), `deploy.resources.limits` nos 14 serviços,
  `GF_AUTH_ANONYMOUS_ENABLED: "false"` no Grafana e nenhuma porta de banco/gateway/monitoramento
  publicada no host.
- Gate novo: job `compose-surface` do `.github/workflows/rust.yml` roda
  `node tools/dev/check-compose-hardening.mjs` (R1–R6) e `docker compose config` nos três modos.
  Evidência local 2026-11-13: `make compose-hardening` exit 0 e `make stack-config` exit 0 com
  `ok: prod recusa TAG vazia` (o overlay falha sem `TAG`, como o aceite pede).
- Ainda **não** coberto (o gate é textual e não finge cobrir): TLS na frente do portal, `healthcheck`
  nos serviços de observabilidade, `cap_drop`/`security_opt` por serviço e log com teto nos oito
  serviços que ainda não têm.

## Passos

1. Decidir e implementar a terminação TLS: proxy reverso no stack (Caddy/Traefik, tag fixa) na
   frente do portal **ou** portal só em loopback + túnel documentado. Registrar a decisão em ADR
   se criar serviço novo (AGENTS.md §2 item 7).
2. Prod: `${TAG:?defina TAG}` nas quatro imagens; `stack-config` passa a falhar sem TAG.
3. Grafana: anônimo e senha default saem do base; em prod, `GF_AUTH_ANONYMOUS_ENABLED: "false"`
   ou Grafana fora do perfil de produção. `monitoring-smoke.mjs` lê a senha do ambiente.
4. Todo serviço: `logging` com `max-size`/`max-file`, `healthcheck` quando participa de
   `up --wait`, `security_opt: [no-new-privileges:true]` e `cap_drop` onde aplicável.
5. Prod: limites de CPU/memória para gateway, canais, prometheus, grafana, loki, alloy —
   incluindo os serviços de observabilidade criados na `DEVOPS-02` (`alertmanager`,
   `alert-sink`, `blackbox-exporter`), que hoje sobem sem teto em qualquer perfil.
6. `alloy`: registrar no próprio arquivo e no runbook que a leitura do socket expõe `env` de
   containers; restringir a superfície (sem `--server.http.listen-addr` público desnecessário) e
   avaliar `docker` log driver → Loki como alternativa sem socket (decisão registrada).
7. `.env` no gateway: montar arquivo dedicado (`run/wyd/staff.env`) com apenas
   `PORTAL_ADMIN_USER`/`PORTAL_ADMIN_PASSWORD`, ou aceitar e documentar a exposição.
8. `docker-compose.reaper.yml`: cabeçalho com propósito, escopo (canal 1), por que 20 s e aviso
   de que não é produção; citá-lo em `docs/rust/05-stack-docker.md`.
9. `--web.enable-lifecycle` move para o override de dev.

## Arquivos

- Alterados: `docker-compose.yml`, `docker-compose.prod.yml`, `docker-compose.override.yml`,
  `docker-compose.reaper.yml`, `tools/dev/monitoring-smoke.mjs`, `Makefile` (`stack-config`),
  `docs/rust/05-stack-docker.md` §2/§5/§7, `docs/rust/10-devops-ci.md` §6, `docs/rust/06-observabilidade-runbooks.md`.
- Criados: config do proxy/TLS e a config do Alertmanager não entra aqui (é `DEVOPS-02`).
- Proibido: publicar 5432/7600/9898 fora de loopback; `down -v`; segredo em arquivo versionado.

## Testes / verificação

```bash
make stack-config
TAG= docker compose -f docker-compose.yml -f docker-compose.prod.yml config --quiet   # deve falhar
TAG=1.0.0 docker compose -f docker-compose.yml -f docker-compose.prod.yml config --quiet
make stack-up && make monitoring-check
```

Saída esperada: sem `TAG` o `config` falha com mensagem clara; com `TAG` valida; `stack-up`
sobe tudo `healthy`; portal acessível apenas pelo caminho TLS/loopback definido.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-server-ops — stack, portas, diagnóstico.
- .agents/skills/wyd-rust-obs — observabilidade e postura de exposição.
- .agents/skills/software-architecture — ADR se houver serviço novo.

## Documentação de referencia

- `docs/rust/10-devops-ci.md` §6.
- `docs/rust/05-stack-docker.md` §2, §7.
- `docs/arquitetura/adr/0022-binarios-de-servico.md`.

## Critérios de aceite

- [x] Portal de produção nunca acessível por HTTP puro; decisão de TLS registrada (não adotar, loopback + túnel).
- [x] `TAG` obrigatório em prod, provado por `stack-config` falhando sem ele.
- [x] Nenhum serviço sem log com teto/healthcheck onde `--wait` depende dele.
- [x] Anônimo do Grafana e senha default fora do perfil de produção.
- [x] Exposição via socket do Docker documentada com escopo e alternativa avaliada.

## Riscos / rollback

- Risco: proxy TLS quebrar o iframe do Grafana ou os cookies `httpOnly` → mitigação: subir o
  proxy atrás de flag e provar `/login`, `/entregas`, `/monitoramento` antes de trocar o padrão.
- Rollback: remover o override de prod novo; dev segue intacto (override é independente).
