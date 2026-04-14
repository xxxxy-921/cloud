import { useMemo, useCallback } from "react"
import {
  ReactFlow,
  Controls,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
} from "@xyflow/react"
import dagre from "@dagrejs/dagre"
import "@xyflow/react/dist/style.css"

import type { TopologyGraph } from "../api"
import { ServiceNode, type ServiceNodeData } from "./topology/service-node"
import { ServiceEdge, type ServiceEdgeData } from "./topology/service-edge"
import type { ColorMode } from "./topology/color-mode-select"

const nodeTypes = { service: ServiceNode }
const edgeTypes = { service: ServiceEdge }

const NODE_WIDTH = 90
const NODE_HEIGHT = 90

interface ServiceMapProps {
  graph: TopologyGraph
  timeStart?: string
  timeEnd?: string
  colorMode?: ColorMode
  p95Map?: Record<string, number>
  filteredNodes?: Set<string>
  selectedNode?: string | null
  onSelectNode?: (name: string | null) => void
}

function layoutGraph(
  graph: TopologyGraph,
  colorMode: ColorMode,
  p95Map: Record<string, number>,
  filteredNodes: Set<string>,
) {
  const g = new dagre.graphlib.Graph()
  g.setGraph({ rankdir: "LR", nodesep: 60, ranksep: 180, marginx: 40, marginy: 40 })
  g.setDefaultEdgeLabel(() => ({}))

  for (const node of graph.nodes) {
    g.setNode(node.serviceName, { width: NODE_WIDTH, height: NODE_HEIGHT })
  }
  for (const edge of graph.edges) {
    g.setEdge(edge.caller, edge.callee)
  }

  dagre.layout(g)

  const nodes: Node[] = graph.nodes.map((n) => {
    const pos = g.node(n.serviceName)
    return {
      id: n.serviceName,
      type: "service",
      position: { x: pos.x - NODE_WIDTH / 2, y: pos.y - NODE_HEIGHT / 2 },
      data: {
        ...n,
        colorMode,
        p95Ms: p95Map[n.serviceName],
        filtered: filteredNodes.has(n.serviceName),
      } satisfies ServiceNodeData,
    }
  })

  const edges: Edge[] = graph.edges.map((e) => {
    const bothFiltered = filteredNodes.has(e.caller) || filteredNodes.has(e.callee)
    return {
      id: `${e.caller}-${e.callee}`,
      source: e.caller,
      target: e.callee,
      type: "service",
      data: {
        ...e,
        colorMode,
        filtered: bothFiltered,
      } satisfies ServiceEdgeData,
    }
  })

  return { nodes, edges }
}

const LEGEND_ERROR_RATE = [
  { color: "bg-emerald-500", label: "Healthy" },
  { color: "bg-amber-500", label: "> 1% err" },
  { color: "bg-red-500", label: "> 5% err" },
]
const LEGEND_LATENCY = [
  { color: "bg-emerald-500", label: "Low (<100ms)" },
  { color: "bg-amber-500", label: "Med (100-500ms)" },
  { color: "bg-red-500", label: "High (>500ms)" },
]
const LEGEND_THROUGHPUT = [
  { color: "bg-sky-400", label: "Low" },
  { color: "bg-blue-500", label: "Medium" },
  { color: "bg-violet-500", label: "High" },
]

export function ServiceMap({
  graph,
  colorMode = "errorRate",
  p95Map = {},
  filteredNodes = new Set(),
  selectedNode,
  onSelectNode,
}: ServiceMapProps) {
  const { initialNodes, initialEdges } = useMemo(() => {
    const { nodes, edges } = layoutGraph(graph, colorMode, p95Map, filteredNodes)
    return { initialNodes: nodes, initialEdges: edges }
  }, [graph, colorMode, p95Map, filteredNodes])

  const [nodes, , onNodesChange] = useNodesState(initialNodes)
  const [edges, , onEdgesChange] = useEdgesState(initialEdges)

  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      onSelectNode?.(node.id)
    },
    [onSelectNode],
  )

  const onPaneClick = useCallback(() => {
    // Don't deselect when clicking pane — user might be panning
  }, [])

  const legendItems = colorMode === "latency" ? LEGEND_LATENCY : colorMode === "throughput" ? LEGEND_THROUGHPUT : LEGEND_ERROR_RATE

  return (
    <div className="relative h-[calc(100vh-10rem)] min-h-[500px] w-full">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        fitView
        fitViewOptions={{ padding: 0.35, maxZoom: 1.1 }}
        proOptions={{ hideAttribution: true }}
        minZoom={0.2}
        maxZoom={2}
        nodesDraggable
      >
        <defs>
          <marker id="arrow" viewBox="0 0 12 8" refX="11" refY="4" markerWidth="9" markerHeight="7" orient="auto">
            <path d="M 0 0.5 L 10 4 L 0 7.5 L 2.5 4 Z" fill="hsl(var(--muted-foreground) / 0.3)" />
          </marker>
          <marker id="arrow-warn" viewBox="0 0 12 8" refX="11" refY="4" markerWidth="9" markerHeight="7" orient="auto">
            <path d="M 0 0.5 L 10 4 L 0 7.5 L 2.5 4 Z" fill="hsl(38 92% 50% / 0.6)" />
          </marker>
          <marker id="arrow-error" viewBox="0 0 12 8" refX="11" refY="4" markerWidth="9" markerHeight="7" orient="auto">
            <path d="M 0 0.5 L 10 4 L 0 7.5 L 2.5 4 Z" fill="hsl(0 72% 51% / 0.6)" />
          </marker>
        </defs>

        <Controls
          showInteractive={false}
          className="!rounded-xl !border-border/60 !bg-card/90 !backdrop-blur-sm !shadow-md [&_button]:!border-border/40 [&_button]:!bg-transparent [&_button]:!text-muted-foreground [&_button:hover]:!bg-muted/80"
        />

        <Background
          variant={BackgroundVariant.Dots}
          gap={24}
          size={1}
          className="!bg-background"
          color="hsl(var(--muted-foreground) / 0.07)"
        />
      </ReactFlow>

      {/* Legend */}
      <div className="absolute bottom-4 right-4 flex items-center gap-4 rounded-lg border bg-card/90 backdrop-blur-sm px-3 py-2 text-[10px] text-muted-foreground shadow-sm">
        {legendItems.map((item) => (
          <div key={item.label} className="flex items-center gap-1.5">
            <span className={`h-2 w-2 rounded-full ${item.color}`} />
            <span>{item.label}</span>
          </div>
        ))}
        <div className="flex items-center gap-1.5 border-l pl-4 border-border/40">
          <span className="flex items-center gap-0.5">
            <span className="h-1.5 w-1.5 rounded-full bg-primary/50" />
            <span>traffic flow</span>
          </span>
        </div>
      </div>
    </div>
  )
}
