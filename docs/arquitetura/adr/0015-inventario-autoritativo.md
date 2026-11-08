# ADR-0015 — Inventário: intenção mínima, slot derivado do catálogo

- **Estado:** aceita (2026-11-08)
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (fechamento do eixo inventário da FASE-03-5)
- **Governa:** `crates/wyd-core/src/systems/inventory.rs`, FASE-03-5
- **Depende de:** ADR-0001 (Hot Path), ADR-0004 (ECS SoA), ADR-0009 (conteúdo no banco),
  ADR-0012 (gema), ADR-0014 (autoridade)

## Contexto

Depois da ADR-0014, movimento, combate, skill e aplicação de gema tinham validação real, mas
**equipar/desequipar/mover item não tinha sistema**: um pacote de troca de item não tinha onde
ser recusado, e nada impedia o cliente de dizer em que slot o item entrava. O `Inventory` do
core só sabia aplicar gema (ADR-0012); o resto era array público sem operação.

O dado de item já existe no banco e não precisa de tabela nova: `item_catalog` traz
`req_level`/`req_str/int/dex/con` e `kind`; `item_kind` mapeia `kind → slot` (`weapon` → mão,
`helmet` → cabeça, `consumable`/`material` → mochila, …) em `docs/wyd/itens.md` §5.

## Decisão

Adotamos **`systems/inventory.rs`** com uma **porta [`ItemRules`]** implementada pelo servidor a
partir de `item_catalog` + `item_kind`, porque assim o core valida sem conhecer `kind` nem
duplicar catálogo:

```rust
pub trait ItemRules {
    fn equip_slot(&self, item_index: u32) -> Option<u8>; // None = não equipável
    fn req_level(&self, item_index: u32) -> u16;
    fn max_stack(&self, item_index: u32) -> u32;
}
```

Regras que decorrem dela:

1. `equip(store, entity, bag_slot, rules, policy)` recebe **só o slot da mochila**: o slot de
   equipamento é derivado do catálogo. `equip_into(...)` existe apenas para quando o protocolo
   trouxer um slot declarado — e ele é **conferido** (`WrongEquipSlot`), nunca obedecido.
2. Requisito de nível vem de `item_catalog.req_level` comparado ao `Stats.level` autoritativo da
   entidade — o cliente não manda nível.
3. Política de troca é explícita (`EquipmentRules { allow_swap }`): com troca, o item anterior
   volta para o slot da mochila de onde o novo saiu; sem troca, slot ocupado é
   `EquipSlotOccupied`. Não existe "sobrescrever e perder".
4. `unequip` escolhe o **primeiro** slot livre da mochila; mochila cheia é `BagFull`, nunca
   descarte nem overwrite.
5. `move_item` aceita origem/destino de mochila (é o que o cliente sabe), mas decide o resultado:
   destino vazio move inteiro; mesmo item empilha até `max_stack` (o excedente fica na origem);
   item diferente é `BagSlotOccupied` — sem troca silenciosa entre slots.
6. Ordem transacional igual aos demais sistemas: viva → pré-condições → fila → evento
   (`ItemEquipped`/`ItemUnequipped`/`ItemMoved`) → mutação. Recusa não muda inventário,
   equipamento nem fila.

## Consequências

- **Positivas:** o eixo inventário fecha com um único ponto de recusa; o teste de pacote forjado
  passa a cobrir duplicação/perda de item (ciclo equipar → trocar → desequipar → reequipar com
  contagem preservada); nenhum bônus é calculado no core — o item equipado é dado, o efeito é da
  FASE-10.
- **Negativas:** o servidor precisa implementar `ItemRules` a partir do catálogo (uma consulta no
  boot/cache), e `req_str/int/dex/con` ainda não entram na validação — declarado como pendência
  (exigem atributos derivados, hoje fora do `Stats`).
- **Invariantes afetados:** nenhum `unsafe`; sem I/O no tick; arrays fixos preservados
  (`EQUIP_SLOTS = 16`, `BAG_SLOTS = 64`); evento só no aceite.

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| Colocar regra em `ecs/components.rs` | `Inventory` é dado (DOD); regra com pré-condições pertence a `systems/` |
| Ler o banco dentro do core | viola ADR-0001/ADR-0009 (Hot Path sem I/O); porta `ItemRules` resolve |
| Criar tabela/tipos novos de slot | o dado já existe (`item_kind.slot`, `item_catalog.kind`); nova tabela seria órfã |
| Aceitar slot final do cliente | cliente viraria autor do layout; `equip_into` confere contra o catálogo |
| Trocar itens entre slots da mochila | comportamento não provado no cliente 7662; silêncio ali vira dupe |
| Deixar atributos (`req_str`…) sem validar e não declarar | meia-verdade: fica registrado como pendência explícita |

## Rastreabilidade

- Issue consumidora: `roadmap-rust/issues/FASE-03-5-autoridade-servidor.md`.
- Docs: `docs/arquitetura/autoridade-servidor.md`, `estrutura-e-padroes.md` §10,
  `docs/rust/03-progresso.md` (entrada datada).
- Evidência: `crates/wyd-core/tests/inventario.rs` (10 testes) +
  `systems/inventory.rs` (7 testes unitários).

## Revisão

- Revisar quando: o protocolo 7662 provar o layout de `MSG_ItemMove`/equipar (FASE-08-6), os
  atributos `req_str/int/dex/con` entrarem no `Stats` (FASE-10), ou o baú/`vault_items` ganhar
  operação (fora do core hoje).
- Sinais de que envelheceu: cliente escolhendo slot de equipamento, item desaparecendo em troca,
  `max_stack` fixo no core, validação de atributo declarada e nunca implementada.
