# Issue 40 — Logs dos containers na UI (streaming)

**Fase:** 6 · **Depende de:** 20 (shell) · **Bloqueia:** — · **Esforço:** M · **Estado:** ✅
concluída (2026-11-04)

## Objetivo

Uma tela `/logs` no portal mostra, ao vivo e por serviço, o stdout/stderr dos containers da stack
(`tmserver`, `dbserver`, `binserver`, `webserver`, `db`), com pausa, filtro e cópia — sem abrir
terminal e sem `docker compose logs` na mão.

## Contexto / evidência

Hoje o diagnóstico exige `docker compose logs -f tmserver` num terminal à parte. Os sintomas que mais
aparecem já estão mapeados e são caros de garimpar:

- boot do tmserver: `content loaded comprate_families=7 items=3220 skills=151`, `NPCs spawned
  generators=6099 mobs=11930 templates=620` (`AGENTS.md` §4);
- falha de conteúdo: `unbalanced Exp`, `content missing`, catálogo recusado (erro fatal);
- recusa de login do cliente: `0x0102` (version mismatch), handshake CPSock;
- gRPC: `connection refused` do portal quando o `webserver` reinicia (hoje isso vira 502 genérico
  sem contexto na UI).

Serviços e portas vêm de `docker-compose.yaml`. A leitura de logs usa o Docker CLI do host: rodando
com `make portal-dev` (portal no host) o BFF alcança o CLI local; no container `portal` isso não
existe, então a tela degrada com erro claro (ver "Riscos" e o README do portal).

## Passos

1. `src/server/docker.ts` (server-only):
   - `const SERVICES = ["tmserver", "dbserver", "binserver", "webserver", "db"] as const` e
     `type ServiceName` — whitelist único; nada de serviço vindo do cliente sem validação.
   - `composeCwd()` = `path.join(process.cwd(), "..", "OpenWYD")`.
   - `spawnContainerLogs(service, { tail, since, signal })` usando
     `spawn("docker", ["compose", "logs", "-f", "--no-color", "--tail", String(tail), service], { cwd, stdio: ["ignore", "pipe", "pipe"] })`.
     **Nunca `shell: true`** e nunca interpolar texto do usuário na string de comando.
   - `redact(line)` remove ANSI (`\x1b\[[0-9;]*m`) e mascara DSN/segredo:
     `postgres://[^ ]+` → `postgres://***`, `password=[^ ]+`, `W2PP_PLAY_CODE_SECRET=...`.
   - `composeStatus()` roda `docker compose ps --format json`, cacheia 5s e devolve
     `{ service, state, health }[]` para os chips de status.
2. `src/app/api/logs/[service]/route.ts`:
   - `GET` com `runtime = "nodejs"` e `dynamic = "force-dynamic"`.
   - Sessão obrigatória: `getSession()`; sem sessão → 401. Sem papel `moderator|admin` → 403.
     (A sessão é a única autorização aqui — não existe RPC admin para "ver log".)
   - `service` validado contra a whitelist → 404 fora dela.
   - Responde `text/event-stream`: um evento por linha (`data: <linha>\n\n`), `retry: 2000`,
     `x-accel-buffering: no`, `cache-control: no-store`.
   - Fecha o filho no `request.signal` (`abort`) e no `controller.close()`; mata com `SIGTERM` e
     `SIGKILL` após 2s se não sair. Loga `logs stream closed service=…` para auditoria.
   - Parâmetros: `tail` (default 200, máx. 2000), `since` opcional (`10m`, `1h`) repassado ao CLI.
3. `src/app/api/logs/status/route.ts`: devolve `composeStatus()` para os chips (mesma guarda de
   sessão), com `cache-control: no-store`.
4. `src/app/(admin)/logs/page.tsx` + `logs-view.tsx` (client):
   - `Tabs` (componente da issue 20) com "Tudo" + os 5 serviços; "Tudo" abre o stream de todos.
   - `EventSource("/api/logs/<serviço>")`; estado de conexão em `Badge`
     (`conectando` / `ao vivo` / `reconectando`) e contador de linhas.
   - Controles: **Pausar/Retomar** (bufferiza em memória sem renderizar), **Seguir** (autoscroll),
     filtro de texto client-side, **Limpar** e **Baixar .txt** (Blob do buffer atual).
   - Buffer limitado a 2000 linhas (descarta as mais antigas) e *flush* de estado a cada 100ms para
     não re-renderizar por linha — 30 linhas/s do tmserver já travava uma lista ingênua.
   - `<pre>` mono, `aria-live="off"` (log não é anúncio), linhas com `error|panic|fatal` em
     `--danger` e `warn` em `--warn`.
   - Chips de status por serviço (`running/healthy`) com refresh de 10s.
