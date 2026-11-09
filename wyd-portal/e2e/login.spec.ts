import { expect, test } from "@playwright/test";

import { login } from "./support";

/**
 * Autenticação do staff pelo portal real: sem sessão a área administrativa
 * redireciona para `/login`; com credencial válida (verificada no banco pelo
 * gateway) a tela de entregas abre.
 */
test("staff sem sessão cai no login e entra na área administrativa", async ({ page }) => {
  await page.goto("/entregas");
  await expect(page).toHaveURL(/\/login$/);

  await login(page);

  await page.goto("/entregas");
  await expect(page.getByRole("heading", { name: "Entregas" })).toBeVisible();
  await expect(page.getByRole("table", { name: "Personagens" })).toBeVisible();
});
