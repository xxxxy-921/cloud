import { useState, useMemo } from "react"
import { useParams, Link } from "react-router"
import { useTranslation } from "react-i18next"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Pencil, Zap, RefreshCw, Plus, Search,
  Star, Trash2, Cpu,
} from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { api, type PaginatedResponse } from "@/lib/api"
import { toast } from "sonner"
import { formatDateTime } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
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
import {
  DataTableActions,
  DataTableActionsCell,
  DataTableActionsHead,
} from "@/components/ui/data-table"
import { getProviderBrand } from "../../lib/provider-brand"
import { StatusDot } from "../../components/status-dot"
import { ProviderSheet, type ProviderItem } from "../../components/provider-sheet"
import { ModelSheet, type ModelItem } from "../../components/model-sheet"

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  active: "default",
  inactive: "secondary",
  error: "destructive",
  deprecated: "outline",
}

const TYPE_ORDER = ["llm", "embed", "rerank", "tts", "stt", "image", ""] as const

function getModelTypeSummary(provider: ProviderItem) {
  return TYPE_ORDER
    .filter((type) => type && (provider.modelTypeCounts?.[type] ?? 0) > 0)
    .map((type) => ({ type, count: provider.modelTypeCounts[type] }))
}

function groupByType(models: ModelItem[]) {
  const groups: Record<string, ModelItem[]> = {}
  for (const m of models) {
    const key = m.type || ""
    const arr = groups[key] || (groups[key] = [])
    arr.push(m)
  }
  return TYPE_ORDER.filter((t) => groups[t]).map((t) => ({ type: t, items: groups[t] }))
}

function getEmptyTypeGroups() {
  return TYPE_ORDER
    .filter((type) => type)
    .map((type) => ({ type, items: [] as ModelItem[] }))
}

// ─── Provider Info Section ──────────────────────────────────────────────────

function ProviderInfoSection({
  provider,
  canUpdate,
  canTest,
  onEdit,
  onTest,
  onSync,
  isTesting,
  isSyncing,
}: {
  provider: ProviderItem
  canUpdate: boolean
  canTest: boolean
  onEdit: () => void
  onTest: () => void
  onSync: () => void
  isTesting: boolean
  isSyncing: boolean
}) {
  const { t } = useTranslation(["ai", "common"])
  const brand = getProviderBrand(provider.type)
  const typeSummary = getModelTypeSummary(provider)

  return (
    <section className="space-y-4 border-b pb-5">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div className="flex min-w-0 items-start gap-4">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border bg-muted/35 text-sm font-bold text-foreground/80">
            {brand.avatarText}
          </div>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-lg font-semibold leading-tight">{provider.name}</h2>
              <Badge variant="outline" className="rounded-full px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
                {t(`ai:types.${provider.type}`, provider.type)}
              </Badge>
            </div>
            <p className="mt-1.5 text-sm leading-6 text-muted-foreground">{provider.baseUrl}</p>
            <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1.5 text-sm text-muted-foreground">
              <div className="flex items-center gap-1.5">
                <StatusDot status={provider.status} loading={isTesting} />
                <span>{t(`ai:statusLabels.${provider.status}`, provider.status)}</span>
              </div>
              <span>{t("ai:providers.protocol")}: {provider.protocol}</span>
              <span>{t("ai:providers.healthCheckedAt")}: {provider.healthCheckedAt ? formatDateTime(provider.healthCheckedAt) : "—"}</span>
              <span>{t("ai:providers.modelCount")}: {provider.modelCount}</span>
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2 lg:justify-end">
          {canTest && (
            <Button variant="outline" size="sm" disabled={isTesting} onClick={onTest}>
              <Zap className="mr-1.5 h-3.5 w-3.5" />
              {isTesting ? t("ai:providers.testing") : t("ai:providers.testConnection")}
            </Button>
          )}
          <Button variant="outline" size="sm" disabled={isSyncing} onClick={onSync}>
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
            {isSyncing ? t("ai:providers.syncing") : t("ai:providers.syncModels")}
          </Button>
          {canUpdate && (
            <Button variant="outline" size="sm" onClick={onEdit}>
              <Pencil className="mr-1.5 h-3.5 w-3.5" />
              {t("common:edit")}
            </Button>
          )}
        </div>
      </div>

      <div className="flex flex-wrap gap-2 pl-[3.75rem]">
        {typeSummary.length > 0 ? typeSummary.map(({ type, count }) => (
          <Badge key={type} variant="outline" className="h-6 rounded-full px-2 text-[11px] font-normal text-muted-foreground">
            <span>{t(`ai:modelTypes.${type}`, type)}</span>
            <span className="ml-1 rounded-full bg-background px-1.5 py-0.5 font-medium tabular-nums text-foreground">
              {count}
            </span>
          </Badge>
        )) : (
          <p className="pl-[3.75rem] text-sm text-muted-foreground">{t("ai:models.empty")}</p>
        )}
      </div>
    </section>
  )
}

