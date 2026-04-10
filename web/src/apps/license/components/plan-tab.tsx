import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { AlertTriangle, Pencil, Plus, Star, Trash2 } from "lucide-react"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { PlanSheet } from "./plan-sheet"

interface PlanItem {
  id: number
  productId: number
  name: string
  constraintValues: Record<string, Record<string, unknown>>
  isDefault: boolean
  sortOrder: number
  createdAt: string
  updatedAt: string
}

interface ConstraintFeature {
  key: string
  label: string
  type: string
  options?: string[]
}

interface ConstraintModule {
  key: string
  label: string
  features: ConstraintFeature[]
}

interface PlanTabProps {
  productId: number
  plans: PlanItem[]
  constraintSchema: ConstraintModule[] | null
  canManage: boolean
  onRequestDefineConstraints?: () => void
}

type ModuleValues = { enabled?: boolean; [featureKey: string]: unknown }

function getCompatibilityWarnings(
  values: Record<string, unknown>,
  schema: ConstraintModule[],
): string[] {
  const warnings: string[] = []
  const schemaKeys = new Set(schema.map((m) => m.key))
  const valueKeys = new Set(Object.keys(values))

  for (const key of schemaKeys) {
    if (!valueKeys.has(key)) {
      const mod = schema.find((m) => m.key === key)
      warnings.push(`缺少模块: ${mod?.label ?? key}`)
    }
  }
  for (const key of valueKeys) {
    if (!schemaKeys.has(key)) {
      warnings.push(`多余模块: ${key}`)
    }
  }
  return warnings
}

function constraintPreview(
  values: Record<string, ModuleValues>,
  schema: ConstraintModule[],
): string {
  const parts: string[] = []
  for (const mod of schema) {
    const modValues = values[mod.key]
    if (!modValues) continue
    if (!modValues.enabled) {
      parts.push(`${mod.label}: 关`)
      continue
    }
    if (mod.features.length === 0) {
      parts.push(`${mod.label}: 开`)
    } else {
      const featureParts = mod.features
        .filter((f) => modValues[f.key] !== undefined)
        .map((f) => {
          const val = modValues[f.key]
          if (Array.isArray(val)) return `${f.label}: ${val.join(",")}`
          return `${f.label}: ${val}`
        })
      parts.push(`${mod.label}(${featureParts.join(", ")})`)
    }
  }
  return parts.join(" | ") || "无配置"
}

export function PlanTab({
  productId,
  plans,
  constraintSchema,
  canManage,
  onRequestDefineConstraints,
}: PlanTabProps) {
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<PlanItem | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<PlanItem | null>(null)
  const modules = constraintSchema && Array.isArray(constraintSchema) ? constraintSchema : []
  const hasSchema = modules.length > 0

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/v1/license/plans/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-product"] })
      toast.success("套餐已删除")
      setDeleteTarget(null)
    },
    onError: (err) => toast.error(err.message),
  })

  const defaultMutation = useMutation({
    mutationFn: (id: number) =>
      api.patch(`/api/v1/license/plans/${id}/default`, { isDefault: true }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-product"] })
      toast.success("已设为默认套餐")
    },
    onError: (err) => toast.error(err.message),
  })

  function handleCreate() {
    if (!hasSchema) return
    setEditing(null)
    setFormOpen(true)
  }

  function handleEdit(plan: PlanItem) {
    setEditing(plan)
    setFormOpen(true)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-end gap-2">
        {canManage && (
          <>
            {!hasSchema && onRequestDefineConstraints && (
              <Button variant="outline" size="sm" onClick={onRequestDefineConstraints}>
                约束定义
              </Button>
            )}
            <Button size="sm" onClick={handleCreate} disabled={!hasSchema}>
              <Plus className="mr-1.5 h-4 w-4" />
              添加套餐
            </Button>
          </>
        )}
      </div>

      {plans.length === 0 ? (
        <div className="rounded-lg border border-dashed px-4 py-8 text-center">
          <p className="font-medium">{hasSchema ? "还没有套餐" : "请先定义约束"}</p>
        </div>
      ) : (
        <div className="space-y-3">
          {plans.map((plan) => {
            const warnings = getCompatibilityWarnings(
              plan.constraintValues as Record<string, unknown>,
              modules,
            )
            return (
              <div key={plan.id} className="flex items-center justify-between gap-4 rounded-lg border px-4 py-4">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{plan.name}</span>
                    {plan.isDefault && (
                      <Badge variant="default" className="gap-1">
                        <Star className="h-3 w-3" />
                        默认
                      </Badge>
                    )}
                    {warnings.length > 0 && (
                      <Badge variant="destructive" className="gap-1" title={warnings.join("\n")}>
                        <AlertTriangle className="h-3 w-3" />
                        需更新
                      </Badge>
                    )}
                  </div>
                  {modules.length > 0 && (
                    <p className="mt-2 text-xs text-muted-foreground">
                      {constraintPreview(
                        plan.constraintValues as Record<string, ModuleValues>,
                        modules,
                      )}
                    </p>
                  )}
                </div>
                {canManage && (
                  <div className="ml-4 flex shrink-0 items-center gap-1">
                    {!plan.isDefault && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => defaultMutation.mutate(plan.id)}
                        disabled={defaultMutation.isPending}
                        className="text-xs"
                      >
                        设为默认
                      </Button>
                    )}
                    <Button variant="ghost" size="icon" onClick={() => handleEdit(plan)}>
                      <Pencil className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setDeleteTarget(plan)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Delete confirmation */}
      <AlertDialog open={!!deleteTarget} onOpenChange={() => setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除套餐</AlertDialogTitle>
            <AlertDialogDescription>
              确定删除套餐「{deleteTarget?.name}」？此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
              disabled={deleteMutation.isPending}
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <PlanSheet
        open={formOpen}
        onOpenChange={setFormOpen}
        productId={productId}
        plan={editing}
        constraintSchema={constraintSchema}
      />
    </div>
  )
}
