# Pesquisa de 2026-11-08 e decisões do servidor

Este arquivo fecha lacunas que estavam marcadas como `UNVERIFIED` e registra o que **nós**
decidimos, separando uma coisa da outra. Regra da casa: número de servidor privado é
**configuração**, nunca regra do jogo.

---

## 1. Drop — semântica dos campos (fechado)

| Dúvida | Resposta | Evidência |
|---|---|---|
| `DropItem` é multiplicador ou toggle? | **Nem um nem outro: é PESO.** O 2º argumento é comparado com o roll; o 4º é flag por item | `propDropEvent`/`propDropEvent.inc` de servidor: `DropItem(II_GEN_MAT_ORICHALCUM01, 14500000, 0, 1)`, `DropItem(..., 1000000000, 0, 1)` — valores muito acima de 1.000.000 mostram que **não é probabilidade absoluta** |
| O roll é por item ou por monstro? | **Por item, independente** | WYD International, *Drop System*: roll 1..1.000.000, cada item tem seu próprio roll; vários itens podem cair no mesmo kill |
| `double = 200` | **Percentual de EXP: 200 = 200% (2×)** | comando de GM `/gm +set double` é descrito como "ativar/desativar double exp" (fórum oficial de servidor) — é um multiplicador em %, não um toggle booleano |
| Taxa por item do build 7662 | **Não existe no cliente** | provado em `docs/cliente/tabelas-bin.md`: nenhuma tabela monstro→item nos `.bin`; o cliente só recebe o pacote de chão |

> **Consequência para o código:** `DropItem` entra como **peso** (`drop_weight`) e a taxa
> efetiva é `peso / soma_dos_pesos`; `double` entra como percentual (`exp_multiplier`,
> default 100). Nada de tratar `200` como boolean.

---

## 2. Refino (fechado onde há fonte oficial)

| Ponto | Valor | Fonte |
|---|---|---|
| +0 → +1 … +8 → +9 | **90, 85, 80, 75, 70, 65, 60, 55, 50%** | WYD International, *Server Information* (`rate1`…`rate9`) |
| +9 → +10 | 2 itens +9 iguais, **4 joias iguais**, **1 Pedra do Sábio**, **50kk**; sucesso ~30–40% | Kersef (EN e PT), wiki Fandom |
| Gema de efeito do +10 | **Diamante +8% drop**, **Esmeralda +40 perfuração**, **Coral +2% EXP**, **Garnet +40 absorção** | Kersef, *Refining +10* |
| +12 → +15 | 4 / 3 / 2 / 1% (oficial) — a wiki lista 15/14/13/12% | divergência de servidor, registrada |
| Falha com Pacote de Lac (+12..+15) | item pode ser destruído; se a taxa total ficar **abaixo de 5%**, chance de destruição **20%** | WYD International, *Refining 12→15* |

**Decisão nossa:** adotar a escada oficial 90…50 e o custo Pedra do Sábio + 4 joias + 50kk,
com `content_source` gravado por degrau. O efeito extra da joia é **definido na composição** e
**pode ser trocado depois** com a gema correspondente.

### As 4 gemas de efeito (fechado — já existem no catálogo)

| `item_catalog` | Item | Efeito que aplica |
|---|---|---|
| **3386** | Gema de Diamante | **+8% de drop** |
| **3387** | Gema de Esmeralda | **perfuração** de dano |
| **3388** | Gema de Coral | **+2% de EXP** |
| **3389** | Gema de Garnet | **absorção** de dano |

**Como se usa:** aplicada **sobre o item** (arrastar a gema), não em NPC. Vale para **Anct** e
para **+10 ou superior**. **Não destrói** o item e pode ser repetida. Vendidas no NPC **Lucy**,
em Azran.

**Escala (fecha a lacuna acima de +10):** perfuração e absorção são **40 × (nível − 9)** —
+10 = 40, +11 = 80, +12 = 120, +13 = 160, +14 = 200, +15 = 240. Drop (8%) e EXP (2%) são
**fixos**. **CONFIRMADO** em Kersef (*Premium Neil*), ABC do WYD e WYD Tutorial.

