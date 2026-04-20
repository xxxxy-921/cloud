import { assistantAgentApi } from "@/lib/api"
import { AgentDetailPage, type AgentDetailPageConfig } from "../_shared/agent-detail-page"

const config: AgentDetailPageConfig = {
  agentType: "assistant",
  i18nKey: "assistantAgents",
  basePath: "/ai/assistant-agents",
  queryKey: "ai-assistant-agent",
  getApiFn: assistantAgentApi.get,
  deleteApiFn: assistantAgentApi.delete,
  listQueryKey: "ai-assistant-agents",
}

export function Component() {
  return <AgentDetailPage config={config} />
}
