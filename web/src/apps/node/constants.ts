export const NODE_STATUS_VARIANTS: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  pending: "secondary",
  online: "default",
  offline: "destructive",
}

export const PROCESS_STATUS_VARIANTS: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  running: "default",
  stopped: "secondary",
  error: "destructive",
  pending_config: "outline",
}
