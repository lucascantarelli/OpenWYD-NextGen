/**
 * Leituras do `CharacterAdminService` (issue 43).
 *
 * Só leitura: o serviço do web-api responde busca e detalhe a partir do estado
 * frio. Toda mutação futura (concessão de item/gold, edição de campo) passa por
 * `delivery_queue` ou por um editor offline — nunca por um UPDATE daqui.
 */
import type { CharacterDetail, CharacterSummary } from "@/lib/character";

import { adminCall } from "./admin";
import {
  characterAdmin,
  type GetCharacterAdminResponse,
  type SearchCharactersResponse,
} from "./grpc";

export async function searchCharacters(query: string): Promise<CharacterSummary[]> {
  const trimmed = query.trim();
  if (trimmed.length < 2) return [];
  const res = await adminCall<{ query: string }, SearchCharactersResponse>(
    characterAdmin.searchCharacters.bind(characterAdmin),
    { query: trimmed },
    { action: "character.search", target: `query:${trimmed.slice(0, 32)}` },
  );
  return res.characters ?? [];
}

export async function getCharacter(characterId: number): Promise<CharacterDetail | null> {
  if (!Number.isInteger(characterId) || characterId <= 0) return null;
  const res = await adminCall<{ character_id: number }, GetCharacterAdminResponse>(
    characterAdmin.getCharacter.bind(characterAdmin),
    { character_id: characterId },
    { action: "character.get", target: `character:${characterId}` },
  );
  if (!res.character) return null;
  // skill_names vive na resposta (mapa), não no detalhe: o detalhe é o save, o
  // mapa é catálogo. Junta aqui para a tela receber um objeto só.
  return {
    ...res.character,
    skill_names: res.skill_names ?? {},
    add_labels: res.add_labels ?? {},
  };
}
