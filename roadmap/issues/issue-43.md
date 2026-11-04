# Issue 43 — Inventário e status de personagem

**Fase:** 6 · **Depende de:** 32, 38 · **Bloqueia:** — · **Esforço:** L · **Estado:** 🟡 parcial — leitura + concessão de item (2026-11-03)

## Objetivo

Ver e ajustar personagens pelo portal: nível, EXP, atributos, skills, gold, posição/cidade,
inventário (16 equip slots + cargo) e itens com adds — com a regra dura do projeto: **a web nunca
escreve estado de personagem vivo**. Concessão e ajuste passam por `delivery_queue`, drenada pelo
tmserver dentro da goroutine do mundo.

## Contexto / evidência

- `character` tem 40+ colunas (`level`, `exp`, `coin`, `str/int/dex/con`, `max_hp/hp`, `learned_skill`,
  `skill_bar`, `save_x/save_y`, `last_city`, `special`, `celestial_*`); `item` guarda `STRUCT_ITEM`
  (8 B: `sIndex`, `stEffect[3]`, `sValue[3]`).
- Invariante 7 do `AGENTS.md`: grants vão por `delivery_queue`; escrita direta no banco enquanto o
  personagem está online é perdida no próximo save (o `STRUCT_MOB` de 816 B vive na memória do mundo).
- Não existe RPC de personagem: `CharacterWebService.ListMyCharacters` é do próprio jogador.
- Base de personagem novo: `Release/DBsrv/run/BaseMob/{TK,FM,BM,HT}` (816 B por classe).

## Passos

1. `web.proto` + `webserver/internal/characteradmin`: `CharacterAdminService` com
   `SearchCharacters(query)` (nome/conta), `GetCharacter(id)`, `GetInventory(id)`,
   `GrantItem(id, item_index, quantity, eff[])`, `GrantCoin(id, amount)`,
   `SetField(id, field, value)` (allowlist de campos com validação de faixa), `SetPosition(id, x, y)`,
   `ResetSkills(id)`. Tudo com `moderator_id` + auditoria.
2. Regra de online: `GetCharacter` devolve `online` (via presença do tmserver). Mutação com `online`
   → `ADMIN_RESULT_INVALID` com mensagem "personagem online: use concessão". `GrantItem`/`GrantCoin`
   **funcionam online** (fila), o resto exige offline.
3. `webserver/internal/characteradmin/delivery.go`: escreve em `delivery_queue` com payload do item
   (index, quantidade, 3 pares eff/effv) — o tmserver já drena essa fila.
4. UI `/personagens`: busca, lista (nome, conta, nível, classe, online) e detalhe com abas
   **Status** (campos com diff antes de salvar), **Inventário** (16 slots + cargo com ícone do item e
   adds), **Skills** (aprendidas + barra) e **Histórico** (últimas ações admin, de `npc_audit`-style).
5. Concessão: diálogo com Combobox de item (reaproveita `/api/search/items`), quantidade 1–255 e 3
   pares de efeito opcionais; aviso "entra na fila e o personagem recebe no próximo tick".
6. Comparação antes/depois obrigatória em `SetField` (mostra valor atual → novo, com motivo digitado) e
   registro de auditoria com os dois valores.

## Arquivos

- Criados: `webserver/internal/characteradmin/*.go`, `api/web/v1/web.proto` (serviço),
  `portal/src/app/(admin)/personagens/{page.tsx,[id]/page.tsx}`,
  `portal/src/components/character/{StatusEditor,InventoryTable,GrantDialog}.tsx`,
  `portal/src/server/characters.ts`, `portal/src/server/actions/characters.ts`,
  `portal/src/lib/character.ts`.
- Alterados: `portal/src/components/{nav.ts,Icon.tsx}`, `portal/src/middleware.ts`.
- Proibido: `UPDATE character` direto pelo portal; gravar `STRUCT_MOB` fora do dbserver.

## Testes / verificação

```bash
go test ./webserver/internal/characteradmin/... -v
cd portal && npx tsc --noEmit && npm run build
```

Evidência: abrir o personagem `Cantarelli`, ver nível/inventário reais; conceder item 1234 ×2 com
`EF_CRITICAL 10` → aparece no próximo login; tentar `SetField` com o personagem online → 409 com
mensagem; editar nível offline → persiste e a auditoria mostra `12 → 20`.

## Critérios de aceite

- [ ] Inventário e status reais visíveis, com ícone e adds por item
- [ ] Concessão de item/gold funciona online (fila) e chega ao jogo
- [ ] Edição de campo bloqueada com 409 quando o personagem está online
- [ ] Diff antes/depois + motivo em toda edição, gravado na auditoria
- [ ] Faixas validadas (nível 1..400, atributos, coordenadas dentro do mundo)
- [ ] Conta `player` recebe 403 e a tentativa fica auditada

## Riscos / rollback

- **Risco:** corromper save (alinhamento MSVC, `STRUCT_MOB` = 816 B). **Mitigação:** só ler/gravar via
  dbserver, campo a campo, com backup do save antes (`pg_dump` da linha + cópia do blob).
- **Risco:** duplicação de item por corrida entre web e save do mundo. **Mitigação:** concessão
  exclusivamente por `delivery_queue` e edição só com personagem offline.
- **Rollback:** cada mutação é um campo; reverter aplicando o valor anterior registrado na auditoria.

## Entregue nesta rodada (2026-11-03) — leitura

Fatiado de propósito: **só leitura**, porque a metade de escrita depende de decisões que o texto da
issue assume resolvidas e que não estão (ver “Bloqueios encontrados” abaixo).

### O que funciona

