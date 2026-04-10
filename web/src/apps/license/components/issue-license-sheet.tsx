import { useState, useMemo } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
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

interface ProductOption {
  id: number
  name: string
  code: string
  status: string
  constraintSchema: ConstraintModule[] | null
  plans: PlanOption[] | null
}

interface PlanOption {
  id: number
  name: string
  constraintValues: Record<string, Record<string, unknown>>
  isDefault: boolean
}

interface LicenseeOption {
  id: number
  name: string
  code: string
}

type ModuleValues = { enabled: boolean; [featureKey: string]: unknown }
type PlanValues = Record<string, ModuleValues>

interface IssueLicenseSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

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

export function IssueLicenseSheet({ open, onOpenChange }: IssueLicenseSheetProps) {
  const queryClient = useQueryClient()

  const [productId, setProductId] = useState("")
  const [licenseeId, setLicenseeId] = useState("")
  const [planId, setPlanId] = useState("")
  const [registrationCode, setRegistrationCode] = useState("")
  const [validFrom, setValidFrom] = useState("")
  const [validUntil, setValidUntil] = useState("")
  const [notes, setNotes] = useState("")
  const [constraintValues, setConstraintValues] = useState<PlanValues>({})

  // Reset form when sheet opens
  const [lastOpen, setLastOpen] = useState(false)
  if (open !== lastOpen) {
    setLastOpen(open)
    if (open) {
      setProductId("")
      setLicenseeId("")
      setPlanId("")
      setRegistrationCode("")
      setValidFrom(new Date().toISOString().split("T")[0])
      setValidUntil("")
      setNotes("")
      setConstraintValues({})
    }
  }

  // Fetch published products with plans
  const { data: products = [] } = useQuery({
    queryKey: ["license-products-published"],
    queryFn: async () => {
      const res = await api.get<{ items: ProductOption[] }>("/api/v1/license/products?status=published&pageSize=100")
      // Fetch detail with plans for each product
      const detailed = await Promise.all(
        res.items.map((p) => api.get<ProductOption>(`/api/v1/license/products/${p.id}`))
      )
      return detailed
    },
    enabled: open,
  })

  // Fetch active licensees
  const { data: licenseesData } = useQuery({
    queryKey: ["license-licensees-active"],
    queryFn: () => api.get<{ items: LicenseeOption[] }>("/api/v1/license/licensees?status=active&pageSize=100"),
    enabled: open,
  })
  const licensees = licenseesData?.items ?? []

  const selectedProduct = useMemo(
    () => products.find((p) => String(p.id) === productId),
    [products, productId],
  )

  const plans = useMemo(
    () => selectedProduct?.plans ?? [],
    [selectedProduct],
  )

  const schema = useMemo(
    () => (selectedProduct?.constraintSchema && Array.isArray(selectedProduct.constraintSchema)
      ? selectedProduct.constraintSchema
      : []),
    [selectedProduct],
  )

  function handleProductChange(value: string) {
    setProductId(value)
    setPlanId("")
    const product = products.find((p) => String(p.id) === value)
    if (product?.constraintSchema) {
      setConstraintValues(buildDefaults(product.constraintSchema))
    } else {
      setConstraintValues({})
    }
  }

  function handlePlanChange(value: string) {
    setPlanId(value)
    if (value === "custom") {
      setConstraintValues(buildDefaults(schema))
      return
    }
    const plan = plans.find((p) => String(p.id) === value)
    if (plan?.constraintValues) {
      setConstraintValues(plan.constraintValues as PlanValues)
    }
  }

  function setModuleEnabled(moduleKey: string, enabled: boolean) {
    const mod = schema.find((m) => m.key === moduleKey)
    const current = constraintValues[moduleKey] ?? { enabled: false }
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
      setConstraintValues({ ...constraintValues, [moduleKey]: { ...current, ...featureDefaults, enabled: true } })
    } else {
      setConstraintValues({ ...constraintValues, [moduleKey]: { ...current, enabled: false } })
    }
  }

  function setFeatureValue(moduleKey: string, featureKey: string, value: unknown) {
    const current = constraintValues[moduleKey] ?? { enabled: true }
    setConstraintValues({ ...constraintValues, [moduleKey]: { ...current, [featureKey]: value } })
  }

  const issueMutation = useMutation({
    mutationFn: (payload: Record<string, unknown>) =>
      api.post("/api/v1/license/licenses", payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-licenses"] })
      onOpenChange(false)
      toast.success("许可签发成功")
    },
    onError: (err) => toast.error(err.message),
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!productId || !licenseeId || !registrationCode || !validFrom) return

    const selectedPlan = plans.find((p) => String(p.id) === planId)
    const planName = planId === "custom" ? "自定义" : (selectedPlan?.name ?? "自定义")

    issueMutation.mutate({
      productId: Number(productId),
      licenseeId: Number(licenseeId),
      planId: planId && planId !== "custom" ? Number(planId) : null,
      planName,
      registrationCode,
      constraintValues,
      validFrom,
      validUntil: validUntil || null,
      notes,
    })
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>签发许可</SheetTitle>
          <SheetDescription className="sr-only">为授权主体签发新许可</SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="flex flex-1 flex-col gap-4 px-4">
          {/* Product */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">商品 *</Label>
            <Select value={productId} onValueChange={handleProductChange}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="选择商品" />
              </SelectTrigger>
              <SelectContent>
                {products.map((p) => (
                  <SelectItem key={p.id} value={String(p.id)}>
                    {p.name} ({p.code})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Licensee */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">授权主体 *</Label>
            <Select value={licenseeId} onValueChange={setLicenseeId}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="选择授权主体" />
              </SelectTrigger>
              <SelectContent>
                {licensees.map((l) => (
                  <SelectItem key={l.id} value={String(l.id)}>
                    {l.name} ({l.code})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Plan */}
          {productId && (
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">套餐</Label>
              <Select value={planId} onValueChange={handlePlanChange}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="选择套餐或自定义" />
                </SelectTrigger>
                <SelectContent>
                  {plans.map((p) => (
                    <SelectItem key={p.id} value={String(p.id)}>
                      {p.name} {p.isDefault && "(默认)"}
                    </SelectItem>
                  ))}
                  <SelectItem value="custom">自定义</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          {/* Constraint Values */}
          {productId && schema.length > 0 && (planId === "custom" || planId) && (
            <div className="space-y-2.5">
              <div className="flex items-center justify-between">
                <Label className="text-xs text-muted-foreground">授权配置</Label>
                <Badge variant="outline" className="rounded-md border-0 bg-muted/60 text-[11px] text-muted-foreground">
                  {schema.length} 个模块
                </Badge>
              </div>
              {schema.map((mod) => {
                const modValues = constraintValues[mod.key] ?? { enabled: false }
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
                      </div>
                      <Switch
                        size="sm"
                        checked={isEnabled}
                        onCheckedChange={(checked) => setModuleEnabled(mod.key, !!checked)}
                        disabled={planId !== "custom"}
                      />
                    </div>
                    {isEnabled && mod.features.length > 0 && (
                      <div className="space-y-2.5 px-3 py-3">
                        {mod.features.map((feature) => (
                          <FeatureField
                            key={feature.key}
                            feature={feature}
                            value={modValues[feature.key]}
                            onChange={(v) => setFeatureValue(mod.key, feature.key, v)}
                            disabled={planId !== "custom"}
                          />
                        ))}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          {/* Registration Code */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">注册码 *</Label>
            <Input
              value={registrationCode}
              onChange={(e) => setRegistrationCode(e.target.value)}
              placeholder="客户端注册码"
              required
            />
          </div>

          {/* Valid From */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">生效日期 *</Label>
              <Input
                type="date"
                value={validFrom}
                onChange={(e) => setValidFrom(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">过期日期</Label>
              <Input
                type="date"
                value={validUntil}
                onChange={(e) => setValidUntil(e.target.value)}
                placeholder="留空为永久"
              />
            </div>
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">备注</Label>
            <Textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="可选备注"
              rows={2}
            />
          </div>

          <SheetFooter>
            <Button
              type="submit"
              size="sm"
              className="h-8 rounded-lg px-3"
              disabled={issueMutation.isPending || !productId || !licenseeId || !registrationCode}
            >
              {issueMutation.isPending ? "签发中..." : "签发"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

function FeatureField({
  feature,
  value,
  onChange,
  disabled,
}: {
  feature: ConstraintFeature
  value: unknown
  onChange: (v: unknown) => void
  disabled?: boolean
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
          disabled={disabled}
        />
      </div>
    )
  }

  if (feature.type === "enum") {
    return (
      <div className="rounded-md bg-background/60 px-3 py-2.5">
        <Label className="text-xs font-medium">{feature.label || feature.key}</Label>
        <Select value={value != null ? String(value) : ""} onValueChange={onChange} disabled={disabled}>
          <SelectTrigger size="sm" className="mt-1.5 w-full rounded-md bg-background/80 text-sm">
            <SelectValue placeholder="请选择" />
          </SelectTrigger>
          <SelectContent>
            {(feature.options ?? []).map((opt) => (
              <SelectItem key={opt} value={opt}>{opt}</SelectItem>
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
                } ${disabled ? "pointer-events-none opacity-60" : ""}`}
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
                  disabled={disabled}
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
