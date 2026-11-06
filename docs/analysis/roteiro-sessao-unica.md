# Roteiro de sessão única — fechar o máximo de issues de uma vez

Companheiro de [`qa-ingame.md`](qa-ingame.md) e [`roteiros-qa.md`](roteiros-qa.md). Aqueles documentos
têm **um roteiro por issue** (bom para depurar um caso). Este troca isso por **uma sessão só**: os
dados são preparados antes, o `tmserver` sobe **uma vez** com todas as flags, e o cliente entra
**uma vez** (mais uma segunda janela só para trade/visão).

## Por que cabe numa sessão

As flags de overlay são **de boot** e independentes entre si — ligar seis de uma vez não muda o
comportamento de nenhuma (cada uma só reescreve a sua tabela). O que difere entre as issues é apenas
**qual dado foi editado no portal antes**; a sessão de cliente é a mesma. São dois restarts no total
(um para ligar as flags, um no fim para provar persistência), sempre com o cliente fechado.

| Issue | O que esta sessão prova | Custo extra |
|---|---|---|
| 24 | preço do portal aparece na loja e o ouro debitado bate | editar `/precos` |
| 25 | `exp` do template aplicado no kill | editar `/mobs` |
| 34 | frequência por slot e `Carry[56]` (sempre cai) | editar `/drops/editar` + `/drops/carry` |
| 36 | célula de refino em 100 → 10/10 sucessos | editar `/config` |
| 44 | efeito base do item muda a ficha | editar `/itens/<id>` |
| 45 | imposto da cidade e Kefra vindos do portal | editar `/castelo` |
| 48 | coordenada do atlas é caminhável no jogo | anotar em `/atlas` |
| 28 | EXP dobrada + notice visíveis | ligar em `/eventos` (recarrega em ~15 s, sem restart) |
| 30 | item da recompensa diária no inventário após login | oferta habilitada em `/diaria` |
| 08 | 4 classes criadas com BaseMob e 1.000.000 de gold | criar 4 personagens |
| 10 | andar, parede, 5 cidades, teleporte por tile, `last_city` | andar |
| 11 | mob morre/desaparece, loot, EXP, level-up, `/restart` com HP=2 | matar mobs |
| 12 | compra/venda, preço 0, recusas sem debitar, cargo, trade | 2ª janela de cliente |
| 09 | inventário/equip/gold/exp/cidade sobrevivem a relogin + restart | restart final |
| 13 | refino (parcial), combine e persistência | refinar/combinar |

Não fecha aqui: 33 (overlay de `ItemList`), 35 (curva de EXP), 37 (spawns), 38 (live ops), 43
(gold/cash e `SetField`), 46 (overlay de skill) — falta código, não evidência.

## 0. Preparação (com o cliente fechado)

### 0.1 Subir o tmserver com as seis flags

```bash
W2PP_MOB_STAT_EDITING=true W2PP_DROP_CONFIG_EDITING=true W2PP_MOB_CARRY_EDITING=true \
W2PP_RATES_EDITING=true W2PP_ITEM_EFFECT_EDITING=true W2PP_TELEPORT_EDITING=true \
docker compose up -d openwyd-nextgen1-tmsrv && docker compose logs --tail 40 openwyd-nextgen1-tmsrv
```

Confira no log, antes de abrir o cliente (se faltar uma linha, aquele passo fica sem prova):

```
mob stat overlay enabled          (ou o texto do overlay de stats)
drop rate overlay applied … applied=1
mob carry overlay enabled
rate overrides applied … rate_overrides_applied=1
content rate overlay effective … sancrate_ori=[…
item effect overlay applied rows=1
teleport config overlay enabled
```

### 0.2 Preparar os dados no portal

Faça tudo nesta ordem e **anote os valores** (o roteiro cobra o número exato no cliente).

