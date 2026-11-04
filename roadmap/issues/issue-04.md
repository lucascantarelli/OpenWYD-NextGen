# Issue 04 — Subir e validar o tmServer com conteúdo

**Fase:** 0 · **Depende de:** 02, 03 · **Bloqueia:** 05, 06, 07, 10–13, 17 · **Esforço:** M · **Estado:** ✅ concluída (2026-11-02)

## Objetivo

`tmserver` no ar em `8281` (jogo) e `80` (status), com a árvore `Release/` carregada de fato:
catálogos, templates de NPC, mapas e itens de mundo.

## Contexto / evidência

O binário carrega conteúdo em `loadContent` (`tmserver/cmd/tmserver/main.go`): taxas e catálogos são
**fatais** se quebrarem, mapas/BaseMob/InitItem são **warning**. Um boot "verde" mas sem conteúdo
produz um servidor que aceita TCP e não entrega jogo — o pior tipo de falso positivo. Por isso o
aceite desta issue é por **linha de log**, não por `docker compose ps`.

## Passos

```bash
# (comandos rodam na raiz do repositório)

# 1. subir o jogo (o compose já passa -content /Release, -client-version 12000, -status-addr :80)
docker compose up -d --build tmserver

# 2. acompanhar o boot
docker compose logs --since 5m tmserver
```

Sequência esperada (nomes exatos das chaves do slog):

```
tmserver config            client_version=12000 content=/Release …
content loaded             items=… skills=… sancrate_ori=[…] sancrate_lac=[…]
status server listening    addr=:80
base mob templates loaded  classes=4
base summon templates loaded count=…
vine template loaded
castle quests loaded       count=…
mob template catalog       layouts=…
walkability grid baked     dim=4096
NPCs spawned               generators=… mobs=… templates=… merchant_blocks_skipped=…
world items seeded         count=…
tmserver listening         addr=:8281
```

3. Verificar ausência de warnings que degradam o jogo:

```bash
docker compose logs tmserver | grep -E "npc template load failed|NPC templates missing|attribute map not loaded|height map not loaded|base mob templates not loaded|unbalanced Exp|training-grounds generators not marked"
```

4. Validar o status HTTP (o que o cliente sonda antes do CPSock):

```bash
curl -sS -D - http://127.0.0.1/ -o /tmp/serv00.txt | head -5
cat /tmp/serv00.txt
```

5. Validar que a porta de jogo aceita conexão:

```bash
# Windows
powershell -NoProfile -Command "Test-NetConnection 127.0.0.1 -Port 8281 | Select TcpTestSucceeded"
# ou, do próprio container/rede
docker run --rm --network host busybox nc -zv 127.0.0.1 8281
```

## Arquivos

- Alterados: nenhum código. Se precisar ajustar flag, altere o `command:` do serviço
  `tmserver` em `docker-compose.yaml` (ou use `docker compose run` para teste pontual).
- Consulta: `tmserver/cmd/tmserver/main.go` (`loadContent`, `spawnNPCs`, `serveStatusHTTP`).

## Testes / verificação

```bash
# a) contagem de mobs plausível (milhares, não dezenas)
docker compose logs tmserver | grep "NPCs spawned"
#    esperado: mobs na casa dos milhares (referência: ~12–13k no equilíbrio do gerador)

# b) mapas realmente usados
docker compose logs tmserver | grep "walkability grid baked"
#    esperado: dim=4096

# c) nenhum template rejeitado
docker compose logs tmserver | grep -c "npc template load failed"   # esperado: 0

# d) status responde 200 com corpo do serv00.htm
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1/
```

## Critérios de aceite

- [ ] Todas as linhas de log da sequência esperada presentes
- [ ] `walkability grid baked dim=4096` (HeightMap + AttributeMap carregados)
- [ ] `base mob templates loaded classes=4`
- [ ] `NPCs spawned` com milhares de mobs e `npc template load failed` = 0
- [ ] `curl http://127.0.0.1/` responde 200 com o conteúdo de `serv00.htm`
- [ ] Porta 8281 aceita conexão
- [ ] Log sem `unbalanced Exp` em massa (se houver, abrir issue de dados — ver issue 13)

## Riscos / rollback

- **Risco:** porta 80 ocupada no Windows → tmServer sobe mas status falha. **Mitigação:** checar
  antes (issue 02); alternativa `-status-addr :8080` com publicação correspondente, ciente de que o
  cliente real espera `:80`.
- **Risco:** conteúdo montado read-only (`./Release:/Release:ro`) — edição no host não recarrega. O
  servidor lê **no boot**: reinicie `tmserver` após mudar conteúdo.
- **Risco:** imagem em cache sem o código novo. **Mitigação:** `docker compose build --no-cache tmserver`
  quando o comportamento não corresponder ao código.
- **Rollback:** `docker compose stop tmserver` (mantém db e dados).
