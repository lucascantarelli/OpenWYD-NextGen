/**
 * Leitura do `ItemEffectAdminService` (issue 44, metade de escrita).
 *
 * `ListEffects` devolve os adds do item já mesclados (ItemList.csv + overlay)
 * mais o vocabulário `ItemEffect.h`, então a tela de edição não tem cópia própria
 * da tabela de efeitos.
 */
import { adminCall } from "./admin";
import { itemEffectAdmin, type ListItemEffectsResponse } from "./admin-client";

export async function getItemEffects(itemIndex: number): Promise<ListItemEffectsResponse> {
  return adminCall<{ item_index: number }, ListItemEffectsResponse>(
    itemEffectAdmin.listEffects.bind(itemEffectAdmin),
    { item_index: itemIndex },
    { action: "item_effect.list", target: `item:${itemIndex}` },
  );
}