// ─── Model Management Section ───────────────────────────────────────────────

function ModelManagementSection({ provider }: { provider: ProviderItem }) {
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()
  const [modelFormOpen, setModelFormOpen] = useState(false)
  const [editingModel, setEditingModel] = useState<ModelItem | null>(null)
  const [searchKeyword, setSearchKeyword] = useState("")

  const canCreateModel = usePermission("ai:model:create")
  const canUpdateModel = usePermission("ai:model:update")
  const canDeleteModel = usePermission("ai:model:delete")
  const canSetDefault = usePermission("ai:model:default")

  const { data, isLoading } = useQuery({
    queryKey: ["ai-models", { providerId: provider.id }],
    queryFn: () =>
      api.get<PaginatedResponse<ModelItem>>(
        `/api/v1/ai/models?providerId=${provider.id}&pageSize=100`,
      ),
  })
  const allModels = data?.items ?? []

  const filteredModels = useMemo(() => {
    if (!searchKeyword) return allModels
    const kw = searchKeyword.toLowerCase()
    return allModels.filter(
      (m) =>
        m.displayName.toLowerCase().includes(kw) ||
        m.modelId.toLowerCase().includes(kw),
    )
  }, [allModels, searchKeyword])

  const groups = filteredModels.length > 0 ? groupByType(filteredModels) : getEmptyTypeGroups()

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

  return (
    <div className="rounded-xl border bg-card">
      <div className="flex items-center justify-between border-b px-5 py-3">
        <h3 className="text-sm font-semibold">{t("ai:models.title")}</h3>
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="absolute left-2.5 top-2 h-3.5 w-3.5 text-muted-foreground" />
            <Input
              placeholder={t("ai:models.searchPlaceholder")}
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              className="h-8 w-48 pl-8 text-xs"
            />
          </div>
          {canCreateModel && (
            <Button
              variant="outline"
              size="sm"
              className="h-8"
              onClick={() => { setEditingModel(null); setModelFormOpen(true) }}
            >
              <Plus className="mr-1 h-3.5 w-3.5" />
              {t("ai:models.create")}
            </Button>
          )}
        </div>
      </div>

      {isLoading ? (
        <div className="px-5 py-8 text-center text-sm text-muted-foreground">
          {t("common:loading")}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 p-4 md:grid-cols-2 xl:grid-cols-3">
          {groups.map(({ type, items }) => (
            <section key={type} className="overflow-hidden rounded-xl border bg-background/60">
              <div className="flex items-center justify-between border-b bg-muted/25 px-4 py-3">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">
                    {type ? t(`ai:modelTypes.${type}`) : t("ai:modelTypes.unclassified")}
                  </span>
                  <Badge variant="outline" className="h-5 rounded-full px-1.5 text-[10px] font-medium text-muted-foreground">
                    {items.length}
                  </Badge>
                </div>
                {type && provider.modelTypeCounts?.[type] ? (
                  <span className="text-[11px] text-muted-foreground">
                    {provider.modelTypeCounts[type]} {t("ai:providers.modelCount")}
                  </span>
                ) : null}
              </div>
              {items.length === 0 ? (
                <div className="flex min-h-[180px] flex-col items-center justify-center gap-2 px-4 py-8 text-center">
                  <Cpu className="h-8 w-8 text-muted-foreground/35" />
                  <p className="text-sm text-muted-foreground">{t("ai:models.empty")}</p>
                </div>
              ) : (
                <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-[160px]">{t("ai:models.displayName")}</TableHead>
                    <TableHead className="w-[120px]">{t("ai:models.modelId")}</TableHead>
                    <TableHead className="w-[70px]">{t("ai:models.status")}</TableHead>
                    <TableHead className="w-[48px]">{t("ai:models.isDefault")}</TableHead>
                    <DataTableActionsHead className="min-w-[132px]">{t("common:actions")}</DataTableActionsHead>
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
                              className="px-2 text-xs"
                              disabled={setDefaultMutation.isPending}
                              onClick={() => setDefaultMutation.mutate(m.id)}
                            >
                              <Star className="mr-1 h-3.5 w-3.5" />
                              {t("ai:models.setDefault")}
                            </Button>
                          )}
                          {canUpdateModel && (
                            <Button
                              variant="ghost"
                              size="sm"
                              className="px-2 text-xs"
                              onClick={() => { setEditingModel(m); setModelFormOpen(true) }}
                            >
                              <Pencil className="mr-1 h-3.5 w-3.5" />
                              {t("common:edit")}
                            </Button>
                          )}
                          {canDeleteModel && (
                            <AlertDialog>
                              <AlertDialogTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="px-2 text-xs text-destructive hover:text-destructive"
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
              )}
            </section>
          ))}
        </div>
      )}

      <ModelSheet
        open={modelFormOpen}
        onOpenChange={setModelFormOpen}
        model={editingModel}
        defaultProviderId={provider.id}
      />
    </div>
  )
}

