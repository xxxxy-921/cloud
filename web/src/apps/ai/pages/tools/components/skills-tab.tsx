import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  Search, Package, Pencil, Trash2, GitBranch, Upload,
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
import { SkillImportSheet } from "./skill-import-sheet"
import { SkillUploadSheet } from "./skill-upload-sheet"
import { SkillDetailSheet, type SkillItem } from "./skill-detail-sheet"

export function SkillsTab() {
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()
  const [importOpen, setImportOpen] = useState(false)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [detailItem, setDetailItem] = useState<SkillItem | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)

  const canCreate = usePermission("ai:skill:create")
  const canUpdate = usePermission("ai:skill:update")
  const canDelete = usePermission("ai:skill:delete")

  const {
    keyword, setKeyword, page, setPage,
    items: skills, total, totalPages, isLoading, handleSearch,
  } = useListPage<SkillItem>({
    queryKey: "ai-skills",
    endpoint: "/api/v1/ai/skills",
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/v1/ai/skills/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-skills"] })
      toast.success(t("ai:tools.skills.deleteSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const toggleMutation = useMutation({
    mutationFn: ({ id, isActive }: { id: number; isActive: boolean }) =>
      api.patch(`/api/v1/ai/skills/${id}/active`, { isActive }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-skills"] })
    },
    onError: (err) => toast.error(err.message),
  })

  function handleDetail(item: SkillItem) {
    setDetailItem(item)
    setDetailOpen(true)
  }

  const colSpan = 7

  return (
    <div className="space-y-4">
      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={t("ai:tools.skills.searchPlaceholder")}
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
              <>
                <Button size="sm" variant="outline" onClick={() => setImportOpen(true)}>
                  <GitBranch className="mr-1.5 h-4 w-4" />
                  {t("ai:tools.skills.importGitHub")}
                </Button>
                <Button size="sm" onClick={() => setUploadOpen(true)}>
                  <Upload className="mr-1.5 h-4 w-4" />
                  {t("ai:tools.skills.upload")}
                </Button>
              </>
            )}
          </form>
        </DataTableToolbarGroup>
      </DataTableToolbar>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-[160px]">{t("ai:tools.skills.displayName")}</TableHead>
              <TableHead className="w-[90px]">{t("ai:tools.skills.sourceType")}</TableHead>
              <TableHead className="w-[80px]">{t("ai:tools.skills.toolCount")}</TableHead>
              <TableHead className="w-[90px]">{t("ai:tools.skills.hasInstructions")}</TableHead>
              <TableHead className="w-[100px]">{t("ai:tools.skills.authType")}</TableHead>
              <TableHead className="w-[150px]">{t("common:createdAt")}</TableHead>
              <TableHead className="w-[180px] text-right">{t("common:actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={colSpan} />
            ) : skills.length === 0 ? (
              <DataTableEmptyRow
                colSpan={colSpan}
                icon={Package}
                title={t("ai:tools.skills.empty")}
                description={canCreate ? t("ai:tools.skills.emptyHint") : undefined}
              />
            ) : (
              skills.map((item) => (
                <TableRow key={item.id}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Switch
                        checked={item.isActive}
                        disabled={!canUpdate || toggleMutation.isPending}
                        size="sm"
                        onCheckedChange={(checked) =>
                          toggleMutation.mutate({ id: item.id, isActive: checked })
                        }
                      />
                      <div>
                        <span className="font-medium">{item.displayName}</span>
                        <p className="text-xs text-muted-foreground font-mono">{item.name}</p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">
                      {t(`ai:tools.skills.sourceTypes.${item.sourceType}`, item.sourceType)}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm">{item.toolCount}</TableCell>
                  <TableCell className="text-sm">
                    {item.hasInstructions ? t("ai:tools.skills.yes") : t("ai:tools.skills.no")}
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
                      {canUpdate && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="px-2"
                          onClick={() => handleDetail(item)}
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
                              <AlertDialogTitle>{t("ai:tools.skills.deleteTitle")}</AlertDialogTitle>
                              <AlertDialogDescription>
                                {t("ai:tools.skills.deleteDesc", { name: item.displayName })}
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => deleteMutation.mutate(item.id)}
                                disabled={deleteMutation.isPending}
                              >
                                {t("ai:tools.skills.confirmDelete")}
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

      <SkillImportSheet open={importOpen} onOpenChange={setImportOpen} />
      <SkillUploadSheet open={uploadOpen} onOpenChange={setUploadOpen} />
      <SkillDetailSheet open={detailOpen} onOpenChange={setDetailOpen} skill={detailItem} />
    </div>
  )
}
