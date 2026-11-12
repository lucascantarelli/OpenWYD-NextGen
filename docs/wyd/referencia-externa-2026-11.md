# Referência externa (pesquisa 2026-11-09) — o que o WYD Global e os emuladores ensinam

Pesquisa feita para a mega revisão do roadmap. **Nada aqui é regra do nosso servidor** até ser
promovido com fonte e teste: o nível de confiança segue `fontes.md` (A oficial · B oficial de
servidor · C privado com guia · D comunidade · E dado do repositório). O que é **número de
servidor** entra como `game_setting`, nunca como constante.

## 1. Personagem e progressão

**B — central Kersef (RaidHut/WYD Global).** As quatro classes e os quatro atributos
(FOR/INT/DES/CON) estão confirmados; atribuição depende da build.
<https://feedback.kersef.com/en/knowledge-bases/21/articles/58186-beginners-guide>

Tabela de pontos por nível (guia do Arch, **B**, potencialmente antiga):

| Faixa | Aprendizado | Status | Skill |
|---|---:|---:|---:|
| 1–200 | 2 | 6 | 4 |
| 201–354 | 3 | 6 | 4 |
| 355 | 3 | 12 | 2 |
| 356+ | 1 | 12 | 2 |

HP/MP por nível: TK 4/1, Foema 1/3, BeastMaster 1/4, Huntress 2/2; defesa 2. Arch 370:
112 aprendizado, 288 atributos, 168 skill; +8 atributos por nível evoluído do personagem normal.
<https://feedback.kersef.com/en/knowledge-bases/21/articles/20384-arch>

**Impacto no nosso dado:** `docs/wyd/progressao.md` documenta *status* e *aprendizado*, mas
**não tem a coluna de pontos de skill** — e o `skill_catalog` já guarda `skill_point` por skill.
Isso é uma lacuna de documentação (não de código): ver pergunta **P-1**.

**Limite de nível:** o build 7662 usa **400 por camada** com transformação; servidores privados
anunciam 800/1500. Não misturar (já registrado em `fontes.md` §5).

## 2. Skills

**B — Kersef.** Três linhagens por classe, oito skills por linhagem (24 base); maestria 200,
255 com a 8ª/item, 320 com a 9ª; 2 pontos de maestria por nível e limite nível × 1,5; 9ª custa
255 pontos e exige nível 201; 10ª exige 4.000 de fama; 11ª exige nível 281 + 1.000 de fama;
12ª exige Celestial > 320 + 12.000 de fama. **Confirma exatamente** `docs/wyd/skills.md` e
`classes.md`.
<https://feedback.kersef.com/en/knowledge-bases/21/articles/20050-skills-usage>
<https://feedback.kersef.com/en/knowledge-bases/21/articles/59032-9th-skill>
<https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/26951-10th-skill>
<https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/50023-12th-skills>

**Lacuna que continua:** lista completa de MP/cooldown/alcance por skill **não é pública e
versionada**. A fonte prática é o `SkillData.csv` do cliente da versão (o projeto WYD2Bot lê
esse arquivo, mas é cliente não oficial — **D**). O nosso caminho correto continua sendo o
`skill_catalog` (dado E) + captura do 7662.

## 3. Itens, refino e composição

**B — Kersef.**
- Grades N/M/A/Le + Anct + E; cada grade acima de N soma ~8% de atributo (**D** repetido em
  vários guias), sets E/Arch não pedem atributo.
- +10: **dois itens +9 iguais e do mesmo tipo + quatro joias iguais + Pedra do Sábio +
  gold** (o guia EN diz 50 milhões; o PT do mesmo site diz 5 milhões — divergência de versão).
  <https://feedback.kersef.com/en/knowledge-bases/21/articles/58872-refining-10-and-changing-additionals>
- Joia define o adicional: Diamante +8% drop · Esmeralda +40 perfuração · Coral +2% EXP ·
  Garnet +40 absorção. **Já é exatamente o nosso `decisoes-servidor.md` §2.**
