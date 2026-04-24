import { expect, type Page } from "@playwright/test"

import { type APIResult, apiRequest } from "../../support/api-client"
import { type E2EUserCredentials, loginAs } from "../../support/auth-behavior"

export type VPNTicketRef = {
  ticketId: number
  ticketCode: string
}

export class AgenticITSMActorFactory {
  private session: AgenticITSMActorSession | null = null

  constructor(private readonly page: Page) {}

  async loginAs(user: E2EUserCredentials) {
    await this.page.context().clearCookies()
    await this.page.goto("/login")
    await this.page.evaluate(() => {
      window.localStorage.clear()
      window.sessionStorage.clear()
    })

    this.session ??= new AgenticITSMActorSession(this.page)
    await this.session.login(user)
    return this.session
  }

  async close() {}
}

export class AgenticITSMActorSession {
  private runtimeErrors: string[] = []

  constructor(readonly page: Page) {
    this.page.on("pageerror", (error) => this.runtimeErrors.push(error.message))
    this.page.on("console", (message) => {
      if (message.type() === "error") {
        this.runtimeErrors.push(message.text())
      }
    })
  }

  async login(user: E2EUserCredentials) {
    this.runtimeErrors = []
    await loginAs(this.page, user)
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
