import { expect, test } from "@playwright/test";

import { characterId, grantsOf, login } from "./support";

/**
 * Caminho de jogo completo e observável: o staff concede ouro pela UI, a linha
 * entra na `delivery_queue` e o canal drena — o ouro do personagem muda de 0
 * para o valor concedido na própria tela (relida do banco).
 */
test("concessão pela tela credita o ouro do personagem", async ({ page }) => {
  await login(page);
  await page.goto("/entregas");

  const row = page.getByTestId(/^char-/).filter({ hasText: "E2E Hero" });
  await expect(row).toBeVisible();
  const id = characterId(await row.getAttribute("data-testid"));

  await expect(page.getByTestId(`gold-${id}`)).toHaveText("0");

  await row.getByLabel(`Ouro a conceder para o personagem #${id}`).fill("250");
  await row.getByRole("button", { name: "Conceder" }).click();

  await expect(row.getByRole("status")).toContainText("enfileirada");

  // O servidor de jogo drena a fila a cada 1 s; espera o estado real do banco.
  await expect
    .poll(
      async () =>
        (await grantsOf(id)).some((grant) => grant.status === "claimed" && grant.gold === 250),
      { timeout: 20_000, message: "a entrega de 250 de ouro não foi aplicada pelo canal" },
    )
    .toBe(true);

  await page.reload();
  await expect(page.getByTestId(`gold-${id}`)).toHaveText("250");
});
