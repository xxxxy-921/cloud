import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"

const schema = z.object({
  appName: z.string().min(1, "系统名称不能为空").max(50, "系统名称不能超过 50 个字符"),
})

type FormValues = z.infer<typeof schema>

export function SiteNameCard({ appName }: { appName: string }) {
  const queryClient = useQueryClient()

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    values: { appName },
  })

  const mutation = useMutation({
    mutationFn: (data: FormValues) => api.put("/api/v1/site-info", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["site-info"] })
      form.reset(form.getValues())
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>基本信息</CardTitle>
        <CardDescription>设置系统的显示名称，将在导航栏中展示</CardDescription>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit((v) => mutation.mutate(v))} className="flex items-end gap-3">
            <FormField
              control={form.control}
              name="appName"
              render={({ field }) => (
                <FormItem className="flex-1">
                  <FormLabel>系统名称</FormLabel>
                  <FormControl>
                    <Input placeholder="请输入系统名称" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button
              type="submit"
              disabled={!form.formState.isDirty || mutation.isPending}
            >
              {mutation.isPending ? "保存中..." : "保存"}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  )
}