| Tela | Ação | Vale para |
|---|---|---|
| `/npcs` | criar mercador `qa-ferreiro`, template de mercador existente, Armia `(100,120)`, dois itens na loja: **907** (Espada Bastarda) e **5134** | 24, 12 |
| `/precos` | preço do item **907** = `12345` (anote o base que a tela mostra) | 24 |
| `/mobs` | escolher um template **de campo, nível baixo**, que você reconheça no jogo; `exp` original anotado → mudar só `exp` para **×10** | 25 |
| `/drops/editar` | slot **12** (equipamento comum) divisor `900` → **4** | 34 |
| `/drops/carry` | no template do mob escolhido, slot **56** = item **412** (`out[56] = 1` = sempre cai) | 34 |
| `/config` | aba Refino: uma célula de PO (âmbar) = **100** | 36, 13 |
| `/itens/<armadura barata>` | `EF_AC` = **+60** | 44 |
| `/castelo` | taxa da zona de Armia = **5**, marcar **Kefra morto** | 45 |
| `/eventos` | item `400`, divisor `5`, faixa `1..50`, **EXP dobrada** ligada | 28 |
| `/diaria` | oferta habilitada: item `400`, 7 dias | 30 |
| `/atlas` | anotar uma coordenada de referência em Armia (canto de parede e um ponto aberto) | 48 |

### 0.3 Conferir que entrou (antes de abrir o cliente)

```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select item_index, price from item_price where item_index=907;"
docker compose exec -T db psql -U postgres -d postgres -c \
  "select * from drop_rate_override;"
docker compose exec -T db psql -U postgres -d postgres -c \
  "select * from mob_carry_override;"
docker compose exec -T db psql -U postgres -d postgres -c \
  "select * from content_rate_override;"
docker compose exec -T db psql -U postgres -d postgres -c \
  "select template_name, exp, level, max_hp from mob_template_stat;"
docker compose exec -T db psql -U postgres -d postgres -c \
  "select zone, charge_guild, city_tax from guild_zone order by zone limit 5;"
```

## 1. Sessão — ordem dos passos

Cada passo diz o que fazer, o que capturar e **quais critérios de qual issue** ele satisfaz. Se um
passo falhar, registre em `docs/migration/ingame-bugs.md` e siga — não feche a issue.

### Passo A — login e chegada (10, 30, 07 regressão)

1. Abrir `WYD.exe` **direto** (nunca o launcher) e logar `test`/`test123`.
2. Antes de selecionar o personagem, **abrir o inventário dele** ao entrar: o item da recompensa
   diária (400) deve estar lá — é a entrega de login da issue 30.
3. `docker compose logs --since 5m tmserver | grep -E "0x020d|0x0213|daily"`

**Captura:** print do inventário com o item; linhas de login.

### Passo B — criação dos 4 personagens (08)

Criar **TK, FM, BM e HT** numa conta com vagas (ou criar a conta em `/contas`). Para cada um:
aparência correta na seleção, equip inicial (corpo + montaria) e **1.000.000** de gold.

```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select c.name, c.level, c.coin, c.last_city from character c
     join account a on a.id=c.account_id where a.name='<conta>' order by c.id;"
```

**Captura:** print da seleção com as quatro classes + o `select`.

### Passo C — mundo, loja e movimento (10, 24, 12)

Com um personagem dentro do mundo:

1. Andar **20+ tiles** (sem travar, sem voltar no tempo) e tentar atravessar uma parede/água.
2. `/armia` → abrir `qa-ferreiro` → o item **907** deve custar **12345** (o do portal, não o base).
   Comprar 1 e conferir o gold debitado; vender de volta.
3. Comprar com gold insuficiente e com inventário cheio (encher com `/gm item`): recusa **sem**
   debitar.
4. `/erion`, `/azran`, `/gelo`, `/noatun`: spawn correto em cada uma.
5. Teleporte por tile cobrando o valor esperado (rota do `/teleportes`).

**Captura:** print da loja com o preço, print do ouro antes/depois, e:

```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select name, coin from character where name='<personagem>';"
```

### Passo D — combate, EXP e drop (11, 25, 34)

