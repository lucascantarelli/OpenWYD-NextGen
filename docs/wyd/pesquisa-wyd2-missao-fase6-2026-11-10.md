# Missão wyd2.md — Fase 6: relatório final consolidado (2026-11-10)

> END STATE. Consolida as Fases 1–5. Só este arquivo é novo; nada mais foi
> tocado, nada commitado, nenhuma issue criada. Níveis: `fontes.md` §1.

## 1. O que foi apurado (por fase)

- **Fase 1** (`fase1-2026-11-10.md`, 152 linhas): imersão — `wyd2.md` integral,
  `pre-release-rust.md`/`start-rust.md`, `docs/` (185 arquivos), skills vigentes,
  `client/WYD.exe` (`04c9d2a1…27ddf8` confere) vs `wyd_launcher` Global (só
  referência), legado `Jean1dev/w2pp-OpenWYD@main` identificado; cobertura
  33 domínios 29✅/18🚧/36⬜/2⛔ (base 2026-11-09).
- **Fase 2** (`fase2-2026-11-10.md`, 173 linhas): cruzamento `Source/` +
  `docs/migration/` (protocol-spec, data-formats, game-rules, flows,
  ingame-bugs) com `wyd-protocol`/Rust; divergência `CreateItem 0x026E` S→C
  reavaliada (binário + fluxo vs tabela congelada); P-1…P-14 reavaliadas sem
  promoção; 9 itens fatiados sem abrir issues.
- **Fase 3** (`fase3-2026-11-10.md`, 494 linhas): 9 itens × 26 campos do §11.2
  (9×26/26 provado por grep); itens 1–2 integrais, 3–9 no mesmo nível; só
  evidência (VA, file:line, caminho de crate) ou UNVERIFIED.
- **Fase 4** (`fase4-2026-11-10.md`, 56 linhas): auditoria — recontagem medida
  28✅/20🚧/32⬜/1⛔/4 sem marca; dispatch real (Account/Secure/Create/Delete/
  Login/Logout/UseItem/ApplyBonus/Attack×3/Action); 6 gaps sem dono.
- **Fase 5** (`fase5-2026-11-10.md`, 124 linhas): normalização das 4 linhas sem
  marca (todas 🚧 + correção 1.13: handler existe, falta teste) + 9 issues
  prontas (título/corpo/labels) sem abri-las.

## 2. Artefatos gerados (5 arquivos, ~75 KB)

`docs/wyd/pesquisa-wyd2-missao-fase{1,2,3,4,5,6}-2026-11-10.md` — cadeia
rastreável requisito → evidência → gap → issue planejada, conforme §§11–12 do
`wyd2.md`. Nenhum código, layout ou fórmula foi inventado em nenhuma fase.

## 3. Decisões e normalizações

1. Legado de referência = `w2pp-OpenWYD` (Go, 7662/proto 7640); demais
   (`snalmir`, org, WASM, Node) só comparativos.
2. Global atual ≠ 7662: versão sempre registrada; defaults (2 canais, teto
   400/camada, 50kk +10, bônus cidadania 0) mantidos até decisão do dono.
3. Fórmulas L3 citadas, não portadas; combine × poeira com rolls distintos;
   `MSG_Trade` UNVERIFIED até captura; tin­tura→feijão = decisão de migração.
4. B6 = comportamento esperado; B1 = observar; D-3 resolvido; D-4 falso positivo.
5. Tabela congelada (`CreateItem` Flow) só pelo rito próprio — registrada como
   `DIVERGENTE-CONFIRMADA`.

## 4. Nove issues planejadas (Fase 5, prontas para abrir)

Caixa+pickup (+rito tabela) · cena `ESCENE_FIELD` · logout com teste (CLIENT-09) ·
cadeia de dano com goldens (GAME-19) · EXP por flags · pity + `UpdateItem` ·
trade atômica (bloqueada-em-layout) · catálogo B1–B17+§6 · harness LCG (base
`wyd-bots`). Ordem e aceite na Fase 5 §3.

## 5. Pendentes / bloqueados (com o que fecha cada um)

Captura CLIENT-06 (frames `0x0215`/`Trade`/S→C, `DestType≠1`, 30-vs-32 B, ids de caixa,
taxas 7662, `SkillData`/curva EXP); rito da tabela congelada; P-10 (5% vs 4%);
`HALFGRIDX/Y`; aplicar diffs Fase 5 §1–§2; abrir as 9 issues; contagem oficial da
checklist após normalizar as 4 linhas. Nada disso impede o uso das Fases 1–5 como
base de implementação.
