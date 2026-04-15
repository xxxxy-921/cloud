import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { fetchTicketVariables, type ProcessVariableItem } from "../api"

function formatValue(item: ProcessVariableItem): string {
  if (item.value === null || item.value === undefined) return "—"
  if (item.valueType === "json") {
    try {
      return JSON.stringify(item.value, null, 2)
    } catch {
      return String(item.value)
    }
  }
  if (item.valueType === "boolean") {
    return item.value === true ? "true" : "false"
  }
  return String(item.value)
}

const TYPE_VARIANT: Record<string, "default" | "secondary" | "outline"> = {
  string: "secondary",
  number: "outline",
  boolean: "outline",
  json: "default",
  date: "secondary",
}

export function VariablesPanel({ ticketId }: { ticketId: number }) {
  const { t } = useTranslation("itsm")

  const { data: variables = [] } = useQuery({
    queryKey: ["itsm-ticket-variables", ticketId],
    queryFn: () => fetchTicketVariables(ticketId),
    enabled: ticketId > 0,
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t("variables.title")}</CardTitle>
      </CardHeader>
      <CardContent>
        {variables.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("variables.empty")}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("variables.key")}</TableHead>
                <TableHead>{t("variables.value")}</TableHead>
                <TableHead>{t("variables.type")}</TableHead>
                <TableHead>{t("variables.source")}</TableHead>
                <TableHead>{t("variables.updatedAt")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {variables.map((v) => (
                <TableRow key={v.id}>
                  <TableCell className="font-mono text-sm">{v.key}</TableCell>
                  <TableCell className="max-w-[300px]">
                    {v.valueType === "json" ? (
                      <pre className="whitespace-pre-wrap break-all rounded bg-muted/50 px-2 py-1 text-xs font-mono">
                        {formatValue(v)}
                      </pre>
                    ) : (
                      <span className="text-sm">{formatValue(v)}</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={TYPE_VARIANT[v.valueType] ?? "secondary"}>
                      {v.valueType}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{v.source}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {new Date(v.updatedAt).toLocaleString()}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}
