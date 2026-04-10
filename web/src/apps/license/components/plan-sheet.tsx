import { useState, useMemo } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface ConstraintFeature {
  key: string
  label: string
  type: string
  min?: number
  max?: number
  default?: unknown
  options?: string[]
}

interface ConstraintModule {
  key: string
  label: string
  features: ConstraintFeature[]
}

interface PlanItem {
  id: number
  productId: number
  name: string
  constraintValues: Record<string, Record<string, unknown>>
  isDefault: boolean
  sortOrder: number
}

interface PlanSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  productId: number
  plan: PlanItem | null
  constraintSchema: ConstraintModule[] | null
}

type ModuleValues = { enabled: boolean; [featureKey: string]: unknown }
type PlanValues = Record<string, ModuleValues>

function buildDefaults(schema: ConstraintModule[]): PlanValues {
  const defaults: PlanValues = {}
  for (const mod of schema) {
    const modValues: ModuleValues = { enabled: true }
    for (const feature of mod.features) {
      if (feature.default !== undefined) {
        modValues[feature.key] = feature.default
      } else if (feature.type === "number") {
        modValues[feature.key] = feature.min ?? 0
      } else if (feature.type === "multiSelect") {
        modValues[feature.key] = []
      }
    }
    defaults[mod.key] = modValues
  }
  return defaults
}

