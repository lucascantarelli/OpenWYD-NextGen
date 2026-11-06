# `Release/` — o que o servidor ainda lê e o que pode sair

Análise de 06/11/2026 da árvore de conteúdo legada: **141 arquivos, 55 MB**, em `Common/`, `DBsrv/` e
`TMsrv/`. Objetivo: separar o que é **entrada de runtime do servidor Go** do que é **insumo de
importação** e do que é **resíduo do servidor Win32**, para que a dependência do legado caia a zero.

## Classificação

### 1. Lido a cada boot pelo tmServer (runtime)

| Arquivo | Quem lê | Observação |
|---|---|---|
| `TMsrv/run/HeightMap.dat` (16 MiB) | `route.Bake` | sem ele, mob anda "cego" e o bônus de campo de treino desliga |
| `TMsrv/run/AttributeMap.dat` (1 MiB) | `route.Bake` | altura 127 = bloqueado |
| `TMsrv/run/gameconfig.txt` | flags de mundo | `double`, `deadpoint`, `partybonus`, matriz 8×8 de drop, `PotionDelay` |
| `TMsrv/run/Regions.txt`, `Guard.txt`, `LevelItem.txt` | geração/limites | citados em `config-ops.md` |
| `Common/Settings/{SancRate,CompRate,QuestsRate,CastleQuest,MobMerc}.txt` | `refine`, `combine`, `content` | **taxa quebrada é erro fatal de boot** |
| `DBsrv/run/BaseMob/{TK,FM,BM,HT}` (816 B) | criação de personagem | equipamento inicial vem daqui |
| `TMsrv/run/BaseSummon/` | invocação do BM | templates de invocação |
| `Common/serv00.htm` | listener HTTP :80 | o cliente sonda o status antes de conectar |

### 2. Insumo de importação (não é runtime desde a issue 58)

`TMsrv/run/npc/*` (1.991 templates), `Common/ItemList.csv`, `Common/SkillData.csv`,
`TMsrv/run/InitItem.csv`, `Common/NPCGener.new.txt`. Depois do import (`tmserver/cmd/importcontent`),
o runtime lê **do Postgres**: `item_catalog`, `skill_catalog`, `npc_template`, `world_item_seed`. O
`importcontent -check` compara os dois e falha quando divergem — é o gate para editar conteúdo.

### 3. Resíduo do servidor Win32 (candidato a arquivamento)

| Item | Por que é resíduo |
|---|---|
| `TMsrv.bat`, `DBsrv.bat`, `deletar Logs.bat` | launchers de Windows para binários que não rodam mais |
| `DBsrv/run/W2 - Editor de Contas.exe` | editor de conta em arquivo; conta hoje é Postgres + portal (`/contas`) |
| `TesteDie.txt`, `AttackDieTeste.txt`, `Chall.txt`, `Chall_00_00.txt`, `Guild*.txt`, `GuildInfo/`, `Ranking.txt` | estado de servidor de teste em arquivo; hoje vive em tabela |
| `ImportUser_sample - Copia.txt`, `ImportUser/.gitkeep` | sobra de importação manual de contas |
| `DBsrv/run/{serverlist.bin,serverlist.txt}` | a lista de servidores do cliente é gerada por `client-tools/serverlist`; o `.bin` versionado com IP privado é justamente o que o `.gitignore` evita |
| `extraitem.bin`, `data00.csv` | sem leitor no Go (confirmar antes de mover) |

## Alvo proposto

```
tmserver/content/            # runtime do jogo: mapas, taxas, gameconfig, BaseMob, BaseSummon, serv00.htm
tmserver/content/import/     # insumo do importcontent: npc/, ItemList.csv, SkillData.csv, InitItem.csv
```

O ganho não é cosmético: com o conteúdo dentro do módulo do servidor, `-content` deixa de apontar para
uma pasta de distribuição do jogo e o teste passa a usar fixture local, em vez de depender de arquivo
proprietário montado.

## Por que **não** foi executado nesta rodada

A movimentação é mecanicamente simples e operacionalmente arriscada, porque toca quatro superfícies ao
mesmo tempo:

1. `docker-compose.yaml` monta `./Release:/Release:ro` em `tmserver` e `webserver` (`-content /Release`).
2. `Release/` está no `.gitignore` por herança (`[Rr]elease/`) — mover para `tmserver/content/` passa a
   versionar binário de 16 MiB; é preciso decidir o que versionar e o que continua fora.
3. **68 arquivos Go** citam caminhos de `Release/` (a maioria testes de conteúdo proprietário).
4. `Dockerfile`/`.dockerignore` e os comandos de `exptool`, `worldatlas` e `importcontent` recebem o
   caminho por flag.

Fazer isso de uma vez, sem o cliente Windows para validar, trocaria um problema de organização por um
risco de servidor não subir. Fica como `LEGACY-02`, executável em quatro passos verificáveis:

1. Criar `tmserver/content/` com os arquivos da categoria 1 e apontar o default de `-content`.
2. Rodar `go test ./...` e corrigir as fixtures (esperado: caminhos, não lógica).
3. Ajustar compose/Dockerfiles e subir a stack; conferir `walkability grid baked` e `NPCs spawned`.
4. Arquivar a categoria 3 em `Release/_legacy/` (ou fora do repo) e só então remover `Release/` do
   caminho de build do servidor.

## Regra que já vale

`Release/` é **entrada**, nunca saída. Nem o portal nem o web-api escrevem ali (invariante 7 do
`AGENTS.md`): taxa editada no portal vira linha em `content_rate_override` e o arquivo continua sendo o
fallback. Qualquer proposta de "salvar direto no Release" reintroduz a segunda fonte de verdade.
