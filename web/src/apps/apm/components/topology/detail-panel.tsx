import { useEffect, useCallback } from "react"
import { useQuery } from "@tanstack/react-query"
import { useNavigate } from "react-router"
import { useTranslation } from "react-i18next"
import { X, ExternalLink, Activity, Clock, AlertTriangle, BarChart3 } from "lucide-react"

import { fetchServiceDetail, type OperationStats } from "../../api"
import { ServiceIcon, getHealthLevel } from "./service-node"

interface DetailPanelProps {
  serviceName: string
  timeStart: string
  timeEnd: string
  onClose: () => void
}

export function DetailPanel({ serviceName, timeStart, timeEnd, onClose }: DetailPanelProps) {
  const { t } = useTranslation("apm")
  const navigate = useNavigate()

  // Escape key to close
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose()
    },
    [onClose],
  )

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown)
    return () => document.removeEventListener("keydown", handleKeyDown)
  }, [handleKeyDown])

  const { data, isLoading } = useQuery({
    queryKey: ["apm-service-detail", serviceName, timeStart, timeEnd],
    queryFn: () => fetchServiceDetail(serviceName, timeStart, timeEnd),
  })

  const health = data ? getHealthLevel(data.errorRate) : "healthy"

  const healthBadge = {
    critical: "bg-red-500/15 text-red-500 border-red-500/25",
    warning: "bg-amber-500/12 text-amber-600 dark:text-amber-400 border-amber-500/25",
    healthy: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/25",
  }

  const healthLabel = {
    critical: "Critical",
    warning: "Warning",
    healthy: "Healthy",
  }

  function handleOperationClick(op: OperationStats) {
    const params = new URLSearchParams()
    params.set("service", serviceName)
    params.set("operation", op.spanName)
    if (timeStart) params.set("start", timeStart)
    if (timeEnd) params.set("end", timeEnd)
    navigate(`/apm/traces?${params}`)
  }

  function handleViewFull() {
    const params = new URLSearchParams()
    if (timeStart) params.set("start", timeStart)
    if (timeEnd) params.set("end", timeEnd)
    navigate(`/apm/services/${encodeURIComponent(serviceName)}?${params}`)
  }

  return (
    <div className="w-[380px] border-l bg-card flex flex-col h-full animate-in slide-in-from-right-2 duration-200">
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-3 border-b">
        <div className="flex items-center justify-center w-9 h-9 rounded-full bg-muted/60">
          <ServiceIcon name={serviceName} className="w-4.5 h-4.5 text-muted-foreground" strokeWidth={2} />
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="text-sm font-semibold truncate">{serviceName}</h3>
          <span
            className={`inline-flex items-center mt-0.5 rounded-full border px-2 py-0.5 text-[10px] font-medium leading-none ${healthBadge[health]}`}
          >
            {healthLabel[health]}
          </span>
        </div>
        <button
          onClick={onClose}
          className="p-1.5 rounded-md hover:bg-muted/80 text-muted-foreground transition-colors"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* Body */}
      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="flex items-center justify-center h-32 text-sm text-muted-foreground">
            {t("loading")}
          </div>
        ) : data ? (
          <>
            {/* Metric Cards */}
            <div className="grid grid-cols-2 gap-2 p-4">
              <MetricCard
                icon={BarChart3}
                label={t("topology.detail.requestCount")}
                value={formatNumber(data.requestCount)}
              />
              <MetricCard
                icon={Clock}
                label={t("topology.detail.avgLatency")}
                value={`${data.avgDurationMs.toFixed(1)} ms`}
              />
              <MetricCard
                icon={Activity}
                label={t("topology.detail.p95")}
                value={`${data.p95Ms.toFixed(1)} ms`}
              />
              <MetricCard
                icon={AlertTriangle}
                label={t("topology.detail.errorRate")}
                value={`${data.errorRate.toFixed(2)}%`}
                variant={data.errorRate > 5 ? "critical" : data.errorRate > 1 ? "warning" : "default"}
              />
            </div>

            {/* Operations Table */}
            {data.operations && data.operations.length > 0 && (
              <div className="px-4 pb-4">
                <h4 className="text-xs font-semibold text-muted-foreground mb-2 uppercase tracking-wider">
                  {t("topology.detail.operations")}
                </h4>
                <div className="rounded-lg border overflow-hidden">
                  <table className="w-full text-[11px]">
                    <thead>
                      <tr className="bg-muted/40 border-b">
                        <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">{t("services.operationName")}</th>
                        <th className="text-right py-1.5 px-2 font-medium text-muted-foreground">{t("services.reqRate")}</th>
                        <th className="text-right py-1.5 px-2 font-medium text-muted-foreground">{t("services.avgDuration")}</th>
                        <th className="text-right py-1.5 px-2 font-medium text-muted-foreground">{t("services.errorRate")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.operations.map((op) => (
                        <tr
                          key={op.spanName}
                          className="border-b last:border-0 hover:bg-muted/30 cursor-pointer transition-colors"
                          onClick={() => handleOperationClick(op)}
                        >
                          <td className="py-1.5 px-2 font-mono truncate max-w-[140px]">{op.spanName}</td>
                          <td className="py-1.5 px-2 text-right font-mono">{formatNumber(op.requestCount)}</td>
                          <td className="py-1.5 px-2 text-right font-mono">{op.avgDurationMs.toFixed(1)}</td>
                          <td className={`py-1.5 px-2 text-right font-mono ${op.errorRate > 5 ? "text-red-500 font-semibold" : op.errorRate > 1 ? "text-amber-500" : ""}`}>
                            {op.errorRate.toFixed(1)}%
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </>
        ) : null}
      </div>

      {/* Footer */}
      <div className="border-t p-3">
        <button
          onClick={handleViewFull}
          className="w-full flex items-center justify-center gap-1.5 rounded-lg bg-primary/10 hover:bg-primary/15 text-primary text-sm font-medium py-2 transition-colors"
        >
          {t("topology.detail.viewFull")}
          <ExternalLink className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  )
}

// --- Metric card ---

function MetricCard({
  icon: Icon,
  label,
  value,
  variant = "default",
}: {
  icon: typeof Activity
  label: string
  value: string
  variant?: "default" | "warning" | "critical"
}) {
  const variantClasses = {
    default: "",
    warning: "text-amber-500",
    critical: "text-red-500",
  }
  return (
    <div className="rounded-lg border bg-muted/20 p-3">
      <div className="flex items-center gap-1.5 mb-1">
        <Icon className="w-3.5 h-3.5 text-muted-foreground" />
        <span className="text-[10px] text-muted-foreground font-medium">{label}</span>
      </div>
      <span className={`text-lg font-semibold font-mono leading-none ${variantClasses[variant]}`}>
        {value}
      </span>
    </div>
  )
}

function formatNumber(n: number): string {
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return String(n)
}
