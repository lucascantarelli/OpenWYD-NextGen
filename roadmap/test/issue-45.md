# Issue 45 — Guildas, castelo e Kefra (estado do mundo)

**Fase:** 6 · **Depende de:** 32, 41 · **Bloqueia:** — · **Esforço:** L · **Estado:** 🟡 (tudo ✅ — leitura, taxa, dono de cidade/torre, Kefra, escritas de guilda/membro/relação; falta só a **janela de horários**, que é escopo da issue 36)

## Feito — fatia 1: snapshot do mundo, taxa da cidade e Kefra

Um único serviço (`WorldStateAdminService`) cobre as três áreas, porque as três compartilham o mesmo
modelo: estado em Postgres que o tmServer lê **no boot**. Três serviços separados repetiriam a mesma
autorização e o mesmo aviso de restart.

- `web.proto`: `GetWorldState` (guildas já com membros/mestre/cidades, relações, as 5 zonas do reino,
  dono da torre, quest do castelo, Kefra), `SetCityTax` e `SetKefraDefeated`.
- `internal/store/guild_admin.go`: `GuildRoster` — **uma** query devolve todas as filiações (a tabela é
  pequena), e o serviço agrupa em Go em vez de fazer N+1 por guilda.
- `internal/store/kefra_admin.go`: `SetKefraDefeated` sobe a `revision` no mesmo `UPDATE`. É isso que
  impede o save do processo em execução (revisão menor) de desfazer a edição — e é por isso que a
  tela diz que o efeito vale **no próximo boot**: o Kefra vivo é da goroutine do mundo.
- `webserver/internal/worldstateadmin`: `Get` monta o snapshot inteiro; `SetCityTax` valida 0..30
  (`CHECK` de `guild_zone.city_tax`) e regrava a **linha inteira** via `LoadGuildZones`/`SaveGuildZone`,
  não um `UPDATE` de uma coluna — a linha é um `STRUCT_GUILDZONE` e o desafio/cofre não podem derivar.
  `SetKefraDefeated` recusa com `NOT_FOUND` quando o tmServer nunca gravou um ciclo (os `CHECK` de
  `next_spawn_unix`/`last_spawn_unix` não têm valor inventável).
- Portal: `/guildas` (lista + diálogo com membros e relações, só leitura) e `/castelo` (taxa por zona com
  `salvar`, quest do castelo, card do Kefra com "marcar como morto/vivo"). Menu ganhou `/castelo`.
- Verificação: `go test ./webserver/internal/worldstateadmin/` (agrupamento de roster, taxa inválida/
  zona inexistente/linha inteira preservada, Kefra sem linha, `player` → 403) e `npx tsc --noEmit` verdes;
  `webserver` reconstruído e servindo com o serviço registrado.

## Feito — fatia 2: escritas de guilda com auditoria

- Migration `0037_guild_admin_audit`: `guild_admin_audit(moderator_id, guild_id, action, target,
  before, after, reason)`, mesmo formato de `character_admin_audit`/`item_effect_audit` — `before`/`after`
  em JSONB lidos da própria tabela dentro da transação, nunca do que o cliente mandou.
- `internal/store/guild_admin_writes.go`: três operações, cada uma em **uma** transação com a sua linha
  de auditoria.
  - `SetGuildMasterByCharacter`: trava a filiação alvo, rebaixa o nível 9 para 0, promove o alvo,
    espelha os dois em `character.guild_level` (é de lá que login/save leem) e grava o retrato de
    `guild_member` antes/depois. Transferência pela metade deixaria a guilda sem mestre utilizável.
  - `RemoveGuildMemberByCharacter`: remove a filiação e limpa `character.guild_id`/`guild_level`.
  - `SetGuildRelationAudited`: upsert (`kind` 1/2) ou remoção (`kind` 0) da relação direcionada.
- `worldstateadmin`: as validações ficam no serviço para o motivo da recusa ser nomeado — motivo
  obrigatório, alvo precisa ser membro, mestre atual não é promovido de novo nem removido, relação
  consigo mesma e `kind` fora de 0..2 recusados, guildas inexistentes → `NOT_FOUND`.
- `web.proto`/`grpcsrv`: `SetGuildMaster`, `RemoveGuildMember`, `SetGuildRelation` devolvem
  `SetWorldStateResponse{result, message}`; o portal mostra a mensagem do web-api.
