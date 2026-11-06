# Fase 7 — Configuração e Operação (w2pp-OpenWYD)

> Catálogo de TODOS os arquivos de config em `tmserver/content/`, topologia de deploy, IPs/portas hardcoded e
> o que deve virar variável de ambiente/secret na stack nova.

---

## Ciclo do Kefra no servidor Go (#342)

- O renascimento semanal usa terça-feira às 12h em `time.Local`, o fuso local
  do processo. Configure o fuso do sistema/container antes de iniciar o
  tmserver; não há agenda própria configurável para esse evento.
- A migração `0023_kefra_state` cria o registro global do ciclo em PostgreSQL.
  `LoadKefraState`/`SaveKefraState` são RPCs internos; o cliente WYD não muda.
  Execute a migração e atualize o dbserver antes do novo tmserver.
- `-kefra-live` / `W2PP_KEFRA_LIVE` é somente a semente quando ainda não existe
  registro: `false` (padrão) inicia com boss vivo e cidade fechada; `true`
  inicia derrotado, com cidade aberta. **O registro persistido prevalece**
  sobre a flag, inclusive depois de reinícios.
- A leitura inicial é assíncrona. Até concluí-la, o evento não gera mobs nem
  aceita entradas na cidade. Leituras e gravações que falham são registradas
  nos logs e repetidas após cinco segundos, sem bloquear o loop. Gravações
  são serializadas e revisões antigas ou repetidas não sobrescrevem o banco.
- O encerramento normal salva também o estado mais recente do evento, mesmo
  se houver uma gravação anterior em andamento. Uma queda abrupta enquanto
  o banco está indisponível pode perder mudanças ainda não confirmadas.
- Se o servidor estiver desligado na terça às 12h, a ocorrência é pulada no
  carregamento. A cidade continua aberta até a próxima terça online.
- Sem banco (`NopPersistence`), o ciclo funciona somente na memória do processo:
  reiniciar usa novamente a flag inicial e não preserva a derrota anterior.

## 0. Variáveis de ambiente (`.env`, compose, flags)

O repositório **não versiona segredo**: `./.env` é gitignored e `.env.example` é o catálogo
versionado. `docker compose` carrega `./.env` sozinho e o **shell vence o arquivo**, então o
liga/desliga por execução continua valendo:

```bash
cp .env.example .env                                 # primeira vez
W2PP_RATES_EDITING=true docker compose up -d openwyd-nextgen1-tmsrv  # só nesta subida
```

Precedência em toda a stack:

| Camada | Fonte | Exemplo |
|---|---|---|
| Argumento de CLI | vence tudo | o compose fixa `-addr :8281`, `-content /content`, `-client-version 12000` |
| Variável de ambiente | default da flag (`os.Getenv`) | `W2PP_RATES_EDITING`, `W2PP_CONTENT`, `W2PP_DB_DSN` |
| `./.env` | interpolado pelo compose (`${VAR:-default}`) | `POSTGRES_PASSWORD`, `W2PP_PORTAL_SESSION_SECRET` |
| Banco / `tmserver/content/` | **conteúdo**, nunca env | rates, drop, preço, EXP, NPCs (overlays do portal) |

Como o compose passa `-addr`, `-content` e `-dbserver` explicitamente, mudar `W2PP_CONTENT` no
`.env` **não** afeta os containers — só os binários rodando no host. Porta e caminho dentro do
compose mudam em `docker-compose.yaml`; segredo e liga/desliga mudam no `.env`.

Chaves que o compose interpola hoje: `POSTGRES_PASSWORD` (as três DSNs derivam dela),
`W2PP_PORTAL_SESSION_SECRET`, `W2PP_PLAY_CODE_SECRET` e os cinco overlays (`W2PP_MOB_STAT_EDITING`,
`W2PP_ITEM_EFFECT_EDITING`, `W2PP_RATES_EDITING`, `W2PP_DROP_CONFIG_EDITING`,
`W2PP_TELEPORT_EDITING`). Os overlays são lidos **no boot** (o de teleporte a cada 15 s). O catálogo
de NPCs **não** é overlay: `npc_definition` é a única fonte de spawn (issue 57) — o tmserver registra
os geradores e materializa as entidades a partir dela, e sem `-dbserver`/`-content` o mundo sobe sem
NPCs. O portal tem catálogo próprio em `portal/.env.example` para rodar no host com
`make portal-dev`.

