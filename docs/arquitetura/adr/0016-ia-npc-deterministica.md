# ADR-0016 — IA de NPC determinística: intenção, não efeito

- **Estado:** aceita (2026-11-08)
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (fechamento da FASE-03-6)
- **Governa:** `crates/wyd-core/src/npc/`, `crates/wyd-core/src/systems/npc.rs`, FASE-03-6
- **Depende de:** ADR-0001 (Hot Path), ADR-0002 (estrutura), ADR-0004 (ECS SoA),
  ADR-0010 (IA de NPC e bots), ADR-0011 (matemática em `wyd-sim`), ADR-0014 (autoridade),
  ADR-0015 (inventário autoritativo)

## Contexto

A ADR-0010 decidiu **onde** a IA de NPC vive e o que ela não faz (LLM no tick, I/O no
processo do jogo, aprendizado). O `docs/arquitetura/ia-bots.md` fixou o modelo de decisão
(P1 percepção → P2 alvo → P3 intenção), a cadência de 5 Hz e o requisito de determinismo
verificável. Faltava **código**: `crates/wyd-core` não tinha `npc/` nem `systems/npc.rs`,
e `mob_template_stat` está vazia — sem números de monstro não havia como calibrar
agressividade nem dano.

Duas pressões apareceram na entrega:

1. **Não duplicar regra** entre `wyd-sim` (matemática de ameaça) e `wyd-core`. O DAG
   (`estrutura-e-padroes.md` §1) proíbe `wyd-core → wyd-sim`; a ADR-0011 já registrou a
   dívida de unificação que exige contrato em `wyd-protocol`.
2. **Não deixar a IA virar autoridade paralela.** Se a FSM movesse a criatura diretamente,
   existiriam dois caminhos de escrita de posição/HP — um validado (`systems/movement.rs`,
   `systems/combat.rs`) e outro não.

## Decisão

1. **A IA emite intenção, nunca efeito.** `systems/npc::decide` devolve
   `Option<Decision { mode, target, intention }>`; `Intention` é `Idle | MoveTo | Attack |
   Flee | Return`. Quem move é `movement::step` e quem fere é `combat::attack`, com as
   mesmas validações de um pedido de cliente. `tests/integracao/npc.rs` prova que uma intenção forjada
   (destino além do orçamento) é recusada com posição e fila intactas.
2. **Decisão é função pura e determinística.** `npc::brain::decide(&mut NpcState, Observed,
   &[Observed], tick, NpcProfile) -> Decision` não lê relógio, banco nem `HashMap`; RNG é
   `xorshift64` semeado por `(world_seed, entity_index)` e guardado no `NpcState` (D-IA04).
   Nada de `rand`, nada de `f32`.
3. **Cadência de 5 Hz com fase por entidade** — `tick % 4 == entity.index % 4`
   (D-IA01), mais `DecisionBudget` com teto por tick. Decidir fora da cadência devolve
   `Ok(None)` **sem** criar estado nem consumir orçamento.
4. **Perfil é dado do servidor, não constante da engine.** `NpcProfile { aggro_range,
   leash_range, attack_range, flee_hp_bps, patrol_radius }` é validado no boot
   (`InvalidNpcProfile`) e chega por parâmetro; enquanto `mob_template_stat` estiver vazia,
   o chamador informa o perfil — nenhum número de balanceamento é inventado no core.
5. **Cópia presa por golden da tabela de ameaça.** `npc::state::ThreatTable` (teto 8, empate
   pelo menor `entity`, TTL em ticks) reimplementa a semântica de
   `crates/wyd-sim/src/formula/threat.rs` **porque o DAG proíbe a dependência**. A dívida de
   unificação continua registrada (`06-matematica.md` §4, FASE-10-1) e `tests/integracao/npc.rs` fixa a
   decisão por hash FNV-1a: mudança de comportamento quebra o golden.
6. **Rota ancorada, não inventada.** `npc::route::Route` limita a patrulha ao raio da âncora
   até o trace do cliente confirmar `route_type`/`generator_data` (UNVERIFIED). Nenhum
   waypoint especulativo é criado.

## Consequências

- A FSM tem sete estados (`Idle`, `Patrol`, `Aggro`, `Chase`, `Attack`, `Flee`, `Return`) com
  precedência explícita em `npc/behavior.rs`: fuga vence tudo, leash vence perseguição,
  ataque exige alvo no alcance. Cada transição tem teste unitário.
