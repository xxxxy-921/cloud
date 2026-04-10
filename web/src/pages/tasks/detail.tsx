import { useState } from "react"
import { useParams, useNavigate } from "react-router"
import { useQuery } from "@tanstack/react-query"
import { ArrowLeft, Clock, RotateCcw, Timer, Zap } from "lucide-react"
import { taskApi, type TaskExecution } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  DataTableCard,
  DataTableEmptyRow,
  DataTableLoadingRow,
  DataTablePagination,
} from "@/components/ui/data-table"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { formatDateTime } from "@/lib/utils"

function ExecStatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
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

function TriggerBadge({ trigger }: { trigger: string }) {
  const map: Record<string, { label: string; icon: typeof Clock }> = {
    cron: { label: "定时", icon: Clock },
    manual: { label: "手动", icon: Zap },
    api: { label: "API", icon: RotateCcw },
  }
  const info = map[trigger] || { label: trigger, icon: Clock }
  return (
    <Badge variant="outline" className="gap-1">
      <info.icon className="h-3 w-3" />
      {info.label}
    </Badge>
  )
}

function formatDuration(exec: TaskExecution): string {
  if (!exec.startedAt || !exec.finishedAt) return "-"
  const ms = new Date(exec.finishedAt).getTime() - new Date(exec.startedAt).getTime()
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

export function Component() {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 20

  const { data: detail } = useQuery({
    queryKey: ["task-detail", name],
    queryFn: () => taskApi.get(name!),
    enabled: !!name,
  })

  const { data: execData, isLoading: execLoading } = useQuery({
    queryKey: ["task-executions", name, page],
    queryFn: () => taskApi.executions(name!, page, pageSize),
    enabled: !!name,
    refetchInterval: 10000,
  })

  const task = detail?.task
  const executions = execData?.list ?? []
  const total = execData?.total ?? 0
  const totalPages = Math.ceil(total / pageSize)

  if (!task) {
    return (
      <div className="flex h-64 items-center justify-center text-muted-foreground">
        加载中...
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="sm" onClick={() => navigate("/tasks")}>
          <ArrowLeft className="mr-1 h-4 w-4" />
          返回
        </Button>
        <h2 className="text-lg font-semibold">{task.name}</h2>
      </div>

      {/* Task Config Card */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">任务配置</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-3">
            <div>
              <p className="text-muted-foreground">名称</p>
              <p className="font-mono font-medium">{task.name}</p>
            </div>
            <div>
              <p className="text-muted-foreground">类型</p>
              <Badge variant="outline">{task.type === "scheduled" ? "定时任务" : "异步任务"}</Badge>
            </div>
            <div>
              <p className="text-muted-foreground">状态</p>
              <Badge variant={task.status === "active" ? "default" : "secondary"}>
                {task.status === "active" ? "运行中" : "已暂停"}
              </Badge>
            </div>
            <div>
              <p className="text-muted-foreground">描述</p>
              <p>{task.description || "-"}</p>
            </div>
            {task.cronExpr && (
              <div>
                <p className="text-muted-foreground">Cron 表达式</p>
                <p className="font-mono">{task.cronExpr}</p>
              </div>
            )}
            <div>
              <p className="text-muted-foreground">超时</p>
              <p>{task.timeoutMs >= 1000 ? `${task.timeoutMs / 1000}s` : `${task.timeoutMs}ms`}</p>
            </div>
            <div>
              <p className="text-muted-foreground">最大重试</p>
              <p>{task.maxRetries} 次</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Execution History */}
      <div>
        <h3 className="mb-2 text-base font-semibold">执行历史</h3>
        <DataTableCard>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[90px]">ID</TableHead>
                <TableHead className="w-[120px]">触发方式</TableHead>
                <TableHead className="w-[120px]">状态</TableHead>
                <TableHead className="w-[100px]">耗时</TableHead>
                <TableHead className="min-w-[240px]">错误信息</TableHead>
                <TableHead className="w-[150px]">时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {execLoading ? (
                <DataTableLoadingRow colSpan={6} />
              ) : executions.length === 0 ? (
                <DataTableEmptyRow colSpan={6} icon={Timer} title="暂无执行记录" />
              ) : (
                executions.map((exec: TaskExecution) => (
                  <TableRow key={exec.id}>
                    <TableCell className="font-mono text-sm">{exec.id}</TableCell>
                    <TableCell>
                      <TriggerBadge trigger={exec.trigger} />
                    </TableCell>
                    <TableCell>
                      <ExecStatusBadge status={exec.status} />
                    </TableCell>
                    <TableCell className="font-mono text-sm">{formatDuration(exec)}</TableCell>
                    <TableCell className="max-w-[200px] truncate text-sm text-muted-foreground" title={exec.error}>
                      {exec.error || "-"}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatDateTime(exec.createdAt)}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </DataTableCard>

        <DataTablePagination
          className="mt-4"
          total={total}
          page={page}
          totalPages={totalPages}
          onPageChange={setPage}
        />
      </div>
    </div>
  )
}