O hot-reload do tmserver não tem mais compose próprio (o antigo `docker-compose.dev.yaml` foi
removido na DEVOPS-01): suba só o banco com `docker compose up -d openwyd-postgres` e rode
`make run` (`go run ./tmserver/cmd/tmserver`). Esse recorte não passa `-dbserver`/`-content`, então
os overlays `W2PP_*_EDITING` ficam de fora de propósito — com eles ligados o boot falha com
`-teleport-editing requires -dbserver (config source)`. Sem catálogo o mundo sobe **sem NPCs**: é o
recorte de desenvolvimento, não uma topologia jogável.

## 0.0 Camadas do compose e padrão de nomes (DEVOPS-01)

Um nome por coisa: **serviço = container = imagem**, para `docker ps` não ter apelido genérico. As
imagens de terceiro mantêm o nome upstream (`postgres`, `prom/prometheus`), porque renomear o que
não é nosso só cria divergência com a doc do fornecedor.

| Serviço / container | Imagem | Papel |
|---|---|---|
| `openwyd-nextgen1-tmsrv` | `openwyd-nextgen1-tmsrv:latest` | canal 1 (ServerIndex 0), CPSock 8281 |
| `openwyd-nextgen2-tmsrv` | `openwyd-nextgen2-tmsrv:latest` | canal 2 (ServerIndex 1), CPSock 8282 |
| `openwyd-dbserver` | `openwyd-dbserver:latest` | persistência gRPC 7514 |
| `openwyd-binserver` | `openwyd-binserver:latest` | billing gRPC 3000 |
| `openwyd-webserver` | `openwyd-webserver:latest` | web-api gRPC 7600 |
| `openwyd-postgres` | `postgres:17-alpine` | banco |
| `openwyd-db-roles` | `postgres:17-alpine` | one-shot das roles (DB-01) |
| `openwyd-portal` | `openwyd-portal:latest` | Next.js 3000 |
| `openwyd-logpipe` | `openwyd-logpipe:latest` | logs por SSE |
| `openwyd-prometheus` | `prom/prometheus:v3.5.0` | métricas (perfil `monitoring`) |
| `openwyd-grafana` | `grafana/grafana:12.2.0` | dashboards (perfil `monitoring`) |
| `openwyd-node-exporter` | `prom/node-exporter:v1.9.1` | disco do host (perfil `monitoring`) |
| `openwyd-mkdocs` | `openwyd-mkdocs:latest` | site de docs (perfil `docs`) |

Três arquivos, cada um com um papel, sem duplicar definição:

| Arquivo | Camada | O que acrescenta |
|---|---|---|
| `docker-compose.yaml` | base | serviço/imagem/comando, healthchecks, dependências, volumes. **Nenhuma porta publicada** |
| `docker-compose.override.yaml` | dev (automático) | publica portas (loopback + IPs de canal `192.168.100.3/.4`) |
| `docker-compose.mtls.yaml` | opcional | certificados de cliente/servidor nos links gRPC |

```bash
docker compose up -d                    # base + override (dev)
docker compose --profile monitoring up -d
docker compose -f docker-compose.yaml up -d      # produção: sem porta no host
docker compose -f docker-compose.yaml -f docker-compose.override.yaml \
               -f docker-compose.mtls.yaml up -d --build   # mTLS local
```

**Ordem de subida é declarada, não implícita.** `depends_on` com `condition:` e healthcheck de
verdade em cada serviço:

- `openwyd-postgres` → healthcheck `pg_isready`;
- `openwyd-db-roles` (one-shot) → `service_completed_successfully` antes de dbserver/webserver;
- `openwyd-dbserver`/`openwyd-webserver` → depois do Postgres saudável e das roles;
- `openwyd-nextgen{1,2}-tmsrv` → só com dbserver e binserver **saudáveis** (não `started`);
- `openwyd-portal` → depois de webserver e logpipe saudáveis;
- `openwyd-grafana` → depois do Prometheus saudável.

