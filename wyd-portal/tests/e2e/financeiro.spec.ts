import { expect, test } from "@playwright/test";

import { login } from "./support";

/**
 * Loja/financeiro do portal: cadastro de item da vitrine e crédito de saldo
 * idempotente pela mesma chave. A compra feita pelo jogador ainda não tem
 * contrato no gateway e pertence à WEB-09; aqui fica provado o lado staff.
 */
test("vitrine cadastra item e crédito repetido devolve replay", async ({ page }) => {
  await login(page);
  await page.goto("/financeiro");

  await expect(page.getByRole("heading", { name: "Financeiro" })).toBeVisible();

  await page.getByLabel("Índice do item").fill("1");
  await page.getByLabel("Preço (créditos)").fill("10");
  await page.getByRole("button", { name: "Adicionar item" }).click();
  await expect(page.getByText("Registrado: loja.", { exact: true })).toBeVisible();
  await expect(page.getByText("10 créditos").first()).toBeVisible();

  const key = `e2e-donate-${Date.now()}`;

  await page.getByLabel("Conta (id)").fill("1");
  await page.getByLabel("Créditos", { exact: true }).fill("25");
  await page.getByLabel("Chave de idempotência").fill(key);
  await page.getByRole("button", { name: "Creditar" }).click();
  await expect(page.getByText(/^Registrado: crédito #/)).toBeVisible();

  await page.getByLabel("Conta (id)").fill("1");
  await page.getByLabel("Créditos", { exact: true }).fill("25");
  await page.getByLabel("Chave de idempotência").fill(key);
  await page.getByRole("button", { name: "Creditar" }).click();
  await expect(page.getByText(/^Registrado: replay #/)).toBeVisible();
});
