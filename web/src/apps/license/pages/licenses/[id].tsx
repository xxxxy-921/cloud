import { useMemo, useState } from "react"
import { useParams, useNavigate } from "react-router"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { ArrowLeft, Ban, Check, Copy, Download, Loader2 } from "lucide-react"
import { api } from "@/lib/api"
import { usePermission } from "@/hooks/use-permission"
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
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { formatDateTime } from "@/lib/utils"

interface LicenseDetail {
  id: number
  productId: number | null
  licenseeId: number | null
  planId: number | null
  planName: string
  registrationCode: string
  constraintValues: Record<string, Record<string, unknown>>
  validFrom: string
  validUntil: string | null
  activationCode: string
  keyVersion: number
  signature: string
  status: string
  issuedBy: number
  revokedAt: string | null
  revokedBy: number | null
  notes: string
  productName: string
  productCode: string
  licenseeName: string
  licenseeCode: string
  createdAt: string
  updatedAt: string
}

interface ConstraintFeature {
  key: string
  label: string
}

interface ConstraintModule {
  key: string
  label: string
  features: ConstraintFeature[]
}

interface ProductConstraintDetail {
  constraintSchema: ConstraintModule[] | null
}

interface SignedActivationClaims {
  pid?: string
  lic?: string
  licn?: string
  reg?: string
  iat?: number
  nbf?: number
  exp?: number | null
}

const STATUS_MAP: Record<string, { label: string; variant: "default" | "destructive" }> = {
  issued: { label: "已签发", variant: "default" },
  revoked: { label: "已吊销", variant: "destructive" },
}

