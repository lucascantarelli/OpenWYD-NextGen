# Guia OnGame (site oficial BR, 2005–2016) via Wayback Machine

Fonte **A — oficial** (publicador licenciado pela HanbitSoft; ver `fontes.md`).
É a melhor fonte pública de **mecânica** do WYD da era 7.x. Números valem como
referência de época (~2010–2012, capturas usadas), não como regra do 7662:
mecânica é estável, número muda entre patches — confrontar com o cliente do
repositório (dado E) antes de virar regra de código.

## Como acessar

Domínio original `wyd.ongame.com.br` (fora do ar desde 25/07/2016). Arquivo:
`https://web.archive.org/web/<TIMESTAMP>/http://wyd.ongame.com.br/...`

Padrão do guia: `guia/?m=<GUIA>&sm=<SEÇÃO>&ism=<SUBSEÇÃO>`

| `m` | Guia | `sm` | Seção | `ism` |
|---|---|---|---|---|
| 1 | Básico | 1 | Começando (config, criar personagem) | 1 config · 2 criar |
| 1 | Básico | 2 | Display (interface, teclas) | 1 interface · 2 atalhos |
| 1 | Básico | 3 | Loja/Depósito (compra, venda, banco, troca, venda automática) | — |
| 1 | Básico | 4 | Skills (mestre, linhagem, aprendizagem) | — |
| 1 | Básico | 5 | Grupo (formação, divisão de XP) | — |
| 1 | Básico | 6 | Guilda (criação, sub-líder, aliança, comandos) | — |
| 1 | Básico | 7 | Chat (atalhos, privada, grupo, mapa, guilda) | — |
| 1 | Básico | 8 | Modo C.C. (combate automático) | — |
| 1 | Básico | 9 | Cidadania (registro, custo, benefícios, bônus) | — |
| 2 | Intermediário | 10 | Refinação de item | 1 (+1~+9) · 2 (+10) · 3 (+11) · 4 (lendários) |
| 2 | Intermediário | 11 | Sistema Replation | — |
| 2 | Intermediário | 12 | Sistema de Montaria | 1 origem · 2 elementos · 3 evolução · 4 info · 5 itens |
| 3 | Avançado | (padrão) | Armas Ancient | — |

Capturas de referência (todas `web.archive.org`, testar `id_` se o layout quebrar):
- `.../web/20101224192748/http://wyd.ongame.com.br/guia/?m=2` (índice intermediário)
- `.../web/20110120152256/http://wyd.ongame.com.br/guia/?m=1&sm=5` (grupo)
- `.../web/20101224192753/http://wyd.ongame.com.br/guia/?m=2&sm=11` (replation)

## Valores extraídos (época ~2010–2012)

### Grupo — divisão de XP (`m=1&sm=5`)

| Membros | Bônus XP |
|---|---|
| 2 | 180% |
| 3 | 200% |
| 4 | 220% |
| 5 | 240% |
| 6 | 260% |
| 7 | 280% |
| 8 ou mais | 300% |

XP dividida por monstro conforme o level (maior level, maior parte).
Foema pode convocar o grupo (exceto locais específicos). Usar em FASE-10.

### Skills (`m=1&sm=4`)

- Skill só no NPC Mestre de Skill da classe, em Armia; custa Pontos de Skill.
- 3 linhagens por classe, cada uma com 8ª skill; 24 skills por personagem.
- Limite de aprendizagem por linhagem: `level × 1,5`; teto 200 pontos,
  255 com item de aprendizagem ou 8ª skill.
- Reset errado: Círculo Divino, Bênção de Deus, quest Balanço do Poder.

### Guilda (`m=1&sm=6`)

- Criação: `/criar [nome]` + 100kk de GOLD no inventário (retira ao equipar a medalha).
- 3 sub-líderes; **1 aliança por guilda** (proíbe PvP entre aliados, inibe Caos).
- Comandos: `guild`, `- msg`, `-- msg`, `guildtax 0~20` (só campeã com zona),
  `/summonguild`, `/fim da guerra`, `/fimirma`, `/criar [nome]`, `/[nome]` (info).
