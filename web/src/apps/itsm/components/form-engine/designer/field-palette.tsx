import { useTranslation } from "react-i18next"
import {
  Type, AlignLeft, Hash, Mail, Link, ChevronDown, ListChecks,
  CircleDot, CheckSquare, ToggleLeft, Calendar, CalendarRange,
  User, Building2, FileText,
} from "lucide-react"
import { cn } from "@/lib/utils"
import type { FieldType } from "../types"

interface FieldTypePaletteProps {
  onAddField: (type: FieldType) => void
}

interface PaletteItem {
  type: FieldType
  icon: React.ElementType
}

const groups: { key: string; items: PaletteItem[] }[] = [
  {
    key: "basic",
    items: [
      { type: "text", icon: Type },
      { type: "textarea", icon: AlignLeft },
      { type: "number", icon: Hash },
      { type: "email", icon: Mail },
      { type: "url", icon: Link },
    ],
  },
  {
    key: "selection",
    items: [
      { type: "select", icon: ChevronDown },
      { type: "multi_select", icon: ListChecks },
      { type: "radio", icon: CircleDot },
      { type: "checkbox", icon: CheckSquare },
      { type: "switch", icon: ToggleLeft },
    ],
  },
  {
    key: "datetime",
    items: [
      { type: "date", icon: Calendar },
      { type: "datetime", icon: Calendar },
      { type: "date_range", icon: CalendarRange },
    ],
  },
  {
    key: "advanced",
    items: [
      { type: "user_picker", icon: User },
      { type: "dept_picker", icon: Building2 },
      { type: "rich_text", icon: FileText },
    ],
  },
]

export function FieldTypePalette({ onAddField }: FieldTypePaletteProps) {
  const { t } = useTranslation("itsm")

  return (
    <div className="space-y-4">
      {groups.map((group) => (
        <div key={group.key}>
          <h4 className="text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wide">
            {t(`forms.fieldGroup.${group.key}`)}
          </h4>
          <div className="grid grid-cols-2 gap-1.5">
            {group.items.map(({ type, icon: Icon }) => (
              <button
                key={type}
                type="button"
                onClick={() => onAddField(type)}
                className={cn(
                  "flex items-center gap-1.5 rounded-md border px-2 py-1.5 text-xs",
                  "hover:bg-accent hover:text-accent-foreground transition-colors",
                  "text-left cursor-pointer",
                )}
              >
                <Icon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                <span className="truncate">{t(`forms.type.${type}`)}</span>
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
