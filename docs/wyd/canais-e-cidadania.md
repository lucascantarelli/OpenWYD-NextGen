# Servidores, canais, cidadania e canal verde (2026-11-08)

Base de conhecimento para o que o jogador chama de "servidor" dentro do WYD e para tudo que
depende disso (cidadania, bônus, guildas donas de cidade, canal novato). Fonte primária é o
cliente oficial `client/` e o legado `Source/Code` (hoje histórico); onde a evidência é de
terceiro — o WYD Global/Raid Hut em funcionamento — está marcado como **referência**, nunca como
requisito nosso.

## 1. Servidor e canal são coisas diferentes

| Conceito | O que é | Onde vive |
|---|---|---|
| **Servidor** (grupo na `serverlist.bin`) | instalação independente: processos, banco e mundo próprios | uma linha de 64 B na grade do `serverlist.bin` (`docs/cliente/formatos.md` §4) |
| **Canal** (célula 1..10 do grupo) | instância de mundo do **mesmo** servidor; compartilha contas, personagens, guildas e economia | célula com o host da instância; o cliente disca **8281** fixo |

Consequência prática: dois canais não se distinguem por porta — cada um precisa de um **IP
próprio** na `8281`, porque o cliente fixa a porta. O que muda entre canais é o `ServerIndex`
(zero-based) do processo, usado como identidade em três lugares do legado:

- `Citizen = ServerIndex + 1` (cidadania, `_MSG_Quest.cpp:2436`);
- NPC de outra cidade recusa o serviço (`Server.cpp:2891`, `Citizen != ServerIndex + 1`);
- id de guilda carrega o canal nos bits altos (`serverIndex * 4096`, `ProcessDBMessage.cpp:760-790`).

O cliente lista **um slot por grupo**: nome vindo de `client/sn.bin` (`%s-%d`, nome + número do
canal) e endereço da célula do grupo. Célula vazia aparece como `Inexistente`.

## 2. Cidadania

Cidadania é o vínculo do personagem com **um canal**, persistido no save (`MobExtra.Citizen`) e
mantido ao relogar e ao trocar de canal.

