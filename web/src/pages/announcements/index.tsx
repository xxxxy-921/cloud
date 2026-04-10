import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, Search, Pencil, Trash2, Megaphone } from "lucide-react"
import { api } from "@/lib/api"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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
import { AnnouncementSheet } from "./announcement-sheet"

interface Announcement {
  id: number
  title: string
  content: string
  createdAt: string
  updatedAt: string
  creatorUsername: string
}

export function Component() {
  const queryClient = useQueryClient()
  const [sheetOpen, setSheetOpen] = useState(false)
  const [editing, setEditing] = useState<Announcement | null>(null)
  const canCreate = usePermission("system:announcement:create")
  const canUpdate = usePermission("system:announcement:update")
  const canDelete = usePermission("system:announcement:delete")

  const {
    keyword, setKeyword, page, setPage,
    items: announcements, total, totalPages, isLoading, handleSearch,
  } = useListPage<Announcement>({ queryKey: "announcements", endpoint: "/api/v1/announcements" })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/v1/announcements/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["announcements"] }),
  })

  function handleCreate() {
    setEditing(null)
    setSheetOpen(true)
  }

  function handleEdit(item: Announcement) {
    setEditing(item)
    setSheetOpen(true)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">公告管理</h2>
        {canCreate && (
          <Button size="sm" onClick={handleCreate}>
            <Plus className="mr-1.5 h-4 w-4" />
            新建公告
          </Button>
        )}
      </div>

      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索公告标题"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                className="pl-8"
              />
            </div>
            <Button type="submit" variant="outline">
              搜索
            </Button>
          </form>
        </DataTableToolbarGroup>
      </DataTableToolbar>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">ID</TableHead>
              <TableHead className="min-w-[240px]">标题</TableHead>
              <TableHead className="w-[140px]">发布者</TableHead>
              <TableHead className="w-[150px]">发布时间</TableHead>
              <DataTableActionsHead className="min-w-[148px]">操作</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={5} />
            ) : announcements.length === 0 ? (
              <DataTableEmptyRow
                colSpan={5}
                icon={Megaphone}
                title="暂无公告"
                description={canCreate ? "点击「新建公告」发布第一条公告" : undefined}
              />
            ) : (
              announcements.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-mono text-sm">{item.id}</TableCell>
                  <TableCell className="max-w-[360px] font-medium">
                    <span className="block truncate" title={item.title}>
                      {item.title}
                    </span>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                    {item.creatorUsername || "-"}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                    {formatDateTime(item.createdAt)}
                  </TableCell>
                  <DataTableActionsCell>
                    <DataTableActions>
                      {canUpdate && (
                        <Button variant="ghost" size="sm" className="px-2.5" onClick={() => handleEdit(item)}>
                          <Pencil className="mr-1 h-3.5 w-3.5" />
                          编辑
                        </Button>
                      )}
                      {canDelete && (
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="px-2.5 text-destructive hover:text-destructive"
                            >
                              <Trash2 className="mr-1 h-3.5 w-3.5" />
                              删除
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>确认删除</AlertDialogTitle>
                              <AlertDialogDescription>
                                确定要删除公告 &ldquo;{item.title}&rdquo; 吗？此操作不可撤销。
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>取消</AlertDialogCancel>
                              <AlertDialogAction onClick={() => deleteMutation.mutate(item.id)}>
                                删除
                              </AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      )}
                    </DataTableActions>
                  </DataTableActionsCell>
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

      <AnnouncementSheet open={sheetOpen} onOpenChange={setSheetOpen} announcement={editing} />
    </div>
  )
}
