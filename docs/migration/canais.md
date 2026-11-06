# Servidores, canais, cidadania e o "canal novato"

> Escrito em 05/11/2026 a partir do código legado (`Source/Code/TMSrv`), do conteúdo
> em `tmserver/content/Common/serverlist.txt` (hoje arquivado em `docs/archive/release-common/`) e da
> documentação pública do WYD. Onde não há
> evidência no repositório a marcação é **UNVERIFIED**.

## 1. O que o jogador chama de "servidor" é uma linha da serverlist

O cliente não descobre nada: ele lê `serverlist.bin`, gerado pelo
`serverlist bin/serverlist editor.exe` a partir do `serverlist.txt` do kit legado, hoje arquivado em
`docs/archive/release-common/serverlist.txt` (a entrada canônica do nosso gerador é
`client-tools/serverlist/servers.txt`):

```
0 0 192.168.2.105
0 1 192.168.2.105
```

O formato é `<grupo> <canal> <ip>` (`docs/migration/config-ops.md:137`), e o editor
grava também a porta (hoje `8281`, ver `AGENTS.md` §2). Ou seja:

- **grupo** = a "linha" que o cliente mostra como nome do servidor (WYDDCDK, PVP, UP…);
- **canal** = a coluna dentro daquele grupo; é o que o jogador escolhe no segundo clique;
- **ip:porta** = o `TMSrv` daquele canal.

Tudo isso aponta para `TMSrv`s distintos, mas **um único `DBSrv`**: contas,
personagens e guildas são compartilhados entre os canais. É por isso que o mesmo
personagem aparece em qualquer canal — o que muda é em qual processo ele entra.

## 2. O que muda de fato entre canais: `ServerIndex`

Cada `TMSrv` recebe um índice zero-based (`ServerIndex`) e o legado o usa como
identidade do canal em três lugares:

| Uso | Evidência |
|-----|-----------|
| Cidadania: `Citizen = ServerIndex + 1` (1-based) | `_MSG_Quest.cpp:2436` |
| NPC de outra cidade recusa o serviço | `Server.cpp:2891` (`Citizen != ServerIndex + 1`) |
| Id de guilda carrega o canal nos bits altos (`serverIndex * 4096`) | `ProcessDBMessage.cpp:760-790`, `handler/dispatch.go` (`ServerIndex`) |

O `Citizen` é persistido no save (`MobExtra.Citizen`), então a cidadania sobrevive
a relog e a troca de canal. O que o jogador ganha com ela:

- bônus de drop/caça no canal de cidadania — `_DN_TOWNSPEOPLE` = "Por ser cidadão
  deste servidor você ganhou %d Bônus Drop." (`Language.txt:423`) e
  `_DN_TOWNBOUNS` = "Bônus Drop (Cidadania) deste canal é %d." (`Language.txt:432`);
- pode criar guilda naquele canal (`handler/guild.go:81` exige `Citizen != 0`).

### Como se compra

Kibita (Merchant 74, em Armia) cobra **4.000.000 de gold** de quem tem `Citizen == 0`
e grava `ServerIndex + 1` (`_MSG_Quest.cpp:2433-2437`). O comando de sussurro
`getout` zera a cidadania (`_MSG_MessageWhisper.cpp:57-64`, `_DN_NO_TOWNSPEOPLE`).

No port isso foi portado nesta rodada: `handler/misc.go` → `kibita` (cidadania →
Soul permanente), e o canal vem da flag nova `-server-index`/`W2PP_SERVER_INDEX`.

## 3. "Canal novato" (canal verde)

No WYD Global o canal verde é o **canal novato**, e em produção ele é um **canal
fixo e identificado** — não uma rotação:

| Regra do canal novato | Fonte |
|---|---|
| lojas pessoais podem ser abertas; **não existem áreas PvP** | guia para novatos: <https://feedback.kersef.com/en/knowledge-bases/21/articles/58186-beginners-guide> |
| o portal do Hall do Kefra **só abre no canal novato** (todos os canais podem matar o Kefra) | Dungeon Kefra: <https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/21703-dungeon-kefra> |
| o **Deserto Desconhecido** só é acessível pelo canal novato | <https://feedback.kersef.com/en/knowledge-bases/21/articles/49456-novo-mapa-deserto-desconhecido> |
| o **coliseu das 22:00** é acessível pelo canal novato | Patchlist 16/01/2020: <https://wydglobal.raidhut.com/pt-br/576> |
| a **Guerra de Reinos** acontece no servidor Novato | <https://feedback.kersef.com/en/knowledge-bases/21/articles/21552-guia-guerra-de-reinos> |
| a **invasão de Noatun** ocorre no canal *seguinte* ao novato — "se o novato for o Canal 1, a invasão ocorre no Canal 2" | Guia de Guilds e Guerras: <https://wydglobal.raidhut.com/pt-br/2119> |

