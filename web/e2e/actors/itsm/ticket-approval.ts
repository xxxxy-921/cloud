import { expect } from "@playwright/test"

import { TicketEvidence } from "../../support/ticket-evidence"
import type { AgenticITSMActorSession, VPNTicketRef } from "./session"

export class TicketApprovalActor {
  private readonly evidence: TicketEvidence

  constructor(private readonly session: AgenticITSMActorSession) {
    this.evidence = new TicketEvidence(session)
  }

  async openNetworkTodo(ticket: VPNTicketRef) {
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
  }

  async expectCurrentNode(ticket: VPNTicketRef, activityName: string) {
    await this.evidence.expectActionableActivity(ticket, activityName)
    await expect(this.session.page.getByText(activityName).first()).toBeVisible({ timeout: 30_000 })
  }

  async approveOpenedVPNRequest(ticket: VPNTicketRef, opinion: string) {
    const { page } = this.session

    await page.getByTestId("itsm-ticket-approve-button").click()
    await page.getByLabel("处理意见").fill(opinion)
    const [response] = await Promise.all([
      page.waitForResponse((res) => {
        const url = new URL(res.url())
        return url.pathname === `/api/v1/itsm/tickets/${ticket.ticketId}/progress` && res.request().method() === "POST"
      }, { timeout: 30_000 }),
      page.getByTestId("itsm-ticket-confirm-approve-button").click(),
    ])
    expect(response.ok()).toBeTruthy()
    await this.evidence.expectTicketStatus(ticket, "completed")
  }
}
