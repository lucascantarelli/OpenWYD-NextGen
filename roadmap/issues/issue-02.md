# Issue 02 — Provisionar ambiente de execução (Go, Docker, portas)

**Fase:** 0 · **Depende de:** 01 · **Bloqueia:** 03, 04, 06, 07, 16 · **Esforço:** S · **Estado:** 🟡 parcial (2026-11-02) — falta `scripts/preflight.sh`

## Objetivo

Ter um ambiente capaz de **buildar e rodar** a stack nesta máquina Windows, com decisão registrada
de como compilar/testar sem Go instalado no host e com portas verificadas.

## Contexto / evidência

```
$ go version        → bash: go: command not found
$ docker --version  → Docker version 29.9.2
$ docker compose version → v5.6.1
$ docker images     → postgres:16 (a stack pede postgres:17-alpine)
$ netstat :80/:8281 → livres
```

A primeira tentativa de subir a stack falhou durante o build:

```
#18 [binserver build 6/7] COPY . .
target dbserver: failed to receive status: rpc error: code = Unavailable
desc = error reading from server: EOF
```

Isso aconteceu após ~54 MB de "transferring context" para 4 builds simultâneos. **Hipótese**:
pressão de memória/CPU no Docker Desktop (WSL2), não erro de Dockerfile.

## Passos

1. Confirmar que o daemon está saudável e com folga: `docker info | grep -E "CPUs|Total Memory"`,
   `docker system df`. Liberar espaço se necessário (`docker builder prune`).
2. Garantir que o Docker Desktop tem memória suficiente (recomendado ≥ 6 GB para 5 serviços + build
   Go) e que o WSL2 está atualizado.
3. Extrair o build do compose: buildar **um serviço por vez** para isolar falha e reduzir pressão:
   ```bash
   docker compose build --pull db
   docker compose build dbserver
   docker compose build binserver
   docker compose build webserver
   docker compose build tmserver
   ```
4. Se o `COPY . .` continuar derrubando o daemon, reduzir o contexto de build:
   revisar `.dockerignore` (hoje já exclui binários legados, `Conversor.rar`, etc.) para excluir
   `Source/`, `docs/`, `openspec/`, `project-analizer/`, `images/`, `.git/` — o build Go só precisa
   de `go.mod`, `go.sum`, código e `Release/` (que é copiado no estágio final).
5. Definir o caminho oficial para Go sem host:
   ```bash
   docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine go build ./...
   ```
   (Opcional: instalar Go 1.26 no host para `make test` direto — decisão do time, não requisito.)
6. Criar `scripts/preflight.sh` com as checagens que já custaram tempo: sparse checkout,
   `Release/` completo, portas 80/8281, daemon Docker, e presença de Go (host ou container).

## Arquivos

- Criado: `scripts/preflight.sh` (executável, `set -euo pipefail`).
- Alterado (se necessário): `.dockerignore` — apenas exclusões de contexto, nunca `Release/`.
- Proibido tocar: `go.mod`/`go.sum` (versão 1.26 é requisito do projeto), `Dockerfile` (a menos que
  o build falhe por causa dele, o que deve ser documentado antes).

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
./scripts/preflight.sh                 # esperado: tudo OK, zero [FAIL]
docker compose build tmserver          # esperado: exit 0, imagem w2pp/tmserver gerada
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine go build ./...
```

Saída esperada do build: `naming to docker.io/library/openwyd-tmserver` (ou similar) e `exit 0`.

## Critérios de aceite

- [ ] `./scripts/preflight.sh` roda e falha de forma legível quando algo essencial falta
- [ ] `docker compose build <serviço>` conclui para os 4 serviços, mesmo que serializado
- [ ] `go build ./...` roda em container `golang:1.26-alpine` sem erro
- [ ] Portas 80 e 8281 livres (ou conflito documentado com plano de porta alternativa)
- [ ] Causa do `rpc error ... EOF` identificada e registrada (recurso × Dockerfile × contexto)

## Riscos / rollback

- **Risco:** mexer no `.dockerignore` e excluir `Release/` → imagem sobe sem conteúdo. **Mitigação:**
  rodar issue 04 (boot) imediatamente após qualquer mudança de contexto de build.
- **Risco:** instalar Go no host e divergir da versão do projeto. **Mitigação:** fixar 1.26.x e
  conferir contra `Dockerfile` (`golang:1.26-alpine`).
- **Rollback:** reverter `.dockerignore` com `git checkout -- .dockerignore`; containers podem ser
  removidos com `docker compose down` (nunca `-v`).
