# Issue 61 — Remover os artefatos do DropTool (`ItemDropList.txt`, `MobDropList.txt`)

**Fase:** 7 · **Depende de:** — · **Esforço:** S · **Estado:** ⏳ proposta

## O que é o DropTool

Ferramenta **externa, legada e Win32**, em `Source/Code/DropTool/main.cpp` (C++). Ela não roda no
servidor nem é chamada por ninguém: é um utilitário manual que

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
(issue 34) recalculam a partir do conteúdo. `grep -rn "ItemDropList" --include=*.go` não encontra
nenhum leitor além de comentários.

## Ações

1. Apagar `Release/TMsrv/run/ItemDropList.txt` e `MobDropList.txt` (242 KB de artefato).
2. Mover `Source/Code/DropTool/` para `Source/Code/_legacy/` (referência histórica, fora de qualquer
   build) — junto com `ItensExceptions.txt` se estiver no `Release`.
3. `Remove-Item`/`rm` guiado por `git rm` (o arquivo é versionado; `Release/` está no `.gitignore` por
   herança, então confirmar com `git status --ignored Release` antes de concluir que saiu).
4. Nota em `docs/analysis/migracao-legado.md` e na skill `wyd-economy`: o relatório é o `/drops`.

## Critérios de aceite

- [ ] `git ls-files Release/TMsrv/run | grep -i droplist` vazio
- [ ] `grep -rn "ItemDropList\|MobDropList" --include=*.go .` só em documentação/comentário
- [ ] `/drops` e `/drops/carry` continuam respondendo com o catálogo real (smoke do portal)
- [ ] Skill `wyd-economy` aponta para as telas, não para o `.txt`
