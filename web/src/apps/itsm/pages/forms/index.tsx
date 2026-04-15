"use client"

import { useState, useEffect } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, Pencil, Trash2, FileText, ExternalLink } from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  DataTableActions, DataTableActionsCell, DataTableActionsHead,
  DataTableCard, DataTableEmptyRow, DataTableLoadingRow,
} from "@/components/ui/data-table"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader,
  AlertDialogTitle, AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import {
  Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetFooter,
} from "@/components/ui/sheet"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  Pagination, PaginationContent, PaginationItem,
  PaginationPrevious, PaginationNext,
} from "@/components/ui/pagination"
import {
  type FormDefItem,
  createFormDef, updateFormDef, deleteFormDef,
} from "../../api"

// ─── Form Schema ────────────────────────────────────────

function useFormDefSchema() {
  const { t } = useTranslation("itsm")
  return z.object({
    name: z.string().min(1, t("validation.nameRequired")),
    code: z.string().min(1, t("validation.codeRequired")),
    description: z.string().optional(),
  })
}

type FormDefFormValues = z.infer<ReturnType<typeof useFormDefSchema>>

const DEFAULT_SCHEMA = JSON.stringify({
  version: 1,
  fields: [],
  layout: null,
}, null, 2)

// ─── Main Page ──────────────────────────────────────────

