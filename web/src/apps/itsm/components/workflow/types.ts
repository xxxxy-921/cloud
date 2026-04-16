// Shared types for workflow editor and viewer
export const NODE_TYPES = [
  "start", "end", "form", "approve", "process", "action", "exclusive", "notify", "wait",
  "timer", "signal", "parallel", "inclusive", "subprocess", "script",
] as const

export type NodeType = (typeof NODE_TYPES)[number]

export interface Participant {
  type: string // "user" | "position" | "department" | "position_department" | "requester_manager"
  id?: string | number
  name?: string
  value?: string
  // position_department fields (LLM output)
  department_code?: string
  position_code?: string
}

export interface GatewayCondition {
  field: string
  operator: "equals" | "not_equals" | "contains_any" | "gt" | "lt" | "gte" | "lte" | "is_empty" | "is_not_empty"
  value: unknown
}

export interface SimpleCondition {
  field: string
  operator: GatewayCondition["operator"]
  value: unknown
}

export interface ConditionGroup {
  logic: "and" | "or"
  conditions: Array<SimpleCondition | ConditionGroup>
}

export interface VariableMapping {
  source: string
  target: string
}

export interface ScriptAssignment {
  variable: string
  expression: string
}

export interface WFNodeData {
  label: string
  nodeType: NodeType
  // form / approve / process
  participants?: Participant[]
  formSchema?: unknown
  formDefinitionId?: number
  // approve
  executionMode?: "single" | "parallel" | "sequential"
  // action
  actionId?: number
  // exclusive gateway
  // (conditions are on edges)
  // notify
  channelType?: string
  template?: string
  // wait / timer
  waitMode?: "signal" | "timer"
  duration?: string // e.g. "PT1H"
  // variable mapping
  inputMapping?: VariableMapping[]
  outputMapping?: VariableMapping[]
  // script
  scriptAssignments?: ScriptAssignment[]
  // subprocess
  subprocessJson?: unknown
  subprocessExpanded?: boolean
}

export interface WFEdgeData {
  outcome?: string
  isDefault?: boolean
  condition?: GatewayCondition | ConditionGroup
}

export const NODE_COLORS: Record<NodeType, string> = {
  start: "#22c55e",
  end: "#ef4444",
  form: "#3b82f6",
  approve: "#f59e0b",
  process: "#8b5cf6",
  action: "#06b6d4",
  exclusive: "#f97316",
  notify: "#ec4899",
  wait: "#6366f1",
  timer: "#6366f1",
  signal: "#6366f1",
  parallel: "#14b8a6",
  inclusive: "#eab308",
  subprocess: "#64748b",
  script: "#475569",
}

// Helpers for condition format migration
export function isConditionGroup(c: GatewayCondition | ConditionGroup | undefined): c is ConditionGroup {
  return !!c && "logic" in c
}

export function toConditionGroup(c: GatewayCondition | ConditionGroup | undefined): ConditionGroup | undefined {
  if (!c) return undefined
  if (isConditionGroup(c)) return c
  return { logic: "and", conditions: [{ field: c.field, operator: c.operator, value: c.value }] }
}

const OP_LABELS: Record<string, string> = {
  equals: "=", not_equals: "≠", contains_any: "∈",
  gt: ">", lt: "<", gte: "≥", lte: "≤",
  is_empty: "为空", is_not_empty: "非空",
}

function shortField(f: string): string {
  return f.replace(/^form\./, "")
}

function shortValue(v: unknown): string {
  const s = String(v ?? "")
  return s.length > 16 ? `${s.slice(0, 16)}…` : s
}

function formatSingleCondition(c: { field: string; operator: string; value: unknown }): string {
  const op = OP_LABELS[c.operator] ?? c.operator
  if (c.operator === "is_empty" || c.operator === "is_not_empty") {
    return `${shortField(c.field)} ${op}`
  }
  return `${shortField(c.field)} ${op} ${shortValue(c.value)}`
}

export function conditionSummary(c: GatewayCondition | ConditionGroup | undefined): string {
  if (!c) return ""
  if (!isConditionGroup(c)) return formatSingleCondition(c)
  return c.conditions
    .map((item) => isConditionGroup(item) ? `(${conditionSummary(item)})` : formatSingleCondition(item))
    .join(c.logic === "and" ? " 且 " : " 或 ")
}
