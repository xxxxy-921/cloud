import { useNavigate } from "react-router"
import { useTranslation } from "react-i18next"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { codingAgentApi } from "@/lib/api"
import { toast } from "sonner"
import { ArrowLeft } from "lucide-react"
import { Button } from "@/components/ui/button"
import { AgentForm, type AgentFormValues } from "../_shared/agent-form-common"

export function Component() {
  const { t } = useTranslation(["ai", "common"])
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: (values: AgentFormValues) => codingAgentApi.create(values),
    onSuccess: (agent) => {
      queryClient.invalidateQueries({ queryKey: ["ai-coding-agents"] })
      toast.success(t("ai:codingAgents.createSuccess"))
      navigate(`/ai/coding-agents/${agent.id}`)
    },
    onError: (err) => toast.error(err.message),
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => navigate("/ai/coding-agents")}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h2 className="text-lg font-semibold">{t("ai:codingAgents.create")}</h2>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => navigate("/ai/coding-agents")}>
            {t("common:cancel")}
          </Button>
          <Button type="submit" form="agent-form" disabled={mutation.isPending}>
            {mutation.isPending ? t("common:saving") : t("common:create")}
          </Button>
        </div>
      </div>

      <AgentForm agentType="coding" onSubmit={(v) => mutation.mutate(v)} />
    </div>
  )
}
