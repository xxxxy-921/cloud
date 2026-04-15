"use client"

import { useState, useCallback, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useParams, useNavigate } from "react-router-dom"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { ArrowLeft, Eye, EyeOff, Save, Loader2, Plus } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { ScrollArea } from "@/components/ui/scroll-area"
import { fetchFormDef, updateFormDef } from "../../api"
import { FormRenderer } from "../../components/form-engine"
import { FieldTypePalette } from "../../components/form-engine/designer/field-palette"
import { DesignerCanvas } from "../../components/form-engine/designer/designer-canvas"
import { FieldPropertyEditor } from "../../components/form-engine/designer/field-property-editor"
import type { FormField, FormSchema, FormLayout, LayoutSection, FieldType } from "../../components/form-engine/types"

// ─── Helpers ────────────────────────────────────────────

function generateKey(type: FieldType, existingKeys: Set<string>): string {
  let idx = 1
  let key = `${type}_${idx}`
  while (existingKeys.has(key)) {
    idx++
    key = `${type}_${idx}`
  }
  return key
}

function createField(type: FieldType, key: string): FormField {
  const base: FormField = {
    key,
    type,
    label: key,
    required: false,
  }
  // Add default options for selection types
  if (["select", "multi_select", "radio", "checkbox"].includes(type)) {
    base.options = [
      { label: "Option 1", value: "opt_1" },
      { label: "Option 2", value: "opt_2" },
    ]
  }
  return base
}

// ─── Main Page ──────────────────────────────────────────

