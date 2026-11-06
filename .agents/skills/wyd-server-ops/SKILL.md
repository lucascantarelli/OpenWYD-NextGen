---
name: wyd-server-ops
description: Sobe, inspeciona, reinicia e faz backup da stack local do servidor WYD2 (Postgres + dbServer + binServer + tmServer + webServer) em Docker, incluindo semear contas, conferir a árvore de conteúdo tmserver/content/, validar as portas 8281/80 e diagnosticar boot que falha ou loga "logins report no account". Use sempre que a tarefa envolver rodar o servidor, derrubar/subir containers, ver logs, resetar banco, aplicar migrações ou investigar por que o servidor não aceita conexão.
---

# Operação do servidor WYD local

O servidor é ``: cinco serviços em Docker Compose, sendo que **apenas o tmServer fala o
protocolo legado com o cliente**. Os outros são internos (gRPC) e um Postgres.

| Serviço | Porta | Papel |
|---------|-------|-------|
| `tmserver` | **8281** (CPSock/jogo) + **80** (status `serv00.htm`) | dono do estado do mundo |
| `dbserver` | 7514 (interno) | persistência PostgreSQL via gRPC |
| `binserver` | 3000 (interno) | billing (allow-all por padrão) |
| `webserver` | 7600 | web-api para a futura plataforma Next.js |
| `db` | interno | PostgreSQL 17 |

## 1. Pré-voo (30 segundos, evita 90% dos falsos bugs)

```bash
# (comandos rodam na raiz do repositório)

# a) a árvore de conteúdo obrigatória está completa? (catálogo de item/skill/NPC vem do Postgres
#    desde a issue 58; o fail-closed do boot é sobre o banco, não sobre arquivo)
test -f tmserver/content/Common/Settings/SancRate.txt && echo "taxas ok"
ls -l tmserver/content/TMsrv/run/HeightMap.dat tmserver/content/TMsrv/run/AttributeMap.dat
ls tmserver/content/DBsrv/run/BaseMob                                        # TK FM BM HT (816 B cada)

# b) as portas estão livres? (80 é a mais disputada no Windows)
netstat -ano | grep LISTENING | grep -E ":(80|8281)\b"

# c) o Docker responde e não há stack antiga pendurada?
docker compose ps
```

Se `tmserver/content/` não existir, a causa quase certa é **sparse checkout** parcial:

```bash
git sparse-checkout disable     # restaura a árvore completa versionada
git sparse-checkout list        # se listar algo, era sparse
```

O sintoma clássico de conteúdo ausente é o servidor subir, aceitar conexão TCP e **nenhum login
funcionar** (sem `-content` o tmserver usa persistência no-op e "logins report no account"). Também
aparece como `npc template load failed` e `world items not loaded` nos logs.

## 2. Subir a stack

```bash
make run-local      # build + up (db, binserver, dbserver, webserver, depois tmserver) + seed conta
```

Equivale a:

```bash
docker compose up -d --build db binserver dbserver webserver
docker compose run --rm dbserver seed-account -name test -pass test123   # idempotente
docker compose up -d --build tmserver
```

`seed-account` cria uma conta **só com senha (argon2id), sem personagens** — os personagens são
criados no cliente. Para outra conta:

```bash
W2PP_LOCAL_ACCOUNT=jl W2PP_LOCAL_PASSWORD=segredo make run-local
# ou direto:
docker compose run --rm dbserver seed-account -name jl -pass segredo
```

O script imprime o `IP:porta` para apontar o cliente. Em Windows, esse IP é o da interface de rede
real (`ipconfig`), não `127.0.0.1`, quando o cliente estiver em outra máquina.

## 3. Confirmar que subiu de verdade

Um `docker compose ps` com `Up` **não** prova prontidão. Verifique os três sinais:

```bash
# 1. o jogo escutou e o conteúdo carregou
docker compose logs --since 3m tmserver | grep -E "content loaded|tmserver listening|NPCs spawned|walkability grid baked"
#    esperado: content loaded (items, skills), status server listening, walkability grid baked,
#              NPCs spawned com "mobs" na casa dos milhares

# 2. o status HTTP responde (o cliente sonda isto ANTES de abrir o CPSock)
curl -s http://127.0.0.1:80/ | head -3

# 3. o banco tem a conta
docker compose exec db psql -U postgres -c "select login, role, is_blocked from account;"
```

Atenção: os logs usam `slog` texto e cada pacote recebido aparece como `recv packet conn=… type=0x…`.
Isso é normal e é a melhor ferramenta de diagnóstico de protocolo (`-log-sends` liga também o
sentido S→C).

## 4. Flags do tmServer que importam na operação

Definidas em `tmserver/cmd/tmserver/main.go`; quase todas têm equivalente `W2PP_*` no ambiente.

