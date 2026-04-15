"use client"

import { useTranslation } from "react-i18next"

interface FormDataDisplayProps {
  data: unknown
}

export function FormDataDisplay({ data }: FormDataDisplayProps) {
  const { t } = useTranslation("itsm")

  if (!data || typeof data !== "object") return null

  const entries = Object.entries(data as Record<string, unknown>)
  if (entries.length === 0) return null

  return (
    <div className="space-y-1.5">
      <p className="text-sm font-medium text-muted-foreground">{t("tickets.formData")}</p>
      <div className="rounded-md border">
        {entries.map(([key, value]) => (
          <div key={key} className="flex justify-between border-b px-3 py-2 text-sm last:border-b-0">
            <span className="text-muted-foreground">{key}</span>
            <span className="text-right max-w-[60%] break-words">{String(value ?? "")}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