O último item é a prova de que o número do canal novato é **fixo**: outros eventos
se referenciam a ele por posição (`novato + 1`). Com isso, o canal novato é uma
propriedade permanente de um canal, não um estado que troca de lugar.

Isso é **separado** do evento de EXP para novatos: no legado existe o comando de GM
`/gm +newbiehour` ("mais experiência para leveis baixos",
<https://www.webcheats.com.br/topic/741855-comandos-para-gm-em-servidor-wyd>), e no
OpenWYD existe a flag `world_event_config.newbie_event_enabled` (EXP +25% abaixo do
nível 100, ±15% global). Evento de EXP é do servidor inteiro; canal novato é
propriedade do canal.

No código legado **não existe uma flag "novato"**: o único discriminador por canal
é o `ServerIndex`, e o PvP é decidido por mapa/zona (`AttributeMap`, `Regions.txt`,
`handler/pvp_zone.go`). Portanto o canal novato é uma **configuração de servidor**
(mesmos mapas, PvP desligado + loja permitida), não um recurso do cliente.

Consequência prática para o OpenWYD: a flag é por processo e o gate vai no dano PvP
(`handler/newbie_channel.go`, gate em `handler/combat.go`). O canal verde é **fixo
por configuração**, como em produção: `-novice-channel` / `W2PP_NOVICE_CHANNEL`
(número do canal, 1-based; `0` desliga). Cada instância compara esse número com o
seu `-server-index`, então o mesmo valor vai para todos os containers.

No primeiro tick cada instância loga `novice channel role` (`role=novice|regular`,
`novice_channel`, `server_index`, `newbie_event`): é a prova explícita de qual
canal obedece, porque no canal verde o gate concorda com o portal e nunca há
transição para logar.

A rotação diária `(dia-1) % group-size` que existiu entre 05/11 e a padronização foi
**removida**: era escolha nossa, não comportamento de produção, e misturava duas
coisas independentes. O canal verde controla **apenas o sistema de novato** (evento
newbie + recusa de dano PvP); a lista de canais do cliente é estática e mostra os
dois canais sempre.

## 4. Montar o OpenWYD com 4 canais (plano)

Estado atual: **um** `tmserver` (`docker-compose.yaml`, porta 8281 + status :80),
`-server-index` default 0, então todo mundo é cidadão do canal 1.

Passos, na ordem em que destravam valor:

1. **Identidade por processo** — feito: `-server-index` (`W2PP_SERVER_INDEX`).
   Cada canal precisa do seu valor (0..3) para a cidadania e o id de guilda.
2. **Quatro instâncias** — quatro serviços no compose, cada um com
   `-addr :828x` e `-status-addr :80xx` próprios, o mesmo `dbserver`/`binserver`,
   o mesmo volume de conteúdo somente-leitura, e `W2PP_SERVER_INDEX` distinto.
   *A conferir:* como o `serverlist editor` grava porta por canal — a serverlist
   hoje só lista IP (**UNVERIFIED** se o `.bin` aceita porta por entrada; se não
   aceitar, canais no mesmo host exigem IPs distintos).
3. **Serverlist de 4 canais** — quatro linhas `0 N <ip>[:porta]` no
   `client-tools/serverlist/servers.txt` e o `.bin` regenerado (nunca versionar o `.bin`).
4. **Canal novato** — feito para o núcleo (`-novice-channel` / `W2PP_NOVICE_CHANNEL`,
   ver §3): fixo por configuração, com gate no dano PvP. Falta liberar loja pessoal
   onde o canal exigir e as demais regras de produção (portal do Kefra, Deserto
   Desconhecido, coliseu) que valem só no canal novato.
5. **Bônus de cidadania** — hoje o texto existe mas a taxa não é aplicada:
   implementar o bônus de drop por `Citizen == ServerIndex+1` antes de vender
   cidadania como vantagem.
6. **Guerra entre canais** — `_DD_SERVERWAR`, `_DN_CHANNELWAR_BEGIN`
   (`Language.txt:426`, `:501`) existem no legado; não portado. Só faz sentido com
   mais de um canal.

Enquanto os passos 2-5 não estiverem prontos, os quatro canais seriam quatro
mundos isolados (sem PvP diferenciado e sem bônus de cidadania), o que é pior que
um canal único: o jogador paga 4M por uma cidadania que não entrega nada.

## 5. Riscos conhecidos

- **Estado dividido por processo**: cada `tmserver` tem seu próprio mundo em
  memória. Mobs, drops e eventos não são compartilhados; o que é compartilhado é o
  save (via `dbserver`) e as guildas. Duas sessões do mesmo personagem em canais
  diferentes precisam ser bloqueadas pelo login (`login_duplicate_test.go` cobre o
  caso no mesmo processo — **UNVERIFIED** entre processos, porque a checagem é
  local à instância).
- **Chat global é por canal** (`_MSG_MessageChat` sem roteamento entre instâncias).
- **Billing e gold** são únicos (um `binserver`), então economia é global.

---

## 7. Dois canais no compose (teste de cidadania)

O plano dos 4 canais (§6) começa com dois, que já dá para exercitar tudo o que depende
de `ServerIndex`: cidadania por canal, id de guilda, NPC de outra cidade.

```bash
docker compose up -d tmserver tmserver2
docker compose logs -f tmserver2               # "refine policy ... server_index=1"
```

| Serviço | Endereço de jogo | Porta interna | Status HTTP | `ServerIndex` | Cidadania vendida |
|---|---|---|---|---|---|
| `tmserver` | `192.168.100.3:8281` | 8281 | `192.168.100.3:80` | 0 | `Citizen = 1` |
| `tmserver2` | `192.168.100.4:8281` | 8282 | `192.168.100.4:80` | 1 | `Citizen = 2` |

O `127.0.0.1:8281/80` continua publicado para teste local. As portas são publicadas **por
endereço** (`IP:porta:porta`), nunca em `0.0.0.0`: os dois canais usam a mesma porta em IPs
diferentes (§7.1), e um bind genérico faria o segundo serviço falhar por conflito.

### 7.1 Os dois IPs na máquina (Windows, verificado em 05/11)

O `.bin` guarda só o host, então cada canal precisa de um IP. Na máquina de teste os dois
endereços convivem na mesma interface, como estáticos — atenção: `netsh ... add address`
**desliga o DHCP** da interface e deixa ela sem gateway/DNS; refaça tudo na mesma tacada:

```powershell
netsh interface ipv4 set address name="Ethernet 2" static 192.168.100.3 255.255.255.0 192.168.100.1
netsh interface ipv4 add address name="Ethernet 2" 192.168.100.4 255.255.255.0
netsh interface ipv4 set dns name="Ethernet 2" static 192.168.100.1
netsh interface ipv4 add dns name="Ethernet 2" 8.8.8.8 index=2
```

Conferência: `curl http://192.168.100.3/serv00.htm` e `curl http://192.168.100.4/serv00.htm`
devem devolver 200, e as duas `8281` aceitar TCP.

Os dois usam o **mesmo** `dbserver`/`binserver` e o mesmo `tmserver/content/` somente-leitura: conta,
personagem e save são únicos; o que muda é o mundo em memória e o canal da cidadania.

### O que testar

1. Comprar cidadania na Kibita (4.000.000 de gold) no canal 1 → `Citizen = 1`; a tela de
   personagem passa a mostrar a cidadania (o cliente lê o efeito 28 do item de corpo,
   `ProcessDBMessage.cpp:763`).
2. No canal 2: o mesmo personagem é estrangeiro (`Citizen = 1 != ServerIndex+1`); a Kibita de
   lá cobra de novo e grava `Citizen = 2`.
3. Falar com o NPC de outra cidade deve recusar enquanto a cidadania não for do canal.

### Canais: resolvido por IP, não por porta (05/11)

O campo da serverlist até cabe `ip:porta` (64 bytes), mas o cliente 7662 disca **8281** e sonda
**:80** fixos. Medido: com `192.168.100.4:8282` no `.bin` o cliente não conectou; com o segundo
canal na **mesma 8281 de outro IP** ele logou. Conclusão: **um IP por canal**, mesma porta.

O par de arquivos que o cliente usa (`client/`, ambos com backup do original):

```
# entrada do gerador ("<grupo> <canal> <endereço>")
0 0 http://192.168.100.3/serv00.htm
0 1 192.168.100.3
1 0 http://192.168.100.4/serv00.htm
1 1 192.168.100.4
```

```
# client/serverlist.txt — referência legível, em pares url + ip:8281
http://192.168.100.3/serv00.htm
192.168.100.3:8281

http://192.168.100.4/serv00.htm
192.168.100.4:8281
```

São dois processos distintos: `docker compose logs tmserver2` mostra
`refine policy ... server_index=1`, e o login aparece **só** nele quando o cliente entra pelo
canal 1. Os nomes UP-1/PVP-1 vêm do `sn.bin` do cliente, não do servidor — duas entradas
apontando para o mesmo IP caem no mesmo processo.

## 8. Gerar o `serverlist.bin` por código nosso

O `serverlist editor.exe` é Windows, fechado, e vive numa pasta que já saiu do git. O gerador
agora é nosso:

```bash
# client-tools/serverlist/servers.txt: "<grupo> <canal> <endereço>"
go run ./client-tools/serverlist -in client-tools/serverlist/servers.txt -out serverlist.bin
go run ./client-tools/serverlist -print serverlist.bin     # confere o que foi gravado

# ou, gerando no caminho padrão e instalando no cliente:
make client-serverlist            # neste Windows: mingw32-make client-serverlist
```

O arquivo tem exatamente 7.040 bytes (`10 grupos × 11 canais × 64`), com a mesma ofuscação do
`BASE_InitializeServerList` (`Basedef.cpp:6874`). Copie para `client/serverlist.bin` **com
backup** do original; o launcher do cliente original pode sobrescrevê-lo (skill
`wyd-client-connect`).

### Layout real das células (corrigido em 05/11)

Gerar um `.bin` com uma linha por canal **não funciona**: o cliente não mostra lista
nenhuma. O arquivo que funcionava, decodificado célula a célula, revelou o padrão:

| Célula (`grupo*11 + canal`) | Conteúdo |
|---|---|
| canal 0 do grupo | URL de status: `http://<ip>:8281/serv00.htm` |
| canal 1..10 do grupo | endereço do canal: `<ip>` |

Exemplo do arquivo original (4 grupos, o último com dois canais):

```
 0 "http://127.0.0.1:8281/serv00.htm"   1 "127.0.0.1"
11 "http://127.0.0.1:8281/serv00.htm"  12 "127.0.0.1"
22 "http://127.0.0.1:8281/serv00.htm"  23 "127.0.0.1"
33 "http://127.0.0.1:8281/serv00.htm"  34 "127.0.0.1"  35 "127.0.0.1"
```

Ou seja: o grupo 3 tinha **dois canais** (34 e 35). O `serverlist.txt`
(`<grupo> <canal> <conteúdo>`) descreve células, e o gerador só precisa gravar isso:

```
0 0 http://192.168.100.3:8281/serv00.htm
0 1 192.168.100.3
0 2 192.168.100.3:8282      # segundo canal: a porta no endereço é UNVERIFIED
```

O campo de endereço é uma string de 64 bytes, então `ip:porta` cabe — mas o cliente
disca 8281 historicamente. Enquanto isso não for confirmado, o caminho seguro para
dois canais é **um IP por canal**; a porta no endereço é o experimento.

### O `client/serverlist.txt` e o limite de 32 caracteres

A pasta do cliente tem **dois** arquivos, e eles andam juntos:

```
client/serverlist.txt     # pares, em texto
http://127.0.0.1:8281/serv00.htm
127.0.0.1:8281
                          # linha em branco separa um grupo
http://127.0.0.1:8281/serv00.htm
127.0.0.1:8281

client/serverlist.bin     # o que o cliente lê
célula 0 = "http://127.0.0.1:8281/serv00.htm"   (32 caracteres)
célula 1 = "127.0.0.1"                          (o porto NÃO vai para o bin)
```

Duas consequências medidas nesta rodada:

1. **A URL de status tem 32 caracteres úteis** no arquivo original
   (`http://127.0.0.1:8281/serv00.htm` = exatamente 32). Gerar
   `http://192.168.100.3:8281/serv00.htm` (37) estoura o campo e o cliente não
   lista nada. Com IP de LAN, use **sem porta**: `http://192.168.100.3/serv00.htm`
   (31) — o status é servido no `:80`, então a porta é dispensável.
2. O bin guarda **só o host**; a porta do jogo vem do host do cliente/URL. Por
   isso dois canais no mesmo IP não se distinguem pelo bin: cada canal precisa de
   um **IP próprio** (a porta no endereço é o experimento em aberto).

Receita para trocar o IP sem quebrar a lista:

```bash
# 1. bin: célula 0 = URL curta, célula 1 = host
printf '0 0 http://192.168.100.3/serv00.htm\n0 1 192.168.100.3\n' > dist/serverlist/serverlist-lan.txt
go run ./client-tools/serverlist -in dist/serverlist/serverlist-lan.txt -out dist/serverlist/serverlist-lan.bin

# 2. txt: pares, com ip:porta na segunda linha e linha em branco entre grupos
cp dist/serverlist/client-serverlist.txt client/serverlist.txt

# 3. bin
cp client/serverlist.bin client/serverlist.bin.bak
cp dist/serverlist/serverlist-lan.bin client/serverlist.bin
```

O `serverlist.bin.bak` guardado na pasta é o **original que funcionava** (127.0.0.1):
se qualquer variante falhar, copiá-lo de volta restaura a lista.

### Nomes de servidor e canal: `client/sn.bin`

O `WYD.exe` carrega **`sn.bin`** (string confirmada dentro do binário) e monta o rótulo de cada
canal como **`%s-%d`** — nome do servidor + número do canal. Não existe "nome de canal"
independente: chamar os canais de "Canal 1/Canal 2" exigiria o servidor se chamar `Canal`;
renomear o servidor para `NextGen` produz `NextGen-1` e `NextGen-2`.

Layout medido (arquivo de 143 bytes): tabela de nomes com **9 bytes por posição** (8 caracteres
úteis + NUL), seguida de uma cauda binária de contadores que **não** deve ser tocada:

| Posição | Conteúdo | Papel |
|---|---|---|
| 0 | `Reserved` | placeholder; o primeiro byte é marcador do arquivo (`0xD6`), não sobrescrever |
| 1 | `UP` | nome do grupo 0 da serverlist |
| 2 | `PVP` | grupo 1 |
| 3 | `WYDCDK` | grupo 2 |

Depois dos 99 bytes de nomes vem a **tabela de contadores**: 11 `int32`
little-endian (offsets 99, 103, 107…). O `WYD.exe` lê o endereço de cada linha da
célula `grupo*11 + canal` da serverlist e escreve o rótulo `%s-%d`; célula vazia
ou que não é host vira o texto `Inexistente` (`client/UI/strdef.bin`).

Duas medições valem mais que a leitura do disassembly:

- o arquivo de fábrica traz `2,3,4` e a lista aparece com **três** linhas sob o
  primeiro grupo;
- gravar `0,1,1` faz o cliente **não listar nada**. O índice 0 em zero derruba a
  lista inteira, então ele não pode ser zerado (a ferramenta recusa).

O contador é a quantidade de canais do grupo; deixe o índice 0 como vem de
fábrica (zerado, ele derruba a lista).

### 8.1 O cliente escolhe o canal pelo calendário

Decodificado em `0x46d6ce-0x46d9cc`: o cliente **não lista um canal por linha**.
Ele monta, para cada grupo (servidor), uma única linha cujo sufixo é o canal do
dia:

```
buffer 16 bytes; GetLocalTime(&st)                 ; SYSTEMTIME
dia = st.wDay                                      ; uint16 em +6
canal = dia % quantidade_de_células_do_grupo        ; 0 → quantidade
rótulo = sprintf("%s-%d", nome_do_grupo, canal)     ; 0x6143d8 = "%s-%d"
endereço = célula[grupo*11 + canal]                 ; o cliente disca esse
```

`quantidade_de_células` conta as células **1..10** com conteúdo (a célula 0, que
guarda a URL de status, não entra).

Esse cálculo do dia **não** é o que numera as linhas na tela: com dia 5 e dois
canais a fórmula daria `1` para as duas, e a tela mostrou `1` e `2` (medido em
05/11, ver abaixo). A fórmula segue **UNVERIFIED** como origem do número; a
hipótese atual é que ele acompanhe a posição do slot/célula.

**Quais** linhas aparecem — e qual célula cada uma lê — vem dos contadores do
`sn.bin`, não das células: a rotina em `0x4b23d9` conta quantos contadores são
não-zero (`n`, `0x4b2403`) e usa `contadores[slot] - 1` como **índice de grupo** do
slot (`0x4b2421`, `0x4b242b`). O cliente desenha **um slot por contador não-zero**,
cada um lendo o grupo indicado por ele:

| contadores | grupo de cada linha | medição na tela (célula do dia do grupo) |
|---|---|---|
| `[2 3 4]` | 3, 2, 1 | três linhas, a primeira `NextGen` |
| `[2 3]` | 2, 1 | duas linhas |
| `[2 2]` | 1, 1 | **dois** `NextGen` (o mesmo grupo duas vezes) |
| `[2 0 …]` | 1 | uma linha `NextGen` e `Inexistente` |
| `[3 3]` | 2, 2 | lista vazia (grupo 2 sem célula 0) |
| `[1 2 0]` | 1, 0 | uma linha |

Ou seja: **uma linha da lista é um grupo**, e um grupo é um canal. Dois contadores
apontando para o mesmo grupo **não** criam dois canais — mostram o mesmo grupo duas
vezes. Para exibir dois canais são necessários dois grupos, cada um com o seu
endereço, e contadores que apontem para eles (`[2 3]` → grupos 2 e 1, ou, com dois
contadores, um canal por grupo preenchido).

O mapeamento contador → grupo foi lido no disassembly (`0x4b23d9`–`0x4b242b`):

```
n = 0
for (i = 0; i < 11; i++)                               ; 0x4b23e0–0x4b2419
    if (counters[i] != 0) n++                          ; 0x4b23fb / 0x4b2403
grupo(linha) = counters[n - 1 - linha] - 1             ; 0x4b241b subtrai o parâmetro
                                                       ; 0x4b2421 carrega o contador
                                                       ; 0x4b242b subtrai 1
```

O parâmetro (`[ebp+0xc]`) é o índice da linha que o chamador está desenhando, então as
linhas consomem os contadores **de trás para frente**: a primeira linha usa o último
contador não-zero. O rótulo tem dois caminhos (`0x46db4e` e `0x46dc6d`):

```
nome_do_grupo + "-" + (dia % células_do_grupo)   ; tabela de nomes de canal vazia
nome_do_grupo + "-" + nome_do_canal             ; tabela de nomes de canal preenchida
```

A tabela de nomes de canal fica em `0x277bb10` (10 grupos × 10 canais × 9 bytes). Ela é
apenas **zerada no boot** — `memset` em `0x549406`, 0x3de bytes — e não é escrita em
nenhum ponto do executável: os nomes de canal chegam em runtime (rede/DLL), não de
`sn.bin` nem de `serverlist.bin`. Consequência prática: **só com os dois arquivos, cada
grupo rende uma linha**, e o número depois do hífen é o canal do dia, não o índice da
célula.

Medições no cliente em 05/11/2026: `[2 3]` com os grupos 1 e 2 preenchidos (uma célula
cada) → duas linhas `NextGen-1`, e o login completou nas duas; `[3]` com o grupo 2 e duas
células → duas linhas `Inexistente`. O `Inexistente` é o texto de fallback do rótulo do
grupo (`names[grupo]`), não falta de endereço.

### 8.1.1 A tabela `0x277bb10` é a lista de canais da janela do ESC

Medido em 05/11/2026 no dump completo do cliente:

| achado | evidência |
|--------|-----------|
| tamanho **990 bytes** = 10 grupos × **11** células × **9** bytes | `0x549409: push 0x3de` antes do `memset(0x277bb10, 0, 0x3de)` em `0x549406` |
| é **zerada no init**, não vem pronta do arquivo | `0x549410: push 0x277bb10` + `call 0x5c5b80` (memset) |
| índice = `grupo * 90 + (contador - 1) * 9` | `0x4b2a71: imul ecx,ecx,0x5a`; `0x4b2a8a: imul edx,edx,0x9`; `0x4b2a8d: movsx eax,[ecx+edx+0x277bb10]` |
| o nome do grupo fica em `0x277be94 + grupo * 9` | `0x4b2ab7..0x4b2aba` |
| célula **vazia** → preenchida com o nome do grupo (`sprintf`, formato em `0x61575c`) | `0x4b2a95: test eax,eax` / `jne 0x4b2b6f` |
| célula **preenchida** → usa o texto de 9 bytes dela (`0x61576c`) | `0x4b2b91: lea edx,[eax+ecx+0x277bb10]` |

Consequências:

1. **não adianta buscar o literal `10 bb 77 02`** para achar quem escreve a tabela: o código chega
   nela por índice (`base + grupo*90 + célula*9`), e o único `push` do endereço é o `memset` de init.
   Era por isso que a busca binária anterior dava negativo.
2. **a célula guarda um texto por canal**, independente do endereço do `serverlist.bin`. Por isso a
   janela do ESC lista todos os canais e ainda assim não conecta: o destino desse caminho não é o
   endereço da grade, e sim o protocolo de crossover abaixo.

### 8.1.2 O crossover (troca de canal em jogo)

Nenhum pacote do cliente chegou ao servidor durante a tentativa pela janela do ESC (logs de
`tmserver`/`tmserver2`, 05/11/2026): essa janela **não pede nada** ao servidor. O caminho que o
legado usa é um comando, em três saltos:

```
/srv <n>  (_MSG_MessageWhisper.cpp:75-95)
  tmsrv → db   MSG_DBServerChange { NewServerID, Slot }          id 20, FLAG_GAME2DB
  db    → tm   MSG_DBCNFServerChange { AccountName, Enc }        id 42, DB2GAME|GAME2CLIENT
               Enc = "*<srv> <Enc[0..7]> <slot>"   (CFileDB.cpp:1996-2011)
  db           grava TempKey = Enc                            (52 bytes)
  cliente      reconecta e manda Enc no campo Zero[52] do MSG_AccountLogin
  db           TempKey == Zero → ChangeServer = 1, login sem senha (CFileDB.cpp:662-812)
```

**Implementação nossa** (`/srv`, sem RPC novo): o token é *assinado* com `W2PP_CROSSOVER_KEY` em vez
 de persistido, então não é preciso coluna nova nem RPC de troca em `api/db/v1/db.proto`.

| peça | onde |
|------|------|
| token `~x1:<canal>:<slot>:<exp>:<hmac>` (≤ 52 bytes, TTL 2 min) | `internal/crossover` |
| frame `MSG_DBCNFServerChange` (42) com `AccountName[16]+Enc[52]` | `tmserver/internal/protocol/serverchange.go` |
| comando `/srv <n>` que emite o frame | `tmserver/internal/handler/serverchange.go` |
| token aceito no campo `Zero[52]` do login, enviado ao db como senha | `handler/login.go` |
| validação do token no login | `dbserver/internal/grpcsrv/server.go` (`WithCrossoverKey`) |

### 8.2 Referência: o `serverlist.bin` do WYD Global

Arquivo real de um servidor em produção (launcher do WYD Global), medido em
05/11/2026:

| Arquivo | Tamanho | Conteúdo |
|---|---|---|
| `serverlist.bin` | **7044** | 4 bytes de cabeçalho (`20 00 20 00`) + a grade de 7040 |
| `sn.bin` | 143 | 3 nomes e contadores `[1 2 0 0 0 0 0 3 0 0 0]` |

Os nomes do `sn.bin` deles: posição 0 = `Global` (primeiro byte é marcador, como
no nosso `\xd6eserved`), posição 1 = `Global`, posição 2 = nome em coreano.Os contadores deles são `[1 2 0 0 0 0 0 3 0 0 0]`, e o grupo 1 tem 6 canais — ou
seja, os contadores de produção não batem com a quantidade de canais; o
significado no build deles segue **UNVERIFIED**.

No nosso build, medições da mesma tela (células do grupo 1: url, `.3`, `.4`):

| Contadores no `sn.bin` | Resultado na tela |
|---|---|
| `[2 3 4]` | funciona: cabeçalho, linha do canal e linhas `Inexistente` |
| `[2 3 0]` | funciona: duas linhas |
| `[1 2 0]` | só cabeçalho e `Inexistente`, sem canal |
| `[3 3 0]` | lista vazia |
| `[0 1 1]` | lista vazia |

**Não há regra confirmada.** Os valores de fábrica `2,3,4` são os únicos que se
mantiveram estáveis; a hipótese "1 + número de canais" foi refutada por `[3 3 0]`.
Até alguém ler o consumidor inteiro (`0x4b2xxx` do cliente, que usa
`0x277bef8`), a orientação é **não mexer nesses três contadores**.

Layout extraído do arquivo deles (chave recuperada em
`docs/migration/wyd-global-serverlist.txt`):

```
1 0 https://server-status.raidhut.com/wyd-s01.php   (status, HTTPS)
1 1 92.38.151.230   1 2 92.38.151.231   1 3 92.38.151.232
1 4 92.38.151.233   1 5 92.38.151.234   1 6 92.38.151.235
```

Conclusões que valem para o nosso servidor:

- **um grupo com vários canais é o layout normal** de produção; o que muda entre
  os canais é o endereço (IPs consecutivos), não uma linha extra na grade;
- a célula 0 é a URL de status e pode ser HTTPS;
- o rótulo de cada canal (`Nome-1`, `Nome-2`, …) de produção ainda **não** foi
  reproduzido no nosso build: com um grupo e duas células a tela mostrou dois
  servidores e um canal `Inexistente` (§8.1). Segue **UNVERIFIED** se o número vem
  de `dia % N` (o cliente calcula isso em `0x6143d8`) ou do índice da célula;
- **um só grupo preenchido**. Medido no nosso build em 05/11/2026: com canais em
  dois grupos (1 e 2) a tela mostrou apenas o cabeçalho `NextGen` e **nenhum
  canal**; com os mesmos canais todos no grupo 1, a linha voltou. O desenho
  depende do grupo selecionado, não de cada grupo ter a sua lista;
- a chave de ofuscação é **por posição dentro da célula** e muda entre builds —
  a deles não decodifica com a nossa tabela. Para ler um arquivo de outro build,
  recupere a chave pela moda byte a byte das células vazias.

### 8.3 Layout padronizado (espelha produção)

Nossos dois arquivos seguem o mesmo desenho do arquivo de produção, trocando apenas
o nome e a quantidade de canais. Duas diferenças são do **build** e não devem ser
copiadas:

| Diferença | WYD Global | Nosso cliente |
|---|---|---|
| Tamanho do `serverlist.bin` | 7044 (4 bytes de cabeçalho `20 00 20 00`) | **7040**, sem cabeçalho |
| Chave de ofuscação | `00 00 bb a4 …` | `d9 b4 aa b3 …` |
| Assinatura do `sn.bin` (byte 0) | `0xAB` | `0xD6` |

Estado alvo dos nossos arquivos:

```
client-tools/serverlist/servers.txt (entrada canônica; gera client/serverlist.bin)
1 0 http://192.168.100.3/serv00.htm
1 1 192.168.100.3
1 2 192.168.100.4

client/sn.bin (143 bytes)
slot 0  d6 65 73 65 72 76 65 64 00   assinatura + "eserved" (mantido como veio)
slot 1  4e 65 78 74 47 65 6e 00 00   "NextGen"
slot 2..10  zerados
contadores  [2 3 4 0 0 0 0 0 0 0 0]   (valores de fábrica: os únicos aceitos pela tela)
```

O primeiro byte da posição 0 é a assinatura do build e ocupa o lugar do primeiro
caractere do nome — é assim no arquivo de fábrica (`\xd6eserved` para `Reserved`) e
no de produção (`\xablobal` para `Global`). A ferramenta preserva esse byte e
grava o nome a partir do segundo. Confirmado no disassembly: `0x54b688` lê `0x63` bytes de nomes e
`0x2c` de contadores; `0x46d770-0x46d7b7` itera `canal < 11`, pula célula vazia e
testa o contador contra zero.

Ferramenta própria (nunca editar à mão, e backup antes):

```bash
cp client/sn.bin client/sn.bin.bak
go run ./client-tools/servernames -print client/sn.bin
go run ./client-tools/servernames -in client/sn.bin -set 1=NextGen -counts 2,2
```

`-set` limita a 8 caracteres (limite da posição); `-counts` grava os contadores
crus a partir do índice 0 e zera o resto.

Em 05/11 o slot 1 virou `NextGen` e a serverlist passou a ter **um grupo com dois canais**
(`0 0` URL de status, `0 1` = `.3`, `0 2` = `.4`) — é assim que o jogador vê dois canais dentro
de um servidor, em vez de dois servidores com um canal cada (a versão de dois grupos mostrava
`UP-1` e `PVP-1`).

Como uma linha é um grupo (§8.1), os dois canais são **dois grupos**, um canal cada, e o
`servers.txt` canônico preenche os grupos 1 e 2: grupo 1 = `192.168.100.3` (tmserver, canal
novato) e grupo 2 = `192.168.100.4` (tmserver2). Cada grupo tem **uma** célula de canal, o
que fixa o endereço: `dia % 1` é sempre 1, então nenhum canal roda entre dias.

Trocar de canal na tela é escolher a outra linha — em um grupo com todas as células no
mesmo IP, trocar de canal não troca de servidor (medido no experimento de 05/11, com todos
os grupos preenchidos: o login completou nos dois tmservers, um por paridade de grupo).

Nomes em uso: `sn.bin` posição 1 = `NextGen`, posição 2 = `NextGen2`. O campo tem **8
caracteres úteis** (9 bytes com NUL), então `NextGen-1` não cabe como nome de servidor —
com hífen só cabe até 8. O cabeçalho de cada entrada é o nome do grupo (posição `grupo`) e
o rótulo do canal sai do nome da posição 1 seguido do número do slot, então a tela mostra as
entradas `NextGen2` e `NextGen`, ambas com o canal `NextGen-1`.

Duas limitações vêm do cliente, não dos arquivos:- **ordem das entradas**: a primeira linha consome o **último** contador não-zero, então a
  ordem na tela é o inverso da ordem dos valores no `sn.bin`. Com `[3 2]` o grupo 1 (novato)
  aparece primeiro; com `[2 3]` é o grupo 2 que vem primeiro;
- **os canais de uma entrada são as células do grupo** (medido em 05/11/2026): um grupo
  com as células 1..10 preenchidas mostra `NextGen-1` até `NextGen-10` em sequência, e
  célula vazia entre elas vira a linha `Inexistente`. Não é o contador que define a
  quantidade — ele só seleciona o grupo;
- **o número do canal é `dia % células`** e não o índice, por isso um grupo com 10 células
  numerou todas as linhas com 5 (dia 5);
- **a lista de canais só é desenhada com 2 ou mais contadores não-zero** (medido em
  05/11/2026): com `[2]` e um grupo de duas células a tela mostrou apenas `NextGen`, sem
  nenhum canal; com `[3 2]` a mesma lista passou a mostrar os canais. Ou seja, o mínimo
  útil é dois contadores — um por entrada;

O rótulo `NextGen-2` como canal dentro de uma única entrada `NextGen` continua fora de
alcance: depende do segundo caminho (`%s-%s`) e da tabela `0x277bb10`, que nenhum módulo do
cliente escreve (nem o exe, nem `ClientPatch.dll`, nem as demais DLLs — a busca pelo padrão
`10 bb 77 02` não casou em nenhuma).

### Sistema de novato: o que já existe

| Peça | Onde | Estado |
|---|---|---|
| Evento newbie (EXP +25% < nível 100, ±15% global, mobs < 120 com 3/4 de HP) | `world_event_config.newbie_event_enabled`, poll de 15 s | **ligado** em 05/11 (version 6, aplicado nas duas instâncias) |
| Quest do Campo de Treino (`TREINADORNEWBIE1..4`) | `handler/newbiequest.go`, `character.newbie_quest` | implementada |
| Canal novato (verde) sem PvP | `newbie_channel.go`, gate em `combat.go` | **implementado**: canal **fixo** por `-novice-channel` / `W2PP_NOVICE_CHANNEL` (1-based, `0` desliga), como em produção; nele o evento liga e o dano PvP é recusado (duelos incluídos). A rotação diária foi removida |

O caminho certo para ligar/desligar o evento é o portal (Eventos → newbie), que grava auditoria
e incrementa a versão; o `tmserver` loga `world event config reloaded version=N` em cada
instância. Um `UPDATE` direto no banco funciona, mas só se a versão em `world_event_meta`
subir junto — e o `-c` do psql roda tudo numa transação, então um erro no meio desfaz os
`UPDATE`s anteriores.
