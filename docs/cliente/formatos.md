# Formatos de dados do cliente — primeira estrutura

> Passada real nos formatos ainda opacos, com amostras do build
> `04c9d2a1…27ddf8` (2026-11-07). Artefato bruto: `docs/cliente/gerado/formatos.json`.
> O que os bytes não sustentam continua declarado **opaco**.

## 1. `.wyt` — tela de interface

Amostras: `NUI/ServerList.wyt`, `NUI/ccmode.wyt`, `UI/AutoTrade.wyt`, `UI/Combine102.wyt`.

```
57 54 31 30  00 00 02 00  00 00 00 00  00 00 00 00
68 02 2c 01  20 08 00 00  00 ff 00 00  00 ff 00 00
W  T  1  0      versão       zeros        zeros
                largura  altura  bpp
```

| Offset | Campo | Evidência |
|---:|---|---|
| 0–3 | magic **`WT10`** | idêntico nas 4 amostras |
| 4 | versão **`0x00020000`** | constante nas 4 amostras |
| 16 | **largura** em pixels | `0x0268`=616 (`main`), `0x011f`=287 (`ServerList`), `0x00e4`=228 (`Combine102`) |
| 18 | **altura** em pixels | `0x012c`=300, `0x0118`=280, `0x0139`=313 |
| 20 | **bits por pixel** (byte): 24 ou 32 | 32 = BGRA, 24 = BGR; os arquivos de 32 bpp têm `0x08` em @21 |
| 22 | início dos **pixels**, `largura*altura*(bits/8)` bytes | raster sem compressão |
| fim−8 | marcador final `00`×8 | idêntico nas 256 amostras que fecham a fórmula |

**Prova (2026-11-09): o `.wyt` é raster puro, não contêiner de controles.**

| Verificação | Resultado |
|---|---|
| fórmula `22 + largura*altura*(bits/8) + 8 = tamanho` | fecha em **256/259** `.wyt` de `client/NUI` + `client/UI` (`NUI` 8/8) |
| dimensões reais | `NUI/ServerList.wyt` 287×280 32 bpp; `NUI/main.wyt` 616×300 32 bpp; `UI/PCK_Background.wyt` 800×600 24 bpp |
| é imagem, não ruído | `ServerList.wyt`: 80.360 pixels, 12.273 cores distintas, alfa 100% opaco |
| conversão para imagem | `node tools/client/wyt.mjs client/NUI/ServerList.wyt --png tmp/wyt/ServerList.png` → moldura e duas colunas de slots legíveis |
| exceções | `UI/NewAmul.wyt` (8 bpp) e `UI/hanui.wyt`/`UI/joyui.wyt` (16 bpp) — o decoder **recusa** com erro, não chuta |

O que o arquivo **não** contém: texto, fontes, posição de controle e eventos — o `WYD.exe` desenha
tudo por cima. Trocar pixel muda o desenho, não o layout. Relatório das 8 telas de `NUI`:
`docs/cliente/gerado/wyt.json`; ferramenta: `tools/client/wyt.mjs` (`--png`, `--bmp`, `--scan`).

## 2. `.msa` — malha

Amostras: `Mesh/Bownew1.msa` (13.839 B), `Mesh/CharmPowder.msa` (1.435 B).

```
12 01 00 00  20 00 00 00  01 00 00 00  00 00 00 00
00 00 00 00  ba 01 00 00  00 00 00 00  5c 01 00 00
b0 20 63 6f 70 79 2e 74 67 61 00            \copy.tga
```

| Offset | Campo | Evidência |
|---:|---|---|
| 0 | **`0x112`** (274) | idêntico nas duas amostras — candidato a versão do formato |
| 4 | `0x20` (32) | idêntico nas duas amostras |
| 8 | `1` | idêntico nas duas amostras |
| 20 | variável | 442 na amostra 1, 60 na amostra 2 |
| 28 | variável | 348 na amostra 1, 32 na amostra 2 |
| 34 | **nome de textura ASCII**, terminado em NUL | `copy.tga` e `\ori.psd` nas duas amostras |

**Opaco:** vértices, esqueleto, animação, materiais e índice de texturas.

## 3. `sn.bin` — nomes de canal (143 B)

```
d6 65 73 65 72 76 65 64 00 4e 65 78 74 47 65 6e 00 00 4e 65 78 74 47 65 6e 32 00 …
.  e  s  e  r  v  e  d \0 N  e  x  t  G  e  n \0 \0 N  e  x  t  G  e  n  2 \0
```

