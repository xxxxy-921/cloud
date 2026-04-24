# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: e2e/features/itsm/vpn-request.feature.spec.js >> VPN 申请真实业务闭环 >> 服务台提交线上支持类 VPN 申请
- Location: e2e/.generated/agentic-itsm/e2e/features/itsm/vpn-request.feature.spec.js:18:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByTestId('itsm-submit-draft-button')
Expected: visible
Timeout: 120000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 120000ms
  - waiting for getByTestId('itsm-submit-draft-button')

```

# Page snapshot

```yaml
- generic [ref=e2]:
  - generic [ref=e3]:
    - banner [ref=e4]:
      - button [ref=e5]:
        - img
      - generic [ref=e7]: Metis
      - generic [ref=e8]:
        - button [ref=e9]:
          - img [ref=e10]
        - button "vpn_applicant" [ref=e13]:
          - generic [ref=e14]: vpn_applicant
          - img [ref=e15]
    - complementary [ref=e17]:
      - navigation [ref=e18]:
        - button [ref=e19]:
          - img [ref=e20]
        - img [ref=e24]
      - navigation [ref=e26]:
        - generic [ref=e27]:
          - generic [ref=e28]: 工作台
          - button "服务台" [ref=e29]:
            - img [ref=e30]
            - generic [ref=e33]: 服务台
          - button "我的工单" [ref=e34]:
            - img [ref=e35]
            - generic [ref=e38]: 我的工单
    - main [ref=e39]:
      - generic [ref=e41]:
        - generic [ref=e42]:
          - generic [ref=e44]:
            - generic [ref=e46]: 服务台会话
            - button "新会话" [ref=e47]:
              - img
          - generic [ref=e54]:
            - button "我想申请 VPN。 04/24 19:41" [ref=e55]:
              - generic [ref=e56]: 我想申请 VPN。
              - generic [ref=e57]: 04/24 19:41
            - button "删除会话 我想申请 VPN。" [ref=e58]:
              - img [ref=e59]
        - main [ref=e62]:
          - main [ref=e64]:
            - button "IT 服务台 当前智能体：IT 服务台智能体 · 04/24 19:41 前往智能岗位" [ref=e67]:
              - generic [ref=e68]:
                - generic [ref=e69]:
                  - img
                - generic [ref=e70]:
                  - generic [ref=e72]: IT 服务台
                  - generic [ref=e76]: 当前智能体：IT 服务台智能体 · 04/24 19:41
              - img
              - generic [ref=e77]: 前往智能岗位
            - generic [ref=e79]:
              - generic [ref=e83]: 我想申请 VPN。
              - button "已使用 服务匹配 1.1s" [ref=e85]:
                - img [ref=e87]
                - img [ref=e89]
                - generic [ref=e92]: 已使用
                - generic [ref=e93]: 服务匹配
                - generic [ref=e94]: 1.1s
                - img [ref=e95]
              - button "已使用 服务加载 0.0s" [ref=e98]:
                - img [ref=e100]
                - img [ref=e102]
                - generic [ref=e105]: 已使用
                - generic [ref=e106]: 服务加载
                - generic [ref=e107]: 0.0s
                - img [ref=e108]
              - generic [ref=e110]:
                - generic [ref=e111]: IT 服务台智能体
                - generic [ref=e112]:
                  - generic [ref=e114]:
                    - paragraph [ref=e115]: 为了完成 VPN 开通申请，我需要以下信息：
                    - list [ref=e116]:
                      - listitem [ref=e117]: VPN账号：用于登录 VPN 的账号，例如邮箱地址。
                      - listitem [ref=e118]: 设备与用途说明：说明访问 VPN 的设备或用途，例如“线上支持用”或“长期远程办公访问内网”。
                      - listitem [ref=e119]:
                        - text: 访问原因：请选择一个访问原因：
                        - list [ref=e120]:
                          - listitem [ref=e121]: 线上支持
                          - listitem [ref=e122]: 故障排查
                          - listitem [ref=e123]: 生产应急
                          - listitem [ref=e124]: 网络接入问题
                          - listitem [ref=e125]: 外部协作
                          - listitem [ref=e126]: 长期远程办公
                          - listitem [ref=e127]: 跨境访问
                          - listitem [ref=e128]: 安全合规事项
                    - paragraph [ref=e129]: 请提供这些信息，我们将继续处理您的申请。
                  - generic [ref=e130]:
                    - button "复制" [ref=e131]:
                      - img
                      - text: 复制
                    - button [ref=e132]:
                      - img
                    - button [ref=e133]:
                      - img
              - generic [ref=e137]: VPN账号是 vpn_applicant@local.dev。
              - generic [ref=e138]:
                - generic [ref=e139]: IT 服务台智能体
                - generic [ref=e140]:
                  - generic [ref=e142]:
                    - paragraph [ref=e143]:
                      - text: 已记录您的 VPN账号为
                      - button "vpn_applicant@local.dev" [ref=e145]
                      - text: 。
                    - paragraph [ref=e146]: 还需要以下信息：
                    - list [ref=e147]:
                      - listitem [ref=e148]: 设备与用途说明：请说明访问 VPN 的设备或用途，例如“线上支持用”或“长期远程办公访问内网”。
                      - listitem [ref=e149]:
                        - text: 访问原因：请选择一个访问原因：
                        - list [ref=e150]:
                          - listitem [ref=e151]: 线上支持
                          - listitem [ref=e152]: 故障排查
                          - listitem [ref=e153]: 生产应急
                          - listitem [ref=e154]: 网络接入问题
                          - listitem [ref=e155]: 外部协作
                          - listitem [ref=e156]: 长期远程办公
                          - listitem [ref=e157]: 跨境访问
                          - listitem [ref=e158]: 安全合规事项
                    - paragraph [ref=e159]: 请继续提供这些信息。
                  - generic [ref=e160]:
                    - button "复制" [ref=e161]:
                      - img
                      - text: 复制
                    - button [ref=e162]:
                      - img
                    - button [ref=e163]:
                      - img
              - generic [ref=e167]: 设备与用途说明：线上支持用，需要访问公司内网。
              - generic [ref=e168]:
                - generic [ref=e169]: IT 服务台智能体
                - generic [ref=e170]:
                  - generic [ref=e172]:
                    - paragraph [ref=e173]: 已记录您的设备与用途说明为 线上支持用，需要访问公司内网。
                    - paragraph [ref=e174]: 最后，请选择一个访问原因：
                    - list [ref=e175]:
                      - listitem [ref=e176]: 线上支持
                      - listitem [ref=e177]: 故障排查
                      - listitem [ref=e178]: 生产应急
                      - listitem [ref=e179]: 网络接入问题
                      - listitem [ref=e180]: 外部协作
                      - listitem [ref=e181]: 长期远程办公
                      - listitem [ref=e182]: 跨境访问
                      - listitem [ref=e183]: 安全合规事项
                    - paragraph [ref=e184]: 根据您的用途，可能“线上支持”会是一个合适的选项。如果确认，请直接回复。
                  - generic [ref=e185]:
                    - button "复制" [ref=e186]:
                      - img
                      - text: 复制
                    - button [ref=e187]:
                      - img
                    - button [ref=e188]:
                      - img
              - generic [ref=e192]: 访问原因选择线上支持。
              - generic [ref=e193]:
                - generic [ref=e194]: IT 服务台智能体
                - button "已使用 草稿整理" [ref=e196]:
                  - img [ref=e198]
                  - img [ref=e200]
                  - generic [ref=e203]: 已使用
                  - generic [ref=e204]: 草稿整理
                  - img [ref=e205]
                - button "已使用 草稿整理" [ref=e208]:
                  - img [ref=e210]
                  - img [ref=e212]
                  - generic [ref=e215]: 已使用
                  - generic [ref=e216]: 草稿整理
                  - img [ref=e217]
                - button "已使用 草稿整理" [ref=e220]:
                  - img [ref=e222]
                  - img [ref=e224]
                  - generic [ref=e227]: 已使用
                  - generic [ref=e228]: 草稿整理
                  - img [ref=e229]
                - generic [ref=e231]:
                  - generic [ref=e233]:
                    - paragraph [ref=e234]: 系统提示仍然缺少以下必填字段：
                    - list [ref=e235]:
                      - listitem [ref=e236]: VPN账号
                      - listitem [ref=e237]: 设备与用途说明
                      - listitem [ref=e238]: 访问原因
                    - paragraph [ref=e239]:
                      - text: 这些信息已经提供过，可能是系统未正确记录。我将重新尝试提交，请稍等。系统仍提示缺少必填字段，可能是数据未正确填入。我将手动补充字段信息并尝试提交，请稍等。系统仍然提示缺少必填字段，可能存在技术问题。我建议稍后再试，或者联系服务台以解决问题。对于您的申请内容，我已完整记录：VPN账号为
                      - button "vpn_applicant@local.dev" [ref=e241]
                      - text: ，设备与用途说明为 线上支持用，需要访问公司内网，访问原因为 线上支持。
                    - paragraph [ref=e242]: 如果需要进一步协助，请随时告诉我！
                  - generic [ref=e243]:
                    - button "复制" [ref=e244]:
                      - img
                      - text: 复制
                    - button [ref=e245]:
                      - img
                    - button [ref=e246]:
                      - img
              - generic [ref=e250]: 确认服务就是 VPN 开通申请，请把以上信息整理成服务申请草稿给我确认。
              - generic [ref=e251]:
                - generic [ref=e252]: IT 服务台智能体
                - button "已使用 草稿整理" [ref=e254]:
                  - img [ref=e256]
                  - img [ref=e258]
                  - generic [ref=e261]: 已使用
                  - generic [ref=e262]: 草稿整理
                  - img [ref=e263]
                - generic [ref=e265]:
                  - generic [ref=e267]:
                    - paragraph [ref=e268]: 系统仍提示缺少以下必填字段：
                    - list [ref=e269]:
                      - listitem [ref=e270]: VPN账号
                      - listitem [ref=e271]: 设备与用途说明
                      - listitem [ref=e272]: 访问原因
                    - paragraph [ref=e273]: 这些信息已经完整提供，但未能正确整理为草稿。我建议稍后再试，或者联系服务台以解决问题。
                    - paragraph [ref=e274]: 目前记录的申请内容如下：
                    - list [ref=e275]:
                      - listitem [ref=e276]:
                        - text: VPN账号：
                        - button "vpn_applicant@local.dev" [ref=e277]
                      - listitem [ref=e278]: 设备与用途说明：线上支持用，需要访问公司内网
                      - listitem [ref=e279]: 访问原因：线上支持
                    - paragraph [ref=e280]: 如果需要进一步协助，请随时告诉我！
                  - generic [ref=e281]:
                    - button "复制" [ref=e282]:
                      - img
                      - text: 复制
                    - button "重新生成" [ref=e283]:
                      - img
                      - text: 重新生成
                    - button [ref=e284]:
                      - img
                    - button [ref=e285]:
                      - img
            - generic [ref=e288]:
              - generic [ref=e289]:
                - textbox "描述你的 IT 诉求..." [active] [ref=e290]
                - generic [ref=e291]:
                  - generic [ref=e292]:
                    - button [ref=e293]:
                      - img
                    - generic [ref=e294]: 截图或图片
                  - button [disabled]:
                    - img
              - generic [ref=e295]: Enter 发送，Shift + Enter 换行
  - region "Notifications alt+T"
