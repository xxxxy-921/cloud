import { useEffect } from "react"
import { useTranslation } from "react-i18next"
import { useForm, useWatch } from "react-hook-form"
import { zodResolver } from "@/lib/form"
import { z } from "zod"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { agentApi, api, type AgentInfo, type PaginatedResponse } from "@/lib/api"
import { toast } from "sonner"
import {
  Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle,
} from "@/components/ui/sheet"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { Bot, Settings2, Cpu, FileText } from "lucide-react"

const agentSchema = z.object({
  name: z.string().min(1),
  description: z.string().optional(),
  type: z.enum(["assistant", "coding"]),
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
})

type AgentFormValues = z.infer<typeof agentSchema>

interface AgentSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  agent: AgentInfo | null
}

interface ProviderItem {
  id: number
  name: string
}

interface ModelItem {
  id: number
  displayName: string
  modelId: string
  type: string
  providerId: number
}

export function AgentSheet({ open, onOpenChange, agent }: AgentSheetProps) {
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()
  const isEditing = !!agent

  const form = useForm<AgentFormValues>({
    resolver: zodResolver(agentSchema),
    defaultValues: {
      name: "",
      description: "",
      type: "assistant",
      visibility: "team",
      strategy: "react",
      temperature: 0.7,
      maxTokens: 4096,
      maxTurns: 10,
      execMode: "local",
    },
  })

  const watchType = useWatch({ control: form.control, name: "type" })
  const watchExecMode = useWatch({ control: form.control, name: "execMode" })
  const selectedProviderId = useWatch({ control: form.control, name: "providerId" }) ?? ""

  // Fetch providers
  const { data: providersData } = useQuery({
    queryKey: ["ai-providers"],
    queryFn: () => api.get<PaginatedResponse<ProviderItem>>("/api/v1/ai/providers?pageSize=100"),
    enabled: open,
  })
  const providers = providersData?.items ?? []

  // For edit mode: resolve provider from the agent's modelId
  const { data: editModelDetail } = useQuery({
    queryKey: ["ai-model-detail", agent?.modelId],
    queryFn: () => api.get<ModelItem>(`/api/v1/ai/models/${agent!.modelId}`),
    enabled: open && !!agent?.modelId,
  })
  const editProviderId = editModelDetail?.providerId ? String(editModelDetail.providerId) : ""

  // Fetch LLM models filtered by selected provider
  const { data: modelsData } = useQuery({
    queryKey: ["ai-models-llm", selectedProviderId],
    queryFn: () => api.get<PaginatedResponse<ModelItem>>(`/api/v1/ai/models?type=llm&providerId=${selectedProviderId}&pageSize=100`),
    enabled: open && selectedProviderId !== "",
  })
  const models = modelsData?.items ?? []

  function handleProviderChange(value: string) {
    form.setValue("providerId", value)
    form.setValue("modelId", undefined)
  }

  useEffect(() => {
    if (!open) return
    if (agent) {
      form.reset({
        name: agent.name,
        description: agent.description,
        type: agent.type,
        visibility: agent.visibility,
        strategy: agent.strategy || "react",
        providerId: editProviderId,
        modelId: agent.modelId ?? undefined,
        systemPrompt: agent.systemPrompt || "",
        temperature: agent.temperature,
        maxTokens: agent.maxTokens,
        maxTurns: agent.maxTurns,
        runtime: agent.runtime || "claude-code",
        execMode: agent.execMode || "local",
        workspace: agent.workspace || "",
        nodeId: agent.nodeId ?? undefined,
        instructions: agent.instructions || "",
      })
    } else {
      form.reset({
        name: "",
        description: "",
        type: "assistant",
        visibility: "team",
        strategy: "react",
        temperature: 0.7,
        maxTokens: 4096,
        maxTurns: 10,
        execMode: "local",
      })
    }
  }, [open, agent, editProviderId, form])

  const mutation = useMutation({
    mutationFn: (values: AgentFormValues) => {
      if (isEditing) {
        return agentApi.update(agent.id, values)
      }
      return agentApi.create(values)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-agents"] })
      toast.success(isEditing ? t("ai:agents.updateSuccess") : t("ai:agents.createSuccess"))
      onOpenChange(false)
    },
    onError: (err) => toast.error(err.message),
  })

  function onSubmit(values: AgentFormValues) {
    mutation.mutate(values)
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-xl overflow-y-auto px-5">
        <SheetHeader className="pb-4">
          <SheetTitle className="flex items-center gap-2">
            <Bot className="h-5 w-5" />
            {isEditing ? t("ai:agents.edit") : t("ai:agents.create")}
          </SheetTitle>
          <SheetDescription className="sr-only">
            {isEditing ? t("ai:agents.edit") : t("ai:agents.create")}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-6">
            {/* === Section: Basic Info === */}
            <div className="space-y-4">
              <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                <FileText className="h-4 w-4" />
                {t("ai:agents.sections.basic")}
              </div>

              <div className="grid grid-cols-2 gap-3">
                <FormField control={form.control} name="name" render={({ field }) => (
                  <FormItem className="col-span-2">
                    <FormLabel>{t("ai:agents.name")}</FormLabel>
                    <FormControl><Input placeholder={t("ai:agents.namePlaceholder")} {...field} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />

                <FormField control={form.control} name="type" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("ai:agents.type")}</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value} disabled={isEditing}>
                      <FormControl><SelectTrigger className="w-full"><SelectValue /></SelectTrigger></FormControl>
                      <SelectContent>
                        <SelectItem value="assistant">{t("ai:agents.agentTypes.assistant")}</SelectItem>
                        <SelectItem value="coding">{t("ai:agents.agentTypes.coding")}</SelectItem>
                      </SelectContent>
                    </Select>
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
            </div>

            <Separator />

            {/* === Section: Assistant-specific fields === */}
            {watchType === "assistant" && (
              <>
                <div className="space-y-4">
                  <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                    <Settings2 className="h-4 w-4" />
                    {t("ai:agents.sections.modelConfig")}
                  </div>

                  <div className="grid grid-cols-2 gap-3">
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
                          onValueChange={(v) => field.onChange(Number(v))}
                          value={field.value ? String(field.value) : ""}
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
                  </div>

                  <div className="grid grid-cols-2 gap-3">
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

                  <div className="grid grid-cols-2 gap-3">
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

                    <FormField control={form.control} name="maxTurns" render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t("ai:agents.maxTurns")}</FormLabel>
                        <FormControl><Input type="number" {...field} /></FormControl>
                        <FormMessage />
                      </FormItem>
                    )} />
                  </div>

                  <FormField control={form.control} name="maxTokens" render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("ai:agents.maxTokens")}</FormLabel>
                      <FormControl><Input type="number" {...field} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )} />
                </div>

                <Separator />
              </>
            )}

            {/* === Section: Coding-specific fields === */}
            {watchType === "coding" && (
              <>
                <div className="space-y-4">
                  <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                    <Cpu className="h-4 w-4" />
                    {t("ai:agents.sections.execution")}
                  </div>

                  <div className="grid grid-cols-2 gap-3">
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
                </div>

                <Separator />
              </>
            )}

            {/* === Section: Instructions === */}
            <div className="space-y-4">
              <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                <FileText className="h-4 w-4" />
                {t("ai:agents.sections.instructions")}
              </div>

              <FormField control={form.control} name="instructions" render={({ field }) => (
                <FormItem>
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
            </div>

            <SheetFooter className="pt-2 border-t">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("common:cancel")}
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? t("common:saving") : t("common:save")}
              </Button>
            </SheetFooter>
          </form>
        </Form>
      </SheetContent>
    </Sheet>
  )
}
