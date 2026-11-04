# Análise técnica — configuração inicial e mapeamento de funcionalidades

Documento de referência do reconhecimento feito em **2026-11-02** sobre o workspace
`D:\dolorestec\wyd2`. Toda afirmação abaixo foi verificada em disco ou no código; o que é inferência
está marcado como **hipótese**.

> **Atualização 2026-11-04:** `portal/`, `roadmap/` e `client/` foram movidos para dentro de
> `OpenWYD/`, que passou a ser a raiz do repositório; os arquivos de IA da raiz foram consolidados
> em `OpenWYD/AGENTS.md` + `OpenWYD/.agents/skills`. A árvore abaixo já reflete o layout novo.

---

## 1. Inventário do workspace

```
OpenWYD/                      # raiz do repositório (2026-11-04 em diante)
├── AGENTS.md, CLAUDE.md      # instruções para agentes
├── .agents/skills/           # skills operacionais (WYD, UI/UX, openspec)
├── portal/                   # UI de administração (Next.js)
├── roadmap/                  # este roadmap
├── client/                   # cliente Windows modificado (não versionado)
├── tmserver/ dbserver/ binserver/ webserver/ internal/   # servidor Go
└── Release/, Source/, docs/  # conteúdo do jogo, fontes legados, engenharia reversa
```

### 1.1 Servidor — Go

- Módulo Go `github.com/jeanluca/w2pp-openwyd`, `go 1.26.10`.
- Serviços: `tmserver` (jogo, CPSock), `dbserver` (gRPC + PostgreSQL), `binserver` (billing),
  `webserver` (web-api gRPC para a futura plataforma Next.js).
- Compartilhado na raiz `internal/`: `store`, `migrations` (até `0033_*`), `domain`, `secure` (mTLS),
  `secret` (argon2id).
- Conteúdo de jogo: `Release/` (67 MB) — `Common/`, `TMsrv/run/`, `DBsrv/run/`.
- Legado C++: `Source/Code/` (TMSrv, DBSrv, BISrv, Basedef, CPSock, ClientPatch_v7662, DropTool,
  EDITAPPMOB/SHOP, NPTool, ExpTool) — usado como referência de paridade.
- Documentação: `docs/migration/` (10 fases de engenharia reversa + contratos dos 58 handlers),
  `docs/agents/` (análises por componente), `docs/game.md` (comandos), `docs/integrations/`.

**Achado bloqueante #1 — working tree parcial.** O repositório estava com `core.sparseCheckout=true`
e a lista `Source/Code, docs, tmserver`. Todo o resto — `Release/`, `scripts/`, `internal/`, `api/`,
`dbserver/`, `binserver/`, `webserver/`, `development-guidelines/`, `serverlist bin/` — existia no
git mas **não em disco**. Sintomas que isso produz: `go build ./...` falha (imports ausentes),
`scripts/run-local.sh` não existe, servidor não tem conteúdo para carregar. Corrigido com
`git sparse-checkout disable` (issue 01).

### 1.2 Cliente — `client/`

- `WYD.exe` (6,49 MB), `WydConfig.exe`, `Updater.exe`, `Launcher.hat`.
- Derivado do **WYD Global / RaidHut**: `config.json` aponta o launcher para
  `https://d16tndoz98nwcc.cloudfront.net/info.json`; `.info.json` lista os jogos Globais com
  `executable: WYD.exe`.
- Proteções: `JProtectDll.dll`, `JPGLib.dll`, `jlock101.sys`/`jlock102.sys`, `version.dll` (proxy),
  `d3dx9_42.dll`.
- Dados: `ItemList.bin` (2,5 MB), `SkillData.bin`, `InitItem.bin`, `ItemSet.bin`, `MountData*.bin`,
  `Missionitems.bin` (26,9 MB), `itemname.bin`, `itemicon.bin`, `serverlist.bin` (7.044 B),
  `sndata.bin`/`sn.bin` (nomes de canal ofuscados), `Config.bin` (32 B), `Graphics.ini`, `Music.ini`.
- `WYD.log` registra apenas inicialização gráfica (`VertexShader HW Accel Enabled`,
  `Error in Init Render Target Texture`) — não há log de conexão nesse arquivo até haver tentativa.

