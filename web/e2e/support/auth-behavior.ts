import { expect, type Page, type Response } from "@playwright/test"

export type E2EUserCredentials = {
  username: string
  password: string
  displayName?: string
}

export const devUsers = {
  admin: {
    username: "admin",
    password: "password",
    displayName: "admin",
  },
  itsmServiceManager: {
    username: "itsm_service_manager",
    password: "password",
    displayName: "itsm_service_manager",
  },
  vpnApplicant: {
    username: "vpn_applicant",
    password: "password",
    displayName: "vpn_applicant",
  },
  vpnNetworkApprover: {
    username: "vpn_network_approver",
    password: "password",
    displayName: "vpn_network_approver",
  },
  vpnSecurityApprover: {
    username: "vpn_security_approver",
    password: "password",
    displayName: "vpn_security_approver",
  },
} satisfies Record<string, E2EUserCredentials>

export async function openLoginPage(page: Page) {
  await page.goto("/login")
  await expect(page.locator("#username")).toBeVisible()
  await expect(page.locator("#password")).toBeVisible()
  await expect(page).toHaveURL(/\/login$/)
}

export async function loginAs(page: Page, user: E2EUserCredentials) {
  await openLoginPage(page)
  await page.locator("#username").fill(user.username)
  await page.locator("#password").fill(user.password)

  const [loginResponse] = await Promise.all([
    page.waitForResponse((response) => {
      const url = new URL(response.url())
      return url.pathname === "/api/v1/auth/login" && response.request().method() === "POST"
    }),
    page.locator('button[type="submit"]').click(),
  ])

  await expectLoginSucceeded(loginResponse)
  await expectAuthenticatedAs(page, user)
}

export async function expectAuthenticatedAs(page: Page, user: E2EUserCredentials) {
  await expect(page).not.toHaveURL(/\/login$/)
  await expect(page.getByRole("button", { name: userDisplayName(user) })).toBeVisible()
}

async function expectLoginSucceeded(response: Response) {
  expect(response.ok()).toBeTruthy()
  const body = await response.json()
  expect(body.code).toBe(0)
}

function userDisplayName(user: E2EUserCredentials) {
  return user.displayName ?? user.username
}
