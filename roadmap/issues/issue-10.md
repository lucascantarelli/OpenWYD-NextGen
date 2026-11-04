# Issue 10 — Movimento, visão, cidades e teleporte

**Fase:** 3 · **Depende de:** 07 · **Bloqueia:** 11, 12, 13 · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Validar o mundo navegável: andar sem travar, ver outros jogadores e NPCs, alternar entre as cinco
cidades e usar o teleporte por tile com custo correto.

## Contexto / evidência

- Movimento entra por `MsgAction` (`0x036C`, +Action2/3) em `handler/movement.go`; o eco S→C é o que
  faz o personagem andar/renderizar para os outros.
- Cidades em `world/city.go`: Armia(0) `(2086,2093)`, Azran(1) `(2494,1707)`, Erion(2) `(2453,2000)`,
  Nippleheim(3) `(3652,3122)`, Noatum(4) `(1050,1706)`. "Última cidade" persiste em 2 bits (0–3;
  Noatum não é salvável).
- Teleporte por **tile**: o cliente pisa e manda `_MSG_ReqTeleport` (`0x0290`, só header); o servidor
  resolve destino e custo pela posição (`world.TeleportDest`). Noatum é hub (cidades pagam 700 para ir,
  volta é grátis); o teleporte vira `MSG_Action` com `Effect=1` + reconciliação de grid.
- Mobs patrulham e são vistos por jogadores (roaming/`roamRadius`); a visão é por grid.

## Passos

1. Andar em linha reta por ~20 tiles e conferir eco de posição; tentar atravessar parede/água
   (`AttributeMap` marca bloqueio) e conferir que não atravessa.
2. Com **dois clientes** (`test` e `test2`, personagens distintos), confirmar que um vê o outro
   andando e que o nome aparece.
3. Aproximar-se de NPCs/mercadores e conferir que são visíveis e clicáveis (issue 12 para a loja).
4. Percorrer as cinco cidades: entrar pelo mapa e/ou pelos comandos `/armia`, `/azran`, `/erion`,
   `/noatun`, `/gelo`; conferir spawn na área correta.
5. Usar uma rota de teleporte por tile: conferir custo em gold e posição final; repetir saindo de
   Noatum (grátis).
6. Trocar de cidade, deslogar e relogar: o spawn deve ser o da última cidade usada.

## Arquivos

- Consulta: `tmserver/internal/handler/movement.go`, `world/city.go`, `world/teleport.go`,
  `world/grid.go`, `handler/view.go`.
- Dados: `Release/TMsrv/run/Regions.txt`, `AttributeMap.dat`, `HeightMap.dat`, `InitItem.csv`.
- Proibido: mexer na tabela de coordenadas sem captura/fonte (`docs/migration/config-ops.md`).

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
# eco de movimento chegando
docker compose logs --since 5m tmserver | grep -E "recv packet.*0x036c" | head
# teleporte pedido e resolvido
docker compose logs --since 10m tmserver | grep -E "0x0290|teleport" | head
# última cidade persistida
docker compose exec -T db psql -U postgres -c "select name, last_city from character order by name;"
```

Saída esperada: pacotes `0x036c` contínuos ao andar; `0x0290` ao pisar na rota; `last_city` mudando
conforme a cidade visitada.

## Critérios de aceite

- [ ] Andar 20+ tiles sem travamento nem teleportar para trás
- [ ] Parede/água bloqueia (não atravessa)
- [ ] Dois clientes se veem e veem NPCs no mesmo mapa
- [ ] Cinco cidades alcançáveis com spawn correto
- [ ] Teleporte por tile cobra o valor esperado e leva à posição correta
- [ ] `last_city` persiste e o relogin nasce na cidade certa

## Riscos / rollback

- **Risco:** cliente "trava" ao entrar em área cheia de mobs → pode ser volume de pacotes, não
  movimento. **Mitigação:** ligar `-log-sends` por janela curta e medir.
- **Risco:** confundir posição não persistida com bug de save. **Mitigação:** ler B6 em
  `docs/migration/ingame-bugs.md`.
- **Rollback:** nenhum (só sessão); teleportes não alteram dados além da cidade.
