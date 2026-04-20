import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Check } from "lucide-react"
import { api, type PaginatedResponse } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet"
import { formatBytes, formatDateTime } from "@/lib/utils"
import { cn } from "@/lib/utils"
import type { SourcePoolItem } from "./types"

interface SourcePickerSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  assetId: number
  addSourcesEndpoint: string
  onSuccess?: () => void
}

export function SourcePickerSheet({
  open,
  onOpenChange,
  assetId,
  addSourcesEndpoint,
  onSuccess,
}: SourcePickerSheetProps) {
  const queryClient = useQueryClient()
  const [selected, setSelected] = useState<Set<number>>(new Set())

  const { data, isLoading } = useQuery({
    queryKey: ["ai-knowledge-source-pool"],
    queryFn: () =>
      api.get<PaginatedResponse<SourcePoolItem>>("/api/v1/ai/knowledge/sources?pageSize=200"),
    enabled: open,
  })

  const sources = data?.items ?? []

  const addMutation = useMutation({
    mutationFn: () =>
      api.post(addSourcesEndpoint, { sourceIds: Array.from(selected) }),
    onSuccess: () => {
      toast.success("素材添加成功")
      setSelected(new Set())
      queryClient.invalidateQueries({ queryKey: ["ai-asset-sources", assetId] })
      onSuccess?.()
      onOpenChange(false)
    },
    onError: (err) => toast.error(err.message),
  })

  function toggleSelect(id: number) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>添加素材</SheetTitle>
          <SheetDescription>从素材库中选择要关联的素材</SheetDescription>
        </SheetHeader>
        <div className="flex flex-col gap-2 px-4 py-4">
          {isLoading ? (
            <p className="text-sm text-muted-foreground py-8 text-center">加载中...</p>
          ) : sources.length === 0 ? (
            <p className="text-sm text-muted-foreground py-8 text-center">暂无可用素材</p>
          ) : (
            sources.map((src) => (
              <button
                key={src.id}
                type="button"
                onClick={() => toggleSelect(src.id)}
                className={cn(
                  "flex items-center gap-3 rounded-lg border p-3 text-left transition-colors",
                  selected.has(src.id)
                    ? "border-primary bg-primary/5"
                    : "hover:bg-muted/50",
                )}
              >
                <div
                  className={cn(
                    "flex h-5 w-5 shrink-0 items-center justify-center rounded border",
                    selected.has(src.id)
                      ? "border-primary bg-primary text-primary-foreground"
                      : "border-muted-foreground/30",
                  )}
                >
                  {selected.has(src.id) && <Check className="h-3.5 w-3.5" />}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{src.title}</p>
                  <p className="text-xs text-muted-foreground">
                    {src.format} · {formatBytes(src.byteSize)} · {formatDateTime(src.createdAt)}
                  </p>
                </div>
              </button>
            ))
          )}
        </div>
        <SheetFooter className="px-4">
          <Button
            size="sm"
            disabled={selected.size === 0 || addMutation.isPending}
            onClick={() => addMutation.mutate()}
          >
            {addMutation.isPending ? "添加中..." : `添加 ${selected.size} 个素材`}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
