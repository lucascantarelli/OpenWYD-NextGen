# PROC-02 — Limpeza do ambiente Go (Windows → Rust)

> **Issue:** `PROC-02` · **Data:** 2026-11-12 · **Ambiente:** Windows, Git Bash, Rust 1.99.1
> **Resultado:** Go removido do host; `where go` não resolve, caches e ferramentas Go
> removidos, `GOPATH`/`GOROOT` vazios e nenhuma extensão Go no VS Code.

## 1. Estado antes

```text
$ where go
D:\Programas\Go\bin\go.exe

$ go version
go version go1.27.3 windows/amd64

$ go env GOPATH GOROOT GOCACHE GOMODCACHE GOENV
C:\Users\Canta\go
D:\Programas\Go
C:\Users\Canta\AppData\Local\go-build
C:\Users\Canta\go\pkg\mod
C:\Users\Canta\AppData\Roaming\go\env

$ env | grep '^GO'
GOPATH=C:\Users\Canta\go
```

O pacote estava instalado pelo MSI e registrado no winget:

```text
$ winget list --id GoLang.Go --exact
Go Programming Language amd64 go1.27.3   GoLang.Go   1.27.3   1.27.4   winget

DisplayName     : Go Programming Language amd64 go1.27.3
UninstallString : MsiExec.exe /X{041AA161-03B7-40AE-AE7B-056B6A6DFB94}
```

`C:\Users\Canta\go` não tinha código-fonte Go (`src` ausente). Continha apenas ferramentas
instaladas pelo usuário: `dlv.exe`, `golangci-lint.exe`, `goplay.exe`, `gopls.exe`,
`impl.exe`, `protoc-gen-go.exe` e `protoc-gen-go-grpc.exe`, além de `pkg/sumdb`.
Não havia `go.mod`, `go.sum`, `go.work` ou `*.go` no repositório. Nenhum processo `go`,
`gopls` ou `dlv` estava em execução. O VS Code já não tinha extensão Go (`code
--list-extensions` sem correspondência e nenhuma pasta Go em `~/.vscode/extensions`).

## 2. Ações executadas

1. **Caches**: `go clean -cache -modcache -testcache` — exit 0; `go/pkg/mod` removido.
2. **Toolchain**: `winget uninstall --id GoLang.Go --exact --silent` com elevação. A
   ferramenta de elevação retornou erro ambíguo depois de iniciar o comando; a verificação
   seguinte mostrou que a desinstalação **foi concluída** (o pacote sumiu do winget/registro,
   `D:\Programas\Go` não existe e o `PATH` da máquina perdeu `D:\Programas\Go\bin`).
3. **Resíduos**: removidos `C:\Users\Canta\go`, `C:\Users\Canta\AppData\Local\go-build` e
   `C:\Users\Canta\AppData\Roaming\go`. O `go-build` tinha só `README`/`trim.txt` depois do
   `go clean`; o `GOPATH` tinha só `bin` e `pkg/sumdb`; nenhum fonte de usuário foi apagado.

## 3. Estado depois

```text
$ where go
INFO: não foi possível localizar arquivos para o(s) padrão(ões) especificado(s).
GO-REMOVIDO

$ go version
bash: go: command not found
GO-COMANDO-AUSENTE

$ cargo --version
cargo 1.99.1 (5f94df478 2026-08-27)

$ rustc --version
rustc 1.99.1 (58cf8f9c3 2026-10-13)

$ winget list --id GoLang.Go --exact
No installed package found matching input criteria.

$ code --list-extensions | grep -iE 'golang|^go'
NENHUMA-EXTENSAO-GO
```

Diretórios e ferramentas:

```text
REMOVIDO /d/Programas/Go
REMOVIDO C:\Users\Canta\go
REMOVIDO C:\Users\Canta\AppData\Local\go-build
REMOVIDO C:\Users\Canta\AppData\Roaming\go
REMOVIDO gopls
REMOVIDO dlv
REMOVIDO golangci-lint
REMOVIDO goplay
REMOVIDO impl
REMOVIDO protoc-gen-go
```

Ambiente persistido (User/Machine), lido do registro:

```text
USER_PATH_GO=NENHUMA-ENTRADA
MACHINE_PATH_GO=NENHUMA-ENTRADA
USER_GOPATH=[]
MACHINE_GOPATH=[]
USER_GOROOT=[]
MACHINE_GOROOT=[]
```

O processo de terminal já aberto pode manter uma cópia antiga do `PATH` até ser reiniciado;
o `PATH` persistido do usuário e da máquina, que é o que novos processos herdam, não tem
entrada Go. `where go` já falha neste processo porque o executável foi removido.

## 4. Critérios de aceite

- [x] Nenhum `go` no `PATH` persistido (User e Machine) e `where go` não resolve.
- [x] Caches Go removidos (`GOCACHE`/`go-build`, `GOMODCACHE` e `GOENV`).
- [x] Toolchain Go desinstalada pelo winget; `D:\Programas\Go` removido.
- [x] Ferramentas Go do `GOPATH/bin` removidas (`gopls`, `dlv`, `golangci-lint` etc.).
- [x] `GOPATH` e `GOROOT` vazios em User e Machine.
- [x] Nenhuma extensão Go no VS Code.
- [x] `cargo 1.99.1` e `rustc 1.99.1` respondem no host.
- [x] Evidência com as saídas antes/depois anexada neste arquivo.

## 5. Rollback

O Go não é usado pelo repositório (100% Rust). Se for necessário reinstalar:

```bash
winget install --id GoLang.Go --exact
```

As ferramentas de usuário (`gopls`, `dlv`, `golangci-lint` etc.) teriam de ser reinstaladas
manualmente; não havia código-fonte Go em `C:\Users\Canta\go` para preservar.