Os serviços Go são `distroless` (sem shell, sem `wget`), então o healthcheck é o próprio binário:
`CMD ["/app", "healthcheck"]` consulta `http://127.0.0.1:<porta>/healthz` e sai 0/1
(`internal/metrics/healthcheck.go`; a porta vem de `W2PP_METRICS_ADDR`, default 9101–9104).
Isso evita copiar `grpc_health_probe` e mantém a imagem mínima.

## 0.1 Segredos: gate, rotação e restore

`make gate` executa `scripts/check-secrets.sh`. O script recusa `.env` real versionado, exige
`SESSION_SECRET` diferente do default e com pelo menos 32 caracteres quando `W2PP_ENV` (ou
`APP_ENV`/`NODE_ENV`) é `production`, `staging` ou `homolog`, e roda `gitleaks` sobre a árvore. Sem o
binário no `PATH`, ele usa `.tools/bin/gitleaks` ou, em último caso, uma varredura determinística de
fallback — e diz qual caminho usou. No CI, o job `Secret scan` instala o gitleaks **pinado** e roda
em todo push/PR, inclusive mudança só de documentação.

```bash
# local (uma vez por máquina, fica em .tools/ que é gitignored)
GOBIN="$PWD/.tools/bin" go install github.com/zricethezav/gitleaks/v8@v8.28.0

# gate local; em dev o default é aceito, fora de dev não
bash scripts/check-secrets.sh
W2PP_ENV=production W2PP_PORTAL_SESSION_SECRET="$(openssl rand -base64 48)" \
  bash scripts/check-secrets.sh
```

O `.gitleaks.toml` só permite material **não versionado** (`.env`, `portal/.next`, `client`,
`.tools`, `tmp`) e os defaults de desenvolvimento já públicos (senha `dev` do Postgres e a chave
`openwyd-dev-crossover`). Nenhuma regra do gitleaks é desligada; um achado novo reprova o job.

**Rotação do `SESSION_SECRET` do portal:** gere um valor novo com `openssl rand -base64 48`, grave no
secret store do ambiente (ou no `.env` local, nunca no git), recrie só o portal
(`docker compose up -d --force-recreate portal`) e confirme que a sessão antiga deixa de valer
(requisição autenticada → 401/redirecionamento para `/login`). Rotacione também
`W2PP_PLAY_CODE_SECRET` quando ele estiver ligado; os códigos de uso único emitidos antes deixam de
valer. As senhas das roles (`OPENWYD_MIGRATOR_PASSWORD`, `OPENWYD_APP_PASSWORD`,
`OPENWYD_RO_PASSWORD`) são trocadas rodando o serviço one-shot `db-roles` com os novos valores e
recriando `dbserver`/`webserver`; o backup não guarda segredo em texto claro.

**Backup/restore:** `scripts/db-backup.sh` gera o dump e `scripts/db-restore.sh` o aplica num banco
descartável; registre a data do último restore validado no runbook de operação. Um restore não
recria `.env` nem secrets: eles vêm do secret store e devem ser reaplicados antes de subir a stack.

---

## 1. Topologia de deploy

**3 processos** (todos Win32), tipicamente na mesma máquina ou LAN:

```
            ┌───────────────┐         ┌──────────────┐
 WYD.exe ──►│ TMSrv  :8281  │────────►│ DBSrv :7514  │──► arquivos ./account/*
 (cliente)  │ (jogo)        │         │ (persistência)│
            └──────┬────────┘         └──────┬───────┘
                   │ :3000 (billing)          │ (NPServer / contas externas)
                   ▼                          ▼
            ┌──────────────┐          ┌──────────────┐
            │ BISrv        │          │ NPTool/NPSrv │
            │ (billing)    │          └──────────────┘
            └──────────────┘
```

- **Portas hardcoded** (`Basedef.h:104-105`): `GAME_PORT = 8281` (cliente→TMSrv),
  `DB_PORT = 7514` (TMSrv→DBSrv). Billing em `:3000` (de `biserver.txt`).
