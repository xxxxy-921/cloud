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

Then("VPN 参考路径按访问原因路由到网络管理员和信息安全管理员", async ({ agenticItsm }) => {
  await agenticItsm.serviceCatalog().expectVPNWorkflowRoutesToNetworkAndSecurity()
  agenticItsm.expectCurrentActorHasNoRuntimeErrors()
})

Given("VPN 申请人已登录", async ({ agenticItsm }) => {
  await agenticItsm.loginAs(devUsers.vpnApplicant)
})

When("他在服务台提交 VPN账号 {string}、设备用途 {string}、访问原因 {string}", async ({ agenticItsm }, account: string, deviceUsage: string, reasonLabel: string) => {
  const ticket = await agenticItsm.serviceDesk().submitVPNRequest(account, deviceUsage, reasonLabel)
  agenticItsm.rememberVPNTicket(ticket)
})

Then("系统生成 VPN 工单编号", async ({ agenticItsm }) => {
  agenticItsm.requireVPNTicket()
  agenticItsm.expectCurrentActorHasNoRuntimeErrors()
})

Then("工单服务为 {string}", async ({ agenticItsm }, serviceName: string) => {
  await agenticItsm.ticketEvidence().expectServiceName(agenticItsm.requireVPNTicket(), serviceName)
})

Then("工单表单记录访问原因 {string}", async ({ agenticItsm }, requestKind: string) => {
  await agenticItsm.ticketEvidence().expectFormField(agenticItsm.requireVPNTicket(), "request_kind", requestKind)
})

Given("信息部网络管理员已登录", async ({ agenticItsm }) => {
  await agenticItsm.loginAs(devUsers.vpnNetworkApprover)
})

When("他打开我的待办中的该 VPN 工单", async ({ agenticItsm }) => {
  await agenticItsm.ticketApproval().openNetworkTodo(agenticItsm.requireVPNTicket())
})

Then("当前处理节点为 {string}", async ({ agenticItsm }, activityName: string) => {
  await agenticItsm.ticketApproval().expectCurrentNode(agenticItsm.requireVPNTicket(), activityName)
})

When("他通过该 VPN 申请并填写处理意见 {string}", async ({ agenticItsm }, opinion: string) => {
  await agenticItsm.ticketApproval().approveOpenedVPNRequest(agenticItsm.requireVPNTicket(), opinion)
})

Then("VPN 工单进入已完成状态", async ({ agenticItsm }) => {
  await agenticItsm.ticketEvidence().expectTicketStatus(agenticItsm.requireVPNTicket(), "completed")
  agenticItsm.expectCurrentActorHasNoRuntimeErrors()
})

When("他打开我的工单中的该 VPN 工单", async ({ agenticItsm }) => {
  await agenticItsm.ticketEvidence().expectMyTicketListContains(agenticItsm.requireVPNTicket())
  await agenticItsm.myTickets().openVPNRequest(agenticItsm.requireVPNTicket())
})

Then("他可以看到 VPN 申请已审批通过", async ({ agenticItsm }) => {
  await agenticItsm.myTickets().expectOpenedVPNRequestApproved(agenticItsm.requireVPNTicket())
  agenticItsm.expectCurrentActorHasNoRuntimeErrors()
})

Then("工单时间线包含网络管理员通过记录", async ({ agenticItsm }) => {
  await agenticItsm.ticketEvidence().expectTimelineContains(agenticItsm.requireVPNTicket(), /vpn_network_approver|网络管理员|同意开通 VPN|approved|通过/)
})
