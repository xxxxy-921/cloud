import { useTranslation } from "react-i18next"
import { GripVertical, Trash2, ChevronUp, ChevronDown } from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import type { FormField, FormLayout } from "../types"

interface DesignerCanvasProps {
  fields: FormField[]
  layout: FormLayout | null
  selectedFieldKey: string | null
  onSelectField: (key: string | null) => void
  onDeleteField: (key: string) => void
  onMoveField: (key: string, direction: "up" | "down") => void
}

export function DesignerCanvas({
  fields,
  layout,
  selectedFieldKey,
  onSelectField,
  onDeleteField,
  onMoveField,
}: DesignerCanvasProps) {
  const { t } = useTranslation("itsm")

  if (fields.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-64 border-2 border-dashed rounded-lg text-muted-foreground">
        <p className="text-sm">{t("forms.empty")}</p>
        <p className="text-xs mt-1">{t("forms.emptyHint")}</p>
      </div>
    )
  }

  // Render fields as section groups or flat list
  if (layout?.sections && layout.sections.length > 0) {
    return (
      <div className="space-y-4">
        {layout.sections.map((section, si) => {
          const sectionFields = section.fields
            .map((key) => fields.find((f) => f.key === key))
            .filter(Boolean) as FormField[]

          return (
            <div key={si} className="border rounded-lg">
              <div className="px-3 py-2 bg-muted/50 border-b rounded-t-lg">
                <h4 className="text-sm font-medium">{section.title}</h4>
                {section.description && (
                  <p className="text-xs text-muted-foreground">{section.description}</p>
                )}
              </div>
              <div className="p-2 space-y-1">
                {sectionFields.map((field) => (
                  <FieldCard
                    key={field.key}
                    field={field}
                    isSelected={selectedFieldKey === field.key}
                    onSelect={() => onSelectField(field.key)}
                    onDelete={() => onDeleteField(field.key)}
                    onMoveUp={() => onMoveField(field.key, "up")}
                    onMoveDown={() => onMoveField(field.key, "down")}
                    t={t}
                  />
                ))}
                {sectionFields.length === 0 && (
                  <p className="text-xs text-muted-foreground py-3 text-center">
                    {t("forms.empty")}
                  </p>
                )}
              </div>
            </div>
          )
        })}
      </div>
    )
  }

  // Flat list
  return (
    <div className="space-y-1">
      {fields.map((field) => (
        <FieldCard
          key={field.key}
          field={field}
          isSelected={selectedFieldKey === field.key}
          onSelect={() => onSelectField(field.key)}
          onDelete={() => onDeleteField(field.key)}
          onMoveUp={() => onMoveField(field.key, "up")}
          onMoveDown={() => onMoveField(field.key, "down")}
          t={t}
        />
      ))}
    </div>
  )
}

function FieldCard({
  field,
  isSelected,
  onSelect,
  onDelete,
  onMoveUp,
  onMoveDown,
  t,
}: {
  field: FormField
  isSelected: boolean
  onSelect: () => void
  onDelete: () => void
  onMoveUp: () => void
  onMoveDown: () => void
  t: (key: string) => string
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-md border px-3 py-2 cursor-pointer transition-colors",
        isSelected
          ? "border-primary bg-primary/5 ring-1 ring-primary/20"
          : "hover:bg-accent/50",
      )}
      onClick={onSelect}
    >
      <GripVertical className="h-4 w-4 shrink-0 text-muted-foreground" />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium truncate">{field.label}</span>
          <span className="text-xs text-muted-foreground">{field.key}</span>
        </div>
        <span className="text-xs text-muted-foreground">
          {t(`forms.type.${field.type}`)}
          {field.required && <span className="text-destructive ml-1">*</span>}
        </span>
      </div>
      {isSelected && (
        <div className="flex items-center gap-0.5" onClick={(e) => e.stopPropagation()}>
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onMoveUp}>
            <ChevronUp className="h-3.5 w-3.5" />
          </Button>
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onMoveDown}>
            <ChevronDown className="h-3.5 w-3.5" />
          </Button>
          <Button variant="ghost" size="icon" className="h-6 w-6 text-destructive hover:text-destructive" onClick={onDelete}>
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      )}
    </div>
  )
}
