# Issue 48 — Mapa de navegação: relevo, zoom e leitura de ponto

**Fase:** 6 · **Depende de:** 41 · **Bloqueia:** — · **Esforço:** M · **Estado:** 🟡 (relevo, zoom e leitura ✅)

## Objetivo

Transformar `/atlas` de uma silhueta de colisão em um **mapa de navegação** que responde onde fica
cada coisa: relevo legível, zoom que acompanha o ponteiro, coordenada sob o cursor e, ao clicar,
o que existe naquele ponto.

## Contexto / evidência

- O PNG anterior vinha só do `AttributeMap.dat` reduzido a 512², com **dois tons de cinza** (`shade`:
  livre / bloqueado) — 7 KB, sem relevo, sem escala, sem leitura.
- O `HeightMap.dat` (4096², um byte por bloco) já é carregado pelo servidor e é o **mesmo grid** que
  `route.Bake` usa no pathfinding dos mobs. É a fonte natural de relevo.
- **Arte real do cliente não é opção**: `client/` não traz `.til`/texturas de mapa, e a arte do
  jogo é proprietária — não se baixa imagem de mapa da internet e se redistribui. O que dá para fazer
  é derivar dos dados que o servidor tem, o que tem a vantagem de **não mentir**: muda a colisão, muda
  o desenho.
- A view antiga tinha `<img>` + `<svg>` sobrepostos num quadrado fixo, sem `wheel` nem arrasto: não
  havia como olhar de perto uma cidade (o mundo tem 4096² blocos).

## Feito nesta rodada

1. **Relevo no gerador** (`tmserver/cmd/worldatlas`): `renderTerrain` amostra o `HeightMap` 4096² em
   1024² (média 4×4), aplica rampa hipsométrica (vazio → planície → campo → colina → montanha → pico),
   **hillshade** de noroeste (declive decide o brilho) e escurece o que o `AttributeMap` marca como
   bloqueado. PNG passou de 512² para 1024².
2. **`source_hash`** agora inclui o `HeightMap.dat`: trocar o mapa muda a versão do atlas.
3. **Zoom e pan** na view: roda (ancorada no cursor, listener não-passivo), arrasto com `pointer
   capture`, duplo clique, botões −/+/ajustar e teclado (setas, `+`, `-`, `0`) no palco focável.
4. **Leitura de ponto**: coordenada sob o ponteiro em tempo real; clique marca o ponto com cruz e
   informa região nomeada, cidade mais próxima (com distância em blocos) e gerador de spawn mais
   próximo. “copiar” leva a coordenada para a área de transferência.
5. **“ver no mapa”** na tabela de cidades: centraliza com zoom 4× no spawn da cidade.
6. **Detalhe em resolução nativa**: o mesmo `renderTerrain` gera também `world-map-hd.png`
   (4096², 1 px por bloco, 1,2 MB) ao lado da visão geral (1024², 189 KB). A página pede o detalhe
   (`/api/atlas/map?detail=1`) quando o zoom passa de 1,5× e o empilha sobre a visão geral, então não
   há tela vazia enquanto o arquivo grande carrega. `image-rendering` saiu de `pixelated` para `auto`
   e o palco subiu de 680 px para 940 px de largura máxima — ampliar agora interpola em vez de
   mostrar blocos 4×4.

## Pendente (o que ainda falta para virar mapa de navegação de verdade)

1. ~~**Contornos de região desenhados**~~ — feito como **raster**, não como polilinha: o gerador
   escreve `region-map.png` (43 KB, 1024²) pintando só as células caminháveis com relevo dentro de cada
   região, então o contorno segue o terreno exato. Marching squares daria centenas de linhas de
   geometria para uma figura que o browser compõe de graça. Cada região ganhou também `color`, gerada
   por ângulo áureo (52 regiões, 52 cores distintas — uma paleta fixa de 12 repetiria cor entre
   vizinhas) usada no raster, no destaque e no swatch da tabela.

