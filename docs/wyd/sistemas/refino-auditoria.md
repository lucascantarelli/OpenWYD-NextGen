# Refino +0..+15 — auditoria público × legado (`origin/main`) × Rust

Auditoria de 2026-11-08. Responde a três perguntas: **o que já temos bate com as fontes
públicas?**, **bate com o OpenWYD (legado)?** e **o que o legado tem que ainda não portamos?**

Fontes conferidas nesta auditoria:

| Lado | Onde |
|---|---|
| Público | Central Kersef (*Refining System +0 to +11*, *Refining +10*, *Armas Ancient*, *Refinação +12 a +15*), WYD International (*Server Information*, *Celestial/God2 Rates*), wiki Fandom (**403** em 2026-11-10), WYD Brazuca, guia WYDGalahan 2009 (5% no +12) |
| Legado | `origin/main`: `Source/Code/TMSrv/_MSG_UseItem.cpp`, `_MSG_CombineItem*.cpp`, `tmserver/content/Common/Settings/{SancRate,RefineRate,CompRate}.txt` |
| Nosso | `crates/wyd-sim/src/formula/refine.rs`, `crates/wyd-db/migrations/{0009,0022,0025,0026}*.sql`, `crates/wyd-core/src/systems/*` |

---

## 1. Escada +0..+15, lado a lado

| Passo | Público | Legado (`origin/main`) | Nosso banco (`refine_rate`) | Nosso `wyd-sim` |
|---|---|---|---|---|
| +0→+1 … +8→+9 | 90/85/80/75/70/65/60/55/50% | **outra régua**: tabela `SancRate` por pó + piedade (ver §2); tetos 6 (Ori) e 9 (Lac) | 9000…5000 bps, `kind='dust'` | `refine_success_bps(1..=9)` ✅ |
| +9→+10 | 30% (faixa publicada 30–40%) | **Ailyn**: `1 + 4 × ChanceBase(10)` = **41%** no `CompRate.txt`, 50 kk | 3000 bps, `kind='compose'` | `refine_success_bps(10)` ✅ |
| +10→+11 | 3% com Poeira de Lac; **100%** com "Lac 100%" | **15% fixo em código** no passo `GEM_LEVEL`; **100%** se o pó for `Lactolerium_100` (4141) | 300 bps, `kind='dust'` | `powder_success_bps` (pó) + `refine_success_bps(11)` ✅ |
| +11→+12 | **5%** (guia da era oficial, 2009) / **4%** (centrais atuais) + 1%/pedra secreta + bênção | **Odin**: base `g_pOdinRate[2] = 100`, penalidades por nível do alvo, **sucesso incondicional no código vivo** | 400 bps hoje (mig. 0022), alvo 500 bps | `plus12_rate_bps` ✅ |
| +12→+13 | 3% | idem (`Plus12NewLevel`) | 300 bps | idem |
| +13→+14 | 2% | idem | 200 bps | idem |
| +14→+15 | 1% | idem | 100 bps | idem |
| Anct | base 5%; +7/+8/+9/+10 = 2/4/10/12% → 53% com 4 reforços +10; resultado **+7** | base **1%**; só +7/+8/+9 = 2/4/10 (10 recusado); sem cap de 4 (até 6); resultado **+7** | — (sem tabela) | `AnctRules::{LEGACY,PUBLISHED}` + `anct_rate_bps` ✅ |
| Gema de efeito | 4 gemas; perfuração/absorção 40×(nível−9); drop 8%, EXP 2% fixos | joia escolhe o `Extra` do resultado (Anct/Ailyn); efeito é dado do item | `refine_gem_effect` (4 linhas) | `apply_gem`/`gem_effect_value` ✅ (core consome) |
| Falha no pó | +1…+6 volta a +0; +7…+9 destrói **item selado** | **nunca destrói** no caminho de equipamento comum: só gasta o pó e soma piedade | — | `powder_failure(sealed, level)` ✅ |
| Falha no +12..+15 | Lac: insumos + item (**20% de destruição se a taxa total < 5%**); Emblema: **cai 1 nível** | ramo de falha do `+12+` **comentado** (sempre sucesso); composição Odin consome tudo | — | `plus12_failure` ✅ |

### Piedade (pity) — mecânica que não existia no nosso lado

O legado não usa a escada publicada: cada tentativa com pó soma a **piedade** do item
(`BASE_GetItemSancSuccess`, guardada no mesmo `cValue` do refino) à célula da tabela.

- Peso por ponto de piedade (`g_pSuccessRate`), indexado por **nível+1**:
  `5, 5, 5, 5, 4, 4, 3, 3, 2, 1, 0` → +0 = 5 … +8 = 1, +9 = 0.
- Só **3 de 4** falhas acumulam (`rand()%4 <= 2`); teto **20** pontos.
- Lac acumula sempre; **Ori só até +5**.
- Consequência real: com Ori, +5→+6 tem célula **0** e só sobe por piedade (20 × 3 = 60%).

---

## 2. Tabela `SancRate` — o erro que a auditoria encontrou no nosso seed