1. Matar **5** mobs do template escolhido e conferir: o mob desaparece para você (sem fantasma),
   loot/gold caem e são coletáveis, a barra de EXP anda.
2. Comparar o ganho de EXP com `exp × 10` do portal (com o evento de EXP dobrada desligado, para não
   misturar) → **25**.
3. Ligar a EXP dobrada em `/eventos` (recarrega em ~15 s, sem restart) e matar outro mob: ganho dobra
   e o **notice** aparece no cliente → **28**.
4. **34 parte A:** com divisor `4` no slot 12, matar **50** mobs e contar os drops de equipamento
   comum (deve cair quase sempre; divisor é `rand()%rate`, não porcentagem).
5. **34 parte B:** com o slot **56** = item 412, matar **5** mobs → o item cai nas 5.
6. Morrer de propósito → `/restart` (ou o comando de renascimento): volta com **HP=2** e recall.

**Captura:** contagem por kill (planilha ou vídeo), print da barra de EXP antes/depois, print do
notice; `select * from drop_rate_override;` e `select * from mob_carry_override;`.

### Passo E — refino e combine (36, 13)

```bash
# no cliente (GM): ouro e material
/gm setgold 1000000
/gm item <id da poeira de refino> 20
```

Refinar **10 vezes** no NPC de refino com a célula em 100 → 10/10 sucessos, score sobe (incluindo o
+9 se chegar lá). Depois `Reset` da célula + restart no fim do roteiro e conferir a distribuição
normal. Combine: só marque ✅ se houver receita acessível no jogo; caso contrário registre
"não executado" (não é falha).

**Captura:** vídeo/print dos refinos, `select * from content_rate_override;` e as linhas
`rate_overrides_applied` + `content rate overlay effective`.

### Passo F — efeito de item na ficha (44)

`/gm item <armadura>` → equipar → abrir a ficha: a defesa sobe **exatamente 60**. Desequipar volta.

**Captura:** print da ficha antes/depois.

### Passo G — imposto da cidade e Kefra (45)

Ir à loja de Armia e conferir o preço com a taxa da zona em **5**; `/kefra` deve levar ao Continente
desconhecido porque o Kefra está marcado como morto.

**Captura:** print do preço na loja, print do `/kefra`, e `select zone, city_tax from guild_zone;`.

### Passo H — atlas contra o mundo (48)

Andar até a coordenada anotada em `/atlas`: o ponto aberto é caminhável; a parede aparece escurecida
no atlas e bloqueia no jogo.

**Captura:** print do atlas e do ponto no jogo.

### Passo I — segunda janela: trade e visão (12, 10)

Abrir **uma segunda instância** do cliente (outra conta, ou outro personagem da mesma conta para o
cargo):

1. Os dois personagens se veem e veem os NPCs no mesmo mapa.
2. Trade: trocar item + gold; **cancelar** um trade no meio e conferir que nada duplicou/perdeu.
3. Cargo: mover item para o cargo com o personagem A, logar B da mesma conta e ver o item; relogar.

**Captura:** prints dos dois lados + conferência SQL de item/gold antes e depois:

```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select name, coin from character where name in ('<A>','<B>');"
```

### Passo J — persistência (09)

Fechar o cliente, **restart do tmserver** (único restart permitido no meio, e é o objetivo do passo),
reabrir e relogar:

- inventário, equip e gold iguais;
- exp/level/MaxHP/MaxMP iguais;
- `last_city` preservada e spawn coerente;
- buffs/affects e barra de skill reidratados, **sem** vazar entre personagens da mesma conexão;
- `role`/`is_blocked` valendo e auditoria de GM no log.

**Captura:** `select` antes e depois + linhas de login/save do tmserver.

## 2. Limpeza (obrigatória)

```bash
# no portal: Reset em /precos (907), /drops/editar (slot 12), /drops/carry (slot 56),
#            /config (célula de refino), /itens (EF_AC), /castelo (taxa e Kefra),
#            /eventos (desligar tudo) e apagar a oferta de /diaria
docker compose up -d openwyd-nextgen1-tmsrv     # volta às flags OFF (sem as variáveis)
```

