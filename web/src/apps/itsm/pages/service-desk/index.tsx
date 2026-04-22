"use client"

import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useNavigate } from "react-router"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import type { UIMessage } from "ai"
import {
  AlertTriangle,
  Bot,
  CheckCircle2,
  History,
  Loader2,
  Plus,
  Send,
  Sparkles,
  Square,
} from "lucide-react"
import { toast } from "sonner"

import { QAPair } from "@/apps/ai/pages/chat/components/message-item"
import { useAiChat } from "@/apps/ai/pages/chat/hooks/use-ai-chat"
import { sessionApi, type AgentSession } from "@/lib/api"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { FormRenderer, type FormSchema } from "../../components/form-engine"
import {
  fetchEngineConfig,
  submitServiceDeskDraft,
  type AgenticUISurface,
  type ITSMDraftFormSurface,
  type ITSMDraftFormSurfacePayload,
} from "../../api"

const SUGGESTED_PROMPTS = [
  "我想申请 VPN，线上支持用",
  "电脑无法连接公司 Wi-Fi",
  "需要临时访问生产服务器",
  "帮我查一下我的工单进度",
]

function groupUIMessagesIntoPairs(messages: UIMessage[]): Array<{ userMessage: UIMessage; aiMessages: UIMessage[] }> {
  const pairs: Array<{ userMessage: UIMessage; aiMessages: UIMessage[] }> = []
  for (const msg of messages) {
    if (msg.role === "user") {
      pairs.push({ userMessage: msg, aiMessages: [] })
      continue
    }
    if (pairs.length > 0) {
      pairs[pairs.length - 1].aiMessages.push(msg)
    }
  }
  return pairs
}

function formatSessionTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ""
  return date.toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })
}

function sessionTitle(session: AgentSession) {
  return session.title || `会话 #${session.id}`
}

function StatusDot({ className }: { className?: string }) {
  return (
    <span className={cn("relative flex size-2.5", className)}>
      <span className="absolute inline-flex size-full animate-ping rounded-full bg-emerald-400 opacity-45" />
      <span className="relative inline-flex size-2.5 rounded-full bg-emerald-500" />
    </span>
  )
}

function ServiceDeskSidebar({
  sessions,
  activeSessionId,
  loading,
  onSelect,
  onNew,
}: {
  sessions: AgentSession[]
  activeSessionId: number | null
  loading: boolean
  onSelect: (sessionId: number) => void
  onNew: () => void
}) {
  return (
    <aside className="hidden min-h-0 w-60 shrink-0 border-r border-border/70 bg-muted/12 md:flex md:flex-col">
      <div className="flex h-14 items-center justify-between px-4">
        <div className="flex items-center gap-2 text-sm font-medium">
          <History className="size-4 text-muted-foreground" />
          会话
        </div>
        <Button type="button" size="icon" variant="ghost" className="size-8" onClick={onNew}>
          <Plus className="size-4" />
        </Button>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
        {loading ? (
          <div className="flex items-center gap-2 px-3 py-3 text-xs text-muted-foreground">
            <Loader2 className="size-3.5 animate-spin" />
            载入会话
          </div>
        ) : sessions.length === 0 ? (
          <div className="px-3 py-3 text-xs leading-5 text-muted-foreground">暂无历史会话</div>
        ) : (
          <div className="space-y-1">
            {sessions.map((session) => {
              const active = session.id === activeSessionId
              return (
                <button
                  key={session.id}
                  type="button"
                  onClick={() => onSelect(session.id)}
                  className={cn(
                    "group flex w-full flex-col rounded-md border border-transparent px-3 py-2 text-left transition-colors",
                    active ? "border-primary/18 bg-primary/8 text-foreground" : "text-muted-foreground hover:bg-muted/45 hover:text-foreground",
                  )}
                >
                  <span className="line-clamp-2 text-sm leading-5">{sessionTitle(session)}</span>
                  <span className="mt-1 text-[11px] text-muted-foreground/75">{formatSessionTime(session.updatedAt)}</span>
                </button>
              )
            })}
          </div>
        )}
      </div>
    </aside>
  )
}

