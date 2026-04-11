import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Wrench, BookOpen, Globe, Code } from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Label } from "@/components/ui/label"
import {
  Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription,
} from "@/components/ui/sheet"

interface ToolItem {
  id: number
  toolkit: string
  name: string
  displayName: string
  description: string
  parametersSchema: Record<string, unknown>
  isActive: boolean
}

interface ToolkitGroup {
  toolkit: string
  tools: ToolItem[]
}

const TOOLKIT_ICONS: Record<string, React.ElementType> = {
  knowledge: BookOpen,
  network: Globe,
  code: Code,
}

export function BuiltinToolsTab() {
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()
  const canUpdate = usePermission("ai:tool:update")
  const [openToolkit, setOpenToolkit] = useState<ToolkitGroup | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ["ai-tools"],
    queryFn: () => api.get<{ items: ToolkitGroup[] }>("/api/v1/ai/tools"),
  })
  const groups = data?.items ?? []

  const toggleMutation = useMutation({
    mutationFn: ({ id, isActive }: { id: number; isActive: boolean }) =>
      api.put(`/api/v1/ai/tools/${id}`, { isActive }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ["ai-tools"] })
      toast.success(
        variables.isActive
          ? t("ai:tools.builtin.enableSuccess")
          : t("ai:tools.builtin.disableSuccess"),
      )
    },
    onError: (err) => toast.error(err.message),
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
        {t("common:loading")}
      </div>
    )
  }

  if (groups.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 py-12 text-center">
        <Wrench className="h-10 w-10 text-muted-foreground/50" />
        <p className="text-sm text-muted-foreground">{t("ai:tools.builtin.empty")}</p>
      </div>
    )
  }

  // Keep the drawer content in sync with latest query data
  const activeDrawerGroup = openToolkit
    ? groups.find((g) => g.toolkit === openToolkit.toolkit) ?? openToolkit
    : null

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {groups.map((group) => {
          const Icon = TOOLKIT_ICONS[group.toolkit] ?? Wrench
          const activeCount = group.tools.filter((t) => t.isActive).length
          const totalCount = group.tools.length

          return (
            <button
              key={group.toolkit}
              type="button"
              className="flex flex-col gap-3 rounded-lg border bg-card p-4 text-left transition-colors hover:border-primary/50 hover:bg-accent/30 cursor-pointer"
              onClick={() => setOpenToolkit(group)}
            >
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary/10">
                  <Icon className="h-5 w-5 text-primary" />
                </div>
                <div className="flex-1 min-w-0">
                  <h3 className="text-sm font-semibold">
                    {t(`ai:tools.toolkits.${group.toolkit}.name`)}
                  </h3>
                  <p className="text-xs text-muted-foreground">
                    {activeCount}/{totalCount} {t("ai:statusLabels.active").toLowerCase()}
                  </p>
                </div>
                <Badge variant={activeCount > 0 ? "default" : "secondary"} className="shrink-0">
                  {activeCount > 0 ? t("ai:statusLabels.active") : t("ai:statusLabels.inactive")}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground line-clamp-2">
                {t(`ai:tools.toolkits.${group.toolkit}.description`)}
              </p>
            </button>
          )
        })}
      </div>

      {/* Toolkit detail drawer */}
      <Sheet open={activeDrawerGroup !== null} onOpenChange={(v) => { if (!v) setOpenToolkit(null) }}>
        <SheetContent className="sm:max-w-lg overflow-y-auto">
          {activeDrawerGroup && (
            <ToolkitDetail
              group={activeDrawerGroup}
              canUpdate={canUpdate}
              toggleMutation={toggleMutation}
              t={t}
            />
          )}
        </SheetContent>
      </Sheet>
    </>
  )
}

function ToolkitDetail({
  group,
  canUpdate,
  toggleMutation,
  t,
}: {
  group: ToolkitGroup
  canUpdate: boolean
  toggleMutation: ReturnType<typeof useMutation<unknown, Error, { id: number; isActive: boolean }>>
  t: (key: string, defaultValue?: string) => string
}) {
  const Icon = TOOLKIT_ICONS[group.toolkit] ?? Wrench

  return (
    <>
      <SheetHeader>
        <SheetTitle className="flex items-center gap-2">
          <Icon className="h-5 w-5 text-primary" />
          {t(`ai:tools.toolkits.${group.toolkit}.name`)}
        </SheetTitle>
        <SheetDescription>
          {t(`ai:tools.toolkits.${group.toolkit}.description`)}
        </SheetDescription>
      </SheetHeader>
      <div className="flex flex-col gap-4 px-4">
        {group.tools.map((tool) => (
          <div key={tool.id} className="rounded-lg border bg-card p-4 space-y-3">
            <div className="flex items-start justify-between gap-3">
              <div className="flex-1 min-w-0">
                <h4 className="text-sm font-medium">{tool.displayName}</h4>
                <p className="text-xs text-muted-foreground font-mono">{tool.name}</p>
              </div>
              <Switch
                checked={tool.isActive}
                disabled={!canUpdate || toggleMutation.isPending}
                onCheckedChange={(checked) =>
                  toggleMutation.mutate({ id: tool.id, isActive: checked })
                }
              />
            </div>
            <p className="text-sm text-muted-foreground">{tool.description}</p>
            {tool.parametersSchema && Object.keys(tool.parametersSchema).length > 0 && (
              <>
                <Separator />
                <div className="space-y-1.5">
                  <Label className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                    Parameters
                  </Label>
                  <div className="rounded-md border bg-muted/30 p-3 text-xs font-mono whitespace-pre-wrap max-h-48 overflow-y-auto">
                    {JSON.stringify(tool.parametersSchema, null, 2)}
                  </div>
                </div>
              </>
            )}
          </div>
        ))}
      </div>
    </>
  )
}