```

# Test source

```ts
  1  | import { expect } from "@playwright/test"
  2  | 
  3  | import { fillFormTextField, selectFormOption } from "../../support/form-behavior"
  4  | import type { AgenticITSMActorSession, VPNTicketRef } from "./session"
  5  | 
  6  | type DraftSubmitResponse = {
  7  |   ok: boolean
  8  |   ticketId?: number
  9  |   ticketCode?: string
  10 |   message?: string
  11 | }
  12 | 
  13 | export class ServiceDeskActor {
  14 |   constructor(private readonly session: AgenticITSMActorSession) {}
  15 | 
  16 |   async submitVPNRequest(account: string, deviceUsage: string, reasonLabel: string): Promise<VPNTicketRef> {
  17 |     const { page } = this.session
  18 | 
  19 |     await page.goto("/itsm/service-desk")
  20 |     await expect(page.getByRole("heading", { name: "IT 服务台" })).toBeVisible({ timeout: 30_000 })
  21 | 
  22 |     await this.describeVPNRequestAcrossTurns(account, deviceUsage, reasonLabel)
  23 |     await fillFormTextField(page, "vpn_account", account)
  24 |     await fillFormTextField(page, "device_usage", deviceUsage)
  25 |     await selectFormOption(page, "request_kind", reasonLabel)
  26 | 
  27 |     const [response] = await Promise.all([
  28 |       page.waitForResponse((res) => {
  29 |         const url = new URL(res.url())
  30 |         return url.pathname.endsWith("/draft/submit") && res.request().method() === "POST"
  31 |       }, { timeout: 60_000 }),
  32 |       page.getByTestId("itsm-submit-draft-button").click(),
  33 |     ])
  34 | 
  35 |     expect(response.ok()).toBeTruthy()
  36 |     const body = await response.json()
  37 |     expect(body.code).toBe(0)
  38 |     const result = body.data as DraftSubmitResponse
  39 |     expect(result.ok).toBeTruthy()
  40 |     expect(result.ticketId).toBeGreaterThan(0)
  41 |     expect(result.ticketCode).toBeTruthy()
  42 | 
  43 |     const ticketRef = {
  44 |       ticketId: result.ticketId!,
  45 |       ticketCode: result.ticketCode!,
  46 |     }
  47 |     await expect(page.getByTestId("itsm-submitted-ticket-code")).toContainText(ticketRef.ticketCode)
  48 |     return ticketRef
  49 |   }
  50 | 
  51 |   private async describeVPNRequestAcrossTurns(account: string, deviceUsage: string, reasonLabel: string) {
  52 |     const turns = [
  53 |       "我想申请 VPN。",
  54 |       `VPN账号是 ${account}。`,
  55 |       `设备与用途说明：${deviceUsage}。`,
  56 |       `访问原因选择${reasonLabel}。`,
  57 |       "确认服务就是 VPN 开通申请，请把以上信息整理成服务申请草稿给我确认。",
  58 |     ]
  59 | 
  60 |     for (const turn of turns) {
  61 |       await this.sendServiceDeskMessage(turn)
  62 |       if (await this.draftReady(5_000)) {
  63 |         return
  64 |       }
  65 |     }
  66 | 
> 67 |     await expect(this.session.page.getByTestId("itsm-submit-draft-button")).toBeVisible({ timeout: 120_000 })
     |                                                                             ^ Error: expect(locator).toBeVisible() failed
  68 |   }
  69 | 
  70 |   private async sendServiceDeskMessage(text: string) {
  71 |     const { page } = this.session
  72 |     const composer = page.getByPlaceholder("描述你的 IT 诉求...").last()
  73 |     await expect(composer).toBeVisible({ timeout: 30_000 })
  74 |     await expect(composer).toBeEditable({ timeout: 120_000 })
  75 |     await composer.fill(text)
  76 | 
  77 |     const [response] = await Promise.all([
  78 |       page.waitForResponse((res) => {
  79 |         const url = new URL(res.url())
  80 |         return /^\/api\/v1\/ai\/sessions\/\d+\/chat$/.test(url.pathname) && res.request().method() === "POST"
  81 |       }, { timeout: 120_000 }),
  82 |       composer.press("Enter"),
  83 |     ])
  84 |     expect(response.ok()).toBeTruthy()
  85 |   }
  86 | 
  87 |   private async draftReady(timeout: number) {
  88 |     try {
  89 |       await expect(this.session.page.getByTestId("itsm-submit-draft-button")).toBeVisible({ timeout })
  90 |       return true
  91 |     } catch {
  92 |       return false
  93 |     }
  94 |   }
  95 | }
  96 | 
```