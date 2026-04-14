import { useTranslation } from "react-i18next"
import { Search, X } from "lucide-react"

interface TopologyToolbarProps {
  searchQuery: string
  onSearchChange: (query: string) => void
  errorOnly: boolean
  onErrorOnlyChange: (value: boolean) => void
}

export function TopologyToolbar({ searchQuery, onSearchChange, errorOnly, onErrorOnlyChange }: TopologyToolbarProps) {
  const { t } = useTranslation("apm")

  return (
    <div className="flex items-center gap-2">
      {/* Search input */}
      <div className="relative">
        <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder={t("topology.search.placeholder")}
          className="h-8 w-40 rounded-lg border bg-muted/30 pl-7 pr-7 text-xs outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground/60"
        />
        {searchQuery && (
          <button
            onClick={() => onSearchChange("")}
            className="absolute right-1.5 top-1/2 -translate-y-1/2 p-0.5 rounded hover:bg-muted/80 text-muted-foreground"
          >
            <X className="w-3 h-3" />
          </button>
        )}
      </div>

      {/* Error-only toggle */}
      <button
        onClick={() => onErrorOnlyChange(!errorOnly)}
        className={`flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs font-medium transition-colors ${
          errorOnly
            ? "bg-red-500/10 border-red-500/25 text-red-500"
            : "bg-muted/30 border-border text-muted-foreground hover:bg-muted/50"
        }`}
      >
        <span className={`h-1.5 w-1.5 rounded-full ${errorOnly ? "bg-red-500" : "bg-muted-foreground/40"}`} />
        {t("topology.search.errorOnly")}
      </button>
    </div>
  )
}
