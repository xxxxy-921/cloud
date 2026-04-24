import { test } from "@playwright/test"

import { ServiceDeskSSEWorld } from "./support/service-desk-bdd"

test.describe("Feature: 服务台聊天流式输出", () => {
  test("Scenario: VPN 申请回复按 AI SDK UI Message Stream 增量渲染", async ({ page }) => {
    const serviceDesk = new ServiceDeskSSEWorld(page)

    try {
      await test.step("Given 管理员已登录且服务台智能体可用", () =>
        serviceDesk.givenAdminCanUseServiceDesk(),
      )

      await test.step("When 用户打开服务台并提交 VPN 申请", async () => {
        await serviceDesk.whenUserOpensServiceDesk()
        await serviceDesk.whenUserSubmitsVPNRequest()
      })

      await test.step("Then 前端通过 /chat 发起标准 AI SDK 流式请求", () =>
        serviceDesk.thenChatEndpointReceivesTheMessage(),
      )

      await test.step("And 首个 delta 到达时立即显示部分正文", () =>
        serviceDesk.thenFirstDeltaAppearsBeforeTheFinalAnswer(),
      )

      await test.step("And 工具状态从运行态切到完成态", () =>
        serviceDesk.thenToolActivityTransitionsFromRunningToCompleted(),
      )

      await test.step("And 草稿表单出现时 suppress 文本并渲染表单", () =>
        serviceDesk.thenDraftSurfaceSuppressesAssistantTextAndShowsForm(),
      )
    } finally {
      await serviceDesk.cleanup()
    }
  })
})
