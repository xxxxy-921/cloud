import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  Plus, Search, Server, Pencil, Trash2, Zap, Globe, Terminal,
} from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import {
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
import { MCPServerSheet, type MCPServerItem } from "./mcp-server-sheet"

export function MCPServersTab() {
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<MCPServerItem | null>(null)

  const canCreate = usePermission("ai:mcp:create")
  const canUpdate = usePermission("ai:mcp:update")
  const canDelete = usePermission("ai:mcp:delete")
  const canTest = usePermission("ai:mcp:test")

  const {
    keyword, setKeyword, page, setPage,
    items: servers, total, totalPages, isLoading, handleSearch,
  } = useListPage<MCPServerItem>({
    queryKey: "ai-mcp-servers",
    endpoint: "/api/v1/ai/mcp-servers",
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/v1/ai/mcp-servers/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-mcp-servers"] })
      toast.success(t("ai:tools.mcp.deleteSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const testMutation = useMutation({
    mutationFn: (id: number) =>
      api.post<{ success: boolean; error?: string; message?: string }>(`/api/v1/ai/mcp-servers/${id}/test`),
    onSuccess: (data) => {
      if (data.success) {
        toast.success(t("ai:tools.mcp.testSuccess"))
      } else {
        const msg = data.error || data.message || ""
        if (msg.toLowerCase().includes("stdio")) {
          toast.info(t("ai:tools.mcp.testStdioHint"))
        } else {
          toast.error(t("ai:tools.mcp.testFailed", { error: msg }))
        }
      }
    },
    onError: (err) => toast.error(err.message),
  })

  function handleCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function handleEdit(item: MCPServerItem) {
    setEditing(item)
    setFormOpen(true)
  }

  const colSpan = 6

  return (
    <div className="space-y-4">
      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={t("ai:tools.mcp.searchPlaceholder")}
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                className="pl-8"
              />
            </div>
            <Button type="submit" variant="outline">
              {t("common:search")}
            </Button>
            <div className="flex-1" />
            {canCreate && (
              <Button size="sm" onClick={handleCreate}>
                <Plus className="mr-1.5 h-4 w-4" />
                {t("ai:tools.mcp.create")}
              </Button>
            )}
          </form>
        </DataTableToolbarGroup>
      </DataTableToolbar>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-[160px]">{t("ai:tools.mcp.name")}</TableHead>
              <TableHead className="w-[120px]">{t("ai:tools.mcp.transport")}</TableHead>
              <TableHead className="min-w-[160px]">{t("ai:tools.mcp.url")}/{t("ai:tools.mcp.command")}</TableHead>
              <TableHead className="w-[100px]">{t("ai:tools.mcp.authType")}</TableHead>
              <TableHead className="w-[150px]">{t("common:createdAt")}</TableHead>
              <TableHead className="w-[200px] text-right">{t("common:actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={colSpan} />
            ) : servers.length === 0 ? (
              <DataTableEmptyRow
                colSpan={colSpan}
                icon={Server}
                title={t("ai:tools.mcp.empty")}
                description={canCreate ? t("ai:tools.mcp.emptyHint") : undefined}
              />
            ) : (
              servers.map((item) => (
                <TableRow key={item.id}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Switch
                        checked={item.isActive}
                        disabled={!canUpdate}
                        size="sm"
                        onCheckedChange={(checked) => {
                          api.put(`/api/v1/ai/mcp-servers/${item.id}`, {
                            name: item.name,
                            description: item.description,
                            transport: item.transport,
                            url: item.url,
                            command: item.command,
                            args: item.args,
                            env: item.env,
                            authType: item.authType,
                            isActive: checked,
                          }).then(() => {
                            queryClient.invalidateQueries({ queryKey: ["ai-mcp-servers"] })
                          })
                        }}
                      />
                      <span className="font-medium">{item.name}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="gap-1">
                      {item.transport === "sse"
                        ? <><Globe className="h-3 w-3" />{t("ai:tools.mcp.transportTypes.sse")}</>
                        : <><Terminal className="h-3 w-3" />{t("ai:tools.mcp.transportTypes.stdio")}</>}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground font-mono truncate max-w-[200px]">
                    {item.transport === "sse" ? item.url : item.command}
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary">
                      {t(`ai:tools.mcp.authTypes.${item.authType}`, item.authType)}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                    {formatDateTime(item.createdAt)}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      {canTest && item.transport === "sse" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="px-2"
                          disabled={testMutation.isPending}
                          onClick={() => testMutation.mutate(item.id)}
                        >
                          <Zap className="mr-1 h-3.5 w-3.5" />
                          {testMutation.isPending ? t("ai:tools.mcp.testing") : t("ai:tools.mcp.testConnection")}
                        </Button>
                      )}
                      {canUpdate && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="px-2"
                          onClick={() => handleEdit(item)}
                        >
                          <Pencil className="mr-1 h-3.5 w-3.5" />
                          {t("common:edit")}
                        </Button>
                      )}
                      {canDelete && (
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="px-2 text-destructive hover:text-destructive"
                            >
                              <Trash2 className="mr-1 h-3.5 w-3.5" />
                              {t("common:delete")}
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>{t("ai:tools.mcp.deleteTitle")}</AlertDialogTitle>
                              <AlertDialogDescription>
                                {t("ai:tools.mcp.deleteDesc", { name: item.name })}
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => deleteMutation.mutate(item.id)}
                                disabled={deleteMutation.isPending}
                              >
                                {t("ai:tools.mcp.confirmDelete")}
                              </AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      )}
                    </div>
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

      <MCPServerSheet open={formOpen} onOpenChange={setFormOpen} server={editing} />
    </div>
  )
}
