"use client"

import { useState, useEffect } from "react"
import { useTranslation } from "react-i18next"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, Pencil, Trash2, Timer, ChevronRight } from "lucide-react"
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
  type SLATemplateItem, type EscalationRuleItem,
  fetchSLATemplates, createSLATemplate, updateSLATemplate, deleteSLATemplate,
  fetchEscalationRules, createEscalationRule, updateEscalationRule, deleteEscalationRule,
} from "../../api"

// ─── SLA Form Schema ────────────────────────────────────

function useSLASchema() {
  const { t } = useTranslation("itsm")
  return z.object({
    name: z.string().min(1, t("validation.nameRequired")),
    code: z.string().min(1, t("validation.codeRequired")),
    description: z.string().optional(),
    responseMinutes: z.number().min(1),
    resolutionMinutes: z.number().min(1),
  })
}

type SLAFormValues = z.infer<ReturnType<typeof useSLASchema>>

// ─── Escalation Form Schema ─────────────────────────────

function useEscalationSchema() {
  return z.object({
    triggerType: z.enum(["response_timeout", "resolution_timeout"]),
    level: z.number().min(1),
    waitMinutes: z.number().min(1),
    actionType: z.enum(["notify", "reassign", "escalate_priority"]),
  })
}

type EscalationFormValues = z.infer<ReturnType<typeof useEscalationSchema>>

// ─── Escalation Rules Sub-Table ─────────────────────────

function EscalationRules({ slaId }: { slaId: number }) {
  const { t } = useTranslation(["itsm", "common"])
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<EscalationRuleItem | null>(null)
  const canUpdate = usePermission("itsm:sla:update")
  const canDelete = usePermission("itsm:sla:delete")
  const escSchema = useEscalationSchema()

  const { data: rules = [], isLoading } = useQuery({
    queryKey: ["itsm-escalation-rules", slaId],
    queryFn: () => fetchEscalationRules(slaId),
  })

  const form = useForm<EscalationFormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(escSchema as any),
    defaultValues: { triggerType: "response_timeout", level: 1, waitMinutes: 30, actionType: "notify" },
  })

  useEffect(() => {
    if (formOpen) {
      if (editing) {
        form.reset({ triggerType: editing.triggerType as "response_timeout" | "resolution_timeout", level: editing.level, waitMinutes: editing.waitMinutes, actionType: editing.actionType as "notify" | "reassign" | "escalate_priority" })
      } else {
        form.reset({ triggerType: "response_timeout", level: 1, waitMinutes: 30, actionType: "notify" })
      }
    }
  }, [formOpen, editing, form])

  const createMut = useMutation({
    mutationFn: (v: EscalationFormValues) => createEscalationRule(slaId, v),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-escalation-rules", slaId] }); setFormOpen(false); toast.success(t("itsm:sla.escalation.createSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  const updateMut = useMutation({
    mutationFn: (v: EscalationFormValues) => updateEscalationRule(slaId, editing!.id, v),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-escalation-rules", slaId] }); setFormOpen(false); toast.success(t("itsm:sla.escalation.updateSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  const deleteMut = useMutation({
    mutationFn: (id: number) => deleteEscalationRule(slaId, id),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-escalation-rules", slaId] }); toast.success(t("itsm:sla.escalation.deleteSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  function onSubmit(v: EscalationFormValues) { if (editing) { updateMut.mutate(v) } else { createMut.mutate(v) } }
  const isPending = createMut.isPending || updateMut.isPending

  const triggerLabel = (v: string) => v === "response_timeout" ? t("itsm:sla.escalation.responseTimeout") : t("itsm:sla.escalation.resolutionTimeout")
  const actionLabel = (v: string) => ({ notify: t("itsm:sla.escalation.notify"), reassign: t("itsm:sla.escalation.reassign"), escalate_priority: t("itsm:sla.escalation.escalatePriority") })[v] ?? v

  return (
    <TableRow>
      <TableCell colSpan={6} className="bg-muted/30 p-4">
        <div className="flex items-center justify-between mb-3">
          <h4 className="text-sm font-medium">{t("itsm:sla.escalations")}</h4>
          {canUpdate && (
            <Button size="sm" variant="outline" onClick={() => { setEditing(null); setFormOpen(true) }}>
              <Plus className="mr-1 h-3.5 w-3.5" />{t("itsm:sla.escalation.create")}
            </Button>
          )}
        </div>
        {isLoading ? (
          <p className="text-sm text-muted-foreground">Loading...</p>
        ) : rules.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("itsm:sla.escalation.empty")}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("itsm:sla.escalation.triggerType")}</TableHead>
                <TableHead className="w-[60px]">{t("itsm:sla.escalation.level")}</TableHead>
                <TableHead className="w-[100px]">{t("itsm:sla.escalation.waitMinutes")}</TableHead>
                <TableHead>{t("itsm:sla.escalation.actionType")}</TableHead>
                <DataTableActionsHead className="min-w-[120px]">{t("common:actions")}</DataTableActionsHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rules.map((rule) => (
                <TableRow key={rule.id}>
                  <TableCell className="text-sm">{triggerLabel(rule.triggerType)}</TableCell>
                  <TableCell className="text-sm">{rule.level}</TableCell>
                  <TableCell className="text-sm">{rule.waitMinutes} min</TableCell>
                  <TableCell><Badge variant="outline">{actionLabel(rule.actionType)}</Badge></TableCell>
                  <DataTableActionsCell>
                    <DataTableActions>
                      {canUpdate && (
                        <Button variant="ghost" size="sm" className="px-2" onClick={() => { setEditing(rule); setFormOpen(true) }}>
                          <Pencil className="mr-1 h-3 w-3" />{t("common:edit")}
                        </Button>
                      )}
                      {canDelete && (
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button variant="ghost" size="sm" className="px-2 text-destructive hover:text-destructive">
                              <Trash2 className="mr-1 h-3 w-3" />{t("common:delete")}
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>{t("itsm:sla.escalation.deleteTitle")}</AlertDialogTitle>
                              <AlertDialogDescription>{t("itsm:sla.escalation.deleteDesc")}</AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel size="sm">{t("common:cancel")}</AlertDialogCancel>
                              <AlertDialogAction size="sm" onClick={() => deleteMut.mutate(rule.id)} disabled={deleteMut.isPending}>{t("itsm:sla.escalation.confirmDelete")}</AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      )}
                    </DataTableActions>
                  </DataTableActionsCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        <Sheet open={formOpen} onOpenChange={setFormOpen}>
          <SheetContent className="sm:max-w-md">
            <SheetHeader>
              <SheetTitle>{editing ? t("itsm:sla.escalation.edit") : t("itsm:sla.escalation.create")}</SheetTitle>
              <SheetDescription className="sr-only">{editing ? t("itsm:sla.escalation.edit") : t("itsm:sla.escalation.create")}</SheetDescription>
            </SheetHeader>
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-5 px-4">
                <FormField control={form.control} name="triggerType" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("itsm:sla.escalation.triggerType")}</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl>
                      <SelectContent>
                        <SelectItem value="response_timeout">{t("itsm:sla.escalation.responseTimeout")}</SelectItem>
                        <SelectItem value="resolution_timeout">{t("itsm:sla.escalation.resolutionTimeout")}</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )} />
                <div className="grid grid-cols-2 gap-4">
                  <FormField control={form.control} name="level" render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("itsm:sla.escalation.level")}</FormLabel>
                      <FormControl><Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )} />
                  <FormField control={form.control} name="waitMinutes" render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("itsm:sla.escalation.waitMinutes")}</FormLabel>
                      <FormControl><Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )} />
                </div>
                <FormField control={form.control} name="actionType" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("itsm:sla.escalation.actionType")}</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl>
                      <SelectContent>
                        <SelectItem value="notify">{t("itsm:sla.escalation.notify")}</SelectItem>
                        <SelectItem value="reassign">{t("itsm:sla.escalation.reassign")}</SelectItem>
                        <SelectItem value="escalate_priority">{t("itsm:sla.escalation.escalatePriority")}</SelectItem>
                      </SelectContent>
                    </Select>
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
      </TableCell>
    </TableRow>
  )
}