- **Portas de observabilidade (stack Go):** `tmserver :9101`, `dbserver :9102`,
  `binserver :9103`, `webserver :9104`; o segundo canal `tmserver2` publica `:9111`.
  Servem `/metrics` (texto Prometheus), `/healthz` e `/readyz`; a flag `-metrics-addr`
  (`W2PP_METRICS_ADDR`) controla o endereço e vazio desliga. São publicadas em `127.0.0.1`
  pelo compose e não substituem as portas de jogo/gRPC.
- **Métricas × logs (não são redundantes):** `/metrics` é série temporal numérica para
  scrape/alerta (contadores, gauges, histogramas); o `logpipe` lê o socket do Docker e o
  `/logs` do portal mostra o texto linha a linha, com filtro e pausa. O primeiro responde
  "quantos/quão rápido/quando piorou"; o segundo responde "o que aconteceu nesta
  requisição". Não se substituem e não precisam de um segundo coletor de logs.
- **`trace_id` ponta a ponta (OBS-02):** cada conexão CPSock e cada ação admin do BFF gera
  um id de 32 hex. Ele viaja no metadata gRPC `x-trace-id` (tmServer→dbServer/binServer,
  portal→web-api) e aparece como `trace_id=` nas linhas de log dos quatro serviços. Use
  `docker compose logs --since=10m | grep <id>` para seguir uma requisição inteira. O id
  **nunca** é label de métrica (cardenalidade).
- **Prometheus + Grafana (OBS-03), perfil opcional:**
  ```bash
  docker compose --profile monitoring up -d
  curl -s http://127.0.0.1:9090/-/ready
  curl -s 'http://127.0.0.1:9090/api/v1/alertmanagers' >/dev/null   # sem Alertmanager: os alertas aparecem em /api/v1/alerts
  ```
  Publica `127.0.0.1:9090` (Prometheus), `127.0.0.1:3001` (Grafana, admin/admin por
  default de dev) e `127.0.0.1:9100` (node-exporter). Prometheus coleta os cinco alvos
  Go (`tmserver:9101`, `tmserver2:9101`, `dbserver:9102`, `binserver:9103`,
  `webserver:9104`) mais o exportador do host. As regras vivem em
  `monitoring/alerts.yml`: fila de entrega > 50 por 2 min, RPC com erro > 1%,
  mundo sem tick (`time() - world_tick_timestamp_seconds > 5`) e disco < 10% livre.
  O dashboard `monitoring/grafana/dashboards/wyd-overview.json` é provisionado no
  boot (tick p50/p95/p99, RPC por código, pool, fila, disco).
  Validar as regras sem subir a stack:
  ```bash
  docker run --rm -v "$PWD/monitoring:/m" --entrypoint promtool \
    prom/prometheus:v3.5.0 test rules /m/alerts_test.yml
  ```
  Em Docker Desktop/Windows o node-exporter mede os sistemas de arquivos da VM
  Linux do Docker, **não** o volume `C:` — o alerta de disco continua útil para
  a VM que hospeda o Postgres, mas não vê o disco do Windows.
  Sem `--profile monitoring` nada disto sobe: a stack de jogo não muda.
- **Auto-restart por `.bat`:** ex. `DBsrv/run/DBsrv.bat` é um loop `:Loop … DBsrv.exe … goto Loop`
  — reinicia o processo se ele cair. (Há `.bat` equivalente para o TMSrv.) Substituir por
  supervisor real (systemd/k8s/Docker restart policy) na stack nova.
- `deletar Logs.bat` na raiz de `tmserver/content/` — housekeeping manual de logs.

---

## 2. Catálogo de configuração

### 2.1. `TMsrv/run/gameconfig.txt` (config principal do jogo)

Parseado posicionalmente por seção. Campos confirmados (valores do exemplo):

