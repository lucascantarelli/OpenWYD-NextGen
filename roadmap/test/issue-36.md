# Issue 36 — Rates editáveis (refino, combine, quests) e visão das flags

**Fase:** 6 · **Depende de:** 21 · **Bloqueia:** — · **Esforço:** M · **Estado:** 🟡 (edição ✅, read-through ✅ e aba de flags ✅ no código; faltam a evidência in-game e a verificação ao vivo da aba — Docker Desktop parou por falta de espaço em `C:`)

## Feito — fatia de edição (overlay + `/config`)

- `internal/ratetables`: **um** parser para SancRate/CompRate/QuestsRate, no nível do módulo para
  web-api e tmserver usarem o mesmo (o `tmserver/internal/content` não é importável pelo `webserver`).
  Reproduz as regras do conteúdo: SancRate só sobrescreve os índices listados (o resto mantém o
  compilado) e Âmago cai na própria linha, não na bug de cópia para a linha PO.
- **A chave da célula é textual: `(file, family, name)`** — `name` é o índice do refino, a chave do
  `CompRate.txt` (`Item_+7`) ou `"<tier>.<coluna>"` nas quests. O teste do parser pegou o defeito da
  primeira versão (posição numérica): todas as chaves de uma família colidiam na mesma célula e
  `Compositor Item_+9` sobrescrevia `Item_+7`.
- Migration `0038_content_rates`: `content_rate_override(file, family, name, value, updated_by)` +
  `content_rate_audit(before/after/reason)`, mesmo formato das auditorias 0035–0037.
- `webserver/internal/ratesadmin`: `List` mescla arquivo + overlay e devolve `original`, efetivo, faixa e
  `has_override`; `Upsert` valida a célula contra o arquivo (célula inexistente → `NOT_FOUND`, valor fora
  da faixa → `INVALID` com a mensagem da faixa) e exige motivo; `Reset` apaga o overlay.
- `ContentRatesService` (`GetRates`/`UpsertRate`/`ResetRate`) + `grpcsrv` + wiring no `webserver`.
- Portal `/config`: abas **Refino/Combine/Quests**, agrupadas por família, cada linha com valor do
  arquivo, valor efetivo, `salvar` e `restaurar` (só quando há edição), com motivo obrigatório e o aviso
  de restart/flag. Menu deixou de mostrar `/config` como "em breve".
- Verificação: `go test` verde em `ratetables` (arquivos reais: PO 3 = 85 do arquivo, índices omitidos
  mantêm o compilado, chaves vizinhas do CompRate independentes, faixas) e em `ratesadmin` (motivo,
  célula inexistente, 0..100, teto de 2 bilhões nas quests, player 403); `npx tsc --noEmit` 0;
  migration `0038_content_rates` aplicada.
- Prova pelo gRPC no `webserver` real: `GetRates` devolveu as células com `original`; `UpsertRate`
  `SancRate/PO/3 = 55` mudou o efetivo e marcou `has_override`; `value = 101` foi recusado; `ResetRate`
  devolveu o valor do arquivo. Auditoria com `before`/`after` e o motivo do probe.

## Feito — read-through no tmServer (`-rates-editing`)

- `api/db/v1/db.proto`: `ContentRateConfigService.ListContentRateOverrides` (leitura pura) +
  `ContentRateOverride{file, family, name, value}` — mesma chave textual do overlay.
- `dbserver/internal/grpcsrv/contentrateconfig.go` sobre `store.ListContentRateOverrides`, registrado no
  `dbserver`; `tmserver/internal/dbclient/contentrateconfig.go` como cliente de leitura.
- `tmserver/internal/content/rateoverlay.go`: `ApplyRateOverrides` escreve nas tabelas tipadas já
  carregadas — `SancRate` (âmbar + índice), `CompRate` (família + chave, sem tocar nas chaves vizinhas) e
  `QuestRates` (`<tier>.<coluna>`). Linha que não encaixa na forma que o cliente entende é **contada
  como ignorada**, nunca inventada.
- `tmserver/cmd/tmserver/main.go`: `-rates-editing` / `W2PP_RATES_EDITING` (default false, exige
  `-dbserver` e `-content`), aplicado **uma vez** antes de `handler.New` — como as demais overlays.
  Dois logs: `rate_overrides_applied=N rate_overrides_ignored=M rows=K` e
  `content rate overlay effective sancrate_ori=[...]`, porque o `content loaded` acontece **antes** do
  overlay e sozinho não provaria o valor que o mundo vai usar.