- Recusas novas e tipadas: `NotACreature`, `ThreatTableFull`, `InvalidNpcProfile`. Entrada de
  rede/dado inválido continua sem `panic!`.
- `ComponentStore::kind_of` nasceu como leitura (IA e visibilidade); não decide regra.
- O custo da IA **está medido** em `crates/wyd-core/benches/tick.rs` (ver adendo): entre
  88,6 e 204,5 µs/tick com 500 NPCs + 50 jogadores em cinco execuções (0,18%–0,41% do
  orçamento de 50 ms a 20 Hz; pior p99 0,74%), 0,71–1,64 µs por decisão; sem alvos,
  12,5–31,8 µs/tick (0,025%–0,064%).

## Alternativas descartadas

| Alternativa | Por que não |
|---|---|
| Fazer `wyd-core` depender de `wyd-sim` para reusar `threat.rs` | fura o DAG (§1, regra 1) e cria caminho de dependência que a ADR-0011 negou; a unificação pertence à FASE-10-1 |
| Mover a FSM para `wyd-sim` e chamá-la do core | inverteria a direção: a simulação descreveria o jogo em vez de medir cenários |
| IA escrevendo posição/HP direto no `ComponentStore` | criaria segundo caminho de escrita sem as validações de `step`/`attack` (ADR-0014) |
| `HashMap<Entity, NpcState>` | alocação e hashing no quente; o estado é `Vec` contíguo indexado por `Entity.index` |
| Constantes de alcance/HP dentro de `npc/` | balanceamento escondido na engine; o perfil vem do conteúdo (ADR-0009) |

## UNVERIFIED que permanece

1. `mob_template_stat`/`mob_template_equip` vazias — o perfil é parâmetro do chamador, não
   derivado do banco ainda (FASE-03-4/04-3).
2. `route_type`/`generator_data` sem trace — patrulha limitada à âncora.
3. Custo por tick em hardware de produção (o número medido é do host de desenvolvimento,
   build release); o teto de 133 decisões/tick não precisa mudar com a folga mínima de ~13×.
4. Ameaça por dano em proporção 1:1 com teto 8 (D-IA02) sem confirmação por captura.
5. Tabelas `npc_behavior`/`npc_dialogue`/`llm_job` não existem — schema só entra quando houver
   código consultando o dado.

## Adendo (2026-11-08) — custo medido, critério de aceite fechado

`crates/wyd-core/benches/tick.rs` (`[[bench]] name = "tick", harness = false`, release, sem
dependência nova) instancia **500 NPCs vivos + 50 jogadores** no `ComponentStore` e roda a cadeia
real do tick por 400 ticks medidos (50 de aquecimento), reportando tempo por tick, p50/p99/pior,
custo por decisão e % do orçamento de 50 ms (20 Hz).

Cinco execuções na mesma árvore (release, host Windows de desenvolvimento) — a variação é
ruído de máquina compartilhada, não do subsistema:

| Cenário | Média/tick | p99/tick | % do tick (média) | % do tick (p99) | Por decisão |
|---|---|---|---|---|---|
| 500 NPCs + 50 jogadores | 88,6 – 204,5 µs | 119,4 – 367,8 µs | 0,18% – 0,41% | 0,24% – 0,74% | 0,71 – 1,64 µs |
| 500 NPCs sem jogador (piso de percepção) | 12,5 – 31,8 µs | 14,8 – 80,3 µs | 0,025% – 0,064% | 0,030% – 0,161% | 0,10 – 0,25 µs |

50.000 decisões em 400 ticks (125/tick, exatamente a cadência de 5 Hz). O bench **sai com
código 1** se a média ou o p99 passarem de 10% — a régua é executada, não escrita.

**Consequência:** o critério "IA ≤ 10% do orçamento do tick com 500 NPCs" deixa de ser
UNVERIFIED — o pior caso observado (p99 0,74%) ainda tem folga de ~13×. Cadência de 5 Hz e
`DecisionBudget` de 133 decisões/tick para 500 criaturas **não** foram afrouxados: o teto existe
para mundo maior/carga hostil. O custo é dominado pela percepção (com 50 jogadores o tick custa
6–9× o piso sem alvo), não pela tabela de ameaça.

Continua UNVERIFIED: medição em hardware de servidor (o número é do host de desenvolvimento,
Windows/release) e o custo de `step`/`attack` aplicados no mesmo tick (o bench mede a decisão;
movimento e combate têm gates próprios em `tests/integracao/autoridade.rs`).