// ─── Main SLA Page ──────────────────────────────────────

export function Component() {
  const { t } = useTranslation(["itsm", "common"])
  const queryClient = useQueryClient()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<SLATemplateItem | null>(null)
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const slaSchema = useSLASchema()

  const canCreate = usePermission("itsm:sla:create")
  const canUpdate = usePermission("itsm:sla:update")
  const canDelete = usePermission("itsm:sla:delete")

  const { data: items = [], isLoading } = useQuery({
    queryKey: ["itsm-sla"],
    queryFn: () => fetchSLATemplates(),
  })

  const form = useForm<SLAFormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(slaSchema as any),
    defaultValues: { name: "", code: "", description: "", responseMinutes: 240, resolutionMinutes: 1440 },
  })

  useEffect(() => {
    if (formOpen) {
      if (editing) {
        form.reset({ name: editing.name, code: editing.code, description: editing.description, responseMinutes: editing.responseMinutes, resolutionMinutes: editing.resolutionMinutes })
      } else {
        form.reset({ name: "", code: "", description: "", responseMinutes: 240, resolutionMinutes: 1440 })
      }
    }
  }, [formOpen, editing, form])

  const createMut = useMutation({
    mutationFn: (v: SLAFormValues) => createSLATemplate({ ...v, description: v.description ?? "" }),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-sla"] }); setFormOpen(false); toast.success(t("itsm:sla.createSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  const updateMut = useMutation({
    mutationFn: (v: SLAFormValues) => updateSLATemplate(editing!.id, { ...v, description: v.description ?? "" }),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-sla"] }); setFormOpen(false); toast.success(t("itsm:sla.updateSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  const deleteMut = useMutation({
    mutationFn: (id: number) => deleteSLATemplate(id),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["itsm-sla"] }); toast.success(t("itsm:sla.deleteSuccess")) },
    onError: (err) => toast.error(err.message),
  })

  function onSubmit(v: SLAFormValues) { if (editing) { updateMut.mutate(v) } else { createMut.mutate(v) } }
  const isPending = createMut.isPending || updateMut.isPending

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t("itsm:sla.title")}</h2>
        {canCreate && (
          <Button onClick={() => { setEditing(null); setFormOpen(true) }}>
            <Plus className="mr-1.5 h-4 w-4" />{t("itsm:sla.create")}
          </Button>
        )}
      </div>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[40px]" />
              <TableHead className="min-w-[140px]">{t("itsm:sla.name")}</TableHead>
              <TableHead className="w-[100px]">{t("itsm:sla.code")}</TableHead>
              <TableHead className="w-[120px]">{t("itsm:sla.responseMinutes")}</TableHead>
              <TableHead className="w-[120px]">{t("itsm:sla.resolutionMinutes")}</TableHead>
              <TableHead className="w-[80px]">{t("common:status")}</TableHead>
              <DataTableActionsHead className="min-w-[140px]">{t("common:actions")}</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={7} />
            ) : items.length === 0 ? (
              <DataTableEmptyRow colSpan={7} icon={Timer} title={t("itsm:sla.empty")} description={canCreate ? t("itsm:sla.emptyHint") : undefined} />
            ) : (
              items.flatMap((item) => {
                const isExpanded = expandedId === item.id
                const rows = [
                  <TableRow key={item.id} className="cursor-pointer" onClick={() => setExpandedId(isExpanded ? null : item.id)}>
                    <TableCell className="w-[40px] px-2">
                      <ChevronRight className={cn("h-4 w-4 transition-transform", isExpanded && "rotate-90")} />
                    </TableCell>
                    <TableCell className="font-medium">{item.name}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{item.code}</TableCell>
                    <TableCell className="text-sm">{item.responseMinutes} min</TableCell>
                    <TableCell className="text-sm">{item.resolutionMinutes} min</TableCell>
                    <TableCell>
                      <Badge variant={item.isActive ? "default" : "secondary"}>
                        {item.isActive ? t("itsm:sla.active") : t("itsm:sla.inactive")}
                      </Badge>
                    </TableCell>
                    <DataTableActionsCell>
                      <DataTableActions>
                        {canUpdate && (
                          <Button variant="ghost" size="sm" className="px-2.5" onClick={(e) => { e.stopPropagation(); setEditing(item); setFormOpen(true) }}>
                            <Pencil className="mr-1 h-3.5 w-3.5" />{t("common:edit")}
                          </Button>
                        )}
                        {canDelete && (
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button variant="ghost" size="sm" className="px-2.5 text-destructive hover:text-destructive" onClick={(e) => e.stopPropagation()}>
                                <Trash2 className="mr-1 h-3.5 w-3.5" />{t("common:delete")}
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>{t("itsm:sla.deleteTitle")}</AlertDialogTitle>
                                <AlertDialogDescription>{t("itsm:sla.deleteDesc", { name: item.name })}</AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel size="sm">{t("common:cancel")}</AlertDialogCancel>
                                <AlertDialogAction size="sm" onClick={() => deleteMut.mutate(item.id)} disabled={deleteMut.isPending}>{t("itsm:sla.confirmDelete")}</AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        )}
                      </DataTableActions>
                    </DataTableActionsCell>
                  </TableRow>,
                ]
                if (isExpanded) {
                  rows.push(<EscalationRules key={`esc-${item.id}`} slaId={item.id} />)
                }
                return rows
              })
            )}
          </TableBody>
        </Table>
      </DataTableCard>

      <Sheet open={formOpen} onOpenChange={setFormOpen}>
        <SheetContent className="sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>{editing ? t("itsm:sla.edit") : t("itsm:sla.create")}</SheetTitle>
            <SheetDescription className="sr-only">{editing ? t("itsm:sla.edit") : t("itsm:sla.create")}</SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-5 px-4">
              <FormField control={form.control} name="name" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:sla.name")}</FormLabel>
                  <FormControl><Input placeholder={t("itsm:sla.namePlaceholder")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="code" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:sla.code")}</FormLabel>
                  <FormControl><Input placeholder={t("itsm:sla.codePlaceholder")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <div className="grid grid-cols-2 gap-4">
                <FormField control={form.control} name="responseMinutes" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("itsm:sla.responseMinutes")}</FormLabel>
                    <FormControl><Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={form.control} name="resolutionMinutes" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("itsm:sla.resolutionMinutes")}</FormLabel>
                    <FormControl><Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
              </div>
              <FormField control={form.control} name="description" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:sla.description")}</FormLabel>
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
