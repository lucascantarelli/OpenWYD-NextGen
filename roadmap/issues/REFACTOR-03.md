# REFACTOR-03 — Comentários e docstrings em português, sem referência a issue/roadmap/legado

**Fase:** revisão · **Depende de:** — · **Esforço:** L · **Estado:** 🟡 em andamento (fátias de 06/11: 9 diretórios no gate; dívida repo-wide medida em 8.297 linhas)

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

## Fatia entregue (06/11/2026)

A issue inteira é L (396 comentários em inglês, 109 referências históricas). A fatia escolhida foi a
que o próprio passo 2 pede primeiro — `internal/*`, o que mais de um binário consome — limitada ao que
cabe com leitura e teste no mesmo dia.

**Diretórios migrados e agora no gate:** `internal/textenc`, `internal/droprate`, `internal/crossover`.

O que mudou neles:

- comentário em inglês traduzido (incluindo docstring de pacote, que é o que mais engana leitor novo);
- referência histórica removida do código: `Source/Code/TMSrv/_MSG_MessageWhisper.cpp:75-95`,
  `docs/migration/canais.md §8.1.2`, `CFileDB.cpp:662-812`, `Basedef.h`, `issue #43`;
- o porquê continua escrito, sem depender do roadmap para ser entendido.

**Gate:** `scripts/check-comments.sh` (novo) reprova comentário em inglês e referência histórica nos
diretórios listados. A lista é incremental de propósito: ligar repo-wide hoje deixaria o job vermelho
para sempre, e gate assim é gate desligado.

```bash
bash scripts/check-comments.sh                        # 3 diretório(s) limpos
bash scripts/check-comments.sh internal/ratetables    # FALHA — prova de que o gate não é vazio
```

A segunda linha é a prova que importa: rodar o gate num diretório ainda sujo **reprova**. A primeira
versão do script passava nos seis diretórios que eu tinha escolhido enquanto eles ainda tinham
comentário em inglês — padrão bom demais (só `// The …`) e falso negativo. Ampliei para uma lista de
palavras que não ocorrem em português e reduzi a fatia ao que ficou de fato limpo.

### Dívida medida que permanece

| Diretório | Comentários a migrar |
|---|---:|
| `tmserver/internal/handler` | 194 |
| `tmserver/internal/world` | 29 |
| `internal/store` | 28 |
| `tmserver/internal/content` | 19 |
| `tmserver/internal/protocol` | 17 |
| `tmserver/internal/refine` | 10 |
| `tmserver/internal/level` | 7 |
| `tmserver/internal/dbclient` | 7 |
| `internal/savefmt` | 5 |
| demais | ~25 |

`internal/savefmt` ficou fora da lista do gate mesmo com a tradução parcial já feita: ainda tem
`Basedef.h`/`data-formats.md` em várias constantes, e listar um diretório meio limpo é pior que não
listar.

## Fatia 2 (06/11/2026)

**Diretórios migrados nesta fatia:** `internal/migrations`, `internal/playcode`, `internal/secret`,
`internal/secure`, `internal/itemeffects`, `internal/teleportdata`. O gate passou de 3 para
**9 diretórios**.

Antes/depois medido com o mesmo regex do gate, dentro dos seis diretórios:

| Diretório | inglês antes | depois | refs históricas antes | depois |
|---|---:|---:|---:|---:|
| `internal/migrations` | 2 | 0 | 0 | 0 |
| `internal/playcode` | 16 | 0 | 0 | 0 |
| `internal/secret` | 15 | 0 | 0 | 0 |
| `internal/secure` | 16 | 0 | 0 | 0 |
| `internal/itemeffects` | 34 | 0 | 2 | 0 |
| `internal/teleportdata` | 21 | 0 | 3 | 0 |
| **total** | **104** | **0** | **5** | **0** |

O que mudou: docstrings de pacote e de símbolo exportado em português; as referências a
`Source/Code/ItemEffect.h`, `GetFunc.cpp`, `internal/migrations/0034_…`, `game-rules.md`,
`migration-plan.md` e `issue 44` saíram do código, e o porquê de cada regra continuou escrito (a
ordem das resistências, o portão da Fada do Vale, o limite de 12 bytes do campo de senha). Nenhuma
linha de código executável mudou de semântica; os testes dos seis pacotes seguem os mesmos.

