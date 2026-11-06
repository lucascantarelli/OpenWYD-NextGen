# Mapa, colisão e teleporte

O servidor não guarda o **nome** do mapa; guarda coordenada e atributo de terreno. Nome que o jogador lê
vem do cliente. Quem decide se um passo é possível é o servidor, a partir dos mapas de conteúdo.

## Mapas de conteúdo

| Arquivo | Tamanho | Uso |
|---|---|---|
| `Release/TMsrv/run/HeightMap.dat` | 16 MiB (4096²) | altura do terreno |
| `Release/TMsrv/run/AttributeMap.dat` | 1 MiB (1024²) | atributo por célula; altura 127 = bloqueado |

No boot, `route.Bake` converte os dois numa grade de caminhabilidade (`world/grid.go`). Sem os arquivos
o servidor sobe, mas os mobs passam a andar "cego" (passo Chebyshev) e o multiplicador de EXP do campo
de treino fica desligado — é o sintoma clássico de conteúdo faltando.

## Movimento e teleporte

- `movement.go` valida passo a passo contra a grade.
- `world/teleport.go` executa a troca de mapa; destino vem de tabela (`internal/teleportdata`), não de
  script.
- `area.go` trata troca de área/zona; `gate.go` as passagens; `training_field.go` identifica o campo de
  treino e o bônus de EXP.

## Onde está

| Peça | Arquivo |
|---|---|
| Grade de caminhabilidade | `internal/route/route.go`, `world/grid.go` |
| Movimento | `movement.go` |
| Teleporte entre mapas | `world/teleport.go`, `internal/teleportdata` |
| Área, portão e treino | `area.go`, `gate.go`, `training_field.go` |
| Atlas do mundo | `world/atlas.go` |
| Operação | `docs/migration/config-ops.md` |
| Portal | `/atlas`, `/mapa`, `/teleportes` |

## Como verificar

```bash
go test ./tmserver/internal/route/ ./tmserver/internal/handler/ -run 'Movement|Area|Gate|Training|Teleport'
go test ./tmserver/internal/world/ -run 'Grid|Teleport|Atlas'
```

No jogo: atravessar uma parede deve ser recusado; teleportar por tile para outra cidade deve colocar o
personagem no ponto de entrada, não na posição de origem.

!!! warning "UNVERIFIED"
    Não existe tabela canônica de **nome** de mapa/cidade ↔ id numérico do cliente. As telas do portal
    mostram o mapa pelo atlas, sem esse nome. Se precisar do nome, ele está no conteúdo do cliente.

## Paralelo e desempenho

A grade é lida dentro do loop do mundo; não há consulta concorrente. O bake é feito no boot justamente
para que o caminho quente (movimento) seja leitura de slice, sem I/O.
