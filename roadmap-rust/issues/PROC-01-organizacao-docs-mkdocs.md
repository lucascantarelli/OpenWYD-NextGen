# PROC-01 — Organização da `docs/`, dedup e MkDocs

> **ID canônico:** `PROC-01`
> **ID histórico:** `DOC-01`
> **Estado 2026-11-12:** 🚧 · ver `roadmap-rust/index.md`.

**Tipo:** Documento · **Esforço:** M · **Depende de:** PROC-13 ✅ · **Bloqueia:** PROC-24

## Objetivo

Transformar `docs/` em projeto MkDocs navegável (índice, navegação, links internos, plugins) e eliminar duplicidades/divergências, mantendo só informação real e usada pelo servidor.

**Público do MkDocs:** desenvolvedores e administradores. É referência interna de engenharia, jogo, operação e skills; a documentação pública para jogadores será publicada no portal, não neste site. Não há perfis, login ou separação de audiência no MkDocs.

## Contexto / evidência

- **132 arquivos Markdown** em `docs/` (arquitetura, cliente, rust, wyd, manuais), com `mkdocs.yml` completo e `docs/index.md`; o `docs-check` mede 181 documentos no escopo total (docs + raiz + `roadmap-rust/`).
- Duplicidades conhecidas registradas em `docs/rust/09-duplicidades.md`: contagem migrations/tabelas agora fixa em **50/86** (`check-migrations.mjs`), com o histórico 31/69 × 36/74 × 44/84 preservado; as 7 pesquisas datadas e os índices legados `wyd/sistemas.md`/`wyd/skills.md` seguem como snapshots rotulados; refino e refino-auditoria continuam separados por papel; skills em `.agents/skills/` agora são **18** e batem com `docs/manual-técnico/skills.md`.
- `wyd-portal/scripts/build-guias.mjs` já consome `docs/` para `/guias` — a fonte é única, os destinos são dois (portal + MkDocs).

## Passos

1. Inventariar `docs/**/*.md`: título, público (jogador/moderador/admin/dev/arquiteto), status (viva/snapshot/histórica), sobreposição com outro arquivo.
2. Fundir sem perder fonte: cada fusão preserva tabela de fontes e classificação (CONFIRMADO/VALIDADO/PROVÁVEL/HIPÓTESE/DIVERGENTE/DESCONHECIDO, padrão `wyd2.md` §3.2); o que for hipótese vira admonition, nunca regra.
3. Atualizar `wyd2.md`, `start-rust.md`, `pre-release-rust.md` como referência histórica (não reescrever história): marcar o que já foi promovido para `docs/` + código com link, e o que segue pendente como lacuna nomeada.
4. Validar MkDocs: `mkdocs build --strict` verde (zero link quebrado) e `mkdocs serve` navegável para dev/admin, sem perfis.
5. Alinhar skills: `.agents/skills/*/SKILL.md` vigentes (`wyd-rust-*`, `wyd-client-reverso`, boas práticas) como norma; legadas Go só como contexto histórico via `docs/manual-técnico/skills.md` + PROC-18.

## Arquivos

- Criados: `mkdocs.yml` (nav completo), `docs/index.md`, `docs/rust/09-duplicidades.md` (inventário de fusões).
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

- docs/index.md — estrutura por tarefa e convenções de escrita para dev/admin.
- software-architecture — fronteiras Hot/Cold nas docs.
- wyd-rust-toolchain — gates.

## Documentação de referência

- `docs/arquitetura/estrutura-e-padroes.md`, `docs/arquitetura/visao-pastas.md`
- `docs/manual-técnico/skills.md`, `docs/manual-técnico/auditoria-roadmap.md`
- `wyd2.md` §3 (classificação), `start-rust.md`, `pre-release-rust.md`

## Subtasks

- [x] Esqueleto MkDocs (`mkdocs.yml` + `docs/index.md`)
- [x] Inventário com sobreposição por arquivo (`docs/rust/09-duplicidades.md`)
- [ ] Fusões com tabela de fontes preservada — refino e pesquisas datadas ainda têm rodada pendente
- [x] `mkdocs build --strict` verde — reexecutado em 2026-11-12 via imagem oficial `squidfunk/mkdocs-material`, exit 0; as 29 páginas que estavam fora do `nav` foram incluídas
- [x] Skills alinhadas sem mudar norma de código (18 skills, `skills-check` 0)

## Critérios de aceite

- [x] `docs/` abre em `mkdocs serve` para dev/admin (sem perfis) — 136 links internos do `nav` responderam 200, 0 falhas; jogadores usam a documentação publicada no portal.
- [ ] Zero duplicidade conhecida sem registro de fusão — o inventário existe; refino e pesquisas datadas ainda têm fusão pendente.
- [x] Nenhum número de servidor privado virou regra sem `game_setting` + teste.
- [x] Gate `make check` continua verde (execução desta rodada).

## Evidência — `mkdocs serve` (2026-11-12)

- **Audiência:** dev/admin. O MkDocs é referência interna; não há perfis, autenticação ou
  divisão jogador/staff/dev. A documentação do jogador será publicada pelo portal.
- **Compose:** serviço `mkdocs` com `squidfunk/mkdocs-material:9.8.0`, `mkdocs.yml` e
  `docs/` read-only; `make docs-serve` publica só `127.0.0.1:${MKDOCS_PORT:-8000}` no
  override de dev. Em produção o serviço fica sem porta, acessível apenas pela rede interna.
- **Comando validado:** `docker compose up -d --wait mkdocs`; o site respondeu em
  `127.0.0.1:8000` após o build.
- **Navegação:** 147 `href`s únicos na página inicial; **136 links internos responderam
  HTTP 200 e 0 falharam**; 11 entradas externas/âncoras/relativas foram ignoradas.
- **Amostras 200:** `/`, `/arquitetura/testes/`, `/manual-técnico/skills/`, `/wyd/`,
  `/cliente/roteiro-prova-visual-7662/` e
  `/rust/evidencias/AUDIT-ISSUES-2026-11-12/`. O roteiro contém o texto esperado.
- **Limite:** a restrição a dev/admin é de implantação/rede, não de perfil no MkDocs; o
  conteúdo do jogador continua no portal.
- **Não fecha a issue:** as fusões de refino e pesquisas datadas continuam pendentes.

## Riscos / rollback

- Risco: apagar fonte histórica útil → mitigação: fundir, nunca deletar sem redirect no índice.
- Rollback: reverter commit de docs (código não é tocado).
