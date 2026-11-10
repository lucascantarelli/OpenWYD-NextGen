# Classes, linhagens e skills de topo

WYD não tem raças jogáveis: o personagem é humano e a identidade vem da **classe** e da
**linhagem de skill**. **CONFIRMADO** (wiki oficial das classes; central Kersef).

---

## 1. As quatro classes

| Classe | Papel | Armas | Skillmaster (Armia) | Atributo que manda |
|---|---|---|---|---|
| **TransKnight** | guardião do Altar dos Deuses; tanque ou dano físico | espada 1/2 mãos, machado, escudo | **Capitão Cavaleiros** | FOR/CON |
| **Foema** | maga elemental; dano mágico e controle | cajado | **Foema Anciã** | INT |
| **BeastMaster** | invocador e híbrido | cajado, arremesso, 1 mão | **Mestre Archi** | FOR/INT (híbrido) |
| **Huntress** | dano à distância e utilidade | arco, garra | **ForeLearner** | DES |

**CONFIRMADO** para nomes, papéis, armas e mestres (central Kersef, "(MAPA) ARMIA";
wiki das classes; WYD International).

### 1.1 Ficha inicial de um personagem novo

Fonte primária: `origin/main:Source/Code/Basedef.cpp` — `BaseSIDCHM`, `g_pIncrementHp/Mp`,
`BASE_GetHpMp`, `BASE_GetBonusScorePoint` e `BASE_GetBonusSkillPoint`; o porte atual vive em
`crates/wyd-server/src/characters.rs`. O nível inicial é 1 e `ClassMaster = MORTAL`
(`DBSrv/CFileDB.cpp`, `_MSG_DBCreateCharacter`).

| Classe | FOR | INT | DES | CON | HP base | MP base | HP/nível | MP/nível | HP nível 1 | MP nível 1 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| TransKnight | 8 | 4 | 7 | 6 | 80 | 45 | 3 | 1 | 83 | 46 |
| Foema | 5 | 8 | 5 | 5 | 60 | 65 | 1 | 3 | 61 | 68 |
| BeastMaster | 6 | 6 | 9 | 5 | 70 | 55 | 1 | 2 | 71 | 57 |
| Huntress | 8 | 9 | 13 | 6 | 75 | 60 | 2 | 1 | 77 | 61 |

Fórmulas no nível 1, sem equipamento:

- `MaxHp = BaseHP + (Con - BaseCon) × 2 + nível × HP/nível`; `MaxMp` idem com `Int` e MP.
- Atributos começam iguais à base da classe; 5 pontos de atributo e 3 de skill ficam livres
  (`BASE_GetBonusScorePoint` e `BASE_GetBonusSkillPoint` no MORTAL).
- HP/MP correntes começam cheios; EXP 0.
- Ouro, ponto de save e aparência/equipamento inicial **não** são default de schema: vêm do
  `BaseMob` da classe, copiado na criação como o `CFileDB.cpp` fazia
  (`memcpy(mob, &g_pBaseSet[cls], sizeof(STRUCT_MOB))`), com `MortalFace = Equip[0].sIndex`,
  `memset(ShortSkill, -1, 16)` e o nome do pacote por cima.

### 1.2 `BaseMob` — arquivos reais e valores iniciais

Fonte: `origin/main:tmserver/content/DBsrv/run/BaseMob/{TK,FM,BM,HT}` — 816 B cada
(`sizeof(STRUCT_MOB)`), embutidos no servidor por `include_bytes!` a partir de
`crates/wyd-server/assets/basemob/`. O layout é o do `Basedef.h` (alinhamento natural do MSVC
x86, **não** `pack(1)`): offsets e prova byte a byte em `crates/wyd-server/src/personagens.rs`.

| Arquivo | Classe | SHA-256 | Ouro (`Coin@28`) | Save (`SPX/SPY@40`) | Rosto (`Equip[0]`) | `Equip[1..6]` | `Carry[0..3]` |
|---|---|---|---:|---|---|---|---|
| `TK` | 0 TransKnight | `19d0d0eb6c02d5212b7dcfed7ed238d37aa1b356d6451363349b0a5aae805586` | 5.000.000 | 2096/2096 | 1 | 1103, 1115, 1127, 1139, 1151, 861 | 401, 406, 4130, 4144 |
| `FM` | 1 Foema | `13284d101b249066999516c5add3f6f8b8a3d580cb7b5c5f04dc6d4eadaa7b78` | 5.000.000 | 2096/2096 | 11 | 1251, 1265, 1277, 1289, 1301, 816 | 401, 406, 4130, 4144 |
| `BM` | 2 BeastMaster | `d0e8d128388ead208636d996974b081dd19daf072d03c0c36c17cb10b84522ec` | 5.000.000 | 2096/2096 | 21 | 1403, 1406, 1409, 1412, 1415, 861 | 401, 406, 4130, 4144 |
| `HT` | 3 Huntress | `4e7411dc8830e6393db900b25b5b826b8219d91a0bf94122b5cf9dd6b180bede` | 5.000.000 | 2096/2096 | 31 | 1553, 1556, 1559, 1562, 1565, 816 | 401, 406, 4130, 4144 |

