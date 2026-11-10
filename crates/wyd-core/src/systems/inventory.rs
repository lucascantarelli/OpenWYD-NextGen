//! Inventário e equipamento server-authoritative (FASE-03-5, eixo inventário).
//!
//! A intenção aceita é mínima: **qual** item da mochila e, quando o protocolo pedir slot, qual
//! slot o cliente *diz* querer. O servidor decide o resto — slot de equipamento derivado do
//! catálogo, requisito de nível, política de troca, destino do item desequipado, limite de
//! pilha e, no loot, **o que não cabe no inventário**. Nenhuma assinatura recebe bônus, item
//! resultante ou "slot final confirmado".
//!
//! Regras vêm de [`ItemRules`], implementado pelo servidor a partir de `item_catalog` +
//! `item_kind` (ver `docs/wyd/itens.md` §5): o core não fixa requisito nem mapa de slot.
//!
//! Toda operação segue a ordem dos demais sistemas: entidade viva → pré-condições completas →
//! espaço na fila → evento → mutação. Recusa não muda item, equipamento, mochila nem fila.

use crate::ecs::components::{ComponentStore, Entity, Item, BAG_SLOTS, EQUIP_SLOTS};
use crate::ecs::resources::DomainEvent;
use crate::error::CoreError;

/// Regras de item do servidor (ponte para o catálogo do banco).
///
/// `equip_slot` devolve `None` para item não equipável (material, consumível, quest…), o que
/// mantém o core ignorante de `kind`: o mapeamento `kind → slot` é dado, não regra da engine.
pub trait ItemRules {
    /// Slot de equipamento do item (`None` = não equipável).
    fn equip_slot(&self, item_index: u32) -> Option<u8>;
    /// Nível mínimo para equipar.
    fn req_level(&self, item_index: u32) -> u16;
    /// Pilha máxima permitida (`1` = não empilha).
    fn max_stack(&self, item_index: u32) -> u32;
}

/// Política de troca quando o slot de equipamento já está ocupado.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct EquipmentRules {
    /// Permite trocar: o item que estava no slot volta para a mochila, no lugar do novo.
    pub allow_swap: bool,
}

impl Default for EquipmentRules {
    fn default() -> Self {
        Self { allow_swap: true }
    }
}

/// Resultado de um equipamento aceito.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Equipped {
    /// Slot de equipamento que passou a valer (derivado do catálogo).
    pub slot: u8,
    /// Slot da mochila de onde o item saiu.
    pub bag_slot: u8,
    /// Item equipado (com refino/Anct/gema que já tinha).
    pub item: Item,
    /// Item que voltou para a mochila, quando houve troca.
    pub swapped_out: Option<Item>,
}

/// Resultado de um desequipamento aceito.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Unequipped {
    /// Slot de equipamento que ficou vazio.
    pub slot: u8,
    /// Slot da mochila que recebeu o item (escolhido pelo servidor).
    pub bag_slot: u8,
    /// Item devolvido.
    pub item: Item,
}

fn item_do_slot(
    store: &ComponentStore,
    entity: Entity,
    bag_slot: usize,
) -> Result<Item, CoreError> {
    if bag_slot >= BAG_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: bag_slot as u16,
            max: (BAG_SLOTS - 1) as u16,
        });
    }
    let inv = store.inventory(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    let item = inv.bag[bag_slot];
    if item.id == 0 {
        return Err(CoreError::EmptySlot {
            slot: bag_slot as u16,
        });
    }
    Ok(item)
}

/// Equipa o item `bag_slot` no slot que o **catálogo** manda.
///
/// Caminho de jogo: o cliente manda o item, nunca o slot. Use [`equip_into`] só quando o
/// protocolo trouxer um slot declarado — e ele será conferido contra o catálogo, não obedecido.
pub fn equip<R: ItemRules>(
    store: &mut ComponentStore,
    entity: Entity,
    bag_slot: usize,
    rules: &R,
    policy: EquipmentRules,
) -> Result<Equipped, CoreError> {
    let item = item_do_slot(store, entity, bag_slot)?;
    let slot = rules.equip_slot(item.id).ok_or(CoreError::NotEquippable {
        item_index: item.id,
    })?;
    equip_into(store, entity, bag_slot, slot, rules, policy)
}

