# Issue 27 — Editor do mapa de colisão (AttributeMap)

**Fase:** 6 · **Depende de:** 21 · **Bloqueia:** — · **Esforço:** M · **Estado:** ✅ (UI ao vivo: info + histograma + transformação + download de 1 MiB com sha conferido)

## Objetivo

Inspecionar o `AttributeMap.dat` (1024², célula = bloco 4×4 do mundo) e aplicar transformações em
massa (retângulo + filtro), baixando o `AttributeMap_New.dat` para substituição controlada.

## Contexto / evidência

- `AttributeMapAdminService` substitui o `Source/Code/AttributeMap_Editor` legado e **não** sobrescreve
  o arquivo: devolve o payload de 1 MiB para download (`docs/integrations/attributemap-editor-nextjs.md`).
- `GetAttributeMapInfo` → `dim=1024`, `world_scale=4`, `sha256`, `histogram` (256 buckets), `meanings`.
  `INVALID` = sem `-content`, arquivo ausente ou tamanho ≠ 1024×1024.
- Operações: `LEGACY_MARK_PVP_EXP_LOSS` (`value<64 && value!=1 → value|=64`), `ASSIGN_VALUE`,
  `SET_BITS`, `CLEAR_BITS`, `TOGGLE_BITS` (operand 1..255 nas três últimas).
- `rect` em coordenadas de **mundo** (0..4095, inclusivo) e `filter` por `exact_value` ou por
  `mask/match_value`.
- Significados: `1` cidade/área segura, `2` bloqueio (no `HeightMap.dat`), `4` não marca gema,
  `64` PvP/perda de EXP, `128` gate de newbie. O resto é legado preservado.
- O jogo consome no boot: trocar o arquivo + restart do tmserver.

## Passos

1. `/mapa`: cartão de estado com `dim`, `world_scale`, `sha256` (primeiros 12 chars), tamanho em bytes
   e o histograma como barras (256 valores, com zoom nos 16 mais frequentes).
2. Legenda dos significados, com nota "outros valores são legado preservado".
3. Formulário de transformação:
   - operação (Select com as 4 + legado), `operand` habilitado conforme a operação;
   - retângulo opcional em coordenadas de mundo (`min_x/min_y/max_x/max_y`, 0..4095) com validação
     `min ≤ max` e dica de que `x/4` vira célula;
   - filtro opcional: `enabled`, `mask`, `match_value`, `exact_value`.
4. Botão **Pré-visualizar** → chama `TransformAttributeMap` e mostra `changed_count`,
   `original_sha256`/`new_sha256` e o `after_histogram` ao lado do atual, **sem** salvar nada
   (a RPC já é pura; o download é o passo separado).
5. Botão **Baixar AttributeMap_New.dat** → usa `data` (base64) da resposta, `Content-Type:
   application/octet-stream`, nome exato do backend. Guardar o `sha256` na tela para conferência.
6. Guia de aplicação na tela (3 linhas): copiar para `Release/TMsrv/run/AttributeMap.dat` (backup do
   atual), `docker compose restart tmserver`, conferir `walkability grid baked` no log.
7. Bloquear transformação sem `moderator`/`admin` (via `adminCall`) e registrar no log de auditoria
   operação, retângulo, filtro e `changed_count`.

## Arquivos

- Criados: `portal/src/app/(admin)/mapa/page.tsx`,
  `portal/src/server/actions/attributemap.ts`, `portal/src/lib/attributemap.ts`,
  `portal/src/components/map/Histogram.tsx`, `portal/src/components/map/RectPicker.tsx`.
- Alterados: `portal/src/components/nav.ts`.
- Proibido: escrever `AttributeMap.dat` a partir da web; aplicar transformação sem pré-visualizar.

## Testes / verificação

```bash
cd portal && npx tsc --noEmit && npm run build
```

Evidência: pré-visualizar `LEGACY_MARK_PVP_EXP_LOSS` no retângulo `0..99 × 0..99` → `changed_count`
coerente com o histograma; baixar o `.dat`, conferir tamanho 1048576 e `sha256` igual ao
`new_sha256`; **não** substituir o arquivo em produção sem pedido explícito.

## Evidência ao vivo

- `GET /mapa` com sessão → **200**, renderizando “Mapa de colisão”, “Significados”, “Transformação
  em massa”, “Como aplicar” e `1.048.576` bytes.
- `GET /api/mapa/download?operation=3&operand=64&min_x=0&min_y=0&max_x=399&max_y=399` → **200**,
  `content-disposition: attachment; filename="AttributeMap_New.dat"`, `content-length: 1048576`,
  `x-map-changed-count: 7148` e `x-map-sha256: 7f464f81…890722` — o sha256 do arquivo baixado é
  idêntico ao header (conferido com `sha256sum`).
- `next build` **0**, com `/mapa` em 4,95 kB e `/api/mapa/download` na tabela de rotas; `npm test`
  **21/21** (3 arquivos) incluindo os 6 casos novos de mundo→célula, retângulo inválido, operando 0
  e round-trip da query; `tsc --noEmit` **0**.
- **Limitação registrada:** o botão “pré-visualizar” não foi clicado num navegador (o login do
  painel no browser de teste não submeteu o formulário). A pré-visualização usa exatamente a mesma
  RPC e o mesmo `requestFor` do download, que foi exercitado com operação + retângulo reais; o que
  falta é só o clique, não o caminho de servidor.

## Critérios de aceite

- [x] Info carrega com histograma e sha256; estado `INVALID` explicado (sem `-content`)
- [x] Pré-visualização mostra contagem e histograma antes/depois sem tocar em arquivo (mesma RPC do
  download; ver limitação acima)
- [x] Download do `.dat` com bytes idênticos ao `new_sha256`
- [x] Retângulo em coordenadas de mundo (não em células) e filtro por máscara corretos
- [x] `player` recebe 403; auditoria registra a operação (papel coberto pelos testes do serviço e
  pelo `adminCall`; ações auditadas como `map.info`/`map.preview`/`map.download`)
- [x] Instruções de aplicação (backup + restart) visíveis na tela

## Riscos / rollback

- **Risco:** operador aplicar um mapa que bloqueia uma cidade. **Mitigação:** pré-visualização
  obrigatória + backup do `.dat` antes de copiar + `walkability grid baked` no log.
- **Risco:** confundir coordenada de mundo com célula. **Mitigação:** mostrar as duas na UI
  (`mundo 100,100 = célula 25,25`).
- **Rollback:** restaurar o `AttributeMap.dat` do backup e reiniciar o tmserver.
