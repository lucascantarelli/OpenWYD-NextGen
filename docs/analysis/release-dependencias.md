# Conteúdo do servidor: o que o Go lê, onde vive e o que foi arquivado

Estado depois da **LEGACY-02** (06/11/2026). A árvore de distribuição do Win32 deixou de existir: o conteúdo
executável vive em **`tmserver/content/`**, dentro da árvore do servidor Go, e nenhum serviço lê
caminho fora dali.

## Inventário final

| Grupo | Onde | Versionado? |
|---|---|---|
| Taxas e config | `tmserver/content/Common/Settings/*.txt`, `serv00.htm` | sim |
| Templates por classe | `tmserver/content/DBsrv/run/BaseMob/{TK,FM,BM,HT}` | sim |
| Invocações do BM | `tmserver/content/TMsrv/run/BaseSummon/*` | sim |
| Referência de formato | `Regions/Guard/LevelItem/Language/ItemEffect.h/Rates/QuestDiaria/gameconfig`, `ItemCSum.h` | sim |
| **Mapas** | `TMsrv/run/HeightMap.dat` (16 MiB), `AttributeMap.dat` (1 MiB) | **não** — `make content-sync` |
| Arquivo morto Win32 | `tmserver/content/_legacy/` | não (ignorado, nunca lido) |

**60 arquivos** cobertos por `tmserver/content/content.sha256` (config + mapas). O manifesto é a
autoridade: `make content-check` reprova arquivo faltando ou alterado.

### Decisão de versionamento

Os mapas são 17 MiB de binário que muda quando o mapa muda e que ninguém revisa em diff. Commitá-los
só engorda o histórico, então ficam fora do git e entram por bootstrap:

```bash
make content-sync W2PP_CONTENT_SOURCE=/caminho/para/Release   # copia o que falta
make content-check                                            # confere o manifesto
```

O script **nunca sobrescreve** arquivo presente e confere o SHA-256 de tudo ao final. Um clone novo
sem os mapas sobe o servidor (aviso `mob pathfinding disabled`), o que é degradação visível, não
falha silenciosa.

## O que foi arquivado (`_legacy/`, 55 MB → 17 MB de conteúdo real)

| Item | Por que saiu |
|---|---|
| `msvcr*/msvcp*/ucrtbase.dll` (7 MB) | runtime do servidor C++ no Windows |
| `TMsrv.bat`, `DBsrv.bat`, `deletar Logs.bat` | launchers de binários que não existem mais |
| `W2 - Editor de Contas.exe` | conta hoje é Postgres + portal (`/contas`) |
| `AttackDieTeste.txt` (4,6 MB), `TesteDie.txt` | estado de teste de dano em arquivo |
| `data00.csv` (19 MB) | dump sem leitor no Go |
| `GuildInfo`, `Guild*`, `Ranking`, `Chall*`, `ChampionCity*` | estado de guilda/ranking em arquivo → tabela |
| `ImportUser*`, `account/A/antonio` | amostra de importação manual de contas |
| `serverlist.{bin,txt}` (TMsrv e DBsrv) | lista gerada por `client-tools/serverlist`; o `.bin` com IP privado é justamente o que o `.gitignore` evita |
| `NPCGener.new.txt`, `boss/`, `DBsrv/run/npc`, `ItemList.bin` (910 KB) | insumo do servidor legado; catálogo e spawns vivem no Postgres desde a issue 58 |
| `extraitem.{bin,csv}`, `Rates.txt`, `Mac/Server/settings/config/admin/localip.txt` | configs do DBsrv Win32 |

Nada foi apagado do disco: o diretório inteiro está em `tmserver/content/_legacy/`, fora do git e
fora da imagem (`.dockerignore`), servindo de referência para engenharia reversa.

## Quem lê o quê (código)

| Consumidor | Caminho |
|---|---|
| `tmserver` | `Common/Settings/{CompRate,SancRate,QuestsRate,CastleQuest,RefineRate}.txt`, `Common/serv00.htm`, `DBsrv/run/BaseMob`, `TMsrv/run/BaseSummon`, os dois `.dat` |
| `webserver` | mesmas taxas + `AttributeMap.dat` (editor de mapa) |
| `portal` | `Common/ItemList.csv` **opcional** (preço base); hoje o catálogo vem do Postgres |
| `worldatlas` | `TMsrv/run/{Regions.txt,HeightMap.dat,AttributeMap.dat}` |
| `importcontent` | `Common/{ItemList.csv,SkillData.csv}`, `TMsrv/run/InitItem.csv` — insumo externo de backup; não existe mais nesta árvore |

Default de `-content` virou `tmserver/content` (host) e `/content` (imagem). O compose monta
`./tmserver/content:/content:ro` em `tmserver`, `tmserver2`, `webserver` e `portal`.

## Regra que continua valendo

`tmserver/content/` é **entrada**, nunca saída. Nem o portal nem o web-api escrevem ali: taxa editada
no portal vira linha em `content_rate_override` e o arquivo é o fallback. Qualquer proposta de
"salvar direto no conteúdo" reintroduz a segunda fonte de verdade.

## Pendências

- `importcontent` segue aceitando uma árvore externa para reprocessar backup, mas o insumo
  (`ItemList.csv`, `SkillData.csv`, `InitItem.csv`) não existe mais neste repositório: o caminho de
  reprodução do catálogo é `seedgen`/seeds SQL.
- `exptool`/`make exp-data` operam numa árvore `npc/` que não está mais aqui pelos mesmos motivos.
- Se um dia os mapas forem regerados por ferramenta nossa, o bootstrap pode virar download versionado
  em vez de cópia da árvore legada.