| Observação | Evidência |
|---|---|
| 11 nomes de **9 B** a partir do **offset 0** (não há cabeçalho) | `Reserved\0\0`, `NextGen\0`, `NextGen2\0`… |
| Primeiro byte `0xd6` é o `R` de `Reserved` **sobrescrito**, não um campo | `d6 65 73 65 72 76 65 64 00` = `Öeserved`, e a posição 0 é desenhada como a primeira linha de servidor |
| 11 contadores `int32` little-endian no offset **99** (11 × 9) | `[... 99] = 03 00 00 00  02 00 00 00 ...` |
| `contadores[0]` = quantas linhas de canal o cliente tenta desenhar (`canais + 1`; o `+1` é o cabeçalho) | `[3, 2, …]` com 2 canais desenha 2 linhas de canal |
| slot 0 é o **cabeçalho reservado** (`Reserved` com o `R` sobrescrito por `0xd6`), não um servidor | nome `Öeserved`, contador 3 |
| slots 1..10 são servidores e o **slot é o próprio grupo** do `serverlist.bin`; o contador é `grupo + 1` | slot 1 com contador 2 → grupo 1 (canais 1 e 2) |

**A inversão é do cliente — confirmada no binário (2026-11-08).** A rotina da lista
(`0x46d6ce`–`0x46dc76`) percorre as células `i = 1..N` do grupo exibido, mas monta o rótulo de cada
linha com o índice **`last - i`** quando esse grupo é o último preenchido (`0x46dafe`, cálculo em
`0x46db16`–`0x46db83`): os nomes são consumidos em ordem decrescente — `[2]`, `[1]`, `[0]`, como
sugerido na análise. A linha que sobra cai no slot reservado, cujo contador fica `-1`, e o cliente
troca o texto por uma string de runtime em `0x109c7b8` (`0x46ddd0`) — o `Inexistente`. No caminho
em que o grupo exibido **não** é o último (`0x46dc6d`), o rótulo usa `nomes[grupo]` e a célula
`i-1`, sem inversão. Se `nomes[grupo]` estiver vazio o rótulo vira `"Canal N Grupp"` (`0x46dc92`),
e com contador negativo a linha vira `Inexistente` (`0x46ddd0`).

Gerar agora é reprodutível — o gerador do `.bin` também escreve o `sn.bin`, preservando o
cabeçalho reservado (nome, `0xd6` e `contador = canais + 1`) e gravando um slot por grupo usado
(`contador = grupo + 1`):

```bash
node tools/client/serverlist.mjs --in tools/client/serverlist-servers.txt \
  --out client/serverlist.bin --sn-out client/sn.bin
node tools/client/serverlist.mjs --sn client/sn.bin --bin client/serverlist.bin  # inspeciona
node tools/client/serverlist.mjs --check client                                # o que o cliente vai desenhar
```

Os **nomes** dos slots vêm de `docs/arquitetura/identidade.tokens` (`servidor.<slot>`), não de
constante no gerador: o mesmo arquivo alimenta `wyd-launcher` (Rust) e portal, e
`crates/wyd-launcher/tests/contrato_identidade.rs` compara o `sn.bin` instalado com o gerado da
identidade. Nome com mais de 8 caracteres reprova (o campo tem 9 B com NUL).

Estado atual de `client/sn.bin`: slot 0 reservado com contador 3 (2 canais + cabeçalho), slot 1
`WYD` (identidade) com contador 2 → grupo 1 com canal 1 (`192.168.100.3`) e canal 2 (`192.168.100.4`). O
**host** de cada canal vem da célula do `.bin`, não do `sn.bin`; quem define é o registro de origem
(`tools/client/serverlist-servers.txt`) e o launcher regenera o `.bin`. Em 2026-11-09 a máquina
perdeu temporariamente esses IPs de LAN: os canais foram publicados também em loopback
(`127.0.0.1`/`127.0.0.2`) e os dois endereços antigos voltaram como alias `/32` no adaptador de
loopback do Windows — assim um cliente já aberto com o `.bin` anterior conecta sem restart.

## 4. `serverlist.bin` — **decodificado** (2026-11-08)

7.040 B = `MAX_SERVERGROUP(10) × MAX_SERVERNUMBER(11) × 64 B`, o `sizeof(g_pServerList)` do
legado (`Source/Code/Basedef.cpp:6874`, `BASE_InitializeServerList`). O arquivo **não** usa a
tabela do CPSock: cada célula de 64 B é ASCII terminado em NUL com cada byte somado de
`obfuscationKey[63 - i]` (tabela estática de 64 bytes, `Basedef.cpp:6927`); decodificar é
subtrair a mesma tabela na ordem invertida.

