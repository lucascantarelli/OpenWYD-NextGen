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
