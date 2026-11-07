# Issue 05-2 — Release LTO + imagem mínima + proveniência

**Fase:** 5 · **Depende de:** 05-1 · **Bloqueia:** FASE-15 · **Esforço:** S

## Objetivo

Release otimizado (LTO) empacotado em imagem mínima multi-stage, com build
reproduzível e smoke do binário.

## Contexto / evidência

`pre-release-rust.md` FASE 5 §5.2: `cargo build --release` com LTO, containers
distroless/Alpine multi-stage. Base: `docker/Dockerfile.rust`, `[profile.release]`
a configurar no workspace.

## Passos

1. Configurar perfil release com LTO no `Cargo.toml` do workspace.
2. Ajustar `docker/Dockerfile.rust` (build + runtime mínima, `wyd-launcher`
   como entrypoint).
3. Validar: `cargo build --release --workspace` + `docker build -f
   docker/Dockerfile.rust .` + smoke do binário.
4. Documentar proveniência (versão, commit, toolchain no artefato).

## Arquivos

- Criados: nenhum novo (configuração).
- Alterados: `Cargo.toml` (perfil release), `docker/Dockerfile.rust`.
- Proibido tocar: segredos na imagem, código das crates.

## Testes / verificação

```bash
cargo build --release --workspace 2>&1 | tail -n 3
docker build -f docker/Dockerfile.rust -t openwyd-rust:check .
```

Saída esperada: release compila sem warnings; imagem builda
(`Successfully tagged openwyd-rust:check`).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - release e proveniência.
- .agents/skills/github-actions-docs - job de release (futuro).

## Documentação de referencia

- pre-release-rust.md (FASE 5 §5.2).
- docker/Dockerfile.rust - Dockerfile vigente.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] LTO ativo no release; imagem mínima builda e roda `wyd-launcher`.
- [ ] Proveniência documentada (versão/commit/toolchain).

## Riscos / rollback

- Risco: LTO estourar tempo/memória do CI → mitigação: medir e, se preciso,
  segmentar (`lto = "thin"` documentado em ADR).
- Rollback: `git checkout -- Cargo.toml docker/Dockerfile.rust`.
