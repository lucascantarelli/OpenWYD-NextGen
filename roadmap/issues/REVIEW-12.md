# REVIEW-12 — Revisão permanente: arquitetura e código do servidor

**Fase:** revisão contínua · **Depende de:** — · **Bloqueia:** — · **Esforço:** contínuo · **Estado:** 🔁 viva

## Objetivo

Ponto de controle técnico do servidor Go (`tmserver`, `dbserver`, `binserver`, `webserver`,
`internal/`): registrar evolução, dívidas, decisões e achados de auditoria a cada rodada, sem fechar a
issue enquanto o domínio precisar de acompanhamento.

## Escopo

- Dono único do estado do mundo: nenhuma mutação fora de `world.World.Run`, nenhum lock novo.
- Tamanho e responsabilidade dos pacotes, duplicação, código morto e caminhos legados vivos.
- Tratamento de erro consistente, contratos explícitos, docstrings em português.
- Testabilidade: cobertura por pacote, testes que só passam por sorte, skips com motivo.
- Compatibilidade de wire a cada mudança de pacote.

## O que registrar a cada rodada

| Item | Como medir |
|---|---|
| Arquivos/linhas Go | `find . -name '*.go' -not -path './.git/*' | wc -l` e soma de linhas |
| Arquivos de teste | `find . -name '*_test.go' | wc -l` |
| TODO/FIXME/HACK no código | `grep -rInE '\b(TODO\|FIXME\|XXX\|HACK)\b' --include='*.go' .` |
| Skips sem motivo | `bash scripts/check-skips.sh` |
| Comentário fora do padrão | `bash scripts/check-comments.sh` |
| Caminho legado vivo | revisão manual + `git grep` do nome antigo |
| Achados novos | uma linha por achado, com arquivo:linha e issue filha |

## Critérios de aceite (por rodada, não da issue)

- [ ] `go build ./...`, `go vet ./...` e `go test ./...` verdes com a saída registrada
- [ ] Todo achado tem issue filha ou justificativa de "não é problema"
- [ ] Dívida nova entrou como issue, nunca como `TODO` solto
- [ ] Nenhum lock novo no estado do mundo (checado por revisão do diff)

## Histórico

| Data | Linhas Go | Testes | Achados | Destino |
|---|---|---|---|---|
| 06/11/2026 | 152.381 em 599 arquivos | 284 arquivos | 1 `TODO(#115)` em `handler/autotrade.go` (EF_NOTRADE), 43 `t.Skip` (TEST-01 afirma 42) | issue filha para o `TODO`; recontagem de skip na próxima rodada |
| 06/11/2026 (onda 2) | +4 skips justificados (paridade de conteúdo e de regiões do worldatlas; três deles por árvore/banco ausentes) | 47 `t.Skip`, todos com `motivo:` | `check-skips` verde | nenhuma ação: os skips são condição de ambiente, não teste escondido |
| 06/11/2026 | novos pacotes `tmserver/internal/bot` e `tmserver/cmd/botlab` (8 arquivos, 1.900 linhas) | 2 arquivos de teste | `check-comments.sh` estava **vermelho** em `internal/crossover`: 2 comentários em inglês e 2 referências a arquivo `.cpp` | corrigido no mesmo dia; gate volta a verde |
