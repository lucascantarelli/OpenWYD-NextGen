# Issue 13 — Quests, refino/combine e progressão

**Fase:** 3 · **Depende de:** 11, 12 · **Bloqueia:** 16, 17 · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Validar as cadeias de conteúdo que sustentam a progressão: trocas de NPC (quests), refino com poeira,
combines e os sistemas de evolução — sem corromper item nem personagem.

## Contexto / evidência

- Quests entram por `_MSG_Quest` (`0x028B`) em `handler/quest.go`; casos implementados incluem Perzen
  (troca `Esfera da Sorte A` 4130 → montaria `Thoroughbred` 3987, expira em 30 dias), Kibita,
  Jardineiro, Coveiro, Mestre Grifo, class masters e o Sobrevivente (Kefra, 100 entradas por
  pergaminho 4127, ajustável por `-kefra-entries-per-scroll`).
- Refino: `refine/` + `Release/Common/Settings/SancRate.txt` (anvil 0 = Ori, 1 = Lac), com bônus por
  `nPos` a partir de +9; `EF_DAMAGE` de arma é tratado fora dos `BaseEffects`.
- Combines: um motor (`handler/combine.go`) para `CombItem` (`0x03A6`) e as variantes
  Odin/Ailyn/Tiny/Shany/Agatha/Lindy/Alquimia/Ehre/Extração, com receitas/famílias de `CompRate.txt`.
- Pista de Runas: inscrição com Uxmal nos minutos 16–19/36–39/56–59, seis salas (+0 Lich a +6 Coelho),
  runas e prêmios (detalhes em `docs/game.md`).
- Itens de progressão por item (não por NPC): Pedra Ideal 1742, Pedra da Fúria 3020, Cythera 3502/3507,
  Diadema do Sub-Celestial, Pedra Misteriosa 4148.

## Passos

1. **Refino**: refinar uma peça com poeira até +9, conferir consumo, taxa e variação de score
   (`MSG_UpdateScore`/equip). Repetir em peça de slot diferente.
2. **Quest de troca**: Perzen — levar 4130, trocar por 3987, conferir consumo e expiração (30 dias).
3. **Quest de classe**: um class master (Merchant 3/31) e uma montaria (Merchant 16/58/23/101-110).
4. **Combines**: uma receita de cada família com itens reais do catálogo; conferir item resultante e
   consumo dos ingredientes. Documentar as que não casarem (receita UNVERIFIED).
5. **Pista de Runas**: inscrever no horário válido, entrar na sala, concluir e conferir runa/prêmio.
6. **Evolução**: seguir um caminho curto de progressão (Mortal → Arch ou uso de Pedra Ideal) conforme
   `docs/game.md`, registrando o que estiver bloqueado por sistema pendente.
7. Após cada passo: relogar e conferir persistência (quests, classe, itens).

## Arquivos

- Consulta: `handler/quest.go`, `handler/combine*.go`, `handler/skill.go`, `handler/celestial*.go`,
  `handler/subcelestial*.go`, `refine/`, `combine/`, `content/catalog.go`.
- Dados: `ItemList.csv`, `SkillData.csv`, `Settings/{CompRate,SancRate,QuestsRate}.txt`,
  templates em `Release/TMsrv/run/npc/`.
- Proibido: inventar coordenada/receita ausente na fonte; marcar como UNVERIFIED e abrir issue.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine \
  go test -race ./tmserver/internal/handler/ -run 'Quest|Combine|Refine|Celestial' -v
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine \
  go test -race ./tmserver/internal/refine/ ./tmserver/internal/combine/
# itens com expiração
docker compose exec -T db psql -U postgres -c \
  "select count(*) from item where expires_at is not null;"
```

Saída esperada: testes verdes; item de quest com `expires_at` preenchido; nenhum item "duplicado"
após combine/refino.

## Critérios de aceite

- [ ] Refino consome poeira, aplica taxa e atualiza score (incluindo +9)
- [ ] Troca de NPC consome e entrega exatamente os itens previstos
- [ ] Combine de cada família testada produz o resultado e consome ingredientes
- [ ] Pista de Runas: inscrição, entrada, conclusão e prêmio
- [ ] Progressão testada registra claramente o que é ✅ e o que é ⏳ (não mascarar pendência)
- [ ] Persistência conferida após relogin em todas as etapas

## Riscos / rollback

- **Risco:** perda de item em combine por falha de save. **Mitigação:** testar com item de baixo valor
  primeiro e conferir SQL antes/depois.
- **Risco:** quest com horário (Runas) exigir espera real. **Mitigação:** planejar janela de teste;
  não alterar relógio do servidor sem necessidade.
- **Risco:** calibrar progressão em cima de valor UNVERIFIED (curva celestial placeholder, coef.
  Dex/Str por classe×arma, DoT). **Mitigação:** registrar como pendência de captura.
- **Rollback:** usar `/gm item` para repor item perdido em ambiente de teste, anotando o que foi feito.
