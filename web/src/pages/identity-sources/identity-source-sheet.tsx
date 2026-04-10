import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2, Plug } from "lucide-react"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
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
  FormDescription,
} from "@/components/ui/form"

export interface IdentitySourceItem {
  id: number
  name: string
  type: string
  enabled: boolean
  domains: string
  forceSso: boolean
  defaultRoleId: number
  conflictStrategy: string
  config: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

const oidcConfigSchema = z.object({
  issuerUrl: z.string().min(1, "Issuer URL 不能为空"),
  clientId: z.string().min(1, "Client ID 不能为空"),
  clientSecret: z.string().optional().default(""),
  callbackUrl: z.string().optional().default(""),
  usePkce: z.boolean().default(true),
  scopes: z.string().optional().default("openid,profile,email"),
})

const ldapConfigSchema = z.object({
  serverUrl: z.string().min(1, "Server URL 不能为空"),
  bindDn: z.string().min(1, "Bind DN 不能为空"),
  bindPassword: z.string().optional().default(""),
  searchBase: z.string().min(1, "Search Base 不能为空"),
  userFilter: z.string().optional().default("(uid={{username}})"),
  useTls: z.boolean().default(false),
  skipVerify: z.boolean().default(false),
})

const schema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("oidc"),
    name: z.string().min(1, "名称不能为空"),
    domains: z.string().min(1, "域名不能为空"),
    forceSso: z.boolean().default(false),
    defaultRoleId: z.coerce.number().default(0),
    conflictStrategy: z.string().default("fail"),
    config: oidcConfigSchema,
  }),
  z.object({
    type: z.literal("ldap"),
    name: z.string().min(1, "名称不能为空"),
    domains: z.string().min(1, "域名不能为空"),
    forceSso: z.boolean().default(false),
    defaultRoleId: z.coerce.number().default(0),
    conflictStrategy: z.string().default("fail"),
    config: ldapConfigSchema,
  }),
])

type FormValues = z.infer<typeof schema>

interface IdentitySourceSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  source: IdentitySourceItem | null
}

const OIDC_DEFAULTS = {
  issuerUrl: "",
  clientId: "",
  clientSecret: "",
  callbackUrl: `${window.location.origin}/sso/callback`,
  usePkce: true,
  scopes: "openid,profile,email",
}

const LDAP_DEFAULTS = {
  serverUrl: "",
  bindDn: "",
  bindPassword: "",
  searchBase: "",
  userFilter: "(uid={{username}})",
  useTls: false,
  skipVerify: false,
}

