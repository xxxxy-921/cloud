import { useNavigate } from "react-router"
import { useTranslation } from "react-i18next"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, MessageSquare, Trash2 } from "lucide-react"
import { sessionApi, type AgentSession } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { cn } from "@/lib/utils"

interface SessionSidebarProps {
  agentId?: number
  currentSessionId?: number
  collapsed?: boolean
}

export function SessionSidebar({ agentId, currentSessionId, collapsed = false }: SessionSidebarProps) {
  const { t } = useTranslation(["ai"])
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data } = useQuery({
    queryKey: ["ai-sessions", agentId],
    queryFn: () => sessionApi.list({ agentId, pageSize: 50 }),
    enabled: !!agentId,
  })
  const sessions = data?.items ?? []

  const createMutation = useMutation({
    mutationFn: () => sessionApi.create(agentId!),
    onSuccess: (session) => {
      queryClient.invalidateQueries({ queryKey: ["ai-sessions"] })
      navigate(`/ai/chat/${session.id}`)
    },
    onError: (err) => toast.error(err.message),
  })

  const deleteMutation = useMutation({
    mutationFn: (sid: number) => sessionApi.delete(sid),
    onSuccess: (_, sid) => {
      queryClient.invalidateQueries({ queryKey: ["ai-sessions"] })
      toast.success(t("ai:chat.sessionDeleted"))
      if (sid === currentSessionId) {
        navigate("/ai/chat")
      }
    },
    onError: (err) => toast.error(err.message),
  })

  if (collapsed) {
    return (
      <div className="w-12 border-r flex flex-col shrink-0 transition-all duration-200">
        <div className="p-2 border-b">
          <Button
            variant="outline"
            size="icon"
            className="w-8 h-8"
            disabled={!agentId || createMutation.isPending}
            onClick={() => createMutation.mutate()}
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
        <ScrollArea className="flex-1">
          <div className="p-1.5 space-y-1">
            {sessions.map((s: AgentSession) => (
              <button
                key={s.id}
                type="button"
                className={cn(
                  "w-full flex items-center justify-center h-8 rounded-md hover:bg-accent",
                  s.id === currentSessionId && "bg-accent"
                )}
                onClick={() => navigate(`/ai/chat/${s.id}`)}
                title={s.title || `#${s.id}`}
              >
                <MessageSquare className={cn(
                  "h-4 w-4 shrink-0",
                  s.id === currentSessionId ? "text-foreground" : "text-muted-foreground"
                )} />
              </button>
            ))}
          </div>
        </ScrollArea>
      </div>
    )
  }

  return (
    <div className="w-64 border-r flex flex-col shrink-0 transition-all duration-200 hidden md:flex">
      <div className="p-3 border-b">
        <Button
          variant="outline"
          size="sm"
          className="w-full"
          disabled={!agentId || createMutation.isPending}
          onClick={() => createMutation.mutate()}
        >
          <Plus className="mr-1.5 h-3.5 w-3.5" />
          {t("ai:chat.newChat")}
        </Button>
      </div>
      <ScrollArea className="flex-1">
        <div className="p-2 space-y-0.5">
          {sessions.length === 0 ? (
            <p className="text-xs text-muted-foreground text-center py-4">{t("ai:chat.noSessions")}</p>
          ) : (
            sessions.map((s: AgentSession) => (
              <div
                key={s.id}
                className={cn(
                  "group flex items-center gap-2 rounded-md px-2.5 py-2 cursor-pointer hover:bg-accent text-sm",
                  s.id === currentSessionId && "bg-accent",
                )}
                onClick={() => navigate(`/ai/chat/${s.id}`)}
              >
                <MessageSquare className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                <span className="flex-1 truncate">{s.title || `#${s.id}`}</span>
                <button
                  type="button"
                  className="opacity-0 group-hover:opacity-100 shrink-0 p-0.5 rounded hover:bg-destructive/10"
                  onClick={(e) => {
                    e.stopPropagation()
                    deleteMutation.mutate(s.id)
                  }}
                >
                  <Trash2 className="h-3 w-3 text-destructive" />
                </button>
              </div>
            ))
          )}
        </div>
      </ScrollArea>
    </div>
  )
}
