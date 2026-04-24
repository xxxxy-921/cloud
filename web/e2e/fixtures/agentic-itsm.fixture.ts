import { expect, type Page } from "@playwright/test"
import { createBdd, test as base } from "playwright-bdd"

import { MyTicketsActor } from "../actors/itsm/my-tickets"
import { ServiceCatalogActor } from "../actors/itsm/service-catalog"
import { ServiceDeskActor } from "../actors/itsm/service-desk"
import { AgenticITSMActorFactory, type AgenticITSMActorSession, type VPNTicketRef } from "../actors/itsm/session"
import { TicketApprovalActor } from "../actors/itsm/ticket-approval"
import { TicketEvidence } from "../support/ticket-evidence"

type AgenticITSMSharedState = {
  vpnTicket: VPNTicketRef | null
}

const sharedState: AgenticITSMSharedState = {
  vpnTicket: null,
}

export class AgenticITSMScenario {
  private readonly actorFactory: AgenticITSMActorFactory
  private session: AgenticITSMActorSession | null = null

  constructor(
    page: Page,
    private readonly state: AgenticITSMSharedState,
  ) {
    this.actorFactory = new AgenticITSMActorFactory(page)
  }

  async loginAs(user: Parameters<AgenticITSMActorFactory["loginAs"]>[0]) {
    this.session = await this.actorFactory.loginAs(user)
  }

  async close() {
    await this.actorFactory.close()
  }

  serviceCatalog() {
    return new ServiceCatalogActor(this.currentSession())
  }

  serviceDesk() {
    return new ServiceDeskActor(this.currentSession())
  }

  ticketApproval() {
    return new TicketApprovalActor(this.currentSession())
  }

  myTickets() {
    return new MyTicketsActor(this.currentSession())
  }

  ticketEvidence() {
    return new TicketEvidence(this.currentSession())
  }

  rememberVPNTicket(ticket: VPNTicketRef) {
    this.state.vpnTicket = ticket
  }

  requireVPNTicket() {
    expect(this.state.vpnTicket, "VPN 工单尚未创建").toBeTruthy()
    return this.state.vpnTicket!
  }

  expectCurrentActorHasNoRuntimeErrors() {
    this.currentSession().expectNoRuntimeErrors()
  }

  private currentSession() {
    expect(this.session, "当前场景尚未登录业务账号").toBeTruthy()
    return this.session!
  }
}

export const test = base.extend<{ agenticItsm: AgenticITSMScenario }>({
  agenticItsm: async ({ page }, run) => {
    const scenario = new AgenticITSMScenario(page, sharedState)
    try {
      await run(scenario)
    } finally {
      await scenario.close()
    }
  },
})

export const { Given, When, Then } = createBdd(test)
