# Issue 40 — Logs dos containers na UI (streaming)

**Fase:** 6 · **Depende de:** 20 (shell) · **Bloqueia:** — · **Esforço:** M · **Estado:** ⏳ aberta

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

- [ ] `/logs` lista os 5 serviços e transmite ao vivo; "Tudo" agrega com prefixo do serviço
- [ ] Sem sessão 401, com `player` 403, serviço fora da whitelist não executa nada
- [ ] Pausar/retomar, filtro, limpar e baixar funcionam sem perder o stream
- [ ] Buffer limitado a 2000 linhas; nenhum vazamento de processo ao fechar a aba ou trocar de serviço
- [ ] Segredos e DSN nunca aparecem na tela (testado com `W2PP_PLAY_CODE_SECRET` setado)
- [ ] Chips de status refletem `docker compose ps` (parar `binserver` e ver o chip mudar)
- [ ] Evidência: os `curl` acima + captura da tela com stream ativo

## Riscos / rollback

- **Risco:** o portal executar Docker CLI é privilégio alto. **Mitigação:** whitelist fixa, sem shell,
  sem flags do cliente, guarda de papel, e a tela nunca é publicada para fora do host/rede interna.
- **Risco:** containerizar o portal quebra este desenho. **Mitigação:** isolar tudo em
  `server/docker.ts` atrás de uma interface (`LogSource`), para trocar por socket-proxy/`docker API`
  ou por um `logs` RPC no `webserver` sem tocar na UI.
- **Risco:** volume de log afogar a UI. **Mitigação:** cap de 2000 linhas, flush a 100ms, `tail`
  limitado, pausa que bufferiza sem renderizar.
- **Rollback:** `soon: true` de volta no item de menu + remover a rota; nada de estado persistido.