| Célula (`grupo * 11 + canal`) | Conteúdo | Papel |
|---|---|---|
| canal 0 do grupo | URL de status, **≤ 32 caracteres úteis** | o cliente faz `GET` dela na porta 80 |
| canais 1..10 do grupo | host do canal | o cliente disca **8281 fixo** nesse host |

Consequências medidas no cliente 7662, não inferidas:

- **32 caracteres** para a URL: `http://192.168.100.3:8281/serv00.htm` (36) **não lista nada**;
  `http://192.168.100.3/serv00.htm` (31) funciona — o status é servido no `:80`, a porta é
  dispensável.
- **Um IP por canal**: `192.168.100.4:8282` no `.bin` não conecta; o segundo canal precisa da
  mesma `8281` **de outro IP**. Porta dentro da célula é UNVERIFIED.
- **Grupo = servidor**: servidores independentes (sistemas, banco, mundo) são grupos distintos;
  canais de um grupo compartilham o banco e diferem só na instância do mundo.

Quem decide **quais** linhas aparecem é o `client/sn.bin`, não o `.bin`: nomes de 9 B por posição
(`Reserved`, `NextGen`…) nos 99 primeiros bytes e **11 contadores `int32` little-endian** a partir
do offset 99. O slot 0 é o cabeçalho reservado e `contadores[0]` é quantas linhas de canal o
cliente tenta desenhar; nos slots 1..10, `contadores[slot] - 1` é o grupo do `.bin`. Os rótulos
saem de `sprintf("%s-%d", nome, canal)` (rotina em `0x46d6ce`–`0x46dc76`), com a inversão
descrita no §3; a linha cujo contador fica negativo vira `Inexistente`. Zerar `contadores[0]`
derruba a lista.

Gerador e inspetor próprios (porte do antigo `client-tools/serverlist` em Go, hoje na `main`):

```bash
node tools/client/serverlist.mjs --in tools/client/serverlist-servers.txt --out client/serverlist.bin
node tools/client/serverlist.mjs --print client/serverlist.bin     # confere célula a célula
```

O `.bin` **nunca é versionado** (`client/` fora do git); a entrada canônica é
`tools/client/serverlist-servers.txt` (`<grupo> <canal> <endereço>`) e o cliente mantém um
`serverlist.txt` legível em pares (`URL` + `host:8281`) como referência humana.

Desde 2026-11-08 a geração/instalação é do **launcher Rust**
(`crates/wyd-launcher/src/serverlist.rs` + `serverlist`/`launch` na CLI, ver
`launcher.md` §8.1); o Node segue como referência de operação e a paridade de **bytes** entre os
dois é teste (`crates/wyd-launcher/tests/golden_serverlist_paridade.rs`).

### 4.1 `serv00.htm` — o que o cliente faz com a resposta (captura estática)

O `GET` da célula 0 sai por WinInet: `InternetOpenA("MS")` → `InternetOpenUrlA(url, …, 0x4000000)`
→ **uma** `InternetReadFile` de `0x400` B com NUL no fim (`client/WYD.exe`, rotina `0x5429e9`).
Consequência de contrato: o corpo útil é de **até 1.023 caracteres** e precisa caber numa leitura
só — resposta longa ou por partes não é suportada pelo cliente.

Duas rotinas consomem esse texto:

| Rotina | O que faz | Evidência |
|---|---|---|
| `0x46d7xx` | `strtok(corpo, ",")` e grava **campo a campo** nas células seguintes do grupo (`%s`) — caminho que **não** governa a tela de seleção (ver abaixo) | `0x46d8b1` (`push $0x2c`), `0x46d9c4` |
| `0x4b013b` | `sscanf(corpo, "%d", …)` com 11 destinos inicializados em `-1` | `0x4b017d`, `0x4b01d9`/`0x4b01e5` |

A rotina `0x46d7xx` (`strtok` por vírgula) **não** governa o *server select*: medida contra o
cliente real, uma resposta com vírgula (`0,0`) preenche **só** o primeiro destino, porque o
`sscanf` de `0x46d9d6` para na vírgula. Quem decide as linhas da coluna Canal é o caminho
numérico descrito a seguir. Resposta vazia ou inalcançável deixa a célula como veio do `.bin`, e
o canal aparece vazio/`Inexistente` — o sintoma já medido com URL acima de 32 caracteres.

