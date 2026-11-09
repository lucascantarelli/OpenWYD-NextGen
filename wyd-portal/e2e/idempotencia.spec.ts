import { expect, test } from "@playwright/test";

import { characterId, grantsOf, login } from "./support";

/**
 * Idempotência pelo caminho real: a mesma chave submetida duas vezes no
 * formulário devolve replay, mantém **uma** linha na fila e credita o ouro
 * uma única vez.
 */
test("repetir a mesma chave não duplica entrega nem ouro", async ({ page }) => {
  await login(page);
  await page.goto("/entregas");

  const row = page.getByTestId(/^char-/).filter({ hasText: "E2E Gêmeo" });
  await expect(row).toBeVisible();
  const id = characterId(await row.getAttribute("data-testid"));

  const key = `e2e-idem-${id}`;
  await row.getByLabel("Chave de idempotência").fill(key);
  await row.getByLabel("Ouro").fill("300");

  await row.getByRole("button", { name: "Conceder ouro" }).click();
  await expect(row.getByRole("status")).toContainText("enfileirada");

  await row.getByRole("button", { name: "Conceder ouro" }).click();
  await expect(row.getByRole("status")).toContainText("replay");

  await expect
    .poll(
      async () => {
        const rows = (await grantsOf(id)).filter((grant) => grant.idempotency_key === key);
        return rows.length === 1 && rows[0]?.status === "claimed";
      },
      { timeout: 20_000, message: "a entrega idempotente não foi aplicada pelo canal" },
    )
    .toBe(true);

  const same = (await grantsOf(id)).filter((grant) => grant.idempotency_key === key);
  expect(same).toHaveLength(1);

  await page.reload();
  await expect(page.getByTestId(`gold-${id}`)).toHaveText("300");
});
