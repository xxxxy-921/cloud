import { expect, type Page } from "@playwright/test"

import { devUsers, expectAuthenticatedAs, loginAs, openLoginPage } from "./auth-behavior"

export class AgenticITSMLiveWorld {
  private readonly runtimeErrors: string[] = []

  constructor(private readonly page: Page) {
    this.page.on("pageerror", (error) => {
      this.runtimeErrors.push(error.message)
    })
    this.page.on("console", (message) => {
      if (message.type() === "error") {
        this.runtimeErrors.push(message.text())
      }
    })
  }

  async givenIsolatedDevEnvironmentIsReady() {
    await openLoginPage(this.page)
  }

  async whenAdminOpensLoginPage() {
    await openLoginPage(this.page)
  }

  async whenAdminSubmitsDevCredentials() {
    await loginAs(this.page, devUsers.admin)
  }

  async thenAdminCanSeeAuthenticatedShell() {
    await expectAuthenticatedAs(this.page, devUsers.admin)
  }

  async thenNoRuntimeErrorsAppeared() {
    expect(this.runtimeErrors).toEqual([])
  }
}
