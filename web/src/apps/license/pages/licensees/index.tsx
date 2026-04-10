import { useState } from "react"
import { Plus, Search, Building2, Pencil, Archive, ArchiveRestore } from "lucide-react"
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
import { LicenseeSheet, type LicenseeItem } from "../../components/licensee-sheet"

const STATUS_MAP: Record<string, { label: string; variant: "default" | "secondary" | "outline" }> = {
  active: { label: "活跃", variant: "default" },
  archived: { label: "已归档", variant: "outline" },
}

export function Component() {
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<LicenseeItem | null>(null)
  const [statusFilter, setStatusFilter] = useState("")
  const [archiveTarget, setArchiveTarget] = useState<LicenseeItem | null>(null)

  const canCreate = usePermission("license:licensee:create")
  const canUpdate = usePermission("license:licensee:update")
  const canArchive = usePermission("license:licensee:archive")

  const {
    keyword, setKeyword, page, setPage,
    items: licensees, total, totalPages, isLoading, handleSearch,
  } = useListPage<LicenseeItem>({
    queryKey: "license-licensees",
    endpoint: "/api/v1/license/licensees",
    extraParams: statusFilter ? { status: statusFilter } : undefined,
  })

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) =>
      api.patch(`/api/v1/license/licensees/${id}/status`, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-licensees"] })
      setArchiveTarget(null)
      toast.success("状态更新成功")
    },
    onError: (err) => toast.error(err.message),
  })

  function handleCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function handleEdit(item: LicenseeItem) {
    setEditing(item)
    setFormOpen(true)
  }

  function handleStatusFilter(value: string) {
    setStatusFilter(value === "all" ? "" : value)
    setPage(1)
  }

  function handleArchive(item: LicenseeItem) {
    setArchiveTarget(item)
  }

  function confirmArchive() {
    if (!archiveTarget) return
    const newStatus = archiveTarget.status === "active" ? "archived" : "active"
    statusMutation.mutate({ id: archiveTarget.id, status: newStatus })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">授权主体</h2>
        {canCreate && (
          <Button size="sm" onClick={handleCreate}>
            <Plus className="mr-1.5 h-4 w-4" />
            新增授权主体
          </Button>
        )}
      </div>

      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索名称或代码"
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
                <SelectItem value="active">活跃</SelectItem>
                <SelectItem value="archived">已归档</SelectItem>
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
              <TableHead className="min-w-[180px]">名称</TableHead>
              <TableHead className="w-[120px]">联系人</TableHead>
              <TableHead className="w-[80px]">状态</TableHead>
              <TableHead className="w-[150px]">创建时间</TableHead>
              <DataTableActionsHead className="min-w-[140px]">操作</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={5} />
            ) : licensees.length === 0 ? (
              <DataTableEmptyRow
                colSpan={5}
                icon={Building2}
                title="暂无授权主体"
                description={canCreate ? "点击「新增授权主体」创建第一个授权主体" : undefined}
              />
            ) : (
              licensees.map((item) => {
                const status = STATUS_MAP[item.status] ?? { label: item.status, variant: "secondary" as const }
                return (
                  <TableRow key={item.id}>
                    <TableCell className="font-medium">{item.name}</TableCell>
                    <TableCell className="text-sm">{item.contactName || "-"}</TableCell>
                    <TableCell>
                      <Badge variant={status.variant}>{status.label}</Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {formatDateTime(item.createdAt)}
                    </TableCell>
                    <DataTableActionsCell>
                      <DataTableActions>
                        {canUpdate && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="px-2.5"
                            onClick={() => handleEdit(item)}
                          >
                            <Pencil className="mr-1 h-3.5 w-3.5" />
                            编辑
                          </Button>
                        )}
                        {canArchive && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="px-2.5"
                            onClick={() => handleArchive(item)}
                          >
                            {item.status === "active" ? (
                              <>
                                <Archive className="mr-1 h-3.5 w-3.5" />
                                归档
                              </>
                            ) : (
                              <>
                                <ArchiveRestore className="mr-1 h-3.5 w-3.5" />
                                恢复
                              </>
                            )}
                          </Button>
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

      <LicenseeSheet open={formOpen} onOpenChange={setFormOpen} licensee={editing} />

      <AlertDialog open={archiveTarget !== null} onOpenChange={(open) => { if (!open) setArchiveTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {archiveTarget?.status === "active" ? "归档授权主体" : "恢复授权主体"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {archiveTarget?.status === "active"
                ? `确定要归档「${archiveTarget?.name}」吗？归档后将不会出现在默认列表中。`
                : `确定要恢复「${archiveTarget?.name}」吗？`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={confirmArchive} disabled={statusMutation.isPending}>
              {statusMutation.isPending ? "处理中..." : "确定"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
