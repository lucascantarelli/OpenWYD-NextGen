# Roteiros de teste in-game — passo a passo para executar

Companheiro de [`qa-ingame.md`](qa-ingame.md). Cada roteiro tem: **o que preparar** (SQL/comando),
**o que fazer no cliente**, **o que capturar** e **quando considerar aprovado/reprovado**. Execute na
ordem — os primeiros destravam os outros.

## 0. Antes de começar (uma vez)

```bash
# stack de pé e saudável
docker compose ps
curl -s -o /dev/null -w "status:%{http_code}\n" http://127.0.0.1/      # serv00.htm (:80)
curl -s -o /dev/null -w "portal:%{http_code}\n" http://127.0.0.1:3000/login

# quem é quem no banco (guarde: você vai usar estes ids)
docker compose exec -T db psql -U postgres -d postgres -c \
  "select id, name, role, is_blocked from account order by id;"
docker compose exec -T db psql -U postgres -d postgres -c \
  "select id, account_id, name, level, coin from character order by id;"
```

Regras que valem para todos os roteiros:

- **Não reinicie serviço no meio de um teste.** Editar conteúdo no portal não aplica em runtime: o
  ciclo é *salvar → fechar o cliente → `docker compose up -d openwyd-nextgen1-tmsrv` com a flag → reabrir*.
- Toda flag de overlay é **de boot** (`-mob-stat-editing`, `-drop-config-editing`,
  `-mob-carry-editing`, `-rates-editing`, `-item-effect-editing`, `-teleport-editing`) e **não** está
  no `docker-compose.yaml` por padrão. O catálogo de NPCs **não** tem flag desde a issue 57:
  `npc_definition` é a única fonte de spawn e o tmServer a lê sempre.
- Flags ligadas por ambiente exigem `docker compose up -d openwyd-nextgen1-tmsrv` **com** a variável (o container
  é recriado). Comando padrão do resto do documento:

```bash
W2PP_MOB_STAT_EDITING=true W2PP_DROP_CONFIG_EDITING=true \
W2PP_MOB_CARRY_EDITING=true W2PP_RATES_EDITING=true W2PP_ITEM_EFFECT_EDITING=true \
docker compose up -d openwyd-nextgen1-tmsrv && docker compose logs --tail 30 openwyd-nextgen1-tmsrv
```

- Registre cada execução no fim deste arquivo (tabela §12), mesmo quando falhar.

---

## 1. Issues 05/06 — conexão e versão de protocolo

**Objetivo:** provar que o cliente desta máquina fala com o servidor e descobrir o `ClientVersion`
exato que ele envia.

**Preparar**

```bash
cd client && cp -n serverlist.bin serverlist.bin.bak   # backup único
# regerar o bin apontando para este host (formato: grupo 0, índice 0, porta 8281)
#   ferramenta: "serverlist bin/serverlist editor.exe"  →  <IP-da-LAN>:8281
#   copiar o resultado para client/serverlist.bin
powershell -NoProfile -Command "Test-NetConnection 127.0.0.1 -Port 8281 | Select TcpTestSucceeded"
```

**No cliente**

1. Abrir `WYD.exe` **direto** (nunca o launcher — ele reescreve o `serverlist.bin`).
2. Observar a tela de seleção de servidor: o canal local deve aparecer (sem "manutenção").
3. Tentar logar `test` / `test123` (se recusar por versão, o cliente mostra erro e o servidor loga).

**Capturar**

```bash
docker compose logs --since 5m tmserver | grep -E "served status|client connected|recv packet|rejected"
```

- Linha de rejeição esperada (se houver): `account login rejected … got=NNNNN want=12000`.
- **Resolvido em 04/11:** o cliente desta máquina envia **12000** (o servidor estava com
  `-client-version 12000` e o login passou). O `got=` só aparece em mismatch, então esta seção agora
  serve para **repetir em outra máquina**: se aparecer `version mismatch`, leia o `got=` e ajuste a flag.

