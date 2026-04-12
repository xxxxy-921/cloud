import { useCallback, useMemo, useRef, useState, useEffect } from "react"
import { useParams, useNavigate } from "react-router"
import { useTranslation } from "react-i18next"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Square, Trash2, Brain, PanelLeft, PanelLeftClose } from "lucide-react"
import { sessionApi, type SessionMessage as SessionMsg } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { useChatStream, type ChatEvent } from "./hooks/use-chat-stream"
import { QAPair, AIResponse } from "./components/message-item"
import { SessionSidebar } from "./components/session-sidebar"
import { MemoryPanel } from "./components/memory-panel"

const SIDEBAR_COLLAPSED_KEY = "ai-chat-sidebar-collapsed"

export function Component() {
  // Group messages into QA pairs - defined inside component to avoid fast-refresh issues
  const groupMessagesIntoPairs = useCallback((messages: SessionMsg[]): Array<{
    userMessage: SessionMsg
    aiMessage?: SessionMsg
    tools: SessionMsg[]
  }> => {
    const pairs: Array<{
      userMessage: SessionMsg
      aiMessage?: SessionMsg
      tools: SessionMsg[]
    }> = []

    let currentPair: {
      userMessage?: SessionMsg
      aiMessage?: SessionMsg
      tools: SessionMsg[]
    } = { tools: [] }

    for (const msg of messages) {
      if (msg.role === "user") {
        // Save previous pair if exists
        if (currentPair.userMessage) {
          pairs.push({
            userMessage: currentPair.userMessage,
            aiMessage: currentPair.aiMessage,
            tools: currentPair.tools,
          })
        }
        // Start new pair
        currentPair = { userMessage: msg, tools: [] }
      } else if (msg.role === "assistant") {
        currentPair.aiMessage = msg
      } else if (msg.role === "tool_call" || msg.role === "tool_result") {
        currentPair.tools.push(msg)
      }
    }

    // Add last pair
    if (currentPair.userMessage) {
      pairs.push({
        userMessage: currentPair.userMessage,
        aiMessage: currentPair.aiMessage,
        tools: currentPair.tools,
      })
    }

    return pairs
  }, [])
  const { sid } = useParams<{ sid: string }>()
  const sessionId = Number(sid)
  const { t } = useTranslation(["ai", "common"])
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [input, setInput] = useState("")
  const [pendingMessages, setPendingMessages] = useState<SessionMsg[]>([])
  const [streamingText, setStreamingText] = useState("")
  const [memoryOpen, setMemoryOpen] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    const saved = localStorage.getItem(SIDEBAR_COLLAPSED_KEY)
    return saved ? saved === "true" : false
  })
  const scrollRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const { data: sessionData, isLoading } = useQuery({
    queryKey: ["ai-session", sessionId],
    queryFn: () => sessionApi.get(sessionId),
    enabled: !!sessionId,
  })

  const messages = useMemo(() => {
    const base = sessionData?.messages ?? []
    // Deduplicate: filter out pending messages that already exist in base (by ID)
    const baseIds = new Set(base.map(m => m.id))
    const uniquePending = pendingMessages.filter(m => !baseIds.has(m.id))
    return [...base, ...uniquePending]
  }, [sessionData, pendingMessages])

  const qaPairs = useMemo(() => {
    return groupMessagesIntoPairs(messages)
  }, [messages, groupMessagesIntoPairs])

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [])

  const handleEvent = useCallback((event: ChatEvent) => {
    if (event.type === "content_delta" && event.text) {
      setStreamingText((prev) => prev + event.text)
    }
  }, [])

  const handleDone = useCallback(() => {
    setStreamingText("")
    setPendingMessages([])
    queryClient.invalidateQueries({ queryKey: ["ai-session", sessionId] })
  }, [queryClient, sessionId])

  const handleStreamError = useCallback((msg: string) => {
    setStreamingText("")
    setPendingMessages([])
    toast.error(msg)
    queryClient.invalidateQueries({ queryKey: ["ai-session", sessionId] })
  }, [queryClient, sessionId])

  const { isStreaming, connect, disconnect } = useChatStream({
    onEvent: handleEvent,
    onDone: handleDone,
    onError: handleStreamError,
  })

  const sendMutation = useMutation({
    mutationFn: (content: string) => sessionApi.sendMessage(sessionId, content),
    onSuccess: (msg) => {
      setPendingMessages((prev) => [...prev, msg])
      setInput("")
      setStreamingText("")
      connect(sessionId)
      scrollToBottom()
    },
    onError: (err) => toast.error(err.message),
  })

  const cancelMutation = useMutation({
    mutationFn: () => sessionApi.cancel(sessionId),
    onSuccess: () => {
      disconnect()
      queryClient.invalidateQueries({ queryKey: ["ai-session", sessionId] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => sessionApi.delete(sessionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-sessions"] })
      toast.success(t("ai:chat.sessionDeleted"))
      navigate("/ai/chat")
    },
    onError: (err) => toast.error(err.message),
  })

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current
    if (!textarea) return

    textarea.style.height = "auto"
    const newHeight = Math.min(textarea.scrollHeight, 200)
    textarea.style.height = `${newHeight}px`
  }, [input])

  // Scroll to bottom when streaming
  useEffect(() => {
    if (isStreaming && streamingText) {
      scrollToBottom()
    }
  }, [isStreaming, streamingText, scrollToBottom])

  // Auto scroll to bottom when messages change (initial load, new messages)
  useEffect(() => {
    scrollToBottom()
  }, [qaPairs, scrollToBottom])

  function handleSend() {
    const content = input.trim()
    if (!content || isStreaming || sendMutation.isPending) return
    sendMutation.mutate(content)
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
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

  return (
    <div className="flex h-full overflow-hidden">
      {/* Sidebar */}
      <SessionSidebar
        agentId={agentId}
        currentSessionId={sessionId}
        collapsed={sidebarCollapsed}
      />

      {/* Main chat area */}
      <div className="flex-1 flex flex-col min-w-0 bg-background h-full">
        {/* Header */}
        <div className="flex items-center justify-between border-b px-4 py-2 shrink-0 h-12">
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              className="h-8 w-8 p-0"
              onClick={toggleSidebar}
            >
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

        {/* Messages - only this area scrolls */}
        <div ref={scrollRef} className="flex-1 overflow-y-auto overflow-x-hidden min-h-0">
          <div className="max-w-5xl mx-auto px-4 pb-4">
            {/* Render completed QA pairs */}
            {qaPairs.map((pair) => (
              <QAPair
                key={pair.userMessage.id}
                userMessage={pair.userMessage}
                aiMessage={pair.aiMessage}
              />
            ))}

            {/* Render streaming AI response only (user message is already in qaPairs) */}
            {isStreaming && streamingText && (
              <div className="py-6 border-b border-border/50">
                <AIResponse
                  content={streamingText}
                  isStreaming={isStreaming}
                />
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>
        </div>

        {/* Input - fixed at bottom */}
        <div className="px-4 pb-3 pt-1 shrink-0">
          <div className="max-w-5xl mx-auto">
            <div className="flex items-end gap-3 rounded-2xl bg-muted/60 px-5 py-3 transition-colors focus-within:bg-muted">
              <textarea
                ref={textareaRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder={t("ai:chat.inputPlaceholder")}
                rows={1}
                className="flex-1 min-h-[28px] max-h-[200px] resize-none bg-transparent py-0.5 text-base leading-relaxed placeholder:text-muted-foreground focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                disabled={isStreaming}
              />
              {isStreaming ? (
                <Button
                  variant="ghost"
                  size="icon"
                  className="shrink-0 h-8 w-8 rounded-full"
                  onClick={() => cancelMutation.mutate()}
                >
                  <Square className="h-4 w-4" />
                </Button>
              ) : (
                <Button
                  size="icon"
                  className="shrink-0 h-8 w-8 rounded-full"
                  onClick={handleSend}
                  disabled={!input.trim() || sendMutation.isPending}
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
            <p className="text-[10px] text-muted-foreground/50 text-center mt-1">
              {t("ai:chat.inputHint")}
            </p>
          </div>
        </div>
      </div>

      {/* Memory panel */}
      {memoryOpen && agentId && (
        <MemoryPanel agentId={agentId} onClose={() => setMemoryOpen(false)} />
      )}
    </div>
  )
}
