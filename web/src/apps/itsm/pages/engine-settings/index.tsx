import { useState, type ComponentType, type ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { ExternalLink, Route, Save, ShieldAlert } from "lucide-react"
import { useNavigate } from "react-router"
import { toast } from "sonner"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Slider } from "@/components/ui/slider"
import {
  type SmartStaffingConfig,
  type SmartStaffingConfigUpdate,
  fetchModels,
  fetchProviders,
  fetchSmartStaffingConfig,
  fetchUsers,
  updateSmartStaffingConfig,
} from "../../api"

function EngineSettingSection({
  icon,
  title,
  description,
  children,
}: {
  icon: ComponentType<{ className?: string }>
  title: string
  description: string
  children: ReactNode
}) {
  const Icon = icon

  return (
    <Card className="w-full gap-0 overflow-hidden py-0">
      <CardContent className="grid gap-5 px-5 py-5 xl:grid-cols-[minmax(240px,320px)_1fr] xl:items-start">
        <div className="flex min-w-0 gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border bg-muted/45">
            <Icon className="h-4.5 w-4.5 text-foreground" />
          </div>
          <div className="min-w-0">
            <CardTitle className="text-[15px]">{title}</CardTitle>
            <CardDescription className="mt-1 text-xs leading-5">{description}</CardDescription>
          </div>
        </div>
        <div className="min-w-0">{children}</div>
      </CardContent>
    </Card>
  )
}

