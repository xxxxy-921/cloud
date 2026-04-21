"use client"

import { useEffect, useMemo, useState } from "react"
import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  AlertTriangle,
  Bot,
  CheckCircle2,
  ChevronDown,
  Clock,
  Gauge,
  Loader2,
  Route,
  ShieldCheck,
  UserRound,
  Wrench,
  XCircle,
} from "lucide-react"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Progress } from "@/components/ui/progress"
import { Separator } from "@/components/ui/separator"
import { type ActivityItem, type TicketItem, fetchTicket, progressTicket } from "../api"
import { AIDecisionPanel } from "./ai-decision-panel"
import { FormDataDisplay } from "./form-data-display"
import { OverrideActions } from "./override-actions"

const TERMINAL_STATUSES = new Set(["completed", "cancelled", "failed"])
const HUMAN_ACTIVITY_TYPES = new Set(["approve", "form", "process"])
const MAX_AI_FAILURES = 3
const POLL_INTERVAL = 3000
const POLL_TIMEOUT = 60000

type SmartState =
  | "terminal"
  | "ai_disabled"
  | "waiting_ai_confirmation"
  | "waiting_human"
  | "action_running"
  | "ai_reasoning"
  | "ai_decided"
  | "idle"

interface SmartCurrentActivityCardProps {
  ticket: TicketItem
  activities: ActivityItem[]
  currentUserId: number
  canOverride?: boolean
}

interface DecisionActivity {
  type?: string
  participant_type?: string
  participant_id?: number
  position_code?: string
  department_code?: string
  action_id?: number
  instructions?: string
}

interface DecisionPlan {
  next_step_type?: string
  execution_mode?: string
  activities?: DecisionActivity[]
  reasoning?: string
  confidence?: number
  evidence?: unknown[]
  tool_calls?: unknown[]
  knowledge_hits?: unknown[]
  action_executions?: unknown[]
  risk_flags?: unknown[]
}

function determineState(ticket: TicketItem, activities: ActivityItem[]): SmartState {
  if (TERMINAL_STATUSES.has(ticket.status)) return "terminal"
  if (ticket.aiFailureCount >= MAX_AI_FAILURES) return "ai_disabled"
  if (ticket.smartState) return ticket.smartState as SmartState

  const currentActivity = activities.find((a) => a.id === ticket.currentActivityId)
  if (currentActivity?.status === "pending_approval") return "waiting_ai_confirmation"

  if (currentActivity?.activityType === "action" && ["pending", "in_progress"].includes(currentActivity.status)) {
    return "action_running"
  }

  const activeHumanActivity = activities.find(
    (a) => ["pending", "in_progress"].includes(a.status) && HUMAN_ACTIVITY_TYPES.has(a.activityType),
  )
  if (activeHumanActivity) return "waiting_human"

  const hasActiveActivity = activities.some(
    (a) => a.status === "pending" || a.status === "in_progress" || a.status === "pending_approval",
  )
  if (!hasActiveActivity) return "ai_reasoning"

  return "idle"
}

function parseDecision(activity?: ActivityItem | null): DecisionPlan | null {
  if (!activity?.aiDecision) return null
  try {
    return JSON.parse(activity.aiDecision) as DecisionPlan
  } catch {
    return null
  }
}

function summarizeDecision(plan: DecisionPlan | null, fallback?: string) {
  const first = plan?.activities?.[0]
  if (!plan) return fallback || "等待 AI 生成决策计划"
  if (first?.instructions) return first.instructions
  if (first?.type) return `${first.type}${plan.execution_mode ? ` · ${plan.execution_mode}` : ""}`
  return plan.next_step_type || fallback || "AI 已形成下一步建议"
}

function confidenceOf(activity?: ActivityItem | null, plan?: DecisionPlan | null) {
  if (activity?.aiConfidence != null) return activity.aiConfidence
  if (plan?.confidence != null) return plan.confidence
  return null
}

function toRecord(value: unknown) {
  return value != null && typeof value === "object" && !Array.isArray(value)
    ? value as Record<string, unknown>
    : null
}

