import { codingAgentApi } from "@/lib/api"
import { AgentListPage, type AgentListPageConfig } from "../_shared/agent-list-page"

const config: AgentListPageConfig = {
  agentType: "coding",
  i18nKey: "codingAgents",
  basePath: "/ai/coding-agents",
  endpoint: "/api/v1/ai/coding-agents",
  queryKey: "ai-coding-agents",
  permissions: {
    create: "ai:coding-agent:create",
    update: "ai:coding-agent:update",
    delete: "ai:coding-agent:delete",
  },
  deleteApiFn: codingAgentApi.delete,
}

export function Component() {
  return <AgentListPage config={config} />
}