- Fim de aliança: `/Fim da Alianca`.

### Cidadania (`m=1&sm=9`)

- Registro no NPC Kibita, Armia, por canal. Mortal grátis.
- Custo: Arch `(Level+200) × 10.000` GOLD; Celestial `(Level+400) × 10.000`.
- Benefícios no canal de cidadania: **+10% drop**; bônus chega a 80% → XP aumenta.
- Kefra morto no canal: **+20% bônus**. A cada 100kk de XP do cidadão: **+1%**.
- Sair: `/fimcidadao`. Mortal sem cidadania vira cidadão do 1º canal que conectar.
- Arch/Celestial compram de NPC de qualquer canal, mas só vendem no da cidadania.

### Refino +1~+9 (`m=2&sm=10&ism=1`)

- Oriharucon até +6 (100% até +2); Lactolerium até +9 (100% até +4).
- Cada nível: +10% em ataque/defesa/opção adicional. Classe alta = taxa menor,
  consome mais pó. Exceção: itens "não refináveis".
- Bônus +9: elmo +30% resistência; peito/calça +25 defesa; luva −1s delay;
  bota +1 movimento; escudo +25 defesa e +25% resistência; lanças/cajados +8%
  ataque mágico; outras armas +40 dano; acessórios +10 efeito.
- 10 Restos + 1kk de gold → 1 Poeira (alquimista).

### Refino +10 (`m=2&sm=10&ism=2`)

- NPC Aylin, Azran: 2 itens +9 **de mesmo nome** + Pedra do Sábio + **50kk**.
- Jóias (1~4): +4% / +8% / +15% / +32% sucesso. Preview de 4 opções antes.
- Resultado: 1 item +10 (sucesso) ou os 2×+9 intactos + resto some (falha).
- Vale para armas Ancient e defesas (inclui escudo); acessórios e armas
  normais padrão, não. Tipos: Normal, Místico, Arcano, Legendário, Ancient
  (Ancient só mesma jóia).
- Jóias: Diamante +8% drop; Esmeralda +40 perfuração; Coral +2% XP;
  Garnet +40 absorção. Bônus base sobe, bônus não.

### Refino +11 (`m=2&sm=10&ism=3`) — exemplo Luna (garnet)

| Nível | Habilidade | Dano | Bônus |
|---|---|---|---|
| 0–6 | +0~60% | 350–560 | +20 absorção |
| 7–8 | +70~80% | 595–630 | — |
| 9 | +90% | 665 | +20 abs +40 dano |
| 10 | +100% | 700 | +40 abs +40 dano |
| 11 | **+120%** | 770 | +40 abs +40 dano |

+11 via Lactolerium no +10. Opções padrão sobem, bônus não.

### Replation (`m=2&sm=11`)

- NPC Lucy, Azran: pedras classe A (100k, muito alta), B (300k, alta),
  C (800k, normal). Falha **destrói** o item.
- Só armaduras +3 ou menores (inclui não-refináveis); escudo/arma/acessório, não.
- Requerido no inventário (não equipado, não no banco). Classe E (Premium Neil)
  é negociável; A–C de NPC não.

### Armas Ancient (`m=3`)

- NPC Compositor, Azran: arma +9 + 1 jóia + 0~4 itens +7~+10
  (+7:+2%, +8:+4%, +9:+10%, +10:+12%; ex.: 4×+9 = +40%).
- Itens auxiliares: no máximo 1 classe abaixo da arma (arma D → C ou maior).
- Falha: **tudo some**. Uma única vez por arma; +11 não serve de auxiliar.
- Resultado: arma Ancient **+7** + atributo da jóia (trocável com outra jóia):
  Diamante +8% drop; Esmeralda +80 perfuração; Coral +2% XP; Garnet +80 absorção.

### Montaria (`m=2&sm=12`)

