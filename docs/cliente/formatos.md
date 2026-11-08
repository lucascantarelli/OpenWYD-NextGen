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
                largura? altura?  ?
```

| Offset | Campo | Evidência |
|---:|---|---|
| 0–3 | magic **`WT10`** | idêntico nas 4 amostras |
| 4 | versão **`0x00020000`** | constante nas 4 amostras |
| 16 | candidato a **largura** | `0x0268`=616, `0x0040`=64, `0x00dc`=220, `0x00e4`=228 |
| 18 | candidato a **altura** | `0x012c`=300, `0x0040`=64, `0x00c7`=199, `0x0139`=313 |
| 20 | desconhecido | 24 em duas amostras, 2080 (`0x820`) em duas |
| 24+ | máscaras/cores | padrão `00 ff 00 00` repetido em algumas telas |

**Opaco:** tabela de controles, fontes, sprites, texto e âncoras — nada disso foi interpretado.
`largura/altura` são hipótese coerente (o par varia por tela), **não validadas em runtime**.

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

Estado atual de `client/sn.bin`: slot 0 reservado com contador 3 (2 canais + cabeçalho), slot 1
`NextGen` com contador 2 → grupo 1 com canal 1 (`192.168.100.3`) e canal 2 (`192.168.100.4`).

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
dois é teste (`crates/wyd-launcher/tests/serverlist_paridade.rs`).

### 4.1 `serv00.htm` — o que o cliente faz com a resposta (captura estática)

O `GET` da célula 0 sai por WinInet: `InternetOpenA("MS")` → `InternetOpenUrlA(url, …, 0x4000000)`
→ **uma** `InternetReadFile` de `0x400` B com NUL no fim (`client/WYD.exe`, rotina `0x5429e9`).
Consequência de contrato: o corpo útil é de **até 1.023 caracteres** e precisa caber numa leitura
só — resposta longa ou por partes não é suportada pelo cliente.

Duas rotinas consomem esse texto:

| Rotina | O que faz | Evidência |
|---|---|---|
| `0x46d7xx` | `strtok(corpo, ",")` e grava **campo a campo** nas células seguintes do grupo (`%s`) | `0x46d8b1` (`push $0x2c`), `0x46d9c4` |
| `0x4b013b` | `sscanf(corpo, "%d", …)` com 11 destinos inicializados em `-1` | `0x4b017d`, `0x4b01d9`/`0x4b01e5` |

Logo o corpo é uma **lista separada por vírgula**: campo de texto alimenta as células de canal do
grupo e campo numérico alimenta o contador do rótulo (`"%s : %d/%d"`, `0x60f838`). Resposta
vazia ou inalcançável deixa a célula como veio do `.bin`, e o canal aparece vazio/`Inexistente` —
o sintoma já medido com URL acima de 32 caracteres.

**Servido pelo Rust desde 2026-11-08:** o `tmserver` (`crates/wyd-server`, ADR-0022 §7) responde
`GET :80/serv00.htm` com o CSV dos hosts dos canais habilitados, lido da tabela `server_channel`
(migration `0029`). Com dois canais o corpo é `192.168.100.3,192.168.100.4`; a URL de status em si
vem da linha `channel = 0` e continua com teto de 32 caracteres.

**UNVERIFIED (pendente de prova dinâmica):** a ordem exata campo↔canal e quantos numéricos o
corpo precisa. A leitura estática mostra *que* o corpo é CSV consumido por `strtok`/`sscanf`, não
fecha o leiaute. A prova é um servidor de probe no `:80` + cliente no *server select*, comparando
corpo servido × coluna desenhada.

## 5. O que ficou explicitamente opaco

- `.wyt`: controles, fontes, sprites e eventos de UI.
- `.msa`: geometria, esqueleto, animação e materiais (só o header e o nome de textura são conhecidos).
- `sn.bin`: semântica do primeiro byte.
- ~~`serverlist.bin`~~: **decodificado** em 2026-11-08 (§4) — grade 10×11×64 com soma da tabela
  estática; nomes/contadores em `sn.bin`.
- `cdata.bin`, `object.bin`, `curse.bin`, `itemicon.bin`, `minimap.dat`: não analisados nesta passada.
