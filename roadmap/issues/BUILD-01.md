# BUILD-01 — Renomeação do módulo (`openwyd-nextgen`): fechar a inconsistência

**Fase:** revisão · **Depende de:** — · **Esforço:** M · **Estado:** ✅ concluída (05/11/2026)

## Objetivo

O repositório compilar e testar com o módulo `openwyd-nextgen`, sem resíduo do path antigo
(`github.com/jeanluca/w2pp-` + `openwyd`) e sem que um rename volte a quebrar o build por semanas.
O path aparece partido de propósito: o gate deste repositório (`scripts/check-build.sh`) proíbe a
string contígua em **qualquer** arquivo versionado, inclusive na documentação.

## Contexto/evidência (05/11/2026)

O rename foi feito à mão e ficou pela metade: `go.mod` já dizia `module openwyd-nextgen` enquanto
**382 arquivos** ainda importavam o path antigo — build impossível. Números medidos:

```bash
# a concatenação de strings do shell monta o path sem deixá-lo contíguo no arquivo
git grep -l "github.com/jeanluca/w2pp-""openwyd" | wc -l   # 382 (antes)
git grep -l "github.com/jeanluca/w2pp-""openwyd" | wc -l   # 0   (depois da varredura de 05/11)
```

Depois da varredura mecânica sobraram **dois imports com o path nu**, que o compilador denuncia:

```
binserver/cmd/binserver/main.go:26:2: package openwyd-nextgen is not in std (/usr/local/go/src/openwyd-nextgen)
binserver/internal/grpcsrv/server_test.go:8:  idem (binv1 "openwyd-nextgen")
```

Ambos deviam ser `binv1 "openwyd-nextgen/api/bin/v1"` — o `go_package` do `api/bin/v1/bin.proto` é
`openwyd-nextgen/api/bin/v1;binv1`. Corrigido nesta execução.

Testes que ainda falham no mesmo run (a investigar com a stack no ar):

```
FAIL openwyd-nextgen/webserver/internal/grpcsrv
FAIL openwyd-nextgen/tmserver/internal/binclient
FAIL openwyd-nextgen/tmserver/internal/dbclient
panic: runtime error: slice bounds out of range [-1:]
```

O `panic` com slice `[-1:]` em pacotes que só falam gRPC aponta para **`.pb.go` divergente do
`.proto`**: o `Makefile` tem o alvo `proto` (`protoc --go_out=. --go_opt=module=openwyd-nextgen`), mas
nenhum gate verifica que os gerados estão atualizados.

## Passos

1. ~~Varredura do path antigo no repo inteiro~~ — feito: 382 → 0.
2. ~~Corrigir os dois imports nus~~ — feito.
3. **Regenerar os protos** com o módulo novo: `make proto` (exige `protoc`, `protoc-gen-go`,
   `protoc-gen-go-grpc`; documentar a instalação no próprio alvo, como o comentário já faz).
4. Rodar `go build ./... && go vet ./... && go test ./...` e tratar o `panic [-1:]` pela causa
   (gerado obsoleto ou descriptor sem campo), não pelo sintoma.
5. **Gate**: no CI, falhar se o path antigo aparecer em qualquer arquivo versionado e se `make proto`
   alterar arquivo versionado. Implementado em [`scripts/check-build.sh`](../../scripts/check-build.sh),
   chamado por `make gate` (CI) e por `make ci` (local). Sem isso, o próximo rename volta a quebrar o
   build em silêncio.
6. Atualizar `docs/migration/SESSION-PRIMER.md` (módulo), `CLAUDE.md` e qualquer comando com o path.

## Arquivos

- `go.mod`, `api/**/*.proto` + `*.pb.go`
- `binserver/cmd/binserver/main.go`, `binserver/internal/grpcsrv/server_test.go`
- `.github/workflows/ci.yml` (gate de path antigo + protos em dia)
- `Makefile` (`proto` com instalação dos plugins documentada/automatizada)

## Testes/verificação

```bash
bash scripts/check-build.sh --path-only    # path antigo: 0 ocorrências
bash scripts/check-build.sh --proto-only   # make proto sem diff em api/
make proto && git diff --exit-code api/    # idem, à mão
go build ./... && go vet ./... && go test ./...
```

## Critérios de aceite

- [x] Zero ocorrência do path antigo no repositório.
- [x] Zero import com o path nu.
- [x] Zero import com o path nu **em arquivo de teste** — o `go build` não os vê e eles ficaram
  para trás na varredura de 05/11: `binserver/internal/grpcsrv/server_test.go` importava
  um import que a varredura mecânica corrompeu (`openwyd-nex…al/billing`, com o `al` costurado
  no lugar errado) e `webserver/internal/itemcatalog/fromentries_test.go` importava
  `internal/domain` nu. Corrigidos; era o que mantinha `go vet`/`go test` vermelhos.
- [x] `go build ./...` e `go vet ./...` verdes neste checkout (05/11, container `golang:1.26-alpine`:
  build exit 0; vet **sem nenhum diagnóstico** depois da correção acima).
- [x] `go test ./...` verde (05/11: RC=0, 57 pacotes `ok`, 21 sem teste, nenhum `FAIL`).
- [x] Gate no CI para path antigo e para gerados desatualizados: `scripts/check-build.sh` varre os
  **1.312 arquivos versionados** com `git grep` (path contíguo, o fragmento que o rename costurou e variantes) e compara os
  **8 arquivos de `api/`** por sha256 antes/depois de `make proto`. É chamado por `make gate` — no
  job `gate` do `.github/workflows/ci.yml` e em `make ci` (`ci: gate vet build test`) — e cai em
  `golang:1.26-alpine` quando o host não tem `protoc`, então o comando é o mesmo no dev e no CI.
  **Provado nos dois sentidos (05/11):**

  ```
  # vermelho 1 — violação real introduzida em docs/game.md e revertida
  FAIL: residuo do rename em arquivo versionado:
  docs/game.md:132:<!-- prova de gate: github.com/jeanluca/w2pp-` + `openwyd -->
  GATE: FALHOU                        RC=1
  # verde 1
  OK: nenhum residuo do path antigo em 1312 arquivos versionados
  GATE: OK                            RC=0

  # vermelho 2 — `// prova de gate` acrescentado ao .pb.go, revertido pelo próprio gate
  FAIL: artefatos proto desatualizados — make proto alterou:
    alterado: api/bin/v1/bin.pb.go
  (artefatos restaurados; rode 'make proto' e commite o resultado)
  GATE: FALHOU                        RC=1
  # verde 2
  OK: make proto idempotente (8 arquivos em api/, nenhum byte mudou)
  GATE: OK                            RC=0
  ```

## Riscos/rollback

Risco: regenerar protos com versão de `protoc` diferente mudar todo o `.pb.go` (diff enorme e ruído de
revisão). Mitigação: gerar com a mesma versão documentada e commitar separado do resto. Rollback: o
rename é reversível com a varredura inversa; os gerados voltam pelo git.
