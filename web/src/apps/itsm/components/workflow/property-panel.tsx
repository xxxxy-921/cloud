import { useTranslation } from "react-i18next"
import { type Node, type Edge, useReactFlow } from "@xyflow/react"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Trash2, X } from "lucide-react"
import type { WFNodeData, WFEdgeData, NodeType, ConditionGroup } from "./types"
import { ParticipantPicker } from "./panels/participant-picker"
import { FormBindingPicker } from "./panels/form-binding-picker"
import { ConditionBuilder } from "./panels/condition-builder"
import { VariableMappingEditor } from "./panels/variable-mapping-editor"
import { ScriptAssignmentEditor } from "./panels/script-assignment-editor"
import { ActionPicker } from "./panels/action-picker"

// ─── Node Property Panel ────────────────────────────────

interface NodePanelProps {
  node: Node & { data: WFNodeData }
  serviceId?: number
  onClose: () => void
}

export function NodePropertyPanel({ node, serviceId, onClose }: NodePanelProps) {
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

  const hasParticipants = nodeType === "form" || nodeType === "approve" || nodeType === "process"
  const hasFormBinding = nodeType === "form"
  const hasApproveMode = nodeType === "approve"
  const hasAction = nodeType === "action"
  const hasScript = nodeType === "script"
  const hasNotify = nodeType === "notify"
  const hasWait = nodeType === "wait" || nodeType === "timer"
  const hasMapping = nodeType === "form" || nodeType === "approve" || nodeType === "process"
  const isProtected = nodeType === "start" || nodeType === "end"

  return (
    <div className="flex w-[300px] flex-col gap-3 overflow-y-auto border-l bg-muted/30 p-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">{t(`workflow.node.${nodeType}`)}</span>
        <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onClose}><X size={14} /></Button>
      </div>

      {/* Label */}
      <div className="space-y-1">
        <Label className="text-xs">{t("workflow.prop.label")}</Label>
        <Input value={data.label} onChange={(e) => updateData({ label: e.target.value })} className="h-8 text-sm" />
      </div>

      {/* Participants */}
      {hasParticipants && (
        <ParticipantPicker
          participants={data.participants ?? []}
          onChange={(participants) => updateData({ participants })}
        />
      )}

      {/* Approve execution mode */}
      {hasApproveMode && (
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

      {/* Form binding (classic mode) */}
      {hasFormBinding && (
        <FormBindingPicker
          formDefinitionId={data.formDefinitionId}
          onChange={(id, schema) => updateData({ formDefinitionId: id, formSchema: schema })}
        />
      )}

      {/* Inline formSchema read-only display (smart mode / LLM-generated) */}
      {hasFormBinding && !data.formDefinitionId && data.formSchema != null && (
        <InlineFormSchemaView schema={data.formSchema} />
      )}

      {/* Action picker */}
      {hasAction && serviceId && (
        <ActionPicker
          serviceId={serviceId}
          actionId={data.actionId}
          onChange={(actionId) => updateData({ actionId })}
        />
      )}

      {/* Script assignments */}
      {hasScript && (
        <ScriptAssignmentEditor
          assignments={data.scriptAssignments ?? []}
          onChange={(scriptAssignments) => updateData({ scriptAssignments })}
        />
      )}

      {/* Notify */}
      {hasNotify && (
        <>
          <div className="space-y-1">
            <Label className="text-xs">{t("workflow.prop.channelType")}</Label>
            <Select value={data.channelType ?? ""} onValueChange={(v) => updateData({ channelType: v })}>
              <SelectTrigger className="h-8 text-sm"><SelectValue placeholder={t("workflow.prop.selectChannel")} /></SelectTrigger>
              <SelectContent>
                <SelectItem value="email">{t("workflow.channel.email")}</SelectItem>
                <SelectItem value="internal">{t("workflow.channel.internal")}</SelectItem>
                <SelectItem value="webhook">{t("workflow.channel.webhook")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{t("workflow.prop.template")}</Label>
            <Input value={data.template ?? ""} onChange={(e) => updateData({ template: e.target.value })} className="h-8 text-sm" />
          </div>
        </>
      )}

      {/* Wait / Timer */}
      {hasWait && (
        <div className="space-y-1">
          <Label className="text-xs">{t("workflow.prop.waitMode")}</Label>
          <Select value={data.waitMode ?? "signal"} onValueChange={(v) => updateData({ waitMode: v as WFNodeData["waitMode"] })}>
            <SelectTrigger className="h-8 text-sm"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="signal">{t("workflow.prop.waitSignal")}</SelectItem>
              <SelectItem value="timer">{t("workflow.prop.waitTimer")}</SelectItem>
            </SelectContent>
          </Select>
          {(data.waitMode === "timer" || nodeType === "timer") && (
            <div className="mt-1 space-y-1">
              <Label className="text-xs">{t("workflow.prop.duration")}</Label>
              <Input value={data.duration ?? ""} onChange={(e) => updateData({ duration: e.target.value })} placeholder="PT1H" className="h-8 text-sm" />
            </div>
          )}
        </div>
      )}

      {/* Variable mapping */}
      {hasMapping && (
        <>
          <VariableMappingEditor
            label={t("workflow.prop.inputMapping")}
            mappings={data.inputMapping ?? []}
            onChange={(inputMapping) => updateData({ inputMapping })}
            sourceLabel={t("workflow.mapping.variable")}
            targetLabel={t("workflow.mapping.formField")}
          />
          <VariableMappingEditor
            label={t("workflow.prop.outputMapping")}
            mappings={data.outputMapping ?? []}
            onChange={(outputMapping) => updateData({ outputMapping })}
            sourceLabel={t("workflow.mapping.formField")}
            targetLabel={t("workflow.mapping.variable")}
          />
        </>
      )}

      {/* Delete */}
      {!isProtected && (
        <Button variant="destructive" size="sm" className="mt-auto" onClick={handleDelete}>
          <Trash2 className="mr-1.5 h-3.5 w-3.5" />{t("workflow.prop.deleteNode")}
        </Button>
      )}
    </div>
  )
}

// ─── Inline FormSchema (read-only, smart mode) ─────────

interface FormSchemaField {
  key: string
  type: string
  label: string
  options?: string[]
}

function parseInlineFields(schema: unknown): FormSchemaField[] {
  if (!schema || typeof schema !== "object") return []
  const s = schema as { fields?: FormSchemaField[] }
  return Array.isArray(s.fields) ? s.fields : []
}

const FIELD_TYPE_LABELS: Record<string, string> = {
  text: "文本", textarea: "多行文本", select: "选择", number: "数字", date: "日期", checkbox: "复选",
}

function InlineFormSchemaView({ schema }: { schema: unknown }) {
  const { t } = useTranslation("itsm")
  const fields = parseInlineFields(schema)
  if (fields.length === 0) return null

  return (
    <div className="space-y-1.5">
      <Label className="text-xs">{t("workflow.prop.formFields")} ({fields.length})</Label>
      <div className="rounded border p-1.5 space-y-0.5">
        {fields.map((f) => (
          <div key={f.key} className="flex items-center justify-between text-[10px]">
            <span>{f.label || f.key}</span>
            <span className="text-muted-foreground">{FIELD_TYPE_LABELS[f.type] ?? f.type}</span>
          </div>
        ))}
      </div>
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

  const isGateway = sourceNodeType === "exclusive" || sourceNodeType === "parallel" || sourceNodeType === "inclusive"

  return (
    <div className="flex w-[300px] flex-col gap-3 overflow-y-auto border-l bg-muted/30 p-3">
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

      {isGateway && !data.isDefault && (
        <ConditionBuilder
          condition={data.condition}
          onChange={(condition: ConditionGroup | undefined) => updateData({ condition })}
        />
      )}

      <Button variant="destructive" size="sm" className="mt-auto" onClick={handleDelete}>
        <Trash2 className="mr-1.5 h-3.5 w-3.5" />{t("workflow.prop.deleteEdge")}
      </Button>
    </div>
  )
}