- Portal: `/guildas` ganhou, no diálogo de cada guilda, campo **motivo** obrigatório, `tornar mestre` e
  `remover` por membro (os dois desabilitados no líder) e o formulário de relação (outra guilda, aliada/
  inimiga, aplicar ou remover).
- Verificação: `go test` verde em `worldstateadmin` (motivo vazio, não-membro, mestre atual, remoção do
  líder recusada sem tocar o banco, relação consigo mesma/guilda inexistente/kind inválido) e em
  `grpcsrv`; `npx tsc --noEmit` 0; migration `0037_guild_admin_audit` aplicada pelo `webserver`.
  Prova de SQL real pelo gRPC (guildas temporárias 9001/9002): `SetGuildRelation kind=2` → linha em
  `guild_relation` + auditoria com `before=[] after=[{...}]`; `kind=0` → relação removida e auditoria com
  `before=[...] after=[]`; `GetWorldState` refletiu a relação no mesmo instante. As guildas de teste
  foram apagadas; as duas linhas de auditoria do probe ficam como registro.
- **Não verificado**: `SetGuildMaster`/`RemoveGuildMember` contra o Postgres — `guild_member` exige
  `character` existente e o banco não tem personagens; criar um exige o cliente do jogo. Estão cobertos
  por teste de validação e `go build`, não por execução de SQL.

## Feito — fatia 3: dono de cidade e dono da torre

"Dono do castelo" neste servidor não é uma tabela: são duas colunas, e são elas que a web passou a
escrever, cada uma com auditoria transacional.

- `internal/store/guild_admin_owner.go`:
  - `SetZoneOwner(zone, guild_id)` → `guild_zone.charge_guild`; `guild_id = 0` limpa (cidade sem guilda no
    poder até o próximo cerco). Trava a linha (`FOR UPDATE`), é no-op quando o dono já é o pedido.
  - `SetTowerOwner(guild_id)` → `guild_tower_state.owner_guild`, atualizando `updated_at_unix`.
- **Dono e taxa são chamadas separadas**, embora vivam na mesma linha: um resultado de cerco que também
  movesse o imposto da loja seria uma mudança de economia silenciosa. `SetCityTax` continua sendo o único
  caminho para a taxa.
- `web.proto`/`grpcsrv`: `SetZoneOwner` e `SetTowerOwner` (`result` + `message`), validação no serviço:
  zona 0..4, guilda precisa existir (ou `0` = sem dono), motivo obrigatório.
- Portal `/castelo`: seletor de dono por cidade com `definir dono`, card **Torre (GTorre)** com seletor +
  `definir dono da torre`, e o campo **motivo** compartilhado no topo do painel.
- Verificação com guilda temporária (9001) pelo gRPC: `SetZoneOwner(0, 9001)` → `charge_guild=9001` e
  `charge_guild_name` no snapshot; `SetTowerOwner(9001)` → torre com dono; reverts para `0` deixaram
  `charge_guild=0`/`owner_guild=0`; `sem motivo` devolveu `INVALID` com a mensagem certa. A auditoria
  guardou os quatro pares before/after (`charge_guild 0 → 9001 → 0`, `owner_guild 0 → 9001 → 0`). A guilda
  de teste foi apagada; as linhas de auditoria do probe ficam como registro.
- Verificação de código: `go test` verde em `worldstateadmin` (zona inválida, sem motivo, guilda inexistente,
  limpar e definir dono, torre) e `grpcsrv`; `go build ./...` 0; `npx tsc --noEmit` 0.

Falta nesta issue: **janela de horários** da guerra/torre/RvR/BR — ela vem do `gameconfig.txt`
(`GTorreHour`, `RVRHour`, `BRHour`), então é a issue 36 (rates e flags) que a expõe e valida.

## Objetivo

Administrar o estado competitivo do servidor pela web: guildas e membros, dono/taxa do castelo e janela
de guerra, e o ciclo do Kefra (vivo/morto, horário). Hoje tudo isso só existe como tabela no Postgres
(`guild`, `guild_member`, `guild_relation`, `guild_tower_state`, `castle_quest_state`, `kefra_state`) e
como constante no código — sem nenhuma RPC e sem tela.

## Contexto / evidência

- Tabelas sem RPC: `guild`, `guild_member`, `guild_relation`, `guild_tower_state`, `castle_quest_state`,
  `kefra_state` (nenhuma aparece no `web.proto`).