export function Component() {
  const { id } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const canRevoke = usePermission("license:license:revoke")
  const [copied, setCopied] = useState(false)

  const { data: license, isLoading } = useQuery({
    queryKey: ["license-license", id],
    queryFn: () => api.get<LicenseDetail>(`/api/v1/license/licenses/${id}`),
    enabled: !!id,
  })

  const { data: productDetail } = useQuery({
    queryKey: ["license-product-constraint", license?.productId],
    queryFn: () => api.get<ProductConstraintDetail>(`/api/v1/license/products/${license?.productId}`),
    enabled: !!license?.productId,
  })

  const revokeMutation = useMutation({
    mutationFn: () => api.patch(`/api/v1/license/licenses/${id}/revoke`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-license", id] })
      queryClient.invalidateQueries({ queryKey: ["license-licenses"] })
      toast.success("许可已吊销")
    },
    onError: (err) => toast.error(err.message),
  })

  async function handleExport() {
    if (!id) return
    try {
      const blob = await api.download(`/api/v1/license/licenses/${id}/export`)
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement("a")
      anchor.href = url
      anchor.download = `${license?.productCode || "license"}_${id}.lic`
      document.body.appendChild(anchor)
      anchor.click()
      anchor.remove()
      URL.revokeObjectURL(url)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "导出失败")
    }
  }

  async function handleCopyRegistrationCode() {
    if (!license?.registrationCode) return
    try {
      await navigator.clipboard.writeText(license.registrationCode)
      setCopied(true)
      toast.success("注册码已复制")
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      toast.error("复制失败，请手动复制")
    }
  }

  const modules = useMemo(() => {
    const constraintValues = license?.constraintValues ?? {}
    const constraintSchema = Array.isArray(productDetail?.constraintSchema) ? productDetail.constraintSchema : []

    const schemaByModule = new Map(constraintSchema.map((m) => [m.key, m]))
    const valueModuleKeys = Object.keys(constraintValues)

    const orderedModuleKeys = [
      ...constraintSchema.map((m) => m.key).filter((key) => key in constraintValues),
      ...valueModuleKeys.filter((key) => !schemaByModule.has(key)),
    ]

    return orderedModuleKeys.map((moduleKey) => {
      const moduleSchema = schemaByModule.get(moduleKey)
      const rawModuleValues = constraintValues[moduleKey]
      const moduleValues =
        rawModuleValues && typeof rawModuleValues === "object" && !Array.isArray(rawModuleValues)
          ? (rawModuleValues as Record<string, unknown>)
          : {}

      const featureLabelByKey = new Map(
        (moduleSchema?.features ?? []).map((feature) => [feature.key, feature.label || feature.key]),
      )

      const features = Object.entries(moduleValues)
        .filter(([key]) => key !== "enabled")
        .map(([key, value]) => ({
          key,
          label: featureLabelByKey.get(key) ?? key,
          value,
        }))

      return {
        key: moduleKey,
        label: moduleSchema?.label || moduleKey,
        isEnabled: moduleValues.enabled !== false,
        features,
      }
    })
  }, [license, productDetail])

  const signedClaims = useMemo(
    () => decodeActivationClaims(license?.activationCode),
    [license?.activationCode],
  )

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!license) {
    return (
      <div className="py-20 text-center text-muted-foreground">许可不存在</div>
    )
  }

  const status = STATUS_MAP[license.status] ?? { label: license.status, variant: "default" as const }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => navigate("/license/licenses")}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h2 className="text-lg font-semibold">许可详情</h2>
          <Badge variant={status.variant}>{status.label}</Badge>
        </div>
        {license.status === "issued" && (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handleExport}>
              <Download className="mr-1.5 h-4 w-4" />
              导出 .lic
            </Button>
            {canRevoke && (
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button variant="destructive" size="sm">
                    <Ban className="mr-1.5 h-4 w-4" />
                    吊销
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>吊销许可</AlertDialogTitle>
                    <AlertDialogDescription>
                      确定要吊销此许可吗？吊销后已导出的 .lic 文件仍可离线使用。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      onClick={() => revokeMutation.mutate()}
                      disabled={revokeMutation.isPending}
                    >
                      {revokeMutation.isPending ? "处理中..." : "确定吊销"}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            )}
          </div>
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Basic Info */}
        <div className="rounded-lg border p-4 space-y-3">
          <h3 className="text-sm font-medium text-muted-foreground">基本信息</h3>
          <dl className="space-y-2 text-sm">
            <InfoRow label="商品" value={license.productName ? `${license.productName} (${license.productCode})` : "-"} />
            <InfoRow label="授权主体" value={license.licenseeName ? `${license.licenseeName} (${license.licenseeCode})` : "-"} />
            <InfoRow label="套餐" value={license.planName} />
            <div className="flex items-start justify-between gap-4">
              <dt className="text-muted-foreground shrink-0">注册码</dt>
              <dd className="flex items-center gap-2">
                <span className="text-right break-all font-mono text-xs">{license.registrationCode}</span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-7 px-2"
                  onClick={handleCopyRegistrationCode}
                >
                  {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                  <span className="ml-1">{copied ? "已复制" : "复制"}</span>
                </Button>
              </dd>
            </div>
          </dl>
        </div>

        {/* Validity */}
        <div className="rounded-lg border p-4 space-y-3">
          <h3 className="text-sm font-medium text-muted-foreground">有效期</h3>
          <dl className="space-y-2 text-sm">
            <InfoRow label="生效时间" value={new Date(license.validFrom).toLocaleString()} />
            <InfoRow
              label="过期时间"
              value={license.validUntil ? new Date(license.validUntil).toLocaleString() : "永久有效"}
            />
          </dl>
        </div>

        {/* Issuance Info */}
        <div className="rounded-lg border p-4 space-y-3">
          <h3 className="text-sm font-medium text-muted-foreground">签发信息</h3>
          <dl className="space-y-2 text-sm">
            <InfoRow label="签发时间" value={formatDateTime(license.createdAt)} />
            <InfoRow label="密钥版本" value={`v${license.keyVersion}`} />
            {license.notes && <InfoRow label="备注" value={license.notes} />}
          </dl>
        </div>

        {signedClaims && (
          <div className="rounded-lg border p-4 space-y-3">
            <h3 className="text-sm font-medium text-muted-foreground">激活码签名信息</h3>
            <dl className="space-y-2 text-sm">
              <InfoRow label="产品编码" value={signedClaims.pid || license.productCode || "-"} mono />
              <InfoRow label="授权主体" value={signedClaims.licn || license.licenseeName || "-"} />
              <InfoRow label="主体编码" value={signedClaims.lic || license.licenseeCode || "-"} mono />
            </dl>
          </div>
        )}

        {/* Revocation Info */}
        {license.status === "revoked" && (
          <div className="rounded-lg border border-destructive/20 bg-destructive/5 p-4 space-y-3">
            <h3 className="text-sm font-medium text-destructive">吊销信息</h3>
            <dl className="space-y-2 text-sm">
              <InfoRow label="吊销时间" value={license.revokedAt ? formatDateTime(license.revokedAt) : "-"} />
            </dl>
          </div>
        )}
      </div>

      {/* Constraint Values */}
      {modules.length > 0 && (
        <div className="rounded-lg border p-4 space-y-2.5">
          <h3 className="text-sm font-medium text-muted-foreground">功能约束</h3>
          <div className="space-y-2">
            {modules.map((module) => {
              return (
                <div key={module.key} className="rounded-md border bg-muted/10 p-2.5">
                  <div className="mb-1.5 flex items-center gap-2">
                    <span className="text-sm font-medium leading-5">{module.label}</span>
                    <Badge variant={module.isEnabled ? "default" : "outline"} className="text-[11px]">
                      {module.isEnabled ? "已启用" : "未启用"}
                    </Badge>
                  </div>
                  {module.isEnabled && module.features.length > 0 && (
                    <div className="grid gap-1 text-sm">
                      {module.features.map((feature) => (
                        <div
                          key={feature.key}
                          className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-md bg-background/65 px-2 py-1"
                        >
                          <span className="truncate text-muted-foreground">{feature.label}</span>
                          <span className="min-w-16 pr-1 text-right font-mono tabular-nums text-foreground">
                            {formatConstraintValue(feature.value)}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}

function decodeActivationClaims(activationCode?: string): SignedActivationClaims | null {
  if (!activationCode) {
    return null
  }

  try {
    const normalized = activationCode.replace(/-/g, "+").replace(/_/g, "/")
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=")
    const binary = window.atob(padded)
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
    const json = new TextDecoder().decode(bytes)
    return JSON.parse(json) as SignedActivationClaims
  } catch {
    return null
  }
}

function formatConstraintValue(value: unknown): string {
  if (value == null) {
    return "-"
  }
  if (Array.isArray(value)) {
    return value.length > 0 ? value.join(", ") : "-"
  }
  if (typeof value === "boolean") {
    return value ? "是" : "否"
  }
  return String(value)
}

function InfoRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-start justify-between gap-4">
      <dt className="text-muted-foreground shrink-0">{label}</dt>
      <dd className={`text-right break-all ${mono ? "font-mono text-xs" : ""}`}>{value}</dd>
    </div>
  )
}
