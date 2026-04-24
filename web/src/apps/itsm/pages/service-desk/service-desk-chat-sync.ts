import type { ChatStatus } from "ai"

export function shouldSyncServiceDeskHistory({
  status,
  hasServerSnapshot,
  serverSignature,
  localSignature,
}: {
  status: ChatStatus
  hasServerSnapshot: boolean
  serverSignature: string
  localSignature: string
}) {
  if (!hasServerSnapshot) return false
  if (status === "submitted" || status === "streaming") return false
  return serverSignature !== localSignature
}

export function shouldProcessServiceDeskHistorySnapshot({
  status,
  hasServerSnapshot,
  serverMessageCount,
  localMessageCount,
  serverSnapshotKey,
  syncedServerSnapshotKey,
}: {
  status: ChatStatus
  hasServerSnapshot: boolean
  serverMessageCount: number
  localMessageCount: number
  serverSnapshotKey: string
  syncedServerSnapshotKey: string
}) {
  if (!hasServerSnapshot) return false
  if (status === "submitted" || status === "streaming") return false
  if (serverMessageCount < localMessageCount) return false
  return serverSnapshotKey !== syncedServerSnapshotKey
}
