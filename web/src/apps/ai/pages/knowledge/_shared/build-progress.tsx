import { useState, useEffect } from "react"
import { Loader2 } from "lucide-react"
import { Progress } from "@/components/ui/progress"
import type { BuildProgress as BuildProgressType } from "./types"

const stageLabels: Record<string, string> = {
  preparing: "准备中",
  extracting: "提取内容",
  chunking: "分块处理",
  calling_llm: "LLM 处理",
  writing_nodes: "写入节点",
  generating_embeddings: "生成向量",
  completed: "已完成",
  idle: "空闲",
}

interface BuildProgressProps {
  buildProgress: BuildProgressType | null
}

export function BuildProgressDisplay({ buildProgress }: BuildProgressProps) {
  const [elapsedSeconds, setElapsedSeconds] = useState(0)

  const isActive = buildProgress && buildProgress.stage !== "idle" && buildProgress.stage !== "completed"

  useEffect(() => {
    if (isActive && buildProgress?.startedAt) {
      const update = () => {
        setElapsedSeconds(Math.floor(Date.now() / 1000) - buildProgress.startedAt)
      }
      update()
      const interval = setInterval(update, 1000)
      return () => clearInterval(interval)
    }
    setElapsedSeconds(0)
  }, [isActive, buildProgress?.startedAt])

  if (!buildProgress || !isActive) return null

  const { stage, sources, items, embeddings, currentItem } = buildProgress
  const totalAll = sources.total + items.total + embeddings.total
  const doneAll = sources.done + items.done + embeddings.done
  const percent = totalAll > 0 ? Math.round((doneAll / totalAll) * 100) : 0

  return (
    <div className="space-y-2 rounded-lg border p-3">
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span className="flex items-center gap-1.5">
          <Loader2 className="h-3 w-3 animate-spin" />
          {stageLabels[stage] ?? stage}
          {elapsedSeconds > 0 && (
            <span className="text-muted-foreground/70">({elapsedSeconds}s)</span>
          )}
        </span>
        <span>{doneAll}/{totalAll}</span>
      </div>
      <Progress value={percent} className="h-1.5" />
      <div className="flex gap-4 text-xs text-muted-foreground">
        <span>素材 {sources.done}/{sources.total}</span>
        <span>条目 {items.done}/{items.total}</span>
        <span>向量 {embeddings.done}/{embeddings.total}</span>
      </div>
      {currentItem && (
        <p className="text-xs text-muted-foreground truncate">当前: {currentItem}</p>
      )}
    </div>
  )
}
