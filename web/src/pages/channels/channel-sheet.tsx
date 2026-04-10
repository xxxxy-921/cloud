import { useEffect, useMemo } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2, Plug } from "lucide-react"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { CHANNEL_TYPES, type ConfigField } from "./channel-types"

export interface ChannelItem {
  id: number
  name: string
  type: string
  config: string
  enabled: boolean
  createdAt: string
  updatedAt: string
}

const schema = z.object({
  name: z.string().min(1, "通道名称不能为空"),
  type: z.string().min(1, "请选择通道类型"),
  config: z.record(z.string(), z.unknown()),
})

type FormValues = z.infer<typeof schema>

interface ChannelSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  channel: ChannelItem | null
}

export function ChannelSheet({ open, onOpenChange, channel }: ChannelSheetProps) {
  const queryClient = useQueryClient()
  const isEditing = channel !== null

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "", type: "email", config: {} },
  })

  const selectedType = form.watch("type")
  const configSchema = useMemo(
    () => CHANNEL_TYPES[selectedType]?.configSchema ?? [],
    [selectedType],
  )

  useEffect(() => {
    if (open) {
      if (channel) {
        let cfg: Record<string, unknown> = {}
        try { cfg = JSON.parse(channel.config) } catch { /* ignore */ }
        form.reset({ name: channel.name, type: channel.type, config: cfg })
      } else {
        // Build defaults from schema
        const defaults: Record<string, unknown> = {}
        const defaultType = "email"
        const fields = CHANNEL_TYPES[defaultType]?.configSchema ?? []
        for (const f of fields) {
          if (f.default !== undefined) defaults[f.key] = f.default
          else if (f.type === "boolean") defaults[f.key] = false
          else if (f.type === "number") defaults[f.key] = 0
          else defaults[f.key] = ""
        }
        form.reset({ name: "", type: defaultType, config: defaults })
      }
    }
  }, [open, channel, form])

  // Reset config defaults when type changes (only in create mode)
  useEffect(() => {
    if (!isEditing && open) {
      const defaults: Record<string, unknown> = {}
      for (const f of configSchema) {
        if (f.default !== undefined) defaults[f.key] = f.default
        else if (f.type === "boolean") defaults[f.key] = false
        else if (f.type === "number") defaults[f.key] = 0
        else defaults[f.key] = ""
      }
      form.setValue("config", defaults)
    }
  }, [selectedType, isEditing, open, configSchema, form])

  const createMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.post("/api/v1/channels", {
        name: values.name,
        type: values.type,
        config: JSON.stringify(values.config),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["channels"] })
      onOpenChange(false)
      toast.success("通道创建成功")
    },
    onError: (err) => toast.error(err.message),
  })

  const updateMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.put(`/api/v1/channels/${channel!.id}`, {
        name: values.name,
        config: JSON.stringify(values.config),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["channels"] })
      onOpenChange(false)
      toast.success("通道更新成功")
    },
    onError: (err) => toast.error(err.message),
  })

  const testMutation = useMutation({
    mutationFn: async () => {
      if (!channel) throw new Error("请先保存通道后再测试")
      const res = await api.post<{ success: boolean; error?: string }>(
        `/api/v1/channels/${channel.id}/test`,
      )
      if (!res.success) throw new Error(res.error || "测试失败")
      return res
    },
    onSuccess: () => toast.success("连接测试成功"),
    onError: (err) => toast.error(`连接测试失败: ${err.message}`),
  })

  function onSubmit(values: FormValues) {
    if (isEditing) {
      updateMutation.mutate(values)
    } else {
      createMutation.mutate(values)
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md">
        <SheetHeader>
          <SheetTitle>{isEditing ? "编辑通道" : "新建通道"}</SheetTitle>
          <SheetDescription className="sr-only">
            {isEditing ? "修改消息通道配置" : "配置新的消息通道"}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-4 px-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>通道名称</FormLabel>
                  <FormControl>
                    <Input placeholder="例如：系统邮件" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="type"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>通道类型</FormLabel>
                  {isEditing ? (
                    <div>
                      <Input
                        value={CHANNEL_TYPES[field.value]?.label ?? field.value}
                        disabled
                      />
                      <p className="text-xs text-muted-foreground mt-1">通道类型创建后不可更改</p>
                    </div>
                  ) : (
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="请选择通道类型" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {Object.entries(CHANNEL_TYPES).map(([key, def]) => (
                          <SelectItem key={key} value={key}>
                            {def.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />

            {configSchema.length > 0 && (
              <div className="space-y-3 rounded-lg border p-4">
                <p className="text-sm font-medium">连接配置</p>
                {configSchema.map((field) => (
                  <ConfigFieldInput
                    key={field.key}
                    field={field}
                    form={form}
                    isEditing={isEditing}
                  />
                ))}
              </div>
            )}

            <SheetFooter>
              {isEditing && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => testMutation.mutate()}
                  disabled={testMutation.isPending}
                >
                  {testMutation.isPending ? (
                    <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <Plug className="mr-1.5 h-3.5 w-3.5" />
                  )}
                  测试连接
                </Button>
              )}
              <Button type="submit" size="sm" disabled={isPending}>
                {isPending ? "保存中..." : "保存"}
              </Button>
            </SheetFooter>
          </form>
        </Form>
      </SheetContent>
    </Sheet>
  )
}

function ConfigFieldInput({
  field,
  form,
  isEditing,
}: {
  field: ConfigField
  form: ReturnType<typeof useForm<FormValues>>
  isEditing: boolean
}) {
  const name = `config.${field.key}` as const

  if (field.type === "boolean") {
    return (
      <FormField
        control={form.control}
        name={name}
        render={({ field: formField }) => (
          <FormItem className="flex items-center justify-between">
            <FormLabel className="mt-0">{field.label}</FormLabel>
            <FormControl>
              <Switch
                checked={!!formField.value}
                onCheckedChange={formField.onChange}
              />
            </FormControl>
          </FormItem>
        )}
      />
    )
  }

  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field: formField }) => (
        <FormItem>
          <FormLabel>{field.label}</FormLabel>
          <FormControl>
            <Input
              type={field.sensitive ? "password" : field.type === "number" ? "number" : "text"}
              placeholder={
                field.sensitive && isEditing
                  ? "留空则不修改"
                  : field.placeholder || ""
              }
              value={
                field.sensitive && isEditing && formField.value === "******"
                  ? ""
                  : (formField.value as string | number) ?? ""
              }
              onChange={(e) => {
                const val = field.type === "number"
                  ? Number(e.target.value)
                  : e.target.value
                formField.onChange(val)
              }}
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}
