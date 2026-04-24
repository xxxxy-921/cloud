import { describe, expect, test } from "bun:test"

import {
  shouldProcessServiceDeskHistorySnapshot,
  shouldSyncServiceDeskHistory,
} from "./service-desk-chat-sync"

describe("shouldSyncServiceDeskHistory", () => {
  test("does not apply server history while a live run is submitted or streaming", () => {
    for (const status of ["submitted", "streaming"] as const) {
      expect(
        shouldSyncServiceDeskHistory({
          status,
          hasServerSnapshot: true,
          serverSignature: "server-new",
          localSignature: "local-live",
        }),
      ).toBe(false)
    }
  })

  test("applies server history only after the chat is idle and the snapshot changed", () => {
    for (const status of ["ready", "error"] as const) {
      expect(
        shouldSyncServiceDeskHistory({
          status,
          hasServerSnapshot: true,
          serverSignature: "server-new",
          localSignature: "local-old",
        }),
      ).toBe(true)
    }
  })

  test("skips sync without a server snapshot or when signatures already match", () => {
    expect(
      shouldSyncServiceDeskHistory({
        status: "ready",
        hasServerSnapshot: false,
        serverSignature: "server-new",
        localSignature: "local-old",
      }),
    ).toBe(false)
    expect(
      shouldSyncServiceDeskHistory({
        status: "ready",
        hasServerSnapshot: true,
        serverSignature: "same",
        localSignature: "same",
      }),
    ).toBe(false)
  })
})

describe("shouldProcessServiceDeskHistorySnapshot", () => {
  test("does not re-process a stale server snapshot after a live run finishes", () => {
    expect(
      shouldProcessServiceDeskHistorySnapshot({
        status: "ready",
        hasServerSnapshot: true,
        serverMessageCount: 0,
        localMessageCount: 0,
        serverSnapshotKey: "101:empty-history",
        syncedServerSnapshotKey: "101:empty-history",
      }),
    ).toBe(false)
  })

  test("processes a newly fetched server snapshot only when the chat is idle", () => {
    expect(
      shouldProcessServiceDeskHistorySnapshot({
        status: "ready",
        hasServerSnapshot: true,
        serverMessageCount: 2,
        localMessageCount: 2,
        serverSnapshotKey: "101:persisted-history",
        syncedServerSnapshotKey: "101:empty-history",
      }),
    ).toBe(true)
    expect(
      shouldProcessServiceDeskHistorySnapshot({
        status: "streaming",
        hasServerSnapshot: true,
        serverMessageCount: 2,
        localMessageCount: 2,
        serverSnapshotKey: "101:persisted-history",
        syncedServerSnapshotKey: "101:empty-history",
      }),
    ).toBe(false)
  })

  test("skips server snapshots that have not caught up with the live messages", () => {
    expect(
      shouldProcessServiceDeskHistorySnapshot({
        status: "ready",
        hasServerSnapshot: true,
        serverMessageCount: 1,
        localMessageCount: 2,
        serverSnapshotKey: "101:user-only",
        syncedServerSnapshotKey: "101:empty-history",
      }),
    ).toBe(false)
  })
})
