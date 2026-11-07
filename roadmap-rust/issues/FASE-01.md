# Issue 01 — Auditoria do roadmap e organização das issues

**Fase:** 1 · **Depende de:** — · **Bloqueia:** FASE-02 a FASE-15 · **Esforço:** M

## Objetivo

Publicar o inventário auditado de `roadmap/` (Go) e `roadmap-rust/` com duplicidades,
obsoletas e lacunas identificadas, sem mover nenhuma issue para `done/` sem evidência.

## Contexto / evidência

O repo carrega dois planejamentos: `roadmap/` (Go, com `issues/`, `done/`, `test/`) e
`roadmap-rust/` (este diretório). Sem auditoria, o time executa issue duplicada ou
considera pronta uma issue só porque saiu de `issues/`. Regra vigente: `pre-release.md`
§§1.1–1.6 e `roadmap-rust/index.md` (ordem estratégica). **Hipótese:** parte de
`roadmap/done/` não tem evidência de implementação — verificar, não presumir.

## Passos

1. Listar issues Go e Rust e contar por estado:
   `ls roadmap/issues roadmap/done roadmap/test roadmap-rust/issues`.
2. Varrer débito técnico no código legado (somente leitura em `.deprecated-go/`):
   `grep -rn --include='*.go' -E 'TODO|FIXME|XXX|HACK' .deprecated-go/tmserver .deprecated-go/internal | head -n 50`.
3. Cruzar achados com issues existentes; marcar duplicada/obsoleta/sem-aceite.
4. Atualizar `roadmap-rust/index.md` e abrir issues permanentes de revisão por domínio.
5. Rodar o gate: `make check`.

## Arquivos

- Criados: `roadmap-rust/issues/FASE-01.md` (este), atualizações em `roadmap-rust/index.md`.
- Alterados: `roadmap-rust/index.md`.
- Proibido tocar: `.deprecated-go/` (leitura apenas), `client/`, `wyd-portal/` sem motivo,
  qualquer `Cargo.toml`/`src/` (fase de auditoria não gera código).

## Testes / verificação

```bash
make check
git status --short
```

Saída esperada: `Finished` sem erro nos 4 gates; `git status` sem remoções fora de
`.deprecated-go/` e sem arquivos novos fora de `roadmap-rust/` e `docs/`.

## Skills Relacionadas do freebuff

- .agents/skills/writing-plans - planejar a auditoria e estruturar o inventário.
- .agents/skills/documentation-writer - redigir diagnóstico e listas.
- .agents/skills/wyd-rust-toolchain - executar `make check`.

## Documentação de referencia

- docs/rust/00-visao-estrutura.md - estado da transição e estrutura vigente.
- pre-release-rust.md (FASE 1) - escopo da auditoria.
- pre-release.md (§§1.1–1.6) - regras de auditoria e rastreabilidade.
- .deprecated-go/roadmap/index.md - planejamento Go auditado.

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
