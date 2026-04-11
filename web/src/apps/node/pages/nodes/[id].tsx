import { useState } from "react"
import { useParams, useNavigate, useSearchParams } from "react-router"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import {
  ArrowLeft,
  Pencil,
  RefreshCw,
  Plus,
  Unlink,
  RotateCw,
  Play,
  Square,
  Loader2,
  Copy,
  Check,
  RefreshCcw,
} from "lucide-react"
import { api } from "@/lib/api"
import { usePermission } from "@/hooks/use-permission"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { DataTablePagination } from "@/components/ui/data-table"
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { formatDateTime } from "@/lib/utils"
import { NodeSheet, type NodeItem } from "../../components/node-sheet"
import { NODE_STATUS_VARIANTS, PROCESS_STATUS_VARIANTS } from "../../constants"

interface NodeDetail {
  id: number
  name: string
  status: string
  labels: Record<string, string> | null
  systemInfo: Record<string, unknown> | null
  capabilities: Record<string, unknown> | null
  version: string
  lastHeartbeat: string | null
  createdAt: string
  updatedAt: string
}

interface NodeProcessItem {
  id: number
  nodeId: number
  processDefId: number
  status: string
  pid: number
  configVersion: string
  processName: string
  displayName: string
  createdAt: string
  updatedAt: string
}

interface CommandItem {
  id: number
  nodeId: number
  type: string
  payload: Record<string, unknown>
  status: string
  result: string
  ackedAt: string | null
  createdAt: string
}

interface ProcessDefOption {
  id: number
  name: string
  displayName: string
  startCommand: string
  stopCommand: string
  reloadCommand: string
  configFiles: unknown[] | null
  env: Record<string, string> | null
}

