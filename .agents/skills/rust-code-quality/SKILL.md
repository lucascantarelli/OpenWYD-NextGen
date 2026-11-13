---
name: rust-code-quality
description: Garante qualidade de código Rust no workspace WYD (ownership/borrow, erros tipados, sem unwrap/panic no caminho quente, clippy/rustfmt limpos, testes por crate). Use ao escrever ou revisar qualquer código em crates/, antes de considerar uma tarefa pronta.
---

# Qualidade de código Rust

Gate e invariantes gerais: `AGENTS.md` §2.8. Estrutura: `docs/arquitetura/estrutura-e-padroes.md`
§6/§8; testes: `docs/arquitetura/testes.md` §4/§8; dependências:
`docs/rust/01-dependencias.md`. Rode o gate, não repita a lista dele.

## O que a revisão precisa pegar

- **Ownership:** prefira `&`/`&mut`; `clone()` só com justificativa (fronteira de thread,
  canal, API que exige posse). Dados do tick não alocam no caminho quente.
- **Erro:** `Result`/`Option` tipado; `unwrap`/`expect` só em teste ou invariante
  demonstrada, com comentário. Entrada de rede/arquivo/banco não causa `panic!`.
- **`unsafe`:** só FFI/interop ou otimização medida, em módulo pequeno, documentado e
  revisado. “Funciona” não é justificativa.
- **Fronteira:** arquivo novo precisa estar na árvore §10–§17 e na issue; helper genérico
  para evitar passar contexto é violação estrutural.
- **Dependência:** nova crate exige justificativa/PoC, lockfile e atualização de
  `docs/rust/01-dependencias.md` quando a decisão for estrutural.

## Checklist

- [ ] `fmt`/`clippy -D warnings`/build/testes verdes no gate.
- [ ] Sem `unwrap`/`expect` fora de teste; sem `println!` como telemetria.
- [ ] Limites de entrada (tamanho, índice, range) têm erro tipado e teste negativo.
- [ ] Nenhum arquivo/dependência fora da árvore e da issue.