```text
$ go build ./...                                    # ok
$ go test ./webserver/internal/characteradmin/ -v
--- PASS: TestSearchRequiresStaffAndRealQuery       (player → Forbidden; query < 2 letras → Invalid)
--- PASS: TestGetReturnsDetailOrNotFound

$ npx tsc --noEmit   # 0
$ npm run build      # 0 · ƒ /personagens · ƒ /personagens/[id]

$ curl /personagens?q=canta     200 · 43.899 bytes · Cantarelli (TK, conta test)
$ curl /personagens/1           200 · 148.051 bytes · Status, Skills, Equipamento, Inventário
$ curl /personagens (sem sessão) 307 → /login
$ curl /teleportes              200 (sem regressão)
```

- **Busca** (`SearchCharacters`): nome do personagem ou da conta (`ILIKE`), mínimo 2 letras, teto de
  50 resultados, ordenada por nível. Gate de staff igual ao resto (`moderator`/`admin`).
- **Detalhe** (`GetCharacter`): status (HP/MP, gold, EXP, atributos, cidade, save point), skills
  (bitmask `learned_skill` + barra + pontos de maestria) e inventário com **os 3 pares de efeito** e
  expiração. Cada item linka o catálogo (`/itens/<index>`), que resolve nome e ícone.
- Arquivos: `internal/store/character_admin.go`, `webserver/internal/characteradmin/{service.go,service_test.go}`,
  `webserver/internal/grpcsrv/characteradmin.go`, `api/web/v1/web.proto` (`CharacterAdminService`),
  `portal/src/lib/character.ts`, `portal/src/server/characters.ts`,
  `portal/src/app/(admin)/personagens/{page.tsx,[id]/page.tsx}`; nav e middleware liberados.

### Bloqueios encontrados (o que trava a escrita)

1. **`delivery_queue` hoje só carrega item.** O schema aceita `kind='item'|'cash'|'coin'`, mas o dreno
   do mundo (`world.ApplyDeliveries`) recebe `Delivery{ID, Item}` — não existe caminho para crédito de
   gold/cash. `GrantCoin` exige novo kind no dreno, query do dbserver por kind e aplicação dentro da
   goroutine do mundo (`world.go`, `persistence.go`). É mudança no save path, não na web.
2. **Não existe presença online.** Nada no `webserver` sabe se o personagem está online; o tmserver não
   publica sessões. O critério “409 se online” precisa de um registro de presença (tabela com heartbeat
   ou RPC de sessões) antes de existir.
3. **Edição de campo exige backup e auditoria.** O risco declarado (corromper o save) pede `pg_dump`/
   cópia do blob e uma tabela `character_admin_audit` (before/after + motivo) — migration nova.

### Pendente (checklist)

- [ ] Concessão de item por `delivery_queue` (payload pronto; falta o comando de UI + auditoria)
- [ ] Concessão de gold/cash (exige o item 1 acima)
- [ ] `SetField`/posição/skills com allowlist + diff + motivo (exige presença online)
- [ ] Aba Histórico e auditoria `character_admin_audit`
- [ ] Ícone do item dentro da tabela (hoje é link para `/itens/<index>`)

## Concessão por fila + auditoria (2026-11-03)

Segunda fatia: a escrita que **já tinha caminho** no mundo. `GrantItem` não toca no personagem —
enfileira em `delivery_queue` (kind `item`), que o tmserver drena no login dentro do loop do mundo.

```text
$ go build ./...   # 0
$ go test ./webserver/internal/characteradmin/ -v
--- PASS: TestGrantValidationAndQueue   (staff, quantidade 1..255, motivo obrigatório, NotFound)
--- PASS: TestListGrantsRequiresStaff

# verificação real contra o Postgres do compose (programa temporário, removido depois)
$ ./tmpcheck
enqueued ids: [1 2] err: <nil>
grants: 2 err: <nil>
  id=1 item=1234 status=pending source=admin:1 at=2026-11-03T12:5x:xx+00
  id=2 item=1234 status=pending source=admin:1 at=2026-11-03T12:5x:xx+00
audit reason: teste da issue 43
after cleanup: pending=0 audit=0
```

- **Migration 0035** `character_admin_audit` (moderator, character, action, before/after JSONB, motivo,
  índices por personagem e moderador).
- **Store** `EnqueueItemGrant` (uma linha por unidade — o payload da fila não tem quantidade, e usar
  `EF_AMOUNT` sobrescreveria um efeito escolhido) + `PendingItemGrants` (histórico), numa transação
  que grava a auditoria junto.
- **RPC** `GrantItem` e `ListCharacterGrants` no `CharacterAdminService`; validação de faixa e motivo
  com 3+ letras no serviço, e o `character_id` agora vem no detalhe (bug encontrado no teste de UI:
  `domain.Character` não tem id, então o resumo devolvia `id=0` e o formulário falhava na validação).
- **UI** diálogo `conceder item` no detalhe: índice do item, quantidade 1–255, 3 pares de efeito
  opcionais e motivo obrigatório, com o aviso de que entra na fila e chega no próximo dreno.

### Limitações desta fatia

- O dreno entrega no **cargo da conta**, não no personagem: `delivery_queue.character_id` é gravado
  para histórico, mas `PendingItemDeliveries` filtra por conta. Um item concedido pode cair no
  primeiro personagem que logar. Mudar isso é alterar o dreno do mundo.
- O clique ponta-a-ponta no navegador não foi concluído nesta sessão (a automação de UI não
  encontrou o botão após navegar); a evidência é a execução real do store contra o Postgres acima
  mais os testes de serviço.
- Continua pendente: gold/cash (exige kind novo no dreno) e `SetField`/posição/skills (exige
  presença online).
