# ADR-0012 — Aplicação de gema é transação do inventário do core, com gema consumida e evento único

- **Estado:** aceita (2026-11-08)
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (missão "ligar `apply_gem` ao inventário real")
- **Governa:** `roadmap-rust/changelog.md`, `GAME-19-equacoes.md`
- **Depende de:** ADR-0001 (Hot Path), ADR-0004 (ECS SoA), ADR-0011 (matemática do `wyd-sim`)

## Contexto

A regra da gema já existia em dois lugares: `crates/wyd-sim/src/formula/refine.rs` (tabela) e
`crates/wyd-core/src/ecs/components.rs` (autoridade). No core, `Inventory::apply_gem` alterava
o item e `ComponentStore::apply_gem` publicava `DomainEvent::GemEffectChanged` — mas **nada era
consumido**: a gema continuava na mochila e podia ser aplicada infinitas vezes. Também não
havia recusa para "sem estoque", e a validação se misturava com a escrita dentro do mesmo
método, o que tornava frágil a garantia de "recusa sem mutação parcial".

A fonte (`docs/wyd/sistemas/refino.md` §3) diz que a gema é aplicada **sobre o item**
(jogar/arrastar a gema no item), vale para Anct e +10+, e que o efeito **não é definitivo** —
pode ser trocado quantas vezes quiser, desde que o item esteja no nível exigido.

## Decisão

Adotamos **`ComponentStore::apply_gem` como caminho único** de aplicação de gema, com a
transação em ordem explícita — validar sem escrever, consumir a gema da mochila, publicar o
evento e só então gravar o efeito —, porque sem consumo a gema é duplicável e sem validação
separada da escrita não há como prometer recusa atômica.

Regras que decorrem dela:

1. `Inventory::plan_gem` valida (slot, item, gema, refino) **sem escrever**;
   `commit_gem` grava; `consume_gem` retira uma unidade e libera a pilha zerada.
2. `plan_gem`, `consume_gem`, `commit_gem` e `apply_gem_from_bag` (que recebe o plano) são
   `pub(crate)`: de fora do crate só existe o caminho com entidade viva, estoque e evento.
   Não há como aplicar gema numa cópia solta.
3. Recusas tipadas: `SlotOutOfRange`, `EmptySlot`, `UnknownGem`, `GemRefineTooLow`,
   `GemRefineTooHigh`, `GemNotInBag`, `EventQueueFull`, `EntityNotAlive`. Nenhuma delas altera
   item, mochila ou fila.
4. Um evento por aplicação bem-sucedida, zero nas recusas; o evento carrega `from`/`to`,
   `slot`, `gem_item` e o refino efetivo.

## Consequências

- **Positivas:** a gema vira item consumível de verdade (economia fecha); a borda recebe o
  delta auditável; a mesma função de tabela continua presa por golden no `wyd-sim`.
- **Negativas:** `Inventory` passa a ter semântica de mochila (não só de equipamento), e o
  chamador precisa ter a gema na mochila para testar a regra — fixtures antigas ganharam
  estoque, sem mudar nenhuma expectativa de jogo.
- **Invariantes afetados:** nenhuma escrita antes de todas as validações; evento nunca é
  `println` nem efeito colateral de retorno; `qty = 0` não pode sobrar na mochila.

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| Aplicar sem consumir (como antes) | gema infinita; contradiz "gema aplicada sobre o item" e quebra a economia |
| Consumir direto dentro de `Inventory::apply_gem` | impede testar a regra do item isolada e mistura validação com escrita |
| Recusar "gema já aplicada" | contradiz `refino.md` §3 ("o efeito não é definitivo"); a troca é permitida |
| Aceitar gema fora da mochila (vinda do cliente) | viola autoridade do servidor (invariante 3 do `AGENTS.md`) |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/changelog.md`, `GAME-19-equacoes.md`.
- Docs atualizadas: `docs/rust/03-progresso.md`, `docs/rust/06-matematica.md` §4,
  `docs/arquitetura/matematica-sim.md` §2/§4 (D-12, D-13).
- Evidência: `crates/wyd-core/tests/integracao/gem.rs` (6 testes de API pública),
  `crates/wyd-core/tests/golden/gema_seed.rs` (4 testes contra o seed real do banco) +
  `crates/wyd-core/src/ecs/components.rs` (`gem_tests`, 11 testes).

## Prova contra o dado real (2026-11-08)

A política do core não pode divergir do catálogo: `refine_gem_effect` é o dado que o servidor
publica, e `gem_effect_code`/`gem_value` são a autoridade do Hot Path. `gema_seed.rs` fecha essa
lacuna lendo **a própria migração** (`crates/wyd-db/migrations/0022_seed_refino.sql`) por
`include_str!` e exigindo igualdade em toda a faixa publicada:

| O que prova | Como |
|---|---|
| As quatro gemas do seed são as quatro do core | `gem_effect_code(item_index)` por linha do `INSERT` |
| O valor bate em **todos** os refinos 10..=15 | `gem_value` × `flat_value`/`per_level_value` derivados do seed |
| Abaixo do mínimo não há valor inventado | `gem_value(code, min_refine - 1) == None` |
| Anct difere do comum com o dado real | no mesmo `min_refine - 1`, Anct aplica o valor do seed e o comum recebe `GemRefineTooLow` |
| Troca de efeito é consistente | item já gemedo recebe o novo efeito e o evento carrega `from`/`to` com os valores do seed |

A fronteira do DAG é preservada: a leitura acontece em tempo de compilação, dentro de `tests/`,
sem `wyd-core` depender de `wyd-db` e sem I/O em runtime. Um `effect_code` ou `per_level_value`
trocado no seed quebra o gate de teste, não a produção.

## Revisão

- Revisar quando: o cliente provar o consumo por trace (D-12), a composição Anct (+10/+12..15)
  entrar na FASE-10 e a regra de gema precisar de contrato em `wyd-protocol`.
- Sinais de que envelheceu: gema aparecendo duplicada em auditoria, evento publicado sem
  mudança de item, item com `qty = 0` na mochila.
