import { useParams, useNavigate } from "react-router"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { ArrowLeft, Ban, Download, Loader2 } from "lucide-react"
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

const STATUS_MAP: Record<string, { label: string; variant: "default" | "destructive" }> = {
  issued: { label: "已签发", variant: "default" },
  revoked: { label: "已吊销", variant: "destructive" },
}

export function Component() {
  const { id } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const canRevoke = usePermission("license:license:revoke")

  const { data: license, isLoading } = useQuery({
    queryKey: ["license-license", id],
    queryFn: () => api.get<LicenseDetail>(`/api/v1/license/licenses/${id}`),
    enabled: !!id,
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

  function handleExport() {
    window.open(`/api/v1/license/licenses/${id}/export`, "_blank")
  }

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
  const constraintValues = license.constraintValues ?? {}
  const modules = Object.entries(constraintValues)

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
            <InfoRow label="注册码" value={license.registrationCode} mono />
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
        <div className="rounded-lg border p-4 space-y-3">
          <h3 className="text-sm font-medium text-muted-foreground">功能约束</h3>
          <div className="space-y-3">
            {modules.map(([moduleKey, moduleValues]) => {
              const mv = moduleValues as Record<string, unknown>
              const isEnabled = mv.enabled !== false
              const features = Object.entries(mv).filter(([k]) => k !== "enabled")
              return (
                <div key={moduleKey} className="rounded-md border bg-muted/10 p-3">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="text-sm font-medium">{moduleKey}</span>
                    <Badge variant={isEnabled ? "default" : "outline"} className="text-[11px]">
                      {isEnabled ? "已启用" : "未启用"}
                    </Badge>
                  </div>
                  {isEnabled && features.length > 0 && (
                    <div className="grid gap-1.5 text-sm text-muted-foreground">
                      {features.map(([key, value]) => (
                        <div key={key} className="flex items-center justify-between">
                          <span>{key}</span>
                          <span className="font-mono text-foreground">
                            {Array.isArray(value) ? value.join(", ") : String(value)}
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

function InfoRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-start justify-between gap-4">
      <dt className="text-muted-foreground shrink-0">{label}</dt>
      <dd className={`text-right break-all ${mono ? "font-mono text-xs" : ""}`}>{value}</dd>
    </div>
  )
}