export function IdentitySourceSheet({ open, onOpenChange, source }: IdentitySourceSheetProps) {
  const queryClient = useQueryClient()
  const isEditing = source !== null

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      type: "oidc",
      name: "",
      domains: "",
      forceSso: false,
      defaultRoleId: 0,
      conflictStrategy: "fail",
      config: OIDC_DEFAULTS,
    } as FormValues,
  })

  const selectedType = form.watch("type")

  useEffect(() => {
    if (open) {
      if (source) {
        form.reset({
          type: source.type as "oidc" | "ldap",
          name: source.name,
          domains: source.domains,
          forceSso: source.forceSso,
          defaultRoleId: source.defaultRoleId,
          conflictStrategy: source.conflictStrategy,
          config: source.config as FormValues["config"],
        })
      } else {
        form.reset({
          type: "oidc",
          name: "",
          domains: "",
          forceSso: false,
          defaultRoleId: 0,
          conflictStrategy: "fail",
          config: OIDC_DEFAULTS,
        } as FormValues)
      }
    }
  }, [open, source, form])

  // Reset config defaults when type changes (only in create mode)
  useEffect(() => {
    if (!isEditing && open) {
      if (selectedType === "oidc") {
        form.setValue("config", OIDC_DEFAULTS)
      } else if (selectedType === "ldap") {
        form.setValue("config", LDAP_DEFAULTS)
      }
    }
  }, [selectedType, isEditing, open, form])

  const createMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.post("/api/v1/identity-sources", values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["identity-sources"] })
      onOpenChange(false)
      toast.success("身份源创建成功")
    },
    onError: (err) => toast.error(err.message),
  })

  const updateMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.put(`/api/v1/identity-sources/${source!.id}`, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["identity-sources"] })
      onOpenChange(false)
      toast.success("身份源更新成功")
    },
    onError: (err) => toast.error(err.message),
  })

  const testMutation = useMutation({
    mutationFn: async () => {
      if (!source) throw new Error("请先保存身份源后再测试")
      const res = await api.post<{ success: boolean; message: string }>(
        `/api/v1/identity-sources/${source.id}/test`,
      )
      if (!res.success) throw new Error(res.message || "测试失败")
      return res
    },
    onSuccess: (res) => toast.success(res.message || "连接测试成功"),
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
      <SheetContent className="sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{isEditing ? "编辑身份源" : "新建身份源"}</SheetTitle>
          <SheetDescription className="sr-only">
            {isEditing ? "修改身份源配置" : "配置新的身份源"}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-4 px-4">
            {/* Basic fields */}
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>名称</FormLabel>
                  <FormControl>
                    <Input placeholder="例如：公司 Okta SSO" {...field} />
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
                  <FormLabel>类型</FormLabel>
                  {isEditing ? (
                    <div>
                      <Input value={field.value.toUpperCase()} disabled />
                      <p className="text-xs text-muted-foreground mt-1">类型创建后不可更改</p>
                    </div>
                  ) : (
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="请选择类型" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="oidc">OIDC</SelectItem>
                        <SelectItem value="ldap">LDAP</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="domains"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>域名</FormLabel>
                  <FormControl>
                    <Input placeholder="例如：acme.com（多个用逗号分隔）" {...field} />
                  </FormControl>
                  <FormDescription>匹配邮箱域名的用户将使用此身份源登录</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="conflictStrategy"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>账号冲突策略</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="fail">拒绝登录</SelectItem>
                      <SelectItem value="link">自动关联</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="defaultRoleId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>默认角色 ID</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      placeholder="0 表示使用系统默认角色"
                      {...field}
                      onChange={(e) => field.onChange(Number(e.target.value))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="forceSso"
              render={({ field }) => (
                <FormItem className="flex items-center gap-2">
                  <FormControl>
                    <Checkbox
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <FormLabel className="!mt-0">强制 SSO 登录</FormLabel>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Dynamic config section */}
            <div className="space-y-3 rounded-lg border p-4">
              <p className="text-sm font-medium">
                {selectedType === "oidc" ? "OIDC 配置" : "LDAP 配置"}
              </p>

              {selectedType === "oidc" && <OidcConfigFields form={form} isEditing={isEditing} />}
              {selectedType === "ldap" && <LdapConfigFields form={form} isEditing={isEditing} />}
            </div>

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

function OidcConfigFields({
  form,
  isEditing,
}: {
  form: ReturnType<typeof useForm<FormValues>>
  isEditing: boolean
}) {
  return (
    <>
      <FormField
        control={form.control}
        name="config.issuerUrl"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Issuer URL</FormLabel>
            <FormControl>
              <Input placeholder="https://accounts.google.com" {...field} value={field.value as string ?? ""} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.clientId"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Client ID</FormLabel>
            <FormControl>
              <Input placeholder="your-client-id" {...field} value={field.value as string ?? ""} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.clientSecret"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Client Secret</FormLabel>
            <FormControl>
              <Input
                type="password"
                placeholder={isEditing ? "留空则不修改" : "your-client-secret"}
                value={field.value === "\u2022\u2022\u2022\u2022\u2022\u2022" ? "" : (field.value as string ?? "")}
                onChange={field.onChange}
                onBlur={field.onBlur}
                name={field.name}
                ref={field.ref}
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.callbackUrl"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Callback URL</FormLabel>
            <FormControl>
              <Input
                readOnly
                className="bg-muted"
                {...field}
                value={field.value as string || `${window.location.origin}/sso/callback`}
              />
            </FormControl>
            <FormDescription>此地址需要配置到 OIDC 提供商</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.usePkce"
        render={({ field }) => (
          <FormItem className="flex items-center gap-2">
            <FormControl>
              <Checkbox
                checked={field.value as boolean}
                onCheckedChange={field.onChange}
              />
            </FormControl>
            <FormLabel className="!mt-0">启用 PKCE</FormLabel>
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.scopes"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Scopes</FormLabel>
            <FormControl>
              <Input placeholder="openid,profile,email" {...field} value={field.value as string ?? ""} />
            </FormControl>
            <FormDescription>多个 scope 用逗号分隔</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
    </>
  )
}

function LdapConfigFields({
  form,
  isEditing,
}: {
  form: ReturnType<typeof useForm<FormValues>>
  isEditing: boolean
}) {
  return (
    <>
      <FormField
        control={form.control}
        name="config.serverUrl"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Server URL</FormLabel>
            <FormControl>
              <Input placeholder="ldaps://ldap.corp.com:636" {...field} value={field.value as string ?? ""} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.bindDn"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Bind DN</FormLabel>
            <FormControl>
              <Input placeholder="cn=admin,dc=example,dc=com" {...field} value={field.value as string ?? ""} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.bindPassword"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Bind Password</FormLabel>
            <FormControl>
              <Input
                type="password"
                placeholder={isEditing ? "留空则不修改" : "bind password"}
                value={field.value === "\u2022\u2022\u2022\u2022\u2022\u2022" ? "" : (field.value as string ?? "")}
                onChange={field.onChange}
                onBlur={field.onBlur}
                name={field.name}
                ref={field.ref}
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.searchBase"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Search Base</FormLabel>
            <FormControl>
              <Input placeholder="ou=users,dc=example,dc=com" {...field} value={field.value as string ?? ""} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.userFilter"
        render={({ field }) => (
          <FormItem>
            <FormLabel>User Filter</FormLabel>
            <FormControl>
              <Input placeholder="(uid={{username}})" {...field} value={field.value as string ?? ""} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.useTls"
        render={({ field }) => (
          <FormItem className="flex items-center gap-2">
            <FormControl>
              <Checkbox
                checked={field.value as boolean}
                onCheckedChange={field.onChange}
              />
            </FormControl>
            <FormLabel className="!mt-0">启用 TLS</FormLabel>
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="config.skipVerify"
        render={({ field }) => (
          <FormItem className="flex items-center gap-2">
            <FormControl>
              <Checkbox
                checked={field.value as boolean}
                onCheckedChange={field.onChange}
              />
            </FormControl>
            <FormLabel className="!mt-0">跳过证书验证</FormLabel>
          </FormItem>
        )}
      />
    </>
  )
}