| Seção | Campo | Exemplo | Efeito |
|-------|-------|--------:|--------|
| Drop Item Event | `evindex/evdelete/evon/evitem/evrate/evstart` | 0/0/0/0/0/0 0 0 1 | evento de drop global (Fase 4 §2.3) |
| Etc Event | `double` | 200 | exp/drop dobrado (DOUBLEMODE) |
| | `deadpoint` | 1 | penalidade de morte |
| | `dungeonevent` | 1 | evento de dungeon |
| | `statsapphire` | 30 | safira/stats |
| | `battleroyal` | 0 | BR on/off (`BrState`) |
| Billing | `billmode` | 0 | modo de billing (`BILLING`) |
| | `freeexp` | 35 | nível até onde é grátis (`FREEEXP`) |
| | `charselbill` | 0 | cobrança na seleção |
| | `potioncount` | 10 | limite de poções |
| | `partybonus` | 100 | bônus de party (`PARTYBONUS`, Fase 4 §1.2) |
| | `guildboard` | 0 | quadro de guilda |
| Item Drop Bonus | matriz 8×8 (`g_pDropBonus[]`) | 50… | bônus de drop por slot (Fase 4 §2.2) |
| Treasure | tabela | 0… | `g_pTreasure[8]` |
| Etc Settings | `partydif` | 200 | diferença de nível p/ party (`PARTY_DIF`) |
| | `kefrastatus` | 156 | estado boss Kefra (`KefraLive`) |
| | `GTorreHour` | 22 | hora da Guerra de Torre |
| | `RVRHour` | 23 | hora do RvR |
| | `DropItem` | 0 | drop habilitado (`isDropItem`) |
| | `BRHour` | 19 | hora do Battle Royal |
| | `maxNightmare` | 3 | limite de pesadelo |
| | `PotionDelay` | 100 | cooldown de poção (ms) |

### 2.2. Rede / IPs

| Arquivo | Conteúdo do exemplo | Significado |
|---------|---------------------|-------------|
| `TMsrv/run/localip.txt` | `192.168.18.12` | IP de bind do TMSrv |
| `DBsrv/run/localip.txt` | `192.168.18.12` | IP de bind do DBSrv |
| `TMsrv/run/biserver.txt` | `54.207.102.145 3000` | **IP+porta do BISrv** (billing) — IP público AWS! |
| `TMsrv/run/serverlist.txt` | `0 0 192.168.18.12` / `0 1 …` | `<grupo> <canal> <ip>` dos canais |
| `DBsrv/run/serverlist.txt` | idem | mesma lista (DB) |
| `*/serverlist.bin` | binário | versão compilada da serverlist (enviada ao cliente) |
| `DBsrv/run/redirect.sample.txt` | `192.168.18.12 8895 admineu admineu1` | redirect/admin (host porta user pass) |

### 2.3. Admin / segurança / MAC

| Arquivo | Exemplo | Significado |
|---------|---------|-------------|
| `*/admin.txt` | `0 192.168.18.12` | `<nível?> <IP autorizado a admin>` |
| `DBsrv/run/Mac.txt` | `0 1060404036.-1187033502.…` | MAC bloqueado/autorizado (`STRUCT_BLOCKMAC.Mac[4]`) |
| `TMsrv/run/admin.txt` | `0 192.168.18.12` | idem TMSrv |

> Comandos de admin chegam por `_MSG_MessageChat` (comandos `/`) e são autorizados por IP
> (`admin.txt`) — ver handler `_MSG_MessageChat` (Fase 5, lote 2).

### 2.4. DBSrv

| Arquivo | Exemplo | Significado |
|---------|---------|-------------|
| `DBsrv/run/config.txt` | `Sapphire 1` / `LastCapsule 0` | estado global (safira/cápsula) |
| `DBsrv/run/settings.txt` | URLs + strings de UI (launcher) | textos do cliente/launcher (i18n) |
| `DBsrv/run/Server.txt` | `0` | `ServerIndex` |

### 2.5. Conteúdo / regras (referência cruzada)

Carregados na inicialização (detalhados nas Fases 2 e 4): `ItemList.csv/.bin`, `SkillData.csv`,
`NPCGener.txt`, `HeightMap.dat`, `AttributeMap.dat`, `LevelItem.txt`,
`Regions.txt`, `Guard.txt`, `InitItem.csv`, `QuestDiaria.txt`, `Language.txt`, e
`Settings/{CompRate,SancRate,CastleQuest,QuestsRate,MobMerc}.txt`.