/// Equipa conferindo o slot **pedido** contra o slot do catálogo.
///
/// Slot divergente é recusa tipada (`WrongEquipSlot`): o cliente não escolhe onde o item entra.
pub fn equip_into<R: ItemRules>(
    store: &mut ComponentStore,
    entity: Entity,
    bag_slot: usize,
    requested_slot: u8,
    rules: &R,
    policy: EquipmentRules,
) -> Result<Equipped, CoreError> {
    let item = item_do_slot(store, entity, bag_slot)?;
    let expected = rules.equip_slot(item.id).ok_or(CoreError::NotEquippable {
        item_index: item.id,
    })?;
    if usize::from(requested_slot) >= EQUIP_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: u16::from(requested_slot),
            max: (EQUIP_SLOTS - 1) as u16,
        });
    }
    if requested_slot != expected {
        return Err(CoreError::WrongEquipSlot {
            expected,
            requested: requested_slot,
        });
    }
    let stats = store.stats_of(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    let required = rules.req_level(item.id);
    if stats.level < required {
        return Err(CoreError::LevelTooLow {
            required,
            current: stats.level,
        });
    }
    let inv = store
        .inventory(entity)
        .expect("entidade viva tem inventário");
    let ocupado = inv.equip[usize::from(expected)];
    if ocupado.id != 0 && !policy.allow_swap {
        return Err(CoreError::EquipSlotOccupied { slot: expected });
    }
    let swapped_out = (ocupado.id != 0).then_some(ocupado);

    store.publish(DomainEvent::ItemEquipped {
        entity,
        bag_slot: bag_slot as u8,
        equip_slot: expected,
        item_index: item.id,
    })?;
    let inv = store
        .inventory_mut(entity)
        .expect("entidade viva tem inventário");
    inv.equip[usize::from(expected)] = item;
    inv.bag[bag_slot] = swapped_out.unwrap_or_default();
    Ok(Equipped {
        slot: expected,
        bag_slot: bag_slot as u8,
        item,
        swapped_out,
    })
}

/// Desequipa e devolve o item ao **primeiro** slot livre da mochila (escolha do servidor).
///
/// Mochila cheia é recusa (`BagFull`): nada é destruído nem sobrescrito para "abrir espaço".
pub fn unequip(
    store: &mut ComponentStore,
    entity: Entity,
    slot: u8,
) -> Result<Unequipped, CoreError> {
    if usize::from(slot) >= EQUIP_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: u16::from(slot),
            max: (EQUIP_SLOTS - 1) as u16,
        });
    }
    let inv = store.inventory(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    let item = inv.equip[usize::from(slot)];
    if item.id == 0 {
        return Err(CoreError::EmptySlot {
            slot: u16::from(slot),
        });
    }
    let bag_slot = inv
        .bag
        .iter()
        .position(|bag| bag.id == 0)
        .ok_or(CoreError::BagFull {
            slots: BAG_SLOTS as u16,
        })?;

    store.publish(DomainEvent::ItemUnequipped {
        entity,
        equip_slot: slot,
        bag_slot: bag_slot as u8,
        item_index: item.id,
    })?;
    let inv = store
        .inventory_mut(entity)
        .expect("entidade viva tem inventário");
    inv.equip[usize::from(slot)] = Item::default();
    inv.bag[bag_slot] = item;
    Ok(Unequipped {
        slot,
        bag_slot: bag_slot as u8,
        item,
    })
}

