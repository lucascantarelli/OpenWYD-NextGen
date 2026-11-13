# Pesquisa de internet — dossiê de mecânicas (2026-11-10)

Segunda rodada de pesquisa externa do projeto, feita **depois** de `referencia-externa-2026-11.md`
(2026-11-09) e com foco no que o manual de domínio (`docs/wyd/`) ainda **não** tinha com número,
fonte e/ou classificação de confiança.

**Nada aqui é regra do nosso servidor por existir neste arquivo.** A promoção para regra segue o
rito normal: fonte registrada → `game_setting`/catálogo com `content_source` → teste. Nível de
confiança em `fontes.md` §1 (A oficial · B oficial de servidor · C privado com guia · D comunidade
· E dado do repositório).

## 0. Como ler este dossiê (passe de auditoria)

A primeira versão deste arquivo foi montada a partir de **resultados de busca**. Numa auditoria no
mesmo dia, **cada URL citada foi aberta** e o texto foi conferido contra o conteúdo da página. O
resultado mudou números, removeu afirmações e derrubou uma "divergência" que era erro de leitura.

Três consequências para quem usar este documento:

1. **Só sustenta número o que está em §7.1** — fontes que foram abertas e cujo texto contém o dado.
   As que apareciam apenas como snippet saíram do texto e estão listadas em §7.2, com o motivo.
2. **Cada linha do §2 diz de que tipo de fonte ela vive.** `[B]` = central oficial de servidor;
   `[C]` = servidor privado que publica as próprias regras; `[D]` = blog/comunidade. Quando a linha
   depende de **uma única** fonte, ela diz `fonte única` na própria linha.
3. **Nenhum link foi citado sem leitura.** Se um número não está na página aberta, ele não está aqui.

**Escopo.** Páginas públicas em PT-BR, EN e ES sobre: evolução (Arch/Celestial/Subcelestial), Sephira
Soul, refino/composição, PK/Chaos, guildas/guerras, montarias, acessórios, mapas e acessos,
chat/comandos, cash/NP. **Não** foram investigados: catálogo de exploits (§6 do `wyd2.md`), engenharia
reversa do cliente 7662 e arquitetura de bots — ver §6.

---

## 1. Como este dossiê difere do manual

O manual já descreve **mecânica** com boa cobertura (refino, drop, guerras, quests, mundo, itens).
Esta rodada acrescenta três coisas que faltavam:

| Tipo de achado | Onde ele entra |
|---|---|
| **Número novo** com fonte (ex.: pontos bônus do Celestial por nível do Arch) | §2, com nível de confiança |
| **Divergência entre fontes** do mesmo sistema (ex.: Emblema da Proteção em 1% ou 10%) | §3 — registrada e resolvida por evidência, ou mantida aberta |
| **Domínio sem documento próprio** (montarias, chat, cash) | §2.6, §2.9, §2.10 |

Onde o achado **confirma** o manual, ele não é repetido: só a referência aparece na matriz (§4).

---

## 2. Achados por domínio

### 2.1 Evolução — Arch, Celestial e Subcelestial

**Arch** — o nível de criação aparecia em três valores diferentes; as páginas abertas mostram que são **coisas distintas**:

