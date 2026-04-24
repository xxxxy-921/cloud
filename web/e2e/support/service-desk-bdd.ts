import http from "node:http"
import { expect, type Page, type Route } from "@playwright/test"

type StreamServer = {
  url: string
  close: () => Promise<void>
}

type UIMessageChunk = Record<string, unknown>

function apiOK(data: unknown) {
  return {
    code: 0,
    message: "ok",
    data,
  }
}

function sse(chunk: UIMessageChunk) {
  return `data: ${JSON.stringify(chunk)}\n\n`
}

async function startServiceDeskStreamServer() {
  const server = http.createServer((_req, res) => {
    res.writeHead(200, {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      Connection: "keep-alive",
      "X-Accel-Buffering": "no",
      "X-Vercel-AI-UI-Message-Stream": "v1",
      "Access-Control-Allow-Origin": "*",
    })

    const chunks = [
      sse({ type: "start", messageId: "msg-e2e" }),
      sse({ type: "text-start", id: "text-e2e" }),
      sse({ type: "text-delta", id: "text-e2e", delta: "正在" }),
      sse({ type: "text-delta", id: "text-e2e", delta: "为你" }),
      sse({
        type: "tool-input-available",
        toolCallId: "tool-service-match",
        toolName: "itsm.service_match",
        input: { query: "VPN" },
      }),
      sse({
        type: "tool-output-available",
        toolCallId: "tool-service-match",
        output: { ok: true, serviceId: 7 },
      }),
      sse({ type: "text-delta", id: "text-e2e", delta: "准备 VPN 申请草稿" }),
      sse({
        type: "data-ui-surface",
        data: {
          surfaceId: "draft-e2e",
          surfaceType: "itsm.draft_form",
          payload: {
            status: "draft",
            draftVersion: 1,
            title: "VPN 开通申请",
            summary: "请确认 VPN 开通信息。",
            values: {
              vpnAccount: "admin",
            },
            schema: {
              version: 1,
              fields: [
                {
                  key: "vpnAccount",
                  type: "text",
                  label: "VPN 账号",
                  required: true,
                },
              ],
            },
          },
        },
      }),
      sse({ type: "text-end", id: "text-e2e" }),
      sse({
        type: "message-metadata",
        messageMetadata: {
          usage: {
            promptTokens: 12,
            completionTokens: 8,
          },
        },
      }),
      sse({ type: "finish", finishReason: "stop" }),
      "data: [DONE]\n\n",
    ]

    let index = 0
    const writeNext = () => {
      if (index >= chunks.length) {
        res.end()
        return
      }
      res.write(chunks[index])
      index += 1
      setTimeout(writeNext, 120)
    }
    writeNext()
  })

  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve))
  const address = server.address()
  if (!address || typeof address === "string") {
    server.close()
    throw new Error("failed to start stream server")
  }

  return {
    url: `http://127.0.0.1:${address.port}/chat`,
    close: () => new Promise<void>((resolve) => server.close(() => resolve())),
  }
}

export class ServiceDeskSSEWorld {
  private streamServer: StreamServer | null = null
  private chatRequestCount = 0

  constructor(private readonly page: Page) {}

  async cleanup() {
    await this.streamServer?.close()
    this.streamServer = null
  }

  async givenAdminCanUseServiceDesk() {
    await this.page.addInitScript(() => {
      localStorage.setItem("metis_access_token", "e2e-access")
      localStorage.setItem("metis_refresh_token", "e2e-refresh")
    })
    this.streamServer = await startServiceDeskStreamServer()
    await this.page.route("**/api/v1/**", (route) => this.routeAPI(route))
  }

  async whenUserOpensServiceDesk() {
    await this.page.goto("/itsm/service-desk")
    await expect(this.page.getByText("IT 服务台")).toBeVisible()
  }

  async whenUserSubmitsVPNRequest() {
    const composer = this.page.getByPlaceholder("描述你的 IT 诉求...")
    await composer.fill("我想申请 VPN，线上支持用\n\nadmin\npassword")
    await composer.press("Enter")
  }

  async thenChatEndpointReceivesTheMessage() {
    await expect.poll(() => this.chatRequestCount).toBe(1)
  }

  async thenFirstDeltaAppearsBeforeTheFinalAnswer() {
    await expect(this.page.getByText("正在")).toBeVisible()
    await expect(this.page.getByText("正在为你准备 VPN 申请草稿")).toHaveCount(0, {
      timeout: 80,
    })
  }

