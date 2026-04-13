import { useState } from "react"
import { useNavigate } from "react-router"
import { useTranslation } from "react-i18next"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  Plus, Search, Bot, Code2, BrainCircuit, MessageSquare,
  MoreHorizontal, Pencil, ExternalLink, Trash2, Loader2,
} from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { agentApi, sessionApi, type AgentInfo } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  DataTablePagination,
  DataTableToolbar,
  DataTableToolbarGroup,
} from "@/components/ui/data-table"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem,
  DropdownMenuSeparator, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { AgentSheet } from "./components/agent-sheet"

const TYPE_CONFIG: Record<string, { icon: typeof Bot; gradient: string }> = {
  assistant: { icon: BrainCircuit, gradient: "from-violet-500/10 to-indigo-500/10" },
  coding: { icon: Code2, gradient: "from-emerald-500/10 to-teal-500/10" },
}

function AgentCard({
  agent,
  onEdit,
  onChat,
  chattingId,
  canUpdate,
  canDelete,
  onDelete,
}: {
  agent: AgentInfo
  onEdit: () => void
  onChat: () => void
  chattingId: number | null
  canUpdate: boolean
  canDelete: boolean
  onDelete: () => void
}) {
  const { t } = useTranslation(["ai", "common"])
  const navigate = useNavigate()
  const config = TYPE_CONFIG[agent.type] ?? { icon: Bot, gradient: "from-gray-500/10 to-gray-400/10" }
  const Icon = config.icon
  const isChatting = chattingId === agent.id

  return (
    <div className={`group relative flex flex-col gap-3 rounded-lg border bg-card p-4 transition-all duration-200 hover:shadow-sm hover:border-primary/30 ${!agent.isActive ? "opacity-50" : ""}`}>
      {/* Top row: icon + name + menu */}
      <div className="flex items-start gap-3">
        <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-gradient-to-br ${config.gradient}`}>
          <Icon className="h-4.5 w-4.5 text-foreground/70" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h3 className="font-medium text-sm truncate">{agent.name}</h3>
            <Badge variant="outline" className="shrink-0 text-[10px] px-1.5 py-0">
              {t(`ai:agents.agentTypes.${agent.type}`)}
            </Badge>
          </div>
          <p className="text-xs text-muted-foreground truncate mt-0.5">
            {agent.description || t(`ai:agents.agentTypes.${agent.type}`)}
          </p>
        </div>

        {/* Action menu */}
        {(canUpdate || canDelete) && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {canUpdate && (
                <DropdownMenuItem onClick={onEdit}>
                  <Pencil className="mr-2 h-3.5 w-3.5" />
                  {t("common:edit")}
                </DropdownMenuItem>
              )}
              <DropdownMenuItem onClick={() => navigate(`/ai/agents/${agent.id}`)}>
                <ExternalLink className="mr-2 h-3.5 w-3.5" />
                {t("ai:agents.viewDetail")}
              </DropdownMenuItem>
              {canDelete && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={onDelete}>
                    <Trash2 className="mr-2 h-3.5 w-3.5" />
                    {t("common:delete")}
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      {/* Bottom: status + chat button */}
      <div className="mt-auto flex items-center justify-between pt-1">
        <Badge variant={agent.isActive ? "default" : "secondary"} className="text-[10px]">
          {agent.isActive ? t("ai:statusLabels.active") : t("ai:statusLabels.inactive")}
        </Badge>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 gap-1.5 text-xs"
          disabled={!agent.isActive || isChatting}
          onClick={(e) => { e.stopPropagation(); onChat() }}
        >
          {isChatting ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <MessageSquare className="h-3.5 w-3.5" />
          )}
          {t("ai:chat.startChat")}
        </Button>
      </div>
    </div>
  )
}

export function Component() {
  const { t } = useTranslation(["ai", "common"])
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<AgentInfo | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<AgentInfo | null>(null)

  const canCreate = usePermission("ai:agent:create")
  const canUpdate = usePermission("ai:agent:update")
  const canDelete = usePermission("ai:agent:delete")

  const {
    keyword, setKeyword, page, setPage,
    items: agents, total, totalPages, isLoading, handleSearch,
  } = useListPage<AgentInfo>({
    queryKey: "ai-agents",
    endpoint: "/api/v1/ai/agents",
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => agentApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-agents"] })
      toast.success(t("ai:agents.deleteSuccess"))
      setDeleteTarget(null)
    },
    onError: (err) => toast.error(err.message),
  })

  const createSessionMutation = useMutation({
    mutationFn: (agentId: number) => sessionApi.create(agentId),
    onSuccess: (session) => navigate(`/ai/chat/${session.id}`),
    onError: (err) => toast.error(err.message),
  })

  function handleCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function handleEdit(item: AgentInfo) {
    setEditing(item)
    setFormOpen(true)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t("ai:agents.title")}</h2>
        {canCreate && (
          <Button size="sm" onClick={handleCreate}>
            <Plus className="mr-1.5 h-4 w-4" />
            {t("ai:agents.create")}
          </Button>
        )}
      </div>

      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={t("ai:agents.searchPlaceholder")}
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                className="pl-8"
              />
            </div>
            <Button type="submit" variant="outline">
              {t("common:search")}
            </Button>
          </form>
        </DataTableToolbarGroup>
      </DataTableToolbar>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : agents.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-12">
          <Bot className="h-10 w-10 text-muted-foreground/50" />
          <p className="text-sm text-muted-foreground">{t("ai:agents.empty")}</p>
          {canCreate && (
            <p className="text-xs text-muted-foreground">{t("ai:agents.emptyHint")}</p>
          )}
        </div>
      ) : (
        <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {agents.map((agent) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              onEdit={() => handleEdit(agent)}
              onChat={() => createSessionMutation.mutate(agent.id)}
              chattingId={createSessionMutation.isPending ? (createSessionMutation.variables ?? null) : null}
              canUpdate={canUpdate}
              canDelete={canDelete}
              onDelete={() => setDeleteTarget(agent)}
            />
          ))}
        </div>
      )}

      <DataTablePagination total={total} page={page} totalPages={totalPages} onPageChange={setPage} />

      <AgentSheet open={formOpen} onOpenChange={setFormOpen} agent={editing} />

      {/* Delete confirmation dialog */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("ai:agents.deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("ai:agents.deleteDesc", { name: deleteTarget?.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
              disabled={deleteMutation.isPending}
            >
              {t("ai:agents.confirmDelete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
