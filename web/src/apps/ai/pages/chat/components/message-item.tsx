"use client";

import { useState, useCallback, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { ChevronDown, ChevronRight, Wrench, Copy, Check, RotateCcw, ThumbsUp, ThumbsDown, Pencil, Search } from "lucide-react"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import type { UIMessage, DynamicToolUIPart } from "ai"
import { MessageResponse } from "@/components/ai-elements/message"

interface QAPairProps {
  userMessage: UIMessage
  aiMessages: UIMessage[]
  agentName?: string
  isStreaming?: boolean
  streamingContent?: string
  onRegenerate?: () => void
  onEditMessage?: (messageId: number, content: string) => void
  doneMetrics?: { durationMs?: number; inputTokens?: number; outputTokens?: number }
  streamingExtras?: React.ReactNode
}

// User query — right-aligned pill (ChatGPT style)
function UserQuery({
  content,
  images,
  messageId,
  onEdit,
}: {
  content: string
  images?: string[]
  messageId?: number
  onEdit?: (messageId: number, content: string) => void
}) {
  const { t } = useTranslation(["ai"])
  const [editing, setEditing] = useState(false)
  const [editContent, setEditContent] = useState(content)

  const handleSave = useCallback(() => {
    const trimmed = editContent.trim()
    if (!trimmed || !messageId || !onEdit) return
    onEdit(messageId, trimmed)
    setEditing(false)
  }, [editContent, messageId, onEdit])

  const handleCancel = useCallback(() => {
    setEditContent(content)
    setEditing(false)
  }, [content])

  if (editing) {
    return (
      <div className="flex justify-end mb-6">
        <div className="max-w-[70%] w-full">
          <div className="rounded-3xl bg-secondary p-4">
            <textarea
              value={editContent}
              onChange={(e) => setEditContent(e.target.value)}
              className="w-full min-h-[60px] max-h-[200px] resize-none bg-transparent text-sm leading-relaxed focus:outline-none"
              autoFocus
            />
            <div className="flex items-center gap-2 mt-3">
              <Button size="sm" onClick={handleSave} disabled={!editContent.trim()}>
                {t("ai:chat.saveAndRegenerate")}
              </Button>
              <Button size="sm" variant="ghost" onClick={handleCancel}>
                {t("ai:chat.cancelEdit")}
              </Button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="group flex justify-end mb-6">
      <div className="flex items-start gap-1.5 max-w-[70%]">
        {onEdit && messageId != null && !Number.isNaN(messageId) && (
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 shrink-0 mt-1 text-muted-foreground/0 group-hover:text-muted-foreground hover:!text-foreground transition-colors"
            onClick={() => setEditing(true)}
          >
            <Pencil className="h-3.5 w-3.5" />
          </Button>
        )}
        <div className="rounded-3xl bg-secondary px-5 py-2.5">
          {images && images.length > 0 && (
            <div className="flex flex-wrap gap-2 mb-2">
              {images.map((src, idx) => (
                <img
                  key={idx}
                  src={src}
                  alt={`image-${idx}`}
                  className="max-h-48 max-w-xs rounded-xl object-cover"
                />
              ))}
            </div>
          )}
          {content && <div className="text-[15px] leading-relaxed whitespace-pre-wrap">{content}</div>}
        </div>
      </div>
    </div>
  )
}

// Tool call component with rich rendering
function ToolCallDisplay({
  toolName,
  toolArgs,
  durationMs,
}: {
  toolName: string
  toolArgs?: string
  durationMs?: number
}) {
  const { t } = useTranslation(["ai"])
  const [expanded, setExpanded] = useState(false)

  const isKnowledgeSearch = toolName === "search_knowledge"
  let parsedArgs: Record<string, unknown> | null = null
  try {
    if (toolArgs) parsedArgs = JSON.parse(toolArgs)
  } catch { /* ignore */ }

  return (
    <div className="py-2 my-2 rounded-lg border bg-muted/30 px-3">
      <button
        type="button"
        className="flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground transition-colors w-full"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center justify-center h-5 w-5 rounded bg-amber-100 dark:bg-amber-900/30 shrink-0">
          {isKnowledgeSearch
            ? <Search className="h-3 w-3 text-amber-600 dark:text-amber-400" />
            : <Wrench className="h-3 w-3 text-amber-600 dark:text-amber-400" />}
        </div>
        {expanded ? <ChevronDown className="h-3 w-3 shrink-0" /> : <ChevronRight className="h-3 w-3 shrink-0" />}
        <span className="truncate">
          {isKnowledgeSearch && parsedArgs
            ? `${t("ai:tools.toolDefs.search_knowledge.name")}: "${parsedArgs.query ?? ""}`
            : t("ai:chat.toolCall", { name: toolName })}
        </span>
        {durationMs != null && (
          <span className="ml-auto text-[10px] text-muted-foreground/60 shrink-0">
            {(durationMs / 1000).toFixed(1)}s
          </span>
        )}
      </button>
      {expanded && toolArgs && (
        <pre className="mt-2 text-xs bg-muted rounded-md p-3 overflow-auto max-h-48 font-mono">
          {toolArgs}
        </pre>
      )}
    </div>
  )
}

// Tool result component
function ToolResultDisplay({ content }: { content: string }) {
  const { t } = useTranslation(["ai"])
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="py-2 my-2 ml-4">
      <button
        type="button"
        className="flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        <span>{t("ai:chat.toolResult")}</span>
      </button>
      {expanded && (
        <pre className="mt-2 text-xs bg-muted rounded-md p-3 overflow-auto max-h-48 font-mono">
          {content}
        </pre>
      )}
    </div>
  )
}

// Streaming tool adapter — renders dynamic-tool parts with existing UI
function StreamingTool({ part }: { part: DynamicToolUIPart }) {
  const inputStr = useMemo(() => {
    if (part.input == null) return undefined
    return typeof part.input === "string" ? part.input : JSON.stringify(part.input, null, 2)
  }, [part.input])

  const outputStr = useMemo(() => {
    if (part.output == null) return undefined
    return typeof part.output === "string" ? part.output : JSON.stringify(part.output, null, 2)
  }, [part.output])

  if (part.state === "input-available" || part.state === "input-streaming") {
    return <ToolCallDisplay toolName={part.toolName} toolArgs={inputStr} />
  }

  return (
    <>
      <ToolCallDisplay toolName={part.toolName} toolArgs={inputStr} />
      {outputStr && <ToolResultDisplay content={outputStr} />}
      {part.errorText && <ToolResultDisplay content={part.errorText} />}
    </>
  )
}

// AI Response display
export function AIResponse({
  content,
  agentName,
  isStreaming,
  onRegenerate,
  doneMetrics,
}: {
  content: string
  agentName?: string
  isStreaming?: boolean
  onRegenerate?: () => void
  doneMetrics?: { durationMs?: number; inputTokens?: number; outputTokens?: number }
}) {
  const { t } = useTranslation(["ai"])
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error("Copy failed")
    }
  }, [content])

  const metricsText = useMemo(() => {
    if (!doneMetrics?.durationMs) return null
    const parts: string[] = []
    const durationSec = (doneMetrics.durationMs / 1000).toFixed(1)
    if (doneMetrics.outputTokens && doneMetrics.durationMs > 0) {
      const tokPerSec = Math.round((doneMetrics.outputTokens / doneMetrics.durationMs) * 1000)
      parts.push(`${tokPerSec} tok/s`)
    }
    parts.push(`${durationSec}s`)
    if (doneMetrics.outputTokens) {
      parts.push(`${doneMetrics.outputTokens} tokens`)
    }
    return parts.join(" · ")
  }, [doneMetrics])

  return (
    <div className="mb-6">
      {agentName && (
        <div className="text-xs font-medium text-muted-foreground mb-1.5">{agentName}</div>
      )}
      <div className="text-base leading-relaxed">
        {content ? <MessageResponse>{content}</MessageResponse> : null}
        {isStreaming && (
          <span className="inline-block w-2 h-4 bg-foreground/40 ml-1 animate-pulse" />
        )}
      </div>

      {!isStreaming && content && (
        <div className="flex items-center gap-1 mt-3">
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
            onClick={handleCopy}
          >
            {copied
              ? <><Check className="h-3.5 w-3.5 mr-1" />{t("ai:chat.copied")}</>
              : <><Copy className="h-3.5 w-3.5 mr-1" />{t("ai:chat.copy")}</>}
          </Button>
          {onRegenerate && (
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
              onClick={onRegenerate}
            >
              <RotateCcw className="h-3.5 w-3.5 mr-1" />
              {t("ai:chat.regenerate")}
            </Button>
          )}
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground">
            <ThumbsUp className="h-3.5 w-3.5" />
          </Button>
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground">
            <ThumbsDown className="h-3.5 w-3.5" />
          </Button>
          {metricsText && (
            <span className="ml-auto text-[10px] text-muted-foreground/50">{metricsText}</span>
          )}
        </div>
      )}
    </div>
  )
}

