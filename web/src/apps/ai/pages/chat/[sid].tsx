"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useParams, useNavigate } from "react-router"
import { useTranslation } from "react-i18next"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Square, Trash2, Brain, PanelLeft, PanelLeftClose, Paperclip, AlertTriangle, RotateCcw, X } from "lucide-react"
import { isDataUIPart, isReasoningUIPart, type UIMessage } from "ai"
import { sessionApi } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { useAiChat } from "./hooks/use-ai-chat"
import { QAPair } from "./components/message-item"
import { ThinkingBlock } from "./components/thinking-block"
import { PlanProgress } from "./components/plan-progress"
import { WelcomeScreen } from "./components/welcome-screen"
import { SessionSidebar } from "./components/session-sidebar"
import { MemoryPanel } from "./components/memory-panel"

const SIDEBAR_COLLAPSED_KEY = "ai-chat-sidebar-collapsed"

interface PendingImage {
  file: File
  preview: string
  uploading?: boolean
}

function groupUIMessagesIntoPairs(messages: UIMessage[]): Array<{ userMessage: UIMessage; aiMessages: UIMessage[] }> {
  const pairs: Array<{ userMessage: UIMessage; aiMessages: UIMessage[] }> = []
  for (const msg of messages) {
    if (msg.role === "user") {
      pairs.push({ userMessage: msg, aiMessages: [] })
    } else if (pairs.length > 0) {
      pairs[pairs.length - 1].aiMessages.push(msg)
    }
  }
  return pairs
}

function getStreamingExtras(
  pair: { aiMessages: UIMessage[] },
  isStreaming: boolean,
  agentName?: string,
): React.ReactNode {
  const mainAiMessage = pair.aiMessages
    .filter((m) => {
      const meta = m.metadata as { originalRole?: string } | undefined
      return !["tool_call", "tool_result"].includes(meta?.originalRole || "")
    })
    .pop()

  const reasoningParts = mainAiMessage?.parts?.filter(isReasoningUIPart) || []
  const thinkingText = reasoningParts.map((p) => p.text).join("")

  const dataParts = mainAiMessage?.parts?.filter(isDataUIPart) || []

  let planSteps: { description: string; durationMs?: number }[] = []
  let planStepIndex = -1
  for (const part of dataParts) {
    const d = part.data as Record<string, unknown> | undefined
    if (!d) continue
    if (part.type === "data-plan" && Array.isArray(d.steps)) {
      planSteps = (d.steps as Array<{ description?: string }>).map((s) => ({
        description: s.description || "",
        durationMs: undefined,
      }))
      planStepIndex = 0
    } else if (part.type === "data-step" && typeof d.index === "number") {
      if (d.state === "start") {
        planStepIndex = d.index as number
      } else if (d.state === "done") {
        const idx = d.index as number
        if (planSteps[idx]) {
          planSteps[idx].durationMs = typeof d.durationMs === "number" ? d.durationMs : undefined
        }
        planStepIndex = idx + 1
      }
    }
  }

  const textParts =
    mainAiMessage?.parts?.filter(
      (p): p is { type: "text"; text: string } => p.type === "text",
    ) || []
  const hasText = textParts.some((p) => p.text)
  const hasTools = mainAiMessage?.parts?.some((p) => p.type === "dynamic-tool")
  const hasContent = hasText || thinkingText || planSteps.length > 0 || hasTools

  const extras: React.ReactNode[] = []
  if (thinkingText) {
    extras.push(<ThinkingBlock key="thinking" content={thinkingText} isStreaming={isStreaming} />)
  }
  if (planSteps.length > 0) {
    extras.push(
      <PlanProgress
        key="plan"
        steps={planSteps}
        currentStepIndex={planStepIndex}
        isStreaming={isStreaming}
      />,
    )
  }
  if (isStreaming && !hasContent) {
    extras.push(
      <div key="loading" className="flex items-center gap-2 text-sm text-muted-foreground mb-4">
        {agentName && <span className="text-xs font-medium">{agentName}</span>}
        <span className="flex gap-1">
          <span className="h-1.5 w-1.5 rounded-full bg-foreground/40 animate-bounce [animation-delay:0ms]" />
          <span className="h-1.5 w-1.5 rounded-full bg-foreground/40 animate-bounce [animation-delay:150ms]" />
          <span className="h-1.5 w-1.5 rounded-full bg-foreground/40 animate-bounce [animation-delay:300ms]" />
        </span>
      </div>,
    )
  }
  return extras.length > 0 ? <>{extras}</> : null
}

