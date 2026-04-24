import { expect, type Browser, type BrowserContext, type Page } from "@playwright/test"

import { type APIResult, apiRequest } from "../../support/api-client"
import { type E2EUserCredentials, loginAs } from "../../support/auth-behavior"

export type VPNTicketRef = {
  ticketId: number
  ticketCode: string
}

export class AgenticITSMActorFactory {
  private readonly contexts: BrowserContext[] = []

  constructor(private readonly browser: Browser) {}

  async loginAs(user: E2EUserCredentials) {
    const context = await this.browser.newContext()
    this.contexts.push(context)
    const page = await context.newPage()
    const session = new AgenticITSMActorSession(page, user)
    await session.login()
    return session
  }

  async close() {
    await Promise.all(this.contexts.map((context) => context.close()))
  }
}

export class AgenticITSMActorSession {
  private readonly runtimeErrors: string[] = []

  constructor(
    readonly page: Page,
    private readonly user: E2EUserCredentials,
  ) {
    this.page.on("pageerror", (error) => this.runtimeErrors.push(error.message))
    this.page.on("console", (message) => {
      if (message.type() === "error") {
        this.runtimeErrors.push(message.text())
      }
    })
  }

  async login() {
    await loginAs(this.page, this.user)
  }

  async api<T>(method: string, path: string, body?: unknown): Promise<T> {
    const result = await apiRequest<T>(this.page, method, path, body)
    this.expectAPIResult(result)
    return result.body.data
  }

  expectNoRuntimeErrors() {
    expect(this.runtimeErrors).toEqual([])
  }

  private expectAPIResult<T>(result: APIResult<T>) {
    expect(result.ok, JSON.stringify(result.body)).toBeTruthy()
    expect(result.body.code, result.body.message).toBe(0)
  }
}