O legado **não** lê o arquivo como tabela completa: `content.SancRate` começa do default
compilado (`Basedef.cpp g_pSancRate`) e o `SancRate.txt` **só sobrescreve os índices que
lista**. A leitura é em `nível + 1` (o slot 0 não é usado).

Tabela **efetiva** (arquivo por cima do default):

| Pó | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| PO | 100 | 100 | 100 | 85 | 70 | 40 | 0 | 0 | 0 | 0 | 0 | 0 |
| PL | 100 | 100 | 100 | 100 | 100 | 100 | 80 | 70 | 70 | 10 | 0 | 0 |
| Âmago | 100 | 80 | 60 | 40 | 20 | 10 | 10 | 10 | 5 | 5 | 5 | 5 |

Célula de índice `i` é a taxa de **+ (i−1) → +i**. Então: PO 5 = 40% é o passo +4→+5; o
+5→+6 lê PO 6 = **0**. PL 9 = 10% é o +8→+9; +9→+10 é recusado por `PL_MAX` antes do sorteio.

**O que corrigimos:** o seed `0025` gravou as 28 linhas do arquivo e tratou ausência como
"sem dado". Faltavam as 8 células que o loader mantém do default (PO 6–11, PL 10–11 = 0).
Sem elas, +5→+6 (Ori) e +9→+10 (Lac) ficariam sem taxa — quando o valor real é
`0 + piedade × peso`. A migração `0026_seed_sanc_rate_defaults.sql` completa a tabela para
**36 células** e a ADR-0019 ganhou emenda.

> Correção adicional à ADR-0019: as células Âmago 10/11 (5%) **não** são a taxa do passo
> +10→+11. `BASE_GetSuccessRate` curto-circuita `level == GEM_LEVEL` para **15%** (ou 100% no
> Âmago) e nunca consulta a tabela nesse nível. O 5% é inalcançável pelo caminho de pó.

---

## 3. Divergências registradas (não são bugs de leitura)

| ID | Ponto | Público | Legado | Decisão no nosso código |
|---|---|---|---|---|
| D-R1 | Escada +0..+9 | 90…50% | ~100% até +5 e caindo, com piedade | `refine_rate` = público (config do servidor); `sanc_rate` = legado; os dois coexistem |
| D-R2 | +9→+10 | 30% (30–40%) | Ailyn 41% (`1 + 4×10`) | seed = 30% (doc §3); sim aceita qualquer base via `AnctRules`/`refine_rate` |
| D-R3 | +10→+11 | 3%; Lac 100 = 100% | 15% fixo; Lac 100 = 100% | seed = 3% (público); sim reproduz **os dois** caminhos |
| D-R4 | +12..+15 | **5/3/2/1%** (guia 2009) vs **4/3/2/1%** (centrais atuais) + pedras + bênçãos | base 100 + runas/penalidades, falha comentada | seed atual = 4/3/2/1% (público atual), alvo 5/3/2/1%; sim implementa a régua; ramo Odin no backlog |
| D-R5 | Anct base | 5% | 1% | `AnctRules::PUBLISHED` e `::LEGACY`, explícitos |
| D-R6 | Bênção por refino | 1/1/2/3/4/5/6/7/8/10% (WYD Intl) **ou** 2/3/4/5/6/7/8/10/12/15% (Kersef) | não existe (usa pedra secreta) | código usa a primeira (`BLESSING_BONUS_BPS`); default documentado passou à do Kersef (2…15%) — P-10 |
| D-R7 | Jóia como taxa | páginas citam +4/+8/+15/+32% | jóia **não** entra na taxa: escolhe o efeito/gema | **não implementado** — falta fonte primária; registrado em §6 |
| D-R8 | Refino de **brinco/acessório** | WYD Int: **15%** (+9→+10) e **10%** (+10→+11), **destrói**; WYD Global Bahamut: **40%** até +9 e **5%** de +9 a +11, **não destrói** | não existe no legado (acessório não refina) | **não implementado** — nenhuma fonte da era oficial; no 7662 fica UNVERIFIED e sem linha no seed |

---

## 4. O que o legado tem e ainda não está no Rust

Status: ✅ implementado · 🟡 fórmula pronta em `wyd-sim`, falta fiação de jogo · ⬜ ausente.

