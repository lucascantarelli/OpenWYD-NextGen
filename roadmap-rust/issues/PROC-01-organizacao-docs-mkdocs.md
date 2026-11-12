# PROC-01 — Organização da `docs/`, dedup e MkDocs

> **ID canônico:** `PROC-01`
> **ID histórico:** `DOC-01`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Tipo:** Documento · **Esforço:** M · **Depende de:** PROC-13 ✅ · **Bloqueia:** PROC-24

## Objetivo

Transformar `docs/` em projeto MkDocs navegável (índice, navegação, links internos, plugins) e eliminar duplicidades/divergências, mantendo só informação real e usada pelo servidor.

## Contexto / evidência

- 103 arquivos em `docs/` (arquitetura, cliente, rust, wyd, manuais); sem `index.md` nem `mkdocs.yml` até esta issue (`mkdocs.yml` criado como esqueleto + `docs/index.md`).
- Duplicidades conhecidas: contagem migrations/tabelas divergiu 3× (31/69 × 36/74 × 44/84 — gate `check-migrations.mjs` agora trava em `DOCUMENTED 44/84`); `docs/wyd/` tem 7 arquivos `pesquisa-wyd2-missao-fase*-2026-11-10.md` + `pesquisa-*.md` sobrepostos; refino em `sistemas/refino.md` + `sistemas/refino-auditoria.md`; skills em `.agents/skills/` (27) × `docs/manual-técnico/skills.md`.
- `wyd-portal/scripts/build-guias.mjs` já consome `docs/` para `/guias` — a fonte é única, os destinos são dois (portal + MkDocs).

## Passos

1. Inventariar `docs/**/*.md`: título, público (jogador/moderador/admin/dev/arquiteto), status (viva/snapshot/histórica), sobreposição com outro arquivo.
2. Fundir sem perder fonte: cada fusão preserva tabela de fontes e classificação (CONFIRMADO/VALIDADO/PROVÁVEL/HIPÓTESE/DIVERGENTE/DESCONHECIDO, padrão `wyd2.md` §3.2); o que for hipótese vira admonition, nunca regra.
3. Atualizar `wyd2.md`, `start-rust.md`, `pre-release-rust.md` como referência histórica (não reescrever história): marcar o que já foi promovido para `docs/` + código com link, e o que segue pendente como lacuna nomeada.
4. Validar MkDocs: `mkdocs build --strict` verde (zero link quebrado), `mkdocs serve` navegável por perfil.
5. Alinhar skills: `.agents/skills/*/SKILL.md` vigentes (`wyd-rust-*`, `wyd-client-reverso`, boas práticas) como norma; legadas Go só como contexto histórico via `docs/manual-técnico/skills.md` + PROC-18.

## Arquivos

- Criados: `mkdocs.yml` (esqueleto nesta issue), `docs/index.md` (nesta issue).
- Alterados: `docs/**` (fusões), `docs/wyd/README.md`, `.agents/skills/*` (só texto de alinhamento, sem mudar norma de código), `wyd-portal/scripts/build-guias.mjs` (se a fonte mudar de lugar).
- Proibido: mudar árvore de `crates/` (exige ADR), versionar `client/`, segredos, `site/`.

## Testes / verificação

```bash
pip install mkdocs mkdocs-material && mkdocs build --strict
node tools/content/check-migrations.mjs
node tools/content/check-issue-status.mjs
```

Saída esperada: build strict exit 0; sem divergência de status.

## Skills relacionadas

- documentation-writer — estrutura Diátaxis por perfil.
- software-architecture — fronteiras Hot/Cold nas docs.
- wyd-rust-toolchain — gates.

## Documentação de referência

- `docs/arquitetura/estrutura-e-padroes.md`, `docs/arquitetura/visao-pastas.md`
- `docs/manual-técnico/skills.md`, `docs/manual-técnico/auditoria-roadmap.md`
- `wyd2.md` §3 (classificação), `start-rust.md`, `pre-release-rust.md`

## Subtasks

- [x] Esqueleto MkDocs (`mkdocs.yml` + `docs/index.md`)
- [ ] Inventário com sobreposição por arquivo
- [ ] Fusões com tabela de fontes preservada
- [ ] `mkdocs build --strict` verde
- [ ] Skills alinhadas sem mudar norma de código

## Critérios de aceite

- [ ] `docs/` abre em `mkdocs serve` por perfil (jogador, staff, dev).
- [ ] Zero duplicidade conhecida sem registro de fusão.
- [ ] Nenhum número de servidor privado virou regra sem `game_setting` + teste.
- [ ] Gate `make check` continua verde.

## Riscos / rollback

- Risco: apagar fonte histórica útil → mitigação: fundir, nunca deletar sem redirect no índice.
- Rollback: reverter commit de docs (código não é tocado).
