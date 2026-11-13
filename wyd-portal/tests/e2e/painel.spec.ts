import { expect, test } from "@playwright/test";

import { login } from "./support";

/**
 * Painel administrativo real: a visão geral do staff abre com os dados que o
 * gateway entrega, e a busca de contas consulta uma conta existente pelo login.
 */
test("staff entra no painel e consulta uma conta", async ({ page }) => {
  await login(page);

  await expect(page.getByRole("heading", { name: "O mundo em tempo real." })).toBeVisible();
  await expect(page.getByRole("link", { name: "Abrir monitoramento" })).toBeVisible();

  await page.goto("/contas");
  await expect(page.getByRole("heading", { name: "Contas", exact: true })).toBeVisible();

  await page.getByLabel("Login exato da conta").fill("admin");
  await page.getByRole("button", { name: "Buscar" }).click();

  await expect(page).toHaveURL(/\/contas\?login=admin/);
  await expect(page.getByRole("heading", { name: "admin", exact: true })).toBeVisible();
  await expect(page.getByRole("main").getByText("Conta #1", { exact: true })).toBeVisible();
  await expect(page.getByText("painel administrativo")).toBeVisible();
});
