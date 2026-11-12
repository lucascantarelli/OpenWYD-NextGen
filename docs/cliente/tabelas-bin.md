# Tabelas binárias do cliente — o que foi provado

Extraído por `node tools/client/extract-tables.mjs` (artefato: `docs/cliente/gerado/tabelas.json`).
Regra do repositório: **só entra aqui o que os bytes provam**; o resto fica declarado como opaco.

## 1. Ofuscação e enquadramento

| Propriedade | Valor | Como foi provado |
|---|---|---|
| Chave | **XOR `0x5A`** | maior taxa de zeros em 200 kB: `ItemList` 83,5%, `SkillData` 92,7% |
| Cabeçalho | **4 bytes** | `(tamanho − 4)` divisível pelo registro em ambos |
| Registro `ItemList.bin` | **140 bytes** (0x8C) | período 140 em 90,3% das posições; (910.004 − 4) / 140 = **6.500** |
| Registro `SkillData.bin` | **192 bytes** (0xC0) | período 192 em 93,6%; (23.812 − 4) / 192 = **124** |

```
bytes do arquivo  ──XOR 0x5A──▶  [4 bytes header][registro × N]
```

## 2. `SkillData.bin` — 124 × 192 B

Perfil de colunas (u16) revelou **21 colunas com cara de enum** (poucos valores, todos ≤ 31):

| Offset | Valores distintos | Máximo | Leitura possível |
|---|---|---|---|
| 0 | 6 | 6 | candidato a **tipo/uso** da skill |
| 12 | 8 | 8 | enum |
| 16 | 8 | 11 | enum |
| 60 | 4 | 4 | candidato a **classe/linhagem** (4 classes) |
| 80 | 4 | 3 | idem |
| 96 | 7 | 6 | enum |
| 112 | 10 | 12 | enum |
| 156/176 | 5 | 4 | idem |

**Não provado:** qual coluna é classe, qual é linhagem, e onde ficam nome e custo. Não existe
coluna com sequência 1..124 — o índice não está no início do registro. Nomear qualquer uma
delas agora seria inventar estrutura; o artefato mantém `offset + estatística` e nada mais.

> Consequência prática: o **mapeamento das 151 skills** (classe/linhagem) continua
> `UNVERIFIED` no banco. O caminho para fechar é trace dinâmico (CLIENT-06) ou decompilação com
> Ghidra — nenhum dos dois existe neste ambiente.

## 3. `ItemList.bin` — 6.500 × 140 B

Nenhuma coluna com sequência detectável em u32 e **zero** colunas enum em u16. Ou seja: o
registro não começa pelo índice, e o índice pode estar em u16 num offset que não bateria com a
ordem de leitura do arquivo.

**Mas o arquivo contém os nomes das skills em texto claro** (após o XOR): 131 dos 151 nomes do
seed `0010` foram localizados por casamento exato, 263 ocorrências no offset 136. O passo entre
nomes é variável (140, 280, 700, 1120…), então essa região **não** é um array fixo de 140 B —
ver `tools/client/map-skills.mjs` (status `diagnostico`) e `docs/wyd/decisoes-servidor.md` §6.

Isso é coerente com o fato de o catálogo já estar no banco por outra via (`ItemList.csv` →
`item_catalog`, 3.220 linhas). **O `.bin` não é necessário para o servidor** — ele é o espelho
do cliente.

## 4. O que **não** está no cliente

| Dado pedido | Onde está | Evidência |
|---|---|---|
| **taxa de drop por item** | configuração do servidor (TMSrv) | nenhum `.bin` do cliente tem tabela por monstro; o cliente só recebe pacote de chão quando o item já caiu |
| **curva de EXP por nível** | lógica do servidor | não existe coluna com 400 valores crescentes; o cliente recebe o EXP atual no `STRUCT_MOB` |
| **mapeamento skill → classe** | provavelmente em `SkillData.bin`, **não decodificado** | 21 enums candidatos, nenhum confirmado |

O script falha de propósito quando o resto da divisão não é zero: tamanho de registro errado
não pode virar evidência silenciosa.

## 5. `InitItem.bin` (768 B)

Sem XOR claro (a melhor chave dá 18,8% de zeros) e período **8** em 67% das posições →
provável **96 registros de 8 bytes**. Conteúdo: itens iniciais por classe (espelha `BaseMob`).
Permanece **UNVERIFIED**: não há como separar índice de item de quantidade sem o struct.

## 6. Como reproduzir

```bash
node tools/client/extract-tables.mjs --verbose   # perfil de colunas no stdout
cat docs/cliente/gerado/tabelas.json            # artefato determinístico
```

O artefato não contém data/hora nem caminho absoluto: rodar duas vezes no mesmo build produz
bytes idênticos.
