# Issue 03-4 — Fim dos TXT/CSV: validação de schema no boot

**Fase:** 3 · **Depende de:** FASE-04 · **Bloqueia:** 03-5 · **Esforço:** M

## Objetivo

Substituir o carregamento de TXT/CSV arcaicos por tabelas validadas, com o
servidor abortando no boot (erro claro) ante dado estático inconsistente.

## Contexto / evidência

`pre-release-rust.md` FASE 3 §3.2: substituir texto plano por tabelas/banco;
validação de schema na inicialização com falha explícita, nunca default
silencioso.

## Passos

1. Inventariar os arquivos planos consumidos pelo legado (somente leitura em
   `docs/arquitetura/modelo-banco.md`).
2. Definir o formato destino (tabelas versionadas ou banco — ver FASE-04) e o
   validador de schema.
3. Implementar a carga validada no boot com relatório de erro por
   arquivo/linha/campo.
4. Teste: dado corrompido aborta com mensagem que identifica o problema.

## Arquivos

- Criados: validador de schema + testes (módulo em `wyd-core` ou `wyd-db`,
  conforme decisão).
- Alterados: docs do formato em `docs/arquitetura/`.
- Proibido tocar: conteúdo de jogo sem motivo, default
  silencioso (proibido por regra).

## Testes / verificação

```bash
cargo test -p wyd-core 2>&1 | tail -n 3
make check
```

Saída esperada: `test result: ok` incluindo teste de dado corrompido (boot
abortado com relatório).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core - carga no boot.
- .agents/skills/wyd-rust-db - tabelas como fonte (se for o destino).
- .agents/skills/rust-code-quality - erros tipados, sem default silencioso.

## Documentação de referencia

- pre-release-rust.md (FASE 3 §3.2).
- docs/rust/05-stack-docker.md - arquivos planos legados.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Zero TXT/CSV lido sem validação; inconsistência aborta com relatório claro.
- [ ] Teste de corrupção verde; evidência anexada.

## Riscos / rollback

- Risco: formato destino incompatível com ferramenta de conteúdo → mitigação:
  conversor versionado + validador antes/depois.
- Rollback: `git checkout -- crates/ docs/`.
