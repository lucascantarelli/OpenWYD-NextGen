import { expect, type Page } from "@playwright/test";

/** Staff semeado pelo próprio gateway no boot (ver `scripts/e2e.mjs`). */
export const STAFF = {
  user: process.env.E2E_ADMIN_USER ?? "admin",
  pass: process.env.E2E_ADMIN_PASS ?? "e2e-admin-pass-123",
};

/** Gateway efêmero do E2E (publicado em loopback pelo provisionamento). */
export const GATEWAY_URL = process.env.E2E_GATEWAY_URL ?? "http://127.0.0.1:7601";

const GATEWAY_TOKEN = process.env.E2E_GATEWAY_TOKEN ?? "e2e-token-do-gateway";

export interface GrantRow {
  id: number;
  kind: string;
  gold: number;
  status: string;
  idempotency_key: string | null;
}

/** Login real do portal (credencial verificada no banco pelo gateway). */
export async function login(page: Page): Promise<void> {
  await page.goto("/login");
  await page.getByLabel("Usuário").fill(STAFF.user);
  await page.getByLabel("Senha").fill(STAFF.pass);
  await page.getByRole("button", { name: "Entrar" }).click();
  await expect(page).toHaveURL(/\/painel$/);
}

/** Fila do personagem lida pela porta real (`character_admin/list_character_grants`). */
export async function grantsOf(characterId: number): Promise<GrantRow[]> {
  const res = await fetch(`${GATEWAY_URL}/admin/v1/character_admin/list_character_grants`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      authorization: `Bearer ${GATEWAY_TOKEN}`,
      "x-wyd-actor": "1:admin",
    },
    body: JSON.stringify({ character_id: characterId, limit: 20 }),
  });
  expect(res.ok).toBeTruthy();
  const body = (await res.json()) as { grants?: GrantRow[] };
  return body.grants ?? [];
}

/** `char-42` → 42; falha alto se o testid não seguir o padrão. */
export function characterId(rowTestId: string | null): number {
  if (!rowTestId || !rowTestId.startsWith("char-")) {
    throw new Error(`testid inesperado na linha do personagem: ${rowTestId}`);
  }
  const id = Number(rowTestId.slice("char-".length));
  if (!Number.isInteger(id) || id <= 0) throw new Error(`id inválido: ${rowTestId}`);
  return id;
}
