import { useEffect, useMemo } from "react"
import { useForm, useWatch } from "react-hook-form"
import { useTranslation } from "react-i18next"
import { zodResolver } from "@/lib/form"
import { z } from "zod"
import { useQuery } from "@tanstack/react-query"
import { api, type AgentWithBindings, type PaginatedResponse } from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { BindingSelectorSection, type BindingGroup, type BindingItem } from "./binding-checkbox-list"

interface ProviderItem {
  id: number
  name: string
}

interface ModelItem {
  id: number
  displayName: string
  modelId: string
  providerId: number
}

interface ToolkitGroup {
  toolkit: string
  tools: BindingItem[]
}


const agentSchema = z.object({
  name: z.string().min(1).max(128),
  description: z.string().optional(),
  visibility: z.enum(["private", "team", "public"]),
  strategy: z.string().optional(),
  providerId: z.string().optional(),
  modelId: z.coerce.number().optional(),
  systemPrompt: z.string().optional(),
  temperature: z.coerce.number().min(0).max(2).optional(),
  maxTokens: z.coerce.number().min(1).optional(),
  maxTurns: z.coerce.number().min(1).max(100).optional(),
  runtime: z.string().optional(),
  execMode: z.string().optional(),
  workspace: z.string().optional(),
  nodeId: z.coerce.number().optional(),
  instructions: z.string().optional(),
  toolIds: z.array(z.number()),
  skillIds: z.array(z.number()),
  mcpServerIds: z.array(z.number()),
  knowledgeBaseIds: z.array(z.number()),
  knowledgeGraphIds: z.array(z.number()),
})

export type AgentFormValues = z.infer<typeof agentSchema>

interface AgentFormProps {
  agentType: "assistant" | "coding"
  agent?: AgentWithBindings
  onSubmit: (values: AgentFormValues) => void
}

const defaultValues: AgentFormValues = {
  name: "",
  description: "",
  visibility: "team",
  strategy: "react",
  providerId: "",
  modelId: undefined,
  systemPrompt: "",
  temperature: 0.7,
  maxTokens: 4096,
  maxTurns: 10,
  runtime: "claude-code",
  execMode: "local",
  workspace: "",
  nodeId: undefined,
  instructions: "",
  toolIds: [],
  skillIds: [],
  mcpServerIds: [],
  knowledgeBaseIds: [],
  knowledgeGraphIds: [],
}

