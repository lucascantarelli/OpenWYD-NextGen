# CLEAN-05 — Definition of Done única e regra de comentário no processo

**Fase:** revisão · **Depende de:** — · **Esforço:** S · **Estado:** ⏳ proposta

## Objetivo

Uma página que responde “quando uma entrega está pronta” e “como se escreve issue/commit/comentário”,
citada pelo `AGENTS.md` — hoje a regra está espalhada em três documentos e não é checada.

## Contexto/evidência (05/11/2026)

- `AGENTS.md` §8 tem convenções de trabalho; `docs/analysis/qualidade-e-processo.md` §4 tem o processo;
  as issues REVIEW-* repetem critérios de aceite. Nenhum é o texto canônico.
- Consequência prática desta sessão: trabalho ficou no meio (rename pela metade, build vermelho,
  mudanças não verificadas) sem um checklist que barre “commit sem build verde”. O custo aparece
  depois, na forma de 382 arquivos inconsistentes.
- `roadmap/index.md` já exige evidência por issue, mas não há gate: issue ✅ pode não citar comando.

## Passos

1. Escrever `docs/development-guidelines/processo.md` com a DoD em checklist:
   `build + vet + teste verde`; comportamento provado com comando e saída; doc/skill atualizado;
   rollback descrito; uma issue = um resultado verificável; revisão < 400 linhas; comentário sem
   número de issue (REFACTOR-03); nada de segredo em arquivo versionado.
2. Definir o formato de commit (pt-BR, imperativo, um assunto) e o de issue (objetivo, evidência,
   passos, verificação, critérios, riscos) — copiando o formato que já funcionou nas issues 56/58/62.
3. Apontar `AGENTS.md` §8 para a página e remover a duplicação de regra nos outros dois documentos.
4. Checklist de PR em `.github/pull_request_template.md` com os itens da DoD.
5. Rodar uma auditoria única: para cada issue ✅ do índice, verificar se a evidência está no arquivo
   (comando + saída). O que não tiver, volta para 🟡.

## Arquivos

- `docs/development-guidelines/processo.md` (novo)
- `AGENTS.md` §8, `docs/analysis/qualidade-e-processo.md`
- `.github/pull_request_template.md` (novo)
- `roadmap/index.md` (auditoria de evidência)

## Testes/verificação

```bash
grep -c "build.*verde" docs/development-guidelines/processo.md
# auditoria: para cada issue ✅ do índice, o arquivo tem bloco de evidência com comando
```

## Critérios de aceite

- [ ] `processo.md` publicado e referenciado no `AGENTS.md`.
- [ ] Template de PR com a DoD.
- [ ] Nenhuma issue ✅ sem evidência de comando no próprio arquivo.
- [ ] Regra de comentário apontando para a REFACTOR-03, sem duplicação.

## Riscos/rollback

Risco: DoD virar burocracia ignorada. Mitigação: poucos itens, todos checáveis por comando, e o
template de PR só com o que reprova de verdade. Rollback: é documentação; reduzir itens é trivial.
