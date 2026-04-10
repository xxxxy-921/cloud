import { Sparkles } from "lucide-react"

import { cn } from "@/lib/utils"

interface AuthBrandLockupProps {
  appName: string
  hasLogo?: boolean
  compact?: boolean
}

export function AuthBrandLockup({ appName, hasLogo = false, compact = false }: AuthBrandLockupProps) {
  return (
    <div className={cn("flex items-center", compact ? "gap-3" : "gap-3.5")}>
      <div
        className={cn(
          "flex shrink-0 items-center justify-center rounded-xl border border-slate-200/80 bg-white text-slate-900 shadow-[0_8px_24px_rgba(15,23,42,0.06)]",
          compact ? "h-10 w-10" : "h-12 w-12"
        )}
      >
        {hasLogo ? (
          <img
            src="/api/v1/site-info/logo"
            alt={appName}
            className={cn("object-contain", compact ? "h-5 w-5" : "h-7 w-7")}
          />
        ) : (
          <Sparkles className={cn("text-[oklch(0.42_0.18_264)]", compact ? "h-4.5 w-4.5" : "h-5 w-5")} />
        )}
      </div>

      <div>
        <div className={cn("font-semibold tracking-tight text-slate-950", compact ? "text-[1.125rem]" : "text-[1.625rem]")}>
          {appName}
        </div>
      </div>
    </div>
  )
}
