# Issue 01 — Auditoria do roadmap e organização das issues

**Fase:** 1 · **Depende de:** — · **Bloqueia:** FASE-02 a FASE-15 · **Esforço:** M

## Objetivo

Publicar o inventário auditado de `roadmap/` (Go) e `roadmap-rust/` com duplicidades,
obsoletas e lacunas identificadas, sem mover nenhuma issue para `done/` sem evidência.

## Contexto / evidência

O repo carrega o planejamento vigente `roadmap-rust/` (15 fases). O planejamento Go histórico
foi removido com o arquivo em 2026-11; a auditoria preservada vive em
`docs/manual-técnico/auditoria-roadmap.md`. Sem auditoria contínua, o time executa issue
duplicada ou considera pronta uma issue sem evidência. Regra vigente:
`pre-release-rust.md` FASE 1 + `roadmap-rust/index.md` (ordem estratégica).

## Passos

1. Listar issues Rust e contar por estado:
   `ls roadmap-rust/issues` + status em `roadmap-rust/index.md` e `docs/rust/03-progresso.md`.
2. Varrer débito técnico no código Rust:
   `grep -rn --include='*.rs' -E 'TODO|FIXME|XXX|HACK' crates tools | head -n 50`.
3. Cruzar achados com issues existentes; marcar duplicada/obsoleta/sem-aceite.
4. Atualizar `roadmap-rust/index.md` e abrir issues permanentes de revisão por domínio.
5. Rodar o gate: `make check`.

## Arquivos

- Criados: `roadmap-rust/issues/FASE-01.md` (este), atualizações em `roadmap-rust/index.md`.
- Alterados: `roadmap-rust/index.md`.
- Proibido tocar: `client/`, `wyd-portal/` sem motivo,
  qualquer `Cargo.toml`/`src/` (fase de auditoria não gera código).

## Testes / verificação

```bash
make check
git status --short
```

Saída esperada: `Finished` sem erro nos 4 gates; `git status` sem arquivos novos fora de
`roadmap-rust/`, `docs/`, `crates/` (conforme árvore normativa) e `tools/`.

## Skills Relacionadas do freebuff

- .agents/skills/writing-plans - planejar a auditoria e estruturar o inventário.
- .agents/skills/documentation-writer - redigir diagnóstico e listas.
- .agents/skills/wyd-rust-toolchain - executar `make check`.

## Documentação de referencia

- docs/rust/00-visao-estrutura.md - estado da transição e estrutura vigente.
- pre-release-rust.md (FASE 1) - escopo da auditoria.
- docs/manual-técnico/auditoria-roadmap.md - auditoria histórica preservada.
- roadmap-rust/index.md - planejamento Rust auditado.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Inventário publicado com estado real de cada issue (aberta/fechada/evidência).
- [ ] Lista de duplicidades, obsoletas e lacunas registrada.
- [ ] `make check` verde; evidência (saída) anexada.

## Riscos / rollback

- Risco: classificar issue errada como concluída → mitigação: exigir link de evidência (commit/teste).
- Rollback: `git checkout -- roadmap-rust/` restaura o planejamento (não committado).