**Aprovado se:** GET de status do IP do cliente + conexão CPSock + pacote `0x020d` no log, e a tela
lista o canal. **Reprovado:** status OK e sem CPSock → firewall/porta; CPSock e `0x102` → versão
(ajuste `-client-version` no `docker-compose.yaml` e reinicie; registre o número).

---

## 2. Issue 07 — login ponta a ponta

**Preparar:** rode primeiro o headless (não gasta o cliente):

```bash
MSYS_NO_PATHCONV=1 docker run --rm --network openwyd_default -v "D:/dolorestec/wyd2/OpenWYD:/src" \
  -e W2PP_E2E_ADDR=openwyd-nextgen1-tmsrv:8281 -e W2PP_E2E_ACCOUNT=test -e W2PP_E2E_PASSWORD=test123 \
  -e W2PP_E2E_VERSION=<valor da issue 06> golang:1.26-alpine \
  sh -c "cd /src && go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v"
```

**No cliente:** logar → tela de seleção (conta `test` pode estar sem personagem, isso é esperado) →
se houver personagem, entrar no mundo e andar 10 passos.

**Capturar:** `PASS` do headless, print da seleção/mundo, e `docker compose logs --since 10m tmserver |
grep -E "0x020d|0x0213"`. **Aprovado se:** headless PASS **e** cliente chega à seleção; se houver
personagem, entra no mapa e vê o HUD (HP/EXP atualizando — invariante do `ESCENE_FIELD`).

---

## 3. Issue 32 — conta bloqueada não entra

**Preparar** (troque `test` se for outra conta):

```bash
# bloquear
docker compose exec -T db psql -U postgres -d postgres -c \
  "update account set is_blocked=true, blocked_reason='teste QA 49' where name='test';"
```

**No cliente:** tentar logar. **Aprovado se:** recusa **com a mensagem na tela** e **sem** entrar no
mundo; o log do tmserver registra a recusa.

> **O que deve aparecer:** `Conta bloqueada Grade X, entre em contato com Suporte.` — texto do
> `Language.txt` (132), enviado como `_MSG_MessagePanel` com `HEADER.ID 0`, como o legado
> (`ProcessDBMessage.cpp:532-570`). Se a caixa aparecer **vazia**, o build do tmserver é anterior à
> correção de 04/11: `docker compose up -d --build tmserver`.

**Capturar:** print da recusa + linha de log + `select name,is_blocked,
blocked_reason from account where name='test';`

**Desfazer (obrigatório):**

```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "update account set is_blocked=false, blocked_reason='' where name='test';"
```

E confirmar que o login volta a funcionar. **Reprovado se:** bloqueio só vale depois de reiniciar o
tmserver, ou a mensagem é genérica ("senha inválida").

---

## 4. Issue 23 — NPC/loja criado no portal aparece no jogo

**Preparar** (seed é pré-requisito; rode uma vez):

```bash
docker compose run --rm openwyd-dbserver import-npcs -content /content \
  -dsn "postgres://postgres:<senha>@db:5432/postgres"
docker compose up -d openwyd-nextgen1-tmsrv
docker compose logs --tail 20 tmserver | grep -E "npc generators registered|npc config applied at boot"
```

No portal (`/npcs`): criar `slug=qa-ferreiro`, template de mercador existente, Armia `(100,120)`,
`merchant=1`, dois itens na loja (ex.: poção e uma arma barata) → salvar → **reiniciar o tmserver com a
flag** (o aviso da tela já diz isso).

**No cliente:** ir a Armia `(100,120)` (atalho `/armia`), abrir o NPC, ver os dois itens e comprar um.

**Capturar:** print da loja com os itens, print do inventário depois da compra, e:

```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select d.slug, d.enabled, s.slot, s.item_index from npc_definition d
     join npc_shop_item s on s.npc_id=d.id where d.slug='qa-ferreiro' order by s.slot;"
```

