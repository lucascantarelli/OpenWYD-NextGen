# Issue 34 — Drops editáveis: taxa por slot e `Carry[]` do mob

**Fase:** 6 · **Depende de:** 25, 26, 33 · **Bloqueia:** — · **Esforço:** L · **Estado:** 🟡 (taxa por slot e `Carry[]` ✅ ponta a ponta — migrações, overlays, RPCs e os dois editores no portal; falta só a evidência in-game de frequência)

## Objetivo

Editar pela web (a) a tabela de chance por slot de drop e (b) quais itens cada mob carrega em
`Carry[]`, com simulação de chance antes de salvar e aplicação no boot do tmserver.

## Contexto / evidência

- A chance por slot é **código**, não config: `DropRate[64]`/`DropBonus[64]` em
  `tmserver/internal/loot/loot.go`, com faixas fixas (0–7 = 900, 8–11 = 4, 16–23 = 20000,
  56 = 1 "sempre cai", 57–63 = 35…20000) e ajuste por nível do mob em `EffectiveDropRate`. **O valor
  é divisor**: chance ≈ `1/rate` por slot sorteado.
- Os itens que um mob pode dropar estão no `Carry[]` do template (`STRUCT_MOB`, 816 B,
  `Release/TMsrv/run/npc/<nome>`). O **slot** define a raridade, não o item — `game-rules.md §2`.
- `MobTemplateAdminService` edita stats e `Equip[]`, **não** `Carry[]`; `NpcAdminService.SetNpcShop`
  edita loja de mercador (`MSG_ShopList` 0..26), que não é drop de monstro.
- O número de slots de `Carry` deve ser confirmado no codec (`tmserver/internal/content/npc.go`)
  antes de codar; `DropRate[64]` e 816 B são consistentes com 64 slots — **confirmar, não assumir**.
- `ItemDropList.txt`/`MobDropList.txt` são artefatos gerados; não são entrada.

## Feito — taxa por slot (fatia 1)

- **Migração `0040_drop_config`** (o texto acima dizia 0036, mas a última aplicada era
  `0039_server_instance_flags`): `drop_rate_override(slot 0..63, value 1..1000000)` + `drop_rate_audit`
  com before/after lidos da tabela.
- **`internal/droprate`**: a tabela base `DropRate[64]` saiu de `loot.init()` para um pacote
  compartilhado, porque `webserver` **não pode** importar `tmserver/internal/loot` (regra de `internal`
  do Go) e o editor precisa mostrar o mesmo valor base. `loot.DropRate` continua existindo como cópia
  mutável — o overlay reescreve essa cópia, nunca a tabela compartilhada.
- **`internal/store/droprate.go`**: list/upsert/delete com auditoria na mesma transação.
- **RPCs**: `db.v1.DropConfigService.ListDropRateOverrides` (tmServer lê) e
  `web.v1.DropAdminService` (`ListDropRates`, `UpsertDropRate`, `ResetDropRate`) + `dropadmin.Service`
  (autoriza por `account.role`, valida slot 0..63 e valor 1..1.000.000) + `grpcsrv`.
- **tmServer**: `-drop-config-editing` / `W2PP_DROP_CONFIG_EDITING` (OFF por padrão), aplicado uma vez
  no boot por `tmserver/internal/dropcfg`, com log
  `drop rate overlay applied` (`drop_overrides_applied`, `ignored`, `rows`, `slot_12`).
  Nenhuma chamada de `rand()` foi adicionada — só valores, então a paridade de RNG continua.
- **Confirmado no codec**: `MaxCarry = 64` (`internal/savefmt/savefmt.go`), `Carry` em `off=268` — os
  64 slots do `DropRate` são os mesmos do `Carry[]` (o "confirmar, não assumir" do contexto).

## Evidência ao vivo (fatia 1)

