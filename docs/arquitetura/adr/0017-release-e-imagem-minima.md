# ADR-0017 — Release otimizado e imagem mínima (LTO thin + distroless)

- **Estado:** aceita (2026-11-08)
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (fechamento da QA-03)
- **Governa:** `Cargo.toml` (`[profile.release]`), `docker/Dockerfile.launcher`, QA-03
- **Depende de:** ADR-0007 (launcher Tauri/CLI), ADR-0016 (bench da IA que serve de linha de base)

## Contexto

`pre-release-rust.md` FASE 5 §5.2 pede "compilações otimizadas para release (`cargo build
--release` com LTO habilitado)" e "containers Docker minimalistas (multi-stage builds baseados
em imagens *distroless* ou Alpine)". O workspace não tinha `[profile.release]` — o release era
o default do Cargo (16 codegen-units, sem LTO, símbolos e `panic = "unwind"`), e o
`docker/Dockerfile.launcher` usava `debian:bookworm-slim` como runtime, que carrega apt/dpkg e
um shell que o processo não precisa.

O risco declarado na issue é LTO estourar tempo/memória de build e valer a pena "segmentar
(`lto = "thin"` documentado em ADR)" — este documento é esse registro.

## Decisão

1. **`[profile.release]` do workspace**: `lto = "thin"`, `codegen-units = 1`,
   `strip = "symbols"`, `panic = "abort"`. Vale para todos os alvos do workspace (hoje um
   binário: `wyd-launcher`), é medido em `docs/rust/02-execucao.md` e não muda contrato de
   crate.
2. **Sem bloco `[profile.bench]`**: o Cargo ignora `panic` no perfil de bench (avisa
   `panic setting is ignored for bench profile`), então o bench herda release nas otimizações e
   continua com `unwind` por definição da ferramenta — declarar o bloco só gerava warning no
   manifesto.
3. **Runtime distroless**: `gcr.io/distroless/cc-debian12:nonroot` — glibc/libgcc para o
   binário gnu, **sem shell, sem gerenciador de pacotes e sem toolchain**, usuário
   `nonroot:nonroot` (uid 65532). O binário continua dinâmico; `cc-debian12` existe exatamente
   para esse caso.
4. **Proveniência por build-arg**: `WYD_VERSION` e `WYD_COMMIT` entram como labels OCI
   (`org.opencontainers.image.version`/`.revision`), com default `0.1.0`/`unknown` para build
   local sem git.
5. **Smoke real**, não só build: `docker run --rm ... check /jogo /jogo/manifest.toml` com um
   manifesto válido gerado em `target/smoke` (fora do git). `--help` do launcher **sai com 2**
   por contrato próprio (uso), então o aceite usa um caminho de sucesso de verdade.

## Consequências

- Tamanho e tempo de build medidos antes/depois ficam em `docs/rust/02-execucao.md`; a issue
  QA-03 carrega a tabela. Imagem final reportada por `docker image inspect`.
- `strip`/LTO removem símbolos: backtrace de release não mostra nomes — depuração usa build
  dev/`bench`, como já era.
- `panic = "abort"` em release significa que um panic **derruba o processo** (sem unwind): é o
  comportamento desejado para o binário e não afeta `cargo test` (perfil dev).
- O portal mantém `node:22-alpine` + `standalone` (já era multi-stage e non-root); a
  validação nesta entrega é de sintaxe (`docker build --check`), porque o build completo exige
  `npm ci` com rede e não é o artefato desta issue.

## Alternativas descartadas

| Alternativa | Por que não |
|---|---|
| `lto = "fat"` | ganho marginal sobre thin em um binário pequeno, build bem mais caro no CI; thin é o equilíbrio registrado |
| `panic = "unwind"` em release | mantém tabelas de unwind num binário que não as usa; abort reduz tamanho e superfície |
| `gcr.io/distroless/static` | o binário é gnu/dinâmico; static exigiria alvo musl e mudaria o link |
| `alpine` como runtime | precisa de `libgcc`/`libc6-compat` extras e ainda carrega apk; distroless cc é menor e sem shell |
| Buildar o portal nesta issue | `npm ci` + `next build` já têm Dockerfile e entram no compose; o alvo da QA-03 é o binário Rust |

## UNVERIFIED / aberto

- Paridade de runner: o build de imagem foi verificado no host de desenvolvimento; o job de CI
  roda em `ubuntu-24.04`.
- `WYD_VERSION`/`WYD_COMMIT` ainda não vêm de tag/CI (default local); quando existir release
  automatizado (FASE-15), o pipeline passa os valores.
- Assinatura/attestation de imagem (SBOM/proveniência criptográfica) não faz parte desta ADR.