| Flag | Padrão | Efeito |
|------|-------:|--------|
| `-addr` | `:8281` | porta do jogo |
| `-content` | env `W2PP_CONTENT` | árvore `tmserver/content/`; vazio = sem conteúdo (inútil para jogar) |
| `-dbserver` / `-binserver` | env | vazio = persistência no-op / billing allow-all |
| `-status-addr` | `:80` | página de status; vazio desliga |
| `-client-version` | `7640` | versão exigida no login; o compose passa **12000**, que é o valor medido no cliente real (2026-11-04) |
| `-kefra-live` | `false` | semente do ciclo do Kefra (o registro no banco prevalece) |
| `-double-exp` / `-newbie-event` | off | modificadores globais de EXP |
| `-training-field-exp` | `6` | multiplicador do Campo de Treino (mortal < 35) |
| `-affect-scale-pct` | `15` | duração de buffs fora da faixa-alvo (100 = legado) |
| `-log-sends` | off | loga todo frame S→C (alto volume) |
| `-mob-stat-editing` | off | overlay de edição por moderador (exige DB + conteúdo); o catálogo de NPCs **não** tem flag: `npc_definition` é a fonte |
| `-reject-checksum` | **off** | só ligue com captura que confirme checksums corretos |

Mudar flag = `docker compose up -d tmserver` com o `command:` ajustado (ou `docker compose run`).
O compose fixa `-client-version 12000` porque este build do cliente manda 12000.

## 5. Ciclo de vida seguro

```bash
docker compose logs -f tmserver          # acompanhar
docker compose restart tmserver          # reinício (dispara save-on-shutdown)
docker compose down                      # para tudo, preserva o volume pgdata
docker compose down -v                   # APAGA o banco (só com pedido explícito)
```

Regras aprendidas na prática:

- **Não reinicie no meio de um teste humano.** O shutdown salva personagens e a reconexão gRPC
  gera falhas transitórias que parecem bug de código.
- **Imagem stale é traiçoeira.** `up --build` às vezes reusa layer e não recompila. Sintoma: o
  banco tem o valor certo e o serviço devolve antigo. Depois de mexer em `.proto`/migração:
  `docker compose build --no-cache dbserver tmserver`.
- **IPAM esgotado** após muitos ciclos (`all predefined address pools have been fully subnetted`):
  o compose já fixa `172.28.0.0/24`; se ainda falhar, reinicie o Docker Desktop.
- **Porta 80 no Windows** pode estar tomada (IIS, `http.sys`, agentes). Nesse caso suba o tmServer
  com `-status-addr :8080` e ajuste a porta publicada no `docker-compose.yaml` — lembrando que o
  cliente real procura status no :80 do IP do servidor, então para o cliente gráfico a porta 80
  livre no host é requisito.

## 6. Banco, migrações e reset controlado

- As migrações são embutidas (`internal/migrations/*.up.sql`, até `0033_*`) e aplicadas no boot do
  primeiro serviço que conectar (`store.Migrate`). Não rode SQL manual para "criar tabela".
- Ver o schema aplicado: `docker compose exec db psql -U postgres -c "\dt"`.
- Backup lógico (o que vale é o banco):
  `docker compose exec -T db pg_dump -U postgres > backup-$(date +%F).sql`
- Contas bloqueadas, papéis e personagens vivem em `account` / `character`; item, cargo, affects e
  barras de skill têm tabelas próprias. Antes de "consertar" personagem à mão, prefira as
  ferramentas in-game (`/gm`) — escritas diretas competem com o save do loop.
- Postgres local é `postgres:17-alpine` no compose (imagens antigas `postgres:16` podem existir no
  cache e não são usadas).

## 7. Quando algo não sobe

| Sintoma | Causa provável | Ação |
|---------|----------------|------|
| `content: ... no such file` no boot | `tmserver/content/` ausente ou sparse checkout | `git sparse-checkout disable` |
| Login para em "conectando" | link gRPC tmServer↔dbServer morto | `docker compose restart dbserver tmserver`; ver logs dos dois |
| `logins report no account` | tmServer sem `-dbserver` | subir via compose, não `go run` sem flag |
| Cliente vê servidor mas cai no login | `ClientVersion` diferente | skill `wyd-client-connect` (0x102) |
| Cliente não lista o servidor | `serverlist.bin` errado ou :80 inacessível | skill `wyd-client-connect` |
| Mobs não se movem direito | mapas não carregados | conferir `HeightMap.dat`/`AttributeMap.dat` e `walkability grid baked` |
| Warn `monster template has unbalanced Exp` | catálogo NPC não restampado | `make exp-data` (skill `wyd-economy`) |

## 8. O que nunca fazer

- Não use `docker compose down -v`, `DROP DATABASE` ou reset de volume sem pedido explícito.
- Não exponha 7514/3000/7600 na internet: são links internos sem TLS obrigatório por padrão.
- Não habilite `-reject-checksum` "para testar": o cliente pode ter a verificação desligada por
  patch e você derruba toda conexão.
- Não suba o tmserver com `-dbserver` apontando para um banco sem `npc_definition`: o catálogo é a
  **única** fonte de spawn (issue 57), então um banco vazio = **mundo sem NPCs**. Rode
  `dbserver import-npcs` (ou deixe o dbserver semear no boot) antes.
