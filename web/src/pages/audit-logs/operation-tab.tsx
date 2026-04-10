import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { Search, ClipboardList } from "lucide-react"
import { api, type PaginatedResponse } from "@/lib/api"
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
  resource: string
  resourceId: string
  summary: string
  level: string
  ipAddress: string
}

const actionLabels: Record<string, string> = {
  "user.create": "创建用户",
  "user.update": "更新用户",
  "user.delete": "删除用户",
  "user.reset_password": "重置密码",
  "user.activate": "启用用户",
  "user.deactivate": "禁用用户",
  "role.create": "创建角色",
  "role.update": "更新角色",
  "role.delete": "删除角色",
  "role.set_permissions": "设置权限",
  "menu.create": "创建菜单",
  "menu.update": "更新菜单",
  "menu.delete": "删除菜单",
  "menu.reorder": "调整排序",
  "settings.update": "更新设置",
  "announcement.create": "创建公告",
  "announcement.update": "更新公告",
  "announcement.delete": "删除公告",
  "channel.create": "创建通道",
  "channel.update": "更新通道",
  "channel.delete": "删除通道",
  "channel.toggle": "切换通道",
  "auth_provider.update": "更新认证源",
  "auth_provider.toggle": "切换认证源",
  "session.kick": "踢出会话",
}

const resourceTypes = [
  { value: "user", label: "用户" },
  { value: "role", label: "角色" },
  { value: "menu", label: "菜单" },
  { value: "settings", label: "设置" },
  { value: "announcement", label: "公告" },
  { value: "channel", label: "通道" },
  { value: "auth_provider", label: "认证源" },
  { value: "session", label: "会话" },
]

export function OperationTab() {
  const [keyword, setKeyword] = useState("")
  const [searchKeyword, setSearchKeyword] = useState("")
  const [resource, setResource] = useState("")
  const [dateFrom, setDateFrom] = useState("")
  const [dateTo, setDateTo] = useState("")
  const [page, setPage] = useState(1)
  const pageSize = 20

  const { data, isLoading } = useQuery({
    queryKey: ["audit-logs", "operation", searchKeyword, resource, dateFrom, dateTo, page],
    queryFn: () => {
      const params = new URLSearchParams({
        category: "operation",
        page: String(page),
        pageSize: String(pageSize),
      })
      if (searchKeyword) params.set("keyword", searchKeyword)
      if (resource) params.set("resource", resource)
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
            placeholder="搜索摘要..."
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className="h-8 w-48"
          />
          <Button type="submit" variant="outline" size="sm">
            <Search className="mr-1 h-3.5 w-3.5" />
            搜索
          </Button>
        </form>
        <Select value={resource} onValueChange={(v) => { setResource(v === "all" ? "" : v); setPage(1) }}>
          <SelectTrigger size="sm" className="w-32">
            <SelectValue placeholder="资源类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部</SelectItem>
            {resourceTypes.map((r) => (
              <SelectItem key={r.value} value={r.value}>{r.label}</SelectItem>
            ))}
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
              <TableHead className="w-[140px]">操作者</TableHead>
              <TableHead className="w-[120px]">操作</TableHead>
              <TableHead className="w-[120px]">资源类型</TableHead>
              <TableHead className="min-w-[260px]">摘要</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={5} />
            ) : items.length === 0 ? (
              <DataTableEmptyRow colSpan={5} icon={ClipboardList} title="暂无操作记录" />
            ) : (
              items.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                    {formatDateTime(log.createdAt)}
                  </TableCell>
                  <TableCell className="font-medium">{log.username || "-"}</TableCell>
                  <TableCell>
                    <Badge variant="outline">
                      {actionLabels[log.action] ?? log.action}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm">
                    {resourceTypes.find((r) => r.value === log.resource)?.label ?? log.resource ?? "-"}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground max-w-[300px] truncate">
                    {log.summary}
                  </TableCell>
                </TableRow>
              ))
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
