import { useTranslation } from "react-i18next"
import { useQuery } from "@tanstack/react-query"
import { api, type PaginatedResponse } from "@/lib/api"
import { Checkbox } from "@/components/ui/checkbox"
import { Loader2 } from "lucide-react"

interface BindingItem {
  id: number
  name: string
  displayName?: string
  description?: string
}

interface BindingCheckboxListProps {
  title: string
  queryKey: string[]
  endpoint: string
  value: number[]
  onChange: (ids: number[]) => void
}

export function BindingCheckboxList({ title, queryKey, endpoint, value, onChange }: BindingCheckboxListProps) {
  const { t } = useTranslation(["ai"])

  const { data: items = [], isLoading } = useQuery({
    queryKey,
    queryFn: () =>
      api.get<PaginatedResponse<BindingItem>>(endpoint).then((r) => r?.items ?? []),
  })

  function toggle(id: number) {
    if (value.includes(id)) {
      onChange(value.filter((v) => v !== id))
    } else {
      onChange([...value, id])
    }
  }

  return (
    <div className="space-y-2">
      <p className="text-sm font-medium">{title}</p>
      <div className="rounded-md border max-h-48 overflow-y-auto">
        {isLoading ? (
          <div className="flex items-center justify-center py-6">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        ) : items.length === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground">
            {t("ai:agents.noItems")}
          </p>
        ) : (
          items.map((item) => (
            <label
              key={item.id}
              className="flex items-center gap-3 px-3 py-2 hover:bg-muted/50 cursor-pointer"
            >
              <Checkbox
                checked={value.includes(item.id)}
                onCheckedChange={() => toggle(item.id)}
              />
              <div className="min-w-0 flex-1">
                <span className="text-sm">{item.displayName || item.name}</span>
                {item.description && (
                  <p className="text-xs text-muted-foreground truncate">{item.description}</p>
                )}
              </div>
            </label>
          ))
        )}
      </div>
    </div>
  )
}
