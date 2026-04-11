import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Plus, Search, Server, Pencil, Trash2, Zap, RefreshCw,
  ChevronRight, ChevronDown, Star, Cpu,
} from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { api, type PaginatedResponse } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  DataTableActions,
  DataTableActionsCell,
  DataTableActionsHead,
  DataTableCard,
  DataTableEmptyRow,
  DataTableLoadingRow,
  DataTablePagination,
  DataTableToolbar,
  DataTableToolbarGroup,
} from "@/components/ui/data-table"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { formatDateTime } from "@/lib/utils"
import { ProviderSheet, type ProviderItem } from "../../components/provider-sheet"
import { ModelSheet, type ModelItem } from "../../components/model-sheet"

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  active: "default",
  inactive: "secondary",
  error: "destructive",
  deprecated: "outline",
}

// ─── Type-grouped model list ─────────────────────────────────────────────────

const TYPE_ORDER = ["llm", "embed", "rerank", "tts", "stt", "image", ""] as const

function groupByType(models: ModelItem[]) {
  const groups: Record<string, ModelItem[]> = {}
  for (const m of models) {
    const key = m.type || ""
    const arr = groups[key] || (groups[key] = [])
    arr.push(m)
  }
  return TYPE_ORDER.filter((t) => groups[t]).map((t) => ({ type: t, items: groups[t] }))
}

interface ModelGroupedListProps {
  models: ModelItem[]
  t: (key: string, defaultValue?: string) => string
  canSetDefault: boolean
  canUpdate: boolean
  canDelete: boolean
  setDefaultMutation: ReturnType<typeof useMutation<unknown, Error, number>>
  deleteMutation: ReturnType<typeof useMutation<unknown, Error, number>>
  onEdit: (m: ModelItem) => void
}

function ModelGroupedList({
  models, t, canSetDefault, canUpdate, canDelete,
  setDefaultMutation, deleteMutation, onEdit,
}: ModelGroupedListProps) {
  const groups = groupByType(models)

  return (
    <div className="divide-y">
      {groups.map(({ type, items }) => (
        <div key={type}>
          <div className="px-4 py-2 bg-muted/40">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              {type ? t(`ai:modelTypes.${type}`) : t("ai:modelTypes.unclassified")}
              <span className="ml-1.5 text-muted-foreground/60 font-normal normal-case">({items.length})</span>
            </span>
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[160px]">{t("ai:models.displayName")}</TableHead>
                <TableHead className="w-[140px]">{t("ai:models.modelId")}</TableHead>
                <TableHead className="w-[70px]">{t("ai:models.status")}</TableHead>
                <TableHead className="w-[50px]">{t("ai:models.isDefault")}</TableHead>
                <DataTableActionsHead className="min-w-[140px]">{t("common:actions")}</DataTableActionsHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((m) => (
                <TableRow key={m.id}>
                  <TableCell className="font-medium">{m.displayName}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{m.modelId}</TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANTS[m.status] ?? "secondary"}>
                      {t(`ai:statusLabels.${m.status}`, m.status)}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {m.isDefault && <Star className="h-4 w-4 text-yellow-500 fill-yellow-500" />}
                  </TableCell>
                  <DataTableActionsCell>
                    <DataTableActions>
                      {canSetDefault && !m.isDefault && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="px-2"
                          disabled={setDefaultMutation.isPending}
                          onClick={() => setDefaultMutation.mutate(m.id)}
                        >
                          <Star className="mr-1 h-3.5 w-3.5" />
                          {t("ai:models.setDefault")}
                        </Button>
                      )}
                      {canUpdate && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="px-2"
                          onClick={() => onEdit(m)}
                        >
                          <Pencil className="mr-1 h-3.5 w-3.5" />
                          {t("common:edit")}
                        </Button>
                      )}
                      {canDelete && (
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="px-2 text-destructive hover:text-destructive"
                            >
                              <Trash2 className="mr-1 h-3.5 w-3.5" />
                              {t("common:delete")}
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>{t("ai:models.deleteTitle")}</AlertDialogTitle>
                              <AlertDialogDescription>
                                {t("ai:models.deleteDesc", { name: m.displayName })}
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => deleteMutation.mutate(m.id)}
                                disabled={deleteMutation.isPending}
                              >
                                {t("ai:models.confirmDelete")}
                              </AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      )}
                    </DataTableActions>
                  </DataTableActionsCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ))}
    </div>
  )
}

