# 12 — Mapas, ambientes e animações

> Base: inventário medido de `client/Env/`, `client/Mesh/`, `docs/wyd/sistemas/mundo.md`,
> migrations `0020/0023/0011`, `crates/wyd-core/src/spatial/grid.rs` e
> `crates/wyd-core/src/npc/route.rs`.

## 1. Arquitetura de mapa neste build

| Peça | Onde | Medida | Papel |
|---|---|---|---|
| Campo/terreno | `Env/FieldXXXX.dat` + `.trn` | 522 arquivos em `Env/` | geometria + textura por campo |
| Texturas de ambiente | `Env/EnvTextureList{,2,3}.bin` | 3 listas | atlas por versão |
| Personagem em cena | `Env/Character.dat`, `Character.trn` | — | modelo base do personagem |
| Atributos/colisão | `Env/AttributeMap.dat`, cópias em `Mesh/` e `UI/` | — | grade de atributos (bloqueio/área) |
| Objetos de cena | `Field*.dat` (por campo) | — | posicionamento de objeto |
| Máscara de meshes | `object.bin` na raiz | **524.292 B = 524.288+4** | bitmask de mesh válida, **não** lista de objetos |
| Minimapa | `minimap.dat` | 4.058 B | texto/coordenadas (provável) |
| Legenda de ícones | `itemicon.bin` | 6.500 × 4 | não é mapa |

**Diferença de geração (Fortemente indicado):** o 7.69+ usa **heightmap global** de 4096×4096
(`HeightMap.dat`, 16.777.216 B) + `AttributeMap.dat` 1024×1024. O 7662 usa **terreno por campo**.
Isso muda o pipeline de mapa: não há “um mapa” para editar, há N campos com `Field` próprio.

## 2. Mapas no servidor (Rust)

| Item | Estado | Evidência |
|---|---|---|
| 52 regiões semeadas | ✅ | `0020_seed_mundo.sql`, `0023_seed_mundo_fixo.sql` |
| Rotas entre mapas | ✅ (dado) | `0011_rotas_mundo.sql` |
| Posição persistida | ✅ | `characters.x/y`, `selchar.rs` |
| Grade espacial / visibilidade | ✅ | `core/src/spatial/grid.rs` + testes |
| Colisão de jogador | ⬜ | só rota de NPC (`npc/route.rs`) |
| Teleporte/portal | ⬜ | `PAR-MOV-03` |
| Spawn por área/densidade | 🚧 | NPC nasce com o mapa |
| Respawn próprio | ⬜ | depende de recarga de definição |
| Instâncias/dungeons | ⬜ | `MU` |
| Clima/ciclo dia-noite | ⬜ | `PAR-MOV-07` |

> O servidor **não** lê `Field*.dat`: a colisão do legado é do cliente + validação de servidor
> sobre atributos. Enquanto `PAR-MOV-02` não existir, o servidor aceita movimento em célula
> bloqueada se o cliente permitir — risco de atravessar parede (`DIV-10`).

## 3. Mapa do mundo conhecido (lore × dados)

| Local | Tipo | Fonte |
|---|---|---|
| Armia | cidade inicial dos humanos | `docs/wyd/sistemas/mundo.md` |
| Noatun | cidade/cerco | `guerras.md` |
| Nippleheim | continente gelado ao norte de Azran | referência externa (Kersef) |
| Kefra | dungeon + boss semanal | `quests.md` |
| Hekalotia / Akeronia | **reinos** canônicos | referência externa |
| Campo de Treinamento | mapa de quest implementado | `GAME-18` |
| 52 regiões | seed do banco | `0020` |

**Correção de premissa (Confirmado por fonte oficial):** Kefra, Armia e Nippleheim **não são
reinos** — são dungeon, cidade e região. Os reinos são Hekalotia e Akeronia. Isso importa para
`PAR-CID-02` (compra de cidadania) e para o desenho de guerra.

## 4. Animações

| Peça | Arquivos | Formato | Estado |
|---|---:|---|---|
| `.ani` | 776 | sem layout provado | Desconhecido |
| `.bon` | 81 | tabela de ossos (registros 8 B, por fonte externa) | Hipótese |
| `.msh` | 903 | malha skinned (FVF/palette/influences) | Hipótese |
| `.msa` | 1.707 | malha estática/efeito | Parcial (header) |
| `AniSound4.txt` | 1 | mapa animação→som (texto) | Confirmado |
| `ValidIndex.bin` (v769) | 100×186 int32 | índices válidos de animação | **Não presente** neste build |

**O que se sabe:** há pipeline de skin (`skinmesh1-8.bin`) e animação por osso. **O que não se
sabe:** quantos frames, timing, interpolação, nomes de clipe (idle/walk/attack/die).

### 4.1 Como criar animação (proposta, não provada)

```
Blender (rig compatível com .bon) → exportar frames por osso → .ani + .bon
                                  → testar em `Mesh/` com um modelo existente
```

Sem o layout de `.ani`, isso é **especulação**. O caminho real é RE no loader
(`MeshManager::InitBoneAnimation`) com Ghidra — `UNK-19`.

## 5. Edição de mapa — o que é viável

| Objetivo | Ferramenta | Licença | Risco |
|---|---|---|---|
| Ver/editar terreno | editores de comunidade (C#/OpenGL) | **incerta** | alto (sem licença declarada) |
| Substituir textura de campo | `.trn` + `.wys` | — | médio |
| Mover NPC/spawn | **banco** (seed) | — | baixo — é o caminho recomendado |
| Novo mapa | RE de `Field*.dat` + servidor | — | muito alto |
| Colisão coerente | derivar `AttributeMap` do terreno | — | alto |

**Recomendação:** para o projeto, **mapa é dado de servidor** (regiões, NPCs, portais no seed).
Editar binário do cliente só quando houver demanda visual e decisão jurídica (`09-client-customization.md`).

## 6. Lacunas

| ID | Lacuna | Impacto |
|---|---|---|
| `UNK-17` | layout de `FieldXXXX.dat/.trn` | sem ele, nenhum editor próprio |
| `UNK-18` | dimensão/origem de `AttributeMap.dat` | colisão server-side impossível |
| `UNK-19` | layout de `.ani`/`.bon` | sem animação nova |
| `UNK-20` | `minimap.dat` texto × binário | minimapa no HUD |
| `UNK-21` | raio de visão do cliente | delta de cena pode divergir |
| — | clima/ciclo | cosmético, baixa prioridade |
