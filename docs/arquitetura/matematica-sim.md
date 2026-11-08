# Matemática de `wyd-sim` — sistemas, unidades e ordem do tick

> **Norma:** a árvore de `crates/wyd-sim/src/` é a desta página (§2), decidida na
> [`ADR-0011`](adr/0011-estrutura-matematica-sim.md). Arquivo novo fora dela exige ADR
> (rito FASE-13-1, `estrutura-e-padroes.md` §14).
>
> **Relação com o planejamento mestre:** `pre-release-rust.md` FASE 10 §10.1 (equações),
> §10.2 (velocidade) e §10.3 (laboratório em milhões de iterações) definem o *objetivo*;
> `start-rust.md` define a razão de ser da reescrita (sem GC, DOD/ECS, sem I/O no Hot Path).
> O que está implementado, o que é plano e o que diverge dos dois está na §6.
>
> **Irmãos:** `docs/rust/06-matematica.md` (inventário curto e dívidas),
> `docs/rust/07-ia-e-bots.md` (IA de NPC e bots). Manual de jogo: `docs/wyd/`.

## 1. Por que este documento existe antes do código

O projeto exige "estrutura antes de código" (`AGENTS.md` §7). Em `wyd-sim` a estrutura é
**consequência da matemática**: só se sabe que `attributes` e `progression` são arquivos
diferentes quando se sabe que um produz derivados por nível e o outro consome uma curva de EXP
que o jogo não publica. Sem essa distinção, todo número novo cai no arquivo que já está
aberto — o erro que a missão chamou de "tudo no `rott`".

Três regras valem para todo módulo abaixo:

1. **Entrada numérica, saída numérica.** Sem I/O, sem relógio, sem `rand` interno.
2. **Fato × hipótese.** Constante publicada fica no código com a fonte no comentário; o que a
   fonte não prova entra como **parâmetro** ou vira decisão provisória numerada (§4).
3. **Zero `f64` em regra.** Percentual é `Bps` (`10_000` = 100%); decimal de item é `Tenths`
   (`7.0` = `70` décimos); EXP é `u64`; duração é tick inteiro.

## 2. Sistemas, módulos e contratos

| # | Sistema | Módulo | Entradas | Saídas | Unidade | Estado |
|---|---|---|---|---|---|---|
| 1 | Atributos e derivados | `formula/attributes.rs` | nível, `AttrPoints`, `GearBonus`, `GrowthProfile` | `Derived { max_hp, max_mp, attack, defense }`, `arch_attribute_points` | `u32` absoluto | ✅ implementado (pesos = dado) |
| 2 | Progressão e XP | `formula/progression.rs` | `LevelState`, `ExpTable`, nível | `LevelState`, pontos, tetos, `SkillRequirement` | EXP `u64`, nível `u16`, pontos `u32` | ✅ implementado (curva = dado) |
| 3 | Dano e mitigação | `formula/damage.rs` | `DamageInput` | dano final | `u32` + `Bps` | ✅ implementado (ordem provisória) |
| 4 | Acerto, evasão, crítico | `formula/hit.rs` | acurácia, evasão, crítico em décimos, modificadores | `Bps` de chance, `bool` | `Bps` | ✅ implementado (base provisória) |
| 5 | Skills e efeitos com duração | `formula/effect.rs` | `affect_type`, magnitude, segundos, maestria, modificador | `EffectSet`, `TimedEffect`, ticks | ticks (20 Hz) | ✅ implementado (não-empilhamento provisório) |
| 6 | Resistências elementais | `formula/resist.rs` | dano, `ResistProfile`, `Element` | dano reduzido | `u32` + `Bps` | ✅ implementado (3 elementos) |
| 7 | Drop e taxa por item | `formula/drop.rs` | `rate`, roll, bônus de gema | `bool`, taxa efetiva | escala `1..1.000.000` | ✅ implementado (teto do bônus = topo da escala) |
| 8 | Regeneração | `formula/regen.rs` | taxa/s, `tick_hz` | HP por tick + `RegenAccumulator` | `u32` absoluto | ✅ implementado |
| 9 | Aggro / IA de criatura | `formula/threat.rs` | entidade, dano, tick, TTL | `ThreatTable`, alvo, `NpcMode` | ameaça `u32` | ✅ **matemática**; FSM no core (FASE-03-6) |
| 10 | Refino e gemas | `formula/refine.rs` | `item_index`, refino, pó, piedade, Anct | `AppliedGem`, taxas em bps | `Bps` / `u32` | ✅ (FASE-03-2/10 + auditoria 2026-11-08): escada de 15 passos, pó com piedade/tetos, Anct e +12..+15; aplicação no core consome a gema (D-12/D-13) |
| 11 | Velocidade e hits | `formula/speed.rs` | — | — | — | ⛔ bloqueado (FASE-10-2, causa não provada) |
| 12 | Ordem do tick e do golpe | `scenario/combat.rs` | `Combatant`, `AttackProfile`, `Die` | `AttackOutcome` | — | ✅ implementado |
| 13 | Saída comparável | `report/table.rs` | linhas de texto | `Table.render()` | texto | ✅ implementado |
| — | Sorteio determinístico | `roll/xorshift64.rs` | seed `u64` | rolls `1..=max` e `0..10.000` | inteiro | ✅ implementado |
| — | Unidades | `units.rs` | — | `Bps`, `Tenths`, `mul_bps` | — | ✅ implementado |
| — | Política de bot | `bot/{persona,policy}.rs` | perfil, estado | intenção | — | ⬜ planejado (FASE-15-3) |

