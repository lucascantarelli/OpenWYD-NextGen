# Missão wyd2.md — Fase 5: normalização + 9 issues prontas (2026-11-10)

> END STATE da pesquisa. Só este arquivo é novo; nada mais foi tocado, nada foi
> commitado, nenhuma issue foi criada (títulos/corpos abaixo são PRONTOS PARA
> ABRIR, não abertos). Fontes: `wyd2.md`, skills vigentes, `pre-release-rust.md`,
> `start-rust.md`, `docs/`, OpenWYD `main`, `Source/`/`docs/migration/`, Fases 1–4,
> `client/WYD.exe` vs `wyd_launcher` (referência). Nada inventado: sem VA/file:line
> ou caminho de crate, UNVERIFIED com captura/rito que fecha.

## 1. As 4 linhas sem marca — normalização proposta (aplicar na Fase 6 ou na issue 0)

As 4 linhas estão na tabela secundária "Itens no código sem prova automatizada
(não são ✅)" (`docs/rust/08-checklist-funcionalidades.md`, linhas 194–197). Não
são linhas numeradas da checklist — são justificativas do porquê os itens ficam
em 🚧. A normalização proposta é explicitar o estado em cada uma (todas 🚧, sem
exceção — código existe, prova automatizada não) e corrigir o texto defasado onde
houver. Diff proposto (aplicar com `str_replace`, não aplicado aqui):

| Linha | Texto atual | Proposta (estado + texto) |
|---|---|---|
| 1.13 (logout) | "só o timeout de ociosidade tem código; pacote de saída não é tratado nem testado" → sem marca | `🚧` + "handler de saída existe (`cpsock.rs:1012-1023`, `CNFCharacterLogout` + `leave_world` + `Drop`); falta **teste dedicado** e golden do frame de saída (captura CLIENT-06)" |
| 4.3 (skill em combate) | "`systems/skill.rs` sem teste dedicado e sem fio" → sem marca | `🚧` + texto mantido (correto e atual) |
| 11.7 (re-tema `.wyt`) | "CLI existe; nenhum teste de ida-e-volta de paleta" → sem marca | `🚧` + texto mantido (correto e atual) |
| 5.4 / 5.6 no fio | "regra no core testada, emissão de pacote não" → sem marca | `🚧` + texto mantido (correto e atual) |

Bloqueio registrado: nenhum — é edição de doc com evidência já citada (E1).
Pendente só a aplicação (2 min, fora do escopo "apenas este artefato" desta fase).

## 2. Correção do texto desatualizado 1.13 (proposta, não aplicada)

Linha 48 atual: "libera a sessão por ociosidade (120 s) em `session.rs` e na queda
de socket; pacote de saída não tratado. **Sem teste dedicado** — issue `CLIENT-09`".
Texto proposto: "libera a sessão por ociosidade (120 s), na queda de socket **e no
pacote de saída** (`CharacterLogout → CNFCharacterLogout + leave_world + Drop`,
`cpsock.rs:1012-1023`). **Sem teste dedicado** — issue `CLIENT-09`".
Motivo: o handler existe desde a entrega que fechou o defeito "cliente fechado
continua online"; o que falta é prova (teste + golden), não código. Evidência:
Fase 3 item 3 campos 11/22; Fase 4 §2 linha item 3.

## 3. Nove issues prontas (título + corpo + labels sugeridas)

Convenção: `labels:` são sugestões (`fase`, `tamanho`, `sistema`); depende/bloqueia
conforme `roadmap-rust/index.md` (fila rápida). Todas exigem `make check` verde e
proibem inventar número/layout.

### Issue 1 — Caixa no chão + pickup (`CreateItem`/`GetItem`) + rito da tabela congelada
`labels: fase-08, tamanho-M, sistema-drop`
Corpo: emitir `CreateItem (0x026E)` S→C 30 B packed por item sorteado
(`loot_at_death`, `event=death/event=loot` existentes), com id de caixa do canal
(`id → item, posição, instante`); tratar `GetItem (0x0270)` C→S 28 B validando
`dist≤3` contra posição autoritativa; gravar `character_items`; responder
`CNFGetItem (0x0171)` + `DecayItem (0x016F)`; expirar por timeout; rejeitar
`CreateItem` C→S. Em entrega separada (mesma issue, passo próprio): corrigir
`opcode_table.rs:881-883` (`CreateItem` → `Game2Client`) pelo rito de tabela
congelada. Aceite: golden 30/28 B por VA (`0x486c25`, `0x4fee1b-7e`); coleta grava
e some para todos; concorrência num id só entrega uma vez; `DestType≠1` e 30-vs-32 B
registrados como UNVERIFIED (captura CLIENT-06). Ref: Fase 3 item 1 (26 campos).

