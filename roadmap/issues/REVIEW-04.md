# REVIEW-04 — CI/CD e DevOps: reuso, cache, banco no pipeline e imagens

> **Análise profunda 06/11/2026:** números medidos, correções das afirmações envelhecidas e
> referências externas em [`REVISAO-PROFUNDA-2026-11-06.md`](../REVISAO-PROFUNDA-2026-11-06.md) §4.

**Fase:** revisão · **Depende de:** REVIEW-03 · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

Transformar o `ci.yml` atual (bom ponto de partida) em pipeline modular e reprodutível: workflows
reutilizáveis, Postgres de serviço, cache efetivo, imagens com SBOM e um `make` que espelha o CI.

## Contexto/evidência (05/11/2026)

- Um único workflow (`.github/workflows/ci.yml`, 4.121 bytes) com 5 jobs: `test` (gofmt, vet, build,
  `go test -race -cover ./...`), `lint` (`only-new-issues`), `govulncheck` pinado em `v1.9.0`,
  `portal` (tsc/vitest/build + docker build com cache GHA) e `docker` (matrix
  `tmserver,dbserver,binserver,webserver`).
- Cada job repete `checkout` + `setup-go` com `cache: true` — sem workflow reutilizável.
- **Não há** Postgres de serviço, teste de migração em base vazia, cobertura com piso, e2e, SBOM,
  assinatura de imagem, path filters nem release.
- `Makefile` tem 21 alvos (`build`, `binaries`, `test`, `vet`, `lint`, `fmt`, `vuln`, `tidy`, `run`,
  `run-local`, `portal-*`, `proto`, `certs`, `exp-data`, `item-icons*`); `scripts/` tem 9 arquivos.
- Build local já quebrou por disco cheio (C: com 6,6 GB livres) e por build paralelo derrubando o
  daemon — o CI não detecta nenhum dos dois.

## Analisar tudo

- Devemos implementar pipelines de CI (Continuous Integration) para automatizar o processo de build e testes, para que possamos otimizar o desempenho e garantir maior segurança na plataforma.
- Devemos usar boas praticas de docker, docker compose, multistage build e outras técnicas de containerização para garantir que o sistema seja facilmente implantado e escalável, para que possamos otimizar o desempenho e garantir maior segurança na plataforma.
- Devemos usar boas praticas de CI e Github Actions para automatizar o processo de build, testes e deploy, para que possamos otimizar o desempenho e garantir maior segurança na plataforma.
- Devemos usar automações makefile, analise nosso atual e proponha ajustes/melhorias, para que possamos otimizar o desempenho e garantir maior segurança na plataforma.
- as actions do projeto devem seguir boas praticas de reuso, modularização, organização, padronização para que possamos otimizar o desempenho e garantir maior segurança na plataforma.
- dentro do CI devemos usar recursos atuais do github actions, como caching, paralelização, matrix builds, etc. para que possamos otimizar o desempenho e garantir maior segurança na plataforma.


## Passos

1. **Workflow reutilizável** `_go.yml` (`workflow_call`: setup-go com cache, gofmt/vet/build/test) e
   `_portal.yml`; `ci.yml` passa a só orquestrar.
2. **Path filters**: portal só roda quando `portal/**` ou `api/**` mudam; Go quando `**/*.go`,
   `go.mod`, `internal/migrations/**`.
3. **Job `db`**: `services: postgres:17`, aplica `internal/migrations` em base vazia com o runner Go,
   compara contagem por tabela (mesmo gate do `scripts/db-baseline.sh`, em modo não destrutivo).
4. **Cobertura**: `-coverprofile` + resumo no job; piso por pacote crítico (ver REVIEW-03).
5. **Imagens**: `provenance: true`, SBOM (`syft` ou buildx `--sbom`), tag por SHA, e cache de camada
   por serviço (já existe para portal; estender aos demais).
6. **Segurança**: `govulncheck` sem pin manual (ou pin documentado no `Makefile`), `npm audit
   --omit=dev`, e `gitleaks`/`git-secrets` no PR.
7. **Make ↔ CI**: um alvo `make ci` que roda exatamente o que o CI roda (gofmt, vet, test, portal),
   para o dev reproduzir local antes do push.

## Arquivos

- `.github/workflows/ci.yml`, `_go.yml`, `_portal.yml`, `db.yml`
- `Makefile` (alvo `ci`)
- `scripts/preflight.sh` (checagem de disco/Docker antes do build)

## Testes/verificação

```bash
make ci                                   # mesmo gate do CI, local
act -j test                               # opcional: simular workflow
docker build --build-arg SVC=tmserver -t tmserver:test .
```

## Critérios de aceite

- [ ] `ci.yml` sem repetição de setup (usa `workflow_call`).
- [ ] Job com Postgres aplica as migrations em base vazia e falha se contagem divergir.
- [ ] Cobertura publicada; piso configurado.
- [ ] Imagens com SBOM e tag por SHA.
- [ ] `make ci` verde em máquina limpa (sem Go no host, via container).

## Riscos/rollback

Risco: matriz + cache GHA estourar o limite de 10 GB e o cache parar de acertar. Mitigação: `scope`
por serviço (já usado) e `cache-to: mode=max` só no `main`. Rollback: cada job novo entra como
`continue-on-error` na primeira semana.

## Issues que este review abre

- `CI-01` workflows reutilizáveis + path filters
- `CI-02` job de banco (migração em base vazia)
- `CI-03` cobertura com piso e comentário no PR
- `CI-04` SBOM + proveniência + tag por SHA
- `CI-05` `make ci` e preflight de disco/Docker
