import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, Pencil, Trash2, Fingerprint, Plug, Loader2 } from "lucide-react"
import { api } from "@/lib/api"
import { usePermission } from "@/hooks/use-permission"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import {
  DataTableActions,
  DataTableActionsCell,
  DataTableActionsHead,
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
import { formatDateTime } from "@/lib/utils"
import { IdentitySourceSheet, type IdentitySourceItem } from "./identity-source-sheet"

const TYPE_LABELS: Record<string, string> = {
  oidc: "OIDC",
  ldap: "LDAP",
}

export function Component() {
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<IdentitySourceItem | null>(null)

  const canCreate = usePermission("system:identity-source:create")
  const canUpdate = usePermission("system:identity-source:update")
  const canDelete = usePermission("system:identity-source:delete")

  const { data: sources = [], isLoading } = useQuery({
    queryKey: ["identity-sources"],
    queryFn: () => api.get<IdentitySourceItem[]>("/api/v1/identity-sources"),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/v1/identity-sources/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["identity-sources"] })
      toast.success("身份源已删除")
    },
    onError: (err) => toast.error(err.message),
  })

  const toggleMutation = useMutation({
    mutationFn: (id: number) => api.patch(`/api/v1/identity-sources/${id}/toggle`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["identity-sources"] })
    },
    onError: (err) => toast.error(err.message),
  })

  const testMutation = useMutation({
    mutationFn: async (id: number) => {
      const res = await api.post<{ success: boolean; message: string }>(
        `/api/v1/identity-sources/${id}/test`,
      )
      if (!res.success) throw new Error(res.message || "连接测试失败")
      return res
    },
    onSuccess: (res) => toast.success(res.message || "连接测试成功"),
    onError: (err) => toast.error(`连接测试失败: ${err.message}`),
  })

  function handleCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function handleEdit(item: IdentitySourceItem) {
    setEditing(item)
    setFormOpen(true)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">身份源管理</h2>
        {canCreate && (
          <Button size="sm" onClick={handleCreate}>
            <Plus className="mr-1.5 h-4 w-4" />
            新建身份源
          </Button>
        )}
      </div>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">ID</TableHead>
              <TableHead className="min-w-[180px]">名称</TableHead>
              <TableHead className="w-[100px]">类型</TableHead>
              <TableHead className="min-w-[220px]">域名</TableHead>
              <TableHead className="w-[100px]">状态</TableHead>
              <TableHead className="w-[150px]">创建时间</TableHead>
              <DataTableActionsHead className="min-w-[244px]">操作</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={7} />
            ) : sources.length === 0 ? (
              <DataTableEmptyRow
                colSpan={7}
                icon={Fingerprint}
                title="暂无身份源"
                description={canCreate ? "点击「新建身份源」配置第一个身份源" : undefined}
              />
            ) : (
              sources.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-mono text-sm">{item.id}</TableCell>
                  <TableCell className="font-medium">{item.name}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">
                      {TYPE_LABELS[item.type] ?? item.type}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-[320px] text-sm text-muted-foreground">
                    <span className="block truncate" title={item.domains || "-"}>
                      {item.domains || "-"}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Switch
                      checked={item.enabled}
                      disabled={!canUpdate}
                      onCheckedChange={() => toggleMutation.mutate(item.id)}
                    />
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
                          disabled={testMutation.isPending}
                          onClick={() => testMutation.mutate(item.id)}
                        >
                          {testMutation.isPending ? (
                            <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                          ) : (
                            <Plug className="mr-1 h-3.5 w-3.5" />
                          )}
                          测试连接
                        </Button>
                      )}
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
                                确定要删除身份源 &ldquo;{item.name}&rdquo; 吗？此操作不可撤销。
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

      <IdentitySourceSheet open={formOpen} onOpenChange={setFormOpen} source={editing} />
    </div>
  )
}
