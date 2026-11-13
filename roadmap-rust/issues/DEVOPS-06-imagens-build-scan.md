# DEVOPS-06 — As quatro imagens no CI: build, proveniência e scan

> **ID canônico:** `DEVOPS-06`
> **ID histórico:** `DEVOPS-06`
> **Estado 2026-11-13:** 🚧 · ver `roadmap-rust/index.md`.

**Tipo:** DevOps · **Esforço:** M · **Depende de:** DEVOPS-01, DEVOPS-05 · **Bloqueia:** DEVOPS-10

## Objetivo

Fazer o CI construir **todas** as imagens que existem, com versão/commit reais, SBOM e scan de
vulnerabilidade falhando em `CRITICAL`, de modo que nenhum Dockerfile apodreça sem sinal e
nenhuma imagem de release saia sem procedência.

## Contexto / evidência

- Job `release` constrói apenas `docker/Dockerfile.launcher`; `Dockerfile.gateway` e
  `Dockerfile.server` **nunca** são construídos no CI. `Dockerfile.portal` só passa por
  `docker build --check` (lint do Dockerfile, não build).
- O `docker build` do CI não passa `--build-arg WYD_VERSION/WYD_COMMIT`: labels OCI saem
  `0.1.0`/`unknown`, ou seja, a proveniência da imagem de CI não corresponde ao commit.
- Não há Trivy/Clair/scout, nem SBOM (`syft`/`--sbom`), em nenhum lugar.
- **Prova de que o gap dói (DEVOPS-01, 2026-11-12):** `docker build -f docker/Dockerfile.launcher`
  estava **quebrado** porque `identidade.rs` lê `docs/arquitetura/identidade.tokens` por
  `include_str!` e nenhum `COPY` trazia o arquivo. Corrigido na DEVOPS-01; o job de CI que
  constrói essa imagem é o único que poderia ter avisado antes.
- Nenhuma imagem é publicada em registry: não há tag imutável por digest para rollback.
- Análise completa em `docs/rust/10-devops-ci.md` §5 (achados B2, B5, B6).

## Progresso (2026-11-13)

- `.github/workflows/images.yml`: matriz das **quatro** imagens (launcher, gateway, server, portal)
  em todo push/PR relevante, com `WYD_VERSION` (tag `v*` ou `0.0.0-<sha>`) e `WYD_COMMIT` (sha
  completo) do run, conferência das labels de proveniência contra esses valores, smoke do artefato,
  Trivy bloqueante em `CRITICAL` com `.trivyignore` versionado (hoje vazio, com regra de prazo) e
  SBOM CycloneDX por imagem como artefato.
- Ainda falta: o primeiro run (nada executou no GitHub), publicar **tamanho** por imagem (o smoke
  imprime, mas não há artefato/série) e a decisão de registry/promoção por digest.

## Passos

1. Job `images` (ou matriz por imagem) que constrói `launcher`, `gateway`, `server` e `portal`
   com `WYD_VERSION` derivado (`vX.Y.Z` da tag, ou `0.0.0-${{ github.sha }}`) e
   `WYD_COMMIT=${{ github.sha }}`, usando `cache-from/cache-to: type=gha`.
2. Smoke por imagem: `launcher --help`/`check`, `gateway --healthcheck` (ou `--help` sem banco),
   `server --help`; portal com `node -e` no `/api/health` não é possível sem backend → smoke é
   `docker run --rm <img> node -e` que apenas valide o entrypoint, e a prova real de portal fica
   no `DEVOPS-07` (build + vitest).
3. Scan: Trivy (`aquasecurity/trivy-action` ou imagem fixa) nas quatro imagens, `exit-code: 1`
   em `CRITICAL`, com `.trivyignore` versionado e cada exceção datada + justificada.
4. SBOM: `--sbom` (BuildKit) ou `syft` por imagem, anexado como artefato do run.
5. Publicação opcional (decisão do dono): push para GHCR com tag semver + `@sha256` registrada
   em `docs/rust/02-execucao.md` §6. Sem registry, a issue ainda entrega build + scan + SBOM.
6. Publicar tamanho de cada imagem no log e comparar com o baseline de `02-execucao.md` §6
   (launcher 37.847.971 B) — regressão grosseira de tamanho fica visível.

## Arquivos

- Alterados: `.github/workflows/rust.yml` (jobs `images`/`release`), `docker/Dockerfile.portal`
  (labels, ver DEVOPS-01), `.trivyignore` (novo, se necessário),
  `docs/rust/10-devops-ci.md` §5, `docs/rust/02-execucao.md` §6.
- Proibido: publicar imagem com segredo, source map ou `.env`; `latest` sem TAG.

## Testes / verificação

```bash
docker build --build-arg WYD_VERSION=0.1.0 --build-arg WYD_COMMIT=$(git rev-parse --short HEAD) \
  -f docker/Dockerfile.gateway -t wyd/gateway:test .
docker run --rm wyd/gateway:test --help
docker image inspect --format '{{json .Config.Labels}}' wyd/gateway:test
trivy image --exit-code 1 --severity CRITICAL wyd/gateway:test
```

Saída esperada: quatro imagens construídas no CI; labels com versão/commit do run; scan exit 0
ou exceção registrada; SBOM como artefato.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain — imagens, release, supply-chain.
- .agents/skills/software-architecture — fronteira de artefato.

## Documentação de referencia

- `docs/rust/10-devops-ci.md` §5.
- `docs/rust/02-execucao.md` §6.
- `docs/arquitetura/adr/0017-release-e-imagem-minima.md`.

## Critérios de aceite

- [ ] Quatro imagens construídas em todo push/PR relevante, com smoke.
- [ ] Labels de versão/commit iguais ao run (nada de `0.1.0`/`unknown` fixos).
- [ ] Scan de vulnerabilidade bloqueante em `CRITICAL` com exceções versionadas.
- [ ] SBOM anexado e tamanho por imagem publicado.

## Riscos / rollback

- Risco: scan bloquear release por CVE em base distroless sem correção disponível → mitigação:
  `.trivyignore` com justificativa + prazo, revisão na entrega seguinte.
- Rollback: remover o job novo; o job `release` atual continua válido.
