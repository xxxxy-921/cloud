import { useState } from "react"
import { Monitor, LogOut } from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { parseUserAgent } from "@/lib/ua-parser"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  DataTableActionsCell,
  DataTableActionsHead,
  DataTableCard,
  DataTableEmptyRow,
  DataTableLoadingRow,
  DataTablePagination,
} from "@/components/ui/data-table"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { formatDateTime } from "@/lib/utils"
import { KickDialog } from "./kick-dialog"

interface Session {
  id: number
  userId: number
  username: string
  ipAddress: string
  userAgent: string
  loginAt: string
  lastSeenAt: string
  isCurrent: boolean
}

export function Component() {
  const [kickTarget, setKickTarget] = useState<{ id: number; username: string } | null>(null)
  const canDelete = usePermission("system:session:delete")

  const {
    page, setPage,
    items: sessions, total, totalPages, isLoading,
  } = useListPage<Session>({ queryKey: "sessions", endpoint: "/api/v1/sessions" })

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">会话管理</h2>
        <span className="text-sm text-muted-foreground">共 {total} 个活跃会话</span>
      </div>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-[160px]">用户</TableHead>
              <TableHead className="w-[140px]">IP 地址</TableHead>
              <TableHead className="min-w-[240px]">设备</TableHead>
              <TableHead className="w-[150px]">登录时间</TableHead>
              <TableHead className="w-[150px]">最后活跃</TableHead>
              <DataTableActionsHead className="min-w-[96px]">操作</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={6} />
            ) : sessions.length === 0 ? (
              <DataTableEmptyRow colSpan={6} icon={Monitor} title="暂无活跃会话" />
            ) : (
              sessions.map((session) => (
                <TableRow key={session.id}>
                  <TableCell className="font-medium">
                    <div className="flex items-center gap-2">
                      {session.username}
                      {session.isCurrent && (
                        <Badge variant="outline" className="text-xs">当前</Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="font-mono text-sm">{session.ipAddress || "-"}</TableCell>
                  <TableCell className="max-w-[280px] text-sm text-muted-foreground">
                    <span className="block truncate" title={parseUserAgent(session.userAgent)}>
                      {parseUserAgent(session.userAgent)}
                    </span>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                    {formatDateTime(session.loginAt)}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                    {formatDateTime(session.lastSeenAt)}
                  </TableCell>
                  <DataTableActionsCell className="text-center">
                    {canDelete && !session.isCurrent && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="px-2.5 text-destructive hover:text-destructive"
                        onClick={() => setKickTarget({ id: session.id, username: session.username })}
                      >
                        <LogOut className="mr-1 h-3.5 w-3.5" />
                        踢出
                      </Button>
                    )}
                  </DataTableActionsCell>
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

      <KickDialog
        open={!!kickTarget}
        onOpenChange={(open) => !open && setKickTarget(null)}
        sessionId={kickTarget?.id ?? null}
        username={kickTarget?.username ?? ""}
      />
    </div>
  )
}
