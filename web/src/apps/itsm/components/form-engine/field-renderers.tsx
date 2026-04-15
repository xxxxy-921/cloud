import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Checkbox } from "@/components/ui/checkbox"
import { Switch } from "@/components/ui/switch"
import type { ControllerRenderProps } from "react-hook-form"
import type { FormField } from "./types"

type FieldProps = {
  field: FormField
  value: unknown
  onChange: ControllerRenderProps["onChange"]
  onBlur: ControllerRenderProps["onBlur"]
  disabled: boolean
  readOnly: boolean
}

function renderText({ field, value, onChange, onBlur, disabled, readOnly }: FieldProps) {
  return (
    <Input
      type={field.type === "email" ? "email" : field.type === "url" ? "url" : "text"}
      placeholder={field.placeholder}
      value={(value as string) ?? ""}
      onChange={(e) => onChange(e.target.value)}
      onBlur={onBlur}
      disabled={disabled}
      readOnly={readOnly}
      maxLength={field.props?.maxLength as number | undefined}
    />
  )
}

function renderTextarea({ field, value, onChange, onBlur, disabled, readOnly }: FieldProps) {
  return (
    <Textarea
      placeholder={field.placeholder}
      value={(value as string) ?? ""}
      onChange={(e) => onChange(e.target.value)}
      onBlur={onBlur}
      disabled={disabled}
      readOnly={readOnly}
      rows={(field.props?.rows as number) ?? 3}
    />
  )
}

function renderNumber({ field, value, onChange, onBlur, disabled, readOnly }: FieldProps) {
  return (
    <Input
      type="number"
      placeholder={field.placeholder}
      value={value === undefined || value === null ? "" : String(value)}
      onChange={(e) => {
        const v = e.target.value
        onChange(v === "" ? undefined : Number(v))
      }}
      onBlur={onBlur}
      disabled={disabled}
      readOnly={readOnly}
      min={field.props?.min as number | undefined}
      max={field.props?.max as number | undefined}
      step={field.props?.step as number | undefined}
    />
  )
}