- 8 espécies + 10 cavalos. Crescimento ovo → cria → adulto.
- Ovo→cria: Poeira Oriharucon/Lactolerium. Cria evolui caçando (+ataque).
  Adulto: essência do animal (+ataque e ataque mágico).
- Absorve parte do dano; pode dar dano/ataque mágico/evasão/resistência.

### Chat (`m=1&sm=7`) e Modo C.C. (`m=1&sm=8`)

- `Enter` conversa local; `/[nome]` privada; `/r` responde; `=` grupo;
  `-` mapa; `--` guilda. `/tab [msg]` nickname (level 70+). `/snd` ausência.
- C.C.: físico (buffs da 1ª barra + ataque auto), mágico (skills abaixo da
  barra vermelha), passivo (buffs + poção/ração auto). Não ataca player em
  área PK.

## Fórum: `comunidade/` (Fórum Livre, S.S., Vídeo, Guilda, Rankings)

O fórum **está** arquivado (~2277 páginas de tópicos em 12/2009). Índice:
`comunidade/?m=1` (`m=2` screenshot, `m=3` vídeo, `m=4` guilda,
`m=5/6` rankings, `m=9` twitter, `m=10` OngameTube).
Tópico: `comunidade/?m=1&sm=2&pl=1&q=&t=0&id=NNNNN` (forma antiga:
`?m=1&sm=2&pag=0&id=N`). Busca: `?m=1&q=<termo>&t=0&p=N`.

### Índice das Dicas do MOD (`id=32224`, GM, 17/09/2009, 6556 visitas)

1. Passar add p/ armas Arch (`id=847`) · 2. Pedras secretas (`id=5`) ·
3. Itens p/ criação do Arch (`id=6`) · 4. Skill Limite da Alma (`id=7`) ·
5. Criando Extração (`id=8`) · 6. Criando Celestial (`id=18`) ·
7. Lentidão do mouse (`id=10`) · 8. Barras Mytrill (`id=11`) ·
9. **Refinações +12~+15** (`id=11135`) · 10. Pedra Espiritual F (`id=13`) ·
11. Drops de Skills Sephira (`id=14`) · 12. Suporte/denúncias (`id=15`) ·
13. Atualização do client (`id=16`) · 14. Refin. Abençoadas na 7.54 (`id=32221`).

### Dica do MOD 9 — +12 até +15 (`id=11135`, MOD-B1/B2, 06/07/2009)

- Armas: +9 → Ancient (Compositor) → volta +7 → +9 Ancient → 2× Ancient +9
  unidas = +10 → Lac até +11 → apto ao +12. Defesa/escudo: 2× +9 → +10 → +11.
- Receita +12 (igual até +15, subindo o item base): item +11 (ou Ancient +11)
  + **2 packs de 10 Poeiras Lac** + **4 Pedras Secretas** (quaisquer),
  no **Alquimista Odin, Kefra (Continente Desconhecido)**. Ordem: PLs, item,
  pedras.
- Sucesso: some tudo, resta só o item +12. Falha: some tudo **e** o item pode
  cair 1 nível (+11→+10, +14→+13) **ou quebrar**.
- 1~4 **Refinações Abençoadas** no lugar das Secretas aumentam a chance
  (mais = melhor; comentário de jogador: cada nível da Abençoada = +1%,
  ex. 4× +9 = +36% — marcar como D, não oficial).
- **2 Emblemas da Proteção** no lugar dos 2 packs de PL (só aos pares): na
  falha, a opção adicional vira **cristal de extração** (salva os adds).

### Criação da Pedra Ideal (cadeia "lendária", `ism=4` nunca capturado)

A página original só tem captura 302. Conteúdo equivalente da época, espelho
`wydgalahan.wordpress.com/guia-avancado/criando-pedra-ideal` (D; confrontar):

