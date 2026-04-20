import * as React from "react"
import { Switch as SwitchPrimitive } from "radix-ui"

import { cn } from "@/lib/utils"

function Switch({
  className,
  size = "default",
  ...props
}: React.ComponentProps<typeof SwitchPrimitive.Root> & {
  size?: "sm" | "default"
}) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      data-size={size}
      className={cn(
        "peer group/switch inline-flex shrink-0 items-center rounded-full border shadow-xs transition-all outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 data-[size=default]:h-[1.15rem] data-[size=default]:w-8 data-[size=sm]:h-3.5 data-[size=sm]:w-6 data-[state=checked]:border-primary/25 data-[state=checked]:bg-primary/92 data-[state=checked]:shadow-[0_10px_24px_-14px_hsl(var(--primary)/0.85)] data-[state=unchecked]:border-border/70 data-[state=unchecked]:bg-surface-soft/90 data-[state=unchecked]:shadow-[inset_0_1px_0_rgba(255,255,255,0.82)] dark:data-[state=unchecked]:border-input dark:data-[state=unchecked]:bg-input/80",
        className
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className={cn(
          "pointer-events-none block rounded-full bg-white ring-0 transition-transform group-data-[size=default]/switch:size-4 group-data-[size=sm]/switch:size-3 group-data-[state=checked]/switch:bg-primary-foreground group-data-[state=checked]/switch:shadow-[0_2px_8px_rgba(15,23,42,0.16)] group-data-[state=unchecked]/switch:shadow-[0_1px_3px_rgba(15,23,42,0.18)] data-[state=checked]:translate-x-[calc(100%-2px)] data-[state=unchecked]:translate-x-0 dark:bg-foreground dark:group-data-[state=checked]/switch:bg-primary-foreground"
        )}
      />
    </SwitchPrimitive.Root>
  )
}

export { Switch }