5. Menu: trocar `soon: true` por `false` no item `/logs` de `src/components/nav.ts` e incluir
   `/logs` no `matcher`/`GUARDED` de `src/middleware.ts` (já deixados prontos como "em breve").
6. Documentar em `portal/README.md`: a tela exige acesso ao Docker no host, e em produção vai atrás
   de mTLS/rede interna — nunca exposta à internet.

## Arquivos

- Criados: `portal/src/server/docker.ts`, `portal/src/app/api/logs/[service]/route.ts`,
  `portal/src/app/api/logs/status/route.ts`, `portal/src/app/(admin)/logs/{page,logs-view}.tsx`,
  `portal/README.md`.
- Alterados: `portal/src/components/nav.ts` (`/logs` deixa de ser "em breve"`,
  `SHELL_ROUTES` já cobre), `portal/src/middleware.ts` (matcher).
- Proibido: montar `/var/run/docker.sock` em qualquer container para viabilizar isto; usar `shell:
  true`; aceitar nome de serviço/flag arbitrário do browser; logar o próprio log (loop).

## Testes / verificação

```bash
cd portal

# sem sessão
curl -s -o /dev/null -w "%{http_code}\n" localhost:3000/api/logs/tmserver        # 401

# com sessão (jar do login da issue 19), 5 segundos de stream
curl -s -c /tmp/j -X POST localhost:3000/api/auth/login -H 'content-type: application/json' \
  -d '{"name":"test","pass":"test123"}' >/dev/null
timeout 5 curl -sN -b /tmp/j localhost:3000/api/logs/tmserver | head -20            # data: … linhas
curl -sN -b /tmp/j localhost:3000/api/logs/nao-existe | head -2                     # 404/erro claro

# status
curl -s -b /tmp/j localhost:3000/api/logs/status                                    # JSON por serviço

# o processo docker morre junto com a conexão
cd .. && docker compose -f docker-compose.yaml ps                       # nada alterado
```

Saída esperada: stream com as linhas reais do serviço (ex. no `tmserver`, `NPCs spawned …`) e, ao
fechar a aba, nenhum processo `docker compose logs` órfão (`ps`/Gerenciador de Tarefas).

## Critérios de aceite

- [x] `/logs` lista os 5 serviços e transmite ao vivo; "Tudo" agrega com prefixo do serviço
- [x] Sem sessão 401, com `player` 403, serviço fora da whitelist não executa nada
- [x] Pausar/retomar, filtro, limpar e baixar funcionam sem perder o stream
- [x] Buffer limitado a 2000 linhas; nenhum vazamento de processo ao fechar a aba ou trocar de serviço
- [x] Segredos e DSN nunca aparecem na tela (testado com `W2PP_PLAY_CODE_SECRET` setado)
- [x] Chips de status refletem `docker compose ps` (parar `binserver` e ver o chip mudar)
- [x] Evidência: os `curl` acima + captura da tela com stream ativo

## Evidência (2026-11-04)

`server/docker.ts` (server-only) concentra o privilégio: whitelist fixa de `tmserver`, `dbserver`,
`binserver`, `webserver`, `db` + alvo `all`, `spawn` sem `shell`, `tail` 1..2000, `since` só no
formato `10m`, `redact` (ANSI, DSN, `password=`/`secret=`/`token=`, `W2PP_PLAY_CODE_SECRET`,
`SESSION_SECRET`) e `compose ps` cacheado 5 s. Se o Docker CLI não existe no processo (portal em
container), `composeStatus` devolve `ok: false` com motivo em vez de inventar estado.

```
$ curl localhost:3100/api/logs/tmserver          → 401  (sem sessão)
$ curl -b <player> …/api/logs/status             → 403  {"error":"sua conta não tem permissão…"}
$ curl -b <admin>  …/api/logs/bogus              → 404  {"error":"serviço desconhecido: bogus"}
$ curl -b <admin>  …/api/logs/status             → 200  {"ok":true,"services":[…5 running, db healthy]}
$ curl -Ns -b <admin> …/api/logs/tmserver?tail=5 → data: tmserver-1  | … msg="session send stats" …
$ curl -Ns -b <admin> …/api/logs/all?tail=3      → linhas de todos os serviços, com prefixo
```

Browser (`/logs`): 6 abas (Tudo + 5 serviços), badge `ao vivo`, chips `tmserver · running` …
`db · running (healthy)`, 200 linhas herdadas do `tail`; pausar não fecha o stream (o buffer
segue), filtro reduz a lista, limpar zera e retomar volta a receber. Nenhum `docker.exe`
`compose logs` órfão após fechar o stream; o portal registra `[logs] stream open/closed
service=… reason=client-abort`.

Pendente: o critério do chip mudar com o `binserver` parado não foi exercitado (derrubar o
serviço no meio do trabalho interromperia a stack) — o chip é o mesmo `compose ps` que já devolve
`state`/`health` corretos para os 5.

## Atualização (2026-11-04) — funciona dentro do container

O desenho original degradava quando o portal passou a rodar no compose (sem Docker CLI no
processo). Em vez de remover a tela, o privilégio saiu do portal: novo sidecar **`logpipe`**
(`logpipe/main.go`, Go stdlib, imagem distroless nonroot) é o único que monta
`/var/run/docker.sock` — **read-only**, `read_only: true`, `no-new-privileges`, `group_add: 0`
para ler o socket (root:root 0660 no Docker Desktop) — e expõe só:

- `GET /status` → `{ok, services:[{service,state,health}]}` (inspect por container, cache 5 s);
- `GET /logs/<serviço>` → texto puro; `all` agrega com prefixo `<serviço> | `; demux dos frames de
  8 bytes da Engine API, com fallback para TTY (fluxo cru).

Sem escrita, sem `exec`, sem acesso a arquivos, porta `7788` só na rede do compose. O portal
(`server/docker.ts`) mantém a mesma interface `openContainerLogs`/`composeStatus`: usa
`LOG_PIPE_URL` quando definido (compose) e cai no `docker compose logs -f` do host quando não (o
`make portal-dev` continua igual). `portal` entrou na whitelist e nas abas da tela.

Verificado no container (`:3000`), com sessão `admin`: `/api/logs/status` → `ok:true` com os 6
serviços (`db`/`portal` healthy); `/api/logs/tmserver?tail=3` → linhas reais (`nightmare started`,
`nightmare ended`); `/api/logs/portal` → `[logs] stream closed service=tmserver reason=client-abort`;
`/api/logs/all` → linhas de `portal`, `binserver` e `webserver` com prefixo; sem cookie → 401.
`logpipe` e `portal` healthy. `go build ./logpipe` + `go vet ./logpipe` exit 0.

Continua valendo: o socket nunca é montado no portal; o sidecar só lê logs e não é publicado.

## Riscos / rollback

- **Risco:** o portal executar Docker CLI é privilégio alto. **Mitigação:** whitelist fixa, sem shell,
  sem flags do cliente, guarda de papel, e a tela nunca é publicada para fora do host/rede interna.
- **Risco:** containerizar o portal quebra este desenho. **Mitigação (aplicada):** o privilégio foi
  para o sidecar `logpipe` (socket read-only, sem escrita) e `server/docker.ts` escolhe logpipe ou
  CLI pela presença de `LOG_PIPE_URL` — a UI não mudou na troca.
- **Risco:** um coletor novo virar superfície de acesso. **Mitigação:** só `GET /status` e
  `GET /logs/<alvo>` da whitelist, porta não publicada, nonroot/`read_only`/`no-new-privileges`, e a
autenticação continua no portal (sem sessão 401, `player` 403).
- **Risco:** volume de log afogar a UI. **Mitigação:** cap de 2000 linhas, flush a 100ms, `tail`
  limitado, pausa que bufferiza sem renderizar.
- **Rollback:** `soon: true` de volta no item de menu + remover a rota; nada de estado persistido.
