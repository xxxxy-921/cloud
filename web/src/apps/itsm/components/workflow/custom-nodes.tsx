import { memo } from "react"
import { Handle, Position, type NodeProps } from "@xyflow/react"
import {
  Play, Square, FileText, ShieldCheck, Wrench, Zap, GitBranch, Bell, Clock,
} from "lucide-react"
import { type WFNodeData, NODE_COLORS } from "./types"

const ICONS: Record<string, typeof Play> = {
  start: Play,
  end: Square,
  form: FileText,
  approve: ShieldCheck,
  process: Wrench,
  action: Zap,
  exclusive: GitBranch,
  notify: Bell,
  wait: Clock,
}

function WorkflowNode({ data, selected }: NodeProps & { data: WFNodeData }) {
  const nodeType = data.nodeType
  const Icon = ICONS[nodeType] ?? Wrench
  const color = NODE_COLORS[nodeType] ?? "#6b7280"
  const isStart = nodeType === "start"
  const isEnd = nodeType === "end"
  const isGateway = nodeType === "exclusive"

  if (isGateway) {
    return (
      <div className="relative" style={{ width: 56, height: 56 }}>
        {!isStart && <Handle type="target" position={Position.Top} className="!bg-muted-foreground" />}
        <div
          className={`flex h-14 w-14 rotate-45 items-center justify-center rounded-sm border-2 shadow-md ${selected ? "ring-2 ring-primary" : ""}`}
          style={{ borderColor: color, backgroundColor: `${color}25` }}
        >
          <Icon className="-rotate-45" style={{ color }} size={20} />
        </div>
        <div className="absolute -bottom-5 left-1/2 -translate-x-1/2 whitespace-nowrap text-xs font-medium" style={{ color }}>
          {data.label}
        </div>
        {!isEnd && <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground" />}
      </div>
    )
  }

  // Start / End use pill shape with solid fill
  if (isStart || isEnd) {
    return (
      <div
        className={`flex min-w-[100px] items-center justify-center gap-2 rounded-full border-2 px-4 py-2 shadow-md ${selected ? "ring-2 ring-primary" : ""}`}
        style={{ borderColor: color, backgroundColor: `${color}18` }}
      >
        {!isStart && <Handle type="target" position={Position.Top} className="!bg-muted-foreground" />}
        <Icon style={{ color }} size={16} />
        <span className="text-sm font-semibold" style={{ color }}>{data.label}</span>
        {!isEnd && <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground" />}
      </div>
    )
  }

  return (
    <div
      className={`flex min-w-[140px] items-center gap-2.5 rounded-lg border-2 px-3 py-2.5 shadow-md ${selected ? "ring-2 ring-primary" : ""}`}
      style={{ borderColor: color, backgroundColor: `${color}0a` }}
    >
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground" />
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg" style={{ backgroundColor: `${color}28` }}>
        <Icon style={{ color }} size={17} />
      </div>
      <span className="text-sm font-medium">{data.label}</span>
      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground" />
    </div>
  )
}

export const CustomNode = memo(WorkflowNode)

export const nodeTypes = {
  workflow: CustomNode,
}