`Equip[0].sIndex` é o rosto (aparência) e é **também** como o cliente descobre a classe na tela
de seleção: `STRUCT_SELCHAR` não tem campo `Class` (`DBGetSelChar`, `CFileDB.cpp:2636`).
`Equip[0].stEffect` do BeastMaster traz `0x63/0x62/0x15` (cor de cabelo/pele), por isso o dump
do slot 0 não é só o índice.

O template também traz `Ac=4`, `Damage=5`, `Merchant=0` (TK/HT) ou `16` (FM/BM) e
`AttackRun=82` nos dois scores; `Direction`/`ChaosRate` chegam `0xCC` do dump de memória do
motor antigo e o porte Go **mantém** esses bytes no login (o cliente passa a usar a direção do
primeiro `_MSG_Action`). `Special[4]`, `LearnedSkill`, `Magic`, `Critical`, `SkillBar` e
`Resist` também vêm do template (zerados, exceto `Critical=0xFF`, `SkillBar=0xFF` e
`Resist=-11`).

O que o Rust **sobrescreve** por cima do template: nome, `Class`, `Coin`/`Exp` (do banco),
`SPX/SPY` (ponto de save persistido, igual ao `sm.PosX = sm.mob.SPX` do legado), `Level`,
`MaxHp/MaxMp/Hp/Mp`, `Str/Int/Dex/Con` (`BaseSIDCHM` + incrementos) e `ScoreBonus`/`SkillBonus`.

**Inventário persistido (entrega de 2026-11-10).** A criação copia o `Equip`/`Carry` do
template para `character_items` (slot `i` = `Equip[i]`, `16 + i` = `Carry[i]`) e grava o kill
mark — item 547 em `Carry[KILL_MARK=63]`, o mesmo que o `ProcessDBMessage.cpp` repõe no login —
e o `ShortSkill[16]` em `character_short_skill`. Daí em diante **o banco é a fonte da
verdade**: `STRUCT_MOB` e `STRUCT_SELCHAR` são montados de `character_items`
(`Mob::from_persisted`) e o template só é relido para personagem anterior a esta entrega, uma
única vez, para materializar o inventário inicial. Cofre (`vault_items` → `Cargo[128]`) e
`Coin` da conta (`account_wallet`) vêm do banco; sem linha, slots vazios e zero — nunca item
ou valor sintético.

O `STRUCT_ITEM` do fio leva `sIndex = item_id`; `stEffect[0]` recebe `EF_AMOUNT` quando a
pilha tem mais de uma unidade (é o `3d 78` = 120 do 401/406 no template) e o item 547 recebe
`EF_CURKILL`/`EF_LTOTKILL`/`EF_HTOTKILL`. Refino, Ancient e gema continuam estado da linha e
ainda não têm codificação de fio (FASE-10): no `BaseMob`, `Equip[1..6]` só tem `sIndex`.

Lacunas registradas: o `STRUCT_MOBEXTRA.MortalFace` é derivável de `Equip[0]` e não ganhou
coluna; a customização de rosto/cabelo (que altera `Equip[0].stEffect`) ainda não existe e usa
o rosto padrão da classe.

**Distribuição de pontos (`_MSG_ApplyBonus`, 0x0277).** Os **5 pontos de atributo** (tipo 0) e
os **3 de skill** (tipo 2) são gastos com o personagem em jogo, e o servidor é a autoridade: a
quantidade vem de `character_scores` (`score_bonus`/`skill_bonus`), nunca do pacote. `Con`/`Int`
somam 2 de `MaxHp`/`MaxMp` por ponto (`_MSG_ApplyBonus.cpp`) — o mesmo que o `BASE_GetHpMp` do
login deriva de `(Con − base) × 2`, então banco e pacote continuam idênticos. A skill é
`5000 + classe × 24 + pos` e custa o `skill_catalog.skill_point`; as posições 7/15/23 são linha
única por classe, exigem as 7 anteriores e cobram 50 000 000 de ouro; nível e maestria vêm de
`item_catalog.req_*`. Progressão e skill aprendida vão para `character_scores`/`characters` +
`character_learned_skill` numa transação, e o cliente recebe `_MSG_UpdateEtc` + `_MSG_UpdateScore`
(`docs/arquitetura/protocolo.md` §5.2).