export function Component() {
  const { t } = useTranslation(["itsm", "common"])
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const formId = Number(id)

  // Editor state
  const [fields, setFields] = useState<FormField[]>([])
  const [layout, setLayout] = useState<FormLayout | null>(null)
  const [selectedFieldKey, setSelectedFieldKey] = useState<string | null>(null)
  const [isPreview, setIsPreview] = useState(false)
  const [formName, setFormName] = useState("")
  const [columns, setColumns] = useState<1 | 2 | 3>(1)

  // Load form definition
  const { isLoading } = useQuery({
    queryKey: ["itsm-form", formId],
    queryFn: () => fetchFormDef(formId),
    enabled: !!formId,
    refetchOnWindowFocus: false,
    select: (data) => {
      if (data) {
        setFormName(data.name)
        try {
          const schema: FormSchema = typeof data.schema === "string"
            ? JSON.parse(data.schema)
            : data.schema as FormSchema
          setFields(schema.fields ?? [])
          setLayout(schema.layout ?? null)
          setColumns(schema.layout?.columns ?? 1)
        } catch {
          setFields([])
          setLayout(null)
        }
      }
      return data
    },
  })

  // Save mutation
  const saveMut = useMutation({
    mutationFn: () => {
      const schema: FormSchema = {
        version: 1,
        fields,
        layout: layout
          ? { ...layout, columns }
          : columns > 1
            ? { columns, sections: [{ title: "Default", fields: fields.map((f) => f.key) }] }
            : null,
      }
      return updateFormDef(formId, { schema: JSON.stringify(schema) })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["itsm-form", formId] })
      queryClient.invalidateQueries({ queryKey: ["itsm-forms"] })
      toast.success(t("itsm:forms.saveSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  // Field operations
  const existingKeys = useMemo(() => new Set(fields.map((f) => f.key)), [fields])

  const handleAddField = useCallback((type: FieldType) => {
    const key = generateKey(type, existingKeys)
    const newField = createField(type, key)
    setFields((prev) => [...prev, newField])
    // Add to first section if layout exists
    setLayout((prev) => {
      if (!prev?.sections?.length) return prev
      const sections = [...prev.sections]
      sections[0] = { ...sections[0], fields: [...sections[0].fields, key] }
      return { ...prev, sections }
    })
    setSelectedFieldKey(key)
  }, [existingKeys])

  const handleDeleteField = useCallback((key: string) => {
    setFields((prev) => prev.filter((f) => f.key !== key))
    setLayout((prev) => {
      if (!prev?.sections) return prev
      return {
        ...prev,
        sections: prev.sections.map((s) => ({
          ...s,
          fields: s.fields.filter((k) => k !== key),
        })),
      }
    })
    if (selectedFieldKey === key) setSelectedFieldKey(null)
  }, [selectedFieldKey])

  const handleMoveField = useCallback((key: string, direction: "up" | "down") => {
    setFields((prev) => {
      const idx = prev.findIndex((f) => f.key === key)
      if (idx < 0) return prev
      const target = direction === "up" ? idx - 1 : idx + 1
      if (target < 0 || target >= prev.length) return prev
      const next = [...prev]
      ;[next[idx], next[target]] = [next[target], next[idx]]
      return next
    })
    // Also update layout section ordering
    setLayout((prev) => {
      if (!prev?.sections) return prev
      return {
        ...prev,
        sections: prev.sections.map((s) => {
          const idx = s.fields.indexOf(key)
          if (idx < 0) return s
          const target = direction === "up" ? idx - 1 : idx + 1
          if (target < 0 || target >= s.fields.length) return s
          const next = [...s.fields]
          ;[next[idx], next[target]] = [next[target], next[idx]]
          return { ...s, fields: next }
        }),
      }
    })
  }, [])

  const handleFieldChange = useCallback((updated: FormField) => {
    setFields((prev) => prev.map((f) => f.key === updated.key ? updated : f))
  }, [])

  const handleAddSection = useCallback(() => {
    setLayout((prev) => {
      const sections: LayoutSection[] = prev?.sections ?? []
      const newSection: LayoutSection = {
        title: `Section ${sections.length + 1}`,
        fields: [],
      }
      return {
        columns: prev?.columns ?? columns,
        sections: [...sections, newSection],
      }
    })
  }, [columns])

  const selectedField = fields.find((f) => f.key === selectedFieldKey) ?? null

  // Preview schema
  const previewSchema: FormSchema = useMemo(() => ({
    version: 1,
    fields,
    layout: layout ? { ...layout, columns } : null,
  }), [fields, layout, columns])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center justify-between border-b px-4 py-2 shrink-0">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="sm" onClick={() => navigate("/itsm/forms")}>
            <ArrowLeft className="mr-1 h-4 w-4" />{t("itsm:forms.backToList")}
          </Button>
          <span className="text-sm font-medium">{formName}</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5">
            <span className="text-xs text-muted-foreground">{t("itsm:forms.columns")}</span>
            <Select value={String(columns)} onValueChange={(v) => setColumns(Number(v) as 1 | 2 | 3)}>
              <SelectTrigger className="h-7 w-16 text-xs"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="1">1</SelectItem>
                <SelectItem value="2">2</SelectItem>
                <SelectItem value="3">3</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button variant="outline" size="sm" onClick={handleAddSection}>
            <Plus className="mr-1 h-3.5 w-3.5" />{t("itsm:forms.addSection")}
          </Button>
          <Button
            variant="outline" size="sm"
            onClick={() => setIsPreview(!isPreview)}
          >
            {isPreview
              ? <><EyeOff className="mr-1 h-3.5 w-3.5" />{t("itsm:forms.exitPreview")}</>
              : <><Eye className="mr-1 h-3.5 w-3.5" />{t("itsm:forms.preview")}</>
            }
          </Button>
          <Button size="sm" onClick={() => saveMut.mutate()} disabled={saveMut.isPending}>
            {saveMut.isPending
              ? <><Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />{t("itsm:forms.saving")}</>
              : <><Save className="mr-1 h-3.5 w-3.5" />{t("itsm:forms.save")}</>
            }
          </Button>
        </div>
      </div>

      {/* Content */}
      {isPreview ? (
        <div className="flex-1 overflow-auto p-6">
          <div className="max-w-3xl mx-auto">
            <FormRenderer
              schema={previewSchema}
              mode="create"
              onSubmit={() => {
                toast.success("Preview submit OK")
              }}
            />
            <div className="mt-4 flex justify-end">
              <Button type="submit" form="form-renderer">
                {t("itsm:forms.preview")} Submit
              </Button>
            </div>
          </div>
        </div>
      ) : (
        <div className="flex flex-1 overflow-hidden">
          {/* Left: Field Palette */}
          <div className="w-56 shrink-0 border-r">
            <ScrollArea className="h-full">
              <div className="p-3">
                <h3 className="text-xs font-medium text-muted-foreground mb-3 uppercase tracking-wide">
                  {t("itsm:forms.fieldType")}
                </h3>
                <FieldTypePalette onAddField={handleAddField} />
              </div>
            </ScrollArea>
          </div>

          {/* Center: Canvas */}
          <div className="flex-1 overflow-auto">
            <ScrollArea className="h-full">
              <div className="p-4">
                <DesignerCanvas
                  fields={fields}
                  layout={layout}
                  selectedFieldKey={selectedFieldKey}
                  onSelectField={setSelectedFieldKey}
                  onDeleteField={handleDeleteField}
                  onMoveField={handleMoveField}
                />
              </div>
            </ScrollArea>
          </div>

          {/* Right: Property Editor */}
          <div className="w-72 shrink-0 border-l">
            <ScrollArea className="h-full">
              <div className="p-3">
                {selectedField ? (
                  <FieldPropertyEditor
                    field={selectedField}
                    allFields={fields}
                    onChange={handleFieldChange}
                  />
                ) : (
                  <p className="text-sm text-muted-foreground text-center py-8">
                    {t("itsm:forms.empty")}
                  </p>
                )}
              </div>
            </ScrollArea>
          </div>
        </div>
      )}
    </div>
  )
}
