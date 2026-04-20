// Shared types for the new Knowledge Base (KB) and Knowledge Graph (KG) pages

export type AssetStatus = "idle" | "building" | "ready" | "error" | "stale"

export interface ProgressCounter {
  total: number
  done: number
}

export interface BuildProgress {
  stage: string
  sources: ProgressCounter
  items: ProgressCounter
  embeddings: ProgressCounter
  currentItem: string
  startedAt: number
}

export interface KnowledgeAsset {
  id: number
  name: string
  description: string
  category: string
  type: string
  status: AssetStatus
  config: Record<string, unknown>
  compileModelId: number
  embeddingProviderId: number | null
  embeddingModelId: string
  autoBuild: boolean
  sourceCount: number
  builtAt: string
  buildProgress: BuildProgress | null
  nodeCount: number
  edgeCount: number
  chunkCount: number
  createdAt: string
  updatedAt: string
}

export interface KnowledgeType {
  category: string
  type: string
  displayName: string
  description: string
  icon: string
  defaultConfig: Record<string, unknown>
}

export interface AssetSourceItem {
  id: number
  title: string
  format: string
  extractStatus: string
  byteSize: number
  sourceType: string
  sourceUrl?: string
  createdAt: string
}

export interface ChunkItem {
  id: number
  content: string
  sourceId: number
  chunkIndex: number
}

export interface NodeItem {
  id: string
  title: string
  summary: string
  nodeType: string
  keywords: string[]
  edgeCount: number
  content?: string
}

export interface EdgeItem {
  fromNodeId: string
  toNodeId: string
  relation: string
  description?: string
}

export interface GraphResponse {
  nodes: NodeItem[]
  edges: EdgeItem[]
}

export interface SearchResult {
  id: string
  title: string
  content: string
  score: number
  metadata?: Record<string, unknown>
}

export interface LogItem {
  id: number
  action: string
  message: string
  createdAt: string
}

export interface SourcePoolItem {
  id: number
  title: string
  format: string
  byteSize: number
  createdAt: string
}
