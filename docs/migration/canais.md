# Servidores, canais, cidadania e o "canal novato"

> Escrito em 05/11/2026 a partir do código legado (`Source/Code/TMSrv`), do conteúdo
> em `Release/Common/serverlist.txt` e da documentação pública do WYD. Onde não há
> evidência no repositório a marcação é **UNVERIFIED**.

## 1. O que o jogador chama de "servidor" é uma linha da serverlist

O cliente não descobre nada: ele lê `serverlist.bin`, gerado pelo
`serverlist bin/serverlist editor.exe` a partir de `Release/Common/serverlist.txt`:

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

Na documentação pública do WYD Global, o canal verde é o **canal novato**: lojas
podem ser abertas e **não existem áreas PvP** — o jogador explora sem ser morto por
outro jogador
(<https://feedback.kersef.com/en/knowledge-bases/21/articles/58186-beginners-guide>).

No código legado **não existe uma flag "novato"**: o único discriminador por canal
é o `ServerIndex`, e o PvP é decidido por mapa/zona (`AttributeMap`, `Regions.txt`,
`handler/pvp_zone.go`). Portanto o canal novato é uma **configuração de servidor**
(mesmos mapas, PvP desligado + loja permitida), não um recurso do cliente.

Consequência prática: para o OpenWYD ter canal novato é preciso uma flag por
processo (ex.: `-no-pvp`), aplicada no gate de dano PvP e nas zonas PvP — hoje
**não implementada**.

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
   `Release/Common/serverlist.txt` e o `.bin` regenerado (nunca versionar o `.bin`).
4. **Canal novato** — flag por processo + gate no dano PvP e nas zonas PvP, e
   liberar loja pessoal onde o canal exigir. Sem isso o "canal verde" não existe.
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
docker compose up -d tmserver tmserver2        # 8281/80 e 8282/8080
docker compose logs -f tmserver2               # "refine policy ... server_index=1"
```

| Serviço | Porta de jogo | Status HTTP | `ServerIndex` | Cidadania vendida |
|---|---|---|---|---|
| `tmserver` | 8281 | 80 | 0 | `Citizen = 1` |
| `tmserver2` | 8282 | 8080 | 1 | `Citizen = 2` |

Os dois usam o **mesmo** `dbserver`/`binserver` e o mesmo `Release/` somente-leitura: conta,
personagem e save são únicos; o que muda é o mundo em memória e o canal da cidadania.

### O que testar

1. Comprar cidadania na Kibita (4.000.000 de gold) no canal 1 → `Citizen = 1`; a tela de
   personagem passa a mostrar a cidadania (o cliente lê o efeito 28 do item de corpo,
   `ProcessDBMessage.cpp:763`).
2. No canal 2: o mesmo personagem é estrangeiro (`Citizen = 1 != ServerIndex+1`); a Kibita de
   lá cobra de novo e grava `Citizen = 2`.
3. Falar com o NPC de outra cidade deve recusar enquanto a cidadania não for do canal.

### Limite conhecido (a verificar no cliente)

O campo de endereço da serverlist é uma string de 64 bytes por canal — cabe `ip:porta`. O
cliente 7662, porém, historicamente disca **8281** e sonda **:80**; se ele ignorar a porta,
dois canais no mesmo host não são alcançáveis e cada canal precisa de um IP próprio (ou de um
proxy na frente). O teste é objetivo: gerar o `.bin` com `0 1 <ip>:8282` e ver se o cliente
conecta no canal 1 (`docker compose logs -f tmserver2` deve mostrar o login). Se não conectar,
o caminho é IP por canal, não porta.

## 8. Gerar o `serverlist.bin` por código nosso

O `serverlist editor.exe` é Windows, fechado, e vive numa pasta que já saiu do git. O gerador
agora é nosso:

```bash
# Release/Common/serverlist.txt: "<grupo> <canal> <endereço>"
go run ./client-tools/serverlist -in Release/Common/serverlist.txt -out serverlist.bin
go run ./client-tools/serverlist -print serverlist.bin     # confere o que foi gravado
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