**Migração para o banco (issue 58, concluída).** Quatro desses arquivos saíram do repositório na fase
5: o banco é a única fonte do runtime e o dado vem dos seeds do baseline.

| Arquivo | Tabela | Conjunto do importador |
|---|---|---|
| `Common/ItemList.csv` | `item_catalog` | `item` |
| `Common/SkillData.csv` | `skill_catalog` | `skill` |
| `TMsrv/run/npc/*` (STRUCT_MOB) | `npc_template` | `npc-template` |
| `TMsrv/run/InitItem.csv` | `world_item_seed` | `world-item` |

Reimportar de uma árvore **externa** (o `tmserver/content/` do repositório não tem mais esses arquivos) e
conferir:

```bash
docker compose run --rm --entrypoint importcontent dbserver \
  -content /content -dsn "postgres://postgres:<senha>@db:5432/postgres"
docker compose run --rm --entrypoint importcontent dbserver \
  -content /content -dsn "postgres://postgres:<senha>@db:5432/postgres" -check
```

O comando é `tmserver/cmd/importcontent` (os parsers vivem em `tmserver/internal/content`, que o
`dbserver` não pode importar por visibilidade de `internal`); a imagem do tmserver o contém. O
`-check` sai 1 quando o banco diverge do conteúdo — é o gate para editar `tmserver/content/` sem reimportar.

**Encoding:** os CSVs são **Windows-1252**. Qualquer import que grave texto no Postgres precisa
passar por `internal/textenc.ToUTF8`; sem isso o banco recusa com
`invalid byte sequence for encoding "UTF8"` no primeiro nome acentuado.

**Quem lê o quê hoje (issue 58, fase 5).** O tmServer busca os quatro catálogos por gRPC
(`ContentCatalogService`) logo depois de abrir a conexão com o dbserver e **não tem mais fallback**:
banco sem catálogo recusa o boot (`load content catalogs from the database` / `item_catalog is
empty`). Um mundo sem itens sobe "saudável" e só quebra no primeiro combate.

Mapas (`HeightMap/AttributeMap`), ícones, taxas (`CompRate/SancRate/QuestsRate`), `CastleQuest.txt`,
`serv00.htm` e `BaseMob/BaseSummon` continuam em arquivo. O `worldatlas` passou a ler
`world_item_seed` e `npc_definition` pelo `-dsn`.

**dbserver:** o `serve` não lê mais `NPCGener.txt`. O reconcile a cada boot — que fazia
`npc_shop_item` divergir do seed (6.711 × 3.538) — foi removido; ele só verifica que existe catálogo
(`npc catalogue ready content_definitions=…`) e falha fechado se estiver vazio.

### 2.6. Estado persistente de mundo (escrito em runtime)

`account/`, `Guilds.txt`, `GuildInfo`, `Guild_<x>_<y>.txt`, `ChampionCity_<x>_<y>.txt`,
`Chall_<x>_<y>.txt`, `Ranking.txt`, `Guild.txt`, `Chall.txt` (estado de zona/guerra).

---

## 3. Hardcodes que precisam virar config/ambiente

| Hardcode | Local | Vira |
|----------|-------|------|
| `GAME_PORT = 8281`, `DB_PORT = 7514` | `Basedef.h:104-105` | env/config |
| `BillServer 54.207.102.145:3000` | `biserver.txt` | env (host/porta) |
| `S:/export/account%d/...` | `CFileDB.cpp:2513` | path configurável |
| `./account/<Key>/<NAME>` | `CFileDB.cpp:2436` | base path configurável |
| IPs em `localip/admin/serverlist` | configs | env/secret |
| Nomes reservados DOS (`COM*/LPT*`) | `CFileDB.cpp:2425` | irrelevante fora de Windows (remover) |

---

## 4. O que vira variável de ambiente / secret na stack nova

**Env (config):**
- `GAME_PORT`, `DB_HOST`, `DB_PORT`, `BILLING_HOST`, `BILLING_PORT`, `SERVER_INDEX`, `SERVER_GROUP`.
- Caminhos: `DATA_DIR` (conteúdo), `ACCOUNT_STORE` (DB/arquivos), `LOG_DIR`, `EXPORT_DIR`.
- Flags de evento: `DOUBLE_EXP`, `FREE_EXP_LEVEL`, `GTORRE_HOUR`, `RVR_HOUR`, `BR_HOUR`,
  `DROP_ENABLED`, `PARTY_BONUS`, `PARTY_DIF`, `MAX_NIGHTMARE`, `POTION_DELAY` (todos do
  `gameconfig.txt`).

