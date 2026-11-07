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
| Nomes em texto claro separados por NUL, a partir do offset 1 | `eserved`, `nova geração`, `nova geração2`… |
| 13 entradas no total | contagem por NUL em 143 B |
| Primeiro byte `0xd6` | não explicado por nenhuma hipótese testada — **UNVERIFIED** |

Hipótese não confirmada: o primeiro caractere seria `R` de “Reserved”, sobrescrito pelo byte de
cabeçalho. Não foi possível provar; o arquivo é tratado como “lista de nomes com 1 byte opaco”.

## 4. `serverlist.bin` — **permanece opaco**

7.040 B. Cabeçalho bruto:

```
41 28 1e 23 db df fb d5 00 d6 01 d5 fa dc ed d5 03 d4 ff d7 ff 17 31 16 41 d4 f7 d2
2d 18 30 a4 c1 a4 bf a4 be a4 bd a4 bc a4 bb a4 ba a4 b8 a4 b7 a4 b5 a4 b2 a4 b1 a4
```

Tentativas executadas e registradas em `gerado/formatos.json` (`serverlist.top5`):

| Método | Resultado |
|---|---|
| CPSock `decode`, 256 keywords, a partir de 4 | melhor proporção imprimível ≈ 0,10 — sem texto |
| CPSock `decode`, 256 keywords, a partir de 0 | idem |
| XOR de 1 byte (256 chaves) | gera texto imprimível **sem alfanuméricos** (só símbolos) — falso positivo, descartado |
| XOR com a tabela do CPSock | sem estrutura |

**Conclusão:** a hipótese “serverlist usa a mesma tabela do CPSock” **não se sustenta** com os
bytes atuais — nenhuma transformação conhecida produziu IP, porta ou `serv00`. O conteúdo
esperado existe em claro em `client/serverlist.txt`
(`http://192.168.100.3/serv00.htm`, `192.168.100.3:8281`, `192.168.100.4:8281`), o que dá o
gabarito para a próxima tentativa (08-6, com trace da leitura do arquivo).

## 5. O que ficou explicitamente opaco

- `.wyt`: controles, fontes, sprites e eventos de UI.
- `.msa`: geometria, esqueleto, animação e materiais (só o header e o nome de textura são conhecidos).
- `sn.bin`: semântica do primeiro byte.
- `serverlist.bin`: codificação e layout de canais — nenhuma hipótese sobreviveu.
- `cdata.bin`, `object.bin`, `curse.bin`, `itemicon.bin`, `minimap.dat`: não analisados nesta passada.