export function PlanSheet({ open, onOpenChange, productId, plan, constraintSchema }: PlanSheetProps) {
  const queryClient = useQueryClient()
  const isEditing = plan !== null
  const modules = useMemo(
    () => (constraintSchema && Array.isArray(constraintSchema) ? constraintSchema : []),
    [constraintSchema],
  )

  const [name, setName] = useState("")
  const [values, setValues] = useState<PlanValues>({})

  // Reset form when sheet opens
  const [lastOpen, setLastOpen] = useState(false)
  if (open !== lastOpen) {
    setLastOpen(open)
    if (open) {
      setName(plan?.name ?? "")
      setValues(
        plan
          ? { ...(plan.constraintValues as PlanValues) }
          : buildDefaults(modules),
      )
    }
  }

  function setModuleEnabled(moduleKey: string, enabled: boolean) {
    const mod = modules.find((m) => m.key === moduleKey)
    const current = values[moduleKey] ?? { enabled: false }
    if (enabled && mod) {
      const featureDefaults: Record<string, unknown> = {}
      for (const f of mod.features) {
        if (current[f.key] !== undefined) {
          featureDefaults[f.key] = current[f.key]
        } else if (f.default !== undefined) {
          featureDefaults[f.key] = f.default
        } else if (f.type === "number") {
          featureDefaults[f.key] = f.min ?? 0
        } else if (f.type === "multiSelect") {
          featureDefaults[f.key] = []
        }
      }
      setValues({ ...values, [moduleKey]: { ...current, ...featureDefaults, enabled: true } })
    } else {
      setValues({ ...values, [moduleKey]: { ...current, enabled: false } })
    }
  }

  function setFeatureValue(moduleKey: string, featureKey: string, value: unknown) {
    const current = values[moduleKey] ?? { enabled: true }
    setValues({ ...values, [moduleKey]: { ...current, [featureKey]: value } })
  }

  const createMutation = useMutation({
    mutationFn: (payload: { name: string; constraintValues: PlanValues }) =>
      api.post(`/api/v1/license/products/${productId}/plans`, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-product"] })
      onOpenChange(false)
      toast.success("套餐创建成功")
    },
    onError: (err) => toast.error(err.message),
  })

  const updateMutation = useMutation({
    mutationFn: (payload: { name: string; constraintValues: PlanValues }) =>
      api.put(`/api/v1/license/plans/${plan!.id}`, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-product"] })
      onOpenChange(false)
      toast.success("套餐更新成功")
    },
    onError: (err) => toast.error(err.message),
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const payload = { name, constraintValues: values }
    if (isEditing) {
      updateMutation.mutate(payload)
    } else {
      createMutation.mutate(payload)
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{isEditing ? "编辑套餐" : "新建套餐"}</SheetTitle>
          <SheetDescription className="sr-only">
            {isEditing ? "修改套餐信息" : "为商品创建新套餐"}
          </SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="flex flex-1 flex-col gap-4 px-4">
          <div className="space-y-1.5 rounded-lg bg-muted/20 px-3 py-3">
            <Label htmlFor="plan-name" className="text-xs text-muted-foreground">套餐名称</Label>
            <Input
              id="plan-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：基础版"
              className="h-8 rounded-md text-sm"
              required
            />
          </div>

          {modules.length > 0 && (
            <div className="space-y-2.5">
              <div className="flex items-center justify-between">
                <Label className="text-xs text-muted-foreground">授权配置</Label>
                <Badge variant="outline" className="rounded-md border-0 bg-muted/60 text-[11px] text-muted-foreground">
                  {modules.length} 个模块
                </Badge>
              </div>
              {modules.map((mod) => {
                const modValues = values[mod.key] ?? { enabled: false }
                const isEnabled = !!modValues.enabled

                return (
                  <div
                    key={mod.key}
                    className={`rounded-lg border transition-colors ${
                      isEnabled ? "bg-muted/15" : "bg-muted/8 opacity-85"
                    }`}
                  >
                    <div className="flex items-center justify-between border-b bg-muted/20 px-3 py-2.5">
                      <div className="min-w-0">
                        <p className="text-sm font-medium">{mod.label || mod.key}</p>
                        {mod.features.length > 0 && (
                          <p className="mt-1 text-xs text-muted-foreground">
                            {mod.features.length} 项授权配置
                          </p>
                        )}
                      </div>
                      <Switch
                        size="sm"
                        checked={isEnabled}
                        onCheckedChange={(checked) => setModuleEnabled(mod.key, !!checked)}
                      />
                    </div>

                    {isEnabled && mod.features.length > 0 && (
                      <div className="space-y-2.5 px-3 py-3">
                        {mod.features.map((feature) => (
                          <FeatureValueField
                            key={feature.key}
                            feature={feature}
                            value={modValues[feature.key]}
                            onChange={(v) => setFeatureValue(mod.key, feature.key, v)}
                          />
                        ))}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          <SheetFooter>
            <Button type="submit" size="sm" className="h-8 rounded-lg px-3" disabled={isPending}>
              {isPending ? "保存中..." : "保存"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

function FeatureValueField({
  feature,
  value,
  onChange,
}: {
  feature: ConstraintFeature
  value: unknown
  onChange: (v: unknown) => void
}) {
  if (feature.type === "number") {
    return (
      <div className="rounded-md bg-background/60 px-3 py-2.5">
        <div className="flex items-center justify-between gap-2">
          <Label className="text-xs font-medium">{feature.label || feature.key}</Label>
          {(feature.min != null || feature.max != null) && (
            <Badge variant="outline" className="rounded-md border-0 bg-muted/60 text-[11px] text-muted-foreground">
              {feature.min ?? "无"} ~ {feature.max ?? "无"}
            </Badge>
          )}
        </div>
        <Input
          type="number"
          value={value != null ? String(value) : ""}
          min={feature.min}
          max={feature.max}
          onChange={(e) => onChange(e.target.value ? Number(e.target.value) : undefined)}
          className="mt-1.5 h-8 rounded-md bg-background/80 text-sm"
        />
      </div>
    )
  }

  if (feature.type === "enum") {
    return (
      <div className="rounded-md bg-background/60 px-3 py-2.5">
        <Label className="text-xs font-medium">{feature.label || feature.key}</Label>
        <Select value={value != null ? String(value) : ""} onValueChange={onChange}>
          <SelectTrigger size="sm" className="mt-1.5 w-full rounded-md bg-background/80 text-sm">
            <SelectValue placeholder="请选择" />
          </SelectTrigger>
          <SelectContent>
            {(feature.options ?? []).map((opt) => (
              <SelectItem key={opt} value={opt}>
                {opt}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    )
  }

  if (feature.type === "multiSelect") {
    const selected: string[] = Array.isArray(value) ? (value as string[]) : []
    return (
      <div className="rounded-md bg-background/60 px-3 py-2.5">
        <Label className="text-xs font-medium">{feature.label || feature.key}</Label>
        <div className="mt-1.5 flex flex-wrap gap-1.5">
          {(feature.options ?? []).map((opt) => {
            const isSelected = selected.includes(opt)
            return (
              <label
                key={opt}
                className={`inline-flex cursor-pointer items-center gap-1 rounded-md border px-2 py-1 text-xs transition-colors ${
                  isSelected
                    ? "border-primary/30 bg-primary/10 text-foreground"
                    : "border-border bg-transparent hover:bg-accent/60"
                }`}
              >
                <input
                  type="checkbox"
                  checked={isSelected}
                  onChange={(e) => {
                    if (e.target.checked) {
                      onChange([...selected, opt])
                    } else {
                      onChange(selected.filter((s) => s !== opt))
                    }
                  }}
                  className="sr-only"
                />
                {opt}
              </label>
            )
          })}
        </div>
      </div>
    )
  }

  return null
}