// ─── Inline model list per provider ──────────────────────────────────────────

function ProviderModels({ provider }: { provider: ProviderItem }) {
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()
  const [modelFormOpen, setModelFormOpen] = useState(false)
  const [editingModel, setEditingModel] = useState<ModelItem | null>(null)

  const canCreateModel = usePermission("ai:model:create")
  const canUpdateModel = usePermission("ai:model:update")
  const canDeleteModel = usePermission("ai:model:delete")
  const canSetDefault = usePermission("ai:model:default")
  const canSync = usePermission("ai:model:sync")

  const { data, isLoading } = useQuery({
    queryKey: ["ai-models", { providerId: provider.id }],
    queryFn: () =>
      api.get<PaginatedResponse<ModelItem>>(
        `/api/v1/ai/models?providerId=${provider.id}&pageSize=100`,
      ),
  })
  const models = data?.items ?? []

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/v1/ai/models/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-models"] })
      queryClient.invalidateQueries({ queryKey: ["ai-providers"] })
      toast.success(t("ai:models.deleteSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const setDefaultMutation = useMutation({
    mutationFn: (id: number) => api.patch(`/api/v1/ai/models/${id}/default`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-models"] })
      toast.success(t("ai:models.setDefaultSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const syncMutation = useMutation({
    mutationFn: () => api.post<{ added: number }>(`/api/v1/ai/providers/${provider.id}/sync-models`),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["ai-models"] })
      queryClient.invalidateQueries({ queryKey: ["ai-providers"] })
      toast.success(t("ai:providers.syncSuccess", { count: data.added }))
    },
    onError: (err) => toast.error(err.message),
  })

  function handleCreateModel() {
    setEditingModel(null)
    setModelFormOpen(true)
  }

  function handleEditModel(item: ModelItem) {
    setEditingModel(item)
    setModelFormOpen(true)
  }

  return (
    <div className="px-4 pb-4">
      <div className="rounded-md border bg-muted/30">
        <div className="flex items-center justify-between px-4 py-2 border-b">
          <span className="text-sm font-medium text-muted-foreground">
            {t("ai:models.title")}
          </span>
          <div className="flex items-center gap-1">
            {canSync && (
              <Button
                variant="ghost"
                size="sm"
                disabled={syncMutation.isPending}
                onClick={() => syncMutation.mutate()}
              >
                <RefreshCw className="mr-1 h-3.5 w-3.5" />
                {syncMutation.isPending ? t("ai:providers.syncing") : t("ai:providers.syncModels")}
              </Button>
            )}
            {canCreateModel && (
              <Button variant="ghost" size="sm" onClick={handleCreateModel}>
                <Plus className="mr-1 h-3.5 w-3.5" />
                {t("ai:models.create")}
              </Button>
            )}
          </div>
        </div>

        {isLoading ? (
          <div className="px-4 py-6 text-center text-sm text-muted-foreground">
            {t("common:loading")}
          </div>
        ) : models.length === 0 ? (
          <div className="flex flex-col items-center gap-1 px-4 py-6 text-center">
            <Cpu className="h-8 w-8 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">{t("ai:models.empty")}</p>
          </div>
        ) : (
          <ModelGroupedList
            models={models}
            t={t}
            canSetDefault={canSetDefault}
            canUpdate={canUpdateModel}
            canDelete={canDeleteModel}
            setDefaultMutation={setDefaultMutation}
            deleteMutation={deleteMutation}
            onEdit={handleEditModel}
          />
        )}
      </div>

      <ModelSheet
        open={modelFormOpen}
        onOpenChange={setModelFormOpen}
        model={editingModel}
        defaultProviderId={provider.id}
      />
    </div>
  )
}

// ─── Main page ───────────────────────────────────────────────────────────────

