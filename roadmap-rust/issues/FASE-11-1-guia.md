# Issue 11-1 — Guia de contribuição + manuais (usuário e técnico) ✅ CONCLUÍDA

**Fase:** 11 · **Depende de:** 02-1 ✅ · **Bloqueia:** — · **Esforço:** M · **Status:** ✅ concluída (2026-11-07)

## Objetivo

Publicar o guia que permite a um novato compilar, testar e contribuir, mais o
esqueleto dos manuais de usuário e técnico.

## O que foi executado

1. **`docs/manual-técnico/guia-contribuicao.md`** (Diátaxis how-to): pré-requisitos,
   primeiro build/teste em 5 comandos, gate `make check`, comandos do portal e das
   ferramentas, fluxo issue→branch→PR de `docs/rust/02-execucao.md`, regras que reprovam
   a contribuição (invariantes de `AGENTS.md` §3) e tabela "onde pedir contexto".
2. **`docs/manual-usuario/README.md`** (a pasta estava vazia): jogar no servidor local,
   CLI real do launcher (`check`/`launch`/`update` com `--dry-run`, conferido em
   `crates/wyd-launcher/src/main.rs`), tabela com as telas reais do portal
   (`wyd-portal/src/app/(admin)/`), o que o portal **não** faz (delivery_queue) e FAQ.
3. **`README.md` normalizado**: bloco de estrutura tinha `docs/` duplicado e
   `.agents/skills/` descrito como "Go (legado) + wyd-rust-*"; virou árvore única com
   subpastas reais e skills vigentes. Skills do §"Começando" ganharam `wyd-rust-testing`;
   tabela de documentação ganhou C4, guia de contribuição e manual do usuário.
4. **`docs/arquitetura/visao-pastas.md`** (feito em 11-2) lista `docs/arquitetura/` completa.

## Evidência (passe 2026-11-07)

```text
cargo fmt --all -- --check                                  → FMT-OK
cargo clippy --workspace --all-targets -- -D warnings       → 0 warnings
cargo test --workspace                                     → 20 suites ok, 0 failed
npm --prefix wyd-portal run test -- --run                  → 118/118 passed
for p in $(grep -o 'docs/[^)]*\.md' README.md | sort -u); do test -e "$p" || echo "QUEBRADO $p"; done → LINKS-OK
grep -rln 'tmserver|.deprecated-go|docs/migration' docs/manual-usuario docs/manual-técnico/guia-contribuicao.md README.md → limpo
```

## Critérios de aceite

- [x] Novato compila/testa só com o guia; links do README todos resolvendo.

## Notas

- Pastas e arquivos conferidos contra o código (`main.rs` do launcher, rotas reais do
  portal) — nenhum comando ou tela inventada.
- Sem `crates/*/src/` tocado. Próxima de menor esforço sugerida: **FASE-12-1**
  (dashboards + alertas + runbooks — docs/config, sem tocar o tick).