export function Component() {
  const { sid } = useParams<{ sid: string }>()
  const sessionId = Number(sid)
  const { t } = useTranslation(["ai", "common"])
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [input, setInput] = useState("")
  const [memoryOpen, setMemoryOpen] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    const saved = localStorage.getItem(SIDEBAR_COLLAPSED_KEY)
    return saved ? saved === "true" : false
  })
  const [pendingImages, setPendingImages] = useState<PendingImage[]>([])
  const scrollRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const { data: sessionData, isLoading } = useQuery({
    queryKey: ["ai-session", sessionId],
    queryFn: () => sessionApi.get(sessionId),
    enabled: !!sessionId,
  })

  const chat = useAiChat(sessionId, sessionData?.messages, {
    onFinish: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-session", sessionId] })
    },
    onError: (err) => {
      toast.error(err.message)
      queryClient.invalidateQueries({ queryKey: ["ai-session", sessionId] })
    },
  })

  const qaPairs = useMemo(() => {
    return groupUIMessagesIntoPairs(chat.messages)
  }, [chat.messages])

  const scrollToBottom = useCallback((instant?: boolean) => {
    messagesEndRef.current?.scrollIntoView({ behavior: instant ? "instant" : "smooth" })
  }, [])

  const uploadImageMutation = useMutation({
    mutationFn: (file: File) => sessionApi.uploadMessageImage(sessionId, file),
    onError: (err) => toast.error(err.message),
  })

  const sendMutation = useMutation({
    mutationFn: async (text: string) => {
      const imageUrls: string[] = []
      for (const img of pendingImages) {
        if (!img.uploading) {
          const res = await uploadImageMutation.mutateAsync(img.file)
          imageUrls.push(res.url)
        }
      }
      chat.setPendingImageUrls(imageUrls)
      return text
    },
    onSuccess: (text) => {
      chat.sendMessage({ text })
      setInput("")
      setPendingImages([])
      scrollToBottom()
    },
    onError: (err) => toast.error(err.message),
  })

  const cancelMutation = useMutation({
    mutationFn: async () => {
      chat.stop()
      return sessionApi.cancel(sessionId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-session", sessionId] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => sessionApi.delete(sessionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-sessions"] })
      toast.success(t("ai:chat.sessionDeleted"))

      const currentData = queryClient.getQueryData<{ items: Array<{ id: number }> }>(["ai-sessions", agentId])
      const otherSession = currentData?.items.find((s) => s.id !== sessionId)

      if (otherSession) {
        navigate(`/ai/chat/${otherSession.id}`)
      } else {
        navigate("/ai/chat")
      }
    },
    onError: (err) => toast.error(err.message),
  })

  const editMessageMutation = useMutation({
    mutationFn: ({ mid, content }: { mid: number; content: string }) =>
      sessionApi.editMessage(sessionId, mid, content),
    onSuccess: (_, { mid, content }) => {
      const editedIndex = chat.messages.findIndex((m) => Number(m.id) === mid)
      if (editedIndex !== -1) {
        const newMessages = chat.messages.slice(0, editedIndex + 1)
        newMessages[editedIndex] = {
          ...newMessages[editedIndex],
          parts:
            newMessages[editedIndex].parts?.map((p) =>
              p.type === "text" ? { ...p, text: content } : p,
            ) || [{ type: "text", text: content }],
        }
        chat.setMessages(newMessages)
        chat.regenerate()
      } else {
        queryClient.invalidateQueries({ queryKey: ["ai-session", sessionId] })
      }
    },
    onError: (err) => toast.error(err.message),
  })

  const handleEditMessage = useCallback(
    (messageId: number, content: string) => {
      editMessageMutation.mutate({ mid: messageId, content })
    },
    [editMessageMutation],
  )

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current
    if (!textarea) return
    textarea.style.height = "auto"
    const newHeight = Math.min(textarea.scrollHeight, 200)
    textarea.style.height = `${newHeight}px`
  }, [input])

  // Scroll to bottom on messages change (smooth for new messages, instant while streaming)
  useEffect(() => {
    const instant = chat.status === "streaming" || chat.status === "submitted"
    scrollToBottom(instant)
  }, [chat.messages, chat.status, scrollToBottom])

  const isBusy = chat.status === "streaming" || chat.status === "submitted"

  function handleSend(content?: string) {
    const text = (content ?? input).trim()
    if ((!text && pendingImages.length === 0) || isBusy || sendMutation.isPending) return
    sendMutation.mutate(text)
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  function handlePaste(e: React.ClipboardEvent) {
    const items = e.clipboardData.items
    const imageFiles: File[] = []

    for (let i = 0; i < items.length; i++) {
      const item = items[i]
      if (item.type.startsWith("image/")) {
        const file = item.getAsFile()
        if (file) {
          imageFiles.push(file)
        }
      }
    }

    if (imageFiles.length > 0) {
      e.preventDefault()
      for (const file of imageFiles) {
        const reader = new FileReader()
        reader.onload = (event) => {
          const preview = event.target?.result as string
          setPendingImages((prev) => [...prev, { file, preview }])
        }
        reader.readAsDataURL(file)
      }
    }
  }

  function removePendingImage(index: number) {
    setPendingImages((prev) => {
      const newImages = [...prev]
      newImages.splice(index, 1)
      return newImages
    })
  }

  function handleRetry() {
    chat.regenerate()
  }

  const toggleSidebar = useCallback(() => {
    setSidebarCollapsed((prev) => {
      const newValue = !prev
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(newValue))
      return newValue
    })
  }, [])

  if (isLoading) {
    return <div className="flex items-center justify-center h-full text-muted-foreground">{t("common:loading")}</div>
  }

  const session = sessionData?.session
  const agentId = session?.agentId
  const agentName = (session as unknown as Record<string, unknown>)?.agentName as string | undefined
  const hasMessages = chat.messages.length > 0
  const showWelcome = !hasMessages && !isBusy
  const lastPairIndex = qaPairs.length - 1

  return (
    <div className="flex h-full overflow-hidden">
      {/* Sidebar */}
      <SessionSidebar agentId={agentId} currentSessionId={sessionId} collapsed={sidebarCollapsed} />

      {/* Main chat area */}
      <div className="flex-1 flex flex-col min-w-0 bg-background h-full">
        {/* Header */}
        <div className="flex items-center justify-between border-b px-4 py-2 shrink-0 h-12">
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" className="h-8 w-8 p-0" onClick={toggleSidebar}>
              {sidebarCollapsed ? <PanelLeft className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
            </Button>
            <h3 className="font-medium truncate">{session?.title || t("ai:chat.newChat")}</h3>
            {session?.status && (
              <Badge variant="outline" className="text-xs">
                {t(`ai:chat.sessionStatus.${session.status}`)}
              </Badge>
            )}
          </div>
          <div className="flex items-center gap-1">
            {agentId && (
              <Button variant="ghost" size="sm" onClick={() => setMemoryOpen(!memoryOpen)}>
                <Brain className="h-4 w-4" />
              </Button>
            )}
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive">
                  <Trash2 className="h-4 w-4" />
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>{t("ai:chat.deleteSession")}</AlertDialogTitle>
                  <AlertDialogDescription>{t("ai:chat.deleteSessionDesc")}</AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
                  <AlertDialogAction onClick={() => deleteMutation.mutate()} disabled={deleteMutation.isPending}>
                    {t("common:delete")}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>

        {/* Messages area */}
        <div ref={scrollRef} className="flex-1 overflow-y-auto overflow-x-hidden min-h-0">
          {showWelcome ? (
            <WelcomeScreen
              agentName={agentName ?? session?.title}
              agentType={(session as unknown as Record<string, unknown>)?.agentType as string | undefined}
              onPromptClick={(prompt) => handleSend(prompt)}
            />
          ) : (
            <div className="max-w-3xl mx-auto px-4 pb-4">
              {qaPairs.map((pair, index) => {
                const isLastPair = index === lastPairIndex
                const isStreamingThisPair = isLastPair && isBusy

                return (
                  <QAPair
                    key={pair.userMessage.id}
                    userMessage={pair.userMessage}
                    aiMessages={pair.aiMessages}
                    agentName={agentName}
                    isStreaming={isStreamingThisPair}
                    onRegenerate={isLastPair ? () => chat.regenerate() : undefined}
                    onEditMessage={handleEditMessage}
                    doneMetrics={
                      isLastPair && chat.status === "ready"
                        ? {
                            inputTokens: chat.lastUsage.promptTokens,
                            outputTokens: chat.lastUsage.completionTokens,
                          }
                        : undefined
                    }
                    streamingExtras={
                      isStreamingThisPair ? getStreamingExtras(pair, true, agentName) : undefined
                    }
                  />
                )
              })}

              {/* Inline error */}
              {chat.error && chat.status !== "streaming" && chat.status !== "submitted" && (
                <div className="py-6">
                  <div className="flex items-center gap-3 p-3 rounded-lg border-l-4 border-destructive bg-destructive/5">
                    <AlertTriangle className="h-4 w-4 text-destructive shrink-0" />
                    <div className="flex-1">
                      <div className="text-sm font-medium text-destructive">{t("ai:chat.generationError")}</div>
                      <div className="text-xs text-muted-foreground mt-0.5">{chat.error.message}</div>
                    </div>
                    <Button variant="outline" size="sm" onClick={handleRetry}>
                      <RotateCcw className="h-3.5 w-3.5 mr-1" />
                      {t("ai:chat.retry")}
                    </Button>
                  </div>
                </div>
              )}

              <div ref={messagesEndRef} />
            </div>
          )}
        </div>

        {/* Centered stop button during streaming */}
        {isBusy && (
          <div className="flex justify-center pb-2 shrink-0">
            <Button
              variant="outline"
              size="sm"
              className="rounded-full px-4"
              onClick={() => cancelMutation.mutate()}
            >
              <Square className="h-3.5 w-3.5 mr-1.5" />
              {t("ai:chat.cancel")}
            </Button>
          </div>
        )}

        {/* Input area — floating card */}
        <div className="px-4 pb-3 pt-1 shrink-0">
          <div className="max-w-3xl mx-auto">
            <div className="rounded-2xl bg-background shadow-lg border transition-colors focus-within:border-primary/30">
              <textarea
                ref={textareaRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                onPaste={handlePaste}
                placeholder={t("ai:chat.inputPlaceholder")}
                rows={1}
                className="w-full min-h-[44px] max-h-[200px] resize-none bg-transparent px-4 pt-3 pb-1 text-base leading-relaxed placeholder:text-muted-foreground focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                disabled={isBusy}
              />
              {/* Pending images preview */}
              {pendingImages.length > 0 && (
                <div className="flex gap-2 px-4 pb-2 overflow-x-auto">
                  {pendingImages.map((img, idx) => (
                    <div key={idx} className="relative group shrink-0">
                      <img
                        src={img.preview}
                        alt={`Pending ${idx}`}
                        className="h-16 w-16 object-cover rounded-md border"
                      />
                      <button
                        type="button"
                        onClick={() => removePendingImage(idx)}
                        className="absolute -top-1.5 -right-1.5 h-5 w-5 rounded-full bg-destructive text-white flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </div>
                  ))}
                </div>
              )}
              {/* Toolbar */}
              <div className="flex items-center justify-between px-3 pb-2">
                <div className="flex items-center gap-1">
                  <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground/40" disabled>
                    <Paperclip className="h-4 w-4" />
                  </Button>
                </div>
                <div className="flex items-center gap-2">
                  {!isBusy && (
                    <Button
                      size="icon"
                      className="h-8 w-8 rounded-full"
                      onClick={() => handleSend()}
                      disabled={(!input.trim() && pendingImages.length === 0) || sendMutation.isPending}
                    >
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="h-4 w-4"
                      >
                        <path d="m22 2-7 20-4-9-9-4Z" />
                        <path d="M22 2 11 13" />
                      </svg>
                    </Button>
                  )}
                </div>
              </div>
            </div>
            <p className="text-[10px] text-muted-foreground/50 text-center mt-1">{t("ai:chat.inputHint")}</p>
          </div>
        </div>
      </div>

      {/* Memory panel */}
      {memoryOpen && agentId && <MemoryPanel agentId={agentId} onClose={() => setMemoryOpen(false)} />}
    </div>
  )
}