- `ListDropRates(1)` → `result=1`, **64 slots**, 0 overrides.
- `UpsertDropRate(slot 12, 450)` → `1`; listagem passa a `base=900 current=450 overridden=true`.
- `slot 64` → `3` (INVALID), `value=0` → `3`, conta `player` → `2` (FORBIDDEN);
  `ResetDropRate` → `1` e, repetido, `4` (NOT_FOUND).
- tmserver one-off na rede do compose com `-drop-config-editing`:
  `drop rate overlay applied drop_overrides_applied=1 drop_overrides_ignored=0 rows=1 slot_12=450`.
- `drop_rate_audit`: 2 upserts + 2 resets (o teste limpou o slot 12 no fim).
- Testes: `go test ./tmserver/internal/dropcfg/ ./tmserver/internal/loot/ ./webserver/internal/dropadmin/ ./webserver/internal/grpcsrv/`
  verdes; `go build ./...` e `go vet` limpos.

## Feito — portal `/drops/editar` (fatia 2)

- `lib/droprate.ts`: espelho de `loot.EffectiveDropRate` (divisões inteiras, cortes de nível, slots
  8–11 fixos, teto 32000), `oddsLabel` ("≈ 1 em N", e `rate <= 0` = "sempre cai", como `Drops`),
  `SLOT_GROUPS` e `hardcodedRate`. A tabela base entra **por parâmetro**, não por estado de módulo:
  server component e client component são instâncias diferentes do módulo.
- `components/drop/drop-rate-editor.tsx`: simulador embutido na tabela (nível do mob e bônus do
  matador recalculam a coluna de chance ao vivo), divisor editável por faixa, badge de estado
  (arquivo/ajustado/fixo no código), salvar com motivo auditado e restaurar por posição.
- `DropsNav`: duas rotas reais (`/drops` e `/drops/editar`) com `aria-current`, não tab client.
- **Sem ID cru como rótulo**: a linha é "Equipamento comum", "Ouro e poção", "Ultra raro",
  "Sempre cai"…; o intervalo do `Carry[]` aparece como posição secundária. Slots 8–11 ficam
  desabilitados e explicados (o runtime reescreve o resultado deles).
- `server/drops.ts` restaurado com as buscas do relatório (cache por processo, teto de 64 filtros)
  junto de `listDropRates`.

## Feito — `Carry[]` por template (fatia 3)

- **Migração `0041_mob_carry`**: `mob_carry_override` (PK `template_name, slot`, item + 3 pares de
  efeito, CHECK de slot 0..63 e índice 1..32767) e `mob_carry_audit` com o array completo
  antes/depois. Semântica autoritativa: salvar substitui a tabela do template; posição ausente é
  posição vazia; `Reset` apaga as linhas do template.
- **`internal/store/mobcarry.go`**: list (snapshot para o tmserver), get, set com auditoria na mesma
  transação e delete com `applied bool`. Diferente do `Equip[]`, **não** exige override de stats: o
  carry é uma tabela independente.
- **RPCs**: `db.v1.DropConfigService.ListMobCarryOverrides` (leitura do tmserver) e
  `web.v1.DropAdminService.ListMobCarry/SetMobCarry/ResetMobCarry`.
- **`dropadmin.Service`**: lê o arquivo com `DecodeMobAny`, sobrepõe as linhas e resolve o nome do
  item pelo catálogo. Do browser vem **só o índice**: o efeito do `STRUCT_ITEM` é preservado quando
  o item não muda e zerado quando muda — a tela não edita efeitos, então não há por que confiar
  neles. Template inexistente ou item fora do catálogo → `INVALID`, nunca um override órfão.
- **`tmserver/internal/mobcarry`** + flag `-mob-carry-editing` (`W2PP_MOB_CARRY_EDITING`, OFF):
  aplica no template em `spawnNPCs` e no loader de NPCs do banco, no boot, com log
  `mob carry overlay enabled`. Nenhuma chamada de `rand()` foi adicionada.
- **Portal `/drops/carry`** (terceira aba de `DropsNav`): template escolhido na URL, faixas de
  raridade nomeadas (as mesmas da taxa), só posições ocupadas viram linha, “adicionar” ocupa a
  próxima livre da faixa, trocar/limpar por posição, motivo auditado e restaurar com confirmação.