Fronteiras por trait (as duas que existem, e por quê):

| Trait | Arquivo | Implementações hoje | Para que serve |
|---|---|---|---|
| `Die` | `roll/mod.rs` | `XorShift64`; reprodutor de rolls no teste de replay | isola o sorteio: fórmula não sorteia, e o replay troca o gerador sem tocar na regra |
| `ExpTable` | `formula/progression.rs` | `ExpCurve` (memória, validada) | a curva de EXP é **conteúdo** (ADR-0009) e não pode virar constante do crate |

Não existe trait para `content`/banco: `wyd-sim` não fala com banco. Quem carrega dado é
`wyd-db`; quem valida no boot é `wyd-core` (`ports/content.rs`).

## 3. Ordem de resolução e arredondamento

### 3.1 Unidades e arredondamento (regra única)

| Grandeza | Representação | Arredondamento |
|---|---|---|
| Percentual (chance, resistência, buff, refino) | `Bps` — `10_000` = 100% | **truncamento (piso)** em `mul_bps` |
| Decimal de item (crítico `7.0`) | `Tenths` — `70` décimos | conversão exata (`décimos × 10` bps) |
| Dano, HP, MP, defesa, absorção, perfuração | `u32` absoluto | saturação em `u32::MAX`; nunca *wrap* |
| EXP | `u64` | soma inteira; excedente no teto é **retido** |
| Nível | `u16` | 1..=400 (Mortal) |
| Duração de efeito | `u32` ticks a 20 Hz (`TICK_HZ`) | multiplicação inteira; 64 s = 1.280 ticks |
| Roll de drop | `u32` em `1..=1.000.000` | comparação `roll ≤ rate` |
| Roll de chance | `u32` em `0..10.000` | metade aberta em cima: `roll < chance` |

Motivo do truncamento: o servidor de referência é C++ com divisão inteira; "arredondar para o
mais próximo" criaria divergência de 1 ponto em qualquer conta de 90% — e divergência de 1 HP
é bug reportado. É decisão **provisória** (D-10), confirmável por trace.

### 3.2 Ordem de um golpe (fases fixas)

```
acerto (1 roll sempre)
  └─ errou → dano 0, sem consumir roll de crítico
crítico (1 roll só quando acertou)
  └─ dano: base + atributo + equipamento
            → × skill (Bps)
            → × buff (Bps)
            → × multiplicador de crítico (Bps; 10_000 se não critou)
            → − (defesa − perfuração), piso 0
            → − resistência elemental (Bps), piso 0
            → − absorção (plana), piso 0
```

`STAGE_ORDER` em `formula/damage.rs` e o teste `stage_order_is_the_documented_one` prendem a
lista. A assimetria "roll de crítico só quando acerta" é o que mantém dois replays alinhados
(`scenario/combat.rs`).

### 3.3 Ordem do tick

`TICK_PHASES` em `scenario/combat.rs`:

| # | Fase | O que acontece | Onde |
|---|---|---|---|
| 1 | `effects_tick` | duração −1 tick; expirado sai | `effect::EffectSet::tick` |
| 2 | `regen` | HP/MP sobem pela taxa acumulada | `regen::per_tick` + `apply_to` |
| 3 | `attacks` | golpes em ordem crescente de `entity` do atacante | `scenario::resolve_attack` |
| 4 | `threat` | ameaça do alvo recebe o dano aplicado | `threat::ThreatTable::add` |
| 5 | `deaths` | HP 0 vira morte (o *despawn* é do core) | `wyd-core` |
| 6 | `events` | eventos saem pela fila, nunca por I/O no meio | `wyd-core::DomainEvents` |

