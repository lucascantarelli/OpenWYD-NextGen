# Autoridade do servidor — o que o cliente pode e o que ele nunca decide

> FASE-03-5. Implementação: `crates/wyd-core/src/systems/` (ADR-0014). Regra de ouro: a borda de
> rede entrega **intenção** (destino, alvo, slot, gema); o estado que vale é o que o core grava.

## 1. Ações e onde cada uma é validada

| Ação pedida | O que o cliente manda | O que o servidor decide | Onde |
|---|---|---|---|
| Andar | destino (`x`, `y`) | se o passo cabe no orçamento de tempo, se o destino existe no mapa e a posição gravada | `systems/movement.rs::step` |
| Atacar | id do alvo | alcance, cooldown, alvo vivo, alvo ≠ autor, **dano** (via `DamageModel`) e HP restante | `systems/combat.rs::attack` |
| Aplicar gema | slot do item + `item_index` da gema | se o item existe, se é gema conhecida, faixa de refino (Anct), estoque na mochila, efeito e valor | `ecs/components.rs::ComponentStore::apply_gem` (ADR-0012) |
| Usar skill | id da skill (+ alvo opcional) | skill no catálogo, skill aprendida, MP, cooldown e alcance; o **efeito** é da FASE-10 | `systems/skill.rs::cast` |
| Equipar | slot da mochila (+ slot declarado, se o protocolo trouxer) | slot derivado do catálogo (`item_kind`), requisito de nível, política de troca, destino do item anterior | `systems/inventory.rs::equip` / `equip_into` |
| Desequipar | slot de equipamento | se há item, primeiro slot livre da mochila (mochila cheia recusa) | `systems/inventory.rs::unequip` |
| Mover / empilhar na mochila | origem e destino | destino vazio × mesmo item (empilha até `max_stack`) × item diferente (recusa); nunca troca silenciosa | `systems/inventory.rs::move_item` |

Nada nesta tabela aceita "resultado confirmado": não existe parâmetro de dano, de posição final
nem de efeito aplicado. Se o cliente mandar o número, o campo não tem onde entrar.

## 2. Garantia de recusa atômica

Os três caminhos implementados seguem a mesma ordem:

```
entidade viva → pré-condições (orçamento/limites/alcance/cooldown/estoque)
→ espaço na fila de eventos → publica evento → grava o efeito
```

Consequência verificável: depois de uma recusa, posição, HP, cooldown, item e mochila estão
**idênticos** e a fila de eventos está vazia (`crates/wyd-core/tests/integracao/autoridade.rs`). Evento é
registro de aceite, não de tentativa.

## 3. De onde vêm os números (e por que não do core)

| Número | Fonte | Situação |
|---|---|---|
| Velocidade (tiles/s), retângulo do mundo | `MoveRules`, montado na configuração do mundo | explícito; valor de jogo a fixar na FASE-10 |
| Alcance e cooldown de ataque | `AttackRules` | explícito; idem |
| Dano | `DamageModel` (trait), implementado pelas equações aprovadas | FASE-10 — hoje só existe modelo de teste |
| Efeito/valor de gema | `refine_gem_effect` (seed `0022`) | **CONFIRMADO** por paridade core × seed (`tests/golden/gema_seed.rs`) |
| Slot de equipamento do item | `item_kind.slot` + `item_catalog.kind` (via porta `ItemRules`) | **CONFIRMADO** pela taxonomia (`0017`) + `docs/wyd/itens.md` §5 |
| Requisito de nível do item | `item_catalog.req_level` (via porta `ItemRules`) | **CONFIRMADO** pelo dado do legado |
| Limite de pilha | `ItemRules::max_stack` | **UNVERIFIED** — o catálogo só tem `stackable` (booleano); o teto numérico segue sem fonte |

O core não esconde constante de balanceamento: quem não fornece regra não roda o sistema. Isso
impede o padrão "número mágico que ninguém sabe de onde veio" e deixa o ajuste de jogo na
FASE-10, onde há trace e laboratório.

## 4. O que ainda é UNVERIFIED

- **Movimento**: o cliente 7662 pode mandar destino (aqui modelado) ou passo por passo; a
  captura da FASE-08-6 decide se o orçamento é por passo ou por destino. O sistema aceita
  destino; trocar a unidade é mudar `MoveRules::budget`, não a validação.
- **Alcance**: `AttackRules` é hipótese até o trace de um ataque real (arco × espada).
- **Atributos de item**: `req_str/int/dex/con` existem em `item_catalog` mas o `Stats` do core
  ainda não carrega os quatro; hoje só o nível é validado. Declarado, não fingido.
- **Layout de rede do inventário**: qual pacote pede equipar/mover (`MSG_*`) segue `UNVERIFIED`
  até a captura da FASE-08-6; o core valida a intenção (item/slots), não o layout.

## 5. Onde testar

```bash
cargo test -p wyd-core                      # unitários de cada sistema + ECS
cargo test -p wyd-core --test integracao_autoridade    # pacote forjado recusado sem mutação
cargo test -p wyd-core --test integracao_inventario    # equipar/desequipar/mover sem dupe nem perda
```