function PathBuilderFields({
  providerId,
  modelId,
  temperature,
  maxRetries,
  timeoutSeconds,
  onProviderChange,
  onModelChange,
  onTemperatureChange,
  onMaxRetriesChange,
  onTimeoutSecondsChange,
}: {
  providerId: number
  modelId: number
  temperature: number
  maxRetries: number
  timeoutSeconds: number
  onProviderChange: (id: number) => void
  onModelChange: (id: number) => void
  onTemperatureChange: (v: number) => void
  onMaxRetriesChange: (v: number) => void
  onTimeoutSecondsChange: (v: number) => void
}) {
  const { t } = useTranslation("itsm")
  const navigate = useNavigate()

  const { data: providers = [] } = useQuery({
    queryKey: ["ai-providers"],
    queryFn: fetchProviders,
  })

  const { data: models = [] } = useQuery({
    queryKey: ["ai-models", providerId],
    queryFn: () => fetchModels(providerId),
    enabled: providerId > 0,
  })

  if (providers.length === 0) {
    return (
      <Alert>
        <AlertDescription className="flex items-center justify-between gap-4">
          <span>{t("engineConfig.noProviders")}</span>
          <Button variant="link" size="sm" className="h-auto p-0" onClick={() => navigate("/ai/providers")}>
            {t("engineConfig.goToProviders")}
            <ExternalLink className="ml-1 h-3 w-3" />
          </Button>
        </AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="grid gap-4 lg:grid-cols-2 2xl:grid-cols-[minmax(180px,220px)_minmax(220px,280px)_minmax(260px,1fr)_160px_180px] 2xl:items-start">
      <div className="space-y-1.5">
        <Label>{t("engineConfig.provider")}</Label>
        <Select
          value={providerId ? String(providerId) : ""}
          onValueChange={(v) => {
            onProviderChange(Number(v))
            onModelChange(0)
          }}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder={t("engineConfig.providerPlaceholder")} />
          </SelectTrigger>
          <SelectContent>
            {providers.map((p) => (
              <SelectItem key={p.id} value={String(p.id)}>
                {p.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-1.5">
        <Label>{t("engineConfig.model")}</Label>
        <Select value={modelId ? String(modelId) : ""} onValueChange={(v) => onModelChange(Number(v))} disabled={!providerId}>
          <SelectTrigger className="w-full">
            <SelectValue placeholder={t("engineConfig.modelPlaceholder")} />
          </SelectTrigger>
          <SelectContent>
            {models.map((m) => (
              <SelectItem key={m.id} value={String(m.id)}>
                {m.displayName}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <div className="flex items-center justify-between gap-3">
          <Label>{t("engineConfig.temperature")}</Label>
          <span className="rounded-full border border-border/55 bg-background/35 px-2 py-0.5 font-mono text-[11px] text-muted-foreground">{temperature.toFixed(2)}</span>
        </div>
        <Slider min={0} max={1} step={0.05} value={[temperature]} onValueChange={([v]) => onTemperatureChange(v)} />
      </div>
      <div className="space-y-1.5">
        <Label>{t("engineConfig.maxRetries")}</Label>
        <Input type="number" min={0} max={10} value={maxRetries} onChange={(e) => onMaxRetriesChange(Number(e.target.value))} />
      </div>
      <div className="space-y-1.5">
        <Label>{t("engineConfig.timeoutSeconds")}</Label>
        <div className="flex items-center gap-2">
          <Input type="number" min={10} max={300} value={timeoutSeconds} onChange={(e) => onTimeoutSecondsChange(Number(e.target.value))} />
          <span className="text-xs text-muted-foreground">{t("engineConfig.seconds")}</span>
        </div>
      </div>
    </div>
  )
}

function configFormKey(config: SmartStaffingConfig) {
  return [
    config.runtime.pathBuilder.providerId,
    config.runtime.pathBuilder.modelId,
    config.runtime.pathBuilder.temperature,
    config.runtime.pathBuilder.maxRetries,
    config.runtime.pathBuilder.timeoutSeconds,
    config.runtime.guard.auditLevel,
    config.runtime.guard.fallbackAssignee,
  ].join(":")
}

function EngineSettingsForm({ config }: { config: SmartStaffingConfig }) {
  const { t } = useTranslation(["itsm", "common"])
  const queryClient = useQueryClient()

  const { data: fallbackUsers = [] } = useQuery({
    queryKey: ["users-for-engine-settings-fallback"],
    queryFn: () => fetchUsers(),
  })

  const [pathProviderId, setPathProviderId] = useState(config.runtime.pathBuilder.providerId)
  const [pathModelId, setPathModelId] = useState(config.runtime.pathBuilder.modelId)
  const [pathTemperature, setPathTemperature] = useState(config.runtime.pathBuilder.temperature)
  const [pathMaxRetries, setPathMaxRetries] = useState(config.runtime.pathBuilder.maxRetries)
  const [pathTimeoutSeconds, setPathTimeoutSeconds] = useState(config.runtime.pathBuilder.timeoutSeconds)
  const [auditLevel, setAuditLevel] = useState(config.runtime.guard.auditLevel)
  const [fallbackAssignee, setFallbackAssignee] = useState(config.runtime.guard.fallbackAssignee)

  const fallbackUserKnown = fallbackAssignee === 0 || fallbackUsers.some((u) => u.id === fallbackAssignee)

  const saveMut = useMutation({
    mutationFn: (data: SmartStaffingConfigUpdate) => updateSmartStaffingConfig(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["itsm-engine-settings-config"] })
      toast.success(t("itsm:engineConfig.engineSaveSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  function handleSave() {
    saveMut.mutate({
      posts: config.posts,
      runtime: {
        pathBuilder: {
          modelId: pathModelId,
          temperature: pathTemperature,
          maxRetries: pathMaxRetries,
          timeoutSeconds: pathTimeoutSeconds,
        },
        guard: { auditLevel, fallbackAssignee },
      },
    })
  }

  return (
    <div className="workspace-page">
      <div className="workspace-page-header">
        <div className="min-w-0">
          <h2 className="workspace-page-title">{t("itsm:engineConfig.engineSettingsTitle")}</h2>
          <p className="workspace-page-description">{t("itsm:engineConfig.engineSettingsDesc")}</p>
        </div>
        <Button className="shrink-0" onClick={handleSave} disabled={saveMut.isPending}>
          <Save className="mr-1.5 h-4 w-4" />
          {saveMut.isPending ? t("common:saving") : t("common:save")}
        </Button>
      </div>

      <EngineSettingSection
        icon={Route}
        title={t("itsm:engineConfig.pathBuilderTitle")}
        description={t("itsm:engineConfig.pathBuilderDesc")}
      >
        <PathBuilderFields
          providerId={pathProviderId}
          modelId={pathModelId}
          temperature={pathTemperature}
          maxRetries={pathMaxRetries}
          timeoutSeconds={pathTimeoutSeconds}
          onProviderChange={setPathProviderId}
          onModelChange={setPathModelId}
          onTemperatureChange={setPathTemperature}
          onMaxRetriesChange={setPathMaxRetries}
          onTimeoutSecondsChange={setPathTimeoutSeconds}
        />
      </EngineSettingSection>

      <EngineSettingSection
        icon={ShieldAlert}
        title={t("itsm:engineConfig.guardTitle")}
        description={t("itsm:engineConfig.guardDesc")}
      >
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-[220px_320px]">
          <div className="space-y-1.5">
            <Label>{t("engineConfig.auditLevel")}</Label>
            <Select value={auditLevel} onValueChange={setAuditLevel}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="full">{t("engineConfig.logFull")}</SelectItem>
                <SelectItem value="summary">{t("engineConfig.logSummary")}</SelectItem>
                <SelectItem value="off">{t("engineConfig.logOff")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>{t("engineConfig.fallbackAssignee")}</Label>
            <Select value={String(fallbackAssignee)} onValueChange={(v) => setFallbackAssignee(Number(v))}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t("engineConfig.fallbackAssigneePlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">{t("engineConfig.fallbackAssigneeNone")}</SelectItem>
                {!fallbackUserKnown && (
                  <SelectItem value={String(fallbackAssignee)}>
                    {t("engineConfig.fallbackAssigneeUnknown", { id: fallbackAssignee })}
                  </SelectItem>
                )}
                {fallbackUsers.map((user) => (
                  <SelectItem key={user.id} value={String(user.id)}>
                    {user.username}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </EngineSettingSection>
    </div>
  )
}

export function Component() {
  const { t } = useTranslation("common")

  const { data: config, isLoading } = useQuery({
    queryKey: ["itsm-engine-settings-config"],
    queryFn: fetchSmartStaffingConfig,
  })

  if (isLoading || !config) {
    return (
      <div className="flex items-center justify-center py-12">
        <span className="text-muted-foreground">{t("loading")}</span>
      </div>
    )
  }

  return <EngineSettingsForm key={configFormKey(config)} config={config} />
}