**Achado #2 — launcher tentou atualizar e falhou.** `.remote_serverlist.bin` (1.245 B, 01/11 04:30)
é uma **página HTML de erro** (`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN"`), não um
serverlist. Consequência: qualquer execução do `Updater.exe`/launcher tende a sobrescrever o
`serverlist.bin` local. Para apontar ao servidor próprio, abrir `WYD.exe` direto.

**Achado #3 — versão de protocolo é uma incógnita a medir, não a supor.** `docker-compose.yaml` sobe
o tmServer com `-client-version 12000`, e `handler/login.go` documenta que o build 7662 "Cavaleiros
de Kersef" manda 12000 enquanto `Basedef.h` documenta 7640. O cliente desta pasta é de outra
linhagem (Global/RaidHut), então o valor **precisa ser lido** no log de rejeição
(`got=… want=…`) em vez de assumido.

---

## 2. Configuração inicial para subir o servidor

### 2.1 Banco de dados

- PostgreSQL 17 (`postgres:17-alpine` no compose), volume Docker `pgdata`.
- DSN por ambiente: `W2PP_DB_DSN=postgres://postgres:dev@db:5432/postgres` (padrão do compose).
- Migrações embutidas no binário (`internal/migrations`), aplicadas no boot do primeiro serviço que
  conectar (`store.Migrate`). Nada de SQL manual.
- Semeadura de conta: `dbserver seed-account -name test -pass test123` (idempotente, argon2id, sem
  personagens).
- Alternativa de importação de contas legadas: `dbserver convert` (arquivos `STRUCT_ACCOUNTFILE`,
  versões 4294/7500–7600/7952 detectadas por tamanho).

### 2.2 Servidor

| Serviço | Porta publicada | Observação |
|---------|----------------|------------|
| tmserver | **8281** | CPSock (jogo); header de 12 bytes |
| tmserver | **80** | HTTP de status `serv00.htm`; o cliente sonda antes de conectar |
| dbserver | interna 7514 | gRPC |
| binserver | interna 3000 | billing allow-all |
| webserver | 7600 | web-api (para BFF Next.js) |

Configuração relevante do `tmserver` (todas com env `W2PP_*`): `-content /Release`,
`-dbserver dbserver:7514`, `-binserver binserver:3000`, `-status-addr :80`,
`-client-version 12000`, `-training-field-exp 6`, `-affect-scale-pct 15`,
`-reject-checksum` **desligado**, `-kefra-live` como semente do ciclo semanal.

Boot válido produz, nesta ordem, no log: `tmserver config` → `content loaded` (items/skills/rates) →
`status server listening` → `base mob templates loaded` → `walkability grid baked` →
`NPCs spawned` (milhares de mobs) → `world items seeded` → `tmserver listening`.

### 2.3 Conteúdo carregado no boot (obrigatório × opcional)

| Caminho | Uso | Falha é… |
|---------|-----|----------|
| `Common/ItemList.csv` | catálogo de itens, preços, efeitos, requisitos | fatal |
| `Common/SkillData.csv` | catálogo de skills | fatal |
| `Common/Settings/{CompRate,SancRate,QuestsRate}.txt` | taxas de combine/refino/quest | fatal |
| `TMsrv/run/NPCGener.txt` + `npc/` | população do mundo (≈1.991 templates) | warning (mundo vazio) |
| `TMsrv/run/HeightMap.dat` + `AttributeMap.dat` | pathfinding e Campo de Treino | warning (IA "cega") |
| `DBsrv/run/BaseMob/{TK,FM,BM,HT}` | template por classe no login/criação | warning (char sem equip visual) |
| `TMsrv/run/InitItem.csv` | portões/objetos do mundo | warning |
| `Common/serv00.htm` | status do canal | fallback interno |

### 2.4 Cliente

1. Gerar `serverlist.bin` com `serverlist bin/serverlist editor.exe` apontando
   `IP_DO_SERVIDOR:8281` (backup do atual primeiro).
2. Liberar `8281/tcp` e `80/tcp` no firewall do host do servidor.
3. Abrir `WYD.exe` direto; não passar pelo launcher.
4. Conferir o handshake nos logs do tmServer (`recv packet type=0x020d`) e, se houver rejeição,
   alinhar `-client-version` ao valor `got=`.

---

## 3. Mapeamento de funcionalidades

### 3.1 Sessão e personagens

