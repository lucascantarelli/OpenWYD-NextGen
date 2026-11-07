# Execução do roadmap Rust — ciclo, gates e documentação viva

> Fonte: `pre-release-rust.md` FASE 2 (§16 ordem estratégica) + `start-rust.md`
> (branch `develop`, planejamento antes de código). Este documento é **normativo**:
> toda issue de `roadmap-rust/issues/` é executada neste ciclo.

## 1. Ciclo obrigatório por issue (uma por vez)

1. **Análise** — ler a issue inteira, as docs de referência citadas e o código
   afetado. Nada de implementar por suposição.
2. **Design** — decidir a forma da mudança em uma frase; se for estrutural
   (fronteira de crate, dependência, formato, concorrência), abrir ADR em
   `docs/arquitetura/adr/` **antes** de codar (FASE-13).
3. **Código** — menor mudança que cumpre o objetivo, nos arquivos listados na
   issue. Fora da lista = parar e registrar por quê.
4. **Teste** — o comando de verificação da própria issue, mais o gate completo.
5. **Docs** — atualizar `docs/rust/03-progresso.md` (status + evidência), o
   status em `roadmap-rust/index.md` e qualquer doc que a mudança invalidou.
6. **Fechamento** — commit em português, branch mergeada em `develop`, issue
   marcada com a evidência anexada (log/saída).

Gate único, executado ao fim de **toda** issue:

```bash
make check      # fmt --check + clippy -D warnings + build + test
```

Sem `make` no host Windows (Git Bash), os quatro comandos `cargo` equivalentes
valem como gate (registrar qual foi usado na evidência).

## 2. Fluxo de branch

- `develop` é a branch principal de desenvolvimento (`start-rust.md`).
- Uma issue = uma branch: `fase/NN-slug` (ex.: `fase/03-1-tick`), criada de
  `develop`, mergeada de volta com `--no-ff`.
- Commits pequenos, descritivos, em português, um por passo relevante.
- Proibido: force-push em `develop`/`main`, commit de segredo, `client/`,
  `target/` ou `.deprecated-go/`.

## 3. Definition of Done por tipo de issue

| Tipo | Pronto quando |
|---|---|
| Código | gate verde + teste da issue + doc de progresso atualizado |
| Documento | arquivo criado/atualizado + links resolvendo + revisão fria |
| Processo | regra escrita em `docs/rust/02-execucao.md` + referenciada no aceite das issues seguintes |
| Auditoria | 100% dos itens classificados, com evidência ou marcação **hipótese** |
| ADR | status explícito (proposta/aceita/substituída) + issues consumidoras vinculadas |

## 4. Revisão consolidada (a cada 10 issues concluídas)

Checklist fixo, registrado em `docs/rust/03-progresso.md`:

1. `make check` + `cargo test --all-targets`.
2. `cargo bench` comparado ao baseline anterior (sem baseline = criar agora).
3. Testes de integração entre módulos tocados nas 10 issues.
4. ADRs pendentes/propostas que precisam ser decididas.
5. Docs vivos: `03-progresso.md`, `roadmap-rust/index.md`, `docs/arquitetura/`.
6. Débito aberto pelas 10 issues (TODO registrado, nunca silencioso).

## 5. Documentação viva (regra permanente)

- **`docs/rust/03-progresso.md`** é o painel de execução: muda a cada issue.
- **`roadmap-rust/index.md`** carrega o status de cada issue.
- **`AGENTS.md` §0** é a fonte única de invariantes; se uma execução mudar um
  invariante, o ADR e o `AGENTS.md` mudam na mesma entrega.
- Documento que descreve comportamento deve ser atualizado **na entrega que
  muda o comportamento**, nunca "depois". Doc desatualizada é defeito.
- Toda afirmação técnica não confirmada leva a marca **hipótese** (ou
  UNVERIFIED, no caso de protocolo), como em `.deprecated-go/docs/migration/`.