1. **4 Pedras Secretas** (Sol/terra/água/vento) — runas da Pista de Runas,
   composição no NPC **Odin, Kefra**, 2kk cada. Sol: Anzuz Elhaz Gebo Mannaz
   Raidho Swoilo Tiwaz · Terra: Dagaz Fehu Kenaz Naudhiz Swoilo Thurisaz
   Raidho · Água: Berkano Elwaz Jara Raidho Swoilo Uraz Wunjo ·
   Vento: Eihwaz Hagalaz Ing Isa Laguz Othel Perthro.
2. **Pedra da Imortalidade** — Alma do Unicórnio (Lugefer Broken, Deserto de
   Kult) + Alma da Fênix (Lichs Krunt/Vatama, dungeon Kefra) + 10 Safiras,
   no Oráculo do reino.
3. **Sephirot** — 8 pedras (classes/drops por monstro) + 30kk no Mestre de
   Skill da classe desejada (ou cash). Imortalidade equipada não é consumida.
4. **Pedra Ideal** — Imortalidade + Sephirot equipados, 4 Secretas no
   inventário, clicar no Rei (`/king`). Feita por Arch (mortal com os mesmos
   itens cria Arch).

Espelho do Guia Avançado fora do arquivo (D): índice em
`wydgalahan.wordpress.com/guia-de-jogo/guia-avancado` — Ideal, Secretas,
Arch, HardCore/Divino, Sephirot, Sub-Celestial, Cristal Extração, ANCT,
ovos/âmagos, Griupan, Refinações, Skill Soul, Drops 7.54, Celestial.

## Notícias, Atualizações e Regras (nível A)

- Índices: `noticias/?m=1` (notícias), `m=2` (eventos), `m=3` (atualizações/
  manutenções). Artigo: `?m=N&sm=1&id=NNNN&p=0`. Ex.: índice 07/2009 e
  12/2009 capturados; paginação profunda (`p=2+`) e busca (`q=`) sem captura.
- **Bug do duelo (30/06/2009, `id=3378`):** exploit derrubava item do alvo
  via pedidos de duelo/troca com o item selecionado no mouse; solução
  definitiva aguardava a produtora (JoyImpact). Mitigações oficiais: não
  mostrar item a estranhos; mostrar sem grupo; não equipar de imediato;
  negar com ESC (nunca clicar p/ cancelar/equipar); equipar via Ctrl+botão
  direito; comando **"whisper"** bloqueia todo pedido de troca/chat.
  Relevante p/ FASE-06-4 (catálogo de ameaças) e segurança de trade.
- **Correção do Alastar 1ª fase (23/06/2009, `id=2214`):** remoção
  permanente de montarias e itens ilegais/duplicados do servidor, com
  limpeza contínua nas semanas seguintes. Precedente oficial de correção
  econômica (respalda política de remoção, não mecânica).
- **Regras do Jogo** (`corp.ongame.com.br/regras-do-jogo/wyd.asp`,
  capturas 2010, texto integral conferido — só há as duas tabelas de grades
  + preâmbulo jurídico, sem seção de apelação/trade): contas Free (GRADE A
  10d / B 15d / X permanente) vs Ongamer/cash (X = 60d + fama zerada +
  inventário zerado salvo cash + gold zerado + HOLD 50%/100% + gold −2bi +
  tag Bad User). Lista GRADE B inclui comércio RMT e compartilhamento de
  conta; GRADE X inclui scam/hack/abuser/clone de GM. Valor p/ moderação
  (FASE-07), não p/ mecânica — frente **esgotada**.
- **Sobre os Suportes (16/11/2009, `id=3909`):** 4 filas — QNA (dúvidas,
  sugestões, **redução de nível e senha numérica**), BAD USER (denúncias +
  re-análise, SS com data/hora/minimapa), BUG (erros, sumiço de item),
  HACKING (invasão/roubo); força-tarefa contra backlog; pedido anti-flood.
  Fecha a frente de moderação/suporte do guia.