function ServiceDeskComposer({
  value,
  disabled,
  pending,
  placeholder,
  onChange,
  onSend,
  compact,
}: {
  value: string
  disabled?: boolean
  pending?: boolean
  placeholder: string
  onChange: (value: string) => void
  onSend: () => void
  compact?: boolean
}) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    const textarea = textareaRef.current
    if (!textarea) return
    textarea.style.height = "auto"
    const maxHeight = compact ? 160 : 200
    textarea.style.height = `${Math.min(textarea.scrollHeight, maxHeight)}px`
  }, [value, compact])

  return (
    <div className={cn("w-full", compact ? "max-w-3xl" : "max-w-[720px]")}>
      <div className="rounded-2xl border bg-background shadow-lg transition-colors focus-within:border-primary/30">
        <textarea
          ref={textareaRef}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault()
              onSend()
            }
          }}
          placeholder={placeholder}
          rows={1}
          className={cn(
            "w-full max-h-[200px] resize-none bg-transparent px-4 pt-3 pb-1 text-base leading-relaxed placeholder:text-muted-foreground focus:outline-none read-only:cursor-text",
            compact ? "min-h-11" : "min-h-24",
          )}
          readOnly={disabled}
        />
        <div className="flex items-center justify-between px-3 pb-2">
          <div />
          <Button
            type="button"
            size="icon"
            className="size-8 rounded-full"
            onClick={onSend}
            disabled={!value.trim() || disabled || pending}
          >
            {pending ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
          </Button>
        </div>
      </div>
    </div>
  )
}

