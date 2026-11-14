# 10 — Assets e formatos do cliente

> Medições diretas em `client/` (2026-11-13) + `docs/cliente/formatos.md` (2026-11-07/09) +
> pesquisa externa de 2026-11-13. **Nenhum formato abaixo está totalmente decodificado**; a coluna
> *Confiança* diz exatamente o que se sabe.

## 1. Inventário de assets (medido)

| Pasta | Arquivos | Tamanho | Conteúdo | Extensões |
|---|---:|---:|---|---|
| `Mesh/` | 5.361 | 189 MB | malhas, texturas, animações, ossos | `.msa` 1.707, `.msh` 903, `.ani` 776, `.wys` 1.865, `.bon` 81 |
| `UI/` | 342 | 73 MB | telas e ícones | `.wyt` (+ `AttributeMap.dat`, `CamAction.bin`) |
| `Env/` | 522 | 15 MB | terreno por campo | `FieldXXXX.dat`, `FieldXXXX.trn`, `Character.dat/.trn`, `EnvTextureList*.bin`, `AttributeMap.dat` |
| `Effect/` | 468 | 4,6 MB | efeitos/partículas | `.msa` + `.wys` |
| `Sound/` | 12 | 38 MB | áudio | — |
| `music/` | 13 | 26 MB | música | — |
| `NUI/` | 8 | 3,3 MB | interface “nova” | `main`, `mainparts`, `mainchat`, `ServerList`, `popup`, `ccmode`, `iconon`, `iconoff` |
| `shader/` | 18 | 40 KB | shaders compilados | `pseffect1-6`, `vseffect1-4`, `skinmesh1-8` |
| `XTrap/` | 6 | 4,7 MB | anticheat | `.xt`, `.dll`, `.Xtp`, `.Log` |
| `ScreenShot/` | 0 | — | saída do cliente | — |

Total da pasta: **~344 MB**. Para comparação, o cliente 7.69 do Global mede `mesh/` 7.947
arquivos/558 MB e `UI/` 396/153 MB — **mesma família de formatos, escala ~3×**.

## 2. Formatos — o que está provado

| Formato | Magic/estrutura provada | O que segue opaco | Confiança | Ferramenta |
|---|---|---|---|---|
| `.wyt` | `WT10` @0; versão `0x00020000` @4; largura @16, altura @18, bpp @20; pixels @22; 8 B finais | nada relevante (é raster) | **Confirmado** (256/259 amostras) | `wyt.mjs` |
| `.wys` | família textura de malha; no ecossistema é `WS10` + DDS (DXT1/3) | magic local não verificado | **Hipótese** forte | nenhum local |
| `.msa` | `0x112` @0, `0x20` @4, `1` @8, contagens @20/@28, nome de textura ASCII @34 | vértices, índices, esqueleto, materiais | Parcial (Confirmado o cabeçalho) | — |
| `.msh` | header com `id`, `parent`, `fvf`, `stride`, `palette`, `influences` (por fonte externa) | tudo além do header | **Hipótese** | — |
| `.bon` | tabela de ossos, registros de 8 B (por fonte externa) | semântica | Desconhecido local | — |
| `.ani` | 776 arquivos; sem layout provado | tudo | Desconhecido | — |
| `FieldXXXX.dat/.trn` | terreno por campo (arquitetura) | layout, altura, textura | Desconhecido | — |
| `AttributeMap.dat` | grade de atributos (colisão/área) | dimensão local | Desconhecido | — |
| `shader1..8.bin` / `shader/*.bin` | **bytecode D3D9 SM 1.1** (dwords) | assembly específico | Fortemente indicado | `D3DDisassemble` |
| `serverlist.bin` | `10×11×64`, XOR de tabela própria | — | **Confirmado** | `serverlist.mjs` |
| `sn.bin` | 11 nomes de 9 B + 11 contadores int32 @99 | — | **Confirmado** | `serverlist.mjs` |
| `ItemList.bin` | 910.004 B = 6.500×140+4 | campos | Fortemente indicado | — |
| `Itemname.bin` | 215.084 B = 3.163×68 | campos | Fortemente indicado | — |
| `SkillData.bin` | 23.812 B = 248×96+4 | campos | Fortemente indicado | — |
| `InitItem.bin` | 768 B = 96×8 | campos | Hipótese | — |
| `itemicon.bin` | 26.000 B = 6.500×4 | — | Fortemente indicado | — |
| `object.bin` | 524.292 B = 524.288+4 → **máscara de meshes** | semântica dos bits | Fortemente indicado | — |
| `cdata.bin` | 8.192 B = 2.048×4 (checksum de terreno) | — | Fortemente indicado | — |
| `curse.bin` | 8.192 B = 256×32 | — | Hipótese | — |
| `extraitem.bin` | 106 B (stub) | — | **Divergente** (Global: 110.390 B) | — |
| `minimap.dat` | 4.058 B | texto × binário | Desconhecido | — |
| `Mixlist.bin` | 272.000 B (6.800×40 ou 8.500×32) | divisor ambíguo | Desconhecido | — |
| `Mixhelp.dat` / `itemhelp.dat` | 11.050 / 99.134 B | texto provável | Desconhecido | — |
| `AniSound4.txt` | 45.164 B | mapa animação→som (texto) | Confirmado (texto) | — |
| `notice.txt`, `settings.txt`, `serverlist.txt` | texto | — | Confirmado | — |
| `.xt` (XTrap) | pacote de anticheat | tudo | Desconhecido | — |