- **Ação anti-bad-user (05/11/2009, `id=3867`):** varredura diária dos
  logs; punição estendida a **todas as contas ligadas (CPF/e-mail)** e a
  quem receber itens transferidos das punidas. Precedente oficial p/
  FASE-06-4/07 (rastreio de transferência).
- **Batalha Real (18–26/06/2009, `id=2210`):** 150 Medalhas Douradas por
  drop global em Kersef + 150 Azuis no BIG SALE (3100 cash); entrada via
  NPC em Armia canal 3 (tolerância 20 min); finalistas: Eirenus Anct +10
  40% magia / Khyrius Anct +10 108 dano; campeão: 48% / 144 + cor exclusiva.
  Válido Malech/Sephira/Alastar. Mecânica de evento completa p/ FASE-10.
- **Linha de patches 7573→7587 (índice `m=3`, captura 2012 — títulos e
  datas, corpos sem captura):** 7573 correção de Safiras (09/08/11); 7580–
  7581 lançamento 758 (08/11/11); 7582 bugs (09/11/11); 7583 segurança
  (16/11/11); Quests 350 p/ Arch (21/12/11); 7584 **correção NPC Ehre**
  (17/01/12 — mecânica de abençoada/alma afetada, detalhe perdido);
  7585 X-TRAP (31/01/12); 7586 segurança (03/02/12); 7587 bug (07/02/12);
  segurança servidores (17/04/12). Corpos `id=6362/6776/6793/6816/6981/
  7096/7164/7187/7198/7582` retornam 404 — lacuna explícita, sem reconstituir.

## Divergências registradas (não decidir no código)

- Primeiros passos 100%: o guia dá Ori 100% até +2 e Lac 100% até +4; o
  `SancRate.txt` legado e os seeds (`0022_seed_refino`, `0025/0026_sanc_rate`)
  usam a escada 90/85/80/… — ver `docs/wyd/sistemas/refino.md` §2, que já
  registra isso como configuração de servidor.
- +10 com jóias +4/+8/+15/+32% (guia) vs 30% base em composição (seed) —
  mesma natureza: configuração, não erro de leitura.
- Comentários de jogadores no fórum (ex.: Abençoada +1%/nível) são nível D.

## Dicas do MOD + lacunas (insumo FASE-10)

### Dicas do MOD extraídas (nível A, GM, 05–09/2009)

- **Dica 1 — passar add p/ arma Arch** (`id=847`): mortal [D]/[E] anct
  +9/+10/+11 com o add + arch anct +9 + 100kk (+ opcional equip [D]/[E] +9
  p/ chance), NPC **Tiny, Niflheim**. Sucesso: arch vira **+7 com o add**;
  falha: ancients intactas, gold e item +9 perdidos. Não transfere de C p/ D.
- **Dica 3 — drops p/ Arch** (`id=6`, 20k visitas): Fênix = Lich Vatama
  (Kefra); Unicórnio = Lugefer Broken (Deserto); 8 pedras Arch por monstro
  (Trooper q121, Gárgula Fogo/submundo, Lugefer, Esqueleto/dungeon,
  Frenzy, Lorde Demônio, Mantícora, Dragão Lich).
- **Dica 4 — Skill Limite da Alma** (`id=7`): mortal 371+ + pedra secreta
  da classe (HT vento, BM terra, TK água, FM sol) + 8ª skill → NPC Kibita
  (canal de cidadania); celestial já recebe. Config no **Ehre, Erion** com
  3 jóias (ex. FOR/CON = 3× Diamante). Mortal: 70% (1 atributo) ou 40/30;
  celestial: 120% ou 80/40. Aumenta stats.
- **Dica 5 — Extração** (`id=8`, 13,5k visitas): Huntress com Barganhar +
  Pedra do Sábio transforma o item em cristal; aplicar o cristal em item
  **< +9** é **100%**, mesma classe e tipo (Luva×Luva). Cristal guarda a
  base (+0); refino soma +10%/nível por cima (ex. base 10 +5 = 10(15)).
  Liga com os Emblemas da Dica 9.