- +12 → +15: **4% · 3% · 2% · 1%** base; cada pedra secreta soma 1 ponto percentual; falha pode
  destruir material e reduzir 1 nível.
  <https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/58750-refining-system-12-to-15>
- Ancient: arma +9 + joia + reforços; falha pode consumir tudo.
- Combinação God: adicionais Ancient (drop/XP/absorção) **não** passam para a peça God.
  <https://feedback.kersef.com/en/knowledge-bases/21/articles/20247-god-class-item-combination>

**Não existe** tabela pública, única e versionada de *todas* as opções de set, kind codes e
receitas de craft. O que existe é receita pontual por servidor. Isso reforça a regra do repo:
`item_catalog` é a verdade do **nosso** servidor; guia externo só calibra.

## 4. Cidadania, canais e guerras

**B — Kersef / RaidHut.**
- Cidadania é registrada por NPC (Kibita) e é **do canal**; reino, guilda e canal são conceitos
  distintos. <https://feedback.kersef.com/en/knowledge-bases/21/articles/20375-mapa-armia>
- O lançamento anunciou **8 canais**; guias atuais citam canais 1–3 para Guerra de Torres e 5/6
  para eventos — confirma arquitetura multicanal, **não** confirma que hoje sejam 8.
  <https://wydglobal.raidhut.com.br/en/wydlaunch>
- Guilda pode disputar vários canais, mas mantém zona de guilda em **um canal por vez**; guerra
  de cidade aos domingos; Noatun quinzenal.
  <https://wydglobal.raidhut.com/en/2120>
- Castelo de Noatun recebe **40% do imposto das três cidades**; portal entre cidades custa
  **700 gold** (50% ao dono do castelo). <https://feedback.kersef.com/en/knowledge-bases/21/articles/20258-siege-warfare>

**Impacto:** nosso plano de **2 canais** (1 normal + 1 verde) continua válido como decisão de
operação; o limite de 10 canais por grupo é do formato `serverlist.bin`, não do jogo. A regra
“uma guilda não domina o mesmo tipo de zona em dois canais” é **estado por canal** — o banco já
tem `guild_tower`/`guild_relations` com canal (0028).

## 5. Economia

- **B:** Premium Neil vende itens de NP; **Wyden Coin = 200 NP**.
  <https://feedback.kersef.com/en/knowledge-bases/21/articles/22192-premium-neil-prisang-neil>
- **B (evento):** Silver Coin associada a 1 milhão de gold — notícia pontual, não catálogo.
- **B:** Guild Coins são moeda de guilda, **não** moeda premium.
- **Não verificado:** preço de NPC vigente por item, percentual atual de imposto por cidade,
  drop de gold por monstro. Continuam fora do código como regra.

## 6. Protocolo e estruturas

- **Fonte técnica independente** (WYD2EncDec) documenta header de 12 bytes
  (`size:u16, key:u8, hash:u8, code:i16, index:i16, timestamp:u32`) e identifica `0x0FDE`
  como tentativa/resposta correta de PIN e `0x0FDF` como incorreta — **coerente** com o nosso
  `crates/wyd-protocol` e a `docs/wyd/senha-numerica.md`.
  <https://github.com/raphaelts3/wyd2encdec>
- **W2PP (`Basedef.h`)** define `ESCENE_FIELD = 30000`, `STRUCT_SELCHAR` e `STRUCT_MOB` com os
  campos que já usamos. **Não é contrato universal**: layout C depende de pack/compilador/versão.
  <https://github.com/ErickAlcan/W2PP/blob/master/Code/Basedef.h>
- **`INITCODE = 0x1F11F311` não aparece em fonte pública**; no nosso repo ele é **medido** no
  cliente 7662 e provado por teste — continua sendo dado **E**, não “verdade da internet”.

## 7. Emuladores e fóruns (para o que servem)

