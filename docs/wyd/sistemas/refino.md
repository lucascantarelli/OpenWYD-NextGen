# Aprimoramento de item — do +0 ao God2

Este é o sistema com mais números do jogo e o que mais gera suporte. Tudo abaixo está
separado por **etapa**, com **taxa**, **custo** e **consequência de falha**. Onde duas fontes
oficiais divergem no custo, a divergência está registrada — porque é exatamente o tipo de
número que cada servidor configura.

---

## 1. A escada completa

| Faixa | NPC | Sistema | Resultado |
|---|---|---|---|
| +0 → +9 | qualquer cidade | **Refino** com Pó de Ori / Pó de Lac | item ganha sufixo `+N` |
| +9 → +10 | **Aylin**, Azran (2529,1725) | **Composição** de dois +9 | +10 com gema de efeito |
| +9 → **Anct** | **Compositor**, Azran (2535,1734) | **Composição Ancient** (armas) | volta a **+7**, com efeito de joia |
| +11 → +15 | **Alquimista Odin**, Kefra | **Composição avançada** | +N, com pedra secreta |
| +N → **Celestial/God2** | **InfWeapon / InfArmor** (dragão de Azran) | selagem e **Essência dos Deuses** | item celestial |

**CONFIRMADO** (central Kersef: "Refining System +0 to +11", "Refining +10 and changing
additionals", "Armas Ancient", "Sistema de Refinação +12 a +15"; WYD International
"Server Information" com as tabelas de taxa).

---

## 2. Refino +0 → +9

### Materiais