---

## 3. Imposto e castelo (fechado)

| Ponto | Valor | Fonte |
|---|---|---|
| Imposto das lojas pessoais | **10%** por cidade onde a loja está aberta | regra do dono do servidor, coerente com o total oficial |
| Fatia do castelo | **40% do imposto das 3 cidades** vai para a guild dona de Noatun | Kersef, *Siege Warfare*: "Castle of Noatun is entitled to 40% of the tax from the 3 towns" |
| Transações nos NPCs do castelo | também vão para o guild master | idem |
| Portal entre cidades | **700 gold** por uso | idem |
| Bônus de drop por doação | **+1% de drop por 1.000.000.000 gold** doado ao Priest Prona, teto **50%** | WYD Blues (sistema de guildas oficial) |

**Decisão nossa:** imposto por cidade = 10% de cada venda em loja pessoal; 40% do acumulado
das três cidades vai para Noatun; o resto fica na cidade. Rateio exato entre as cidades é
`game_setting` (`tax_share_city`, default 60/40).

---

## 4. Guerras e eventos — o que a fonte oficial diz

| Evento | Regra oficial | Observação |
|---|---|---|
| **Guerra de Cidades** | Aposta na **sexta**, guerra no **domingo 20:00–20:30**; **14 membros** por guild por cidade; aposta mínima **100 de fama** | cidades: Armia, Azran, Erion |
| **Guerra de Noatun** | **Quinzenal**, domingo **21:00–22:00**; só entra na sala do Altar quem tem cidade no servidor; vencer = dominar o Altar de Thor | Kersef + WYD Brazuca (2º round com troca de campeão) |
| **Kefra** | Respawn **fixo no sábado**; **+100 de fama** para a guild que matar | Kersef, *Guia: Guild* |
| **Sand Lich** | Nasce **1× por dia no servidor novato, 23:30** | idem |
| **Guerra de Reinos (RvR)** | Seg–sex **22:00–22:50** no servidor novato; sem perda de fama ao morrer | Kersef, *Guerra de Reinos* |
| **Arena Real** | Rodadas por faixa de nível/classe (15 min cada); guild com membro vencedor ganha *guild influence* | WYD International, *PVP Event Times* |

**Decisões nossas:**

- **Kefra: sábado, 14:00** (horário do servidor). Configurável em `game_setting`.
- **Noatun: 3 rounds** — round 1 e 2 normais e **round 3 de desempate** (2 rounds vencidos
  leva; empate vai ao terceiro). O oficial só publica "até 2"; os 3 rounds são nossos.
- **Calendário**: não existe calendário no build 7662; a agenda é nossa e vive em
  `game_setting` (dia + hora + duração por evento), nunca hardcoded.
- **Horário**: o servidor usa **UTC** internamente e converte por `server_timezone`
  (default `America/Sao_Paulo`). Isso responde a pergunta: hoje o horário é o do host, e
  implementar timezone explícito é justamente o que evita o evento "andar" quando o host
  muda de fuso ou entra em horário de verão.

---

## 5. Quests — o que fecha e o que é nosso

| Ponto | Situação |
|---|---|
| **Pista de Runas, salas 1–4** | requisito = **a pista refinada correspondente** (+0 na sala 1, +1 na sala 2, …) — decisão nossa, coerente com as pistas publicadas |
| **Castelo Zakun** | ligado ao **item Imp** e à cadeia de quests; não há layout público de propósito (é conteúdo de progressão) |
| **Karden** | quest, não mapa solto; entra na cadeia principal, não como ilha |
| **Story line completa do 7662** | **continua aberta**: só as faixas publicadas são conhecidas; o resto é cadeia a reconstruir com o cliente |
| **Recompensa de Desert Fury / Boss Fight no Deserto** | **customizável** — decisão pendente (ver perguntas) |
| **Texto de NPC de quest** | o 7662 tem pouquíssimo diálogo; o que existe está em mensagens do cliente, não em tabela de quest |