| Fonte | Serve para | Cuidado |
|---|---|---|
| [w2pp-OpenWYD](https://github.com/Jean1dev/w2pp-OpenWYD) | reescrita Go do servidor, alvo 7662/protocolo 7640 | engenharia reversa, estado de implementação variável |
| [WYD-Server-CPP / W2PP](https://github.com/Amatsukan/WYD-Server-CPP) | source C++ (DBSrv/TMSrv) | o próprio README diz “para estudo” |
| [snalmir](https://github.com/venantivs/snalmir) | emulador aberto do cliente 7.54 | versão diferente da nossa |
| [WYD2Bot](https://github.com/kevinkouketsu/WYD2Bot) | leitura de `ItemList.csv`/`SkillData.csv` | cliente não oficial, não é servidor |
| [RaGEZONE WYD2](https://forum.ragezone.com/community/wyd2.302) | releases, clientes, tutoriais | conteúdo comunitário; avaliar versão e procedência |
| [WYD ItemList Editor](https://github.com/Rechdan/WYD-ItemList-Editor) | confirma `ItemList.bin`, sem semântica completa | não define kind codes |
| [RaidHut droplist](https://global-droplist.raidhut.com/) | “o que cai onde” no Global | reflete balanceamento atual do Global |

## 8. O que esta pesquisa **muda** no plano (e o que não muda)

**Não muda:** nenhuma constante, nenhuma fórmula, nenhum layout. `docs/wyd/` continua sendo a
fonte de mecânica; `item_catalog`/`skill_catalog`/`game_setting` continuam sendo a verdade do
nosso servidor. Nenhuma issue nova nasceu desta pesquisa — ela **confirma** o que já estava
documentado e isola o que é número de servidor.

**Muda (documentação):**
1. `progressao.md` passa a citar a coluna de **pontos de skill** por faixa como lacuna (P-1).
2. `fontes.md` ganha as fontes técnicas de código (§2) e o aviso de que `INITCODE` é medido, não
   publicado.
3. Registro explícito de que a lista completa skill × MP × delay **não é pública**: o caminho é
   `SkillData`/captura, não curadoria por nome.

## 9. Perguntas abertas para o dono (não bloqueiam código)

| # | Pergunta | Por que importa | Default seguro hoje |
|---|---|---|---|
| P-1 | Pontos de **skill** por nível entram junto de status/aprendizado na progressão? | a tabela oficial sugere 4/nível (1–200); não temos coluna equivalente no doc | manter `skill_point` por skill no catálogo e não derivar de nível |
| P-2 | +10 custa **50kk** (EN) ou **5kk** (PT) no nosso servidor? | é o maior dreno de ouro do jogo | 50kk em `decisoes-servidor.md` §2 |
| P-3 | Queremos **8 canais** como o Global ou os 2 do plano (normal + verde)? | custo por canal e guerra entre canais | 2 canais; crescer é linha em `server_channel` |
| P-4 | Bônus de cidadania: usamos +1%/1bi de imposto até 50% (Global) ou valor fixo? | afeta drop e guerra | 0 até haver número aprovado (`game_setting`) |
| P-5 | Limite de nível: 400 por camada com transformação, ou servidor custom? | define `exp_table` e teto de skill | 400 por camada |
| P-6 | O Arch precisa das 4 pedras + Sephirot já no primeiro corte, ou entra depois do jogo base? | escopo de progressão | depois do mundo visível (GAME-09) |
| P-7 | Aceitamos `STRUCT_SELCHAR` de fonte W2PP como layout provisório (marcado UNVERIFIED)? | destrava a tela de personagens antes da captura | sim, com golden e marca UNVERIFIED |
| P-8 | A loja de donate/NP entra no escopo do servidor ou fica só no portal? | mexe em economia e invariante 7 | só portal (fila), nunca estado vivo |

## 10. Como reproduzir esta pesquisa

As URLs acima foram lidas em 2026-11-09. Guias do Kersef têm versões PT e EN do **mesmo** artigo
que divergem em custo; ao usar qualquer número, registrar **URL + data + servidor**. Regra do
repo: mecânica é estável, número não (`docs/wyd/fontes.md` §1).
