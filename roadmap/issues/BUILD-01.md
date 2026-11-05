# BUILD-01 — Renomeação do módulo (`openwyd-nextgen`): fechar a inconsistência

**Fase:** revisão · **Depende de:** — · **Esforço:** M · **Estado:** 🟡 em execução (05/11/2026)

## Objetivo

O repositório compilar e testar com o módulo `openwyd-nextgen`, sem resíduo do path antigo
(`github.com/jeanluca/w2pp-openwyd`) e sem que um rename volte a quebrar o build por semanas.

## Contexto/evidência (05/11/2026)

O rename foi feito à mão e ficou pela metade: `go.mod` já dizia `module openwyd-nextgen` enquanto
**382 arquivos** ainda importavam o path antigo — build impossível. Números medidos:

```bash
git grep -l "github.com/jeanluca/w2pp-openwyd" | wc -l   # 382 (antes)
git grep -l "github.com/jeanluca/w2pp-openwyd" | wc -l   # 0   (depois da varredura de 05/11)
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
5. **Gate**: no CI, falhar se `git grep "jeanluca/w2pp-openwyd"` achar algo e se `make proto` alterar
   arquivo versionado (`git diff --exit-code` após gerar). Sem isso, o próximo rename volta a quebrar
   o build em silêncio.
6. Atualizar `docs/migration/SESSION-PRIMER.md` (módulo), `CLAUDE.md` e qualquer comando com o path.

## Arquivos

- `go.mod`, `api/**/*.proto` + `*.pb.go`
- `binserver/cmd/binserver/main.go`, `binserver/internal/grpcsrv/server_test.go`
- `.github/workflows/ci.yml` (gate de path antigo + protos em dia)
- `Makefile` (`proto` com instalação dos plugins documentada/automatizada)

## Testes/verificação

```bash
git grep -n "jeanluca/w2pp-openwyd" || echo "path antigo: 0"
make proto && git diff --exit-code api/    # gerados em dia
go build ./... && go vet ./... && go test ./...
```

## Critérios de aceite

- [x] Zero ocorrência do path antigo no repositório.
- [x] Zero import com o path nu.
- [ ] `go build ./...` e `go vet ./...` verdes neste checkout.
- [ ] `go test ./...` verde (ou falhas explicadas com issue filha).
- [ ] Gate no CI para path antigo e para gerados desatualizados.

## Riscos/rollback

Risco: regenerar protos com versão de `protoc` diferente mudar todo o `.pb.go` (diff enorme e ruído de
revisão). Mitigação: gerar com a mesma versão documentada e commitar separado do resto. Rollback: o
rename é reversível com a varredura inversa; os gerados voltam pelo git.
