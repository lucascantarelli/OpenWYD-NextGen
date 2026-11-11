# Campo de Treinamento — pesquisa para o primeiro gameplay (2026-11-10)

> Missão: login → criar personagem → **entrar no mundo** → primeira quest (Campo de
> Treinamento) funcional. Este documento reúne o que as fontes oficiais/licenciadas e o
> **código legado** (`Jean1dev/w2pp-OpenWYD`, o mesmo alvo 7662) dizem sobre a quest.
> Nada aqui vira regra sem escolha registrada + teste (rito `docs/rust/02-execucao.md`).
> Níveis de confiança em `fontes.md`: **A/B** oficial · **C** privado com guia · **D**
> comunidade · **E** dado do repositório.

## 1. Fluxo canônico (CONFIRMADO em fonte oficial)

`Unicorn (arma inicial) → buff do Cabuncle/Carbunkle → guardião da 1ª chave → treinador 1 →
2ª chave → treinador 2 → 3ª chave → treinador 3 → boss Orc → emblema → Chefe de Treino`.

Mapa: campo a **leste de Armia**; faixa de nível **1–35** (o texto oficial diz “abaixo do
nível 35”, o título/FAQ dizem 1~35 — divergência registrada, decisão pendente).

### 1.1 NPCs e coordenadas (oficial, WYD Global)

| NPC | Coord. | Função |
|---|---|---|
| **Unicorn** | 2130,2040 | entrega/escolhe a **arma inicial** (física para classe física, mágica para classe mágica) |
| **CabuncleWind / Carbunkle Wind** | 2131,2033 | **NPC de buff**; pode ser clicado de novo quando o buff acaba; há outro depois do 1º e do 3º portão |
| **FirstTrainer** | 2077,2017 | recebe a 1ª chave |
| **SecondTrainer** | 2144,1989 | recebe a 2ª chave |
| **ThirdTrainer** | 2082,1966 | recebe a 3ª chave |
| **TrainerChief / Chefe de Treino** | 2122,2041 | recebe o **Emblema Orc** e entrega a recompensa final |

Fonte: <https://wydglobal.raidhut.com/pt-br/2402> (oficial) e
<https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/20369-campo-de-treinamento-lv-135>.

### 1.2 Portões e chaves (oficial)

| Chave | Guardião (drop) | Guardião (x,y) | Portão (x,y) |
|---|---|---|---|
| Chave da Primeira Porta | **GremlinDecoy** | 2076,2022 | 2075,2016 |
| Chave da Segunda Porta | **GiantCrill** | 2146,1997 | 2143,1986 |
| Chave da Última Porta | **OrcSniper** | 2078,1971 | 2081,1962 |

Regras oficiais: a chave é **consumida no uso** e o portão **fecha depois de um tempo**; se a
chave for perdida, o guardião volta a dropá-la enquanto a quest estiver ativa. A base oficial
classifica a chance como **“Muito Alta”**, não 100%.

### 1.3 Boss final (oficial)

| Campo | Valor |
|---|---|
| Nome | **OrcWarrior / Guerreiro Orc** (2135,1948) |
| Nível / HP / MP / EXP | 40 / **2.000** / 0 / **240** |
| Drops | **Âmago de Javali** e **Emblema Orc** (ambos “Muito Alta”) |
| Mecânica | boss de campo simples; sem fases/invocações documentadas; o Emblema **não é garantido** |

Fontes: <https://global-droplist.raidhut.com/element?id=OrcWarrior&type=mob> e o guia oficial.

### 1.4 Drops dos guardiões (oficial, qualitativo)

| Monstro | Nv | HP | EXP | Drops |
|---|---:|---:|---:|---|
| GremlinDecoy | 30 | 600 | 120 | Chave da 1ª Porta (Muito Alta) + armas (Alta) |
| GiantCrill | 50 | 2.000 | 140 | Chave da 2ª Porta (Muito Alta), Rubi do Carbunkle |
| OrcSniper | 50 | 2.500 | 160 | Chave da Última Porta (Muito Alta), Ovo de Javali, Resto de Oriharucon (Baixíssima) |
| OrcWarrior | 40 | 2.000 | 240 | Emblema Orc, Âmago de Javali |

## 2. O que o **código legado** faz (fonte primária para o nosso porte)

O Campo **não tem script**: é `switch` em `Source/Code/TMSrv/_MSG_Quest.cpp` sobre o NPC,
com progresso em `STRUCT_MOBEXTRA.QuestInfo.Mortal.Newbie` (**0 → 4**, literalmente “quest com
quatro etapas” no `Basedef.h`).

