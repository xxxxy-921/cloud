"use client"

import { useState, useEffect } from "react"
import { useTranslation } from "react-i18next"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, Pencil, Trash2, Flag } from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"
import {
  DataTableActions,
  DataTableActionsCell,
  DataTableActionsHead,
  DataTableCard,
  DataTableEmptyRow,
  DataTableLoadingRow,
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
  type PriorityItem, fetchPriorities, createPriority, updatePriority, deletePriority,
} from "../../api"

function usePrioritySchema() {
  const { t } = useTranslation("itsm")
  return z.object({
    name: z.string().min(1, t("validation.nameRequired")),
    code: z.string().min(1, t("validation.codeRequired")),
    value: z.number().min(0),
    color: z.string().min(1),
    description: z.string().optional(),
    defaultResponseMinutes: z.number().min(0),
    defaultResolutionMinutes: z.number().min(0),
  })
}

type FormValues = z.infer<ReturnType<typeof usePrioritySchema>>

export function Component() {
  const { t } = useTranslation(["itsm", "common"])
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<PriorityItem | null>(null)
  const schema = usePrioritySchema()

  const canCreate = usePermission("itsm:priority:create")
  const canUpdate = usePermission("itsm:priority:update")
  const canDelete = usePermission("itsm:priority:delete")

  const { data: items = [], isLoading } = useQuery({
    queryKey: ["itsm-priorities"],
    queryFn: () => fetchPriorities(),
  })

  const form = useForm<FormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(schema as any),
    defaultValues: { name: "", code: "", value: 0, color: "#000000", description: "", defaultResponseMinutes: 0, defaultResolutionMinutes: 0 },
  })

  useEffect(() => {
    if (formOpen) {
      if (editing) {
        form.reset({
          name: editing.name,
          code: editing.code,
          value: editing.value,
          color: editing.color,
          description: editing.description,
          defaultResponseMinutes: editing.defaultResponseMinutes,
          defaultResolutionMinutes: editing.defaultResolutionMinutes,
        })
      } else {
        form.reset({ name: "", code: "", value: 0, color: "#000000", description: "", defaultResponseMinutes: 0, defaultResolutionMinutes: 0 })
      }
    }
  }, [formOpen, editing, form])

  const createMut = useMutation({
    mutationFn: (v: FormValues) => createPriority({ ...v, description: v.description ?? "" }),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-priorities"] }); setFormOpen(false); toast.success(t("itsm:priorities.createSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  const updateMut = useMutation({
    mutationFn: (v: FormValues) => updatePriority(editing!.id, { ...v, description: v.description ?? "" }),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-priorities"] }); setFormOpen(false); toast.success(t("itsm:priorities.updateSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  const deleteMut = useMutation({
    mutationFn: (id: number) => deletePriority(id),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-priorities"] }); toast.success(t("itsm:priorities.deleteSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  function handleCreate() { setEditing(null); setFormOpen(true) }
  function handleEdit(item: PriorityItem) { setEditing(item); setFormOpen(true) }
  function onSubmit(values: FormValues) { if (editing) { updateMut.mutate(values) } else { createMut.mutate(values) } }
  const isPending = createMut.isPending || updateMut.isPending

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t("itsm:priorities.title")}</h2>
        {canCreate && (
          <Button onClick={handleCreate}>
            <Plus className="mr-1.5 h-4 w-4" />
            {t("itsm:priorities.create")}
          </Button>
        )}
      </div>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-[140px]">{t("itsm:priorities.name")}</TableHead>
              <TableHead className="w-[80px]">{t("itsm:priorities.code")}</TableHead>
              <TableHead className="w-[80px]">{t("itsm:priorities.value")}</TableHead>
              <TableHead className="w-[120px]">{t("itsm:priorities.defaultResponseMinutes")}</TableHead>
              <TableHead className="w-[120px]">{t("itsm:priorities.defaultResolutionMinutes")}</TableHead>
              <TableHead className="w-[80px]">{t("common:status")}</TableHead>
              <DataTableActionsHead className="min-w-[140px]">{t("common:actions")}</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={7} />
            ) : items.length === 0 ? (
              <DataTableEmptyRow colSpan={7} icon={Flag} title={t("itsm:priorities.empty")} description={canCreate ? t("itsm:priorities.emptyHint") : undefined} />
            ) : (
              items.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-medium">
                    <span className="mr-2 inline-block h-3 w-3 rounded-full" style={{ backgroundColor: item.color }} />
                    {item.name}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">{item.code}</TableCell>
                  <TableCell className="text-sm">{item.value}</TableCell>
                  <TableCell className="text-sm">{item.defaultResponseMinutes} min</TableCell>
                  <TableCell className="text-sm">{item.defaultResolutionMinutes} min</TableCell>
                  <TableCell>
                    <Badge variant={item.isActive ? "default" : "secondary"}>
                      {item.isActive ? t("itsm:priorities.active") : t("itsm:priorities.inactive")}
                    </Badge>
                  </TableCell>
                  <DataTableActionsCell>
                    <DataTableActions>
                      {canUpdate && (
                        <Button variant="ghost" size="sm" className="px-2.5" onClick={() => handleEdit(item)}>
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
                              <AlertDialogTitle>{t("itsm:priorities.deleteTitle")}</AlertDialogTitle>
                              <AlertDialogDescription>{t("itsm:priorities.deleteDesc", { name: item.name })}</AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel size="sm">{t("common:cancel")}</AlertDialogCancel>
                              <AlertDialogAction size="sm" onClick={() => deleteMut.mutate(item.id)} disabled={deleteMut.isPending}>{t("itsm:priorities.confirmDelete")}</AlertDialogAction>
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

      <Sheet open={formOpen} onOpenChange={setFormOpen}>
        <SheetContent className="sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>{editing ? t("itsm:priorities.edit") : t("itsm:priorities.create")}</SheetTitle>
            <SheetDescription className="sr-only">{editing ? t("itsm:priorities.edit") : t("itsm:priorities.create")}</SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-5 px-4">
              <FormField control={form.control} name="name" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:priorities.name")}</FormLabel>
                  <FormControl><Input placeholder={t("itsm:priorities.namePlaceholder")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="code" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:priorities.code")}</FormLabel>
                  <FormControl><Input placeholder={t("itsm:priorities.codePlaceholder")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <div className="grid grid-cols-2 gap-4">
                <FormField control={form.control} name="value" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("itsm:priorities.value")}</FormLabel>
                    <FormControl><Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={form.control} name="color" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("itsm:priorities.color")}</FormLabel>
                    <FormControl><Input type="color" {...field} className="h-9 p-1" /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <FormField control={form.control} name="defaultResponseMinutes" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("itsm:priorities.defaultResponseMinutes")}</FormLabel>
                    <FormControl><Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={form.control} name="defaultResolutionMinutes" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("itsm:priorities.defaultResolutionMinutes")}</FormLabel>
                    <FormControl><Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
              </div>
              <FormField control={form.control} name="description" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:priorities.description")}</FormLabel>
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
