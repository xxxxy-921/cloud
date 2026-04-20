import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "react-router"
import {
  Plus, Search, Network, Pencil, Trash2, RefreshCw, ExternalLink,
} from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { api } from "@/lib/api"
import { toast } from "sonner"
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
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { formatDateTime } from "@/lib/utils"
import { AssetStatusBadge } from "../_shared/asset-status-badge"
import { KgFormSheet } from "./components/kg-form-sheet"
import type { KnowledgeAsset, KnowledgeType } from "../_shared/types"

export function Component() {
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<KnowledgeAsset | null>(null)

  const canCreate = usePermission("ai:knowledge:create")
  const canUpdate = usePermission("ai:knowledge:update")
  const canDelete = usePermission("ai:knowledge:delete")
  const canBuild = usePermission("ai:knowledge:compile")

  const {
    keyword, setKeyword, page, setPage,
    items, total, totalPages, isLoading, handleSearch,
  } = useListPage<KnowledgeAsset>({
    queryKey: "ai-kg-list",
    endpoint: "/api/v1/ai/knowledge/graphs",
  })

  const { data: kgTypes = [] } = useQuery({
    queryKey: ["ai-knowledge-types-kg"],
    queryFn: () => api.get<KnowledgeType[]>("/api/v1/ai/knowledge/types?category=kg"),
  })

  const typeMap = new Map(kgTypes.map((t) => [t.type, t.displayName]))

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/v1/ai/knowledge/graphs/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-kg-list"] })
      toast.success("删除成功")
    },
    onError: (err) => toast.error(err.message),
  })

  const buildMutation = useMutation({
    mutationFn: (id: number) => api.post(`/api/v1/ai/knowledge/graphs/${id}/build`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-kg-list"] })
      toast.success("构建任务已启动")
    },
    onError: (err) => toast.error(err.message),
  })

  function handleCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function handleEdit(item: KnowledgeAsset) {
    setEditing(item)
    setFormOpen(true)
  }

  const colSpan = 8

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">知识图谱</h2>
        {canCreate && (
          <Button size="sm" onClick={handleCreate}>
            <Plus className="mr-1.5 h-4 w-4" />
            新建知识图谱
          </Button>
        )}
      </div>

      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索知识图谱..."
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                className="pl-8"
              />
            </div>
            <Button type="submit" variant="outline">搜索</Button>
          </form>
        </DataTableToolbarGroup>
      </DataTableToolbar>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-[160px]">名称</TableHead>
              <TableHead className="w-[120px]">类型</TableHead>
              <TableHead className="w-[100px]">状态</TableHead>
              <TableHead className="w-[80px]">素材数</TableHead>
              <TableHead className="w-[80px]">节点数</TableHead>
              <TableHead className="w-[80px]">边数</TableHead>
              <TableHead className="w-[150px]">创建时间</TableHead>
              <DataTableActionsHead className="min-w-[180px]">操作</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={colSpan} />
            ) : items.length === 0 ? (
              <DataTableEmptyRow
                colSpan={colSpan}
                icon={Network}
                title="暂无知识图谱"
                description={canCreate ? "点击「新建知识图谱」开始" : undefined}
              />
            ) : (
              items.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-medium">
                    <Link
                      to={`/ai/knowledge/graphs/${item.id}`}
                      className="hover:underline text-primary"
                    >
                      {item.name}
                    </Link>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {typeMap.get(item.type) ?? item.type}
                  </TableCell>
                  <TableCell>
                    <AssetStatusBadge status={item.status} />
                  </TableCell>
                  <TableCell className="text-sm">{item.sourceCount}</TableCell>
                  <TableCell className="text-sm">{item.nodeCount}</TableCell>
                  <TableCell className="text-sm">{item.edgeCount}</TableCell>
                  <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                    {formatDateTime(item.createdAt)}
                  </TableCell>
                  <DataTableActionsCell>
                    <DataTableActions>
                      <Button variant="ghost" size="sm" className="px-2" asChild>
                        <Link to={`/ai/knowledge/graphs/${item.id}`}>
                          <ExternalLink className="mr-1 h-3.5 w-3.5" />
                          查看
                        </Link>
                      </Button>
                      {canBuild && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="px-2"
                          disabled={buildMutation.isPending || item.status === "building"}
                          onClick={() => buildMutation.mutate(item.id)}
                        >
                          <RefreshCw className="mr-1 h-3.5 w-3.5" />
                          构建
                        </Button>
                      )}
                      {canUpdate && (
                        <Button variant="ghost" size="sm" className="px-2" onClick={() => handleEdit(item)}>
                          <Pencil className="mr-1 h-3.5 w-3.5" />
                          编辑
                        </Button>
                      )}
                      {canDelete && (
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button variant="ghost" size="sm" className="px-2 text-destructive hover:text-destructive">
                              <Trash2 className="mr-1 h-3.5 w-3.5" />
                              删除
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>删除知识图谱</AlertDialogTitle>
                              <AlertDialogDescription>
                                确定要删除「{item.name}」吗？此操作不可撤销。
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>取消</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => deleteMutation.mutate(item.id)}
                                disabled={deleteMutation.isPending}
                              >
                                确认删除
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

      <KgFormSheet
        open={formOpen}
        onOpenChange={setFormOpen}
        knowledgeGraph={editing}
      />
    </div>
  )
}