Conferir que o boot voltou ao baseline e que nada de `tmserver/content/` mudou:

```bash
docker compose logs --tail 20 tmserver | grep -E "overlay|generators|mobs="
git status --porcelain
```

## 3. Registro

Preencha a tabela §12 de [`roteiros-qa.md`](roteiros-qa.md) e cole a evidência no arquivo de cada
issue, marcando **só** o critério que o passo provou. Uma sessão bem executada fecha **9 issues**
(24, 25, 34, 36, 44, 45, 48, 28, 30) e avança **09** e **08** para ✅, deixando 10/11/12/13 com o
que foi coberto e o que falta.

## Preparação aplicada (05/11/2026, 13:54) — stack local

Valores que estão no banco agora; use-os como referência ao conferir no cliente.

| O que | Valor preparado | Onde conferir |
|---|---|---|
| Preço do item 907 | **12345** | `item_price` · `/precos` |
| NPC | `qa-ferreiro` (template `Acessorios`, mercador) em Armia **(2504,1684)** vendendo **907** e **5134** | `npc_definition` · `npc_shop_item` |
| Mob alvo | **Troll** natural (97 geradores, a partir de 2454,1836) — o override de stat foi **removido**, ver QA-02 | `mob_carry_override` |
| Drop | slot **12** (equipamento comum) divisor **4** | `drop_rate_override` |
| Carry | **Troll** slot **56** = item **412** (Poeira de Oriharucon, sempre cai) | `mob_carry_override` |
| Refino | `SancRate` **PO** índices 0..11 = **100** | `content_rate_override` |
| Imposto | cidade **5** nas 5 zonas | `guild_zone` |
| Kefra | **morto** (revision 2) | `kefra_state` |
| Evento global | ligado, item 400, **1 em 5**, faixa 1..50, **EXP dobrada**, notice | `world_event_config` |
| Diária | item **400**, 7 dias, habilitada | `daily_reward_item` |
| Efeito | **Armadura Couro Duro (1116)**: `AC +60` | `item_effect_override` |

O que o boot registrou (log real do `tmserver`):

```
mob template stat overlay enabled (moderator editing) overrides=0
mob carry overlay enabled (moderator editing) templates=1
content rate overlay applied rate_overrides_applied=12 rows=12
content rate overlay effective sancrate_ori="[100 100 100 100 100 100 100 100 100 100 100 100]"
drop rate overlay applied drop_overrides_applied=1 slot_12=4
item effect overlay applied rows=1 replaced=1
npc config applied at boot version=27 npcs=5162
world event config applied at boot version=2
```

Nota de infraestrutura: o `docker-compose.yaml` repassava cinco das seis flags — `W2PP_MOB_CARRY_EDITING`
não chegava ao container e o overlay de `Carry` nunca ligava. A variável foi adicionada ao serviço
`tmserver`; o alvo de drop é o **Troll**. O override de stat do Troll foi removido no mesmo dia: ele
zerava `class`, `clan` e `equip` e deixava o mob com modelo de TK e pacífico (QA-02).

## Resultado da primeira execução (05/11/2026, 14:05–14:33)

Cliente real, conta `test`, personagens `TestHT` (lv 35), `TestTK`, `TestFM`, `TestBM`. Cada achado
abaixo foi confirmado no servidor (log + SQL), não é impressão do cliente.