Ordem crescente de `entity` no passo 3 não é estética: sem ela, a ordem de iteração do
`Vec` de entidades mudaria o resultado quando dois golpes matam o mesmo alvo, e o replay
divergiria do servidor.

### 3.4 Ligação com os dados (banco → fórmula)

| Fórmula recebe | Vem de | Estado do dado |
|---|---|---|
| `GrowthProfile`, `AttrWeight` | `mob_template_stat` (stats de criatura) + classe | ⚠️ tabela vazia hoje |
| `ExpTable` / `ExpCurve` | curva de EXP exportada/medida | ⛔ não publicada (`docs/wyd/progressao.md` §7) |
| `rate` de drop | `drop_entry` (uma linha por item+grade) | ⚠️ parcial (`docs/wyd/sistemas/drops.md` §7) |
| `skill_catalog.affect_type/value/time`, `delay_ticks` | `skill_catalog` (154 linhas) | ✅ carregado |
| `refine_rate.rate_bps`, `refine_gem_effect` | banco | ✅ carregado |
| `game_setting` (`partydif`, `double`) | banco | ✅ carregado |

Nenhuma fórmula lê tabela: o chamador resolve o dado e passa número. É o que permite rodar
milhões de iterações sem Postgres (`pre-release-rust.md` §10.3).

## 4. Decisões provisórias (ambiguidade não inventada)

Cada linha é uma ambiguidade **da fonte**, com a escolha mais simples, a justificativa e onde o
trace do cliente confirma. Enquanto não houver trace, a decisão vale só no laboratório;
`wyd-core` não aplica ordem não provada (invariante 3 do `AGENTS.md`).

| ID | Ambiguidade | Escolha | Justificativa (mais simples) | Confirma com |
|---|---|---|---|---|
| D-01 | Quanto HP/ataque vale 1 ponto de atributo | pesos são entrada (`AttrWeight`), composição fixa nível → atributo → equipamento | nenhuma fonte publica a tabela; somar em ordem fixa não inventa escala | stats de personagem recém-criado em `mob_template_stat`/cliente |
| D-02 | Defesa é subtrativa ou multiplicativa | **subtrativa** (`dano − defesa`), piso 0 | é a leitura que o manual descreve como "redução por defesa" e a que o legado implementa; multiplicativa exigiria constante de curva | log de dano do cliente com defesa variável |
| D-03 | Perfuração age na defesa ou no dano final | reduz a **defesa efetiva** (`defesa − perfuração`) | "perfuração" descreve atravessar proteção; aplicada no dano final seria só dano extra | gema de esmeralda +10 vs alvo de defesa alta |
| D-04 | Absorção antes ou depois do crítico | **depois** de tudo, última etapa | absorção é plana (40..240): aplicada antes do crítico, o multiplicador a ampliaria, o que contradiz "valor fixo por nível" | golpe crítico em alvo com garnet |
| D-05 | Nível mínimo da 10ª skill | 201 (o da 9ª) | a 10ª exige todas as anteriores; o piso é consequência, não número novo | NPC Skillmaster recusando abaixo de 201 |
| D-06 | Grandeza do crítico de item (`7.0`) | décimos de ponto percentual (`700` bps) | preserva o decimal exibido sem inventar curva | tooltip do item × log de crítico |
| D-07 | Dois efeitos com o mesmo `affect_type` empilham? | **não**: substitui magnitude, renova para a maior duração | empilhar exigiria regra de teto por efeito, que a fonte não dá; "usar outra skill desativa" sugere substituição | sequência de dois buffs iguais no cliente |
| D-08 | Teto do bônus de drop | saturado em `1.000.000` (topo da escala do roll) | a fonte manda aplicar bônus "com teto"; o único teto definido é o da própria escala | evento de drop de guilda no cliente |
| D-09 | Regra de ameaça (aggro) | ameaça = dano; alvo = maior ameaça, empate pelo **menor `entity`** | é a política mínima que produz decisão determinística; decaimento/taunt ficam como TTL de entrada | comportamento de dois atacantes no cliente |
| D-10 | Arredondamento de percentual | **truncamento (piso)** | é a divisão inteira do legado; "mais próximo" criaria divergência de 1 ponto em 90% | dano de 99 com 33% de resistência |
| D-11 | Elementos existentes | `Physical`, `Fire`, `Frost` (os que a fonte nomeia) | enum exaustivo obriga revisão quando o catálogo provar os demais; inventar 6 elementos criaria tipo sem dado | `item_effect`/`EF_*` + trace de resistência |
| D-12 | A gema é consumida ao ser aplicada? | **sim**, uma unidade da mochila por aplicação (`GemNotInBag` quando não há) | a fonte descreve "jogar/arrastar a gema no item": gema é item aplicável, não ação gratuita — sem consumo o estoque seria infinito | inventário do cliente antes/depois de aplicar |
| D-13 | Reaplicar a mesma gema é "já aplicada"? | **não**: troca permitida, consome outra unidade; "esgotada" = sem estoque na mochila | `refino.md` §3 diz que o efeito **não é definitivo** e pode ser trocado quantas vezes quiser — recusar por "já aplicada" contradiria a fonte | aplicar a mesma gema duas vezes seguidas no cliente |

