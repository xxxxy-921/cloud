import { expect, type Page } from "@playwright/test"

export async function fillFormTextField(page: Page, key: string, value: string) {
  const field = page.getByTestId(`itsm-form-field-${key}`)
  await expect(field).toBeVisible()
  await field.locator("input, textarea").first().fill(value)
}

export async function selectFormOption(page: Page, key: string, option: string) {
  const field = page.getByTestId(`itsm-form-field-${key}`)
  await expect(field).toBeVisible()
  await field.getByRole("combobox").click()
  await page.getByRole("option", { name: option }).click()
}
