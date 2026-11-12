# Matemática do jogo (`wyd-sim`) — inventário e dívidas

Este documento é o **mapa curto** do laboratório: o que existe, onde mora e o que falta.
O documento normativo (entradas, saídas, unidade, ordem no tick, arredondamento e as
decisões provisórias numeradas) é **`docs/arquitetura/matematica-sim.md`**; a árvore está em
`estrutura-e-padroes.md` §14 e na **ADR-0011**.

## 1. Princípios

1. **Fórmula pura, sem I/O.** `formula/` recebe números e devolve números. Nada de rede,
   banco, relógio ou `rand` — quem sorteia é `roll::Die` (seed explícita), nunca a fórmula.
2. **Nada de `f32`/`f64` em regra de jogo.** Percentual é `Bps` (`10_000` = 100%), decimal de
   item é `Tenths`, duração é tick inteiro.
3. **Fato × hipótese separados.** O que é confirmado por fonte mora na fórmula; o que é
   hipótese vira **parâmetro** (`GrowthProfile`, `ExpTable`, `AttrWeight`) ou decisão
   provisória em `matematica-sim.md` §4.
4. **Um sistema = um arquivo.** Fórmula nova entra como módulo novo, com golden próprio (OCP).
5. **Servidor é autoridade.** `wyd-sim` é o laboratório; a regra aplicada em produção vive em
   `wyd-core`, e as duas pontas são presas pela mesma tabela de golden.

## 2. Árvore e responsabilidade

```
crates/wyd-sim/
├── src/
│   ├── lib.rs              # fachada: declara e reexporta (sem regra)
│   ├── error.rs            # SimError
│   ├── units.rs            # Bps, Tenths, mul_bps (truncamento único)
│   ├── roll/               # Die + XorShift64 (determinismo com seed)
│   ├── formula/            # um sistema por arquivo
│   │   ├── mod.rs          # só declara/reexporta
│   │   ├── attributes.rs   # derivados: nível → atributo → equipamento
│   │   ├── progression.rs  # pontos, tetos de maestria, degraus, EXP (tabela = dado)
│   │   ├── damage.rs       # base→skill→buff→crítico→defesa/perfuração→resist→absorção
│   │   ├── hit.rs          # acerto, evasão, crítico
│   │   ├── effect.rs       # efeitos com duração em ticks
│   │   ├── resist.rs       # resistência elemental
│   │   ├── drop.rs         # roll por item (1..1.000.000)
│   │   ├── regen.rs        # HP/MP por tick com resto acumulado
│   │   ├── threat.rs       # aggro e estado de criatura (matemática)
│   │   └── refine.rs       # +0..+15, gemas (GAME-02/10)
│   ├── scenario/combat.rs  # ordem do golpe e do tick
│   └── report/table.rs     # saída comparável em texto
└── tests/
    ├── goldens.rs          # valor exato com fonte no nome (inalterado)
    └── determinism.rs      # mesma seed → mesma partida; rolls gravados → replay
```

`mod.rs` **nunca** carrega lógica: declara módulo e reexporta (mesma regra de `ecs/mod.rs`).

## 3. Inventário por domínio

