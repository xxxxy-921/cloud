import { expect } from "@playwright/test"

type APIReader = {
  api<T>(method: string, path: string, body?: unknown): Promise<T>
}

export type TicketEvidenceRef = {
  ticketId: number
  ticketCode: string
}

type TicketItem = {
  id: number
  code: string
  title: string
  status: string
  serviceName: string
  currentActivityId: number | null
  currentOwnerName?: string
  formData: unknown
}

type TicketListResponse = {
  items: TicketItem[]
  total: number
}

type ActivityItem = {
  id: number
  name: string
  status: string
  canAct: boolean
}

type TimelineItem = {
  eventType: string
  message: string
  operatorName: string
  content: string
}

export class TicketEvidence {
  constructor(private readonly reader: APIReader) {}

  async expectServiceName(ticket: TicketEvidenceRef, serviceName: string) {
    const detail = await this.ticket(ticket)
    expect(detail.serviceName).toBe(serviceName)
  }

  async expectFormField(ticket: TicketEvidenceRef, field: string, value: unknown) {
    const detail = await this.ticket(ticket)
    const formData = this.formDataObject(detail.formData)
    expect(formData[field]).toBe(value)
  }

  async expectMyTicketListContains(ticket: TicketEvidenceRef) {
    const data = await this.reader.api<TicketListResponse>(
      "GET",
      `/api/v1/itsm/tickets/mine?keyword=${encodeURIComponent(ticket.ticketCode)}&page=1&pageSize=20`,
    )
    expect(data.items.some((item) => item.id === ticket.ticketId && item.code === ticket.ticketCode)).toBeTruthy()
  }

  async expectCurrentActivity(ticket: TicketEvidenceRef, activityName: string) {
    await expect(async () => {
      const activities = await this.activities(ticket)
      const activity = activities.find((item) => item.name === activityName && ["pending", "in_progress"].includes(item.status))
      expect(activity, this.activityDebugMessage(activities, activityName)).toBeTruthy()
    }).toPass({ timeout: 90_000, intervals: [1_000, 2_000, 3_000] })
  }

  async expectActionableActivity(ticket: TicketEvidenceRef, activityName: string) {
    await expect(async () => {
      const activities = await this.activities(ticket)
      const activity = activities.find((item) => item.name === activityName && item.canAct)
      expect(activity, this.activityDebugMessage(activities, activityName)).toBeTruthy()
    }).toPass({ timeout: 90_000, intervals: [1_000, 2_000, 3_000] })
  }

  async expectTicketStatus(ticket: TicketEvidenceRef, status: string) {
    await expect(async () => {
      const detail = await this.ticket(ticket)
      expect(detail.status).toBe(status)
    }).toPass({ timeout: 120_000, intervals: [1_000, 2_000, 3_000] })
  }

  async expectTimelineContains(ticket: TicketEvidenceRef, pattern: RegExp) {
    await expect(async () => {
      const timeline = await this.timeline(ticket)
      const text = timeline.map((item) => [item.eventType, item.operatorName, item.message, item.content].join(" ")).join("\n")
      expect(text).toMatch(pattern)
    }).toPass({ timeout: 60_000, intervals: [1_000, 2_000, 3_000] })
  }

  private async ticket(ticket: TicketEvidenceRef) {
    const detail = await this.reader.api<TicketItem>("GET", `/api/v1/itsm/tickets/${ticket.ticketId}`)
    expect(detail.code).toBe(ticket.ticketCode)
    return detail
  }

  private async activities(ticket: TicketEvidenceRef) {
    return await this.reader.api<ActivityItem[]>("GET", `/api/v1/itsm/tickets/${ticket.ticketId}/activities`)
  }

  private async timeline(ticket: TicketEvidenceRef) {
    return await this.reader.api<TimelineItem[]>("GET", `/api/v1/itsm/tickets/${ticket.ticketId}/timeline`)
  }

  private formDataObject(value: unknown) {
    expect(value && typeof value === "object" && !Array.isArray(value), `formData is not an object: ${JSON.stringify(value)}`).toBeTruthy()
    return value as Record<string, unknown>
  }

  private activityDebugMessage(activities: ActivityItem[], expectedName: string) {
    return `expected activity ${expectedName}; actual=${JSON.stringify(activities.map((item) => ({
      id: item.id,
      name: item.name,
      status: item.status,
      canAct: item.canAct,
    })))}`
  }
}
