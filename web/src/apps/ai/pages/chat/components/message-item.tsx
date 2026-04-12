import { useState, useCallback, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { ChevronDown, ChevronRight, Wrench, Copy, RotateCcw, ThumbsUp, ThumbsDown } from "lucide-react"
import { type SessionMessage } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import ReactMarkdown from "react-markdown"
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter"
import { vscDarkPlus } from "react-syntax-highlighter/dist/esm/styles/prism"
import remarkGfm from "remark-gfm"

interface MessageItemProps {
  message: SessionMessage
  isStreaming?: boolean
  onRegenerate?: () => void
}

interface QAPairProps {
  userMessage: SessionMessage
  aiMessage?: SessionMessage
  isStreaming?: boolean
  streamingContent?: string
  onRegenerate?: () => void
}

// Code block component with copy button
function CodeBlock({
  language,
  code,
}: {
  language: string
  code: string
}) {
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
      toast.success("已复制到剪贴板")
    } catch {
      toast.error("复制失败")
    }
  }, [code])

  return (
    <div className="relative group rounded-lg overflow-hidden my-4">
      <div className="flex items-center justify-between px-3 py-1.5 bg-zinc-900 border-b border-zinc-800">
        <span className="text-xs text-zinc-400 font-mono">{language || "text"}</span>
        <button
          type="button"
          onClick={handleCopy}
          className="text-xs text-zinc-400 hover:text-zinc-200 transition-colors"
        >
          {copied ? "已复制" : "复制"}
        </button>
      </div>
      <SyntaxHighlighter
        language={language || "text"}
        style={vscDarkPlus}
        customStyle={{
          margin: 0,
          padding: "1rem",
          fontSize: "0.875rem",
          background: "#18181b",
        }}
      >
        {code}
      </SyntaxHighlighter>
    </div>
  )
}

// Markdown components configuration
const markdownComponents = {
  h1: ({ children }: { children?: React.ReactNode }) => <h1 className="text-2xl font-semibold mt-8 mb-4">{children}</h1>,
  h2: ({ children }: { children?: React.ReactNode }) => <h2 className="text-xl font-semibold mt-6 mb-3">{children}</h2>,
  h3: ({ children }: { children?: React.ReactNode }) => <h3 className="text-lg font-semibold mt-5 mb-2">{children}</h3>,
  p: ({ children }: { children?: React.ReactNode }) => <p className="leading-7 mb-4 last:mb-0">{children}</p>,
  ul: ({ children }: { children?: React.ReactNode }) => <ul className="list-disc pl-6 space-y-1 mb-4">{children}</ul>,
  ol: ({ children }: { children?: React.ReactNode }) => <ol className="list-decimal pl-6 space-y-1 mb-4">{children}</ol>,
  li: ({ children }: { children?: React.ReactNode }) => <li className="leading-7">{children}</li>,
  blockquote: ({ children }: { children?: React.ReactNode }) => (
    <blockquote className="border-l-2 border-border pl-4 italic text-muted-foreground my-4">
      {children}
    </blockquote>
  ),
  code: ({ className, children }: { className?: string; children?: React.ReactNode }) => {
    const language = className?.replace("language-", "") ?? ""
    const code = String(children).replace(/\n$/, "")

    // Inline code (no language or single line)
    if (!language && !code.includes("\n")) {
      return (
        <code className="bg-muted px-1.5 py-0.5 rounded text-sm font-mono">
          {children}
        </code>
      )
    }

    // Code block
    return <CodeBlock language={language} code={code} />
  },
  table: ({ children }: { children?: React.ReactNode }) => (
    <div className="overflow-x-auto my-4">
      <table className="w-full border-collapse">
        {children}
      </table>
    </div>
  ),
  thead: ({ children }: { children?: React.ReactNode }) => <thead className="bg-muted">{children}</thead>,
  th: ({ children }: { children?: React.ReactNode }) => (
    <th className="border border-border px-3 py-2 text-left font-semibold text-sm">
      {children}
    </th>
  ),
  td: ({ children }: { children?: React.ReactNode }) => (
    <td className="border border-border px-3 py-2 text-sm">{children}</td>
  ),
  tr: ({ children }: { children?: React.ReactNode }) => <tr className="even:bg-muted/50">{children}</tr>,
  a: ({ children, href }: { children?: React.ReactNode; href?: string }) => (
    <a href={href} className="text-primary hover:underline" target="_blank" rel="noopener noreferrer">
      {children}
    </a>
  ),
  hr: () => <hr className="my-6 border-border" />,
}