- Verificação (containers one-off da imagem, portas :8282/:8088): com overlay `SancRate/PO/3=55`,
  `rate_overrides_applied=1 rows=1` e `sancrate_ori="[100 100 100 55 70 40 100 0 …]"`; sem a flag,
  nenhuma linha de overlay e `sancrate_ori` com 85 (arquivo intacto) — paridade. O override do probe foi
  revertido pelo RPC e `content_rate_override` voltou a 0 linhas. `go test` verde em `content`
  (`ApplyRateOverrides`: âmbar, chave vizinha do CompRate, tiers de quest, célula inválida ignorada),
  `dbclient`, `dbserver/internal/grpcsrv` e `ratetables`.

## Feito — aba "Flags" (snapshot do processo)

- Migration `0039_server_instance_flags`: `server_instance_flags(server, flags jsonb, captured_at)`, uma
  linha **por boot**. Guarda histórico de propósito: comparar com o snapshot anterior é como se descobre
  que um restart mudou algo.
- `db.proto`: `InstanceFlagsService.RecordInstanceFlags(server, flags_json)` — escrita feita pelo próprio
  serviço no boot; `dbserver/internal/grpcsrv/instanceflags.go` → `store.RecordInstanceFlags`.
- `tmserver`: `dbclient.InstanceFlags.Record` serializa a lista tipada
  `[{name, value, restart_required, hint}]` e ela é gravada logo antes de `handler.New`, com 16 flags
  (addr, status-addr, client-version, dbserver, binserver, content, double-exp, newbie-event, kefra-live,
  training-field-exp, max-nightmare, npc-editing, mob-stat-editing, teleport-editing,
  item-effect-editing, rates-editing). Falha ao gravar vira **warning**, não impede o boot: o snapshot é
  observabilidade, não gameplay.
- `web.proto`/`ratesadmin`: `GetServerFlags` devolve o último snapshot com `available`,
  `captured_at` e `message` ("ainda não gravou"); `store.LatestInstanceFlags` lê a linha mais recente.
- Portal `/config`: seção **Flags do tmserver** acima das taxas — nome, valor atual, efeito e badge de
  restart, com aviso de que nenhuma flag é recarregada em runtime (mudar exige o compose/ambiente e
  `docker compose up -d tmserver`).
- Verificação de código: `go build ./...` 0; `go test` verde em `ratesadmin` (snapshot ausente →
  `available=false`, snapshot com JSON → flag parseada com `restart_required` e `captured_at`),
  `grpcsrv`, `dbclient`, `dbserver/internal/grpcsrv` e `ratetables`; `npx tsc --noEmit` 0.

**Verificação ao vivo (Docker disponível):** migração `0039_server_instance_flags` aplicada e o
  tmserver gravou `tmserver | 2026-11-03 18:11:47Z | 16` flags no boot; `GetServerFlags` respondeu
  `available=true` com `addr=:8281`, `status-addr=:80`, `client-version=12000`, `dbserver=dbserver:7514`,
  `binserver=binserver:3000` e `restart_required=true` em todas; `/config` devolveu **HTTP 200** com
  `Flags do tmserver` e `snapshot do último boot`. Observação de ambiente: `C:` está com 6,7 GB livres
  (95%) e os builds Go só passaram com `GOTMPDIR`/`TMPDIR` em `D:\tmp-go` — a falta de espaço já
  derrubou o daemon uma vez nesta sessão.

Falta também a **evidência in-game** (refino/combine com taxa alterada), que depende de cliente + restart.

## Objetivo

Editar as taxas de conteúdo — refino (`SancRate`), combine (`CompRate`), quests (`QuestsRate`),
castelo (`CastleQuest`) e mercadores (`MobMerc`) — pela web, e ver num só lugar as flags com que o
tmserver está rodando e como mudá-las.

## Contexto / evidência

- As taxas são arquivos em `Release/Common/Settings/` (`SancRate.txt`, `CompRate.txt`,
  `QuestsRate.txt`, `CastleQuest.txt`, `MobMerc.txt`), carregadas no boot; o log prova o que entrou:
  `content loaded comprate_families=7 sancrate_ori="[100 100 100 85 70 40 …]" sancrate_lac="[…]"`.
- `Release/` é `:ro` → edição precisa de overlay no Postgres (mesmo padrão de `npc_definition`).
- As **flags** são de processo (`docker-compose.yaml` `command:` + `W2PP_*`): `-double-exp`,
  `-newbie-event`, `-kefra-live`, `-training-field-exp`, `-max-nightmare`,
  `-kefra-entries-per-scroll`, `-npc-editing`, `-mob-stat-editing`, `-client-version`. Algumas têm
  equivalente em evento global (issue 28) e valem sem restart; as demais exigem restart.
- `gameconfig.txt` **não** é lido pelo Go (só as flags equivalentes) — a tela não deve sugerir
  editá-lo.

## Passos

