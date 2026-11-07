---
name: rust-code-quality
description: Garante qualidade de código Rust no workspace OpenWYD (ownership/borrow, erros tipados, sem unwrap/panic no caminho quente, clippy/rustfmt limpos, testes por crate). Use ao escrever ou revisar qualquer código em crates/, antes de considerar uma tarefa pronta.
---

# Qualidade de código Rust

Vale para todas as `crates/wyd-*`. Referências: `rust-toolchain.toml`,
`.github/workflows/rust.yml`, `roadmap-rust/issues/FASE-02.md`.

## Regras inegociáveis

1. **Ownership primeiro.** Prefira empréstimo (`&`/`&mut`) a `clone()`; `clone`
   só com justificativa (ex.: fronteira de thread com canal). Dados do tick loop
   nunca alocam no caminho quente — reusar buffers/arena.
2. **Erros tipados, nunca pânico em produção.** `Result`/`Option` com
   `thiserror`/`anyhow`; `unwrap`/`expect` só em testes ou invariantes
   demonstradas (comente o porquê). Input de rede, arquivo ou banco jamais causa `panic!`.
3. **Sem `unsafe` por conveniência.** `unsafe` só em interop/FFI ou otimização
   comprovada por benchmark, isolado em módulo pequeno, documentado e revisado.
4. **Avisos são erros.** `cargo clippy --workspace -- -D warnings` e
   `cargo fmt --check` limpos antes de qualquer validação.
5. **Teste onde nasce o comportamento.** Toda regra nova (fórmula, parser,
   transição de estado) nasce com teste unitário na crate; golden tests para
   byte-exato de protocolo e regressões matemáticas (`wyd-sim`).
6. **Estrutura antes de escrever.** Antes de criar arquivo, confira a árvore de
   `docs/arquitetura/estrutura-e-padroes.md` (ADR-0002) e a issue: arquivo fora
   do domínio planejado não é aceito, mesmo que "funcione".

## Checklist de revisão

- [ ] Compila sem warnings; clippy e fmt limpos.
- [ ] Sem `unwrap`/`expect` fora de testes; sem `println!` (usar `tracing`).
- [ ] Fronteiras validadas (tamanhos, índices, ranges) com erro tipado.
- [ ] Teste cobre o caminho feliz, o limite e o erro.
- [ ] Sem dependência nova sem justificativa e lockfile consistente
      (ver `docs/rust/01-dependencias.md`).
- [ ] Arquivos criados/alterados são exatamente os listados na árvore planejada e
      na issue; gerados isolados em `generated/`.
