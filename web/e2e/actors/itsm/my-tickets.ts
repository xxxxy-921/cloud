import { expect } from "@playwright/test"

import type { AgenticITSMActorSession, VPNTicketRef } from "./session"

type TicketResponse = {
  id: number
  code: string
  title: string
  status: string
  serviceName: string
}

export class MyTicketsActor {
  constructor(private readonly session: AgenticITSMActorSession) {}

  async openVPNRequest(ticket: VPNTicketRef) {
    const { page } = this.session

    await page.goto("/itsm/tickets/mine")
    await expect(page.getByRole("heading", { name: "我的工单" })).toBeVisible()

    const row = page.getByTestId(`itsm-ticket-row-${ticket.ticketCode}`)
    await expect(row).toBeVisible({ timeout: 30_000 })
    await expect(row).toContainText("VPN")
    await row.click()

    await expect(page.getByText(ticket.ticketCode)).toBeVisible()
  }

  async expectOpenedVPNRequestApproved(ticket: VPNTicketRef) {
    const { page } = this.session

    await expect(page.getByText("已完成").first()).toBeVisible({ timeout: 30_000 })
    const detail = await this.session.api<TicketResponse>("GET", `/api/v1/itsm/tickets/${ticket.ticketId}`)
    expect(detail.status).toBe("completed")
    expect(detail.serviceName).toBe("VPN 开通申请")
  }
}
