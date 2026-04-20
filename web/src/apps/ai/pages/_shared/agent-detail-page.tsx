import { useState, useMemo } from "react"
import { useParams, useNavigate, Link } from "react-router"
import { useTranslation } from "react-i18next"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { ArrowLeft, BookOpen, Bot, BrainCircuit, ChevronRight, Code2, Loader2, MessageSquare, Package, Pencil, Plug, Share2, Trash2, Wrench } from "lucide-react"
import { sessionApi, api, type AgentWithBindings, type AgentSession, type PaginatedResponse } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import {
  Sheet, SheetContent, SheetHeader, SheetTitle,
} from "@/components/ui/sheet"
import { cn, formatDateTime } from "@/lib/utils"

const TYPE_ICON: Record<string, typeof Bot> = {
  assistant: BrainCircuit,
  coding: Code2,
}

const SESSION_STATUS_VARIANT: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  running: "default",
  completed: "secondary",
  cancelled: "outline",
  error: "destructive",
}

export interface AgentDetailPageConfig {
  agentType: "assistant" | "coding"
  i18nKey: string
  basePath: string
  queryKey: string
  getApiFn: (id: number) => Promise<AgentWithBindings>
  deleteApiFn: (id: number) => Promise<unknown>
  listQueryKey: string
}

interface NamedItem {
  id: number
  name: string
  displayName?: string
  description?: string
}

interface ToolkitGroup {
  toolkit: string
  tools: NamedItem[]
}

const BINDING_ROWS = [
  { key: "tools", icon: Wrench, labelKey: "ai:agents.tools" },
  { key: "mcp", icon: Plug, labelKey: "ai:agents.mcpServers" },
  { key: "skills", icon: Package, labelKey: "ai:agents.skills" },
  { key: "kb", icon: BookOpen, labelKey: "ai:agents.knowledgeBases" },
  { key: "kg", icon: Share2, labelKey: "ai:agents.knowledgeGraphs" },
] as const

