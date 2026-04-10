import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"

export interface AuthProvider {
  id: number
  providerKey: string
  displayName: string
  enabled: boolean
  clientId: string
  clientSecret: string
  scopes: string
  callbackUrl: string
  sortOrder: number
  createdAt: string
  updatedAt: string
}

const schema = z.object({
  clientId: z.string().min(1, "Client ID 不能为空"),
  clientSecret: z.string(),
  scopes: z.string(),
  callbackUrl: z.string(),
})

type FormValues = z.infer<typeof schema>

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  provider: AuthProvider | null
}

export function ProviderSheet({ open, onOpenChange, provider }: Props) {
  const queryClient = useQueryClient()

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      clientId: "",
      clientSecret: "",
      scopes: "",
      callbackUrl: "",
    },
  })

  useEffect(() => {
    if (provider && open) {
      form.reset({
        clientId: provider.clientId,
        clientSecret: "",
        scopes: provider.scopes,
        callbackUrl: provider.callbackUrl,
      })
    }
  }, [provider, open, form])

  const mutation = useMutation({
    mutationFn: (values: FormValues) => {
      const body: Record<string, string> = {
        clientId: values.clientId,
        scopes: values.scopes,
        callbackUrl: values.callbackUrl,
      }
      if (values.clientSecret) {
        body.clientSecret = values.clientSecret
      }
      return api.put(
        `/api/v1/admin/auth-providers/${provider!.providerKey}`,
        body,
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["auth-providers"] })
      onOpenChange(false)
    },
  })

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>编辑认证源 — {provider?.displayName}</SheetTitle>
          <SheetDescription>
            配置 {provider?.displayName} OAuth 参数
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
            className="space-y-4 px-4 pt-4"
          >
            <FormField
              control={form.control}
              name="clientId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Client ID</FormLabel>
                  <FormControl>
                    <Input placeholder="OAuth Client ID" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="clientSecret"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Client Secret</FormLabel>
                  <FormControl>
                    <Input
                      type="password"
                      placeholder={
                        provider?.clientSecret
                          ? "留空保持不变"
                          : "OAuth Client Secret"
                      }
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {provider?.clientSecret
                      ? "已配置，留空则保持原值"
                      : "尚未配置"}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="scopes"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Scopes</FormLabel>
                  <FormControl>
                    <Input placeholder="例如: user:email,read:user" {...field} />
                  </FormControl>
                  <FormDescription>
                    多个 scope 用逗号分隔，留空使用默认值
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="callbackUrl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>回调地址</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="https://your-domain.com/oauth/callback"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    OAuth 回调地址，需与 OAuth App 配置一致
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "保存中..." : "保存"}
            </Button>
          </form>
        </Form>
      </SheetContent>
    </Sheet>
  )
}
