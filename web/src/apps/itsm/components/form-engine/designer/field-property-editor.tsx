import { useTranslation } from "react-i18next"
import { Plus, Trash2 } from "lucide-react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Label } from "@/components/ui/label"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import type { FormField, FieldType, ValidationRule, VisibilityCondition } from "../types"

interface FieldPropertyEditorProps {
  field: FormField
  allFields: FormField[]
  onChange: (updated: FormField) => void
}

const NEEDS_OPTIONS: FieldType[] = ["select", "multi_select", "radio", "checkbox"]

const VALIDATION_RULE_TYPES = [
  "required", "minLength", "maxLength", "min", "max", "pattern", "email", "url",
] as const

const OPERATORS = [
  "equals", "not_equals", "in", "not_in", "is_empty", "is_not_empty",
] as const

export function FieldPropertyEditor({ field, allFields, onChange }: FieldPropertyEditorProps) {
  const { t } = useTranslation("itsm")

  function update(patch: Partial<FormField>) {
    onChange({ ...field, ...patch })
  }

  return (
    <div className="space-y-5">
      {/* Basic Properties */}
      <section className="space-y-3">
        <div>
          <Label className="text-xs">{t("forms.fieldKey")}</Label>
          <Input
            className="mt-1"
            value={field.key}
            onChange={(e) => update({ key: e.target.value })}
          />
        </div>
        <div>
          <Label className="text-xs">{t("forms.fieldLabel")}</Label>
          <Input
            className="mt-1"
            value={field.label}
            onChange={(e) => update({ label: e.target.value })}
          />
        </div>
        <div>
          <Label className="text-xs">{t("forms.fieldType")}</Label>
          <Input className="mt-1" value={t(`forms.type.${field.type}`)} disabled />
        </div>
        <div>
          <Label className="text-xs">{t("forms.fieldPlaceholder")}</Label>
          <Input
            className="mt-1"
            value={field.placeholder ?? ""}
            onChange={(e) => update({ placeholder: e.target.value || undefined })}
          />
        </div>
        <div>
          <Label className="text-xs">{t("forms.fieldDescription")}</Label>
          <Input
            className="mt-1"
            value={field.description ?? ""}
            onChange={(e) => update({ description: e.target.value || undefined })}
          />
        </div>
        <div className="flex items-center justify-between">
          <Label className="text-xs">{t("forms.fieldRequired")}</Label>
          <Switch
            checked={!!field.required}
            onCheckedChange={(v) => update({ required: v })}
          />
        </div>
        <div>
          <Label className="text-xs">{t("forms.fieldWidth")}</Label>
          <Select value={field.width ?? "full"} onValueChange={(v) => update({ width: v as "full" | "half" | "third" })}>
            <SelectTrigger className="mt-1"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="full">{t("forms.fieldWidthFull")}</SelectItem>
              <SelectItem value="half">{t("forms.fieldWidthHalf")}</SelectItem>
              <SelectItem value="third">{t("forms.fieldWidthThird")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </section>

      {/* Options (for select/multi_select/radio/checkbox) */}
      {NEEDS_OPTIONS.includes(field.type) && (
        <section className="space-y-2">
          <div className="flex items-center justify-between">
            <Label className="text-xs font-medium">{t("forms.fieldOptions")}</Label>
            <Button
              variant="ghost" size="sm" className="h-6 px-2 text-xs"
              onClick={() => {
                const opts = [...(field.options ?? []), { label: "", value: `opt_${Date.now()}` }]
                update({ options: opts })
              }}
            >
              <Plus className="h-3 w-3 mr-1" />{t("forms.addOption")}
            </Button>
          </div>
          {(field.options ?? []).map((opt, i) => (
            <div key={i} className="flex items-center gap-1.5">
              <Input
                className="flex-1 h-7 text-xs"
                placeholder={t("forms.optionLabel")}
                value={opt.label}
                onChange={(e) => {
                  const opts = [...(field.options ?? [])]
                  opts[i] = { ...opts[i], label: e.target.value }
                  update({ options: opts })
                }}
              />
              <Input
                className="w-24 h-7 text-xs"
                placeholder={t("forms.optionValue")}
                value={String(opt.value)}
                onChange={(e) => {
                  const opts = [...(field.options ?? [])]
                  opts[i] = { ...opts[i], value: e.target.value }
                  update({ options: opts })
                }}
              />
              <Button
                variant="ghost" size="icon" className="h-6 w-6 shrink-0 text-destructive"
                onClick={() => {
                  const opts = (field.options ?? []).filter((_, j) => j !== i)
                  update({ options: opts })
                }}
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>
          ))}
        </section>
      )}

      {/* Validation Rules */}
      <section className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium">{t("forms.validationRules")}</Label>
          <Button
            variant="ghost" size="sm" className="h-6 px-2 text-xs"
            onClick={() => {
              const rules: ValidationRule[] = [...(field.validation ?? []), { rule: "required", message: "" }]
              update({ validation: rules })
            }}
          >
            <Plus className="h-3 w-3 mr-1" />{t("forms.addRule")}
          </Button>
        </div>
        {(field.validation ?? []).map((rule, i) => (
          <div key={i} className="flex items-center gap-1.5">
            <Select
              value={rule.rule}
              onValueChange={(v) => {
                const rules = [...(field.validation ?? [])]
                rules[i] = { ...rules[i], rule: v as ValidationRule["rule"] }
                update({ validation: rules })
              }}
            >
              <SelectTrigger className="w-28 h-7 text-xs"><SelectValue /></SelectTrigger>
              <SelectContent>
                {VALIDATION_RULE_TYPES.map((r) => (
                  <SelectItem key={r} value={r}>{r}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            {!["required", "email", "url"].includes(rule.rule) && (
              <Input
                className="w-16 h-7 text-xs"
                placeholder="value"
                value={rule.value !== undefined ? String(rule.value) : ""}
                onChange={(e) => {
                  const rules = [...(field.validation ?? [])]
                  rules[i] = { ...rules[i], value: e.target.value }
                  update({ validation: rules })
                }}
              />
            )}
            <Input
              className="flex-1 h-7 text-xs"
              placeholder="message"
              value={rule.message}
              onChange={(e) => {
                const rules = [...(field.validation ?? [])]
                rules[i] = { ...rules[i], message: e.target.value }
                update({ validation: rules })
              }}
            />
            <Button
              variant="ghost" size="icon" className="h-6 w-6 shrink-0 text-destructive"
              onClick={() => {
                const rules = (field.validation ?? []).filter((_, j) => j !== i)
                update({ validation: rules })
              }}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
        ))}
      </section>

      {/* Visibility Conditions */}
      <section className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium">{t("forms.visibilityConditions")}</Label>
          <Button
            variant="ghost" size="sm" className="h-6 px-2 text-xs"
            onClick={() => {
              const conds: VisibilityCondition[] = [
                ...(field.visibility?.conditions ?? []),
                { field: "", operator: "equals", value: "" },
              ]
              update({
                visibility: {
                  conditions: conds,
                  logic: field.visibility?.logic ?? "and",
                },
              })
            }}
          >
            <Plus className="h-3 w-3 mr-1" />{t("forms.addCondition")}
          </Button>
        </div>
        {(field.visibility?.conditions ?? []).length > 1 && (
          <div className="flex items-center gap-2">
            <Label className="text-xs">{t("forms.conditionLogic")}</Label>
            <Select
              value={field.visibility?.logic ?? "and"}
              onValueChange={(v) => update({
                visibility: {
                  ...field.visibility!,
                  logic: v as "and" | "or",
                },
              })}
            >
              <SelectTrigger className="w-20 h-7 text-xs"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="and">AND</SelectItem>
                <SelectItem value="or">OR</SelectItem>
              </SelectContent>
            </Select>
          </div>
        )}
        {(field.visibility?.conditions ?? []).map((cond, i) => (
          <div key={i} className="flex items-center gap-1.5">
            <Select
              value={cond.field}
              onValueChange={(v) => {
                const conds = [...(field.visibility?.conditions ?? [])]
                conds[i] = { ...conds[i], field: v }
                update({ visibility: { ...field.visibility!, conditions: conds } })
              }}
            >
              <SelectTrigger className="w-28 h-7 text-xs">
                <SelectValue placeholder={t("forms.conditionField")} />
              </SelectTrigger>
              <SelectContent>
                {allFields.filter((f) => f.key !== field.key).map((f) => (
                  <SelectItem key={f.key} value={f.key}>{f.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select
              value={cond.operator}
              onValueChange={(v) => {
                const conds = [...(field.visibility?.conditions ?? [])]
                conds[i] = { ...conds[i], operator: v as VisibilityCondition["operator"] }
                update({ visibility: { ...field.visibility!, conditions: conds } })
              }}
            >
              <SelectTrigger className="w-24 h-7 text-xs"><SelectValue /></SelectTrigger>
              <SelectContent>
                {OPERATORS.map((op) => (
                  <SelectItem key={op} value={op}>{op}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            {!["is_empty", "is_not_empty"].includes(cond.operator) && (
              <Input
                className="flex-1 h-7 text-xs"
                placeholder={t("forms.conditionValue")}
                value={cond.value !== undefined ? String(cond.value) : ""}
                onChange={(e) => {
                  const conds = [...(field.visibility?.conditions ?? [])]
                  conds[i] = { ...conds[i], value: e.target.value }
                  update({ visibility: { ...field.visibility!, conditions: conds } })
                }}
              />
            )}
            <Button
              variant="ghost" size="icon" className="h-6 w-6 shrink-0 text-destructive"
              onClick={() => {
                const conds = (field.visibility?.conditions ?? []).filter((_, j) => j !== i)
                update({
                  visibility: conds.length > 0
                    ? { ...field.visibility!, conditions: conds }
                    : undefined,
                })
              }}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
        ))}
      </section>
    </div>
  )
}
