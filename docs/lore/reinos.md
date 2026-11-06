# Reinos e cidades

O mundo do cliente é feito de mapas com nome próprio. O servidor **não** guarda o nome do mapa: ele
guarda coordenada e atributo de terreno (`HeightMap.dat`, `AttributeMap.dat`). O nome que o jogador lê
vem do conteúdo do cliente.

## O que o servidor conhece

| Conceito | Onde está no servidor | Observação |
|---|---|---|
| Mapa e colisão | `tmserver/content/TMsrv/run/HeightMap.dat` + `AttributeMap.dat` | 4096² e 1024²; alimentam `route.Bake` |
| Teleporte entre mapas | `handler/teleport*.go`, `internal/teleportdata` | destino por tabela, não por script |
| Cidade com loja | `npc_template` + `npc_shop_item` no Postgres | NPC vendedor e sua bolsa |
| Zona de treino | multiplicador de EXP por região | campo de treino documentado nas regras |
| Castelo | estado por guilda | ver [Guerras](guerras.md) |

## Armia

Armia é a cidade de referência do servidor local: é onde ficam os mercadores usados nos testes de
catálogo. Ela também guarda um episódio útil de memória: 20 linhas de seed curado foram sombreadas por
um bloco importado e **duplicaram os mercadores**. A correção foi remover a importação duplicada, não
"limpar" a tela — a lição vale para qualquer cidade: uma fonte de verdade por NPC.

## Como adicionar uma cidade ao conteúdo

1. Criar os `npc_template` e `npc_definition` no banco (não em CSV — CSV saiu na fase 5).
2. Dar aos NPCs vendedores uma bolsa (`npc_shop_item`) com itens que existam em `item_catalog`.
3. Definir teleporte de entrada na tabela de teleportes.
4. Se a cidade tiver treino, registrar o multiplicador na região.
5. Verificar no jogo: entrar pelo teleporte, abrir a loja, comprar um item e conferir no inventário.

!!! note "O que falta"
    Uma tabela canônica de **nomes** de mapa/cidade com o id numérico usado pelo cliente. Hoje o mapa
    aparece nas telas do portal pelo arquivo de atlas (`/atlas`), sem esse nome canônico.
