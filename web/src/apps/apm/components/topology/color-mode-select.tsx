import { useTranslation } from "react-i18next"
import { Palette } from "lucide-react"

export type ColorMode = "errorRate" | "latency" | "throughput"

interface ColorModeSelectProps {
  value: ColorMode
  onChange: (mode: ColorMode) => void
}

const OPTIONS: { value: ColorMode; labelKey: string }[] = [
  { value: "errorRate", labelKey: "topology.colorMode.errorRate" },
  { value: "latency", labelKey: "topology.colorMode.latency" },
  { value: "throughput", labelKey: "topology.colorMode.throughput" },
]

export function ColorModeSelect({ value, onChange }: ColorModeSelectProps) {
  const { t } = useTranslation("apm")

  return (
    <div className="flex items-center gap-1.5 rounded-lg border bg-muted/30 px-2 py-1">
      <Palette className="w-3.5 h-3.5 text-muted-foreground" />
      <select
        value={value}
        onChange={(e) => onChange(e.target.value as ColorMode)}
        className="bg-transparent text-xs font-medium text-foreground outline-none cursor-pointer pr-1"
      >
        {OPTIONS.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {t(opt.labelKey)}
          </option>
        ))}
      </select>
    </div>
  )
}