| Sistema | Fonte legada | O que faz | Status |
|---|---|---|---|
| Pó + piedade + tetos | `_MSG_UseItem.cpp:802-976`, `SancRate.txt`, `RefineRate.txt` | taxa, +1 por sucesso, piedade, recusas, +10→+11 mantendo a gema | 🟡 `powder_*`/`RefinePolicy` |
| Ovos de montaria | `_MSG_UseItem.cpp:890-951` | ovo 2300–2329 choca em +30, `EF_INCUBATE`/`EF_INCUDELAY`, cooldown | ⬜ |
| Tintura → Feijão Mágico | `_MSG_UseItem.cpp` (ramo tintura) | 3397–3406 → 3407–3416, determinístico | ⬜ |
| Recusa `EF_NOSANC` (126) | `_MSG_UseItem.cpp:178` | item que nunca refina | ⬜ |
| Ramo **selado** do pó | `_MSG_UseItem.cpp:224` (adiado no legado) | `rand()%115` com −15, destrói em falha | ⬜ (só a regra de falha) |
| Ramos celestial/HC, pedras arch, brincos +10..14 | `_MSG_UseItem.cpp:385/514/716` (adiados) | destruição/rebaixamento na falha | ⬜ |
| Anct (Compositor) | `_MSG_CombineItem.cpp`, `combine/match.go:MatchAnct/AnctResult` | taxa 1% + 2/4/10, joia+`Extra`, resultado +7, gates de pos/itemlevel/mobtype | 🟡 taxa; match/resultado ⬜ |
| Ailyn (+10) | `_MSG_CombineItemAilyn.cpp`, `MatchAilyn` | 2 itens +9 iguais + 4 jóias + Pedra do Sábio + 50 kk; resultado herda efeitos do 2º, sanc 10, gema = joia−2441 | ⬜ |
| Tiny | `MatchTiny`, `_MSG_CombineItem.cpp` | base 15 + itemlevel×5, 100 kk, falha destrói o alvo | ⬜ |
| Agatha | `MatchAgatha` | base 15 + grade×5 + (itemlevel 5 → 30) | ⬜ |
| Shany | `MatchShany` | base 30, Arch ≥ 355, entrega item 633 | ⬜ |
| Alquimia | `MatchAlquimia` | classe 3, 10 receitas → itens 3200+id, taxa `(Special[2]+1)/6` | ⬜ |
| Lindy | `MatchLindy` | Arch 354, desbloqueia 355/370, receita de 7 slots | ⬜ |
| Ehre | `MatchEhre`, `g_pEhreRate` | 8 receitas (100/100/40/10/100/100/100/100) | ⬜ |
| Extracao | `combine.go:combineExtracao` | Pedra do Sábio como catalisador, `(Special[2]+1)/6`, resultados 3021–3025 | ⬜ |
| Odin (11 receitas) | `_MSG_CombineItemOdin.cpp` | +12, Pista de Runas, Destrave Lv40, Pedra da Fúria, 4 pedras Secretas, Semente de Cristal, Capa Celestial, Composição de Sets/armas | 🟡 só a taxa do +12 |
| Sorteio de combine | `_MSG_CombineItem.cpp` | `rand()%115`, achata ≥100 em −15 (distribuição dupla em 85..99); Odin usa `rand()%199` −99 | ⬜ |
| Bônus de item por faixa (`SetItemBonus`) | `Server.cpp:1777-2717` | 3 slots de efeito, tiers por `lvdif`, grade eleva sanc, ruído `EF_UNKNOWN` | ⬜ |
| Bônus por classe (`SetItemBonus2`) | `Server.cpp:2719-2861` | elmo/peito/calça/luva/bota: re-sorteia 2 efeitos e sanc até +6 | ⬜ |
| Repleção / Imbuir | docs §7 | estados independentes do item | ⬜ (fora do refino) |

---

## 5. O que já estava certo (não mexer)

- **4 gemas** (3386–3389), efeito por aplicação direta, troca ilimitada, Anct aceito abaixo
  de +10 (`apply_gem` + `refine_gem_effect` + `tests/golden/gema_seed.rs`).
- **40 × (nível − 9)** para perfuração/absorção; **8%** drop e **2%** EXP fixos.
- **15 passos** em `refine_rate` com família `dust`/`compose` e fonte por linha.
- **Escada +0..+9 publicada** e tetos do legado (6/9/10/11) agora explícitos.

## 6. O que continua UNVERIFIED

| Ponto | Por quê |
|---|---|
| Jóia como bônus de taxa (+4/+8/+15/+32%) | páginas citam o número, mas o legado não usa jóia na taxa e não há tabela primária; não virou código |
| Chance de destruição com Pacote de Lac ("6%") | a fonte diz "pequena chance" sem número; o 6% não tem origem primária confirmada |
| Fórmula do refino sobre atributos (1,9× no +9) | afirmação pública sem tabela por peça |
| Taxa do +10→+11 em servidores oficiais | 3% × 15% × 5% são três configurações; o nosso seed adota 3% e registra as outras |
| Ramo de falha do `+12+` do legado | está comentado no código vivo; portar o comentário seria inventar |

## 7. Evidência

- `crates/wyd-sim/src/formula/refine.rs` — escada de 15 passos, `Powder`, `RefinePolicy`,
  piedade, falha, bênção, Anct (duas réguas), +12..+15, gemas. 16 testes de unidade.
- `crates/wyd-sim/tests/golden/refino_seed.rs` — a fórmula contra `0022`/`0025`/`0026`: 15 passos e
  36 células, leitura em `nível+1`, passos especiais fora da tabela.
- `crates/wyd-db/migrations/0026_seed_sanc_rate_defaults.sql` — 8 células de default compilado.
- Gates: `cargo test -p wyd-sim`, `cargo test --workspace`, `check-migrations`,
  `gen-catalogo-banco --check`, `verify-schema` (registrados em `docs/rust/03-progresso.md`).
