import { useEffect, useMemo, useState } from "react"
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
import { useTranslation } from "react-i18next"

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

interface RegistrationOption {
  id: number
  code: string
  expiresAt: string | null
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
  const { t } = useTranslation(["license", "common"])
  const queryClient = useQueryClient()

  const [productId, setProductId] = useState("")
  const [licenseeId, setLicenseeId] = useState("")
  const [planId, setPlanId] = useState("")
  const [registrationCode, setRegistrationCode] = useState("")
  const [registrationMode, setRegistrationMode] = useState<"select" | "generate" | "manual">("select")
  const [validFrom, setValidFrom] = useState("")
  const [validUntil, setValidUntil] = useState("")
  const [notes, setNotes] = useState("")
  const [constraintValues, setConstraintValues] = useState<PlanValues>({})

  // Reset form when sheet opens/closes
  useEffect(() => {
    if (open) {
      queueMicrotask(() => {
        setProductId("")
        setLicenseeId("")
        setPlanId("")
        setRegistrationCode("")
        setRegistrationMode("select")
        setValidFrom(new Date().toISOString().split("T")[0])
        setValidUntil("")
        setNotes("")
        setConstraintValues({})
      })
    }
  }, [open])

  // Fetch published products with plans
  const { data: products = [] } = useQuery({
    queryKey: ["license-products-published"],
    queryFn: async () => {
      const res = await api.get<{ items: ProductOption[] }>("/api/v1/license/products?status=published&pageSize=100")
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

  // Fetch available registrations for selected product
  const { data: registrationsData } = useQuery({
    queryKey: ["license-registrations-available", productId],
    queryFn: () =>
      api.get<{ items: RegistrationOption[] }>(
        `/api/v1/license/registrations?productId=${productId}&status=unbound&pageSize=100`
      ),
    enabled: open && !!productId && registrationMode === "select",
  })
  const registrations = registrationsData?.items ?? []

  const selectedProduct = useMemo(
    () => products.find((p) => String(p.id) === productId),
    [products, productId]
  )

  const plans = useMemo(
    () => selectedProduct?.plans ?? [],
    [selectedProduct]
  )

  const schema = useMemo(
    () => (selectedProduct?.constraintSchema && Array.isArray(selectedProduct.constraintSchema)
      ? selectedProduct.constraintSchema
      : []),
    [selectedProduct]
  )

  function handleProductChange(value: string) {
    setProductId(value)
    setPlanId("")
    setRegistrationCode("")
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
      toast.success(t("license:licenses.issueSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const generateRegMutation = useMutation({
    mutationFn: () =>
      api.post<RegistrationOption>("/api/v1/license/registrations/generate", {
        productId: Number(productId) || undefined,
        licenseeId: Number(licenseeId) || undefined,
      }),
    onSuccess: (data) => {
      setRegistrationCode(data.code)
      toast.success(t("license:registrations.generateSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!productId || !licenseeId || !registrationCode || !validFrom) return

    const selectedPlan = plans.find((p) => String(p.id) === planId)
    const planName = planId === "custom" ? t("license:licenses.custom") : (selectedPlan?.name ?? t("license:licenses.custom"))

    issueMutation.mutate({
      productId: Number(productId),
      licenseeId: Number(licenseeId),
      planId: planId && planId !== "custom" ? Number(planId) : null,
      planName,
      registrationCode,
      autoCreateRegistration: registrationMode === "manual",
      constraintValues,
      validFrom,
      validUntil: validUntil || null,
      notes,
    })
  }

  const isPlanSelected = planId && planId !== "custom"
  const isCustomPlan = planId === "custom"

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{t("license:licenses.issue")}</SheetTitle>
          <SheetDescription className="sr-only">{t("license:licenses.issueLicenseDesc")}</SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="flex flex-1 flex-col gap-4 px-4">
          {/* Product */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">{t("license:licenses.productRequired")}</Label>
            <Select value={productId} onValueChange={handleProductChange}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t("license:licenses.selectProduct")} />
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
            <Label className="text-xs text-muted-foreground">{t("license:licenses.licenseeRequired")}</Label>
            <Select value={licenseeId} onValueChange={setLicenseeId}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t("license:licenses.selectLicensee")} />
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
              <Label className="text-xs text-muted-foreground">{t("license:licenses.selectPlanOrCustom")}</Label>
              <Select value={planId} onValueChange={handlePlanChange}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={t("license:licenses.selectPlanOrCustom")} />
                </SelectTrigger>
                <SelectContent>
                  {plans.map((p) => (
                    <SelectItem key={p.id} value={String(p.id)}>
                      {p.name} {p.isDefault && t("license:licenses.defaultSuffix")}
                    </SelectItem>
                  ))}
                  <SelectItem value="custom">{t("license:licenses.custom")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          {/* Constraint Values - read-only summary for preset plan */}
          {productId && schema.length > 0 && isPlanSelected && (
            <div className="space-y-2.5 rounded-lg border bg-muted/15 p-3">
              <div className="flex items-center justify-between">
                <Label className="text-xs text-muted-foreground">{t("license:licenses.planSummary")}</Label>
                <Badge variant="outline" className="rounded-md border-0 bg-muted/60 text-[11px] text-muted-foreground">
                  {schema.length} {t("license:plans.moduleCount", { count: schema.length })}
                </Badge>
              </div>
              {schema.map((mod) => {
                const modValues = constraintValues[mod.key] ?? { enabled: false }
                const isEnabled = !!modValues.enabled
                const enabledFeatures = mod.features.filter((f) => modValues[f.key] !== undefined)
                return (
                  <div key={mod.key} className="text-sm">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{mod.label || mod.key}</span>
                      <Badge variant={isEnabled ? "default" : "outline"} className="text-[10px]">
                        {isEnabled ? t("license:licenses.moduleEnabled") : t("license:licenses.moduleDisabled")}
                      </Badge>
                    </div>
                    {isEnabled && enabledFeatures.length > 0 && (
                      <div className="mt-1 flex flex-wrap gap-1 text-xs text-muted-foreground">
                        {enabledFeatures.map((f) => (
                          <span key={f.key} className="rounded bg-muted/60 px-1.5 py-0.5">
                            {f.label || f.key}: {String(modValues[f.key])}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          {/* Constraint Values - editable for custom plan */}
          {productId && schema.length > 0 && isCustomPlan && (
            <div className="space-y-2.5">
              <div className="flex items-center justify-between">
                <Label className="text-xs text-muted-foreground">{t("license:plans.constraintConfig")}</Label>
                <Badge variant="outline" className="rounded-md border-0 bg-muted/60 text-[11px] text-muted-foreground">
                  {schema.length} {t("license:plans.moduleCount", { count: schema.length })}
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
                        checked={isEnabled}
                        onCheckedChange={(checked) => setModuleEnabled(mod.key, !!checked)}
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
          <div className="space-y-2 rounded-lg border p-3">
            <Label className="text-xs text-muted-foreground">{t("license:licenses.registrationCodeRequired")}</Label>
            <div className="flex gap-2">
              <Button
                type="button"
                variant={registrationMode === "select" ? "default" : "outline"}
                size="sm"
                onClick={() => { setRegistrationMode("select"); setRegistrationCode("") }}
              >
                {t("license:licenses.selectRegistrationCode")}
              </Button>
              <Button
                type="button"
                variant={registrationMode === "generate" ? "default" : "outline"}
                size="sm"
                onClick={() => { setRegistrationMode("generate"); setRegistrationCode("") }}
              >
                {t("license:licenses.autoGenerate")}
              </Button>
              <Button
                type="button"
                variant={registrationMode === "manual" ? "default" : "outline"}
                size="sm"
                onClick={() => { setRegistrationMode("manual"); setRegistrationCode("") }}
              >
                {t("license:licenses.manualInput")}
              </Button>
            </div>
            {registrationMode === "select" && (
              <>
                <Select value={registrationCode} onValueChange={setRegistrationCode}>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder={t("license:licenses.selectRegistrationCode")} />
                  </SelectTrigger>
                  <SelectContent>
                    {registrations.map((r) => (
                      <SelectItem key={r.id} value={r.code}>
                        {r.code} {r.expiresAt ? `(${t("license:registrations.expiresAt")}: ${r.expiresAt.split("T")[0]})` : ""}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {productId && registrations.length === 0 && !registrationsData && (
                  <p className="text-xs text-muted-foreground">{t("license:licenses.noAvailableRegistrations")}</p>
                )}
              </>
            )}
            {registrationMode === "generate" && (
              <div className="flex gap-2">
                <Input
                  value={registrationCode}
                  onChange={(e) => setRegistrationCode(e.target.value)}
                  placeholder={t("license:licenses.registrationCodePlaceholder")}
                  readOnly={generateRegMutation.isSuccess}
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => generateRegMutation.mutate()}
                  disabled={generateRegMutation.isPending || !productId}
                >
                  {generateRegMutation.isPending ? t("common:processing") : t("license:licenses.generateRegistrationCode")}
                </Button>
              </div>
            )}
            {registrationMode === "manual" && (
              <Input
                value={registrationCode}
                onChange={(e) => setRegistrationCode(e.target.value)}
                placeholder={t("license:licenses.registrationCodePlaceholder")}
                className="w-full"
              />
            )}
          </div>

          {/* Valid From */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">{t("license:licenses.validFromDate")}</Label>
              <Input
                type="date"
                value={validFrom}
                onChange={(e) => setValidFrom(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">{t("license:licenses.validUntilDate")}</Label>
              <Input
                type="date"
                value={validUntil}
                onChange={(e) => setValidUntil(e.target.value)}
                placeholder={t("license:licenses.emptyForPermanent")}
              />
            </div>
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">{t("license:licenses.notes")}</Label>
            <Textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder={t("license:licenses.optionalNotes")}
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
              {issueMutation.isPending ? t("license:licenses.issuing") : t("license:licenses.issueBtn")}
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
}: {
  feature: ConstraintFeature
  value: unknown
  onChange: (v: unknown) => void
}) {
  const { t } = useTranslation("license")
  if (feature.type === "number") {
    return (
      <div className="rounded-md bg-background/60 px-3 py-2.5">
        <div className="flex items-center justify-between gap-2">
          <Label className="text-xs font-medium">{feature.label || feature.key}</Label>
          {(feature.min != null || feature.max != null) && (
            <Badge variant="outline" className="rounded-md border-0 bg-muted/60 text-[11px] text-muted-foreground">
              {feature.min ?? t("license:plans.noLimit")} ~ {feature.max ?? t("license:plans.noLimit")}
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
          <SelectTrigger className="mt-1.5 w-full rounded-md bg-background/80 text-sm h-8">
            <SelectValue placeholder={t("license:plans.selectPlaceholder")} />
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