1. **Migração** `0038_content_rates`:
   ```sql
   CREATE TABLE IF NOT EXISTS content_rate_override (
     file   text NOT NULL,   -- 'SancRate' | 'CompRate' | 'QuestsRate' | 'CastleQuest' | 'MobMerc'
     family text NOT NULL,   -- chave da linha/família no arquivo
     pos    integer NOT NULL, -- coluna dentro da família
     value  integer NOT NULL,
     updated_by bigint NOT NULL REFERENCES account(id),
     updated_at timestamptz NOT NULL DEFAULT now(),
     PRIMARY KEY (file, family, pos)
   );
   ```
2. **Parsers**: reaproveitar `tmserver/internal/content` para ler cada arquivo e expor
   `[]RateRow{file, family, pos, value, label}`. O web-api usa os mesmos parsers (pacote `content` é
   compartilhado) para servir `GetRates`.
3. **Proto**: `ContentRatesService` com `GetRates` (valores efetivos + original + limite por coluna),
   `UpsertRate(file, family, pos, value)`, `ResetRate(...)`, `GetServerFlags` (snapshot **read-only**
   das flags com que o tmserver subiu, gravado pelo tmserver no banco no boot).
4. **tmserver**: grava o snapshot de flags em `server_instance_flags` (nova tabela na mesma migração)
   no boot; aplica `content_rate_override` após carregar cada arquivo, atrás de `-rates-editing` /
   `W2PP_RATES_EDITING` (OFF = arquivos puros). Log de boot passa a incluir
   `rate_overrides_applied=N`.
5. **Portal** `/config` com abas: **Refino** (Ori/Lac por nível, com o valor de hoje), **Combine**
   (por família), **Quests**, **Castelo**, **Mercadores** — cada linha mostra original, atual, e
   "restaurar". Validação 0..100 onde é porcentagem (o backend decide o limite real).
6. Aba **Flags**: tabela `flag | valor atual | efeito | precisa restart?` lida de `GetServerFlags`,
   com o comando exato para mudar (ex.: editar `docker-compose.yaml` e `docker compose up -d
   tmserver`) — **não** editável pela web nesta issue.
7. Aviso: "rates valem após restart; flags de evento (double EXP/newbie) têm caminho sem restart em
   `/eventos`".

## Arquivos

- Criados: `internal/migrations/0038_content_rates.{up,down}.sql`,
  `tmserver/internal/ratescfg/{ratescfg.go,ratescfg_test.go}`,
  `webserver/internal/ratesadmin/{service.go,service_test.go}`,
  `webserver/internal/grpcsrv/ratesadmin.go`,
  `portal/src/app/(admin)/config/page.tsx`, `portal/src/server/actions/rates.ts`,
  `portal/src/lib/rates.ts`.
- Alterados: `api/web/v1/web.proto`, `webserver/cmd/webserver/main.go`,
  `tmserver/cmd/tmserver/main.go`, `portal/src/components/nav.ts`,
  `docs/migration/config-ops.md` (seção "edição pela web").
- Proibido: escrever em `Release/Common/Settings/`; expor edição de flags de processo nesta issue.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
MSYS_NO_PATHCONV=1 docker run --rm -v "D:/dolorestec/wyd2/OpenWYD:/src" -w /src golang:1.26-alpine \
  sh -c "go build ./... && go test ./tmserver/internal/content/ ./tmserver/internal/ratescfg/ ./webserver/internal/ratesadmin/"
```

```bash
W2PP_RATES_EDITING=true docker compose up -d tmserver
docker compose logs --tail 25 tmserver | grep -E "content loaded|rate_overrides_applied"
```

Evidência: mudar a taxa de refino de um nível e conferir no jogo 10 tentativas de refino com a
ferramenta de GM/skill `wyd-economy`; restaurar e ver o valor original voltar após o boot.

## Critérios de aceite

- [x] As tabelas são lidas com um parser único (`internal/ratetables`) — SancRate, CompRate e QuestsRate
- [x] Override aplicado no boot e logado (`rate_overrides_applied` + linha efetiva)
- [x] Flag OFF = valores idênticos aos arquivos (probe sem a flag: `sancrate_ori` com 85 do arquivo)
- [x] Flags de processo apenas exibidas (`GetServerFlags`, `server_instance_flags`), com `restart_required` e o comando de mudança — falta conferir ao vivo
- [ ] Aviso de restart e de `gameconfig.txt` não lido
- [ ] Evidência in-game de mudança de taxa (refino ou combine)

## Riscos / rollback

- **Risco:** taxa fora de faixa quebrar refino (100% ou 0%). **Mitigação:** validação por coluna no
  store + `ResetRate`.
- **Risco:** divergência entre web-api e tmserver na leitura. **Mitigação:** usar o pacote `content`
  compartilhado; teste que compara as duas leituras.
- **Rollback:** flag OFF; `ResetRate` em lote; `.down.sql`.