/// Move (ou empilha) item entre slots da mochila.
///
/// Destino vazio recebe o item inteiro; destino com o **mesmo** item empilhável recebe até o
/// limite do catálogo (`max_stack`), e o que não couber fica na origem. Item diferente no
/// destino é recusa (`BagSlotOccupied`) — nada de troca silenciosa entre slots.
pub fn move_item<R: ItemRules>(
    store: &mut ComponentStore,
    entity: Entity,
    from: usize,
    to: usize,
    rules: &R,
) -> Result<u32, CoreError> {
    if from >= BAG_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: from as u16,
            max: (BAG_SLOTS - 1) as u16,
        });
    }
    if to >= BAG_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: to as u16,
            max: (BAG_SLOTS - 1) as u16,
        });
    }
    if from == to {
        return Err(CoreError::SameSlot { slot: from as u16 });
    }
    let origem = item_do_slot(store, entity, from)?;
    let inv = store
        .inventory(entity)
        .expect("entidade viva tem inventário");
    let destino = inv.bag[to];
    let movidos = if destino.id == 0 {
        origem.qty
    } else if destino.id == origem.id {
        let limite = rules.max_stack(origem.id).max(1);
        let espaco = limite.saturating_sub(destino.qty);
        if espaco == 0 {
            return Err(CoreError::StackFull { slot: to as u16 });
        }
        origem.qty.min(espaco)
    } else {
        return Err(CoreError::BagSlotOccupied { slot: to as u16 });
    };

    store.publish(DomainEvent::ItemMoved {
        entity,
        from: from as u8,
        to: to as u8,
        item_index: origem.id,
        qty: movidos,
    })?;
    let inv = store
        .inventory_mut(entity)
        .expect("entidade viva tem inventário");
    inv.bag[from].qty -= movidos;
    if inv.bag[from].qty == 0 {
        inv.bag[from] = Item::default();
    }
    if inv.bag[to].id == 0 {
        inv.bag[to] = Item {
            qty: movidos,
            ..origem
        };
    } else {
        inv.bag[to].qty += movidos;
    }
    Ok(movidos)
}

/// Resultado de depositar loot no inventário.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Deposited {
    /// Quanto entrou no inventário.
    pub stored: u32,
    /// Quanto **não** coube e vai para o chão (a caixa de `GroundLoot`).
    pub leftover: u32,
}

/// Deposita loot: **completa pilha existente antes de abrir slot novo**.
///
/// Regra do drop do servidor (`docs/wyd/sistemas/drops.md` §8): o que cai de um monstro vai
/// direto para o inventário e só o excedente vira caixa no chão. A capacidade é **por item**
/// ([`ItemRules::max_stack`], o `EF_AMOUNT` do template): 120 poções, 100 moedas, 10 por pack
/// de poeira — nunca um teto global.
///
/// Item vazio (`id == 0`) ou `qty == 0` não é erro: devolve tudo como sobra, porque quem chama
/// já decidiu que não há o que guardar. Não publica evento: a auditoria do loot é do abate
/// (`event=loot` no `cpsock`), que é quem persiste o resultado.
pub fn deposit<R: ItemRules>(
    store: &mut ComponentStore,
    entity: Entity,
    item: Item,
    qty: u32,
    rules: &R,
) -> Result<Deposited, CoreError> {
    store.inventory(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    if item.id == 0 || qty == 0 {
        return Ok(Deposited {
            stored: 0,
            leftover: qty,
        });
    }
    let inv = store
        .inventory_mut(entity)
        .expect("entidade viva tem inventário");
    Ok(deposit_into_bag(&mut inv.bag, item, qty, rules))
}

/// Mesma regra de [`deposit`], sobre uma fatia de mochila — é ela que o drop do servidor usa.
///
/// O inventário do jogador em jogo é persistido por linhas (`character_items`, slots 16..79),
/// não pelo `Inventory` do ECS: o `cpsock` monta a fatia a partir das linhas, chama esta
/// função e grava de volta. Sem esta porta o depósito só existiria em teste.
pub fn deposit_into_bag<R: ItemRules>(
    bag: &mut [Item],
    item: Item,
    qty: u32,
    rules: &R,
) -> Deposited {
    if item.id == 0 || qty == 0 {
        return Deposited {
            stored: 0,
            leftover: qty,
        };
    }
    let limite = rules.max_stack(item.id).max(1);
    let mut restante = qty;
    for slot in bag.iter_mut() {
        if restante == 0 {
            break;
        }
        if slot.id == item.id && slot.qty < limite {
            let posto = (limite - slot.qty).min(restante);
            slot.qty += posto;
            restante -= posto;
        }
    }
    for slot in bag.iter_mut() {
        if restante == 0 {
            break;
        }
        if slot.id == 0 {
            let posto = limite.min(restante);
            *slot = Item { qty: posto, ..item };
            restante -= posto;
        }
    }
    Deposited {
        stored: qty - restante,
        leftover: restante,
    }
}

/// Teto de pilha de item **aglutinável**: 120 unidades por slot.
///
/// **Não vem do catálogo.** O `EF_AMOUNT` do template é *conteúdo de pack* (o legado
/// multiplica o preço por ele com o comentário "itens no pack"), e o campo de quantidade é
/// 1 byte — logo 120 é limite de projeto do jogo, não da estrutura. A regra e as fontes estão
/// em `docs/wyd/sistemas/aglutinacao.md` §3; a constante é nossa (`game_setting` pode mudar).
pub const STACK_CAP: u32 = 120;

/// Tetos e conteúdos de pack do catálogo de produção — implementação real de [`ItemRules`].
///
/// Construída uma vez a partir de `item_catalog` (`wyd_db::PgBackend::item_stack_caps`) e lida
/// sem lock no caminho do drop: os mapas são imutáveis depois do boot.
///
/// `caps` é o teto de pilha (`STACK_CAP` para aglutinável, 1 para o resto); `contained` é o
/// `EF_AMOUNT` do template, que é **quantas unidades o pack carrega** — usado ao abrir pack e
/// ao dropar "Poeira x10", nunca como teto de pilha.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct StackCaps {
    caps: std::collections::BTreeMap<u32, u32>,
    contained: std::collections::BTreeMap<u32, u32>,
}