**Secret:**
- Credenciais de DB/conexão entre serviços.
- Chave de assinatura de sessão (substituir INITCODE/keytable estática — Fase 1/9).
- Credenciais de billing/redirect (`redirect.sample.txt` tem user/pass em claro → secret).
- `Mac.txt`/listas de admin → mover para storage seguro, não arquivo plano versionado.

---

## 5. Operação (runbook resumido)

1. Subir DBSrv (porta 7514), depois TMSrv (conecta ao DB, escuta 8281), depois BISrv (3000).
2. `.bat` de auto-restart hoje; migrar para supervisor com health-check e backoff.
3. Backup: copiar `account/` + `Guilds*`/estado de zona (são os dados vivos). Na stack nova: backup
   do banco.
4. Logs: `fLogFile`/`fChatLogFile`/`fItemLogFile` (TMSrv) — centralizar (stdout/JSON) na migração.

### 5.1. Disco do Docker Desktop (WSL2) fora do `C:`

O disco que guarda imagens, volumes e build cache Ã© o `docker_data.vhdx` (~24 GB), em
`%LOCALAPPDATA%\Docker\wsl\disk`. Ele **nÃ£o** Ã© uma distro WSL: nÃ£o aparece em `wsl -l -v` nem tem
entrada em `HKCU\...\Lxss`, entÃ£o nada do lado do WSL o move:

| Tentativa | Resultado |
|---|---|
| `.wslconfig` | sÃ³ tem `memory`, `processors` e `swap` â€” nÃ£o existe chave de localizaÃ§Ã£o |
| `wsl --manage <distro> --move` | move distro registrada; a `docker-desktop` tem sÃ³ ~100 MB |
| `WslDataFolder` no `settings-store.json` | **ignorada** pelo backend (testado em 05/11) |
| Settings â†’ Resources â†’ Advanced â†’ *Disk image location* | caminho suportado (`moveWSLDisk`), exige a GUI |

Nesta mÃ¡quina o disco foi movido e o caminho original virou uma **junction**:

```
%LOCALAPPDATA%\Docker\wsl\disk  --Junction-->  D:\Docker\wsl\disk\docker_data.vhdx
```

Procedimento (Git Bash, com o Docker parado):

1. `docker desktop stop && wsl --shutdown` e confirme que o `.vhdx` estÃ¡ sem lock.
2. `MSYS_NO_PATHCONV=1 robocopy "<origem>" "D:\Docker\wsl\disk" /MOVE /E /MT:16` â€” sem
   `MSYS_NO_PATHCONV` os argumentos `/MOVE` e `/E` sÃ£o convertidos em caminhos e o robocopy falha.
3. `New-Item -ItemType Junction -Path "<origem>" -Target "D:\Docker\wsl\disk"` â€” `mklink /J` via
   `cmd` quebra no Git Bash pela mesma conversÃ£o.
4. `docker desktop start` e valide: `docker volume ls` lista `openwyd_pgdata`, o `PG_VERSION` do
   volume responde `17`, o `docker compose up -d` chega em `tmserver listening` e o `.vhdx` no D:
   passa a ter `mtime` novo (sinal de que o Docker usa o disco movido).

Antes de mover, `scripts/db-backup.sh` grava o dump fora do repositÃ³rio â€” Ã© a rede de seguranÃ§a do
banco do jogo. ReversÃ£o: parar o Docker, remover a junction (`rmdir`) e mover a pasta de volta; a
configuraÃ§Ã£o do Docker nÃ£o muda em nenhum dos dois sentidos.
> **Status da Fase 7: COMPLETO** para o catálogo de config e topologia. Parsing posicional exato de
> `gameconfig.txt`/`Treasure`/`Guild.txt` deve ser confirmado contra `CReadFiles.cpp` ao
> reimplementar o loader.
