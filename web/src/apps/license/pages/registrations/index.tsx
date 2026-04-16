import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Plus, Search, Ticket, Loader2 } from "lucide-react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { formatDateTime } from "@/lib/utils"

interface RegistrationItem {
  id: number
  code: string
  source: string
  expiresAt: string | null
  boundLicenseId: number | null
  productId: number | null
  licenseeId: number | null
  createdAt: string
}

export function Component() {
  const { t } = useTranslation(["license", "common"])
  const queryClient = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [newCode, setNewCode] = useState("")
  const [newExpiresAt, setNewExpiresAt] = useState("")

  const canCreate = usePermission("license:registration:create")
  const canGenerate = usePermission("license:registration:generate")

  const {
    keyword, setKeyword, page, setPage,
    items: registrations, total, totalPages, isLoading, handleSearch,
  } = useListPage<RegistrationItem>({
    queryKey: "license-registrations",
    endpoint: "/api/v1/license/registrations",
  })

  const createMutation = useMutation({
    mutationFn: () =>
      api.post("/api/v1/license/registrations", {
        code: newCode,
        expiresAt: newExpiresAt || null,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-registrations"] })
      setCreateOpen(false)
      setNewCode("")
      setNewExpiresAt("")
      toast.success(t("license:licensees.createSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const generateMutation = useMutation({
    mutationFn: () => api.post("/api/v1/license/registrations/generate", {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-registrations"] })
      toast.success(t("license:registrations.generateSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t("license:registrations.title")}</h2>
        <div className="flex gap-2">
          {canGenerate && (
            <Button size="sm" variant="outline" onClick={() => generateMutation.mutate()} disabled={generateMutation.isPending}>
              {generateMutation.isPending ? <Loader2 className="mr-1.5 h-4 w-4 animate-spin" /> : <Plus className="mr-1.5 h-4 w-4" />}
              {t("license:registrations.generate")}
            </Button>
          )}
          {canCreate && (
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1.5 h-4 w-4" />
              {t("license:registrations.create")}
            </Button>
          )}
        </div>
      </div>

      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={t("license:licensees.searchPlaceholder")}
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                className="pl-8"
              />
            </div>
            <Button type="submit" variant="outline">
              {t("common:search")}
            </Button>
          </form>
        </DataTableToolbarGroup>
      </DataTableToolbar>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-[180px]">{t("license:registrations.code")}</TableHead>
              <TableHead className="w-[100px]">{t("license:registrations.source")}</TableHead>
              <TableHead className="w-[100px]">{t("license:registrations.status")}</TableHead>
              <TableHead className="w-[120px]">{t("license:registrations.expiresAt")}</TableHead>
              <TableHead className="w-[150px]">{t("common:createdAt")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={5} />
            ) : registrations.length === 0 ? (
              <DataTableEmptyRow
                colSpan={5}
                icon={Ticket}
                title={t("license:registrations.empty")}
                description={canCreate ? t("license:registrations.emptyHint") : undefined}
              />
            ) : (
              registrations.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-mono text-xs">{item.code}</TableCell>
                  <TableCell className="text-sm capitalize">{item.source}</TableCell>
                  <TableCell>
                    <Badge variant={item.boundLicenseId ? "outline" : "default"} className="text-[10px]">
                      {item.boundLicenseId ? t("license:registrations.bound") : t("license:registrations.unbound")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                    {item.expiresAt ? formatDateTime(item.expiresAt, { dateOnly: true }) : "-"}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                    {formatDateTime(item.createdAt)}
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

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("license:registrations.create")}</DialogTitle>
            <DialogDescription>{t("license:registrations.emptyHint")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("license:registrations.code")}</label>
              <Input value={newCode} onChange={(e) => setNewCode(e.target.value)} placeholder="REG-XXXX" />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("license:registrations.expiresAt")}</label>
              <Input type="date" value={newExpiresAt} onChange={(e) => setNewExpiresAt(e.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>{t("common:cancel")}</Button>
            <Button onClick={() => createMutation.mutate()} disabled={createMutation.isPending || !newCode}>
              {createMutation.isPending ? t("common:processing") : t("common:create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