impl StackCaps {
    /// Monta a tabela a partir de `(item_index, stackable, EF_AMOUNT)`.
    ///
    /// `stackable` é `None` quando o catálogo não classificou o item; `amount` é o
    /// `EF_AMOUNT` do template (conteúdo de pack), quando existe.
    #[must_use]
    pub fn from_rows(rows: impl IntoIterator<Item = (u32, Option<bool>, Option<u32>)>) -> Self {
        let mut caps = Self::default();
        for (item_index, stackable, amount) in rows {
            caps.insert(item_index, stackable, amount);
        }
        caps
    }

    /// Registra um item (sobrescreve a entrada anterior).
    pub fn insert(&mut self, item_index: u32, stackable: Option<bool>, amount: Option<u32>) {
        self.insert_cap(item_index, cap_for(stackable), amount);
    }

    /// Monta a tabela a partir do teto **já decidido** no banco (view `item_stack_cap`).
    ///
    /// É o caminho de produção: a classificação (ruído/real/placeholder e a família do item)
    /// vive em SQL, que enxerga o catálogo inteiro; aqui só se copia `(índice, teto, conteúdo)`.
    #[must_use]
    pub fn from_caps(rows: impl IntoIterator<Item = (u32, u32, Option<u32>)>) -> Self {
        let mut caps = Self::default();
        for (item_index, cap, amount) in rows {
            caps.insert_cap(item_index, cap, amount);
        }
        caps
    }

    /// Registra teto explícito (`1` ou [`STACK_CAP`]) e o conteúdo de pack, quando houver.
    ///
    /// O piso 1 é aplicado aqui, não no chamador: teto 0 travaria o depósito (nada entra) e
    /// não existe item que ocupe zero espaço.
    pub fn insert_cap(&mut self, item_index: u32, cap: u32, amount: Option<u32>) {
        if item_index == 0 {
            return;
        }
        if let Some(conteudo) = amount.filter(|valor| *valor > 0) {
            self.contained.insert(item_index, conteudo);
        }
        self.caps.insert(item_index, cap.max(1));
    }

    /// Teto registrado para o item, sem o piso 1 de [`ItemRules::max_stack`].
    #[must_use]
    pub fn cap_of(&self, item_index: u32) -> Option<u32> {
        self.caps.get(&item_index).copied()
    }

    /// Quantas unidades o pack carrega (`EF_AMOUNT` do template), se for embalagem.
    #[must_use]
    pub fn contained_of(&self, item_index: u32) -> Option<u32> {
        self.contained.get(&item_index).copied()
    }

    /// Quantos itens a tabela conhece.
    #[must_use]
    pub fn len(&self) -> usize {
        self.caps.len()
    }

    /// Tabela vazia (nenhum item classificado ⇒ todo item com teto 1).
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.caps.is_empty()
    }
}

/// Teto por item: aglutinável empilha até [`STACK_CAP`]; o resto, 1.
///
/// `stackable` nulo = não sabemos se agrupa: 1 é a escolha conservadora, porque agrupar sem
/// prova pode perder refino/Anct/gema (duas peças "iguais" são instâncias diferentes).
fn cap_for(stackable: Option<bool>) -> u32 {
    if stackable == Some(true) {
        STACK_CAP
    } else {
        1
    }
}