**Maestria (tipo 1).** `SpecialBonus` é gasto em `Special[0..3]` (Str/Int/Dex/Con) com dois
tetos do legado: `3 × (nível + 1) ÷ 2` por atributo e 200 — 255 quando a skill de linha
correspondente já foi aprendida (`Detail` 1/2/3 → bits 7/15/23). Fica em `character_special`
e volta no `STRUCT_MOB` (`BaseScore`/`CurrentScore.Special`). O `Special[4]` do
`_MSG_UpdateScore` é a sentinela `0xCC` do `SendScore`, não a maestria. **Fora:** o ganho de
pontos por nível (`CMob.cpp:1128/1142`, +2) e o bônus celestial (`+3 × 400` no limite) —
`ClassMaster` CELESTIAL/SCELESTIAL ainda não existe nesta stack; até lá os pontos entram por
concessão, com o mesmo caminho de validação e gravação.

Cada classe tem **três linhagens**, cada linhagem tem **oito skills base + 9ª, 10ª, 11ª e
12ª**, e um personagem pode ter **24 skills** no total. **CONFIRMADO**.

---

## 2. Linhagens — mapa completo

### TransKnight

| Linhagem | Como joga | Skills conhecidas |
|---|---|---|
| **Faith** (Confiança) | tanque sagrado: invulnerabilidade, cura, área | Heaven's Dust, Holy Touch, Luster Furnish, Aura of Life, Attract, Destiny |
| **Trans** | dano físico direto, buffs de ataque, duas armas | Charge, Two Hand Master, Deadly Stroke, Assault, Rapid Piercing, Rescue |
| **Magic Sword** (Espada Mágica) | híbrido arma + magia, congelamento e controle | Blaze Launcher, Ice Storm (8ª) |

**9ª/10ª/11ª/12ª — CONFIRMADO** (central Kersef, "9th Skill", "10th Skill", "12th skills"):

| Linhagem | 9ª | 10ª | 12ª |
|---|---|---|---|
| Faith | **Proteção Divina** (buff: invulnerável por tempo + amplifica a cura de *Aura de Vida* em 10% dos pontos de INT) | **Benção Divina** (passiva: HP regenera 100% ao usar a 9ª) | **Clod Attack** (passiva: acertar *Destino* pode reduzir a evasão do oponente em 10% por 64 s) |
| Trans | **Mãos Sangrentas / Blood** (passiva: com duas armas de 1 mão, crítico pode regenerar HP) | **Mestre das Armas** (passiva: aumenta crítico) | **Wind Storm** (passiva: remove a penalidade de HP e maximiza ataque ao buffar *Assalto*) |
| Magic Sword | **Espelho Mágico** (passiva: aumenta chance de acertar ataque mágico) | **Conexão de Gelo** (passiva: a 8ª — *Tempestade de Gelo* — pode congelar) | **Seal Master** (passiva: melhora a lentidão após congelar) |

### Foema

| Linhagem | Como joga | Skills conhecidas |
|---|---|---|
| **Magia Branca** | cura, proteção, invulnerabilidade | Divine Blessing |
| **Magia Negra** | dano elemental em área, debuffs | Freezing Skillful, Flash |
| **Magia Especial** | utilidade: remover buff, trocar HP por MP | Incapacitador |

| Linhagem | 10ª | 12ª |
|---|---|---|
| Branca | **Proteção Absoluta** — escudo de gelo, **invulnerável por 64 s**; usar qualquer outra skill desativa | **Holy Power** (passiva: dobra a duração do debuff de *Flash*, −10% evasão por 64 s) |
| Negra | **Congelamento Proficiente** (passiva: skills de gelo podem prender o oponente no lugar) | **Thunder Lising** (passiva: +5% de ataque mágico) |
| Especial | **Incapacitador** — remove todos os buffs positivos do oponente | **Another Change** — troca HP por MP e recupera 2000 HP/s |

### BeastMaster

| Linhagem | Como joga | Skills conhecidas |
|---|---|---|
| **Elemental** | dano elemental + escudo antimagia | Kiss of Salamander, Mana Burn, Fury of Undine, Anti-Magic Shield (9ª), Resi Decrease |
| **Evocação** | invoca criaturas que lutam por você | Condor Summon, Wild Boar Summon, FinalVerid |
| **Natureza** | reflexão, veneno, sobrevivência | Ambush |