function compactValue(value: unknown) {
  if (value == null || value === "") return "—"
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") return String(value)
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function formatOwner(ticket: TicketItem, activity?: ActivityItem | null) {
  if (ticket.currentOwnerName) return ticket.currentOwnerName
  if (ticket.assigneeName) return ticket.assigneeName
  if (activity?.status === "pending_approval") return "决策确认人"
  if (activity?.activityType === "action") return "自动化动作"
  return "AI 智能引擎"
}

function buildStateView(state: SmartState, ticket: TicketItem) {
  switch (state) {
    case "terminal":
      return {
        icon: CheckCircle2,
        title: ticket.status === "completed" ? "工单已完成" : "工单已结束",
        sentence: ticket.status === "completed" ? "流程已闭环，时间线保留完整审计记录。" : "流程已结束，可查看审计记录。",
        tone: "border-emerald-200 bg-emerald-50 text-emerald-800",
      }
    case "ai_disabled":
      return {
        icon: AlertTriangle,
        title: "AI 熔断，需人工接管",
        sentence: `AI 已连续失败 ${ticket.aiFailureCount} 次，自动推进已暂停。`,
        tone: "border-amber-200 bg-amber-50 text-amber-800",
      }
    case "waiting_ai_confirmation":
      return {
        icon: ShieldCheck,
        title: "等待 AI 决策确认",
        sentence: "AI 已给出低置信建议，需要有权限的处理人确认后才会推进。",
        tone: "border-yellow-200 bg-yellow-50 text-yellow-800",
      }
    case "waiting_human":
      return {
        icon: UserRound,
        title: "等待人工处理",
        sentence: `当前责任人：${formatOwner(ticket)}。`,
        tone: "border-blue-200 bg-blue-50 text-blue-800",
      }
    case "action_running":
      return {
        icon: Wrench,
        title: "自动化动作执行中",
        sentence: "系统正在执行 AI 触发的动作，执行结果会进入活动轨迹与时间线。",
        tone: "border-cyan-200 bg-cyan-50 text-cyan-800",
      }
    case "ai_decided":
      return {
        icon: Route,
        title: "AI 已决策",
        sentence: "AI 已形成下一步计划，正在等待执行结果或下一轮推进。",
        tone: "border-violet-200 bg-violet-50 text-violet-800",
      }
    case "ai_reasoning":
      return {
        icon: Bot,
        title: "AI 正在分析下一步",
        sentence: "智能引擎正在读取上下文、协作规范和历史轨迹。",
        tone: "border-sky-200 bg-sky-50 text-sky-800",
      }
    default:
      return {
        icon: Clock,
        title: "等待下一步",
        sentence: "系统正在同步当前活动状态。",
        tone: "border-border bg-muted/40 text-foreground",
      }
  }
}

function DetailItem({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="rounded-md border bg-background/50 p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  )
}

function EvidenceList({ title, items }: { title: string; items?: unknown[] }) {
  if (!items?.length) return null
  return (
    <div className="space-y-2">
      <p className="text-sm font-medium">{title}</p>
      <div className="space-y-2">
        {items.slice(0, 5).map((item, idx) => (
          <div key={idx} className="rounded-md border bg-background/60 p-3 text-xs text-muted-foreground">
            {compactValue(item)}
          </div>
        ))}
      </div>
    </div>
  )
}

function AICockpit({
  ticket,
  activity,
  plan,
}: {
  ticket: TicketItem
  activity?: ActivityItem | null
  plan: DecisionPlan | null
}) {
  const [open, setOpen] = useState(false)
  const formRecord = toRecord(ticket.formData)
  const firstActivity = plan?.activities?.[0]
  const confidence = confidenceOf(activity, plan)
  const confidencePct = confidence == null ? null : Math.round(confidence * 100)

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-center gap-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <Bot className="h-4 w-4" />
            AI 驾驶舱
          </CardTitle>
          <Badge variant="outline" className="ml-auto">
            默认摘要，可展开证据
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 md:grid-cols-3">
          <DetailItem
            label="AI 依据"
            value={activity?.aiReasoning ? "已记录推理摘要" : formRecord ? "表单字段 + 运行轨迹" : "运行轨迹"}
          />
          <DetailItem
            label="下一步计划"
            value={summarizeDecision(plan, ticket.nextStepSummary)}
          />
          <DetailItem
            label="置信度"
            value={confidencePct == null ? "—" : `${confidencePct}%`}
          />
        </div>

        {confidencePct != null && (
          <div className="space-y-1.5">
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <span>置信度边界</span>
              <span>{confidencePct >= 80 ? "高置信自动推进" : confidencePct >= 50 ? "中置信观察" : "低置信需确认"}</span>
            </div>
            <Progress value={confidencePct} className="h-2" />
          </div>
        )}

        <Collapsible open={open} onOpenChange={setOpen}>
          <CollapsibleTrigger asChild>
            <Button variant="ghost" size="sm" className="px-0">
              <ChevronDown className={cn("mr-1 h-4 w-4 transition-transform", open && "rotate-180")} />
              {open ? "收起 AI 证据" : "展开 AI 证据"}
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent className="space-y-4 pt-2">
            <Separator />
            {formRecord && (
              <div className="space-y-2">
                <p className="text-sm font-medium">使用的表单字段</p>
                <div className="grid gap-2 md:grid-cols-2">
                  {Object.entries(formRecord).slice(0, 8).map(([key, value]) => (
                    <div key={key} className="rounded-md border bg-background/60 p-3 text-xs">
                      <span className="text-muted-foreground">{key}</span>
                      <p className="mt-1 truncate font-medium">{compactValue(value)}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {activity?.aiReasoning && (
              <div className="space-y-2">
                <p className="text-sm font-medium">AI 推理摘要</p>
                <p className="whitespace-pre-wrap rounded-md border bg-background/60 p-3 text-sm text-muted-foreground">
                  {activity.aiReasoning}
                </p>
              </div>
            )}

            {firstActivity && (
              <div className="space-y-2">
                <p className="text-sm font-medium">结构化计划</p>
                <div className="grid gap-2 md:grid-cols-2">
                  <DetailItem label="步骤类型" value={firstActivity.type || plan?.next_step_type || "—"} />
                  <DetailItem label="执行模式" value={plan?.execution_mode || "single"} />
                  <DetailItem label="参与者" value={firstActivity.participant_type || firstActivity.participant_id || "—"} />
                  <DetailItem label="动作 ID" value={firstActivity.action_id || "—"} />
                </div>
              </div>
            )}

            <EvidenceList title="知识库命中" items={plan?.knowledge_hits} />
            <EvidenceList title="工具调用" items={plan?.tool_calls} />
            <EvidenceList title="动作执行" items={plan?.action_executions} />
            <EvidenceList title="风险标记" items={plan?.risk_flags} />

            {!formRecord && !activity?.aiReasoning && !plan && (
              <p className="text-sm text-muted-foreground">暂无可展开的结构化证据。</p>
            )}
          </CollapsibleContent>
        </Collapsible>
      </CardContent>
    </Card>
  )
}

export function SmartCurrentActivityCard({
  ticket,
  activities,
  currentUserId,
  canOverride = false,
}: SmartCurrentActivityCardProps) {
  const { t } = useTranslation("itsm")
  const queryClient = useQueryClient()
  const state = determineState(ticket, activities)
  const stateView = buildStateView(state, ticket)
  const StateIcon = stateView.icon
  const pollKey = `${ticket.id}:${ticket.currentActivityId ?? "none"}:${ticket.updatedAt}:${state}`
  const [timedOutRun, setTimedOutRun] = useState<string | null>(null)

  useEffect(() => {
    if (state !== "ai_reasoning") return
    const pollRun = Date.now()
    const currentPollKey = pollKey
    const interval = window.setInterval(async () => {
      if (Date.now() - pollRun > POLL_TIMEOUT) {
        setTimedOutRun(currentPollKey)
        window.clearInterval(interval)
        return
      }
      try {
        const fresh = await fetchTicket(ticket.id)
        if (fresh) {
          queryClient.setQueryData(["itsm-ticket", ticket.id], fresh)
          queryClient.invalidateQueries({ queryKey: ["itsm-ticket-activities", ticket.id] })
          queryClient.invalidateQueries({ queryKey: ["itsm-ticket-timeline", ticket.id] })
        }
      } catch {
        // polling is opportunistic; the explicit refresh button remains available.
      }
    }, POLL_INTERVAL)
    return () => window.clearInterval(interval)
  }, [state, ticket.id, queryClient, pollKey])

  const currentActivity = activities.find((a) => a.id === ticket.currentActivityId)
  const activeHumanActivity = activities.find(
    (a) => ["pending", "in_progress"].includes(a.status) && HUMAN_ACTIVITY_TYPES.has(a.activityType),
  )
  const explanationActivity = currentActivity ?? [...activities].reverse().find((a) => a.aiDecision || a.aiReasoning)
  const plan = useMemo(() => parseDecision(explanationActivity), [explanationActivity])
  const confidence = confidenceOf(explanationActivity, plan)
  const confidencePct = confidence == null ? null : Math.round(confidence * 100)
  const pollTimedOut = state === "ai_reasoning" && timedOutRun === pollKey
  const ownerName = formatOwner(ticket, currentActivity ?? activeHumanActivity)
  const ownerType = ticket.currentOwnerType || currentActivity?.activityType || "ai"
  const nextStep = ticket.nextStepSummary || currentActivity?.name || activeHumanActivity?.name || summarizeDecision(plan)
  const isCurrentUserResponsible = Boolean(ticket.canAct || activeHumanActivity?.canAct || ticket.assigneeId === currentUserId)
  const showOverride = canOverride && !TERMINAL_STATUSES.has(ticket.status)

  const terminalDuration = ticket.finishedAt
    ? Math.round((new Date(ticket.finishedAt).getTime() - new Date(ticket.createdAt).getTime()) / 60000)
    : null
  const durationDisplay = terminalDuration == null
    ? null
    : terminalDuration >= 60
      ? `${Math.floor(terminalDuration / 60)} ${t("smart.hours", { defaultValue: "小时" })} ${terminalDuration % 60} ${t("smart.minutes")}`
      : `${terminalDuration} ${t("smart.minutes")}`

  return (
    <div className="space-y-4">
      <div className={cn("rounded-md border px-4 py-3 text-sm", stateView.tone)}>
        <div className="flex flex-wrap items-center gap-2">
          <StateIcon className="h-4 w-4" />
          <span className="font-medium">{stateView.title}</span>
          <span className="text-current/80">{stateView.sentence}</span>
        </div>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex flex-wrap items-start gap-3">
            <div className="space-y-1">
              <CardTitle className="flex items-center gap-2 text-base">
                <Gauge className="h-4 w-4" />
                当前步骤主卡片
              </CardTitle>
              <p className="text-sm text-muted-foreground">谁在驾驶、下一步是什么、你能做什么。</p>
            </div>
            <Badge variant="outline" className="ml-auto">
              {stateView.title}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 md:grid-cols-4">
            <DetailItem label="当前责任方" value={ownerName} />
            <DetailItem label="责任类型" value={ownerType} />
            <DetailItem label="下一步" value={nextStep} />
            <DetailItem label="SLA 风险" value={ticket.slaStatus || "未命中"} />
          </div>

          {confidencePct != null && (
            <div className="flex flex-wrap items-center gap-3 rounded-md border bg-background/60 p-3">
              <div className="min-w-32">
                <p className="text-xs text-muted-foreground">AI 置信度</p>
                <p className="text-sm font-medium">{confidencePct}%</p>
              </div>
              <Progress value={confidencePct} className="h-2 flex-1" />
              <Badge variant={confidencePct >= 80 ? "default" : confidencePct >= 50 ? "secondary" : "destructive"}>
                {confidencePct >= 80 ? "可自动推进" : confidencePct >= 50 ? "需观察" : "需人工确认"}
              </Badge>
            </div>
          )}

          {state === "ai_reasoning" && (
            <Alert className="border-sky-200 bg-sky-50 text-sky-800 [&>svg]:text-sky-700">
              {pollTimedOut ? <AlertTriangle className="h-4 w-4" /> : <Loader2 className="h-4 w-4 animate-spin" />}
              <AlertDescription className="flex flex-wrap items-center gap-2">
                {pollTimedOut ? t("smart.aiReasoningTimeout") : t("smart.aiReasoningDesc")}
                {pollTimedOut && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setTimedOutRun(null)
                      queryClient.invalidateQueries({ queryKey: ["itsm-ticket", ticket.id] })
                      queryClient.invalidateQueries({ queryKey: ["itsm-ticket-activities", ticket.id] })
                    }}
                  >
                    {t("smart.refresh")}
                  </Button>
                )}
              </AlertDescription>
            </Alert>
          )}

          {state === "ai_disabled" && (
            <Alert variant="destructive" className="border-amber-200 bg-amber-50 text-amber-800 [&>svg]:text-amber-600">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>{t("smart.aiDisabledDesc", { count: ticket.aiFailureCount })}</AlertDescription>
            </Alert>
          )}

          {state === "terminal" && durationDisplay && (
            <div className="rounded-md border bg-background/60 p-3 text-sm">
              <span className="text-muted-foreground">{t("smart.processDuration")}</span>
              <span className="ml-2 font-medium">{durationDisplay}</span>
            </div>
          )}

          {activeHumanActivity?.formData ? <FormDataDisplay data={activeHumanActivity.formData} /> : null}

          <div className="flex flex-wrap gap-2 border-t pt-4">
            {state === "waiting_human" && activeHumanActivity && isCurrentUserResponsible && (
              <HumanActivityActions ticketId={ticket.id} activity={activeHumanActivity} />
            )}
            {state === "waiting_human" && !isCurrentUserResponsible && (
              <p className="text-sm text-muted-foreground">当前步骤正在等待责任人处理。</p>
            )}
            {showOverride && (
              <OverrideActions
                ticketId={ticket.id}
                currentActivityId={ticket.currentActivityId}
                aiFailureCount={ticket.aiFailureCount}
              />
            )}
            {!showOverride && state !== "waiting_human" && state !== "terminal" && (
              <p className="text-sm text-muted-foreground">你可以查看 AI 依据；需要接管时请联系管理员。</p>
            )}
          </div>
        </CardContent>
      </Card>

      {state === "waiting_ai_confirmation" && currentActivity && (
        <AIDecisionPanel ticketId={ticket.id} activity={currentActivity} />
      )}

      <AICockpit ticket={ticket} activity={explanationActivity} plan={plan} />
    </div>
  )
}

function HumanActivityActions({ ticketId, activity }: { ticketId: number; activity: ActivityItem }) {
  const { t } = useTranslation("itsm")
  const queryClient = useQueryClient()

  const progressMut = useMutation({
    mutationFn: (outcome: string) => progressTicket(ticketId, { activityId: activity.id, outcome }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["itsm-ticket", ticketId] })
      queryClient.invalidateQueries({ queryKey: ["itsm-ticket-activities", ticketId] })
      queryClient.invalidateQueries({ queryKey: ["itsm-ticket-timeline", ticketId] })
      toast.success(t("tickets.progressSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  if (activity.activityType === "approve") {
    return (
      <>
        <Button size="sm" onClick={() => progressMut.mutate("approved")} disabled={progressMut.isPending}>
          <CheckCircle2 className="mr-1 h-3.5 w-3.5" />
          {t("approval.approve")}
        </Button>
        <Button size="sm" variant="outline" onClick={() => progressMut.mutate("rejected")} disabled={progressMut.isPending}>
          <XCircle className="mr-1 h-3.5 w-3.5" />
          {t("approval.deny")}
        </Button>
      </>
    )
  }

  return (
    <Button size="sm" onClick={() => progressMut.mutate("completed")} disabled={progressMut.isPending}>
      <CheckCircle2 className="mr-1 h-3.5 w-3.5" />
      {t("smart.submit")}
    </Button>
  )
}