**Aprovado se:** NPC visível na posição, loja com os itens, compra debita ouro e entrega o item.
**Reprovado:** NPC não aparece → conferir `enabled`, `map_id`/coordenada e o log do overlay; aparece
sem itens → `SetNpcShop` não aplicou (reenviar o array completo).

**Limpeza:** ocultar (`enabled=false`) ou apagar o NPC de teste no portal.

---

## 5. Issue 24 — preço global refletido na loja

**Preparar:** no portal `/precos`, definir preço para um item que a loja acima vende (anote o valor
base do CSV mostrado na tela). Reiniciar o tmserver (preço é lido com o catálogo).

```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select item_index, price from item_price order by updated_at desc limit 5;"
```

**No cliente:** abrir a mesma loja e ler o preço do item; comprar 1. **Aprovado se:** o preço exibido é
o do override (não o do CSV) e o ouro debitado bate. Depois "remover override" no portal → reiniciar →
preço volta ao base. **Capturar:** print antes (base), depois (override) e o `select` acima.

---

## 6. Issue 25 — stats de mob aplicados (EXP)

**Preparar:** no portal `/mobs`, abrir um mob **comum de campo** que você consiga matar (anote o
`template_name` e o `exp` original). Alterar só `exp` para um valor evidente (ex.: ×10) → salvar →
reiniciar com `W2PP_MOB_STAT_EDITING=true`.

```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select template_name, exp, level, max_hp from mob_template_stat;"
docker compose logs --tail 30 openwyd-nextgen1-tmsrv | grep -iE "mob stat|overlay"
```

**No cliente:** anotar a EXP atual (barra/`/nick`), matar **um** mob do mesmo tipo antes e depois do
restart — ou usar dois mobs iguais e comparar o ganho. **Aprovado se:** o ganho de EXP corresponde ao
novo valor (com os modificadores de evento/campo conhecidos), e reverter o override volta ao normal.
**Capturar:** print do ganho de EXP antes/depois + o `select` + linha de log do overlay.

---

## 7. Issue 34 — drop: taxa por slot e `Carry[]`

**Preparar (parte A — frequência):** no portal `/drops/editar`, escolher um slot usado pelo mob alvo
(ex.: faixa "equipamento comum", slot 12) e mudar o divisor de 900 para 4 (muito mais comum) → salvar →
reiniciar com `W2PP_DROP_CONFIG_EDITING=true`; o log deve dizer
`drop rate overlay applied … slot_12=4`.

**Preparar (parte B — item garantido):** em `/drops/carry`, no template do mob, colocar um item
identificável (ex.: `412`) no slot **56** ("sempre cai") → salvar → reiniciar com
`W2PP_MOB_CARRY_EDITING=true`; log `mob carry overlay enabled`.

**No cliente:** matar **50** mobs do mesmo tipo com a parte A ligada e contar drops; depois ligar a
parte B e matar 5.

**Aprovado se:** parte A muda a frequência de forma perceptível (50 kills com divisor 4 deveriam dropar
quase sempre) e parte B entrega o item nas 5 mortes. **Capturar:** contagem por kill (planilha ou vídeo),
`select * from drop_rate_override;` e `select * from mob_carry_override;`.

**Cuidado:** a taxa é **divisor** (`rand()%rate`), não porcentagem; e a raridade é o **slot**, não o
item. **Limpeza:** `Reset` nas duas telas.

---

## 8. Issue 36 — rates de refino/combine

**Preparar:** no portal `/config`, aba Refino, pegar uma célula de PO (âmbar) e mudar o valor para
`100` (sucesso garantido) → salvar com motivo → reiniciar com `W2PP_RATES_EDITING=true`. O log deve
mostrar `rate_overrides_applied=1` **e** `content rate overlay effective sancrate_ori=[…100…]` — as duas
linhas, porque a segunda prova o valor que o mundo usa.