// QA Pair component — document-flow layout
export function QAPair({
  userMessage,
  aiMessages,
  agentName,
  isStreaming,
  streamingContent,
  onRegenerate,
  onEditMessage,
  doneMetrics,
  streamingExtras,
}: QAPairProps) {
  const userImages = (userMessage.metadata as { images?: string[] } | undefined)?.images
  const userText = userMessage.parts
    ?.filter((p): p is { type: "text"; text: string } => p.type === "text")
    .map((p) => p.text)
    .join("") || ""

  // Separate historical tools and main AI messages
  const toolMessages = aiMessages.filter((m) => {
    const meta = m.metadata as { originalRole?: string } | undefined
    return meta?.originalRole === "tool_call" || meta?.originalRole === "tool_result"
  })

  const mainAiMessages = aiMessages.filter((m) => {
    const meta = m.metadata as { originalRole?: string } | undefined
    return !["tool_call", "tool_result"].includes(meta?.originalRole || "")
  })

  const mainAiMessage = mainAiMessages[mainAiMessages.length - 1]

  // Extract streaming tool parts from the active assistant message
  const streamingToolParts = mainAiMessage?.parts?.filter(
    (p): p is DynamicToolUIPart => p.type === "dynamic-tool"
  ) || []

  const mainContent = streamingContent || (
    mainAiMessage?.parts
      ?.filter((p): p is { type: "text"; text: string } => p.type === "text")
      .map((p) => p.text)
      .join("") || ""
  )

  return (
    <div className="py-6">
      <UserQuery
        content={userText}
        images={userImages}
        messageId={Number(userMessage.id)}
        onEdit={onEditMessage}
      />

      {/* Historical tool calls/results */}
      {toolMessages.map((tool) => {
        const meta = tool.metadata as {
          originalRole?: string
          tool_name?: string
          tool_args?: string
          duration_ms?: number
        } | undefined
        if (meta?.originalRole === "tool_call") {
          return (
            <ToolCallDisplay
              key={tool.id}
              toolName={meta?.tool_name || "unknown"}
              toolArgs={meta?.tool_args}
              durationMs={meta?.duration_ms}
            />
          )
        }
        const text = tool.parts
          ?.filter((p): p is { type: "text"; text: string } => p.type === "text")
          .map((p) => p.text)
          .join("") || ""
        return <ToolResultDisplay key={tool.id} content={text} />
      })}

      {/* Streaming tool parts */}
      {streamingToolParts.map((part, idx) => (
        <StreamingTool key={`${part.toolCallId}-${idx}`} part={part} />
      ))}

      {/* Streaming extras (thinking block, plan progress, loading dots) */}
      {streamingExtras}

      {/* AI response */}
      {(mainAiMessage || streamingContent) && (
        <AIResponse
          content={mainContent}
          agentName={agentName}
          isStreaming={!!streamingContent && isStreaming}
          onRegenerate={onRegenerate}
          doneMetrics={doneMetrics}
        />
      )}
    </div>
  )
}

export type { QAPairProps }