## 5. Como provar

| Nível | O que valida | Onde |
|---|---|---|
| unidade | faixa, borda, erro tipado, saturação | `#[cfg(test)]` em cada módulo |
| golden | **valor exato** de tabela publicada, com a fonte no nome | `tests/goldens.rs` (refino, gemas, absorção) |
| composição | sistemas conversando na mesma unidade | `formula/mod.rs::tests::systems_compose_with_the_same_units` |
| determinismo | mesma seed → mesma partida; rolls gravados → mesma partida | `tests/determinism.rs` (6 testes) |
| relatório | saída comparável estável | `report/table.rs` (alinhamento determinístico) |

`tests/goldens.rs` mantém **as mesmas asserções** de antes da reorganização: a mudança foi de
estrutura, não de expectativa de jogo. Golden novo entra com a fonte citada no nome
(`gem_scale_matches_kersef`), como manda `docs/rust/06-matematica.md` §5.

## 6. O que está implementado, o que é plano, o que diverge do planejamento

**Implementado nesta entrega:** os 9 sistemas da missão (atributos, progressão, dano,
acerto/crítico, efeitos, resistência, drop, regeneração, aggro) + refino existente, ordem do
tick, sorteio determinístico, relatório e testes de determinismo.

**Plano, com dono:** `formula/speed.rs` (FASE-10-2, ⛔ até provar a anomalia de ~200%),
`bot/{persona,policy}.rs` (FASE-15-3), laboratório em milhões de iterações com CLI
(FASE-10-3 — o crate já é biblioteca pura, falta o *harness* de lote).

**Divergências registradas, não silenciadas:**

| Fonte | O que ela diz | O que está no repositório |
|---|---|---|
| `pre-release-rust.md` §10.1 | "documentar e codificar as equações **exatas**" | a composição está codificada; a ordem exata é hipótese (D-02..D-04) porque `docs/wyd/skills.md` §5 a marca UNVERIFIED |
| `pre-release-rust.md` §10.2 | investigar velocidade ~200% | nada implementado: módulo marcado ⛔ para ninguém "resolver" por chute |
| `pre-release-rust.md` §10.3 | simular milhões de interações | funções puras prontas e testadas; falta o *harness* de lote/CLI (FASE-10-3) |
| `start-rust.md` | "não escrever código até tudo documentado" | cumprido nesta ordem: ADR-0011 + esta página antes do código; o que a fonte não prova ficou como parâmetro |
| `estructura-e-padroes.md` §14 (antes) | `formula/{damage,refine,speed,crit}.rs` | árvore substituída por sistema (§2) — `crit` virou `hit`, `speed` segue bloqueado |

## 7. Lacunas de insumo (não de código)

| Lacuna | Consequência | Onde está registrada |
|---|---|---|
| Curva de EXP do 7662 | `ExpTable` sem conteúdo real; cenários usam curva sintética | `docs/wyd/progressao.md` §7 |
| `mob_template_stat`/`mob_template_equip` vazias | sem stats de criatura, `GrowthProfile` é hipótese | `crates/wyd-db/migrations/README.md` §Lacunas |
| Trace de combate do cliente | D-02..D-04 e D-06 não fecham | `docs/cliente/depuracao.md`, FASE-08-6 🚧 |
| `drop_entry` parcial | drop roda com taxa de fonte secundária | `docs/wyd/sistemas/drops.md` §7 |
