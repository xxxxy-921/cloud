import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"

const schema = z.object({
  // Password policy
  passwordMinLength: z.number().int().min(1, "最小长度不能小于 1"),
  passwordRequireUpper: z.boolean(),
  passwordRequireLower: z.boolean(),
  passwordRequireNumber: z.boolean(),
  passwordRequireSpecial: z.boolean(),
  passwordExpiryDays: z.number().int().min(0, "不能小于 0"),

  // Login security
  loginMaxAttempts: z.number().int().min(0, "不能小于 0"),
  loginLockoutMinutes: z.number().int().min(0, "不能小于 0"),
  captchaProvider: z.enum(["none", "image"]),

  // Session
  maxConcurrentSessions: z.number().int().min(0, "不能小于 0"),
  sessionTimeoutMinutes: z.number().int().min(1, "不能小于 1"),

  // Two-factor
  requireTwoFactor: z.boolean(),

  // Registration
  registrationOpen: z.boolean(),
  defaultRoleCode: z.string(),
})

type FormValues = z.infer<typeof schema>

interface SecuritySettings extends FormValues {}

const defaultValues: FormValues = {
  passwordMinLength: 8,
  passwordRequireUpper: false,
  passwordRequireLower: false,
  passwordRequireNumber: false,
  passwordRequireSpecial: false,
  passwordExpiryDays: 0,
  loginMaxAttempts: 5,
  loginLockoutMinutes: 30,
  captchaProvider: "none",
  maxConcurrentSessions: 5,
  sessionTimeoutMinutes: 10080,
  requireTwoFactor: false,
  registrationOpen: false,
  defaultRoleCode: "",
}

interface RoleOption {
  id: number
  name: string
  code: string
}

function NumberField({
  control,
  name,
  label,
  description,
  min = 0,
}: {
  control: any
  name: keyof FormValues
  label: string
  description?: string
  min?: number
}) {
  return (
    <FormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>{label}</FormLabel>
          <FormControl>
            <Input
              type="number"
              min={min}
              className="max-w-[200px]"
              {...field}
              onChange={(e) => field.onChange(e.target.valueAsNumber)}
            />
          </FormControl>
          {description && <FormDescription>{description}</FormDescription>}
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

function SwitchField({
  control,
  name,
  label,
  description,
}: {
  control: any
  name: keyof FormValues
  label: string
  description?: string
}) {
  return (
    <FormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem className="flex items-center justify-between rounded-lg border p-3">
          <div className="space-y-0.5">
            <FormLabel className="text-sm">{label}</FormLabel>
            {description && (
              <FormDescription className="text-xs">{description}</FormDescription>
            )}
          </div>
          <FormControl>
            <Switch checked={field.value as boolean} onCheckedChange={field.onChange} />
          </FormControl>
        </FormItem>
      )}
    />
  )
}

export function SecurityCard() {
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ["settings", "security"],
    queryFn: () => api.get<SecuritySettings>("/api/v1/settings/security"),
  })

  const { data: roles } = useQuery({
    queryKey: ["roles-options"],
    queryFn: () =>
      api.get<{ items: RoleOption[] }>("/api/v1/roles").then((r) => r.items),
  })

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues,
  })

  useEffect(() => {
    if (data) {
      form.reset({ ...defaultValues, ...data })
    }
  }, [data, form])

  const mutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.put("/api/v1/settings/security", values),
    onSuccess: (_data, values) => {
      queryClient.invalidateQueries({ queryKey: ["settings", "security"] })
      form.reset(values)
      toast.success("安全设置已保存")
    },
    onError: () => toast.error("保存失败"),
  })

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex h-32 items-center justify-center text-muted-foreground">
          加载中...
        </CardContent>
      </Card>
    )
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
        className="space-y-6"
      >
        {/* 密码策略 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">密码策略</CardTitle>
            <CardDescription>设置密码复杂度和过期规则</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <NumberField
              control={form.control}
              name="passwordMinLength"
              label="最小长度"
              min={1}
            />
            <div className="space-y-2">
              <SwitchField
                control={form.control}
                name="passwordRequireUpper"
                label="要求大写字母"
              />
              <SwitchField
                control={form.control}
                name="passwordRequireLower"
                label="要求小写字母"
              />
              <SwitchField
                control={form.control}
                name="passwordRequireNumber"
                label="要求数字"
              />
              <SwitchField
                control={form.control}
                name="passwordRequireSpecial"
                label="要求特殊字符"
              />
            </div>
            <NumberField
              control={form.control}
              name="passwordExpiryDays"
              label="密码过期天数"
              description="设为 0 表示永不过期"
            />
          </CardContent>
        </Card>

        {/* 登录安全 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">登录安全</CardTitle>
            <CardDescription>登录失败锁定和验证码设置</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <NumberField
                control={form.control}
                name="loginMaxAttempts"
                label="最大失败次数"
                description="设为 0 表示不限制"
              />
              <NumberField
                control={form.control}
                name="loginLockoutMinutes"
                label="锁定时长（分钟）"
              />
            </div>
            <FormField
              control={form.control}
              name="captchaProvider"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>登录验证码</FormLabel>
                  <Select
                    key={field.value ?? defaultValues.captchaProvider}
                    defaultValue={field.value ?? defaultValues.captchaProvider}
                    onValueChange={field.onChange}
                  >
                    <FormControl>
                      <SelectTrigger className="max-w-[200px]">
                        <SelectValue placeholder="选择验证码方式" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent
                      position="popper"
                      side="bottom"
                      sideOffset={4}
                      className="bg-background shadow-md"
                    >
                      <SelectItem value="none">关闭</SelectItem>
                      <SelectItem value="image">图形验证码</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        {/* 会话管理 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">会话管理</CardTitle>
            <CardDescription>控制用户会话行为</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <NumberField
                control={form.control}
                name="maxConcurrentSessions"
                label="最大并发会话数"
                description="设为 0 表示不限制"
              />
              <NumberField
                control={form.control}
                name="sessionTimeoutMinutes"
                label="会话超时（分钟）"
                min={1}
              />
            </div>
          </CardContent>
        </Card>

        {/* 两步验证 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">两步验证</CardTitle>
            <CardDescription>TOTP 两步验证全局策略</CardDescription>
          </CardHeader>
          <CardContent>
            <SwitchField
              control={form.control}
              name="requireTwoFactor"
              label="强制所有用户启用两步验证"
              description="启用后未设置 2FA 的用户登录时会被要求配置"
            />
          </CardContent>
        </Card>

        {/* 注册设置 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">注册设置</CardTitle>
            <CardDescription>控制用户自主注册</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <SwitchField
              control={form.control}
              name="registrationOpen"
              label="开放注册"
              description="允许新用户自行注册账号"
            />
            <FormField
              control={form.control}
              name="defaultRoleCode"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>默认角色</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger className="max-w-[200px]">
                        <SelectValue placeholder="选择角色" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {roles?.map((r) => (
                        <SelectItem key={r.code} value={r.code}>
                          {r.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    新注册用户自动分配的角色
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <Button
          type="submit"
          disabled={!form.formState.isDirty || mutation.isPending}
        >
          {mutation.isPending ? "保存中..." : "保存"}
        </Button>
      </form>
    </Form>
  )
}
