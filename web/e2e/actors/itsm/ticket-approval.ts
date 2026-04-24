import { expect } from "@playwright/test"

import type { AgenticITSMActorSession, VPNTicketRef } from "./session"

type TicketResponse = {
  id: number
  code: string
  status: string
}

export class TicketApprovalActor {
  constructor(private readonly session: AgenticITSMActorSession) {}

  async approveNetworkTodo(ticket: VPNTicketRef) {
    const { page } = this.session

    await page.goto("/itsm/tickets/approvals/pending")
    await expect(page.getByRole("heading", { name: "我的待办" })).toBeVisible()

    const row = page.getByTestId(`itsm-ticket-row-${ticket.ticketCode}`)
    await expect(async () => {
      await page.reload()
      await expect(row).toBeVisible({ timeout: 5_000 })
    }).toPass({ timeout: 90_000 })
    await row.click()

    await expect(page.getByText(ticket.ticketCode)).toBeVisible()
    await expect(page.getByText("网络管理员处理").first()).toBeVisible({ timeout: 30_000 })

    await page.getByTestId("itsm-ticket-approve-button").click()
    await page.getByLabel("处理意见").fill("同意开通 VPN，用于线上支持。")
    const [response] = await Promise.all([
      page.waitForResponse((res) => {
        const url = new URL(res.url())
        return url.pathname === `/api/v1/itsm/tickets/${ticket.ticketId}/progress` && res.request().method() === "POST"
      }, { timeout: 30_000 }),
      page.getByTestId("itsm-ticket-confirm-approve-button").click(),
    ])
    expect(response.ok()).toBeTruthy()
    await this.waitForTicketStatus(ticket.ticketId, "completed")
  }

  private async waitForTicketStatus(ticketId: number, status: string) {
    await expect(async () => {
      const ticket = await this.session.api<TicketResponse>("GET", `/api/v1/itsm/tickets/${ticketId}`)
      expect(ticket.status).toBe(status)
    }).toPass({ timeout: 120_000, intervals: [1_000, 2_000, 3_000] })
  }
}
