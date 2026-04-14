"use client"

import { useState, useEffect, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, Pencil, Trash2, FolderTree, ChevronRight } from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
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
  type CatalogItem, fetchCatalogTree, createCatalog, updateCatalog, deleteCatalog,
} from "../../api"

function useCatalogSchema() {
  const { t } = useTranslation("itsm")
  return z.object({
    name: z.string().min(1, t("validation.nameRequired")),
    parentId: z.number().nullable(),
    sortOrder: z.number().default(0),
    description: z.string().optional(),
  })
}

type FormValues = z.infer<ReturnType<typeof useCatalogSchema>>

function flattenTree(nodes: CatalogItem[], depth = 0): Array<CatalogItem & { depth: number }> {
  const result: Array<CatalogItem & { depth: number }> = []
  for (const node of nodes) {
    result.push({ ...node, depth })
    if (node.children?.length) {
      result.push(...flattenTree(node.children, depth + 1))
    }
  }
  return result
}

function collectIds(nodes: CatalogItem[]): Set<number> {
  const ids = new Set<number>()
  for (const n of nodes) {
    ids.add(n.id)
    if (n.children?.length) {
      for (const id of collectIds(n.children)) ids.add(id)
    }
  }
  return ids
}

export function Component() {
  const { t } = useTranslation(["itsm", "common"])
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<CatalogItem | null>(null)
  const [expandOverride, setExpandOverride] = useState<Set<number> | null>(null)
  const [keyword, setKeyword] = useState("")
  const schema = useCatalogSchema()

  const canCreate = usePermission("itsm:catalog:create")
  const canUpdate = usePermission("itsm:catalog:update")
  const canDelete = usePermission("itsm:catalog:delete")

  const { data: tree = [], isLoading } = useQuery({
    queryKey: ["itsm-catalogs"],
    queryFn: () => fetchCatalogTree(),
  })

  // Default expand all — computed from tree, overridden by manual toggles
  const defaultExpanded = useMemo(() => collectIds(tree), [tree])
  const expanded = expandOverride ?? defaultExpanded

  const flat = useMemo(() => flattenTree(tree), [tree])

  const filteredFlat = useMemo(() => {
    if (!keyword.trim()) return null
    const kw = keyword.toLowerCase()
    return flat.filter((n) => n.name.toLowerCase().includes(kw))
  }, [keyword, flat])

  const form = useForm<FormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(schema as any),
    defaultValues: { name: "", parentId: null, sortOrder: 0, description: "" },
  })

  useEffect(() => {
    if (formOpen) {
      if (editing) {
        form.reset({ name: editing.name, parentId: editing.parentId, sortOrder: editing.sortOrder, description: editing.description })
      } else {
        form.reset({ name: "", parentId: null, sortOrder: 0, description: "" })
      }
    }
  }, [formOpen, editing, form])

  const createMut = useMutation({
    mutationFn: (v: FormValues) => createCatalog({ name: v.name, parentId: v.parentId, sortOrder: v.sortOrder, description: v.description }),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-catalogs"] }); setFormOpen(false); toast.success(t("itsm:catalogs.createSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  const updateMut = useMutation({
    mutationFn: (v: FormValues) => updateCatalog(editing!.id, { name: v.name, parentId: v.parentId, sortOrder: v.sortOrder, description: v.description }),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-catalogs"] }); setFormOpen(false); toast.success(t("itsm:catalogs.updateSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  const deleteMut = useMutation({
    mutationFn: (id: number) => deleteCatalog(id),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-catalogs"] }); toast.success(t("itsm:catalogs.deleteSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  function onSubmit(v: FormValues) { if (editing) { updateMut.mutate(v) } else { createMut.mutate(v) } }
  const isPending = createMut.isPending || updateMut.isPending

  function toggleExpand(id: number) {
    const base = expandOverride ?? defaultExpanded
    const next = new Set(base)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    setExpandOverride(next)
  }

  function renderRows(nodes: CatalogItem[], depth: number): React.ReactNode[] {
    const rows: React.ReactNode[] = []
    for (const node of nodes) {
      const hasChildren = (node.children?.length ?? 0) > 0
      const isExpanded = expanded.has(node.id)
      rows.push(
        <TableRow key={node.id}>
          <TableCell style={{ paddingLeft: depth * 28 + 12 }} className="font-medium">
            <span className="inline-flex items-center gap-1.5">
              {hasChildren ? (
                <button type="button" onClick={() => toggleExpand(node.id)} className="p-0.5 rounded hover:bg-accent">
                  <ChevronRight className={cn("h-4 w-4 transition-transform", isExpanded && "rotate-90")} />
                </button>
              ) : (
                <span className="w-5" />
              )}
              {node.name}
            </span>
          </TableCell>
          <TableCell className="text-sm">{node.sortOrder}</TableCell>
          <TableCell>
            <Badge variant={node.isActive ? "default" : "secondary"}>
              {node.isActive ? t("itsm:catalogs.active") : t("itsm:catalogs.inactive")}
            </Badge>
          </TableCell>
          <DataTableActionsCell>
            <DataTableActions>
              {canUpdate && (
                <Button variant="ghost" size="sm" className="px-2.5" onClick={() => { setEditing(node); setFormOpen(true) }}>
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
                      <AlertDialogTitle>{t("itsm:catalogs.deleteTitle")}</AlertDialogTitle>
                      <AlertDialogDescription>{t("itsm:catalogs.deleteDesc", { name: node.name })}</AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel size="sm">{t("common:cancel")}</AlertDialogCancel>
                      <AlertDialogAction size="sm" onClick={() => deleteMut.mutate(node.id)} disabled={deleteMut.isPending}>{t("itsm:catalogs.confirmDelete")}</AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              )}
            </DataTableActions>
          </DataTableActionsCell>
        </TableRow>,
      )
      if (hasChildren && isExpanded) {
        rows.push(...renderRows(node.children!, depth + 1))
      }
    }
    return rows
  }

  // Flat parent options for the Sheet select
  const parentOptions = useMemo(() => flattenTree(tree), [tree])

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t("itsm:catalogs.title")}</h2>
        {canCreate && (
          <Button onClick={() => { setEditing(null); setFormOpen(true) }}>
            <Plus className="mr-1.5 h-4 w-4" />{t("itsm:catalogs.create")}
          </Button>
        )}
      </div>

      <div className="relative w-full sm:max-w-sm">
        <Input
          placeholder={t("itsm:catalogs.searchPlaceholder")}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
      </div>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-[240px]">{t("itsm:catalogs.name")}</TableHead>
              <TableHead className="w-[80px]">{t("itsm:catalogs.sort")}</TableHead>
              <TableHead className="w-[100px]">{t("common:status")}</TableHead>
              <DataTableActionsHead className="min-w-[140px]">{t("common:actions")}</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={4} />
            ) : tree.length === 0 ? (
              <DataTableEmptyRow colSpan={4} icon={FolderTree} title={t("itsm:catalogs.empty")} description={canCreate ? t("itsm:catalogs.emptyHint") : undefined} />
            ) : filteredFlat ? (
              filteredFlat.length === 0 ? (
                <DataTableEmptyRow colSpan={4} icon={FolderTree} title={t("itsm:catalogs.empty")} />
              ) : (
                filteredFlat.map((node) => (
                  <TableRow key={node.id}>
                    <TableCell className="font-medium">{node.name}</TableCell>
                    <TableCell className="text-sm">{node.sortOrder}</TableCell>
                    <TableCell>
                      <Badge variant={node.isActive ? "default" : "secondary"}>
                        {node.isActive ? t("itsm:catalogs.active") : t("itsm:catalogs.inactive")}
                      </Badge>
                    </TableCell>
                    <DataTableActionsCell>
                      <DataTableActions>
                        {canUpdate && (
                          <Button variant="ghost" size="sm" className="px-2.5" onClick={() => { setEditing(node); setFormOpen(true) }}>
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
                                <AlertDialogTitle>{t("itsm:catalogs.deleteTitle")}</AlertDialogTitle>
                                <AlertDialogDescription>{t("itsm:catalogs.deleteDesc", { name: node.name })}</AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel size="sm">{t("common:cancel")}</AlertDialogCancel>
                                <AlertDialogAction size="sm" onClick={() => deleteMut.mutate(node.id)} disabled={deleteMut.isPending}>{t("itsm:catalogs.confirmDelete")}</AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        )}
                      </DataTableActions>
                    </DataTableActionsCell>
                  </TableRow>
                ))
              )
            ) : (
              renderRows(tree, 0)
            )}
          </TableBody>
        </Table>
      </DataTableCard>

      <Sheet open={formOpen} onOpenChange={setFormOpen}>
        <SheetContent className="sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>{editing ? t("itsm:catalogs.edit") : t("itsm:catalogs.create")}</SheetTitle>
            <SheetDescription className="sr-only">{editing ? t("itsm:catalogs.edit") : t("itsm:catalogs.create")}</SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-5 px-4">
              <FormField control={form.control} name="name" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:catalogs.name")}</FormLabel>
                  <FormControl><Input placeholder={t("itsm:catalogs.namePlaceholder")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="parentId" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:catalogs.parent")}</FormLabel>
                  <Select onValueChange={(v) => field.onChange(v === "0" ? null : Number(v))} value={String(field.value ?? 0)}>
                    <FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl>
                    <SelectContent>
                      <SelectItem value="0">{t("itsm:catalogs.topCatalog")}</SelectItem>
                      {parentOptions.filter((o) => o.id !== editing?.id).map((o) => (
                        <SelectItem key={o.id} value={String(o.id)}>
                          {"─".repeat(o.depth)} {o.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="sortOrder" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:catalogs.sort")}</FormLabel>
                  <FormControl><Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="description" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:catalogs.description")}</FormLabel>
                  <FormControl><Textarea rows={3} {...field} /></FormControl>
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