---

## 6. Skills → classe/linhagem (o que foi tentado e onde parou)

Estado real: **82 das 154 linhas continuam `unmapped`**. O seed `0019` já trazia 12 com
`rule:name_confirmed`; a curadoria fechou **57 novas** com fonte oficial (9ª e 10ª skill da
Kersef e listas por linhagem das páginas de classe), gravadas em
`crates/wyd-db/migrations/0021_seed_skill_lineage.sql` por
`tools/content/gen-skill-lineage.mjs` — cada linha cita fonte, `skill_role` e confiança. O
`down` é seletivo: restaura as 6 linhas herdadas do seed e só zera as 57 que a `0021` criou.
Cobertura: **69 das 151 do seed**; três nomes publicados pela Kersef não existiam no seed do
cliente (`Roubar_Dinheiro`, `Evocar_Lobo_Selvagem`, `Metamorfose_Superior`) e viraram linhas
de catálogo em `0024_seed_skill_catalog_kersef.sql`, com linhagem, fonte, confiança e índice
provisório declarado em `notes` — total **63 de 154**. O que não tem fonte pública por
linhagem segue `NULL`, não chutado.

O que foi provado nesta rodada:

1. `SkillData.bin` **não contém texto** (124 × 192 B, 21 colunas enum candidatas) — sem nome,
   não há como ligar registro a skill por leitura estática;
2. **os nomes das skills estão em `ItemList.bin`**, em texto claro após o XOR `0x5A`:
   **131 dos 151 nomes** foram localizados por casamento exato, com **263 ocorrências**
   começando no offset 136 (`Agressividade`, `Encantar_Gelo`, `Imunidade`, `Meditação`);
3. mas o arquivo **não é array fixo de 140 B na região de nomes**: o passo entre nomes é
   variável (140, 280, 420, 700, 980, 1120…, com nomes de até 13 bytes cruzando o que
   seria a borda do registro). A divisão exata `910.004 = 4 + 140 × 6.500` vale para o
   total, não para essa região — então o mapa registro→campos ainda não está provado.

**Conclusão honesta:** fechar skill→classe exige o struct do registro (Ghidra/IDA) ou trace
dinâmico com o cliente rodando — nenhum dos dois existe neste ambiente. O script
`tools/client/map-skills.mjs` fica como diagnóstico (status `diagnostico`), recusando casar
nome por aproximação: `Titã` não pode casar com `Tit`.

Caminho recomendado para fechar, em ordem de custo:

1. **Trace dirigido** (1 h de jogo): abrir a loja de skills de cada classe e logar
   `(classe, lista de skills)`; cruzar com os 151 nomes do seed → fecha a coluna `class_code`;
2. **Ghidra** no `WYD.exe` 7662: achar a tabela que a loja de skills percorre;
3. **Curadoria por linhagem**: usar as listas oficiais por linhagem (Kersef/Fandom) e marcar
   `classification_source` com `confidence` menor — é o que a migração `0020` já faz para 53
   linhas curadas; a mesma tabela aceita as demais sem mudar código. Regra do gerador: nome da
   fonte sem correspondência literal no seed é **reportado**, nunca aproximado
   (`Titã` ≠ `Tit`, `Roubar_Dinheiro` não existe no seed).

O que a curadoria já cobre por linhagem (53 linhas, fonte e confiança por linha):
TransKnight `faith`/`trans`/`magic_sword`; Foema `white_magic`/`black_magic`/`special_magic`;
Huntress `survival`/`trade`/`capture`; BeastMaster `elemental`/`evocation`/`nature`. O que
falta concentra-se nas **11ª e 12ª skills** e nas skills de transformação Arch, cuja lista por
linhagem não está publicada — daí ficarem `NULL`.

---

## 7. Bug conhecido do próprio manual

A tabela de `drops.md` ainda dizia que o `rate` por item "vem do binário". **Está errado** e
corrigido nesta rodada: não vem do cliente, vem da configuração do servidor.