> **Achado do enquadramento (issue 48):** enquadrar no `bounds` não ajuda, porque as áreas jogáveis
> ocupam o mundo inteiro — a união das regiões vai de 128 a 4076, os spawns de 321 a 3474 (p05–p95) e
> as cidades vão de Noatum (1050,1706) a Nippleheim (3652,3122). O vazio está **entre** as ilhas. A
> solução adotada foi abrir a 3× em Armia; “ajustar” volta ao mundo todo. Enquadrar de verdade exigiria
> recortar por densidade (calor de spawn), não por bounding box.
2. ~~**Mapa de calor de spawn**~~ — feito: o gerador publica `spawn_heat` (grade 64×64, célula de 64²
   blocos; pico de 813 geradores, 6.099 no total, 360 células ocupadas) e a view pinta a camada “calor
   de spawn” com raiz quadrada na escala (linear deixaria tudo uniforme por causa do pico).

   Junto veio um bug de cache: `getWorldAtlas` usava `unstable_cache`, que grava também o resultado
   `null` e não percebe o arquivo regerado — o atlas novo só aparecia depois do TTL. Agora é cache
   manual por `mtime`: regenerou, aparece; arquivo ausente, segue o caminho de “ensinar a gerar”.
3. ~~**Teleportes com nome de zona**~~ — feito: origem e destino mostram o lugar (cidade até 400
   blocos, senão a região nomeada que contém o ponto, senão a cidade mais próxima com distância) e a
   coordenada fica em mono ao lado. Ex.: `Erion 2668,2156`, `Dugeon_1_Andar_Hidra 144,3772`.
4. ~~**Ir para coordenada**~~ — feito: campos `x`/`y` na barra do mapa centralizam a 4× e marcam o
   ponto; coordenada fora de 0–4096 é recusada com aviso. Vai casar direto com a 38 (online com
   posição).
5. **Prévia de rota**: usar o pathfinding do servidor (`tmserver/internal/route`) para desenhar o
   caminho entre dois pontos, mostrando o custo em blocos e se atravessa área bloqueada.
6. **Exportar SVG/PNG do recorte atual**: o mapa já tem camadas; exportar do zoom atual serve para
   anexar em relatório de bug.
7. **Miniatura para a lista de mapas**: a visão geral (189 KB) serve bem à página, mas uma lista com
   várias miniaturas vai querer 256². Se o detalhe crescer muito, o passo seguinte é **pirâmide de
   tiles** em vez de um PNG único — hoje 1,2 MB carrega rápido em `localhost`.
8. ~~**Mobile/touch**~~ — feito: pinch-zoom de dois dedos (a distância entre ponteiros vira fator de
   escala ancorado no ponto médio; o arrasto simples sai de cena enquanto há dois dedos) e botão
   **tela cheia**, com o palco quadrado limitado pelo menor lado da janela e `Esc` para sair.

## Arquivos

- Alterados: `tmserver/cmd/worldatlas/main.go` (`renderTerrain`, `terrainColor`, `clampByte`),
  `portal/src/app/(admin)/atlas/atlas-view.tsx` (zoom/pan/leitura),
  `portal/src/app/globals.css` (`.ui-atlas__stage`, `__canvas`, `__bar`).
- Proibido: baixar/redistribuir arte de mapa do jogo; escrever em `Release/` (o gerador só lê).

## Testes / verificação

```bash
go build ./... && go run ./tmserver/cmd/worldatlas -content Release -out dist
cd portal && npx tsc --noEmit
```

Evidência: `world-map.png` em 1024² com relevo; `/atlas` com roda, arrasto, `+`/`-` e leitura de
coordenada; clique em cima de Armia informa a cidade e a distância.

## Critérios de aceite

- [x] Mapa com relevo vindo do `HeightMap` (sem arte de terceiros)
- [x] Zoom ancorado no cursor, pan por arrasto e controles de teclado
- [x] Clique lê coordenada, região, cidade e spawn mais próximos
- [x] Zoom com detalhe em resolução nativa (4096²) e ampliação suave
- [ ] Contorno de região derivado da máscara
- [ ] Mapa de calor de spawn agregado
- [x] Teleportes rotulados por zona
- [x] Ir para coordenada
- [x] Pinch-zoom e tela cheia
- [x] Mapa de calor de spawn agregado