Provas desta fatia: `bash scripts/check-comments.sh` RC 0 (**9 diretórios**), `go test -race -count=1`
nos seis pacotes RC 0 (18 s), `go build ./...` RC 0 (42 s), `go vet ./...` RC 0 (15 s),
`check-skips` RC 0 (47, inalterado), `check-docs-links` RC 0 (141, inalterado), `gofmt -l` vazio.

## Fatia 4 (06/11/2026)

**Diretórios migrados:** `webserver/cmd` (17 → 0) e `binserver` (22 → 0). Gate: 11 → **13
diretórios**. Migração por arquivo inteiro (ler o arquivo, reescrever todos os comentários numa edição),
com verificação de lote (`gofmt -w` + `go build ./...` + detector) ao fim de cada bloco. Saíram
`web-platform-plan.md`, `migration-plan.md §4`, `issue 32/34/58/62`; ficou o porquê (códigos de login
só por ambiente para não aparecerem na lista de processos, catálogo vazio = seed faltando e não
"sem conteúdo", `_AUTH_GAME` de 196 bytes não reproduzido).

O detector pegou um falso positivo real de redação: `papéis` casa `\bis\b` porque `é` não é caractere
de palavra no grep; a linha foi reescrita com `permissões` em vez de afrouxar o regex.

## Fatia 3 (06/11/2026)

**Diretórios migrados:** `internal/ratetables` e `internal/npctemplate`. Placar pelo detector do
próprio gate (`bash scripts/check-comments.sh <dir>`, contando as linhas de violação que ele imprime):

| Diretório | violações antes | depois |
|---|---:|---:|
| `internal/ratetables` | 45 | 0 |
| `internal/npctemplate` | 56 | 0 |
| **total** | **101** | **0** |

O gate passou de 9 para **11 diretórios**. Traduções de docstring de pacote, de símbolo exportado e
de comentário de teste; saíram as referências a `data-formats.md`, `issue #244` e `issue 58`, e o
porquê continuou escrito (a tabela de refino só sobrescreve índices listados, o cardume de chaves por
família no CompRate, o layout cp1252 do SancRate, a normalização 756/920→816 bytes). Nenhuma linha
executável mudou de semântica.

Provas desta fatia: `bash scripts/check-comments.sh` RC 0 (11 diretórios),
`go test -race -count=1 ./internal/ratetables/ ./internal/npctemplate/` RC 0 (15 s), `gofmt -l` nos
sete arquivos vazio. O gate continua reprovando diretório sujo:
`bash scripts/check-comments.sh tmserver/internal/world` → RC 1.

### Dívida restante, medida (não estimada)

O critério repo-wide desta issue **não** está cumprido. Placar autoritativo = detector do gate por
diretório, medido em 06/11 depois das três fatias:

| Área | violações do gate |
|---|---:|
| `tmserver/internal/handler` | 6.095 |
| `tmserver/internal/world` | 1.497 |
| resto de `tmserver/internal` (content, protocol, dbclient, …) | 2.221 |
| `webserver/internal` | 1.239 |
| `internal/store` | 808 |
| `tmserver/cmd` | 359 |
| `dbserver` | 198 |
| `internal/domain` | 173 |
| `internal/savefmt` | 164 |
| **total** | **12.754** |

Zero: `internal/{crossover,droprate,itemeffects,migrations,npctemplate,playcode,ratetables,secret,secure,teleportdata,textenc}`,
`binserver` e `webserver/cmd` (13 diretórios, os que o gate cobre).

Essa é a razão de a issue continuar aberta: são ~12,8 mil comentários distintos, e a maioria exige
ler o código do símbolo para traduzir sem mentir — o risco que a própria issue nomeia. O gate já roda
no CI (`.github/workflows/_go.yml`), então a dívida não cresce enquanto é paga, e o placar acima é o
roteiro: próximo lote `internal/domain` (173) e `internal/savefmt` (164), depois `internal/store`
(808); `tmserver/internal/handler` (6.095) precisa ser fatiado por arquivo.

Por que a issue continua aberta: o volume é de ~8,2 mil linhas de comentário em mais de 500 arquivos.
Traduzir isso em massa, sem ler o código de cada símbolo, produz exatamente o defeito que o risco
desta issue descreve — comentário que mente —, então a migração segue pacote a pacote, com o gate
crescendo a cada fatia. O gate já roda no CI (`.github/workflows/_go.yml`), então a dívida não
cresce enquanto é paga.

## Riscos/rollback

Risco: tradução em massa produzir comentário que mente sobre o código. Mitigação: um pacote por
commit, leitura do código antes de traduzir, teste verde antes/depois. Rollback: commit por pacote.