## 3. `.wyt` em detalhe (o formato mais útil)

```
57 54 31 30 | 00 00 02 00 | 00 00 00 00 | 00 00 00 00 | ww ww | hh hh | bpp | …
W  T  1  0   versão         zeros          zeros        largura altura  bpp      pixels
```

| Propriedade | Valor |
|---|---|
| Fórmula | `22 + largura*altura*(bpp/8) + 8 = tamanho` |
| Casamento | **256 de 259** arquivos de `NUI/` + `UI/` |
| Bpp suportados | **24** (BGR) e **32** (BGRA); `8` e `16` são recusados com erro (não chutados) |
| Amostra | `NUI/ServerList.wyt` 287×280 32 bpp → moldura + slots legíveis |
| Exceções | `UI/NewAmul.wyt` (8 bpp), `UI/hanui.wyt`/`joyui.wyt` (16 bpp) |
| O que **não** tem | texto, fonte, posição de controle, evento |

> **Consequência para UI:** mexer em `.wyt` muda aparência, não comportamento. Para mover um
> botão é preciso patch no `.exe` (`09-client-customization.md`).

## 4. Tabelas `.bin` — a hipótese de layout e por que ela importa

O cliente carrega **localmente** `ItemList.bin`, `Itemname.bin`, `SkillData.bin`, `InitItem.bin`,
`itemicon.bin` e `Mixlist.bin`. O servidor tem o **mesmo conteúdo** no Postgres. Isso cria duas
verdades:

| Risco | Exemplo |
|---|---|
| Índice diverge | item 500 no cliente é “Poção”; no banco é “Espada” |
| Nome diverge | cliente mostra nome antigo após renomear no catálogo |
| Ícone diverge | `itemicon.bin` fixo em 6.500 posições |
| Skill diverge | `SkillData.bin` 248 registros × `skill_catalog` |

**Regra:** cruzar sempre por **índice**, nunca por nome (`docs/wyd/fontes.md` §5.3). Ampliar
conteúdo além de 6.500 itens/248 skills **exige** mudar o leitor do cliente — logo, é projeto de
patch, não de seed.

### 4.1 Divergência de geração (fortemente indicada)

| Tabela | Este build (7662) | Cliente 7.69+ | Registro |
|---|---:|---:|---|
| `ItemList.bin` | 910.004 (140 B) | 1.066.004 (164 B) | **+24 B** por item no 7.69 |
| `SkillData.bin` | 23.812 (96 B) | 25.796 (104 B) | **+8 B** por skill |

Interpretação: o 7662 usa um *schema* anterior. Não é “cliente pior”, é **cliente mais velho** —
e isso explica parte da diferença de conteúdo (sets, montarias, coleções) que o Global tem em
arquivos próprios (`ItemSet.bin`, `MountData.bin`, `CollectionData.bin`, `Missionitems.bin`).

## 5. Pipeline de asset — estado real

| Etapa | UI (`.wyt`) | Malha (`.msa`/`.msh`) | Mapa | Shader |
|---|---|---|---|---|
| Ler | ✅ `wyt.mjs` | ❌ | ❌ | ⚠️ precisa `D3DDisassemble` |
| Editar | ✅ GIMP/Krita | ⚠️ Blender + conversor inexistente | ⚠️ editores de licença incerta | ❌ |
| Escrever | ❌ (falta `--write`) | ❌ | ❌ | ❌ |
| Testar | ✅ abrir o cliente | ⚠️ precisa entrar no mundo | ⚠️ idem | ⚠️ |

**Primeira ferramenta a criar:** escritor `.wyt` (é 30 linhas de Node: cabeçalho + raster +
trailer). É o que destrava identidade visual sem patch.

## 6. O que falta investigar (formato)

| Pergunta | Como responder | Custo |
|---|---|---|
| Layout de `ItemList.bin`/`SkillData.bin` | Ghidra nos leitores `BASE_ReadItemList`/`BASE_ReadSkillBin` | M |
| `.msa`/`.msh`: vértices e FVF | idem no `TMMesh::LoadMsa`/`CMesh::LoadMesh` | L |
| `.ani`/`.bon`: frames e ossos | idem | L |
| `FieldXXXX.dat/.trn` | idem + comparação com `AttributeMap` | L |
| `minimap.dat` texto × binário | `head -c` + `file` | S |
| `extraitem.bin` stub | comparar com o do Global (tamanho) | S |
| `object.bin` máscara | correlacionar bits com `Mesh/` | M |

Todos estão em `18-unknown-features.md` com ID.