export function Component() {
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<ProviderItem | null>(null)
  const [expandedIds, setExpandedIds] = useState<Set<number>>(new Set())

  const canCreate = usePermission("ai:provider:create")
  const canUpdate = usePermission("ai:provider:update")
  const canDelete = usePermission("ai:provider:delete")
  const canTest = usePermission("ai:provider:test")

  const {
    keyword, setKeyword, page, setPage,
    items: providers, total, totalPages, isLoading, handleSearch,
  } = useListPage<ProviderItem>({
    queryKey: "ai-providers",
    endpoint: "/api/v1/ai/providers",
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/v1/ai/providers/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-providers"] })
      toast.success(t("ai:providers.deleteSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const testMutation = useMutation({
    mutationFn: (id: number) =>
      api.post<{ success: boolean; error?: string }>(`/api/v1/ai/providers/${id}/test`),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["ai-providers"] })
      if (data.success) {
        toast.success(t("ai:providers.testSuccess"))
      } else {
        toast.error(t("ai:providers.testFailed", { error: data.error }))
      }
    },
    onError: (err) => toast.error(err.message),
  })

  function toggleExpand(id: number) {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  function handleCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function handleEdit(item: ProviderItem) {
    setEditing(item)
    setFormOpen(true)
  }

  const colSpan = 7

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t("ai:providers.title")}</h2>
        {canCreate && (
          <Button size="sm" onClick={handleCreate}>
            <Plus className="mr-1.5 h-4 w-4" />
            {t("ai:providers.create")}
          </Button>
        )}
      </div>

      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={t("ai:providers.searchPlaceholder")}
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

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[40px]" />
              <TableHead className="min-w-[160px]">{t("ai:providers.name")}</TableHead>
              <TableHead className="w-[100px]">{t("ai:providers.type")}</TableHead>
              <TableHead className="w-[90px]">{t("ai:providers.status")}</TableHead>
              <TableHead className="w-[80px]">{t("ai:providers.modelCount")}</TableHead>
              <TableHead className="w-[150px]">{t("common:createdAt")}</TableHead>
              <DataTableActionsHead className="min-w-[160px]">{t("common:actions")}</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={colSpan} />
            ) : providers.length === 0 ? (
              <DataTableEmptyRow
                colSpan={colSpan}
                icon={Server}
                title={t("ai:providers.empty")}
                description={canCreate ? t("ai:providers.emptyHint") : undefined}
              />
            ) : (
              providers.map((item) => {
                const expanded = expandedIds.has(item.id)
                return (
                  <TableRow key={item.id} className="group">
                    <TableCell className="w-[40px] pr-0">
                      <button
                        type="button"
                        className="p-1 rounded hover:bg-accent"
                        onClick={() => toggleExpand(item.id)}
                      >
                        {expanded
                          ? <ChevronDown className="h-4 w-4 text-muted-foreground" />
                          : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
                      </button>
                    </TableCell>
                    <TableCell className="font-medium">{item.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{t(`ai:types.${item.type}`, item.type)}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={STATUS_VARIANTS[item.status] ?? "secondary"}>
                        {t(`ai:statusLabels.${item.status}`, item.status)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm">{item.modelCount}</TableCell>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {formatDateTime(item.createdAt)}
                    </TableCell>
                    <DataTableActionsCell>
                      <DataTableActions>
                        {canTest && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="px-2.5"
                            disabled={testMutation.isPending}
                            onClick={() => testMutation.mutate(item.id)}
                          >
                            <Zap className="mr-1 h-3.5 w-3.5" />
                            {testMutation.isPending ? t("ai:providers.testing") : t("ai:providers.testConnection")}
                          </Button>
                        )}
                        {canUpdate && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="px-2.5"
                            onClick={() => handleEdit(item)}
                          >
                            <Pencil className="mr-1 h-3.5 w-3.5" />
                            {t("common:edit")}
                          </Button>
                        )}
                        {canDelete && (
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="px-2.5 text-destructive hover:text-destructive"
                              >
                                <Trash2 className="mr-1 h-3.5 w-3.5" />
                                {t("common:delete")}
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>{t("ai:providers.deleteTitle")}</AlertDialogTitle>
                                <AlertDialogDescription>
                                  {t("ai:providers.deleteDesc", { name: item.name })}
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={() => deleteMutation.mutate(item.id)}
                                  disabled={deleteMutation.isPending}
                                >
                                  {t("ai:providers.confirmDelete")}
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        )}
                      </DataTableActions>
                    </DataTableActionsCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>

        {/* Inline model panels — rendered outside <Table> to avoid nesting issues */}
        {providers.map((item) =>
          expandedIds.has(item.id) ? (
            <ProviderModels key={`models-${item.id}`} provider={item} />
          ) : null,
        )}
      </DataTableCard>

      <DataTablePagination
        total={total}
        page={page}
        totalPages={totalPages}
        onPageChange={setPage}
      />

      <ProviderSheet open={formOpen} onOpenChange={setFormOpen} provider={editing} />
    </div>
  )
}