**No cliente:** com um item refinável e ouro (`/gm setgold 1000000`, `/gm item <id> 10`), refinar 10
vezes no NPC de refino. **Aprovado se:** 10/10 sucessos com a célula em 100 e a distribuição volta ao
normal depois do `Reset` + restart. **Capturar:** vídeo/print dos refinos + as duas linhas de log +
`select * from content_rate_override;`.

> Combine (`CompRate`) segue o mesmo formato, mas o item/par correto precisa ser escolhido no jogo —
> se não houver receita acessível, registre "não executado" em vez de marcar a issue.

---

## 9. Issue 44 — efeitos de item (parte testável hoje)

**Preparar:** no portal `/itens/<index>`, adicionar/alterar um efeito base de um item **equipável**
(ex.: `EF_AC` de uma armadura barata para +60) → salvar → reiniciar com `W2PP_ITEM_EFFECT_EDITING=true`;
log `item effect overlay applied rows=1 replaced=1`.

**No cliente:** `/gm item <index>` → equipar → abrir a ficha e ver a defesa subir 60.

**Aprovado se:** o valor na ficha muda pelo delta exato. **Não testável ainda:** `refine_rule` /
`combine_recipe` (escopo pendente da issue).

---

## 10. Issue 45 — guilda/castelo/Kefra

**Preparar:** no portal `/castelo`, mudar a taxa de uma zona (0..30) e marcar/desmarcar o Kefra; em
`/guildas`, conferir a lista. Reiniciar o tmserver.

```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select zone, charge_guild, city_tax from guild_zone order by zone;"
```

**No cliente:** ir à cidade da zona e conferir o imposto na loja (o preço de venda/compra reflete a
taxa); se o Kefra foi marcado como morto, `/kefra` deve levar ao Continente desconhecido.

**Aprovado se:** imposto e Kefra mudam conforme o portal após o boot. **Não testável ainda:** janela de
horário de cerco/guerra (escopo da 36) e pacotes S→C de guerra/castelo (UNVERIFIED) — registre como
pendência, não como falha.

---

## 11. Issue 48 — mapa do atlas contra o mundo

**Preparar:** abrir `/atlas` no portal e anotar uma coordenada de referência (ex.: spawn de Armia).

**No cliente:** `/armia`, andar até a coordenada anotada e comparar o que o atlas desenha (relevo, se é
caminhável, região) com o que você vê. **Aprovado se:** o ponto é caminhável no jogo e o atlas diz o
mesmo; uma parede/montanha no jogo aparece escurecida no atlas.

---

## 12. Registro das execuções

Preencha e cole no arquivo da issue correspondente (ou aqui, se quiser um histórico único):

| Issue | Data | Resultado | Evidência (log/print) | Observação |
|-------|------|-----------|------------------------|------------|
| 05 | 04/11 | ✅ | status + `0x020d` (issue-05) | bin de 02/11, sem `.bak` |
| 06 | 04/11 | ✅ | `account login: OK … chars=2` (issue-06) | client envia **12000** |
| 07 | 04/11 | ✅ | entrou no mundo, andou, loja e compra (`0x0379`); headless PASS | item comprado custou 0 |
| 32 | 04/11 | ✅ | “Conta bloqueada Grade X…” na tela; `0x0101` ID 0 / 140 B; conta liberada | exigiu corrigir o formato do aviso |
| 23 | | | | |
| 24 | | | | |
| 25 | | | | |
| 34 | | | | |
| 36 | | | | |
| 44 | | | | |
| 45 | | | | |
| 46 | — | **não testável** | — | overlay de skill ainda não implementado |
| 48 | | | | |

Quando um roteiro reprovar, a linha vai para `docs/migration/ingame-bugs.md` com: sintoma, o que foi
feito, log, suspeita e o que falta. A issue **não** fecha.
