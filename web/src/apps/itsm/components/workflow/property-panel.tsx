import { useTranslation } from "react-i18next"
import { type Node, type Edge, useReactFlow } from "@xyflow/react"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Trash2, X } from "lucide-react"
import type { WFNodeData, WFEdgeData, NodeType, GatewayCondition } from "./types"

// ─── Node Property Panel ────────────────────────────────

interface NodePanelProps {
  node: Node & { data: WFNodeData }
  onClose: () => void
}

export function NodePropertyPanel({ node, onClose }: NodePanelProps) {
  const { t } = useTranslation("itsm")
  const { setNodes, deleteElements } = useReactFlow()
  const data = node.data
  const nodeType = data.nodeType as NodeType

  function updateData(patch: Partial<WFNodeData>) {
    setNodes((nds) => nds.map((n) => n.id === node.id ? { ...n, data: { ...n.data, ...patch } } : n))
  }

  function handleDelete() {
    deleteElements({ nodes: [{ id: node.id }] })
    onClose()
  }

  return (
    <div className="flex w-[280px] flex-col gap-3 border-l bg-muted/30 p-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">{t(`workflow.node.${nodeType}`)}</span>
        <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onClose}><X size={14} /></Button>
      </div>

      <div className="space-y-1">
        <Label className="text-xs">{t("workflow.prop.label")}</Label>
        <Input value={data.label} onChange={(e) => updateData({ label: e.target.value })} className="h-8 text-sm" />
      </div>

      {(nodeType === "form" || nodeType === "approve" || nodeType === "process") && (
        <div className="space-y-1">
          <Label className="text-xs">{t("workflow.prop.participants")}</Label>
          <p className="text-xs text-muted-foreground">{t("workflow.prop.participantsHint")}</p>
        </div>
      )}

      {nodeType === "approve" && (
        <div className="space-y-1">
          <Label className="text-xs">{t("workflow.prop.executionMode")}</Label>
          <Select value={data.executionMode ?? "single"} onValueChange={(v) => updateData({ executionMode: v as WFNodeData["executionMode"] })}>
            <SelectTrigger className="h-8 text-sm"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="single">{t("workflow.prop.modeSingle")}</SelectItem>
              <SelectItem value="parallel">{t("workflow.prop.modeParallel")}</SelectItem>
              <SelectItem value="sequential">{t("workflow.prop.modeSequential")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      )}

      {nodeType === "wait" && (
        <div className="space-y-1">
          <Label className="text-xs">{t("workflow.prop.waitMode")}</Label>
          <Select value={data.waitMode ?? "signal"} onValueChange={(v) => updateData({ waitMode: v as WFNodeData["waitMode"] })}>
            <SelectTrigger className="h-8 text-sm"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="signal">{t("workflow.prop.waitSignal")}</SelectItem>
              <SelectItem value="timer">{t("workflow.prop.waitTimer")}</SelectItem>
            </SelectContent>
          </Select>
          {data.waitMode === "timer" && (
            <div className="mt-1 space-y-1">
              <Label className="text-xs">{t("workflow.prop.duration")}</Label>
              <Input value={data.duration ?? ""} onChange={(e) => updateData({ duration: e.target.value })} placeholder="PT1H" className="h-8 text-sm" />
            </div>
          )}
        </div>
      )}

      {nodeType !== "start" && nodeType !== "end" && (
        <Button variant="destructive" size="sm" className="mt-auto" onClick={handleDelete}>
          <Trash2 className="mr-1.5 h-3.5 w-3.5" />{t("workflow.prop.deleteNode")}
        </Button>
      )}
    </div>
  )
}

// ─── Edge Property Panel ────────────────────────────────

interface EdgePanelProps {
  edge: Edge & { data?: WFEdgeData }
  sourceNodeType?: NodeType
  onClose: () => void
}

export function EdgePropertyPanel({ edge, sourceNodeType, onClose }: EdgePanelProps) {
  const { t } = useTranslation("itsm")
  const { setEdges, deleteElements } = useReactFlow()
  const data = (edge.data ?? {}) as WFEdgeData

  function updateData(patch: Partial<WFEdgeData>) {
    setEdges((eds) => eds.map((e) => e.id === edge.id ? { ...e, data: { ...e.data, ...patch } } : e))
  }

  function handleDelete() {
    deleteElements({ edges: [{ id: edge.id }] })
    onClose()
  }

  return (
    <div className="flex w-[280px] flex-col gap-3 border-l bg-muted/30 p-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">{t("workflow.prop.edge")}</span>
        <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onClose}><X size={14} /></Button>
      </div>

      <div className="space-y-1">
        <Label className="text-xs">{t("workflow.prop.outcome")}</Label>
        <Input value={data.outcome ?? ""} onChange={(e) => updateData({ outcome: e.target.value })} placeholder="e.g. approved" className="h-8 text-sm" />
      </div>

      <div className="flex items-center gap-2">
        <Switch checked={data.isDefault ?? false} onCheckedChange={(v) => updateData({ isDefault: v })} />
        <Label className="text-xs">{t("workflow.prop.defaultEdge")}</Label>
      </div>

      {sourceNodeType === "exclusive" && !data.isDefault && (
        <div className="space-y-2 rounded-md border p-2">
          <Label className="text-xs font-medium">{t("workflow.prop.condition")}</Label>
          <div className="space-y-1">
            <Label className="text-xs">{t("workflow.prop.condField")}</Label>
            <Input value={data.condition?.field ?? ""} onChange={(e) => updateData({ condition: { ...data.condition, field: e.target.value, operator: data.condition?.operator ?? "equals", value: data.condition?.value ?? "" } })} className="h-8 text-sm" />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{t("workflow.prop.condOperator")}</Label>
            <Select value={data.condition?.operator ?? "equals"} onValueChange={(v) => updateData({ condition: { ...data.condition!, operator: v as GatewayCondition["operator"] } })}>
              <SelectTrigger className="h-8 text-sm"><SelectValue /></SelectTrigger>
              <SelectContent>
                {["equals", "not_equals", "contains_any", "gt", "lt", "gte", "lte"].map((op) => (
                  <SelectItem key={op} value={op}>{op}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{t("workflow.prop.condValue")}</Label>
            <Input value={String(data.condition?.value ?? "")} onChange={(e) => updateData({ condition: { ...data.condition!, value: e.target.value } })} className="h-8 text-sm" />
          </div>
        </div>
      )}

      <Button variant="destructive" size="sm" className="mt-auto" onClick={handleDelete}>
        <Trash2 className="mr-1.5 h-3.5 w-3.5" />{t("workflow.prop.deleteEdge")}
      </Button>
    </div>
  )
}