### Issue 2 — Cena autoritativa `ESCENE_FIELD` (30000) no ataque/morte/XP
`labels: fase-03, tamanho-M, sistema-cena`
Corpo: broadcast de ataque do mob, `UpdateEtc`/`UpdateScore` no abate e `RemoveMob`
com `ID=ESCENE_FIELD` (nunca id da conexão); `MobID` em +16; `RemoveType` 0/1/2.
Aceite: header de ataque-mob = 30000; barra de XP anda e morte aplica no 7662;
regressão melee `SkillIndex=0` (B12) e guarda `HP<=0` (B9). Ref: Fase 3 item 2; L5 B8/B9.

### Issue 3 — Logout explícito com teste (CLIENT-09)
`labels: fase-08, tamanho-S, sistema-sessao`
Corpo: handler já existe — adicionar `tests/sistema/logout.rs`: `0x0215` responde `0x0116`,
libera `character_session` na hora, salva, segunda conexão entra sem timeout;
queda sem pacote continua no timeout. Atualizar linha 48 (texto §2 acima) e as 4
linhas (§1) na mesma entrega. Aceite: `cargo test` novo verde; sem órfãos no mundo.
Ref: Fase 3 item 3.

### Issue 4 — Cadeia de dano com goldens (GAME-19, núcleo)
`labels: fase-10, tamanho-M, sistema-combate`
Corpo: portar L3 §4 para `wyd-sim` com goldens (`BASE_GetDamage/SkillDamage`,
`rnd` por maestria, pisos, `Ac*=3` PvP, parry `rand()%1000+1`, reflect, mín 1);
travar `STAGE_ORDER`; tabelar `Basedef.cpp:3245+` por classe×arma; LCG MSVC para o
`rand()`; PROIBIDO tocar `wyd-core` até os goldens passarem. Ref: Fase 3 item 4.

### Issue 5 — Árvore de EXP por flags (default inalterado)
`labels: fase-10, tamanho-M, sistema-progressao`
Corpo: cada ajuste L3 §1 (party/`HALFGRID`, divisores tier, −40%, item/fada, RvR,
newbie, `DOUBLEMODE`, `KefraLive`, ±15%, `Hold`, log diário) atrás de flag em
`game_setting`, default = comportamento atual. Aceite: um teste por flag + default
com flags off idêntico ao atual. Ref: Fase 3 item 5.

### Issue 6 — Piedade persistida + `UpdateItem` de resposta
`labels: fase-03, tamanho-M, sistema-refino`
Corpo: persistir pity por item; responder `UpdateItem`/`SendItem` no refino;
manter `SancRate[sanc+1]`, 3/4 (`rand()%4≤2`), travas +6/+9/selado, `EF_NOSANC`,
`Lactolerium_100`→Âmago; B16 (multiplicador, promoção acessório, isenções,
`EF_RUNSPEED`) como subpasso com goldens. Aceite: taxa por passo antes/depois da
recarga; `MissingRate` sem mutação; P-10 registrada. Ref: Fase 3 item 6.

### Issue 7 — Trade atômica por par
`labels: fase-08, tamanho-M, sistema-trade — bloqueada-em-layout`
Corpo: SOMENTE após captura do `MSG_Trade` real (layout UNVERIFIED, invariante 5):
swap atômico item+gold por par, `QuitTrade` cancela ambos, rollback em desconexão,
revalidação de slots, idempotência por oferta. Testes de contenção (dup sob race).
Ref: Fase 3 item 7.

### Issue 8 — Catálogo de ameaças B1–B17 + §6 do `wyd2.md`
`labels: fase-06, tamanho-S, sistema-seguranca`
Corpo: uma linha por ameaça (§6, 10 subitens cada: comportamento, versão, causa,
evidência, impacto, sistema, condições, prevenção, validação, proteção atual);
matriz B1–B17 contra o Rust (passa/falha/aberto); B6 = comportamento esperado
(não implementar); B1 = observar. Aceite: checklist §6 sai de ❌ com a matriz
publicada. Ref: Fase 3 item 8.

### Issue 9 — Harness de replay/paridade (LCG) — base de `wyd-bots`
`labels: fase-15, tamanho-M, sistema-testes`
Corpo: fixture + replay + proxy nos 3 links + paridade LCG MSVC + matriz de
cobertura (espelho de `parity-tests.md`); goldens S→C após CLIENT-06; sanitize de
fixtures; sem `#[ignore]`; bot descobre, teste reproduz sem rede. Aceite:
determinismo antes de carga; benches com números. Ref: Fase 3 item 9.

## 4. Pendentes por dado faltante (não bloqueiam as demais)

Captura CLIENT-06 (frames `0x0215`/`Trade`/S→C, `DestType≠1`, 30-vs-32 B, ids de caixa,
taxas 7662, `SkillData`/curva EXP); rito da tabela congelada; decisão P-10 (5% vs
4%); `HALFGRIDX/Y` exatos. Tudo registrado como UNVERIFIED nos itens da Fase 3 —
nenhuma issue acima depende deles para existir, só a 7 depende de layout para
codar (podendo abrir como `bloqueada-em-layout` desde já, a critério do dono).