- **Dica 6 — Celestial** (`id=18`, 19k visitas): Pista de Runas em grupo
  (Ansuz 2×2 com boss; Uraz 1–3 defendendo Thor 15 min, ±50% XP); mortal 400
  + arch 356+; **1001 stats base, teto 2600** (+ bônus por nível do arch e
  quest dos 4 cristais); **855 aprendizagem + 1500 skill**; Cythera por
  nível do arch (Prateada 356–380: 80 def/10% mana/30 perf; Dourada 381–390:
  +30 abs; Mística 400: 110 def/40 perf/40 abs); Limite da Alma automática;
  inventário só com medalha + capa; destrava 40 e 90.
- **Dica 8 — Barras Mytrill** (`id=11`): NPCs do reino (Berdes/Redmiron),
  **100kk**; 4 tipos (dano físico/mágico, crítico, defesa); item arch +11+
  equipado; re-rola **um** add existente (ex. luva 30 dano → 36 ou 24).
  Tetos: físico 81 arma/42 set; mágico 36/14; crítico 9%; defesa 70.
- **Dica 10 — Pedra Espiritual F** (`id=13`): 2× espirituais +9 ABS/PERF +
  1× +0~+9 + 4 Lac, Arch 256+, NPC **Shany, Niflheim (gelo)**; falha perde
  Lacs + pedra do slot 3 + 500k; sucesso custa **2kk XP**; resultado F+0
  aleatório (abs 15 / perf 15 / ambos).
- **Dica 14 — Abençoadas na 7.54** (`id=32221`): base +12:5% +13:3% +14:2%
  +15:1%; Abençoada +N soma (+0:2% … +8:12% +9:15% — ex. 4× +9 no +12 =
  65%); **4 Abençoadas +1+ = sem quebra** (só −1 nível); refinar Abençoada
  no **Ehre, Erion** (só celestial >40, Divino não): 1 Abençoada + 2 Safiras
  + 5kk XP, chance por nível do celestial (30% <150 … 100% >191).

- **Dica 2 — pedras secretas** (`id=5`, 24k visitas): confirma as 4
  sequências do guia (corrige Swoilo→Sowilo nos comentários) + destrave de
  level do celestial no **Odin**: nv 40 = Fehu Mannaz Thurisaz Ansuz Selado
  Selado Selo Mágico; nv 90 = Sowilo Kenaz Uraz Thurisaz Isa Tiwaz Hagalaz.
  Custo 2kk, 4 slots livres.
- **Dica 12 — suportes** (`id=15`): BUG / BAD USER (denúncia, SS com
  minimapa+data, sem edição) / Hacking (até 3 dias; revisão e devolução só
  p/ mortal 351+ ou arch 256+; **devolvido com −2 de refino**, consumível e
  gold não voltam) / QNA. Política útil p/ FASE-04/07, não é mecânica de jogo.
- **Dica 13 — atualizar o client** (`id=16`, 12,7k visitas): launcher
  auto-patch; patch manual `75337540.exe` (extrair com overwrite);
  Vista: rodar `WYD.EXE` + launcher como administrador (`INITIALIZE DATA
  FAILED`). Valor p/ launcher, não p/ taxa.

### Lacunas irrecuperáveis (marcadas, sem inventar)

- **Dica 11** (drops de livros Sephira, `id=14`): 404 em todos os
  timestamps testados (2009, 2011, 2012). Sem fonte.
- **`ism=4` (Criação de Itens Lendários)**: só existe captura 302 (redirect
  de crawl p/ `capa.php`); conteúdo equivalente coberto pelo espelho
  WYDGalahan (nível D) + Dicas 2/3/6 acima.
- **Guia Avançado além de Ancient**: o HTML cru da captura mais antiga
  (2009) não tem nenhum link de subseção — o Avançado oficial tinha só
  Armas Ancient. O índice "Avançado" do WYDGalahan é organização do blog,
  não do guia oficial (marcado como D onde citado).
