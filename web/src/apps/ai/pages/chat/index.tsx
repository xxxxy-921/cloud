import { useNavigate } from "react-router"
import { useTranslation } from "react-i18next"
import { useQuery, useMutation } from "@tanstack/react-query"
import { Bot, BrainCircuit, Code2 } from "lucide-react"
import { agentApi, sessionApi, type AgentInfo } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"

const TYPE_CONFIG: Record<string, { icon: typeof Bot; label: string; gradient: string }> = {
  assistant: {
    icon: BrainCircuit,
    label: "AI 助手",
    gradient: "from-violet-500/10 to-indigo-500/10",
  },
  coding: {
    icon: Code2,
    label: "编程助手",
    gradient: "from-emerald-500/10 to-teal-500/10",
  },
}

function AgentCard({ agent, onChat }: { agent: AgentInfo; onChat: () => void }) {
  const { t } = useTranslation(["ai"])
  const config = TYPE_CONFIG[agent.type] ?? { icon: Bot, label: agent.type, gradient: "from-gray-500/10 to-gray-400/10" }
  const Icon = config.icon

  return (
    <div
      className="group relative flex flex-col gap-3 rounded-lg border bg-card p-4 transition-all duration-200 hover:shadow-sm hover:border-primary/30 cursor-pointer"
      onClick={onChat}
    >
      {/* 图标 + 名称 - 单行 */}
      <div className="flex items-center gap-3">
        <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-gradient-to-br ${config.gradient}`}>
          <Icon className="h-4 w-4 text-foreground/70" />
        </div>
        <h3 className="font-medium text-sm truncate">{agent.name}</h3>
      </div>

      {/* 描述 - 单行截断 */}
      <p className="text-xs text-muted-foreground truncate leading-relaxed">
        {agent.description || config.label}
      </p>

      {/* 底部按钮 */}
      <div className="mt-auto pt-2">
        <Button
          variant="secondary"
          size="sm"
          className="w-full h-8 text-xs"
        >
          {t("ai:chat.startChat")}
        </Button>
      </div>
    </div>
  )
}

export function Component() {
  const { t } = useTranslation(["ai"])
  const navigate = useNavigate()

  const { data, isLoading } = useQuery({
    queryKey: ["ai-agents-for-chat"],
    queryFn: () => agentApi.list({ pageSize: 50 }),
  })
  const agents = (data?.items ?? []).filter((a: AgentInfo) => a.isActive)

  const createSessionMutation = useMutation({
    mutationFn: (agentId: number) => sessionApi.create(agentId),
    onSuccess: (session) => {
      navigate(`/ai/chat/${session.id}`)
    },
    onError: (err) => toast.error(err.message),
  })

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">{t("ai:chat.selectAgent")}</h2>
      <p className="text-sm text-muted-foreground">{t("ai:chat.selectAgentHint")}</p>

      {isLoading ? (
        <div className="py-12 text-center text-muted-foreground">Loading...</div>
      ) : agents.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-12">
          <Bot className="h-10 w-10 text-muted-foreground/50" />
          <p className="text-sm text-muted-foreground">{t("ai:agents.empty")}</p>
        </div>
      ) : (
        <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
          {agents.map((agent: AgentInfo) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              onChat={() => createSessionMutation.mutate(agent.id)}
            />
          ))}
        </div>
      )}
    </div>
  )
}
