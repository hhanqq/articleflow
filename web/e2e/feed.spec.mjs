import { expect, test } from "@playwright/test";

test("article feed UI loads and can request stored search", async ({ page }) => {
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Поиск статей" })).toBeVisible();
  await expect(page.getByLabel("Query").or(page.getByLabel("Запрос"))).toBeVisible();
  await expect(page.getByRole("button", { name: "Искать в базе" })).toBeVisible();

  await page.getByLabel("Запрос").fill("go kafka");
  await page.getByLabel("Лимит").fill("5");
  await page.getByRole("button", { name: "Искать в базе" }).click();

  await expect(page.locator("#feed")).toBeVisible();
  await expect(page.locator("#lastError")).toBeEmpty();
  expect(errors).toEqual([]);
});
