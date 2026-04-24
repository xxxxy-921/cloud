import { expect } from "@playwright/test"

import type { AgenticITSMActorSession } from "./session"

type ServiceListResponse = {
  items: Array<{ id: number; code: string; name: string }>
  total: number
}

export class ServiceCatalogActor {
  constructor(private readonly session: AgenticITSMActorSession) {}

  async generateVPNWorkflowReference() {
    const { page } = this.session

    await page.goto("/itsm/services")
    await expect(page.getByRole("heading", { name: "服务目录" })).toBeVisible()

    const service = await this.vpnService()
    const serviceCard = page.getByTestId("itsm-service-card-vpn-access-request")
    await expect(serviceCard).toBeVisible()
    await serviceCard.click()
    await expect(page).toHaveURL(new RegExp(`/itsm/services/${service.id}$`))
    await expect(page.getByRole("heading", { name: "VPN 开通申请" })).toBeVisible()

    const [response] = await Promise.all([
      page.waitForResponse((res) => {
        const url = new URL(res.url())
        return url.pathname === "/api/v1/itsm/workflows/generate" && res.request().method() === "POST"
      }, { timeout: 120_000 }),
      page.getByTestId("itsm-generate-workflow-button").click(),
    ])

    expect(response.ok()).toBeTruthy()
    const body = await response.json()
    expect(body.code).toBe(0)
    const workflowJSON = JSON.stringify(body.data?.workflowJson ?? {})
    expect(workflowJSON).toContain("network_admin")
    expect(workflowJSON).toContain("security_admin")

    await expect(page.getByText("填写 VPN 开通申请").first()).toBeVisible({ timeout: 20_000 })
    await expect(page.getByText("网络管理员处理").first()).toBeVisible()
    await expect(page.getByText("信息安全管理员处理").first()).toBeVisible()
  }

  private async vpnService() {
    const data = await this.session.api<ServiceListResponse>("GET", "/api/v1/itsm/services?page=1&pageSize=100")
    const service = data.items.find((item) => item.code === "vpn-access-request")
    expect(service).toBeTruthy()
    return service!
  }
}
