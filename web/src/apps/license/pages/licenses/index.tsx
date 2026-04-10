import { useState } from "react"
import { useNavigate } from "react-router"
import { Plus, Search, FileBadge, Ban, Download, Eye } from "lucide-react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  DataTableActions,
  DataTableActionsCell,
  DataTableActionsHead,
  DataTableCard,
  DataTableEmptyRow,
  DataTableLoadingRow,
  DataTablePagination,
  DataTableToolbar,
  DataTableToolbarGroup,
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { formatDateTime } from "@/lib/utils"
import { IssueLicenseSheet } from "../../components/issue-license-sheet"

export interface LicenseItem {
  id: number
  productId: number | null
  licenseeId: number | null
  planName: string
  registrationCode: string
  status: string
  validFrom: string
  validUntil: string | null
  productName: string
  licenseeName: string
  createdAt: string
}

const STATUS_MAP: Record<string, { label: string; variant: "default" | "destructive" | "outline" }> = {
  issued: { label: "已签发", variant: "default" },
  revoked: { label: "已吊销", variant: "destructive" },
}

export function Component() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [statusFilter, setStatusFilter] = useState("")
  const [revokeTarget, setRevokeTarget] = useState<LicenseItem | null>(null)

  const canIssue = usePermission("license:license:issue")
  const canRevoke = usePermission("license:license:revoke")

  const {
    keyword, setKeyword, page, setPage,
    items: licenses, total, totalPages, isLoading, handleSearch,
  } = useListPage<LicenseItem>({
    queryKey: "license-licenses",
    endpoint: "/api/v1/license/licenses",
    extraParams: statusFilter ? { status: statusFilter } : undefined,
  })

  const revokeMutation = useMutation({
    mutationFn: (id: number) => api.patch(`/api/v1/license/licenses/${id}/revoke`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-licenses"] })
      setRevokeTarget(null)
      toast.success("许可已吊销")
    },
    onError: (err) => toast.error(err.message),
  })

  function handleStatusFilter(value: string) {
    setStatusFilter(value === "all" ? "" : value)
    setPage(1)
  }

  function handleExport(item: LicenseItem) {
    window.open(`/api/v1/license/licenses/${item.id}/export`, "_blank")
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">许可签发</h2>
        {canIssue && (
          <Button size="sm" onClick={() => setFormOpen(true)}>
            <Plus className="mr-1.5 h-4 w-4" />
            签发许可
          </Button>
        )}
      </div>

      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索套餐名或注册码"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                className="pl-8"
              />
            </div>
            <Select value={statusFilter || "all"} onValueChange={handleStatusFilter}>
              <SelectTrigger className="w-full sm:w-[130px]">
                <SelectValue placeholder="全部状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="issued">已签发</SelectItem>
                <SelectItem value="revoked">已吊销</SelectItem>
              </SelectContent>
            </Select>
            <Button type="submit" variant="outline" size="sm">
              搜索
            </Button>
          </form>
        </DataTableToolbarGroup>
      </DataTableToolbar>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-[120px]">套餐</TableHead>
              <TableHead className="min-w-[100px]">商品</TableHead>
              <TableHead className="min-w-[100px]">授权主体</TableHead>
              <TableHead className="w-[80px]">状态</TableHead>
              <TableHead className="w-[100px]">生效时间</TableHead>
              <TableHead className="w-[100px]">过期时间</TableHead>
              <TableHead className="w-[150px]">签发时间</TableHead>
              <DataTableActionsHead className="min-w-[180px]">操作</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={8} />
            ) : licenses.length === 0 ? (
              <DataTableEmptyRow
                colSpan={8}
                icon={FileBadge}
                title="暂无许可记录"
                description={canIssue ? "点击「签发许可」创建第一条许可" : undefined}
              />
            ) : (
              licenses.map((item) => {
                const status = STATUS_MAP[item.status] ?? { label: item.status, variant: "outline" as const }
                return (
                  <TableRow key={item.id}>
                    <TableCell className="font-medium">{item.planName}</TableCell>
                    <TableCell className="text-sm">{item.productName || "-"}</TableCell>
                    <TableCell className="text-sm">{item.licenseeName || "-"}</TableCell>
                    <TableCell>
                      <Badge variant={status.variant}>{status.label}</Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {item.validFrom ? new Date(item.validFrom).toLocaleDateString() : "-"}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {item.validUntil ? new Date(item.validUntil).toLocaleDateString() : "永久"}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {formatDateTime(item.createdAt)}
                    </TableCell>
                    <DataTableActionsCell>
                      <DataTableActions>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="px-2.5"
                          onClick={() => navigate(`/license/licenses/${item.id}`)}
                        >
                          <Eye className="mr-1 h-3.5 w-3.5" />
                          详情
                        </Button>
                        {item.status === "issued" && (
                          <>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="px-2.5"
                              onClick={() => handleExport(item)}
                            >
                              <Download className="mr-1 h-3.5 w-3.5" />
                              导出
                            </Button>
                            {canRevoke && (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="px-2.5 text-destructive hover:text-destructive"
                                onClick={() => setRevokeTarget(item)}
                              >
                                <Ban className="mr-1 h-3.5 w-3.5" />
                                吊销
                              </Button>
                            )}
                          </>
                        )}
                      </DataTableActions>
                    </DataTableActionsCell>
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

      <IssueLicenseSheet open={formOpen} onOpenChange={setFormOpen} />

      <AlertDialog open={revokeTarget !== null} onOpenChange={(open) => { if (!open) setRevokeTarget(null) }}>
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
              onClick={() => revokeTarget && revokeMutation.mutate(revokeTarget.id)}
              disabled={revokeMutation.isPending}
            >
              {revokeMutation.isPending ? "处理中..." : "确定吊销"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