| Fluxo | Type | Handler Go | Estado |
|-------|------|-----------|--------|
| Login de conta | `0x020D` | `handler/login.go` | ✅ (inclui lockout por senha errada, `account.is_blocked`, `role`) |
| Criar personagem | `0x020F` | `handler/character.go` | ✅ (4 classes; BaseMob + equip inicial + 1.000.000 gold) |
| Deletar personagem | — | `handler/character.go` | ✅ |
| Entrar no mundo | `0x0213` | `handler/character.go` | ✅ (`CNFCharacterLogin` com spawn da última cidade) |
| Logout / seleção | — | `handler/character.go` | ✅ (`SaveCharacterThen` confirma após commit) |
| Restart (morte) | `0x0289` | `handler/character.go` | ✅ (HP=2 + recall à última cidade) |
| Segurança de conta | — | `handler/account.go` | ✅ (PIN hash) |

### 3.2 Movimento, mundo e cidades

| Fluxo | Type | Implementação | Estado |
|-------|------|--------------|--------|
| Andar / ação | `0x036C` (+Action2/3) | `handler/movement.go` | ✅ |
| Motion / efeitos | — | `handler/movement.go` | ✅ |
| Trocar de cidade | — | `handler/movement.go` + `world/city.go` | ✅ (5 cidades + spawn) |
| Teleporte por tile | `0x0290` | `world/teleport.go` | ✅ (rotas + custo, Noatum como hub) |
| Visão/saída de mob | — | `handler/view.go` + grid | ✅ |
| PK mode | — | `handler/movement.go` | ✅ |
| Mapa de atributos/altura | — | `content.Grid` + `route.Bake` | ✅ |

### 3.3 Combate e progressão

| Fluxo | Implementação | Estado |
|-------|--------------|--------|
| Dano melee/skill server-authoritative | `handler/combat.go`, `combat/` | ✅ (acerto/parry/reflect, resist) |
| Skills e affects | `handler/skill.go`, `world/affect.go`, `handler/affect_score.go` | ✅ (catálogo, cast, buff, expiração) |
| IA de mob (aggro, persegue, ataca, recua) | `handler/mobai.go`, `world/tick.go` | ✅ (clan hostility, ranged, pathfinding `route.Next`) |
| Roaming/patrulha e geradores | `world/generator.go` | ✅ (RouteType 0–6, grupos, respawn) |
| Drop e loot | `loot/`, `ItemDropList.txt` | ✅ (LCG parity) |
| EXP, level-up e curva | `level/`, `handler/level*` | ✅ Mortal/Arch; Celestial com curva placeholder |
| Morte/respawn do player | `handler/character.go` | ✅ |
| Evocações (BM) e transformação | `handler/summon.go`, `handler/transform.go` | ✅ |

### 3.4 Itens, comércio e economia

| Fluxo | Type | Implementação | Estado |
|-------|------|--------------|--------|
| Loja de NPC | `MsgREQShopList`/`Buy`/`Sell` | `handler/shop.go` | ✅ |
| Usar/equipar item | `0x0373` | `handler/item.go` | ✅ (requisitos de nível/atributo) |
| Mover/equipar por drag | `0x0376` | `handler/item.go` | ✅ |
| Dropar / pegar do chão | `0x0272` / `0x0270` | `handler/item.go` | ✅ |
| Dividir / deletar / atualizar | — | `handler/item.go` | ✅ |
| Cargo (banco da conta) | `0x0387` / `0x0388` | `handler/cargo.go` | ✅ (gold + itens) |
| Trade P2P | `MsgTrade`/`TradingItem`/`QuitTrade` | `handler/trade.go` | ✅ |
| Autotrade / loja pessoal | `SendAutoTrade`/`ReqTradeList`/`ReqBuy` | `handler/autotrade.go` | ✅ |
| Refino com poeira | `UseItem` | `refine/` + `SancRate.txt` | ✅ (+9 e bônus por `nPos`) |
| Combines | `0x03A6` + variantes | `combine/`, `handler/combine*.go` | ✅ (receitas parcialmente UNVERIFIED) |

### 3.5 Social, quests e eventos

