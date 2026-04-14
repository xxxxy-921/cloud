import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useQuery } from "@tanstack/react-query"
import { Network } from "lucide-react"

import { fetchTopology, fetchServices } from "../../api"
import { useTimeRange } from "../../hooks/use-time-range"
import { TimeRangePicker } from "../../components/time-range-picker"
import { ServiceMap } from "../../components/service-map"
import { DetailPanel } from "../../components/topology/detail-panel"
import { TopologyToolbar } from "../../components/topology/topology-toolbar"
import { ColorModeSelect, type ColorMode } from "../../components/topology/color-mode-select"

function TopologyPage() {
  const { t } = useTranslation("apm")
  const { range, selectPreset, setCustomRange, refresh, presets, refreshInterval, setRefreshInterval } = useTimeRange("last1h")

  const [selectedNode, setSelectedNode] = useState<string | null>(null)
  const [colorMode, setColorMode] = useState<ColorMode>("errorRate")
  const [searchQuery, setSearchQuery] = useState("")
  const [errorOnly, setErrorOnly] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ["apm-topology", range.start, range.end],
    queryFn: () => fetchTopology(range.start, range.end),
  })

  // Fetch services list for latency data (task 4.5)
  const { data: servicesData } = useQuery({
    queryKey: ["apm-services-for-topology", range.start, range.end],
    queryFn: () => fetchServices(range.start, range.end),
    enabled: colorMode === "latency",
  })

  // Build p95 lookup map from services data
  const p95Map: Record<string, number> = {}
  if (servicesData?.services) {
    for (const svc of servicesData.services) {
      p95Map[svc.serviceName] = svc.p95Ms
    }
  }

  // Compute filtered state for nodes
  const filteredNodes = new Set<string>()
  if (data?.nodes) {
    const query = searchQuery.toLowerCase().trim()
    for (const node of data.nodes) {
      const matchSearch = !query || node.serviceName.toLowerCase().includes(query)
      const matchError = !errorOnly || node.errorRate > 0
      if (!(matchSearch && matchError)) {
        filteredNodes.add(node.serviceName)
      }
    }
  }

  const hasData = data && data.nodes && data.nodes.length > 0

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Network className="h-5 w-5 text-muted-foreground" />
          <h1 className="text-lg font-semibold">{t("topology.title")}</h1>
          {hasData && (
            <span className="ml-1 text-xs text-muted-foreground font-mono">
              {data.nodes.length} services · {data.edges.length} edges
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <TopologyToolbar
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            errorOnly={errorOnly}
            onErrorOnlyChange={setErrorOnly}
          />
          <ColorModeSelect value={colorMode} onChange={setColorMode} />
          <TimeRangePicker value={range.label} presets={presets} onSelect={selectPreset} onRefresh={refresh} onCustomRange={setCustomRange} refreshInterval={refreshInterval} onRefreshIntervalChange={setRefreshInterval} />
        </div>
      </div>

      {isLoading ? (
        <div className="flex-1 flex items-center justify-center text-muted-foreground">{t("loading")}</div>
      ) : !hasData ? (
        <div className="flex-1 flex flex-col items-center justify-center">
          <Network className="h-12 w-12 text-muted-foreground/20 mb-3" />
          <p className="text-muted-foreground">{t("topology.noData")}</p>
          <p className="mt-1 text-sm text-muted-foreground/60">{t("topology.noDataHint")}</p>
        </div>
      ) : (
        <div className="flex-1 flex rounded-xl border bg-card overflow-hidden">
          <div className="flex-1 min-w-0">
            <ServiceMap
              graph={data}
              timeStart={range.start}
              timeEnd={range.end}
              colorMode={colorMode}
              p95Map={p95Map}
              filteredNodes={filteredNodes}
              selectedNode={selectedNode}
              onSelectNode={setSelectedNode}
            />
          </div>
          {selectedNode && (
            <DetailPanel
              serviceName={selectedNode}
              timeStart={range.start}
              timeEnd={range.end}
              onClose={() => setSelectedNode(null)}
            />
          )}
        </div>
      )}
    </div>
  )
}

export function Component() {
  return <TopologyPage />
}