| Domínio | Módulo | Estado | Fonte do número |
|---|---|---|---|
| Atributos e derivados | `formula/attributes.rs` | ✅ | pesos = conteúdo; Arch +8 por nível (`classes.md` §4) |
| Progressão e pontos | `formula/progression.rs` | ✅ | `progressao.md` §2, `skills.md` §2 |
| Curva de EXP | `formula/progression.rs` (`ExpTable`) | ⛔ sem dado | não publicada (`progressao.md` §7) |
| Dano e mitigação | `formula/damage.rs` | ✅ (ordem D-02..D-04) | `skills.md` §5 (composição) + invariante de absorção |
| Acerto/evasão/crítico | `formula/hit.rs` | ✅ (base D-06) | `classes.md` §2, `itens.md` l. 149 |
| Efeitos com duração | `formula/effect.rs` | ✅ (D-07) | `skills.md` §4 (`affect_type/value/time`, `delay_ticks`) |
| Resistência elemental | `formula/resist.rs` | ✅ (D-11) | `classes.md` §2 (fogo/gelo) |
| Drop por item | `formula/drop.rs` | ✅ | `sistemas/drops.md` §1 (roll 1..1.000.000) |
| Regeneração | `formula/regen.rs` | ✅ | `classes.md` §2 (2.000 HP/s) |
| Aggro / IA de criatura | `formula/threat.rs` | ✅ matemática; FSM na GAME-06 | `adr/0010-ia-npc-e-bots.md` §5 |
| Refino +0→+15 (taxa por passo) | `formula/refine.rs` | ✅ | WYD International (90…50%) + Aylin 30% + Lac 3% + Odin 4/3/2/1% |
| Refino com pó do legado (piedade, tetos) | `formula/refine.rs` | ✅ | `SancRate.txt` (36 células efetivas) + `RefineRate.txt` + `g_pSuccessRate` |
| Gema de efeito (+10..+15) | `formula/refine.rs` | ✅ | Kersef + ABC (40 × (nível−9); drop 8%, EXP 2%) |
| Composição Anct / +12..+15 (bônus) | `formula/refine.rs` | ✅ taxas; fiação ⬜ | `docs/wyd/sistemas/refino-auditoria.md` §3 |
| Velocidade / hits duplicados | `formula/speed.rs` | ⛔ bloqueado | GAME-20 (causa não provada) |
| Economia (imposto, preço, drop de gold) | `formula/economia.rs` | ⬜ planejado | Kersef (10% loja, 40% castelo) |
| Arena / Battle Royale (pontuação) | `formula/arena.rs` | ⬜ planejado | `docs/wyd/sistemas/battle-royale.md` |

⛔ = **não implementar sem prova**: o módulo existe no plano para ninguém "resolver" por
chute; `scenario/` pode explorar hipótese, `formula/` não.

## 4. Dívida consciente: regra duplicada em `core` e `sim`

`apply_gem` (política da gema) existe nos dois lados por direção de dependência: `wyd-core`
não pode depender de `wyd-sim` (laboratório offline não entra no binário do servidor) e
`wyd-sim` não depende de `wyd-core`.

Mitigação aplicada:

- os **números** são os mesmos e estão presos por golden nos dois crates;
- a **fonte** é o manual (`docs/wyd/sistemas/refino.md` §3) e o banco (`refine_gem_effect`);
- a unificação (enum/contrato em `wyd-protocol`) é dívida registrada para a GAME-19, quando
  a ordem de combate fechar e o contrato de efeito estabilizar.

Estado atual da dívida: a **regra** existe só como tabela duplicada; a **transação** existe
só no core. `ComponentStore::apply_gem` é o caminho único de aplicação — valida sem escrever,
consome uma unidade da mochila, publica o evento e só então grava o efeito. Recusa é erro
tipado e o inventário fica idêntico (`crates/wyd-core/tests/integracao/gem.rs`). `Inventory::{plan_gem,
consume_gem, commit_gem, apply_gem_from_bag}` são `pub(crate)`: de fora do crate não há como
aplicar gema numa cópia solta.

## 5. Como provar que a matemática está certa

| Nível | O que valida | Onde |
|---|---|---|
| unitário | faixa, borda, erro tipado, saturação | `#[cfg(test)]` no próprio módulo |
| golden | **valor exato** de tabela publicada | `tests/golden/formulas.rs` |
| composição | sistemas na mesma unidade (derivado → dano → drop) | `formula/mod.rs` (teste de composição) |
| determinismo | mesma seed → mesma partida; replay com rolls gravados | `tests/integracao/determinism.rs` |
| servidor | a regra aplicada de verdade | teste de `wyd-core` com o mesmo número |

Regra: golden novo entra **com a fonte citada no nome do teste** (`gem_scale_matches_kersef`)
— número sem procedência é chute com aparência de teste.