function WelcomeStage({
  agentName,
  value,
  disabled,
  pending,
  onChange,
  onSend,
}: {
  agentName: string
  value: string
  disabled?: boolean
  pending?: boolean
  onChange: (value: string) => void
  onSend: () => void
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col items-center justify-center px-5 py-8">
      <div className="flex flex-col items-center text-center">
        <div className="flex size-16 items-center justify-center rounded-full border border-primary/20 bg-primary/8 text-primary shadow-[0_18px_44px_-34px_hsl(var(--primary))]">
          <Bot className="size-8" />
        </div>
        <div className="mt-4 flex items-center gap-2 text-sm text-muted-foreground">
          <StatusDot />
          <span>{agentName}</span>
        </div>
        <h1 className="mt-3 text-3xl font-semibold tracking-normal text-foreground">IT 服务台</h1>
        <p className="mt-3 max-w-xl text-sm leading-6 text-muted-foreground">
          直接描述 IT 诉求，服务台会澄清信息、生成草稿，并在你确认后沉淀为工单。
        </p>
      </div>
      <div className="mt-9 flex w-full flex-col items-center">
        <ServiceDeskComposer
          value={value}
          onChange={onChange}
          onSend={onSend}
          disabled={disabled}
          pending={pending}
          placeholder="描述你的 IT 诉求..."
        />
        <div className="mt-4 flex max-w-3xl flex-wrap justify-center gap-2">
          {SUGGESTED_PROMPTS.map((prompt) => (
            <button
              key={prompt}
              type="button"
              onClick={() => onChange(prompt)}
              className="rounded-full border border-border/80 bg-background/76 px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent/45 hover:text-foreground"
              disabled={disabled}
            >
              {prompt}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

function NotOnDutyState({ loading }: { loading: boolean }) {
  const navigate = useNavigate()
  return (
    <div className="flex flex-1 items-center justify-center p-8">
      <div className="w-full max-w-xl rounded-lg border border-dashed border-border bg-background p-8 text-center">
        {loading ? (
          <Loader2 className="mx-auto size-7 animate-spin text-muted-foreground" />
        ) : (
          <AlertTriangle className="mx-auto size-7 text-amber-600" />
        )}
        <h2 className="mt-4 text-lg font-semibold">服务台智能体未配置</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          需要在引擎配置中绑定 itsm.servicedesk 默认智能体。
        </p>
        <Button className="mt-5" onClick={() => navigate("/itsm/engine-config")}>
          前往引擎配置
        </Button>
      </div>
    </div>
  )
}

function readDraftSurface(part: UIMessage["parts"][number]): ITSMDraftFormSurface | null {
  if (part.type !== "data-ui-surface") return null
  const data = (part as { data?: unknown }).data
  if (!data || typeof data !== "object") return null
  const surface = data as AgenticUISurface<ITSMDraftFormSurfacePayload>
  if (surface.surfaceType !== "itsm.draft_form") return null
  if (!surface.payload || typeof surface.payload !== "object") return null
  return surface
}

function isFormSchema(schema: unknown): schema is FormSchema {
  return Boolean(schema && typeof schema === "object" && Array.isArray((schema as FormSchema).fields))
}

function ServiceDeskDataPart({
  part,
  sessionId,
  onSubmitted,
}: {
  part: UIMessage["parts"][number]
  sessionId: number
  onSubmitted: () => void
}) {
  const surface = readDraftSurface(part)
  if (!surface) return null
  return (
    <ITSMDraftFormSurfaceCard
      key={`${surface.surfaceId}:${surface.payload.status}:${surface.payload.draftVersion ?? ""}`}
      surface={surface}
      sessionId={sessionId}
      onSubmitted={onSubmitted}
    />
  )
}

function ITSMDraftFormSurfaceCard({
  surface,
  sessionId,
  onSubmitted,
}: {
  surface: ITSMDraftFormSurface
  sessionId: number
  onSubmitted: () => void
}) {
  const payload = surface.payload
  const initialFormData = useMemo(() => payload.values ?? {}, [payload.values])
  const [formData, setFormData] = useState<Record<string, unknown>>(payload.values ?? {})
  const [submittedSurface, setSubmittedSurface] = useState<ITSMDraftFormSurface | null>(null)
  const [inlineError, setInlineError] = useState<string | null>(null)

  const submitMutation = useMutation({
    mutationFn: () => {
      if (!payload.draftVersion) {
        throw new Error("草稿版本缺失，请重新整理草稿")
      }
      return submitServiceDeskDraft(sessionId, {
        draftVersion: payload.draftVersion,
        summary: payload.summary ?? payload.title ?? "",
        formData,
      })
    },
    onSuccess: (result) => {
      if (!result.ok) {
        setInlineError(result.guidance || result.failureReason || result.message || "提交未通过")
        return
      }
      if (result.surface?.surfaceType === "itsm.draft_form") {
        setSubmittedSurface(result.surface as ITSMDraftFormSurface)
      } else {
        setSubmittedSurface({
          surfaceId: `${surface.surfaceId}:submitted`,
          surfaceType: "itsm.draft_form",
          payload: {
            status: "submitted",
            title: payload.title,
            summary: payload.summary,
            values: formData,
            ticketId: result.ticketId,
            ticketCode: result.ticketCode,
            message: result.message,
          },
        })
      }
      onSubmitted()
    },
    onError: (err) => setInlineError(err.message),
  })

  const currentSurface = submittedSurface ?? surface
  const currentPayload = currentSurface.payload

  if (currentPayload.status === "loading") {
    return (
      <div className="mb-6 max-w-[720px] rounded-md border border-border/60 bg-muted/18 px-4 py-3">
        <div className="flex items-center gap-2 text-sm font-medium text-foreground">
          <Loader2 className="size-4 animate-spin text-primary" />
          {currentPayload.title || "正在整理草稿"}
        </div>
        <div className="mt-3 space-y-2">
          <div className="h-2.5 w-2/3 animate-pulse rounded bg-muted" />
          <div className="h-2.5 w-5/6 animate-pulse rounded bg-muted" />
        </div>
      </div>
    )
  }

  if (currentPayload.status === "submitted") {
    return (
      <div className="mb-6 max-w-[720px] rounded-md border border-emerald-500/25 bg-emerald-500/6 px-4 py-3">
        <div className="flex items-center gap-2 text-sm font-semibold text-emerald-700 dark:text-emerald-300">
          <CheckCircle2 className="size-4" />
          {currentPayload.message || "工单已提交"}
        </div>
        {currentPayload.ticketCode && (
          <div className="mt-2 text-sm text-foreground">
            工单编号：<span className="font-medium">{currentPayload.ticketCode}</span>
          </div>
        )}
      </div>
    )
  }

  if (!isFormSchema(currentPayload.schema)) {
    return (
      <div className="mb-6 max-w-[720px] rounded-md border border-destructive/25 bg-destructive/5 px-4 py-3 text-sm text-destructive">
        表单定义不可用，请重新整理草稿。
      </div>
    )
  }

  return (
    <div className="mb-6 max-w-[720px] rounded-md border border-border/65 bg-background px-4 py-4 shadow-sm">
      <div className="mb-4">
        <div className="text-xs font-medium text-muted-foreground">草稿确认</div>
        <div className="mt-1 text-base font-semibold text-foreground">{currentPayload.title || "服务申请草稿"}</div>
        {currentPayload.summary && (
          <div className="mt-1 text-sm leading-6 text-muted-foreground">{currentPayload.summary}</div>
        )}
      </div>

      <FormRenderer
        schema={currentPayload.schema}
        data={initialFormData}
        mode="edit"
        onChange={setFormData}
        disabled={submitMutation.isPending}
      />

      {inlineError && (
        <div className="mt-4 rounded-md border border-destructive/25 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {inlineError}
        </div>
      )}

      <div className="mt-4 flex justify-end">
        <Button
          type="button"
          onClick={() => submitMutation.mutate()}
          disabled={submitMutation.isPending}
        >
          {submitMutation.isPending ? <Loader2 className="mr-1.5 size-4 animate-spin" /> : <CheckCircle2 className="mr-1.5 size-4" />}
          提交工单
        </Button>
      </div>
    </div>
  )
}

function ServiceDeskConversation({
  session,
  agentName,
  initialPrompt,
  onInitialPromptSent,
}: {
  session: AgentSession
  agentName: string
  initialPrompt?: string
  onInitialPromptSent: () => void
}) {
  const queryClient = useQueryClient()
  const [input, setInput] = useState("")
  const scrollRef = useRef<HTMLDivElement>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const initialPromptSentRef = useRef(false)

  const { data: sessionData, isLoading } = useQuery({
    queryKey: ["ai-session", session.id],
    queryFn: () => sessionApi.get(session.id),
  })

  const invalidateWorkspace = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ["ai-session", session.id] })
    queryClient.invalidateQueries({ queryKey: ["itsm-service-desk-state", session.id] })
    queryClient.invalidateQueries({ queryKey: ["itsm-service-desk-tickets", session.id] })
    queryClient.invalidateQueries({ queryKey: ["ai-sessions"] })
  }, [queryClient, session.id])

  const chat = useAiChat(session.id, sessionData?.messages, {
    onFinish: invalidateWorkspace,
    onError: (err) => {
      toast.error(err.message)
      invalidateWorkspace()
    },
  })

  const isBusy = chat.status === "streaming" || chat.status === "submitted"
  const qaPairs = useMemo(() => groupUIMessagesIntoPairs(chat.messages), [chat.messages])

  useEffect(() => {
    if (!initialPrompt || initialPromptSentRef.current || isLoading) return
    initialPromptSentRef.current = true
    chat.sendMessage({ text: initialPrompt })
    onInitialPromptSent()
  }, [chat, initialPrompt, isLoading, onInitialPromptSent])

  useEffect(() => {
    const container = scrollRef.current
    if (!container) return
    container.scrollTo({ top: container.scrollHeight, behavior: isBusy ? "instant" : "smooth" })
  }, [chat.messages.length, isBusy])

  const sendMutation = useMutation({
    mutationFn: async (text: string) => text,
    onSuccess: (text) => {
      chat.sendMessage({ text })
      setInput("")
    },
    onError: (err) => toast.error(err.message),
  })

  const cancelMutation = useMutation({
    mutationFn: async () => {
      await chat.stop()
      return sessionApi.cancel(session.id)
    },
    onSuccess: invalidateWorkspace,
    onError: (err) => toast.error(err.message),
  })

  const handleSend = useCallback(() => {
    const text = input.trim()
    if (!text || isBusy || sendMutation.isPending) return
    sendMutation.mutate(text)
  }, [input, isBusy, sendMutation])

  const showEmpty = !isLoading && qaPairs.length === 0 && !initialPrompt

  return (
    <main className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background">
        <div className="flex h-14 shrink-0 items-center justify-between border-b border-border/70 px-5">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex size-8 shrink-0 items-center justify-center rounded-full border border-primary/20 bg-primary/8 text-primary">
              <Bot className="size-4" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <h1 className="truncate text-sm font-semibold">IT 服务台</h1>
                <StatusDot />
              </div>
              <div className="mt-0.5 truncate text-xs font-medium text-foreground/70">
                当前智能体：{agentName} · {formatSessionTime(session.updatedAt)}
              </div>
            </div>
          </div>
          <div />
        </div>

        <div ref={scrollRef} className="relative min-h-0 flex-1 overflow-y-auto overflow-x-hidden">
          {isLoading ? (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              <Loader2 className="mr-2 size-4 animate-spin" />
              载入会话
            </div>
          ) : showEmpty ? (
            <div className="mx-auto flex h-full max-w-3xl flex-col justify-center px-6 py-12">
              <div className="flex size-14 items-center justify-center rounded-full border border-primary/25 bg-primary/10 text-primary">
                <Bot className="size-7" />
              </div>
              <h2 className="mt-5 text-2xl font-semibold">继续描述 IT 诉求</h2>
              <p className="mt-2 max-w-xl text-sm leading-6 text-muted-foreground">
                服务台会沿用当前会话上下文继续澄清、填槽和创建工单。
              </p>
            </div>
          ) : (
            <div className="mx-auto max-w-3xl px-4 pb-4">
              {qaPairs.map((pair, index) => {
                const isLastPair = index === qaPairs.length - 1
                return (
                  <QAPair
                    key={pair.userMessage.id}
                    userMessage={pair.userMessage}
                    aiMessages={pair.aiMessages}
                    agentName={agentName}
                    isStreaming={isLastPair && isBusy}
                    onRegenerate={isLastPair ? () => chat.regenerate() : undefined}
                    renderDataPart={(part) => (
                      <ServiceDeskDataPart
                        part={part}
                        sessionId={session.id}
                        onSubmitted={invalidateWorkspace}
                      />
                    )}
                    suppressTextWhenDataPart
                    doneMetrics={
                      isLastPair && chat.status === "ready"
                        ? {
                            inputTokens: chat.lastUsage.promptTokens,
                            outputTokens: chat.lastUsage.completionTokens,
                          }
                        : undefined
                    }
                  />
                )
              })}
              {chat.error && !isBusy && (
                <div className="mb-6 rounded-lg border border-destructive/25 bg-destructive/5 p-3 text-sm text-destructive">
                  {chat.error.message}
                </div>
              )}
              <div ref={messagesEndRef} />
            </div>
          )}
        </div>

        {isBusy && (
          <div className="flex shrink-0 justify-center pb-2">
            <Button
              variant="outline"
              size="sm"
              className="rounded-full px-4"
              onClick={() => cancelMutation.mutate()}
              disabled={cancelMutation.isPending}
            >
              {cancelMutation.isPending ? <Loader2 className="mr-1.5 size-3.5 animate-spin" /> : <Square className="mr-1.5 size-3.5" />}
              停止
            </Button>
          </div>
        )}

        <div className="shrink-0 bg-background px-4 pb-3 pt-1">
          <div className="mx-auto max-w-3xl">
            <ServiceDeskComposer
              value={input}
              onChange={setInput}
              onSend={handleSend}
              disabled={isBusy}
              pending={sendMutation.isPending}
              placeholder="描述你的 IT 诉求..."
              compact
            />
            <p className="mt-1 text-center text-[10px] text-muted-foreground/50">Enter 发送，Shift + Enter 换行</p>
          </div>
        </div>
    </main>
  )
}

export function Component() {
  const queryClient = useQueryClient()
  const [selectedSessionId, setSelectedSessionId] = useState<number | null>(null)
  const [createdSession, setCreatedSession] = useState<AgentSession | null>(null)
  const [landingInput, setLandingInput] = useState("")
  const [pendingInitialPrompt, setPendingInitialPrompt] = useState<{ sessionId: number; text: string } | null>(null)

  const { data: config, isLoading: configLoading } = useQuery({
    queryKey: ["itsm-engine-config"],
    queryFn: fetchEngineConfig,
  })

  const serviceDeskAgentId = config?.servicedesk?.agentId ?? 0
  const serviceDeskAgentName = config?.servicedesk?.agentName || "IT 服务台"

  const sessionsQuery = useQuery({
    queryKey: ["ai-sessions", serviceDeskAgentId],
    queryFn: () => sessionApi.list({ agentId: serviceDeskAgentId, page: 1, pageSize: 30 }),
    enabled: serviceDeskAgentId > 0,
  })

  const sessions = sessionsQuery.data?.items ?? []
  const activeSession = selectedSessionId == null
    ? null
    : sessions.find((item) => item.id === selectedSessionId) ?? (createdSession?.id === selectedSessionId ? createdSession : null)

  const createSessionMutation = useMutation({
    mutationFn: async (text: string) => {
      const session = await sessionApi.create(serviceDeskAgentId)
      return { session, text }
    },
    onSuccess: ({ session, text }) => {
      setCreatedSession(session)
      setSelectedSessionId(session.id)
      setPendingInitialPrompt({ sessionId: session.id, text })
      setLandingInput("")
      queryClient.invalidateQueries({ queryKey: ["ai-sessions", serviceDeskAgentId] })
    },
    onError: (err) => toast.error(err.message),
  })

  const handleLandingSend = useCallback(() => {
    const text = landingInput.trim()
    if (!text || serviceDeskAgentId <= 0 || createSessionMutation.isPending) return
    createSessionMutation.mutate(text)
  }, [createSessionMutation, landingInput, serviceDeskAgentId])

  const handleSelectSession = useCallback((sessionId: number) => {
    setSelectedSessionId(sessionId)
    setCreatedSession(null)
    setPendingInitialPrompt(null)
  }, [])

  const handleNewSession = useCallback(() => {
    setSelectedSessionId(null)
    setCreatedSession(null)
    setPendingInitialPrompt(null)
    setLandingInput("")
  }, [])

  const clearPendingInitialPrompt = useCallback(() => {
    setPendingInitialPrompt(null)
  }, [])

  return (
    <div className="grid h-full min-h-0 grid-cols-1 overflow-hidden bg-[linear-gradient(180deg,hsl(var(--background)),hsl(var(--muted)/0.18))] md:grid-cols-[240px_minmax(0,1fr)]">
      <ServiceDeskSidebar
        sessions={sessions}
        activeSessionId={activeSession?.id ?? null}
        loading={sessionsQuery.isLoading}
        onSelect={handleSelectSession}
        onNew={handleNewSession}
      />

      {serviceDeskAgentId <= 0 || configLoading ? (
        <main className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
          <NotOnDutyState loading={configLoading} />
        </main>
      ) : activeSession ? (
        <ServiceDeskConversation
          key={activeSession.id}
          session={activeSession}
          agentName={serviceDeskAgentName}
          initialPrompt={pendingInitialPrompt?.sessionId === activeSession.id ? pendingInitialPrompt.text : undefined}
          onInitialPromptSent={clearPendingInitialPrompt}
        />
      ) : (
        <main className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
            <div className="flex h-14 shrink-0 items-center justify-between border-b border-border/70 px-5">
              <div className="flex min-w-0 items-center gap-3">
                <div className="flex size-8 shrink-0 items-center justify-center rounded-full border border-primary/20 bg-primary/8 text-primary">
                  <Sparkles className="size-4" />
                </div>
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <h1 className="truncate text-sm font-semibold">IT 服务台</h1>
                    <StatusDot />
                  </div>
                  <div className="mt-0.5 truncate text-xs font-medium text-foreground/70">
                    当前智能体：{serviceDeskAgentName}
                  </div>
                </div>
              </div>
              <Button type="button" size="sm" variant="outline" className="md:hidden" onClick={handleNewSession}>
                <Plus className="mr-1.5 size-3.5" />
                新会话
              </Button>
            </div>
            <WelcomeStage
              agentName={serviceDeskAgentName}
              value={landingInput}
              onChange={setLandingInput}
              onSend={handleLandingSend}
              disabled={createSessionMutation.isPending}
              pending={createSessionMutation.isPending}
            />
          </main>
      )}
    </div>
  )
}