function AgentBindingsCard({ agent }: { agent: AgentWithBindings }) {
  const { t } = useTranslation(["ai"])
  const [openType, setOpenType] = useState<string | null>(null)

  const { data: toolGroups = [] } = useQuery({
    queryKey: ["ai-agent-binding-tools"],
    queryFn: () =>
      api.get<{ items: ToolkitGroup[] }>("/api/v1/ai/tools").then((r) => r?.items ?? []),
  })

  const { data: mcpItems = [] } = useQuery({
    queryKey: ["ai-binding-mcp-servers"],
    queryFn: () =>
      api.get<PaginatedResponse<NamedItem>>("/api/v1/ai/mcp-servers?pageSize=100").then((r) => r?.items ?? []),
  })

  const { data: skillItems = [] } = useQuery({
    queryKey: ["ai-binding-skills"],
    queryFn: () =>
      api.get<PaginatedResponse<NamedItem>>("/api/v1/ai/skills?pageSize=100").then((r) => r?.items ?? []),
  })

  const { data: kbItems = [] } = useQuery({
    queryKey: ["ai-binding-knowledge-bases"],
    queryFn: () =>
      api.get<PaginatedResponse<NamedItem>>("/api/v1/ai/knowledge-bases?pageSize=100").then((r) => r?.items ?? []),
  })

  const { data: kgItems = [] } = useQuery({
    queryKey: ["ai-binding-knowledge-graphs"],
    queryFn: () =>
      api.get<PaginatedResponse<NamedItem>>("/api/v1/ai/knowledge/graphs?pageSize=100").then((r) => r?.items ?? []),
  })

  const idsMap = useMemo<Record<string, number[]>>(() => ({
    tools: agent.toolIds,
    mcp: agent.mcpServerIds,
    skills: agent.skillIds,
    kb: agent.knowledgeBaseIds,
    kg: agent.knowledgeGraphIds,
  }), [agent.toolIds, agent.mcpServerIds, agent.skillIds, agent.knowledgeBaseIds, agent.knowledgeGraphIds])

  const sheetData = useMemo(() => {
    if (!openType) return { title: "", groups: null as { title: string; items: { name: string; description?: string }[] }[] | null, items: [] as { name: string; description?: string }[] }

    const row = BINDING_ROWS.find((r) => r.key === openType)!
    const ids = idsMap[openType] ?? []
    const label = t(row.labelKey)
    const title = `${label}（${ids.length}）`

    if (openType === "tools") {
      const boundGroups = toolGroups
        .map((g) => ({
          title: t(`ai:tools.toolkits.${g.toolkit}.name`),
          items: g.tools
            .filter((tool) => ids.includes(tool.id))
            .map((tool) => ({
              name: t(`ai:tools.toolDefs.${tool.name}.name`, { defaultValue: tool.displayName || tool.name }),
              description: tool.description
                ? t(`ai:tools.toolDefs.${tool.name}.description`, { defaultValue: tool.description })
                : undefined,
            })),
        }))
        .filter((g) => g.items.length > 0)
      return { title, groups: boundGroups, items: [] }
    }

    const sourceMap: Record<string, NamedItem[]> = { mcp: mcpItems, skills: skillItems, kb: kbItems, kg: kgItems }
    const source = sourceMap[openType] ?? []
    const items = source
      .filter((item) => ids.includes(item.id))
      .map((item) => ({ name: item.displayName || item.name, description: item.description }))
    return { title, groups: null, items }
  }, [openType, toolGroups, mcpItems, skillItems, kbItems, kgItems, idsMap, t])

  return (
    <>
      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-base">{t("ai:agents.bindings")}</CardTitle>
        </CardHeader>
        <CardContent className="divide-y divide-border/45">
          {BINDING_ROWS.map((row) => {
            const Icon = row.icon
            const ids = idsMap[row.key]
            return (
              <button
                key={row.key}
                type="button"
                onClick={() => ids.length > 0 && setOpenType(row.key)}
                className={cn(
                  "flex w-full items-center gap-3 py-3.5 text-left transition-colors first:pt-0 last:pb-0",
                  ids.length > 0 ? "cursor-pointer hover:text-foreground" : "cursor-default opacity-50"
                )}
              >
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-border/55 bg-background/70 text-primary">
                  <Icon className="size-4" />
                </div>
                <span className="flex-1 text-sm font-medium">{t(row.labelKey)}</span>
                <span className="text-sm tabular-nums text-muted-foreground">
                  {t("ai:agents.itemCount", { count: ids.length })}
                </span>
                <ChevronRight className="size-4 text-muted-foreground" />
              </button>
            )
          })}
        </CardContent>
      </Card>

      <Sheet open={openType !== null} onOpenChange={(open) => { if (!open) setOpenType(null) }}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader className="border-b border-border/50 pb-4">
            <SheetTitle>{sheetData.title}</SheetTitle>
          </SheetHeader>
          <div className="flex-1 overflow-y-auto px-4 py-4">
            {sheetData.groups ? (
              <div className="space-y-6">
                {sheetData.groups.map((group) => (
                  <div key={group.title} className="space-y-3">
                    <h4 className="text-sm font-semibold text-foreground">{group.title}</h4>
                    <div className="space-y-2">
                      {group.items.map((item, i) => (
                        <div key={i} className="rounded-xl border border-border/55 bg-background/30 px-4 py-3">
                          <p className="text-sm font-medium text-foreground">{item.name}</p>
                          {item.description && (
                            <p className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">{item.description}</p>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="space-y-2">
                {sheetData.items.map((item, i) => (
                  <div key={i} className="rounded-xl border border-border/55 bg-background/30 px-4 py-3">
                    <p className="text-sm font-medium text-foreground">{item.name}</p>
                    {item.description && (
                      <p className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">{item.description}</p>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>
    </>
  )
}

function AgentConfiguration({ agent }: { agent: AgentWithBindings }) {
  const { t } = useTranslation(["ai"])

  return (
    <div className="space-y-6">
      {/* Basic Info */}
      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-base">{t("ai:agents.sections.basic")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
            <div>
              <span className="text-sm text-muted-foreground">{t("ai:agents.visibility")}</span>
              <p className="font-medium">{t(`ai:agents.visibilityOptions.${agent.visibility}`)}</p>
            </div>
            {agent.description && (
              <div className="col-span-2 sm:col-span-3">
                <span className="text-sm text-muted-foreground">{t("ai:agents.description")}</span>
                <p className="text-sm mt-1">{agent.description}</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Model Config (assistant) */}
      {agent.type === "assistant" && (
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="text-base">{t("ai:agents.sections.modelConfig")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
              <div>
                <span className="text-sm text-muted-foreground">{t("ai:agents.strategy")}</span>
                <p className="font-medium">{agent.strategy ? t(`ai:agents.strategies.${agent.strategy}`) : "-"}</p>
              </div>
              <div>
                <span className="text-sm text-muted-foreground">{t("ai:agents.temperature")}</span>
                <p className="font-medium">{agent.temperature}</p>
              </div>
              <div>
                <span className="text-sm text-muted-foreground">{t("ai:agents.maxTokens")}</span>
                <p className="font-medium">{agent.maxTokens}</p>
              </div>
              <div>
                <span className="text-sm text-muted-foreground">{t("ai:agents.maxTurns")}</span>
                <p className="font-medium">{agent.maxTurns}</p>
              </div>
            </div>
            {agent.systemPrompt && (
              <div className="mt-4">
                <span className="text-sm text-muted-foreground">{t("ai:agents.systemPrompt")}</span>
                <pre className="mt-1 text-sm whitespace-pre-wrap bg-muted/50 rounded-md p-3 max-h-64 overflow-auto">{agent.systemPrompt}</pre>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Runtime Config (coding) */}
      {agent.type === "coding" && (
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="text-base">{t("ai:agents.sections.execution")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
              <div>
                <span className="text-sm text-muted-foreground">{t("ai:agents.runtime")}</span>
                <p className="font-medium">{agent.runtime ? t(`ai:agents.runtimes.${agent.runtime}`) : "-"}</p>
              </div>
              <div>
                <span className="text-sm text-muted-foreground">{t("ai:agents.execMode")}</span>
                <p className="font-medium">{agent.execMode ? t(`ai:agents.execModes.${agent.execMode}`) : "-"}</p>
              </div>
              {agent.workspace && (
                <div className="col-span-2 sm:col-span-3">
                  <span className="text-sm text-muted-foreground">{t("ai:agents.workspace")}</span>
                  <p className="font-mono text-sm">{agent.workspace}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      <AgentBindingsCard agent={agent} />

      {/* Instructions */}
      {agent.instructions && (
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="text-base">{t("ai:agents.sections.instructions")}</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="text-sm whitespace-pre-wrap bg-muted/50 rounded-md p-3 max-h-40 overflow-auto">{agent.instructions}</pre>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function AgentSessions({ agentId }: { agentId: number }) {
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ["ai-agent-sessions", agentId],
    queryFn: () => sessionApi.list({ agentId, pageSize: 50 }),
  })
  const sessions = data?.items ?? []

  const deleteMutation = useMutation({
    mutationFn: (sid: number) => sessionApi.delete(sid),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-agent-sessions"] })
      toast.success(t("ai:chat.sessionDeleted"))
    },
    onError: (err) => toast.error(err.message),
  })

  if (isLoading) return <p className="text-sm text-muted-foreground py-4">{t("common:loading")}</p>

  if (sessions.length === 0) {
    return <p className="text-sm text-muted-foreground py-4">{t("ai:chat.noSessions")}</p>
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>ID</TableHead>
          <TableHead>{t("ai:chat.title")}</TableHead>
          <TableHead>{t("ai:agents.status")}</TableHead>
          <TableHead>{t("common:createdAt")}</TableHead>
          <TableHead className="w-[80px]" />
        </TableRow>
      </TableHeader>
      <TableBody>
        {sessions.map((s: AgentSession) => (
          <TableRow key={s.id}>
            <TableCell className="font-mono text-xs">{s.id}</TableCell>
            <TableCell>{s.title || "-"}</TableCell>
            <TableCell>
              <Badge variant={SESSION_STATUS_VARIANT[s.status] ?? "secondary"}>
                {t(`ai:chat.sessionStatus.${s.status}`)}
              </Badge>
            </TableCell>
            <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
              {formatDateTime(s.createdAt)}
            </TableCell>
            <TableCell>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button variant="ghost" size="sm" className="px-2 text-destructive hover:text-destructive">
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>{t("ai:chat.deleteSession")}</AlertDialogTitle>
                    <AlertDialogDescription>{t("ai:chat.deleteSessionDesc")}</AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
                    <AlertDialogAction onClick={() => deleteMutation.mutate(s.id)} disabled={deleteMutation.isPending}>
                      {t("common:delete")}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function AgentDetailPage({ config }: { config: AgentDetailPageConfig }) {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation(["ai", "common"])
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: agent, isLoading } = useQuery({
    queryKey: [config.queryKey, id],
    queryFn: () => config.getApiFn(Number(id)),
    enabled: !!id,
  })

  const deleteMutation = useMutation({
    mutationFn: () => config.deleteApiFn(Number(id)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [config.listQueryKey] })
      toast.success(t(`ai:${config.i18nKey}.deleteSuccess`))
      navigate(config.basePath)
    },
    onError: (err) => toast.error(err.message),
  })

  const createSessionMutation = useMutation({
    mutationFn: () => sessionApi.create(Number(id)),
    onSuccess: (session) => navigate(`/ai/chat/${session.id}`),
    onError: (err) => toast.error(err.message),
  })

  if (isLoading || !agent) {
    return <div className="py-8 text-center text-muted-foreground">{t("common:loading")}</div>
  }

  const TypeIcon = TYPE_ICON[agent.type] ?? Bot

  return (
    <div className="workspace-page">
      <div className="workspace-page-header gap-4">
        <div className="min-w-0 flex-1">
          <nav className="mb-3 flex items-center gap-1.5 text-sm text-muted-foreground">
            <Link to={config.basePath} className="inline-flex items-center gap-1 transition-colors hover:text-foreground">
              <ArrowLeft className="h-3.5 w-3.5" />
              {t(`ai:${config.i18nKey}.title`)}
            </Link>
            <span className="text-muted-foreground/50">/</span>
            <span className="text-foreground">{agent.name}</span>
          </nav>
          <div className="flex items-start gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-border/60 bg-surface-soft/78 text-foreground/80 shadow-[inset_0_1px_0_rgba(255,255,255,0.72)]">
              <TypeIcon className="h-5 w-5" />
            </div>
            <div className="min-w-0">
              <h2 className="workspace-page-title">{agent.name}</h2>
              <div className="mt-2 flex items-center gap-2">
                <Badge variant={agent.isActive ? "default" : "secondary"}>
                  {agent.isActive ? t("ai:statusLabels.active") : t("ai:statusLabels.inactive")}
                </Badge>
                <span className="text-sm text-muted-foreground">{t(`ai:agents.agentTypes.${agent.type}`)}</span>
              </div>
            </div>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2 lg:justify-end">
          <Button
            variant="outline"
            size="sm"
            disabled={!agent.isActive || createSessionMutation.isPending}
            onClick={() => createSessionMutation.mutate()}
          >
            {createSessionMutation.isPending ? (
              <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
            ) : (
              <MessageSquare className="mr-1.5 h-3.5 w-3.5" />
            )}
            {t("ai:chat.startChat")}
          </Button>
          <Button variant="outline" size="sm" onClick={() => navigate(`${config.basePath}/${id}/edit`)}>
            <Pencil className="mr-1.5 h-3.5 w-3.5" />
            {t("common:edit")}
          </Button>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="outline" size="sm" className="text-destructive hover:text-destructive">
                <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                {t("common:delete")}
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t(`ai:${config.i18nKey}.deleteTitle`)}</AlertDialogTitle>
                <AlertDialogDescription>{t(`ai:${config.i18nKey}.deleteDesc`, { name: agent.name })}</AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
                <AlertDialogAction onClick={() => deleteMutation.mutate()} disabled={deleteMutation.isPending}>
                  {t(`ai:${config.i18nKey}.confirmDelete`)}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
            </AlertDialog>
        </div>
      </div>

      <Tabs defaultValue="config">
        <TabsList className="workspace-surface rounded-xl p-1.5" variant="default">
          <TabsTrigger value="config">{t("ai:agents.tabs.config")}</TabsTrigger>
          <TabsTrigger value="sessions">{t("ai:agents.tabs.sessions")}</TabsTrigger>
        </TabsList>
        <TabsContent value="config" className="pt-4">
          <AgentConfiguration agent={agent} />
        </TabsContent>
        <TabsContent value="sessions" className="pt-4">
          <AgentSessions agentId={agent.id} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
