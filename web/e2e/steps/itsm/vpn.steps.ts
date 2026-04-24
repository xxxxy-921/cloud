import { openLoginPage, devUsers } from "../../support/auth-behavior"
import { Given, Then, When } from "../../fixtures/agentic-itsm.fixture"

Given("Agentic ITSM 开发环境已经 seed-dev 并启动", async ({ page }) => {
  await openLoginPage(page)
})

Given("服务目录管理员已登录", async ({ agenticItsm }) => {
  await agenticItsm.loginAs(devUsers.itsmServiceManager)
})

When("他在服务目录打开 VPN 开通申请并生成参考路径", async ({ agenticItsm }) => {
  await agenticItsm.serviceCatalog().generateVPNWorkflowReference()
})

Then("VPN 参考路径包含申请填写、网络管理员处理和信息安全管理员处理", async ({ agenticItsm }) => {
  agenticItsm.expectCurrentActorHasNoRuntimeErrors()
})

Given("VPN 申请人已登录", async ({ agenticItsm }) => {
  await agenticItsm.loginAs(devUsers.vpnApplicant)
})

When("他在服务台提交线上支持类 VPN 申请", async ({ agenticItsm }) => {
  const ticket = await agenticItsm.serviceDesk().submitOnlineSupportVPNRequest()
  agenticItsm.rememberVPNTicket(ticket)
})

Then("系统生成 VPN 工单编号", async ({ agenticItsm }) => {
  agenticItsm.requireVPNTicket()
  agenticItsm.expectCurrentActorHasNoRuntimeErrors()
})

Given("信息部网络管理员已登录", async ({ agenticItsm }) => {
  await agenticItsm.loginAs(devUsers.vpnNetworkApprover)
})

When("他在我的待办中通过该 VPN 申请", async ({ agenticItsm }) => {
  await agenticItsm.ticketApproval().approveNetworkTodo(agenticItsm.requireVPNTicket())
})

Then("VPN 工单进入已完成状态", async ({ agenticItsm }) => {
  agenticItsm.expectCurrentActorHasNoRuntimeErrors()
})

When("他打开我的工单中的该 VPN 工单", async ({ agenticItsm }) => {
  await agenticItsm.myTickets().expectVPNRequestApproved(agenticItsm.requireVPNTicket())
})

Then("他可以看到 VPN 申请已审批通过", async ({ agenticItsm }) => {
  agenticItsm.expectCurrentActorHasNoRuntimeErrors()
})
