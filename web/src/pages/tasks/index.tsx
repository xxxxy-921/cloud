import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "react-router"
import { Clock, Play, Pause, Zap, Activity, CheckCircle, XCircle, Timer } from "lucide-react"
import { taskApi, type TaskInfo } from "@/lib/api"
import { usePermission } from "@/hooks/use-permission"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import {
  DataTableActions,
  DataTableCard,
  DataTableEmptyRow,
  DataTableLoadingRow,
} from "@/components/ui/data-table"
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

function formatRelativeTime(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return "刚刚"
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  return `${days}天前`
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
    active: { label: "运行中", variant: "default" },
    paused: { label: "已暂停", variant: "secondary" },
    completed: { label: "完成", variant: "default" },
    failed: { label: "失败", variant: "destructive" },
    timeout: { label: "超时", variant: "destructive" },
    pending: { label: "等待中", variant: "outline" },
    running: { label: "执行中", variant: "default" },
    stale: { label: "中断", variant: "secondary" },
  }
  const info = map[status] || { label: status, variant: "outline" as const }
  return <Badge variant={info.variant}>{info.label}</Badge>
}

export function Component() {
  const [tab, setTab] = useState<"all" | "scheduled" | "async">("all")
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const canPause = usePermission("system:task:pause")
  const canResume = usePermission("system:task:resume")
  const canTrigger = usePermission("system:task:trigger")

  const { data: stats } = useQuery({
    queryKey: ["task-stats"],
    queryFn: () => taskApi.stats(),
    refetchInterval: 10000,
  })

  const typeFilter = tab === "all" ? undefined : tab
  const { data: tasks, isLoading } = useQuery({
    queryKey: ["tasks", typeFilter],
    queryFn: () => taskApi.list(typeFilter),
    refetchInterval: 10000,
  })

  const pauseMutation = useMutation({
    mutationFn: (name: string) => taskApi.pause(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] })
      queryClient.invalidateQueries({ queryKey: ["task-stats"] })
    },
  })

  const resumeMutation = useMutation({
    mutationFn: (name: string) => taskApi.resume(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] })
      queryClient.invalidateQueries({ queryKey: ["task-stats"] })
    },
  })

  const triggerMutation = useMutation({
    mutationFn: (name: string) => taskApi.trigger(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] })
      queryClient.invalidateQueries({ queryKey: ["task-stats"] })
    },
  })

  const statCards = [
    { label: "总任务数", value: stats?.totalTasks ?? 0, icon: Clock, color: "text-blue-500" },
    { label: "运行中", value: stats?.running ?? 0, icon: Activity, color: "text-green-500" },
    { label: "今日完成", value: stats?.completedToday ?? 0, icon: CheckCircle, color: "text-emerald-500" },
    { label: "今日失败", value: stats?.failedToday ?? 0, icon: XCircle, color: "text-red-500" },
  ]

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">任务中心</h2>

      {/* Stats Cards */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {statCards.map((card) => (
          <Card key={card.label}>
            <CardContent className="flex items-center gap-3 p-4">
              <card.icon className={`h-8 w-8 ${card.color}`} />
              <div>
                <p className="text-2xl font-bold">{card.value}</p>
                <p className="text-xs text-muted-foreground">{card.label}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Tab Buttons */}
      <div className="flex gap-1 border-b pb-2">
        {([
          { key: "all", label: "全部" },
          { key: "scheduled", label: "定时任务" },
          { key: "async", label: "异步队列" },
        ] as const).map((t) => (
          <Button
            key={t.key}
            variant={tab === t.key ? "default" : "ghost"}
            size="sm"
            onClick={() => setTab(t.key)}
          >
            {t.label}
          </Button>
        ))}
      </div>

      {/* Task Table */}
      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[220px]">名称</TableHead>
              <TableHead className="min-w-[220px]">描述</TableHead>
              <TableHead className="w-[140px]">类型</TableHead>
              <TableHead className="w-[120px]">状态</TableHead>
              <TableHead className="min-w-[180px]">上次执行</TableHead>
              <TableHead className="w-[180px] text-center">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={6} />
            ) : !tasks || tasks.length === 0 ? (
              <DataTableEmptyRow colSpan={6} icon={Timer} title="暂无任务" />
            ) : (
              tasks.map((task: TaskInfo) => (
                <TableRow
                  key={task.name}
                  className="cursor-pointer"
                  onClick={() => navigate(`/tasks/${task.name}`)}
                  >
                    <TableCell className="font-mono text-sm font-medium">{task.name}</TableCell>
                    <TableCell className="max-w-[360px] text-sm text-muted-foreground">
                      <span className="block truncate" title={task.description || "-"}>
                        {task.description || "-"}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {task.type === "scheduled" ? task.cronExpr || "定时" : "异步"}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={task.status} />
                  </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {task.lastExecution ? (
                        <span className="flex items-center gap-1.5">
                        <StatusBadge status={task.lastExecution.status} />
                        <span>{formatRelativeTime(task.lastExecution.timestamp)}</span>
                      </span>
                    ) : (
                      "-"
                    )}
                    </TableCell>
                    <TableCell className="text-center" onClick={(e) => e.stopPropagation()}>
                      <DataTableActions className="justify-center">
                        {task.type === "scheduled" && task.status === "active" && canPause && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="px-2.5"
                            onClick={() => pauseMutation.mutate(task.name)}
                            disabled={pauseMutation.isPending}
                          >
                          <Pause className="mr-1 h-3.5 w-3.5" />
                          暂停
                        </Button>
                      )}
                      {task.type === "scheduled" && task.status === "paused" && canResume && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="px-2.5"
                            onClick={() => resumeMutation.mutate(task.name)}
                            disabled={resumeMutation.isPending}
                          >
                          <Play className="mr-1 h-3.5 w-3.5" />
                          恢复
                        </Button>
                      )}
                        {canTrigger && (
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button variant="ghost" size="sm" className="px-2.5">
                                <Zap className="mr-1 h-3.5 w-3.5" />
                                触发
                              </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>手动触发任务</AlertDialogTitle>
                              <AlertDialogDescription>
                                确定要立即执行任务 &ldquo;{task.name}&rdquo; 吗？
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>取消</AlertDialogCancel>
                              <AlertDialogAction onClick={() => triggerMutation.mutate(task.name)}>
                                执行
                              </AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                          </AlertDialog>
                        )}
                      </DataTableActions>
                    </TableCell>
                  </TableRow>
                ))
              )}
          </TableBody>
        </Table>
      </DataTableCard>
    </div>
  )
}