export function Component() {
  const { t } = useTranslation(["itsm", "common"])
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<FormDefItem | null>(null)
  const formDefSchema = useFormDefSchema()

  const canCreate = usePermission("itsm:form:create")
  const canUpdate = usePermission("itsm:form:update")
  const canDelete = usePermission("itsm:form:delete")

  const {
    keyword, setKeyword, handleSearch,
    items, total, totalPages,
    page, setPage, isLoading,
  } = useListPage<FormDefItem>({
    queryKey: "itsm-forms",
    endpoint: "/api/v1/itsm/forms",
  })

  const form = useForm<FormDefFormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(formDefSchema as any),
    defaultValues: { name: "", code: "", description: "" },
  })

  useEffect(() => {
    if (formOpen) {
      if (editing) {
        form.reset({ name: editing.name, code: editing.code, description: editing.description })
      } else {
        form.reset({ name: "", code: "", description: "" })
      }
    }
  }, [formOpen, editing, form])

  const createMut = useMutation({
    mutationFn: (v: FormDefFormValues) => createFormDef({ ...v, description: v.description ?? "", schema: DEFAULT_SCHEMA }),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-forms"] }); setFormOpen(false); toast.success(t("itsm:forms.createSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  const updateMut = useMutation({
    mutationFn: (v: FormDefFormValues) => updateFormDef(editing!.id, { ...v, description: v.description ?? "" }),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-forms"] }); setFormOpen(false); toast.success(t("itsm:forms.updateSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  const deleteMut = useMutation({
    mutationFn: (id: number) => deleteFormDef(id),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-forms"] }); toast.success(t("itsm:forms.deleteSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  function onSubmit(v: FormDefFormValues) { if (editing) { updateMut.mutate(v) } else { createMut.mutate(v) } }
  const isPending = createMut.isPending || updateMut.isPending

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t("itsm:forms.title")}</h2>
        <div className="flex items-center gap-3">
          <form onSubmit={handleSearch} className="flex items-center gap-2">
            <Input
              className="w-60"
              placeholder={t("itsm:forms.searchPlaceholder")}
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </form>
          {canCreate && (
            <Button onClick={() => { setEditing(null); setFormOpen(true) }}>
              <Plus className="mr-1.5 h-4 w-4" />{t("itsm:forms.create")}
            </Button>
          )}
        </div>
      </div>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-[160px]">{t("itsm:forms.name")}</TableHead>
              <TableHead className="w-[180px]">{t("itsm:forms.code")}</TableHead>
              <TableHead className="w-[80px]">{t("itsm:forms.version")}</TableHead>
              <TableHead className="w-[80px]">{t("itsm:forms.scope")}</TableHead>
              <TableHead className="w-[80px]">{t("common:status")}</TableHead>
              <DataTableActionsHead className="min-w-[180px]">{t("common:actions")}</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={6} />
            ) : items.length === 0 ? (
              <DataTableEmptyRow colSpan={6} icon={FileText} title={t("itsm:forms.empty")} description={canCreate ? t("itsm:forms.emptyHint") : undefined} />
            ) : (
              items.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-medium">{item.name}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">{item.code}</TableCell>
                  <TableCell className="text-sm">v{item.version}</TableCell>
                  <TableCell>
                    <Badge variant="outline">
                      {item.scope === "service" ? t("itsm:forms.scopeService") : t("itsm:forms.scopeGlobal")}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={item.isActive ? "default" : "secondary"}>
                      {item.isActive ? t("itsm:forms.active") : t("itsm:forms.inactive")}
                    </Badge>
                  </TableCell>
                  <DataTableActionsCell>
                    <DataTableActions>
                      {canUpdate && (
                        <Button variant="ghost" size="sm" className="px-2.5" onClick={() => navigate(`/itsm/forms/${item.id}`)}>
                          <ExternalLink className="mr-1 h-3.5 w-3.5" />{t("itsm:forms.design")}
                        </Button>
                      )}
                      {canUpdate && (
                        <Button variant="ghost" size="sm" className="px-2.5" onClick={() => { setEditing(item); setFormOpen(true) }}>
                          <Pencil className="mr-1 h-3.5 w-3.5" />{t("common:edit")}
                        </Button>
                      )}
                      {canDelete && (
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button variant="ghost" size="sm" className="px-2.5 text-destructive hover:text-destructive">
                              <Trash2 className="mr-1 h-3.5 w-3.5" />{t("common:delete")}
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>{t("itsm:forms.deleteTitle")}</AlertDialogTitle>
                              <AlertDialogDescription>{t("itsm:forms.deleteDesc", { name: item.name })}</AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel size="sm">{t("common:cancel")}</AlertDialogCancel>
                              <AlertDialogAction size="sm" onClick={() => deleteMut.mutate(item.id)} disabled={deleteMut.isPending}>{t("itsm:forms.confirmDelete")}</AlertDialogAction>
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

      {totalPages > 1 && (
        <Pagination>
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious onClick={() => setPage(Math.max(1, page - 1))} aria-disabled={page <= 1} />
            </PaginationItem>
            <PaginationItem>
              <span className="text-sm text-muted-foreground px-2">
                {page} / {totalPages} ({total})
              </span>
            </PaginationItem>
            <PaginationItem>
              <PaginationNext onClick={() => setPage(Math.min(totalPages, page + 1))} aria-disabled={page >= totalPages} />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      )}

      <Sheet open={formOpen} onOpenChange={setFormOpen}>
        <SheetContent className="sm:max-w-md">
          <SheetHeader>
            <SheetTitle>{editing ? t("itsm:forms.edit") : t("itsm:forms.create")}</SheetTitle>
            <SheetDescription className="sr-only">{editing ? t("itsm:forms.edit") : t("itsm:forms.create")}</SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-5 px-4">
              <FormField control={form.control} name="name" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:forms.name")}</FormLabel>
                  <FormControl><Input placeholder={t("itsm:forms.namePlaceholder")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="code" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:forms.code")}</FormLabel>
                  <FormControl><Input placeholder={t("itsm:forms.codePlaceholder")} {...field} disabled={!!editing} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="description" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:forms.description")}</FormLabel>
                  <FormControl><Input placeholder={t("itsm:forms.description")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <SheetFooter>
                <Button type="submit" size="sm" disabled={isPending}>
                  {isPending ? t("common:saving") : editing ? t("common:save") : t("common:create")}
                </Button>
              </SheetFooter>
            </form>
          </Form>
        </SheetContent>
      </Sheet>
    </div>
  )
}