- **Dica 7** (lentidão do mouse, `id=10`): fora do escopo FASE-10, não extraída.
- **Corpos de tópicos comuns do fórum (ex. `id=61806` QUANTO VALE):** CDX
  vazio + 404 — só tópicos linkados como fixos/indexados (Dicas do MOD)
  foram capturados. Preços de mercado da época no fórum, **irrecuperáveis**.
- Busca textual no fórum (`?m=1&q=<sistema>&t=0&p=N`) por refino, bug, kefra.
- Cruzar cada número com o cliente 7662 (dado E) antes de virar seed/regra.

**Pronto p/ FASE-10:** refino +1~+15 (guias + Dicas 9/14), ancient (guia +
Dica 1), extração (Dica 5), abençoadas (Dica 14), mytrill (Dica 8),
espiritual F (Dica 10), alma (Dica 4), ideal/celestial/secretas (Dica 2/3/6 +
espelho D). Lacunas acima são finais — não reextrair sem fonte nova.

## Droplist 7.54 (espelho D, substituto parcial da Dica 11)

`wydgalahan.../tabela-de-drops-versao-7-54-atualizado` (blog, 2009–2012).
Padrão "o que cai onde", sem taxa — mecânica qualitativa p/ FASE-04/10:

- **Tauron/deserto:** sets C/D/E, poeiras, gems Siren; Lugefer → Alma Uni +
  Pedra Lugefer; Mantícora → Pedra Mantícora; Assassino Tauron → armas
  (Olho Carbunkle, Gungnir...); Adamant → adamantita + ankh crítico.
- **Kefra:** restos/poeiras em tudo; Lich Vatama/Crunt → Alma Fênix +
  Concentração + Spectral; Simio → âmago Unicórnio; Batorero → pedra
  espiritual + adamantita; Pista de Runas via Cropper.
- **Submundo:** Argos → convite Hydra; Aquagolem → Replation D + Emblema;
  Gárgula Fogo → Pedra Gárgula; Frenzy → Pedra Rei Demonlord.
- **Lan N/M/A:** armas D / Ressurreição / Espectral + set E mortais.
- **Gelo:** Lobo/Urso Polar → luvas, Pedra da Luz, replation E, ingred.
  Griupan; Valquírias → Legionário TOP + Pegasus; Beriel → jóias + Uni/Pegasus.
- Comentários de jogador confirmam/corrigem (Sábio: Tauron no deserto;
  Pedra do Sábio: Morloqui/submundo; Pegasus: Lich Vatama) — tudo D.

### Conferência Droplist (D) × Dica 3 (A) — 7.54

**Concordam (usar como base):** Lich Vatama/Crunt → Alma Fênix;
Lugefer → Alma Uni + Pedra do Lugefer; Mantícora → Pedra da Mantícora;
Gárgula de Fogo (submundo) → Pedra do Gárgula (+ Escudo de Runas);
DemonLord → pedra do demonlord + alma da fênix.

**Só na Dica 3 (droplist omite — lacuna do espelho, não contradição):**
Pedra do Lorde Orc Trooper (quest reset 121); Pedra do Esqueleto
(Guerreiro Esqueleto, dungeon); Pedra do Rei Demonlord (Frenzy, sala final
submundo); Pedra do Dragão Lich (último andar dungeon); Lugefer Broken
como mob específico (droplist lista Lugefer/Cavaleiro apenas).

**Só no espelho (D, confrontar com 7662):** Uni também via Simio,
Beriel e Templário Amaldiçoado; Pegasus via Valquíria Rozen **vs**
comentário que diz Lich Vatama (divergência D×D, sem árbitro);
Coral no Morlock **contestado** por jogador ("fico la e nn drop nd");
Replation E dropando no Gelo **contradiz** o guia (E = Premium Neil) —
vale o guia (A) até prova no 7662.

**Proveniência:** comentário aponta a lista como cópia do fórum WebCheats
— reforça o teto D. Regra de acesso (D): Lan A = celestial + guilda dona
de cidade.
