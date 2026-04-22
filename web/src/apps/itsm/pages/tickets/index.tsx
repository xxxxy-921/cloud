"use client"

import { useState, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router"
import { useQuery } from "@tanstack/react-query"
import { Search, Ticket } from "lucide-react"
import { useListPage } from "@/hooks/use-list-page"
import { withActiveMenuPermission } from "@/lib/navigation-state"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  DataTableCard, DataTableEmptyRow, DataTableLoadingRow,
  DataTablePagination, DataTableToolbar, DataTableToolbarGroup,
} from "@/components/ui/data-table"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  type TicketItem, fetchPriorities, fetchServiceDefs,
} from "../../api"
import { SLABadge } from "../../components/sla-badge"
import { TICKET_STATUS_OPTIONS } from "../../components/ticket-status"
import { TicketStatusBadge } from "../../components/ticket-status-badge"
import { TICKET_MENU_PERMISSION } from "./navigation"

export function Component() {
  const { t } = useTranslation(["itsm", "common"])
  const navigate = useNavigate()

  const [statusFilter, setStatusFilter] = useState("")
  const [priorityFilter, setPriorityFilter] = useState("")
  const [serviceFilter, setServiceFilter] = useState("")

  const extraParams = useMemo(() => {
    const params: Record<string, string> = {}
    if (statusFilter) params.status = statusFilter
    if (priorityFilter) params.priorityId = priorityFilter
    if (serviceFilter) params.serviceId = serviceFilter
    return params
  }, [statusFilter, priorityFilter, serviceFilter])

  const {
    keyword, setKeyword, page, setPage,
    items, total, totalPages, isLoading, handleSearch,
  } = useListPage<TicketItem>({
    queryKey: "itsm-tickets",
    endpoint: "/api/v1/itsm/tickets",
    extraParams,
  })

  const { data: priorities = [] } = useQuery({
    queryKey: ["itsm-priorities"],
    queryFn: () => fetchPriorities(),
  })

  const { data: servicesData } = useQuery({
    queryKey: ["itsm-services-list"],
    queryFn: () => fetchServiceDefs({ page: 1, pageSize: 100 }),
  })
  const services = servicesData?.items ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t("itsm:tickets.title")}</h2>
      </div>

      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center sm:flex-wrap">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input placeholder={t("itsm:tickets.searchPlaceholder")} value={keyword} onChange={(e) => setKeyword(e.target.value)} className="pl-8" />
            </div>
            <Select value={statusFilter || "all"} onValueChange={(v) => { setStatusFilter(v === "all" ? "" : v); setPage(1) }}>
              <SelectTrigger className="w-[140px]"><SelectValue placeholder={t("itsm:tickets.allStatuses")} /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("itsm:tickets.allStatuses")}</SelectItem>
                {Object.entries(TICKET_STATUS_OPTIONS).map(([k, v]) => (
                  <SelectItem key={k} value={k}>{t(`itsm:tickets.${v.key}`)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={priorityFilter || "all"} onValueChange={(v) => { setPriorityFilter(v === "all" ? "" : v); setPage(1) }}>
              <SelectTrigger className="w-[140px]"><SelectValue placeholder={t("itsm:tickets.allPriorities")} /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("itsm:tickets.allPriorities")}</SelectItem>
                {priorities.map((p) => (
                  <SelectItem key={p.id} value={String(p.id)}>
                    <span className="mr-1.5 inline-block h-2.5 w-2.5 rounded-full" style={{ backgroundColor: p.color }} />
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={serviceFilter || "all"} onValueChange={(v) => { setServiceFilter(v === "all" ? "" : v); setPage(1) }}>
              <SelectTrigger className="w-[160px]"><SelectValue placeholder={t("itsm:tickets.allServices")} /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("itsm:tickets.allServices")}</SelectItem>
                {services.map((s) => (
                  <SelectItem key={s.id} value={String(s.id)}>{s.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button type="submit" variant="outline" size="sm">{t("common:search")}</Button>
          </form>
        </DataTableToolbarGroup>
      </DataTableToolbar>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[180px] min-w-[180px]">{t("itsm:tickets.code")}</TableHead>
              <TableHead className="min-w-[200px]">{t("itsm:tickets.ticketTitle")}</TableHead>
              <TableHead className="w-[90px]">{t("itsm:tickets.requester")}</TableHead>
              <TableHead className="w-[100px]">{t("itsm:tickets.priority")}</TableHead>
              <TableHead className="w-[100px]">{t("itsm:tickets.status")}</TableHead>
              <TableHead className="w-[100px]">{t("itsm:tickets.service")}</TableHead>
              <TableHead className="w-[110px]">当前责任方</TableHead>
              <TableHead className="min-w-[160px]">下一步</TableHead>
              <TableHead className="w-[150px]">{t("itsm:tickets.slaStatus")}</TableHead>
              <TableHead className="w-[140px]">{t("itsm:tickets.createdAt")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={9} />
            ) : items.length === 0 ? (
              <DataTableEmptyRow colSpan={9} icon={Ticket} title={t("itsm:tickets.empty")} />
            ) : (
              items.map((item) => (
                <TableRow
                  key={item.id}
                  className="cursor-pointer"
                  onClick={() => navigate(`/itsm/tickets/${item.id}`, { state: withActiveMenuPermission(TICKET_MENU_PERMISSION.list) })}
                >
                  <TableCell className="font-mono text-sm whitespace-nowrap">{item.code}</TableCell>
                  <TableCell className="font-medium">{item.title}</TableCell>
                  <TableCell className="text-sm">{item.requesterName}</TableCell>
                  <TableCell>
                    <span className="inline-flex items-center gap-1.5 text-sm">
                      <span className="inline-block h-2.5 w-2.5 rounded-full" style={{ backgroundColor: item.priorityColor }} />
                      {item.priorityName}
                    </span>
                  </TableCell>
                  <TableCell>
                    <TicketStatusBadge ticket={item} />
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">{item.serviceName}</TableCell>
                  <TableCell className="text-sm">{item.currentOwnerName || item.assigneeName || "—"}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">{item.nextStepSummary || "等待受理"}</TableCell>
                  <TableCell>
                    <SLABadge slaStatus={item.slaStatus} slaResolutionDeadline={item.slaResolutionDeadline} />
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">{new Date(item.createdAt).toLocaleString()}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </DataTableCard>

      <DataTablePagination total={total} page={page} totalPages={totalPages} onPageChange={setPage} />
    </div>
  )
}