### 2.1 Contrato do pacote

| Item | Valor |
|---|---|
| Opcode | **`_MSG_Quest` = 0x028B** (`139 \| FLAG_CLIENT2GAME`) |
| Struct | **`MSG_STANDARDPARM2`** — `Parm1 = npcIndex`, `Parm2 = confirm` |
| Como o servidor identifica o NPC | `MOB.Merchant` (36/40/41) e, para o 4º, `Merchant == 100` + grade 16 do `Equip[0]` |
| Feedback ao cliente | **não existe janela de quest**: `SendSay` (chat do NPC), `_MSG_MessagePanel`, `_MSG_MessageBoxOk`, `_MSG_UpdateEtc`, `_MSG_UpdateCarry` |
| Pacotes inexistentes | `MSG_QuestInfo`, `MSG_QuestComplete`, `MSG_QuestAccept`, `MSG_QuestReward` (não há símbolo no 7662) |

### 2.2 Cadeia dos quatro treinadores (código, `_MSG_Quest.cpp`)

| Etapa | NPC (Merchant) | Item exigido | Flag | Recompensa no código |
|---:|---|---|---|---|
| 1 | 36 (`TREINADORNEWBIE1`) | **451** | Newbie 0→1 | item **682** com `EF 61` = **20** + affect 44 (200,200); recarrega score/carry |
| 2 | 40 (`TREINADORNEWBIE2`) | **452** | 1→2 | **encanta a arma equipada** (`Equip[6]`) com `SetItemBonus(50 + ReqLvl, 1, 0)`; só `ReqLvl <= 39` |
| 3 | 41 (`TREINADORNEWBIE3`) | **453** | 2→3 | **encanta `Equip[1..7]`** com `50 + ReqLvl` **e** aplica **sanc 3** |
| 4 | 100 + grade 16 (`TREINADORNEWBIE4`) | **524** | 3→4 | consome 524 e sorteia `rand()%3`: **682** (pot 20), **481** ou **652** |

Pré-requisitos comuns: `ClassMaster == MORTAL`, nível abaixo de `FREEEXP` e a flag no estágio
anterior. Faltando o item, o NPC responde a mensagem `_NN_NewbieQuest_CheerupN`.

### 2.3 Quem dropa os itens de prova (`ItemDropList.txt`)

| Item | Monstros |
|---|---|
| 451 | `GremlinDecoy`, `Gremlin_Artif` |
| 452 | `GiantCrill`, `Krill_Gigante` |
| 453 | `OrcSniper`, `Orc_Sniper` |
| 524 | **`OrcWarrior`, `Orc_Guerreiro`** |

### 2.4 Portões/chaves no legado (genérico, `_MSG_UpdateItem.cpp`)

1. `ItemID` do portão = id do item do mundo + 10000.
2. `gatekey = EF_KEYID` do item-portão; estado `STATE_LOCKED` (3).
3. Se `gatekey != 0`, procura no inventário um item com o mesmo `EF_KEYID`; sem ele responde
   `_NN_No_Key`.
4. Com a chave: **apaga a chave** (`SendItem`), `UpdateItem(gate, STATE_OPEN)` e
   `GridMulticast`; `Server.cpp` fecha de novo pelo `Delay`.

### 2.5 Estruturas correlatas

- `STRUCT_QUEST`/`STRUCT_QUESTDIARIA`/`QuestsRate.txt` são **quests de nível/diárias**, não o
  Campo.
- `STRUCT_CAMP` (`BauIndex`, `BauQuest`, `KeyIndex`, `Prize[12]`, `CoinPrize`, `ExpPrize[6]`,
  `RenewTime`) existe no header, mas **não há arquivo de camp versionado** no dump público.
- `Guard.txt` + `BASE_ReadGuardFile` carregam 5 linhas encadeadas (5617→5612→5613→5614→5615);
  **não foi provado** que sejam o “guarda” do Campo — parece troca encadeada de NPC.

## 3. Sequências de recompensa encontradas (escolha do dono: **A**)

> **DECISÃO (2026-11-10, dono):** sequência **A — Legado OpenWYD**, limite de nível **1 a 35**
> e arma inicial **Unicorn oficial**. D-1, D-2 e D-6 fechadas; o quadro abaixo fica como
> registro das alternativas descartadas.