## Evidência ao vivo (fatia 3)

- `ListMobCarry(@@Gargula)` → `OK`, nível 399, `overridden=false`, 0 posições (a tabela do arquivo
  está vazia nesse template).
- `SetMobCarry(slot 56, item 412)` → `OK`; a listagem passa a `overridden=true` com
  `item_name=Poeira de Oriharucon` e `from_override=true`.
- Rejeições: item 99999 → `INVALID`; template inexistente → `INVALID`; slot duplicado → `INVALID`;
  conta `player` → `FORBIDDEN`.
- `ResetMobCarry` → `OK` e, repetido, `NOT_FOUND`; `mob_carry_override` volta a 0 linhas.
- `mob_carry_audit`: um `set` e um `reset` em `@@Gargula`, com o motivo do teste.
- Boot one-off com `-mob-carry-editing` (rede do compose, `-content /Release`):
  `mob carry overlay enabled (moderator editing) templates=1` seguido de `tmserver listening`.
- `/drops/carry?template=@@Gargula` → **200** com o item, “posição 56”, “Sempre cai” e o badge
  “tabela do banco (override)”; `next build` 0 com a rota em 5,57 kB.

## Pendente

- Evidência in-game de frequência de drop e de item em slot 56.

## Passos

1. **Migração** `0036_drop_config`:
   ```sql
   CREATE TABLE IF NOT EXISTS drop_rate_override (
     slot integer PRIMARY KEY CHECK (slot BETWEEN 0 AND 63),
     value integer NOT NULL CHECK (value BETWEEN 1 AND 1000000),
     updated_by bigint NOT NULL REFERENCES account(id),
     updated_at timestamptz NOT NULL DEFAULT now()
   );
   CREATE TABLE IF NOT EXISTS mob_carry_override (
     template_name text NOT NULL,
     slot integer NOT NULL CHECK (slot BETWEEN 0 AND 63),
     item_index integer NOT NULL,
     eff integer[] ,        -- 3 pares (eff, effv), 0 = nenhum
     updated_by bigint NOT NULL REFERENCES account(id),
     updated_at timestamptz NOT NULL DEFAULT now(),
     PRIMARY KEY (template_name, slot)
   );
   ```
   (`.down.sql` simétrico; nome do array de efeitos alinhado ao codec de `STRUCT_ITEM`.)
2. **tmserver**: `loot.DropRate`/`DropBonus` passam a ser **sobrescrevíveis no boot** por
   `tmserver/internal/dropcfg` (Source via dbServer), atrás de `-drop-config-editing` /
   `W2PP_DROP_CONFIG_EDITING` (OFF = valores atuais byte a byte). `mob_carry_override` é aplicado no
   carregamento do template, junto de `-mob-stat-editing` (mesmo pacote ou pacote irmão, decisão
   registrada na issue).
3. **Preservar paridade de RNG**: nenhuma chamada extra a `rand()` pode ser introduzida — só os
   valores mudam (invariante 3 do `AGENTS.md`). Teste de paridade do `loot` continua obrigatório.
4. **Proto**: `DropAdminService` com `ListDropRates`, `UpsertDropRate(slot, value)`,
   `ResetDropRate(slot)`, `ListMobCarry(template_name)`, `SetMobCarry(template_name, slots[])`,
   `ResetMobCarry(template_name)`; `moderator_id` + `AdminResult`.
5. **Portal** `/drops/editar` (nova aba em `/drops`):
   - tabela dos 64 slots com valor base (do código, embutido como constante no portal com o
     comentário "espelho de loot.go"), valor atual, override e "restaurar";
   - por template: editor dos slots de `Carry` com Combobox de item, mesma UX da loja (issue 23),
     enviando o array completo;
   - **simulador**: para um slot e um nível de mob, calcular `EffectiveDropRate` em TS espelhando
     `loot.go` e mostrar "≈ 1 em N" com a ressalva de que o sorteio é `rand()%rate`.