| Linhagem | 9ª | 10ª | 12ª |
|---|---|---|---|
| Elemental | **Anti-Magia** (escudo que permite esquivar de magias; duração cresce com os pontos na linhagem) | **Chamas Resistentes** (proteção de fogo com chance de invocar fera flamejante contra o oponente mais próximo) | **Manaburn Master** (aumenta a queima de mana de *Chamas Etéreas*) |
| Evocação | **Unidade Mental** (passiva: evocações mais poderosas) | **Invocação Final** — invoca o **FinalVerid** | **Ice Armor** (passiva: aumenta a defesa) |
| Natureza | **Concha Resistente** (passiva: reduz a chance de receber crítico) | **Espinhos Fortalecidos** (passiva: chance de refletir o dano recebido) | — não confirmada |

### Huntress

| Linhagem | Como joga | Skills conhecidas |
|---|---|---|
| **Sobrevivência** | dano com arco/garra, esquiva, crítico | Fatal Attack, Two Weapon Fighting, Enchant Frost, Immunity, Transfer Armor Class, Piercing Attack, Rapid Hit |
| **Troca (Trade)** | craft, joias, comércio, dinheiro | Weapon Mastery, Craft Armor Ability, Rob Money |
| **Captura (Capture)** | laços, controle e captura de criaturas | Skill Master Book |

| Linhagem | 9ª | 10ª | 12ª |
|---|---|---|---|
| Sobrevivência | **Contra-Ataque** (passiva: depois de esquivar, o próximo ataque é maior) | **Claw Skillful** (passiva: +DES e +30% de dano crítico com garra) | **Freezing** |
| Troca | **Ponto do Mestre** (passiva) | — não confirmada | **Reinforce Soul** |
| Captura | — não confirmada | — não confirmada | — não confirmada |

> As lacunas acima são **lacunas reais de fonte pública**, não preguiça: a central Kersef
> publica as três linhagens de cada classe, mas os artigos de 9ª/10ª/12ª são truncados em
> algumas seções. Registrado como **UNVERIFIED** e sem invenção. O `skill_catalog` já traz
> o comportamento (mana, delay, alcance, efeito) — o que falta é só o **nome**.

---

## 3. Regras de skill que o servidor precisa respeitar

Todas **CONFIRMADO** (central Kersef, "Skills" e "Skills Usage"):

| Regra | Valor |
|---|---|
| Skills por personagem | **24** |
| Skills por linhagem (base) | **8** |
| Teto de maestria (pontos por linhagem) | **200** base |
| Teto com 8ª skill ou item de maestria | **255** |
| Teto com 9ª skill | **320** |
| Pontos de maestria ganhos por nível | **2** |
| Limite por nível | nível × 1,5 |
| Custo da 9ª | **255 pontos de skill** |
| Nível mínimo da 9ª | **201**, personagem Celestial/Subcelestial |
| Local da 9ª | NPC **9thMaster** (X:2136 Y:2115), Armia |
| 10ª | skills anteriores + **4.000 fama** |
| 11ª | nível **281+**, **1.000 fama**, livro da skill |
| 12ª | Celestial **> 320**, 11ª da mesma linhagem, **12.000 fama**, NPC **Jazeg** (Armia) |

Regras adicionais:

- A 9ª só pode ser aprendida **na mesma linhagem da 8ª**; destravar usa **Círculo Divino
  Composto**. **CONFIRMADO**.
- Reset de linhagem: **Círculo Divino**, **Benção de Deus** e a quest **Balanço do Poder**.
  **CONFIRMADO**.
- **Aprender arma** também soma dano físico; pontos de maestria e de arma são ajustáveis
  conforme a build. **CONFIRMADO**.
- Skill é comprada **somente** no Skillmaster da classe. **CONFIRMADO**.

### Implicação de balanceamento

O teto de maestria é o verdadeiro limitador de build: com 320 pontos e 24 skills, o jogador
**não** consegue maximizar tudo. Um personagem de PvP distribui em 2–3 linhagens; um de
farm concentra em área. O servidor deve tratar `skill_mastery` como estado persistido por
linhagem, não como derivado do nível.

---

## 4. Arch — a classe transcendente

**CONFIRMADO** (central Kersef, "Arch"; guias de Arch).

| Item | Regra |
|---|---|
| Pré-requisitos | completar **"A procura da vida eterna"** e **"Um novo começo"** |
| Aparência | segue a classe inicial |
| Classe Arch | escolhe **uma das quatro** classes Arch |
| Skills | vêm das escolhidas na quest; evoluem com o personagem principal |
| Elmo/capacete | **não usa** |
| Defesa | usa a da classe inicial |
| Armas | **não** equipa armas de classe E nem Lydians; equipa armas **Arch** |
| Bônus no nível 370 | **112** pontos de aprendizado, **288** de atributos, **168** de skill |
| Bônus por nível | +**8** pontos de atributo por nível evoluído do personagem normal |

