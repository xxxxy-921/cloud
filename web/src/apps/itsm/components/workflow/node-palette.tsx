import { useTranslation } from "react-i18next"
import {
  Play, Square, FileText, ShieldCheck, Wrench, Zap, GitBranch, Bell, Clock,
} from "lucide-react"
import { NODE_TYPES, NODE_COLORS, type NodeType } from "./types"

const ICONS: Record<string, typeof Play> = {
  start: Play, end: Square, form: FileText, approve: ShieldCheck,
  process: Wrench, action: Zap, exclusive: GitBranch, notify: Bell, wait: Clock,
}

export function NodePalette() {
  const { t } = useTranslation("itsm")

  function onDragStart(event: React.DragEvent, nodeType: NodeType) {
    event.dataTransfer.setData("application/reactflow-nodetype", nodeType)
    event.dataTransfer.effectAllowed = "move"
  }

  return (
    <div className="flex w-[160px] flex-col gap-1 border-r bg-muted/30 p-3">
      <div className="mb-1 text-xs font-medium text-muted-foreground">{t("workflow.nodeTypes")}</div>
      {NODE_TYPES.map((nt) => {
        const Icon = ICONS[nt]
        const color = NODE_COLORS[nt]
        return (
          <div
            key={nt}
            draggable
            onDragStart={(e) => onDragStart(e, nt)}
            className="flex cursor-grab items-center gap-2 rounded-md border bg-background px-2 py-1.5 text-sm hover:border-primary/50 active:cursor-grabbing"
          >
            <Icon size={14} style={{ color }} />
            <span>{t(`workflow.node.${nt}`)}</span>
          </div>
        )
      })}
    </div>
  )
}
