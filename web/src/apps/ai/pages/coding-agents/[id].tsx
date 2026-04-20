import { codingAgentApi } from "@/lib/api"
import { AgentDetailPage, type AgentDetailPageConfig } from "../_shared/agent-detail-page"

const config: AgentDetailPageConfig = {
  agentType: "coding",
  i18nKey: "codingAgents",
  basePath: "/ai/coding-agents",
  queryKey: "ai-coding-agent",
  getApiFn: codingAgentApi.get,
  deleteApiFn: codingAgentApi.delete,
  listQueryKey: "ai-coding-agents",
}

export function Component() {
  return <AgentDetailPage config={config} />
}