Quests de criação (itens): **Pedra da Imortalidade** e **Sephirot** equipados, Mortal **nível 300+**.
**CONFIRMADO** (dois guias de 2009). A **capa Red/Blue não é item de criação**: é conquista do
Mortal (borda grossa no **355**, com Selo do Guerreiro). O **400** é o requisito do **Celestial**
e a recomendação para maximizar pontos — ver dossiê §2.1 (D-3).

> O Arch é o único ponto do jogo em que **duas entidades de personagem** coexistem: o Mortal
> e o Arch com o mesmo nome. No banco isso vira duas linhas ligadas por `arch_of_id`, e a
> sincronização é uma regra de servidor — não um trigger.

---

## 5. O que cada classe faz melhor

| Objetivo | Melhor escolha | Por quê |
|---|---|---|
| Farm em área | Foema (Magia Negra) | dano em área com poucos pontos |
| Boss / dano sustentado | TransKnight (Trans) | crítico + duas armas + regeneração |
| PvP de duelo | Huntress (Sobrevivência) | esquiva, contra-ataque, crítico com garra |
| Guerra / castelo | Foema (Branca) + TransKnight (Faith) | invulnerabilidade e cura em massa |
| Economia / craft | Huntress (Troca) | craft, joias, roubo de dinheiro |
| Farm de mob único | BeastMaster (Evocação) | invocação tanka enquanto você bate |

**UNVERIFIED** como "meta" — é leitura de ferramenta por skill confirmada, não tier list de
servidor específico. Cada servidor balanceia diferente; o que não muda é **qual skill existe
em qual linhagem**.

---

## 6. O que o banco já tem e o que falta

`skill_catalog` (**154 linhas**: 151 do seed do cliente + 3 da Kersef que o cliente não trazia —
`Roubar_Dinheiro`, `Evocar_Lobo_Selvagem`, `Metamorfose_Superior`, em
`0024_seed_skill_catalog_kersef.sql`, com índice provisório declarado em `notes`) guarda:
`skill_index`, `name`, `skill_point`, `target_type`,
`mana_spent`, `delay_ticks`, `range_tiles`, `instance_type/value`, `tick_type/value`,
`affect_type/value/time`, `instance_attribute`, `tick_attribute`, `aggressive`,
`max_target`, `party`, `affect_resist`, `passive`.

Ou seja: **comportamento está no dado**. `class_code`, `lineage_code` e `skill_role`
(`damage`, `heal`, `buff`, `debuff`, `summon`, `mobility`, `passive`, `craft`) estão preenchidos
em **69 das 154** linhas — as outras continuam `NULL` com `classification_source = 'unmapped'`.

Regra de preenchimento: só por **nome confirmado** nas fontes acima; nome fora da lista fica
`NULL` com `classification_source = 'unmapped'`. Nunca chute — um `skill_role` errado faz o
bot de teste tomar decisão errada e o bug aparece como "IA burra", não como dado ruim.

A cobertura subiu de 60 para **69** com a tabela de **12ª skill por linhagem** (10 das 12
linhagens; Natureza/BeastMaster e Captura/Huntress continuam "não confirmada" na fonte) e a 9ª
de Espada Mágica (`Espelho_Mágico`). `Freezing` e `Reinforce_Soul` entram com `skill_role`
**NULL**: a linhagem está confirmada, o efeito não está descrito — preencher seria dedução.

> **Conflitos declarados (seed × fonte oficial).** `Clod_Attack` está no seed como
> `magic_sword`, mas a tabela oficial o coloca em **Faith**; a curadoria `0021` (fonte
> `kersef:12th`, confiança 90) prevalece porque roda depois do seed. `Another_Change` vinha como
> `passive` e entra como `buff` (troca HP por MP). Ambos aparecem no relatório do gerador como
> "já mapeadas (repetidas)" — a divergência é visível, não silenciosa.
>
> **Down seletivo (2026-11-08).** Reverter só a `0021` devolve as 6 linhas que a `0019` já
> classificava ao estado do seed (`Bênção_Divina`, `Chama_Resistente`, `Carga`, `Assalto`,
> `Clod_Attack`, `Another_Change` — inclusive o conflito de linhagem, que volta ao valor do
> seed) e as 57 novas a `unmapped`. Os valores restaurados saem da própria `0019`, lidos pelo
> gerador, então o `down` não carrega cópia mantida à mão.
