import { assistantAgentApi } from "@/lib/api"
import { AgentListPage, type AgentListPageConfig } from "../_shared/agent-list-page"

const config: AgentListPageConfig = {
  agentType: "assistant",
  i18nKey: "assistantAgents",
  basePath: "/ai/assistant-agents",
  endpoint: "/api/v1/ai/assistant-agents",
  queryKey: "ai-assistant-agents",
  permissions: {
    create: "ai:assistant-agent:create",
    update: "ai:assistant-agent:update",
    delete: "ai:assistant-agent:delete",
  },
  deleteApiFn: assistantAgentApi.delete,
}

export function Component() {
  return <AgentListPage config={config} />
}