| Fluxo | Implementação | Estado |
|-------|--------------|--------|
| Chat, sussurro, gritar | `handler/chat.go` | ✅ (`/gritar` consome Trombeta 3330) |
| Party | `handler/party.go` | ✅ |
| Guild (nome, fama, guerra, torre) | `handler/guild*.go`, `world/guild.go` | ✅ parcial (criação de guild ⏳) |
| Quests de NPC | `handler/quest.go` | ✅ (Kibita, Jardineiro, Coveiro, Perzen, classe, montaria…) |
| Pista de Runas | `handler/runequest*.go` | ✅ (6 salas, chefe, prêmios) |
| Campo de Treino | `handler/training*.go` | ✅ EXP ×6 (trava nível 36+ ⏳) |
| Kefra (ciclo semanal) | `handler/kefra.go` | ✅ persistido no banco |
| Nightmare / worlds events / clima | `handler/worldevents*.go` | ✅ (parte UNVERIFIED) |
| Celestial / Sub-Celestial | `handler/celestial*.go` | ✅ parcial (transformação Mortal→Celestial ⏳) |
| Ranking, cápsula, selo | `handler/*` | ✅ básico |

### 3.6 Administração

| Recurso | Implementação | Estado |
|---------|--------------|--------|
| Comandos de jogador (`/armia`, `/cp`, `/nick`, `/gritar`, `/buffs`…) | `handler/chat.go` | ✅ |
| Comandos de GM (`/gm …`) | `handler/gm.go` | ✅ (14 subcomandos, auditado) |
| Papéis e banimento | `account.role`, `account.is_blocked` | ✅ |
| Edição de NPC por moderador | `handler/npccfg` + `-npc-editing` | ✅ (off por padrão; exige `import-npcs`) |
| Edição de stats de mob | `mobstat` + `-mob-stat-editing` | ✅ (aplica no boot) |
| Eventos globais via web | `worldcfg` + `delivery_queue` (plano) | parcial |

### 3.7 Protocolo (fundamento de tudo)

- CPSock: HEADER de **12 bytes**; ofuscação por tabela estática (`pKeyWord`); checksum
  **não-rejeitante** (rejeição desligada por padrão).
- `ClientVersion`: 7640 documentado; 12000 no build usado. Comparado no `0x020D`.
- Wire `pack(1)`; saves com alinhamento natural MSVC x86 (`STRUCT_MOB`=816,
  `STRUCT_ACCOUNTFILE`=7952).
- `HEADER.ID = ESCENE_FIELD` (30000) é obrigatório em pacote autoritativo de cena.
- RNG: LCG do MSVC reimplementado; ordem de chamadas é contrato.

---

## 4. Lacunas e riscos identificados

1. **Build Docker instável nesta máquina.** A primeira tentativa morreu com
   `failed to receive status: rpc error: code = Unavailable desc = error reading from server: EOF`
   durante o `COPY . .` de um serviço, após ~54 MB de contexto. Hipótese principal: pressão de
   recurso no Docker Desktop (contexto de build grande + 4 builds paralelos), não erro de código.
   Plano: `docker compose build` por serviço, `.dockerignore` mais agressivo, ou build serializado.
2. **Sem Go no host.** Todo desenvolvimento/teste precisa do container `golang:1.26-alpine`.
3. **Cliente sem prova de conexão.** Não há registro de tentativa de login no `WYD.log`; a versão de
   protocolo e o comportamento das proteções ainda não foram observados.
4. **`Release/` protegido por engano** pelo padrão `[Rr]elease/` do `.gitignore` herdado.
5. **Curva Celestial placeholder** e vários campos UNVERIFIED (DoT, parry, coef. Dex/Str por
   classe×arma) — não calibrar economia em cima disso.
6. **Item novo exige paridade de cliente** (`ItemList.bin`/`itemname.bin`), senão fica invisível.
7. **Dois consumidores do mesmo `ItemList.csv`** (servidor e ferramentas web) — mudança de preço
   afeta loja in-game e plataforma web.

---

## 5. Conclusão operacional

O servidor está **estruturalmente pronto**: código completo, conteúdo completo, topologia definida e
documentação de paridade madura. O que falta para a meta principal é **execução de ambiente**:
buildar a stack nesta máquina, semear conta, validar boot, gerar `serverlist.bin` e medir a versão de
protocolo do cliente. É exatamente o caminho crítico 01→07 do roadmap; nenhuma issue de gameplay deve
ser aberta antes disso.