6. Aviso de restart em toda escrita + link para `/operacao`; badge "drop alterado" no dashboard.
7. Testes Go: aplicar overlay muda só os valores esperados; com flag OFF o resultado é idêntico ao
   atual (golden test de drop inalterado); `SetMobCarry` em template inexistente → `INVALID`.

## Arquivos

- Criados: `internal/migrations/0036_drop_config.{up,down}.sql`,
  `tmserver/internal/dropcfg/{dropcfg.go,dropcfg_test.go}`,
  `webserver/internal/dropadmin/{service.go,service_test.go}`,
  `webserver/internal/grpcsrv/dropadmin.go`,
  `portal/src/app/(admin)/drops/editar/page.tsx`,
  `portal/src/server/actions/dropadmin.ts`, `portal/src/lib/droprate.ts` (espelho de `loot.go`).
- Alterados: `api/web/v1/web.proto`, `webserver/cmd/webserver/main.go`,
  `tmserver/cmd/tmserver/main.go`, `tmserver/internal/world/*` (aplicação do carry),
  `docker-compose.yaml`, `portal/src/components/nav.ts`,
  `docs/migration/game-rules.md` (nota de que a taxa virou config).
- Proibido: alterar a **ordem** das chamadas de `rand()`; editar `ItemDropList.txt`; mexer em
  `DropBonus` sem decisão registrada (bônus do matador, matriz 8×8 do legado).

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
MSYS_NO_PATHCONV=1 docker run --rm -v "D:/dolorestec/wyd2/OpenWYD:/src" -w /src golang:1.26-alpine \
  sh -c "go build ./... && go test -race ./tmserver/internal/loot/ ./tmserver/internal/dropcfg/"
```

```bash
# (comandos rodam na raiz do repositório)
W2PP_DROP_CONFIG_EDITING=true docker compose up -d tmserver
docker compose logs --tail 20 tmserver | grep -i "drop"
```

Evidência: baixar `drop_rate_override` do slot 8 de 4 para 1.000 (mais raro), matar 50 mobs no jogo e
comparar a frequência de drop antes/depois; restaurar e conferir o retorno. Para `Carry`: colocar um
item específico num slot 56 ("sempre") de um mob de teste e vê-lo cair em 5 mortes seguidas.

## Critérios de aceite

- [x] Flag OFF reproduz exatamente o comportamento atual (`TestApplyChangesOnlyTheEditedSlot`: overlay vazio
  não muda a tabela; `TestEffectiveRateFollowsTheOverride` cobre a fórmula)
- [x] Override aplicado no boot, com log de quantos slots foram sobrescritos
- [x] Simulador do portal bate com `EffectiveDropRate` do Go (5 casos em `lib/droprate.test.ts`:
  cortes de nível, bônus do matador, slots fixos e teto de 32000)
- [x] Editor dos 64 slots com base/atual/override e restaurar por posição
- [x] `SetMobCarry` valida item e template; `Reset` volta ao arquivo (testes
  `TestSetCarryPreservesEffectsOnlyForTheSameItem`, `TestSetCarryRejectsUnknownItemTemplateAndDuplicateSlot`,
  `TestResetCarryReturnsNotFoundWithoutOverride`)
- [x] Nenhuma mudança na ordem de consumo do RNG (o pacote só escreve números; `loot` segue verde)
- [ ] Evidência in-game de mudança de frequência e de item "sempre cai"

## Riscos / rollback

- **Risco:** economia explodir com taxa digitada errada. **Mitigação:** simulador obrigatório,
  faixa validada, auditoria com valor antigo/novo, `Reset` por slot e por template.
- **Risco:** `Carry[]` com tamanho errado corromper o template em memória. **Mitigação:** confirmar o
  layout no codec antes; override é aplicado em memória sobre os 816 B, nunca no arquivo; teste que
  valida o tamanho final.
- **Rollback:** flag OFF (volta ao código puro) ou `Reset*` em lote; `.down.sql` remove as tabelas.