| Rota | Requisitos | Fonte |
|---|---|---|
| **Original (300+)** | Mortal **nível 300 ou superior**; **Pedra da Imortalidade** e **Sephirot** da classe **equipados**; falar com o **Rei**. Os pontos de status do Arch **crescem com o nível do Mortal** (528 no 300) | `[D]` [WYDGalahan, 2009](https://wydgalahan.wordpress.com/guia-de-jogo/guia-avancado/criando-personagem-arch) · [WarLokos, 2009](http://warlokos.blogspot.com/2009/12/arch-criacao.html) |
| **Com capa (355)** | Mortal **355**; capa azul/vermelha **com borda grossa** (Selo do Guerreiro, disponível no 355); Pedra + Sephirot; **100kk** | `[C]` [WYD Brazuca, guia 99](https://wydbrazuca.com.br/guias-do-wyd/99-Evoluindo-para-Arch) |
| **400** | nível do **Mortal** exigido para o **Celestial** (Arch 356+); recomendado para maximizar pontos, mas **não** é gate de criação do Arch | `[B]` [Kersef 22279](https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/22279-criacao-do-celestial) · `[D]` WarLokos |


Ou seja: nos guias da **era oficial**, o 355 **não** é o nível de criação do Arch — é o nível da
**borda grossa da capa**
(Selo do Guerreiro) e do **destrave de XP do Arch** no NPC **Lindy** (24 Poeiras de Lac + 1
Pergaminho Selado); o **370** tem outro destrave. O **400** é o nível do **Mortal** para o
**Celestial**, e serve de recomendação para maximizar pontos — guias modernos (wydtutoriais, 2017)
resumem a cadeia inteira como "Mortal 400", e foi daí que veio a confusão registrada em D-3. A rota
"355 + capa com borda" do WYD Brazuca é **config de servidor privado**, que amarra a criação do Arch
à capa em vez do nível.

Composição dos dois itens de entrada (consistente entre as páginas abertas):

| Item | Como se obtém |
|---|---|
| **Pedra da Imortalidade** | **Alma do Unicórnio** + **Alma da Fênix** + **10 safiras** entregues ao **Oráculo** atrás do Rei; as almas precisam estar **lado a lado e viradas uma para a outra** (Unicórnio à esquerda, Fênix à direita) |
| Almas | Fênix: **Lich Crunt**, **Lich Bazama**, **Verid Amald**, **Sombra Negra**; Unicórnio: **Lugefer Broken** |
| **Sephirot** | **8 pedras de Sephira** no inventário + **30.000.000 de ouro**, compostas no **Skill Master da classe desejada** — é aí que a classe do Arch é escolhida |

> **Variação de versão, não de mecânica:** a página do Wyd DoL (5.0) atribui as almas a
> **Duel Letter** e a bosses de LAN, enquanto o tutorial 2.0 lista os bosses de campo acima. A
> *regra* (2 almas + 10 safiras no Oráculo) é a mesma; a *origem* do drop muda por servidor/versão.

**Celestial** — números que o manual não tinha (`[B]` [Kersef 22279](https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/22279-criacao-do-celestial);
tabela replicada em `[D]` [Point dos XD](https://pointdosxd.blogspot.com/2011/06/mega-tutorial-celestial-e-sub-celestial.html) e [ABC do WYD](http://abcdowyd.blogspot.com/2010/06/criacao-do-personagem-celestial.html)):

| Item | Regra |
|---|---|
| Requisito | Mortal **400** + Arch **356** (mínimo); quanto maior o nível do Arch, mais forte o Celestial |
| Itens | **Sephirot** da classe **equipado** + **Pedra da Imortalidade** ("Eternal Stone") **equipada** + **4 pedras secretas no inventário** (Sol, Terra, Vento, Água) |
| NPC | **Rei do reino** → gera a **Pedra Ideal**; usar a pedra transforma o Arch |
| Restrição | a composição tem de ser feita **por um Arch** — feita por um Mortal, gera **outro Arch**, não um Celestial |
| Efeitos | capa muda (Mestre dos Aventureiros), **skills resetadas**, **EXP resetada**, aprende a Sephira **Limite da Alma (Soul)**, nasce no **nível 1** e precisa destravar **40** e **90** |
| Cythera (tiara) | **prateada / dourada / mística**, conforme nível do Arch e a quest Arch **"Os 4 Cristais"** |

As pedras secretas, no WYD Int, saem de **combinações de runas** compostas no NPC **Odin**, em Kefra
(`[C]`, guia 23) — detalhe de servidor; o original documentado só exige as 4 pedras no inventário.

**Cythera — o que cada faixa entrega** (`[D]`, ABC do WYD; o Kersef só publica as três cores):

| Cythera | Nível do Arch na criação | Opções |
|---|---|---|
| Prateada | **356–380** | Defesa **80**, economia de mana **10%**, dano perfuração **30** |
| Dourada | **381–390** | Defesa **90**, mana **10%**, perfuração **30**, absorção **30** |
| Mística | **400** | Defesa **110**, mana **10%**, perfuração **40**, absorção **40** |

**Pontos bônus de status na criação** — a tabela que faltava (`progressao.md` cita a existência do
bônus, não os valores). `[D]` fonte única dupla: ABC do WYD e Point dos XD publicam a mesma tabela,
mas **nenhuma central oficial aberta** a publica:

| Origem | Pontos |
|---|---|
| Base fixa do Celestial | **1001** |
| Arch **356–370** | **+100** |
| Arch **371–380** | **+300** |
| Arch **381–398** | **+600** |
| Arch **399** | **+900** |
| Arch **400** | **+1200** |
| Quest "Contrato com os Espíritos" fase **1 / 2 / 3 / 4** | **+100 / +200 / +300 / +400** |
| **Teto** de pontos de status na criação | **2600** |
| Aprendizado / Skill na criação | **855 / 1500** |

**Subcelestial** (`[D]` fonte única — Point dos XD; o WYD Int tem guia próprio, não aberto nesta rodada):

| Item | Regra |
|---|---|
| Requisito | **Celestial nível 120+** e **100 de fama** |
| Itens | **Sephirot** da classe desejada + **Pedra Ideal** |
| Procedimento | equipar o Sephirot e usar a Pedra Ideal |
| Observação | Celestial e Subcelestial **ocupam o mesmo slot** — não nasce personagem novo |

> Confirma o desenho do `progressao.md` §1 (Subcelestial é camada do Arch, não quinta classe) e
> acrescenta o gatilho numérico. **Não** temos fonte aberta para o custo/efeito do God2 — UNVERIFIED.

### 2.2 Sephira Soul (Limite da Alma)

O manual só citava a existência; este é o funcionamento (`[B]` [Kersef 22279](https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/22279-criacao-do-celestial)
para combinações e NPC; `[C/D]` [worldwyd](https://worldwyd.webnode.com.br/wyd/skill-limite-da-alma)
— que declara fonte "WYD BR" — para requisitos e percentuais):

| Item | Regra |
|---|---|
| Desbloqueio | **Mortal nível 371+**, **8ª skill** aprendida e **1 pedra secreta da classe**: Huntress = **Vento**, BeastMaster = **Terra**, TransKnight = **Água**, Foema = **Sol** |
| Confirmação | falar com o NPC **Kibita** (saída de Armia, canal da cidadania) — o personagem volta com a capa **Manto do Herói +0** e a Soul nos slots Sephira |
| Configuração | NPC **Ehre**, em **Erion**; combinação de **3 jóias** define os atributos |
| Efeito | a skill **aumenta pontos de atributo** (não dá dano direto) |

Combinações publicadas (idênticas em mortal e celestial):

| Atributos | Combinação |
|---|---|
| FOR/CON | Diamante ×3 |
| INT/CON | Esmeralda ×3 |
| DES/CON | Coral ×3 |
| FOR/DES | Garnet ×3 |
| INT/DES | Diamante + Esmeralda + Coral |
| CON/DES | Diamante + Coral + Garnet |
| FOR | Esmeralda + Coral + Garnet |
| INT | Esmeralda + Diamante + Coral |
| CON | Coral + Esmeralda + Garnet |
| DES | Garnet + Diamante + Coral |

Percentual aplicado — **`[D]` fonte única**: só o worldwyd (combinado com o `fontes.md` §2) publica
estes números; o Kersef não os publica. Fica como **UNVERIFIED** até captura ou segunda fonte:

| Personagem | 1 atributo | 2 atributos |
|---|---|---|
| Mortal | **+70%** | **+40%** no primeiro, **+30%** no segundo |
| Celestial | **+120%** | **+80%** no primeiro, **+40%** no segundo |

**12ª skill que mexe na Soul** (`[B]` [Kersef, 12th skills](https://feedback.kersef.com/en/knowledge-bases/21/articles/50023-12th-skills)):
Huntress / Exchange → **Reinforce Soul** (passiva) aumenta o bônus da Soul em **+20%**; se a Soul
estiver em dois atributos, **+20% em cada**. Custo de aprender a 12ª: **12.000 de fama** (reduzido a
**1.200** no período de teste), no NPC **Jazeg**, em Armia; exige nível **320+** celestial e a 11ª
skill da mesma linhagem.

> **Correção importante da auditoria:** a linha "20% em cada" **não** é percentual da Soul — é o
> bônus da passiva de 12ª skill. A versão anterior deste dossiê tratou as duas coisas como
> divergentes; ver D-4.

### 2.3 Combate: velocidade, golpe duplo/triplo, crítico e evasão

**O item mais relevante da rodada**, porque a `GAME-20` está **bloqueada** por falta de causa
provada da anomalia de velocidade (`docs/rust/06-matematica.md` §3: `formula/speed.rs` ⛔).

Fonte: **`[C/D]`** — [FreeWyd, velocidade de ataque](https://pathoflords.com/mecanicas/velocidade-de-ataque)
e [FreeWyd, evasão](https://pathoflords.com/mecanicas/evasao) (servidor privado que publica a
própria matemática e **declara que lista o que muda em relação ao original**). Serve como
**hipótese testável**, nunca como regra pronta.

| Regra publicada | Valor |
|---|---|
| Velocidade base | **100** (% na janela de personagem) |
| Ganho por Destreza | **+1 a cada 5 de DES** (conta DES total: base + item + buff + alma) |
| Golpe duplo | começa em **110** com **10%**, sobe **10% a cada 10 pontos**; **100% em 200** |
| Golpe triplo | começa **acima de 300**, ganha **1% a cada 2 pontos**; **100% em 500** |
| Sorteio | duplo e triplo saem do **mesmo** sorteio: o triplo **substitui** o duplo (não soma) |
| Dano médio | `×1` até 100; `×2` em 200; `×2,5` em 400; `×3` em 500; acima de 500 não sobe |
| Crítico | sorteado **à parte**, **dobra** o golpe por cima de duplo/triplo → triplo crítico ≈ **×6** |
| Lentidão | skills de gelo tiram **30**; "Lâmina Congelante" tira **nível/10 + 10** |

**Evasão** (`[C/D]`, mesma fonte, página própria — aberta):

| Regra | Valor |
|---|---|
| Base | **0%** se o atacante é tão rápido ou mais rápido; cresce com a **diferença** de velocidade |
| Curva | **+1,2% a cada +10** de vantagem; **+25% em +200**; **+47,8% em +500** |
| Teto | **80%**, alcançado com **+2.140** de vantagem |
| Modificadores | itens de evasão **somam** (e a evasão do atacante **subtrai**, como precisão); buffs/montaria (**0 a +8%**) só valem se já existe base |
| Penalidades | atacante **TK** reduz **30 pontos percentuais**; contra **Tempestade de Raios** e **Exterminar** sobra só **30%** da esquiva |

> **Por que isso importa para a GAME-20:** a hipótese do projeto é que a "anomalia de ~200%" não é
> bug de leitura, e sim **duplo golpe 100% garantido em 200 de velocidade**. A tabela explica
> numericamente o DPS dobrado e o ×3 acima de 500. Ela **não prova** o 7662 — prova que existe um
> modelo coerente. O caminho de prova continua sendo captura/golden, como a issue exige.
>
> **Armadilha:** "velocidade de ataque" no cliente é o **mesmo número** que dá esquiva. Tratar as
> duas como coisas independentes produz dois erros ao mesmo tempo (dano alto sem esquiva, ou o
> contrário).

Dado de servidor correlato (**`[C]`** [WYD Int, changelog 2026-09](https://wyd-international.com/news/143)):
resistência passou a escalar **linearmente de 0 a 250**, com teto de **40%** em 250. **Antes**, acima
de 100 não contava — 250 dava a mesma redução que 100. Vale para dano de skill elemental (lentidão,
congelamento, veneno, fogo, raio), **não** para ataque normal. É tuning de servidor, não fórmula do
original.

### 2.4 PK — Chaos Points e C.Point (fecha a lacuna do `guerras.md` §6)

`docs/wyd/sistemas/guerras.md` §6 descreve o PK **qualitativamente** e lista "pontos altos mudam a
penalidade" sem número. Estes são os números (`[A/B]`
[Kersef, Chaos](https://feedback.kersef.com/en/knowledge-bases/21/articles/20250-chaos-system),
[Kersef, C.point](https://feedback.kersef.com/en/knowledge-bases/21/articles/20249-cpoint-system);
`[C]` [WYD Int, guia do Chaos](https://wyd.international/guides/46/chaos-system-pk-system) e
[patch de CP](https://wyd-international.com/news/38)):

| Item | Regra |
|---|---|
| Faixa de CP | **−75 … +75**; personagem novo começa em **0** |
| Dedução por kill | **2 a 3 pontos** (Kersef), dependendo do CP da vítima; WYD Int: **3–4** com capa branca, **2–3** sem |
| Ganho por nível | **+3** (1–350), **+1** (351–370), **+5** (371–400) |
| Lista negra | abaixo de **−65** o nome muda de cor (Kersef e WYD Int); o **PK fica bloqueado em −75 ou menos** (WYD Int) |
| Matar listado | **não** dá penalidade de caos |
| Iniciar PK | o autor fica **temporariamente listado por 1 minuto** (pode ser atacado sem penalidade) |
| Exceções | **guerra de guilda**, **zona PK das cidades** e **Hall do Kefra** (WYD Int: sem redução de CP lá dentro) |
| Comando | `/cp` mostra o CP atual |

> **Implementado no servidor (2026-11-11):** `characters.cp` + `wyd-sim/chaos` (faixa, cor,
> `penalidade_de_kill` 2/3 e piso de PK `-75`), byte do fio `cp + 75` no `CreateMob`,
> `UpdateScore` e `STRUCT_SELCHAR`, e abate PvP no motor com PvP ligado pelo papel do canal.
> C.Point, Kill Points e o `/cp` continuam pendentes.
| Cores | `[C]` **fonte única** (guia 46): < −65 rosa/vermelho; −50…−65 vermelho; −25…−49 laranja; 0…−24 amarelo; 1…24 verde; 25…49 azul; 50…74 branco; 75+ branco. O Kersef só diz que a cor **muda gradualmente** |

**C.Point (EXP retida ao morrer em PvP)** — sistema separado do CP:

| Item | Regra |
|---|---|
| O que é | ao morrer em PvP a EXP **não é perdida**: fica **retida** como C.Point |
| Ganho | **+5.000** por morte em PvP (Kersef, valor fixo) |
| Ganho (variante) | `[C]` WYD Int usa faixas: **5.000 / 15.000 / 25.000** conforme o CP |
| Quitação | o C.Point **desce** ao ganhar EXP de monstro; em **0** a barra volta a render |
| Teto | **1/10 da EXP** necessária para o próximo nível |
| Penalidade | em **80%** do teto, **dano / defesa / mapa (−50%)** |
| Exceção | sem penalidade ao descer de nível na faixa **350–355** |

> **Ambiguidade da fonte oficial:** o artigo do Kersef diz, em pontos diferentes, que o C.Point
> "diminui ao ganhar EXP de monstros" e que "matar monstros não afeta o C. Point". Lido com o guia
> do WYD Int (que diz "reduz ao ganhar EXP; matar não **aumenta**"), a leitura compatível é:
> caçar **reduz**, matar **não aumenta**. O texto oficial sozinho não fecha — por isso a regra fica
> em `game_setting`, não em código.

**Kill Points** (`[C]`, guia 46 + patch): matar jogador do **reino oposto** dá **+1 Kill Point** (não
conta em guerra nem em zona PK de cidade); **100 Kill Points** trocam por **10 CP + 10 de fama** no
NPC do governo.

> **Regra de ouro que sai daqui:** EXP retida ≠ EXP perdida. Se o nosso servidor "perder EXP" na
> morte em PvP, o comportamento diverge do original em **dois** eixos (valor e reversibilidade).

### 2.5 Guildas, torres, Noatun e RvR — números que faltavam

**`[B]`** — [Kersef, Guia Guild](https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/57606-guia-guild),
[Kersef, Siege](https://feedback.kersef.com/en/knowledge-bases/21/articles/20258-siege-warfare),
[Kersef, Guerra de Reinos](https://feedback.kersef.com/en/knowledge-bases/21/articles/21552-guia-guerra-de-reinos),
[WYD Global, guild/war](https://wydglobal.raidhut.com/en/2120); `[C]` [Wyd DoL, comandos](https://wyddol.com/command-list)
e [WYD Int, Server Info](https://wyd-international.com/information) para o custo de criação:

| Item | Regra |
|---|---|
| Criar guilda | **100.000.000** de ouro (`/create nome`) — confirmado em duas páginas abertas |
| Nível de guilda 1 | **1.000 de influência** (desbloqueia "Intel Report") |
| Guerra de Torres | **segunda a sexta, 21:00–21:30** (Kersef); o WYD Global publica **21:30, 15 min, canais 1–3**, com inscrição 30 min antes — ver D-9 |
| Fama de guilda | **teto 200** |
| Noatun | **domingo, 21:00–22:00**, **a cada duas semanas**; +**200** de fama de guilda na **segunda** ao dono |
| Manutenção de cidade | dono precisa de **400 de influência**; abaixo disso **perde o território**. O custo é **por território** (o WYD Global exemplifica 600 para dois) |
| Portal entre cidades | **700 de ouro** por uso, **50%** para o dono do castelo |
| Cerco | aviso às **18:00**, portões fecham **18:05**, vitória da defesa às **19:55**; PK fora do stockade não mexe no caos, mas a morte soma C.Point |
| Guerra de cidade | **domingo, 20:00–20:30**; aposta no **sábado**, exigindo **100 de fama**; **26 × 26** (Armia/Azran); Erion só aceita mortais |
| RvR — horário | **22:00–22:50** (a mesma página também publica "todo sábado 22:00" — ver D-6) |
| RvR — entrada | guilda com **torre** + fama alta; **26** da guilda + **13** do reino |
| RvR — vitória | Rei rival morto **ou** 22:50; vencedor leva **160 pontos de safira**; guilda **+100** de fama; perdedor **−100** |
| RvR — invencibilidade | **8 minutos** ao entrar; **+1 de fama** por kill de reino oposto; **sem CP**; morrer **não** perde fama |
| RvR — bloqueio | não se entra nos **últimos 5 minutos** (22:45) |
| Pontuação de guerra | **nível − 1** por participante; **God2 soma +400 fixos** (nível 400 → 799 pontos) |

> **Duas divergências de calendário, não uma:** (a) o RvR aparece como "seg–sex 22:00–22:50" e como
> "todo sábado 22:00" **na mesma página oficial** (D-6); (b) a guerra de torres aparece como
> **21:00–21:30** no Kersef e **21:30, 15 min** no Global, com requisito de influência **400** contra
> **1.000** (D-9). Agenda e influência têm de ser `game_setting` por canal/servidor.

### 2.6 Montarias (domínio sem documento próprio)

Não existe `docs/wyd/sistemas/montarias.md`; o índice só lista `montaria` como sistema do catálogo.
Funcionamento consolidado de **`[B]`** ([Kersef, Mount System](https://feedback.kersef.com/en/knowledge-bases/21/articles/20121-mount-system),
[WYD Global, guia de montarias](https://wydglobal.raidhut.com/pt-br/2874)) e **`[C]`**
([WYD Int, Mount System](https://wyd-international.com/guides/34/mount-system)):

| Item | Regra |
|---|---|
| Variedade | **8 espécies** de animal + **10 raças** de cavalo montáveis |
| Ciclo | **Ovo → Cria → Montaria**: a cria nasce nível 1 e vira montaria adulta no **nível 100** |
| Fome | `feed` cai **2–4** por hora; em **0** a montaria morre de fome e perde vida |
| Vida | nasce com **10–30**; cada renascimento tira **0–3** (aleatório); recuperar custa ouro no mestre de montaria (Erion) |
| Bônus | ao montar: **ataque mágico, esquiva, defesa e absorção**; **absorção fixa em 25%**; os bônus **não** mudam com o nível, exceto **dano** e **ataque mágico** |
| Evolução | essências refinadas: **+0 a +9** para nível **0–179**; a partir de **180**, usa **+9/+10** |
| Essência +10 | **100% de sucesso**, mas é **exclusiva de montaria nível 180+** |
| Falha de nível | em nível **par**, a falha **perde um nível**; em nível **ímpar**, **permanece** |
| Trava | nível **181+** fica **imóvel e não-negociável**; NPC **MountKeeper** + item **Restore** volta ao 180 |
| Qualidade | ativa no **nível 200** (valor aleatório **1–5**); **sobe até 70**; cada ponto = **+1%** sobre o **valor base** |

> Observação de modelagem: "qualidade" incide **sobre o valor base** (exemplo publicado: 2530 →
> 4301 com 70 pontos), não sobre o total. É a diferença entre aditivo e composto — e ela aparece no
> balanceamento de endgame.

### 2.7 Acessórios, Pedra Espiritual e Repleção

**`[D]`** — [Revolution WYD, acessórios](http://revolution-wyd.blogspot.com/2009/09/acessorios.html)
(que declara copiar o site oficial da época), **`[B]`** [Kersef, Repletion](https://feedback.kersef.com/en/knowledge-bases/21/articles/20120-repletion-system):

| Item | Nível | Efeito |
|---|---|---|
| Anel de Hércules | 35 | Dano **+10** |
| Anel de Titã | 35 | HP **+10%** |
| Anel de Zeus | 35 | Velocidade de ataque **+10%** |
| Anel de Athena | 37 | MP **+10%** |
| Anel de Gaia | 37 | Regeneração HP/MP **+3%** |
| Anel de Hecate | 37 | INT **+8** |
| Bracelete de Hércules | 89 | Dano **+20** |
| Bracelete de Athena | 89 | MP **+13%** |
| Bracelete de Zeus | 90 | Velocidade de ataque **+15%** |
| Pingente de Hércules | 116 | Dano **+35** |
| Brinco de Zeus | 160 | Velocidade de ataque **+21%** |
| Brinco de Hércules | 152 | Dano **+45** |
| Pedra Espiritual | 256 | **+10 perfuração** e **+100 HP**, ou **+10 absorção** e **+100 HP** |
| Pedra Espiritual A/B/C | 256 | upgrade (Arch **356+** com a quest de evolução): perfuração **15** e/ou absorção **15**, mantendo **+100 HP** |

**Repleção** (troca de adicional de armadura): pedra vendida nas lojas de **Armia, Azran e Erion**,
nas **grades A–C**; exige a peça **no inventário** (não equipada, não no cofre); **escudo, arma e
acessório não** podem ser repletados; o resultado é **aleatório**, pode voltar opção "não refinada"
e **remover refinos anteriores**; quanto **menor** o nível do item, maior a chance de bom resultado.
Confirma o que `itens.md` já diz sobre grades e amplia a regra de uso — que é o ponto onde servidor
privado costuma divergir.

### 2.8 Mapas e acessos que faltavam no `mundo.md`

**`[B]`** — [Kersef, Pergaminhos da Água](https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/22545-pergaminhos-da-agua),
[Kersef, Dragon Field](https://feedback.kersef.com/en/knowledge-bases/21/articles/49455-novo-mapa-dragon-field),
[Kersef, mapa de Armia](https://feedback.kersef.com/en/knowledge-bases/21/articles/20375-mapa-armia),
[Kersef, Dungeon Kefra](https://feedback.kersef.com/en/knowledge-bases/21/articles/21703-dungeon-kefra):

| Local / acesso | Regra |
|---|---|
| **Caverna da Água** | entrada em **Azran (X:1965 Y:1771)**, portal abaixo da fonte; pergaminhos **N/M** (mortal e arch) e **A** (só celestial); **8 andares** por pergaminho; o pergaminho ganha +1 nível ao limpar o andar; se o tempo acabar, **perde o pergaminho**; **grupo de no mínimo 8 jogadores** |
| **Dragon Field** | NPC **Gold Dragon** em Azran; exige **Dragon Guardian equipado**, composto no **Dragon Elder** a partir de **Dragon Soul** (chance de falha; item entregue equipado e **não pode ser reequipado** se desequipar) |
| Dragon Field — chefes | Parte 1: **Legend Verid** → **Mark of Archangel** (abre a parte 2 e também compõe livros de 12ª skill); Parte 2: **Nesses** e **Haden** — recompensa **proporcional ao dano**, com dano mínimo para contar; celestial **360** pode destravar nível ali |
| **Dungeon Kefra** | Hall exige **Pergaminho Selado** entregue ao NPC **Sobrevivente (2371, 3887)** — cada pergaminho dá **100 entradas**; o boss aparece **1× por semana em todos os canais**, mas o **portal só abre no canal novato**; **sem restrição de nível ou classe**; o último golpe leva **Pedra da Imortalidade, os 4 Sephirots, Poeira de Lac e a skill Força Espectral a 100%** |
| **Armia — NPCs** | **Premium Neil** (itens cash), **Kibita** (cidadania por canal), **Metre Grifo** (teleporte para dungeons/quests), **Dragão Dourado** (atributos na arma com os 7 cristais), **Mestre de Habilidade** (reconfigura status por **30 safiras**), **Guarda Carga** ×2 |

> Nada disso vira constante: **coordenada é dado de mundo** (`fontes.md` §5, item 4). O que vale
> como mecânica é a **regra de acesso** — "o Dragon Guardian não pode ser reequipado", "o portal do
> Kefra só abre no canal novato", "a Água exige 8 no grupo". As coordenadas de NPC de refino que
> apareciam na versão anterior vinham de uma página que **não foi aberta** e saíram (§7.2).

### 2.9 Chat, comandos e Trombeta

**`[A/B]`** — [Kersef, Chatting](https://feedback.kersef.com/en/knowledge-bases/21/articles/20086-chatting);
**`[C]`** [WYD Int, Server Info](https://wyd-international.com/information),
[Wyd DoL, comandos](https://wyddol.com/command-list),
[Origem Destiny, comandos](https://origemdestiny.com.br/blog/guia-de-comandos-ingame):

| Prefixo | Alcance |
|---|---|
| `Enter` | local (ao redor do personagem) |
| `/nome msg` | sussurro (privado); com `!` vira carta |
| `/r msg` | responde ao último sussurro |
| `= msg` | grupo |
| `- msg` | guilda; **sem guilda** cai no global do canal |
| `-- msg` | aliança de guildas |
| `@ msg` | **cidadãos do canal** |
| `/spk msg` | **broadcast do servidor** (chat verde) — exige o item de gritaria/**trombeta** (`[C]`, Wyd DoL + Origem Destiny) |

Comandos relevantes (todos nas páginas abertas): `/cp` (caos), `/tab msg` (etiqueta sobre o nick,
**nível 70+**), `/create nome` (**100kk**), `/handover` (**100kk**), `/guildtax` (**0–25%**, só o
Guild Master; é a guilda dona de Noatun que taxa o comércio do servidor), `/summonguild`, `/guild`,
`/war`, `/allystop`, `/expel`, `/getout`, `/kingdom` e `/king` (exigem **pergaminho de portal**),
`/relo`, `/info`, `/event`.

> **Trombeta é mecânica, não item cosmético:** o servidor publica em **todos os canais ativos**, o
> que no nosso desenho é um fan-out de chat entre processos de canal — precisa de contrato
> explícito, não de boost por canal. **O que as páginas abertas não dizem:** se o item é consumido.
> Fica como P-14.

### 2.10 Cash, NP e entrega premium

**`[B]`** — [Kersef, Premium Neil](https://feedback.kersef.com/en/knowledge-bases/21/articles/22192-premium-neil-prisang-neil),
[WYD Global, inventário virtual](https://wydglobal.raidhut.com/pt-br/3536):

| Item | Regra |
|---|---|
| **NP / Cash** | moeda premium; NPC **Premium Neil** em todas as cidades, ao lado do **Guarda Carga**, e **exige a senha** para abrir |
| Organização | loja por **abas com nome de cidade** |
| **Wyden Coin** | vale **200 NP** |
| **Virtual Capsule** | **gratuita**, comprada no NPC **Aki**; move **cosmético** (traje/skin de montaria) para o **inventário virtual** no site; é **imóvel** |
| Inventário virtual | **120 slots gratuitos** por conta; resgate gratuito e ilimitado, com **10 min** de espera entre resgates; **puramente cosmético** |
| Entrega | itens de loja/cash caem **no inventário**; se o inventário virtual estiver cheio, skin e cápsula vão para o **Guarda-Carga** |

> **Correção da auditoria:** a versão anterior dizia "Virtual Capsule = 200 NP" e "Shop Points 100 =
> 100 NP". A página aberta do WYD Global diz que a cápsula é **gratuita**; a outra afirmação vinha
> de snippet de uma loja em subdomínio `preprod`, que **não abriu** — as duas saíram (§7.2).
>
> **Lacuna de rastreabilidade:** nenhuma fonte aberta descreve um **ID de origem por item** (o que o
> `wyd2.md` §4.25 chama de "identificação de itens"). O mais próximo é a marcação de **não-negociável
> / imóvel** (montaria 181+, cápsula). Nosso caminho: `item_instance.origin` (tabela nossa), sem
> alegar que o original fazia isso.

---

## 3. Divergências registradas (resolvidas por evidência ou mantidas em aberto)

| # | Tema | Versão A | Versão B | Default seguro |
|---|---|---|---|---|
| D-1 | Refino **+12** | **5%** — guia da era oficial (WYDGalahan, 2009), mesma composição no Odin; +13/+14/+15 = 3/2/1% | **4%** — centrais atuais (Kersef EN **e** PT, WYD Int, 2022–2025), com os mesmos 3/2/1% depois | **resolvido como época, não erro**: default do 7662 em **5%**, 4% como variante; seed atual ainda 4% (mig. 0022) |
| D-2 | Bônus de **refinação abençoada** por unidade (+0…+9) | **2/3/4/5/6/7/8/10/12/15%** — Kersef (EN e PT) e `itens.md`; 4×+9 = **60%** | **1/1/2/3/4/5/6/7/8/10%** — WYD Int (Server Information e guia 47); 4×+9 = **40%** | default **2…15%** (oficial licenciada, coerente com `itens.md`); seed atual ainda 1…10 (`BLESSING_BONUS_BPS`) |
| D-3 | **Arch** — nível de criação | **300** mínimo na era oficial, com pontos por nível do Mortal (WYDGalahan e WarLokos, 2009 — abertas) | **355** + capa com borda + 100kk (WYD Brazuca — aberta); **400** é o nível do **Celestial** (Kersef 22279 — aberta) | **resolvido como erro de categoria**: 300 é criação, 355 é capa/destrave, 400 é Celestial; `progressao.md` e `classes.md` alinhados; escolha do gate segue em P-11 |
| D-4 | **"Soul 20% em cada"** | **falso positivo** — a frase é da passiva **Reinforce Soul** (12ª skill), não do percentual da Soul (Kersef 50023 — aberta) | percentual da Soul: **40/30** e **80/40** (worldwyd — aberta, fonte única) | não aplicar percentual até decidir (P-9); a passiva de +20% é mecânica separada |
| D-5 | **Lista negra** de PK | **< −65** muda a cor do nome (Kersef 20250; WYD Int 46 — abertas) | **−75 ou menos** bloqueia o PK (WYD Int 46 e patch 38 — abertas) | são **dois limiares diferentes**, não duas versões: −65 pinta, −75 trava |
| D-6 | **RvR** dia/hora | **"seg–sex 22:00–22:50"** (Kersef 21552 — aberta) | **"todo sábado 22:00"** — na **mesma página**; `guerras.md` também registra sábado | agenda por servidor em `game_setting`; a página oficial não é suficiente sozinha |
| D-7 | **Kefra** spawn | **1×/semana, todos os canais, portal só no canal novato** (Kersef 21703 — aberta) | **"respawn fixo no sábado"** (Kersef 57606 — aberta) | `game_setting`; a parte estável é "portal só no canal novato" |
| D-8 | Refino de **brinco** | **15%** (+9→+10) e **10%** (+10→+11), **destrói na falha** (WYD Int, patch 38 — aberta) | **40%** (+0→+9) e **5%** (+9→+11), **não destrói** — consome materiais e devolve Bahamut Tear em 10% (WYD Global, acessórios Bahamut — aberta) | **de servidor**: nenhuma fonte da era oficial documenta refino de brinco; no 7662 fica UNVERIFIED e fora do código |
| D-9 | **Guerra de torres / influência** | **seg–sex 21:00–21:30**; manutenção com **400** de influência (Kersef 57606 — aberta) | **21:30, 15 min, canais 1–3**; nível 1 = **1.000** de influência e exemplo de **600** para dois territórios (WYD Global 2120 — aberta) | agenda e custo em `game_setting`; não hardcodar |
| D-10 | Bônus do **Emblema da Proteção** | **+1%** por emblema (Kersef 58750 — aberta) | **+10%** por emblema (WYD Int 47 — aberta) | `game_setting`; default **1%** (oficial licenciada) |

> **D-4 é o achado mais importante deste passe.** Uma linha de artigo EN ("increases 20% on each")
> foi lida como percentual da Soul e virou "divergência". A página aberta mostra que ela descreve a
> **12ª skill**. Fica registrada como falso positivo para não ser reintroduzida por quem reler o
> histórico.

> **D-1 e D-2 fechados em seguida, com as fontes abertas.** A versão EN do artigo do Kersef **não**
> publica a tabela de 1%: ela repete **2…15%**, igual à PT. A tabela de **1…10%** é do **WYD
> International** (`Server Information` e guia 47) — outro servidor. D-2 nunca foi divergência de
> tradução; é de servidor, e cada tabela fecha com o próprio exemplo (4×+9 = **60%** no Kersef,
> **40%** no WYD Int). No **+12**, o 5% não vem de wiki: vem do **guia da era oficial de 2009**
> (WYDGalahan), com a mesma composição no Odin; a wiki Fandom que repetia 5% respondeu **403** e
> **não** foi usada. No `refino.md` o **default documentado** passou a **5%** e **2…15%** (alvo 7662), com **4%** e
> **1…10%** como variantes; o seed atual (`migrations/0022`, `BLESSING_BONUS_BPS`) **ainda** grava
> 4% e 1…10, e a troca ficou como P-10. De quebra apareceu **D-10**: o Emblema da Proteção vale
> **+1%** no Kersef e **+10%** no WYD Int.

> **D-3 e D-8 fechados no mesmo dia.** Em D-3, o "400 vs 355" era **erro de categoria**: os guias
> da era oficial (2009) criam o Arch no **300**, o **355** é o nível da **borda grossa da capa** e do
> **destrave de XP** do Arch, e o **400** é o nível do Mortal para o **Celestial**. A rota
> "355 + capa + 100kk" é config de servidor privado (WYD Brazuca). Em D-8, o WYD Int **não é a
> única** fonte: o WYD Global publica refino de brinco próprio (Bahamut: **40%** até +9, **5%** de
> +9 a +11, **sem destruir**), incompatível com os **15%/10% com destruição** do WYD Int. Nenhuma
> página da era oficial descreve refino de brinco — logo D-8 é **de servidor**, não lacuna de
> leitura, e no 7662 fica UNVERIFIED.

---

## 4. Matriz de rastreabilidade (requisito → onde está)

| Requisito do `wyd2.md` | Documento | Estado após esta rodada |
|---|---|---|
| 4.1 atributos e pontos | `classes.md` §1.1–1.2, `progressao.md` §2 | ✅ documentado; derivados por equipamento/skill/buff **parcial** |
| 4.2 combate (velocidade, crítico, evasão) | **este dossiê §2.3** | ⛔ bloqueado na `GAME-20`; modelo candidato registrado |
| 4.3 EXP e progressão | `progressao.md` §7 | ⛔ curva não publicada; só tabelas de recompensa por faixa (B) |
| 4.4 Mortal/Arch/Celestial/Sub | `progressao.md`, `classes.md` §4 | 🚧 **§2.1** preenche os números de criação |
| 4.5 cidadania/Kibita/reinos | `canais-e-cidadania.md`, `lore.md` §7 | ✅ |
| 4.6 mapas e zonas | `sistemas/mundo.md` | 🚧 **§2.8** acrescenta Água, Dragon Field, Kefra e NPCs de Armia |
| 4.7 monstros e bosses | `sistemas/drops.md`, `sistemas/quests.md` | 🚧 IA/FSM na `GAME-06`; catálogo vem do banco (E) |
| 4.8 itens e equipamentos | `itens.md` | 🚧 **§2.7** acrescenta acessórios, Pedra Espiritual, repleção |
| 4.9 drop | `sistemas/drops.md` §1 | ✅ algoritmo 1..1.000.000 por item |
| 4.10 refino e composição | `sistemas/refino.md`, `refino-auditoria.md` | 🚧 D-1 (4/5%), D-2 (tabela de bless) e D-8 (brinco) seguem abertas |
| 4.11 montarias, crias, amagos | **este dossiê §2.6** | 🚧 primeiro registro com número (qualidade **70**); falta doc própria |
| 4.12 capas e pinturas | `progressao.md` §4, `sistemas/quests.md` | ✅ capas; pintura ainda **D** |
| 4.13 skills | `skills.md`, `classes.md` §2 | 🚧 **§2.2** preenche a Soul e a passiva de 12ª |
| 4.14 quests | `sistemas/quests.md` | ✅ |
| 4.15 NPCs | `sistemas/mundo.md` §1 | ✅ (+ NPCs de Armia em §2.8) |
| 4.16 consumíveis | `itens.md` §2 | ✅ |
| 4.17 guildas, fama, guerras | `sistemas/guerras.md` | 🚧 **§2.5** preenche números; D-6 e D-9 impedem hardcode de agenda |
| 4.18 eventos | `sistemas/eventos.md` | 🚧 D-7 |
| 4.19 Kefra e especiais | `sistemas/quests.md` §3–4 | ✅ regra de acesso fechada em §2.8 |
| 4.20 cidades e impostos | `sistemas/guerras.md` §2–3, `economia.md` §4 | ✅ |
| 4.21 economia | `economia.md` | ✅ |
| 4.22 trocas, grupos, loja pessoal | `economia.md` §6 | 🚧 concorrência/duplicação exige teste (`FASE-15`) |
| 4.23 PvP | `sistemas/guerras.md` §6 | 🚧 **§2.4** preenche CP/C.Point/Kill Points |
| 4.24 comunicação e Trombeta | **este dossiê §2.9** | 🚧 tabela de prefixos e comandos; consumo da trombeta em aberto |
| 4.25 cash e premium mail | **este dossiê §2.10** | 🚧 corrigido (cápsula grátis); rastreabilidade por item é **lacuna** |
| 4.26 troca de canal | `canais-e-cidadania.md` §5.2 | 🚧 plano desenhado, sem prova de campo |
| 4.27 login/logout/sessão | `senha-numerica.md`, `CLIENT-09` | ✅ com PIN |
| 4.28 tela de personagem | `classes.md` §1.2 | ✅ layout por offset |
| 4.29 barra de skills | `classes.md` §1.2 (`ShortSkill[16]`) | ✅ persistida |
| 4.30 efeitos visuais | `docs/cliente/` | 🚧 fora do escopo desta rodada |
| 4.31 itens iniciais | `classes.md` §1.2 (`BaseMob`) | ✅ |
| 4.32 buffs e temporizadores | `skills.md` §4, `arquitetura/matematica-sim.md` | ✅ unidade em tick |
| 4.33 tempos e resets | `sistemas/eventos.md` §5 | 🚧 horário por servidor (D-6/D-9) |

---

## 5. Perguntas abertas para o dono (continuam as P-1…P-8)

| # | Pergunta | Por que importa | Default seguro hoje |
|---|---|---|---|
| P-9 | A Soul aplica **70% / 40+30** (mortal) e **120% / 80+40** (celestial), como publica a fonte única? E o **+20% da Reinforce Soul** entra depois ou sobre o total? | é percentual de atributo — muda a build inteira | não aplicar percentual até captura; a passiva de +20% entra como multiplicador separado e configurável |
| P-10 | Confirmamos **5%** no +12 e **2…15%** na refinação abençoada como default do 7662 (4% e 1…10% ficam como variante de servidor moderno)? | fecha D-1/D-2 no seed | default documentado já em 5% e 2…15%; seed ainda em 4% e 1…10 até a decisão |
| P-11 | Gate do Arch: **300** (mínimo da era oficial), **355** (config de servidor privado / nível da capa) ou **400** (o que `progressao.md` assume hoje)? | muda `exp_table` e o gate de evolução | 300 como mínimo, com pontos por nível; 355/400 viram `game_setting` |
| P-12 | O RvR roda **sábado 22:00**, **seg–sex 22:00–22:50** — ou os dois, por canal? | agenda e imposto; a fonte oficial se contradiz | agenda por canal em `game_setting` |
| P-13 | Montaria entra no primeiro corte ou depois do mundo visível? Qualidade sobe até **70** no nosso caso? | é um sistema inteiro (ovo→cria→montaria→qualidade) | depois; qualidade como taxa, não constante |
| P-14 | Queremos **Trombeta** com fan-out entre canais reais desde já — e o item é **consumido**? | é contrato entre processos de canal | sim como evento de chat; consumo a decidir |

---

## 6. O que esta rodada **não** cobriu (honestidade de escopo)

1. **Catálogo de exploits e bugs históricos** (`wyd2.md` §6). Nada foi executado contra servidor de
   terceiros; e não houve coleta sistemática de relatos de duplicação. Continua pendente.
2. **Cliente 7662** — nenhum arquivo de `client/` foi analisado nesta rodada (`docs/cliente/` segue
   como está).
3. **Bots/LLM** (`wyd2.md` §9) — não pesquisado aqui; `docs/rust/07-ia-e-bots.md` continua a fonte.
4. **Curva de EXP** — segue sem fonte pública. As tabelas de recompensa por faixa que apareceram
   (ex.: 1–339 → 50kk; 390–399 → 10kk) são **recompensa de quest/evento**, não requisito de nível.
5. **Pintura de capa / customização de rosto** — só fonte **D**, sem número.
6. **Fontes que só viram snippet** — não foram usadas para número. Estão em §7.2: se alguma delas
   for necessária, o caminho é abrir a página (ou obter captura) antes de citar.

---

## 7. Fontes

### 7.1 Abertas e conferidas em 2026-11-10 (sustentam o texto acima)

| Fonte | Nível | Usada para |
|---|---|---|
| [Kersef — criação do Celestial](https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/22279-criacao-do-celestial) | B | §2.1 requisitos, 4 pedras, Soul, Cythera |
| [Kersef — Refining +12 a +15 (PT)](https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/58750-refining-system-12-to-15) / [(EN)](https://feedback.kersef.com/en/knowledge-bases/21/articles/58750-refining-system-12-to-15) | B | D-1/D-2 (4%; bless 2…15% nas duas línguas) |
| [Kersef — Refining +10](https://feedback.kersef.com/en/knowledge-bases/21/articles/58872-refining-10-and-changing-additionals) | B | confirma +10 (Pedra do Sábio + 50kk + 4 joias) |
| [WYD Int — Refining +12 a +15 (guia 47)](https://wyd-international.com/guides/47/refining-system-12-to-15) | C | D-2, D-10, destruição de 20% abaixo de 5% |
| [WYDGalahan — Refinações do WYD (2009)](https://wydgalahan.wordpress.com/guia-de-jogo/guia-avancado/refinacoes-do-wyd) | D | D-1 (5% no +12, era oficial) |
| [WYDGalahan — Criando Personagem ARCH (2009)](https://wydgalahan.wordpress.com/guia-de-jogo/guia-avancado/criando-personagem-arch) | D | D-3 (nível 300 mínimo) |
| [WarLokos — Arch: criação e destraves (2009)](http://warlokos.blogspot.com/2009/12/arch-criacao.html) | D | D-3 (300 mínimo, 400 recomendado, 355/370 destrave) |
| [ABC do WYD — Capas](http://abcdowyd.blogspot.com/2010/06/capas-mortalcelestial.html) | D | D-3 (355 = borda grossa, 371 = cor) |
| [WYD Global — Bahamut Accessories](https://wydglobal.raidhut.com/en/3391) | B | D-8 (brinco: 40%/5%, sem destruir) |
| [Kersef — Chaos System](https://feedback.kersef.com/en/knowledge-bases/21/articles/20250-chaos-system) | A/B | §2.4 faixa, dedução, ganho por nível, 1 min |
| [Kersef — C.point](https://feedback.kersef.com/en/knowledge-bases/21/articles/20249-cpoint-system) | A/B | §2.4 C.Point, 1/10, 80%, 350–355 |
| [Kersef — Guia Guild](https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/57606-guia-guild) | B | §2.5 torres, fama 200, Noatun, 400 de influência, Kefra sábado |
| [Kersef — Siege Warfare](https://feedback.kersef.com/en/knowledge-bases/21/articles/20258-siege-warfare) | B | §2.5 portal 700/50%, cerco 18:00–19:55 |
| [Kersef — Guerra de Reinos](https://feedback.kersef.com/en/knowledge-bases/21/articles/21552-guia-guerra-de-reinos) | B | §2.5 RvR completo; D-6 |
| [Kersef — Mount System](https://feedback.kersef.com/en/knowledge-bases/21/articles/20121-mount-system) | B | §2.6 fome, vida, bônus fixos |
| [Kersef — Premium Neil](https://feedback.kersef.com/en/knowledge-bases/21/articles/22192-premium-neil-prisang-neil) | B | §2.10 NP, senha, abas, Wyden Coin |
| [Kersef — Chatting](https://feedback.kersef.com/en/knowledge-bases/21/articles/20086-chatting) | A/B | §2.9 prefixos, `/tab` 70+ |
| [Kersef — Pergaminhos da Água](https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/22545-pergaminhos-da-agua) | B | §2.8 água (8 andares, grupo 8, N/M/A) |
| [Kersef — Dragon Field](https://feedback.kersef.com/en/knowledge-bases/21/articles/49455-novo-mapa-dragon-field) | B | §2.8 Dragon Guardian/Verid/Nesses/Haden |
| [Kersef — mapa de Armia](https://feedback.kersef.com/en/knowledge-bases/21/articles/20375-mapa-armia) | B | §2.8 NPCs de Armia |
| [Kersef — Dungeon Kefra](https://feedback.kersef.com/en/knowledge-bases/21/articles/21703-dungeon-kefra) | B | §2.8 acesso, 100 entradas, canal novato, drop 100% |
| [Kersef — Repletion](https://feedback.kersef.com/en/knowledge-bases/21/articles/20120-repletion-system) | B | §2.7 repleção |
| [Kersef — 12th skills](https://feedback.kersef.com/en/knowledge-bases/21/articles/50023-12th-skills) | B | §2.2 Reinforce Soul +20%; D-4 |
| [WYD Global — montarias](https://wydglobal.raidhut.com/pt-br/2874) | B | §2.6 essência +10, trava 181, qualidade **70** |
| [WYD Global — guild & war](https://wydglobal.raidhut.com/en/2120) | B | §2.5 pontuação de guerra, torres, influência; D-9 |
| [WYD Global — inventário virtual](https://wydglobal.raidhut.com/pt-br/3536) | B | §2.10 cápsula gratuita, 120 slots |
| [WYD Int — Chaos/PK](https://wyd.international/guides/46/chaos-system-pk-system) | C | §2.4 cores, dedução 3–4, Kill Points, faixas de C.Point |
| [WYD Int — patch de CP](https://wyd-international.com/news/38) | C | §2.4 penalidade 5/15/25k, −75, brinco; D-8 |
| [WYD Int — Mount System](https://wyd-international.com/guides/34/mount-system) | C | §2.6 8 espécies/10 raças, absorção 25% |
| [WYD Int — criação do Celestial](https://wyd-international.com/guides/23) | C | §2.1 400/356, Pedra Ideal, 4 cristais, runas |
| [WYD Int — changelog 2026-09](https://wyd-international.com/news/143) | C | §2.3 resistência 0–250 / 40% |
| [WYD Int — Server Info](https://wyd-international.com/information) | C | §2.5 `/create` 100kk; §2.9 comandos |
| [Wyd DoL — Arch](https://wyddol.com/arch) | C | §2.1 Imortalidade/Sephirot (8 pedras + 30kk) |
| [Wyd DoL — comandos](https://wyddol.com/command-list) | C | §2.5 `/create` 100kk; §2.9 `/spk`, `/guildtax` 0–25% |
| [Origem Destiny — comandos](https://origemdestiny.com.br/blog/guia-de-comandos-ingame) | C | §2.9 `/gritar` via trombeta, `/guildtax` de Noatun |
| [FreeWyd — velocidade](https://pathoflords.com/mecanicas/velocidade-de-ataque) | C/D | §2.3 duplo/triplo/crítico (hipótese) |
| [FreeWyd — evasão](https://pathoflords.com/mecanicas/evasao) | C/D | §2.3 evasão, teto 80%, TK −30 pp |
| [worldwyd — Limite da Alma](https://worldwyd.webnode.com.br/wyd/skill-limite-da-alma) | C/D | §2.2 requisitos, combinações, percentuais (fonte única) |
| [Point dos XD — Celestial/Sub](https://pointdosxd.blogspot.com/2011/06/mega-tutorial-celestial-e-sub-celestial.html) | D | §2.1 bônus, Subcelestial |
| [ABC do WYD — criando Celestial](http://abcdowyd.blogspot.com/2010/06/criacao-do-personagem-celestial.html) | D | §2.1 1001/2600, 855/1500, Cythera por faixa |
| [WYD Brazuca — Arch 355](https://wydbrazuca.com.br/guias-do-wyd/99-Evoluindo-para-Arch) | C | §2.1 rota 355, bloqueio 350; D-3 |
| [wydtutoriais — criando Arch](https://wydtutoriais.wordpress.com/criando-arch) | C/D | §2.1 almas, Oráculo, 8 pedras + 30kk |
| [Revolution WYD — acessórios](http://revolution-wyd.blogspot.com/2009/09/acessorios.html) | D | §2.7 anéis, braceletes, Pedra Espiritual |

### 7.2 Removidas nesta auditoria (não abertas — não sustentam número)

Estavam citadas na primeira versão, vieram de snippet de busca ou não abriram. Saíram do texto e
ficam aqui para não voltarem por engano:

| Fonte | Por que saiu |
|---|---|
| `feedback.kersef.com/.../20246` (Ancient) | não abriu; regra de "uma grade abaixo" não é usada no §2 |
| `feedback.kersef.com/.../22278` (Pista de Runas) | não abriu; a menção a Kefra ficou coberta por 21703 |
| `feedback.kersef.com/.../67189` (lista negra) | **não localizada**; o limiar −75 está em 20250/46/38, que abriram |
| `wyd-international.com/guides/35` (drop) | não abriu; nada do §2 depende dela |
| `blogwydtutoriais.blogspot.com` (sets) | não abriu; vocabulário de adicional não entra em número |
| `preprod.wyd-international.com/donate-store` | subdomínio de pré-produção **não abriu**; era a única base de "Shop Points 100 NP" e "cápsula 200 NP" — ambas removidas |
| `wydglobal.raidhut.com/en/1582` (Merchant's Ville) | não abriu; era a origem das coordenadas de NPC de refino, que saíram do §2.8 |
| `wyddol.com/creation-of-the-celestial` | não abriu; 22279 e o guia 23 cobrem a criação |
| Wiki Fandom (fanwyd) — tabela de bless e 5% no +12 | **HTTP 403**: não abriu; o 5% ficou sustentado pelo guia de 2009, que abriu |
| Portais UOK/Brazuca (agenda de Kefra) | só snippet; D-7 usa as duas páginas oficiais abertas |

**Como repetir:** as páginas acima mudam de conteúdo (a central do Kersef mantém artigos de 7 anos
atrás no ar ao lado dos atuais). Ao usar qualquer número, registre **URL + data + servidor** e
**abra a página** — é a regra de `fontes.md` §1: *mecânica é estável, número não é*.