impl ItemRules for StackCaps {
    fn max_stack(&self, item_index: u32) -> u32 {
        self.caps.get(&item_index).copied().unwrap_or(1)
    }

    /// `StackCaps` não carrega slot de equipamento: devolve `None`.
    ///
    /// O tipo existe para o caminho de **drop/empilhamento**. Quem equipa usa a tabela de
    /// equipamento do snapshot; passar esta tabela para `equip` recusa (nunca chuta slot).
    fn equip_slot(&self, _item_index: u32) -> Option<u8> {
        None
    }

    /// Sem dado de requisito, o item nunca é considerável equipável por esta tabela.
    ///
    /// `u16::MAX` em vez de `0`: um zero silencioso liberaria equipar qualquer coisa se a
    /// tabela fosse usada por engano; o topo da faixa recusa.
    fn req_level(&self, _item_index: u32) -> u16 {
        u16::MAX
    }
}

#[cfg(test)]
mod stack_caps_tests {
    use super::*;

    fn vazio() -> Item {
        Item {
            id: 0,
            qty: 0,
            refine: 0,
            ancient: false,
            gem_effect: 0,
            gem_value: 0,
        }
    }

    fn com_id(id: u32) -> Item {
        Item {
            id,
            qty: 1,
            refine: 0,
            ancient: false,
            gem_effect: 0,
            gem_value: 0,
        }
    }

    /// Valores reais medidos no `item_catalog` (2026-11-10).
    fn catalogo() -> StackCaps {
        StackCaps::from_rows([
            // poção 400: `stackable`, sem `EF_AMOUNT`
            (400, Some(true), None),
            // poeira 412/413: `stackable`, sem `EF_AMOUNT`
            (412, Some(true), None),
            (413, Some(true), None),
            // kit 429: `stackable` E conteúdo 10
            (429, Some(true), Some(10)),
            // arma: não agrupa
            (831, Some(false), None),
            // Antídoto 416: sem classificação, conteúdo 5
            (416, None, Some(5)),
            // linha sem dado nenhum
            (9999, None, None),
        ])
    }

    #[test]
    fn teto_e_conteudo_sao_coisas_diferentes() {
        let caps = catalogo();
        assert_eq!(caps.max_stack(400), STACK_CAP, "poção empilha até 120");
        assert_eq!(caps.max_stack(412), STACK_CAP, "poeira empilha até 120");
        assert_eq!(
            caps.max_stack(429),
            STACK_CAP,
            "o kit empilha até 120; 10 é o que vem dentro"
        );
        assert_eq!(caps.contained_of(429), Some(10));
        assert_eq!(caps.contained_of(416), Some(5));
        assert_eq!(caps.contained_of(400), None);
        assert_eq!(caps.max_stack(831), 1, "arma nunca agrupa");
        assert_eq!(caps.max_stack(9999), 1, "sem classificação não agrupa");
        assert_eq!(caps.max_stack(416), 1, "conteúdo não promove a aglutinável");
    }

    #[test]
    fn deposito_completa_pilha_depois_usa_slot_livre() {
        let caps = catalogo();
        let mut bag = [vazio(); 4];
        bag[0] = Item {
            id: 400,
            qty: 100,
            ..vazio()
        };
        let entrou = deposit_into_bag(&mut bag, com_id(400), 50, &caps);
        assert_eq!(entrou.stored, 50, "20 na pilha + 30 no slot livre");
        assert_eq!(entrou.leftover, 0, "ainda havia slot: nada vai para o chão");
        assert_eq!(bag[0].qty, 120, "pilha existente completada primeiro");
        assert_eq!(bag[1].id, 400);
        assert_eq!(bag[1].qty, 30);
    }

    #[test]
    fn mochila_cheia_devolve_o_drop_como_sobra_para_a_caixa() {
        let caps = catalogo();
        let mut bag = [vazio(); 2];
        for slot in bag.iter_mut() {
            // Todas as pilhas no teto: nenhuma aceita mais nada.
            *slot = Item {
                id: 400,
                qty: STACK_CAP,
                ..vazio()
            };
        }
        let entrou = deposit_into_bag(&mut bag, com_id(400), 5, &caps);
        assert_eq!(entrou.stored, 0, "nenhuma pilha aceita e não há slot");
        assert_eq!(entrou.leftover, 5, "os 5 viram caixa no chão");
        assert_eq!(bag[0].qty, STACK_CAP);
        assert_eq!(bag[1].qty, STACK_CAP);
    }