**O corpo é NUMÉRICO, não a lista de hosts (medido em 2026-11-08).** A rotina do *server select*
(`0x46d6ce`–`0x46de80`) busca a célula 0 do grupo exibido (`0x5429e9`, o mesmo `GET` de 0x400 B) e
consome o corpo com `sscanf(corpo, "%d", …)` em **10 destinos inicializados em `-1`**
(`0x46d9d6`–`0x46da54`). Cada destino é o contador da linha de canal daquele grupo e a linha cujo
contador continua `-1` é trocada por `Inexistente` (`0x46ddc0`–`0x46ddd0`). O host de cada canal
**não** vem do `serv00.htm`: vem da célula do `serverlist.bin` (§4).

Consequência de contrato: com dois canais o corpo é **um número por linha** — a contagem de
jogadores online **daquele** canal (sem ninguém conectado, `0\n0\n`). O número não é agregado do
grupo: cada linha conta `character_session` filtrada por grupo e canal, então nenhum canal publica
a presença do outro (`crates/wyd-server/src/status.rs::online_por_canal`). O separador importa: `sscanf("%d")` pula espaço/quebra de linha
entre as conversões, mas **para em vírgula**, então `0,0` e `192.168.100.3,192.168.100.4` deixam
todos os destinos depois do primeiro em `-1` e a coluna Canal desenha **um canal real e o seguinte
como `Inexistente`**. É exatamente o sintoma corrigido em `crates/wyd-server/src/channels.rs`
(`render_status_body`).

Servidor de referência do mesmo cliente 7662 (`w2pp-OpenWYD`, `Release/Common/serv00.htm`,
20 B): dez linhas de `1` (`31 0a` repetido) — confirma o leiaute de um número por linha, tantos
quantos canais o servidor publica.

**Referências externas conferidas em 2026-11-08** (o `.bin` de 7040 B é consenso entre elas):

| Fonte | `serverlist.bin` | O que mostra |
|---|---|---|
| `w2pp-OpenWYD` (mesmo alvo 7662) | 7040 B | grupo 0 com as células 0 e 1 preenchidas por IP; `Release/Common/serverlist.txt` no formato `<grupo> <canal> <ip>` |
| `open-wyd/ows-clientdata` | 7040 B | 3 grupos × 3 células (todas no mesmo IP) e `sn.bin` em texto |
| cliente real WYD Global/Raidhut (`%LOCALAPPDATA%\wyd_launcher`) | **7044 B** | 4 bytes de cabeçalho + 7040, cifrado com outra tabela (nenhum deslocamento da nossa decodifica) — referência de tamanho, não de leiaute |

O `sn.bin` do cliente real tem o **mesmo formato de 143 B** daqui (11 slots de 13 bytes) e traz
`Global` no slot 1 com contador 2 — um servidor, dois canais, exatamente o arranjo deste
repositório.

**Sonda UDP de status (`:8281`).** O cliente manda um datagrama de **4 bytes** (`01 02 03 04`) para
o host do canal e só considera o canal acessível quando o mesmo datagrama volta (captura
`target/teste-udp.json`); sem resposta o `Conectar` não abre o TCP. O canal Rust responde em eco
(`crates/wyd-server/src/status.rs`, `serve_udp`) na mesma porta do CPSock — TCP e UDP convivem.

**UNVERIFIED:** a semântica de cada número além de "contador da linha" (o rótulo `"%s : %d/%d"`
em `0x60f838` sugere atual/máximo) e o efeito visual dos contadores acima de 500
(`0x46db8b` compara com `0x1f4`). A prova de renderização é o cliente no *server select*.

## 5. O que ficou explicitamente opaco

- `.wyt`: controles, fontes, sprites e eventos de UI.
- `.msa`: geometria, esqueleto, animação e materiais (só o header e o nome de textura são conhecidos).
- `sn.bin`: semântica do primeiro byte.
- ~~`serverlist.bin`~~: **decodificado** em 2026-11-08 (§4) — grade 10×11×64 com soma da tabela
  estática; nomes/contadores em `sn.bin`.
- `cdata.bin`, `object.bin`, `curse.bin`, `itemicon.bin`, `minimap.dat`: não analisados nesta passada.

## 6. Densidade de pixel na exibição

O arquivo não carrega escala: o `.exe` desenha o raster em um retângulo fixo. Medição com o
`ServerList.wyt` (287×280, duas colunas de 140 px) na tela de 1030×788: as colunas aparecem com
181 px — **ampliação de ~1,29×**, ou seja, a arte é autorada abaixo da densidade exibida. Detalhe,
limites e o teste de textura 2× estão em
[`customizacao.md` §9](customizacao.md#9-densidade-de-pixel-por-que-a-arte-parece-ampliada-medição-de-2026-11-09).