export function AgentForm({ agentType, agent, onSubmit }: AgentFormProps) {
  const { t } = useTranslation(["ai", "common"])

  // For edit mode: resolve providerId from the agent's modelId
  const agentModelId = agent?.type === "assistant" ? agent.modelId : undefined
  const { data: editModelDetail } = useQuery({
    queryKey: ["ai-model-detail", agentModelId],
    queryFn: () => api.get<ModelItem>(`/api/v1/ai/models/${agentModelId}`),
    enabled: !!agentModelId,
  })
  const editProviderId = editModelDetail?.providerId ? String(editModelDetail.providerId) : ""

  const resetValues = useMemo<AgentFormValues>(() => {
    if (!agent) return defaultValues
    return {
      name: agent.name,
      description: agent.description || "",
      visibility: agent.visibility,
      strategy: agent.type === "assistant" ? (agent.strategy || "react") : undefined,
      providerId: "",
      modelId: agent.type === "assistant" ? (agent.modelId ?? undefined) : undefined,
      systemPrompt: agent.type === "assistant" ? (agent.systemPrompt || "") : "",
      temperature: agent.temperature,
      maxTokens: agent.maxTokens,
      maxTurns: agent.maxTurns,
      runtime: agent.type === "coding" ? (agent.runtime || "claude-code") : undefined,
      execMode: agent.type === "coding" ? (agent.execMode || "local") : undefined,
      workspace: agent.type === "coding" ? (agent.workspace || "") : "",
      nodeId: agent.type === "coding" ? (agent.nodeId ?? undefined) : undefined,
      instructions: agent.instructions || "",
      toolIds: agent.toolIds ?? [],
      skillIds: agent.skillIds ?? [],
      mcpServerIds: agent.mcpServerIds ?? [],
      knowledgeBaseIds: agent.knowledgeBaseIds ?? [],
      knowledgeGraphIds: agent.knowledgeGraphIds ?? [],
    }
  }, [agent])

  const form = useForm<AgentFormValues>({
    resolver: zodResolver(agentSchema),
    defaultValues: resetValues,
  })

  useEffect(() => {
    form.reset(resetValues)
  }, [resetValues, form])

  useEffect(() => {
    if (!agentModelId || !editProviderId) return
    if (form.getValues("providerId") !== "") return
    form.setValue("providerId", editProviderId, { shouldDirty: false, shouldTouch: false })
  }, [agentModelId, editProviderId, form])

  const watchExecMode = useWatch({ control: form.control, name: "execMode" })
  const selectedProviderId = useWatch({ control: form.control, name: "providerId" }) ?? ""

  // Fetch providers
  const { data: providers = [] } = useQuery({
    queryKey: ["ai-providers"],
    queryFn: () =>
      api.get<PaginatedResponse<ProviderItem>>("/api/v1/ai/providers?pageSize=100").then((r) => r?.items ?? []),
  })

  // Fetch LLM models filtered by selected provider
  const { data: models = [] } = useQuery({
    queryKey: ["ai-models", selectedProviderId],
    queryFn: () =>
      api.get<PaginatedResponse<ModelItem>>(`/api/v1/ai/models?type=llm&providerId=${selectedProviderId}&pageSize=100`).then((r) => r?.items ?? []),
    enabled: selectedProviderId !== "",
  })

  // Fetch binding lists
  const { data: toolGroups = [], isLoading: toolsLoading } = useQuery({
    queryKey: ["ai-agent-binding-tools"],
    queryFn: () =>
      api.get<{ items: ToolkitGroup[] }>("/api/v1/ai/tools").then((r) => r?.items ?? []),
  })

  const { data: mcpItems = [], isLoading: mcpLoading } = useQuery({
    queryKey: ["ai-binding-mcp-servers"],
    queryFn: () =>
      api.get<PaginatedResponse<BindingItem>>("/api/v1/ai/mcp-servers?pageSize=100").then((r) => r?.items ?? []),
  })

  const { data: skillItems = [], isLoading: skillsLoading } = useQuery({
    queryKey: ["ai-binding-skills"],
    queryFn: () =>
      api.get<PaginatedResponse<BindingItem>>("/api/v1/ai/skills?pageSize=100").then((r) => r?.items ?? []),
  })

  const { data: kbItems = [], isLoading: kbLoading } = useQuery({
    queryKey: ["ai-binding-knowledge-bases"],
    queryFn: () =>
      api.get<PaginatedResponse<BindingItem>>("/api/v1/ai/knowledge-bases?pageSize=100").then((r) => r?.items ?? []),
  })

  const { data: kgItems = [], isLoading: kgLoading } = useQuery({
    queryKey: ["ai-binding-knowledge-graphs"],
    queryFn: () =>
      api.get<PaginatedResponse<BindingItem>>("/api/v1/ai/knowledge/graphs?pageSize=100").then((r) => r?.items ?? []),
  })

  function handleProviderChange(value: string) {
    form.setValue("providerId", value)
    form.setValue("modelId", undefined)
  }

  const toolBindingGroups = useMemo<BindingGroup[]>(() => {
    return toolGroups.map((group) => ({
      key: group.toolkit,
      title: t(`ai:tools.toolkits.${group.toolkit}.name`),
      description: t(`ai:tools.toolkits.${group.toolkit}.description`),
      items: group.tools,
    }))
  }, [toolGroups, t])

  return (
    <Form {...form}>
      <form id="agent-form" onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        {/* === Basic Info === */}
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="text-base">{t("ai:agents.sections.basic")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <FormField control={form.control} name="name" render={({ field }) => (
                <FormItem className="sm:col-span-2">
                  <FormLabel>{t("ai:agents.name")}</FormLabel>
                  <FormControl><Input placeholder={t("ai:agents.namePlaceholder")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="visibility" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("ai:agents.visibility")}</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl><SelectTrigger className="w-full"><SelectValue /></SelectTrigger></FormControl>
                    <SelectContent>
                      <SelectItem value="private">{t("ai:agents.visibilityOptions.private")}</SelectItem>
                      <SelectItem value="team">{t("ai:agents.visibilityOptions.team")}</SelectItem>
                      <SelectItem value="public">{t("ai:agents.visibilityOptions.public")}</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )} />
            </div>
            <FormField control={form.control} name="description" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("ai:agents.description")}</FormLabel>
                <FormControl><Textarea placeholder={t("ai:agents.descriptionPlaceholder")} rows={2} {...field} /></FormControl>
                <FormMessage />
              </FormItem>
            )} />
          </CardContent>
        </Card>

        {/* === Model Config (assistant only) === */}
        {agentType === "assistant" && (
          <Card>
            <CardHeader className="pb-4">
              <CardTitle className="text-base">{t("ai:agents.sections.modelConfig")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <FormField control={form.control} name="providerId" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("ai:agents.provider")}</FormLabel>
                    <Select value={field.value ?? ""} onValueChange={handleProviderChange}>
                      <FormControl><SelectTrigger className="w-full"><SelectValue placeholder={t("ai:agents.selectProvider")} /></SelectTrigger></FormControl>
                      <SelectContent>
                        {providers.map((p) => (
                          <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={form.control} name="modelId" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("ai:agents.model")}</FormLabel>
                    <Select
                      value={field.value ? String(field.value) : ""}
                      onValueChange={(v) => field.onChange(Number(v))}
                      disabled={selectedProviderId === ""}
                    >
                      <FormControl><SelectTrigger className="w-full"><SelectValue placeholder={t("ai:agents.selectModel")} /></SelectTrigger></FormControl>
                      <SelectContent>
                        {models.map((m) => (
                          <SelectItem key={m.id} value={String(m.id)}>{m.displayName}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={form.control} name="strategy" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("ai:agents.strategy")}</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value || "react"}>
                      <FormControl><SelectTrigger className="w-full"><SelectValue /></SelectTrigger></FormControl>
                      <SelectContent>
                        <SelectItem value="react">{t("ai:agents.strategies.react")}</SelectItem>
                        <SelectItem value="plan_and_execute">{t("ai:agents.strategies.plan_and_execute")}</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )} />
              </div>

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <FormField control={form.control} name="temperature" render={({ field }) => (
                  <FormItem>
                    <FormLabel className="flex items-center gap-2">
                      {t("ai:agents.temperature")}
                      <span className="text-xs font-mono bg-muted px-2 py-0.5 rounded">{field.value}</span>
                    </FormLabel>
                    <FormControl>
                      <input
                        type="range"
                        min={0} max={2} step={0.1}
                        value={field.value ?? 0.7}
                        onChange={(e) => field.onChange(parseFloat(e.target.value))}
                        className="w-full h-2 bg-muted rounded-lg appearance-none cursor-pointer accent-primary"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={form.control} name="maxTokens" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("ai:agents.maxTokens")}</FormLabel>
                    <FormControl><Input type="number" {...field} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={form.control} name="maxTurns" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("ai:agents.maxTurns")}</FormLabel>
                    <FormControl><Input type="number" {...field} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
              </div>
            </CardContent>
          </Card>
        )}

        {/* === Runtime Config (coding only) === */}
        {agentType === "coding" && (
          <Card>
            <CardHeader className="pb-4">
              <CardTitle className="text-base">{t("ai:agents.sections.execution")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <FormField control={form.control} name="runtime" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("ai:agents.runtime")}</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value || "claude-code"}>
                      <FormControl><SelectTrigger className="w-full"><SelectValue /></SelectTrigger></FormControl>
                      <SelectContent>
                        <SelectItem value="claude-code">{t("ai:agents.runtimes.claude-code")}</SelectItem>
                        <SelectItem value="codex">{t("ai:agents.runtimes.codex")}</SelectItem>
                        <SelectItem value="opencode">{t("ai:agents.runtimes.opencode")}</SelectItem>
                        <SelectItem value="aider">{t("ai:agents.runtimes.aider")}</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={form.control} name="execMode" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("ai:agents.execMode")}</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value || "local"}>
                      <FormControl><SelectTrigger className="w-full"><SelectValue /></SelectTrigger></FormControl>
                      <SelectContent>
                        <SelectItem value="local">{t("ai:agents.execModes.local")}</SelectItem>
                        <SelectItem value="remote">{t("ai:agents.execModes.remote")}</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )} />
              </div>
              <FormField control={form.control} name="workspace" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("ai:agents.workspace")}</FormLabel>
                  <FormControl><Input placeholder={t("ai:agents.workspacePlaceholder")} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              {watchExecMode === "remote" && (
                <FormField control={form.control} name="nodeId" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("ai:agents.node")}</FormLabel>
                    <FormControl><Input type="number" placeholder={t("ai:agents.selectNode")} {...field} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
              )}
            </CardContent>
          </Card>
        )}

        {/* === Capability Bindings === */}
        <BindingSelectorSection
          title={t("ai:agents.tools")}
          description={t("ai:agents.toolsDescription")}
          sheetTitle={t("ai:agents.selectTools")}
          sheetDescription={t("ai:agents.toolsSheetDescription")}
          groups={toolBindingGroups}
          isLoading={toolsLoading}
          value={form.watch("toolIds")}
          onChange={(ids) => form.setValue("toolIds", ids)}
        />
        <BindingSelectorSection
          title={t("ai:agents.mcpServers")}
          description={t("ai:agents.mcpServersDescription")}
          sheetTitle={t("ai:agents.selectMcpServers")}
          sheetDescription={t("ai:agents.mcpServersDescription")}
          items={mcpItems}
          isLoading={mcpLoading}
          value={form.watch("mcpServerIds")}
          onChange={(ids) => form.setValue("mcpServerIds", ids)}
        />
        <BindingSelectorSection
          title={t("ai:agents.skills")}
          description={t("ai:agents.skillsDescription")}
          sheetTitle={t("ai:agents.selectSkills")}
          sheetDescription={t("ai:agents.skillsDescription")}
          items={skillItems}
          isLoading={skillsLoading}
          value={form.watch("skillIds")}
          onChange={(ids) => form.setValue("skillIds", ids)}
        />
        <BindingSelectorSection
          title={t("ai:agents.knowledgeBases")}
          description={t("ai:agents.knowledgeBasesDescription")}
          sheetTitle={t("ai:agents.selectKnowledgeBases")}
          sheetDescription={t("ai:agents.knowledgeBasesDescription")}
          items={kbItems}
          isLoading={kbLoading}
          value={form.watch("knowledgeBaseIds")}
          onChange={(ids) => form.setValue("knowledgeBaseIds", ids)}
        />
        <BindingSelectorSection
          title={t("ai:agents.knowledgeGraphs")}
          description={t("ai:agents.knowledgeGraphsDescription")}
          sheetTitle={t("ai:agents.selectKnowledgeGraphs")}
          sheetDescription={t("ai:agents.knowledgeGraphsDescription")}
          items={kgItems}
          isLoading={kgLoading}
          value={form.watch("knowledgeGraphIds")}
          onChange={(ids) => form.setValue("knowledgeGraphIds", ids)}
        />

        {/* === Prompts (always visible) === */}
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="text-base">{t("ai:agents.sections.prompts")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField control={form.control} name="systemPrompt" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("ai:agents.systemPrompt")}</FormLabel>
                <FormControl>
                  <Textarea
                    placeholder={t("ai:agents.systemPromptPlaceholder")}
                    rows={6}
                    className="min-h-[120px] resize-y font-mono text-sm"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="instructions" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("ai:agents.instructions")}</FormLabel>
                <FormControl>
                  <Textarea
                    placeholder={t("ai:agents.instructionsPlaceholder")}
                    rows={5}
                    className="min-h-[100px] resize-y"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )} />
          </CardContent>
        </Card>
      </form>
    </Form>
  )
}
