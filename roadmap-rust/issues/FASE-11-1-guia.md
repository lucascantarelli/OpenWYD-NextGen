# Issue 11-1 — Guia de contribuição + manuais (usuário e técnico)

**Fase:** 11 · **Depende de:** 02-1 · **Bloqueia:** — · **Esforço:** M

## Objetivo

Publicar o guia que permite a um novato compilar, testar e contribuir, mais o
esqueleto dos manuais de usuário e técnico.

## Contexto / evidência

`pre-release-rust.md` FASE 11: MkDocs + guias de contribuição. Pastas criadas:
`docs/manual-usuario/`, `docs/manual-técnico/` (com `.gitkeep`).

## Passos

1. Escrever o guia (pré-requisitos, `make check`, fluxo issue→branch→PR).
2. Esboçar `manual-usuario/` (portal, launcher, manuais do jogo) e
   `manual-técnico/` (arquitetura, implementação, padrões, testes, CI/CD).
3. Validar por leitura fria: seguir o guia do zero compila e testa?
4. `make check`.

## Arquivos

- Criados: `docs/manual-usuario/*.md`, `docs/manual-técnico/*.md`.
- Alterados: `README.md` (tabela de docs, se necessário).
- Proibido tocar: `crates/*/src/`, `.deprecated-go/` (citar, não duplicar).

## Testes / verificação

```bash
make check
for p in $(grep -o 'docs/[^)]*\.md' README.md | sort -u); do test -e "$p" || echo "QUEBRADO $p"; done; echo LINKS-OK
```

Saída esperada: gates verdes + `LINKS-OK`.

## Skills Relacionadas do freebuff

- .agents/skills/documentation-writer - Diátaxis e redação.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- pre-release-rust.md (FASE 11).
- docs/rust/00-visao-estrutura.md - estrutura a documentar.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Novato compila/testa só com o guia; links do README todos resolvendo.

## Riscos / rollback

- Risco: documentar intenção em vez do código real → mitigação: gerar a partir
  de `Cargo.toml`/imports + revisão da FASE-01.
- Rollback: `git checkout -- docs/ README.md`.
