import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { Search, ShieldAlert } from "lucide-react"
import { api, type PaginatedResponse } from "@/lib/api"
import { parseUserAgent } from "@/lib/ua-parser"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  DataTableCard,
  DataTableEmptyRow,
  DataTableLoadingRow,
  DataTablePagination,
  DataTableToolbar,
} from "@/components/ui/data-table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { formatDateTime } from "@/lib/utils"

interface AuditLog {
  id: number
  createdAt: string
  category: string
  userId: number | null
  username: string
  action: string
  summary: string
  level: string
  ipAddress: string
  userAgent: string
  detail: string | null
}

const actionLabels: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  login_success: { label: "登录成功", variant: "default" },
  login_failed: { label: "登录失败", variant: "destructive" },
  logout: { label: "登出", variant: "secondary" },
}

export function AuthTab() {
  const [keyword, setKeyword] = useState("")
  const [searchKeyword, setSearchKeyword] = useState("")
  const [action, setAction] = useState("")
  const [dateFrom, setDateFrom] = useState("")
  const [dateTo, setDateTo] = useState("")
  const [page, setPage] = useState(1)
  const pageSize = 20

  const { data, isLoading } = useQuery({
    queryKey: ["audit-logs", "auth", searchKeyword, action, dateFrom, dateTo, page],
    queryFn: () => {
      const params = new URLSearchParams({
        category: "auth",
        page: String(page),
        pageSize: String(pageSize),
      })
      if (searchKeyword) params.set("keyword", searchKeyword)
      if (action) params.set("action", action)
      if (dateFrom) params.set("dateFrom", dateFrom)
      if (dateTo) params.set("dateTo", dateTo)
      return api.get<PaginatedResponse<AuditLog>>(`/api/v1/audit-logs?${params}`)
    },
  })

  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / pageSize)

  function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    setSearchKeyword(keyword)
    setPage(1)
  }

  return (
    <div className="space-y-4 pt-4">
      <DataTableToolbar className="flex-wrap items-center gap-2">
        <form onSubmit={handleSearch} className="flex items-center gap-2">
          <Input
            placeholder="搜索用户名..."
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className="h-8 w-48"
          />
          <Button type="submit" variant="outline">
            <Search className="mr-1 h-3.5 w-3.5" />
            搜索
          </Button>
        </form>
        <Select value={action} onValueChange={(v) => { setAction(v === "all" ? "" : v); setPage(1) }}>
          <SelectTrigger size="sm" className="w-32">
            <SelectValue placeholder="事件类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部</SelectItem>
            <SelectItem value="login_success">登录成功</SelectItem>
            <SelectItem value="login_failed">登录失败</SelectItem>
            <SelectItem value="logout">登出</SelectItem>
          </SelectContent>
        </Select>
        <Input
          type="date"
          value={dateFrom}
          onChange={(e) => { setDateFrom(e.target.value); setPage(1) }}
          className="h-8 w-36"
        />
        <span className="text-muted-foreground text-sm">至</span>
        <Input
          type="date"
          value={dateTo}
          onChange={(e) => { setDateTo(e.target.value); setPage(1) }}
          className="h-8 w-36"
        />
      </DataTableToolbar>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[150px]">时间</TableHead>
              <TableHead className="w-[140px]">用户</TableHead>
              <TableHead className="w-[120px]">事件</TableHead>
              <TableHead className="w-[140px]">IP 地址</TableHead>
              <TableHead className="min-w-[220px]">设备</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={5} />
            ) : items.length === 0 ? (
              <DataTableEmptyRow colSpan={5} icon={ShieldAlert} title="暂无登录活动记录" />
            ) : (
              items.map((log) => {
                const actionInfo = actionLabels[log.action] ?? { label: log.action, variant: "outline" as const }
                return (
                  <TableRow key={log.id}>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {formatDateTime(log.createdAt)}
                    </TableCell>
                    <TableCell className="font-medium">{log.username || "-"}</TableCell>
                    <TableCell>
                      <Badge variant={actionInfo.variant}>{actionInfo.label}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-sm">{log.ipAddress || "-"}</TableCell>
                    <TableCell className="text-sm text-muted-foreground max-w-[200px] truncate">
                      {log.userAgent ? parseUserAgent(log.userAgent) : "-"}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </DataTableCard>

      <DataTablePagination
        total={total}
        page={page}
        totalPages={totalPages}
        onPageChange={setPage}
      />
    </div>
  )
}
