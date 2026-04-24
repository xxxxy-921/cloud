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

  async submitOnlineSupportVPNRequest(): Promise<VPNTicketRef> {
    const { page } = this.session

    await page.goto("/itsm/service-desk")
    await expect(page.getByRole("heading", { name: "IT 服务台" })).toBeVisible()

    const prompt = [
      "我想申请 VPN。",
      "VPN账号是 vpn_applicant@local.dev。",
      "设备与用途说明：线上支持用，需要访问公司内网。",
      "访问原因选择线上支持。",
    ].join(" ")

    const composer = page.getByPlaceholder("描述你的 IT 诉求...").first()
    await expect(composer).toBeVisible()
    await composer.fill(prompt)
    await composer.press("Enter")

    await expect(page.getByTestId("itsm-submit-draft-button")).toBeVisible({ timeout: 120_000 })
    await fillFormTextField(page, "vpn_account", "vpn_applicant@local.dev")
    await fillFormTextField(page, "device_usage", "线上支持用，需要访问公司内网。")
    await selectFormOption(page, "request_kind", "线上支持")

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
}