export function Component() {
  const { t } = useTranslation(["node", "common"])
  const { id } = useParams()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const [editOpen, setEditOpen] = useState(false)
  const [bindDialogOpen, setBindDialogOpen] = useState(false)
  const [selectedProcessDef, setSelectedProcessDef] = useState("")
  const [overrideVarsText, setOverrideVarsText] = useState("")
  const [tokenDialogOpen, setTokenDialogOpen] = useState(false)
  const [newToken, setNewToken] = useState("")
  const [copied, setCopied] = useState(false)
  const [cmdPage, setCmdPage] = useState(1)
  const [logProcessId, setLogProcessId] = useState("")
  const [logStream, setLogStream] = useState("all")
  const [logPage, setLogPage] = useState(1)

  const canUpdate = usePermission("node:update")

  const { data: node, isLoading } = useQuery({
    queryKey: ["node", id],
    queryFn: () => api.get<NodeDetail>(`/api/v1/nodes/${id}`),
    enabled: !!id,
  })

  const { data: processesData } = useQuery({
    queryKey: ["node-processes", id],
    queryFn: () => api.get<NodeProcessItem[]>(`/api/v1/nodes/${id}/processes`),
    enabled: !!id,
  })

  const { data: commandsData } = useQuery({
    queryKey: ["node-commands", id, cmdPage],
    queryFn: () => api.get<{ items: CommandItem[]; total: number }>(`/api/v1/nodes/${id}/commands?page=${cmdPage}&pageSize=20`),
    enabled: !!id,
  })

  const { data: processDefsData } = useQuery({
    queryKey: ["process-defs-options"],
    queryFn: () => api.get<{ items: ProcessDefOption[] }>("/api/v1/process-defs?pageSize=100"),
    enabled: bindDialogOpen,
  })

  const rotateTokenMutation = useMutation({
    mutationFn: () => api.post<{ token: string }>(`/api/v1/nodes/${id}/rotate-token`),
    onSuccess: (data) => {
      setNewToken(data.token)
      setTokenDialogOpen(true)
      setCopied(false)
      toast.success(t("node:nodes.rotateSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const bindMutation = useMutation({
    mutationFn: ({ processDefId, overrideVars }: { processDefId: number; overrideVars?: Record<string, unknown> }) =>
      api.post(`/api/v1/nodes/${id}/processes`, { processDefId, overrideVars }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["node-processes", id] })
      setBindDialogOpen(false)
      setSelectedProcessDef("")
      setOverrideVarsText("")
      toast.success(t("node:nodes.bindSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const unbindMutation = useMutation({
    mutationFn: (processDefId: number) =>
      api.delete(`/api/v1/nodes/${id}/processes/${processDefId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["node-processes", id] })
      toast.success(t("node:nodes.unbindSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const restartMutation = useMutation({
    mutationFn: (processDefId: number) =>
      api.post(`/api/v1/nodes/${id}/processes/${processDefId}/restart`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["node-commands", id] })
      toast.success(t("node:nodes.restartSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const startMutation = useMutation({
    mutationFn: (processDefId: number) =>
      api.post(`/api/v1/nodes/${id}/processes/${processDefId}/start`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["node-commands", id] })
      toast.success(t("node:nodes.startSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const stopMutation = useMutation({
    mutationFn: (processDefId: number) =>
      api.post(`/api/v1/nodes/${id}/processes/${processDefId}/stop`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["node-commands", id] })
      toast.success(t("node:nodes.stopSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const reloadMutation = useMutation({
    mutationFn: (processDefId: number) =>
      api.post(`/api/v1/nodes/${id}/processes/${processDefId}/reload`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["node-commands", id] })
      toast.success(t("node:processDefs.reloadSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  const { data: logsData } = useQuery({
    queryKey: ["node-process-logs", id, logProcessId, logStream, logPage],
    queryFn: () => {
      let url = `/api/v1/nodes/${id}/processes/${logProcessId}/logs?page=${logPage}&pageSize=50`
      if (logStream !== "all") url += `&stream=${logStream}`
      return api.get<{ items: Array<{ id: number; stream: string; content: string; processName: string; timestamp: string }>; total: number }>(url)
    },
    enabled: !!id && !!logProcessId,
  })

  const processes = processesData ?? []
  const commands = commandsData?.items ?? []
  const commandsTotal = commandsData?.total ?? 0
  const processDefs = processDefsData?.items ?? []
  const requestedTab = searchParams.get("tab")
  const activeTab =
    requestedTab === "info" || requestedTab === "processes" || requestedTab === "commands" || requestedTab === "logs"
      ? requestedTab
      : "info"

  if (isLoading || !node) {
    return (
      <div className="flex min-h-[200px] items-center justify-center text-muted-foreground">
        {t("common:loading")}
      </div>
    )
  }

  const variant = NODE_STATUS_VARIANTS[node.status] ?? ("secondary" as const)
  const sysInfo = node.systemInfo as Record<string, unknown> | null

  function handleTabChange(value: string) {
    const nextParams = new URLSearchParams(searchParams)
    nextParams.set("tab", value)
    setSearchParams(nextParams, { replace: true })
  }

  function handleCopyToken() {
    navigator.clipboard.writeText(newToken).then(() => {
      setCopied(true)
      toast.success(t("node:nodes.tokenCopied"))
      setTimeout(() => setCopied(false), 2000)
    })
  }

  const editableNode: NodeItem = {
    id: node.id,
    name: node.name,
    status: node.status,
    labels: node.labels,
    systemInfo: node.systemInfo,
    version: node.version,
    lastHeartbeat: node.lastHeartbeat,
    processCount: processes.length,
    createdAt: node.createdAt,
    updatedAt: node.updatedAt,
  }

  const selectedPd = selectedProcessDef
    ? processDefs.find((d) => String(d.id) === selectedProcessDef) ?? null
    : null

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate("/node/nodes")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold">{node.name}</h2>
            <Badge variant={variant}>{t(`node:status.${node.status}`, node.status)}</Badge>
          </div>
          {node.version && (
            <p className="text-sm text-muted-foreground">v{node.version}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          {canUpdate && (
            <>
              <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
                <Pencil className="mr-1.5 h-3.5 w-3.5" />
                {t("common:edit")}
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button variant="outline" size="sm">
                    <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                    {t("node:nodes.rotateToken")}
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>{t("node:nodes.rotateTokenTitle")}</AlertDialogTitle>
                    <AlertDialogDescription>{t("node:nodes.rotateTokenDesc")}</AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
                    <AlertDialogAction
                      onClick={() => rotateTokenMutation.mutate()}
                      disabled={rotateTokenMutation.isPending}
                    >
                      {rotateTokenMutation.isPending && <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />}
                      {t("node:nodes.confirmRotate")}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </>
          )}
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList className="h-auto w-fit max-w-full flex-wrap justify-start gap-1 rounded-lg bg-muted/50 p-1">
          <TabsTrigger value="info" className="h-8 flex-none px-3 text-xs sm:text-sm">
            {t("node:nodes.basicInfo")}
          </TabsTrigger>
          <TabsTrigger value="processes" className="h-8 flex-none px-3 text-xs sm:text-sm">
            {t("node:nodes.processes")} ({processes.length})
          </TabsTrigger>
          <TabsTrigger value="commands" className="h-8 flex-none px-3 text-xs sm:text-sm">
            {t("node:nodes.commands")}
          </TabsTrigger>
          <TabsTrigger value="logs" className="h-8 flex-none px-3 text-xs sm:text-sm">
            {t("node:processDefs.logs")}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="info" className="space-y-4">
          <div className="rounded-lg border">
            <div className="grid gap-x-6 gap-y-4 px-4 py-4 text-sm sm:grid-cols-2 lg:grid-cols-3">
              <div>
                <p className="text-muted-foreground">{t("common:name")}</p>
                <p className="mt-1 font-medium">{node.name}</p>
              </div>
              <div>
                <p className="text-muted-foreground">{t("common:status")}</p>
                <div className="mt-1">
                  <Badge variant={variant}>{t(`node:status.${node.status}`, node.status)}</Badge>
                </div>
              </div>
              <div>
                <p className="text-muted-foreground">{t("node:nodes.version")}</p>
                <p className="mt-1">{node.version || "-"}</p>
              </div>
              <div>
                <p className="text-muted-foreground">{t("node:nodes.lastHeartbeat")}</p>
                <p className="mt-1">
                  {node.lastHeartbeat ? formatDateTime(node.lastHeartbeat) : t("node:nodes.neverConnected")}
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">{t("common:createdAt")}</p>
                <p className="mt-1">{formatDateTime(node.createdAt)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">{t("common:updatedAt")}</p>
                <p className="mt-1">{formatDateTime(node.updatedAt)}</p>
              </div>
              {node.labels && Object.keys(node.labels).length > 0 && (
                <div className="sm:col-span-2 lg:col-span-3">
                  <p className="text-muted-foreground">{t("node:nodes.labels")}</p>
                  <div className="mt-1 flex flex-wrap gap-1.5">
                    {Object.entries(node.labels).map(([k, v]) => (
                      <Badge key={k} variant="outline" className="text-xs">
                        {k}: {v}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {sysInfo && Object.keys(sysInfo).length > 0 && (
            <div className="rounded-lg border">
              <div className="px-4 py-3 border-b">
                <p className="font-medium text-sm">{t("node:nodes.systemInfo")}</p>
              </div>
              <div className="grid gap-x-6 gap-y-4 px-4 py-4 text-sm sm:grid-cols-2 lg:grid-cols-4">
                {sysInfo.hostname != null && (
                  <div>
                    <p className="text-muted-foreground">{t("node:nodes.hostname")}</p>
                    <p className="mt-1 font-medium">{String(sysInfo.hostname)}</p>
                  </div>
                )}
                {sysInfo.os != null && (
                  <div>
                    <p className="text-muted-foreground">{t("node:nodes.os")}</p>
                    <p className="mt-1">{String(sysInfo.os)}</p>
                  </div>
                )}
                {sysInfo.arch != null && (
                  <div>
                    <p className="text-muted-foreground">{t("node:nodes.arch")}</p>
                    <p className="mt-1">{String(sysInfo.arch)}</p>
                  </div>
                )}
                {sysInfo.cpus != null && (
                  <div>
                    <p className="text-muted-foreground">{t("node:nodes.cpus")}</p>
                    <p className="mt-1">{String(sysInfo.cpus)}</p>
                  </div>
                )}
              </div>
            </div>
          )}
        </TabsContent>

        <TabsContent value="processes" className="space-y-4">
          {canUpdate && (
            <div className="flex justify-end">
              <Button size="sm" onClick={() => setBindDialogOpen(true)}>
                <Plus className="mr-1.5 h-4 w-4" />
                {t("node:nodes.bindProcess")}
              </Button>
            </div>
          )}

          {processes.length === 0 ? (
            <div className="rounded-lg border p-8 text-center text-sm text-muted-foreground">
              <p>{t("node:nodes.noProcesses")}</p>
              {canUpdate && <p className="mt-1">{t("node:nodes.noProcessesHint")}</p>}
            </div>
          ) : (
            <div className="overflow-hidden rounded-xl border bg-card">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("node:processDefs.displayName")}</TableHead>
                    <TableHead className="w-[100px]">{t("common:status")}</TableHead>
                    <TableHead className="w-[80px]">{t("node:nodes.pid")}</TableHead>
                    <TableHead className="w-[120px]">{t("node:nodes.configVersion")}</TableHead>
                    {canUpdate && <TableHead className="w-[280px]">{t("common:actions")}</TableHead>}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {processes.map((proc) => {
                    const pVariant = PROCESS_STATUS_VARIANTS[proc.status] ?? ("secondary" as const)
                    return (
                      <TableRow key={proc.id}>
                        <TableCell>
                          <div>
                            <p className="font-medium">{proc.displayName || proc.processName}</p>
                            <p className="text-xs text-muted-foreground font-mono">{proc.processName}</p>
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant={pVariant}>{t(`node:status.${proc.status}`, proc.status)}</Badge>
                        </TableCell>
                        <TableCell className="text-sm font-mono">{proc.pid || "-"}</TableCell>
                        <TableCell className="text-xs text-muted-foreground font-mono">
                          {proc.configVersion ? proc.configVersion.substring(0, 12) : "-"}
                        </TableCell>
                        {canUpdate && (
                          <TableCell>
                            <div className="flex items-center gap-1">
                              {proc.status === "running" ? (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="px-2"
                                  onClick={() => stopMutation.mutate(proc.processDefId)}
                                  disabled={stopMutation.isPending}
                                >
                                  <Square className="mr-1 h-3.5 w-3.5" />
                                  {t("node:nodes.stop")}
                                </Button>
                              ) : (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="px-2"
                                  onClick={() => startMutation.mutate(proc.processDefId)}
                                  disabled={startMutation.isPending}
                                >
                                  <Play className="mr-1 h-3.5 w-3.5" />
                                  {t("node:nodes.start")}
                                </Button>
                              )}
                              <Button
                                variant="ghost"
                                size="sm"
                                className="px-2"
                                onClick={() => restartMutation.mutate(proc.processDefId)}
                                disabled={restartMutation.isPending}
                              >
                                <RotateCw className="mr-1 h-3.5 w-3.5" />
                                {t("node:nodes.restart")}
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="px-2"
                                onClick={() => reloadMutation.mutate(proc.processDefId)}
                                disabled={reloadMutation.isPending}
                              >
                                <RefreshCcw className="mr-1 h-3.5 w-3.5" />
                                {t("node:processDefs.reload")}
                              </Button>
                              <AlertDialog>
                                <AlertDialogTrigger asChild>
                                  <Button variant="ghost" size="sm" className="px-2 text-destructive hover:text-destructive">
                                    <Unlink className="mr-1 h-3.5 w-3.5" />
                                    {t("node:nodes.unbind")}
                                  </Button>
                                </AlertDialogTrigger>
                                <AlertDialogContent>
                                  <AlertDialogHeader>
                                    <AlertDialogTitle>{t("node:nodes.unbindTitle")}</AlertDialogTitle>
                                    <AlertDialogDescription>
                                      {t("node:nodes.unbindDesc", { name: proc.displayName || proc.processName })}
                                    </AlertDialogDescription>
                                  </AlertDialogHeader>
                                  <AlertDialogFooter>
                                    <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
                                    <AlertDialogAction onClick={() => unbindMutation.mutate(proc.processDefId)}>
                                      {t("node:nodes.unbind")}
                                    </AlertDialogAction>
                                  </AlertDialogFooter>
                                </AlertDialogContent>
                              </AlertDialog>
                            </div>
                          </TableCell>
                        )}
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          )}
        </TabsContent>

        <TabsContent value="commands" className="space-y-4">
          <div className="flex justify-end">
            <Button
              variant="outline"
              size="sm"
              onClick={() => queryClient.invalidateQueries({ queryKey: ["node-commands", id] })}
            >
              <RefreshCcw className="mr-1.5 h-3.5 w-3.5" />
              {t("node:processDefs.refresh")}
            </Button>
          </div>
          {commands.length === 0 ? (
            <div className="rounded-lg border p-8 text-center text-sm text-muted-foreground">
              {t("node:nodes.noCommands")}
            </div>
          ) : (
            <>
              <div className="overflow-hidden rounded-xl border bg-card">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("node:nodes.commandType")}</TableHead>
                      <TableHead className="w-[100px]">{t("node:nodes.commandStatus")}</TableHead>
                      <TableHead>{t("node:nodes.commandResult")}</TableHead>
                      <TableHead className="w-[150px]">{t("node:nodes.commandTime")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {commands.map((cmd) => {
                      const cmdVariant = cmd.status === "acked" ? "default" : cmd.status === "failed" ? "destructive" : "secondary"
                      return (
                        <TableRow key={cmd.id}>
                          <TableCell className="font-mono text-sm">{cmd.type}</TableCell>
                          <TableCell>
                            <Badge variant={cmdVariant}>{t(`node:status.${cmd.status}`, cmd.status)}</Badge>
                          </TableCell>
                          <TableCell className="text-sm text-muted-foreground max-w-[300px] truncate">
                            {cmd.result || "-"}
                          </TableCell>
                          <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                            {formatDateTime(cmd.createdAt)}
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </div>
              <DataTablePagination
                total={commandsTotal}
                page={cmdPage}
                totalPages={Math.ceil(commandsTotal / 20)}
                onPageChange={setCmdPage}
              />
            </>
          )}
        </TabsContent>

        <TabsContent value="logs" className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            <Select value={logProcessId} onValueChange={(v) => { setLogProcessId(v); setLogPage(1) }}>
              <SelectTrigger className="w-[200px]">
                <SelectValue placeholder={t("node:nodes.selectProcess")} />
              </SelectTrigger>
              <SelectContent>
                {processes.map((p) => (
                  <SelectItem key={p.processDefId} value={String(p.processDefId)}>
                    {p.displayName || p.processName}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={logStream} onValueChange={(v) => { setLogStream(v); setLogPage(1) }}>
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder={t("node:processDefs.allStreams")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("node:processDefs.allStreams")}</SelectItem>
                <SelectItem value="stdout">{t("node:processDefs.stdout")}</SelectItem>
                <SelectItem value="stderr">{t("node:processDefs.stderr")}</SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="outline"
              size="sm"
              onClick={() => queryClient.invalidateQueries({ queryKey: ["node-process-logs", id, logProcessId, logStream, logPage] })}
              disabled={!logProcessId}
            >
              <RefreshCcw className="mr-1.5 h-3.5 w-3.5" />
              {t("node:processDefs.refresh")}
            </Button>
          </div>

          {!logProcessId ? (
            <div className="rounded-lg border p-8 text-center text-sm text-muted-foreground">
              {t("node:nodes.selectProcess")}
            </div>
          ) : !logsData?.items?.length ? (
            <div className="rounded-lg border p-8 text-center text-sm text-muted-foreground">
              {t("node:processDefs.noLogs")}
            </div>
          ) : (
            <>
              <div className="overflow-hidden rounded-xl border bg-card">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[160px]">{t("node:nodes.commandTime")}</TableHead>
                      <TableHead className="w-[80px]">{t("node:processDefs.stream")}</TableHead>
                      <TableHead>{t("node:processDefs.logs")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {logsData.items.map((log) => (
                      <TableRow key={log.id}>
                        <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                          {formatDateTime(log.timestamp)}
                        </TableCell>
                        <TableCell>
                          <Badge variant={log.stream === "stderr" ? "destructive" : "secondary"}>
                            {log.stream}
                          </Badge>
                        </TableCell>
                        <TableCell className="font-mono text-xs whitespace-pre-wrap break-all">
                          {log.content}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              <DataTablePagination
                total={logsData.total ?? 0}
                page={logPage}
                totalPages={Math.ceil((logsData.total ?? 0) / 50)}
                onPageChange={setLogPage}
              />
            </>
          )}
        </TabsContent>
      </Tabs>

      <NodeSheet open={editOpen} onOpenChange={setEditOpen} node={editableNode} />

      {/* Bind process sheet */}
      <Sheet open={bindDialogOpen} onOpenChange={(open) => {
        setBindDialogOpen(open)
        if (!open) { setSelectedProcessDef(""); setOverrideVarsText("") }
      }}>
        <SheetContent className="sm:max-w-md overflow-y-auto">
          <SheetHeader>
            <SheetTitle>{t("node:nodes.bindProcess")}</SheetTitle>
            <SheetDescription className="sr-only">{t("node:nodes.bindProcess")}</SheetDescription>
          </SheetHeader>
          <div className="flex flex-col gap-4 px-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("node:nodes.selectProcessDef")}</label>
              <Select value={selectedProcessDef} onValueChange={setSelectedProcessDef}>
                <SelectTrigger>
                  <SelectValue placeholder={t("node:nodes.selectProcessDef")} />
                </SelectTrigger>
                <SelectContent>
                  {processDefs.map((pd) => (
                    <SelectItem key={pd.id} value={String(pd.id)}>
                      {pd.displayName} ({pd.name})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {selectedPd && (
              <div className="rounded-lg border bg-muted/30 p-3 space-y-2 text-sm">
                <div>
                  <span className="text-muted-foreground">{t("node:processDefs.startCommand")}:</span>
                  <p className="font-mono text-xs mt-0.5 break-all">{selectedPd.startCommand}</p>
                </div>
                {selectedPd.stopCommand && (
                  <div>
                    <span className="text-muted-foreground">{t("node:processDefs.stopCommand")}:</span>
                    <p className="font-mono text-xs mt-0.5 break-all">{selectedPd.stopCommand}</p>
                  </div>
                )}
                {selectedPd.reloadCommand && (
                  <div>
                    <span className="text-muted-foreground">{t("node:processDefs.reloadCommand")}:</span>
                    <p className="font-mono text-xs mt-0.5 break-all">{selectedPd.reloadCommand}</p>
                  </div>
                )}
              </div>
            )}

            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("node:nodes.overrideVars")}</label>
              <Textarea
                placeholder={t("node:nodes.overrideVarsPlaceholder")}
                rows={4}
                className="font-mono text-sm"
                value={overrideVarsText}
                onChange={(e) => setOverrideVarsText(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">{t("node:nodes.overrideVarsHint")}</p>
            </div>
          </div>
          <SheetFooter>
            <Button
              size="sm"
              disabled={!selectedProcessDef || bindMutation.isPending}
              onClick={() => {
                let overrideVars: Record<string, unknown> | undefined
                if (overrideVarsText.trim()) {
                  try {
                    overrideVars = JSON.parse(overrideVarsText)
                  } catch {
                    toast.error(t("node:nodes.overrideVarsInvalid"))
                    return
                  }
                }
                bindMutation.mutate({ processDefId: Number(selectedProcessDef), overrideVars })
              }}
            >
              {bindMutation.isPending && <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />}
              {t("node:nodes.bindProcess")}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      {/* Token display after rotation */}
      <Dialog open={tokenDialogOpen} onOpenChange={setTokenDialogOpen}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{t("node:nodes.tokenTitle")}</DialogTitle>
            <DialogDescription>{t("node:nodes.tokenDesc")}</DialogDescription>
          </DialogHeader>
          <div className="relative">
            <pre className="rounded-lg bg-muted p-3 pr-10 text-xs font-mono break-all whitespace-pre-wrap">{newToken}</pre>
            <Button
              variant="ghost"
              size="icon"
              className="absolute right-1 top-1 h-7 w-7"
              onClick={handleCopyToken}
            >
              {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
