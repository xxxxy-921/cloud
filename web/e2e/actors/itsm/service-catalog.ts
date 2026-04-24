import { expect } from "@playwright/test"

import type { AgenticITSMActorSession } from "./session"

type ServiceListResponse = {
  items: Array<{ id: number; code: string; name: string }>
  total: number
}

type ServiceDetailResponse = {
  id: number
  code: string
  name: string
  workflowJson: unknown
}

type WorkflowNode = {
  id: string
  data?: {
    label?: string
    participants?: Array<Record<string, unknown>>
  }
}

type WorkflowEdge = {
  id: string
  source: string
  target: string
  data?: {
    condition?: {
      field?: string
      operator?: string
      value?: unknown
    }
  }
}

type WorkflowJSON = {
  nodes?: WorkflowNode[]
  edges?: WorkflowEdge[]
}

export class ServiceCatalogActor {
  constructor(private readonly session: AgenticITSMActorSession) {}

  async generateVPNWorkflowReference() {
    const { page } = this.session

    await page.goto("/itsm/services")
    await expect(page.getByRole("heading", { name: "服务目录" })).toBeVisible()

    const service = await this.vpnService()
    await page.getByRole("button", { name: /网络与 VPN/ }).click()
    const serviceCard = page.getByTestId("itsm-service-card-vpn-access-request")
    await expect(serviceCard).toBeVisible()
    const [detailResponse] = await Promise.all([
      page.waitForResponse((res) => {
        const url = new URL(res.url())
        return url.pathname === `/api/v1/itsm/services/${service.id}` && res.request().method() === "GET"
      }),
      serviceCard.click(),
    ])
    expect(detailResponse.ok()).toBeTruthy()
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

  async expectVPNWorkflowRoutesToNetworkAndSecurity() {
    const service = await this.vpnService()
    const detail = await this.session.api<ServiceDetailResponse>("GET", `/api/v1/itsm/services/${service.id}`)
    const workflow = this.workflow(detail.workflowJson)
    const networkNode = this.nodeByLabel(workflow, "网络管理员处理")
    const securityNode = this.nodeByLabel(workflow, "信息安全管理员处理")

    expect(JSON.stringify(networkNode.data?.participants ?? [])).toContain("network_admin")
    expect(JSON.stringify(securityNode.data?.participants ?? [])).toContain("security_admin")

    const networkEdge = this.routeEdgeTo(workflow, networkNode.id)
    const securityEdge = this.routeEdgeTo(workflow, securityNode.id)

    expect(networkEdge.data?.condition?.field).toBe("form.request_kind")
    expect(networkEdge.data?.condition?.operator).toBe("contains_any")
    expect(this.conditionValues(networkEdge)).toContain("online_support")

    expect(securityEdge.data?.condition?.field).toBe("form.request_kind")
    expect(securityEdge.data?.condition?.operator).toBe("contains_any")
    expect(this.conditionValues(securityEdge)).toContain("security_compliance")
  }

  private async vpnService() {
    const data = await this.session.api<ServiceListResponse>("GET", "/api/v1/itsm/services?page=1&pageSize=100")
    const service = data.items.find((item) => item.code === "vpn-access-request")
    expect(service).toBeTruthy()
    return service!
  }

  private workflow(value: unknown): WorkflowJSON {
    expect(value && typeof value === "object" && !Array.isArray(value), `workflowJson is not an object: ${JSON.stringify(value)}`).toBeTruthy()
    return value as WorkflowJSON
  }

  private nodeByLabel(workflow: WorkflowJSON, label: string): WorkflowNode {
    const nodes = workflow.nodes ?? []
    const node = nodes.find((item) => item.data?.label === label)
    expect(node, `expected workflow node "${label}"; actual=${JSON.stringify(nodes.map((item) => item.data?.label))}`).toBeTruthy()
    return node!
  }

  private routeEdgeTo(workflow: WorkflowJSON, nodeId: string): WorkflowEdge {
    const edges = workflow.edges ?? []
    const edge = edges.find((item) => item.target === nodeId && item.data?.condition)
    expect(edge, `expected route edge to ${nodeId}; actual=${JSON.stringify(edges)}`).toBeTruthy()
    return edge!
  }

  private conditionValues(edge: WorkflowEdge): unknown[] {
    const value = edge.data?.condition?.value
    expect(Array.isArray(value), `condition value is not an array: ${JSON.stringify(value)}`).toBeTruthy()
    return value as unknown[]
  }
}