function renderSelect({ field, value, onChange, disabled, readOnly }: FieldProps) {
  if (readOnly) {
    const label = field.options?.find((o) => String(o.value) === String(value))?.label
    return <Input value={label ?? String(value ?? "")} readOnly disabled />
  }
  return (
    <Select
      value={value === undefined || value === null ? "" : String(value)}
      onValueChange={onChange}
      disabled={disabled}
    >
      <SelectTrigger>
        <SelectValue placeholder={field.placeholder ?? "请选择"} />
      </SelectTrigger>
      <SelectContent>
        {field.options?.map((opt) => (
          <SelectItem key={String(opt.value)} value={String(opt.value)}>
            {opt.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

function renderMultiSelect({ field, value, onChange, disabled, readOnly }: FieldProps) {
  const selected = Array.isArray(value) ? (value as string[]) : []

  if (readOnly) {
    const labels = selected
      .map((v) => field.options?.find((o) => String(o.value) === v)?.label ?? v)
      .join(", ")
    return <Input value={labels} readOnly disabled />
  }

  return (
    <div className="flex flex-wrap gap-2">
      {field.options?.map((opt) => {
        const checked = selected.includes(String(opt.value))
        return (
          <label key={String(opt.value)} className="flex items-center gap-1.5 text-sm cursor-pointer">
            <Checkbox
              checked={checked}
              disabled={disabled}
              onCheckedChange={(c) => {
                const next = c
                  ? [...selected, String(opt.value)]
                  : selected.filter((v) => v !== String(opt.value))
                onChange(next)
              }}
            />
            {opt.label}
          </label>
        )
      })}
    </div>
  )
}

function renderRadio({ field, value, onChange, disabled, readOnly }: FieldProps) {
  if (readOnly) {
    const label = field.options?.find((o) => String(o.value) === String(value))?.label
    return <Input value={label ?? String(value ?? "")} readOnly disabled />
  }
  return (
    <div className="flex flex-wrap gap-3">
      {field.options?.map((opt) => (
        <label key={String(opt.value)} className="flex items-center gap-1.5 text-sm cursor-pointer">
          <input
            type="radio"
            name={field.key}
            value={String(opt.value)}
            checked={String(value) === String(opt.value)}
            onChange={() => onChange(String(opt.value))}
            disabled={disabled}
            className="accent-primary"
          />
          {opt.label}
        </label>
      ))}
    </div>
  )
}

function renderCheckbox({ field, value, onChange, disabled, readOnly }: FieldProps) {
  // Single checkbox (no options) — boolean toggle
  if (!field.options || field.options.length === 0) {
    return (
      <div className="flex items-center gap-2">
        <Checkbox
          checked={!!value}
          disabled={disabled || readOnly}
          onCheckedChange={onChange}
        />
        {field.description && <span className="text-sm text-muted-foreground">{field.description}</span>}
      </div>
    )
  }
  // Multiple options — checkbox group (delegate to multi_select renderer)
  return renderMultiSelect({ field, value, onChange, onBlur: () => {}, disabled, readOnly })
}

function renderSwitch({ field, value, onChange, disabled, readOnly }: FieldProps) {
  return (
    <div className="flex items-center gap-2">
      <Switch
        checked={!!value}
        disabled={disabled || readOnly}
        onCheckedChange={onChange}
      />
      {field.description && <span className="text-sm text-muted-foreground">{field.description}</span>}
    </div>
  )
}

function renderDate({ field, value, onChange, onBlur, disabled, readOnly }: FieldProps) {
  return (
    <Input
      type="date"
      placeholder={field.placeholder}
      value={(value as string) ?? ""}
      onChange={(e) => onChange(e.target.value)}
      onBlur={onBlur}
      disabled={disabled}
      readOnly={readOnly}
    />
  )
}

function renderDatetime({ field, value, onChange, onBlur, disabled, readOnly }: FieldProps) {
  return (
    <Input
      type="datetime-local"
      placeholder={field.placeholder}
      value={(value as string) ?? ""}
      onChange={(e) => onChange(e.target.value)}
      onBlur={onBlur}
      disabled={disabled}
      readOnly={readOnly}
    />
  )
}

function renderDateRange({ value, onChange, disabled, readOnly }: FieldProps) {
  const range = (value as { start?: string; end?: string }) ?? {}
  return (
    <div className="flex items-center gap-2">
      <Input
        type="date"
        value={range.start ?? ""}
        onChange={(e) => onChange({ ...range, start: e.target.value })}
        disabled={disabled}
        readOnly={readOnly}
        placeholder="开始日期"
      />
      <span className="text-muted-foreground">—</span>
      <Input
        type="date"
        value={range.end ?? ""}
        onChange={(e) => onChange({ ...range, end: e.target.value })}
        disabled={disabled}
        readOnly={readOnly}
        placeholder="结束日期"
      />
    </div>
  )
}

function renderUserPicker({ field, value, onChange, onBlur, disabled, readOnly }: FieldProps) {
  // Basic text input — full Combobox + User API integration deferred
  return (
    <Input
      placeholder={field.placeholder ?? "输入用户名搜索"}
      value={(value as string) ?? ""}
      onChange={(e) => onChange(e.target.value)}
      onBlur={onBlur}
      disabled={disabled}
      readOnly={readOnly}
    />
  )
}

function renderDeptPicker({ field, value, onChange, onBlur, disabled, readOnly }: FieldProps) {
  // Basic text input — full TreeSelect + Org API integration deferred
  return (
    <Input
      placeholder={field.placeholder ?? "输入部门名搜索"}
      value={(value as string) ?? ""}
      onChange={(e) => onChange(e.target.value)}
      onBlur={onBlur}
      disabled={disabled}
      readOnly={readOnly}
    />
  )
}

function renderRichText({ field, value, onChange, onBlur, disabled, readOnly }: FieldProps) {
  // Textarea + markdown preview deferred — use plain textarea for now
  return (
    <Textarea
      placeholder={field.placeholder ?? "支持 Markdown 格式"}
      value={(value as string) ?? ""}
      onChange={(e) => onChange(e.target.value)}
      onBlur={onBlur}
      disabled={disabled}
      readOnly={readOnly}
      rows={(field.props?.rows as number) ?? 6}
    />
  )
}

function renderFallback({ field, value, onChange, onBlur, disabled, readOnly }: FieldProps) {
  return (
    <div>
      <Input
        placeholder={field.placeholder}
        value={(value as string) ?? ""}
        onChange={(e) => onChange(e.target.value)}
        onBlur={onBlur}
        disabled={disabled}
        readOnly={readOnly}
      />
      <p className="text-xs text-amber-500 mt-1">未知字段类型: {field.type}</p>
    </div>
  )
}

// Renderer lookup
const renderers: Record<string, (props: FieldProps) => React.ReactNode> = {
  text: renderText,
  email: renderText,
  url: renderText,
  textarea: renderTextarea,
  number: renderNumber,
  select: renderSelect,
  multi_select: renderMultiSelect,
  radio: renderRadio,
  checkbox: renderCheckbox,
  switch: renderSwitch,
  date: renderDate,
  datetime: renderDatetime,
  date_range: renderDateRange,
  user_picker: renderUserPicker,
  dept_picker: renderDeptPicker,
  rich_text: renderRichText,
}

export function renderField(props: FieldProps): React.ReactNode {
  const renderer = renderers[props.field.type] ?? renderFallback
  return renderer(props)
}