| Regra | Evidência |
|---|---|
| compra na **Kibita** (Merchant 74, em Armia) por **4.000.000 de gold** quando `Citizen == 0`, gravando `ServerIndex + 1` | `_MSG_Quest.cpp:2433-2437` |
| `/getout` (sussurro) zera a cidadania | `_MSG_MessageWhisper.cpp:57-64` (`_DN_NO_TOWNSPEOPLE`) |
| bônus de caça/drop no canal de cidadania | `Language.txt:423` (`_DN_TOWNSPEOPLE`), `:432` (`_DN_TOWNBOUNS`) |
| só cidadão cria guilda naquele canal | `handler/guild.go:81` exige `Citizen != 0` |
| cidade/guilda só é disputável no canal onde a guilda é reconhecida como cidadã | guias de guild (referência: <https://fanwyd.fandom.com/pt-br/wiki/Criando_Guild>) |
| auto-venda permitida apenas no canal da cidadania | guia de guild (referência) |

Referência de produção (WYD Global): o bônus de drop do canal cresce com a economia do canal —
**+1% por 1.000.000.000 de gold** coletado de imposto, até 50%, e cai de novo quando o senhor do
castelo coleta — ou seja, o bônus é **estado do canal**, não do jogador
(<https://fanwyd.fandom.com/pt-br/wiki/Criando_Guild>).

## 3. Canal verde ("canal novato")

O canal verde é uma **propriedade fixa de um canal**, não uma rotação nossa: eventos de produção
se referenciam a ele por posição (`novato + 1`).

| Regra do canal novato | Fonte |
|---|---|
| lojas pessoais abertas e **sem área PvP** | <https://feedback.kersef.com/en/knowledge-bases/21/articles/58186-beginners-guide> |
| portal do **Hall do Kefra** só abre no canal novato (todos os canais podem matar o Kefra) | <https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/21703-dungeon-kefra> |
| **Deserto Desconhecido** acessível só pelo canal novato | FAQ WYD Global, "Novo mapa Deserto Desconhecido" |
| **Guerra de Reinos** acontece no servidor novato | <https://feedback.kersef.com/en/knowledge-bases/21/articles/21552-guia-guerra-de-reinos> |
| **invasão de Noatun** ocorre no canal *seguinte* ao novato | <https://wydglobal.raidhut.com/pt-br/2119> |

Não confundir com o **evento de novato** (EXP extra para nível baixo, flag
`world_event_config.newbie_event_enabled` ou `/gm +newbiehour` no legado): evento é do servidor
inteiro; canal novato é do canal.

No legado **não existe flag "novato"**: PvP é decidido por mapa/zona (`AttributeMap`,
`Regions.txt`). Portanto canal verde = configuração do processo (mesmos mapas, PvP desligado e
loja permitida), aplicada por `-novice-channel`/`W2PP_NOVICE_CHANNEL` no desenho Go antigo.

**No Rust (FASE-03-10)** o papel é dado do canal: tabela `channel_role` (migration `0041`,
seed `1 = normal`, `2 = verde`), sobrescrita por processo em `WYD_SERVER_CHANNEL_ROLE`
(`normal`/`verde`; valor desconhecido é ignorado e vale o banco). O processo resolve o papel no
boot (`channels::effective_role`), publica em `GET /health` (`role`, `pvp`, `shop`) e loga
`event=channel role=… source=banco|env`. A política vive em `channels::ChannelRules`:
`pvp_allowed(map_allows_pvp)` e `personal_shop_allowed()` — o canal só **restringe** o PvP do
mapa, nunca libera além dele. Os handlers de PvP (checklist 4.5) e de loja pessoal (7.2) ainda
não existem no fio; até lá a regra é provada como política (`tests/canal_verde.rs`), sem pacote
inventado.

## 4. Guildas donas de cidade (por canal)

| Sistema | Regra observável |
|---|---|
| Guerra de Torres | diária, nos canais de guerra; dá **fama** (teto 200) e é a moeda de aposta das guerras de cidade |
| Guerra de Cidades | guilda com ≥ 100 de fama aposta numa cidade; vencedora passa a ser dona do território **naquele canal** |
| Posse | uma guilda não pode dominar o mesmo tipo de zona em dois canais ao mesmo tempo: vencer no canal B sem abrir mão do A libera o território de B |
| Posto de Defesa / Torre Eterna | ocupados por cidadãos do canal; sem eles um canal não aplica guerra de outro canal |
| Guerra entre Canais | o mestre da guilda dona do castelo declara guerra a outro canal; só cidadãos participam |

Fontes: <https://wydglobal.raidhut.com/pt-br/2119>, <https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/57606-guia-guild>,
<https://fanwyd.fandom.com/pt-br/wiki/Criando_Guild>. No nosso modelo, isso é estado **por canal +
por banco compartilhado**: a posse da cidade é linha de guild (`guild_tower`, `guild_relations` em
`crates/wyd-db/migrations/0028_estado_de_jogo.sql`) com o canal como chave, e a guerra é evento do
processo daquele canal.

## 5. O que isso exige do servidor Rust

1. **Registro de canais** (servidor → grupo; canal → host + `ServerIndex` + papel novato) é a
   fonte única do `serverlist.bin` e do roteamento — não o `.bin` em si (§`docs/cliente/launcher.md` §8).
2. Um processo por canal, **mesmo banco**: `ServerIndex` na configuração, cidadania e guildas
   derivadas dele.
3. Canal novato como flag por processo (PvP desligado, loja permitida) e o conjunto de eventos
   presos a ele — não como rotação diária escolhida por nós.
4. Bônus de cidadania **efetivamente aplicado** no cálculo de drop antes de vender cidadania.
   **Feito (FASE-03-10) como parâmetro:** `game_setting` (`Channel Settings.citizen_bonus_bps`,
   default `0`, teto 50%) entra em `Gameplay::loot*` pelo argumento `citizen`; o cidadão usa a
   taxa corrigida (`peso + peso × bônus`), o visitante fica na taxa do legado. Sem número com
   fonte o default `0` mantém o drop idêntico ao original — nenhuma taxa foi inventada. Falta
   **quem** é cidadão persistido (`MobExtra.Citizen`): hoje a cidadania é derivada do canal
   (checklist 2.1), então todo personagem no canal é cidadão dele.
5. Guerra entre canais só faz sentido com 2+ instâncias; até lá, um canal é o único mundo real.

### 5.1 Plano de canais para desenvolvimento (recomendado)

**Dois canais**, porque é o mínimo que exercita a diferença: canal 1 = normal, canal 2 = verde
(novato). Acima disso o custo sobe (um processo + um IP por canal) sem retorno antes de existir
cidadania aplicada e guerra entre canais.

| Canal | Papel | O que ele prova |
|---|---|---|
| 1 | normal | referência: PvP conforme mapa, regras completas |
| 2 | verde/novato | só existe no canal novato: sem PvP, loja pessoal aberta, portal do Hall do Kefra, Deserto Desconhecido |

Crescer depois é barato e **não exige build novo**: uma linha
`1 <canal> <ip>` em `tools/client/serverlist-servers.txt` → `cargo run -p wyd-launcher -- serverlist
…` → reabrir o cliente. Tetos do formato: **10 canais por servidor** (células 1..10 do grupo) e
**10 servidores** (grupos); cada canal precisa de IP próprio porque o cliente fixa a porta 8281.
Quais linhas *aparecem* é decisão do `sn.bin` do cliente (`%s-%d` por slot), não do servidor.

### 5.2 Sessão única e trânsito entre canais

Canal é mundo separado, mas o personagem é **um só** e mora no Postgres do grupo. Três regras
saem daí (implementadas em 2026-11-08):

- **uma sessão viva por conta** — `character_session` (migration `0030`): o índice parcial
  `character_session_uma_por_conta` recusa a segunda entrada enquanto a primeira vive, em
  qualquer canal do grupo. É o `_MSG_DBAlreadyPlaying` do legado resolvido por constraint, não
  por checagem em memória (canais são processos distintos);
- **trânsito é relogar, sem perder nada** — sair libera a sessão (`released_at`) e a entrada no
  outro canal relê o personagem do banco; nada é copiado. O legado precisava de
  `_MSG_DBServerChange` (`0x0814`) / `_MSG_DBCNFServerChange` (`0x052A`) porque **cada canal tinha
  o próprio DB**; com um Postgres por grupo, trocar de canal não move dado;
- **sessão órfã não prende jogador** — `renewed_at` é renovado enquanto o cliente está conectado;
  processo que morre sem liberar tem a sessão retomada após `WYD_SERVER_SESSION_STALE_SECS`
  (default 180 s).

O isolamento entre canais continua por construção: cada processo tem o próprio `World` e não há
IPC nem estado de mundo compartilhado — o banco compartilha identidade e persistência, **não** a
cena.

## 6. Fontes

- Cliente oficial `client/7662`: `serverlist.bin`, `sn.bin` (nomes + contadores) — `docs/cliente/formatos.md` §4.
- Legado C++ (`Source/Code`): `Basedef.cpp:6874` (formato), `_MSG_Quest.cpp:2433-2437` (cidadania),
  `Server.cpp:2891`, `ProcessDBMessage.cpp:760-790` (id de guilda por canal), `Language.txt:423/432/426/501`.
- Documento histórico da `main` (`docs/migration/canais.md`) — decodificação do `sn.bin` e do
  cálculo de canal por dia (`0x46d6ce-0x46d9cc`); preservado aqui porque a árvore Go saiu do repositório.
- Referência de cliente real em funcionamento (terceiro, só comportamento observável): FAQ WYD
  Global/Kersef e notícias do Raid Hut citadas acima.