// ─── Main Detail Page ───────────────────────────────────────────────────────

export function Component() {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()
  const [editOpen, setEditOpen] = useState(false)

  const canUpdate = usePermission("ai:provider:update")
  const canTest = usePermission("ai:provider:test")

  const { data: provider, isLoading, isError } = useQuery({
    queryKey: ["ai-provider", id],
    queryFn: () => api.get<ProviderItem>(`/api/v1/ai/providers/${id}`),
    enabled: !!id,
  })

  const testMutation = useMutation({
    mutationFn: () =>
      api.post<{ success: boolean; error?: string }>(`/api/v1/ai/providers/${id}/test`),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["ai-provider", id] })
      queryClient.invalidateQueries({ queryKey: ["ai-providers"] })
      if (data.success) {
        toast.success(t("ai:providers.testSuccess"))
      } else {
        toast.error(t("ai:providers.testFailed", { error: data.error }))
      }
    },
    onError: (err) => toast.error(err.message),
  })

  const syncMutation = useMutation({
    mutationFn: () =>
      api.post<{ added: number }>(`/api/v1/ai/providers/${id}/sync-models`),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["ai-models"] })
      queryClient.invalidateQueries({ queryKey: ["ai-provider", id] })
      queryClient.invalidateQueries({ queryKey: ["ai-providers"] })
      toast.success(t("ai:providers.syncSuccess", { count: data.added }))
    },
    onError: (err) => toast.error(err.message),
  })

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="h-48 animate-pulse rounded-xl border bg-muted/30" />
        <div className="h-64 animate-pulse rounded-xl border bg-muted/30" />
      </div>
    )
  }

  if (isError || !provider) {
    return (
      <div className="flex flex-col items-center gap-3 py-16 text-center">
        <p className="text-sm text-muted-foreground">{t("ai:providers.empty")}</p>
        <Button variant="outline" size="sm" asChild>
          <Link to="/ai/providers">
            <ArrowLeft className="mr-1.5 h-4 w-4" />
            {t("ai:providers.backToList")}
          </Link>
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <ProviderInfoSection
        provider={provider}
        canUpdate={canUpdate}
        canTest={canTest}
        onEdit={() => setEditOpen(true)}
        onTest={() => testMutation.mutate()}
        onSync={() => syncMutation.mutate()}
        isTesting={testMutation.isPending}
        isSyncing={syncMutation.isPending}
      />

      <ModelManagementSection provider={provider} />

      <ProviderSheet
        open={editOpen}
        onOpenChange={setEditOpen}
        provider={provider}
      />
    </div>
  )
}
