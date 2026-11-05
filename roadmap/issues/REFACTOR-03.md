# REFACTOR-03 — Comentários e docstrings em português, sem referência a issue/roadmap/legado

**Fase:** revisão · **Depende de:** — · **Esforço:** L · **Estado:** ⏳ proposta

## Objetivo

O código Go fala a mesma língua do resto do repositório (português), descreve **o que faz hoje** e não
depende de número de issue, caminho de roadmap ou arquivo legado para ser entendido.

## Contexto/evidência (05/11/2026)

Medido neste checkout (570 arquivos `.go`, 263 de teste):

```bash
git grep -l -E '// (The|This|Returns|Sets|Creates|Handles|Loads|Parses|Builds|Reads) ' -- '*.go' | wc -l
# 187 arquivos com comentário no padrão inglês
git grep -l -E 'issue [0-9]|roadmap/|docs/' -- '*.go' | wc -l
# 80 arquivos citando issue/roadmap/docs dentro do código
```

Exemplos que mostram o problema (comentário que envelhece junto com o roadmap):

- `webserver/internal/itemcatalog/itemcatalog.go` — doc do pacote descreve `ItemList.csv`, que saiu
  do repositório na issue 58; o leitor de hoje é induzido a procurar arquivo inexistente.
- `webserver/cmd/webserver/main.go` — comentários citam `ListMerchantTemplates/ListItemCatalog`
  “return empty lists” quando o comportamento mudou (issue 62).
- `internal/itemeffects/itemeffects.go` — inglês + caminho de `Source/Code/ItemEffect.h`.

## Passos

1. Congelar o padrão em `docs/development-guidelines/Go-comments.md`: docstring em português, começa
   pelo nome do símbolo, descreve contrato e efeitos colaterais, **sem** número de issue, caminho de
   roadmap ou nome de arquivo legado (histórico fica no git e no `roadmap/`).
2. Varredura por pacote, começando por `internal/*` e `api/*` (consumidos por mais de um binário),
   depois `tmserver/internal/{world,handler,protocol}`, `webserver/internal/*`, `dbserver/*`.
3. Ativar no `.golangci.yml`: `revive` (`exported`, `package-comments`), `godot` e uma regra própria
   (`forbidigo`/`gci` não cobrem texto) via `scripts/check-comments.sh` que falha em
   `issue [0-9]`/`roadmap/`/`Source/Code` dentro de `.go` **escrito à mão** (gerados: ver passo 5).
4. Ligar o script no CI em modo incremental (só arquivos alterados) para a dívida não crescer.
5. Para `*.pb.go`: o comentário vem do `.proto`. Corrigir no `.proto` e regenerar (BUILD-01), nunca
   editar o gerado.

## Arquivos

- `docs/development-guidelines/Go-comments.md` (novo)
- `scripts/check-comments.sh` (novo), `.golangci.yml`, `.github/workflows/ci.yml`
- `internal/**`, `api/**`, `tmserver/**`, `dbserver/**`, `webserver/**`, `binserver/**`

## Testes/verificação

```bash
go build ./... && go test ./...
bash scripts/check-comments.sh          # esperado: 0 violação nos diretórios limpos
git grep -n -E 'issue [0-9]|roadmap/|Source/Code' -- '*.go' | grep -v '\.pb\.go' | wc -l   # 0
```

## Critérios de aceite

- [ ] 0 arquivo com padrão de comentário inglês nos pacotes limpos (lista versionada no script).
- [ ] 0 citação de `issue N`/`roadmap/`/`docs/`/`Source/` em `.go` escrito à mão.
- [ ] Todo símbolo exportado de `internal/` e `api/` com docstring no padrão.
- [ ] Script de verificação rodando no CI.

## Riscos/rollback

Risco: tradução em massa produzir comentário que mente sobre o código. Mitigação: um pacote por
commit, leitura do código antes de traduzir, teste verde antes/depois. Rollback: commit por pacote.