| ID | Origem | Treinador 1 | Treinador 2 | Treinador 3 | Chefe final |
|---|---|---|---|---|---|
| **A** | **Legado OpenWYD (código)** | item 682 (20 potes) + affect 44 | encanta a arma (`Equip[6]`) | encanta `Equip[1..7]` + **sanc 3** | `rand()` entre 682 / 481 / 652 |
| **B** | **Oficial atual (PT)** | 20 Poção de Cura (M) | adicionais aleatórios nas armas | refino até **+6** | **Olho Crescente** (3.000 EXP) |
| **C** | Oficial EN antigo + comunidade | 20 poções | arma refinada + adicionais | 5 “Classe A” | Rubi do Carbunkle **ou** Olho Crescente **ou** 20 poções (sorteio) |
| **D** | Privado (WYD Brazuca) | **Rei dos Nbs** vende **Arma Sephira +9** (dano full) no campo e no centro de Armia | — | — | — |

Observações:

- **Sephira não é oficial**: nas fontes oficiais o NPC de arma inicial é o **Unicorn** e a arma
  não se chama Sephira. O “+9 full” é configuração de servidor privado.
- **Não existe escolha de recompensa por classe** em nenhuma fonte; a única escolha por classe
  documentada é a **arma inicial do Unicorn**.
- Os nomes dos itens 451/452/453/524/682/481/652 não foram confirmados por `ItemList.csv`
  (arquivo grande demais para leitura web). No **nosso** catálogo (`item_catalog`) os ids e
  nomes precisam ser conferidos antes de virar seed.

## 4. Divergências e decisões pendentes

| # | Dúvida | Fontes | Decisão |
|---|---|---|---|
| D-1 | Nível 35 entra? | texto “abaixo de 35” × título/FAQ “1~35” | ✅ **1 a 35 entra** (título + FAQ oficiais) |
| D-2 | Sequência de recompensa (A/B/C/D acima) | legado × oficial × privado | ✅ **A — legado OpenWYD**: 682+affect 44 → encanta arma → encanta `Equip[1..7]`+sanc 3 → sorteio 682/481/652 |
| D-3 | 3º guardião: `OrcSniper` × “Arqueiro Orc” | oficial atual × guias 2009 | usar `OrcSniper` (oficial) até captura |
| D-4 | Treinador 2: só armas × armas e armaduras | oficial PT × oficial EN | só arma (`Equip[6]`), como o código |
| D-5 | Recompensa final: fixa × sorteio | PT × EN/comunidade | **sorteio** `rand()%3` = 682/481/652 (sequência A) |
| D-6 | Arma Sephira +9 | privado | ✅ **não entra**: arma inicial é o **Unicorn** oficial (sem Sephira) |

## 5. Encaixe no nosso servidor (o que falta)

1. **Entrada no mundo visível** — o bloqueio atual: id de cena do jogador acima de `MAX_USER`
   (1000) e falta do `UpdateScore` de entrada (corrigido em `FASE-03-17`).
2. **Handler `_MSG_Quest` (0x028B)** com validação de NPC por `merchant`, estágio
   `Newbie` persistido por personagem (tabela própria, sem escrever em tabela de personagem
   vivo) e as recompensas escolhidas.
3. **Portões/chaves**: `_MSG_UpdateItem` com `EF_KEYID`, consumo da chave e fechamento por
   tempo — hoje o opcode cai em `outro`.
4. **Buff NPC**: affect 44 do legado (200,200) ou o que o snapshot do nosso servidor definir.
5. **Boss e drops**: `npc_definition` + `drop_table` do `OrcWarrior` e dos três guardiões.
6. **Bots**: 4 bots autônomos (um por raça) executando a quest inteira (`FASE-15-4`).

### 2.6 Tabela de implementação fixada pela decisão do dono

| Etapa | Merchant | Item de prova | Efeito no servidor |
|---:|---|---|---|
| 1 | **36** | 451 (Chave da Primeira Porta) | entrega **682** com `EF_HP` 200 + affect **44 (200,200)**; `Newbie 0→1` |
| 2 | **40** | 452 (Chave da Segunda Porta) | encanta **`Equip[6]`** com `SetItemBonus(50 + ReqLvl, 1, 0)`; `1→2` |
| 3 | **41** | 453 (Chave da Última Porta) | encanta **`Equip[1..7]`** com `50 + ReqLvl` **e** sanc **3**; `2→3` |
| 4 | **100** + grade 16 | 524 (Emblema Orc) | consome 524 e sorteia **682 / 481 / 652** (1/3 cada); `3→4` |

Limite: nível **1..=35** (36 já recusa). Arma inicial: **Unicorn**, sem Sephira. Pré-requisitos
comuns: `MORTAL`, nível no limite, estágio anterior concluído.

Issues: `FASE-03-17` (entrada no mundo fiel ao legado, entregue), `FASE-03-18` (Campo de
Treinamento), `FASE-15-4` (bots).