// Memoized markdown content to prevent re-rendering during streaming
const MarkdownContent = ({ content }: { content: string }) => {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={markdownComponents}
    >
      {content}
    </ReactMarkdown>
  )
}

// Tool call component
function ToolCall({ message }: { message: SessionMessage }) {
  const { t } = useTranslation(["ai"])
  const [expanded, setExpanded] = useState(false)
  const meta = message.metadata as { tool_name?: string; tool_args?: string } | undefined

  return (
    <div className="py-2 my-2 rounded-lg border bg-muted/30 px-3">
      <button
        type="button"
        className="flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground transition-colors w-full"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center justify-center h-5 w-5 rounded bg-amber-100 dark:bg-amber-900/30 shrink-0">
          <Wrench className="h-3 w-3 text-amber-600 dark:text-amber-400" />
        </div>
        {expanded ? <ChevronDown className="h-3 w-3 shrink-0" /> : <ChevronRight className="h-3 w-3 shrink-0" />}
        <span className="truncate">{t("ai:chat.toolCall", { name: meta?.tool_name ?? "unknown" })}</span>
      </button>
      {expanded && meta?.tool_args && (
        <pre className="mt-2 text-xs bg-muted rounded-md p-3 overflow-auto max-h-48 font-mono">
          {meta.tool_args}
        </pre>
      )}
    </div>
  )
}

// Tool result component
function ToolResult({ message }: { message: SessionMessage }) {
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
          {message.content}
        </pre>
      )}
    </div>
  )
}

// User query display (right-aligned pill/tag)
function UserQuery({ content }: { content: string }) {
  return (
    <div className="flex items-start gap-3 mb-4 justify-end">
      <div className="flex-1 flex justify-end">
        <div className="inline-flex items-center px-4 py-2 rounded-2xl bg-primary text-primary-foreground text-sm font-medium">
          {content}
        </div>
      </div>
    </div>
  )
}

// AI Response display (main content area)
export function AIResponse({
  content,
  isStreaming,
  onRegenerate,
}: {
  content: string
  isStreaming?: boolean
  onRegenerate?: () => void
}) {
  const { t } = useTranslation(["ai"])
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
      toast.success("已复制到剪贴板")
    } catch {
      toast.error("复制失败")
    }
  }, [content])

  // Memoize markdown rendering to prevent flickering during streaming
  const markdownContent = useMemo(() => {
    return <MarkdownContent content={content} />
  }, [content])

  return (
    <div className="flex items-start gap-3">
      <div className="flex-1 min-w-0">
        <div className="text-base leading-relaxed">
          {markdownContent}
          {isStreaming && (
            <span className="inline-block w-2 h-4 bg-foreground/40 ml-1 animate-pulse" />
          )}
        </div>

        {/* Message actions */}
        {!isStreaming && (
          <div className="flex items-center gap-1 mt-3 opacity-0 hover:opacity-100 transition-opacity">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
              onClick={handleCopy}
            >
              <Copy className="h-3.5 w-3.5 mr-1" />
              {copied ? t("ai:chat.copied") : t("ai:chat.copy")}
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
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground"
            >
              <ThumbsUp className="h-3.5 w-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground"
            >
              <ThumbsDown className="h-3.5 w-3.5" />
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}

// QA Pair component (Kimi style layout)
export function QAPair({
  userMessage,
  aiMessage,
  isStreaming,
  streamingContent,
  onRegenerate,
}: QAPairProps) {
  const displayContent = isStreaming && streamingContent ? streamingContent : (aiMessage?.content ?? "")

  return (
    <div className="py-6 border-b border-border/50 last:border-0">
      {/* User query at top (Kimi style pill) */}
      <UserQuery content={userMessage.content} />

      {/* AI response below */}
      {aiMessage && (
        <AIResponse
          content={displayContent}
          isStreaming={isStreaming}
          onRegenerate={onRegenerate}
        />
      )}
    </div>
  )
}

// Legacy single message item (for tool calls and simple display)
export function MessageItem({ message, isStreaming, onRegenerate }: MessageItemProps) {
  if (message.role === "tool_call") {
    return <ToolCall message={message} />
  }

  if (message.role === "tool_result") {
    return <ToolResult message={message} />
  }

  // For standalone messages, use simple display
  if (message.role === "user") {
    return (
      <div className="py-4">
        <UserQuery content={message.content} />
      </div>
    )
  }

  return (
    <div className="py-4">
      <AIResponse
        content={message.content}
        isStreaming={isStreaming}
        onRegenerate={onRegenerate}
      />
    </div>
  )
}

export type { QAPairProps }
