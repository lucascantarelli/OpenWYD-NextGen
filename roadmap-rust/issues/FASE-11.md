# Issue 11 — Documentação e padronização

**Fase:** 11 · **Depende de:** FASE-01 · **Bloqueia:** — (contínua) · **Esforço:** M

## Objetivo

Preencher `docs/manual-usuario/`, `docs/manual-técnico/` e `docs/arquitetura/`
(C4 + contratos) com guias que permitam a um novato compilar, testar e contribuir.

## Contexto / evidência

Estrutura criada (pastas com `.gitkeep`); conteúdo pendente. Padrão vigente:
MkDocs + C4 + specs de pacotes (`pre-release-rust.md` FASE 11). Manual do usuário
cobre wyd-portal/launcher/manuais do jogo; manual técnico cobre arquitetura,
implementação, padrões, testes e CI/CD; arquitetura cobre componentes, módulos,
diagramas, fluxos, protocolos e ADRs (ver `README.md` e `docs/arquitetura/adr/`).

## Passos

1. Escrever o guia de contribuição (pré-requisitos, `make check`, fluxo de issue→PR).
2. Gerar diagramas C4 (níveis 1–3) a partir da estrutura real (`crates/`, portal,
   banco) — sem inventar componentes.
3. Documentar contratos de rede vigentes (o que FASE-08 já cristalizou).
4. Revisar PT-BR e links (todo `docs/*` citado no README deve existir).
5. `make check`.

## Arquivos

- Criados: `docs/manual-usuario/*.md`, `docs/manual-técnico/*.md`,
  `docs/arquitetura/*.md` (diagramas como código-fonte + render).
- Alterados: `README.md` (tabela de docs, se necessário), `roadmap-rust/issues/FASE-11.md`.
- Proibido tocar: `crates/*/src/` (referência, não cópia),
  `client/` e seus binários.

## Testes / verificação

```bash
make check
for p in $(grep -o 'docs/[^)]*\.md' README.md | sort -u); do test -e "$p" || echo "QUEBRADO $p"; done; echo LINKS-OK
```

Saída esperada: gates verdes e `LINKS-OK` (nenhum link `docs/*` quebrado no README).

## Skills Relacionadas do freebuff

- .agents/skills/documentation-writer - estrutura Diátaxis e redação.
- .agents/skills/software-architecture - C4 e fronteiras reais.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- docs/rust/00-visao-estrutura.md - visão da transição.
- docs/arquitetura/adr/ - ADRs como fonte de decisões.
- pre-release-rust.md (FASE 11) - escopo de docs.
- docs/ - fonte histórica (citar, não duplicar).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Novato compila/testa só seguindo o guia (validado por leitura fria).
- [ ] C4 reflete o código real; zero componente inventado.
- [ ] Links `docs/*` do README todos resolvendo (script acima limpo).

## Riscos / rollback

- Risco: documentar intenção em vez do código real → mitigação: gerar diagramas
  a partir de `Cargo.toml`/imports, revisão cruzada com FASE-01.
- Rollback: `git checkout -- docs/ README.md`.