  async thenToolActivityTransitionsFromRunningToCompleted() {
    const toolActivity = this.page.getByTestId("chat-tool-activity")
    await expect(toolActivity).toHaveAttribute("data-status", "running")
    await expect(toolActivity).toHaveAttribute("data-status", "completed")
  }

  async thenDraftSurfaceSuppressesAssistantTextAndShowsForm() {
    await expect(this.page.getByTestId("itsm-draft-form-surface")).toBeVisible()
    await expect(this.page.getByText("VPN 账号")).toBeVisible()
    await expect(this.page.getByText("正在为你准备 VPN 申请草稿")).toHaveCount(0)
  }

  private async routeAPI(route: Route) {
    const request = route.request()
    const url = new URL(request.url())

    if (url.pathname === "/api/v1/ai/sessions/101/chat") {
      this.chatRequestCount += 1
      await route.continue({ url: this.requireStreamServer().url })
      return
    }

    if (url.pathname === "/api/v1/install/status") {
      await route.fulfill({ json: apiOK({ installed: true }) })
      return
    }
    if (url.pathname === "/api/v1/site-info") {
      await route.fulfill({
        json: apiOK({
          appName: "Metis",
          hasLogo: false,
          locale: "zh-CN",
          timezone: "Asia/Shanghai",
          version: "e2e",
          gitCommit: "e2e",
          buildTime: "2026-04-24T00:00:00Z",
        }),
      })
      return
    }
    if (url.pathname === "/api/v1/auth/me") {
      await route.fulfill({
        json: apiOK({
          user: {
            id: 1,
            username: "admin",
            nickname: "Admin",
            locale: "zh-CN",
            timezone: "Asia/Shanghai",
            twoFactorEnabled: true,
          },
        }),
      })
      return
    }
    if (url.pathname === "/api/v1/menus/user-tree") {
      await route.fulfill({
        json: apiOK({
          permissions: ["itsm:service-desk:use"],
          menus: [
            {
              id: 1,
              parentId: null,
              name: "ITSM",
              type: "directory",
              path: "",
              icon: "bot",
              permission: "itsm",
              sort: 1,
              isHidden: false,
              children: [
                {
                  id: 2,
                  parentId: 1,
                  name: "服务台",
                  type: "menu",
                  path: "/itsm/service-desk",
                  icon: "message-square",
                  permission: "itsm:service-desk:use",
                  sort: 1,
                  isHidden: false,
                  children: [],
                },
              ],
            },
          ],
        }),
      })
      return
    }
    if (url.pathname === "/api/v1/notifications/unread-count") {
      await route.fulfill({ json: apiOK({ count: 0 }) })
      return
    }
    if (url.pathname === "/api/v1/itsm/smart-staffing/config") {
      await route.fulfill({
        json: apiOK({
          posts: {
            intake: { agentId: 10, agentName: "IT 服务台智能体" },
            decision: { agentId: 0, agentName: "", mode: "manual" },
            slaAssurance: { agentId: 0, agentName: "" },
          },
          health: { items: [] },
        }),
      })
      return
    }
    if (url.pathname === "/api/v1/ai/sessions" && request.method() === "GET") {
      await route.fulfill({ json: apiOK({ items: [], total: 0 }) })
      return
    }
    if (url.pathname === "/api/v1/ai/sessions" && request.method() === "POST") {
      await route.fulfill({
        json: apiOK({
          id: 101,
          agentId: 10,
          userId: 1,
          status: "completed",
          title: "服务台会话",
          pinned: false,
          createdAt: "2026-04-24T06:00:00Z",
          updatedAt: "2026-04-24T06:00:00Z",
        }),
      })
      return
    }
    if (url.pathname === "/api/v1/ai/sessions/101") {
      await route.fulfill({
        json: apiOK({
          session: {
            id: 101,
            agentId: 10,
            userId: 1,
            status: "completed",
            title: "服务台会话",
            pinned: false,
            createdAt: "2026-04-24T06:00:00Z",
            updatedAt: "2026-04-24T06:00:00Z",
          },
          messages: [],
        }),
      })
      return
    }

    await route.fulfill({ json: apiOK(null) })
  }

  private requireStreamServer() {
    if (!this.streamServer) {
      throw new Error("stream server is not ready")
    }
    return this.streamServer
  }
}