- `CityTax` é `g_pGuildZone[village].CityTax` — 5% fixo no código (`Basedef.cpp:54-61`), com override de
  cerco em runtime; é o imposto que a loja pessoal paga (`world/city.go:60`).
- Horários vêm do `gameconfig.txt`: `GTorreHour 22`, `RVRHour 23`, `BRHour 19`, `kefrastatus 156`
  (chaves sem validação e sem UI).
- O portal do Deserto → Kefra só abre com `KefraDefeated` (`world/teleport.go`), estado que hoje não tem
  como ser consultado nem resetado pela web.
- Guildas em memória: mudanças de dono/membro podem divergir do mundo até o próximo boot — por isso o
  aviso de restart é obrigatório e a issue 38 (RPC no tmserver) é o caminho definitivo.

## Passos

1. `web.proto` + `webserver/internal/guildadmin`: `GuildAdminService` com `ListGuilds`,
   `GetGuild(id)` (membros, relações, torre), `CreateGuild(name, master)`, `SetGuildMaster`,
   `UpsertMember`, `RemoveMember`, `SetRelation`, `DeleteGuild`.
2. `CastleAdminService`: `GetCastleState`, `SetCastleOwner(guild_id)`, `SetCityTax(village, 0..20)`,
   `SetWarWindow(day, hour)`, `ResetSiege`.
3. `KefraAdminService`: `GetKefraState`, `SetKefraAlive(bool)`, `SetKefraSchedule`, `ResetKefra`.
4. UI `/guildas`: lista com membros/relações, criação, troca de mestre (com confirmação dupla: nome da
   guilda + motivo) e relações (aliada/inimiga). `/castelo`: dono atual, taxa da cidade com preview no
   preço de loja pessoal, janela de guerra e botão de reset com aviso de restart.
5. Bloco **Kefra** dentro de `/eventos` (ou `/castelo`): estado, horário e "marcar como morto" com
   auditoria — é o que destrava o portal do Deserto.
6. Toda escrita por `adminCall` com `target=guild:<id>` / `castle:<village>` / `kefra` e aviso fixo de
   aplicação no próximo restart (`docker compose restart tmserver`).

## Arquivos

- Criados: `webserver/internal/{guildadmin,castleadmin,kefraadmin}/*.go`,
  `portal/src/app/(admin)/guildas/{page.tsx,guild-table.tsx}`,
  `portal/src/app/(admin)/castelo/page.tsx`, `portal/src/server/{guilds,castle,kefra}.ts`,
  `portal/src/server/actions/world-state.ts`, `portal/src/lib/world-state.ts`.
- Alterados: `api/web/v1/web.proto`, `portal/src/components/{nav.ts,Icon.tsx}`,
  `portal/src/middleware.ts`.
- Proibido: `UPDATE guild`/`castle_*` direto sem RPC e sem auditoria; resetar Kefra com jogadores online
  sem aviso na tela.

## Testes / verificação

```bash
go test ./webserver/internal/guildadmin/... -v
cd portal && npx tsc --noEmit && npm run build
```

Evidência: criar a guilda `Teste` com mestre `Cantarelli` e vê-la no jogo após restart; mudar a taxa de
Armia 5 → 7 e ver o imposto na loja pessoal; marcar Kefra como morto e o portal do Deserto abrir.

## Critérios de aceite

- [x] Guildas, membros e relações editáveis com auditoria (`guild_admin_audit`, motivo obrigatório)
- [x] Taxa da cidade alterável (0..30) e dono da cidade/torre (`SetZoneOwner`/`SetTowerOwner`), com aviso de restart
- [ ] Janela de guerra (tôrre/RvR/battle royale) visível e editável — horários do `gameconfig.txt`, escopo da issue 36
- [x] Estado do Kefra consultável e resetável, destravando o portal
- [x] Conta `player` recebe 403; negação auditada (`adminCall` registra a tentativa)
- [x] Nada escrito em `Release/`

## Riscos / rollback

- **Risco:** divergência com o mundo em memória (guilda/taxa) até o restart. **Mitigação:** aviso
  explícito + `docker compose restart tmserver` no banner; a 38 traz a RPC ao vivo.
- **Risco:** deletar guilda com membros. **Mitigação:** confirmação dupla e `ON DELETE` explícito em vez
  de cascata silenciosa; auditar antes/depois.
- **Rollback:** cada entidade tem estado anterior na auditoria; restauração por linha.
