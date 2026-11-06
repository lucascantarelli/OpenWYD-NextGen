# Issue 61 — Remover os artefatos do DropTool (`ItemDropList.txt`, `MobDropList.txt`)

**Fase:** 7 · **Depende de:** — · **Esforço:** S · **Estado:** ✅ fechada (05/11/2026)

## O que é o DropTool

Ferramenta **externa, legada e Win32**, em `Source/Code/_legacy/DropTool/main.cpp` (C++). Ela não roda
no servidor nem é chamada por ninguém: é um utilitário manual que

1. lê `ItensExceptions.txt` (faixas de item a ignorar),
2. varre `./npc/*` carregando os `STRUCT_MOB` de 816 bytes,
3. e escreve **dois relatórios em texto**: `ItemDropList.txt` ("item N: quais mobs o dropam") e
   `MobDropList.txt` ("mob X: quais itens ele dropa").

O drop real **não** sai daí: ele é o `Carry[64]` de cada template, e a raridade é o **slot** do Carry
(`DropRate[64]` no código), não o item. Por isso `ItemDropList.txt` é um **artefato gerado**, uma
fotografia textual do conteúdo — não uma entrada.

## Por que não migrar para o banco

O que ele responde já é respondido por consulta: `npc_template.Carry[]` (issue 58) + `item_catalog`
dão "quem dropa o item N" com um `JOIN`, sempre atualizado, sem arquivo intermediário. Guardar o
relatório seria guardar uma **view materializada obsoleta** — modelar o vazio.

No Go isso já foi substituído: `internal/droptool` + as telas `/drops` e `/drops/carry` do portal
(issue 34) recalculam a partir do conteúdo. Não existe leitor Go dos dois `.txt`.

## Ações

1. ~~Apagar `Release/TMsrv/run/ItemDropList.txt` e `MobDropList.txt`~~ — feito (`MobDropList.txt` já
   não existia; o `ItemDropList.txt` de 242.676 B foi removido).
2. ~~Mover `Source/Code/DropTool/` para `Source/Code/_legacy/`~~ — feito com `git mv` (blame
   preservado), junto com o `.vcxproj` e o `main.h`.
3. Remoção guiada por `git rm`/`rm`; `Release/` está no `.gitignore` por herança, então o arquivo
   nunca esteve versionado — a remoção é local por natureza.
4. ~~Nota em `docs/analysis/migracao-legado.md` e na skill `wyd-economy`~~ — feito: a skill passou a
   apontar `npc_template.Carry[]` + telas, e o doc marca o novo caminho `_legacy`.

## Critérios de aceite

- [x] `git ls-files Release/TMsrv/run | grep -i droplist` vazio
- [x] `grep -rn "ItemDropList\|MobDropList" --include=*.go .` só em documentação/comentário (zero em código)
- [x] `/drops` e `/drops/carry` continuam respondendo (307 sem sessão, o mesmo guard de `/itens`)
- [x] Skill `wyd-economy` aponta para as telas, não para o `.txt`

## Evidência de execução (05/11/2026)

```text
$ ls -l Release/TMsrv/run/ItemDropList.txt
-rw-r--r-- 1 Canta 197609 242676 ... ItemDropList.txt      # antes
$ rm -f Release/TMsrv/run/ItemDropList.txt                  # removido
$ ls Release/TMsrv/run | grep -iE 'drop|exception'          # vazio

$ git ls-files | grep -iE 'DropList'                        # vazio (nunca versionado)

$ grep -rn --include=*.go -E "ItemDropList|MobDropList" .    # zero ocorrências

$ git mv Source/Code/DropTool Source/Code/_legacy/DropTool
$ ls Source/Code/_legacy/DropTool
DropTool.vcxproj  DropTool.vcxproj.filters  main.cpp  main.h

$ for p in /drops /drops/carry /itens; do curl -s -o /dev/null -w "$p %{http_code}\n" localhost:3000$p; done
/drops 307 · /drops/carry 307 · /itens 307     # guard de sessão, mesma resposta da rota já validada
```

Documentação ajustada na mesma entrega: `AGENTS.md` (árvore de conteúdo sem `ItemDropList.txt`),
`docs/migration/config-ops.md`, `docs/migration/data-formats.md` (linha marcada como artefato
removido), `docs/analysis/migracao-legado.md` (caminho `_legacy`) e
`.agents/skills/wyd-economy/SKILL.md` (tabela de drop = `Carry[]` + `/drops`).
