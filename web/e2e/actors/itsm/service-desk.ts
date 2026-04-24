import { expect } from "@playwright/test"

import { fillFormTextField, selectFormOption } from "../../support/form-behavior"
import type { AgenticITSMActorSession, VPNTicketRef } from "./session"

type DraftSubmitResponse = {
  ok: boolean
  ticketId?: number
  ticketCode?: string
  message?: string
}

export class ServiceDeskActor {
  constructor(private readonly session: AgenticITSMActorSession) {}

  async submitVPNRequest(account: string, deviceUsage: string, reasonLabel: string): Promise<VPNTicketRef> {
    const { page } = this.session

    await page.goto("/itsm/service-desk")
    await expect(page.getByRole("heading", { name: "IT 服务台" })).toBeVisible({ timeout: 30_000 })

    await this.describeVPNRequestAcrossTurns(account, deviceUsage, reasonLabel)
    await fillFormTextField(page, "vpn_account", account)
    await fillFormTextField(page, "device_usage", deviceUsage)
    await selectFormOption(page, "request_kind", reasonLabel)

    const [response] = await Promise.all([
      page.waitForResponse((res) => {
        const url = new URL(res.url())
        return url.pathname.endsWith("/draft/submit") && res.request().method() === "POST"
      }, { timeout: 60_000 }),
      page.getByTestId("itsm-submit-draft-button").click(),
    ])

    expect(response.ok()).toBeTruthy()
    const body = await response.json()
    expect(body.code).toBe(0)
    const result = body.data as DraftSubmitResponse
    expect(result.ok).toBeTruthy()
    expect(result.ticketId).toBeGreaterThan(0)
    expect(result.ticketCode).toBeTruthy()

    const ticketRef = {
      ticketId: result.ticketId!,
      ticketCode: result.ticketCode!,
    }
    await expect(page.getByTestId("itsm-submitted-ticket-code")).toContainText(ticketRef.ticketCode)
    return ticketRef
  }

  private async describeVPNRequestAcrossTurns(account: string, deviceUsage: string, reasonLabel: string) {
    const turns = [
      "我想申请 VPN。",
      `VPN账号是 ${account}。`,
      `设备与用途说明：${deviceUsage}。`,
      `访问原因选择${reasonLabel}。`,
      "确认服务就是 VPN 开通申请，请把以上信息整理成服务申请草稿给我确认。",
    ]

    for (const turn of turns) {
      await this.sendServiceDeskMessage(turn)
      if (await this.draftReady(5_000)) {
        return
      }
    }

    await expect(this.session.page.getByTestId("itsm-submit-draft-button")).toBeVisible({ timeout: 120_000 })
  }

  private async sendServiceDeskMessage(text: string) {
    const { page } = this.session
    const composer = page.getByPlaceholder("描述你的 IT 诉求...").last()
    await expect(composer).toBeVisible({ timeout: 30_000 })
    await expect(composer).toBeEditable({ timeout: 120_000 })
    await composer.fill(text)

    const [response] = await Promise.all([
      page.waitForResponse((res) => {
        const url = new URL(res.url())
        return /^\/api\/v1\/ai\/sessions\/\d+\/chat$/.test(url.pathname) && res.request().method() === "POST"
      }, { timeout: 120_000 }),
      composer.press("Enter"),
    ])
    expect(response.ok()).toBeTruthy()
  }

  private async draftReady(timeout: number) {
    try {
      await expect(this.session.page.getByTestId("itsm-submit-draft-button")).toBeVisible({ timeout })
      return true
    } catch {
      return false
    }
  }
}
