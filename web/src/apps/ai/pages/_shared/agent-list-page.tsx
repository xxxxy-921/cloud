import { useState } from "react"
import { useNavigate } from "react-router"
import { useTranslation } from "react-i18next"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  Plus, Search, Bot, SquareTerminal, MessageSquare,
  MoreHorizontal, Pencil, ExternalLink, Trash2, Loader2,
} from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { sessionApi, type AgentInfo } from "@/lib/api"
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

const TYPE_CONFIG: Record<string, { icon: typeof Bot }> = {
  assistant: { icon: Bot },
  coding: { icon: SquareTerminal },
}

export interface AgentListPageConfig {
  agentType: "assistant" | "coding"
  /** i18n namespace key, e.g. "assistantAgents" or "codingAgents" */
  i18nKey: string
  basePath: string
  endpoint: string
  queryKey: string
  permissions: {
    create: string
    update: string
    delete: string
  }
  deleteApiFn: (id: number) => Promise<unknown>
}

function AgentCard({
  agent,
  basePath,
  i18nKey,
  onChat,
  chattingId,
  canUpdate,
  canDelete,
  onDelete,
}: {
  agent: AgentInfo
  basePath: string
  i18nKey: string
  onChat: () => void
  chattingId: number | null
  canUpdate: boolean
  canDelete: boolean
  onDelete: () => void
}) {
  const { t } = useTranslation(["ai", "common"])
  const navigate = useNavigate()
  const config = TYPE_CONFIG[agent.type] ?? { icon: Bot }
  const Icon = config.icon
  const isChatting = chattingId === agent.id

  return (
    <div className={`group relative flex min-h-[164px] flex-col rounded-xl border bg-card p-4 transition-all duration-200 hover:border-border/90 hover:shadow-sm ${!agent.isActive ? "opacity-55" : ""}`}>
      <div className="flex items-start gap-3">
        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border bg-muted/35">
          <Icon className="h-5 w-5 text-foreground/80" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h3 className="truncate text-base font-semibold leading-tight">{agent.name}</h3>
              </div>
              <p className="mt-1.5 line-clamp-2 text-sm leading-5 text-muted-foreground">
                {agent.description || t(`ai:agents.agentTypes.${agent.type}`)}
              </p>
            </div>

            {(canUpdate || canDelete) && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 shrink-0 opacity-0 transition-opacity group-hover:opacity-100"
                  >
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  {canUpdate && (
                    <DropdownMenuItem onClick={() => navigate(`${basePath}/${agent.id}/edit`)}>
                      <Pencil className="mr-2 h-3.5 w-3.5" />
                      {t("common:edit")}
                    </DropdownMenuItem>
                  )}
                  <DropdownMenuItem onClick={() => navigate(`${basePath}/${agent.id}`)}>
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
        </div>
      </div>

      <div className="mt-auto flex items-center justify-between border-t pt-3">
        <Badge variant="secondary" className="h-7 rounded-full px-2.5 text-[11px] font-medium text-foreground">
          {agent.isActive ? t("ai:statusLabels.active") : t("ai:statusLabels.inactive")}
        </Badge>
        <Button
          variant="ghost"
          size="sm"
          className="h-8 gap-1.5 px-2.5 text-xs font-medium"
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

export function AgentListPage({ config }: { config: AgentListPageConfig }) {
  const { t } = useTranslation(["ai", "common"])
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [deleteTarget, setDeleteTarget] = useState<AgentInfo | null>(null)

  const canCreate = usePermission(config.permissions.create)
  const canUpdate = usePermission(config.permissions.update)
  const canDelete = usePermission(config.permissions.delete)

  const {
    keyword, setKeyword, page, setPage,
    items: agents, total, totalPages, isLoading, handleSearch,
  } = useListPage<AgentInfo>({
    queryKey: config.queryKey,
    endpoint: config.endpoint,
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => config.deleteApiFn(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [config.queryKey] })
      toast.success(t(`ai:${config.i18nKey}.deleteSuccess`))
      setDeleteTarget(null)
    },
    onError: (err) => toast.error(err.message),
  })

  const createSessionMutation = useMutation({
    mutationFn: (agentId: number) => sessionApi.create(agentId),
    onSuccess: (session) => navigate(`/ai/chat/${session.id}`),
    onError: (err) => toast.error(err.message),
  })

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t(`ai:${config.i18nKey}.title`)}</h2>
        {canCreate && (
          <Button size="sm" onClick={() => navigate(`${config.basePath}/create`)}>
            <Plus className="mr-1.5 h-4 w-4" />
            {t(`ai:${config.i18nKey}.create`)}
          </Button>
        )}
      </div>

      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={t(`ai:${config.i18nKey}.searchPlaceholder`)}
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
          <p className="text-sm text-muted-foreground">{t(`ai:${config.i18nKey}.empty`)}</p>
          {canCreate && (
            <p className="text-xs text-muted-foreground">{t(`ai:${config.i18nKey}.emptyHint`)}</p>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(340px,1fr))] gap-4">
          {agents.map((agent) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              basePath={config.basePath}
              i18nKey={config.i18nKey}
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

      <AlertDialog open={!!deleteTarget} onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t(`ai:${config.i18nKey}.deleteTitle`)}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(`ai:${config.i18nKey}.deleteDesc`, { name: deleteTarget?.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
              disabled={deleteMutation.isPending}
            >
              {t(`ai:${config.i18nKey}.confirmDelete`)}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
