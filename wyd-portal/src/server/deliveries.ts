/**
 * Entregas (FASE-07-4): busca de personagem, fila e concessão de ouro.
 *
 * Toda chamada passa por `adminCall` (sessão de staff + auditoria) e chega ao
 * gateway pelo contrato real (`character_admin/*`, `grants/request`). O portal
 * nunca escreve estado de personagem: enfileira e relê; quem credita é o
 * servidor de jogo, que drena a `delivery_queue`.
 */
import { adminCall } from "./admin";
import {
  characterAdmin,
  grantsAdmin,
  type CharacterWire,
  type GrantReceiptWire,
  type GrantRowWire,
  type ListCharacterGrantsResponse,
  type SearchCharactersResponse,
} from "./admin-client";

export type { CharacterWire, GrantRowWire };

/** Personagens vivos por nome (vazio = todos), no máximo 20. */
export async function searchCharacters(term: string): Promise<CharacterWire[]> {
  const clean = term.trim().slice(0, 32);
  const res = await adminCall<{ term: string; limit: number }, SearchCharactersResponse>(
    characterAdmin.searchCharacters.bind(characterAdmin),
    { term: clean, limit: 20 },
    { action: "character.search", target: clean === "" ? "*" : `term:${clean}` },
  );
  return res.characters ?? [];
}

/** Fila do personagem (pending/claimed), mais recente primeiro. */
export async function listCharacterGrants(characterId: number): Promise<GrantRowWire[]> {
  const res = await adminCall<
    { character_id: number; limit: number },
    ListCharacterGrantsResponse
  >(
    characterAdmin.listCharacterGrants.bind(characterAdmin),
    { character_id: characterId, limit: 20 },
    { action: "grants.list", target: `char:${characterId}` },
  );
  return res.grants ?? [];
}

/** Enfileira ouro; a chave é a identidade da entrega (replay não duplica). */
export async function requestGoldGrant(input: {
  characterId: number;
  amount: number;
  key: string;
}): Promise<GrantReceiptWire> {
  const res = await adminCall<
    {
      idempotency_key: string;
      character_id: number;
      kind: { type: string; amount: number };
    },
    GrantReceiptWire
  >(
    grantsAdmin.request.bind(grantsAdmin),
    {
      idempotency_key: input.key,
      character_id: input.characterId,
      kind: { type: "gold", amount: input.amount },
    },
    { action: "grants.request", target: `char:${input.characterId}` },
  );
  return res;
}
