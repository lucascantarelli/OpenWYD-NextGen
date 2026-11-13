import { expect, test } from "@playwright/test";

import { login } from "./support";

/**
 * Área do jogador com sessão real: conta, personagens, ranking, histórico e
 * logout. O staff usa a mesma tela; o que muda é apenas o destino pós-login.
 */
test("login abre a área do jogador e o logout encerra a sessão", async ({ page }) => {
  await login(page);
  await page.goto("/conta");

  await expect(page.getByRole("heading", { level: 1 })).toContainText("Olá, admin.");
  await expect(page.getByRole("heading", { name: "Personagens" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Ranking" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Conta", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Sessões" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Sair" })).toBeVisible();

  // A resposta pessoal nunca carrega hash nem segredo de credencial.
  expect(await page.content()).not.toContain("$argon2");

  await page.getByRole("button", { name: "Sair" }).click();
  await expect(page).toHaveURL(/\/login$/);

  await page.goto("/conta");
  await expect(page).toHaveURL(/\/login$/);
});