| Passo | Resultado | Evidência |
|---|---|---|
| A — login e diária | login OK, **diária não recebida** | `daily_reward_claim=0`, `delivery_queue=0`; nenhum caminho de diária no `tmserver` → **QA-01** |
| B — criação | 4 personagens com equip inicial e 1M de gold | `select * from character` (4 linhas, `class` 0..3) |
| C — cidades | `/armia /erion /azran /gelo /noatun` OK | `msg=teleport … to_x/to_y` de cada comando |
| C — `qa-ferreiro` | não apareceu: estava em (2504,1684), o jogador em (2502,1716) | `npc config applied at boot … npcs=5162`; reposicionado para (2098,2093) |
| C/D — mobs da cidade | não atacam e não podem ser atacados | `world/city.go` `nonCombatNPC`: `clan=0` em aldeia ⇒ NPC de serviço → **QA-04** |
| D — mob alvo | Troll com modelo de TK e intocável | overlay de stat zerou `class`/`clan`/`equip` → **QA-02** |
| E — refino | `/gm setgold` OK; `/gm item 412 20` **derrubou o cliente** | `msg="gm item" … item=412 amount=20 slot=7` (servidor aceitou) → **QA-03** |
| F..J | não executados | crash no passo E |

O que já foi corrigido na sessão: flag `W2PP_MOB_CARRY_EDITING` repassada pelo compose, override do
`Troll` removido (`overrides=0` no boot de 14:48), `qa-ferreiro` em (2098,2093) ao lado do spawn e
conta `test2` (papel `player`, mesma senha) para o teste de duas janelas.

### O que dá para retestar agora, sem reiniciar mais nada

1. **C** — andar até o spawn de Armia e abrir o `qa-ferreiro` em (2098,2093): o item 907 aparece por
   **30000**. Esse é o `item_catalog.price` real: `npc_shop_item` não tem coluna de preço, então a
   preparação de 12345 não tinha onde gravar. O preço não é editável por NPC hoje.
2. **D** — matar **Troll fora da cidade** (a partir de 2454,1836): drop garantido de 412 (carry do Troll)
   e a frequência do slot 12 com divisor 4. Mobs **dentro** da aldeia continuam pacíficos por design.
3. **E (resto)** — refino com `SancRate` PO em 100 e o AC +60 da `Armadura_Couro_Duro` (1116): nenhum
   dos dois usa o `/gm item` com quantidade.
4. **F..J** — ficha, imposto/Kefra, atlas, duas janelas (conta `test2`) e persistência.

## Segunda execução (05/11/2026, tarde)

| Passo | Resultado |
|---|---|
| C — NPC e preço | `qa-ferreiro` ao lado do spawn; 907 por **30000** (preço do catálogo) |
| D — mob alvo | Troll fora da cidade: **412 cai sempre**, várias mortes |
| E — refino com Ori | para em **+6** (limite do código, `oriMaxSanc`); outro pó levou a +9 |
| E.2/E.3 | túnica de pano +9: defesa 149 → 189; arco curto +9: ataque 69 → 118 |
| E.4 | **nenhuma** mensagem de refino aparecia no cliente |
| F — `/kefra` | ok |
| H — `/atlas` | nada acontece — não é comando in-game |
| Troca e grupo | sem segunda sessão simultânea não dá para testar |
| Guilda | `/create` exige clã 7/8 + cidadania; nada concede → QA-05 |
| Duelo | teleporte para a arena PvP, ok |

### Correções desta rodada

1. **Mensagens de refino (E.4).** `sendRefineMessage` mandava o `MSG_MessagePanel` por `w.Send`, que
   carimba `HEADER.ID = conn`; o cliente 7662 só trata painel com `ID = 0` (`TMScene.cpp:1371`) — o
   mesmo motivo já documentado em `notice.go` para `sendClientMessage`. Passou a usar
   `SendTo(Header{Type: MsgMessagePanel})`. Isso cobre sucesso, falha, "não pode ser mais refinado"
   (inclusive o muro do Ori em +6) e as recusas do Classe.
2. **Atlas.** `/atlas` é a página do portal, não um comando do cliente; o passo H deve ser lido assim.
   Um `/atlas` digitado no jogo cai como whisper para o jogador "atlas" e não mostra nada.
3. **Troca e grupo exigem duas sessões.** `_MSG_Trade`/`_MSG_SendReqParty` agem sobre o outro jogador;
   com um cliente só, o convite não tem alvo. O teste é o passo I (duas janelas), com `test2`.
