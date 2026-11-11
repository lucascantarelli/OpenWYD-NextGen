---
name: rust-test-quality
description: Garante tipo, nome e higiene dos testes Rust do WYD (unidade/integração/contrato/golden/sistema), com gate explícito para Postgres/socket/processo, sem skip silencioso, sleep de sincronização ou monólito. Use ao criar, revisar ou reorganizar qualquer teste em crates/ e antes de fechar uma issue.
---

# Qualidade e organização dos testes

Norma canônica: **`docs/arquitetura/testes.md`** (taxonomia, matriz por crate, auditoria
caso a caso e dívida priorizada). Complementa `rust-code-quality` e
`docs/arquitetura/estrutura-e-padroes.md` §7. Mudança de taxonomia exige ADR.

## 1. Escolha o tipo antes de escrever

| O teste precisa de… | Tipo | Onde / prefixo |
|---|---|---|
| nada além da CPU | unidade | `src/**` em `#[cfg(test)] mod tests` |
| só a API pública da crate | integração | `tests/integracao_<tema>.rs` |
| doc/schema/TS/assets do repo | contrato | `tests/contrato_<tema>.rs` |
| fixture ou seed versionada | golden | `tests/golden_<tema>.rs` |
| Postgres, socket, processo, FS | sistema | `tests/sistema_<tema>.rs` |

Regra dura: se o teste falha por falta de infraestrutura, ele **não** é unidade nem
integração — é sistema e precisa de gate explícito. Não misture tipos no mesmo arquivo;
ao passar de ~400 linhas ou dois tipos de infraestrutura, divida por tema.

## 2. Nome e local

- Um tema por arquivo, `snake_case`; prefixo identifica o tipo (o Cargo só descobre
  `tests/*.rs`, então subpasta exigiria `[[test]]` por arquivo).
- Teste em português, estado esperado + condição: `pin_errado_e_recusado_sem_sessao`.
- `tests/common/mod.rs` é o único arquivo sem `#[test]` em `tests/`; helper reutilizado
  mora lá, não copiado em cada arquivo.
- Teste de binário (`src/main.rs`) não conta como unidade: extraia a regra para a lib.

## 3. Gate de infraestrutura (obrigatório)

```rust
macro_rules! exigir_dsn {
    () => {
        match dsn_teste() {
            Some(dsn) => dsn,
            None => {
                eprintln!("SKIP: WYD_TEST_DSN/WYD_TEST_DATABASE_URL/DATABASE_URL ausente — <o que não rodou>");
                return;
            }
        }
    };
}
```

- Nunca `return;` mudo, nunca `#[ignore]`, nunca `panic` por env ausente.
- A mensagem nomeia **todas** as variáveis aceitas e o que deixou de ser provado.
- Um helper de DSN por crate em `tests/common/mod.rs`; helper de conexão devolve
  `Result`, não `.expect` antes do gate.

## 4. Higiene

- **Sem `sleep` fixo para sincronizar**: poll com prazo (`esperar_ate`) e falha
  diagnóstica. Exceção: prova viva que mantém sessão aberta de propósito
  (`WYD_LIVE_HOLD_SECS`), documentada.
- **Isolamento**: dados únicos por execução, limpeza no fim, porta efêmera, escrita só
  em temp ou sob flag (`WYD_CAPTURE_DIR`).
- **Asserção concreta**: valor/erro/byte, caso negativo e determinismo (seed/relógio
  injetados). Não enfraqueça asserção para o CI passar.
- **Golden**: um byte/valor de diferença falha; cite a fonte (fixture, seed `00xx`,
  captura datada). Nunca ajuste golden sem motivo no commit.
- Unitário não abre socket, banco, processo nem arquivo do repo.

## 5. Comandos

```bash
cargo fmt --all --check
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace                 # unidade + integração + contrato + golden
cargo test -p wyd-server -j 1          # no Windows evita LNK1102
cargo test -p wyd-db --test sistema_pg # opt-in com WYD_TEST_DSN
```

Teste de sistema sem a env **passa com `SKIP`**; registre na evidência que ele não rodou.
Nunca reporte um `SKIP` como prova de comportamento.

## 6. Checklist de revisão

- [ ] Tipo correto no nome e nenhuma infraestrutura escondida em unidade/integração.
- [ ] Gate nomeia variáveis e o que não rodou; sem `#[ignore]`/`return` mudo.
- [ ] Sem `sleep` de sincronização; timeout finito.
- [ ] Dados isolados, limpeza e porta efêmera; sem escrita no repo por padrão.
- [ ] Asserção concreta + caso negativo + determinismo.
- [ ] Arquivo ≤~400 linhas, um tema, helper no `common`.
- [ ] Makefile/CI/docs atualizados no mesmo commit; teste afetado executado.