    #[test]
    fn teto_do_banco_entra_pelo_caminho_de_producao() {
        // Valores da view `item_stack_cap` no banco real (2026-11-10): poeira e resto
        // empilham 120 mesmo sem `kind`, e o placeholder não empilha.
        let caps = StackCaps::from_caps([
            (412, STACK_CAP, None),
            (419, STACK_CAP, None),
            (416, STACK_CAP, Some(5)),
            (2433, 1, Some(5)),
            (831, 1, None),
        ]);
        assert_eq!(caps.max_stack(412), STACK_CAP);
        assert_eq!(caps.max_stack(419), STACK_CAP, "resto sem kind empilha");
        assert_eq!(caps.contained_of(416), Some(5), "pack de antídoto");
        assert_eq!(caps.max_stack(2433), 1, "placeholder N/A não empilha");
        assert_eq!(caps.max_stack(831), 1);
        assert_eq!(caps.max_stack(9999), 1, "fora da tabela: conservador");
        assert_eq!(caps.cap_of(9999), None);
    }

    #[test]
    fn teto_zero_nao_trava_o_deposito() {
        let caps = StackCaps::from_caps([(413, 0, None)]);
        assert_eq!(caps.max_stack(413), 1, "piso 1: nada entra com teto 0");
    }

    #[test]
    fn item_nao_aglutinavel_ocupa_um_slot_por_unidade() {
        let caps = catalogo();
        let mut bag = [vazio(); 4];
        let entrou = deposit_into_bag(&mut bag, com_id(831), 3, &caps);
        assert_eq!(entrou.stored, 3);
        assert_eq!(bag[0].qty, 1);
        assert_eq!(bag[1].qty, 1);
        assert_eq!(bag[2].qty, 1);
        assert_eq!(bag[3].id, 0, "o quarto slot continua livre");
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ecs::components::{EntityKind, Inventory, Position, Stats};
    use std::collections::HashMap;

    /// Catálogo de teste: espada no slot 0 (req 10), elmo no 2 (req 40), minério não equipa.
    struct Catalogo(HashMap<u32, (Option<u8>, u16)>);

    impl ItemRules for Catalogo {
        fn equip_slot(&self, item_index: u32) -> Option<u8> {
            self.0.get(&item_index).and_then(|(slot, _)| *slot)
        }
        fn req_level(&self, item_index: u32) -> u16 {
            self.0.get(&item_index).map_or(0, |(_, req)| *req)
        }
        fn max_stack(&self, _item_index: u32) -> u32 {
            10
        }
    }

    fn catalogo() -> Catalogo {
        Catalogo(HashMap::from([
            (100, (Some(0), 10)),
            (101, (Some(2), 40)),
            (200, (None, 0)),
        ]))
    }

    fn stats(level: u16) -> Stats {
        Stats {
            level,
            hp: 100,
            max_hp: 100,
            mp: 0,
            max_mp: 0,
            attack: 1,
            defense: 1,
        }
    }

    fn cenario(level: u16, bag: &[(usize, u32, u32)]) -> (ComponentStore, Entity) {
        let mut inv = Inventory::default();
        for (slot, id, qty) in bag {
            inv.bag[*slot] = Item {
                id: *id,
                qty: *qty,
                ..Item::default()
            };
        }
        let mut store = ComponentStore::default();
        let e = store
            .spawn(EntityKind::Player, Position::default(), stats(level), inv)
            .expect("spawn válido");
        (store, e)
    }

    #[test]
    fn equipa_no_slot_do_catalogo() {
        let (mut store, e) = cenario(50, &[(0, 100, 1)]);
        let feito =
            equip(&mut store, e, 0, &catalogo(), EquipmentRules::default()).expect("equipa");
        assert_eq!((feito.slot, feito.bag_slot), (0, 0));
        assert_eq!(store.inventory(e).expect("vivo").equip[0].id, 100);
        assert_eq!(store.inventory(e).expect("vivo").bag[0].id, 0);
        assert_eq!(
            store.drain_events(),
            vec![DomainEvent::ItemEquipped {
                entity: e,
                bag_slot: 0,
                equip_slot: 0,
                item_index: 100
            }]
        );
    }

    #[test]
    fn slot_pedido_diferente_do_catalogo_e_recusado() {
        let (mut store, e) = cenario(50, &[(0, 100, 1)]);
        assert_eq!(
            equip_into(&mut store, e, 0, 2, &catalogo(), EquipmentRules::default()),
            Err(CoreError::WrongEquipSlot {
                expected: 0,
                requested: 2
            })
        );
        assert_eq!(store.inventory(e).expect("vivo").bag[0].id, 100);
        assert!(store.events().is_empty());
    }

    #[test]
    fn troca_devolve_o_anterior_para_a_mochila() {
        // Duas espadas (mesmo slot 0): a segunda entra no lugar da primeira, que volta para a
        // mochila no slot de onde a nova saiu. O refino distingue as duas.
        let mut inv = Inventory::default();
        inv.bag[0] = Item {
            id: 100,
            qty: 1,
            ..Item::default()
        };
        inv.bag[1] = Item {
            id: 100,
            qty: 1,
            refine: 3,
            ..Item::default()
        };
        let mut store = ComponentStore::default();
        let e = store
            .spawn(EntityKind::Player, Position::default(), stats(50), inv)
            .expect("spawn válido");

        equip(&mut store, e, 0, &catalogo(), EquipmentRules::default()).expect("primeira espada");
        let troca = equip(&mut store, e, 1, &catalogo(), EquipmentRules::default())
            .expect("troca de espada");
        assert_eq!(troca.item.id, 100);
        assert_eq!(troca.item.refine, 3, "a nova espada é a equipada");
        assert_eq!(
            troca.swapped_out.map(|item| item.refine),
            Some(0),
            "a anterior volta para a mochila"
        );
        assert_eq!(store.inventory(e).expect("vivo").equip[0].refine, 3);
        assert_eq!(store.inventory(e).expect("vivo").bag[1].refine, 0);
    }

    #[test]
    fn politica_sem_troca_recusa_slot_ocupado() {
        let (mut store, e) = cenario(50, &[(0, 100, 1), (1, 100, 1)]);
        equip(&mut store, e, 0, &catalogo(), EquipmentRules::default()).expect("primeira");
        store.drain_events();
        assert_eq!(
            equip(
                &mut store,
                e,
                1,
                &catalogo(),
                EquipmentRules { allow_swap: false }
            ),
            Err(CoreError::EquipSlotOccupied { slot: 0 })
        );
        assert!(store.events().is_empty());
        assert_eq!(store.inventory(e).expect("vivo").bag[1].id, 100);
    }

    #[test]
    fn desequipa_para_o_primeiro_slot_livre() {
        let (mut store, e) = cenario(50, &[(0, 100, 1)]);
        equip(&mut store, e, 0, &catalogo(), EquipmentRules::default()).expect("equipa");
        store.drain_events();
        let saida = unequip(&mut store, e, 0).expect("desequipa");
        assert_eq!((saida.slot, saida.bag_slot), (0, 0));
        assert_eq!(store.inventory(e).expect("vivo").bag[0].id, 100);
        assert_eq!(store.inventory(e).expect("vivo").equip[0].id, 0);
    }

    #[test]
    fn desequipar_sem_item_e_recusado() {
        let (mut store, e) = cenario(50, &[]);
        assert_eq!(
            unequip(&mut store, e, 3),
            Err(CoreError::EmptySlot { slot: 3 })
        );
        assert!(store.events().is_empty());
    }

    #[test]
    fn mochila_cheia_recusa_desequipar() {
        let mut inv = Inventory::default();
        inv.equip[0] = Item {
            id: 100,
            qty: 1,
            ..Item::default()
        };
        for slot in &mut inv.bag {
            *slot = Item {
                id: 200,
                qty: 1,
                ..Item::default()
            };
        }
        let mut store = ComponentStore::default();
        let e = store
            .spawn(EntityKind::Player, Position::default(), stats(50), inv)
            .expect("spawn válido");
        assert_eq!(
            unequip(&mut store, e, 0),
            Err(CoreError::BagFull {
                slots: BAG_SLOTS as u16
            })
        );
        assert_eq!(store.inventory(e).expect("vivo").equip[0].id, 100);
        assert!(store.events().is_empty());
    }

    #[test]
    fn mover_empilha_ate_o_limite_do_catalogo() {
        let (mut store, e) = cenario(50, &[(0, 200, 6), (1, 200, 7)]);
        let movidos = move_item(&mut store, e, 0, 1, &catalogo()).expect("empilha");
        assert_eq!(movidos, 3);
        assert_eq!(store.inventory(e).expect("vivo").bag[0].qty, 3);
        assert_eq!(store.inventory(e).expect("vivo").bag[1].qty, 10);
    }

    /// Catálogo com teto por item — é o `EF_AMOUNT` do template (120 poções, 100 moedas…).
    struct Teto(HashMap<u32, u32>);

    impl ItemRules for Teto {
        fn equip_slot(&self, _item_index: u32) -> Option<u8> {
            None
        }
        fn req_level(&self, _item_index: u32) -> u16 {
            0
        }
        fn max_stack(&self, item_index: u32) -> u32 {
            self.0.get(&item_index).copied().unwrap_or(1)
        }
    }

    fn teto(pares: &[(u32, u32)]) -> Teto {
        Teto(pares.iter().copied().collect())
    }

    #[test]
    fn deposito_completa_a_pilha_existente_antes_de_abrir_slot() {
        let (mut store, e) = cenario(50, &[(0, 200, 6)]);
        let feito = deposit(
            &mut store,
            e,
            Item {
                id: 200,
                qty: 1,
                ..Item::default()
            },
            7,
            &teto(&[(200, 10)]),
        )
        .expect("deposita");
        assert_eq!((feito.stored, feito.leftover), (7, 0));
        let inv = store.inventory(e).expect("vivo");
        assert_eq!((inv.bag[0].qty, inv.bag[1].qty), (10, 3));
    }

    #[test]
    fn deposito_derrama_o_excedente_da_pilha_em_slots_livres() {
        let (mut store, e) = cenario(50, &[]);
        let feito = deposit(
            &mut store,
            e,
            Item {
                id: 413,
                qty: 1,
                ..Item::default()
            },
            200,
            &teto(&[(413, 120)]),
        )
        .expect("deposita");
        assert_eq!((feito.stored, feito.leftover), (200, 0));
        let inv = store.inventory(e).expect("vivo");
        assert_eq!((inv.bag[0].qty, inv.bag[1].qty), (120, 80));
        assert_eq!(inv.bag[0].id, 413);
    }

    #[test]
    fn mochila_cheia_devolve_o_drop_como_sobra_para_o_chao() {
        let mut inv = Inventory::default();
        for slot in &mut inv.bag {
            *slot = Item {
                id: 999,
                qty: 1,
                ..Item::default()
            };
        }
        let mut store = ComponentStore::default();
        let e = store
            .spawn(EntityKind::Player, Position::default(), stats(50), inv)
            .expect("spawn válido");
        let feito = deposit(
            &mut store,
            e,
            Item {
                id: 200,
                qty: 1,
                ..Item::default()
            },
            5,
            &teto(&[(200, 120)]),
        )
        .expect("deposita");
        assert_eq!((feito.stored, feito.leftover), (0, 5));
        assert!(store
            .inventory(e)
            .expect("vivo")
            .bag
            .iter()
            .all(|slot| slot.id == 999));
    }

    #[test]
    fn item_sem_aglutinacao_ocupa_um_slot_por_unidade() {
        let (mut store, e) = cenario(50, &[]);
        let feito = deposit(
            &mut store,
            e,
            Item {
                id: 1234,
                qty: 1,
                ..Item::default()
            },
            3,
            &teto(&[]),
        )
        .expect("deposita");
        assert_eq!((feito.stored, feito.leftover), (3, 0));
        let inv = store.inventory(e).expect("vivo");
        assert_eq!((inv.bag[0].qty, inv.bag[1].qty, inv.bag[2].qty), (1, 1, 1));
    }

    #[test]
    fn item_vazio_nao_entra_e_volta_como_sobra() {
        let (mut store, e) = cenario(50, &[]);
        let feito =
            deposit(&mut store, e, Item::default(), 3, &teto(&[(0, 120)])).expect("deposita");
        assert_eq!((feito.stored, feito.leftover), (0, 3));
        assert!(store
            .inventory(e)
            .expect("vivo")
            .bag
            .iter()
            .all(|s| s.id == 0));
    }
}