| Material | Uso |
|---|---|
| **Pó de Oriharukon (Ori)** | refino de item comum |
| **Pó de Laktolerium (Lac)** | refino e composições de alto nível |
| **Pedra do Sábio (Wise Man's Stone)** | obrigatória no +10 |
| **Pó de Ori em barra / resto** | receitas de NPC (10 restos + 1kk → Jephi, Azran) |

**CONFIRMADO**. Armaduras **e** armas refinam; **escudos refinam**; **acessórios não** — no original. Refino de **brinco** é sistema posterior, de servidor (§7.3).

### Taxas

As taxas oficiais publicadas (WYD International, "Server Information") decrescem com o nível
em passos de 5 pontos percentuais:

| Passo | Taxa |
|---|---|
| +0 → +1 | **90%** |
| +1 → +2 | **85%** |
| +2 → +3 | **80%** |
| +3 → +4 | **75%** |
| +4 → +5 | **70%** |
| +5 → +6 | **65%** |
| +6 → +7 | **60%** |
| +7 → +8 | **55%** |
| +8 → +9 | **50%** |
| +9 → +10 | **30%** (composição, §3) |
| +10 → +11 | **3%** refino com **Poeira de Lactolerium** ("Lac 100%" = 100%) |

**CONFIRMADO** para a escada +0→+9 na tabela do WYD International (`rate1`…`rate9`).

> **+10 → +11 é refino, não composição.** A central Kersef ("Refining System +0 to +11")
> descreve o passo como *"usando Poeira de Lactolerium no item +10 você poderá refiná-lo até
> +11"*; as wikis publicam **3%** para a Lac normal e **100%** para o item de cash
> **"Lac 100%"**. O legado tem **15%** fixo em código no passo `GEM_LEVEL` (+10) e **100%**
> quando o pó é o `Lactolerium_100` — as células Âmago 10/11 do `SancRate.txt` (5%) **não**
> são consultadas nesse nível. O dado registra as três configurações; o seed adota os 3%
> publicados.
>
> **Tabela efetiva do legado (36 células).** O `SancRate.txt` é aplicado **por cima** do
> default compilado (`Basedef.cpp g_pSancRate`), e a leitura é em `nível + 1`. O arquivo lista
> PO 0–5, PL 0–9 e Âmago 0–11; o resto (PO 6–11, PL 10–11) vale **0** — não "sem dado".
> Consequência: PO 5 = 40% é o passo +4→+5, e +5→+6 (PO 6 = 0%) só sobe por **piedade**.
> A auditoria completa, com as divergências e o backlog de port, está em
> [`refino-auditoria.md`](refino-auditoria.md).

> **Divergência entre fontes, registrada:** a wiki Fandom descreve +0→+4 como **100%** e
> só depois começa a cair; outra fonte ainda lista +12→+15 como **15/14/13/12%**. São
> **configurações de servidor diferentes**, não erro de leitura: o valor base é do
> `game_setting`, e a sequência 90/85/80/75/70/65/60/55/50 é a do servidor oficial atual.
>
> No banco isso vira `refine_rate(step, rate)` semeado com a escada acima, e o portal
> mostra a origem (`content_source`) de cada degrau — nunca um chute silencioso.

### Falha

| Faixa | Consequência |
|---|---|
| +1 … +6 | item **volta a +0** |
| +7 … +9 | item é **destruído** (em item celestial/selado) |

**CONFIRMADO** (central Kersef, "Refining System +0 to +11"; itens selados).

> **Item selado (sealed)** refina só até **+9** e, em falha de +7 a +9, **destrói**.
> É o item de recompensa de boss; a diferença entre selado e comum precisa estar no dado
> (`item.sealed`), não no nome.

### Refino abençoado (Bless Refining)

Cada nível de refino do item dá bônus percentual na **composição** (não no refino):

A tabela **muda por servidor**, e as duas publicadas são internamente coerentes com o próprio
exemplo:

| Refino do insumo | Kersef / WYD Global (**default**) | WYD International (variante) |
|---|---|---|
| +0 | **2%** | 1% |
| +1 | **3%** | 1% |
| +2 | **4%** | 2% |
| +3 | **5%** | 3% |
| +4 | **6%** | 4% |
| +5 | **7%** | 5% |
| +6 | **8%** | 6% |
| +7 | **10%** | 7% |
| +8 | **12%** | 8% |
| +9 | **15%** | **10%** |

Exemplo de cada lado: 4 refinações **+9** valem **60%** no Kersef (15% × 4) e **40%** no WYD Int
(10% × 4). **CONFIRMADO** nas duas centrais (artigo "Refining System +12 to +15", EN e PT, e
"Server Information"). O **default documentado** para o 7662 é a tabela do Kersef — a mesma que [`itens.md`](../itens.md)
já publica ("2% a 15%"). O seed atual (`BLESSING_BONUS_BPS`, `wyd-sim`) **ainda** usa a coluna do
WYD Int; a troca é o P-10 do dossiê, registrada na auditoria (D-R6).

---

## 3. +10 — Aylin, em Azran

**Local:** NPC **Aylin / Aileen** (X:2529 Y:1725), Azran.

| Requisito | Quantidade |
|---|---|
| Itens **+9 iguais**, mesmo tipo (N, M, A, Le ou Anct) | **2** |
| Jóias **iguais** | **4** |
| **Pedra do Sábio** | **1** |
| Ouro | **50.000.000** |

**Sucesso:** todos os insumos são destruídos e sai **um item +10**.
**Falha:** jóias, pedra e ouro são perdidos.

**CONFIRMADO** (central Kersef, versão EN, "REFINING +10 AND CHANGING ADDITIONALS").

> **Divergência registrada:** a mesma central, na versão PT-BR do mesmo artigo, lista
> **[30] Poeiras de Lac + 5.000.000 de ouro** em vez de Pedra do Sábio + 50kk. Não é erro de
> tradução: são **duas configurações de servidor diferentes** (a EN é Global, a PT é build
> antigo). A wiki Fandom confirma a versão Global (Pedra do Sábio + 50kk + 4 joias iguais) e
> acrescenta a taxa: **30–40%** de sucesso (o número exato é do servidor). No banco, os dois
> lados viram linha de `game_setting`; o default adotado é o da EN, com `content_source`
> apontando a divergência.

### Gema de efeito (obrigatória no +10)

| Jóia | Efeito extra |
|---|---|
| **Diamante** | +8% taxa de drop |
| **Esmeralda** | +40 pontos de **perfuração** |
| **Coral** | +2% taxa de experiência |
| **Garnet** | +40 pontos de **absorção de dano** |

**CONFIRMADO**. Para arma **Anct**, os efeitos das duas armas precisam ser iguais e a gema
precisa ser compatível com o efeito.

### Troca de efeito depois (as 4 gemas)

O efeito **não é definitivo**: ele pode ser trocado usando a **gema correspondente**, e são
**4 itens** — um para cada efeito. Todos existem no `item_catalog`:

| Item (`item_catalog`) | Nome | Efeito que aplica |
|---|---|---|
| **3386** | Gema de Diamante | **+8% de drop** |
| **3387** | Gema de Esmeralda | **perfuração** de dano |
| **3388** | Gema de Coral | **+2% de EXP** |
| **3389** | Gema de Garnet | **absorção** de dano |

**Como se usa:** a gema é aplicada **sobre o item** (jogar/arrastar a gema no item), não em
NPC. Vale para **item Anct** e para **item +10 ou superior**. A troca **não destrói** o item
e pode ser repetida quantas vezes quiser, desde que o item esteja no nível exigido.

**Escala (fecha a lacuna do +10 pra cima):** perfuração e absorção **escalam linearmente**,
**40 por nível acima de +9**:

| Refino | Perfuração / Absorção |
|---|---|
| +10 | 40 |
| +11 | 80 |
| +12 | 120 |
| +13 | 160 |
| +14 | 200 |
| +15 | 240 |

Drop (**8%**) e EXP (**2%**) são **fixos**, não escalam com o refino.

**CONFIRMADO** (central Kersef, *Premium Neil*: "Gema de Garnet — colocar o adicional de
absorção de dano em itens Anct, +10 ou superior", e as outras três; ABC do WYD e WYD Tutorial
com a escala 40/80/120/160/200/240 e o uso por aplicação direta). As gemas são vendidas no
NPC **Lucy**, em Azran.

### Adicionais

Toda arma tem **dois adicionais internos**, mesmo quando o cliente mostra só um. Na
composição, o resultado pode herdar **um dos dois** de cada arma de origem — não a soma.
Exemplo oficial: cajado com 8 de ataque mágico + cajado com 42 de skill → o resultado pode
sair **8 ou 42**, nunca 50. **CONFIRMADO**.

---

## 4. Arma Ancient (Anct) — Compositor, Azran

**Local:** NPC **Compositor** (X:2535 Y:1734), Azran.

| Requisito | Detalhe |
|---|---|
| Arma **+9** | base |
| **1 jóia** | Diamante, Esmeralda, Coral ou Garnet |
| **0 a 4 itens de reforço** | qualquer item com refino **+7 ou maior**, até **uma classe abaixo** da arma |

**Taxa base: 5%.** Reforços somam:

| Reforço | Bônus |
|---|---|
| +7 | +2% |
| +8 | +4% |
| +9 | +10% |
| +10 | +12% |

Exemplo oficial: 4 reforços +10 = **+48%** → total **53%**. **CONFIRMADO**.

- **Sucesso:** todos os insumos destruídos; a nova arma nasce **+7** com o efeito da joia.
- **Falha:** **tudo** é destruído, inclusive a arma +9.
- Só **armas** viram Anct (defesa **não**). Item +11 **não** serve de reforço.

**CONFIRMADO** (central Kersef "Armas Ancient"; WYD Brazuca, "Composição de Arma Anct").

> Não existe "Anct de segunda vez": recompor uma Anct não funciona. É `item.ancient = true`
> com `UNIQUE`, não uma flag reaplicável.

---

## 5. +12 a +15 — Alquimista Odin, Kefra

**Local:** NPC **Alquimista Odin**, Continente Desconhecido (cidade do Kefra).

| Requisito | Quantidade |
|---|---|
| Item **+11 ou superior** (Anct, no caso de arma) | 1 |
| **Pedras secretas** aleatórias **ou** refinações abençoadas | **4** |
| **Pacotes de Lac com 10** **ou** **Emblemas da Proteção** | **2** |

### Taxa fixa

| Passo | Taxa |
|---|---|
| +11 → +12 | **5%** |
| +12 → +13 | **3%** |
| +13 → +14 | **2%** |
| +14 → +15 | **1%** |

> **Só o +12 muda entre as fontes abertas:** o guia da era oficial (2009) publica **5%**; as
> centrais atuais (Kersef EN **e** PT, WYD Int, 2022–2025) publicam **4%**. +13/+14/+15 são
> **3/2/1%** em todas. Como o alvo do projeto é o build **7662**, o **default documentado** é **5%**; o **4%**
> fica como variante de servidor moderno. O seed atual (`migrations/0022_seed_refino.sql`,
> `PLUS12_BASE_BPS`) **ainda** grava **400 bps**; a troca é o P-10, registrada em D-R4.

### Bônus por insumo

| Insumo | Bônus |
|---|---|
| **Pedra secreta** | **+1%** cada |
| **Refinação abençoada** | **+2% a +15%** conforme o refino (tabela do §2) |

Exemplo da central (base 4%): +11→+12 com 4 pedras secretas = 4% + 4% = **8%**. Com o default documentado de
**5%**, a mesma tentativa dá **9%**. **CONFIRMADO** (exemplo da central) e **configurável**
(taxa).

### Consequência de falha (é aqui que o jogador perde tudo)

| Insumo usado | Falha |
|---|---|
| **Pacote de Lac** | todos os itens destruídos **e** pequena chance de o item ser destruído |
| **Emblema da Proteção** | item **cai um nível** (+11 → +10, +14 → +13); insumos destruídos |

**CONFIRMADO** (central Kersef, "Sistema de Refinação +12 a +15").

> Emblema da Proteção é o item mais valioso do jogo por isso: ele transforma "perdi o item"
> em "perdi 1 nível". O banco precisa guardar `protection_used` na tentativa para auditoria.

> **Número que faltava para o Pacote de Lac** (`[C]` WYD Int, guia 47): se a **taxa total** da
> tentativa ficar **abaixo de 5%**, a falha tem **20% de chance de destruir o item**.
>
> **Divergência nova (D-10):** o bônus do **Emblema da Proteção** é **+1% cada** no Kersef e
> **+10% cada** no WYD Int. Mesma função, valor dez vezes maior — é `game_setting`, não constante.

---

## 6. Celestial / God2 — Essência dos Deuses

Itens celestiais sobem de nível com **Essência dos Deuses (Essence of the Gods)**. Cada
essência tem seu próprio **+N**, e o bônus é a soma:

| Essência | Bônus |
|---|---|
| +0 | 1% |
| +1 | 1% |
| +2 … +8 | +1% por nível |
| +9 | **10%** |

Exemplo oficial: **4 essências +9 = 40%** de sucesso. **CONFIRMADO** (WYD International,
"Celestial/God2 Upgrade Rates").

Antes disso, **InfWeapon / InfArmor** (perto do dragão de Azran) transforma item em celestial:

| Caminho | Custo |
|---|---|
| A | **5 Símbolos do Infinito** |
| B | **2 itens selados iguais + 1 Pergaminho Selado + 1.000.000 de ouro** |

Resultado: **item celestial selado +4**. **CONFIRMADO**.

### Combinação God (armadura e arma)

| Componente | Arma | Armadura |
|---|---|---|
| Base | celestial **+9** | Arch **+15** |
| Catalisador | **Joia da Escuridão** | **Pedra Lunar** |
| Pedras | 4 pedras secretas | 4 pedras secretas |

**CONFIRMADO** (central Kersef, guia PT-BR de combinação god).

---

## 7. Repleção e Imbuir (sistemas vizinhos — não confundir)

### Repleção (`replecao`)

- Pedras de repleção **grau A, B e C**, vendidas em **Armia, Azran e Erion**.
- Só em **armadura**, e só **no inventário** (não equipada, não no baú).
- Gera **opção adicional aleatória**; pode sair "não refinada" e **remover** a anterior.
- **CONFIRMADO** (central Kersef, "Repletion System").

### Imbuir (Agatha)

| Grau do item | Chance |
|---|---|
| D (nenhum) | 6% |
| D (Normal) | 12% |
| D (Mystic) | 18% |
| D (Arcane) | 24% |
| D (Legendary) | 30% |

**CONFIRMADO** (WYD International, "NPC Agatha — Imbue").

> Repleção, imbuir e refino são **três estados independentes** no item. Um item pode ser
> +10, com repleção e com imbuir. Modelar como uma tabela `item_instance` com três colunas
> separadas evita o bug de "refinar apaga a repleção".

### Brinco (sistema posterior ao original — D-8)

Nenhuma fonte da era oficial (2009–2011) descreve refino de brinco: o guia de refino de 2009 cobre
armas/armaduras e o +10/+11, e **não** lista acessório. O que existe são **duas implementações de
servidor**, com regras incompatíveis e nenhuma relação com o build 7662:

| Servidor | Passos | Taxa | Falha |
|---|---|---|---|
| **WYD International** (patch 2025-02) | +9→+10 / +10→+11 | **15% / 10%** | **destrói o brinco**; aplica efeito fixo de absorção |
| **WYD Global** (Bahamut) | +0→+9 / +9→+11 | **40% / 5%** | **não destrói**: consome materiais e devolve 10% de chance de Bahamut Tear |

No 7662 isso é **UNVERIFIED**: não há regra do original para portar. No nosso banco virou dado
explícito, não constante: `accessory_refine_rule(kind, recipe, step_from, rate_bps, fail_mode,
fail_bonus_bps, gold_cost, material_name, material_qty, content_source)`, semeada pela
migration `0038` com as receitas do Bahamut (brinco, colar e cinto). O portal edita a taxa em
`/configuracoes`; a regra pura está em `crates/wyd-sim/src/formula/accessory.rs`.

**Amuleto, ankh e pedra espiritual** não têm regra publicada na era oficial nem no guia aberto
do Bahamut: a tabela aceita a linha, mas o seed não inventa passo nem taxa. O portal mostra o
acessório como “sem regra” até existir fonte — é a mesma política do UNVERIFIED.

---

## 8. Modelo de dado para o servidor

```
item_instance (
  id, item_index, owner_id, refine, ancient, celestial, quality,
  gem_effect,          -- diamond|emerald|coral|garnet|NULL
  repletion_option,    -- opção gerada pela pedra de repleção
  imbue_grade,         -- D-normal..legendary
  additional_a, additional_b,  -- os 2 adicionais internos
  bound_to             -- EF_NOTRADE -> personagem
)
refine_rate (step_from, step_to, base_rate, source)   -- semeado, auditável
refine_attempt (id, item_instance_id, npc, items_used, rng_roll, success, destroyed, at)
```

### Regras

1. **RNG auditável.** Gravar o `roll` e a taxa efetiva de cada tentativa. Sem isso, não há
   como responder reclamação de jogador nem detectar exploit.
2. **Sucesso consome tudo, numa transação.** Falha no meio do processo não pode deixar o
   jogador com o item antigo **e** o novo.
3. **Taxa vem do banco** (`refine_rate` / `game_setting`), nunca constante no core.
4. **`ancient` é irreversível e única**; `celestial` é estágio posterior.
5. **Pó de Ori/Lac são materiais** (`item_catalog.kind = 'material'`), com `primary_system =
   'refino'` — já classificados pelo gerador.

## 9. O que ainda não está provado

| Ponto | Status |
|---|---|
| taxas +0→+5 | **90/85/80/75/70%** (WYD International) — configurável |
| custo do +10 (Pedra do Sábio+50kk vs 30 Lac+5kk) | **divergência de servidor**, registrada |
| fórmula do refino sobre atributos (1,9× no +9) | **CONFIRMADO** como afirmação, sem tabela por peça |
| se absorção/perfuração acima de +10 escala linear | **SIM: 40 × (nível − 9)** → +10 = 40 … +15 = 240 |
| `+10 → +11` no legado | **15%** fixo em código (100% no Âmago) — divergente dos 3% publicados |
| taxa do **+12** | **5%** (guia da era oficial, 2009) vs **4%** (centrais atuais) — configurável, default 5% |
| bônus de **refinação abençoada** | **2…15%** (Kersef) vs **1…10%** (WYD Int) — configurável, default 2…15% |
| bônus do **Emblema da Proteção** | **+1%** (Kersef) vs **+10%** (WYD Int) — D-10 |
| chance de destruição com Pacote de Lac (+12..+15) | **20%** quando a taxa total fica **abaixo de 5%** (WYD Int) |
| refino de **brinco** | **não existe no original**; WYD Int 15/10% (destrói) vs WYD Global 5% (não destrói) — de servidor, fora do código |
