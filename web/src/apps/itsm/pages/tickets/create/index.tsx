"use client"

import { useState, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useQuery, useMutation } from "@tanstack/react-query"
import { ArrowLeft } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  Card, CardContent, CardHeader, CardTitle,
} from "@/components/ui/card"
import {
  type CatalogItem, fetchCatalogTree, fetchServiceDefs,
  fetchPriorities, createTicket,
} from "../../../api"

function flattenCatalogs(nodes: CatalogItem[], depth = 0): Array<CatalogItem & { depth: number }> {
  const result: Array<CatalogItem & { depth: number }> = []
  for (const n of nodes) {
    result.push({ ...n, depth })
    if (n.children?.length) result.push(...flattenCatalogs(n.children, depth + 1))
  }
  return result
}

function useTicketSchema() {
  const { t } = useTranslation("itsm")
  return z.object({
    title: z.string().min(1, t("validation.titleRequired")),
    description: z.string().optional(),
    serviceId: z.number().min(1, t("validation.serviceRequired")),
    priorityId: z.number().min(1, t("validation.priorityRequired")),
  })
}

type FormValues = z.infer<ReturnType<typeof useTicketSchema>>

export function Component() {
  const { t } = useTranslation(["itsm", "common"])
  const navigate = useNavigate()
  const schema = useTicketSchema()
  const [catalogFilter, setCatalogFilter] = useState<number | null>(null)

  const { data: catalogs = [] } = useQuery({
    queryKey: ["itsm-catalogs"],
    queryFn: () => fetchCatalogTree(),
  })

  const flatCatalogs = useMemo(() => flattenCatalogs(catalogs), [catalogs])

  const { data: servicesData } = useQuery({
    queryKey: ["itsm-services-for-create", catalogFilter],
    queryFn: () => fetchServiceDefs({
      page: 1,
      pageSize: 100,
      isActive: true,
      ...(catalogFilter ? { catalogId: catalogFilter } : {}),
    }),
  })
  const services = servicesData?.items ?? []

  const { data: priorities = [] } = useQuery({
    queryKey: ["itsm-priorities"],
    queryFn: () => fetchPriorities(),
  })

  const form = useForm<FormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(schema as any),
    defaultValues: { title: "", description: "", serviceId: 0, priorityId: 0 },
  })

  const createMut = useMutation({
    mutationFn: (v: FormValues) => createTicket({
      title: v.title,
      description: v.description,
      serviceId: v.serviceId,
      priorityId: v.priorityId,
    }),
    onSuccess: (data) => {
      toast.success(t("itsm:tickets.createSuccess"))
      navigate(`/itsm/tickets/${data.id}`)
    },
    onError: (err) => toast.error(err.message),
  })

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate("/itsm/tickets")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h2 className="text-lg font-semibold">{t("itsm:tickets.create")}</h2>
      </div>

      <Card className="mx-auto max-w-2xl">
        <CardHeader>
          <CardTitle className="text-base">{t("itsm:tickets.create")}</CardTitle>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit((v) => createMut.mutate(v))} className="space-y-5">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("itsm:tickets.servicePlaceholder")}</label>
                <Select
                  value={catalogFilter ? String(catalogFilter) : "all"}
                  onValueChange={(v) => {
                    setCatalogFilter(v === "all" ? null : Number(v))
                    form.setValue("serviceId", 0)
                  }}
                >
                  <SelectTrigger><SelectValue placeholder={t("itsm:services.allCatalogs")} /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">{t("itsm:services.allCatalogs")}</SelectItem>
                    {flatCatalogs.map((c) => (
                      <SelectItem key={c.id} value={String(c.id)}>{"─".repeat(c.depth)} {c.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <FormField control={form.control} name="serviceId" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:tickets.service")}</FormLabel>
                  <Select onValueChange={(v) => field.onChange(Number(v))} value={field.value ? String(field.value) : ""}>
                    <FormControl><SelectTrigger><SelectValue placeholder={t("itsm:tickets.servicePlaceholder")} /></SelectTrigger></FormControl>
                    <SelectContent>
                      {services.map((s) => (
                        <SelectItem key={s.id} value={String(s.id)}>{s.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )} />

              <FormField control={form.control} name="title" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:tickets.ticketTitle")}</FormLabel>
                  <FormControl><Input placeholder={t("itsm:tickets.titlePlaceholder")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />

              <FormField control={form.control} name="priorityId" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:tickets.priority")}</FormLabel>
                  <Select onValueChange={(v) => field.onChange(Number(v))} value={field.value ? String(field.value) : ""}>
                    <FormControl><SelectTrigger><SelectValue placeholder={t("itsm:tickets.priorityPlaceholder")} /></SelectTrigger></FormControl>
                    <SelectContent>
                      {priorities.map((p) => (
                        <SelectItem key={p.id} value={String(p.id)}>
                          <span className="mr-1.5 inline-block h-2.5 w-2.5 rounded-full" style={{ backgroundColor: p.color }} />
                          {p.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )} />

              <FormField control={form.control} name="description" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("itsm:tickets.description")}</FormLabel>
                  <FormControl><Textarea rows={4} placeholder={t("itsm:tickets.descriptionPlaceholder")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />

              <div className="flex justify-end gap-2 pt-2">
                <Button type="button" variant="outline" onClick={() => navigate("/itsm/tickets")}>{t("common:cancel")}</Button>
                <Button type="submit" disabled={createMut.isPending}>
                  {createMut.isPending ? t("common:saving") : t("common:create")}
                </Button>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  )
}
