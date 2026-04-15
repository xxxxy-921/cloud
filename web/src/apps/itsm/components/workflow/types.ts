// Shared types for workflow editor and viewer
export const NODE_TYPES = [
  "start", "end", "form", "approve", "process", "action", "exclusive", "notify", "wait",
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
  operator: "equals" | "not_equals" | "contains_any" | "gt" | "lt" | "gte" | "lte"
  value: unknown
}

export interface WFNodeData {
  label: string
  nodeType: NodeType
  // form / approve / process
  participants?: Participant[]
  formSchema?: unknown
  // approve
  executionMode?: "single" | "parallel" | "sequential"
  // action
  actionId?: number
  // exclusive gateway
  // (conditions are on edges)
  // notify
  channelType?: string
  template?: string
  // wait
  waitMode?: "signal" | "timer"
  duration?: string // e.g. "PT1H"
}

export interface WFEdgeData {
  outcome?: string
  isDefault?: boolean
  condition?: GatewayCondition
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
}
