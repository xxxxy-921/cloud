import { Badge } from "@/components/ui/badge"
import type { AssetStatus } from "./types"

const statusConfig: Record<AssetStatus, { label: string; className: string }> = {
  idle: {
    label: "未构建",
    className: "",
  },
  building: {
    label: "构建中",
    className: "border-transparent bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400 animate-pulse",
  },
  ready: {
    label: "就绪",
    className: "border-transparent bg-green-500/20 text-green-700 dark:bg-green-500/20 dark:text-green-400",
  },
  error: {
    label: "错误",
    className: "",
  },
  stale: {
    label: "已过期",
    className: "border-transparent bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
  },
}

export function AssetStatusBadge({ status }: { status: string }) {
  const s = (status as AssetStatus) || "idle"
  const cfg = statusConfig[s] ?? statusConfig.idle

  if (s === "error") {
    return <Badge variant="destructive">{cfg.label}</Badge>
  }
  if (s === "idle") {
    return <Badge variant="secondary">{cfg.label}</Badge>
  }
  return (
    <Badge variant="outline" className={cfg.className}>
      {cfg.label}
    </Badge>
  )
}
